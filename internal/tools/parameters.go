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
	typeInt    = "integer"
	typeFloat  = "float"
	typeBool   = "boolean"
	typeArray  = "array"
)

type ParamValues []ParamValue

type ParamValue struct {
	Name  string
	Value any
}

func (p ParamValues) SliceValuesPG() []any {
	params := []any{}

	for _, p := range p {
		params = append(params, p.Value)
	}
	return params
}

func (p ParamValues) MapValuesPG() map[string]interface{} {
	params := make(map[string]interface{})

	for i, p := range p {
		key := fmt.Sprintf("p%d", i+1)
		params[key] = p.Value
	}
	return params
}

func (p ParamValues) MapValuesGSQL() map[string]interface{} {
	params := make(map[string]interface{})

	for _, p := range p {
		params[p.Name] = p.Value
	}
	return params
}

// ParseParams is a helper function for parsing Parameters from an arbitraryJSON object.
func ParseParams(ps Parameters, data map[string]any) (ParamValues, error) {
	params := []ParamValue{}
	for _, p := range ps {
		name := p.GetName()
		v, ok := data[name]
		if !ok {
			return nil, fmt.Errorf("parameter %q is required!", p.GetName())
		}
		newV, err := p.Parse(v)
		if err != nil {
			return nil, fmt.Errorf("unable to parse value for %q: %w", p.GetName(), err)
		}
		params = append(params, ParamValue{Name: name, Value: newV})
	}
	return params, nil
}

type Parameter interface {
	// Note: It's typically not idiomatic to include "Get" in the function name,
	// but this is done to differentiate it from the fields in CommonParameter.
	GetName() string
	GetType() string
	Parse(any) (any, error)
	Manifest() ParameterManifest
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
		p, err := parseFromYamlNode(&n)
		if err != nil {
			return err
		}
		(*c) = append((*c), p)
	}
	return nil
}

func parseFromYamlNode(node *yaml.Node) (Parameter, error) {
	var p CommonParameter
	err := node.Decode(&p)
	if err != nil {
		return nil, fmt.Errorf("parameter missing required fields")
	}
	switch p.Type {
	case typeString:
		a := &StringParameter{}
		if err := node.Decode(a); err != nil {
			return nil, fmt.Errorf("unable to parse as %q: %w", p.Type, err)
		}
		return a, nil
	case typeInt:
		a := &IntParameter{}
		if err := node.Decode(a); err != nil {
			return nil, fmt.Errorf("unable to parse as %q: %w", p.Type, err)
		}
		return a, nil
	case typeFloat:
		a := &FloatParameter{}
		if err := node.Decode(a); err != nil {
			return nil, fmt.Errorf("unable to parse as %q: %w", p.Type, err)
		}
		return a, nil
	case typeBool:
		a := &BooleanParameter{}
		if err := node.Decode(a); err != nil {
			return nil, fmt.Errorf("unable to parse as %q: %w", p.Type, err)
		}
		return a, nil
	case typeArray:
		a := &ArrayParameter{}
		if err := node.Decode(a); err != nil {
			return nil, fmt.Errorf("unable to parse as %q: %w", p.Type, err)
		}
		return a, nil
	}
	return nil, fmt.Errorf("%q is not valid type for a parameter!", p.Type)
}

func (ps Parameters) Manifest() []ParameterManifest {
	rtn := make([]ParameterManifest, 0, len(ps))
	for _, p := range ps {
		rtn = append(rtn, p.Manifest())
	}
	return rtn
}

// ParameterManifest represents parameters when served as part of a ToolManifest.
type ParameterManifest struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	// Parameter   *ParameterManifest `json:"parameter,omitempty"`
}

// CommonParameter are default fields that are emebdding in most Parameter implementations. Embedding this stuct will give the object Name() and Type() functions.
type CommonParameter struct {
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

// GetType returns the type specified for the Parameter.
func (p *CommonParameter) Manifest() ParameterManifest {
	return ParameterManifest{
		Name:        p.Name,
		Type:        p.Type,
		Description: p.Desc,
	}
}

// ParseTypeError is a custom error for incorrectly typed Parameters.
type ParseTypeError struct {
	Name  string
	Type  string
	Value any
}

func (e ParseTypeError) Error() string {
	return fmt.Sprintf("%q not type %q", e.Value, e.Type)
}

// NewStringParameter is a convenience function for initializing a StringParameter.
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
	newV, ok := v.(string)
	if !ok {
		return nil, &ParseTypeError{p.Name, p.Type, v}
	}
	return newV, nil
}

// NewIntParameter is a convenience function for initializing a IntParameter.
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
	newV, ok := v.(int)
	if !ok {
		return nil, &ParseTypeError{p.Name, p.Type, v}
	}
	return newV, nil
}

// NewFloatParameter is a convenience function for initializing a FloatParameter.
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
	newV, ok := v.(float64)
	if !ok {
		return nil, &ParseTypeError{p.Name, p.Type, v}
	}
	return newV, nil
}

// NewBooleanParameter is a convenience function for initializing a BooleanParameter.
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
	newV, ok := v.(bool)
	if !ok {
		return nil, &ParseTypeError{p.Name, p.Type, v}
	}
	return newV, nil
}

// NewArrayParameter is a convenience function for initializing an ArrayParameter.
func NewArrayParameter(name, desc string, items Parameter) *ArrayParameter {
	return &ArrayParameter{
		CommonParameter: CommonParameter{
			Name: name,
			Type: typeArray,
			Desc: desc,
		},
		Items: items,
	}
}

var _ Parameter = &ArrayParameter{}

// ArrayParameter is a parameter representing the "array" type.
type ArrayParameter struct {
	CommonParameter `yaml:",inline"`
	Items           Parameter `yaml:"items"`
}

func (p *ArrayParameter) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&p.CommonParameter); err != nil {
		return err
	}
	// Find the node that represents the "items" field name
	idx, ok := findIdxByValue(node.Content, "items")
	if !ok {
		return fmt.Errorf("array parameter missing 'items' field!")
	}
	// Parse items from the "value" of "items" field
	i, err := parseFromYamlNode(node.Content[idx+1])
	if err != nil {
		return fmt.Errorf("unable to parse 'items' field: %w", err)
	}
	p.Items = i
	return nil
}

// findIdxByValue returns the index of the first node where value matches
func findIdxByValue(nodes []*yaml.Node, value string) (int, bool) {
	for idx, n := range nodes {
		if n.Value == value {
			return idx, true
		}
	}
	return 0, false
}

func (p *ArrayParameter) Parse(v any) (any, error) {
	arrVal, ok := v.([]any)
	if !ok {
		return nil, &ParseTypeError{p.Name, p.Type, arrVal}
	}
	rtn := make([]any, 0, len(arrVal))
	for idx, val := range arrVal {
		val, err := p.Items.Parse(val)
		if err != nil {
			return nil, fmt.Errorf("unable to parse element #%d: %w", idx, err)
		}
		rtn = append(rtn, val)
	}
	return rtn, nil
}
