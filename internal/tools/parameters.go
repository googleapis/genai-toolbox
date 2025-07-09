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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/googleapis/genai-toolbox/internal/util"
)

const (
	typeString = "string"
	typeInt    = "integer"
	typeFloat  = "float"
	typeBool   = "boolean"
	typeArray  = "array"
)

// ParamValues is an ordered list of ParamValue
type ParamValues []ParamValue

// ParamValue represents the parameter's name and value.
type ParamValue struct {
	Name  string
	Value any
}

// AsSlice returns a slice of the Param's values (in order).
func (p ParamValues) AsSlice() []any {
	params := []any{}

	for _, p := range p {
		params = append(params, p.Value)
	}
	return params
}

// AsMap returns a map of ParamValue's names to values.
func (p ParamValues) AsMap() map[string]interface{} {
	params := make(map[string]interface{})
	for _, p := range p {
		params[p.Name] = p.Value
	}
	return params
}

// AsMapByOrderedKeys returns a map of a key's position to it's value, as necessary for Spanner PSQL.
// Example { $1 -> "value1", $2 -> "value2" }
func (p ParamValues) AsMapByOrderedKeys() map[string]interface{} {
	params := make(map[string]interface{})

	for i, p := range p {
		key := fmt.Sprintf("p%d", i+1)
		params[key] = p.Value
	}
	return params
}

// AsMapWithDollarPrefix ensures all keys are prefixed with a dollar sign for Dgraph.
// Example:
// Input:  {"role": "admin", "$age": 30}
// Output: {"$role": "admin", "$age": 30}
func (p ParamValues) AsMapWithDollarPrefix() map[string]interface{} {
	params := make(map[string]interface{})

	for _, param := range p {
		key := param.Name
		if !strings.HasPrefix(key, "$") {
			key = "$" + key
		}
		params[key] = param.Value
	}
	return params
}

func parseFromAuthService(paramAuthServices []ParamAuthService, claimsMap map[string]map[string]any) (any, error) {
	// parse a parameter from claims using its specified auth services
	for _, a := range paramAuthServices {
		claims, ok := claimsMap[a.Name]
		if !ok {
			// not validated for this authservice, skip to the next one
			continue
		}
		v, ok := claims[a.Field]
		if !ok {
			// claims do not contain specified field
			return nil, fmt.Errorf("no field named %s in claims", a.Field)
		}
		return v, nil
	}
	return nil, fmt.Errorf("missing or invalid authentication header")
}

// ParseParams is a helper function for parsing Parameters from an arbitraryJSON object.
func ParseParams(ps Parameters, data map[string]any, claimsMap map[string]map[string]any) (ParamValues, error) {
	params := make([]ParamValue, 0, len(ps))
	for _, p := range ps {
		var v any
		paramAuthServices := p.GetAuthServices()
		name := p.GetName()
		if len(paramAuthServices) == 0 {
			// parse non auth-required parameter
			var ok bool
			v, ok = data[name]
			if !ok {
				v = p.GetDefault()
				if v == nil {
					return nil, fmt.Errorf("parameter %q is required", name)
				}
			}
		} else {
			// parse authenticated parameter
			var err error
			v, err = parseFromAuthService(paramAuthServices, claimsMap)
			if err != nil {
				return nil, fmt.Errorf("error parsing authenticated parameter %q: %w", name, err)
			}
		}
		newV, err := p.Parse(v)
		if err != nil {
			return nil, fmt.Errorf("unable to parse value for %q: %w", name, err)
		}
		params = append(params, ParamValue{Name: name, Value: newV})
	}
	return params, nil
}

// helper function to convert a string array parameter to a comma separated string
func ConvertArrayParamToString(param any) (string, error) {
	switch v := param.(type) {
	case []any:
		var stringValues []string
		for _, item := range v {
			stringVal, ok := item.(string)
			if !ok {
				return "", fmt.Errorf("templateParameter only supports string arrays")
			}
			stringValues = append(stringValues, stringVal)
		}
		return strings.Join(stringValues, ", "), nil
	default:
		return "", fmt.Errorf("invalid parameter type, expected array of type string")
	}
}

