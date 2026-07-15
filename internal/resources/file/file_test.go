package file

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/resources"
)

func TestFileResource_Validation(t *testing.T) {
	tmpDir := t.TempDir()

	exePath := filepath.Join(tmpDir, "test.exe")
	if err := os.WriteFile(exePath, []byte("run"), 0644); err != nil {
		t.Fatal(err)
	}

	dirPath := filepath.Join(tmpDir, "fake.txt")
	if err := os.Mkdir(dirPath, 0755); err != nil {
		t.Fatal(err)
	}

	secretPath := filepath.Join(tmpDir, "secret.exe")
	if err := os.WriteFile(secretPath, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	symlinkPath := filepath.Join(tmpDir, "symlink.txt")
	if err := os.Symlink(secretPath, symlinkPath); err != nil {
		t.Fatal(err)
	}

	validPath := filepath.Join(tmpDir, "valid.txt")
	if err := os.WriteFile(validPath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		yamlStr    string
		wantErrMsg string
		failDecode bool
	}{
		{
			name:       "missing path",
			yamlStr:    "type: file",
			wantErrMsg: "requires a 'path'",
		},
		{
			name:       "path traversal",
			yamlStr:    fmt.Sprintf("type: file\npath: %s", "../secrets.txt"),
			wantErrMsg: "relative path \"../secrets.txt\" is unsafe",
		},
		{
			name:       "invalid extension",
			yamlStr:    fmt.Sprintf("type: file\npath: %s", filepath.ToSlash(exePath)),
			wantErrMsg: "invalid extension",
		},
		{
			name:       "non-regular file directory",
			yamlStr:    fmt.Sprintf("type: file\npath: %s", filepath.ToSlash(dirPath)),
			wantErrMsg: "not a regular file",
		},
		{
			name:       "symlink evasion",
			yamlStr:    fmt.Sprintf("type: file\npath: %s", filepath.ToSlash(symlinkPath)),
			wantErrMsg: "invalid extension",
		},
		{
			name:       "max_size zero",
			yamlStr:    fmt.Sprintf("type: file\npath: %s\nmax_size: 0", filepath.ToSlash(validPath)),
			wantErrMsg: "must be greater than 0",
		},
		{
			name:       "max_size negative",
			yamlStr:    fmt.Sprintf("type: file\npath: %s\nmax_size: -50", filepath.ToSlash(validPath)),
			wantErrMsg: "must be greater than 0",
		},
		{
			name:       "max_size type string",
			yamlStr:    fmt.Sprintf("type: file\npath: %s\nmax_size: 50MB", filepath.ToSlash(validPath)),
			wantErrMsg: "cannot unmarshal",
			failDecode: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			decoder := yaml.NewDecoder(bytes.NewReader([]byte(tt.yamlStr)), yaml.Strict())
			cfg, err := resources.DecodeConfig(ctx, "file", "my-file", decoder)

			if tt.failDecode {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Fatalf("expected DecodeConfig to fail with %q, got err: %v", tt.wantErrMsg, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("DecodeConfig failed unexpectedly: %v", err)
			}

			_, err = cfg.Initialize(ctx)
			if err == nil || !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Fatalf("expected Initialize to fail with %q, got err: %v", tt.wantErrMsg, err)
			}
		})
	}
}

func TestFileResource_PathResolution(t *testing.T) {
	tmpDir := t.TempDir()

	absPath := filepath.Join(tmpDir, "abs.txt")
	if err := os.WriteFile(absPath, []byte("absolute data"), 0644); err != nil {
		t.Fatal(err)
	}

	relPath := "rel.txt"
	if err := os.WriteFile(filepath.Join(tmpDir, relPath), []byte("relative data"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		yamlStr  string
		expected string
	}{
		{
			name:     "absolute path",
			yamlStr:  fmt.Sprintf("type: file\npath: %s", filepath.ToSlash(absPath)),
			expected: "absolute data",
		},
		{
			name:     "relative path anchored to baseDir",
			yamlStr:  fmt.Sprintf("type: file\npath: %s", filepath.ToSlash(relPath)),
			expected: "relative data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), resources.BaseDirKey, tmpDir)
			decoder := yaml.NewDecoder(bytes.NewReader([]byte(tt.yamlStr)), yaml.Strict())
			cfg, err := resources.DecodeConfig(ctx, "file", "test", decoder)
			if err != nil {
				t.Fatalf("DecodeConfig failed: %v", err)
			}
			res, err := cfg.Initialize(ctx)
			if err != nil {
				t.Fatalf("Initialize failed: %v", err)
			}
			data, err := res.Read(ctx, nil)
			if err != nil {
				t.Fatalf("Read failed: %v", err)
			}
			if data.(string) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, data)
			}
		})
	}
}

