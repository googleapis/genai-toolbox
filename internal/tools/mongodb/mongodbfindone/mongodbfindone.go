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
package mongodbfindone

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/goccy/go-yaml"
	mongosrc "github.com/googleapis/genai-toolbox/internal/sources/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "mongodb-find-one"

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
	AuthRequired   []string         `yaml:"authRequired" validate:"required"`
	Description    string           `yaml:"description" validate:"required"`
	Database       string           `yaml:"database" validate:"required"`
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
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `mongodb`", kind)
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

	filterMcpManifest := cfg.FilterParams.McpManifest()
	projectMcpManifest := cfg.ProjectParams.McpManifest()
	sortMcpManifest := cfg.SortParams.McpManifest()

	// Concatenate parameters for MCP `required` field
	concatRequiredManifest := slices.Concat(
		filterMcpManifest.Required,
		projectMcpManifest.Required,
		sortMcpManifest.Required,
	)
	if concatRequiredManifest == nil {
		concatRequiredManifest = []string{}
	}

	// Concatenate parameters for MCP `properties` field
	concatPropertiesManifest := make(map[string]tools.ParameterMcpManifest)
	for name, p := range filterMcpManifest.Properties {
		concatPropertiesManifest[name] = p
	}
	for name, p := range projectMcpManifest.Properties {
		concatPropertiesManifest[name] = p
	}
	for name, p := range sortMcpManifest.Properties {
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
		AuthRequired:   cfg.AuthRequired,
		Collection:     cfg.Collection,
		FilterPayload:  cfg.FilterPayload,
		FilterParams:   cfg.FilterParams,
		ProjectPayload: cfg.ProjectPayload,
		ProjectParams:  cfg.ProjectParams,
		SortPayload:    cfg.SortPayload,
		SortParams:     cfg.SortParams,
		AllParams:      allParameters,
		database:       s.Client.Database(cfg.Database),
		manifest:       tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:    mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name           string           `yaml:"name"`
	Kind           string           `yaml:"kind"`
	AuthRequired   []string         `yaml:"authRequired"`
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

func getOptions(sortParameters tools.Parameters, projectPayload string, paramsMap map[string]any) (*options.FindOneOptions, error) {
	opts := options.FindOne()

	sort := bson.M{}
	for _, p := range sortParameters {
		sort[p.GetName()] = paramsMap[p.GetName()]
	}
	opts = opts.SetSort(sort)

	if len(projectPayload) == 0 {
		return opts, nil
	}

	result, err := tools.PopulateTemplateWithJSON("MongoDBFindOneProjectString", projectPayload, paramsMap)

	if err != nil {
		return nil, fmt.Errorf("error populating project payload: %s", err)
	}

	var projection any
	err = bson.Unmarshal([]byte(result), &projection)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling projection: %s", err)
	}
	opts = opts.SetProjection(projection)

	return opts, nil
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) (any, error) {
	paramsMap := params.AsMap()

	filterString, err := tools.PopulateTemplateWithJSON("MongoDBFindOneFilterString", t.FilterPayload, paramsMap)

	if err != nil {
		return nil, fmt.Errorf("error populating filter: %s", err)
	}

	opts, err := getOptions(t.SortParams, t.ProjectPayload, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("error populating options: %s", err)
	}

	var filter = bson.D{}
	err = bson.UnmarshalExtJSON([]byte(filterString), false, &filter)
	if err != nil {
		return nil, err
	}

	res := t.database.Collection(t.Collection).FindOne(ctx, filter, opts)
	if res.Err() != nil {
		return nil, res.Err()
	}

	var data any
	err = res.Decode(&data)
	if err != nil {
		return nil, err
	}

	var final []any
	tmp, _ := bson.MarshalExtJSON(data, false, false)
	var tmp2 any
	err = json.Unmarshal(tmp, &tmp2)
	if err != nil {
		return nil, err
	}
	final = append(final, tmp2)

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
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}
