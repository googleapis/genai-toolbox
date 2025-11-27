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

package postgresdatabaseoverview

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"github.com/jackc/pgx/v5/pgxpool"
)

const kind string = "postgres-database-overview"

const databaseOverviewStatement = `
	SELECT
    current_setting('server_version') AS pg_version,
    pg_is_in_recovery() AS is_replica,
    (now() - pg_postmaster_start_time())::TEXT AS uptime,
    current_setting('max_connections')::int AS max_connections,
	(SELECT count(*) FROM pg_stat_activity) AS current_connections,
	(SELECT count(*) FROM pg_stat_activity WHERE state = 'active') AS active_connections,
    round(
        (100.0 * (SELECT count(*) FROM pg_stat_activity) / current_setting('max_connections')::int),
        2
    ) AS pct_connections_used;
`

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
	PostgresPool() *pgxpool.Pool
}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	allParameters := parameters.Parameters{}
	description := cfg.Description
	if description == "" {
		description = "Fetches the current state of the PostgreSQL server, returning the version, whether it's a replica, uptime duration, maximum connection limit, number of current connections, number of active connections, and the percentage of connections in use."
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, description, cfg.AuthRequired, allParameters)

	// finish tool setup
	return Tool{
		Config:    cfg,
		allParams: allParameters,
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   allParameters.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	allParams   parameters.Parameters `yaml:"allParams"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	s, ok := resourceMgr.GetSource(t.Source)
	if !ok {
		return nil, fmt.Errorf("unable to retrieve source %s in tool %s", t.Source, t.Name)
	}
	source, ok := s.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source %q not compatible", kind, t.Source)
	}

	sliceParams := params.AsSlice()

	results, err := source.PostgresPool().Query(ctx, databaseOverviewStatement, sliceParams...)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer results.Close()

	fields := results.FieldDescriptions()
	var out []map[string]any

	for results.Next() {
		values, err := results.Values()
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}
		rowMap := make(map[string]any)
		for i, field := range fields {
			rowMap[string(field.Name)] = values[i]
		}
		out = append(out, rowMap)
	}

	// this will catch actual query execution errors
	if err := results.Err(); err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}

	return out, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.allParams, data, claims)
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

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
