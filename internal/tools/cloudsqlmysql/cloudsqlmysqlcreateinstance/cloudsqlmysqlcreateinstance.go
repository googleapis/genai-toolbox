// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloudsqlmysqlcreateinstance

import (
	"context"
	"fmt"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqladmin"
	"github.com/googleapis/genai-toolbox/internal/tools"
	sqladmin "google.golang.org/api/sqladmin/v1"
)

const kind string = "cloud-sql-mysql-create-instance"

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

// Config defines the configuration for the create-instances tool.
type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Description  string   `yaml:"description"`
	Source       string   `yaml:"source" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

// ToolConfigKind returns the kind of the tool.
func (cfg Config) ToolConfigKind() string {
	return kind
}

// Initialize initializes the tool from the configuration.
func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}
	s, ok := rawS.(*cloudsqladmin.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `cloud-sql-admin`", kind)
	}

	allParameters := tools.Parameters{
		tools.NewStringParameter("project", "The project ID"),
		tools.NewStringParameter("name", "The name of the instance"),
		tools.NewStringParameterWithDefault("databaseVersion", "MYSQL_8_4", "The database version for MySQL. If not specified, defaults to the latest available version (e.g., MYSQL_8_4)."),
		tools.NewStringParameter("rootPassword", "The root password for the instance"),
		tools.NewStringParameterWithDefault("editionPreset", "Development", "The edition of the instance. Can be `Production` or `Development`. This determines the default machine type and availability. Defaults to `Development`."),
	}
	paramManifest := allParameters.Manifest()

	inputSchema := allParameters.McpManifest()
	inputSchema.Required = []string{"project", "name", "editionPreset", "rootPassword"}

	description := cfg.Description
	if description == "" {
		description = "Creates a MySQL instance using `Production` and `Development` presets. For the `Development` template, it chooses a 2 vCPU, 16 GiB RAM, 100 GiB SSD configuration with Non-HA/zonal availability. For the `Production` template, it chooses an 8 vCPU, 64 GiB RAM, 250 GiB SSD configuration with HA/regional availability. The Enterprise Plus edition is used in both cases. The default database version is `MYSQL_8_4`. The agent should ask the user if they want to use a different version."
	}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: description,
		InputSchema: inputSchema,
	}

	return Tool{
		Name:         cfg.Name,
		Kind:         kind,
		AuthRequired: cfg.AuthRequired,
		Source:       s,
		AllParams:    allParameters,
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}, nil
}

// Tool represents the create-instances tool.
type Tool struct {
	Name         string   `yaml:"name"`
	Kind         string   `yaml:"kind"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`

	Source      *cloudsqladmin.Source
	AllParams   tools.Parameters `yaml:"allParams"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// Invoke executes the tool's logic.
func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	project, ok := paramsMap["project"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'project' parameter")
	}
	name, ok := paramsMap["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'name' parameter")
	}
	dbVersion, ok := paramsMap["databaseVersion"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'databaseVersion' parameter")
	}
	rootPassword, ok := paramsMap["rootPassword"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'rootPassword' parameter")
	}
	editionPreset, ok := paramsMap["editionPreset"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'editionPreset' parameter")
	}

	settings := sqladmin.Settings{}
	switch strings.ToLower(editionPreset) {
	case "production":
		settings.AvailabilityType = "REGIONAL"
		settings.Edition = "ENTERPRISE_PLUS"
		settings.Tier = "db-perf-optimized-N-8"
		settings.DataDiskSizeGb = 250
		settings.DataDiskType = "PD_SSD"
	case "development":
		settings.AvailabilityType = "ZONAL"
		settings.Edition = "ENTERPRISE_PLUS"
		settings.Tier = "db-perf-optimized-N-2"
		settings.DataDiskSizeGb = 100
		settings.DataDiskType = "PD_SSD"
	default:
		return nil, fmt.Errorf("invalid 'editionPreset': %q. Must be either 'Production' or 'Development'", editionPreset)
	}

	instance := sqladmin.DatabaseInstance{
		Name:            name,
		DatabaseVersion: dbVersion,
		RootPassword:    rootPassword,
		Settings:        &settings,
		Project:         project,
	}

	service, err := t.Source.GetService(ctx, string(accessToken))
	if err != nil {
		return nil, err
	}

	resp, err := service.Instances.Insert(project, &instance).Do()
	if err != nil {
		return nil, fmt.Errorf("error creating instance: %w", err)
	}

	return resp, nil
}

// ParseParams parses the parameters for the tool.
func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.AllParams, data, claims)
}

// Manifest returns the tool's manifest.
func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

// McpManifest returns the tool's MCP manifest.
func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

// Authorized checks if the tool is authorized.
func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return true
}

func (t Tool) RequiresClientAuthorization() bool {
	return t.Source.UseClientAuthorization()
}
