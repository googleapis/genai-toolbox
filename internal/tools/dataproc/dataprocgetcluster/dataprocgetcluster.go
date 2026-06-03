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

package dataprocgetcluster

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/sources/dataproc"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const kind = "dataproc-get-cluster"

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
func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("source %q not found", cfg.Source)
	}

	_, ok = rawS.(*dataproc.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source type must be `%s`", kind, dataproc.SourceType)
	}

	desc := cfg.Description
	if desc == "" {
		desc = "Gets a Dataproc cluster"
	}

	allParameters := parameters.Parameters{
		parameters.NewStringParameterWithRequired("clusterName", "The short name of the cluster, e.g. for \"projects/my-project/regions/us-central1/clusters/my-cluster\", pass \"my-cluster\" (the project and region are inherited from the source)", false),
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

type compatibleSource interface {
	GetCluster(context.Context, string) (any, error)
}

// Invoke executes the tool's operation.
func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Cfg.Source, t.Cfg.Name, kind)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	paramMap := params.AsMap()
	name, ok := paramMap["clusterName"].(string)
	if !ok {
		return nil, util.NewAgentError("missing required parameter: clusterName", nil)
	}
	if strings.Contains(name, "/") {
		return nil, util.NewAgentError(fmt.Sprintf("clusterName must be a short name without '/': %s", name), nil)
	}

	res, err := source.GetCluster(ctx, name)
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}
	return res, nil
}
