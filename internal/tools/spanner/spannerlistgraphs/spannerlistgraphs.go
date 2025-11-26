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
	"encoding/json"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	spannerdb "github.com/googleapis/genai-toolbox/internal/sources/spanner"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/api/iterator"
)

const kind string = "spanner-list-graphs"

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
}

// validate compatible sources are still compatible
var _ compatibleSource = &spannerdb.Source{}

var compatibleSources = [...]string{spannerdb.SourceKind}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description"`
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

	// verify the dialect is GoogleSQL
	if strings.ToLower(s.DatabaseDialect()) != "googlesql" {
		return nil, fmt.Errorf("invalid source dialect for %q tool: source dialect must be GoogleSQL", kind)
	}

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
		description = "Lists detailed graph schema information (node tables, edge tables, labels and property declarations) as JSON for user-created graphs. Filters by a comma-separated list of names. If names are omitted, lists all graphs in user schemas."
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, description, cfg.AuthRequired, allParameters, nil)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		AllParams:   allParameters,
		Client:      s.SpannerClient(),
		dialect:     s.DatabaseDialect(),
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
	Client      *spanner.Client
	dialect     string
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// processRows iterates over the spanner.RowIterator and converts each row to a map[string]any.
func processRows(iter *spanner.RowIterator) ([]any, error) {
	var out []any
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}

		vMap := make(map[string]any)
		cols := row.ColumnNames()
		for i, c := range cols {
			if c == "object_details" {
				jsonString := row.ColumnValue(i).AsInterface().(string)
				var details map[string]interface{}
				if err := json.Unmarshal([]byte(jsonString), &details); err != nil {
					return nil, fmt.Errorf("unable to unmarshal JSON: %w", err)
				}
				vMap[c] = details
			} else {
				vMap[c] = row.ColumnValue(i)
			}
		}
		out = append(out, vMap)
	}
	return out, nil
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
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

	stmt := spanner.Statement{
		SQL:    googleSQLStatement,
		Params: stmtParams,
	}

	// Execute the query (read-only)
	iter := t.Client.Single().Query(ctx, stmt)
	results, err := processRows(iter)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}

	return results, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.AllParams, data, claims)
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

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
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