// GetParams return the ParamValues that are associated with the Parameters.
func GetParams(params Parameters, paramValuesMap map[string]any) (ParamValues, error) {
	resultParamValues := make(ParamValues, 0)
	for _, p := range params {
		k := p.GetName()
		v, ok := paramValuesMap[k]
		if !ok {
			return nil, fmt.Errorf("missing parameter %s", k)
		}
		resultParamValues = append(resultParamValues, ParamValue{Name: k, Value: v})
	}
	return resultParamValues, nil
}

func ResolveTemplateParams(templateParams Parameters, originalStatement string, paramsMap map[string]any) (string, error) {
	templateParamsValues, err := GetParams(templateParams, paramsMap)
	templateParamsMap := templateParamsValues.AsMap()
	if err != nil {
		return "", fmt.Errorf("error getting template params %s", err)
	}

	funcMap := template.FuncMap{
		"array": ConvertArrayParamToString,
	}
	t, err := template.New("statement").Funcs(funcMap).Parse(originalStatement)
	if err != nil {
		return "", fmt.Errorf("error creating go template %s", err)
	}
	var result bytes.Buffer
	err = t.Execute(&result, templateParamsMap)
	if err != nil {
		return "", fmt.Errorf("error executing go template %s", err)
	}

	modifiedStatement := result.String()
	return modifiedStatement, nil
}

// ProcessParameters concatenate templateParameters and parameters from a tool.
// It returns a list of concatenated parameters, concatenated Toolbox manifest, and concatenated MCP Manifest.
func ProcessParameters(templateParams Parameters, params Parameters) (Parameters, []ParameterManifest, McpToolsSchema) {
	allParameters := slices.Concat(params, templateParams)

	paramManifest := slices.Concat(
		params.Manifest(),
		templateParams.Manifest(),
	)
	if paramManifest == nil {
		paramManifest = make([]ParameterManifest, 0)
	}

	parametersMcpManifest := params.McpManifest()
	templateParametersMcpManifest := templateParams.McpManifest()

	// Concatenate parameters for MCP `required` field
	concatRequiredManifest := slices.Concat(
		parametersMcpManifest.Required,
		templateParametersMcpManifest.Required,
	)
	if concatRequiredManifest == nil {
		concatRequiredManifest = []string{}
	}

	// Concatenate parameters for MCP `properties` field
	concatPropertiesManifest := make(map[string]ParameterMcpManifest)
	for name, p := range parametersMcpManifest.Properties {
		concatPropertiesManifest[name] = p
	}
	for name, p := range templateParametersMcpManifest.Properties {
		concatPropertiesManifest[name] = p
	}

	// Create a new McpToolsSchema with all parameters
	paramMcpManifest := McpToolsSchema{
		Type:       "object",
		Properties: concatPropertiesManifest,
		Required:   concatRequiredManifest,
	}
	return allParameters, paramManifest, paramMcpManifest
}

type Parameter interface {
	// Note: It's typically not idiomatic to include "Get" in the function name,
	// but this is done to differentiate it from the fields in CommonParameter.
	GetName() string
	GetType() string
	GetItems() Parameter
	GetDefault() any
	GetAuthServices() []ParamAuthService
	Parse(any) (any, error)
	Manifest() ParameterManifest
	McpManifest() ParameterMcpManifest
}

// McpToolsSchema is the representation of input schema for McpManifest.
type McpToolsSchema struct {
	Type       string                          `json:"type"`
	Properties map[string]ParameterMcpManifest `json:"properties"`
	Required   []string                        `json:"required"`
}

// Parameters is a type used to allow unmarshal a list of parameters
type Parameters []Parameter

func (c *Parameters) UnmarshalYAML(ctx context.Context, unmarshal func(interface{}) error) error {
	*c = make(Parameters, 0)
	var rawList []util.DelayedUnmarshaler
	if err := unmarshal(&rawList); err != nil {
		return err
	}
	for _, u := range rawList {
		p, err := parseParamFromDelayedUnmarshaler(ctx, &u)
		if err != nil {
			return err
		}
		(*c) = append((*c), p)
	}
	return nil
}

