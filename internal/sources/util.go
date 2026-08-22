// Copyright 2025 Google LLC
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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/googleapis/mcp-toolbox/internal/log"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GetCloudSQLDialOpts retrieve dial options with the right ip type and user agent for cloud sql
// databases.
func GetCloudSQLOpts(ipType, userAgent string, useIAM bool) ([]cloudsqlconn.Option, error) {
	opts := []cloudsqlconn.Option{cloudsqlconn.WithUserAgent(userAgent)}
	switch strings.ToLower(ipType) {
	case "private":
		opts = append(opts, cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPrivateIP()))
	case "public":
		opts = append(opts, cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPublicIP()))
	case "psc":
		opts = append(opts, cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPSC()))
	default:
		return nil, fmt.Errorf("invalid ipType %s. Must be one of `public`, `private`, or `psc`", ipType)
	}

	if useIAM {
		opts = append(opts, cloudsqlconn.WithIAMAuthN())
	}
	return opts, nil
}

// GetIAMPrincipalEmailFromADC finds the email associated with ADC
func GetIAMPrincipalEmailFromADC(ctx context.Context, dbType string) (string, error) {
	// Finds ADC and returns an HTTP client associated with it
	client, err := google.DefaultClient(ctx,
		"https://www.googleapis.com/auth/userinfo.email")
	if err != nil {
		return "", fmt.Errorf("failed to call userinfo endpoint: %w", err)
	}

	// Retrieve the email associated with the token
	resp, err := client.Get("https://oauth2.googleapis.com/tokeninfo")
	if err != nil {
		return "", fmt.Errorf("failed to call tokeninfo endpoint: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body %d: %s", resp.StatusCode, string(bodyBytes))
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tokeninfo endpoint returned non-OK status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Unmarshal response body and get `email`
	var responseJSON map[string]any
	err = json.Unmarshal(bodyBytes, &responseJSON)
	if err != nil {

		return "", fmt.Errorf("error parsing JSON: %v", err)
	}

	emailValue, ok := responseJSON["email"]
	if !ok {
		return "", fmt.Errorf("email not found in response: %v", err)
	}

	fullEmail, ok := emailValue.(string)
	if !ok {
		return "", fmt.Errorf("email field is not a string")
	}

	var username string
	// Format the username based on Database Type
	switch strings.ToLower(dbType) {
	case "mysql":
		username, _, _ = strings.Cut(fullEmail, "@")

	case "postgres":
		// service account email used for IAM should trim the suffix
		username = strings.TrimSuffix(fullEmail, ".gserviceaccount.com")

	default:
		return "", fmt.Errorf("unsupported dbType: %s. Use 'mysql' or 'postgres'", dbType)
	}

	if username == "" {
		return "", fmt.Errorf("username from ADC cannot be an empty string")
	}

	return username, nil
}

func GetIAMAccessToken(ctx context.Context) (string, error) {
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return "", fmt.Errorf("failed to find default credentials (run 'gcloud auth application-default login'?): %w", err)
	}

	token, err := creds.TokenSource.Token() // This gets an oauth2.Token
	if err != nil {
		return "", fmt.Errorf("failed to get token from token source: %w", err)
	}

	if !token.Valid() {
		return "", fmt.Errorf("retrieved token is invalid or expired")
	}
	return token.AccessToken, nil
}

// reloadingTokenSource wraps an Application Default Credentials token source
// so that, when the cached source can no longer mint a token and the ADC on
// disk has changed since the source was built (for example after a user
// re-runs `gcloud auth application-default login`, which writes a new refresh
// token and revokes the old one), it reloads ADC from disk and recovers,
// instead of failing until the process is restarted. It is safe for
// concurrent use.
type reloadingTokenSource struct {
	mu sync.Mutex
	// gen is incremented on every swap of cur, so a caller can tell that a
	// concurrent caller already reloaded while it waited for the lock. (The
	// exported constructor wraps this type in oauth2.ReuseTokenSource, whose
	// own mutex already serializes Token calls; the generation check is
	// defense in depth for any caller that uses the type directly.)
	gen uint64
	cur oauth2.TokenSource
	// curJSON is the ADC JSON cur was built from (nil for JSON-less
	// credentials, such as ones served by a metadata server). A reload
	// happens only when the JSON on disk no longer matches it.
	curJSON []byte
	// baseJSON is the ADC JSON at construction. Reloaded credentials may
	// differ from it only in fields a routine re-auth or key rotation is
	// expected to touch; see checkADCDrift. It is never updated, so repeated
	// reloads cannot walk the credentials away from what the server started
	// with in small steps.
	baseJSON []byte
	// discover re-reads ADC from disk. It is a field so tests can stub it;
	// production code sets google.FindDefaultCredentials (see
	// newReloadingTokenSource).
	discover func() (*google.Credentials, error)
	// name identifies the owning source in logs. logger is the server's
	// configured logger captured at construction (Token has no context); nil
	// falls back to the process-default slog logger.
	name   string
	logger log.Logger
	// pinnedQuotaProject and skipQuotaGuard configure the quota-project half
	// of checkADCDrift; baseQuotaProject is the ADC file's value at
	// construction. See ADCReloadParams.
	pinnedQuotaProject string
	skipQuotaGuard     bool
	baseQuotaProject   string
}

