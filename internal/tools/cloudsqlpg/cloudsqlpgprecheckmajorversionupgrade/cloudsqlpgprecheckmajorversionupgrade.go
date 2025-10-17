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

package cloudsqlpgprecheckmajorversionupgrade

import (
	"context"
	"fmt"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqladmin"
	"github.com/googleapis/genai-toolbox/internal/tools"
	sqladmin "google.golang.org/api/sqladmin/v1"
	"time"
)

const kind string = "postgres-upgrade-precheck"

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

// Config defines the configuration for the precheck-upgrade tool.
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
	s, ok := rawS.(*cloudsqladmin.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `cloud-sql-admin`", kind)
	}

	allParameters := tools.Parameters{
		tools.NewStringParameter("project", "The project ID"),
		tools.NewStringParameter("instance", "The name of the instance to check"),
		tools.NewStringParameterWithDefault("targetDatabaseVersion", "POSTGRES_18", "The target PostgreSQL version for the upgrade (e.g., POSTGRES_18). If not specified, defaults to the PostgreSQL 18."),
	}
	paramManifest := allParameters.Manifest()

	description := cfg.Description
	if description == "" {
		description = "Performs a pre-check for a Cloud SQL PostgreSQL instance major version upgrade to identify potential issues before attempting the actual upgrade."
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, description, cfg.AuthRequired, allParameters)

	return Tool{
		Name:         cfg.Name,
		Kind:         kind,
		AuthRequired: cfg.AuthRequired,
		Source:       s,
		AllParams:    allParameters,
		manifest:     tools.Manifest{Description: description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}, nil
}

// Tool represents the precheck-upgrade tool.
type Tool struct {
	Name         string   `yaml:"name"`
	Kind         string   `yaml:"kind"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`

	Source      *cloudsqladmin.Source
	AllParams   tools.Parameters `yaml:"allParams"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// PreCheckResultItem holds the details of a single check result.
type PreCheckResultItem struct {
	Message         string   `json:"message"`
	MessageType     string   `json:"messageType"` // INFO, WARNING, ERROR
	ActionsRequired []string `json:"actionsRequired"`
}

// PreCheckAPIResponse holds the array of pre-check results.
type PreCheckAPIResponse struct {
	Items []PreCheckResultItem `json:"preCheckResponse"`
}

// Helper function to convert from []*sqladmin.PreCheckResponse to []PreCheckResultItem
func convertResults(items []*sqladmin.PreCheckResponse) []PreCheckResultItem {
	if len(items) == 0 { // Handle nil or empty slice
		return []PreCheckResultItem{}
	}
	results := make([]PreCheckResultItem, len(items))
	for i, item := range items {
		results[i] = PreCheckResultItem{
			Message:         item.Message,
			MessageType:     item.MessageType,
			ActionsRequired: item.ActionsRequired,
		}
	}
	return results
}

// Invoke executes the tool's logic.
func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	project, ok := paramsMap["project"].(string)
	if !ok || project == "" {
		return nil, fmt.Errorf("missing or empty 'project' parameter")
	}
	instanceName, ok := paramsMap["instance"].(string)
	if !ok || instanceName == "" {
		return nil, fmt.Errorf("missing or empty 'instance' parameter")
	}
	targetVersion, ok := paramsMap["targetDatabaseVersion"].(string)
	if !ok || targetVersion == "" {
		// This should not happen due to the default value
		return nil, fmt.Errorf("missing or empty 'targetDatabaseVersion' parameter")
	}

	service, err := t.Source.GetService(ctx, string(accessToken))
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP client from source: %w", err)
	}

	reqBody := &sqladmin.InstancesPreCheckMajorVersionUpgradeRequest{
		PreCheckMajorVersionUpgradeContext: &sqladmin.PreCheckMajorVersionUpgradeContext{
			TargetDatabaseVersion: targetVersion,
		},
	}

	call := service.Instances.PreCheckMajorVersionUpgrade(project, instanceName, reqBody).Context(ctx)
	op, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to start pre-check operation: %w", err)
	}

	const pollTimeout = 5 * time.Minute
	timeout := time.After(pollTimeout)

	for {
		currentOp, err := service.Operations.Get(project, op.Name).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to get operation status: %w", err)
		}

		if currentOp.Status == "DONE" {
			if currentOp.Error != nil && len(currentOp.Error.Errors) > 0 {
				errMsg := fmt.Sprintf("pre-check operation LRO failed: %s", currentOp.Error.Errors[0].Message)
				if currentOp.Error.Errors[0].Code != "" {
					errMsg = fmt.Sprintf("%s (Code: %s)", errMsg, currentOp.Error.Errors[0].Code)
				}
				return nil, fmt.Errorf("%s", errMsg)
			}

			var preCheckItems []*sqladmin.PreCheckResponse
			if currentOp.PreCheckMajorVersionUpgradeContext != nil {
				preCheckItems = currentOp.PreCheckMajorVersionUpgradeContext.PreCheckResponse
			}
			// convertResults handles nil or empty preCheckItems
			return PreCheckAPIResponse{Items: convertResults(preCheckItems)}, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("timed out after %v waiting for operation %s to complete", pollTimeout, op.Name)
		case <-time.After(10 * time.Second):
		}
	}
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
	return t.Source.UseClientAuthorization()
}
