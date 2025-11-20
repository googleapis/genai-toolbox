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

package firebirdsql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/firebird"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind string = "firebird-sql"

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
	FirebirdDB() *sql.DB
}

// validate compatible sources are still compatible
var _ compatibleSource = &firebird.Source{}

var compatibleSources = [...]string{firebird.SourceKind}

type Config struct {
	Name               string                `yaml:"name" validate:"required"`
	Kind               string                `yaml:"kind" validate:"required"`
	Source             string                `yaml:"source" validate:"required"`
	Description        string                `yaml:"description" validate:"required"`
	Statement          string                `yaml:"statement" validate:"required"`
	AuthRequired       []string              `yaml:"authRequired"`
	Parameters         parameters.Parameters `yaml:"parameters"`
	TemplateParameters parameters.Parameters `yaml:"templateParameters"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	allParameters, paramManifest, err := parameters.ProcessParameters(cfg.TemplateParameters, cfg.Parameters)
	if err != nil {
		return nil, err
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		AllParams:   allParameters,
		Db:          s.FirebirdDB(),
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

	Db          *sql.DB
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()
	statement, err := parameters.ResolveTemplateParams(t.TemplateParameters, t.Statement, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract template params: %w", err)
	}

	newParams, err := parameters.GetParams(t.Parameters, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract standard params: %w", err)
	}

	namedArgs := make([]any, 0, len(newParams))
	// To support both named args (e.g :id) and positional args (e.g ?), check
	// if arg name is contained in the statement.
	for _, p := range t.Parameters {
		name := p.GetName()
		value := paramsMap[name]
		if strings.Contains(statement, ":"+name) {
			namedArgs = append(namedArgs, sql.Named(name, value))
		} else {
			namedArgs = append(namedArgs, value)
		}
	}

	rows, err := t.Db.QueryContext(ctx, statement, namedArgs...)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("unable to get columns: %w", err)
	}

	values := make([]any, len(cols))
	scanArgs := make([]any, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	var out []any
	for rows.Next() {

		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}

		vMap := make(map[string]any)
		for i, col := range cols {
			if b, ok := values[i].([]byte); ok {
				vMap[col] = string(b)
			} else {
				vMap[col] = values[i]
			}
		}
		out = append(out, vMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// In most cases, DML/DDL statements like INSERT, UPDATE, CREATE, etc. might return no rows
	// However, it is also possible that this was a query that was expected to return rows
	// but returned none, a case that we cannot distinguish here.
	return out, nil
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
