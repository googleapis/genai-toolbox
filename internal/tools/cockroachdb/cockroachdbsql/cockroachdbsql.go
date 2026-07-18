// Copyright 2026 Google LLC
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

package cockroachdbsql

import (
	"context"
	"fmt"
	"net/http"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/sources/cockroachdb"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/orderedmap"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	"github.com/jackc/pgx/v5"
)

const resourceType string = "cockroachdb-sql"

func init() {
	if !tools.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", resourceType))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{ConfigBase: tools.ConfigBase{Name: name}}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type compatibleSource interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
}

var _ compatibleSource = &cockroachdb.Source{}

type Config struct {
	tools.ConfigBase   `yaml:",inline"`
	Type               string                 `yaml:"type" validate:"required"`
	Source             string                 `yaml:"source" validate:"required"`
	Statement          string                 `yaml:"statement" validate:"required"`
	Parameters         parameters.Parameters  `yaml:"parameters"`
	TemplateParameters parameters.Parameters  `yaml:"templateParameters"`
	Annotations        *tools.ToolAnnotations `yaml:"annotations,omitempty"`
}

var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(context.Context) (tools.Tool, error) {
	if cfg.Description == "" {
		return nil, fmt.Errorf("description is required for tool %q", cfg.Name)
	}

	allParameters, paramManifest, err := parameters.ProcessParameters(cfg.TemplateParameters, cfg.Parameters)
	if err != nil {
		return nil, err
	}

	return Tool{
		BaseTool: tools.NewBaseTool(
			cfg,
			tools.GetAnnotationsOrDefault(cfg.Annotations, tools.NewDestructiveAnnotations),
			tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
			allParameters,
		),
	}, nil
}

var _ tools.Tool = Tool{}

type Tool struct {
	tools.BaseTool[Config]
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Cfg
}

func (t Tool) validate(srcs map[string]sources.Source) error {
	_, err := tools.GetCompatibleSourceFromMap[compatibleSource](srcs, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	return err
}

func (t Tool) GetParameters(srcs map[string]sources.Source) (parameters.Parameters, error) {
	if err := t.validate(srcs); err != nil {
		return nil, err
	}
	return t.BaseTool.GetParameters(srcs)
}

func (t Tool) Manifest(srcs map[string]sources.Source) (tools.Manifest, error) {
	if err := t.validate(srcs); err != nil {
		return tools.Manifest{}, err
	}
	return t.BaseTool.Manifest(srcs)
}

func (t Tool) Invoke(ctx context.Context, primitiveMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](primitiveMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	paramsMap := params.AsMap()
	newStatement, err := parameters.ResolveTemplateParams(t.Cfg.TemplateParameters, t.Cfg.Statement, paramsMap)
	if err != nil {
		return nil, util.NewAgentError(fmt.Sprintf("unable to resolve template params: %v", err), err)
	}

	newParams, err := parameters.GetParams(t.Cfg.Parameters, paramsMap)
	if err != nil {
		return nil, util.NewAgentError(fmt.Sprintf("unable to extract standard params: %v", err), err)
	}
	sliceParams := newParams.AsSlice()
	results, err := source.Query(ctx, newStatement, sliceParams...)
	if err != nil {
		return nil, util.ProcessGeneralError(fmt.Errorf("unable to execute query: %w", err))
	}
	defer results.Close()

	fields := results.FieldDescriptions()

	out := []any{}
	for results.Next() {
		v, err := results.Values()
		if err != nil {
			return nil, util.NewClientServerError("unable to parse row", http.StatusInternalServerError, err)
		}
		row := orderedmap.Row{}
		for i, f := range fields {
			row.Add(f.Name, v[i])
		}
		out = append(out, row)
	}

	if err := results.Err(); err != nil {
		return nil, util.ProcessGeneralError(fmt.Errorf("unable to execute query: %w", err))
	}

	return out, nil
}
