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

package util

import (
	"context"
	"testing"
)

func TestClientExtensions(t *testing.T) {
	ctx := context.Background()

	// Context without extensions attached
	if SupportsExtension(ctx, ExtSecureParams) {
		t.Errorf("expected false when no extensions attached")
	}

	// Attach extensions map
	exts := ClientExtensions{
		ExtSecureParams: true,
	}
	ctx = WithClientExtensions(ctx, exts)

	if !SupportsExtension(ctx, ExtSecureParams) {
		t.Errorf("expected true for ExtSecureParams")
	}

	if SupportsExtension(ctx, "com.google.cloud/unsupported") {
		t.Errorf("expected false for unsupported extension")
	}
}

func TestExtractClientExtensions(t *testing.T) {
	tests := []struct {
		name         string
		experimental map[string]any
		expectedUri  string
		expectedVal  bool
	}{
		{
			name:         "nil experimental map",
			experimental: nil,
			expectedUri:  ExtSecureParams,
			expectedVal:  false,
		},
		{
			name: "enabled extension",
			experimental: map[string]any{
				ExtSecureParams: true,
			},
			expectedUri: ExtSecureParams,
			expectedVal: true,
		},
		{
			name: "disabled extension",
			experimental: map[string]any{
				ExtSecureParams: false,
			},
			expectedUri: ExtSecureParams,
			expectedVal: false,
		},
		{
			name: "non-bool value ignored",
			experimental: map[string]any{
				ExtSecureParams: "true",
			},
			expectedUri: ExtSecureParams,
			expectedVal: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exts := ExtractClientExtensions(tc.experimental)
			if exts[tc.expectedUri] != tc.expectedVal {
				t.Errorf("ExtractClientExtensions() value for %s = %v, want %v", tc.expectedUri, exts[tc.expectedUri], tc.expectedVal)
			}
		})
	}
}
