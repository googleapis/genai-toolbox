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

package resources_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/resources"
)

func TestTextResource(t *testing.T) {
	ctx := context.Background()

	t.Run("basic text resource initialization and read", func(t *testing.T) {
		yamlStr := `
name: my-info
type: text
text: "Hello, this is static information."
description: "A test text resource"
title: "Test Resource"
`
		var raw map[string]any
		if err := yaml.Unmarshal([]byte(yamlStr), &raw); err != nil {
			t.Fatalf("failed to unmarshal yaml: %v", err)
		}
		
		cfg, err := resources.DecodeConfig(ctx, "text", "my-info", yaml.NewDecoder(strings.NewReader(yamlStr)))
		if err != nil {
			t.Fatalf("DecodeConfig failed: %v", err)
		}

		res, err := cfg.Initialize(ctx, "/some/dir", false)
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		if res.ResourceName() != "my-info" {
			t.Errorf("expected name %q, got %q", "my-info", res.ResourceName())
		}
		if res.ResourceURI() != "info://my-info" {
			t.Errorf("expected default URI %q, got %q", "info://my-info", res.ResourceURI())
		}
		if res.ResourceMimeType() != "text/plain" {
			t.Errorf("expected MIME type %q, got %q", "text/plain", res.ResourceMimeType())
		}

		// Verify default priority annotation
		ann := res.ResourceAnnotations()
		if ann == nil || ann.Priority == nil || *ann.Priority != 1.0 {
			t.Errorf("expected default priority annotation 1.0, got %v", ann)
		}

		contents, err := res.Read(ctx, nil)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}

		if len(contents) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(contents))
		}

		expectedContent := resources.ResourceContent{
			URI:      "info://my-info",
			MimeType: "text/plain",
			Text:     "Hello, this is static information.",
		}

		if diff := cmp.Diff(expectedContent, contents[0]); diff != "" {
			t.Errorf("ResourceContent mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("missing text parameter validation", func(t *testing.T) {
		yamlStr := `
name: invalid-info
type: text
`
		cfg, err := resources.DecodeConfig(ctx, "text", "invalid-info", yaml.NewDecoder(strings.NewReader(yamlStr)))
		if err != nil {
			t.Fatalf("DecodeConfig failed: %v", err)
		}

		_, err = cfg.Initialize(ctx, "/some/dir", false)
		if err == nil || !strings.Contains(err.Error(), "text is required") {
			t.Errorf("expected validation error for missing text, got: %v", err)
		}
	})
}

