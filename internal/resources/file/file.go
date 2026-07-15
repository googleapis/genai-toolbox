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

package file

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/resources"
)

const defaultMaxFileSize = 5 * 1024 * 1024 // 5MB

func init() {
	resources.Register("file", func(ctx context.Context, name string, decoder *yaml.Decoder) (resources.ResourceConfig, error) {
		var cfg Config
		if err := decoder.Decode(&cfg); err != nil {
			return nil, err
		}
		cfg.Name = name
		cfg.Type = "file"
		return &cfg, nil
	})
}

// Config represents the configuration for a file resource.
type Config struct {
	resources.BaseConfig `yaml:",inline"`
	Path                 string `yaml:"path"`
	MaxSize              *int64 `yaml:"max_size,omitempty"`

	absPath         string
	resolvedBaseDir string
	isRelative      bool
}

// ResourceConfigType returns the resource type identifier.
func (c *Config) ResourceConfigType() string {
	return "file"
}

var allowedExts = map[string]bool{
	".txt": true, ".md": true, ".csv": true, ".json": true,
	".yaml": true, ".yml": true, ".xml": true, ".sql": true,
}

func validateExtension(path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	if !allowedExts[ext] {
		return fmt.Errorf("file extension %q is not allowed", ext)
	}
	return nil
}

// Initialize validates the configuration and initializes the file resource.
func (c *Config) Initialize(ctx context.Context) (resources.Resource, error) {
	if c.Path == "" {
		return nil, fmt.Errorf("file resource %q requires a 'path'", c.Name)
	}

	if c.MaxSize == nil {
		limit := int64(defaultMaxFileSize)
		c.MaxSize = &limit
	} else if *c.MaxSize <= 0 {
		return nil, fmt.Errorf("file resource %q max_size must be greater than 0", c.Name)
	} else if *c.MaxSize > 1024*1024*1024 {
		return nil, fmt.Errorf("file resource %q max_size cannot exceed 1GB", c.Name)
	}

	if filepath.IsAbs(c.Path) {
		c.absPath = filepath.Clean(c.Path)
		c.isRelative = false
	} else {
		if !filepath.IsLocal(c.Path) {
			return nil, fmt.Errorf("relative path %q is unsafe", c.Path)
		}
		c.isRelative = true
		baseDir := resources.GetBaseDirFromContext(ctx)
		if baseDir == "" {
			baseDir = "."
		}
		c.resolvedBaseDir = baseDir
		c.absPath = filepath.Clean(filepath.Join(baseDir, c.Path))
	}

	abs, err := filepath.Abs(c.absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %q: %w", c.absPath, err)
	}
	c.absPath = abs

	if c.Annotations == nil {
		c.Annotations = &resources.ResourceAnnotations{}
	}

	if c.MimeType == "" {
		ext := strings.ToLower(filepath.Ext(c.absPath))
		c.MimeType = mime.TypeByExtension(ext)
		if c.MimeType == "" {
			c.MimeType = "text/plain"
		}
	}

	if c.isRelative && c.resolvedBaseDir != "" {
		absBase, err := filepath.Abs(c.resolvedBaseDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for base directory: %w", err)
		}
		resolvedBase, err := filepath.EvalSymlinks(absBase)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to evaluate symlinks for base directory: %w", err)
			}
			c.resolvedBaseDir = absBase
		} else {
			c.resolvedBaseDir = resolvedBase
		}
	}

	resolvedPath, err := filepath.EvalSymlinks(c.absPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := validateExtension(c.absPath); err != nil {
				return nil, fmt.Errorf("invalid extension for resource %q: %w", c.Name, err)
			}
			return &FileResource{config: c}, nil
		}
		return nil, fmt.Errorf("failed to evaluate symlinks for resource %q: %w", c.Name, err)
	}
	c.absPath = resolvedPath

	if c.isRelative && c.resolvedBaseDir != "" {
		rel, err := filepath.Rel(c.resolvedBaseDir, c.absPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("security violation: resolved path %q escapes base directory %q", c.absPath, c.resolvedBaseDir)
		}
	}

	if err := validateExtension(c.absPath); err != nil {
		return nil, fmt.Errorf("invalid extension for resource %q: %w", c.Name, err)
	}

	info, err := os.Stat(c.absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %q for resource %q: %w", c.absPath, c.Name, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("path %q for resource %q is not a regular file (devices, pipes, sockets are blocked)", c.absPath, c.Name)
	}

	return &FileResource{
		config: c,
	}, nil
}

