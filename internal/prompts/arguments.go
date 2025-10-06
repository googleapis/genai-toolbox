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

package prompts

import (
	"context"
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
)

// Argument is an interface that is compatible with tools.Parameter.
type Argument interface {
	tools.Parameter
	McpPromptManifest() McpPromptArg
}

// Arguments is a slice of Argument.
type Arguments []Argument

// UnmarshalYAML is a custom unmarshaler for a slice of Arguments.
func (a *Arguments) UnmarshalYAML(ctx context.Context, unmarshal func(interface{}) error) error {
	*a = make(Arguments, 0)
	var rawList []util.DelayedUnmarshaler
	if err := unmarshal(&rawList); err != nil {
		return err
	}
	for _, u := range rawList {
		p, err := parseArgFromDelayedUnmarshaler(ctx, &u)
		if err != nil {
			return err
		}
		(*a) = append((*a), p)
	}
	return nil
}

// parseArgFromDelayedUnmarshaler is a helper function to parse arguments based on their type.
func parseArgFromDelayedUnmarshaler(ctx context.Context, u *util.DelayedUnmarshaler) (Argument, error) {
	var p map[string]any
	if err := u.Unmarshal(&p); err != nil {
		return nil, fmt.Errorf("error parsing arguments: %w", err)
	}

	t, ok := p["type"]
	if !ok {
		t = "any"
	}

	dec, err := util.NewStrictDecoder(p)
	if err != nil {
		return nil, fmt.Errorf("error creating decoder: %w", err)
	}

	switch t {
	case "string":
		var toolParam tools.StringParameter
		if err := dec.DecodeContext(ctx, &toolParam); err != nil {
			return nil, err
		}
		return &StringArgument{StringParameter: toolParam}, nil
	case "integer":
		var toolParam tools.IntParameter
		if err := dec.DecodeContext(ctx, &toolParam); err != nil {
			return nil, err
		}
		return &IntArgument{IntParameter: toolParam}, nil
	case "float":
		var toolParam tools.FloatParameter
		if err := dec.DecodeContext(ctx, &toolParam); err != nil {
			return nil, err
		}
		return &FloatArgument{FloatParameter: toolParam}, nil
	case "boolean":
		var toolParam tools.BooleanParameter
		if err := dec.DecodeContext(ctx, &toolParam); err != nil {
			return nil, err
		}
		return &BooleanArgument{BooleanParameter: toolParam}, nil
	case "array":
		var toolParam tools.ArrayParameter
		if err := dec.DecodeContext(ctx, &toolParam); err != nil {
			return nil, err
		}
		return &ArrayArgument{ArrayParameter: toolParam}, nil
	case "map":
		var toolParam tools.MapParameter
		if err := dec.DecodeContext(ctx, &toolParam); err != nil {
			return nil, err
		}
		return &MapArgument{MapParameter: toolParam}, nil
	default: // "any"
		arg := &AnyArgument{}
		if err := dec.DecodeContext(ctx, arg); err != nil {
			return nil, err
		}
		return arg, nil
	}
}

// BaseArgument provides the common implementation for McpPromptManifest.
type BaseArgument struct{}

func (b *BaseArgument) McpPromptManifest(p tools.Parameter) McpPromptArg {
	return McpPromptArg{
		Name:        p.GetName(),
		Description: p.Manifest().Description,
		Required:    tools.CheckParamRequired(p.GetRequired(), p.GetDefault()),
	}
}

// --- Argument Struct Implementations ---

type AnyArgument struct {
	tools.CommonParameter `yaml:",inline"`
	Default               *any `yaml:"default"`
	BaseArgument
}

func (a *AnyArgument) Parse(v any) (any, error) { return v, nil }
func (a *AnyArgument) GetDefault() any {
	if a.Default == nil {
		return nil
	}
	return *a.Default
}
func (a *AnyArgument) GetAuthServices() []tools.ParamAuthService { return a.AuthServices }
func (a *AnyArgument) Manifest() tools.ParameterManifest {
	return tools.ParameterManifest{
		Name: a.GetName(), Type: a.GetType(), Description: a.Desc,
		Required: tools.CheckParamRequired(a.GetRequired(), a.GetDefault()),
	}
}
func (a *AnyArgument) McpManifest() (tools.ParameterMcpManifest, []string) {
	return a.CommonParameter.McpManifest()
}
func (a *AnyArgument) McpPromptManifest() McpPromptArg { return a.BaseArgument.McpPromptManifest(a) }

type StringArgument struct {
	tools.StringParameter
	BaseArgument
}

func (a *StringArgument) McpPromptManifest() McpPromptArg { return a.BaseArgument.McpPromptManifest(a) }

type IntArgument struct {
	tools.IntParameter
	BaseArgument
}

func (a *IntArgument) McpPromptManifest() McpPromptArg { return a.BaseArgument.McpPromptManifest(a) }

type FloatArgument struct {
	tools.FloatParameter
	BaseArgument
}

func (a *FloatArgument) McpPromptManifest() McpPromptArg { return a.BaseArgument.McpPromptManifest(a) }

type BooleanArgument struct {
	tools.BooleanParameter
	BaseArgument
}

func (a *BooleanArgument) McpPromptManifest() McpPromptArg {
	return a.BaseArgument.McpPromptManifest(a)
}

type ArrayArgument struct {
	tools.ArrayParameter
	BaseArgument
}

func (a *ArrayArgument) McpPromptManifest() McpPromptArg { return a.BaseArgument.McpPromptManifest(a) }

type MapArgument struct {
	tools.MapParameter
	BaseArgument
}

func (a *MapArgument) McpPromptManifest() McpPromptArg { return a.BaseArgument.McpPromptManifest(a) }
