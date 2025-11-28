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
package lookerrundashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	lookersrc "github.com/googleapis/genai-toolbox/internal/sources/looker"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/looker/lookercommon"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"

	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const kind string = "looker-run-dashboard"

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
	Name         string                 `yaml:"name" validate:"required"`
	Kind         string                 `yaml:"kind" validate:"required"`
	Source       string                 `yaml:"source" validate:"required"`
	Description  string                 `yaml:"description" validate:"required"`
	AuthRequired []string               `yaml:"authRequired"`
	Annotations  *tools.ToolAnnotations `yaml:"annotations,omitempty"`
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

	dashboardidParameter := parameters.NewStringParameter("dashboard_id", "The id of the dashboard to run.")

	params := parameters.Parameters{
		dashboardidParameter,
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, cfg.Annotations)

	// finish tool setup
	return Tool{
		Config:              cfg,
		Parameters:          params,
		UseClientOAuth:      s.UseClientAuthorization(),
		AuthTokenHeaderName: s.GetAuthTokenHeaderName(),
		Client:              s.Client,
		ApiSettings:         s.ApiSettings,
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   params.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	UseClientOAuth      bool
	AuthTokenHeaderName string
	Client              *v4.LookerSDK
	ApiSettings         *rtl.ApiSettings
	Parameters          parameters.Parameters `yaml:"parameters"`
	manifest            tools.Manifest
	mcpManifest         tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}
	logger.DebugContext(ctx, "params = ", params)
	paramsMap := params.AsMap()

	dashboard_id := paramsMap["dashboard_id"].(string)

	sdk, err := lookercommon.GetLookerSDK(t.UseClientOAuth, t.ApiSettings, t.Client, accessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting sdk: %w", err)
	}
	dashboard, err := sdk.Dashboard(dashboard_id, "", t.ApiSettings)
	if err != nil {
		return nil, fmt.Errorf("error getting dashboard: %w", err)
	}

	data := make(map[string]any)
	data["tiles"] = make([]any, 0)
	if dashboard.Title != nil {
		data["title"] = *dashboard.Title
	}
	if dashboard.Description != nil {
		data["description"] = *dashboard.Description
	}

	channels := make([]<-chan map[string]any, len(*dashboard.DashboardElements))
	for i, element := range *dashboard.DashboardElements {
		channels[i] = tileQueryWorker(ctx, sdk, t.ApiSettings, i, element)
	}

	for resp := range merge(channels...) {
		data["tiles"] = append(data["tiles"].([]any), resp)
	}

	logger.DebugContext(ctx, "data = ", data)

	return data, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Parameters, data, claims)
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
	return t.UseClientOAuth
}

func tileQueryWorker(ctx context.Context, sdk *v4.LookerSDK, options *rtl.ApiSettings, index int, element v4.DashboardElement) <-chan map[string]any {
	out := make(chan map[string]any)

	go func() {
		defer close(out)

		data := make(map[string]any)
		data["index"] = index
		if element.Title != nil {
			data["title"] = *element.Title
		}
		if element.TitleText != nil {
			data["title_text"] = *element.TitleText
		}
		if element.SubtitleText != nil {
			data["subtitle_text"] = *element.SubtitleText
		}
		if element.BodyText != nil {
			data["body_text"] = *element.BodyText
		}

		var q v4.Query
		if element.Query != nil {
			data["element_type"] = "query"
			q = *element.Query
		} else if element.Look != nil {
			data["element_type"] = "look"
			q = *element.Look.Query
		} else {
			// Just a text element
			data["element_type"] = "text"
			out <- data
			return
		}

		wq := v4.WriteQuery{
			Model:         q.Model,
			View:          q.View,
			Fields:        q.Fields,
			Pivots:        q.Pivots,
			Filters:       q.Filters,
			Sorts:         q.Sorts,
			QueryTimezone: q.QueryTimezone,
			Limit:         q.Limit,
		}
		query_result, err := lookercommon.RunInlineQuery(ctx, sdk, &wq, "json", options)
		if err != nil {
			data["query_status"] = "error running query"
			out <- data
			return
		}
		var resp []any
		e := json.Unmarshal([]byte(query_result), &resp)
		if e != nil {
			data["query_status"] = "error parsing query result"
			out <- data
			return
		}
		data["query_status"] = "success"
		data["query_result"] = resp
		out <- data
	}()
	return out
}

func merge(channels ...<-chan map[string]any) <-chan map[string]any {
	var wg sync.WaitGroup
	out := make(chan map[string]any)

	output := func(c <-chan map[string]any) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(channels))
	for _, c := range channels {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func (t Tool) GetAuthTokenHeaderName() string {
	return t.AuthTokenHeaderName
}
