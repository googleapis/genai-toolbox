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

package alloydbcreatecluster

import (
	"context"
	"fmt"
	"net/http"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const resourceType string = "alloydb-create-cluster"

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
	CreateCluster(context.Context, string, string, string, string, string, string, string) (any, error)
}

// Configuration for the create-cluster tool.
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
		cfg.Description = "Creates a new AlloyDB cluster. This is a long-running operation, but the API call returns quickly. This will return operation id to be used by get operations tool. Take all parameters from user in one go."
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

// Tool represents the create-cluster tool.
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
	if !ok || project == "" {
		return nil, util.NewAgentError("invalid or missing 'project' parameter; expected a non-empty string", nil)
	}

	location, ok := paramsMap["location"].(string)
	if !ok {
		return nil, util.NewAgentError("invalid 'location' parameter; expected a string", nil)
	}

	clusterID, ok := paramsMap["cluster"].(string)
	if !ok || clusterID == "" {
		return nil, util.NewAgentError("invalid or missing 'cluster' parameter; expected a non-empty string", nil)
	}

	password, ok := paramsMap["password"].(string)
	if !ok || password == "" {
		return nil, util.NewAgentError("invalid or missing 'password' parameter; expected a non-empty string", nil)
	}

	network, ok := paramsMap["network"].(string)
	if !ok {
		return nil, util.NewAgentError("invalid 'network' parameter; expected a string", nil)
	}

	user, ok := paramsMap["user"].(string)
	if !ok {
		return nil, util.NewAgentError("invalid 'user' parameter; expected a string", nil)
	}
	resp, err := source.CreateCluster(ctx, project, location, network, user, password, clusterID, string(accessToken))

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
	projectParam := parameters.NewStringParameter("project", "The GCP project ID.")
	if project != "" {
		projectParam = parameters.NewStringParameter("project", "The GCP project ID. This is pre-configured; do not ask for it unless the user explicitly provides a different one.", parameters.WithStringDefault(project))
	}
	return parameters.Parameters{
		projectParam,
		parameters.NewStringParameter("location", "The location to create the cluster in. The default value is us-central1. If quota is exhausted then use other regions.", parameters.WithStringDefault("us-central1")),
		parameters.NewStringParameter("cluster", "A unique ID for the AlloyDB cluster."),
		parameters.NewStringParameter("password", "A secure password for the initial user."),
		parameters.NewStringParameter("network", "The name of the VPC network to connect the cluster to (e.g., 'default').", parameters.WithStringDefault("default")),
		parameters.NewStringParameter("user", "The name for the initial superuser. Defaults to 'postgres' if not provided.", parameters.WithStringDefault("postgres")),
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
