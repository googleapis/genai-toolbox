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
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	redissrc "github.com/googleapis/genai-toolbox/internal/sources/redis"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
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

var compatibleSources = [...]string{redissrc.SourceKind}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

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

	queryParameter := tools.NewStringParameter("cmd", "The command to execute.")
	parameters := tools.Parameters{queryParameter}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	t := Tool{
		Name:         cfg.Name,
		Kind:         cfg.Kind,
		Parameters:   parameters,
		AuthRequired: cfg.AuthRequired,
		Client:       s.RedisClient(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

var _ tools.Tool = Tool{}

type Tool struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`

	Client      redissrc.RedisClient
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// Authorized implements tools.Tool.
func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

// Invoke implements tools.Tool.
func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) (any, error) {
	paramsMap := params.AsMap()
	query, ok := paramsMap["cmd"].(string)
	if !ok {
		return nil, fmt.Errorf("unable to get cast %s", paramsMap["cmd"])
	}

	// Log the query executed for debugging.
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting logger: %s", err)
	}
	logger.DebugContext(ctx, "executing `%s` tool command: %s", kind, query)
	cmds := toAnySlice(query)
	if len(cmds) == 0 {
		return nil, fmt.Errorf("invalid command statement")
	}

	result, err := t.Client.Do(ctx, cmds...).Result()
	if err != nil {
		return nil, fmt.Errorf("error from executing command: %v", err)
	}

	var out any
	// If result is a map, convert map[any]any to map[string]any
	// Because the Go's built-in json/encoding marshalling doesn't support
	// map[any]any as an input
	var strMap map[string]any
	if m, ok := result.(map[any]any); ok {
		var json = jsoniter.ConfigCompatibleWithStandardLibrary
		mapStr, err := json.Marshal(m)
		if err != nil {
			return nil, fmt.Errorf("error marshalling result: %s", err)
		}
		err = json.Unmarshal(mapStr, &strMap)
		if err != nil {
			return nil, fmt.Errorf("error parsing response: %v", err)
		}
		return strMap, nil
	}
	return out, nil
}

// Manifest implements tools.Tool.
func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

// McpManifest implements tools.Tool.
func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

// ParseParams implements tools.Tool.
func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data, claims)
}

func toAnySlice(query string) []any {
	strs := strings.Fields(query)
	var result []any
	for _, v := range strs {
		result = append(result, v)
	}
	return result
}
