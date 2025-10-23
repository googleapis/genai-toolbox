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
	"reflect"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"github.com/googleapis/genai-toolbox/internal/util"
)

const (
	TypeString = "string"
	TypeInt    = "integer"
	TypeFloat  = "float"
	TypeBool   = "boolean"
	TypeArray  = "array"
	TypeMap    = "map"
)

// delimiters for string parameter escaping
const (
	escapeBackticks      = "backticks"
	escapeDoubleQuotes   = "double-quotes"
	escapeSingleQuotes   = "single-quotes"
	escapeSquareBrackets = "square-brackets"
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
	return nil, fmt.Errorf("missing or invalid authentication header: %w", ErrUnauthorized)
}

// CheckParamRequired checks if a parameter is required based on the required and default field.
func CheckParamRequired(required bool, defaultV any) bool {
	return required && defaultV == nil
}

// ParseParams is a helper function for parsing Parameters from an arbitraryJSON object.
func ParseParams(ps Parameters, data map[string]any, claimsMap map[string]map[string]any) (ParamValues, error) {
	params := make([]ParamValue, 0, len(ps))
	for _, p := range ps {
		var v, newV any
		var err error
		paramAuthServices := p.GetAuthServices()
		name := p.GetName()
		if len(paramAuthServices) == 0 {
			// parse non auth-required parameter
			var ok bool
			v, ok = data[name]
			if !ok {
				v = p.GetDefault()
				// if the parameter is required and no value given, throw an error
				if CheckParamRequired(p.GetRequired(), v) {
					return nil, fmt.Errorf("parameter %q is required", name)
				}
			}
		} else {
			// parse authenticated parameter
			v, err = parseFromAuthService(paramAuthServices, claimsMap)
			if err != nil {
				return nil, fmt.Errorf("error parsing authenticated parameter %q: %w", name, err)
			}
		}
		if v != nil {
			newV, err = p.Parse(v)
			if err != nil {
				return nil, fmt.Errorf("unable to parse value for %q: %w", name, err)
			}
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
func ProcessParameters(templateParams Parameters, params Parameters) (Parameters, []ParameterManifest, error) {
	allParameters := slices.Concat(params, templateParams)

	// verify no duplicate parameter names
	err := CheckDuplicateParameters(allParameters)
	if err != nil {
		return nil, nil, err
	}

	// create Toolbox manifest
	paramManifest := allParameters.Manifest()
	if paramManifest == nil {
		paramManifest = make([]ParameterManifest, 0)
	}

	return allParameters, paramManifest, nil
}

type Parameter interface {
	// Note: It's typically not idiomatic to include "Get" in the function name,
	// but this is done to differentiate it from the fields in CommonParameter.
	GetName() string
	GetType() string
	GetDefault() any
	GetRequired() bool
	GetAuthServices() []ParamAuthService
	Parse(any) (any, error)
	Manifest() ParameterManifest
	McpManifest() (ParameterMcpManifest, []string)
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
		return nil, fmt.Errorf("parameter is missing 'type' field")
	}

	return ParseParameter(ctx, p, t.(string))
}

// ParseParameter parses a raw map into a Parameter object based on its "type" field.
func ParseParameter(ctx context.Context, p map[string]any, paramType string) (Parameter, error) {
	dec, err := util.NewStrictDecoder(p)
	if err != nil {
		return nil, fmt.Errorf("error creating decoder: %w", err)
	}
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	switch paramType {
	case TypeString:
		a := &StringParameter{}
		if err := dec.DecodeContext(ctx, a); err != nil {
			return nil, fmt.Errorf("unable to parse as %q: %w", paramType, err)
		}
		if a.AuthSources != nil {
			logger.WarnContext(ctx, "`authSources` is deprecated, use `authServices` for parameters instead")
			a.AuthServices = append(a.AuthServices, a.AuthSources...)
			a.AuthSources = nil
		}
		return a, nil
	case TypeInt:
		a := &IntParameter{}
		if err := dec.DecodeContext(ctx, a); err != nil {
			return nil, fmt.Errorf("unable to parse as %q: %w", paramType, err)
		}
		if a.AuthSources != nil {
			logger.WarnContext(ctx, "`authSources` is deprecated, use `authServices` for parameters instead")
			a.AuthServices = append(a.AuthServices, a.AuthSources...)
			a.AuthSources = nil
		}
		return a, nil
	case TypeFloat:
		a := &FloatParameter{}
		if err := dec.DecodeContext(ctx, a); err != nil {
			return nil, fmt.Errorf("unable to parse as %q: %w", paramType, err)
		}
		if a.AuthSources != nil {
			logger.WarnContext(ctx, "`authSources` is deprecated, use `authServices` for parameters instead")
			a.AuthServices = append(a.AuthServices, a.AuthSources...)
			a.AuthSources = nil
		}
		return a, nil
	case TypeBool:
		a := &BooleanParameter{}
		if err := dec.DecodeContext(ctx, a); err != nil {
			return nil, fmt.Errorf("unable to parse as %q: %w", paramType, err)
		}
		if a.AuthSources != nil {
			logger.WarnContext(ctx, "`authSources` is deprecated, use `authServices` for parameters instead")
			a.AuthServices = append(a.AuthServices, a.AuthSources...)
			a.AuthSources = nil
		}
		return a, nil
	case TypeArray:
		a := &ArrayParameter{}
		if err := dec.DecodeContext(ctx, a); err != nil {
			return nil, fmt.Errorf("unable to parse as %q: %w", paramType, err)
		}
		if a.AuthSources != nil {
			logger.WarnContext(ctx, "`authSources` is deprecated, use `authServices` for parameters instead")
			a.AuthServices = append(a.AuthServices, a.AuthSources...)
			a.AuthSources = nil
		}
		return a, nil
	case TypeMap:
		a := &MapParameter{}
		if err := dec.DecodeContext(ctx, a); err != nil {
			return nil, fmt.Errorf("unable to parse as %q: %w", paramType, err)
		}
		if a.AuthSources != nil {
			logger.WarnContext(ctx, "`authSources` is deprecated, use `authServices` for parameters instead")
			a.AuthServices = append(a.AuthServices, a.AuthSources...)
			a.AuthSources = nil
		}
		return a, nil
	}
	return nil, fmt.Errorf("%q is not valid type for a parameter", paramType)
}

func (ps Parameters) Manifest() []ParameterManifest {
	rtn := make([]ParameterManifest, 0, len(ps))
	for _, p := range ps {
		rtn = append(rtn, p.Manifest())
	}
	return rtn
}

func (ps Parameters) McpManifest() (McpToolsSchema, map[string][]string) {
	properties := make(map[string]ParameterMcpManifest)
	required := make([]string, 0)
	authParam := make(map[string][]string)

	for _, p := range ps {
		name := p.GetName()
		paramManifest, authParamList := p.McpManifest()
		properties[name] = paramManifest
		// parameters that doesn't have a default value are added to the required field
		if CheckParamRequired(p.GetRequired(), p.GetDefault()) {
			required = append(required, name)
		}
		if len(authParamList) > 0 {
			authParam[name] = authParamList
		}
	}
	return McpToolsSchema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}, authParam
}

// ParameterManifest represents parameters when served as part of a ToolManifest.
type ParameterManifest struct {
	Name                 string             `json:"name"`
	Type                 string             `json:"type"`
	Required             bool               `json:"required"`
	Description          string             `json:"description"`
	AuthServices         []string           `json:"authSources"`
	Items                *ParameterManifest `json:"items,omitempty"`
	AdditionalProperties any                `json:"additionalProperties,omitempty"`
}

// ParameterMcpManifest represents properties when served as part of a ToolMcpManifest.
type ParameterMcpManifest struct {
	Type                 string                `json:"type"`
	Description          string                `json:"description"`
	Items                *ParameterMcpManifest `json:"items,omitempty"`
	AdditionalProperties any                   `json:"additionalProperties,omitempty"`
}

// CommonParameter are default fields that are emebdding in most Parameter implementations. Embedding this stuct will give the object Name() and Type() functions.
type CommonParameter struct {
	Name          string             `yaml:"name" validate:"required"`
	Type          string             `yaml:"type" validate:"required"`
	Desc          string             `yaml:"description" validate:"required"`
	Required      *bool              `yaml:"required"`
	AllowedValues []any              `yaml:"allowedValues"`
	AuthServices  []ParamAuthService `yaml:"authServices"`
	AuthSources   []ParamAuthService `yaml:"authSources"` // Deprecated: Kept for compatibility.
}

// GetName returns the name specified for the Parameter.
func (p *CommonParameter) GetName() string {
	return p.Name
}

// GetType returns the type specified for the Parameter.
func (p *CommonParameter) GetType() string {
	return p.Type
}

// GetRequired returns the type specified for the Parameter.
func (p *CommonParameter) GetRequired() bool {
	// parameters are defaulted to required
	if p.Required == nil {
		return true
	}
	return *p.Required
}

// GetAllowedValues returns the allowed values for the Parameter.
func (p *CommonParameter) GetAllowedValues() []any {
	return p.AllowedValues
}

// IsAllowedValues checks if the value is allowed.
func (p *CommonParameter) IsAllowedValues(v any) bool {
	if len(p.AllowedValues) == 0 {
		return true
	}
	for _, av := range p.AllowedValues {
		if MatchStringOrRegex(v, av) {
			return true
		}
	}
	return false
}

// MatchStringOrRegex checks if the input matches the target
func MatchStringOrRegex(input, target any) bool {
	targetS, ok := target.(string)
	if !ok {
		return input == target
	}
	re, err := regexp.Compile(targetS)
	if err != nil {
		// if target is not regex, run direct comparison
		return input == target
	}
	inputS := fmt.Sprintf("%s", input)
	return re.MatchString(inputS)
}

// McpManifest returns the MCP manifest for the Parameter.
func (p *CommonParameter) McpManifest() (ParameterMcpManifest, []string) {
	authServiceNames := getAuthServiceNames(p.AuthServices)
	return ParameterMcpManifest{
		Type:        p.Type,
		Description: p.Desc,
	}, authServiceNames
}

// getAuthServiceNames retrieves the list of auth services names
func getAuthServiceNames(authServices []ParamAuthService) []string {
	authServiceNames := make([]string, len(authServices))
	for i, a := range authServices {
		authServiceNames[i] = a.Name
	}
	return authServiceNames
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
			Type:         TypeString,
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
			Type:         TypeString,
			Desc:         desc,
			AuthServices: nil,
		},
		Default: &defaultV,
	}
}

