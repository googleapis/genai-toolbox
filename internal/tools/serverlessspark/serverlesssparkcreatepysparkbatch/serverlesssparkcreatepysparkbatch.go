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
	"fmt"

	dataproc "cloud.google.com/go/dataproc/v2/apiv1/dataprocpb"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/serverlessspark/createbatch"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind = "serverless-spark-create-pyspark-batch"

func init() {
	if !tools.Register(kind, newConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", kind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	baseCfg, err := createbatch.NewConfig(ctx, name, decoder)
	if err != nil {
		return nil, err
	}
	return Config{baseCfg}, nil
}

type Config struct {
	createbatch.Config
}

// validate interface
var _ tools.ToolConfig = Config{}

// ToolConfigKind returns the unique name for this tool.
func (cfg Config) ToolConfigKind() string {
	return kind
}

// Initialize creates a new Tool instance.
func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	return createbatch.NewTool(cfg.Config, cfg, srcs, &PySparkBatchBuilder{})
}

type PySparkBatchBuilder struct{}

func (b *PySparkBatchBuilder) Parameters() parameters.Parameters {
	return parameters.Parameters{
		parameters.NewStringParameterWithRequired("mainFile", "The path to the main Python file, as a gs://... URI.", true),
		parameters.NewArrayParameterWithRequired("args", "Optional. A list of arguments passed to the main file.", false, parameters.NewStringParameter("arg", "An argument.")),
		parameters.NewStringParameterWithRequired("version", "Optional. The Serverless runtime version to execute with.", false),
	}
}

func (b *PySparkBatchBuilder) BuildBatch(params parameters.ParamValues) (*dataproc.Batch, error) {
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

	return batch, nil
}
