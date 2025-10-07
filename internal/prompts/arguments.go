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

// Argument is a wrapper around a tools.Parameter that provides prompt-specific functionality.
type Argument struct {
	tools.Parameter
}

// McpPromptManifest returns the simplified manifest structure required for prompts.
func (a Argument) McpPromptManifest() McpPromptArg {
	return McpPromptArg{
		Name:        a.GetName(),
		Description: a.Manifest().Description,
		Required:    tools.CheckParamRequired(a.GetRequired(), a.GetDefault()),
	}
}

// Arguments is a slice of Argument.
type Arguments []Argument

// UnmarshalYAML is a custom unmarshaler that parses YAML into a slice of Arguments.
func (a *Arguments) UnmarshalYAML(ctx context.Context, unmarshal func(interface{}) error) error {
	var rawList []util.DelayedUnmarshaler
	if err := unmarshal(&rawList); err != nil {
		return err
	}
	*a = make(Arguments, 0, len(rawList))
	for _, u := range rawList {
		p, err := parseArgFromDelayedUnmarshaler(ctx, &u)
		if err != nil {
			return err
		}
		*a = append(*a, p)
	}
	return nil
}

func parseArgFromDelayedUnmarshaler(ctx context.Context, u *util.DelayedUnmarshaler) (Argument, error) {
	var p map[string]any
	if err := u.Unmarshal(&p); err != nil {
		return Argument{}, fmt.Errorf("error parsing argument: %w", err)
	}

	t, ok := p["type"]
	if !ok {
		t = "any" // This is the prompt-specific behavior
	}

	dec, err := util.NewStrictDecoder(p)
	if err != nil {
		return Argument{}, fmt.Errorf("error creating decoder for argument: %w", err)
	}

	var param tools.Parameter
	switch t {
	case "string":
		param = &tools.StringParameter{}
	case "integer":
		param = &tools.IntParameter{}
	case "float":
		param = &tools.FloatParameter{}
	case "boolean":
		param = &tools.BooleanParameter{}
	case "array":
		param = &tools.ArrayParameter{}
	case "map":
		param = &tools.MapParameter{}
	case "any":
		// 'any' type is specific to prompts, so we handle it here.
		param = &AnyParameter{}
	default:
		return Argument{}, fmt.Errorf("unknown argument type: %q", t)
	}

	if err := dec.DecodeContext(ctx, param); err != nil {
		return Argument{}, fmt.Errorf("unable to parse argument as type %q: %w", t, err)
	}

	return Argument{Parameter: param}, nil
}

// AnyParameter is a parameter representing any type, specific to prompts.
type AnyParameter struct {
	tools.CommonParameter `yaml:",inline"`
	Default               *any `yaml:"default"`
}

func (a *AnyParameter) Parse(v any) (any, error) { return v, nil }
func (a *AnyParameter) GetDefault() any {
	if a.Default == nil {
		return nil
	}
	return *a.Default
}
func (a *AnyParameter) GetAuthServices() []tools.ParamAuthService { return a.AuthServices }
func (a *AnyParameter) Manifest() tools.ParameterManifest {
	return tools.ParameterManifest{
		Name: a.GetName(), Type: a.GetType(), Description: a.Desc,
		Required: tools.CheckParamRequired(a.GetRequired(), a.GetDefault()),
	}
}
func (a *AnyParameter) McpManifest() (tools.ParameterMcpManifest, []string) {
	return a.CommonParameter.McpManifest()
}
