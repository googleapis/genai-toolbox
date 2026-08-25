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

package piipolicy

import (
	"context"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestApplyPolicy(t *testing.T) {
	ctx := context.Background()

	config := Config{
		Name:        "test_policy",
		DefaultTier: "guest",
		Tiers: []Tier{
			{
				Name: "admin",
				MatchClaims: map[string]string{
					"role": "admin",
				},
				Action: Unmask,
			},
			{
				Name: "user",
				MatchClaims: map[string]string{
					"role": "user",
				},
				Action: MaskPartial,
			},
			{
				Name:   "guest",
				Action: MaskFull,
			},
			{
				Name: "denied",
				MatchClaims: map[string]string{
					"role": "denied",
				},
				Action: DenyField,
			},
		},
		Rules: []Rule{
			{
				Pattern:       `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
				CompiledRegex: regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
			},
			{
				Column: "ssn",
			},
		},
	}

	tests := []struct {
		name       string
		claims     map[string]any
		data       any
		wantResult any
		wantErr    bool
	}{
		{
			name:       "unstructured_admin_unmask",
			claims:     map[string]any{"role": "admin"},
			data:       "Contact me at admin@example.com for more info.",
			wantResult: "Contact me at admin@example.com for more info.",
		},
		{
			name:       "unstructured_user_partial",
			claims:     map[string]any{"role": "user"},
			data:       "Contact me at test@example.com for more info.",
			wantResult: "Contact me at test@exa******** for more info.",
		},
		{
			name:       "unstructured_guest_full",
			claims:     map[string]any{"role": "guest"},
			data:       "Contact me at test@example.com.",
			wantResult: "Contact me at ****************.",
		},
		{
			name:   "structured_admin_unmask",
			claims: map[string]any{"role": "admin"},
			data: []map[string]any{
				{"name": "Alice", "ssn": "123-45-678", "password": "secret_password"},
			},
			wantResult: []map[string]any{
				{"name": "Alice", "ssn": "123-45-678", "password": "secret_password"},
			},
		},
		{
			name:   "structured_user_full_mask_ssn",
			claims: map[string]any{"role": "user"},
			data: []map[string]any{
				{"name": "Alice", "email": "alice@example.com", "ssn": "123-45-678", "password": "secret_password"},
			},
			wantResult: []map[string]any{
				{"name": "Alice", "email": "alice@ex*********", "ssn": "123-4*****", "password": "secret_password"},
			},
		},
		{
			name:   "structured_denied_field",
			claims: map[string]any{"role": "denied"},
			data: []map[string]any{
				{"name": "Alice", "ssn": "123-45-678"},
			},
			wantResult: []map[string]any{
				{"name": "Alice", "ssn": "[DENIED]"},
			},
		},
		{
			name:   "structured_nested_map",
			claims: map[string]any{"role": "user"},
			data: map[string]any{
				"metadata": map[string]any{
					"ssn": "987-65-432",
					"emails": []any{"nested@example.com"},
				},
			},
			wantResult: map[string]any{
				"metadata": map[string]any{
					"ssn": "987-6*****",
					"emails": []any{"nested@ex*********"},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ApplyPolicy(ctx, config, tc.claims, tc.data)
			if (err != nil) != tc.wantErr {
				t.Errorf("ApplyPolicy() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !cmp.Equal(got, tc.wantResult) {
				t.Errorf("ApplyPolicy() diff = %v", cmp.Diff(tc.wantResult, got))
			}
		})
	}
}
