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

package bigqueryexecutesql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	bigqueryapi "cloud.google.com/go/bigquery"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	bigqueryds "github.com/googleapis/genai-toolbox/internal/sources/bigquery"
	"github.com/googleapis/genai-toolbox/internal/tools"
	bqutil "github.com/googleapis/genai-toolbox/internal/tools/bigquery/bigquerycommon"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/orderedmap"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	bigqueryrestapi "google.golang.org/api/bigquery/v2"
	"google.golang.org/api/iterator"
)

const kind string = "bigquery-execute-sql"

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
	BigQuerySession() bigqueryds.BigQuerySessionProvider
	BigQueryWriteMode() string
	BigQueryRestService() *bigqueryrestapi.Service
	BigQueryClientCreator() bigqueryds.BigqueryClientCreator
	UseClientAuthorization() bool
	IsDatasetAllowed(projectID, datasetID string) bool
	BigQueryAllowedDatasets() []string
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

	var sqlDescriptionBuilder strings.Builder
	switch s.BigQueryWriteMode() {
	case bigqueryds.WriteModeBlocked:
		sqlDescriptionBuilder.WriteString("The SQL to execute. In 'blocked' mode, only SELECT statements are allowed; other statement types will fail.")
	case bigqueryds.WriteModeProtected:
		sqlDescriptionBuilder.WriteString("The SQL to execute. Only SELECT statements and writes to the session's temporary dataset are allowed (e.g., `CREATE TEMP TABLE ...`).")
	default: // WriteModeAllowed
		sqlDescriptionBuilder.WriteString("The SQL to execute.")
	}

	allowedDatasets := s.BigQueryAllowedDatasets()
	if len(allowedDatasets) > 0 {
		if len(allowedDatasets) == 1 {
			datasetFQN := allowedDatasets[0]
			parts := strings.Split(datasetFQN, ".")
			if len(parts) < 2 {
				return nil, fmt.Errorf("expected allowedDataset to have at least 2 parts (project.dataset): %s", datasetFQN)
			}
			datasetID := parts[1]
			sqlDescriptionBuilder.WriteString(fmt.Sprintf(" The query must only access the `%s` dataset. "+
				"To query a table within this dataset (e.g., `my_table`), "+
				"qualify it with the dataset id (e.g., `%s.my_table`).", datasetFQN, datasetID))
		} else {
			datasetIDs := []string{}
			for _, ds := range allowedDatasets {
				datasetIDs = append(datasetIDs, fmt.Sprintf("`%s`", ds))
			}
			sqlDescriptionBuilder.WriteString(fmt.Sprintf(" The query must only access datasets from the following list: %s.", strings.Join(datasetIDs, ", ")))
		}
	}

	sqlParameter := parameters.NewStringParameter("sql", sqlDescriptionBuilder.String())
	dryRunParameter := parameters.NewBooleanParameterWithDefault(
		"dry_run",
		false,
		"If set to true, the query will be validated and information about the execution will be returned "+
			"without running the query. Defaults to false.",
	)
	params := parameters.Parameters{sqlParameter, dryRunParameter}
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, nil)

	// finish tool setup
	t := Tool{
		Config:           cfg,
		Parameters:       params,
		UseClientOAuth:   s.UseClientAuthorization(),
		ClientCreator:    s.BigQueryClientCreator(),
		Client:           s.BigQueryClient(),
		RestService:      s.BigQueryRestService(),
		WriteMode:        s.BigQueryWriteMode(),
		SessionProvider:  s.BigQuerySession(),
		IsDatasetAllowed: s.IsDatasetAllowed,
		AllowedDatasets:  allowedDatasets,
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
	WriteMode        string
	SessionProvider  bigqueryds.BigQuerySessionProvider
	ClientCreator    bigqueryds.BigqueryClientCreator
	IsDatasetAllowed func(projectID, datasetID string) bool
	AllowedDatasets  []string
	manifest         tools.Manifest
	mcpManifest      tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()
	sql, ok := paramsMap["sql"].(string)
	if !ok {
		return nil, fmt.Errorf("unable to cast sql parameter %s", paramsMap["sql"])
	}
	dryRun, ok := paramsMap["dry_run"].(bool)
	if !ok {
		return nil, fmt.Errorf("unable to cast dry_run parameter %s", paramsMap["dry_run"])
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

	var connProps []*bigqueryapi.ConnectionProperty
	var session *bigqueryds.Session
	if t.WriteMode == bigqueryds.WriteModeProtected {
		session, err = t.SessionProvider(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get BigQuery session for protected mode: %w", err)
		}
		connProps = []*bigqueryapi.ConnectionProperty{
			{Key: "session_id", Value: session.ID},
		}
	}

	dryRunJob, err := bqutil.DryRunQuery(ctx, restService, bqClient.Project(), bqClient.Location, sql, nil, connProps)
	if err != nil {
		return nil, fmt.Errorf("query validation failed: %w", err)
	}

	statementType := dryRunJob.Statistics.Query.StatementType

	switch t.WriteMode {
	case bigqueryds.WriteModeBlocked:
		if statementType != "SELECT" {
			return nil, fmt.Errorf("write mode is 'blocked', only SELECT statements are allowed")
		}
	case bigqueryds.WriteModeProtected:
		if dryRunJob.Configuration != nil && dryRunJob.Configuration.Query != nil {
			if dest := dryRunJob.Configuration.Query.DestinationTable; dest != nil && dest.DatasetId != session.DatasetID {
				return nil, fmt.Errorf("protected write mode only supports SELECT statements, or write operations in the anonymous "+
					"dataset of a BigQuery session, but destination was %q", dest.DatasetId)
			}
		}
	}

	if len(t.AllowedDatasets) > 0 {
		switch statementType {
		case "CREATE_SCHEMA", "DROP_SCHEMA", "ALTER_SCHEMA":
			return nil, fmt.Errorf("dataset-level operations like '%s' are not allowed when dataset restrictions are in place", statementType)
		case "CREATE_FUNCTION", "CREATE_TABLE_FUNCTION", "CREATE_PROCEDURE":
			return nil, fmt.Errorf("creating stored routines ('%s') is not allowed when dataset restrictions are in place, as their contents cannot be safely analyzed", statementType)
		case "CALL":
			return nil, fmt.Errorf("calling stored procedures ('%s') is not allowed when dataset restrictions are in place, as their contents cannot be safely analyzed", statementType)
		}

		// Use a map to avoid duplicate table names.
		tableIDSet := make(map[string]struct{})

		// Get all tables from the dry run result. This is the most reliable method.
		queryStats := dryRunJob.Statistics.Query
		if queryStats != nil {
			for _, tableRef := range queryStats.ReferencedTables {
				tableIDSet[fmt.Sprintf("%s.%s.%s", tableRef.ProjectId, tableRef.DatasetId, tableRef.TableId)] = struct{}{}
			}
			if tableRef := queryStats.DdlTargetTable; tableRef != nil {
				tableIDSet[fmt.Sprintf("%s.%s.%s", tableRef.ProjectId, tableRef.DatasetId, tableRef.TableId)] = struct{}{}
			}
			if tableRef := queryStats.DdlDestinationTable; tableRef != nil {
				tableIDSet[fmt.Sprintf("%s.%s.%s", tableRef.ProjectId, tableRef.DatasetId, tableRef.TableId)] = struct{}{}
			}
		}

		var tableNames []string
		if len(tableIDSet) > 0 {
			for tableID := range tableIDSet {
				tableNames = append(tableNames, tableID)
			}
		} else if statementType != "SELECT" {
			// If dry run yields no tables, fall back to the parser for non-SELECT statements
			// to catch unsafe operations like EXECUTE IMMEDIATE.
			parsedTables, parseErr := bqutil.TableParser(sql, t.Client.Project())
			if parseErr != nil {
				// If parsing fails (e.g., EXECUTE IMMEDIATE), we cannot guarantee safety, so we must fail.
				return nil, fmt.Errorf("could not parse tables from query to validate against allowed datasets: %w", parseErr)
			}
			tableNames = parsedTables
		}

		for _, tableID := range tableNames {
			parts := strings.Split(tableID, ".")
			if len(parts) == 3 {
				projectID, datasetID := parts[0], parts[1]
				if !t.IsDatasetAllowed(projectID, datasetID) {
					return nil, fmt.Errorf("query accesses dataset '%s.%s', which is not in the allowed list", projectID, datasetID)
				}
			}
		}
	}

	if dryRun {
		if dryRunJob != nil {
			jobJSON, err := json.MarshalIndent(dryRunJob, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal dry run job to JSON: %w", err)
			}
			return string(jobJSON), nil
		}
		// This case should not be reached, but as a fallback, we return a message.
		return "Dry run was requested, but no job information was returned.", nil
	}

	query := bqClient.Query(sql)
	query.Location = bqClient.Location

	query.ConnectionProperties = connProps

	// Log the query executed for debugging.
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting logger: %s", err)
	}
	logger.DebugContext(ctx, fmt.Sprintf("executing `%s` tool query: %s", kind, sql))

	// This block handles SELECT statements, which return a row set.
	// We iterate through the results, convert each row into a map of
	// column names to values, and return the collection of rows.
	var out []any
	job, err := query.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	it, err := job.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to read query results: %w", err)
	}
	for {
		var val []bigqueryapi.Value
		err = it.Next(&val)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("unable to iterate through query results: %w", err)
		}
		schema := it.Schema
		row := orderedmap.Row{}
		for i, field := range schema {
			row.Add(field.Name, val[i])
		}
		out = append(out, row)
	}
	// If the query returned any rows, return them directly.
	if len(out) > 0 {
		return out, nil
	}

	// This handles the standard case for a SELECT query that successfully
	// executes but returns zero rows.
	if statementType == "SELECT" {
		return "The query returned 0 rows.", nil
	}
	// This is the fallback for a successful query that doesn't return content.
	// In most cases, this will be for DML/DDL statements like INSERT, UPDATE, CREATE, etc.
	// However, it is also possible that this was a query that was expected to return rows
	// but returned none, a case that we cannot distinguish here.
	return "Query executed successfully and returned no content.", nil
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