// parseParamFromDelayedUnmarshaler is a helper function that is required to parse
// parameters because there are multiple different types
func parseParamFromDelayedUnmarshaler(ctx context.Context, u *util.DelayedUnmarshaler) (Parameter, error) {
	var p map[string]any
	err := u.Unmarshal(&p)
	if err != nil {
		return nil, fmt.Errorf("error parsing parameters: %w", err)
	}

	t, ok := p["type"]
	if !ok {
		return nil, fmt.Errorf("parameter is missing 'type' field: %w", err)
	}

	dec, err := util.NewStrictDecoder(p)
	if err != nil {
		return nil, fmt.Errorf("error creating decoder: %w", err)
	}
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	switch t {
	case typeString:
		a := &StringParameter{}
		if err := dec.DecodeContext(ctx, a); err != nil {
			return nil, fmt.Errorf("unable to parse as %q: %w", t, err)
		}
		if a.AuthSources != nil {
			logger.WarnContext(ctx, "`authSources` is deprecated, use `authServices` for parameters instead")
			a.AuthServices = append(a.AuthServices, a.AuthSources...)
			a.AuthSources = nil
		}
		return a, nil
	case typeInt:
		a := &IntParameter{}
		if err := dec.DecodeContext(ctx, a); err != nil {
			return nil, fmt.Errorf("unable to parse as %q: %w", t, err)
		}
		if a.AuthSources != nil {
			logger.WarnContext(ctx, "`authSources` is deprecated, use `authServices` for parameters instead")
			a.AuthServices = append(a.AuthServices, a.AuthSources...)
			a.AuthSources = nil
		}
		return a, nil
	case typeFloat:
		a := &FloatParameter{}
		if err := dec.DecodeContext(ctx, a); err != nil {
			return nil, fmt.Errorf("unable to parse as %q: %w", t, err)
		}
		if a.AuthSources != nil {
			logger.WarnContext(ctx, "`authSources` is deprecated, use `authServices` for parameters instead")
			a.AuthServices = append(a.AuthServices, a.AuthSources...)
			a.AuthSources = nil
		}
		return a, nil
	case typeBool:
		a := &BooleanParameter{}
		if err := dec.DecodeContext(ctx, a); err != nil {
			return nil, fmt.Errorf("unable to parse as %q: %w", t, err)
		}
		if a.AuthSources != nil {
			logger.WarnContext(ctx, "`authSources` is deprecated, use `authServices` for parameters instead")
			a.AuthServices = append(a.AuthServices, a.AuthSources...)
			a.AuthSources = nil
		}
		return a, nil
	case typeArray:
		a := &ArrayParameter{}
		if err := dec.DecodeContext(ctx, a); err != nil {
			return nil, fmt.Errorf("unable to parse as %q: %w", t, err)
		}
		if a.AuthSources != nil {
			logger.WarnContext(ctx, "`authSources` is deprecated, use `authServices` for parameters instead")
			a.AuthServices = append(a.AuthServices, a.AuthSources...)
			a.AuthSources = nil
		}
		return a, nil
	}
	return nil, fmt.Errorf("%q is not valid type for a parameter", t)
}

func (ps Parameters) Manifest() []ParameterManifest {
	rtn := make([]ParameterManifest, 0, len(ps))
	for _, p := range ps {
		rtn = append(rtn, p.Manifest())
	}
	return rtn
}

func (ps Parameters) McpManifest() McpToolsSchema {
	properties := make(map[string]ParameterMcpManifest)
	required := make([]string, 0)

	for _, p := range ps {
		name := p.GetName()
		properties[name] = p.McpManifest()
		// parameters that doesn't have a default value are added to the required field
		if p.GetDefault() == nil {
			required = append(required, name)
		}
	}

	return McpToolsSchema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}
}

// ParameterManifest represents parameters when served as part of a ToolManifest.
type ParameterManifest struct {
	Name         string             `json:"name"`
	Type         string             `json:"type"`
	Required     bool               `json:"required"`
	Description  string             `json:"description"`
	AuthServices []string           `json:"authSources"`
	Items        *ParameterManifest `json:"items,omitempty"`
}

// ParameterMcpManifest represents properties when served as part of a ToolMcpManifest.
type ParameterMcpManifest struct {
	Type        string                `json:"type"`
	Description string                `json:"description"`
	Items       *ParameterMcpManifest `json:"items,omitempty"`
}

// CommonParameter are default fields that are emebdding in most Parameter implementations. Embedding this stuct will give the object Name() and Type() functions.
type CommonParameter struct {
	Name         string             `yaml:"name" validate:"required"`
	Type         string             `yaml:"type" validate:"required"`
	Desc         string             `yaml:"description" validate:"required"`
	AuthServices []ParamAuthService `yaml:"authServices"`
	AuthSources  []ParamAuthService `yaml:"authSources"` // Deprecated: Kept for compatibility.
}