// Token returns a token from the current source. When the source can no
// longer mint one, it checks whether the ADC on disk changed since the
// current source was built: if unchanged, the failure is returned as-is
// (rate limits, server errors, timeouts and network drops all land here, and
// reloading cannot fix any of them); if changed, the credentials are
// reloaded from disk and the token is minted from the reloaded source. The
// changed-on-disk check costs one small local file read and runs only on
// requests that already failed to mint, so transient failures never trigger
// a reload no matter what error shape they surface as.
func (r *reloadingTokenSource) Token() (*oauth2.Token, error) {
	r.mu.Lock()
	cur, gen := r.cur, r.gen
	r.mu.Unlock()
	tok, mintErr := cur.Token()
	if mintErr == nil {
		return tok, nil
	}

	// Serialize the reload under the lock so concurrent callers that all saw
	// the same failing source don't each hit disk (a thundering herd).
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.gen != gen {
		// A concurrent caller already reloaded while we waited for the lock;
		// serve from the reloaded source instead of rediscovering.
		tok, err := r.cur.Token()
		if err == nil {
			return tok, nil
		}
		mintErr = err
	}
	newCred, err := r.discover()
	if err != nil {
		// The mint failure is attached as text rather than wrapped: a
		// wrapped *oauth2.RetrieveError would be re-rendered by the client
		// libraries' auth adapters, replacing this message with the raw
		// token-endpoint error and hiding what actually needs fixing.
		return nil, fmt.Errorf("failed to reload default credentials after a mint failure (%v): %w", mintErr, err)
	}
	if bytes.Equal(newCred.JSON, r.curJSON) {
		// The credentials on disk are the same ones that just failed, so a
		// reload cannot help; report the mint failure untouched.
		return nil, mintErr
	}
	if err := r.checkADCDrift(newCred.JSON); err != nil {
		// A refusal is the "the credentials were swapped under a running
		// server" signal, so log it server-side too: the error itself only
		// reaches whichever caller happened to trigger this mint.
		r.warn("refused to adopt reloaded application default credentials", "source", r.name, "error", err.Error())
		// Keep the mint failure that triggered the reload visible alongside
		// the refusal, as text: wrapping it would let the auth adapters
		// re-render the inner token-endpoint error and hide the actionable
		// refusal from the caller.
		return nil, fmt.Errorf("%w (after mint failure: %v)", err, mintErr)
	}
	r.cur, r.curJSON = newCred.TokenSource, newCred.JSON
	r.gen++
	r.warn("adopted reloaded application default credentials from disk after a token mint failure", "source", r.name, "credentialType", adcFieldString(newCred.JSON, "type"))
	tok, err = r.cur.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to mint a token from the reloaded default credentials: %w", err)
	}
	return tok, nil
}

// warn logs through the server's configured logger when one was captured at
// construction, falling back to the process-default slog logger otherwise
// (Token has no context to resolve a logger from).
func (r *reloadingTokenSource) warn(msg string, keysAndValues ...any) {
	if r.logger != nil {
		r.logger.WarnContext(context.Background(), msg, keysAndValues...)
		return
	}
	slog.Warn(msg, keysAndValues...)
}

// adcRotationFields are the top-level ADC JSON fields a routine re-auth or
// service-account key rotation is expected to change. quota_project_id is
// here because it has its own guard with pin-awareness (see checkADCDrift);
// everything else in the file either identifies the principal or controls
// where credential material is sent (token_uri, credential_source, ...), so
// a change to any field outside this set is refused rather than adopted.
// Failing closed on unenumerated fields is deliberate: it also covers fields
// future credential formats add.
var adcRotationFields = map[string]bool{
	"refresh_token":    true,
	"private_key":      true,
	"private_key_id":   true,
	"expiry":           true,
	"quota_project_id": true,
}