// NewStringParameterWithEscape is a convenience function for initializing a StringParameter.
func NewStringParameterWithEscape(name, desc, escape string) *StringParameter {
	return &StringParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeString,
			Desc:         desc,
			AuthServices: nil,
		},
		Escape: &escape,
	}
}

// NewStringParameterWithRequired is a convenience function for initializing a StringParameter.
func NewStringParameterWithRequired(name string, desc string, required bool) *StringParameter {
	return &StringParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         TypeString,
			Desc:         desc,
			Required:     &required,
			AuthServices: nil,
		},
	}
}

// NewStringParameterWithAuth is a convenience function for initializing a StringParameter with a list of ParamAuthService.
func NewStringParameterWithAuth(name string, desc string, authServices []ParamAuthService) *StringParameter {
	return &StringParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         TypeString,
			Desc:         desc,
			AuthServices: authServices,
		},
	}
}

// NewStringParameterWithAllowedValues is a convenience function for initializing a StringParameter with a list of allowedValues
func NewStringParameterWithAllowedValues(name string, desc string, allowedValues []any) *StringParameter {
	return &StringParameter{
		CommonParameter: CommonParameter{
			Name:          name,
			Type:          typeString,
			Desc:          desc,
			AllowedValues: allowedValues,
			AuthServices:  nil,
		},
	}
}

