// Copyright 2026 Google LLC
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

package spannercreateinstance

import (
	"context"
	"fmt"

	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind string = "spanner-create-instance"

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
	GetDefaultProject() string
	GetClient(context.Context, string) (*instance.InstanceAdminClient, error)
	UseClientAuthorization() bool
}

// Config defines the configuration for the create-instance tool.
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
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source %q not compatible", kind, cfg.Source)
	}

	project := s.GetDefaultProject()
	var projectParam parameters.Parameter
	if project != "" {
		projectParam = parameters.NewStringParameterWithDefault("project", project, "The GCP project ID.")
	} else {
		projectParam = parameters.NewStringParameter("project", "The project ID")
	}

	allParameters := parameters.Parameters{
		projectParam,
		parameters.NewStringParameter("instanceId", "The ID of the instance"),
		parameters.NewStringParameter("displayName", "The display name of the instance"),
		parameters.NewStringParameter("config", "The instance configuration (e.g., regional-us-central1)"),
		parameters.NewIntParameter("nodeCount", "The number of nodes, mutually exclusive with processingUnits (one must be 0)"),
		parameters.NewIntParameter("processingUnits", "The number of processing units, mutually exclusive with nodeCount (one must be 0)"),
		parameters.NewStringParameter("edition", "The edition of the instance (STANDARD, ENTERPRISE, ENTERPRISE_PLUS)"),
	}

	paramManifest := allParameters.Manifest()

	description := cfg.Description
	if description == "" {
		description = "Creates a Spanner instance."
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, description, cfg.AuthRequired, allParameters, nil)

	return Tool{
		Config:      cfg,
		AllParams:   allParameters,
		manifest:    tools.Manifest{Description: description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}, nil
}

// Tool represents the create-instance tool.
type Tool struct {
	Config
	AllParams   parameters.Parameters
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

// Invoke executes the tool's logic.
func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	project, _ := paramsMap["project"].(string)
	instanceId, _ := paramsMap["instanceId"].(string)
	displayName, _ := paramsMap["displayName"].(string)
	config, _ := paramsMap["config"].(string)
	nodeCount, _ := paramsMap["nodeCount"].(int)
	processingUnits, _ := paramsMap["processingUnits"].(int)
	editionStr, _ := paramsMap["edition"].(string)

	if (nodeCount > 0 && processingUnits > 0) || (nodeCount == 0 && processingUnits == 0) {
		return nil, fmt.Errorf("one of nodeCount or processingUnits must be positive, and the other must be 0")
	}

	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Kind)
	if err != nil {
		return nil, err
	}

	client, err := source.GetClient(ctx, string(accessToken))
	if err != nil {
		return nil, err
	}
	if source.UseClientAuthorization() {
		defer client.Close()
	}

	parent := fmt.Sprintf("projects/%s", project)
	instanceConfig := fmt.Sprintf("projects/%s/instanceConfigs/%s", project, config)

	var edition instancepb.Instance_Edition
	switch editionStr {
	case "STANDARD":
		edition = instancepb.Instance_STANDARD
	case "ENTERPRISE":
		edition = instancepb.Instance_ENTERPRISE
	case "ENTERPRISE_PLUS":
		edition = instancepb.Instance_ENTERPRISE_PLUS
	default:
		edition = instancepb.Instance_EDITION_UNSPECIFIED
	}

	// Construct the instance object
	instance := &instancepb.Instance{
		Config:          instanceConfig,
		DisplayName:     displayName,
		Edition:         edition,
		NodeCount:       int32(nodeCount),
		ProcessingUnits: int32(processingUnits),
	}

	req := &instancepb.CreateInstanceRequest{
		Parent:     parent,
		InstanceId: instanceId,
		Instance:   instance,
	}

	op, err := client.CreateInstance(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	// Wait for the operation to complete
	resp, err := op.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for create instance operation: %w", err)
	}

	return resp, nil
}

// ParseParams parses the parameters for the tool.
func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.AllParams, data, claims)
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.AllParams, paramValues, embeddingModelsMap, nil)
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

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Kind)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}
