// Copyright 2026 Google LLC
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
package util

import (
	"net/http"
	"testing"
)

func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{
			name:     "No headers",
			headers:  map[string]string{},
			expected: "",
		},
		{
			name: "Only X-Real-IP",
			headers: map[string]string{
				"X-Real-IP": "1.2.3.4",
			},
			expected: "1.2.3.4",
		},
		{
			name: "X-Real-IP with whitespace",
			headers: map[string]string{
				"X-Real-IP": "  1.2.3.4  ",
			},
			expected: "1.2.3.4",
		},
		{
			name: "Only X-Forwarded-For single IP",
			headers: map[string]string{
				"X-Forwarded-For": "1.2.3.4",
			},
			expected: "1.2.3.4",
		},
		{
			name: "Only X-Forwarded-For multiple IPs",
			headers: map[string]string{
				"X-Forwarded-For": "1.2.3.4, 5.6.7.8, 9.10.11.12",
			},
			expected: "1.2.3.4",
		},
		{
			name: "X-Forwarded-For starts with empty and whitespace",
			headers: map[string]string{
				"X-Forwarded-For": ", 1.2.3.4",
			},
			expected: "1.2.3.4",
		},
		{
			name: "X-Forwarded-For starts with multiple spaces",
			headers: map[string]string{
				"X-Forwarded-For": "   ,  , 1.2.3.4",
			},
			expected: "1.2.3.4",
		},
		{
			name: "X-Forwarded-For and X-Real-IP preferred X-Forwarded-For",
			headers: map[string]string{
				"X-Forwarded-For": "1.2.3.4",
				"X-Real-IP":       "5.6.7.8",
			},
			expected: "1.2.3.4",
		},
		{
			name: "X-Forwarded-For empty and X-Real-IP fallback",
			headers: map[string]string{
				"X-Forwarded-For": ", ",
				"X-Real-IP":       "5.6.7.8",
			},
			expected: "5.6.7.8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			actual := ExtractClientIP(req.Header)
			if actual != tt.expected {
				t.Errorf("ExtractClientIP() = %q, expected %q", actual, tt.expected)
			}
		})
	}
}

func TestSnakeFromCamelCase(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "displayName", want: "display_name"},
		{in: "ownerEmails", want: "owner_emails"},
		{in: "accessGroups", want: "access_groups"},
		{in: "description", want: "description"},
		{in: "display_name", want: "display_name"},
		{in: "a", want: "a"},
		{in: "A", want: "a"},
		{in: "locationId", want: "location_id"},
		{in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := SnakeFromCamelCase(tt.in)
			if got != tt.want {
				t.Errorf("SnakeFromCamelCase(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
