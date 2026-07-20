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

package cloudsqlcloneinstance

import (
	"context"
	"fmt"
	"net/http"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	sqladmin "google.golang.org/api/sqladmin/v1"
)

const resourceType string = "cloud-sql-clone-instance"

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
	GetService(context.Context, string) (*sqladmin.Service, error)
	UseClientAuthorization() bool
	CloneInstance(context.Context, string, string, string, string, string, string, string) (any, error)
}

// Config defines the configuration for the clone-instance tool.
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
		cfg.Description = "Clone an existing Cloud SQL instance into a new instance. The clone can be a direct copy of the source instance, or a point-in-time-recovery (PITR) clone from a specific timestamp. The call returns a Cloud SQL Operation object. Call wait_for_operation tool after this, make sure to use multiplier as 4 to poll the opertation status till it is marked DONE."
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

// Tool represents the clone-instance tool.
type Tool struct {
	tools.BaseTool[Config]
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Cfg
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
		return nil, util.NewAgentError(fmt.Sprintf("error casting 'project' parameter: %v", paramsMap["project"]), nil)
	}
	sourceInstanceName, ok := paramsMap["sourceInstanceName"].(string)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("error casting 'sourceInstanceName' parameter: %v", paramsMap["sourceInstanceName"]), nil)
	}
	destinationInstanceName, ok := paramsMap["destinationInstanceName"].(string)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("error casting 'destinationInstanceName' parameter: %v", paramsMap["destinationInstanceName"]), nil)
	}

	pointInTime, _ := paramsMap["pointInTime"].(string)
	preferredZone, _ := paramsMap["preferredZone"].(string)
	preferredSecondaryZone, _ := paramsMap["preferredSecondaryZone"].(string)

	resp, err := source.CloneInstance(ctx, project, sourceInstanceName, destinationInstanceName, pointInTime, preferredZone, preferredSecondaryZone, string(accessToken))
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}
	return resp, nil
}

// Authorized checks if the tool is authorized.
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

// buildParams builds the tool's parameters. A non-empty project means the source has a
// configured default project, which is baked into the project param; otherwise the plain form is used.
func buildParams(project string) parameters.Parameters {
	projectParam := parameters.NewStringParameter("project", "The project ID")
	if project != "" {
		projectParam = parameters.NewStringParameter("project", "The GCP project ID. This is pre-configured; do not ask for it unless the user explicitly provides a different one.", parameters.WithStringDefault(project))
	}
	return parameters.Parameters{
		projectParam,
		parameters.NewStringParameter("sourceInstanceName", "The name of the instance to be cloned."),
		parameters.NewStringParameter("destinationInstanceName", "The name of the new instance that will be created by cloning the source instance."),
		parameters.NewStringParameter("pointInTime", "The timestamp in RFC 3339 format to which the source instance should be cloned.", parameters.WithStringRequired(false)),
		parameters.NewStringParameter("preferredZone", "The preferred zone for the new instance.", parameters.WithStringRequired(false)),
		parameters.NewStringParameter("preferredSecondaryZone", "The preferred secondary zone for the new instance.", parameters.WithStringRequired(false)),
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
