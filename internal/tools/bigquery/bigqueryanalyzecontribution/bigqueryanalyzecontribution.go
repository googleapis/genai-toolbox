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

package bigqueryanalyzecontribution

import (
	"context"
	"fmt"
	"strings"

	bigqueryapi "cloud.google.com/go/bigquery"
	yaml "github.com/goccy/go-yaml"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/sources"
	bigqueryds "github.com/googleapis/genai-toolbox/internal/sources/bigquery"
	"github.com/googleapis/genai-toolbox/internal/tools"
	bigqueryrestapi "google.golang.org/api/bigquery/v2"
	"google.golang.org/api/iterator"
)

const kind string = "bigquery-analyze-contribution"

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
	BigQuerySession() *bigqueryds.Session
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

	inputDataParameter := tools.NewStringParameter("input_data",
		"The data that contain the test and control data to analyze. Can be a fully qualified BigQuery table ID or a SQL query.")
	contributionMetricParameter := tools.NewStringParameter("contribution_metric",
		`The name of the column that contains the metric to analyze.
		Provides the expression to use to calculate the metric you are analyzing.
		To calculate a summable metric, the expression must be in the form SUM(metric_column_name),
		where metric_column_name is a numeric data type.

		To calculate a summable ratio metric, the expression must be in the form
		SUM(numerator_metric_column_name)/SUM(denominator_metric_column_name),
		where numerator_metric_column_name and denominator_metric_column_name are numeric data types.

		To calculate a summable by category metric, the expression must be in the form
		SUM(metric_sum_column_name)/COUNT(DISTINCT categorical_column_name). The summed column must be a numeric data type.
		The categorical column must have type BOOL, DATE, DATETIME, TIME, TIMESTAMP, STRING, or INT64.`)
	isTestColParameter := tools.NewStringParameter("is_test_col",
		"The name of the column that identifies whether a row is in the test or control group.")
	dimensionIDColsParameter := tools.NewArrayParameterWithRequired("dimension_id_cols",
		"An array of column names that uniquely identify each dimension.", false, tools.NewStringParameter("dimension_id_col", "A dimension column name."))
	topKInsightsParameter := tools.NewIntParameterWithDefault("top_k_insights_by_apriori_support", 30,
		"The number of top insights to return, ranked by apriori support.")
	pruningMethodParameter := tools.NewStringParameterWithDefault("pruning_method", "PRUNE_REDUNDANT_INSIGHTS",
		"The method to use for pruning redundant insights. Can be 'NO_PRUNING' or 'PRUNE_REDUNDANT_INSIGHTS'.")

	parameters := tools.Parameters{
		inputDataParameter,
		contributionMetricParameter,
		isTestColParameter,
		dimensionIDColsParameter,
		topKInsightsParameter,
		pruningMethodParameter,
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, parameters)

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
		Session:        s.BigQuerySession(),
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
	Session       *bigqueryds.Session
	manifest      tools.Manifest
	mcpManifest   tools.McpManifest
}

