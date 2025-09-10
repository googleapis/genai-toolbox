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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	httpsrc "github.com/googleapis/genai-toolbox/internal/sources/http"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"golang.org/x/oauth2/google"
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
	Description  string   `yaml:"description" validate:"required"`
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
	s, ok := rawS.(*httpsrc.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `http`", kind)
	}

	if s.BaseURL != "https://sqladmin.googleapis.com" && !strings.HasPrefix(s.BaseURL, "http://127.0.0.1") {
		return nil, fmt.Errorf("invalid source for %q tool: baseUrl must be `https://sqladmin.googleapis.com`", kind)
	}

	allParameters := tools.Parameters{
		tools.NewStringParameter("project", "The project ID"),
		tools.NewStringParameter("instance", "The ID of the instance where the user will be created."),
		tools.NewStringParameter("name", "The name for the new user. Must be unique within the instance."),
		tools.NewStringParameterWithRequired("password", "A secure password for the new user. Not required for IAM users.", false),
		tools.NewBooleanParameterWithDefault("iamUser", false, "Set to true to create a Cloud IAM user."),
	}
	paramManifest := allParameters.Manifest()

	inputSchema := allParameters.McpManifest()

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: inputSchema,
	}

	return Tool{
		Name:         cfg.Name,
		Kind:         kind,
		AuthRequired: cfg.AuthRequired,
		Client:       s.Client,
		AllParams:    allParameters,
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
		BaseURL:      s.BaseURL,
	}, nil
}

// Tool represents the create-user tool.
type Tool struct {
	Name         string   `yaml:"name"`
	Kind         string   `yaml:"kind"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`
	BaseURL      string

	AllParams   tools.Parameters `yaml:"allParams"`
	Client      *http.Client
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// userCreateRequest is the request body for creating a user.
type userCreateRequest struct {
	Name     string `json:"name"`
	Password string `json:"password,omitempty"`
	Type     string `json:"type,omitempty"`
}

// Invoke executes the tool's logic.
func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
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

	reqBody := userCreateRequest{
		Name: name,
	}

	if iamUser {
		reqBody.Type = "CLOUD_IAM_USER"
	} else {
		reqBody.Type = "BUILT_IN"
		password, ok := paramsMap["password"].(string)
		if !ok || password == "" {
			return nil, fmt.Errorf("missing 'password' parameter for non-IAM user")
		}
		reqBody.Password = password
	}

	urlString := fmt.Sprintf("%s/v1/projects/%s/instances/%s/users", t.BaseURL, project, instance)

	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, urlString, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	tokenSource, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/sqlservice.admin")
	if err != nil {
		return nil, fmt.Errorf("error creating token source: %w", err)
	}
	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("error retrieving token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, response body: %s", resp.StatusCode, string(body))
	}

	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("error unmarshaling response body: %w", err)
	}

	return data, nil
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
	return false
}
