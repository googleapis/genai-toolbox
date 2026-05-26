// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cassandracql

import (
	"context"
	"fmt"
	"net/http"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const resourceType string = "cassandra-cql"

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
	CassandraSession() *gocql.Session
	RunSQL(context.Context, string, parameters.ParamValues) (any, error)
}

type Config struct {
	Name               string                 `yaml:"name" validate:"required"`
	Type               string                 `yaml:"type" validate:"required"`
	Source             string                 `yaml:"source" validate:"required"`
	Description        string                 `yaml:"description" validate:"required"`
	Statement          string                 `yaml:"statement" validate:"required"`
	AuthRequired       []string               `yaml:"authRequired"`
	Parameters         parameters.Parameters  `yaml:"parameters"`
	TemplateParameters parameters.Parameters  `yaml:"templateParameters"`
	Annotations        *tools.ToolAnnotations `yaml:"annotations,omitempty"`

	ScopesRequired []string `yaml:"scopesRequired"`
}

var _ tools.ToolConfig = Config{}

// ToolConfigType implements tools.ToolConfig.
func (c Config) ToolConfigType() string {
	return resourceType
}

// Initialize implements tools.ToolConfig.
func (c Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	allParameters, paramManifest, err := parameters.ProcessParameters(c.TemplateParameters, c.Parameters)
	if err != nil {
		return nil, err
	}

	return Tool{
		BaseTool: tools.BaseTool{
			Name:             c.Name,
			Description:      c.Description,
			Metadata:         tools.Manifest{Description: c.Description, Parameters: paramManifest, AuthRequired: c.AuthRequired},
			StaticParameters: allParameters,
			ScopesRequired:   c.ScopesRequired,
			Annotations:      tools.GetAnnotationsOrDefault(c.Annotations, tools.NewDestructiveAnnotations),
		},
		cfg: c,
	}, nil
}

var _ tools.Tool = Tool{}

type Tool struct {
	tools.BaseTool
	cfg Config
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.cfg
}

// Invoke implements tools.Tool.
func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.cfg.Source, t.cfg.Name, t.cfg.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	paramsMap := params.AsMap()
	newStatement, err := parameters.ResolveTemplateParams(t.cfg.TemplateParameters, t.cfg.Statement, paramsMap)
	if err != nil {
		return nil, util.NewAgentError("unable to extract template params", err)
	}

	newParams, err := parameters.GetParams(t.cfg.Parameters, paramsMap)
	if err != nil {
		return nil, util.NewAgentError("unable to extract standard params", err)
	}
	resp, err := source.RunSQL(ctx, newStatement, newParams)
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}
	return resp, nil
}
