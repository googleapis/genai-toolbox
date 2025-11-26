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

package alloydbcreateinstance

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	alloydbadmin "github.com/googleapis/genai-toolbox/internal/sources/alloydbadmin"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/api/alloydb/v1"
)

const kind string = "alloydb-create-instance"

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

// Configuration for the create-instance tool.
type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description"`
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
		return nil, fmt.Errorf("source %q not found", cfg.Source)
	}

	s, ok := rawS.(*alloydbadmin.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `alloydb-admin`", kind)
	}

	project := s.DefaultProject
	var projectParam parameters.Parameter
	if project != "" {
		projectParam = parameters.NewStringParameterWithDefault("project", project, "The GCP project ID. This is pre-configured; do not ask for it unless the user explicitly provides a different one.")
	} else {
		projectParam = parameters.NewStringParameter("project", "The GCP project ID.")
	}

	allParameters := parameters.Parameters{
		projectParam,
		parameters.NewStringParameter("location", "The location of the cluster (e.g., 'us-central1')."),
		parameters.NewStringParameter("cluster", "The ID of the cluster to create the instance in."),
		parameters.NewStringParameter("instance", "A unique ID for the new AlloyDB instance."),
		parameters.NewStringParameterWithDefault("instanceType", "PRIMARY", "The type of instance to create. Valid values are: PRIMARY and READ_POOL. Default is PRIMARY"),
		parameters.NewStringParameterWithRequired("displayName", "An optional, user-friendly name for the instance.", false),
		parameters.NewIntParameterWithDefault("nodeCount", 1, "The number of nodes in the read pool. Required only if instanceType is READ_POOL. Default is 1."),
	}
	paramManifest := allParameters.Manifest()

	description := cfg.Description
	if description == "" {
		description = "Creates a new AlloyDB instance (PRIMARY or READ_POOL) within a cluster. This is a long-running operation. This will return operation id to be used by get operations tool. Take all parameters from user in one go."
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, description, cfg.AuthRequired, allParameters, nil)

	return Tool{
		Config:      cfg,
		Source:      s,
		AllParams:   allParameters,
		manifest:    tools.Manifest{Description: description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}, nil
}

// Tool represents the create-instance tool.
type Tool struct {
	Config
	Source    *alloydbadmin.Source
	AllParams parameters.Parameters `yaml:"allParams"`

	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

// Invoke executes the tool's logic.
func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()
	project, ok := paramsMap["project"].(string)
	if !ok || project == "" {
		return nil, fmt.Errorf("invalid or missing 'project' parameter; expected a non-empty string")
	}

	location, ok := paramsMap["location"].(string)
	if !ok || location == "" {
		return nil, fmt.Errorf("invalid or missing 'location' parameter; expected a non-empty string")
	}

	cluster, ok := paramsMap["cluster"].(string)
	if !ok || cluster == "" {
		return nil, fmt.Errorf("invalid or missing 'cluster' parameter; expected a non-empty string")
	}

	instanceID, ok := paramsMap["instance"].(string)
	if !ok || instanceID == "" {
		return nil, fmt.Errorf("invalid or missing 'instance' parameter; expected a non-empty string")
	}

	instanceType, ok := paramsMap["instanceType"].(string)
	if !ok || (instanceType != "READ_POOL" && instanceType != "PRIMARY") {
		return nil, fmt.Errorf("invalid 'instanceType' parameter; expected 'PRIMARY' or 'READ_POOL'")
	}

	service, err := t.Source.GetService(ctx, string(accessToken))
	if err != nil {
		return nil, err
	}

	urlString := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, cluster)

	// Build the request body using the type-safe Instance struct.
	instance := &alloydb.Instance{
		InstanceType: instanceType,
		NetworkConfig: &alloydb.InstanceNetworkConfig{
			EnablePublicIp: true,
		},
		DatabaseFlags: map[string]string{
			"password.enforce_complexity": "on",
		},
	}

	if displayName, ok := paramsMap["displayName"].(string); ok && displayName != "" {
		instance.DisplayName = displayName
	}

	if instanceType == "READ_POOL" {
		nodeCount, ok := paramsMap["nodeCount"].(int)
		if !ok {
			return nil, fmt.Errorf("invalid 'nodeCount' parameter; expected an integer for READ_POOL")
		}
		instance.ReadPoolConfig = &alloydb.ReadPoolConfig{
			NodeCount: int64(nodeCount),
		}
	}

	// The Create API returns a long-running operation.
	resp, err := service.Projects.Locations.Clusters.Instances.Create(urlString, instance).InstanceId(instanceID).Do()
	if err != nil {
		return nil, fmt.Errorf("error creating AlloyDB instance: %w", err)
	}

	return resp, nil
}

// ParseParams parses the parameters for the tool.
func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.AllParams, data, claims)
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

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
