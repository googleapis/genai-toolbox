// Copyright 2025 Google LLC
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

package elasticsearchesql

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	es "github.com/googleapis/mcp-toolbox/internal/sources/elasticsearch"
	"github.com/googleapis/mcp-toolbox/internal/tools"
)

const resourceType string = "elasticsearch-esql"

func init() {
	if !tools.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", resourceType))
	}
}

type compatibleSource interface {
	ElasticsearchClient() es.EsClient
	RunSQL(ctx context.Context, format, query string, params []map[string]any) (any, error)
}

type Config struct {
	Name         string                 `yaml:"name" validate:"required"`
	Type         string                 `yaml:"type" validate:"required"`
	Source       string                 `yaml:"source" validate:"required"`
	Description  string                 `yaml:"description" validate:"required"`
	AuthRequired []string               `yaml:"authRequired" validate:"required"`
	Query        string                 `yaml:"query" validate:"required"`
	Format       string                 `yaml:"format"`
	Timeout      int                    `yaml:"timeout"`
	Parameters   parameters.Parameters  `yaml:"parameters"`
	Annotations  *tools.ToolAnnotations `yaml:"annotations,omitempty"`

	ScopesRequired []string `yaml:"scopesRequired"`
}

var _ tools.ToolConfig = Config{}

func (c Config) ToolConfigType() string {
	return resourceType
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Tool struct {
	tools.BaseTool
	cfg Config
}

var _ tools.Tool = Tool{}

func (c Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	return Tool{
		BaseTool: tools.BaseTool{
			Name:             c.Name,
			Description:      c.Description,
			Metadata:         tools.Manifest{Description: c.Description, Parameters: c.Parameters.Manifest(), AuthRequired: c.AuthRequired},
			StaticParameters: c.Parameters,
			ScopesRequired:   c.ScopesRequired,
			Annotations:      tools.GetAnnotationsOrDefault(c.Annotations, tools.NewReadOnlyAnnotations),
		},
		cfg: c,
	}, nil
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.cfg
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.cfg.Source, t.cfg.Name, t.cfg.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	var cancel context.CancelFunc
	if t.cfg.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(t.cfg.Timeout)*time.Second)
		defer cancel()
	} else {
		ctx, cancel = context.WithTimeout(ctx, time.Minute)
		defer cancel()
	}

	query := t.cfg.Query
	paramMap := params.AsMap()

	var paramsList []map[string]any
	for _, param := range t.cfg.Parameters {
		if param.GetType() == "array" {
			return nil, util.NewAgentError("array parameters are not supported yet", nil)
		}

		// ES|QL requires an array of single-key objects for named parameters
		if val, ok := paramMap[param.GetName()]; ok {
			paramsList = append(paramsList, map[string]any{param.GetName(): val})
		}
	}

	resp, err := source.RunSQL(ctx, t.cfg.Format, query, paramsList)
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}
	return resp, nil
}
