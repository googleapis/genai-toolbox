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

package file_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/resources"
	"github.com/googleapis/mcp-toolbox/internal/resources/file"
)

// TestFileResource_Validation verifies that the file resource correctly validates
// configurations at boot and runtime, blocking invalid paths, missing fields,
// negative size limits, directory evasions, and non-allowed file extensions.
func TestFileTemplate_AllowedPaths(t *testing.T) {
	tmpDir := t.TempDir()

	allowedDir := filepath.Join(tmpDir, "allowed")
	if err := os.Mkdir(allowedDir, 0755); err != nil {
		t.Fatal(err)
	}

	forbiddenDir := filepath.Join(tmpDir, "forbidden")
	if err := os.Mkdir(forbiddenDir, 0755); err != nil {
		t.Fatal(err)
	}

	allowedFile := filepath.Join(allowedDir, "test.txt")
	if err := os.WriteFile(allowedFile, []byte("allowed content"), 0644); err != nil {
		t.Fatal(err)
	}

	forbiddenFile := filepath.Join(forbiddenDir, "secret.txt")
	if err := os.WriteFile(forbiddenFile, []byte("secret content"), 0644); err != nil {
		t.Fatal(err)
	}

	yamlStr := fmt.Sprintf("uriTemplate: file://{path}\nallowedPaths:\n  - %s\n", filepath.ToSlash(allowedDir))
	ctx := context.Background()
	decoder := yaml.NewDecoder(bytes.NewReader([]byte(yamlStr)), yaml.Strict())
	cfg, err := resources.DecodeTemplateConfig(ctx, "file", "test_template", decoder)
	if err != nil {
		t.Fatalf("DecodeTemplateConfig failed: %v", err)
	}

	resTmpl, err := cfg.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test reading an allowed file
	content, err := resTmpl.Read(ctx, map[string]any{"path": allowedFile})
	if err != nil {
		t.Fatalf("expected to read allowed file, got error: %v", err)
	}
	if content.(string) != "allowed content" {
		t.Errorf("unexpected content: %v", content)
	}

	// Test reading a forbidden file
	_, err = resTmpl.Read(ctx, map[string]any{"path": forbiddenFile})
	if err == nil {
		t.Fatal("expected error when reading forbidden file, got none")
	}
	if !strings.Contains(err.Error(), "security violation") {
		t.Errorf("expected security violation error, got: %v", err)
	}

	// Test directory traversal attempt
	traversalPath := filepath.Join(allowedDir, "..", "forbidden", "secret.txt")
	_, err = resTmpl.Read(ctx, map[string]any{"path": traversalPath})
	if err == nil {
		t.Fatal("expected error when reading via directory traversal, got none")
	}
	if !strings.Contains(err.Error(), "security violation") {
		t.Errorf("expected security violation error, got: %v", err)
	}

	// Test hidden file success (allowed since we explicitly trust allowedPaths)
	hiddenFile := filepath.Join(allowedDir, ".hidden.txt")
	if err := os.WriteFile(hiddenFile, []byte("hidden content"), 0644); err != nil {
		t.Fatal(err)
	}
	content, err = resTmpl.Read(ctx, map[string]any{"path": hiddenFile})
	if err != nil {
		t.Fatalf("unexpected error when reading hidden file in allowed path: %v", err)
	}
	if content.(string) != "hidden content" {
		t.Errorf("expected 'hidden content', got: %s", content)
	}
}

