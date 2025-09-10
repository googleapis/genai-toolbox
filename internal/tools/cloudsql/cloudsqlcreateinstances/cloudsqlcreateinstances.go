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

package cloudsqlcreateinstances

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	httpsource "github.com/googleapis/genai-toolbox/internal/sources/http"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"golang.org/x/oauth2/google"
)

const kind string = "cloud-sql-create-instances"

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
	Description  string   `yaml:"description" validate:"required"`
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

	s, ok := rawS.(*httpsource.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `http`", kind)
	}

	if s.BaseURL != "https://sqladmin.googleapis.com" && !strings.HasPrefix(s.BaseURL, "http://127.0.0.1") {
		return nil, fmt.Errorf("invalid source for %q tool: baseUrl must be `https://sqladmin.googleapis.com/`", kind)
	}

	allParameters := tools.Parameters{
		tools.NewStringParameter("project", "The project ID"),
		tools.NewStringParameter("name", "The name of the instance"),
		tools.NewStringParameter("databaseVersion", "The database version. If not specified, defaults to the latest available version for the engine (e.g., POSTGRES_17, MYSQL_8_4, SQLSERVER_2022_STANDARD)."),
		tools.NewStringParameter("rootPassword", "The root password for the instance"),
		tools.NewStringParameter("editionPreset", "The edition of the instance. Can be `Production` or `Development`. This determines the default machine type and availability. Defaults to `Development`."),
	}
	paramManifest := allParameters.Manifest()

	inputSchema := allParameters.McpManifest()
	inputSchema.Required = []string{"project", "name", "databaseVersion", "editionPreset", "rootPassword"}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: inputSchema,
	}

	return Tool{
		Name:        cfg.Name,
		Kind:        kind,
		BaseURL:     s.BaseURL,
		Client:      s.Client,
		AllParams:   allParameters,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: paramManifest},
		mcpManifest: mcpManifest,
	}, nil
}

// Tool represents the create-instances tool.
type Tool struct {
	Name        string `yaml:"name"`
	Kind        string `yaml:"kind"`
	Description string `yaml:"description"`

	BaseURL   string           `yaml:"baseURL"`
	AllParams tools.Parameters `yaml:"allParams"`

	Client      *http.Client
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

	urlString := fmt.Sprintf("%s/v1/projects/%s/instances", t.BaseURL, project)

	// Set default values
	dbVersion, ok := paramsMap["databaseVersion"].(string)
	if !ok || dbVersion == "" {
		dbVersion = "POSTGRES_17" // Default to Postgres if nothing is specified
	}

	paramsMap["databaseVersion"] = dbVersion
	upperDBVersion := strings.ToUpper(dbVersion)

	settings := make(map[string]any)

	// Determine logical edition for setting defaults
	if ed, ok := paramsMap["editionPreset"].(string); ok {
		logicalEdition := "Development"
		if strings.EqualFold(ed, "Production") {
			logicalEdition = "Production"
		}

		// Set engine-specific defaults
		if strings.HasPrefix(upperDBVersion, "SQLSERVER") {
			if logicalEdition == "Production" {
				settings["availabilityType"] = "REGIONAL"
				settings["edition"] = "ENTERPRISE"
				settings["tier"] = "db-custom-4-26624" // 4vCPU, 16GB RAM
			} else { // Development
				settings["availabilityType"] = "ZONAL"
				settings["edition"] = "ENTERPRISE"
				settings["tier"] = "db-custom-2-8192" // 2vCPU, 8GB RAM
			}
		} else { // Default to Postgres/MySQL style defaults
			if logicalEdition == "Production" {
				settings["availabilityType"] = "REGIONAL"
				settings["edition"] = "ENTERPRISE_PLUS"
				settings["tier"] = "db-perf-optimized-N-8"
			} else { // Development
				settings["availabilityType"] = "ZONAL"
				settings["edition"] = "ENTERPRISE_PLUS"
				settings["tier"] = "db-perf-optimized-N-4"
			}
		}
	}

	paramsMap["settings"] = settings

	body, err := json.Marshal(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlString, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %w", err)
	}

	tokenSource, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/sqlservice.admin")
	if err != nil {
		return nil, fmt.Errorf("error creating token source: %w", err)
	}
	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("error retrieving token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

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

	var data any
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
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
