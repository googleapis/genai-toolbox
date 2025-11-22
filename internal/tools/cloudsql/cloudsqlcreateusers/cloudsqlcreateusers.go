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

package cloudsqlcreateusers

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqladmin"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	sqladmin "google.golang.org/api/sqladmin/v1"
)

const kind string = "cloud-sql-create-users"

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

// Config defines the configuration for the create-user tool.
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
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}
	s, ok := rawS.(*cloudsqladmin.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `cloud-sql-admin`", kind)
	}

	project := s.DefaultProject
	var projectParam parameters.Parameter
	if project != "" {
		projectParam = parameters.NewStringParameterWithDefault("project", project, "The GCP project ID. This is pre-configured; do not ask for it unless the user explicitly provides a different one.")
	} else {
		projectParam = parameters.NewStringParameter("project", "The project ID")
	}

	allParameters := parameters.Parameters{
		projectParam,
		parameters.NewStringParameter("instance", "The ID of the instance where the user will be created."),
		parameters.NewStringParameter("name", "The name for the new user. Must be unique within the instance."),
		parameters.NewStringParameterWithRequired("password", "A secure password for the new user. Not required for IAM users.", false),
		parameters.NewBooleanParameter("iamUser", "Set to true to create a Cloud IAM user."),
	}
	paramManifest := allParameters.Manifest()

	description := cfg.Description
	if description == "" {
		description = "Creates a new user in a Cloud SQL instance. Both built-in and IAM users are supported. IAM users require an email account as the user name. IAM is the more secure and recommended way to manage users. The agent should always ask the user what type of user they want to create. For more information, see https://cloud.google.com/sql/docs/postgres/add-manage-iam-users"
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, description, cfg.AuthRequired, allParameters, nil)

	return Tool{
		Config:      cfg,
		Source:      s,
		AllParams:   allParameters,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}, nil
}

// Tool represents the create-user tool.
type Tool struct {
	Config
	Source      *cloudsqladmin.Source
	AllParams   parameters.Parameters `yaml:"allParams"`
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
	if !ok {
		return nil, fmt.Errorf("missing 'project' parameter")
	}
	instance, ok := paramsMap["instance"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'instance' parameter")
	}
	name, ok := paramsMap["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'name' parameter")
	}

	iamUser, _ := paramsMap["iamUser"].(bool)

	user := sqladmin.User{
		Name: name,
	}

	if iamUser {
		user.Type = "CLOUD_IAM_USER"
	} else {
		user.Type = "BUILT_IN"
		password, ok := paramsMap["password"].(string)
		if !ok || password == "" {
			return nil, fmt.Errorf("missing 'password' parameter for non-IAM user")
		}
		user.Password = password
	}

	service, err := t.Source.GetService(ctx, string(accessToken))
	if err != nil {
		return nil, err
	}

	resp, err := service.Users.Insert(project, instance, &user).Do()
	if err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
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
