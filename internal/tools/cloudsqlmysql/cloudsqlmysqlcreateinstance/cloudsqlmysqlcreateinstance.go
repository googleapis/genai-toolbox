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
	"net/http"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	sqladmin "google.golang.org/api/sqladmin/v1"
)

const resourceType string = "cloud-sql-mysql-create-instance"

func init() {
	if !tools.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", resourceType))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{ConfigBase: tools.ConfigBase{Name: name}}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type compatibleSource interface {
	GetDefaultProject() string
	UseClientAuthorization() bool
	CreateInstance(context.Context, string, string, string, string, sqladmin.Settings, string) (any, error)
}

// Config defines the configuration for the create-instances tool.
type Config struct {
	tools.ConfigBase `yaml:",inline"`
	Type             string                 `yaml:"type" validate:"required"`
	Source           string                 `yaml:"source" validate:"required"`
	Annotations      *tools.ToolAnnotations `yaml:"annotations,omitempty"`
}

// validate interface
var _ tools.ToolConfig = Config{}

// ToolConfigType returns the type of the tool.
func (cfg Config) ToolConfigType() string {
	return resourceType
}

// Initialize initializes the tool from the configuration.
func (cfg Config) Initialize(context.Context) (tools.Tool, error) {

	if cfg.Description == "" {
		cfg.Description = "Creates a MySQL instance using `Production` and `Development` presets. For the `Development` template, it chooses a 2 vCPU, 16 GiB RAM, 100 GiB SSD configuration with Non-HA/zonal availability. For the `Production` template, it chooses an 8 vCPU, 64 GiB RAM, 250 GiB SSD configuration with HA/regional availability. The Enterprise Plus edition is used in both cases. The default database version is `MYSQL_8_4`. The agent should ask the user if they want to use a different version."
	}
	params := buildParams("")
	return Tool{
		BaseTool: tools.NewBaseTool(
			cfg,
			tools.GetAnnotationsOrDefault(cfg.Annotations, tools.NewDestructiveAnnotations),
			tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
			params,
		),
	}, nil
}

// Tool represents the create-instances tool.
type Tool struct {
	tools.BaseTool[Config]
}

// Invoke executes the tool's logic.
func (t Tool) Invoke(ctx context.Context, primitiveMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](primitiveMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	paramsMap := params.AsMap()

	project, ok := paramsMap["project"].(string)
	if !ok {
		return nil, util.NewAgentError("missing 'project' parameter", nil)
	}
	name, ok := paramsMap["name"].(string)
	if !ok {
		return nil, util.NewAgentError("missing 'name' parameter", nil)
	}
	dbVersion, ok := paramsMap["databaseVersion"].(string)
	if !ok {
		return nil, util.NewAgentError("missing 'databaseVersion' parameter", nil)
	}
	rootPassword, ok := paramsMap["rootPassword"].(string)
	if !ok {
		return nil, util.NewAgentError("missing 'rootPassword' parameter", nil)
	}
	editionPreset, ok := paramsMap["editionPreset"].(string)
	if !ok {
		return nil, util.NewAgentError("missing 'editionPreset' parameter", nil)
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
		return nil, util.NewAgentError(fmt.Sprintf("invalid 'editionPreset': %q. Must be either 'Production' or 'Development'", editionPreset), nil)
	}

	resp, err := source.CreateInstance(ctx, project, name, dbVersion, rootPassword, settings, string(accessToken))
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}
	return resp, nil
}

// Authorized always returns true: this admin tool intentionally bypasses
// per-request auth-service checks (server-side credentials handle access).
func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return true
}

func (t Tool) RequiresClientAuthorization(primitiveMgr tools.SourceProvider) (bool, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](primitiveMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Cfg
}

// buildParams builds the tool's parameters. A non-empty project means the source has a
// configured default project, which is baked into the project param; otherwise the plain form is used.
func buildParams(project string) parameters.Parameters {
	projectParam := parameters.NewStringParameter("project", "The project ID")
	if project != "" {
		projectParam = parameters.NewStringParameter("project", "The GCP project ID. This is pre-configured; do not ask for it unless the user explicitly provides a different one.", parameters.WithStringDefault(project))
	}
	return parameters.Parameters{
		projectParam,
		parameters.NewStringParameter("name", "The name of the instance"),
		parameters.NewStringParameter("databaseVersion", "The database version for MySQL. If not specified, defaults to the latest available version (e.g., MYSQL_8_4).", parameters.WithStringDefault("MYSQL_8_4")),
		parameters.NewStringParameter("rootPassword", "The root password for the instance"),
		parameters.NewStringParameter("editionPreset", "The edition of the instance. Can be `Production` or `Development`. This determines the default machine type and availability. Defaults to `Development`.", parameters.WithStringDefault("Development")),
	}
}

// resolveParams builds the tool's parameters using the source's configured default GCP project.
func (t Tool) resolveParams(srcs map[string]sources.Source) (parameters.Parameters, error) {
	s, err := tools.GetCompatibleSourceFromMap[compatibleSource](srcs, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return nil, err
	}
	return buildParams(s.GetDefaultProject()), nil
}

// GetParameters returns the tool's parameters, resolved against the source.
func (t Tool) GetParameters(srcs map[string]sources.Source) (parameters.Parameters, error) {
	return t.resolveParams(srcs)
}

// Manifest returns the tool's manifest, resolved against the source.
func (t Tool) Manifest(srcs map[string]sources.Source) (tools.Manifest, error) {
	allParameters, err := t.resolveParams(srcs)
	if err != nil {
		return tools.Manifest{}, err
	}
	return tools.Manifest{Description: t.Cfg.Description, Parameters: allParameters.Manifest(), AuthRequired: t.Cfg.AuthRequired}, nil
}
