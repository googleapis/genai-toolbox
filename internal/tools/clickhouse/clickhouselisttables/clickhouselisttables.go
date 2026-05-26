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

package clickhouse

import (
	"context"
	"fmt"
	"net/http"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const listTablesType string = "clickhouse-list-tables"
const databaseKey string = "database"

func init() {
	if !tools.Register(listTablesType, newListTablesConfig) {
		panic(fmt.Sprintf("tool type %q already registered", listTablesType))
	}
}

func newListTablesConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type compatibleSource interface {
	RunSQL(context.Context, string, parameters.ParamValues) (any, error)
}

type Config struct {
	Name         string                 `yaml:"name" validate:"required"`
	Type         string                 `yaml:"type" validate:"required"`
	Source       string                 `yaml:"source" validate:"required"`
	Description  string                 `yaml:"description" validate:"required"`
	AuthRequired []string               `yaml:"authRequired"`
	Parameters   parameters.Parameters  `yaml:"parameters"`
	Annotations  *tools.ToolAnnotations `yaml:"annotations,omitempty"`

	ScopesRequired []string `yaml:"scopesRequired"`
}

var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return listTablesType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	databaseParameter := parameters.NewStringParameter(databaseKey, "The database to list tables from.")
	params := parameters.Parameters{databaseParameter}

	allParameters, paramManifest, _ := parameters.ProcessParameters(nil, params)

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

var _ tools.Tool = Tool{}

type Tool struct {
	tools.BaseTool
	cfg Config
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.cfg
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, token tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.cfg.Source, t.cfg.Name, t.cfg.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	mapParams := params.AsMap()
	database, ok := mapParams[databaseKey].(string)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("invalid or missing '%s' parameter; expected a string", databaseKey), nil)
	}

	// Query to list all tables in the specified database
	// Note: formatting identifier directly is risky if input is untrusted, but standard for this tool structure.
	query := fmt.Sprintf("SHOW TABLES FROM %s", database)

	out, err := source.RunSQL(ctx, query, nil)
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}

	res, ok := out.([]any)
	if !ok {
		return nil, util.NewClientServerError("unable to convert result to list", http.StatusInternalServerError, nil)
	}

	var tables []map[string]any
	for _, item := range res {
		tableMap, ok := item.(map[string]any)
		if !ok {
			return nil, util.NewClientServerError(fmt.Sprintf("unexpected type in result: got %T, want map[string]any", item), http.StatusInternalServerError, nil)
		}
		tableMap["database"] = database
		tables = append(tables, tableMap)
	}
	return tables, nil
}
