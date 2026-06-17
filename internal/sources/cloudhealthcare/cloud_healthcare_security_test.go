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

package cloudhealthcare

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateFHIRPageURL(t *testing.T) {
	tcs := []struct {
		desc    string
		pageURL string
		wantErr bool
	}{
		{"allowed global host", "https://healthcare.googleapis.com/v1/projects/p/locations/l/datasets/d/fhirStores/s/fhir/Patient?_page_token=abc", false},
		{"allowed mtls host", "https://healthcare.mtls.googleapis.com/v1/projects/p/fhir", false},
		{"attacker host", "https://attacker.example/steal", true},
		{"http scheme on valid host", "http://healthcare.googleapis.com/v1/fhir", true},
		{"metadata server", "http://169.254.169.254/computeMetadata/v1/", true},
		{"lookalike host", "https://healthcare.googleapis.com.attacker.example/", true},
		{"empty", "", true},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			err := validateFHIRPageURL(tc.pageURL)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for %q, got nil", tc.pageURL)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.pageURL, err)
			}
		})
	}
}

func TestFHIRFetchPageDoesNotLeakTokenToUntrustedHost(t *testing.T) {
	var hits int
	attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
	}))
	defer attacker.Close()

	s := &Source{Config: Config{UseClientOAuth: true}}
	_, err := s.FHIRFetchPage(context.Background(), attacker.URL, "ya29.secret-token")
	if err == nil {
		t.Fatal("expected FHIRFetchPage to reject untrusted host, got nil error")
	}
	if hits != 0 {
		t.Fatalf("untrusted host was contacted %d time(s); token may have leaked", hits)
	}
}
