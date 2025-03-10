// Copyright 2024 Google LLC
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

package spanner

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/googleapis/genai-toolbox/internal/sources"
	spannerdb "github.com/googleapis/genai-toolbox/internal/sources/spanner"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const ToolKind string = "spanner-sql"

type compatibleSource interface {
	SpannerPool() *sql.DB
	DatabaseDialect() string
}

// validate compatible sources are still compatible
var _ compatibleSource = &spannerdb.Source{}

var compatibleSources = [...]string{spannerdb.SourceKind}

type Config struct {
	Name         string           `yaml:"name" validate:"required"`
	Kind         string           `yaml:"kind" validate:"required"`
	Source       string           `yaml:"source" validate:"required"`
	Description  string           `yaml:"description" validate:"required"`
	Statement    string           `yaml:"statement" validate:"required"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return ToolKind
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
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", ToolKind, compatibleSources)
	}

	// finish tool setup
	t := Tool{
		Name:         cfg.Name,
		Kind:         ToolKind,
		Parameters:   cfg.Parameters,
		Statement:    cfg.Statement,
		AuthRequired: cfg.AuthRequired,
		Pool:         s.SpannerPool(),
		dialect:      s.DatabaseDialect(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: cfg.Parameters.Manifest()},
	}
	return t, nil
}

func NewGenericTool(name string, stmt string, authRequired []string, desc string, db *sql.DB, dialect string, parameters tools.Parameters) Tool {
	return Tool{
		Name:         name,
		Kind:         ToolKind,
		Statement:    stmt,
		AuthRequired: authRequired,
		Pool:         db,
		dialect:      dialect,
		manifest:     tools.Manifest{Description: desc, Parameters: parameters.Manifest()},
		Parameters:   parameters,
	}
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`

	Pool      *sql.DB
	dialect   string
	Statement string
	manifest  tools.Manifest
}

func (t Tool) Invoke(params tools.ParamValues) ([]any, error) {
	namedArgs := make([]any, 0, len(params))
	paramsMap := params.AsReversedMap()
	// To support both named args (e.g @id) and positional args (e.g @p1), check if arg name is contained in the statement.
	for _, v := range params.AsSlice() {
		paramName := paramsMap[v]
		if strings.Contains(t.Statement, "@"+paramName) {
			namedArgs = append(namedArgs, sql.Named(paramName, v))
		} else {
			namedArgs = append(namedArgs, v)
		}
	}

	rows, err := t.Pool.QueryContext(context.Background(), t.Statement, namedArgs...)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("unable to fetch column types: %w", err)
	}

	// create an array of values for each column, which can be re-used to scan each row
	rawValues := make([]any, len(cols))
	values := make([]any, len(cols))
	for i := range rawValues {
		values[i] = &rawValues[i]
	}

	var out []any
	for rows.Next() {
		err = rows.Scan(values...)
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}
		vMap := make(map[string]any)
		for i, name := range cols {
			vMap[name] = rawValues[i]
		}
		out = append(out, vMap)
	}
	err = rows.Close()
	if err != nil {
		return nil, fmt.Errorf("unable to close rows: %w", err)
	}

	// Check if error occured during iteration
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data, claims)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}
