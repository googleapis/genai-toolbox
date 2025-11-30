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

package serverlesssparkcreatesparkbatch

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

const kind = "serverless-spark-create-spark-batch"

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
	return createbatch.NewTool(cfg.Config, cfg, srcs, &SparkBatchBuilder{})
}

type SparkBatchBuilder struct{}

func (b *SparkBatchBuilder) Parameters() parameters.Parameters {
	return parameters.Parameters{
		parameters.NewStringParameterWithRequired("mainJarFile", "Optional. The gs:// URI of the jar file that contains the main class. Exactly one of mainJarFile or mainClass must be specified.", false),
		parameters.NewStringParameterWithRequired("mainClass", "Optional. The name of the driver's main class. Exactly one of mainJarFile or mainClass must be specified.", false),
		parameters.NewArrayParameterWithRequired("jarFiles", "Optional. A list of gs:// URIs of jar files to add to the CLASSPATHs of the Spark driver and tasks.", false, parameters.NewStringParameter("jarFile", "A jar file URI.")),
		parameters.NewArrayParameterWithRequired("args", "Optional. A list of arguments passed to the driver.", false, parameters.NewStringParameter("arg", "An argument.")),
		parameters.NewStringParameterWithRequired("version", "Optional. The Serverless runtime version to execute with.", false),
	}
}

func (b *SparkBatchBuilder) BuildBatch(params parameters.ParamValues) (*dataproc.Batch, error) {
	paramMap := params.AsMap()

	mainJar, _ := paramMap["mainJarFile"].(string)
	mainClass, _ := paramMap["mainClass"].(string)

	if mainJar == "" && mainClass == "" {
		return nil, fmt.Errorf("must provide either mainJarFile or mainClass")
	}
	if mainJar != "" && mainClass != "" {
		return nil, fmt.Errorf("cannot provide both mainJarFile and mainClass")
	}

	sparkBatch := &dataproc.SparkBatch{}
	if mainJar != "" {
		sparkBatch.Driver = &dataproc.SparkBatch_MainJarFileUri{MainJarFileUri: mainJar}
	} else {
		sparkBatch.Driver = &dataproc.SparkBatch_MainClass{MainClass: mainClass}
	}

	if jarFileUris, ok := paramMap["jarFiles"].([]any); ok {
		for _, uri := range jarFileUris {
			sparkBatch.JarFileUris = append(sparkBatch.JarFileUris, fmt.Sprintf("%v", uri))
		}
	} else if mainClass != "" {
		return nil, fmt.Errorf("jarFiles is required when mainClass is provided")
	}

	if args, ok := paramMap["args"].([]any); ok {
		for _, arg := range args {
			sparkBatch.Args = append(sparkBatch.Args, fmt.Sprintf("%v", arg))
		}
	}

	return &dataproc.Batch{
		BatchConfig: &dataproc.Batch_SparkBatch{
			SparkBatch: sparkBatch,
		},
	}, nil
}
