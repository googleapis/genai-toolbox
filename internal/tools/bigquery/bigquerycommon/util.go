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
	"math/big"
	"reflect"
	"sort"
	"strings"

	bigqueryapi "cloud.google.com/go/bigquery"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	bigqueryrestapi "google.golang.org/api/bigquery/v2"
)

// DryRunQuery performs a dry run of the SQL query to validate it and get metadata.
func DryRunQuery(ctx context.Context, restService *bigqueryrestapi.Service, projectID string, location string, sql string, params []*bigqueryrestapi.QueryParameter, connProps []*bigqueryapi.ConnectionProperty) (*bigqueryrestapi.Job, error) {
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
			},
		},
	}

	insertResponse, err := restService.Jobs.Insert(projectID, jobToInsert).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to insert dry run job: %w", err)
	}
	return insertResponse, nil
}

// BQTypeStringFromToolType converts a tool parameter type string to a BigQuery standard SQL type string.
func BQTypeStringFromToolType(toolType string) (string, error) {
	switch toolType {
	case "string":
		return "STRING", nil
	case "integer":
		return "INT64", nil
	case "float":
		return "FLOAT64", nil
	case "boolean":
		return "BOOL", nil
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
			datasetParam = parameters.NewStringParameterWithDefault(datasetKey, datasetID, datasetDescription)
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

	projectParam = parameters.NewStringParameterWithDefault(projectKey, defaultProjectID, projectDescription)

	return projectParam, datasetParam
}

// NormalizeValue converts BigQuery specific types to standard JSON-compatible types.
// Specifically, it handles *big.Rat (used for NUMERIC/BIGNUMERIC) by converting
// them to decimal strings with up to 38 digits of precision, trimming trailing zeros.
// It recursively handles slices (arrays) and maps (structs) using reflection.
func NormalizeValue(v any) any {
	if v == nil {
		return nil
	}

	// Handle *big.Rat specifically.
	if rat, ok := v.(*big.Rat); ok {
		// Convert big.Rat to a decimal string.
		// Use a precision of 38 digits (enough for BIGNUMERIC and NUMERIC)
		// and trim trailing zeros to match BigQuery's behavior.
		s := rat.FloatString(38)
		if strings.Contains(s, ".") {
			s = strings.TrimRight(s, "0")
			s = strings.TrimRight(s, ".")
		}
		return s
	}

	// Use reflection for slices and maps to handle various underlying types.
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		// Preserve []byte as is, so json.Marshal encodes it as Base64 string (BigQuery BYTES behavior).
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return v
		}
		newSlice := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			newSlice[i] = NormalizeValue(rv.Index(i).Interface())
		}
		return newSlice
	case reflect.Map:
		// Ensure keys are strings to produce a JSON-compatible map.
		if rv.Type().Key().Kind() != reflect.String {
			return v
		}
		newMap := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			newMap[iter.Key().String()] = NormalizeValue(iter.Value().Interface())
		}
		return newMap
	}

	return v
}
