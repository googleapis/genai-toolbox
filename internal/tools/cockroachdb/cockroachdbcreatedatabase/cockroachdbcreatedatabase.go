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

package cockroachdbcreatedatabase

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

const kind string = "cockroachdb-create-database"

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

	databaseParam := parameters.NewStringParameter("database_name", "The name of the database to create")
	params := parameters.Parameters{databaseParam}

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

	// Check if write mode is enabled
	if source.IsReadOnlyMode() {
		return nil, fmt.Errorf("create-database requires write mode to be enabled (set enableWriteMode: true)")
	}

	paramsMap := params.AsMap()
	databaseName, ok := paramsMap["database_name"].(string)
	if !ok {
		return nil, fmt.Errorf("database_name parameter is required and must be a string")
	}

	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting logger: %s", err)
	}

	createSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", databaseName)

	logger.DebugContext(ctx, fmt.Sprintf("executing `%s` tool: creating database %s", kind, databaseName))

	_, err = source.Query(ctx, createSQL)
	if err != nil {
		source.EmitTelemetry(ctx, cockroachdb.TelemetryEvent{
			Timestamp:   time.Now(),
			ToolName:    kind,
			SQLRedacted: "CREATE DATABASE IF NOT EXISTS ***",
			Status:      "failure",
			ErrorCode:   cockroachdb.ErrCodeQueryExecutionFailed,
			ErrorMsg:    err.Error(),
			LatencyMs:   time.Since(startTime).Milliseconds(),
		})
		return nil, fmt.Errorf("unable to create database: %w", err)
	}

	source.EmitTelemetry(ctx, cockroachdb.TelemetryEvent{
		Timestamp:    time.Now(),
		ToolName:     kind,
		SQLRedacted:  "CREATE DATABASE IF NOT EXISTS ***",
		Status:       "success",
		LatencyMs:    time.Since(startTime).Milliseconds(),
		RowsAffected: 1,
	})

	result := orderedmap.Row{}
	result.Add("database_name", databaseName)
	result.Add("status", "created")
	result.Add("message", fmt.Sprintf("Database '%s' created successfully", databaseName))

	return []any{result}, nil
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