func TestFileTemplate_SymlinkEscape(t *testing.T) {
	tmpDir := t.TempDir()

	allowedDir := filepath.Join(tmpDir, "allowed")
	if err := os.Mkdir(allowedDir, 0755); err != nil {
		t.Fatal(err)
	}

	forbiddenDir := filepath.Join(tmpDir, "forbidden")
	if err := os.Mkdir(forbiddenDir, 0755); err != nil {
		t.Fatal(err)
	}

	secretFile := filepath.Join(forbiddenDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("secret content"), 0644); err != nil {
		t.Fatal(err)
	}

	symlinkPath := filepath.Join(allowedDir, "link.txt")
	if err := os.Symlink(secretFile, symlinkPath); err != nil {
		t.Skipf("symlink creation failed: %v", err)
	}

	yamlStr := fmt.Sprintf("uriTemplate: file://{path}\nallowedPaths:\n  - %s\n", filepath.ToSlash(allowedDir))
	ctx := context.Background()
	decoder := yaml.NewDecoder(bytes.NewReader([]byte(yamlStr)), yaml.Strict())
	cfg, err := resources.DecodeTemplateConfig(ctx, "file", "test_template", decoder)
	if err != nil {
		t.Fatalf("DecodeTemplateConfig failed: %v", err)
	}

	resTmpl, err := cfg.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	_, err = resTmpl.Read(ctx, map[string]any{"path": symlinkPath})
	if err == nil {
		t.Fatal("expected error when reading via symlink escape, got none")
	}
	if !strings.Contains(err.Error(), "security violation: path") && !strings.Contains(err.Error(), "not within any allowedPaths") {
		t.Errorf("expected security violation error, got: %v", err)
	}
}

func TestFileTemplate_ExtensionValidation(t *testing.T) {
	tmpDir := t.TempDir()

	binFile := filepath.Join(tmpDir, "image.png")
	if err := os.WriteFile(binFile, []byte("fake png content"), 0644); err != nil {
		t.Fatal(err)
	}

	yamlStr := fmt.Sprintf("uriTemplate: file://{path}\nallowedPaths:\n  - %s\n", filepath.ToSlash(tmpDir))
	ctx := context.Background()
	decoder := yaml.NewDecoder(bytes.NewReader([]byte(yamlStr)), yaml.Strict())
	cfg, err := resources.DecodeTemplateConfig(ctx, "file", "test_template", decoder)
	if err != nil {
		t.Fatalf("DecodeTemplateConfig failed: %v", err)
	}

	resTmpl, err := cfg.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	_, err = resTmpl.Read(ctx, map[string]any{"path": binFile})
	if err == nil {
		t.Fatal("expected error when reading binary file, got none")
	}
	if !strings.Contains(err.Error(), "file extension not allowed") {
		t.Errorf("expected extension validation error, got: %v", err)
	}
}

func TestFileTemplate_FileSizeLimit(t *testing.T) {
	tmpDir := t.TempDir()

	largeFile := filepath.Join(tmpDir, "large.txt")
	// defaultMaxFileSize is 10MB. We'll write 10MB + 10 bytes
	size := 10*1024*1024 + 10
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		data[i] = 'a'
	}
	if err := os.WriteFile(largeFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	yamlStr := fmt.Sprintf("uriTemplate: file://{path}\nallowedPaths:\n  - %s\n", filepath.ToSlash(tmpDir))
	ctx := context.Background()
	decoder := yaml.NewDecoder(bytes.NewReader([]byte(yamlStr)), yaml.Strict())
	cfg, err := resources.DecodeTemplateConfig(ctx, "file", "test_template", decoder)
	if err != nil {
		t.Fatalf("DecodeTemplateConfig failed: %v", err)
	}

	resTmpl, err := cfg.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	content, err := resTmpl.Read(ctx, map[string]any{"path": largeFile})
	if err != nil {
		t.Fatalf("expected to read large file, got error: %v", err)
	}
	contentStr := content.(string)

	if len(contentStr) > 10*1024*1024+200 {
		t.Errorf("expected content to be truncated, but size is %d", len(contentStr))
	}
	if !strings.Contains(contentStr, "[TRUNCATED BY SERVER") {
		t.Errorf("expected truncation warning, got %q", contentStr[len(contentStr)-200:])
	}
}

