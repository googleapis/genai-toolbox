// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lookerconversationalanalytics

import "testing"

func TestParseExploreReferences(t *testing.T) {
	const baseURL = "https://looker.example.com"

	t.Run("valid references are parsed", func(t *testing.T) {
		raw := []any{
			map[string]any{"model": "m1", "explore": "e1"},
			map[string]any{"model": "m2", "explore": "e2"},
		}
		got, err := parseExploreReferences(raw, baseURL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 references, got %d", len(got))
		}
		if got[0].LookmlModel != "m1" || got[0].Explore != "e1" || got[0].LookerInstanceUri != baseURL {
			t.Fatalf("unexpected first reference: %+v", got[0])
		}
	})

	t.Run("nil input yields an empty slice without error", func(t *testing.T) {
		got, err := parseExploreReferences(nil, baseURL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected no references, got %d", len(got))
		}
	})

	// A model that hallucinates the explore_references shape (a non-object element,
	// a missing field, or a non-string value) must surface a clean agent error
	// rather than panic the tool on an unchecked type assertion.
	errCases := []struct {
		desc string
		raw  []any
	}{
		{"element is not an object", []any{"not-a-map"}},
		{"missing model", []any{map[string]any{"explore": "e1"}}},
		{"model is not a string", []any{map[string]any{"model": 123, "explore": "e1"}}},
		{"missing explore", []any{map[string]any{"model": "m1"}}},
		{"explore is not a string", []any{map[string]any{"model": "m1", "explore": 7}}},
	}
	for _, tc := range errCases {
		t.Run(tc.desc, func(t *testing.T) {
			if _, err := parseExploreReferences(tc.raw, baseURL); err == nil {
				t.Fatalf("expected an error for %q, got nil", tc.desc)
			}
		})
	}
}
