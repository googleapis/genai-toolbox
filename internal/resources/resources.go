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
	GetName() string
	GetTitle() string
	GetDescription() string
	GetMimeType() string
	Initialize(ctx context.Context) (Resource, error)
}

// Resource is the initialized object that handles data execution.
type Resource interface {
	Read(ctx context.Context, params map[string]any) (any, error)
	ToConfig() ResourceConfig
}

type ResourceAnnotations struct {
	Priority     *float64       `yaml:"priority,omitempty"`
	Audience     []AudienceRole `yaml:"audience,omitempty"`
	LastModified string         `yaml:"lastModified,omitempty"`
}

// BaseConfig contains the common fields for all resource and template configurations.
type BaseConfig struct {
	Name        string               `yaml:"name"`
	Type        string               `yaml:"type"`
	Description string               `yaml:"description,omitempty"`
	Title       string               `yaml:"title,omitempty"`
	MimeType    string               `yaml:"mimeType,omitempty"`
	Annotations *ResourceAnnotations `yaml:"annotations,omitempty"`
}

func (c BaseConfig) GetName() string        { return c.Name }
func (c BaseConfig) GetTitle() string       { return c.Title }
func (c BaseConfig) GetDescription() string { return c.Description }
func (c BaseConfig) GetMimeType() string    { return c.MimeType }

// BaseResourceConfig contains the fields for a specific resource configuration.
type BaseResourceConfig struct {
	BaseConfig `yaml:",inline"`
	URI        string `yaml:"uri,omitempty"`
	Size       *int64 `yaml:"-"`
}

// GetURI returns the URI of the resource configuration.
func (c BaseResourceConfig) GetURI() string {
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
func (c BaseConfig) Validate() error {
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

// DecodeConfig looks up the registered factory for the given type and uses it
// to decode the resource configuration.
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

	if defaulter, ok := config.(interface{ SetDefaults() }); ok {
		defaulter.SetDefaults()
	}

	if validatable, ok := config.(interface{ Validate() error }); ok {
		if err := validatable.Validate(); err != nil {
			return nil, fmt.Errorf("validation failed for resource %q: %w", name, err)
		}
	}

	return config, nil
}

// ResourceTemplateConfig represents the uninitialized configuration for a resource template.
type ResourceTemplateConfig interface {
	ResourceTemplateConfigType() string
	GetURITemplate() string
	GetName() string
	GetTitle() string
	GetDescription() string
	GetMimeType() string
	Initialize(ctx context.Context) (ResourceTemplate, error)
}

// ResourceTemplate is the initialized object that handles data execution.
type ResourceTemplate interface {
	Read(ctx context.Context, params map[string]any) (any, error)
	ToConfig() ResourceTemplateConfig
}

// BaseResourceTemplateConfig contains the specific fields for resource template configurations.
type BaseResourceTemplateConfig struct {
	BaseConfig  `yaml:",inline"`
	URITemplate string `yaml:"uriTemplate"`
}

// GetURITemplate returns the URI template of the resource configuration.
func (c BaseResourceTemplateConfig) GetURITemplate() string {
	return c.URITemplate
}

// ResourceTemplateConfigFactory defines the signature for a function that creates and
// decodes a specific resource template's configuration.
type ResourceTemplateConfigFactory func(ctx context.Context, name string, decoder *yaml.Decoder) (ResourceTemplateConfig, error)

var (
	templateRegistryMu sync.RWMutex
	templateRegistry   = make(map[string]ResourceTemplateConfigFactory)
)

// RegisterTemplate allows individual resource packages to register their template configuration
// factory function. It returns true if the registration was successful, and false
// if the factory is nil or a template with the same type was already registered.
func RegisterTemplate(resourceType string, factory ResourceTemplateConfigFactory) bool {
	if factory == nil {
		return false
	}
	templateRegistryMu.Lock()
	defer templateRegistryMu.Unlock()
	if _, exists := templateRegistry[resourceType]; exists {
		return false
	}
	templateRegistry[resourceType] = factory
	return true
}

// DecodeTemplateConfig looks up the registered factory for the given type and uses it
// to decode the resource template configuration.
func DecodeTemplateConfig(ctx context.Context, resourceType, name string, decoder *yaml.Decoder) (ResourceTemplateConfig, error) {
	if decoder == nil {
		return nil, fmt.Errorf("decoder cannot be nil for resource template %q", name)
	}
	templateRegistryMu.RLock()
	factory, found := templateRegistry[resourceType]
	templateRegistryMu.RUnlock()
	if !found {
		return nil, fmt.Errorf("unknown resource template type: %q", resourceType)
	}

	config, err := factory(ctx, name, decoder)
	if err != nil {
		return nil, fmt.Errorf("unable to parse resource template %q as type %q: %w", name, resourceType, err)
	}
	if config == nil {
		return nil, fmt.Errorf("factory returned nil config for resource template %q as type %q", name, resourceType)
	}

	if defaulter, ok := config.(interface{ SetDefaults() }); ok {
		defaulter.SetDefaults()
	}

	if validatable, ok := config.(interface{ Validate() error }); ok {
		if err := validatable.Validate(); err != nil {
			return nil, fmt.Errorf("validation failed for resource template %q: %w", name, err)
		}
	}

	return config, nil
}
