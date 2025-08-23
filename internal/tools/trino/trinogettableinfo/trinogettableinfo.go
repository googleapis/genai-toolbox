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

package trinogettableinfo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/trino"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "trino-get-table-info"

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

	// Define parameters
	parameters := tools.Parameters{
		tools.NewStringParameter("table_name", "Required: Table name (can be fully qualified: catalog.schema.table)."),
		tools.NewStringParameter("catalog", "Optional: Catalog name. If not provided and table_name is not fully qualified, uses current catalog."),
		tools.NewStringParameter("schema", "Optional: Schema name. If not provided and table_name is not fully qualified, uses current schema."),
		tools.NewBooleanParameterWithDefault("include_stats", false, "If true, includes table statistics using SHOW STATS. Default is false."),
		tools.NewBooleanParameterWithDefault("include_sample", false, "If true, includes a sample of data from the table. Default is false."),
		tools.NewIntParameterWithDefault("sample_size", 5, "Number of sample rows to include if include_sample is true. Default is 5."),
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

// ColumnDetail represents detailed information about a column
type ColumnDetail struct {
	ColumnName      string  `json:"columnName"`
	DataType        string  `json:"dataType"`
	OrdinalPosition int     `json:"ordinalPosition"`
	IsNullable      string  `json:"isNullable"`
	ColumnDefault   *string `json:"columnDefault,omitempty"`
	ColumnComment   *string `json:"columnComment,omitempty"`
	// Statistics fields (when available)
	DataSize            *float64 `json:"dataSize,omitempty"`
	DistinctValuesCount *float64 `json:"distinctValuesCount,omitempty"`
	NullsFraction       *float64 `json:"nullsFraction,omitempty"`
	MinValue            *string  `json:"minValue,omitempty"`
	MaxValue            *string  `json:"maxValue,omitempty"`
}

// TableMetadata represents comprehensive table information
type TableMetadata struct {
	CatalogName     string                   `json:"catalogName"`
	SchemaName      string                   `json:"schemaName"`
	TableName       string                   `json:"tableName"`
	TableType       string                   `json:"tableType"`
	Columns         []ColumnDetail           `json:"columns"`
	ColumnCount     int                      `json:"columnCount"`
	RowCount        *int64                   `json:"rowCount,omitempty"`
	DataSizeBytes   *int64                   `json:"dataSizeBytes,omitempty"`
	CreateStatement string                   `json:"createStatement,omitempty"`
	SampleData      []map[string]interface{} `json:"sampleData,omitempty"`
	Statistics      []TableStatistic         `json:"statistics,omitempty"`
}

// TableStatistic represents a row from SHOW STATS
type TableStatistic struct {
	ColumnName          *string  `json:"columnName,omitempty"`
	DataSize            *float64 `json:"dataSize,omitempty"`
	DistinctValuesCount *float64 `json:"distinctValuesCount,omitempty"`
	NullsFraction       *float64 `json:"nullsFraction,omitempty"`
	RowCount            *float64 `json:"rowCount,omitempty"`
	LowValue            *string  `json:"lowValue,omitempty"`
	HighValue           *string  `json:"highValue,omitempty"`
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
		return nil, fmt.Errorf("'table_name' parameter is required")
	}

	catalog, _ := paramsMap["catalog"].(string)
	schema, _ := paramsMap["schema"].(string)
	includeStats, _ := paramsMap["include_stats"].(bool)
	includeSample, _ := paramsMap["include_sample"].(bool)
	sampleSize, _ := paramsMap["sample_size"].(int)
	if sampleSize <= 0 {
		sampleSize = 5
	}

	// Parse the table name to extract catalog, schema, and table parts
	catalogName, schemaName, actualTableName := t.parseTableName(tableName, catalog, schema)

	// Get table metadata
	metadata, err := t.getTableMetadata(ctx, catalogName, schemaName, actualTableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get table metadata: %w", err)
	}

	// Get column information
	columns, err := t.getColumnInfo(ctx, catalogName, schemaName, actualTableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get column information: %w", err)
	}
	metadata.Columns = columns
	metadata.ColumnCount = len(columns)

	// Get CREATE TABLE statement
	createStmt, err := t.getCreateTableStatement(ctx, catalogName, schemaName, actualTableName)
	if err == nil && createStmt != "" {
		metadata.CreateStatement = createStmt
	}

	// Get table statistics if requested
	if includeStats {
		stats, err := t.getTableStatistics(ctx, catalogName, schemaName, actualTableName)
		if err == nil {
			metadata.Statistics = stats
			// Update column details with statistics
			t.mergeColumnStatistics(metadata, stats)
		}
	}

	// Get sample data if requested
	if includeSample {
		sampleData, err := t.getSampleData(ctx, catalogName, schemaName, actualTableName, sampleSize)
		if err == nil {
			metadata.SampleData = sampleData
		}
	}

	return metadata, nil
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

	return catalogName, schemaName, actualTableName
}

