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

package bigquerycommon

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	bigqueryapi "cloud.google.com/go/bigquery"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	bigqueryrestapi "google.golang.org/api/bigquery/v2"
)

// validBQTableID matches BigQuery table identifiers in 'dataset.table' or
// 'project.dataset.table' form. Components are restricted to letters, digits,
// and underscores — the character set that BigQuery allows for dataset and
// table IDs and that is safe to interpolate inside a backtick-quoted SQL
// identifier.
var validBQTableID = regexp.MustCompile(`^[a-zA-Z0-9_-]+(\.([a-zA-Z0-9_]+)){1,2}$`)

// validBQColumnName matches BigQuery column names: a letter or underscore
// followed by letters, digits, or underscores.
var validBQColumnName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ValidTableID returns true if s is a safe BigQuery table identifier of the
// form 'dataset.table' or 'project.dataset.table'. Values that fail this check
// must not be interpolated into backtick-quoted SQL.
func ValidTableID(s string) bool {
	return validBQTableID.MatchString(s)
}

// ValidColumnParam returns true if s (stripped of leading/trailing single quotes) is a safe column name.
func ValidColumnParam(s string) bool {
	return ValidColumnName(StripSingleQuotes(s))
}

// ValidContributionMetricParam returns true if s (stripped of leading/trailing single quotes) is a safe contribution metric (does not contain single quotes).
func ValidContributionMetricParam(s string) bool {
	return !strings.ContainsRune(StripSingleQuotes(s), '\'')
}

// StripSingleQuotes removes leading and trailing single quotes from a string if both are present.
func StripSingleQuotes(s string) string {
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}

// ValidColumnName returns true if s is a safe BigQuery column name.
// Values that fail this check must not be interpolated as SQL identifiers
// or into single-quoted SQL string arguments that represent column references.
func ValidColumnName(s string) bool {
	return validBQColumnName.MatchString(s)
}

// DryRunQuery performs a dry run of the SQL query to validate it and get metadata.
func DryRunQuery(
	ctx context.Context,
	restService *bigqueryrestapi.Service,
	projectID string,
	location string,
	sql string,
	params []*bigqueryrestapi.QueryParameter,
	connProps []*bigqueryapi.ConnectionProperty,
	maximumBytesBilled int64,
	createSession bool,
) (*bigqueryrestapi.Job, error) {
	useLegacySql := false
	restConnProps := make([]*bigqueryrestapi.ConnectionProperty, len(connProps))
	for i, prop := range connProps {
		restConnProps[i] = &bigqueryrestapi.ConnectionProperty{Key: prop.Key, Value: prop.Value}
	}

	jobToInsert := &bigqueryrestapi.Job{
		JobReference: &bigqueryrestapi.JobReference{
			ProjectId: projectID,
			Location:  location,
		},
		Configuration: &bigqueryrestapi.JobConfiguration{
			DryRun: true,
			Query: &bigqueryrestapi.JobConfigurationQuery{
				Query:                sql,
				UseLegacySql:         &useLegacySql,
				ConnectionProperties: restConnProps,
				QueryParameters:      params,
				MaximumBytesBilled:   maximumBytesBilled,
				CreateSession:        createSession,
			},
		},
	}

	insertResponse, err := restService.Jobs.Insert(projectID, jobToInsert).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to insert dry run job: %w", err)
	}
	return insertResponse, nil
}

// DatasetValidator defines the interface for checking if a dataset is allowed.
type DatasetValidator interface {
	IsDatasetAllowed(projectID, datasetID string) bool
}

