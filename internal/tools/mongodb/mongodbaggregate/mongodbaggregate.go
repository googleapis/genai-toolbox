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
	"fmt"
	"net/http"
	"slices"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
)

const resourceType string = "mongodb-aggregate"

func init() {
	if !tools.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", resourceType))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type compatibleSource interface {
	MongoClient() *mongo.Client
	Aggregate(context.Context, string, bool, bool, string, string) ([]any, error)
}

type Config struct {
	Name            string                 `yaml:"name" validate:"required"`
	Type            string                 `yaml:"type" validate:"required"`
	Source          string                 `yaml:"source" validate:"required"`
	AuthRequired    []string               `yaml:"authRequired" validate:"required"`
	Description     string                 `yaml:"description" validate:"required"`
	Database        string                 `yaml:"database" validate:"required"`
	Collection      string                 `yaml:"collection" validate:"required"`
	PipelinePayload string                 `yaml:"pipelinePayload" validate:"required"`
	PipelineParams  parameters.Parameters  `yaml:"pipelineParams" validate:"required"`
	Canonical       bool                   `yaml:"canonical"`
	ReadOnly        bool                   `yaml:"readOnly"`
	Annotations     *tools.ToolAnnotations `yaml:"annotations,omitempty"`

	ScopesRequired []string `yaml:"scopesRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	allParameters := slices.Concat(cfg.PipelineParams)

	paramManifest := allParameters.Manifest()
	if paramManifest == nil {
		paramManifest = make([]parameters.ParameterManifest, 0)
	}

	return Tool{
		BaseTool: tools.BaseTool{
			Name:             cfg.Name,
			Description:      cfg.Description,
			Metadata:         tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
			StaticParameters: allParameters,
			ScopesRequired:   cfg.ScopesRequired,
			Annotations:      tools.GetAnnotationsOrDefault(cfg.Annotations, tools.NewReadOnlyAnnotations),
		},
		cfg: cfg,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	tools.BaseTool
	cfg Config
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.cfg
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.cfg.Source, t.cfg.Name, t.cfg.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	paramsMap := params.AsMap()
	pipelineString, err := parameters.PopulateTemplateWithJSON("MongoDBAggregatePipeline", t.cfg.PipelinePayload, paramsMap)
	if err != nil {
		return nil, util.NewAgentError("error populating pipeline", err)
	}
	resp, err := source.Aggregate(ctx, pipelineString, t.cfg.Canonical, t.cfg.ReadOnly, t.cfg.Database, t.cfg.Collection)
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}
	return resp, nil
}
