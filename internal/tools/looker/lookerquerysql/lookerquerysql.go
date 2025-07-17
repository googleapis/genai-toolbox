// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package lookerquerysql

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	lookersrc "github.com/googleapis/genai-toolbox/internal/sources/looker"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"

	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"

	"github.com/thlib/go-timezone-local/tzlocal"
)

const kind string = "looker_query_sql"

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

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
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
	s, ok := rawS.(*lookersrc.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `looker`", kind)
	}

	modelParameter := tools.NewStringParameter("model", "The model containing the explore.")
	exploreParameter := tools.NewStringParameter("explore", "The explore to be queried.")
	fieldsParameter := tools.NewArrayParameterWithDefault("fields",
		[]any{},
		"The fields to be retrieved.",
		tools.NewStringParameter("field", "A field to be returned in the query"),
	)
	filtersParameter := tools.NewMapParameterWithDefault("filters",
		map[string]any{},
		"The filters for the query",
		"",
	)
	pivotsParameter := tools.NewArrayParameterWithDefault("pivots",
		[]any{},
		"The query pivots (must be included in fields as well).",
		tools.NewStringParameter("pivot_field", "A field to be used as a pivot in the query"),
	)
	sortsParameter := tools.NewArrayParameterWithDefault("sorts",
		[]any{},
		"The sorts like \"field.id desc 0\".",
		tools.NewStringParameter("sort_field", "A field to be used as a sort in the query"),
	)
	limitParameter := tools.NewIntParameterWithDefault("limit", 500, "The row limit.")

	tzParameter := tools.NewStringParameterWithRequired("tz", "The query timezone.", false)

	parameters := tools.Parameters{
		modelParameter,
		exploreParameter,
		fieldsParameter,
		filtersParameter,
		pivotsParameter,
		sortsParameter,
		limitParameter,
		tzParameter,
	}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	// finish tool setup
	return Tool{
		Name:         cfg.Name,
		Kind:         kind,
		Parameters:   parameters,
		AuthRequired: cfg.AuthRequired,
		Client:       s.Client,
		ApiSettings:  s.ApiSettings,
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   parameters.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string `yaml:"name"`
	Kind         string `yaml:"kind"`
	Client       *v4.LookerSDK
	ApiSettings  *rtl.ApiSettings
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`
	manifest     tools.Manifest
	mcpManifest  tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) (any, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}
	logger.DebugContext(ctx, "params = ", params)
	paramsMap := params.AsMap()

	logger.DebugContext(ctx, fmt.Sprintf("fields = %v %T", paramsMap["fields"], paramsMap["fields"]))
	f, err := tools.ConvertAnySliceToTyped(paramsMap["fields"].([]any), "string")
	if err != nil {
		return nil, fmt.Errorf("can't convert fields to array of strings: %s", err)
	}
	fields := f.([]string)
	filters := paramsMap["filters"].(map[string]any)
	// Sometimes filters come as "'field.id'": "expression" so strip extra ''
	for k, v := range filters {
		if len(k) > 0 && k[0] == '\'' && k[len(k)-1] == '\'' {
			delete(filters, k)
			filters[k[1:len(k)-1]] = v
		}
	}
	p, err := tools.ConvertAnySliceToTyped(paramsMap["pivots"].([]any), "string")
	if err != nil {
		return nil, fmt.Errorf("can't convert pivots to array of strings: %s", err)
	}
	pivots := p.([]string)
	s, err := tools.ConvertAnySliceToTyped(paramsMap["sorts"].([]any), "string")
	if err != nil {
		return nil, fmt.Errorf("can't convert sorts to array of strings: %s", err)
	}
	sorts := s.([]string)
	limit := int64(paramsMap["limit"].(int))

	var tz string
	if paramsMap["tz"] != nil {
		tz = paramsMap["tz"].(string)
	} else {
		tzname, err := tzlocal.RuntimeTZ()
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Error getting local timezone: %s", err))
			tzname = "Etc/UTC"
		}
		tz = tzname
	}

	wq := v4.WriteQuery{
		Model:         paramsMap["model"].(string),
		View:          paramsMap["explore"].(string),
		Fields:        &fields,
		Pivots:        &pivots,
		Filters:       &filters,
		Sorts:         &sorts,
		QueryTimezone: &tz,
	}
	req := v4.RequestRunInlineQuery{
		ResultFormat: "sql",
		Limit:        &limit,
		Body:         wq,
	}
	resp, err := t.Client.RunInlineQuery(req, t.ApiSettings)
	if err != nil {
		return nil, fmt.Errorf("error making query_sql request: %s", err)
	}
	logger.DebugContext(ctx, "resp = ", resp)

	return resp, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data, claims)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return true
}