// GetName returns the name specified for the Parameter.
func (p *CommonParameter) GetName() string {
	return p.Name
}

// GetType returns the type specified for the Parameter.
func (p *CommonParameter) GetType() string {
	return p.Type
}

// McpManifest returns the MCP manifest for the Parameter.
func (p *CommonParameter) McpManifest() ParameterMcpManifest {
	return ParameterMcpManifest{
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

type ParamAuthService struct {
	Name  string `yaml:"name"`
	Field string `yaml:"field"`
}

// NewStringParameter is a convenience function for initializing a StringParameter.
func NewStringParameter(name string, desc string) *StringParameter {
	return &StringParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeString,
			Desc:         desc,
			AuthServices: nil,
		},
	}
}

// NewStringParameterWithDefault is a convenience function for initializing a StringParameter with default value.
func NewStringParameterWithDefault(name string, defaultV, desc string) *StringParameter {
	return &StringParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeString,
			Desc:         desc,
			AuthServices: nil,
		},
		Default: &defaultV,
	}
}

// NewStringParameterWithAuth is a convenience function for initializing a StringParameter with a list of ParamAuthService.
func NewStringParameterWithAuth(name string, desc string, authServices []ParamAuthService) *StringParameter {
	return &StringParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeString,
			Desc:         desc,
			AuthServices: authServices,
		},
	}
}

var _ Parameter = &StringParameter{}

// StringParameter is a parameter representing the "string" type.
type StringParameter struct {
	CommonParameter `yaml:",inline"`
	Default         *string `yaml:"default"`
}

// Parse casts the value "v" as a "string".
func (p *StringParameter) Parse(v any) (any, error) {
	newV, ok := v.(string)
	if !ok {
		return nil, &ParseTypeError{p.Name, p.Type, v}
	}
	return newV, nil
}

func (p *StringParameter) GetAuthServices() []ParamAuthService {
	return p.AuthServices
}

func (p *StringParameter) GetDefault() any {
	if p.Default == nil {
		return nil
	}
	return *p.Default
}

func (p *StringParameter) GetItems() Parameter {
	return nil
}

// Manifest returns the manifest for the StringParameter.
func (p *StringParameter) Manifest() ParameterManifest {
	// only list ParamAuthService names (without fields) in manifest
	authNames := make([]string, len(p.AuthServices))
	for i, a := range p.AuthServices {
		authNames[i] = a.Name
	}
	required := p.Default == nil
	return ParameterManifest{
		Name:         p.Name,
		Type:         p.Type,
		Required:     required,
		Description:  p.Desc,
		AuthServices: authNames,
	}
}

// NewIntParameter is a convenience function for initializing a IntParameter.
func NewIntParameter(name string, desc string) *IntParameter {
	return &IntParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeInt,
			Desc:         desc,
			AuthServices: nil,
		},
	}
}

// NewIntParameterWithDefault is a convenience function for initializing a IntParameter with default value.
func NewIntParameterWithDefault(name string, defaultV int, desc string) *IntParameter {
	return &IntParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeInt,
			Desc:         desc,
			AuthServices: nil,
		},
		Default: &defaultV,
	}
}

// NewIntParameterWithAuth is a convenience function for initializing a IntParameter with a list of ParamAuthService.
func NewIntParameterWithAuth(name string, desc string, authServices []ParamAuthService) *IntParameter {
	return &IntParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeInt,
			Desc:         desc,
			AuthServices: authServices,
		},
	}
}

var _ Parameter = &IntParameter{}

// IntParameter is a parameter representing the "int" type.
type IntParameter struct {
	CommonParameter `yaml:",inline"`
	Default         *int `yaml:"default"`
}

func (p *IntParameter) Parse(v any) (any, error) {
	var out int
	switch newV := v.(type) {
	default:
		return nil, &ParseTypeError{p.Name, p.Type, v}
	case int:
		out = int(newV)
	case int32:
		out = int(newV)
	case int64:
		out = int(newV)
	case json.Number:
		newI, err := newV.Int64()
		if err != nil {
			return nil, &ParseTypeError{p.Name, p.Type, v}
		}
		out = int(newI)
	}
	return out, nil
}

func (p *IntParameter) GetAuthServices() []ParamAuthService {
	return p.AuthServices
}

func (p *IntParameter) GetDefault() any {
	if p.Default == nil {
		return nil
	}
	return *p.Default
}

