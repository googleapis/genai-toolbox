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
	"testing"

	"google.golang.org/api/option"
)

func TestNormalizeAPIEndpoint(t *testing.T) {
	tcs := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"  ", ""},
		{"https://proxy.example.com", "https://proxy.example.com:443"},
		{"https://proxy.example.com/", "https://proxy.example.com:443"},
		{"http://proxy.example.com", "http://proxy.example.com:80"},
		{"http://localhost:9050", "http://localhost:9050"},
		{"proxy.example.com", "https://proxy.example.com:443"},
		{"proxy.example.com:8443", "https://proxy.example.com:8443"},
	}
	for _, tc := range tcs {
		t.Run(tc.in, func(t *testing.T) {
			if got := normalizeAPIEndpoint(tc.in); got != tc.want {
				t.Fatalf("normalizeAPIEndpoint(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestAppendAPIEndpointOption(t *testing.T) {
	base := []option.ClientOption{option.WithUserAgent("test")}

	if got := appendAPIEndpointOption(base, ""); len(got) != len(base) {
		t.Fatalf("empty endpoint: got %d options, want %d", len(got), len(base))
	}
	if got := appendAPIEndpointOption(base, "https://proxy.example.com"); len(got) != len(base)+1 {
		t.Fatalf("set endpoint: got %d options, want %d", len(got), len(base)+1)
	}
}
