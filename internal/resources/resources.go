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

	yaml "github.com/goccy/go-yaml"
)

// ResourceConfigFactory creates a configuration instance from YAML.
type ResourceConfigFactory func(ctx context.Context, name string, decoder *yaml.Decoder) (ResourceConfig, error)

var resourceRegistry = make(map[string]ResourceConfigFactory)

// Register adds a resource type handler to the system registry.
// Returns true if the registration was successful, or false if the type was already registered.
func Register(resourceType string, factory ResourceConfigFactory) bool {
	if _, exists := resourceRegistry[resourceType]; exists {
		return false
	}
	resourceRegistry[resourceType] = factory
	return true
}

// DecodeConfig looks up the factory and parses YAML into a ResourceConfig.
func DecodeConfig(ctx context.Context, resourceType, name string, decoder *yaml.Decoder) (ResourceConfig, error) {
	factory, found := resourceRegistry[resourceType]
	if !found {
		return nil, fmt.Errorf("unknown resource type: %q", resourceType)
	}
	return factory(ctx, name, decoder)
}

// ResourceConfig represents the uninitialized configuration data for a resource.
type ResourceConfig interface {
	ResourceConfigType() string
	Initialize(ctx context.Context, configDir string, usingConfigFolder bool) (Resource, error)
}

// Resource represents the initialized and validated resource.
type Resource interface {
	ResourceName() string
	ResourceURI() string
	ResourceType() string
	ResourceDescription() string
	ResourceTitle() string
	ResourceMimeType() string
	ResourceAnnotations() *Annotations
	Read(ctx context.Context, params map[string]any) ([]ResourceContent, error)
	ToConfig() ResourceConfig
}

// Annotations represent standard MCP resource annotations.
type Annotations struct {
	Audience []string `yaml:"audience,omitempty" json:"audience,omitempty"`
	Priority *float64 `yaml:"priority,omitempty" json:"priority,omitempty"`
}

// ResourceContent represents the payload content of a read resource.
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text"`
}

// ResourceSet represents a grouping of resources, used for protocol-level filtering.
type ResourceSet struct {
	Name         string
	ResourceURIs []string
}

// ContainsResource checks if a resource URI is part of the resource set.
func (rs ResourceSet) ContainsResource(uri string) bool {
	for _, u := range rs.ResourceURIs {
		if u == uri {
			return true
		}
	}
	return false
}