func TestFileResource_Truncation(t *testing.T) {
	tmpDir := t.TempDir()

	size := defaultMaxFileSize + 100
	largeContent := make([]byte, size)
	for i := 0; i < size; i++ {
		largeContent[i] = 'A'
	}
	largePath := filepath.Join(tmpDir, "large.txt")
	if err := os.WriteFile(largePath, largeContent, 0644); err != nil {
		t.Fatal(err)
	}

	smallContent := make([]byte, 100)
	for i := 0; i < 100; i++ {
		smallContent[i] = 'B'
	}
	smallPath := filepath.Join(tmpDir, "small.txt")
	if err := os.WriteFile(smallPath, smallContent, 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		yamlStr   string
		wantSize  int64
		wantTrunc bool
	}{
		{
			name:      "default truncation 5MB",
			yamlStr:   fmt.Sprintf("type: file\npath: %s", filepath.ToSlash(largePath)),
			wantSize:  defaultMaxFileSize,
			wantTrunc: true,
		},
		{
			name:      "override max_size small",
			yamlStr:   fmt.Sprintf("type: file\npath: %s\nmax_size: 50", filepath.ToSlash(smallPath)),
			wantSize:  50,
			wantTrunc: true,
		},
		{
			name:      "no truncation needed",
			yamlStr:   fmt.Sprintf("type: file\npath: %s\nmax_size: 200", filepath.ToSlash(smallPath)),
			wantSize:  100, // Actual file is 100 bytes
			wantTrunc: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			decoder := yaml.NewDecoder(bytes.NewReader([]byte(tt.yamlStr)), yaml.Strict())
			cfg, err := resources.DecodeConfig(ctx, "file", "test", decoder)
			if err != nil {
				t.Fatalf("DecodeConfig failed: %v", err)
			}
			res, err := cfg.Initialize(ctx)
			if err != nil {
				t.Fatalf("Initialize failed: %v", err)
			}

			fileCfg := res.ToConfig().(*Config)
			if fileCfg.Size == nil || *fileCfg.Size != tt.wantSize {
				t.Errorf("expected size annotation %d, got %v", tt.wantSize, fileCfg.Size)
			}

			data, err := res.Read(ctx, nil)
			if err != nil {
				t.Fatalf("Read failed: %v", err)
			}

			strData := data.(string)
			hasWarning := strings.HasSuffix(strData, "safety limit]...")
			if hasWarning != tt.wantTrunc {
				t.Errorf("expected truncation warning %v, got %v", tt.wantTrunc, hasWarning)
			}
		})
	}
}

func TestFileResource_TOCTOU(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		swapAction func(path string)
	}{
		{
			name: "swap with symlink to binary",
			swapAction: func(path string) {
				target := filepath.Join(tmpDir, "malicious.exe")
				os.WriteFile(target, []byte("bad"), 0644)
				os.Remove(path)
				os.Symlink(target, path)
			},
		},
		{
			name: "swap with directory",
			swapAction: func(path string) {
				os.Remove(path)
				os.Mkdir(path, 0755)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a unique file for each test case
			filePath := filepath.Join(tmpDir, strings.ReplaceAll(tt.name, " ", "_")+".txt")
			if err := os.WriteFile(filePath, []byte("safe content"), 0644); err != nil {
				t.Fatal(err)
			}

			yamlStr := fmt.Sprintf("type: file\npath: %s", filepath.ToSlash(filePath))
			ctx := context.Background()
			decoder := yaml.NewDecoder(bytes.NewReader([]byte(yamlStr)), yaml.Strict())
			cfg, err := resources.DecodeConfig(ctx, "file", "test", decoder)
			if err != nil {
				t.Fatalf("DecodeConfig failed: %v", err)
			}

			res, err := cfg.Initialize(ctx)
			if err != nil {
				t.Fatalf("Initialize failed: %v", err)
			}

			// Execute malicious swap
			tt.swapAction(filePath)

			_, err = res.Read(ctx, nil)
			if err == nil {
				t.Fatalf("expected Read to fail after TOCTOU swap")
			}
			if !strings.Contains(err.Error(), "security violation") {
				t.Errorf("expected security violation error, got: %v", err)
			}
		})
	}
}

func TestFileResource_Metadata(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "notes.md")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	yamlStr := fmt.Sprintf("type: file\npath: %s", filepath.ToSlash(filePath))
	ctx := context.Background()
	decoder := yaml.NewDecoder(bytes.NewReader([]byte(yamlStr)), yaml.Strict())
	cfg, _ := resources.DecodeConfig(ctx, "file", "notes", decoder)
	res, err := cfg.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	fileCfg := res.ToConfig().(*Config)

	// Verify MIME type inference
	if !strings.HasPrefix(fileCfg.MimeType, "text/markdown") && !strings.HasPrefix(fileCfg.MimeType, "text/plain") && fileCfg.MimeType != "" {
		t.Errorf("expected reasonable MimeType, got: %v", fileCfg.MimeType)
	}

	// Verify LastModified
	if fileCfg.Annotations == nil || fileCfg.Annotations.LastModified == "" {
		t.Errorf("expected LastModified annotation to be set")
	} else {
		_, err := time.Parse(time.RFC3339, fileCfg.Annotations.LastModified)
		if err != nil {
			t.Errorf("expected valid RFC3339 LastModified, got %q", fileCfg.Annotations.LastModified)
		}
	}
}
