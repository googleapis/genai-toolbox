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
package mongodbaggregate

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/goccy/go-yaml"
	mongosrc "github.com/googleapis/genai-toolbox/internal/sources/mongodb"
	"github.com/googleapis/genai-toolbox/internal/tools/mongodb/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "mongodb-aggregate"

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
	Name            string           `yaml:"name" validate:"required"`
	Kind            string           `yaml:"kind" validate:"required"`
	Source          string           `yaml:"source" validate:"required"`
	AuthRequired    []string         `yaml:"authRequired" validate:"required"`
	Description     string           `yaml:"description" validate:"required"`
	Database        string           `yaml:"database" validate:"required"`
	Collection      string           `yaml:"collection" validate:"required"`
	PipelinePayload string           `yaml:"pipelinePayload" validate:"required"`
	PipelineParams  tools.Parameters `yaml:"pipelineParams" validate:"required"`
	Canonical       bool             `yaml:"canonical"`
	ReadOnly        bool             `yaml:"readOnly"`
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
	allParameters := slices.Concat(cfg.PipelineParams)

	// Create parameter MCP manifest
	paramManifest := slices.Concat(
		cfg.PipelineParams.Manifest(),
	)
	if paramManifest == nil {
		paramManifest = make([]tools.ParameterManifest, 0)
	}

	pipelineMcpManifest := cfg.PipelineParams.McpManifest()

	// Concatenate parameters for MCP `required` field
	concatRequiredManifest := slices.Concat(
		pipelineMcpManifest.Required,
	)
	if concatRequiredManifest == nil {
		concatRequiredManifest = []string{}
	}

	// Concatenate parameters for MCP `properties` field
	concatPropertiesManifest := make(map[string]tools.ParameterMcpManifest)
	for name, p := range pipelineMcpManifest.Properties {
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
			return nil, fmt.Errorf("parameter name must be unique across pipelineParams, and sortParams. Duplicate parameter: %s", param.Name)
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
		Name:            cfg.Name,
		Kind:            kind,
		AuthRequired:    cfg.AuthRequired,
		Collection:      cfg.Collection,
		PipelinePayload: cfg.PipelinePayload,
		PipelineParams:  cfg.PipelineParams,
		Canonical:       cfg.Canonical,
		ReadOnly:        cfg.ReadOnly,
		AllParams:       allParameters,
		database:        s.Client.Database(cfg.Database),
		manifest:        tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:     mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name            string           `yaml:"name"`
	Kind            string           `yaml:"kind"`
	Description     string           `yaml:"description"`
	AuthRequired    []string         `yaml:"authRequired"`
	Collection      string           `yaml:"collection"`
	PipelinePayload string           `yaml:"pipelinePayload"`
	PipelineParams  tools.Parameters `yaml:"pipelineParams"`
	Canonical       bool             `yaml:"canonical"`
	ReadOnly        bool             `yaml:"readOnly"`
	AllParams       tools.Parameters `yaml:"allParams"`

	database    *mongo.Database
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	paramsMap := params.AsMap()

	pipelineString, err := common.ParsePayloadTemplate(t.PipelineParams, t.PipelinePayload, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("error populating pipeline: %s", err)
	}

	var pipeline = []bson.M{}
	err = bson.UnmarshalExtJSON([]byte(pipelineString), t.Canonical, &pipeline)
	if err != nil {
		return nil, err
	}

	if t.ReadOnly {
		//fail if we do a merge or an out
		for _, stage := range pipeline {
			for key, _ := range stage {
				if key == "$merge" || key == "$out" {
					return nil, fmt.Errorf("this is not a read-only pipeline: %+v", stage)
				}
			}
		}
	}

	cur, err := t.database.Collection(t.Collection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var data = []any{}
	err = cur.All(ctx, &data)
	if err != nil {
		return nil, err
	}

	var final []any
	for _, item := range data {
		tmp, _ := bson.MarshalExtJSON(item, false, false)
		var tmp2 any
		err = json.Unmarshal(tmp, &tmp2)
		if err != nil {
			return nil, err
		}
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
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}
