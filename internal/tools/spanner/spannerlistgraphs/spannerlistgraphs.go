// Copyright 2025 Google LLC
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

package spannerlistgraphs

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const resourceType string = "spanner-list-graphs"

func init() {
	if !tools.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", resourceType))
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
	RunSQL(context.Context, bool, string, map[string]any) (any, error)
}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Type         string   `yaml:"type" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// Define parameters for the tool
	allParameters := parameters.Parameters{
		parameters.NewStringParameterWithDefault(
			"graph_names",
			"",
			"Optional: A comma-separated list of graph names. If empty, details for all graphs in user-accessible schemas will be listed.",
		),
		parameters.NewStringParameterWithDefault(
			"output_format",
			"detailed",
			"Optional: Use 'simple' to return graph names only or use 'detailed' to return the full information schema.",
		),
	}

	description := cfg.Description
	if description == "" {
		description = "Lists detailed graph schema information (node tables, edge tables, labels and property declarations) as JSON for user-created graphs. Filters by a comma-separated list of graph names. If names are omitted, lists all graphs. The output can be 'simple' (graph names only) or 'detailed' (full schema)."
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, description, cfg.AuthRequired, allParameters, nil)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		AllParams:   allParameters,
		manifest:    tools.Manifest{Description: description, Parameters: allParameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	AllParams   parameters.Parameters `yaml:"allParams"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	// Check dialect here at RUNTIME instead of startup
	if strings.ToLower(source.DatabaseDialect()) != "googlesql" {
		return nil, util.NewAgentError(fmt.Sprintf("operation not supported: The 'spanner-list-graphs' tool is only available for GoogleSQL dialect databases. Your current database dialect is '%s'", source.DatabaseDialect()), nil)
	}

	paramsMap := params.AsMap()

	graphNames, _ := paramsMap["graph_names"].(string)
	outputFormat, _ := paramsMap["output_format"].(string)
	if outputFormat == "" {
		outputFormat = "detailed"
	}

	stmtParams := map[string]interface{}{
		"graph_names":   graphNames,
		"output_format": outputFormat,
	}
	resp, err := source.RunSQL(ctx, true, googleSQLStatement, stmtParams)
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}
	return resp, nil
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.AllParams, paramValues, embeddingModelsMap, nil)
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

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	return false, nil
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}

func (t Tool) GetParameters() parameters.Parameters {
	return t.AllParams
}

// GoogleSQL statement for listing graphs
const googleSQLStatement = `
WITH FilterGraphNames AS (
  SELECT DISTINCT TRIM(name) AS GRAPH_NAME
  FROM UNNEST(IF(@graph_names = '' OR @graph_names IS NULL, ['%'], SPLIT(@graph_names, ','))) AS name
)

SELECT 
	PG.PROPERTY_GRAPH_SCHEMA AS schema_name,
  PG.PROPERTY_GRAPH_NAME AS object_name,
  CASE
    WHEN @output_format = 'simple' THEN
      -- IF format is 'simple', return basic JSON
          CONCAT('{"name":"', IFNULL(REPLACE(PG.PROPERTY_GRAPH_NAME, '"', '\"'), ''), '"}')
    ELSE
      CONCAT(
        '{',
        '"schema_name":"', IFNULL(PG.PROPERTY_GRAPH_SCHEMA, ''), '",',
        '"object_name":"', IFNULL(PG.PROPERTY_GRAPH_NAME, ''), '",',
				'"catalog":"', IFNULL(JSON_VALUE(PG.PROPERTY_GRAPH_METADATA_JSON,"$.catalog"), ''), '",',
        '"node_tables":', TO_JSON_STRING(PG.PROPERTY_GRAPH_METADATA_JSON.nodeTables), ',',
				'"edge_tables":', TO_JSON_STRING(PG.PROPERTY_GRAPH_METADATA_JSON.edgeTables), ',',
				'"labels":', TO_JSON_STRING(PG.PROPERTY_GRAPH_METADATA_JSON.labels), ',',
				'"property_declarations":', TO_JSON_STRING(PG.PROPERTY_GRAPH_METADATA_JSON.propertyDeclarations),
        '}'
      )
  END AS object_details
FROM INFORMATION_SCHEMA.PROPERTY_GRAPHS PG
WHERE 
	EXISTS (SELECT 1 FROM FilterGraphNames WHERE FilterGraphNames.GRAPH_NAME = '%') OR PG.PROPERTY_GRAPH_NAME IN (SELECT GRAPH_NAME FROM FilterGraphNames)
`
