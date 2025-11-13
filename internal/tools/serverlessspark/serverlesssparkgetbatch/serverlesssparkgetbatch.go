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

package serverlesssparkgetbatch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cloud.google.com/go/dataproc/v2/apiv1/dataprocpb"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/serverlessspark"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/protobuf/encoding/protojson"
)

const kind = "serverless-spark-get-batch"

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
		desc = "Gets a Serverless Spark (aka Dataproc Serverless) batch"
	}

	allParameters := parameters.Parameters{
		parameters.NewStringParameter("name", "The short name of the batch, e.g. for \"projects/my-project/locations/us-central1/batches/my-batch\", pass \"my-batch\" (the project and location are inherited from the source)"),
	}
	inputSchema, _ := allParameters.McpManifest()

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: desc,
		InputSchema: inputSchema,
	}

	return Tool{
		Name:         cfg.Name,
		Kind:         kind,
		Source:       ds,
		AuthRequired: cfg.AuthRequired,
		manifest:     tools.Manifest{Description: desc, Parameters: allParameters.Manifest()},
		mcpManifest:  mcpManifest,
		Parameters:   allParameters,
	}, nil
}

// Tool is the implementation of the tool.
type Tool struct {
	Name         string   `yaml:"name"`
	Kind         string   `yaml:"kind"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`

	Source *serverlessspark.Source

	manifest    tools.Manifest
	mcpManifest tools.McpManifest
	Parameters  parameters.Parameters
}

// Invoke executes the tool's operation.
func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	client := t.Source.GetBatchControllerClient()

	paramMap := params.AsMap()
	name, ok := paramMap["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing required parameter: name")
	}

	if strings.Contains(name, "/") {
		return nil, fmt.Errorf("name must be a short batch name without '/': %s", name)
	}

	req := &dataprocpb.GetBatchRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/batches/%s", t.Source.Project, t.Source.Location, name),
	}

	batchPb, err := client.GetBatch(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch: %w", err)
	}

	jsonBytes, err := protojson.Marshal(batchPb)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch to JSON: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch JSON: %w", err)
	}

	return result, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Parameters, data, claims)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t Tool) Authorized(services []string) bool {
	return tools.IsAuthorized(t.AuthRequired, services)
}

func (t Tool) RequiresClientAuthorization() bool {
	// Client OAuth not supported, rely on ADCs.
	return false
}
