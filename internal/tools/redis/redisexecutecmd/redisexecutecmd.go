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
package redisexecutecmd

import (
	"context"
	"fmt"
	"net/http"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	redissrc "github.com/googleapis/genai-toolbox/internal/sources/redis"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	jsoniter "github.com/json-iterator/go"
)

const kind string = "redis-execute-cmd"

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
	RedisClient() redissrc.RedisClient
}

// validate compatible sources are still compatible
var _ compatibleSource = &redissrc.Source{}

var compatibleSources = [...]string{redissrc.SourceType}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"type" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
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

	queryParameter := parameters.NewArrayParameter("cmd", "The command to execute, represented as an array of strings.", parameters.NewStringParameter("token", "An individual word or token in a command, such as a command name, key, or value"))
	params := parameters.Parameters{queryParameter}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, nil)

	t := Tool{
		Name:         cfg.Name,
		Kind:         cfg.Kind,
		Source:       cfg.Source,
		Parameters:   params,
		AuthRequired: cfg.AuthRequired,
		Client:       s.RedisClient(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

var _ tools.Tool = Tool{}

type Tool struct {
	Name         string                `yaml:"name"`
	Kind         string                `yaml:"kind"`
	Source       string                `yaml:"source"`
	AuthRequired []string              `yaml:"authRequired"`
	Parameters   parameters.Parameters `yaml:"parameters"`

	Client      redissrc.RedisClient
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// Authorized implements tools.Tool.
func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

// Invoke implements tools.Tool.
func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	paramsMap := params.AsMap()
	cmds, ok := paramsMap["cmd"].([]any)
	if !ok {
		return nil, util.NewAgentError("unable to cast cmd parameter", fmt.Errorf("unable to get cast %s", paramsMap["cmd"]))
	}

	// Log the query executed for debugging.
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, util.NewClientServerError("error getting logger", http.StatusInternalServerError, err)
	}
	logger.DebugContext(ctx, "executing `%s` tool command: %v", kind, cmds)

	if len(cmds) == 0 {
		return nil, util.NewAgentError("invalid command statement", fmt.Errorf("command array is empty"))
	}

	result, err := t.Client.Do(ctx, cmds...).Result()
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}

	// If result is a map, convert map[any]any to map[string]any
	// Because the Go's built-in json/encoding marshalling doesn't support
	// map[any]any as an input
	var strMap map[string]any
	if m, ok := result.(map[any]any); ok {
		var json = jsoniter.ConfigCompatibleWithStandardLibrary
		mapStr, err := json.Marshal(m)
		if err != nil {
			return nil, util.NewAgentError("error marshalling result", err)
		}
		err = json.Unmarshal(mapStr, &strMap)
		if err != nil {
			return nil, util.NewAgentError("error parsing response", err)
		}
		return strMap, nil
	}
	return result, nil
}

// Manifest implements tools.Tool.
func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

// McpManifest implements tools.Tool.
func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

// EmbedParams implements tools.Tool.
func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.Parameters, paramValues, embeddingModelsMap, nil)
}

// ToConfig implements tools.Tool.
func (t Tool) ToConfig() tools.ToolConfig {
	return Config{
		Name:         t.Name,
		Kind:         t.Kind,
		Source:       t.Source,
		Description:  t.manifest.Description,
		AuthRequired: t.AuthRequired,
	}
}

// RequiresClientAuthorization implements tools.Tool.
func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	return false, nil
}

// GetAuthTokenHeaderName implements tools.Tool.
func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}

// GetParameters implements tools.Tool.
func (t Tool) GetParameters() parameters.Parameters {
	return t.Parameters
}
