// Copyright 2026 Google LLC
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

package dataprocgetjob

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const kind = "dataproc-get-job"

func init() {
	if !tools.Register(kind, newConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", kind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{ConfigBase: tools.ConfigBase{Name: name}}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	tools.ConfigBase `yaml:",inline"`
	Type             string                 `yaml:"type" validate:"required"`
	Source           string                 `yaml:"source" validate:"required"`
	Annotations      *tools.ToolAnnotations `yaml:"annotations,omitempty"`
}

// validate interface
var _ tools.ToolConfig = Config{}

// ToolConfigType returns the unique name for this tool.
func (cfg Config) ToolConfigType() string {
	return kind
}

// Initialize creates a new Tool instance.
func (cfg Config) Initialize(context.Context) (tools.Tool, error) {
	desc := cfg.Description
	if desc == "" {
		desc = "Gets a Dataproc job"
	}

	allParameters := parameters.Parameters{
		parameters.NewStringParameter("jobId", "The job ID, e.g. for \"projects/my-project/regions/us-central1/jobs/my-job\", pass \"my-job\" (the project and region are inherited from the source)", parameters.WithStringRequired(false)),
	}
	return Tool{
		BaseTool: tools.NewBaseTool(
			cfg,
			tools.GetAnnotationsOrDefault(cfg.Annotations, tools.NewReadOnlyAnnotations),
			tools.Manifest{Description: desc, Parameters: allParameters.Manifest(), AuthRequired: cfg.AuthRequired},
			allParameters,
		),
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

// Tool is the implementation of the tool.
type Tool struct {
	tools.BaseTool[Config]
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Cfg
}

func (t Tool) validate(srcs map[string]sources.Source) error {
	_, err := tools.GetCompatibleSourceFromMap[compatibleSource](srcs, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	return err
}

func (t Tool) GetParameters(srcs map[string]sources.Source) (parameters.Parameters, error) {
	if err := t.validate(srcs); err != nil {
		return nil, err
	}
	return t.BaseTool.GetParameters(srcs)
}

func (t Tool) Manifest(srcs map[string]sources.Source) (tools.Manifest, error) {
	if err := t.validate(srcs); err != nil {
		return tools.Manifest{}, err
	}
	return t.BaseTool.Manifest(srcs)
}

type compatibleSource interface {
	GetJob(context.Context, string) (any, error)
}

// Invoke executes the tool's operation.
func (t Tool) Invoke(ctx context.Context, primitiveMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](primitiveMgr, t.Cfg.Source, t.Cfg.Name, kind)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	paramMap := params.AsMap()
	jobId, ok := paramMap["jobId"].(string)
	if !ok {
		return nil, util.NewAgentError("missing required parameter: jobId", nil)
	}
	if strings.Contains(jobId, "/") {
		return nil, util.NewAgentError(fmt.Sprintf("jobId must be a short name without '/': %s", jobId), nil)
	}

	res, err := source.GetJob(ctx, jobId)
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}
	return res, nil
}
