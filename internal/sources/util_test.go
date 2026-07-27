// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sources

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// stubTokenSource is a test oauth2.TokenSource returning a fixed token/error.
type stubTokenSource struct {
	tok *oauth2.Token
	err error
}

func (s stubTokenSource) Token() (*oauth2.Token, error) { return s.tok, s.err }

// revokedGrantError mimics the error the ADC token source returns after
// `gcloud auth application-default login` rotates the on-disk refresh token
// and the old one is revoked: an OAuth token-endpoint rejection with RFC 6749
// error code "invalid_grant".
func revokedGrantError() error {
	return &oauth2.RetrieveError{
		Response:  &http.Response{StatusCode: http.StatusBadRequest},
		ErrorCode: "invalid_grant",
	}
}

const (
	// adcOldJSON stands in for the ADC file the source was built from;
	// adcRotatedJSON is the same identity and quota project after a routine
	// re-auth (only the refresh token changed).
	adcOldJSON     = `{"type":"authorized_user","client_id":"id","refresh_token":"old","quota_project_id":"proj-a"}`
	adcRotatedJSON = `{"type":"authorized_user","client_id":"id","refresh_token":"new","quota_project_id":"proj-a"}`
)

// newTestReloader builds a reloadingTokenSource around a failing current
// source and a stubbed discovery, mirroring what newReloader wires up in
// production.
func newTestReloader(cur oauth2.TokenSource, curJSON string, discover func() (*google.Credentials, error)) *reloadingTokenSource {
	var j []byte
	if curJSON != "" {
		j = []byte(curJSON)
	}
	return &reloadingTokenSource{
		cur:              cur,
		curJSON:          j,
		baseJSON:         j,
		discover:         discover,
		baseQuotaProject: adcQuotaProjectFromJSON(j),
	}
}

// TestReloadingTokenSource_HappyPathNoReload verifies that when the current
// source can mint a token, its token is returned and discovery never runs.
func TestReloadingTokenSource_HappyPathNoReload(t *testing.T) {
	r := newTestReloader(
		stubTokenSource{tok: &oauth2.Token{AccessToken: "cached"}},
		adcOldJSON,
		func() (*google.Credentials, error) {
			t.Error("discover must not be called on the happy path")
			return nil, errors.New("unreachable")
		},
	)
	tok, err := r.Token()
	if err != nil {
		t.Fatalf("Token() returned error, want success: %v", err)
	}
	if tok.AccessToken != "cached" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "cached")
	}
}

// TestReloadingTokenSource_ReloadsWhenDiskChanged verifies that when the
// current source fails and the ADC on disk changed, the source reloads,
// swaps in the rediscovered credentials, and serves their token: a live
// session recovers from a re-auth without a process restart. The reload is
// keyed on the on-disk change, not the error's shape, so recovery works for
// every ADC credential type regardless of how its token source wraps errors.
func TestReloadingTokenSource_ReloadsWhenDiskChanged(t *testing.T) {
	tcs := []struct {
		desc    string
		mintErr error
	}{
		{"revoked grant", revokedGrantError()},
		{"opaque wrapper error", errors.New("oauth2/google: unable to generate access token: invalid_grant")},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			fresh := stubTokenSource{tok: &oauth2.Token{AccessToken: "fresh"}}
			var discoveries int
			r := newTestReloader(
				stubTokenSource{err: tc.mintErr},
				adcOldJSON,
				func() (*google.Credentials, error) {
					discoveries++
					return &google.Credentials{TokenSource: fresh, JSON: []byte(adcRotatedJSON)}, nil
				},
			)
			tok, err := r.Token()
			if err != nil {
				t.Fatalf("Token() returned error, want successful reload: %v", err)
			}
			if tok.AccessToken != "fresh" {
				t.Errorf("AccessToken = %q, want %q (the reloaded source's token)", tok.AccessToken, "fresh")
			}
			if discoveries != 1 {
				t.Errorf("discover called %d times, want 1", discoveries)
			}
			if !strings.Contains(string(r.curJSON), `"refresh_token":"new"`) {
				t.Error("expected curJSON to be updated to the reloaded ADC JSON")
			}
			if r.gen != 1 {
				t.Errorf("gen = %d, want 1 after one reload", r.gen)
			}
		})
	}
}

