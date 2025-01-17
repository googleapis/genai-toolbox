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

package mssql

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqlmssql"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const ToolKind string = "mssql"

type compatibleSource interface {
	MssqlDb() *sql.DB
}

// validate compatible sources are still compatible
var _ compatibleSource = &cloudsqlmssql.Source{}

var compatibleSources = [...]string{cloudsqlmssql.SourceKind}

type Config struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	Source       string           `yaml:"source"`
	Description  string           `yaml:"description"`
	Statement    string           `yaml:"statement"`
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
		Db:           s.MssqlDb(),
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

func createTypedRow(types []*sql.ColumnType) []any {
	v := make([]any, len(types))
	for i := range v {
		switch types[i].DatabaseTypeName() {
		case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
			v[i] = new(sql.NullString)
		case "BOOL":
			v[i] = new(sql.NullBool)
		case "INT":
			v[i] = new(sql.NullInt32)
		case "BIGINT":
			v[i] = new(sql.NullInt64)
		case "DECIMAL":
			v[i] = new(sql.NullFloat64)
		default:
			v[i] = new(sql.NullString)
		}
	}
	return v
}

func (t Tool) Invoke(params tools.ParamValues) (string, error) {
	fmt.Printf("Invoked tool %s\n", t.Name)

	// Convert params into named
	namedArgs := make([]any, 0, len(params))
	paramsMap := params.AsReversedMap()
	for _, v := range params.AsSlice() {
		paramName := paramsMap[v]
		if strings.Contains(t.Statement, "@"+paramName) {
			namedArgs = append(namedArgs, sql.Named(paramName, v))
		} else {
			namedArgs = append(namedArgs, v)
		}
	}
	rows, err := t.Db.Query(t.Statement, namedArgs...)
	if err != nil {
		return "", fmt.Errorf("unable to execute query: %w", err)
	}

	types, err := rows.ColumnTypes()
	if err != nil {
		return "", fmt.Errorf("unable to fetch column types: %w", err)
	}
	v := createTypedRow(types)

	// fetch result into a string
	var out strings.Builder

	for rows.Next() {
		err = rows.Scan(v...)
		if err != nil {
			return "", fmt.Errorf("unable to parse row: %w", err)
		}
		out.WriteString("[")
		for i, res := range v {
			if i > 0 {
				out.WriteString(" ")
			}
			// Print output variables as string to match other tools' output
			if resValue, ok := res.(*sql.NullBool); ok {
				out.WriteString(fmt.Sprintf("%s", resValue.Bool)) //nolint:all
				continue
			}
			if resValue, ok := res.(*sql.NullString); ok {
				out.WriteString(resValue.String) //nolint:all
				continue
			}
			if resValue, ok := res.(*sql.NullInt32); ok {
				out.WriteString(fmt.Sprintf("%s", resValue.Int32)) //nolint:all
				continue
			}
			if resValue, ok := res.(*sql.NullInt64); ok {
				out.WriteString(fmt.Sprintf("%s", resValue.Int64)) //nolint:all
				continue
			}
			if resValue, ok := res.(*sql.NullFloat64); ok {
				out.WriteString(fmt.Sprintf("%s", resValue.Float64)) //nolint:all
				continue
			}
		}
		out.WriteString("]")
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
