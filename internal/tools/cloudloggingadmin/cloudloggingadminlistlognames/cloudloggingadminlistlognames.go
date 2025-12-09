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
package cloudloggingadminlistlognames

import (
	"context"
	"fmt"

	"cloud.google.com/go/logging/logadmin"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	cla "github.com/googleapis/genai-toolbox/internal/sources/cloudloggingadmin"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/api/iterator"
)

const kind string = "cloud-logging-admin-list-log-names"

const defaultLimit int = 200

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
	LogAdminClient() *logadmin.Client
	LogAdminClientCreator() cla.LogAdminClientCreator
	UseClientAuthorization() bool
}

var _ compatibleSource = &cla.Source{}

var compatibleSources = [...]string{cla.SourceKind}

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
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	limitDescription := fmt.Sprintf("Maximum number of log entries to return (default: %d).", defaultLimit)
	params := parameters.Parameters{
		parameters.NewIntParameterWithRequired("limit", limitDescription, false),
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params)

	t := Tool{
		Config:      cfg,
		source:      s,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
		params:      params,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config

	source      compatibleSource
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
	params      parameters.Parameters
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	var client *logadmin.Client

	if t.source.UseClientAuthorization() {
		tokenString, err := accessToken.ParseBearerToken()
		if err != nil {
			return nil, fmt.Errorf("failed to parse access token: %w", err)
		}
		client, err = t.source.LogAdminClientCreator()(tokenString)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}
	} else {
		client = t.source.LogAdminClient()
		if client == nil {
			return nil, fmt.Errorf("source client is not initialized")
		}
	}

	limit := defaultLimit
	paramsMap := params.AsMap()
	if val, ok := paramsMap["limit"].(int); ok && val > 0 {
		limit = val
	} else if ok && val < 0 {
		return nil, fmt.Errorf("limit must be greater than or equal to 1")
	}

	it := client.Logs(ctx)
	var logNames []string
	for len(logNames) < limit {
		logName, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		logNames = append(logNames, logName)
	}

	return logNames, nil
}

func (t Tool) ParseParams(data map[string]any, claimsMap map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.params, data, claimsMap)
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
	return t.source.UseClientAuthorization()
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
