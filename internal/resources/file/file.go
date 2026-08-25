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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/resources"
)

const (
	defaultMaxFileSize = 5 * 1024 * 1024 // 5MB
	resourceType       = "file"
)

func init() {
	if !resources.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("resource type %q already registered", resourceType))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (resources.ResourceConfig, error) {
	cfg := &Config{
		BaseConfig: resources.BaseConfig{
			Name: name,
			Type: resourceType,
			URI:  fmt.Sprintf("file://%s", name),
		},
	}
	if err := decoder.DecodeContext(ctx, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Config represents the configuration for a file resource.
type Config struct {
	resources.BaseConfig `yaml:",inline"`
	Path                 string `yaml:"path" validate:"required"`
	MaxSize              *int64 `yaml:"max_size,omitempty"`
}

var _ resources.ResourceConfig = &Config{}
var _ resources.Resource = &FileResource{}

// ResourceConfigType returns the resource type identifier.
func (c *Config) ResourceConfigType() string {
	return resourceType
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

func (c *Config) Validate() error {
	if err := c.BaseConfig.Validate(); err != nil {
		return err
	}
	parsed, _ := url.Parse(c.URI)
	if parsed.Scheme != "file" {
		return fmt.Errorf("invalid scheme for file resource %q: must be 'file'", c.Name)
	}

	if c.MaxSize != nil {
		if *c.MaxSize <= 0 {
			return fmt.Errorf("file resource %q max_size must be greater than 0", c.Name)
		} else if *c.MaxSize > 1024*1024*1024 {
			return fmt.Errorf("file resource %q max_size cannot exceed 1GB", c.Name)
		}
	}
	return nil
}

// Initialize validates the configuration and initializes the file resource.
func (c *Config) Initialize(ctx context.Context) (resources.Resource, error) {
	if c.MaxSize == nil {
		limit := int64(defaultMaxFileSize)
		c.MaxSize = &limit
	}

	var absPath string
	var resolvedBaseDir string
	var isRelative bool

	if filepath.IsAbs(c.Path) {
		absPath = filepath.Clean(c.Path)
		isRelative = false
	} else {
		if !filepath.IsLocal(c.Path) {
			return nil, fmt.Errorf("relative path %q is unsafe", c.Path)
		}
		isRelative = true
		baseDir := resources.GetBaseDirFromContext(ctx)
		if baseDir == "" {
			baseDir = "."
		}
		resolvedBaseDir = baseDir
		absPath = filepath.Clean(filepath.Join(baseDir, c.Path))
	}

	abs, err := filepath.Abs(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %q: %w", absPath, err)
	}
	absPath = abs

	if c.Annotations == nil {
		c.Annotations = &resources.ResourceAnnotations{}
	}

	if c.MimeType == "" {
		ext := strings.ToLower(filepath.Ext(absPath))
		c.MimeType = mime.TypeByExtension(ext)
		if c.MimeType == "" {
			c.MimeType = "text/plain"
		}
	}

	if isRelative && resolvedBaseDir != "" {
		absBase, err := filepath.Abs(resolvedBaseDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for base directory: %w", err)
		}
		resolvedBase, err := filepath.EvalSymlinks(absBase)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to evaluate symlinks for base directory: %w", err)
			}
			resolvedBaseDir = absBase
		} else {
			resolvedBaseDir = resolvedBase
		}
	}

	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := validateExtension(absPath); err != nil {
				return nil, fmt.Errorf("invalid extension for resource %q: %w", c.Name, err)
			}
			return &FileResource{
				Config:          *c,
				absPath:         absPath,
				resolvedBaseDir: resolvedBaseDir,
				isRelative:      isRelative,
			}, nil
		}
		return nil, fmt.Errorf("failed to evaluate symlinks for resource %q: %w", c.Name, err)
	}
	absPath = resolvedPath

	if isRelative && resolvedBaseDir != "" {
		rel, err := filepath.Rel(resolvedBaseDir, absPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("security violation: resolved path %q escapes base directory %q", absPath, resolvedBaseDir)
		}
	}

	if err := validateExtension(absPath); err != nil {
		return nil, fmt.Errorf("invalid extension for resource %q: %w", c.Name, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %q for resource %q: %w", absPath, c.Name, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("path %q for resource %q is not a regular file (devices, pipes, sockets are blocked)", absPath, c.Name)
	}

	size := info.Size()
	if size > *c.MaxSize {
		size = *c.MaxSize
	}
	return &FileResource{
		Config:          *c,
		Size:            size,
		absPath:         absPath,
		resolvedBaseDir: resolvedBaseDir,
		isRelative:      isRelative,
	}, nil
}

// FileResource handles reading content from a local file.
type FileResource struct {
	Config
	Size int64

	absPath         string
	resolvedBaseDir string
	isRelative      bool
}

func (r *FileResource) GetSize() *int64 {
	size := r.Size
	return &size
}

// Read retrieves the file content.
func (r *FileResource) Read(ctx context.Context, params map[string]any) (any, error) {
	resolvedPath, err := filepath.EvalSymlinks(r.absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate symlinks for resource %q at runtime: %w", r.Config.Name, err)
	}

	if r.isRelative && r.resolvedBaseDir != "" {
		resolvedBaseDir := r.resolvedBaseDir
		if resolved, err := filepath.EvalSymlinks(resolvedBaseDir); err == nil {
			resolvedBaseDir = resolved
		}
		rel, err := filepath.Rel(resolvedBaseDir, resolvedPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("security violation: resolved path %q escapes base directory %q at runtime", resolvedPath, resolvedBaseDir)
		}
	}

	if err := validateExtension(resolvedPath); err != nil {
		return nil, fmt.Errorf("security violation: file extension changed post-boot for resource %q: %w", r.Config.Name, err)
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

	limit := *r.Config.MaxSize
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

// GetAnnotations returns the resource annotations, dynamically computing the LastModified timestamp.
func (r *FileResource) GetAnnotations() *resources.ResourceAnnotations {
	var ret resources.ResourceAnnotations
	if r.Config.Annotations != nil {
		ret = *r.Config.Annotations
	}

	resolvedPath := r.absPath
	if resolved, err := filepath.EvalSymlinks(r.absPath); err == nil {
		resolvedPath = resolved
	}

	if r.isRelative && r.resolvedBaseDir != "" {
		resolvedBaseDir := r.resolvedBaseDir
		if resolved, err := filepath.EvalSymlinks(resolvedBaseDir); err == nil {
			resolvedBaseDir = resolved
		}
		rel, err := filepath.Rel(resolvedBaseDir, resolvedPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return &ret
		}
	}

	if info, err := os.Stat(resolvedPath); err == nil && info.Mode().IsRegular() {
		ret.LastModified = info.ModTime().Format(time.RFC3339)
	}

	return &ret
}

// ToConfig returns the underlying configuration.
func (r *FileResource) ToConfig() resources.ResourceConfig {
	return &r.Config
}

// Size dynamically retrieves the current size of the file on disk.
func (r *FileResource) GetCurrentSize() (int64, error) {
	resolvedPath := r.absPath
	if resolved, err := filepath.EvalSymlinks(r.absPath); err == nil {
		resolvedPath = resolved
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat file for size: %w", err)
	}
	if !info.Mode().IsRegular() {
		return 0, fmt.Errorf("not a regular file")
	}

	size := info.Size()
	if size > *r.Config.MaxSize {
		size = *r.Config.MaxSize
	}
	return size, nil
}
