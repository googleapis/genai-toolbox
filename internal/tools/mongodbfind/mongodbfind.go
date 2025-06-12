// Copyright 2025 Google LLC
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
package mongodbfind

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/goccy/go-yaml"
	mongosrc "github.com/googleapis/genai-toolbox/internal/sources/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"slices"
	"text/template"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "mongodb-find"

func init() {
	if !tools.Register(kind, newConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", kind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name           string           `yaml:"name" validate:"required"`
	Kind           string           `yaml:"kind" validate:"required"`
	Source         string           `yaml:"source" validate:"required"`
	AuthRequired   []string         `yaml:"authRequired"`
	Description    string           `yaml:"description" validate:"required"`
	Collection     string           `yaml:"collection" validate:"required"`
	FilterPayload  string           `yaml:"filterPayload" validate:"required"`
	FilterParams   tools.Parameters `yaml:"filterParams" validate:"required"`
	ProjectPayload string           `yaml:"projectPayload"`
	ProjectParams  tools.Parameters `yaml:"projectParams"`
	SortPayload    string           `yaml:"sortPayload"`
	SortParams     tools.Parameters `yaml:"sortParams"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(*mongosrc.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `mongo-query`", kind)
	}

	// Create a slice for all parameters
	allParameters := slices.Concat(cfg.FilterParams, cfg.ProjectParams, cfg.SortParams)

	// Create parameter MCP manifest
	paramManifest := slices.Concat(
		cfg.FilterParams.Manifest(),
		cfg.ProjectParams.Manifest(),
		cfg.SortParams.Manifest(),
	)
	if paramManifest == nil {
		paramManifest = make([]tools.ParameterManifest, 0)
	}

	queryMcpManifest := cfg.FilterParams.McpManifest()
	bodyMcpManifest := cfg.ProjectParams.McpManifest()
	headerMcpManifest := cfg.SortParams.McpManifest()

	// Concatenate parameters for MCP `required` field
	concatRequiredManifest := slices.Concat(
		queryMcpManifest.Required,
		bodyMcpManifest.Required,
		headerMcpManifest.Required,
	)
	if concatRequiredManifest == nil {
		concatRequiredManifest = []string{}
	}

	// Concatenate parameters for MCP `properties` field
	concatPropertiesManifest := make(map[string]tools.ParameterMcpManifest)
	for name, p := range queryMcpManifest.Properties {
		concatPropertiesManifest[name] = p
	}
	for name, p := range bodyMcpManifest.Properties {
		concatPropertiesManifest[name] = p
	}
	for name, p := range headerMcpManifest.Properties {
		concatPropertiesManifest[name] = p
	}

	// Create a new McpToolsSchema with all parameters
	paramMcpManifest := tools.McpToolsSchema{
		Type:       "object",
		Properties: concatPropertiesManifest,
		Required:   concatRequiredManifest,
	}

	// Verify there are no duplicate parameter names
	seenNames := make(map[string]bool)
	for _, param := range paramManifest {
		if _, exists := seenNames[param.Name]; exists {
			return nil, fmt.Errorf("parameter name must be unique across filterParams, projectParams, and sortParams. Duplicate parameter: %s", param.Name)
		}
		seenNames[param.Name] = true
	}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: paramMcpManifest,
	}

	// finish tool setup
	return Tool{
		Name:           cfg.Name,
		Kind:           kind,
		Collection:     cfg.Collection,
		FilterPayload:  cfg.FilterPayload,
		FilterParams:   cfg.FilterParams,
		ProjectPayload: cfg.ProjectPayload,
		ProjectParams:  cfg.ProjectParams,
		SortPayload:    cfg.SortPayload,
		SortParams:     cfg.SortParams,
		AllParams:      allParameters,
		database:       s.Database,
		manifest:       tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: nil},
		mcpManifest:    mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name           string           `yaml:"name"`
	Kind           string           `yaml:"kind"`
	Description    string           `yaml:"description"`
	Collection     string           `yaml:"collection"`
	FilterPayload  string           `yaml:"filterPayload"`
	FilterParams   tools.Parameters `yaml:"filterParams"`
	ProjectPayload string           `yaml:"projectPayload"`
	ProjectParams  tools.Parameters `yaml:"projectParams"`
	SortPayload    string           `yaml:"sortPayload"`
	SortParams     tools.Parameters `yaml:"sortParams"`
	AllParams      tools.Parameters `yaml:"allParams"`

	database    *mongo.Database
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// helper function to convert a parameter to JSON formatted string.
func convertParamToJSON(param any) (string, error) {
	jsonData, err := json.Marshal(param)
	if err != nil {
		return "", fmt.Errorf("failed to marshal param to JSON: %w", err)
	}
	return string(jsonData), nil
}

func getFilter(filterParams tools.Parameters, filterPayload string, paramsMap map[string]any) (string, error) {
	// Create a map for request body parameters
	filterParamsMap := make(map[string]any)
	for _, p := range filterParams {
		k := p.GetName()
		v, ok := paramsMap[k]
		if !ok {
			return "", fmt.Errorf("missing filter parameter %s", k)
		}
		filterParamsMap[k] = v
	}

	// Create a FuncMap to format array parameters
	funcMap := template.FuncMap{
		"json": convertParamToJSON,
	}
	templ, err := template.New("filter").Funcs(funcMap).Parse(filterPayload)
	if err != nil {
		return "", fmt.Errorf("error parsing filter: %s", err)
	}
	var result bytes.Buffer
	err = templ.Execute(&result, filterParamsMap)
	if err != nil {
		return "", fmt.Errorf("error replacing filter payload: %s", err)
	}
	return result.String(), nil
}

func getOptions(sortParameters tools.Parameters, projectParams tools.Parameters, paramsMap map[string]any) (*options.FindOptions, error) {
	opts := options.Find()

	sort := bson.M{}
	for _, p := range sortParameters {
		sort[p.GetName()] = paramsMap[p.GetName()]
	}
	opts = opts.SetSort(sort)

	project := bson.M{}
	for _, p := range projectParams {
		project[p.GetName()] = paramsMap[p.GetName()]
	}
	opts = opts.SetProjection(project)

	return opts, nil
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	paramsMap := params.AsMap()

	filterString, err := getFilter(t.FilterParams, t.FilterPayload, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("error populating filter: %s", err)
	}

	opts, err := getOptions(t.SortParams, t.ProjectParams, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("error populating options: %s", err)
	}

	var filter = bson.D{}
	err = bson.UnmarshalExtJSON([]byte(filterString), true, &filter)
	if err != nil {
		return nil, err
	}
	fmt.Println(filter)

	cur, err := t.database.Collection(t.Collection).Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var data = []interface{}{}
	err = cur.All(context.TODO(), &data)
	if err != nil {
		return nil, err
	}
	fmt.Println(data)

	var final []any
	for _, item := range data {
		tmp, _ := bson.MarshalExtJSON(item, false, false)
		var tmp2 any
		_ = json.Unmarshal(tmp, &tmp2)
		final = append(final, tmp2)
	}

	return final, err
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.AllParams, data, claims)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized([]string{}, verifiedAuthServices)
}
