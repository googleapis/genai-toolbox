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

package v20260728

import (
	"context"
	"testing"
)

const testExtURI = "com.google.cloud/test-extension"

type mockCapabilities struct {
	extensions   map[string]any
	experimental map[string]any
}

func (m *mockCapabilities) GetExtensions() map[string]any {
	return m.extensions
}

func (m *mockCapabilities) GetExperimental() map[string]any {
	return m.experimental
}

func TestClientExtensions(t *testing.T) {
	ctx := context.Background()

	// Context without extensions attached
	if SupportsExtension(ctx, testExtURI) {
		t.Errorf("expected false when no extensions attached")
	}

	// Attach extensions map
	exts := ClientExtensions{
		testExtURI: true,
	}
	ctx = WithClientExtensions(ctx, exts)

	if !SupportsExtension(ctx, testExtURI) {
		t.Errorf("expected true for testExtURI")
	}

	if SupportsExtension(ctx, "com.google.cloud/unsupported") {
		t.Errorf("expected false for unsupported extension")
	}
}

func TestExtractClientExtensions(t *testing.T) {
	tests := []struct {
		name         string
		extensions   map[string]any
		experimental map[string]any
		expectedUri  string
		expectedVal  bool
	}{
		{
			name:         "nil maps",
			extensions:   nil,
			experimental: nil,
			expectedUri:  testExtURI,
			expectedVal:  false,
		},
		{
			name:       "enabled extension via boolean in experimental",
			extensions: nil,
			experimental: map[string]any{
				testExtURI: true,
			},
			expectedUri: testExtURI,
			expectedVal: true,
		},
		{
			name:       "disabled extension via boolean in experimental",
			extensions: nil,
			experimental: map[string]any{
				testExtURI: false,
			},
			expectedUri: testExtURI,
			expectedVal: false,
		},
		{
			name: "enabled extension via empty settings object in extensions",
			extensions: map[string]any{
				testExtURI: map[string]any{},
			},
			experimental: nil,
			expectedUri:  testExtURI,
			expectedVal:  true,
		},
		{
			name: "enabled extension via settings object with values in extensions",
			extensions: map[string]any{
				testExtURI: map[string]any{"setting": "val"},
			},
			experimental: nil,
			expectedUri:  testExtURI,
			expectedVal:  true,
		},
		{
			name: "nil value ignored",
			extensions: map[string]any{
				testExtURI: nil,
			},
			experimental: nil,
			expectedUri:  testExtURI,
			expectedVal:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exts := ExtractClientExtensions(tc.extensions, tc.experimental)
			if exts[tc.expectedUri] != tc.expectedVal {
				t.Errorf("ExtractClientExtensions() value for %s = %v, want %v", tc.expectedUri, exts[tc.expectedUri], tc.expectedVal)
			}
		})
	}
}

func TestWithClientCapabilities(t *testing.T) {
	ctx := context.Background()

	// nil capabilities
	ctxNil := WithClientCapabilities(ctx, nil)
	if SupportsExtension(ctxNil, testExtURI) {
		t.Errorf("expected false for nil capabilities")
	}

	// mock capabilities with standard extension object
	caps := &mockCapabilities{
		extensions: map[string]any{
			testExtURI: map[string]any{},
		},
	}
	ctxCaps := WithClientCapabilities(ctx, caps)
	if !SupportsExtension(ctxCaps, testExtURI) {
		t.Errorf("expected true for testExtURI via WithClientCapabilities")
	}
}

func TestServerExtensions(t *testing.T) {
	if GetServerExtensions() != nil {
		t.Errorf("expected nil when no server extensions registered")
	}
	if GetServerExperimental() != nil {
		t.Errorf("expected nil when no server experimental extensions registered")
	}

	RegisterServerExtension(testExtURI, map[string]any{})
	RegisterServerExperimental(testExtURI, true)

	exts := GetServerExtensions()
	if exts == nil || exts[testExtURI] == nil {
		t.Errorf("expected testExtURI to be registered in server extensions")
	}

	exp := GetServerExperimental()
	if exp == nil || exp[testExtURI] != true {
		t.Errorf("expected testExtURI to be registered in server experimental")
	}
}