var _ Parameter = &StringParameter{}

// StringParameter is a parameter representing the "string" type.
type StringParameter struct {
	CommonParameter `yaml:",inline"`
	Default         *string `yaml:"default"`
	Escape          *string `yaml:"escape"`
}

// Parse casts the value "v" as a "string".
func (p *StringParameter) Parse(v any) (any, error) {
	newV, ok := v.(string)
	if !ok {
		return nil, &ParseTypeError{p.Name, p.Type, v}
	}
	if !p.IsAllowedValues(newV) {
		return nil, fmt.Errorf("%s is not an allowed value", newV)
	}
	if p.Escape != nil {
		return applyEscape(*p.Escape, newV)
	}
	return newV, nil
}

func applyEscape(escape, v string) (any, error) {
	switch escape {
	case escapeBackticks:
		return fmt.Sprintf("`%s`", v), nil
	case escapeDoubleQuotes:
		return fmt.Sprintf(`"%s"`, v), nil
	case escapeSingleQuotes:
		return fmt.Sprintf(`'%s'`, v), nil
	case escapeSquareBrackets:
		return fmt.Sprintf("[%s]", v), nil
	default:
		return nil, fmt.Errorf("%s is not an allowed escaping delimiter", escape)
	}
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

// Manifest returns the manifest for the StringParameter.
func (p *StringParameter) Manifest() ParameterManifest {
	// only list ParamAuthService names (without fields) in manifest
	authServiceNames := getAuthServiceNames(p.AuthServices)
	r := CheckParamRequired(p.GetRequired(), p.GetDefault())
	return ParameterManifest{
		Name:         p.Name,
		Type:         p.Type,
		Required:     r,
		Description:  p.Desc,
		AuthServices: authServiceNames,
	}
}

// NewIntParameter is a convenience function for initializing a IntParameter.
func NewIntParameter(name string, desc string) *IntParameter {
	return &IntParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         TypeInt,
			Desc:         desc,
			AuthServices: nil,
		},
	}
}

