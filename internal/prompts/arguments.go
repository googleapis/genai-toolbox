// Copyright 2025 Google LLC
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

package prompts

import (
	"context"
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
)

// McpArgManifest is the simplified manifest structure for an argument required for prompts.
type McpArgManifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// Argument is a wrapper around a tools.Parameter that provides prompt-specific functionality.
// If the 'type' field is not specified in a YAML definition, it defaults to 'string'.
type Argument struct {
	tools.Parameter
}

// McpArgManifest returns the simplified manifest structure required for prompts.
func (a Argument) McpArgManifest() McpArgManifest {
	return McpArgManifest{
		Name:        a.GetName(),
		Description: a.Manifest().Description,
		Required:    tools.CheckParamRequired(a.GetRequired(), a.GetDefault()),
	}
}

// Arguments is a slice of Argument.
type Arguments []Argument

// UnmarshalYAML provides custom unmarshaling logic for Arguments.
func (args *Arguments) UnmarshalYAML(ctx context.Context, unmarshal func(interface{}) error) error {
	*args = make(Arguments, 0)
	var rawList []util.DelayedUnmarshaler
	if err := unmarshal(&rawList); err != nil {
		return err
	}

	for _, u := range rawList {
		var p map[string]any
		if err := u.Unmarshal(&p); err != nil {
			return fmt.Errorf("error parsing argument: %w", err)
		}

		// If 'type' is missing, default it to string.
		paramType, ok := p["type"]
		if !ok {
			p["type"] = tools.TypeString
			paramType = tools.TypeString
		}

		// Call the clean, exported parser from the tools package. No more duplicated logic!
		param, err := tools.ParseParameter(ctx, p, paramType.(string))
		if err != nil {
			return err
		}

		*args = append(*args, Argument{Parameter: param})
	}
	return nil
}

// ParseArguments validates and processes the user-provided arguments against the prompt's requirements.
func ParseArguments(arguments Arguments, args map[string]any, data map[string]map[string]any) (tools.ParamValues, error) {
	var parameters tools.Parameters
	for _, arg := range arguments {
		parameters = append(parameters, arg)
	}
	return tools.ParseParams(parameters, args, data)
}
