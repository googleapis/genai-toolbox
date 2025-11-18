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

package serverlesssparkcreatepysparkbatch

import (
	"context"
	"encoding/json"
	"fmt"

	dataproc "cloud.google.com/go/dataproc/v2/apiv1/dataprocpb"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/serverlessspark"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const kind = "serverless-spark-create-pyspark-batch"

func init() {
	if !tools.Register(kind, newConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", kind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	// Use a temporary struct to decode the YAML, so that we can handle the proto
	// conversion for RuntimeConfig and EnvironmentConfig.
	var ymlCfg struct {
		Name              string   `yaml:"name"`
		Kind              string   `yaml:"kind"`
		Source            string   `yaml:"source"`
		Description       string   `yaml:"description"`
		RuntimeConfig     any      `yaml:"runtimeConfig"`
		EnvironmentConfig any      `yaml:"environmentConfig"`
		AuthRequired      []string `yaml:"authRequired"`
	}

	if err := decoder.DecodeContext(ctx, &ymlCfg); err != nil {
		return nil, err
	}

	cfg := Config{
		Name:         name,
		Kind:         ymlCfg.Kind,
		Source:       ymlCfg.Source,
		Description:  ymlCfg.Description,
		AuthRequired: ymlCfg.AuthRequired,
	}

	if ymlCfg.RuntimeConfig != nil {
		rc := &dataproc.RuntimeConfig{}
		jsonData, err := json.Marshal(ymlCfg.RuntimeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal runtimeConfig: %w", err)
		}
		if err := protojson.Unmarshal(jsonData, rc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal runtimeConfig: %w", err)
		}
		cfg.RuntimeConfig = rc
	}

	if ymlCfg.EnvironmentConfig != nil {
		ec := &dataproc.EnvironmentConfig{}
		jsonData, err := json.Marshal(ymlCfg.EnvironmentConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal environmentConfig: %w", err)
		}
		if err := protojson.Unmarshal(jsonData, ec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal environmentConfig: %w", err)
		}
		cfg.EnvironmentConfig = ec
	}

	return cfg, nil
}

type Config struct {
	Name              string                      `yaml:"name" validate:"required"`
	Kind              string                      `yaml:"kind" validate:"required"`
	Source            string                      `yaml:"source" validate:"required"`
	Description       string                      `yaml:"description"`
	RuntimeConfig     *dataproc.RuntimeConfig     `yaml:"runtimeConfig"`
	EnvironmentConfig *dataproc.EnvironmentConfig `yaml:"environmentConfig"`
	AuthRequired      []string                    `yaml:"authRequired"`
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
		desc = "Creates a Serverless Spark (aka Dataproc Serverless) PySpark batch operation."
	}

	allParameters := parameters.Parameters{
		parameters.NewStringParameterWithRequired("mainFile", "The path to the main Python file, as a gs://... URI.", true),
		parameters.NewArrayParameterWithRequired("args", "Optional. A list of arguments passed to the main file.", false, parameters.NewStringParameter("arg", "An argument.")),
		parameters.NewStringParameterWithRequired("version", "Optional. The Serverless runtime version to execute with.", false),
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
	client := t.Source.GetBatchControllerClient()

	paramMap := params.AsMap()

	mainFile := paramMap["mainFile"].(string)

	batch := &dataproc.Batch{
		BatchConfig: &dataproc.Batch_PysparkBatch{
			PysparkBatch: &dataproc.PySparkBatch{
				MainPythonFileUri: mainFile,
			},
		},
	}

	if args, ok := paramMap["args"].([]any); ok {
		for _, arg := range args {
			batch.GetPysparkBatch().Args = append(batch.GetPysparkBatch().Args, fmt.Sprintf("%v", arg))
		}
	}

	if t.Config.RuntimeConfig != nil {
		batch.RuntimeConfig = proto.Clone(t.Config.RuntimeConfig).(*dataproc.RuntimeConfig)
	}

	if t.Config.EnvironmentConfig != nil {
		batch.EnvironmentConfig = proto.Clone(t.Config.EnvironmentConfig).(*dataproc.EnvironmentConfig)
	}

	if version, ok := paramMap["version"].(string); ok && version != "" {
		if batch.RuntimeConfig == nil {
			batch.RuntimeConfig = &dataproc.RuntimeConfig{}
		}
		batch.RuntimeConfig.Version = version
	}

	req := &dataproc.CreateBatchRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", t.Source.Project, t.Source.Location),
		Batch:  batch,
	}

	op, err := client.CreateBatch(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create batch: %w", err)
	}

	meta, err := op.Metadata()
	if err != nil {
		return nil, fmt.Errorf("failed to get create batch op metadata: %w", err)
	}

	jsonBytes, err := protojson.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create batch op metadata to JSON: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create batch op metadata JSON: %w", err)
	}

	return result, nil
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

func (t *Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