func (p *IntParameter) GetItems() Parameter {
	return nil
}

// Manifest returns the manifest for the IntParameter.
func (p *IntParameter) Manifest() ParameterManifest {
	// only list ParamAuthService names (without fields) in manifest
	authNames := make([]string, len(p.AuthServices))
	for i, a := range p.AuthServices {
		authNames[i] = a.Name
	}
	required := p.Default == nil
	return ParameterManifest{
		Name:         p.Name,
		Type:         p.Type,
		Required:     required,
		Description:  p.Desc,
		AuthServices: authNames,
	}
}

// NewFloatParameter is a convenience function for initializing a FloatParameter.
func NewFloatParameter(name string, desc string) *FloatParameter {
	return &FloatParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeFloat,
			Desc:         desc,
			AuthServices: nil,
		},
	}
}

// NewFloatParameterWithDefault is a convenience function for initializing a FloatParameter with default value.
func NewFloatParameterWithDefault(name string, defaultV float64, desc string) *FloatParameter {
	return &FloatParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeFloat,
			Desc:         desc,
			AuthServices: nil,
		},
		Default: &defaultV,
	}
}

// NewFloatParameterWithAuth is a convenience function for initializing a FloatParameter with a list of ParamAuthService.
func NewFloatParameterWithAuth(name string, desc string, authServices []ParamAuthService) *FloatParameter {
	return &FloatParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeFloat,
			Desc:         desc,
			AuthServices: authServices,
		},
	}
}

var _ Parameter = &FloatParameter{}

// FloatParameter is a parameter representing the "float" type.
type FloatParameter struct {
	CommonParameter `yaml:",inline"`
	Default         *float64 `yaml:"default"`
}

func (p *FloatParameter) Parse(v any) (any, error) {
	var out float64
	switch newV := v.(type) {
	default:
		return nil, &ParseTypeError{p.Name, p.Type, v}
	case float32:
		out = float64(newV)
	case float64:
		out = newV
	case json.Number:
		newI, err := newV.Float64()
		if err != nil {
			return nil, &ParseTypeError{p.Name, p.Type, v}
		}
		out = float64(newI)
	}
	return out, nil
}

func (p *FloatParameter) GetAuthServices() []ParamAuthService {
	return p.AuthServices
}

func (p *FloatParameter) GetDefault() any {
	if p.Default == nil {
		return nil
	}
	return *p.Default
}

func (p *FloatParameter) GetItems() Parameter {
	return nil
}

// Manifest returns the manifest for the FloatParameter.
func (p *FloatParameter) Manifest() ParameterManifest {
	// only list ParamAuthService names (without fields) in manifest
	authNames := make([]string, len(p.AuthServices))
	for i, a := range p.AuthServices {
		authNames[i] = a.Name
	}
	required := p.Default == nil
	return ParameterManifest{
		Name:         p.Name,
		Type:         p.Type,
		Required:     required,
		Description:  p.Desc,
		AuthServices: authNames,
	}
}

// NewBooleanParameter is a convenience function for initializing a BooleanParameter.
func NewBooleanParameter(name string, desc string) *BooleanParameter {
	return &BooleanParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeBool,
			Desc:         desc,
			AuthServices: nil,
		},
	}
}

// NewBooleanParameterWithDefault is a convenience function for initializing a BooleanParameter with default value.
func NewBooleanParameterWithDefault(name string, defaultV bool, desc string) *BooleanParameter {
	return &BooleanParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeBool,
			Desc:         desc,
			AuthServices: nil,
		},
		Default: &defaultV,
	}
}

// NewBooleanParameterWithAuth is a convenience function for initializing a BooleanParameter with a list of ParamAuthService.
func NewBooleanParameterWithAuth(name string, desc string, authServices []ParamAuthService) *BooleanParameter {
	return &BooleanParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeBool,
			Desc:         desc,
			AuthServices: authServices,
		},
	}
}

var _ Parameter = &BooleanParameter{}

// BooleanParameter is a parameter representing the "boolean" type.
type BooleanParameter struct {
	CommonParameter `yaml:",inline"`
	Default         *bool `yaml:"default"`
}

func (p *BooleanParameter) Parse(v any) (any, error) {
	newV, ok := v.(bool)
	if !ok {
		return nil, &ParseTypeError{p.Name, p.Type, v}
	}
	return newV, nil
}

