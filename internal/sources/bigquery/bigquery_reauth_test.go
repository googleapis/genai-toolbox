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

package bigquery

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// stubTokenSource is a test oauth2.TokenSource returning a fixed token/error.
type stubTokenSource struct {
	tok *oauth2.Token
	err error
}

func (s stubTokenSource) Token() (*oauth2.Token, error) { return s.tok, s.err }

// errTokenSource mints nothing, simulating an ADC source whose on-disk refresh
// token has been revoked by a fresh `gcloud auth application-default login`.
type errTokenSource struct{}

func (errTokenSource) Token() (*oauth2.Token, error) {
	return nil, errors.New("oauth2: cannot fetch token: invalid_grant")
}

// TestReloadingTokenSource_HappyPathNoReload verifies that when the current
// source can mint a token, its token is returned and no reload happens.
func TestReloadingTokenSource_HappyPathNoReload(t *testing.T) {
	reloaded := false
	r := &reloadingTokenSource{
		cur: stubTokenSource{tok: &oauth2.Token{AccessToken: "cached"}},
		reload: func() (oauth2.TokenSource, error) {
			reloaded = true
			return nil, errors.New("reload must not be called on the happy path")
		},
	}
	tok, err := r.Token()
	if err != nil {
		t.Fatalf("Token() returned error, want success: %v", err)
	}
	if tok.AccessToken != "cached" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "cached")
	}
	if reloaded {
		t.Error("reload was called although the current source could mint a token")
	}
}

// TestReloadingTokenSource_ReloadsWhenCurFails verifies that when the current
// source can no longer mint a token, the source reloads ADC from disk, swaps in
// the reloaded source, and serves the token from it -- i.e. a live session
// recovers without a process restart.
func TestReloadingTokenSource_ReloadsWhenCurFails(t *testing.T) {
	fresh := stubTokenSource{tok: &oauth2.Token{AccessToken: "fresh"}}
	var reloads int
	r := &reloadingTokenSource{
		cur: errTokenSource{},
		reload: func() (oauth2.TokenSource, error) {
			reloads++
			return fresh, nil
		},
	}
	tok, err := r.Token()
	if err != nil {
		t.Fatalf("Token() returned error, want successful reload: %v", err)
	}
	if tok.AccessToken != "fresh" {
		t.Errorf("AccessToken = %q, want %q (the reloaded source's token)", tok.AccessToken, "fresh")
	}
	if reloads != 1 {
		t.Errorf("reload called %d times, want 1", reloads)
	}
	if _, ok := r.cur.(stubTokenSource); !ok {
		t.Error("expected cur to be swapped to the reloaded source")
	}
}

// TestReloadingTokenSource_ReloadFailurePropagates verifies that if reloading
// ADC also fails, the error is returned (not a nil token).
func TestReloadingTokenSource_ReloadFailurePropagates(t *testing.T) {
	r := &reloadingTokenSource{
		cur:    errTokenSource{},
		reload: func() (oauth2.TokenSource, error) { return nil, errors.New("ADC gone from disk") },
	}
	tok, err := r.Token()
	if err == nil {
		t.Fatalf("Token() returned nil error, want reload failure; token=%v", tok)
	}
	if tok != nil {
		t.Errorf("Token() = %v, want nil on reload failure", tok)
	}
}

// TestReloadingTokenSource_ConcurrentReload exercises the mutex under concurrent
// callers (run with -race). All callers must observe a successful token once the
// reload succeeds.
func TestReloadingTokenSource_ConcurrentReload(t *testing.T) {
	fresh := stubTokenSource{tok: &oauth2.Token{AccessToken: "fresh"}}
	var reloads atomic.Int32
	r := &reloadingTokenSource{
		cur: errTokenSource{},
		reload: func() (oauth2.TokenSource, error) {
			reloads.Add(1)
			return fresh, nil
		},
	}
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
	if got := reloads.Load(); got != 1 {
		t.Errorf("reload ran %d times, want exactly 1 (double-checked locking should serialize concurrent reloads)", got)
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

// TestNewReloadingDefaultTokenSource_CachesValidToken verifies that the
// oauth2.ReuseTokenSource wrapper caches a valid token, so the reload/inner
// path runs only on an actual mint failure and not on every request.
func TestNewReloadingDefaultTokenSource_CachesValidToken(t *testing.T) {
	initial := &countingTokenSource{tok: &oauth2.Token{AccessToken: "cached", Expiry: time.Now().Add(time.Hour)}}
	ts := newReloadingDefaultTokenSource(nil, initial)
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