// ValidateQueryAgainstAllowedDatasets validates a SQL query against a list of allowed datasets.
// It uses both dry run and a local parser to support authorized views.
func ValidateQueryAgainstAllowedDatasets(
	ctx context.Context,
	restService *bigqueryrestapi.Service,
	projectID string,
	location string,
	sql string,
	params []*bigqueryrestapi.QueryParameter,
	connProps []*bigqueryapi.ConnectionProperty,
	validator DatasetValidator,
	maximumBytesBilled int64,
	createSession bool,
) (*bigqueryrestapi.Job, error) {
	dryRunJob, err := DryRunQuery(ctx, restService, projectID, location, sql, params, connProps, maximumBytesBilled, createSession)
	if err != nil {
		return nil, fmt.Errorf("query validation failed: %w", err)
	}

	if dryRunJob.Statistics == nil || dryRunJob.Statistics.Query == nil {
		return nil, fmt.Errorf("dry run failed to return query statistics")
	}

	// Use a map to avoid duplicate table names from the dry run result.
	tableIDSet := make(map[string]struct{})
	queryStats := dryRunJob.Statistics.Query
	if queryStats != nil {
		for _, tableRef := range queryStats.ReferencedTables {
			if tableRef != nil {
				tableIDSet[fmt.Sprintf("%s.%s.%s", tableRef.ProjectId, tableRef.DatasetId, tableRef.TableId)] = struct{}{}
			}
		}
		if tableRef := queryStats.DdlTargetTable; tableRef != nil {
			tableIDSet[fmt.Sprintf("%s.%s.%s", tableRef.ProjectId, tableRef.DatasetId, tableRef.TableId)] = struct{}{}
		}
		if tableRef := queryStats.DdlDestinationTable; tableRef != nil {
			tableIDSet[fmt.Sprintf("%s.%s.%s", tableRef.ProjectId, tableRef.DatasetId, tableRef.TableId)] = struct{}{}
		}
	}

	var violatingTables []string
	for tableID := range tableIDSet {
		parts := strings.Split(tableID, ".")
		if len(parts) == 3 {
			// Skip validation for specific system functions that BigQuery reports as referenced tables.
			if IsSystemResource(parts[1], parts[2]) {
				continue
			}
			if !validator.IsDatasetAllowed(parts[0], parts[1]) {
				violatingTables = append(violatingTables, tableID)
			}
		}
	}


	// If violations were found, check if they are explicitly in the SQL to support authorized views.
	if len(violatingTables) > 0 {
		explicitlyReferenced, err := IsAnyTableExplicitlyReferenced(sql, projectID, violatingTables)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze query for explicit table references: %w", err)
		}
		if explicitlyReferenced {
			violatingDatasets := []string{}
			seenDatasets := make(map[string]struct{})
			for _, tableID := range violatingTables {
				datasetFQN := strings.Join(strings.Split(tableID, ".")[:2], ".")
				if _, seen := seenDatasets[datasetFQN]; !seen {
					violatingDatasets = append(violatingDatasets, fmt.Sprintf("'%s'", datasetFQN))
					seenDatasets[datasetFQN] = struct{}{}
				}
			}
			plural := ""
			if len(violatingDatasets) > 1 {
				plural = "s"
			}
			return nil, fmt.Errorf("access to dataset%s %s is not allowed", plural, strings.Join(violatingDatasets, ", "))
		}
	}

	// Fall back to TableParser for final intent verification or if dry run was inconclusive.
	parsedTables, parseErr := TableParser(sql, projectID)
	if parseErr != nil {
		return nil, fmt.Errorf("could not safely analyze query with dataset restrictions: %w", parseErr)
	}

	var parsedViolatingDatasets []string
	seenParsedDatasets := make(map[string]struct{})
	for _, tableID := range parsedTables {
		parts := strings.Split(tableID, ".")
		if len(parts) == 3 {
			if IsSystemResource(parts[1], parts[2]) {
				continue
			}
			if !validator.IsDatasetAllowed(parts[0], parts[1]) {
				datasetFQN := fmt.Sprintf("%s.%s", parts[0], parts[1])
				if _, seen := seenParsedDatasets[datasetFQN]; !seen {
					parsedViolatingDatasets = append(parsedViolatingDatasets, fmt.Sprintf("'%s'", datasetFQN))
					seenParsedDatasets[datasetFQN] = struct{}{}
				}
			}
		}
	}
	if len(parsedViolatingDatasets) > 0 {
		plural := ""
		if len(parsedViolatingDatasets) > 1 {
			plural = "s"
		}
		return nil, fmt.Errorf("access to dataset%s %s is not allowed", plural, strings.Join(parsedViolatingDatasets, ", "))
	}

	return dryRunJob, nil
}

