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

package pii

import (
	"testing"
)

func TestEngineMasking(t *testing.T) {
	config := PiiPolicyConfig{
		Name:        "test-policy",
		DefaultTier: ActionFull,
		Tiers: []Tier{
			{
				Name: "admin",
				MatchClaims: MatchClaims{
					"scope": []string{"admin"},
				},
				Action: ActionUnmask,
			},
			{
				Name: "analyst",
				MatchClaims: MatchClaims{
					"scope": []string{"analyst"},
				},
				Action: ActionPartial,
			},
		},
		Rules: []Rule{
			{
				Type:    "EMAIL",
				Pattern: `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
			},
			{
				Type:   "CREDIT_CARD",
				Column: "cc_number",
			},
		},
	}

	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	data := map[string]any{
		"user": map[string]any{
			"email":     "john.doe@example.com",
			"cc_number": "1234-5678-9012-3456",
			"name":      "John Doe",
		},
		"message": "Contact me at alice@wonderland.com for details.",
	}

	tests := []struct {
		name       string
		claims     map[string]map[string]any
		expectFunc func(any)
	}{
		{
			name: "Admin gets unmasked data",
			claims: map[string]map[string]any{
				"generic": {"scope": []string{"admin"}},
			},
			expectFunc: func(res any) {
				m := res.(map[string]any)
				if m["message"] != "Contact me at alice@wonderland.com for details." {
					t.Errorf("Expected unmasked message, got %v", m["message"])
				}
			},
		},
		{
			name: "Analyst gets partially masked data",
			claims: map[string]map[string]any{
				"generic": {"scope": []string{"analyst"}},
			},
			expectFunc: func(res any) {
				m := res.(map[string]any)
				expectedMsg := "Contact me at a***e@wonderland.com for details."
				if m["message"] != expectedMsg {
					t.Errorf("Expected %q, got %v", expectedMsg, m["message"])
				}
				user := m["user"].(map[string]any)
				expectedEmail := "j******e@example.com"
				if user["email"] != expectedEmail {
					t.Errorf("Expected %q, got %v", expectedEmail, user["email"])
				}
				expectedCC := "1*****************6"
				if user["cc_number"] != expectedCC {
					t.Errorf("Expected %q, got %v", expectedCC, user["cc_number"])
				}
			},
		},
		{
			name: "Unknown role gets fully masked data",
			claims: map[string]map[string]any{
				"generic": {"scope": []string{"agent"}},
			},
			expectFunc: func(res any) {
				m := res.(map[string]any)
				expectedMsg := "Contact me at [REDACTED:EMAIL] for details."
				if m["message"] != expectedMsg {
					t.Errorf("Expected %q, got %v", expectedMsg, m["message"])
				}
				user := m["user"].(map[string]any)
				if user["email"] != "[REDACTED:EMAIL]" {
					t.Errorf("Expected [REDACTED:EMAIL], got %v", user["email"])
				}
				if user["cc_number"] != "[REDACTED:CREDIT_CARD]" {
					t.Errorf("Expected [REDACTED:CREDIT_CARD], got %v", user["cc_number"])
				}
				if user["name"] != "John Doe" {
					t.Errorf("Expected unmodified name, got %v", user["name"])
				}
			},
		},
		{
			name: "Deny field action",
			claims: map[string]map[string]any{
				"generic": {"scope": []string{"untrusted"}},
			},
			expectFunc: func(res any) {
				// Let's modify the engine's config temporarily just for this test
				engine.Config.DefaultTier = ActionDenyField
				
				resMasked, _ := engine.Mask(data, map[string]map[string]any{})
				m := resMasked.(map[string]any)
				user := m["user"].(map[string]any)
				if _, exists := user["cc_number"]; exists {
					t.Errorf("Expected cc_number to be removed")
				}
				if user["email"] != "[REDACTED:EMAIL]" {
					// String pattern fields fallback to full mask on deny field
					t.Errorf("Expected [REDACTED:EMAIL], got %v", user["email"])
				}
				
				// Reset
				engine.Config.DefaultTier = ActionFull
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := engine.Mask(data, tc.claims)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			tc.expectFunc(res)
		})
	}
}

func TestPartialMask(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"john@example.com", "j**n@example.com"},
		{"a@b.com", "*@b.com"},
		{"123456789", "1*******9"},
		{"ab", "**"},
		{"a", "*"},
	}

	for _, tc := range tests {
		res := partialMask(tc.input)
		if res != tc.expected {
			t.Errorf("partialMask(%q) = %q; want %q", tc.input, res, tc.expected)
		}
	}
}
