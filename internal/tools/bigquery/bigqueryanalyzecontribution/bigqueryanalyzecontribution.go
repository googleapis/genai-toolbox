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
	bqutil "github.com/googleapis/genai-toolbox/internal/tools/bigquery/bigquerycommon"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
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
	IsDatasetAllowed(projectID, datasetID string) bool
	BigQueryAllowedDatasets() []string
	BigQuerySession() bigqueryds.BigQuerySessionProvider
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

	allowedDatasets := s.BigQueryAllowedDatasets()
	inputDataDescription := "The data that contain the test and control data to analyze. Can be a fully qualified BigQuery table ID or a SQL query."
	if len(allowedDatasets) > 0 {
		datasetIDs := []string{}
		for _, ds := range allowedDatasets {
			datasetIDs = append(datasetIDs, fmt.Sprintf("`%s`", ds))
		}
		inputDataDescription += fmt.Sprintf(" The query or table must only access datasets from the following list: %s.", strings.Join(datasetIDs, ", "))
	}

	inputDataParameter := parameters.NewStringParameter("input_data", inputDataDescription)
	contributionMetricParameter := parameters.NewStringParameter("contribution_metric",
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
	isTestColParameter := parameters.NewStringParameter("is_test_col",
		"The name of the column that identifies whether a row is in the test or control group.")
	dimensionIDColsParameter := parameters.NewArrayParameterWithRequired("dimension_id_cols",
		"An array of column names that uniquely identify each dimension.", false, parameters.NewStringParameter("dimension_id_col", "A dimension column name."))
	topKInsightsParameter := parameters.NewIntParameterWithDefault("top_k_insights_by_apriori_support", 30,
		"The number of top insights to return, ranked by apriori support.")
	pruningMethodParameter := parameters.NewStringParameterWithDefault("pruning_method", "PRUNE_REDUNDANT_INSIGHTS",
		"The method to use for pruning redundant insights. Can be 'NO_PRUNING' or 'PRUNE_REDUNDANT_INSIGHTS'.")

	params := parameters.Parameters{
		inputDataParameter,
		contributionMetricParameter,
		isTestColParameter,
		dimensionIDColsParameter,
		topKInsightsParameter,
		pruningMethodParameter,
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, nil)

	// finish tool setup
	t := Tool{
		Config:           cfg,
		Parameters:       params,
		UseClientOAuth:   s.UseClientAuthorization(),
		ClientCreator:    s.BigQueryClientCreator(),
		Client:           s.BigQueryClient(),
		RestService:      s.BigQueryRestService(),
		IsDatasetAllowed: s.IsDatasetAllowed,
		AllowedDatasets:  allowedDatasets,
		SessionProvider:  s.BigQuerySession(),
		manifest:         tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:      mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	UseClientOAuth bool                  `yaml:"useClientOAuth"`
	Parameters     parameters.Parameters `yaml:"parameters"`

	Client           *bigqueryapi.Client
	RestService      *bigqueryrestapi.Service
	ClientCreator    bigqueryds.BigqueryClientCreator
	IsDatasetAllowed func(projectID, datasetID string) bool
	AllowedDatasets  []string
	SessionProvider  bigqueryds.BigQuerySessionProvider
	manifest         tools.Manifest
	mcpManifest      tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

// Invoke runs the contribution analysis.
func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()
	inputData, ok := paramsMap["input_data"].(string)
	if !ok {
		return nil, fmt.Errorf("unable to cast input_data parameter %s", paramsMap["input_data"])
	}

	bqClient := t.Client
	restService := t.RestService
	var err error

	// Initialize new client if using user OAuth token
	if t.UseClientOAuth {
		tokenStr, err := accessToken.ParseBearerToken()
		if err != nil {
			return nil, fmt.Errorf("error parsing access token: %w", err)
		}
		bqClient, restService, err = t.ClientCreator(tokenStr, true)
		if err != nil {
			return nil, fmt.Errorf("error creating client from OAuth access token: %w", err)
		}
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
		if len(t.AllowedDatasets) > 0 {
			var connProps []*bigqueryapi.ConnectionProperty
			session, err := t.SessionProvider(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get BigQuery session: %w", err)
			}
			if session != nil {
				connProps = []*bigqueryapi.ConnectionProperty{
					{Key: "session_id", Value: session.ID},
				}
			}
			dryRunJob, err := bqutil.DryRunQuery(ctx, restService, t.Client.Project(), t.Client.Location, inputData, nil, connProps)
			if err != nil {
				return nil, fmt.Errorf("query validation failed: %w", err)
			}
			statementType := dryRunJob.Statistics.Query.StatementType
			if statementType != "SELECT" {
				return nil, fmt.Errorf("the 'input_data' parameter only supports a table ID or a SELECT query. The provided query has statement type '%s'", statementType)
			}

			queryStats := dryRunJob.Statistics.Query
			if queryStats != nil {
				for _, tableRef := range queryStats.ReferencedTables {
					if !t.IsDatasetAllowed(tableRef.ProjectId, tableRef.DatasetId) {
						return nil, fmt.Errorf("query in input_data accesses dataset '%s.%s', which is not in the allowed list", tableRef.ProjectId, tableRef.DatasetId)
					}
				}
			} else {
				return nil, fmt.Errorf("could not analyze query in input_data to validate against allowed datasets")
			}
		}
		inputDataSource = fmt.Sprintf("(%s)", inputData)
	} else {
		if len(t.AllowedDatasets) > 0 {
			parts := strings.Split(inputData, ".")
			var projectID, datasetID string
			switch len(parts) {
			case 3: // project.dataset.table
				projectID, datasetID = parts[0], parts[1]
			case 2: // dataset.table
				projectID, datasetID = t.Client.Project(), parts[0]
			default:
				return nil, fmt.Errorf("invalid table ID format for 'input_data': %q. Expected 'dataset.table' or 'project.dataset.table'", inputData)
			}
			if !t.IsDatasetAllowed(projectID, datasetID) {
				return nil, fmt.Errorf("access to dataset '%s.%s' (from table '%s') is not allowed", projectID, datasetID, inputData)
			}
		}
		inputDataSource = fmt.Sprintf("SELECT * FROM `%s`", inputData)
	}

	// Use temp model to skip the clean up at the end. To use TEMP MODEL, queries have to be
	// in the same BigQuery session.
	createModelSQL := fmt.Sprintf("CREATE TEMP MODEL %s OPTIONS(%s) AS %s",
		modelID,
		strings.Join(options, ", "),
		inputDataSource,
	)

	createModelQuery := bqClient.Query(createModelSQL)

	// Get session from provider if in protected mode.
	// Otherwise, a new session will be created by the first query.
	session, err := t.SessionProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get BigQuery session: %w", err)
	}

	if session != nil {
		createModelQuery.ConnectionProperties = []*bigqueryapi.ConnectionProperty{
			{Key: "session_id", Value: session.ID},
		}
	} else {
		// If not in protected mode, create a session for this invocation.
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

	// Determine the session ID to use for subsequent queries.
	// It's either from the pre-existing session (protected mode) or the one just created.
	var sessionID string
	if session != nil {
		sessionID = session.ID
	} else if status.Statistics != nil && status.Statistics.SessionInfo != nil {
		sessionID = status.Statistics.SessionInfo.SessionID
	} else {
		return nil, fmt.Errorf("failed to get or create a BigQuery session ID")
	}

	getInsightsSQL := fmt.Sprintf("SELECT * FROM ML.GET_INSIGHTS(MODEL %s)", modelID)

	getInsightsQuery := bqClient.Query(getInsightsSQL)
	getInsightsQuery.ConnectionProperties = []*bigqueryapi.ConnectionProperty{{Key: "session_id", Value: sessionID}}

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

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Parameters, data, claims)
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

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
