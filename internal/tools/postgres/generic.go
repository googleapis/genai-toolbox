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

package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/postgres"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ToolKind string = "postgres-generic"

// validate interface
var _ tools.Config = GenericConfig{}

type GenericConfig struct {
	Name        string           `yaml:"name"`
	Kind        string           `yaml:"kind"`
	Source      string           `yaml:"source"`
	Description string           `yaml:"description"`
	Statement   string           `yaml:"statement"`
	Parameters  tools.Parameters `yaml:"parameters"`
}

func (cfg GenericConfig) ToolKind() string {
	return ToolKind
}

func (cfg GenericConfig) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is the right kind
	s, ok := rawS.(postgres.Source)
	if !ok {
		return nil, fmt.Errorf("sources for %q tools must be of kind %q", ToolKind, postgres.SourceKind)
	}

	// finish tool setup
	t := NewGenericTool(cfg.Name, cfg.Statement, cfg.Description, s.Pool, cfg.Parameters)
	return t, nil
}

func NewGenericTool(name, stmt, desc string, pool *pgxpool.Pool, parameters tools.Parameters) GenericTool {
	return GenericTool{
		Name:       name,
		Kind:       ToolKind,
		Statement:  stmt,
		Pool:       pool,
		manifest:   tools.Manifest{Description: desc, Parameters: parameters.Manifest()},
		Parameters: parameters,
	}
}

// validate interface
var _ tools.Tool = GenericTool{}

type GenericTool struct {
	Name       string           `yaml:"name"`
	Kind       string           `yaml:"kind"`
	Parameters tools.Parameters `yaml:"parameters"`

	Pool      *pgxpool.Pool
	Statement string
	manifest  tools.Manifest
}

func (t GenericTool) Invoke(params []any) (string, error) {
	fmt.Printf("Invoked tool %s\n", t.Name)
	results, err := t.Pool.Query(context.Background(), t.Statement, params...)
	if err != nil {
		return "", fmt.Errorf("unable to execute query: %w", err)
	}

	var out strings.Builder
	for results.Next() {
		v, err := results.Values()
		if err != nil {
			return "", fmt.Errorf("unable to parse row: %w", err)
		}
		out.WriteString(fmt.Sprintf("%s", v))
	}

	return fmt.Sprintf("Stub tool call for %q! Parameters parsed: %q \n Output: %s", t.Name, params, out.String()), nil
}

func (t GenericTool) ParseParams(data map[string]any) ([]any, error) {
	return tools.ParseParams(t.Parameters, data)
}

func (t GenericTool) Manifest() tools.Manifest {
	return t.manifest
}
