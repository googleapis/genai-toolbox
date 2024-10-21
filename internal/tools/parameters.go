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

	"gopkg.in/yaml.v3"
)

const (
	typeString = "string"
	typeInt    = "int"
	typeFloat  = "float"
	typeBool   = "bool"
)

type Parameter interface {
	// Note: It's typically not idiomatic to include "Get" in the function name,
	// but this is done to differentiate it from the fields in CommonParameter.
	GetName() string
	GetType() string
	Parse(any) (any, error)
}

// Parameters is a type used to allow unmarshal a list of parameters
type Parameters []Parameter

func (c *Parameters) UnmarshalYAML(node *yaml.Node) error {
	*c = make(Parameters, 0)
	// Parse the 'kind' fields for each source
	var nodeList []yaml.Node
	if err := node.Decode(&nodeList); err != nil {
		return err
	}
	for _, n := range nodeList {
		var p CommonParameter
		err := n.Decode(&p)
		if err != nil {
			return fmt.Errorf("parameter missing required fields")
		}
		switch p.Type {
		case typeString:
			a := &StringParameter{}
			if err := n.Decode(a); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", p.Type, err)
			}
			*c = append(*c, a)
		case typeInt:
			a := &IntParameter{}
			if err := n.Decode(a); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", p.Type, err)
			}
			*c = append(*c, a)
		case typeFloat:
			a := &FloatParameter{}
			if err := n.Decode(a); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", p.Type, err)
			}
			*c = append(*c, a)
		case typeBool:
			a := &BooleanParameter{}
			if err := n.Decode(a); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", p.Type, err)
			}
			*c = append(*c, a)
		default:
			return fmt.Errorf("%q is not valid type for a parameter!", p.GetName())
		}
	}

	return nil
}

// CommonParameter are default fields that are emebdding in most Parameter implementations. Embedding this stuct will give the object Name() and Type() functions.
type CommonParameter struct {
	// These fields are prefixed with "P" for parameter to differntiate between the methods
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Desc string `yaml:"description"`
}

// GetName returns the name specified for the Parameter.
func (p *CommonParameter) GetName() string {
	return p.Name
}

// GetType returns the type specified for the Parameter.
func (p *CommonParameter) GetType() string {
	return p.Type
}

// ParseTypeError is a custom error for incorrectly typed Parameters.
type ParseTypeError struct {
	Name  string
	Type  string
	Value any
}

func (e ParseTypeError) Error() string {
	return fmt.Sprintf("Error parsing parameter %q: %q not type %q", e.Name, e.Value, e.Type)
}

// NewStringParameter is a convience function for initalizing a StringParameter.
func NewStringParameter(name, desc string) *StringParameter {
	return &StringParameter{
		CommonParameter: CommonParameter{
			Name: name,
			Type: typeString,
			Desc: desc,
		},
	}
}

var _ Parameter = &StringParameter{}

// StringParameter is a parameter representing the "string" type.
type StringParameter struct {
	CommonParameter `yaml:",inline"`
}

// Parse casts the value "v" as a "string".
func (p *StringParameter) Parse(v any) (any, error) {
	v, ok := v.(string)
	if !ok {
		return nil, &ParseTypeError{p.Name, p.Type, v}
	}
	return v, nil
}

// NewIntParameter is a convience function for initalizing a IntParameter.
func NewIntParameter(name, desc string) *IntParameter {
	return &IntParameter{
		CommonParameter: CommonParameter{
			Name: name,
			Type: typeInt,
			Desc: desc,
		},
	}
}

var _ Parameter = &IntParameter{}

// IntParameter is a parameter representing the "int" type.
type IntParameter struct {
	CommonParameter `yaml:",inline"`
}

func (p *IntParameter) Parse(v any) (any, error) {
	v, ok := v.(int64)
	if !ok {
		return nil, &ParseTypeError{p.Name, p.Type, v}
	}
	return v, nil
}

// NewFloatParameter is a convience function for initalizing a FloatParameter.
func NewFloatParameter(name, desc string) *FloatParameter {
	return &FloatParameter{
		CommonParameter: CommonParameter{
			Name: name,
			Type: typeFloat,
			Desc: desc,
		},
	}
}

var _ Parameter = &FloatParameter{}

// FloatParameter is a parameter representing the "float" type.
type FloatParameter struct {
	CommonParameter `yaml:",inline"`
}

func (p *FloatParameter) Parse(v any) (any, error) {
	v, ok := v.(float64)
	if !ok {
		return nil, &ParseTypeError{p.Name, p.Type, v}
	}
	return v, nil
}

// NewBooleanParameter is a convience function for initalizing a BooleanParameter.
func NewBooleanParameter(name, desc string) *BooleanParameter {
	return &BooleanParameter{
		CommonParameter: CommonParameter{
			Name: name,
			Type: typeBool,
			Desc: desc,
		},
	}
}

var _ Parameter = &BooleanParameter{}

// BooleanParameter is a parameter representing the "boolean" type.
type BooleanParameter struct {
	CommonParameter `yaml:",inline"`
}

func (p *BooleanParameter) Parse(v any) (any, error) {
	v, ok := v.(bool)
	if !ok {
		return nil, &ParseTypeError{p.Name, p.Type, v}
	}
	return v, nil
}

// ParseParams is a helper function for parsing Parameters from an arbitraryJSON object.
func ParseParams(ps Parameters, data map[string]any) ([]any, error) {
	params := []any{}
	for _, p := range ps {
		v, ok := data[p.GetName()]
		if !ok {
			return nil, fmt.Errorf("Parameter %q is required!", p.GetName())
		}
		v, err := p.Parse(v)
		if err != nil {
			return nil, fmt.Errorf("unable to parse value for %q: %w", p.GetName(), err)
		}
		params = append(params, v)
	}
	return params, nil
}
