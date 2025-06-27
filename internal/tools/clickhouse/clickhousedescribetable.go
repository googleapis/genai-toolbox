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

package clickhouse

import (
	"context"
	"database/sql"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/clickhouse"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const describeTableKind string = "clickhouse-describe-table"

func init() {
	if !tools.Register(describeTableKind, newDescribeTableConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", describeTableKind))
	}
}

func newDescribeTableConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := DescribeTableConfig{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type compatibleSource interface {
	ClickHousePool() *sql.DB
}

// validate compatible sources are still compatible
var _ compatibleSource = &clickhouse.Source{}

var compatibleSources = [...]string{clickhouse.SourceKind}

type DescribeTableConfig struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = DescribeTableConfig{}

func (cfg DescribeTableConfig) ToolConfigKind() string {
	return describeTableKind
}

func (cfg DescribeTableConfig) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", describeTableKind, compatibleSources)
	}

	tableParameter := tools.NewStringParameter("table_name", "The table name to describe.")
	parameters := tools.Parameters{tableParameter}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	// finish tool setup
	t := DescribeTableTool{
		Name:         cfg.Name,
		Kind:         describeTableKind,
		Parameters:   parameters,
		AuthRequired: cfg.AuthRequired,
		Pool:         s.ClickHousePool(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = DescribeTableTool{}

type DescribeTableTool struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`

	Pool        *sql.DB
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t DescribeTableTool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	sliceParams := params.AsSlice()
	table, ok := sliceParams[0].(string)
	if !ok {
		return nil, fmt.Errorf("unable to cast table_name parameter %v", sliceParams[0])
	}

	// Get table info first, then columns separately due to ClickHouse limitations
	tableQuery := `
	SELECT
		database,
		name as table_name,
		engine,
		primary_key,
		sorting_key,
		partition_key,
		total_rows,
		total_bytes,
		comment
	FROM system.tables
	WHERE name = ? AND database = currentDatabase()
	`
	
	columnsQuery := `
	SELECT
		name as column_name,
		type as data_type,
		position as ordinal_position,
		default_expression as column_default,
		comment as column_comment,
		is_in_primary_key,
		is_in_sorting_key,
		is_in_partition_key
	FROM system.columns
	WHERE table = ? AND database = currentDatabase()
	ORDER BY position
	`

	// Get table information
	tableRows, err := t.Pool.QueryContext(ctx, tableQuery, table)
	if err != nil {
		return nil, fmt.Errorf("unable to execute table query: %w", err)
	}
	defer tableRows.Close()

	var tableInfo map[string]any
	if tableRows.Next() {
		var dbName, tableName, engine sql.NullString
		var primaryKey, sortingKey, partitionKey, comment sql.NullString
		var totalRows, totalBytes sql.NullInt64
		
		err := tableRows.Scan(&dbName, &tableName, &engine, &primaryKey, &sortingKey, &partitionKey, &totalRows, &totalBytes, &comment)
		if err != nil {
			return nil, fmt.Errorf("unable to scan table info: %w", err)
		}
		
		tableInfo = map[string]any{
			"database":      nullStringValue(dbName),
			"table_name":    nullStringValue(tableName),
			"engine":        nullStringValue(engine),
			"primary_key":   nullStringValue(primaryKey),
			"sorting_key":   nullStringValue(sortingKey),
			"partition_key": nullStringValue(partitionKey),
			"total_rows":    nullInt64Value(totalRows),
			"total_bytes":   nullInt64Value(totalBytes),
			"comment":       nullStringValue(comment),
		}
	} else {
		return nil, fmt.Errorf("table %s not found", table)
	}
	
	if err := tableRows.Err(); err != nil {
		return nil, fmt.Errorf("errors encountered during table query: %w", err)
	}

	// Get column information
	columnRows, err := t.Pool.QueryContext(ctx, columnsQuery, table)
	if err != nil {
		return nil, fmt.Errorf("unable to execute columns query: %w", err)
	}
	defer columnRows.Close()

	var columns []map[string]any
	for columnRows.Next() {
		var columnName, dataType sql.NullString
		var columnDefault, columnComment sql.NullString
		var ordinalPosition sql.NullInt64
		var isInPrimaryKey, isInSortingKey, isInPartitionKey uint8
		
		err := columnRows.Scan(&columnName, &dataType, &ordinalPosition, &columnDefault, &columnComment, &isInPrimaryKey, &isInSortingKey, &isInPartitionKey)
		if err != nil {
			return nil, fmt.Errorf("unable to scan column info: %w", err)
		}
		
		column := map[string]any{
			"column_name":          nullStringValue(columnName),
			"data_type":           nullStringValue(dataType),
			"ordinal_position":    nullInt64Value(ordinalPosition),
			"column_default":      nullStringValue(columnDefault),
			"column_comment":      nullStringValue(columnComment),
			"is_in_primary_key":   isInPrimaryKey == 1,
			"is_in_sorting_key":   isInSortingKey == 1,
			"is_in_partition_key": isInPartitionKey == 1,
		}
		columns = append(columns, column)
	}
	
	if err := columnRows.Err(); err != nil {
		return nil, fmt.Errorf("errors encountered during columns query: %w", err)
	}

	// Return each column as a separate dictionary, including table metadata in each row
	// This matches the standard database result format where each row is a dictionary
	var result []any
	for _, column := range columns {
		columnDict := column // column is already map[string]any
		// Add table-level information to each column record
		columnDict["table_database"] = tableInfo["database"]
		columnDict["table_name"] = tableInfo["table_name"]
		columnDict["table_engine"] = tableInfo["engine"]
		columnDict["table_primary_key"] = tableInfo["primary_key"]
		columnDict["table_sorting_key"] = tableInfo["sorting_key"]
		columnDict["table_partition_key"] = tableInfo["partition_key"]
		columnDict["table_total_rows"] = tableInfo["total_rows"]
		columnDict["table_total_bytes"] = tableInfo["total_bytes"]
		columnDict["table_comment"] = tableInfo["comment"]
		
		result = append(result, columnDict)
	}
	
	return result, nil
}

func (t DescribeTableTool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data, claims)
}

func (t DescribeTableTool) Manifest() tools.Manifest {
	return t.manifest
}

func (t DescribeTableTool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t DescribeTableTool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

// Helper functions for handling NULL values
func nullStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func nullInt64Value(ni sql.NullInt64) int64 {
	if ni.Valid {
		return ni.Int64
	}
	return 0
}