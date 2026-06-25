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

package resources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "github.com/goccy/go-yaml"
)

const FileResourceType = "file"
const MaxFileSize = 5 * 1024 * 1024 // 5 MB

func init() {
	Register(FileResourceType, newFileConfig)
}

func newFileConfig(ctx context.Context, name string, decoder *yaml.Decoder) (ResourceConfig, error) {
	cfg := &FileConfig{Name: name}
	if err := decoder.DecodeContext(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to decode file resource config %q: %w", name, err)
	}
	return cfg, nil
}

// FileConfig represents the configuration for a file-based resource or template.
type FileConfig struct {
	Name         string       `yaml:"name"`
	Type         string       `yaml:"type"`
	URI          string       `yaml:"uri"`
	Description  string       `yaml:"description,omitempty"`
	Title        string       `yaml:"title,omitempty"`
	MimeType     string       `yaml:"mimeType,omitempty"`
	AllowedPaths []string     `yaml:"allowedPaths,omitempty"`
	Annotations  *Annotations `yaml:"annotations,omitempty"`
}

func (c *FileConfig) ResourceConfigType() string {
	return FileResourceType
}

// Initialize validates and prepares the FileResource.
func (c *FileConfig) Initialize(ctx context.Context, configDir string, usingConfigFolder bool) (Resource, error) {
	// Strict Scheme Whitelisting
	if !strings.HasPrefix(c.URI, "file://") {
		return nil, fmt.Errorf("resource %q URI must start with file://", c.Name)
	}

	// Safety Check: Enforce allowed paths if started with config-folder
	if usingConfigFolder && len(c.AllowedPaths) == 0 {
		return nil, fmt.Errorf("security error: resource %q of type 'file' must explicitly define 'allowedPaths' when starting the server with a config directory flag", c.Name)
	}

	// Determine sandbox roots
	var sandboxRoots []string
	if len(c.AllowedPaths) > 0 {
		for _, p := range c.AllowedPaths {
			abs, err := filepath.Abs(p)
			if err != nil {
				return nil, fmt.Errorf("invalid allowed path %q for resource %q: %w", p, c.Name, err)
			}
			sandboxRoots = append(sandboxRoots, abs)
		}
	} else {
		// Default sandbox is the directory where the configuration file lives
		abs, err := filepath.Abs(configDir)
		if err != nil {
			return nil, fmt.Errorf("invalid config directory: %w", err)
		}
		sandboxRoots = append(sandboxRoots, abs)
	}

	// Default annotations: priority to 1.0 if not defined
	ann := c.Annotations
	if ann == nil {
		prio := 1.0
		ann = &Annotations{
			Priority: &prio,
		}
	} else if ann.Priority == nil {
		prio := 1.0
		ann.Priority = &prio
	}

	return &FileResource{
		cfg:          c,
		sandboxRoots: sandboxRoots,
		ann:          ann,
	}, nil
}

// FileResource represents the initialized file resource.
type FileResource struct {
	cfg          *FileConfig
	sandboxRoots []string
	ann          *Annotations
}

func (r *FileResource) ResourceName() string {
	return r.cfg.Name
}

func (r *FileResource) ResourceURI() string {
	return r.cfg.URI
}

func (r *FileResource) ResourceType() string {
	return FileResourceType
}

func (r *FileResource) ResourceDescription() string {
	return r.cfg.Description
}

func (r *FileResource) ResourceTitle() string {
	return r.cfg.Title
}

func (r *FileResource) ResourceMimeType() string {
	if r.cfg.MimeType != "" {
		return r.cfg.MimeType
	}
	return "text/plain"
}

func (r *FileResource) ResourceAnnotations() *Annotations {
	return r.ann
}

// Read resolves the requested path, applies strict sandboxing and security checks, and returns the file content.
func (r *FileResource) Read(ctx context.Context, params map[string]any) ([]ResourceContent, error) {
	// Substitute template parameters if this is a template
	actualURI := r.cfg.URI
	if strings.Contains(actualURI, "{path}") {
		val, ok := params["path"]
		if !ok {
			return nil, fmt.Errorf("missing required parameter 'path' for resource template %q", r.cfg.Name)
		}
		pathStr, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("parameter 'path' must be a string for resource template %q", r.cfg.Name)
		}
		actualURI = strings.ReplaceAll(actualURI, "{path}", pathStr)
	}

	// Extract file path from the URI
	filePath := strings.TrimPrefix(actualURI, "file://")

	// 1. Clean the path to resolve traversals like '../'
	cleanedPath := filepath.Clean(filePath)

	// 2. Resolve symlinks to find the true physical path
	resolvedPath, err := filepath.EvalSymlinks(cleanedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access resource path: %w", err)
	}

	// 3. Sandbox Enforcement: verify path is strictly inside one of the sandbox roots
	isInside := false
	for _, root := range r.sandboxRoots {
		if isPathInside(resolvedPath, root) {
			isInside = true
			break
		}
	}
	if !isInside {
		return nil, fmt.Errorf("permission denied: path %q escapes the allowed sandbox roots", filePath)
	}

	// 4. Hidden Files Check: block reading hidden files or directories (starting with '.')
	segments := strings.Split(resolvedPath, string(filepath.Separator))
	for _, seg := range segments {
		if strings.HasPrefix(seg, ".") && seg != "." && seg != ".." {
			return nil, fmt.Errorf("permission denied: reading hidden files or directories is not allowed")
		}
	}

	// 5. Size and Directory validation
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("invalid resource: %q is a directory, not a file", filePath)
	}
	if info.Size() > MaxFileSize {
		return nil, fmt.Errorf("file size %d bytes exceeds the maximum limit of 5MB", info.Size())
	}

	// Read the file content
	contentBytes, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return []ResourceContent{
		{
			URI:      actualURI,
			MimeType: r.ResourceMimeType(),
			Text:     string(contentBytes),
		},
	}, nil
}

func (r *FileResource) ToConfig() ResourceConfig {
	return r.cfg
}

// isPathInside returns true if path matches root or lies within root.
func isPathInside(path, root string) bool {
	if path == root {
		return true
	}
	return strings.HasPrefix(path, root+string(filepath.Separator))
}
