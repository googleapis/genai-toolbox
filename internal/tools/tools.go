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

package tools

import (
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"gopkg.in/yaml.v3"
)

type Config interface {
	toolKind() string
	Describe() ToolManifest
	Initialize(map[string]sources.Source) (Tool, error)
}

// SourceConfigs is a type used to allow unmarshal of the data source config map
type Configs map[string]Config

// validate interface
var _ yaml.Unmarshaler = &Configs{}

func (c *Configs) UnmarshalYAML(node *yaml.Node) error {
	*c = make(Configs)
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
		case CloudSQLPgSQLGenericKind:
			actual := CloudSQLPgGenericConfig{Name: name}
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

type Tool interface {
	Invoke([]any) (string, error)
	ParseParams(data map[string]any) ([]any, error)
}

type Parameter struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
}

// ParseTypeError is a customer error for incorrectly typed Parameters.
type ParseTypeError struct {
	Name  string
	Type  string
	Value any
}

func (e ParseTypeError) Error() string {
	return fmt.Sprintf("Error parsing parameter %q: %q not type %q", e.Name, e.Value, e.Type)
}

// ParseParams is a helper function for parsing Parameters from an arbitraryJSON object.
func parseParams(ps []Parameter, data map[string]any) ([]any, error) {
	params := []any{}
	for _, p := range ps {
		v, ok := data[p.Name]
		if !ok {
			return nil, fmt.Errorf("Parameter %q is required!", p.Name)
		}
		switch p.Type {
		case "string":
			v, ok = v.(string)
		case "integer":
			v, ok = v.(int)
		case "float":
			v, ok = v.(float32)
		case "boolean":
			v, ok = v.(bool)
		}
		if !ok {
			return nil, &ParseTypeError{p.Name, p.Type, v}
		}
		params = append(params, v)
	}
	return params, nil
}

type ToolManifest struct {
	Description string      `json:"description"`
	Parameters  []Parameter `json:"parameters"`
}
