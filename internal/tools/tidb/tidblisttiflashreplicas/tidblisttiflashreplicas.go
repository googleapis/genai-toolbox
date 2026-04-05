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

// Package tidblisttiflashreplicas provides a tool to list TiFlash replica status.
// TiFlash is TiDB's columnar storage engine for real-time analytics.
// Note: TiFlash is available in TiDB 4.0+. This tool will return an empty list
// or an error on older versions.
package tidblisttiflashreplicas

import (
	"context"
	"errors"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-sql-driver/mysql"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const resourceType string = "tidb-list-tiflash-replicas"

// MySQL error code for "Unknown column"
const mysqlErrUnknownColumn = 1054

// listTiFlashReplicasStatement queries TiFlash replica status from TiDB.
// This is a TiDB-specific feature not available in MySQL.
// Uses IFNULL to handle potential NULL values gracefully.
const listTiFlashReplicasStatement = `
    SELECT
        JSON_OBJECT(
            'table_schema', T.TABLE_SCHEMA,
            'table_name', T.TABLE_NAME,
            'replica_count', IFNULL(T.TIFLASH_REPLICA_COUNT, 0),
            'available', IFNULL(TR.AVAILABLE, 0),
            'progress', IFNULL(TR.PROGRESS, 0)
        ) AS tiflash_info
    FROM
        INFORMATION_SCHEMA.TABLES T
    LEFT JOIN
        INFORMATION_SCHEMA.TIFLASH_REPLICA TR
        ON T.TABLE_SCHEMA = TR.TABLE_SCHEMA AND T.TABLE_NAME = TR.TABLE_NAME
    WHERE
        T.TABLE_SCHEMA NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys', 'METRICS_SCHEMA', 'INSPECTION_SCHEMA')
        AND T.TABLE_TYPE = 'BASE TABLE'
        AND IFNULL(T.TIFLASH_REPLICA_COUNT, 0) > 0
    ORDER BY
        T.TABLE_SCHEMA, T.TABLE_NAME;
`

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
	TiDBPool() *sql.DB
	RunSQL(context.Context, string, []any) (any, error)
}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Type         string   `yaml:"type" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// No parameters needed - this tool returns all TiFlash replicas
	allParameters := parameters.Parameters{}
	paramManifest := allParameters.Manifest()
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		AllParams:   allParameters,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	AllParams parameters.Parameters `yaml:"allParams"`

	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// isTiFlashUnsupportedError checks if the error indicates TiFlash is not available
// (older TiDB version or TiFlash not deployed). Uses MySQL error code 1054 (Unknown column)
// and checks if the missing column is TiFlash-related.
func isTiFlashUnsupportedError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if ok := errors.As(err, &mysqlErr); ok {
		if mysqlErr.Number == mysqlErrUnknownColumn {
			msg := strings.ToUpper(mysqlErr.Message)
			return strings.Contains(msg, "TIFLASH_REPLICA_COUNT") || strings.Contains(msg, "TIFLASH_REPLICA")
		}
	}
	return false
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	// Log the query for debugging
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, util.NewClientServerError("error getting logger", http.StatusInternalServerError, err)
	}
	logger.DebugContext(ctx, fmt.Sprintf("executing `%s` tool", resourceType))

	resp, err := source.RunSQL(ctx, listTiFlashReplicasStatement, nil)
	if err != nil {
		// Check for TiFlash-related "Unknown column" errors (MySQL error 1054)
		if isTiFlashUnsupportedError(err) {
			return nil, util.NewAgentError("TiFlash is not available on this TiDB version (requires TiDB 4.0+) or TiFlash is not deployed", err)
		}
		return nil, util.ProcessGeneralError(err)
	}

	// if there's no results, return empty list instead of null
	resSlice, ok := resp.([]any)
	if !ok || len(resSlice) == 0 {
		return []any{}, nil
	}
	return resp, nil
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.AllParams, paramValues, embeddingModelsMap, nil)
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

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	return false, nil
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}

func (t Tool) GetParameters() parameters.Parameters {
	return t.AllParams
}
