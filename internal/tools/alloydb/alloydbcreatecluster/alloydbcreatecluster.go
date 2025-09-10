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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	httpsrc "github.com/googleapis/genai-toolbox/internal/sources/http"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"golang.org/x/oauth2/google"
)

const kind string = "alloydb-create-cluster"

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

// Configuration for the create-cluster tool.
type Config struct {
	Name         string            `yaml:"name" validate:"required"`
	Kind         string            `yaml:"kind" validate:"required"`
	Source       string            `yaml:"source" validate:"required"`
	Description  string            `yaml:"description" validate:"required"`
	AuthRequired []string          `yaml:"authRequired"`
	BaseURL      string            `yaml:"baseURL"`
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

	s, ok := rawS.(*httpsrc.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `http`", kind)
	}

	allParameters := tools.Parameters{
		tools.NewStringParameter("projectId", "The GCP project ID."),
		tools.NewStringParameter("locationId", "The location to create the cluster in."),
		tools.NewStringParameter("clusterId", "A unique ID for the AlloyDB cluster."),
		tools.NewStringParameter("password", "A secure password for the initial user."),
		tools.NewStringParameterWithDefault("network", "default", "The name of the VPC network to connect the cluster to (e.g., 'default')."),
		tools.NewStringParameterWithDefault("user", "postgres", "The name for the initial superuser. Defaults to 'postgres' if not provided."),
	}
	paramManifest := allParameters.Manifest()

	inputSchema := allParameters.McpManifest()
	inputSchema.Required = []string{"projectId", "locationId", "clusterId", "password"}
	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: inputSchema,
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://alloydb.googleapis.com"
	}

	return Tool{
		Name:         cfg.Name,
		Kind:         kind,
		BaseURL:      baseURL,
		AuthRequired: cfg.AuthRequired,
		Client:       s.Client,
		AllParams:    allParameters,
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}, nil
}

// Tool represents the create-cluster tool.
type Tool struct {
	Name         string   `yaml:"name"`
	Kind         string   `yaml:"kind"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`

	BaseURL   string           `yaml:"baseURL"`
	AllParams tools.Parameters `yaml:"allParams"`

	Client      *http.Client
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// Invoke executes the tool's logic.
func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()
	projectId, ok := paramsMap["projectId"].(string)
    if !ok || projectId == "" {
        return nil, fmt.Errorf("invalid or missing 'projectId' parameter; expected a non-empty string")
    }

	locationId, ok := paramsMap["locationId"].(string)
    if !ok {
        return nil, fmt.Errorf("iinvalid or missing 'locationId' parameter; expected a non-empty string")
    }

    clusterId, ok := paramsMap["clusterId"].(string)
    if !ok || clusterId == "" {
        return nil, fmt.Errorf("invalid or missing 'clusterId' parameter; expected a non-empty string")
    }

    password, ok := paramsMap["password"].(string)
    if !ok || password == "" {
        return nil, fmt.Errorf("invalid or missing 'password' parameter; expected a non-empty string")
    }

    network, ok := paramsMap["network"].(string)
    if !ok {
        return nil, fmt.Errorf("invalid 'network' parameter; expected a string")
    }

    user, ok := paramsMap["user"].(string)
    if !ok {
        return nil, fmt.Errorf("invalid 'user' parameter; expected a string")
    }

	urlString := fmt.Sprintf("%s/v1/projects/%s/locations/%s/clusters?clusterId=%s", t.BaseURL, projectId, locationId, clusterId)

	requestBodyMap := map[string]any{
		"networkConfig": map[string]string{
			"network": fmt.Sprintf("projects/%s/global/networks/%s", projectId, network),
		},
		"initialUser": map[string]string{
			"password": password,
			"user":     user,
		},
	}

	bodyBytes, err := json.Marshal(requestBodyMap)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlString, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	tokenSource, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, response body: %s", resp.StatusCode, string(respBody))
	}

	var result any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON response: %w", err)
	}

	return result, nil
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