// FileResource handles reading content from a local file.
type FileResource struct {
	config *Config
}

// Read retrieves the file content.
func (r *FileResource) Read(ctx context.Context, params map[string]any) (any, error) {
	resolvedPath, err := filepath.EvalSymlinks(r.config.absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate symlinks for resource %q at runtime: %w", r.config.Name, err)
	}

	if r.config.isRelative && r.config.resolvedBaseDir != "" {
		resolvedBaseDir := r.config.resolvedBaseDir
		if resolved, err := filepath.EvalSymlinks(resolvedBaseDir); err == nil {
			resolvedBaseDir = resolved
		}
		rel, err := filepath.Rel(resolvedBaseDir, resolvedPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("security violation: resolved path %q escapes base directory %q at runtime", resolvedPath, resolvedBaseDir)
		}
	}

	if err := validateExtension(resolvedPath); err != nil {
		return nil, fmt.Errorf("security violation: file extension changed post-boot for resource %q: %w", r.config.Name, err)
	}

	statInfo, err := os.Lstat(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to lstat file %q: %w", resolvedPath, err)
	}

	if statInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("security violation: TOCTOU symlink swap detected on %q", resolvedPath)
	}

	f, err := os.Open(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", resolvedPath, err)
	}
	defer f.Close()

	openInfo, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat opened file %q: %w", resolvedPath, err)
	}

	if !os.SameFile(statInfo, openInfo) {
		return nil, fmt.Errorf("security violation: TOCTOU file swap detected on %q between stat and open", resolvedPath)
	}

	if !openInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("security violation: file %q was swapped with a non-regular file during read", resolvedPath)
	}

	limit := *r.config.MaxSize
	limitedReader := io.LimitReader(f, limit+1)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", resolvedPath, err)
	}

	if int64(len(content)) > limit {
		truncated := content[:limit]
		for len(truncated) > 0 {
			r, size := utf8.DecodeLastRune(truncated)
			if r == utf8.RuneError && size == 1 {
				truncated = truncated[:len(truncated)-1]
			} else {
				break
			}
		}
		warning := fmt.Sprintf("\n\n...[TRUNCATED BY SERVER: Payload exceeded %d byte safety limit]...", limit)
		return string(truncated) + warning, nil
	}

	return string(content), nil
}

// ToConfig returns the runtime config struct back to the caller.
func (r *FileResource) ToConfig() resources.ResourceConfig {
	cfgCopy := *r.config

	if r.config.Annotations != nil {
		ann := *r.config.Annotations
		cfgCopy.Annotations = &ann
	} else {
		cfgCopy.Annotations = &resources.ResourceAnnotations{}
	}

	resolvedPath := r.config.absPath
	if resolved, err := filepath.EvalSymlinks(r.config.absPath); err == nil {
		resolvedPath = resolved
	}

	if r.config.isRelative && r.config.resolvedBaseDir != "" {
		resolvedBaseDir := r.config.resolvedBaseDir
		if resolved, err := filepath.EvalSymlinks(resolvedBaseDir); err == nil {
			resolvedBaseDir = resolved
		}
		rel, err := filepath.Rel(resolvedBaseDir, resolvedPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return &cfgCopy
		}
	}

	if info, err := os.Stat(resolvedPath); err == nil {
		size := info.Size()
		if size > *cfgCopy.MaxSize {
			size = *cfgCopy.MaxSize
		}
		cfgCopy.Size = &size
		cfgCopy.Annotations.LastModified = info.ModTime().Format(time.RFC3339)
	}

	return &cfgCopy
}