func (p *BooleanParameter) GetAuthServices() []ParamAuthService {
	return p.AuthServices
}

func (p *BooleanParameter) GetDefault() any {
	if p.Default == nil {
		return nil
	}
	return *p.Default
}

func (p *BooleanParameter) GetItems() Parameter {
	return nil
}

// Manifest returns the manifest for the BooleanParameter.
func (p *BooleanParameter) Manifest() ParameterManifest {
	// only list ParamAuthService names (without fields) in manifest
	authNames := make([]string, len(p.AuthServices))
	for i, a := range p.AuthServices {
		authNames[i] = a.Name
	}
	required := p.Default == nil
	return ParameterManifest{
		Name:         p.Name,
		Type:         p.Type,
		Required:     required,
		Description:  p.Desc,
		AuthServices: authNames,
	}
}

// NewArrayParameter is a convenience function for initializing a ArrayParameter.
func NewArrayParameter(name string, desc string, items Parameter) *ArrayParameter {
	return &ArrayParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeArray,
			Desc:         desc,
			AuthServices: nil,
		},
		Items: items,
	}
}

// NewArrayParameterWithDefault is a convenience function for initializing a ArrayParameter with default value.
func NewArrayParameterWithDefault(name string, defaultV []any, desc string, items Parameter) *ArrayParameter {
	return &ArrayParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeArray,
			Desc:         desc,
			AuthServices: nil,
		},
		Items:   items,
		Default: &defaultV,
	}
}

// NewArrayParameterWithAuth is a convenience function for initializing a ArrayParameter with a list of ParamAuthService.
func NewArrayParameterWithAuth(name string, desc string, items Parameter, authServices []ParamAuthService) *ArrayParameter {
	return &ArrayParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeArray,
			Desc:         desc,
			AuthServices: authServices,
		},
		Items: items,
	}
}

var _ Parameter = &ArrayParameter{}

// ArrayParameter is a parameter representing the "array" type.
type ArrayParameter struct {
	CommonParameter `yaml:",inline"`
	Default         *[]any    `yaml:"default"`
	Items           Parameter `yaml:"items"`
}

func (p *ArrayParameter) UnmarshalYAML(ctx context.Context, unmarshal func(interface{}) error) error {
	var rawItem struct {
		CommonParameter `yaml:",inline"`
		Default         *[]any                  `yaml:"default"`
		Items           util.DelayedUnmarshaler `yaml:"items"`
	}
	if err := unmarshal(&rawItem); err != nil {
		return err
	}
	p.CommonParameter = rawItem.CommonParameter
	p.Default = rawItem.Default
	i, err := parseParamFromDelayedUnmarshaler(ctx, &rawItem.Items)
	if err != nil {
		return fmt.Errorf("unable to parse 'items' field: %w", err)
	}
	if i.GetAuthServices() != nil && len(i.GetAuthServices()) != 0 {
		return fmt.Errorf("nested items should not have auth services")
	}
	p.Items = i

	return nil
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

func (p *ArrayParameter) GetAuthServices() []ParamAuthService {
	return p.AuthServices
}

func (p *ArrayParameter) GetDefault() any {
	if p.Default == nil {
		return nil
	}
	return *p.Default
}

func (p *ArrayParameter) GetItems() Parameter {
	return p.Items
}

// Manifest returns the manifest for the ArrayParameter.
func (p *ArrayParameter) Manifest() ParameterManifest {
	// only list ParamAuthService names (without fields) in manifest
	authNames := make([]string, len(p.AuthServices))
	for i, a := range p.AuthServices {
		authNames[i] = a.Name
	}
	items := p.Items.Manifest()
	required := p.Default == nil
	items.Required = required
	return ParameterManifest{
		Name:         p.Name,
		Type:         p.Type,
		Required:     required,
		Description:  p.Desc,
		AuthServices: authNames,
		Items:        &items,
	}
}

// McpManifest returns the MCP manifest for the ArrayParameter.
func (p *ArrayParameter) McpManifest() ParameterMcpManifest {
	// only list ParamAuthService names (without fields) in manifest
	authNames := make([]string, len(p.AuthServices))
	for i, a := range p.AuthServices {
		authNames[i] = a.Name
	}
	items := p.Items.McpManifest()
	return ParameterMcpManifest{
		Type:        p.Type,
		Description: p.Desc,
		Items:       &items,
	}
}