func TestFileTemplate_ToConfigAndType(t *testing.T) {
	cfg := file.TemplateConfig{}
	if cfg.ResourceTemplateConfigType() != "file" {
		t.Errorf("expected ResourceTemplateConfigType to be 'file', got %q", cfg.ResourceTemplateConfigType())
	}

	r, _ := cfg.Initialize(context.Background())
	res := r.(*file.FileTemplate)
	if res.ToConfig() == nil {
		t.Errorf("expected ToConfig to return a valid config")
	}
}

func TestFileTemplate_HiddenFilesWhenNoAllowedPaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a visible file
	visibleFile := filepath.Join(tmpDir, "visible.txt")
	if err := os.WriteFile(visibleFile, []byte("visible"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a hidden file
	hiddenFile := filepath.Join(tmpDir, ".hidden.txt")
	if err := os.WriteFile(hiddenFile, []byte("hidden"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a file inside a hidden directory
	hiddenDir := filepath.Join(tmpDir, ".secrets")
	if err := os.Mkdir(hiddenDir, 0755); err != nil {
		t.Fatal(err)
	}
	hiddenDirFile := filepath.Join(hiddenDir, "data.txt")
	if err := os.WriteFile(hiddenDirFile, []byte("hidden dir"), 0644); err != nil {
		t.Fatal(err)
	}

	// NO allowedPaths specified
	yamlStr := "uriTemplate: file://{path}\n"
	ctx := context.Background()
	decoder := yaml.NewDecoder(bytes.NewReader([]byte(yamlStr)), yaml.Strict())
	cfg, err := resources.DecodeTemplateConfig(ctx, "file", "test", decoder)
	if err != nil {
		t.Fatalf("DecodeTemplateConfig failed: %v", err)
	}
	resTmpl, err := cfg.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Visible file should succeed
	_, err = resTmpl.Read(ctx, map[string]any{"path": visibleFile})
	if err != nil {
		t.Fatalf("Expected visible file to succeed, got: %v", err)
	}

	// Hidden file should fail
	_, err = resTmpl.Read(ctx, map[string]any{"path": hiddenFile})
	if err == nil || !strings.Contains(err.Error(), "security violation: access to hidden file") {
		t.Fatalf("Expected hidden file to fail with security violation, got: %v", err)
	}

	// File inside hidden dir should fail
	_, err = resTmpl.Read(ctx, map[string]any{"path": hiddenDirFile})
	if err == nil || !strings.Contains(err.Error(), "security violation: access to hidden file") {
		t.Fatalf("Expected hidden dir file to fail with security violation, got: %v", err)
	}
}

func TestFileTemplate_RelativePathMiddleURITemplate(t *testing.T) {
	tmpDir := t.TempDir()

	// Target file: <tmpDir>/logs/server1/data.txt
	serverDir := filepath.Join(tmpDir, "logs", "server1")
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(serverDir, "data.txt")
	if err := os.WriteFile(targetFile, []byte("server logs"), 0644); err != nil {
		t.Fatal(err)
	}

	// Convert tmpDir to a slash path for the URI template
	tmpDirSlash := filepath.ToSlash(tmpDir)
	if !strings.HasPrefix(tmpDirSlash, "/") {
		tmpDirSlash = "/" + tmpDirSlash
	}
	uriTemplate := fmt.Sprintf("file://%s/logs/{path}/data.txt", tmpDirSlash)

	yamlStr := fmt.Sprintf("uriTemplate: %s\n", uriTemplate)
	ctx := context.Background()
	decoder := yaml.NewDecoder(bytes.NewReader([]byte(yamlStr)), yaml.Strict())
	cfg, err := resources.DecodeTemplateConfig(ctx, "file", "test", decoder)
	if err != nil {
		t.Fatalf("DecodeTemplateConfig failed: %v", err)
	}
	resTmpl, err := cfg.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// We pass "server1" as the path parameter.
	content, err := resTmpl.Read(ctx, map[string]any{"path": "server1"})
	if err != nil {
		t.Fatalf("Expected to read file, got error: %v", err)
	}
	if content.(string) != "server logs" {
		t.Errorf("Expected 'server logs', got %q", content)
	}
}
