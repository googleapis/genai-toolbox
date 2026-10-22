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

package resources_test

import (
	"context"
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/resources"
)

func TestDynamicUIResource(t *testing.T) {
	called := false
	expectedHTML := "<html><body>Dynamic Content</body></html>"
	csp := &resources.CSPConfig{
		ConnectDomains:  []string{"https://api.example.com"},
		ResourceDomains: []string{"https://cdn.example.com"},
	}
	prefersBorder := true

	res := resources.NewDynamicUIResource(
		"test_dynamic_ui",
		"ui://test/dynamic.html",
		"Test dynamic UI resource",
		csp,
		&prefersBorder,
		func(ctx context.Context) (string, error) {
			called = true
			return expectedHTML, nil
		},
	)

	if res.GetName() != "test_dynamic_ui" {
		t.Errorf("Expected name 'test_dynamic_ui', got %q", res.GetName())
	}
	if res.GetURI() != "ui://test/dynamic.html" {
		t.Errorf("Expected URI 'ui://test/dynamic.html', got %q", res.GetURI())
	}
	if res.GetMimeType() != "text/html;profile=mcp-app" {
		t.Errorf("Expected mimeType 'text/html;profile=mcp-app', got %q", res.GetMimeType())
	}
	if res.GetSize() != nil {
		t.Errorf("Expected GetSize() to be nil, got %v", res.GetSize())
	}

	uiMeta := res.GetResourceUIMetadata()
	if uiMeta == nil {
		t.Fatalf("Expected GetResourceUIMetadata() to not be nil")
	}

	content, err := res.Read(context.Background(), nil)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if !called {
		t.Errorf("Expected readFunc to be called")
	}
	if content != expectedHTML {
		t.Errorf("Expected content %q, got %v", expectedHTML, content)
	}
}

func TestDynamicUIResource_Fallback(t *testing.T) {
	res := resources.NewDynamicUIResource(
		"fallback_ui",
		"ui://test/fallback.html",
		"Fallback test",
		nil,
		nil,
		func(ctx context.Context) (string, error) {
			return "", nil
		},
	)

	content, err := res.Read(context.Background(), nil)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if content != resources.DefaultBaseUIHTML {
		t.Errorf("Expected fallback DefaultBaseUIHTML, got %v", content)
	}
}