// Invoke runs the contribution analysis.
func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()
	inputData, ok := paramsMap["input_data"].(string)
	if !ok {
		return nil, fmt.Errorf("unable to cast input_data parameter %s", paramsMap["input_data"])
	}

	modelID := fmt.Sprintf("contribution_analysis_model_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))

	var options []string
	options = append(options, "MODEL_TYPE = 'CONTRIBUTION_ANALYSIS'")
	options = append(options, fmt.Sprintf("CONTRIBUTION_METRIC = '%s'", paramsMap["contribution_metric"]))
	options = append(options, fmt.Sprintf("IS_TEST_COL = '%s'", paramsMap["is_test_col"]))

	if val, ok := paramsMap["dimension_id_cols"]; ok {
		if cols, ok := val.([]any); ok {
			var strCols []string
			for _, c := range cols {
				strCols = append(strCols, fmt.Sprintf("'%s'", c))
			}
			options = append(options, fmt.Sprintf("DIMENSION_ID_COLS = [%s]", strings.Join(strCols, ", ")))
		} else {
			return nil, fmt.Errorf("unable to cast dimension_id_cols parameter %s", paramsMap["dimension_id_cols"])
		}
	}
	if val, ok := paramsMap["top_k_insights_by_apriori_support"]; ok {
		options = append(options, fmt.Sprintf("TOP_K_INSIGHTS_BY_APRIORI_SUPPORT = %v", val))
	}
	if val, ok := paramsMap["pruning_method"].(string); ok {
		upperVal := strings.ToUpper(val)
		if upperVal != "NO_PRUNING" && upperVal != "PRUNE_REDUNDANT_INSIGHTS" {
			return nil, fmt.Errorf("invalid pruning_method: %s", val)
		}
		options = append(options, fmt.Sprintf("PRUNING_METHOD = '%s'", upperVal))
	}

	var inputDataSource string
	trimmedUpperInputData := strings.TrimSpace(strings.ToUpper(inputData))
	if strings.HasPrefix(trimmedUpperInputData, "SELECT") || strings.HasPrefix(trimmedUpperInputData, "WITH") {
		inputDataSource = fmt.Sprintf("(%s)", inputData)
	} else {
		inputDataSource = fmt.Sprintf("SELECT * FROM `%s`", inputData)
	}

	// Use temp model to skip the clean up at the end. To use TEMP MODEL, queries have to be
	// in the same BigQuery session.
	createModelSQL := fmt.Sprintf("CREATE TEMP MODEL %s OPTIONS(%s) AS %s",
		modelID,
		strings.Join(options, ", "),
		inputDataSource,
	)

	bqClient := t.Client
	var err error

	// Initialize new client if using user OAuth token
	if t.UseClientOAuth {
		tokenStr, err := accessToken.ParseBearerToken()
		if err != nil {
			return nil, fmt.Errorf("error parsing access token: %w", err)
		}
		bqClient, _, err = t.ClientCreator(tokenStr, false)
		if err != nil {
			return nil, fmt.Errorf("error creating client from OAuth access token: %w", err)
		}
	}

	createModelQuery := bqClient.Query(createModelSQL)
	if t.Session != nil {
		createModelQuery.ConnectionProperties = []*bigqueryapi.ConnectionProperty{
			{Key: "session_id", Value: t.Session.ID},
		}
	} else {
		createModelQuery.CreateSession = true
	}
	createModelJob, err := createModelQuery.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start create model job: %w", err)
	}

	status, err := createModelJob.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for create model job: %w", err)
	}
	if err := status.Err(); err != nil {
		return nil, fmt.Errorf("create model job failed: %w", err)
	}

	var sessionID string
	if t.Session != nil {
		sessionID = t.Session.ID
	} else if status.Statistics != nil && status.Statistics.SessionInfo != nil {
		sessionID = status.Statistics.SessionInfo.SessionID
	} else {
		return nil, fmt.Errorf("failed to get a BigQuery session ID")
	}

	getInsightsSQL := fmt.Sprintf("SELECT * FROM ML.GET_INSIGHTS(MODEL %s)", modelID)

	getInsightsQuery := bqClient.Query(getInsightsSQL)
	getInsightsQuery.ConnectionProperties = []*bigqueryapi.ConnectionProperty{
		{Key: "session_id", Value: sessionID},
	}

	job, err := getInsightsQuery.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get insights query: %w", err)
	}
	it, err := job.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to read query results: %w", err)
	}

	var out []any
	for {
		var row map[string]bigqueryapi.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate through query results: %w", err)
		}
		vMap := make(map[string]any)
		for key, value := range row {
			vMap[key] = value
		}
		out = append(out, vMap)
	}

	if len(out) > 0 {
		return out, nil
	}

	// This handles the standard case for a SELECT query that successfully
	// executes but returns zero rows.
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