func (t Tool) getTableMetadata(ctx context.Context, catalog, schema, table string) (*TableMetadata, error) {
	var query string
	var args []interface{}

	query = `
		SELECT 
			table_catalog,
			table_schema,
			table_name,
			table_type
		FROM information_schema.tables
		WHERE table_name = ?
	`
	args = append(args, table)

	if catalog != "" {
		query += ` AND table_catalog = ?`
		args = append(args, catalog)
	} else {
		query += ` AND table_catalog = CURRENT_CATALOG`
	}

	if schema != "" {
		query += ` AND table_schema = ?`
		args = append(args, schema)
	} else {
		query += ` AND table_schema = CURRENT_SCHEMA`
	}

	var metadata TableMetadata
	err := t.Db.QueryRowContext(ctx, query, args...).Scan(
		&metadata.CatalogName,
		&metadata.SchemaName,
		&metadata.TableName,
		&metadata.TableType,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("table not found: %s", table)
	}
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}

func (t Tool) getColumnInfo(ctx context.Context, catalog, schema, table string) ([]ColumnDetail, error) {
	var query string
	var args []interface{}

	query = `
		SELECT 
			column_name,
			data_type,
			ordinal_position,
			is_nullable,
			column_default,
			column_comment
		FROM information_schema.columns
		WHERE table_name = ?
	`
	args = append(args, table)

	if catalog != "" {
		query += ` AND table_catalog = ?`
		args = append(args, catalog)
	} else {
		query += ` AND table_catalog = CURRENT_CATALOG`
	}

	if schema != "" {
		query += ` AND table_schema = ?`
		args = append(args, schema)
	} else {
		query += ` AND table_schema = CURRENT_SCHEMA`
	}

	query += ` ORDER BY ordinal_position`

	rows, err := t.Db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnDetail
	for rows.Next() {
		var col ColumnDetail
		var defaultValue, comment sql.NullString

		err := rows.Scan(
			&col.ColumnName,
			&col.DataType,
			&col.OrdinalPosition,
			&col.IsNullable,
			&defaultValue,
			&comment,
		)
		if err != nil {
			return nil, err
		}

		if defaultValue.Valid {
			col.ColumnDefault = &defaultValue.String
		}
		if comment.Valid {
			col.ColumnComment = &comment.String
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

func (t Tool) getCreateTableStatement(ctx context.Context, catalog, schema, table string) (string, error) {
	var query string

	// Build fully qualified table name for SHOW CREATE TABLE
	var fullTableName string
	if catalog != "" && schema != "" {
		fullTableName = fmt.Sprintf("%s.%s.%s", catalog, schema, table)
	} else if schema != "" {
		fullTableName = fmt.Sprintf("%s.%s", schema, table)
	} else {
		fullTableName = table
	}

	query = fmt.Sprintf("SHOW CREATE TABLE %s", fullTableName)

	rows, err := t.Db.QueryContext(ctx, query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var createStatement strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return "", err
		}
		createStatement.WriteString(line)
		createStatement.WriteString("\n")
	}

	return strings.TrimSpace(createStatement.String()), rows.Err()
}

func (t Tool) getTableStatistics(ctx context.Context, catalog, schema, table string) ([]TableStatistic, error) {
	var query string

	// Build fully qualified table name for SHOW STATS
	var fullTableName string
	if catalog != "" && schema != "" {
		fullTableName = fmt.Sprintf("%s.%s.%s", catalog, schema, table)
	} else if schema != "" {
		fullTableName = fmt.Sprintf("%s.%s", schema, table)
	} else {
		fullTableName = table
	}

	query = fmt.Sprintf("SHOW STATS FOR %s", fullTableName)

	rows, err := t.Db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []TableStatistic

	// SHOW STATS typically returns: column_name, data_size, distinct_values_count, nulls_fraction, row_count, low_value, high_value
	for rows.Next() {
		var stat TableStatistic
		var columnName, lowValue, highValue sql.NullString
		var dataSize, distinctCount, nullsFraction, rowCount sql.NullFloat64

		err := rows.Scan(
			&columnName,
			&dataSize,
			&distinctCount,
			&nullsFraction,
			&rowCount,
			&lowValue,
			&highValue,
		)
		if err != nil {
			// Some connectors might not return all columns
			continue
		}

		if columnName.Valid {
			stat.ColumnName = &columnName.String
		}
		if dataSize.Valid {
			stat.DataSize = &dataSize.Float64
		}
		if distinctCount.Valid {
			stat.DistinctValuesCount = &distinctCount.Float64
		}
		if nullsFraction.Valid {
			stat.NullsFraction = &nullsFraction.Float64
		}
		if rowCount.Valid {
			stat.RowCount = &rowCount.Float64
		}
		if lowValue.Valid {
			stat.LowValue = &lowValue.String
		}
		if highValue.Valid {
			stat.HighValue = &highValue.String
		}

		stats = append(stats, stat)
	}

	return stats, rows.Err()
}

func (t Tool) mergeColumnStatistics(metadata *TableMetadata, stats []TableStatistic) {
	// Create a map for quick lookup
	statMap := make(map[string]TableStatistic)
	for _, stat := range stats {
		if stat.ColumnName != nil {
			statMap[*stat.ColumnName] = stat
		}
	}

	// Update column details with statistics
	for i := range metadata.Columns {
		if stat, ok := statMap[metadata.Columns[i].ColumnName]; ok {
			metadata.Columns[i].DataSize = stat.DataSize
			metadata.Columns[i].DistinctValuesCount = stat.DistinctValuesCount
			metadata.Columns[i].NullsFraction = stat.NullsFraction
			metadata.Columns[i].MinValue = stat.LowValue
			metadata.Columns[i].MaxValue = stat.HighValue
		}
	}

	// Check for table-level statistics (null column name)
	for _, stat := range stats {
		if stat.ColumnName == nil && stat.RowCount != nil {
			rowCount := int64(*stat.RowCount)
			metadata.RowCount = &rowCount
		}
		if stat.ColumnName == nil && stat.DataSize != nil {
			dataSize := int64(*stat.DataSize)
			metadata.DataSizeBytes = &dataSize
		}
	}
}

func (t Tool) getSampleData(ctx context.Context, catalog, schema, table string, limit int) ([]map[string]interface{}, error) {
	var query string

	// Build fully qualified table name
	var fullTableName string
	if catalog != "" && schema != "" {
		fullTableName = fmt.Sprintf("%s.%s.%s", catalog, schema, table)
	} else if schema != "" {
		fullTableName = fmt.Sprintf("%s.%s", schema, table)
	} else {
		fullTableName = table
	}

	query = fmt.Sprintf("SELECT * FROM %s LIMIT %d", fullTableName, limit)

	rows, err := t.Db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var sampleData []map[string]interface{}

	// Create a slice to hold the values
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		err := rows.Scan(valuePtrs...)
		if err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Handle byte arrays
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		sampleData = append(sampleData, row)
	}

	return sampleData, rows.Err()
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
