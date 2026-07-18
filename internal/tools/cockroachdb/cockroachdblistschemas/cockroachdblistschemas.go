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

package cockroachdblistschemas

import (
	"context"
	"fmt"
	"net/http"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/sources/cockroachdb"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	"github.com/jackc/pgx/v5"
)

const resourceType string = "cockroachdb-list-schemas"

const listSchemasStatement = `
	SELECT
		catalog_name,
		schema_name,
		crdb_is_user_defined
	FROM information_schema.schemata
	WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast', 'crdb_internal', 'pg_extension')
		AND schema_name NOT LIKE 'pg_temp_%'
		AND schema_name NOT LIKE 'pg_toast_temp_%'
	ORDER BY schema_name;
`

func init() {
	if !tools.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", resourceType))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{ConfigBase: tools.ConfigBase{Name: name}}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type compatibleSource interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
}

var _ compatibleSource = &cockroachdb.Source{}

type Config struct {
	tools.ConfigBase `yaml:",inline"`
	Type             string                 `yaml:"type" validate:"required"`
	Source           string                 `yaml:"source" validate:"required"`
	Annotations      *tools.ToolAnnotations `yaml:"annotations,omitempty"`
}

var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(context.Context) (tools.Tool, error) {
	if cfg.Description == "" {
		return nil, fmt.Errorf("description is required for tool %q", cfg.Name)
	}

	allParameters := parameters.Parameters{}
	paramManifest := allParameters.Manifest()

	return Tool{
		BaseTool: tools.NewBaseTool(
			cfg,
			tools.GetAnnotationsOrDefault(cfg.Annotations, tools.NewReadOnlyAnnotations),
			tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
			allParameters,
		),
	}, nil
}

var _ tools.Tool = Tool{}

type Tool struct {
	tools.BaseTool[Config]
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Cfg
}

func (t Tool) validate(srcs map[string]sources.Source) error {
	_, err := tools.GetCompatibleSourceFromMap[compatibleSource](srcs, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	return err
}

func (t Tool) GetParameters(srcs map[string]sources.Source) (parameters.Parameters, error) {
	if err := t.validate(srcs); err != nil {
		return nil, err
	}
	return t.BaseTool.GetParameters(srcs)
}

func (t Tool) Manifest(srcs map[string]sources.Source) (tools.Manifest, error) {
	if err := t.validate(srcs); err != nil {
		return tools.Manifest{}, err
	}
	return t.BaseTool.Manifest(srcs)
}

func (t Tool) Invoke(ctx context.Context, primitiveMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](primitiveMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	results, err := source.Query(ctx, listSchemasStatement)
	if err != nil {
		return nil, util.ProcessGeneralError(fmt.Errorf("unable to execute query: %w", err))
	}
	defer results.Close()

	fields := results.FieldDescriptions()
	var out []map[string]any

	for results.Next() {
		values, err := results.Values()
		if err != nil {
			return nil, util.NewClientServerError("unable to parse row", http.StatusInternalServerError, err)
		}
		rowMap := make(map[string]any)
		for i, field := range fields {
			rowMap[string(field.Name)] = values[i]
		}
		out = append(out, rowMap)
	}

	if err := results.Err(); err != nil {
		return nil, util.ProcessGeneralError(fmt.Errorf("error reading query results: %w", err))
	}

	return out, nil
}
