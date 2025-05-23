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

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/memorystoreredis"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/redis/go-redis/v9"
)

const ToolKind string = "redis"

type compatibleSource interface {
	RedisClient() memorystoreredis.RedisClient
}

// validate compatible sources are still compatible
var _ compatibleSource = &memorystoreredis.Source{}

var compatibleSources = [...]string{memorystoreredis.SourceKind}

type Config struct {
	Name         string           `yaml:"name" validate:"required"`
	Kind         string           `yaml:"kind" validate:"required"`
	Source       string           `yaml:"source" validate:"required"`
	Description  string           `yaml:"description" validate:"required"`
	Commands     [][]string       `yaml:"commands" validate:"required"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return ToolKind
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
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", ToolKind, compatibleSources)
	}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: cfg.Parameters.McpManifest(),
	}

	// finish tool setup
	t := Tool{
		Name:         cfg.Name,
		Kind:         ToolKind,
		Parameters:   cfg.Parameters,
		Commands:     cfg.Commands,
		AuthRequired: cfg.AuthRequired,
		Client:       s.RedisClient(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: cfg.Parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`

	Client      memorystoreredis.RedisClient
	Commands    [][]string
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	cmds, err := tools.ReplaceCommandsParams(t.Commands, t.Parameters, params)
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
		result, err := resp.Result()
		if err != nil {
			return nil, fmt.Errorf("error getting result: %s", err)
		}
		out[i] = result
	}

	return out, nil
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
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}
