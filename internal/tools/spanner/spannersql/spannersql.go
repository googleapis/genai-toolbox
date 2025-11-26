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

package spannersql

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	spannerdb "github.com/googleapis/genai-toolbox/internal/sources/spanner"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/api/iterator"
)

const kind string = "spanner-sql"

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
	SpannerClient() *spanner.Client
	DatabaseDialect() string
}

// validate compatible sources are still compatible
var _ compatibleSource = &spannerdb.Source{}

var compatibleSources = [...]string{spannerdb.SourceKind}

type Config struct {
	Name               string                `yaml:"name" validate:"required"`
	Kind               string                `yaml:"kind" validate:"required"`
	Source             string                `yaml:"source" validate:"required"`
	Description        string                `yaml:"description" validate:"required"`
	Statement          string                `yaml:"statement" validate:"required"`
	ReadOnly           bool                  `yaml:"readOnly"`
	AuthRequired       []string              `yaml:"authRequired"`
	Parameters         parameters.Parameters `yaml:"parameters"`
	TemplateParameters parameters.Parameters `yaml:"templateParameters"`
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

	allParameters, paramManifest, err := parameters.ProcessParameters(cfg.TemplateParameters, cfg.Parameters)
	if err != nil {
		return nil, err
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		AllParams:   allParameters,
		Client:      s.SpannerClient(),
		dialect:     s.DatabaseDialect(),
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	AllParams   parameters.Parameters `yaml:"allParams"`
	Client      *spanner.Client
	dialect     string
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func getMapParams(params parameters.ParamValues, dialect string) (map[string]interface{}, error) {
	switch strings.ToLower(dialect) {
	case "googlesql":
		return params.AsMap(), nil
	case "postgresql":
		return params.AsMapByOrderedKeys(), nil
	default:
		return nil, fmt.Errorf("invalid dialect %s", dialect)
	}
}

// processRows iterates over the spanner.RowIterator and converts each row to a map[string]any.
func processRows(iter *spanner.RowIterator) ([]any, error) {
	var out []any
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}

		vMap := make(map[string]any)
		cols := row.ColumnNames()
		for i, c := range cols {
			vMap[c] = row.ColumnValue(i)
		}
		out = append(out, vMap)
	}
	return out, nil
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()
	newStatement, err := parameters.ResolveTemplateParams(t.TemplateParameters, t.Statement, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract template params %w", err)
	}

	newParams, err := parameters.GetParams(t.Parameters, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract standard params %w", err)
	}

	for i, p := range t.Parameters {
		name := p.GetName()
		value := newParams[i].Value

		// Spanner only accepts typed slices as input
		// This checks if the param is an array.
		// If yes, convert []any to typed slice (e.g []string, []int)
		switch arrayParam := p.(type) {
		case *parameters.ArrayParameter:
			arrayParamValue, ok := value.([]any)
			if !ok {
				return nil, fmt.Errorf("unable to convert parameter `%s` to []any %w", name, err)
			}
			itemType := arrayParam.GetItems().GetType()
			var err error
			value, err = parameters.ConvertAnySliceToTyped(arrayParamValue, itemType)
			if err != nil {
				return nil, fmt.Errorf("unable to convert parameter `%s` from []any to typed slice: %w", name, err)
			}
		}
		newParams[i] = parameters.ParamValue{Name: name, Value: value}
	}

	mapParams, err := getMapParams(newParams, t.dialect)
	if err != nil {
		return nil, fmt.Errorf("fail to get map params: %w", err)
	}

	var results []any
	var opErr error
	stmt := spanner.Statement{
		SQL:    newStatement,
		Params: mapParams,
	}

	if t.ReadOnly {
		iter := t.Client.Single().Query(ctx, stmt)
		results, opErr = processRows(iter)
	} else {
		_, opErr = t.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			iter := txn.Query(ctx, stmt)
			results, err = processRows(iter)
			if err != nil {
				return err
			}
			return nil
		})
	}

	if opErr != nil {
		return nil, fmt.Errorf("unable to execute client: %w", opErr)
	}

	return results, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.AllParams, data, claims)
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

func (t Tool) RequiresClientAuthorization() bool {
	return false
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