// checkADCDrift refuses reloaded ADC that diverges from what the server
// started with in any way a routine re-auth would not. Two guards:
//
// Field drift: every top-level JSON field outside adcRotationFields must be
// byte-identical to its value at construction. This pins the principal
// (type, client_email, client_id, audience) and, just as importantly, where
// credential material travels (token_uri, credential_source, universe
// domain): a reload is acceptance of a credential configuration from the
// filesystem mid-flight, so anything unexpected fails closed with an error
// naming the field. For authorized_user credentials a re-auth that switches
// Google accounts under the same OAuth client remains indistinguishable on
// disk from the routine re-auth this type exists to survive; it is adopted,
// which is why every adoption is also logged.
//
// Quota project: X-Goog-User-Project is resolved once at client
// construction, so clients that derive it from the ADC file would keep
// sending the original value after a reload. When neither the source config
// nor GOOGLE_CLOUD_QUOTA_PROJECT pins it, a change to a non-empty baked
// value is refused with an actionable error; a value appearing where none
// was baked in is adopted (nothing stale is sent, and refusing would trade
// an availability outage for nothing).
func (r *reloadingTokenSource) checkADCDrift(newJSON []byte) error {
	if field := adcFieldDrift(r.baseJSON, newJSON); field != "" {
		return fmt.Errorf("failed to reload default credentials: the %s field on disk changed from what this server started with, which a routine re-auth would not do; restart the server to adopt the new credentials", field)
	}
	if !r.skipQuotaGuard && r.pinnedQuotaProject == "" && r.baseQuotaProject != "" {
		if qp := adcQuotaProjectFromJSON(newJSON); qp != r.baseQuotaProject {
			return fmt.Errorf("failed to reload default credentials: their quota project changed from %q to %q and clients built from this source still send the original value; restart the server to pick up the change", r.baseQuotaProject, qp)
		}
	}
	return nil
}

// adcFieldDrift returns the name of the first top-level field that differs
// between two ADC JSON files and is not expected to change on a routine
// rotation (see adcRotationFields), or "" if there is none. An absent
// universe_domain is normalized to its default before comparing, since the
// client libraries treat them identically and newer tooling spells the
// default out.
func adcFieldDrift(baseJSON, newJSON []byte) string {
	base, ok := adcFields(baseJSON)
	if !ok {
		return "unparseable baseline"
	}
	next, ok := adcFields(newJSON)
	if !ok {
		return "unparseable reload"
	}
	fields := make([]string, 0, len(base)+len(next))
	for f := range base {
		fields = append(fields, f)
	}
	for f := range next {
		if _, seen := base[f]; !seen {
			fields = append(fields, f)
		}
	}
	sort.Strings(fields)
	for _, f := range fields {
		if adcRotationFields[f] {
			continue
		}
		if !bytes.Equal(base[f], next[f]) {
			return f
		}
	}
	return ""
}

// adcFields parses the top level of an ADC JSON file, normalizing an absent
// universe_domain to the libraries' default. A nil file (JSON-less
// credentials, e.g. from a metadata server) is an empty field set.
func adcFields(credJSON []byte) (map[string]json.RawMessage, bool) {
	fields := map[string]json.RawMessage{}
	if len(credJSON) == 0 {
		return fields, true
	}
	if err := json.Unmarshal(credJSON, &fields); err != nil {
		return nil, false
	}
	if _, ok := fields["universe_domain"]; !ok {
		fields["universe_domain"] = json.RawMessage(`"googleapis.com"`)
	}
	return fields, true
}

// adcFieldString extracts one top-level string field from an ADC JSON file,
// returning "" when absent or unparseable.
func adcFieldString(credJSON []byte, field string) string {
	fields, ok := adcFields(credJSON)
	if !ok {
		return ""
	}
	var v string
	if err := json.Unmarshal(fields[field], &v); err != nil {
		return ""
	}
	return v
}

// quotaProjectEnvVar is the environment variable the client libraries give
// precedence over the ADC file's quota_project_id (see
// google.golang.org/api/internal.GetQuotaProject).
const quotaProjectEnvVar = "GOOGLE_CLOUD_QUOTA_PROJECT"