func TestFileResource(t *testing.T) {
	ctx := context.Background()

	// Setup a temporary directory acting as our workspace/sandbox
	tempDir := t.TempDir()
	
	// Create some test files inside the sandbox
	goodFilePath := filepath.Join(tempDir, "data.txt")
	if err := os.WriteFile(goodFilePath, []byte("This is safe content."), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	hiddenFilePath := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(hiddenFilePath, []byte("SECRET_KEY=12345"), 0644); err != nil {
		t.Fatalf("failed to write hidden test file: %v", err)
	}

	largeFilePath := filepath.Join(tempDir, "large.txt")
	largeData := make([]byte, 6*1024*1024) // 6MB (exceeds 5MB cap)
	if err := os.WriteFile(largeFilePath, largeData, 0644); err != nil {
		t.Fatalf("failed to write large test file: %v", err)
	}

	subDir := filepath.Join(tempDir, "logs")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create sub directory: %v", err)
	}
	templateFilePath := filepath.Join(subDir, "app.log")
	if err := os.WriteFile(templateFilePath, []byte("2026-06-24 Log trace"), 0644); err != nil {
		t.Fatalf("failed to write template test file: %v", err)
	}

	t.Run("basic file resource initialization and read", func(t *testing.T) {
		yamlStr := `
name: my-file
type: file
uri: "file://` + filepath.ToSlash(goodFilePath) + `"
description: "A static file resource"
`
		cfg, err := resources.DecodeConfig(ctx, "file", "my-file", yaml.NewDecoder(strings.NewReader(yamlStr)))
		if err != nil {
			t.Fatalf("DecodeConfig failed: %v", err)
		}

		res, err := cfg.Initialize(ctx, tempDir, false)
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		contents, err := res.Read(ctx, nil)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}

		if len(contents) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(contents))
		}

		if contents[0].Text != "This is safe content." {
			t.Errorf("expected content %q, got %q", "This is safe content.", contents[0].Text)
		}
	})

	t.Run("allowed paths validation on config directory mode", func(t *testing.T) {
		// When usingConfigFolder is true, allowedPaths MUST be defined
		yamlStr := `
name: my-file
type: file
uri: "file://` + filepath.ToSlash(goodFilePath) + `"
`
		cfg, err := resources.DecodeConfig(ctx, "file", "my-file", yaml.NewDecoder(strings.NewReader(yamlStr)))
		if err != nil {
			t.Fatalf("DecodeConfig failed: %v", err)
		}

		_, err = cfg.Initialize(ctx, tempDir, true) // usingConfigFolder = true
		if err == nil || !strings.Contains(err.Error(), "must explicitly define 'allowedPaths'") {
			t.Errorf("expected security error when allowedPaths is empty in config-folder mode, got: %v", err)
		}
	})

	t.Run("directory traversal path escape rejection", func(t *testing.T) {
		// Try to configure a file outside of the sandbox
		// Sandbox root is tempDir/logs
		sandboxRoot := subDir
		outsideFile := goodFilePath // outside tempDir/logs

		yamlStr := `
name: bad-file
type: file
uri: "file://` + filepath.ToSlash(outsideFile) + `"
`
		cfg, err := resources.DecodeConfig(ctx, "file", "bad-file", yaml.NewDecoder(strings.NewReader(yamlStr)))
		if err != nil {
			t.Fatalf("DecodeConfig failed: %v", err)
		}

		res, err := cfg.Initialize(ctx, sandboxRoot, false)
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		_, err = res.Read(ctx, nil)
		if err == nil || !strings.Contains(err.Error(), "escapes the allowed sandbox roots") {
			t.Errorf("expected path escape permission denied, got: %v", err)
		}
	})

	t.Run("hidden file block guardrail", func(t *testing.T) {
		yamlStr := `
name: secret-file
type: file
uri: "file://` + filepath.ToSlash(hiddenFilePath) + `"
`
		cfg, err := resources.DecodeConfig(ctx, "file", "secret-file", yaml.NewDecoder(strings.NewReader(yamlStr)))
		if err != nil {
			t.Fatalf("DecodeConfig failed: %v", err)
		}

		res, err := cfg.Initialize(ctx, tempDir, false)
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		_, err = res.Read(ctx, nil)
		if err == nil || !strings.Contains(err.Error(), "reading hidden files or directories is not allowed") {
			t.Errorf("expected hidden file block error, got: %v", err)
		}
	})

	t.Run("file size limit cap guardrail", func(t *testing.T) {
		yamlStr := `
name: huge-file
type: file
uri: "file://` + filepath.ToSlash(largeFilePath) + `"
`
		cfg, err := resources.DecodeConfig(ctx, "file", "huge-file", yaml.NewDecoder(strings.NewReader(yamlStr)))
		if err != nil {
			t.Fatalf("DecodeConfig failed: %v", err)
		}

		res, err := cfg.Initialize(ctx, tempDir, false)
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		_, err = res.Read(ctx, nil)
		if err == nil || !strings.Contains(err.Error(), "exceeds the maximum limit of 5MB") {
			t.Errorf("expected file size limit error, got: %v", err)
		}
	})

	t.Run("directory read rejection", func(t *testing.T) {
		yamlStr := `
name: dir-resource
type: file
uri: "file://` + filepath.ToSlash(tempDir) + `"
`
		cfg, err := resources.DecodeConfig(ctx, "file", "dir-resource", yaml.NewDecoder(strings.NewReader(yamlStr)))
		if err != nil {
			t.Fatalf("DecodeConfig failed: %v", err)
		}

		res, err := cfg.Initialize(ctx, tempDir, false)
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		_, err = res.Read(ctx, nil)
		if err == nil || !strings.Contains(err.Error(), "is a directory, not a file") {
			t.Errorf("expected directory rejection error, got: %v", err)
		}
	})

	t.Run("resource template parameter substitution", func(t *testing.T) {
		// URI contains {path}
		templateURI := filepath.Join(subDir, "{path}")
		yamlStr := `
name: logs-template
type: file
uri: "file://` + filepath.ToSlash(templateURI) + `"
`
		cfg, err := resources.DecodeConfig(ctx, "file", "logs-template", yaml.NewDecoder(strings.NewReader(yamlStr)))
		if err != nil {
			t.Fatalf("DecodeConfig failed: %v", err)
		}

		res, err := cfg.Initialize(ctx, subDir, false) // Sandbox root is subDir
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		// Read with parameter "path" = "app.log"
		params := map[string]any{"path": "app.log"}
		contents, err := res.Read(ctx, params)
		if err != nil {
			t.Fatalf("Template Read failed: %v", err)
		}

		if len(contents) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(contents))
		}
		if contents[0].Text != "2026-06-24 Log trace" {
			t.Errorf("expected content %q, got %q", "2026-06-24 Log trace", contents[0].Text)
		}

		// Read with parameter attempting directory traversal traversal escape
		badParams := map[string]any{"path": "../data.txt"}
		_, err = res.Read(ctx, badParams)
		if err == nil || !strings.Contains(err.Error(), "escapes the allowed sandbox roots") {
			t.Errorf("expected template path escape to be rejected, got: %v", err)
		}

		// Read with missing parameter
		_, err = res.Read(ctx, nil)
		if err == nil || !strings.Contains(err.Error(), "missing required parameter 'path'") {
			t.Errorf("expected missing parameter error, got: %v", err)
		}
	})
}