// TestReloadingTokenSource_NoReloadWhenDiskUnchanged verifies that when the
// ADC on disk still matches what the current source was built from, the mint
// failure is returned untouched, whatever its shape: rate limits, server
// errors, network drops, and even a revoked grant (nothing new on disk means
// nothing to reload). The JSON-less case covers metadata-server credentials,
// which have nothing on disk to compare or reload.
func TestReloadingTokenSource_NoReloadWhenDiskUnchanged(t *testing.T) {
	tcs := []struct {
		desc    string
		curJSON string
		mintErr error
	}{
		{"server error", adcOldJSON, &oauth2.RetrieveError{Response: &http.Response{StatusCode: http.StatusInternalServerError}}},
		{"rate limited", adcOldJSON, &oauth2.RetrieveError{Response: &http.Response{StatusCode: http.StatusTooManyRequests}}},
		{"network drop", adcOldJSON, errors.New("net/http: TLS handshake timeout")},
		{"revoked grant, no re-auth yet", adcOldJSON, revokedGrantError()},
		{"JSON-less metadata-server credentials", "", errors.New("metadata: connection error")},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			cur := stubTokenSource{err: tc.mintErr}
			r := newTestReloader(cur, tc.curJSON, func() (*google.Credentials, error) {
				var j []byte
				if tc.curJSON != "" {
					j = []byte(tc.curJSON)
				}
				return &google.Credentials{TokenSource: stubTokenSource{tok: &oauth2.Token{AccessToken: "must-not-serve"}}, JSON: j}, nil
			})
			_, err := r.Token()
			if !errors.Is(err, tc.mintErr) {
				t.Errorf("Token() error = %v, want the original mint error %v", err, tc.mintErr)
			}
			if r.gen != 0 {
				t.Error("token source was swapped although the ADC on disk did not change")
			}
		})
	}
}

// TestReloadingTokenSource_DiscoverFailurePropagates verifies that if
// rediscovering ADC fails, the discovery error surfaces (inspectable via
// errors.Is) with the original mint failure preserved as text. The mint
// error is deliberately not wrapped: a wrapped *oauth2.RetrieveError would
// be re-rendered by the client libraries' auth adapters, hiding the reload
// context from the caller.
func TestReloadingTokenSource_DiscoverFailurePropagates(t *testing.T) {
	mintErr := revokedGrantError()
	discoverErr := errors.New("ADC gone from disk")
	r := newTestReloader(stubTokenSource{err: mintErr}, adcOldJSON, func() (*google.Credentials, error) {
		return nil, discoverErr
	})
	tok, err := r.Token()
	if err == nil {
		t.Fatalf("Token() returned nil error, want discovery failure; token=%v", tok)
	}
	if !strings.Contains(err.Error(), "invalid_grant") {
		t.Errorf("Token() error %q does not mention the mint failure", err)
	}
	if !errors.Is(err, discoverErr) {
		t.Errorf("Token() error %q does not wrap the discovery error", err)
	}
}

// TestReloadingTokenSource_ReloadedSourceAlsoFails covers the branch where
// the reloaded credentials are adopted but still cannot mint, and verifies
// the error names the reload so an operator knows one already happened.
func TestReloadingTokenSource_ReloadedSourceAlsoFails(t *testing.T) {
	stillBroken := errors.New("still broken after reload")
	r := newTestReloader(stubTokenSource{err: revokedGrantError()}, adcOldJSON, func() (*google.Credentials, error) {
		return &google.Credentials{TokenSource: stubTokenSource{err: stillBroken}, JSON: []byte(adcRotatedJSON)}, nil
	})
	_, err := r.Token()
	if !errors.Is(err, stillBroken) {
		t.Fatalf("Token() error = %v, want the reloaded source's mint error", err)
	}
	if !strings.Contains(err.Error(), "reloaded") {
		t.Errorf("Token() error %q does not say a reload already happened", err)
	}
}

// TestReloadingTokenSource_ConcurrentReload exercises the mutex under
// concurrent callers (run with -race). All callers must observe a successful
// token, and the generation check must keep the reload single-flight.
func TestReloadingTokenSource_ConcurrentReload(t *testing.T) {
	fresh := stubTokenSource{tok: &oauth2.Token{AccessToken: "fresh"}}
	var discoveries atomic.Int32
	r := newTestReloader(stubTokenSource{err: revokedGrantError()}, adcOldJSON, func() (*google.Credentials, error) {
		discoveries.Add(1)
		return &google.Credentials{TokenSource: fresh, JSON: []byte(adcRotatedJSON)}, nil
	})
	const n = 32
	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			tok, err := r.Token()
			if err != nil || tok.AccessToken != "fresh" {
				t.Errorf("concurrent Token() = (%v, %v), want (fresh, nil)", tok, err)
			}
		}()
	}
	wg.Wait()
	if got := discoveries.Load(); got != 1 {
		t.Errorf("discover ran %d times, want exactly 1 (the generation check should serialize concurrent reloads)", got)
	}
}

