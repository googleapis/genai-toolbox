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

package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/jackc/pgx/v5/pgxpool"
)

const PostgresSQLGenericKind string = "postgres-generic"

// validate interface
var _ Config = PostgresGenericConfig{}

type PostgresGenericConfig struct {
	Name        string     `yaml:"name"`
	Kind        string     `yaml:"kind"`
	Source      string     `yaml:"source"`
	Description string     `yaml:"description"`
	Statement   string     `yaml:"statement"`
	Parameters  Parameters `yaml:"parameters"`
}

func (cfg PostgresGenericConfig) toolKind() string {
	return PostgresSQLGenericKind
}

func (cfg PostgresGenericConfig) Initialize(srcs map[string]sources.Source) (Tool, error) {
	// verify source exists
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is the right kind
	s, ok := rawS.(sources.PostgresSource)
	if !ok {
		return nil, fmt.Errorf("sources for %q tools must be of kind %q", PostgresSQLGenericKind, sources.PostgresKind)
	}

	// finish tool setup
	t := PostgresGenericTool{
		Name:       cfg.Name,
		Kind:       PostgresSQLGenericKind,
		Pool:       s.Pool,
		Statement:  cfg.Statement,
		Parameters: cfg.Parameters,
		manifest:   ToolManifest{cfg.Description, generateManifests(cfg.Parameters)},
	}
	return t, nil
}

// validate interface
var _ Tool = PostgresGenericTool{}

type PostgresGenericTool struct {
	Name       string `yaml:"name"`
	Kind       string `yaml:"kind"`
	Pool       *pgxpool.Pool
	Statement  string
	Parameters Parameters `yaml:"parameters"`
	manifest   ToolManifest
}

func (t PostgresGenericTool) Invoke(params []any) (string, error) {
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

func (t PostgresGenericTool) ParseParams(data map[string]any) ([]any, error) {
	return ParseParams(t.Parameters, data)
}

func (t PostgresGenericTool) Manifest() ToolManifest {
	return t.manifest
}
