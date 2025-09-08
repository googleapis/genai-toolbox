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

package bigqueryforecast

import (
	"context"
	"fmt"
	"strings"

	bigqueryapi "cloud.google.com/go/bigquery"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	bigqueryds "github.com/googleapis/genai-toolbox/internal/sources/bigquery"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	bigqueryrestapi "google.golang.org/api/bigquery/v2"
	"google.golang.org/api/iterator"
)

const kind string = "bigquery-forecast"

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
	BigQueryClient() *bigqueryapi.Client
	BigQueryRestService() *bigqueryrestapi.Service
	BigQueryClientCreator() bigqueryds.BigqueryClientCreator
	UseClientAuthorization() bool
}

// validate compatible sources are still compatible
var _ compatibleSource = &bigqueryds.Source{}

var compatibleSources = [...]string{bigqueryds.SourceKind}

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

	historyDataParameter := tools.NewStringParameter("history_data",
		"The table id or the query of the history time series data.")
	timestampColumnNameParameter := tools.NewStringParameter("timestamp_col",
		"The name of the time series timestamp column.")
	dataColumnNameParameter := tools.NewStringParameter("data_col",
		"The name of the time series data column.")
	idColumnNameParameter := tools.NewArrayParameterWithDefault("id_cols", []any{},
		"An array of the time series id column names.",
		tools.NewStringParameter("id_col", "The name of time series id column."))
	horizonParameter := tools.NewIntParameterWithDefault("horizon", 10, "The number of forecasting steps.")
	parameters := tools.Parameters{historyDataParameter,
		timestampColumnNameParameter, dataColumnNameParameter, idColumnNameParameter, horizonParameter}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	// finish tool setup
	t := Tool{
		Name:           cfg.Name,
		Kind:           kind,
		Parameters:     parameters,
		AuthRequired:   cfg.AuthRequired,
		UseClientOAuth: s.UseClientAuthorization(),
		ClientCreator:  s.BigQueryClientCreator(),
		Client:         s.BigQueryClient(),
		RestService:    s.BigQueryRestService(),
		manifest:       tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:    mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name           string           `yaml:"name"`
	Kind           string           `yaml:"kind"`
	AuthRequired   []string         `yaml:"authRequired"`
	UseClientOAuth bool             `yaml:"useClientOAuth"`
	Parameters     tools.Parameters `yaml:"parameters"`

	Client        *bigqueryapi.Client
	RestService   *bigqueryrestapi.Service
	ClientCreator bigqueryds.BigqueryClientCreator
	manifest      tools.Manifest
	mcpManifest   tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()
	historyData, ok := paramsMap["history_data"].(string)
	if !ok {
		return nil, fmt.Errorf("unable to cast history_data parameter %v", paramsMap["history_data"])
	}
	timestampCol, ok := paramsMap["timestamp_col"].(string)
	if !ok {
		return nil, fmt.Errorf("unable to cast timestamp_col parameter %v", paramsMap["timestamp_col"])
	}
	dataCol, ok := paramsMap["data_col"].(string)
	if !ok {
		return nil, fmt.Errorf("unable to cast data_col parameter %v", paramsMap["data_col"])
	}
	idColsRaw, ok := paramsMap["id_cols"].([]any)
	if !ok {
		return nil, fmt.Errorf("unable to cast id_cols parameter %v", paramsMap["id_cols"])
	}
	var idCols []string
	for _, v := range idColsRaw {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("id_cols contains non-string value: %v", v)
		}
		idCols = append(idCols, s)
	}
	horizon, ok := paramsMap["horizon"].(int)
	if !ok {
		if h, ok := paramsMap["horizon"].(float64); ok {
			horizon = int(h)
		} else {
			return nil, fmt.Errorf("unable to cast horizon parameter %v", paramsMap["horizon"])
		}
	}

	var historyDataSource string
	trimmedUpperHistoryData := strings.TrimSpace(strings.ToUpper(historyData))
	if strings.HasPrefix(trimmedUpperHistoryData, "SELECT") || strings.HasPrefix(trimmedUpperHistoryData, "WITH") {
		historyDataSource = fmt.Sprintf("(%s)", historyData)
	} else {
		historyDataSource = fmt.Sprintf("TABLE `%s`", historyData)
	}

	idColsArg := ""
	if len(idCols) > 0 {
		idColsFormatted := fmt.Sprintf("['%s']", strings.Join(idCols, "', '"))
		idColsArg = fmt.Sprintf(", id_cols => %s", idColsFormatted)
	}

	sql := fmt.Sprintf(`SELECT * 
		FROM AI.FORECAST(
			%s,
			data_col => '%s',
			timestamp_col => '%s',
			horizon => %d%s)`,
		historyDataSource, dataCol, timestampCol, horizon, idColsArg)

	bqClient := t.Client
	var err error

	// Initialize new client if using user OAuth token
	if t.UseClientOAuth {
		bqClient, _, err = t.ClientCreator(accessToken, false)
		if err != nil {
			return nil, fmt.Errorf("error creating client from OAuth access token: %w", err)
		}
	}

	// JobStatistics.QueryStatistics.StatementType
	query := bqClient.Query(sql)
	query.Location = bqClient.Location

	// Log the query executed for debugging.
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting logger: %s", err)
	}
	logger.DebugContext(ctx, "executing `%s` tool query: %s", kind, sql)

	// This block handles SELECT statements, which return a row set.
	// We iterate through the results, convert each row into a map of
	// column names to values, and return the collection of rows.
	var out []any
	it, err := query.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	for {
		var row map[string]bigqueryapi.Value
		err = it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("unable to iterate through query results: %w", err)
		}
		vMap := make(map[string]any)
		for key, value := range row {
			vMap[key] = value
		}
		out = append(out, vMap)
	}
	// If the query returned any rows, return them directly.
	if len(out) > 0 {
		return out, nil
	}

	// This handles the standard case for a SELECT query that successfully
	return "The query returned 0 rows.", nil
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
	return t.UseClientOAuth
}
