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
	"database/sql"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

type compatibleSource interface {
	ClickHousePool() *sql.DB
}

var compatibleSources = []string{"clickhouse"}

const listDatabasesKind string = "clickhouse-list-databases"

func init() {
	if !tools.Register(listDatabasesKind, newListDatabasesConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", listDatabasesKind))
	}
}

func newListDatabasesConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name         string                `yaml:"name" validate:"required"`
	Kind         string                `yaml:"kind" validate:"required"`
	Source       string                `yaml:"source" validate:"required"`
	Description  string                `yaml:"description" validate:"required"`
	AuthRequired []string              `yaml:"authRequired"`
	Parameters   parameters.Parameters `yaml:"parameters"`
}

var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return listDatabasesKind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", listDatabasesKind, compatibleSources)
	}

	allParameters, paramManifest, _ := parameters.ProcessParameters(nil, cfg.Parameters)
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	t := Tool{
		Config:      cfg,
		AllParams:   allParameters,
		Pool:        s.ClickHousePool(),
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

var _ tools.Tool = Tool{}

type Tool struct {
	Config
	AllParams parameters.Parameters `yaml:"allParams"`

	Pool        *sql.DB
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, token tools.AccessToken) (any, error) {
	// Query to list all databases
	query := "SHOW DATABASES"

	results, err := t.Pool.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer results.Close()

	var databases []map[string]any
	for results.Next() {
		var dbName string
		err := results.Scan(&dbName)
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}
		databases = append(databases, map[string]any{
			"name": dbName,
		})
	}

	if err := results.Err(); err != nil {
		return nil, fmt.Errorf("errors encountered by results.Scan: %w", err)
	}

	return databases, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.AllParams, data, claims)
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

func (t Tool) RequiresClientAuthorization() bool {
	return false
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
