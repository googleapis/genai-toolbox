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

package mysqlgetqueryplan

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/orderedmap"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const resourceType string = "mysql-get-query-plan"

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
	MySQLPool() *sql.DB
	RunSQL(context.Context, string, []any) (any, error)
}

type Config struct {
	Name         string                 `yaml:"name" validate:"required"`
	Type         string                 `yaml:"type" validate:"required"`
	Source       string                 `yaml:"source" validate:"required"`
	Description  string                 `yaml:"description" validate:"required"`
	AuthRequired []string               `yaml:"authRequired"`
	Annotations  *tools.ToolAnnotations `yaml:"annotations,omitempty"`

	ScopesRequired []string `yaml:"scopesRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	sqlParameter := parameters.NewStringParameter("sql_statement", "The sql statement to explain.")
	params := parameters.Parameters{sqlParameter}

	return Tool{
		BaseTool: tools.BaseTool{
			Name:             cfg.Name,
			Description:      cfg.Description,
			Metadata:         tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
			StaticParameters: params,
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

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.cfg.Source, t.cfg.Name, t.cfg.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	paramsMap := params.AsMap()
	sqlStr, ok := paramsMap["sql_statement"].(string)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("unable to get cast %s", paramsMap["sql_statement"]), nil)
	}

	// Log the query executed for debugging.
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, util.NewClientServerError("error getting logger", http.StatusInternalServerError, err)
	}
	logger.DebugContext(ctx, fmt.Sprintf("executing `%s` tool query: %s", resourceType, sqlStr))

	query := fmt.Sprintf("EXPLAIN FORMAT=JSON %s", sqlStr)
	result, err := source.RunSQL(ctx, query, nil)
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}
	// extract and return only the query plan object
	resSlice, ok := result.([]any)
	if !ok || len(resSlice) == 0 {
		return nil, util.NewClientServerError("no query plan returned", http.StatusInternalServerError, nil)
	}
	row, ok := resSlice[0].(orderedmap.Row)
	if !ok || len(row.Columns) == 0 {
		return nil, util.NewClientServerError("no query plan returned in row", http.StatusInternalServerError, nil)
	}
	plan, ok := row.Columns[0].Value.(string)
	if !ok {
		return nil, util.NewClientServerError("unable to convert plan object to string", http.StatusInternalServerError, nil)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(plan), &out); err != nil {
		return nil, util.NewClientServerError("failed to unmarshal query plan json", http.StatusInternalServerError, err)
	}
	return out, nil
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.cfg
}
