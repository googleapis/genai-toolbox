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
package redis

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	redissrc "github.com/googleapis/genai-toolbox/internal/sources/redis"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	jsoniter "github.com/json-iterator/go"
	"github.com/redis/go-redis/v9"
)

const kind string = "redis"

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
	Name         string                `yaml:"name" validate:"required"`
	Kind         string                `yaml:"kind" validate:"required"`
	Source       string                `yaml:"source" validate:"required"`
	Description  string                `yaml:"description" validate:"required"`
	Commands     [][]string            `yaml:"commands" validate:"required"`
	AuthRequired []string              `yaml:"authRequired"`
	Parameters   parameters.Parameters `yaml:"parameters"`
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

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, cfg.Parameters, nil)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		Client:      s.RedisClient(),
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: cfg.Parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config

	Client      redissrc.RedisClient
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	cmds, err := replaceCommandsParams(t.Commands, t.Parameters, params)
	if err != nil {
		return nil, fmt.Errorf("error replacing commands' parameters: %s", err)
	}

	// Execute commands
	responses := make([]*redis.Cmd, len(cmds))
	for i, cmd := range cmds {
		responses[i] = t.Client.Do(ctx, cmd...)
	}
	// Parse responses
	out := make([]any, len(t.Commands))
	for i, resp := range responses {
		if err := resp.Err(); err != nil {
			// Add error from each command to `errSum`
			errString := fmt.Sprintf("error from executing command at index %d: %s", i, err)
			out[i] = errString
			continue
		}
		val, err := resp.Result()
		if err != nil {
			return nil, fmt.Errorf("error getting result: %s", err)
		}
		// If result is a map, convert map[any]any to map[string]any
		// Because the Go's built-in json/encoding marshalling doesn't support
		// map[any]any as an input
		var strMap map[string]any
		var json = jsoniter.ConfigCompatibleWithStandardLibrary
		mapStr, err := json.Marshal(val)
		if err != nil {
			return nil, fmt.Errorf("error marshalling result: %s", err)
		}
		err = json.Unmarshal(mapStr, &strMap)
		if err != nil {
			// result is not a map
			out[i] = val
			continue
		}
		out[i] = strMap
	}

	return out, nil
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
	return false
}

// replaceCommandsParams is a helper function to replace parameters in the commands

func replaceCommandsParams(commands [][]string, params parameters.Parameters, paramValues parameters.ParamValues) ([][]any, error) {
	paramMap := paramValues.AsMapWithDollarPrefix()
	typeMap := make(map[string]string, len(params))
	for _, p := range params {
		placeholder := "$" + p.GetName()
		typeMap[placeholder] = p.GetType()
	}
	newCommands := make([][]any, len(commands))
	for i, cmd := range commands {
		newCmd := make([]any, 0)
		for _, part := range cmd {
			v, ok := paramMap[part]
			if !ok {
				// Command part is not a Parameter placeholder
				newCmd = append(newCmd, part)
				continue
			}
			if typeMap[part] == "array" {
				for _, item := range v.([]any) {
					// Nested arrays will only be expanded once
					// e.g., [A, [B, C]]  --> ["A", "[B C]"]
					newCmd = append(newCmd, fmt.Sprintf("%s", item))
				}
				continue
			}
			newCmd = append(newCmd, fmt.Sprintf("%s", v))
		}
		newCommands[i] = newCmd
	}
	return newCommands, nil
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
