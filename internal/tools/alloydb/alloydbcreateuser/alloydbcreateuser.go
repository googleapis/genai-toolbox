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

package alloydbcreateuser

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

const kind string = "alloydb-create-user"

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

// Configuration for the create-user tool.
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
		parameters.NewStringParameter("cluster", "The ID of the cluster where the user will be created."),
		parameters.NewStringParameter("user", "The name for the new user. Must be unique within the cluster."),
		parameters.NewStringParameterWithRequired("password", "A secure password for the new user. Required only for ALLOYDB_BUILT_IN userType.", false),
		parameters.NewArrayParameterWithDefault("databaseRoles", []any{}, "Optional. A list of database roles to grant to the new user (e.g., ['pg_read_all_data']).", parameters.NewStringParameter("role", "A single database role to grant to the user (e.g., 'pg_read_all_data').")),
		parameters.NewStringParameter("userType", "The type of user to create. Valid values are: ALLOYDB_BUILT_IN and ALLOYDB_IAM_USER. ALLOYDB_IAM_USER is recommended."),
	}
	paramManifest := allParameters.Manifest()

	description := cfg.Description
	if description == "" {
		description = "Creates a new AlloyDB user within a cluster. Takes the new user's name and a secure password. Optionally, a list of database roles can be assigned. Always ask the user for the type of user to create. ALLOYDB_IAM_USER is recommended."
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

// Tool represents the create-user tool.
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
		return nil, fmt.Errorf("invalid or missing'location' parameter; expected a non-empty string")
	}

	cluster, ok := paramsMap["cluster"].(string)
	if !ok || cluster == "" {
		return nil, fmt.Errorf("invalid or missing 'cluster' parameter; expected a non-empty string")
	}

	userID, ok := paramsMap["user"].(string)
	if !ok || userID == "" {
		return nil, fmt.Errorf("invalid or missing 'user' parameter; expected a non-empty string")
	}

	userType, ok := paramsMap["userType"].(string)
	if !ok || (userType != "ALLOYDB_BUILT_IN" && userType != "ALLOYDB_IAM_USER") {
		return nil, fmt.Errorf("invalid or missing 'userType' parameter; expected 'ALLOYDB_BUILT_IN' or 'ALLOYDB_IAM_USER'")
	}

	service, err := t.Source.GetService(ctx, string(accessToken))
	if err != nil {
		return nil, err
	}

	urlString := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, cluster)

	// Build the request body using the type-safe User struct.
	user := &alloydb.User{
		UserType: userType,
	}

	if userType == "ALLOYDB_BUILT_IN" {
		password, ok := paramsMap["password"].(string)
		if !ok || password == "" {
			return nil, fmt.Errorf("password is required when userType is ALLOYDB_BUILT_IN")
		}
		user.Password = password
	}

	if dbRolesRaw, ok := paramsMap["databaseRoles"].([]any); ok && len(dbRolesRaw) > 0 {
		var roles []string
		for _, r := range dbRolesRaw {
			if role, ok := r.(string); ok {
				roles = append(roles, role)
			}
		}
		if len(roles) > 0 {
			user.DatabaseRoles = roles
		}
	}

	// The Create API returns a long-running operation.
	resp, err := service.Projects.Locations.Clusters.Users.Create(urlString, user).UserId(userID).Do()
	if err != nil {
		return nil, fmt.Errorf("error creating AlloyDB user: %w", err)
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
