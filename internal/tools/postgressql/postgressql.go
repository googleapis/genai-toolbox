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

package postgressql

import (
	"context"
	"fmt"
	"strings"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/alloydbpg"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqlpg"
	"github.com/googleapis/genai-toolbox/internal/sources/postgres"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ToolKind string = "postgres-sql"

type compatibleSource interface {
	PostgresPool() *pgxpool.Pool
}

// validate compatible sources are still compatible
var _ compatibleSource = &alloydbpg.Source{}
var _ compatibleSource = &cloudsqlpg.Source{}
var _ compatibleSource = &postgres.Source{}

var compatibleSources = [...]string{alloydbpg.SourceKind, cloudsqlpg.SourceKind, postgres.SourceKind}

type Config struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	Source       string           `yaml:"source"`
	AuthRequired bool             `yaml:"auth_required"`
	Description  string           `yaml:"description"`
	Statement    string           `yaml:"statement"`
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
		AuthRequired: cfg.AuthRequired,
		Statement:    cfg.Statement,
		Pool:         s.PostgresPool(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: cfg.Parameters.Manifest()},
	}
	return t, nil
}

func NewGenericTool(name, stmt, desc string, pool *pgxpool.Pool, parameters tools.Parameters) Tool {
	return Tool{
		Name:       name,
		Kind:       ToolKind,
		Statement:  stmt,
		Pool:       pool,
		manifest:   tools.Manifest{Description: desc, Parameters: parameters.Manifest()},
		Parameters: parameters,
	}
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	Parameters   tools.Parameters `yaml:"parameters"`
	AuthRequired bool             `yaml:"auth_required"`
	Pool         *pgxpool.Pool
	Statement    string
	manifest     tools.Manifest
}

func (t Tool) Invoke(params []any) (string, error) {
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

func (t Tool) ParseParams(data map[string]any, claims map[string]any) ([]any, error) {
	return tools.ParseParams(t.Parameters, data, claims)
}

func (t Tool) Authenticate(authSource string, authToken string) (map[string]any, error) {
	claims := make(map[string]any)
	return claims, nil
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}
