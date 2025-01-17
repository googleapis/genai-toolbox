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

package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqlmysql"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const ToolKind string = "mysql"

type compatibleSource interface {
	MySQLPool() *sql.DB
}

// validate compatible sources are still compatible
var _ compatibleSource = &cloudsqlmysql.Source{}

var compatibleSources = [...]string{cloudsqlmysql.SourceKind}

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
		Pool:         s.MySQLPool(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: cfg.Parameters.Manifest()},
	}
	return t, nil
}

func NewGenericTool(name string, stmt string, authRequired []string, desc string, pool *sql.DB, parameters tools.Parameters) Tool {
	return Tool{
		Name:         name,
		Kind:         ToolKind,
		Statement:    stmt,
		AuthRequired: authRequired,
		Pool:         pool,
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
	Statement string
	manifest  tools.Manifest
}

func (t Tool) Invoke(params tools.ParamValues) (string, error) {
	sliceParams := params.AsSlice()

	results, err := t.Pool.QueryContext(context.Background(), t.Statement, sliceParams...)
	if err != nil {
		return "", fmt.Errorf("unable to execute query: %w", err)
	}

	cols, err := results.Columns()
	if err != nil {
		return "", fmt.Errorf("unable to retrieve rows column name: %w", err)
	}

	cl := len(cols)
	v := make([]any, cl)
	pointers := make([]any, cl)
	for i := range v {
		pointers[i] = &v[i]
	}
	var out strings.Builder
	for results.Next() {
		err := results.Scan(pointers...)
		if err != nil {
			return "", fmt.Errorf("unable to parse row: %w", err)
		}
		out.WriteString(fmt.Sprintf("%s", v))
	}

	err = results.Close()
	if err != nil {
		return "", fmt.Errorf("unable to close rows: %w", err)
	}

	if err := results.Err(); err != nil {
		return "", fmt.Errorf("errors encountered by results.Scan: %w", err)
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
