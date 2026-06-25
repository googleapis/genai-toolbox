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

const TextResourceType = "text"

func init() {
	Register(TextResourceType, newTextConfig)
}

func newTextConfig(ctx context.Context, name string, decoder *yaml.Decoder) (ResourceConfig, error) {
	cfg := &TextConfig{Name: name}
	if err := decoder.DecodeContext(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to decode text resource config %q: %w", name, err)
	}
	return cfg, nil
}

// TextConfig represents the configuration for a static text/information resource.
type TextConfig struct {
	Name        string       `yaml:"name"`
	Type        string       `yaml:"type"`
	URI         string       `yaml:"uri,omitempty"`
	Description string       `yaml:"description,omitempty"`
	Title       string       `yaml:"title,omitempty"`
	Text        string       `yaml:"text"`
	Annotations *Annotations `yaml:"annotations,omitempty"`
}

func (c *TextConfig) ResourceConfigType() string {
	return TextResourceType
}

// Initialize validates the configuration and builds the initialized TextResource.
func (c *TextConfig) Initialize(ctx context.Context, configDir string, usingConfigFolder bool) (Resource, error) {
	if c.Text == "" {
		return nil, fmt.Errorf("text is required for resource %q", c.Name)
	}

	uri := c.URI
	if uri == "" {
		uri = fmt.Sprintf("info://%s", c.Name)
	}

	// Set default annotations: priority to 1.0 if not defined
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

	return &TextResource{
		cfg: c,
		uri: uri,
		ann: ann,
	}, nil
}

// TextResource is the initialized written information resource.
type TextResource struct {
	cfg *TextConfig
	uri string
	ann *Annotations
}

func (r *TextResource) ResourceName() string {
	return r.cfg.Name
}

func (r *TextResource) ResourceURI() string {
	return r.uri
}

func (r *TextResource) ResourceType() string {
	return TextResourceType
}

func (r *TextResource) ResourceDescription() string {
	return r.cfg.Description
}

func (r *TextResource) ResourceTitle() string {
	return r.cfg.Title
}

func (r *TextResource) ResourceMimeType() string {
	// Standard written text resources default to plain text MIME type
	return "text/plain"
}

func (r *TextResource) ResourceAnnotations() *Annotations {
	return r.ann
}

// Read returns the configured written information.
func (r *TextResource) Read(ctx context.Context, params map[string]any) ([]ResourceContent, error) {
	return []ResourceContent{
		{
			URI:      r.uri,
			MimeType: r.ResourceMimeType(),
			Text:     r.cfg.Text,
		},
	}, nil
}

func (r *TextResource) ToConfig() ResourceConfig {
	return r.cfg
}