// NewIntParameterWithRange is a convenience function for initializing a IntParameter.
func NewIntParameterWithRange(name string, desc string, minValue *int, maxValue *int) *IntParameter {
	return &IntParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeInt,
			Desc:         desc,
			AuthServices: nil,
		},
		MinValue: minValue,
		MaxValue: maxValue,
	}
}

// NewIntParameterWithDefault is a convenience function for initializing a IntParameter with default value.
func NewIntParameterWithDefault(name string, defaultV int, desc string) *IntParameter {
	return &IntParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         TypeInt,
			Desc:         desc,
			AuthServices: nil,
		},
		Default: &defaultV,
	}
}

// NewIntParameterWithRequired is a convenience function for initializing a IntParameter.
func NewIntParameterWithRequired(name string, desc string, required bool) *IntParameter {
	return &IntParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         TypeInt,
			Desc:         desc,
			Required:     &required,
			AuthServices: nil,
		},
	}
}

// NewIntParameterWithAuth is a convenience function for initializing a IntParameter with a list of ParamAuthService.
func NewIntParameterWithAuth(name string, desc string, authServices []ParamAuthService) *IntParameter {
	return &IntParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         TypeInt,
			Desc:         desc,
			AuthServices: authServices,
		},
	}
}

// NewIntParameterWithAllowedValues is a convenience function for initializing a IntParameter with a list of allowedValues
func NewIntParameterWithAllowedValues(name string, desc string, allowedValues []any) *IntParameter {
	return &IntParameter{
		CommonParameter: CommonParameter{
			Name:          name,
			Type:          typeString,
			Desc:          desc,
			AllowedValues: allowedValues,
			AuthServices:  nil,
		},
	}
}

var _ Parameter = &IntParameter{}

