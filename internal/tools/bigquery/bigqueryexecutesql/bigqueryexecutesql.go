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
	bigqueryrestapi "google.golang.org/api/bigquery/v2"
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
	BigQueryRestService() *bigqueryrestapi.Service
	BigQueryClientCreator() bigqueryds.BigqueryClientCreator
	UseClientAuthorization() bool
	IsDatasetAllowed(projectID, datasetID string) bool
	BigQueryAllowedDatasets() []string
	RetrieveBQClient(tools.AccessToken) (*bigqueryapi.Client, *bigqueryrestapi.Service, error)
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

	sqlDescription := "The sql to execute."
	allowedDatasets := s.BigQueryAllowedDatasets()
	if len(allowedDatasets) > 0 {
		datasetIDs := []string{}
		for _, ds := range allowedDatasets {
			datasetIDs = append(datasetIDs, fmt.Sprintf("`%s`", ds))
		}

		if len(datasetIDs) == 1 {
			parts := strings.Split(allowedDatasets[0], ".")
			if len(parts) < 2 {
				return nil, fmt.Errorf("expected split to have 2 parts: %s", allowedDatasets[0])
			}
			datasetID := parts[1]
			sqlDescription += fmt.Sprintf(" The query must only access the %s dataset. "+
				"To query a table within this dataset (e.g., `my_table`), "+
				"qualify it with the dataset id (e.g., `%s.my_table`).", datasetIDs[0], datasetID)
		} else {
			sqlDescription += fmt.Sprintf(" The query must only access datasets from the following list: %s.", strings.Join(datasetIDs, ", "))
		}
	}
	sqlParameter := tools.NewStringParameter("sql", sqlDescription)
	dryRunParameter := tools.NewBooleanParameterWithDefault(
		"dry_run",
		false,
		"If set to true, the query will be validated and information about the execution "+
			"will be returned without running the query. Defaults to false.",
	)
	parameters := tools.Parameters{sqlParameter, dryRunParameter}
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, parameters)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		Parameters:  parameters,
		Source:      s,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	Parameters  tools.Parameters
	Source      compatibleSource
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	s := t.Source
	paramsMap := params.AsMap()
	sql, ok := paramsMap["sql"].(string)
	if !ok {
		return nil, fmt.Errorf("unable to cast sql parameter %s", paramsMap["sql"])
	}
	dryRun, ok := paramsMap["dry_run"].(bool)
	if !ok {
		return nil, fmt.Errorf("unable to cast dry_run parameter %s", paramsMap["dry_run"])
	}

	bqClient, restService, err := s.RetrieveBQClient(accessToken)
	if err != nil {
		return nil, err
	}

	dryRunJob, err := bqutil.DryRunQuery(ctx, restService, bqClient.Project(), bqClient.Location, sql, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("query validation failed during dry run: %w", err)
	}
	statementType := dryRunJob.Statistics.Query.StatementType
	if len(s.BigQueryAllowedDatasets()) > 0 {
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
			parsedTables, parseErr := bqutil.TableParser(sql, t.Source.BigQueryClient().Project())
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
				if !s.IsDatasetAllowed(projectID, datasetID) {
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

	return bqutil.RunQuery(ctx, sql, query)
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
	return t.Source.UseClientAuthorization()
}
