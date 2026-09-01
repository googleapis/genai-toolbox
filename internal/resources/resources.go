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
	"net/url"
	"strings"
	"sync"

	"github.com/goccy/go-yaml"
)

type contextKey string

// BaseDirKey is the context key for storing the base directory path during config parsing.
const BaseDirKey contextKey = "baseDir"

// GetBaseDirFromContext extracts the base directory path from the context.
func GetBaseDirFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val, ok := ctx.Value(BaseDirKey).(string); ok {
		return val
	}
	return ""
}

// ResourceConfig represents the uninitialized configuration for a resource.
type ResourceConfig interface {
	ResourceConfigType() string
	GetURI() string
	SetDefaults()
	Validate() error
	Initialize(ctx context.Context) (Resource, error)
}

// Resource is the initialized object that handles data execution.
type Resource interface {
	GetName() string
	GetTitle() string
	GetDescription() string
	GetMimeType() string
	GetURI() string
	GetSize() *int64
	GetAnnotations() *ResourceAnnotations
	Read(ctx context.Context, params map[string]any) (any, error)
	ToConfig() ResourceConfig
}

type ResourceAnnotations struct {
	Priority     *float64       `yaml:"priority,omitempty"`
	Audience     []AudienceRole `yaml:"audience,omitempty"`
	LastModified string         `yaml:"lastModified,omitempty"`
}

// BaseConfig contains the common fields for all resource configurations.
type BaseConfig struct {
	Name        string               `yaml:"name"`
	Type        string               `yaml:"type"`
	URI         string               `yaml:"uri,omitempty"`
	Description string               `yaml:"description,omitempty"`
	Title       string               `yaml:"title,omitempty"`
	MimeType    string               `yaml:"mimeType,omitempty"`
	Annotations *ResourceAnnotations `yaml:"annotations,omitempty"`
}

func (c BaseConfig) GetName() string                      { return c.Name }
func (c BaseConfig) GetTitle() string                     { return c.Title }
func (c BaseConfig) GetDescription() string               { return c.Description }
func (c BaseConfig) GetMimeType() string                  { return c.MimeType }
func (c BaseConfig) GetAnnotations() *ResourceAnnotations { return c.Annotations }

// GetURI returns the URI of the resource configuration.
func (c BaseConfig) GetURI() string {
	return c.URI
}

type AudienceRole string

const (
	RoleUser      AudienceRole = "user"
	RoleAssistant AudienceRole = "assistant"
)

func (r *AudienceRole) UnmarshalYAML(b []byte) error {
	var s string
	if err := yaml.Unmarshal(b, &s); err != nil {
		return err
	}
	switch s {
	case string(RoleUser), string(RoleAssistant):
		*r = AudienceRole(s)
		return nil
	default:
		return fmt.Errorf("invalid audience %q: must be 'user' or 'assistant'", s)
	}
}

// SetDefaults applies system defaults (like priority=1.0) for unspecified optional fields.
func (c *BaseConfig) SetDefaults() {
	if c.Annotations == nil {
		c.Annotations = &ResourceAnnotations{}
	}
	if c.Annotations.Priority == nil {
		p := 1.0
		c.Annotations.Priority = &p
	}
}

// Validate performs base configuration validation, such as checking for duplicate audiences.
func (c *BaseConfig) Validate() error {
	if c.URI == "" {
		return fmt.Errorf("missing required 'uri' field for resource %q", c.Name)
	}

	parsed, err := url.Parse(c.URI)
	if err != nil || parsed.Scheme == "" {
		return fmt.Errorf("invalid 'uri' field for resource %q: must be a valid RFC-compliant absolute URI with a scheme", c.Name)
	}

	// Normalize scheme and host to lowercase for consistent comparison and usage
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	c.URI = parsed.String()

	if c.Annotations != nil && len(c.Annotations.Audience) > 0 {
		seen := make(map[AudienceRole]bool)
		for _, aud := range c.Annotations.Audience {
			if seen[aud] {
				return fmt.Errorf("duplicate audience %q is not allowed", aud)
			}
			seen[aud] = true
		}
	}
	return nil
}

// ResourceConfigFactory defines the signature for a function that creates and
// decodes a specific resource's configuration.
type ResourceConfigFactory func(ctx context.Context, name string, decoder *yaml.Decoder) (ResourceConfig, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]ResourceConfigFactory)
)

// Register allows individual resource packages to register their configuration
// factory function. It returns true if the registration was successful, and false
// if the factory is nil or a resource with the same type was already registered.
func Register(resourceType string, factory ResourceConfigFactory) bool {
	if factory == nil {
		return false
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[resourceType]; exists {
		return false
	}
	registry[resourceType] = factory
	return true
}

// DecodeConfig decodes a YAML document into the appropriate ResourceConfig implementation.
func DecodeConfig(ctx context.Context, resourceType, name string, decoder *yaml.Decoder) (ResourceConfig, error) {
	if decoder == nil {
		return nil, fmt.Errorf("decoder cannot be nil for resource %q", name)
	}
	registryMu.RLock()
	factory, found := registry[resourceType]
	registryMu.RUnlock()
	if !found {
		return nil, fmt.Errorf("unknown resource type: %q", resourceType)
	}

	config, err := factory(ctx, name, decoder)
	if err != nil {
		return nil, fmt.Errorf("unable to parse resource %q as type %q: %w", name, resourceType, err)
	}
	if config == nil {
		return nil, fmt.Errorf("factory returned nil config for resource %q as type %q", name, resourceType)
	}

	config.SetDefaults()

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed for resource %q: %w", name, err)
	}

	return config, nil
}
