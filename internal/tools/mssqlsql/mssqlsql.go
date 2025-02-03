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

package mssqlsql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqlmssql"
	"github.com/googleapis/genai-toolbox/internal/sources/mssql"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const ToolKind string = "mssql-sql"

type compatibleSource interface {
	MSSQLDB() *sql.DB
}

// validate compatible sources are still compatible
var _ compatibleSource = &cloudsqlmssql.Source{}
var _ compatibleSource = &mssql.Source{}

var compatibleSources = [...]string{cloudsqlmssql.SourceKind, mssql.SourceKind}

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
		Db:           s.MSSQLDB(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: cfg.Parameters.Manifest()},
	}
	return t, nil
}

func NewGenericTool(name string, stmt string, authRequired []string, desc string, Db *sql.DB, parameters tools.Parameters) Tool {
	return Tool{
		Name:         name,
		Kind:         ToolKind,
		Statement:    stmt,
		AuthRequired: authRequired,
		Db:           Db,
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

	Db        *sql.DB
	Statement string
	manifest  tools.Manifest
}

func (t Tool) Invoke(params tools.ParamValues) (string, error) {
	fmt.Printf("Invoked tool %s\n", t.Name)

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
	rows, err := t.Db.QueryContext(context.Background(), t.Statement, namedArgs...)
	if err != nil {
		return "", fmt.Errorf("unable to execute query: %w", err)
	}

	types, err := rows.ColumnTypes()
	if err != nil {
		return "", fmt.Errorf("unable to fetch column types: %w", err)
	}
	v := make([]any, len(types))
	pointers := make([]any, len(types))
	for i := range types {
		pointers[i] = &v[i]
	}

	// fetch result into a string
	var out strings.Builder

	for rows.Next() {
		err = rows.Scan(pointers...)
		if err != nil {
			return "", fmt.Errorf("unable to parse row: %w", err)
		}
		out.WriteString(fmt.Sprintf("%s", v))
	}
	err = rows.Close()
	if err != nil {
		return "", fmt.Errorf("unable to close rows: %w", err)
	}

	// Check if error occured during iteration
	if err := rows.Err(); err != nil {
		return "", err
	}
	return fmt.Sprintf("Stub tool call for %q! Parameters parsed: %q \n Output: %s", t.Name, params, out.String()), nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data, claims)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) Authorized(verifiedAuthSources []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthSources)
}
