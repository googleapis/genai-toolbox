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

package bigtable

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigtable"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	bigtabledb "github.com/googleapis/genai-toolbox/internal/sources/bigtable"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "bigtable-sql"

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
	BigtableClient() *bigtable.Client
}

// validate compatible sources are still compatible
var _ compatibleSource = &bigtabledb.Source{}

var compatibleSources = [...]string{bigtabledb.SourceKind}

type Config struct {
	Name               string           `yaml:"name" validate:"required"`
	Kind               string           `yaml:"kind" validate:"required"`
	Source             string           `yaml:"source" validate:"required"`
	Description        string           `yaml:"description" validate:"required"`
	Statement          string           `yaml:"statement" validate:"required"`
	AuthRequired       []string         `yaml:"authRequired"`
	Parameters         tools.Parameters `yaml:"parameters"`
	TemplateParameters tools.Parameters `yaml:"templateParameters"`
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

	allParameters, paramManifest, paramMcpManifest := tools.ProcessParameters(cfg.TemplateParameters, cfg.Parameters)

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: paramMcpManifest,
	}

	// finish tool setup
	t := Tool{
		Name:               cfg.Name,
		Kind:               kind,
		Parameters:         cfg.Parameters,
		TemplateParameters: cfg.TemplateParameters,
		AllParams:          allParameters,
		Statement:          cfg.Statement,
		AuthRequired:       cfg.AuthRequired,
		Client:             s.BigtableClient(),
		manifest:           tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:        mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name               string           `yaml:"name"`
	Kind               string           `yaml:"kind"`
	AuthRequired       []string         `yaml:"authRequired"`
	Parameters         tools.Parameters `yaml:"parameters"`
	TemplateParameters tools.Parameters `yaml:"templateParameters"`
	AllParams          tools.Parameters `yaml:"allParams"`

	Client      *bigtable.Client
	Statement   string
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func getMapParamsType(tparams tools.Parameters, params tools.ParamValues) (map[string]bigtable.SQLType, error) {
	paramTypeMap := make(map[string]string)
	for _, p := range tparams {
		paramTypeMap[p.GetName()] = p.GetType()
	}

	btParams := make(map[string]bigtable.SQLType)
	for _, p := range params {
		switch paramTypeMap[p.Name] {
		case "boolean":
			btParams[p.Name] = bigtable.BoolSQLType{}
		case "string":
			btParams[p.Name] = bigtable.StringSQLType{}
		case "integer":
			btParams[p.Name] = bigtable.Int64SQLType{}
		case "float":
			btParams[p.Name] = bigtable.Float64SQLType{}
		case "array":
			btParams[p.Name] = bigtable.ArraySQLType{}
		}
	}

	return btParams, nil
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	paramsMap := params.AsMap()
	newStatement, err := tools.ResolveTemplateParams(t.TemplateParameters, t.Statement, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract template params %w", err)
	}

	newParams, err := tools.GetParams(t.Parameters, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract standard params %w", err)
	}

	mapParamsType, err := getMapParamsType(t.Parameters, newParams)
	if err != nil {
		return nil, fmt.Errorf("fail to get map params: %w", err)
	}

	ps, err := t.Client.PrepareStatement(
		ctx,
		newStatement,
		mapParamsType,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to prepare statement: %w", err)
	}

	bs, err := ps.Bind(newParams.AsMap())
	if err != nil {
		return nil, fmt.Errorf("unable to bind: %w", err)
	}

	var out []any
	err = bs.Execute(ctx, func(resultRow bigtable.ResultRow) bool {
		vMap := make(map[string]any)
		cols := resultRow.Metadata.Columns

		for _, c := range cols {
			var columValue any
			err = resultRow.GetByName(c.Name, &columValue)
			vMap[c.Name] = columValue
		}

		out = append(out, vMap)

		return true
	})
	if err != nil {
		return nil, fmt.Errorf("unable to execute client: %w", err)
	}

	return out, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.AllParams, data, claims)
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
