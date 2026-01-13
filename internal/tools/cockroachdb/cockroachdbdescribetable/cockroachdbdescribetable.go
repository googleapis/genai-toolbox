// Copyright 2026 Google LLC
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

package cockroachdbdescribetable

import (
	"context"
	"fmt"
	"time"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/cockroachdb"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/orderedmap"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"github.com/jackc/pgx/v5"
)

const kind string = "cockroachdb-describe-table"

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

type compatibleSource interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	IsReadOnlyMode() bool
	EmitTelemetry(ctx context.Context, event cockroachdb.TelemetryEvent)
}

var compatibleSources = [...]string{cockroachdb.SourceKind}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	_, ok = rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	// Define parameters for describe_table
	schemaParam := parameters.NewStringParameter("schema_name", "The schema name (e.g., 'public')")
	tableParam := parameters.NewStringParameter("table_name", "The table name to describe")
	params := parameters.Parameters{schemaParam, tableParam}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, nil)

	t := Tool{
		Config:      cfg,
		Parameters:  params,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

var _ tools.Tool = Tool{}

type Tool struct {
	Config
	Parameters parameters.Parameters `yaml:"parameters"`

	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	startTime := time.Now()

	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Kind)
	if err != nil {
		return nil, err
	}

	paramsMap := params.AsMap()
	schemaName, ok := paramsMap["schema_name"].(string)
	if !ok {
		return nil, fmt.Errorf("schema_name parameter is required and must be a string")
	}

	tableName, ok := paramsMap["table_name"].(string)
	if !ok {
		return nil, fmt.Errorf("table_name parameter is required and must be a string")
	}

	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting logger: %s", err)
	}

	// CockroachDB v25.4.2 - Query to describe table columns
	// Uses information_schema which is standard across PostgreSQL-compatible databases
	sql := `
SELECT 
    column_name,
    data_type,
    is_nullable,
    column_default,
    is_hidden,
    generation_expression as is_generated,
    ordinal_position
FROM information_schema.columns
WHERE table_schema = $1 AND table_name = $2
ORDER BY ordinal_position`

	logger.DebugContext(ctx, fmt.Sprintf("executing `%s` tool query for schema=%s, table=%s", kind, schemaName, tableName))

	results, err := source.Query(ctx, sql, schemaName, tableName)
	if err != nil {
		// Emit telemetry for failure
		source.EmitTelemetry(ctx, cockroachdb.TelemetryEvent{
			Timestamp:   time.Now(),
			ToolName:    kind,
			SQLRedacted: cockroachdb.RedactSQL(sql),
			Status:      "failure",
			ErrorCode:   cockroachdb.ErrCodeQueryExecutionFailed,
			ErrorMsg:    err.Error(),
			LatencyMs:   time.Since(startTime).Milliseconds(),
		})
		return nil, fmt.Errorf("unable to describe table: %w", err)
	}
	defer results.Close()

	fields := results.FieldDescriptions()

	var out []any
	rowCount := int64(0)
	for results.Next() {
		rowCount++
		v, err := results.Values()
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}
		row := orderedmap.Row{}
		for i, f := range fields {
			row.Add(f.Name, v[i])
		}
		out = append(out, row)
	}

	if err := results.Err(); err != nil {
		source.EmitTelemetry(ctx, cockroachdb.TelemetryEvent{
			Timestamp:   time.Now(),
			ToolName:    kind,
			SQLRedacted: cockroachdb.RedactSQL(sql),
			Status:      "failure",
			ErrorCode:   cockroachdb.ErrCodeQueryExecutionFailed,
			ErrorMsg:    err.Error(),
			LatencyMs:   time.Since(startTime).Milliseconds(),
		})
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	// Emit telemetry for success
	source.EmitTelemetry(ctx, cockroachdb.TelemetryEvent{
		Timestamp:    time.Now(),
		ToolName:     kind,
		SQLRedacted:  cockroachdb.RedactSQL(sql),
		Status:       "success",
		LatencyMs:    time.Since(startTime).Milliseconds(),
		RowsAffected: rowCount,
	})

	return out, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Parameters, data, claims)
}

func (t Tool) EmbedParams(ctx context.Context, params parameters.ParamValues, models map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return params, nil
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

func (t Tool) RequiresClientAuthorization(_ tools.SourceProvider) (bool, error) {
	return false, nil
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}