// TestCheckADCDrift exercises the guards that refuse reloaded credentials
// that differ from what the server started with in ways a routine rotation
// would not produce.
func TestCheckADCDrift(t *testing.T) {
	tcs := []struct {
		desc      string
		baseJSON  string
		pinned    string
		skipQuota bool
		newJSON   string
		wantErr   string
	}{
		{
			desc:     "routine re-auth, only the refresh token changed",
			baseJSON: adcOldJSON,
			newJSON:  adcRotatedJSON,
		},
		{
			desc:     "service account key rotation keeps the principal",
			baseJSON: `{"type":"service_account","client_email":"sa@p.iam.gserviceaccount.com","private_key_id":"k1","private_key":"pem1"}`,
			newJSON:  `{"type":"service_account","client_email":"sa@p.iam.gserviceaccount.com","private_key_id":"k2","private_key":"pem2"}`,
		},
		{
			desc:     "universe_domain spelled out as the default is not drift",
			baseJSON: `{"type":"service_account","client_email":"sa@p.iam.gserviceaccount.com"}`,
			newJSON:  `{"type":"service_account","client_email":"sa@p.iam.gserviceaccount.com","universe_domain":"googleapis.com"}`,
		},
		{
			desc:     "credential type changed",
			baseJSON: adcOldJSON,
			newJSON:  `{"type":"service_account","client_id":"id","refresh_token":"x","quota_project_id":"proj-a"}`,
			wantErr:  "the type field on disk changed",
		},
		{
			desc:     "service account principal changed",
			baseJSON: `{"type":"service_account","client_email":"a@p.iam.gserviceaccount.com"}`,
			newJSON:  `{"type":"service_account","client_email":"b@p.iam.gserviceaccount.com"}`,
			wantErr:  "the client_email field on disk changed",
		},
		{
			desc:     "token endpoint changed",
			baseJSON: `{"type":"service_account","client_email":"sa@p.iam.gserviceaccount.com","token_uri":"https://oauth2.googleapis.com/token"}`,
			newJSON:  `{"type":"service_account","client_email":"sa@p.iam.gserviceaccount.com","token_uri":"https://attacker.example/token"}`,
			wantErr:  "the token_uri field on disk changed",
		},
		{
			desc:     "workload-identity credential_source changed",
			baseJSON: `{"type":"external_account","audience":"//iam.googleapis.com/x","credential_source":{"file":"/var/run/token"}}`,
			newJSON:  `{"type":"external_account","audience":"//iam.googleapis.com/x","credential_source":{"file":"/etc/shadow"}}`,
			wantErr:  "the credential_source field on disk changed",
		},
		{
			desc:     "universe domain changed to a sovereign cloud",
			baseJSON: `{"type":"service_account","client_email":"sa@p.iam.gserviceaccount.com"}`,
			newJSON:  `{"type":"service_account","client_email":"sa@p.iam.gserviceaccount.com","universe_domain":"example.sovereign.cloud"}`,
			wantErr:  "the universe_domain field on disk changed",
		},
		{
			desc:     "quota project changed and clients derive it from ADC",
			baseJSON: adcOldJSON,
			newJSON:  `{"type":"authorized_user","client_id":"id","refresh_token":"new","quota_project_id":"proj-b"}`,
			wantErr:  `"proj-a" to "proj-b"`,
		},
		{
			desc:     "quota project disappeared and clients derive it from ADC",
			baseJSON: adcOldJSON,
			newJSON:  `{"type":"authorized_user","client_id":"id","refresh_token":"new"}`,
			wantErr:  `"proj-a" to ""`,
		},
		{
			desc:     "quota project appeared where none was baked in",
			baseJSON: `{"type":"authorized_user","client_id":"id","refresh_token":"old"}`,
			newJSON:  `{"type":"authorized_user","client_id":"id","refresh_token":"new","quota_project_id":"proj-b"}`,
		},
		{
			desc:     "quota project changed but explicitly pinned on the clients",
			baseJSON: adcOldJSON,
			pinned:   "pinned-proj",
			newJSON:  `{"type":"authorized_user","client_id":"id","refresh_token":"new","quota_project_id":"proj-b"}`,
		},
		{
			desc:      "quota project changed but no client reads it from these credentials",
			baseJSON:  adcOldJSON,
			skipQuota: true,
			newJSON:   `{"type":"authorized_user","client_id":"id","refresh_token":"new","quota_project_id":"proj-b"}`,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			r := newTestReloader(stubTokenSource{}, tc.baseJSON, nil)
			r.pinnedQuotaProject = tc.pinned
			r.skipQuotaGuard = tc.skipQuota
			err := r.checkADCDrift([]byte(tc.newJSON))
			if tc.wantErr == "" {
				if err != nil {
					t.Errorf("checkADCDrift() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("checkADCDrift() = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

// TestReloadingTokenSource_RefusalIsTerminal verifies the promise in the
// refusal messages: after a refused drift, subsequent calls keep returning
// the refusal without adopting anything (nothing is swapped, the generation
// never moves) until the process restarts.
func TestReloadingTokenSource_RefusalIsTerminal(t *testing.T) {
	drifted := `{"type":"authorized_user","client_id":"other-client","refresh_token":"new","quota_project_id":"proj-a"}`
	r := newTestReloader(stubTokenSource{err: revokedGrantError()}, adcOldJSON, func() (*google.Credentials, error) {
		return &google.Credentials{TokenSource: stubTokenSource{tok: &oauth2.Token{AccessToken: "must-not-serve"}}, JSON: []byte(drifted)}, nil
	})
	for i := range 2 {
		_, err := r.Token()
		if err == nil || !strings.Contains(err.Error(), "client_id") {
			t.Fatalf("call %d: Token() = %v, want the client_id drift refusal", i+1, err)
		}
	}
	if r.gen != 0 {
		t.Error("a refused reload must not swap the token source")
	}
}

// countingTokenSource records how many times Token() is called.
type countingTokenSource struct {
	tok   *oauth2.Token
	calls atomic.Int32
}

func (c *countingTokenSource) Token() (*oauth2.Token, error) {
	c.calls.Add(1)
	return c.tok, nil
}

// TestNewReloadingTokenSource_CachesValidToken verifies that the
// oauth2.ReuseTokenSource wrapper caches a valid token, so the reload/inner
// path runs only on an actual mint failure and not on every request.
func TestNewReloadingTokenSource_CachesValidToken(t *testing.T) {
	initial := &countingTokenSource{tok: &oauth2.Token{AccessToken: "cached", Expiry: time.Now().Add(time.Hour)}}
	ts := newReloadingTokenSource(context.Background(), &google.Credentials{TokenSource: initial}, ADCReloadParams{})
	for range 3 {
		tok, err := ts.Token()
		if err != nil || tok.AccessToken != "cached" {
			t.Fatalf("Token() = (%v, %v), want (cached, nil)", tok, err)
		}
	}
	if got := initial.calls.Load(); got != 1 {
		t.Errorf("initial source called %d times, want 1 (ReuseTokenSource should cache the valid token)", got)
	}
}

// TestNewReloader_QuotaProjectPinning verifies the pin resolution:
// GOOGLE_CLOUD_QUOTA_PROJECT counts as a pin (the client libraries give it
// precedence over the ADC file), and an explicit pin wins over the
// environment.
func TestNewReloader_QuotaProjectPinning(t *testing.T) {
	t.Setenv(quotaProjectEnvVar, "env-proj")
	tcs := []struct {
		desc       string
		pin        string
		wantPinned string
	}{
		{"env var pins when no explicit pin", "", "env-proj"},
		{"explicit pin wins over the env var", "arg-proj", "arg-proj"},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			r := newReloader(context.Background(), &google.Credentials{
				TokenSource: stubTokenSource{},
				JSON:        []byte(adcOldJSON),
			}, ADCReloadParams{QuotaProjectPin: tc.pin})
			if r.pinnedQuotaProject != tc.wantPinned {
				t.Errorf("pinnedQuotaProject = %q, want %q", r.pinnedQuotaProject, tc.wantPinned)
			}
		})
	}
}

// TestNewReloader_ParamsWiring pins the ADCReloadParams-to-field hops the
// call sites rely on.
func TestNewReloader_ParamsWiring(t *testing.T) {
	r := newReloader(context.Background(), &google.Credentials{
		TokenSource: stubTokenSource{},
		JSON:        []byte(adcOldJSON),
	}, ADCReloadParams{Name: "my-source", SkipQuotaProjectGuard: true})
	if r.name != "my-source" {
		t.Errorf("name = %q, want %q", r.name, "my-source")
	}
	if !r.skipQuotaGuard {
		t.Error("skipQuotaGuard = false, want true (SkipQuotaProjectGuard must reach the guard)")
	}
}

// TestNewReloader_EnvQuotaProjectPinAllowsDrift verifies end to end that a
// quota-project drift is adopted when GOOGLE_CLOUD_QUOTA_PROJECT pins the
// value the clients actually send.
func TestNewReloader_EnvQuotaProjectPinAllowsDrift(t *testing.T) {
	t.Setenv(quotaProjectEnvVar, "env-proj")
	fresh := stubTokenSource{tok: &oauth2.Token{AccessToken: "fresh"}}
	r := newReloader(context.Background(), &google.Credentials{
		TokenSource: stubTokenSource{err: revokedGrantError()},
		JSON:        []byte(adcOldJSON),
	}, ADCReloadParams{})
	r.discover = func() (*google.Credentials, error) {
		return &google.Credentials{TokenSource: fresh, JSON: []byte(`{"type":"authorized_user","client_id":"id","refresh_token":"new","quota_project_id":"proj-b"}`)}, nil
	}
	tok, err := r.Token()
	if err != nil {
		t.Fatalf("Token() = %v, want reload to proceed because GOOGLE_CLOUD_QUOTA_PROJECT pins the quota project", err)
	}
	if tok.AccessToken != "fresh" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "fresh")
	}
}

// TestNewReloadingDefaultCredentials_DiscoveryError verifies the exported
// constructor's error path when no ADC can be found at all.
func TestNewReloadingDefaultCredentials_DiscoveryError(t *testing.T) {
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(t.TempDir(), "missing.json"))
	_, err := NewReloadingDefaultCredentials(context.Background(), ADCReloadParams{Name: "test"})
	if err == nil || !strings.Contains(err.Error(), "failed to find default Google Cloud credentials") {
		t.Errorf("NewReloadingDefaultCredentials() error = %v, want discovery failure", err)
	}
}

// TestNewReloadingTokenSource_DiskIntegration exercises the real discovery
// path end to end via GOOGLE_APPLICATION_CREDENTIALS: the current source
// fails, discovery re-reads the (test-controlled) ADC file from disk, and
// the changed-on-disk predicate plus the drift guards decide the outcome. No
// network is involved: every case resolves before a reloaded source would
// mint.
func TestNewReloadingTokenSource_DiskIntegration(t *testing.T) {
	tcs := []struct {
		desc     string
		diskJSON string
		wantErr  string
	}{
		{
			desc:     "unchanged file returns the mint error untouched",
			diskJSON: `{"type":"authorized_user","client_id":"id","client_secret":"s","refresh_token":"old","quota_project_id":"proj-a"}`,
			wantErr:  "invalid_grant",
		},
		{
			desc:     "quota project drift is refused with both projects named",
			diskJSON: `{"type":"authorized_user","client_id":"id","client_secret":"s","refresh_token":"t","quota_project_id":"proj-b"}`,
			wantErr:  `"proj-a" to "proj-b"`,
		},
		{
			desc:     "identity drift is refused naming the field",
			diskJSON: `{"type":"authorized_user","client_id":"other-client","client_secret":"s","refresh_token":"t","quota_project_id":"proj-a"}`,
			wantErr:  "client_id field on disk changed",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			adcPath := filepath.Join(t.TempDir(), "adc.json")
			if err := os.WriteFile(adcPath, []byte(tc.diskJSON), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", adcPath)
			// Neutralize any ambient pin so the quota guard is exercised
			// (a developer machine may export GOOGLE_CLOUD_QUOTA_PROJECT,
			// which would otherwise let the drift case reload and reach the
			// network).
			t.Setenv(quotaProjectEnvVar, "")
			ts := newReloadingTokenSource(context.Background(), &google.Credentials{
				TokenSource: stubTokenSource{err: revokedGrantError()},
				JSON:        []byte(`{"type":"authorized_user","client_id":"id","client_secret":"s","refresh_token":"old","quota_project_id":"proj-a"}`),
			}, ADCReloadParams{Name: "disk-integration"})
			_, err := ts.Token()
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("Token() = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}
