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

package setenvvariable

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "update-mcp-settings"

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
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	projectIDParam := tools.NewStringParameter("ALLOYDB_POSTGRES_PROJECT", "The Google Cloud project ID.")
	regionParam := tools.NewStringParameter("ALLOYDB_POSTGRES_REGION", "The region for AlloyDB.")
	clusterParam := tools.NewStringParameter("ALLOYDB_POSTGRES_CLUSTER", "The AlloyDB cluster name.")
	instanceParam := tools.NewStringParameter("ALLOYDB_POSTGRES_INSTANCE", "The AlloyDB instance name.")
	databaseParam := tools.NewStringParameter("ALLOYDB_POSTGRES_DATABASE", "The AlloyDB database name (defaults to 'postgres').")
	userParam := tools.NewStringParameter("ALLOYDB_POSTGRES_USER", "The database username.")
	passwordParam := tools.NewStringParameter("ALLOYDB_POSTGRES_PASSWORD", "The database password.")
	mcpSettingsFile := tools.NewStringParameter("mcpSettingsFile", "The MCP Settings json file which contains information about server to run for the IDE")

	parameters := tools.Parameters{
		projectIDParam,
		regionParam,
		clusterParam,
		instanceParam,
		databaseParam,
		userParam,
		passwordParam,
		mcpSettingsFile,
	}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	t := Tool{
		Name:        cfg.Name,
		Kind:        kind,
		Parameters:  parameters,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name        string
	Kind        string
	Parameters  tools.Parameters
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	paramsMap := params.AsMap()
	mcpSettingsFile, ok := paramsMap["mcpSettingsFile"]
	if !ok {
		return nil, fmt.Errorf("mcpSettingsFile not found in params")
	}

	mcpSettingsFileStr, ok := mcpSettingsFile.(string)
	if !ok {
		return nil, fmt.Errorf("mcpSettingsFile is not a string")
	}

	data, err := os.ReadFile(mcpSettingsFileStr)
	if err != nil {
		return nil, fmt.Errorf("failed to read mcp settings file: %w", err)
	}

	var mcpSettings map[string]interface{}
	if err := json.Unmarshal(data, &mcpSettings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mcp settings file: %w", err)
	}

	mcpServers, ok := mcpSettings["mcpServers"].(map[string]interface{})
	if !ok {
		if servers, found := mcpSettings["servers"].(map[string]interface{}); found {
			mcpServers = servers
		} else {
			mcpServers = make(map[string]interface{})
			mcpSettings["mcpServers"] = mcpServers
		}
	}

	var targetServer map[string]interface{}
	var targetServerName string
	for serverName, server := range mcpServers {
		serverMap, ok := server.(map[string]interface{})
		if !ok {
			continue
		}
		args, ok := serverMap["args"].([]interface{})
		if !ok {
			continue
		}
		for _, arg := range args {
			if argStr, ok := arg.(string); ok && argStr == "alloydb-postgres" {
				targetServer = serverMap
				targetServerName = serverName
				break
			}
		}
		if targetServer != nil {
			break
		}
	}

	if targetServer == nil {
		targetServerName = "alloydb"
		targetServer = make(map[string]interface{})
		targetServer["args"] = []interface{}{"--prebuilt", "alloydb-postgres", "--stdio"}
		mcpServers[targetServerName] = targetServer
	}

	if _, ok := targetServer["command"]; !ok {
		targetServer["command"] = "./PATH/TO/toolbox"
	}

	env, ok := targetServer["env"].(map[string]interface{})
	if !ok {
		env = make(map[string]interface{})
		targetServer["env"] = env
	}

	for key, value := range paramsMap {
		if key != "mcpSettingsFile" {
			env[key] = value
		}
	}

	updatedData, err := json.MarshalIndent(mcpSettings, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal mcp settings file: %w", err)
	}

	if err := os.WriteFile(mcpSettingsFileStr, updatedData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write mcp settings file: %w", err)
	}

	return []any{"Successfully updated MCP settings file"}, nil
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
