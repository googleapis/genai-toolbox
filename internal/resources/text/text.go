// Copyright 2024 Google LLC
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

package text

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/resources"
)

const resourceType = "text"

func init() {
	if !resources.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("resource type %q already registered", resourceType))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (resources.ResourceConfig, error) {
	cfg := &Config{BaseConfig: resources.BaseConfig{Name: name, Type: resourceType}}
	if err := decoder.DecodeContext(ctx, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Config represents the uninitialized textual resource configuration from YAML.
type Config struct {
	resources.BaseConfig `yaml:",inline"`
	Text                 string `yaml:"text"`
}

var _ resources.ResourceConfig = (*Config)(nil)

func (c *Config) ResourceConfigType() string {
	return resourceType
}

func (c *Config) Initialize(ctx context.Context) (resources.Resource, error) {
	if c.Text == "" {
		return nil, fmt.Errorf("missing required 'text' field for text resource %q", c.Name)
	}

	// Default MimeType if unset
	if c.MimeType == "" {
		c.MimeType = "text/plain"
	}

	// Default annotations.priority to 1.0 (critical) if unset
	if c.Annotations == nil {
		c.Annotations = &resources.ResourceAnnotations{}
	}
	if c.Annotations.Priority == nil {
		p := 1.0
		c.Annotations.Priority = &p
	}

	// Invert at boot by calculating the length of the string array as per design doc
	size := int64(len([]byte(c.Text)))
	c.Size = &size

	return &Resource{config: *c}, nil
}

// Resource represents the initialized textual resource that returns plain text payloads.
type Resource struct {
	config Config
}

var _ resources.Resource = (*Resource)(nil)

func (r *Resource) Read(ctx context.Context, params map[string]any) (any, error) {
	return r.config.Text, nil
}

func (r *Resource) ToConfig() resources.ResourceConfig {
	return &r.config
}
