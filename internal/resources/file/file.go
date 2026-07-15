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

	absPath string
}

// ResourceConfigType returns the resource type identifier.
func (c *Config) ResourceConfigType() string {
	return "file"
}

func validateExtension(path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	allowedExts := map[string]bool{
		".txt": true, ".md": true, ".csv": true, ".json": true,
		".yaml": true, ".yml": true, ".xml": true, ".sql": true,
	}
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
		return nil, fmt.Errorf("file resource %q max_size must be greater than 0MB", c.Name)
	}

	if filepath.IsAbs(c.Path) {
		c.absPath = filepath.Clean(c.Path)
	} else {
		if !filepath.IsLocal(c.Path) {
			return nil, fmt.Errorf("relative path %q is unsafe", c.Path)
		}
		baseDir := resources.GetBaseDirFromContext(ctx)
		c.absPath = filepath.Clean(filepath.Join(baseDir, c.Path))
	}

	if abs, err := filepath.Abs(c.absPath); err == nil {
		c.absPath = abs
	}

	resolvedPath, err := filepath.EvalSymlinks(c.absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate symlinks for resource %q: %w", c.Name, err)
	}
	c.absPath = resolvedPath

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

	size := info.Size()
	if size > *c.MaxSize {
		size = *c.MaxSize
	}
	c.Size = &size

	if c.Annotations == nil {
		c.Annotations = &resources.ResourceAnnotations{}
	}
	if c.Annotations.LastModified == "" {
		c.Annotations.LastModified = info.ModTime().Format(time.RFC3339)
	}

	if c.MimeType == "" {
		ext := strings.ToLower(filepath.Ext(c.absPath))
		c.MimeType = mime.TypeByExtension(ext)
		if c.MimeType == "" {
			c.MimeType = "text/plain"
		}
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

	if err := validateExtension(resolvedPath); err != nil {
		return nil, fmt.Errorf("security violation: file extension changed post-boot for resource %q: %w", r.config.Name, err)
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
		warning := fmt.Sprintf("\n\n...[TRUNCATED BY SERVER: Payload exceeded %d byte safety limit]...", limit)
		return string(content[:limit]) + warning, nil
	}

	return string(content), nil
}

// ToConfig returns the runtime config struct back to the caller.
func (r *FileResource) ToConfig() resources.ResourceConfig {
	return r.config
}