// IntParameter is a parameter representing the "int" type.
type IntParameter struct {
	CommonParameter `yaml:",inline"`
	Default         *int `yaml:"default"`
	MinValue        *int `yaml:"minValue"`
	MaxValue        *int `yaml:"maxValue"`
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
	if !p.IsAllowedValues(out) {
		return nil, fmt.Errorf("%d is not an allowed value", out)
	}
	if p.MinValue != nil && out < *p.MinValue {
		return nil, fmt.Errorf("%d is under the minimum value", out)
	}
	if p.MaxValue != nil && out > *p.MaxValue {
		return nil, fmt.Errorf("%d is above the maximum value", out)
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

// Manifest returns the manifest for the IntParameter.
func (p *IntParameter) Manifest() ParameterManifest {
	// only list ParamAuthService names (without fields) in manifest
	authServiceNames := getAuthServiceNames(p.AuthServices)
	r := CheckParamRequired(p.GetRequired(), p.GetDefault())
	return ParameterManifest{
		Name:         p.Name,
		Type:         p.Type,
		Required:     r,
		Description:  p.Desc,
		AuthServices: authServiceNames,
	}
}

// NewFloatParameter is a convenience function for initializing a FloatParameter.
func NewFloatParameter(name string, desc string) *FloatParameter {
	return &FloatParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         TypeFloat,
			Desc:         desc,
			AuthServices: nil,
		},
	}
}

// NewFloatParameterWithRange is a convenience function for initializing a FloatParameter.
func NewFloatParameterWithRange(name string, desc string, minValue *float64, maxValue *float64) *FloatParameter {
	return &FloatParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         typeFloat,
			Desc:         desc,
			AuthServices: nil,
		},
		MinValue: minValue,
		MaxValue: maxValue,
	}
}

// NewFloatParameterWithDefault is a convenience function for initializing a FloatParameter with default value.
func NewFloatParameterWithDefault(name string, defaultV float64, desc string) *FloatParameter {
	return &FloatParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         TypeFloat,
			Desc:         desc,
			AuthServices: nil,
		},
		Default: &defaultV,
	}
}

// NewFloatParameterWithRequired is a convenience function for initializing a FloatParameter.
func NewFloatParameterWithRequired(name string, desc string, required bool) *FloatParameter {
	return &FloatParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         TypeFloat,
			Desc:         desc,
			Required:     &required,
			AuthServices: nil,
		},
	}
}

// NewFloatParameterWithAuth is a convenience function for initializing a FloatParameter with a list of ParamAuthService.
func NewFloatParameterWithAuth(name string, desc string, authServices []ParamAuthService) *FloatParameter {
	return &FloatParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         TypeFloat,
			Desc:         desc,
			AuthServices: authServices,
		},
	}
}

// NewFloatParameterWithAllowedValues is a convenience function for initializing a FloatParameter with a list of allowed values.
func NewFloatParameterWithAllowedValues(name string, desc string, allowedValues []any) *FloatParameter {
	return &FloatParameter{
		CommonParameter: CommonParameter{
			Name:          name,
			Type:          typeFloat,
			Desc:          desc,
			AllowedValues: allowedValues,
			AuthServices:  nil,
		},
	}
}

var _ Parameter = &FloatParameter{}

// FloatParameter is a parameter representing the "float" type.
type FloatParameter struct {
	CommonParameter `yaml:",inline"`
	Default         *float64 `yaml:"default"`
	MinValue        *float64 `yaml:"minValue"`
	MaxValue        *float64 `yaml:"maxValue"`
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
	if !p.IsAllowedValues(out) {
		return nil, fmt.Errorf("%g is not an allowed value", out)
	}
	if p.MinValue != nil && out < *p.MinValue {
		return nil, fmt.Errorf("%g is under the minimum value", out)
	}
	if p.MaxValue != nil && out > *p.MaxValue {
		return nil, fmt.Errorf("%g is above the maximum value", out)
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

// Manifest returns the manifest for the FloatParameter.
func (p *FloatParameter) Manifest() ParameterManifest {
	// only list ParamAuthService names (without fields) in manifest
	authServiceNames := getAuthServiceNames(p.AuthServices)
	r := CheckParamRequired(p.GetRequired(), p.GetDefault())
	return ParameterManifest{
		Name:         p.Name,
		Type:         p.Type,
		Required:     r,
		Description:  p.Desc,
		AuthServices: authServiceNames,
	}
}

// McpManifest returns the MCP manifest for the FloatParameter.
// json schema only allow numeric types of 'integer' and 'number'.
func (p *FloatParameter) McpManifest() (ParameterMcpManifest, []string) {
	authServiceNames := getAuthServiceNames(p.AuthServices)
	return ParameterMcpManifest{
		Type:        "number",
		Description: p.Desc,
	}, authServiceNames
}

// NewBooleanParameter is a convenience function for initializing a BooleanParameter.
func NewBooleanParameter(name string, desc string) *BooleanParameter {
	return &BooleanParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         TypeBool,
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
			Type:         TypeBool,
			Desc:         desc,
			AuthServices: nil,
		},
		Default: &defaultV,
	}
}

