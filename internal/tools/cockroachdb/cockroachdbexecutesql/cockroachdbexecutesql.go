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

package cockroachdbexecutesql

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/cockroachdb"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/orderedmap"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"github.com/jackc/pgx/v5"
)

const kind string = "cockroachdb-execute-sql"

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
}

var _ compatibleSource = &cockroachdb.Source{}

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

	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	sqlParameter := parameters.NewStringParameter("sql", "The sql to execute.")
	params := parameters.Parameters{sqlParameter}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params)

	t := Tool{
		Config:      cfg,
		Parameters:  params,
		Source:      s,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

var _ tools.Tool = Tool{}

type Tool struct {
	Config
	Parameters parameters.Parameters `yaml:"parameters"`

	Source      compatibleSource
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()
	sql, ok := paramsMap["sql"].(string)
	if !ok {
		return nil, fmt.Errorf("unable to get cast %s", paramsMap["sql"])
	}
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting logger: %s", err)
	}
	logger.DebugContext(ctx, fmt.Sprintf("executing `%s` tool query: %s", kind, sql))

	results, err := t.Source.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer results.Close()

	fields := results.FieldDescriptions()

	var out []any
	for results.Next() {
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
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}

	return out, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Parameters, data, claims)
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

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
