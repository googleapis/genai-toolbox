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

package bigquerysql

import (
	"context"
	"fmt"
	"strings"

	bigqueryapi "cloud.google.com/go/bigquery"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	bigqueryds "github.com/googleapis/genai-toolbox/internal/sources/bigquery"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"google.golang.org/api/iterator"
)

const kind string = "bigquery-sql"

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
	BigQueryClient() *bigqueryapi.Client
}

// validate compatible sources are still compatible
var _ compatibleSource = &bigqueryds.Source{}

var compatibleSources = [...]string{bigqueryds.SourceKind}

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
		Client:             s.BigQueryClient(),
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

	Client      *bigqueryapi.Client
	Statement   string
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	namedArgs := make([]bigqueryapi.QueryParameter, 0, len(params))
	paramsMap := params.AsMap()
	newStatement, err := tools.ResolveTemplateParams(t.TemplateParameters, t.Statement, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract template params %w", err)
	}

	for _, p := range t.Parameters {
		name := p.GetName()
		value := paramsMap[name]

		// BigQuery's QueryParameter only accepts typed slices as input
		// This checks if the param is an array.
		// If yes, convert []any to typed slice (e.g []string, []int)
		switch arrayParam := value.(type) {
		case []any:
			var err error
			itemType := p.McpManifest().Items.Type
			value, err = convertAnySliceToTyped(arrayParam, itemType, name)
			if err != nil {
				return nil, fmt.Errorf("unable to convert []any to typed slice: %w", err)
			}
		}

		if strings.Contains(t.Statement, "@"+name) {
			namedArgs = append(namedArgs, bigqueryapi.QueryParameter{
				Name:  name,
				Value: value,
			})
		} else {
			namedArgs = append(namedArgs, bigqueryapi.QueryParameter{
				Value: value,
			})
		}
	}

	query := t.Client.Query(newStatement)
	query.Parameters = namedArgs
	query.Location = t.Client.Location

	it, err := query.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}

	var out []any
	for {
		var row map[string]bigqueryapi.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("unable to iterate through query results: %w", err)
		}
		vMap := make(map[string]any)
		for key, value := range row {
			vMap[key] = value
		}
		out = append(out, vMap)
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

func convertAnySliceToTyped(s []any, itemType, paramName string) (any, error) {
	var typedSlice any
	switch itemType {
	case "string":
		tempSlice := make([]string, len(s))
		for j, item := range s {
			if s, ok := item.(string); ok {
				tempSlice[j] = s
			} else {
				return nil, fmt.Errorf("parameter '%s': expected item at index %d to be string, got %T", paramName, j, item)
			}
			typedSlice = tempSlice
		}
		typedSlice = tempSlice
	case "integer":
		tempSlice := make([]int64, len(s))
		for j, item := range s {
			i, ok := item.(int)
			if !ok {
				return nil, fmt.Errorf("parameter '%s': expected item at index %d to be integer, got %T", paramName, j, item)
			}
			tempSlice[j] = int64(i)
		}
		typedSlice = tempSlice
	case "float":
		tempSlice := make([]float64, len(s))
		for j, item := range s {
			if f, ok := item.(float64); ok {
				tempSlice[j] = f
			} else {
				return nil, fmt.Errorf("parameter '%s': expected item at index %d to be float, got %T", paramName, j, item)
			}
		}
		typedSlice = tempSlice
	case "boolean":
		tempSlice := make([]bool, len(s))
		for j, item := range s {
			if b, ok := item.(bool); ok {
				tempSlice[j] = b
			} else {
				return nil, fmt.Errorf("parameter '%s': expected item at index %d to be boolean, got %T", paramName, j, item)
			}
		}
		typedSlice = tempSlice
	}
	return typedSlice, nil
}
