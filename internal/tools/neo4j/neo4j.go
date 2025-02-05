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

package neo4j

import (
	"context"
	"fmt"

	neo4jsc "github.com/googleapis/genai-toolbox/internal/sources/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const ToolKind string = "neo4j-cypher"

type compatibleSource interface {
	Neo4jDriver() neo4j.DriverWithContext
	Neo4jDatabase() string
}

// validate compatible sources are still compatible
var _ compatibleSource = &neo4jsc.Source{}

var compatibleSources = [...]string{neo4jsc.SourceKind}

type Config struct {
	Name        string           `yaml:"name" validate:"required"`
	Kind        string           `yaml:"kind" validate:"required"`
	Source      string           `yaml:"source" validate:"required"`
	Description string           `yaml:"description" validate:"required"`
	Statement   string           `yaml:"statement" validate:"required"`
	Parameters  tools.Parameters `yaml:"parameters"`
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
		Name:       cfg.Name,
		Kind:       ToolKind,
		Parameters: cfg.Parameters,
		Statement:  cfg.Statement,
		Driver:     s.Neo4jDriver(),
		Database:   s.Neo4jDatabase(),
		manifest:   tools.Manifest{Description: cfg.Description, Parameters: cfg.Parameters.Manifest()},
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	Parameters   tools.Parameters `yaml:"parameters"`
	AuthRequired []string         `yaml:"authRequired"`

	Driver    neo4j.DriverWithContext
	Database  string
	Statement string
	manifest  tools.Manifest
}

func (t Tool) Invoke(params tools.ParamValues) ([]any, error) {
	paramsMap := params.AsMap()

	ctx := context.Background()
	config := neo4j.ExecuteQueryWithDatabase(t.Database)
	results, err := neo4j.ExecuteQuery[*neo4j.EagerResult](ctx, t.Driver, t.Statement, paramsMap,
		neo4j.EagerResultTransformer, config)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}

	var out []any
	keys := results.Keys
	records := results.Records
	for _, record := range records {
		vMap := make(map[string]any)
		for col, value := range record.Values {
			vMap[keys[col]] = value
		}
		out = append(out, vMap)
	}

	return out, nil
}

func (t Tool) ParseParams(data map[string]any, claimsMap map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data, claimsMap)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) Authorized(verifiedAuthSources []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthSources)
}
