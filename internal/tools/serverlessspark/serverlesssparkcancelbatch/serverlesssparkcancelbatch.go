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

package serverlesssparkcancelbatch

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/serverlessspark"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind = "serverless-spark-cancel-batch"

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

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

// ToolConfigKind returns the unique name for this tool.
func (cfg Config) ToolConfigKind() string {
	return kind
}

// Initialize creates a new Tool instance.
func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("source %q not found", cfg.Source)
	}

	ds, ok := rawS.(*serverlessspark.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `%s`", kind, serverlessspark.SourceKind)
	}

	desc := cfg.Description
	if desc == "" {
		desc = "Cancels a running Serverless Spark (aka Dataproc Serverless) batch operation. Note that the batch state will not change immediately after the tool returns; it can take a minute or so for the cancellation to be reflected."
	}

	allParameters := parameters.Parameters{
		parameters.NewStringParameter("operation", "The name of the operation to cancel, e.g. for \"projects/my-project/locations/us-central1/operations/my-operation\", pass \"my-operation\""),
	}
	inputSchema, _ := allParameters.McpManifest()

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: desc,
		InputSchema: inputSchema,
	}

	return &Tool{
		Config:      cfg,
		Source:      ds,
		manifest:    tools.Manifest{Description: desc, Parameters: allParameters.Manifest()},
		mcpManifest: mcpManifest,
		Parameters:  allParameters,
	}, nil
}

// Tool is the implementation of the tool.
type Tool struct {
	Config

	Source *serverlessspark.Source

	manifest    tools.Manifest
	mcpManifest tools.McpManifest
	Parameters  parameters.Parameters
}

// Invoke executes the tool's operation.
func (t *Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	client, err := t.Source.GetOperationsClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get operations client: %w", err)
	}

	paramMap := params.AsMap()
	operation, ok := paramMap["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing required parameter: operation")
	}

	if strings.Contains(operation, "/") {
		return nil, fmt.Errorf("operation must be a short operation name without '/': %s", operation)
	}

	req := &longrunningpb.CancelOperationRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", t.Source.Project, t.Source.Location, operation),
	}

	err = client.CancelOperation(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel operation: %w", err)
	}

	return fmt.Sprintf("Cancelled [%s].", operation), nil
}

func (t *Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Parameters, data, claims)
}

func (t *Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t *Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t *Tool) Authorized(services []string) bool {
	return tools.IsAuthorized(t.AuthRequired, services)
}

func (t *Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) bool {
	// Client OAuth not supported, rely on ADCs.
	return false
}

func (t *Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