// IsSystemResource checks if a given dataset and table/function ID refer to a BigQuery system resource
// (like a built-in AI or ML function) that should be exempted from dataset restriction checks.
func IsSystemResource(datasetID, resourceID string) bool {
	datasetID = strings.ToUpper(datasetID)
	resourceID = strings.ToUpper(resourceID)

	if datasetID == "AI" {
		switch resourceID {
		case "FORECAST", "GENERATE_TEXT", "EXTRACT_ENTITY", "SUMMARIZE":
			return true
		}
	}
	if datasetID == "ML" {
		switch resourceID {
		case "GET_INSIGHTS", "EXPLAIN_PREDICT", "GENERATE_TEXT", "DISTANCE", "PREDICT":
			return true
		}
	}
	return false
}

// BQTypeStringFromToolType converts a tool parameter type string to a BigQuery standard SQL type string.
func BQTypeStringFromToolType(toolType string) (string, error) {
	switch toolType {
	case parameters.TypeString:
		return "STRING", nil
	case parameters.TypeInt:
		return "INT64", nil
	case parameters.TypeFloat:
		return "FLOAT64", nil
	case parameters.TypeBool:
		return "BOOL", nil
	case parameters.TypeMap:
		return "STRUCT", nil
	default:
		return "", fmt.Errorf("unsupported tool parameter type for BigQuery: %s", toolType)
	}
}

// InitializeDatasetParameters generates project and dataset tool parameters based on allowedDatasets.
func InitializeDatasetParameters(
	allowedDatasets []string,
	defaultProjectID string,
	projectKey, datasetKey string,
	projectDescription, datasetDescription string,
) (projectParam, datasetParam parameters.Parameter) {
	if len(allowedDatasets) > 0 {
		if len(allowedDatasets) == 1 {
			parts := strings.Split(allowedDatasets[0], ".")
			defaultProjectID = parts[0]
			datasetID := parts[1]
			projectDescription += fmt.Sprintf(" Must be `%s`.", defaultProjectID)
			datasetDescription += fmt.Sprintf(" Must be `%s`.", datasetID)
			datasetParam = parameters.NewStringParameter(datasetKey, datasetDescription, parameters.WithStringDefault(datasetID))
		} else {
			datasetIDsByProject := make(map[string][]string)
			for _, ds := range allowedDatasets {
				parts := strings.Split(ds, ".")
				project := parts[0]
				dataset := parts[1]
				datasetIDsByProject[project] = append(datasetIDsByProject[project], fmt.Sprintf("`%s`", dataset))
			}

			var datasetDescriptions, projectIDList []string
			for project, datasets := range datasetIDsByProject {
				sort.Strings(datasets)
				projectIDList = append(projectIDList, fmt.Sprintf("`%s`", project))
				datasetList := strings.Join(datasets, ", ")
				datasetDescriptions = append(datasetDescriptions, fmt.Sprintf("%s from project `%s`", datasetList, project))
			}
			sort.Strings(projectIDList)
			sort.Strings(datasetDescriptions)
			projectDescription += fmt.Sprintf(" Must be one of the following: %s.", strings.Join(projectIDList, ", "))
			datasetDescription += fmt.Sprintf(" Must be one of the allowed datasets: %s.", strings.Join(datasetDescriptions, "; "))
			datasetParam = parameters.NewStringParameter(datasetKey, datasetDescription)
		}
	} else {
		datasetParam = parameters.NewStringParameter(datasetKey, datasetDescription)
	}

	projectParam = parameters.NewStringParameter(projectKey, projectDescription, parameters.WithStringDefault(defaultProjectID))

	return projectParam, datasetParam
}

// ProcessGcpError converts a Google API error into a user-friendly error.
func ProcessGcpError(err error) util.ToolboxError {
	return util.ProcessGcpError(err)
}

// NewAgentError returns a new AgentError.
func NewAgentError(message string, err error) util.ToolboxError {
	return util.NewAgentError(message, err)
}

// NewClientServerError returns a new ClientServerError.
func NewClientServerError(message string, statusCode int, err error) util.ToolboxError {
	return util.NewClientServerError(message, statusCode, err)
}
