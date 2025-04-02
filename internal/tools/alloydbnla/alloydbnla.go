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

package alloydbnla

import (
	"context"
	"fmt"
	"strings"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/alloydbpg"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ToolKind string = "alloydb-ai-nl"

type compatibleSource interface {
	PostgresPool() *pgxpool.Pool
}

// validate compatible sources are still compatible
var _ compatibleSource = &alloydbpg.Source{}

var compatibleSources = [...]string{alloydbpg.SourceKind}

type Config struct {
	Name                string          `yaml:"name" validate:"required"`
	Kind                string          `yaml:"kind" validate:"required"`
	Source              string          `yaml:"source" validate:"required"`
	Description         string          `yaml:"description" validate:"required"`
	NLConfig            string          `yaml:"nlConfig" validate:"required"`
	AuthRequired       []string         `yaml:"authRequired"`
	NLConfigParameters tools.Parameters `yaml:"nlConfigParameters"`
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

	paramNames := make([]string, 0, len(cfg.NLConfigParameters))
	for _, paramDef := range cfg.NLConfigParameters {
		paramNames = append(paramNames, paramDef.GetName())
	}
	quotedParamNames := make([]string, len(paramNames))
	for i, name := range paramNames {
		// Basic escaping for single quotes within the name itself
		escapedName := strings.ReplaceAll(name, "'", "''")
		quotedParamNames[i] = fmt.Sprintf("'%s'", escapedName)
	}
	paramNamesSQL := "ARRAY []" // Default for no parameters
	if len(quotedParamNames) > 0 {
		paramNamesSQL = fmt.Sprintf("ARRAY [%s]", strings.Join(quotedParamNames, ", "))
	}
	paramValuePlaceholders := make([]string, len(paramNames))
	for i := 0; i < len(paramNames); i++ {
		// Placeholders start from $2 ($1 is reserved for the natural language query)
		paramValuePlaceholders[i] = fmt.Sprintf("$%d", i+2)
	}
	paramValuesSQL := "ARRAY []" // Default for no parameters
	if len(paramValuePlaceholders) > 0 {
		paramValuesSQL = fmt.Sprintf("ARRAY [%s]", strings.Join(paramValuePlaceholders, ", "))
	}

	// execute_nl_query is the AlloyDB AI function that executes the natural language query
	// The first parameter is the natural language query, which is passed as $1
	// The second parameter is the NLConfig, which is passed as a string
	// The third and fourth parameters are the list of nl_config parameter names and values, respectively
	stmtFormat := "SELECT alloydb_ai_nl.execute_nl_query($1, '%s', param_names => %s, param_values => %s);"
	stmt := fmt.Sprintf(stmtFormat, cfg.NLConfig, paramNamesSQL, paramValuesSQL)

	newQuestionParam := tools.NewStringParameter(
    "question",                      				 // name
    "The natural language question to ask.", // description
	)

	cfg.NLConfigParameters = append([]tools.Parameter{newQuestionParam}, cfg.NLConfigParameters...)

	t := Tool{
		Name:         cfg.Name,
		Kind:         ToolKind,
		Parameters:   cfg.NLConfigParameters,
		Statement:    stmt,
		AuthRequired: cfg.AuthRequired,
		Pool:         s.PostgresPool(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: cfg.NLConfigParameters.Manifest()},
	}

	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`

	Pool      *pgxpool.Pool
	Statement string
	manifest  tools.Manifest
}

func (t Tool) Invoke(params tools.ParamValues) ([]any, error) {
	sliceParams := params.AsSlice()
	results, err := t.Pool.Query(context.Background(), t.Statement, sliceParams...)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w. Query: %v , Values: %v", err, t.Statement, sliceParams)
	}

	fields := results.FieldDescriptions()

	var out []any
	for results.Next() {
		v, err := results.Values()
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}
		vMap := make(map[string]any)
		for i, f := range fields {
			vMap[f.Name] = v[i]
		}
		out = append(out, vMap)
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