// ADCReloadParams configures NewReloadingDefaultCredentials.
type ADCReloadParams struct {
	// Name identifies the owning source in reload log lines.
	Name string
	// Scopes are requested at discovery and at every rediscovery.
	Scopes []string
	// QuotaProjectPin, when non-empty, declares that quota attribution for
	// the clients built from these credentials is managed explicitly (the
	// same value is passed to option.WithQuotaProject), so a change to the
	// ADC file's quota_project_id never reaches request headers and must not
	// block a reload. Leave empty when the clients derive their quota
	// project from the ADC file. The GOOGLE_CLOUD_QUOTA_PROJECT environment
	// variable, to which the client libraries give the same precedence, is
	// honored automatically.
	QuotaProjectPin string
	// SkipQuotaProjectGuard declares that no user-visible client reads the
	// quota project from these credentials' JSON, so quota-project drift
	// must never refuse a reload regardless of pinning. Set it for
	// impersonation base credentials: their JSON-derived quota project only
	// labels the internal IAM generateAccessToken call, and refusing a
	// reload over it would trade a full outage of the source for the quota
	// attribution of that internal call. The field-drift guard still
	// applies.
	SkipQuotaProjectGuard bool
}

// NewReloadingDefaultCredentials finds Application Default Credentials with
// the given scopes and wraps their token source so a live server survives
// the on-disk credentials being rotated (for example by a user re-running
// `gcloud auth application-default login`) without a restart: when the
// wrapped source can no longer mint a token and the ADC file changed, the
// credentials are transparently reloaded from disk. Reloaded credentials may
// differ from the originals only in fields a routine rotation touches, and
// must preserve the effective quota project; anything else is refused with
// an error saying a restart is required (see checkADCDrift). Adoptions and
// refusals are logged through the logger carried by ctx, when present.
//
// The returned credentials are the discovered ones with only TokenSource
// replaced, so client options that read credential metadata (such as the
// quota project in the JSON, applied via option.WithCredentials) behave
// exactly as they do with unwrapped ADC.
func NewReloadingDefaultCredentials(ctx context.Context, p ADCReloadParams) (*google.Credentials, error) {
	cred, err := google.FindDefaultCredentials(ctx, p.Scopes...)
	if err != nil {
		return nil, fmt.Errorf("failed to find default Google Cloud credentials with scopes %v: %w", p.Scopes, err)
	}
	cred.TokenSource = newReloadingTokenSource(ctx, cred, p)
	return cred, nil
}

// newReloadingTokenSource wraps cred's token source per
// NewReloadingDefaultCredentials. The result is wrapped in
// oauth2.ReuseTokenSource so a valid access token is cached and the reload
// path runs only on an actual mint failure, not on every request. cred must
// come from google.FindDefaultCredentials (in particular, TokenSource must
// be non-nil).
func newReloadingTokenSource(ctx context.Context, cred *google.Credentials, p ADCReloadParams) oauth2.TokenSource {
	return oauth2.ReuseTokenSource(nil, newReloader(ctx, cred, p))
}

// newReloader builds the inner reloading source for newReloadingTokenSource;
// it is separate so tests can exercise it without the oauth2.ReuseTokenSource
// wrapper in the way.
func newReloader(ctx context.Context, cred *google.Credentials, p ADCReloadParams) *reloadingTokenSource {
	pinned := p.QuotaProjectPin
	if pinned == "" {
		// The client libraries resolve the effective quota project as
		// option.WithQuotaProject first, then GOOGLE_CLOUD_QUOTA_PROJECT,
		// then the ADC file; when the environment variable is set, the
		// file's value never reaches request headers, so drift in it is
		// harmless. Read once at construction, matching when the libraries
		// read it (at dial).
		pinned = os.Getenv(quotaProjectEnvVar)
	}
	// Token() has no context, so capture the server's configured logger now;
	// a missing logger (some tests) falls back to the process default.
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		logger = nil
	}
	scopes := p.Scopes
	return &reloadingTokenSource{
		cur:      cred.TokenSource,
		curJSON:  cred.JSON,
		baseJSON: cred.JSON,
		discover: func() (*google.Credentials, error) {
			// A token source outlives any single request, so rediscover with
			// a background context rather than capturing a request or
			// initialization context that may later be cancelled.
			return google.FindDefaultCredentials(context.Background(), scopes...)
		},
		name:               p.Name,
		logger:             logger,
		pinnedQuotaProject: pinned,
		skipQuotaGuard:     p.SkipQuotaProjectGuard,
		baseQuotaProject:   adcQuotaProjectFromJSON(cred.JSON),
	}
}

// adcQuotaProjectFromJSON extracts the quota_project_id field from an ADC
// JSON credential file, returning "" if absent (for example metadata-server
// credentials, which carry no JSON).
func adcQuotaProjectFromJSON(credJSON []byte) string {
	if len(credJSON) == 0 {
		return ""
	}
	var f struct {
		QuotaProjectID string `json:"quota_project_id"`
	}
	if err := json.Unmarshal(credJSON, &f); err != nil {
		return ""
	}
	return f.QuotaProjectID
}
