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

package trinotablestatistics

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/trino"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
)

const kind string = "trino-table-statistics"

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
	TrinoDB() *sql.DB
}

// validate compatible sources are still compatible
var _ compatibleSource = &trino.Source{}

var compatibleSources = [...]string{trino.SourceKind}

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

	// Define parameters for the tool
	parameters := tools.Parameters{
		tools.NewStringParameter("table_name", "The fully qualified table name (catalog.schema.table) or just table name to get statistics for."),
		tools.NewStringParameter("catalog", "Optional: The catalog name. If not provided, uses the current catalog."),
		tools.NewStringParameter("schema", "Optional: The schema name. If not provided, uses the current schema."),
		tools.NewBooleanParameterWithDefault("include_columns", true, "If true, includes detailed column statistics. Default is true."),
		tools.NewBooleanParameterWithDefault("include_partitions", false, "If true, includes partition information if the table is partitioned. Default is false."),
		tools.NewBooleanParameterWithDefault("analyze_table", false, "If true, runs ANALYZE on the table to update statistics before retrieving them. Default is false."),
	}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	// finish tool setup
	t := Tool{
		Name:         cfg.Name,
		Kind:         kind,
		Parameters:   parameters,
		AuthRequired: cfg.AuthRequired,
		Db:           s.TrinoDB(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

// TableStatistics represents comprehensive table statistics
type TableStatistics struct {
	TableName        string             `json:"tableName"`
	CatalogName      string             `json:"catalogName"`
	SchemaName       string             `json:"schemaName"`
	TableType        string             `json:"tableType,omitempty"`
	RowCount         *int64             `json:"rowCount,omitempty"`
	DataSizeBytes    *int64             `json:"dataSizeBytes,omitempty"`
	DataSizeMB       *float64           `json:"dataSizeMB,omitempty"`
	LastAnalyzedTime *string            `json:"lastAnalyzedTime,omitempty"`
	TableProperties  map[string]string  `json:"tableProperties,omitempty"`
	ColumnStatistics []ColumnStatistics `json:"columnStatistics,omitempty"`
	PartitionInfo    *PartitionInfo     `json:"partitionInfo,omitempty"`
	StorageInfo      *StorageInfo       `json:"storageInfo,omitempty"`
	AccessInfo       *AccessInfo        `json:"accessInfo,omitempty"`
	Errors           []string           `json:"errors,omitempty"`
}

// ColumnStatistics represents statistics for a single column
type ColumnStatistics struct {
	ColumnName    string   `json:"columnName"`
	DataType      string   `json:"dataType"`
	NullCount     *int64   `json:"nullCount,omitempty"`
	DistinctCount *int64   `json:"distinctCount,omitempty"`
	MinValue      *string  `json:"minValue,omitempty"`
	MaxValue      *string  `json:"maxValue,omitempty"`
	AvgLength     *float64 `json:"avgLength,omitempty"`
	MaxLength     *int64   `json:"maxLength,omitempty"`
	NumTrues      *int64   `json:"numTrues,omitempty"`
	NumFalses     *int64   `json:"numFalses,omitempty"`
	DataSizeBytes *int64   `json:"dataSizeBytes,omitempty"`
}

// PartitionInfo contains partition-related information
type PartitionInfo struct {
	IsPartitioned    bool              `json:"isPartitioned"`
	PartitionColumns []string          `json:"partitionColumns,omitempty"`
	PartitionCount   int               `json:"partitionCount,omitempty"`
	Partitions       []PartitionDetail `json:"partitions,omitempty"`
}

// PartitionDetail represents details of a single partition
type PartitionDetail struct {
	PartitionValues map[string]string `json:"partitionValues"`
	RowCount        *int64            `json:"rowCount,omitempty"`
	DataSizeBytes   *int64            `json:"dataSizeBytes,omitempty"`
}

// StorageInfo contains storage-related information
type StorageInfo struct {
	Location       string `json:"location,omitempty"`
	InputFormat    string `json:"inputFormat,omitempty"`
	OutputFormat   string `json:"outputFormat,omitempty"`
	SerdeLib       string `json:"serdeLib,omitempty"`
	Compressed     bool   `json:"compressed"`
	NumFiles       *int64 `json:"numFiles,omitempty"`
	TotalSizeBytes *int64 `json:"totalSizeBytes,omitempty"`
}

// AccessInfo contains access-related information
type AccessInfo struct {
	Owner            string `json:"owner,omitempty"`
	CreatedTime      string `json:"createdTime,omitempty"`
	LastModifiedTime string `json:"lastModifiedTime,omitempty"`
	LastAccessTime   string `json:"lastAccessTime,omitempty"`
	AccessCount      *int64 `json:"accessCount,omitempty"`
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`

	Db          *sql.DB
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	tableName, ok := paramsMap["table_name"].(string)
	if !ok || tableName == "" {
		return nil, fmt.Errorf("'table_name' parameter is required and must be a non-empty string")
	}

	catalog, _ := paramsMap["catalog"].(string)
	schema, _ := paramsMap["schema"].(string)
	includeColumns, _ := paramsMap["include_columns"].(bool)
	if _, ok := paramsMap["include_columns"]; !ok {
		includeColumns = true // Default to true
	}
	includePartitions, _ := paramsMap["include_partitions"].(bool)
	analyzeTable, _ := paramsMap["analyze_table"].(bool)

	// Parse the table name to extract catalog, schema, and table parts
	catalogName, schemaName, actualTableName := t.parseTableName(tableName, catalog, schema)

	// Log the statistics request
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting logger: %s", err)
	}
	logger.DebugContext(ctx, "getting statistics for table %s.%s.%s", catalogName, schemaName, actualTableName)

	// Run ANALYZE if requested
	if analyzeTable {
		if err := t.analyzeTable(ctx, catalogName, schemaName, actualTableName); err != nil {
			logger.WarnContext(ctx, "failed to analyze table: %v", err)
		}
	}

	// Get table statistics
	stats, err := t.getTableStatistics(ctx, catalogName, schemaName, actualTableName, includeColumns, includePartitions)
	if err != nil {
		return nil, fmt.Errorf("failed to get table statistics: %w", err)
	}

	return stats, nil
}

func (t Tool) parseTableName(tableName, catalog, schema string) (string, string, string) {
	parts := strings.Split(tableName, ".")

	var catalogName, schemaName, actualTableName string

	switch len(parts) {
	case 3:
		// Fully qualified: catalog.schema.table
		catalogName = parts[0]
		schemaName = parts[1]
		actualTableName = parts[2]
	case 2:
		// Schema qualified: schema.table
		catalogName = catalog
		schemaName = parts[0]
		actualTableName = parts[1]
	case 1:
		// Just table name
		catalogName = catalog
		schemaName = schema
		actualTableName = parts[0]
	default:
		// Invalid format, use as-is
		catalogName = catalog
		schemaName = schema
		actualTableName = tableName
	}

	// Use CURRENT_CATALOG and CURRENT_SCHEMA if not specified
	if catalogName == "" {
		catalogName = "CURRENT_CATALOG"
	}
	if schemaName == "" {
		schemaName = "CURRENT_SCHEMA"
	}

	return catalogName, schemaName, actualTableName
}

func (t Tool) analyzeTable(ctx context.Context, catalog, schema, table string) error {
	var query string
	if catalog == "CURRENT_CATALOG" && schema == "CURRENT_SCHEMA" {
		query = fmt.Sprintf("ANALYZE %s", table)
	} else if catalog == "CURRENT_CATALOG" {
		query = fmt.Sprintf("ANALYZE %s.%s", schema, table)
	} else {
		query = fmt.Sprintf("ANALYZE %s.%s.%s", catalog, schema, table)
	}

	_, err := t.Db.ExecContext(ctx, query)
	return err
}

func (t Tool) getTableStatistics(ctx context.Context, catalog, schema, table string, includeColumns, includePartitions bool) (*TableStatistics, error) {
	stats := &TableStatistics{
		TableName:   table,
		CatalogName: catalog,
		SchemaName:  schema,
		Errors:      []string{},
	}

	// Get basic table information
	if err := t.getBasicTableInfo(ctx, stats, catalog, schema, table); err != nil {
		stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to get basic table info: %v", err))
	}

	// Get table properties
	if props, err := t.getTableProperties(ctx, catalog, schema, table); err == nil {
		stats.TableProperties = props
	} else {
		stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to get table properties: %v", err))
	}

	// Get row count and data size
	if err := t.getTableSizeStats(ctx, stats, catalog, schema, table); err != nil {
		stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to get table size: %v", err))
	}

	// Get column statistics if requested
	if includeColumns {
		if colStats, err := t.getColumnStatistics(ctx, catalog, schema, table); err == nil {
			stats.ColumnStatistics = colStats
		} else {
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to get column statistics: %v", err))
		}
	}

	// Get partition information if requested
	if includePartitions {
		if partInfo, err := t.getPartitionInfo(ctx, catalog, schema, table); err == nil {
			stats.PartitionInfo = partInfo
		} else {
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to get partition info: %v", err))
		}
	}

	// Get storage information
	if storageInfo, err := t.getStorageInfo(ctx, catalog, schema, table); err == nil {
		stats.StorageInfo = storageInfo
	}

	return stats, nil
}

func (t Tool) getBasicTableInfo(ctx context.Context, stats *TableStatistics, catalog, schema, table string) error {
	var query string
	if catalog == "CURRENT_CATALOG" && schema == "CURRENT_SCHEMA" {
		query = `
			SELECT table_type
			FROM information_schema.tables
			WHERE table_catalog = CURRENT_CATALOG 
				AND table_schema = CURRENT_SCHEMA
				AND table_name = ?`
		rows, err := t.Db.QueryContext(ctx, query, table)
		if err != nil {
			return err
		}
		defer rows.Close()

		if rows.Next() {
			if err := rows.Scan(&stats.TableType); err != nil {
				return err
			}
		}
	} else {
		query = `
			SELECT table_type
			FROM information_schema.tables
			WHERE table_catalog = ? 
				AND table_schema = ?
				AND table_name = ?`
		rows, err := t.Db.QueryContext(ctx, query, catalog, schema, table)
		if err != nil {
			return err
		}
		defer rows.Close()

		if rows.Next() {
			if err := rows.Scan(&stats.TableType); err != nil {
				return err
			}
		}
	}

	// Update actual catalog and schema names if they were CURRENT_*
	if catalog == "CURRENT_CATALOG" || schema == "CURRENT_SCHEMA" {
		actualQuery := `SELECT CURRENT_CATALOG, CURRENT_SCHEMA`
		rows, err := t.Db.QueryContext(ctx, actualQuery)
		if err == nil {
			defer rows.Close()
			if rows.Next() {
				var actualCatalog, actualSchema string
				if err := rows.Scan(&actualCatalog, &actualSchema); err == nil {
					if catalog == "CURRENT_CATALOG" {
						stats.CatalogName = actualCatalog
					}
					if schema == "CURRENT_SCHEMA" {
						stats.SchemaName = actualSchema
					}
				}
			}
		}
	}

	return nil
}

func (t Tool) getTableProperties(ctx context.Context, catalog, schema, table string) (map[string]string, error) {
	// Try to get table properties using SHOW CREATE TABLE
	var query string
	if catalog == "CURRENT_CATALOG" && schema == "CURRENT_SCHEMA" {
		query = fmt.Sprintf("SHOW CREATE TABLE %s", table)
	} else if catalog == "CURRENT_CATALOG" {
		query = fmt.Sprintf("SHOW CREATE TABLE %s.%s", schema, table)
	} else {
		query = fmt.Sprintf("SHOW CREATE TABLE %s.%s.%s", catalog, schema, table)
	}

	rows, err := t.Db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	properties := make(map[string]string)
	// Parse CREATE TABLE statement for properties
	// This is a simplified version - real implementation would need more sophisticated parsing

	return properties, nil
}

func (t Tool) getTableSizeStats(ctx context.Context, stats *TableStatistics, catalog, schema, table string) error {
	// Try to get row count
	var countQuery string
	if catalog == "CURRENT_CATALOG" && schema == "CURRENT_SCHEMA" {
		countQuery = fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	} else if catalog == "CURRENT_CATALOG" {
		countQuery = fmt.Sprintf("SELECT COUNT(*) FROM %s.%s", schema, table)
	} else {
		countQuery = fmt.Sprintf("SELECT COUNT(*) FROM %s.%s.%s", catalog, schema, table)
	}

	rows, err := t.Db.QueryContext(ctx, countQuery)
	if err == nil {
		defer rows.Close()
		if rows.Next() {
			var count int64
			if err := rows.Scan(&count); err == nil {
				stats.RowCount = &count
			}
		}
	}

	return nil
}

func (t Tool) getColumnStatistics(ctx context.Context, catalog, schema, table string) ([]ColumnStatistics, error) {
	// First get column information
	var query string
	if catalog == "CURRENT_CATALOG" && schema == "CURRENT_SCHEMA" {
		query = `
			SELECT column_name, data_type
			FROM information_schema.columns
			WHERE table_catalog = CURRENT_CATALOG
				AND table_schema = CURRENT_SCHEMA
				AND table_name = ?
			ORDER BY ordinal_position`
		rows, err := t.Db.QueryContext(ctx, query, table)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var colStats []ColumnStatistics
		for rows.Next() {
			var col ColumnStatistics
			if err := rows.Scan(&col.ColumnName, &col.DataType); err != nil {
				continue
			}

			// Get additional statistics for each column using SHOW STATS
			if err := t.getColumnDetailedStats(ctx, &col, catalog, schema, table); err == nil {
				colStats = append(colStats, col)
			} else {
				colStats = append(colStats, col)
			}
		}

		return colStats, rows.Err()
	} else {
		query = `
			SELECT column_name, data_type
			FROM information_schema.columns
			WHERE table_catalog = ?
				AND table_schema = ?
				AND table_name = ?
			ORDER BY ordinal_position`
		rows, err := t.Db.QueryContext(ctx, query, catalog, schema, table)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var colStats []ColumnStatistics
		for rows.Next() {
			var col ColumnStatistics
			if err := rows.Scan(&col.ColumnName, &col.DataType); err != nil {
				continue
			}

			// Get additional statistics for each column
			if err := t.getColumnDetailedStats(ctx, &col, catalog, schema, table); err == nil {
				colStats = append(colStats, col)
			} else {
				colStats = append(colStats, col)
			}
		}

		return colStats, rows.Err()
	}
}

func (t Tool) getColumnDetailedStats(ctx context.Context, col *ColumnStatistics, catalog, schema, table string) error {
	// Use SHOW STATS to get column statistics
	var query string
	if catalog == "CURRENT_CATALOG" && schema == "CURRENT_SCHEMA" {
		query = fmt.Sprintf("SHOW STATS FOR %s", table)
	} else if catalog == "CURRENT_CATALOG" {
		query = fmt.Sprintf("SHOW STATS FOR %s.%s", schema, table)
	} else {
		query = fmt.Sprintf("SHOW STATS FOR %s.%s.%s", catalog, schema, table)
	}

	rows, err := t.Db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Parse SHOW STATS output
	// The output typically has columns: column_name, data_size, distinct_values_count, nulls_fraction, row_count, low_value, high_value
	for rows.Next() {
		var columnName sql.NullString
		var dataSize, distinctCount, nullsFraction, rowCount sql.NullFloat64
		var lowValue, highValue sql.NullString

		if err := rows.Scan(&columnName, &dataSize, &distinctCount, &nullsFraction, &rowCount, &lowValue, &highValue); err != nil {
			continue
		}

		if columnName.Valid && columnName.String == col.ColumnName {
			if dataSize.Valid {
				dataSizeInt := int64(dataSize.Float64)
				col.DataSizeBytes = &dataSizeInt
			}
			if distinctCount.Valid {
				distinctInt := int64(distinctCount.Float64)
				col.DistinctCount = &distinctInt
			}
			if nullsFraction.Valid && rowCount.Valid {
				nullCount := int64(nullsFraction.Float64 * rowCount.Float64)
				col.NullCount = &nullCount
			}
			if lowValue.Valid {
				col.MinValue = &lowValue.String
			}
			if highValue.Valid {
				col.MaxValue = &highValue.String
			}
			break
		}
	}

	return nil
}

func (t Tool) getPartitionInfo(ctx context.Context, catalog, schema, table string) (*PartitionInfo, error) {
	partInfo := &PartitionInfo{
		IsPartitioned: false,
	}

	// Try to get partition columns from SHOW CREATE TABLE
	var query string
	if catalog == "CURRENT_CATALOG" && schema == "CURRENT_SCHEMA" {
		query = fmt.Sprintf("SHOW CREATE TABLE %s", table)
	} else if catalog == "CURRENT_CATALOG" {
		query = fmt.Sprintf("SHOW CREATE TABLE %s.%s", schema, table)
	} else {
		query = fmt.Sprintf("SHOW CREATE TABLE %s.%s.%s", catalog, schema, table)
	}

	rows, err := t.Db.QueryContext(ctx, query)
	if err != nil {
		return partInfo, err
	}
	defer rows.Close()

	if rows.Next() {
		var createStatement string
		if err := rows.Scan(&createStatement); err == nil {
			// Check if the table is partitioned by looking for PARTITIONED BY clause
			if strings.Contains(strings.ToUpper(createStatement), "PARTITIONED BY") {
				partInfo.IsPartitioned = true
				// Extract partition columns (simplified parsing)
				// Real implementation would need more sophisticated parsing
			}
		}
	}

	// If partitioned, try to get partition details
	if partInfo.IsPartitioned {
		// Try SHOW PARTITIONS
		var partQuery string
		if catalog == "CURRENT_CATALOG" && schema == "CURRENT_SCHEMA" {
			partQuery = fmt.Sprintf("SELECT * FROM \"%s$partitions\"", table)
		} else if catalog == "CURRENT_CATALOG" {
			partQuery = fmt.Sprintf("SELECT * FROM %s.\"%s$partitions\"", schema, table)
		} else {
			partQuery = fmt.Sprintf("SELECT * FROM %s.%s.\"%s$partitions\"", catalog, schema, table)
		}

		// This might fail if the table doesn't have a $partitions table
		partRows, err := t.Db.QueryContext(ctx, partQuery)
		if err == nil {
			defer partRows.Close()
			// Parse partition information
			// Real implementation would need to handle this properly
		}
	}

	return partInfo, nil
}

func (t Tool) getStorageInfo(ctx context.Context, catalog, schema, table string) (*StorageInfo, error) {
	storageInfo := &StorageInfo{}

	// Try to get storage information from table properties or system tables
	// This is connector-specific and may not be available for all connectors

	return storageInfo, nil
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
