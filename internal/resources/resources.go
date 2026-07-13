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
	Initialize(ctx context.Context) (Resource, error)
}

// Resource is the initialized object that handles data execution.
type Resource interface {
	Read(ctx context.Context, params map[string]any) (any, error)
	ToConfig() ResourceConfig
}

// BaseConfig holds the shared properties for all resource configurations.
type BaseConfig struct {
	Name        string         `yaml:"name"`
	Type        string         `yaml:"type"`
	URI         string         `yaml:"uri"`
	Description string         `yaml:"description"`
	Title       string         `yaml:"title"`
	MimeType    string         `yaml:"mimeType"`
	Annotations map[string]any `yaml:"annotations"`
}

// GetURI returns the URI of the resource configuration.
func (c BaseConfig) GetURI() string {
	return c.URI
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
	return config, nil
}