// NewBooleanParameterWithRequired is a convenience function for initializing a BooleanParameter.
func NewBooleanParameterWithRequired(name string, desc string, required bool) *BooleanParameter {
	return &BooleanParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         TypeBool,
			Desc:         desc,
			Required:     &required,
			AuthServices: nil,
		},
	}
}

// NewBooleanParameterWithAuth is a convenience function for initializing a BooleanParameter with a list of ParamAuthService.
func NewBooleanParameterWithAuth(name string, desc string, authServices []ParamAuthService) *BooleanParameter {
	return &BooleanParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         TypeBool,
			Desc:         desc,
			AuthServices: authServices,
		},
	}
}

// NewBooleanParameterWithAllowedValues is a convenience function for initializing a BooleanParameter with a list of allowed values.
func NewBooleanParameterWithAllowedValues(name string, desc string, allowedValues []any) *BooleanParameter {
	return &BooleanParameter{
		CommonParameter: CommonParameter{
			Name:          name,
			Type:          typeBool,
			Desc:          desc,
			AllowedValues: allowedValues,
			AuthServices:  nil,
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
	if !p.IsAllowedValues(newV) {
		return nil, fmt.Errorf("%t is not an allowed value", newV)
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

// Manifest returns the manifest for the BooleanParameter.
func (p *BooleanParameter) Manifest() ParameterManifest {
	// only list ParamAuthService names (without fields) in manifest
	authServiceNames := getAuthServiceNames(p.AuthServices)
	r := CheckParamRequired(p.GetRequired(), p.GetDefault())
	return ParameterManifest{
		Name:         p.Name,
		Type:         p.Type,
		Required:     r,
		Description:  p.Desc,
		AuthServices: authServiceNames,
	}
}

// NewArrayParameter is a convenience function for initializing a ArrayParameter.
func NewArrayParameter(name string, desc string, items Parameter) *ArrayParameter {
	return &ArrayParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         TypeArray,
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
			Type:         TypeArray,
			Desc:         desc,
			AuthServices: nil,
		},
		Items:   items,
		Default: &defaultV,
	}
}

// NewArrayParameterWithRequired is a convenience function for initializing a ArrayParameter with default value.
func NewArrayParameterWithRequired(name string, desc string, required bool, items Parameter) *ArrayParameter {
	return &ArrayParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         TypeArray,
			Desc:         desc,
			Required:     &required,
			AuthServices: nil,
		},
		Items: items,
	}
}

// NewArrayParameterWithAuth is a convenience function for initializing a ArrayParameter with a list of ParamAuthService.
func NewArrayParameterWithAuth(name string, desc string, items Parameter, authServices []ParamAuthService) *ArrayParameter {
	return &ArrayParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         TypeArray,
			Desc:         desc,
			AuthServices: authServices,
		},
		Items: items,
	}
}

