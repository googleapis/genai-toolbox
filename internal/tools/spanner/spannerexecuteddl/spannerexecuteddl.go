// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package spannerexecuteddl

import (
	"context"
	"fmt"
	"os/exec"

	"cloud.google.com/go/spanner"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	spannerdb "github.com/googleapis/genai-toolbox/internal/sources/spanner"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
)

var (
	execLookPath       = exec.LookPath
	execCommandContext = exec.CommandContext
)

const kind string = "spanner-execute-ddl"

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

type compatibleSource interface {
	SpannerClient() *spanner.Client
	DatabaseDialect() string
	SpannerProjectID() string
	SpannerInstanceID() string
	SpannerDatabaseID() string
}

// validate compatible sources are still compatible
var _ compatibleSource = &spannerdb.Source{}

var compatibleSources = [...]string{spannerdb.SourceKind}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	ddlParameter := tools.NewStringParameter("ddl", "The DDL statement to execute.")
	parameters := tools.Parameters{ddlParameter}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, parameters)

	// finish tool setup
	t := Tool{
		Name:         cfg.Name,
		Kind:         kind,
		Parameters:   parameters,
		AuthRequired: cfg.AuthRequired,
		projectID:    s.SpannerProjectID(),
		instanceID:   s.SpannerInstanceID(),
		databaseID:   s.SpannerDatabaseID(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`
	projectID    string
	instanceID   string
	databaseID   string
	manifest     tools.Manifest
	mcpManifest  tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()
	ddl, ok := paramsMap["ddl"].(string)
	if !ok {
		return nil, fmt.Errorf("unable to get cast %s", paramsMap["ddl"])
	}

	// Log the query executed for debugging.
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting logger: %s", err)
	}
	logger.DebugContext(ctx, "executing `%s` tool ddl: %s", kind, ddl)

	// Check if gcloud is installed
	if _, err := execLookPath("gcloud"); err != nil {
		return nil, fmt.Errorf("gcloud is not installed or not in your PATH. Please install the Google Cloud SDK to use this tool")
	}

	cmd := execCommandContext(ctx, "gcloud", "spanner", "databases", "ddl", "update", t.databaseID,
		"--instance", t.instanceID,
		"--project", t.projectID,
		"--ddl", ddl)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error executing gcloud command: %w\n%s", err, string(output))
	}

	return string(output), nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data, claims)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) RequiresClientAuthorization() bool {
	return false
}
