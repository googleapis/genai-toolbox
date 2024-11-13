// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package server

import (
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/authSources"
	"github.com/googleapis/genai-toolbox/internal/sources"
	alloydbpgsrc "github.com/googleapis/genai-toolbox/internal/sources/alloydbpg"
	cloudsqlpgsrc "github.com/googleapis/genai-toolbox/internal/sources/cloudsqlpg"
	postgressrc "github.com/googleapis/genai-toolbox/internal/sources/postgres"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/postgressql"
	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	// Server version
	Version string
	// Address is the address of the interface the server will listen on.
	Address string
	// Port is the port the server will listen on.
	Port int
	// SourceConfigs defines what sources of data are available for tools.
	SourceConfigs SourceConfigs
	// AuthSourceConfigs defines what sources of authentication are available for tools.
	AuthSourceConfigs AuthSourceConfigs
	// ToolConfigs defines what tools are available.
	ToolConfigs ToolConfigs
	// ToolsetConfigs defines what tools are available.
	ToolsetConfigs ToolsetConfigs
}

// SourceConfigs is a type used to allow unmarshal of the data source config map
type SourceConfigs map[string]sources.SourceConfig

// validate interface
var _ yaml.Unmarshaler = &SourceConfigs{}

func (c *SourceConfigs) UnmarshalYAML(node *yaml.Node) error {
	*c = make(SourceConfigs)
	// Parse the 'kind' fields for each source
	var raw map[string]yaml.Node
	if err := node.Decode(&raw); err != nil {
		return err
	}

	for name, n := range raw {
		var k struct {
			Kind string `yaml:"kind"`
		}
		err := n.Decode(&k)
		if err != nil {
			return fmt.Errorf("missing 'kind' field for %q", k)
		}
		switch k.Kind {
		case alloydbpgsrc.SourceKind:
			actual := alloydbpgsrc.Config{Name: name}
			if err := n.Decode(&actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", k.Kind, err)
			}
			(*c)[name] = actual
		case cloudsqlpgsrc.SourceKind:
			actual := cloudsqlpgsrc.Config{Name: name}
			if err := n.Decode(&actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", k.Kind, err)
			}
			(*c)[name] = actual
		case postgressrc.SourceKind:
			actual := postgressrc.Config{Name: name}
			if err := n.Decode(&actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", k.Kind, err)
			}
			(*c)[name] = actual
		default:
			return fmt.Errorf("%q is not a valid kind of data source", k.Kind)
		}

	}
	return nil
}

// AuthSourceConfigs is a type used to allow unmarshal of the data authSource config map
type AuthSourceConfigs map[string]authSources.AuthSourceConfig

// validate interface
var _ yaml.Unmarshaler = &SourceConfigs{}

func (c *AuthSourceConfigs) UnmarshalYAML(node *yaml.Node) error {
	*c = make(AuthSourceConfigs)
	// Parse the 'kind' fields for each authSource
	var raw map[string]yaml.Node
	if err := node.Decode(&raw); err != nil {
		return err
	}

	for _, n := range raw {
		var k struct {
			Kind string `yaml:"kind"`
		}
		err := n.Decode(&k)
		if err != nil {
			return fmt.Errorf("missing 'kind' field for %q", k)
		}
		switch k.Kind {
		default:
			return fmt.Errorf("%q is not a valid kind of auth source", k.Kind)
		}
	}
	return nil
}

// ToolConfigs is a type used to allow unmarshal of the tool configs
type ToolConfigs map[string]tools.ToolConfig

// validate interface
var _ yaml.Unmarshaler = &ToolConfigs{}

func (c *ToolConfigs) UnmarshalYAML(node *yaml.Node) error {
	*c = make(ToolConfigs)
	// Parse the 'kind' fields for each source
	var raw map[string]yaml.Node
	if err := node.Decode(&raw); err != nil {
		return err
	}

	for name, n := range raw {
		var k struct {
			Kind string `yaml:"kind"`
		}
		err := n.Decode(&k)
		if err != nil {
			return fmt.Errorf("missing 'kind' field for %q", name)
		}
		switch k.Kind {
		case postgressql.ToolKind:
			actual := postgressql.Config{Name: name}
			if err := n.Decode(&actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", k.Kind, err)
			}
			(*c)[name] = actual
		default:
			return fmt.Errorf("%q is not a valid kind of tool", k.Kind)
		}

	}
	return nil
}

// ToolConfigs is a type used to allow unmarshal of the toolset configs
type ToolsetConfigs map[string]tools.ToolsetConfig

// validate interface
var _ yaml.Unmarshaler = &ToolsetConfigs{}

func (c *ToolsetConfigs) UnmarshalYAML(node *yaml.Node) error {
	*c = make(ToolsetConfigs)

	var raw map[string][]string
	if err := node.Decode(&raw); err != nil {
		return err
	}

	for name, toolList := range raw {
		(*c)[name] = tools.ToolsetConfig{Name: name, ToolNames: toolList}
	}
	return nil
}