// NewArrayParameterWithAllowedValues is a convenience function for initializing a ArrayParameter with a list of allowed values.
func NewArrayParameterWithAllowedValues(name string, desc string, allowedValues []any, items Parameter) *ArrayParameter {
	return &ArrayParameter{
		CommonParameter: CommonParameter{
			Name:          name,
			Type:          typeArray,
			Desc:          desc,
			AllowedValues: allowedValues,
			AuthServices:  nil,
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

func (p *ArrayParameter) IsAllowedValues(v []any) bool {
	a := p.GetAllowedValues()
	if len(a) == 0 {
		return true
	}
	for _, av := range a {
		if reflect.DeepEqual(v, av) {
			return true
		}
	}
	return false
}

func (p *ArrayParameter) Parse(v any) (any, error) {
	arrVal, ok := v.([]any)
	if !ok {
		return nil, &ParseTypeError{p.Name, p.Type, arrVal}
	}
	if !p.IsAllowedValues(arrVal) {
		return nil, fmt.Errorf("%s is not an allowed value", arrVal)
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
	authServiceNames := getAuthServiceNames(p.AuthServices)
	items := p.Items.Manifest()
	// if required value is true, or there's no default value
	r := CheckParamRequired(p.GetRequired(), p.GetDefault())
	items.Required = r
	return ParameterManifest{
		Name:         p.Name,
		Type:         p.Type,
		Required:     r,
		Description:  p.Desc,
		AuthServices: authServiceNames,
		Items:        &items,
	}
}

// McpManifest returns the MCP manifest for the ArrayParameter.
func (p *ArrayParameter) McpManifest() (ParameterMcpManifest, []string) {
	// only list ParamAuthService names (without fields) in manifest
	authServiceNames := getAuthServiceNames(p.AuthServices)
	items, _ := p.Items.McpManifest()
	return ParameterMcpManifest{
		Type:        p.Type,
		Description: p.Desc,
		Items:       &items,
	}, authServiceNames
}

// MapParameter is a parameter representing a map with string keys. If ValueType is
// specified (e.g., "string"), values are validated against that type. If ValueType
// is empty, it is treated as a generic map[string]any.
type MapParameter struct {
	CommonParameter `yaml:",inline"`
	Default         *map[string]any `yaml:"default,omitempty"`
	ValueType       string          `yaml:"valueType,omitempty"`
}

// Ensure MapParameter implements the Parameter interface.
var _ Parameter = &MapParameter{}

// NewMapParameter is a convenience function for initializing a MapParameter.
func NewMapParameter(name string, desc string, valueType string) *MapParameter {
	return &MapParameter{
		CommonParameter: CommonParameter{
			Name: name,
			Type: "map",
			Desc: desc,
		},
		ValueType: valueType,
	}
}

// NewMapParameterWithDefault is a convenience function for initializing a MapParameter with a default value.
func NewMapParameterWithDefault(name string, defaultV map[string]any, desc string, valueType string) *MapParameter {
	return &MapParameter{
		CommonParameter: CommonParameter{
			Name: name,
			Type: "map",
			Desc: desc,
		},
		ValueType: valueType,
		Default:   &defaultV,
	}
}

// NewMapParameterWithRequired is a convenience function for initializing a MapParameter as required.
func NewMapParameterWithRequired(name string, desc string, required bool, valueType string) *MapParameter {
	return &MapParameter{
		CommonParameter: CommonParameter{
			Name:     name,
			Type:     "map",
			Desc:     desc,
			Required: &required,
		},
		ValueType: valueType,
	}
}

// NewMapParameterWithAuth is a convenience function for initializing a MapParameter with auth services.
func NewMapParameterWithAuth(name string, desc string, valueType string, authServices []ParamAuthService) *MapParameter {
	return &MapParameter{
		CommonParameter: CommonParameter{
			Name:         name,
			Type:         "map",
			Desc:         desc,
			AuthServices: authServices,
		},
		ValueType: valueType,
	}
}

// NewMapParameterWithAllowedValues is a convenience function for initializing a MapParameter with a list of allowed values.
func NewMapParameterWithAllowedValues(name string, desc string, allowedValues []any, valueType string) *MapParameter {
	return &MapParameter{
		CommonParameter: CommonParameter{
			Name:          name,
			Type:          "map",
			Desc:          desc,
			AllowedValues: allowedValues,
		},
		ValueType: valueType,
	}
}

// UnmarshalYAML handles parsing the MapParameter from YAML input.
func (p *MapParameter) UnmarshalYAML(ctx context.Context, unmarshal func(interface{}) error) error {
	var rawItem struct {
		CommonParameter `yaml:",inline"`
		Default         *map[string]any `yaml:"default"`
		ValueType       string          `yaml:"valueType"`
	}
	if err := unmarshal(&rawItem); err != nil {
		return err
	}

	// Validate `ValueType` to be one of the supported basic types
	if rawItem.ValueType != "" {
		if _, err := getPrototypeParameter(rawItem.ValueType); err != nil {
			return err
		}
	}

	p.CommonParameter = rawItem.CommonParameter
	p.Default = rawItem.Default
	p.ValueType = rawItem.ValueType
	return nil
}

// getPrototypeParameter is a helper factory to create a temporary parameter
// based on a type string for parsing and manifest generation.
func getPrototypeParameter(typeName string) (Parameter, error) {
	switch typeName {
	case "string":
		return NewStringParameter("", ""), nil
	case "integer":
		return NewIntParameter("", ""), nil
	case "boolean":
		return NewBooleanParameter("", ""), nil
	case "float":
		return NewFloatParameter("", ""), nil
	default:
		return nil, fmt.Errorf("unsupported valueType %q for map parameter", typeName)
	}
}

func (p *MapParameter) IsAllowedValues(v map[string]any) bool {
	a := p.GetAllowedValues()
	if len(a) == 0 {
		return true
	}
	for _, av := range a {
		if reflect.DeepEqual(v, av) {
			return true
		}
	}
	return false
}

// Parse validates and parses an incoming value for the map parameter.
func (p *MapParameter) Parse(v any) (any, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, &ParseTypeError{p.Name, p.Type, m}
	}
	if !p.IsAllowedValues(m) {
		return nil, fmt.Errorf("%s is not an allowed value", m)
	}
	// for generic maps, convert json.Numbers to their corresponding types
	if p.ValueType == "" {
		convertedData, err := util.ConvertNumbers(m)
		if err != nil {
			return nil, fmt.Errorf("failed to parse integer or float values in map: %s", err)
		}
		convertedMap, ok := convertedData.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("internal error: ConvertNumbers should return a map, but got type %T", convertedData)
		}
		return convertedMap, nil
	}

	// Otherwise, get a prototype and parse each value in the map.
	prototype, err := getPrototypeParameter(p.ValueType)
	if err != nil {
		return nil, err
	}

	rtn := make(map[string]any, len(m))
	for key, val := range m {
		parsedVal, err := prototype.Parse(val)
		if err != nil {
			return nil, fmt.Errorf("unable to parse value for key %q: %w", key, err)
		}
		rtn[key] = parsedVal
	}
	return rtn, nil
}

func (p *MapParameter) GetAuthServices() []ParamAuthService {
	return p.AuthServices
}

func (p *MapParameter) GetDefault() any {
	if p.Default == nil {
		return nil
	}
	return *p.Default
}

func (p *MapParameter) GetValueType() string {
	return p.ValueType
}

// Manifest returns the manifest for the MapParameter.
func (p *MapParameter) Manifest() ParameterManifest {
	authServiceNames := getAuthServiceNames(p.AuthServices)
	r := CheckParamRequired(p.GetRequired(), p.GetDefault())

	var additionalProperties any
	if p.ValueType != "" {
		_, err := getPrototypeParameter(p.ValueType)
		if err != nil {
			panic(err)
		}
		valueSchema := map[string]any{"type": p.ValueType}
		additionalProperties = valueSchema
	} else {
		// If no valueType is given, allow any properties.
		additionalProperties = true
	}

	return ParameterManifest{
		Name:                 p.Name,
		Type:                 "object",
		Required:             r,
		Description:          p.Desc,
		AuthServices:         authServiceNames,
		AdditionalProperties: additionalProperties,
	}
}

// McpManifest returns the MCP manifest for the MapParameter.
func (p *MapParameter) McpManifest() (ParameterMcpManifest, []string) {
	authServiceNames := getAuthServiceNames(p.AuthServices)
	var additionalProperties any
	if p.ValueType != "" {
		_, err := getPrototypeParameter(p.ValueType)
		if err != nil {
			panic(err)
		}
		valueSchema := map[string]any{"type": p.ValueType}
		additionalProperties = valueSchema
	} else {
		// If no valueType is given, allow any properties.
		additionalProperties = true
	}

	return ParameterMcpManifest{
		Type:                 "object",
		Description:          p.Desc,
		AdditionalProperties: additionalProperties,
	}, authServiceNames
}
