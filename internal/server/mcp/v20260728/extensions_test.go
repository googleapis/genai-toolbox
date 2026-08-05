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

package v20260728

import (
	"testing"
)

const testExtURI = "com.google.cloud/test-extension"

func TestParseSupportedExtensions(t *testing.T) {
	orig := ServerExtensions
	t.Cleanup(func() {
		ServerExtensions = orig
	})
	tests := []struct {
		name        string
		extensions  map[string]any
		serverExts  []string
		expectedUri string
		expectedVal bool
	}{
		{
			name:        "nil map",
			extensions:  nil,
			serverExts:  nil,
			expectedUri: testExtURI,
			expectedVal: false,
		},
		{
			name: "enabled extension via empty settings object in extensions",
			extensions: map[string]any{
				testExtURI: map[string]any{},
			},
			serverExts:  nil,
			expectedUri: testExtURI,
			expectedVal: true,
		},
		{
			name: "enabled extension via settings object with values in extensions",
			extensions: map[string]any{
				testExtURI: map[string]any{"setting": "val"},
			},
			serverExts:  []string{testExtURI},
			expectedUri: testExtURI,
			expectedVal: true,
		},
		{
			name: "server does not support extension",
			extensions: map[string]any{
				testExtURI: map[string]any{},
			},
			serverExts:  []string{"other-extension"},
			expectedUri: testExtURI,
			expectedVal: false,
		},
		{
			name: "nil value ignored",
			extensions: map[string]any{
				testExtURI: nil,
			},
			serverExts:  nil,
			expectedUri: testExtURI,
			expectedVal: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			origSupported := SupportedExtensions
			t.Cleanup(func() {
				SupportedExtensions = origSupported
			})
			if tc.serverExts != nil {
				SupportedExtensions = tc.serverExts
			} else {
				SupportedExtensions = []string{testExtURI}
			}
			InitializeExtensions(nil)
			exts := ParseSupportedExtensions(tc.extensions)
			if SupportsExtension(exts, tc.expectedUri) != tc.expectedVal {
				t.Errorf("SupportsExtension() value for %s = %v, want %v", tc.expectedUri, SupportsExtension(exts, tc.expectedUri), tc.expectedVal)
			}
		})
	}
}

func TestServerExtensions(t *testing.T) {
	orig := ServerExtensions
	t.Cleanup(func() {
		ServerExtensions = orig
	})
	ServerExtensions = nil
	if ServerExtensions != nil {
		t.Errorf("expected nil when no server extensions registered")
	}

	ServerExtensions = map[string]any{testExtURI: map[string]any{}}
	if ServerExtensions == nil || ServerExtensions[testExtURI] == nil {
		t.Errorf("expected testExtURI to be registered in server extensions")
	}
}
