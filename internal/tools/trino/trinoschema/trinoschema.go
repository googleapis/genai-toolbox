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

package trinoschema

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/trino"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/neo4j/neo4jschema/cache"
)

const kind string = "trino-schema"

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
	Name               string   `yaml:"name" validate:"required"`
	Kind               string   `yaml:"kind" validate:"required"`
	Source             string   `yaml:"source" validate:"required"`
	Description        string   `yaml:"description" validate:"required"`
	AuthRequired       []string `yaml:"authRequired"`
	CacheExpireMinutes *int     `yaml:"cacheExpireMinutes"`
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

	// Set default cache expiration if not specified
	cacheExpireMinutes := 10
	if cfg.CacheExpireMinutes != nil {
		cacheExpireMinutes = *cfg.CacheExpireMinutes
	}

	// Create a cache instance
	cacheInstance := cache.NewCache()

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: tools.Parameters{}.McpManifest(),
	}

	// finish tool setup
	t := Tool{
		Name:               cfg.Name,
		Kind:               kind,
		AuthRequired:       cfg.AuthRequired,
		Db:                 s.TrinoDB(),
		cache:              cacheInstance,
		cacheExpireMinutes: &cacheExpireMinutes,
		manifest:           tools.Manifest{Description: cfg.Description, Parameters: tools.Parameters{}.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:        mcpManifest,
	}
	return t, nil
}

// SchemaInfo represents the complete Trino schema information
type SchemaInfo struct {
	Catalogs    []CatalogInfo `json:"catalogs"`
	ClusterInfo ClusterInfo   `json:"clusterInfo"`
	Statistics  Statistics    `json:"statistics"`
	Errors      []string      `json:"errors,omitempty"`
}

// CatalogInfo represents a Trino catalog with its schemas
type CatalogInfo struct {
	Name    string       `json:"name"`
	Schemas []SchemaData `json:"schemas"`
}

// SchemaData represents a schema with its tables
type SchemaData struct {
	Name   string      `json:"name"`
	Tables []TableInfo `json:"tables"`
}

// TableInfo represents a table with its columns
type TableInfo struct {
	Name       string       `json:"name"`
	Type       string       `json:"type"`
	Columns    []ColumnInfo `json:"columns"`
	RowCount   *int64       `json:"rowCount,omitempty"`
	DataSizeMB *float64     `json:"dataSizeMB,omitempty"`
}

// ColumnInfo represents a column in a table
type ColumnInfo struct {
	Name         string  `json:"name"`
	DataType     string  `json:"dataType"`
	Position     int     `json:"position"`
	IsNullable   string  `json:"isNullable"`
	DefaultValue *string `json:"defaultValue,omitempty"`
	Comment      *string `json:"comment,omitempty"`
}

// ClusterInfo contains Trino cluster information
type ClusterInfo struct {
	TotalNodes   int        `json:"totalNodes"`
	Coordinators []NodeInfo `json:"coordinators"`
	Workers      []NodeInfo `json:"workers"`
	Version      string     `json:"version,omitempty"`
}

// NodeInfo represents a node in the Trino cluster
type NodeInfo struct {
	NodeID      string `json:"nodeId"`
	HttpURI     string `json:"httpUri"`
	NodeVersion string `json:"nodeVersion"`
	State       string `json:"state"`
}

// Statistics contains database statistics
type Statistics struct {
	TotalCatalogs int            `json:"totalCatalogs"`
	TotalSchemas  int            `json:"totalSchemas"`
	TotalTables   int            `json:"totalTables"`
	TotalColumns  int            `json:"totalColumns"`
	TablesByType  map[string]int `json:"tablesByType"`
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name               string   `yaml:"name"`
	Kind               string   `yaml:"kind"`
	AuthRequired       []string `yaml:"authRequired"`
	Db                 *sql.DB
	cache              *cache.Cache
	cacheExpireMinutes *int
	manifest           tools.Manifest
	mcpManifest        tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	// Check if a valid schema is already in the cache
	if cachedSchema, ok := t.cache.Get("schema"); ok {
		if schema, ok := cachedSchema.(*SchemaInfo); ok {
			return schema, nil
		}
	}

	// If not cached, extract the schema from the database
	schema, err := t.extractSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract database schema: %w", err)
	}

	// Cache the newly extracted schema for future use
	expiration := time.Duration(*t.cacheExpireMinutes) * time.Minute
	t.cache.Set("schema", schema, expiration)

	return schema, nil
}

func (t Tool) extractSchema(ctx context.Context) (*SchemaInfo, error) {
	schema := &SchemaInfo{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	errCh := make(chan error, 4)

	// Define the different schema extraction tasks
	tasks := []struct {
		name string
		fn   func() error
	}{
		{
			name: "catalogs-schemas",
			fn: func() error {
				catalogs, err := t.extractCatalogsAndSchemas(ctx)
				if err != nil {
					return fmt.Errorf("failed to extract catalogs and schemas: %w", err)
				}
				mu.Lock()
				defer mu.Unlock()
				schema.Catalogs = catalogs
				return nil
			},
		},
		{
			name: "tables",
			fn: func() error {
				// Wait for catalogs to be extracted first
				time.Sleep(100 * time.Millisecond)
				mu.Lock()
				catalogs := schema.Catalogs
				mu.Unlock()

				if len(catalogs) > 0 {
					tables, err := t.extractTables(ctx, catalogs)
					if err != nil {
						return fmt.Errorf("failed to extract tables: %w", err)
					}
					mu.Lock()
					defer mu.Unlock()
					schema.Catalogs = tables
				}
				return nil
			},
		},
		{
			name: "cluster-info",
			fn: func() error {
				clusterInfo, err := t.extractClusterInfo(ctx)
				if err != nil {
					return fmt.Errorf("failed to extract cluster info: %w", err)
				}
				mu.Lock()
				defer mu.Unlock()
				schema.ClusterInfo = *clusterInfo
				return nil
			},
		},
		{
			name: "statistics",
			fn: func() error {
				// Wait for other data to be collected
				time.Sleep(200 * time.Millisecond)
				mu.Lock()
				stats := t.calculateStatistics(schema)
				schema.Statistics = stats
				mu.Unlock()
				return nil
			},
		},
	}

	// Execute all tasks concurrently
	for _, task := range tasks {
		wg.Add(1)
		go func(taskName string, taskFn func() error) {
			defer wg.Done()
			if err := taskFn(); err != nil {
				errCh <- fmt.Errorf("%s: %w", taskName, err)
			}
		}(task.name, task.fn)
	}

	// Wait for all tasks to complete
	go func() {
		wg.Wait()
		close(errCh)
	}()

	// Collect any errors
	var errors []string
	for err := range errCh {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		schema.Errors = errors
	}

	return schema, nil
}

func (t Tool) extractCatalogsAndSchemas(ctx context.Context) ([]CatalogInfo, error) {
	query := `
		SELECT DISTINCT catalog_name, schema_name 
		FROM information_schema.schemata 
		ORDER BY catalog_name, schema_name
	`

	rows, err := t.Db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	catalogMap := make(map[string][]SchemaData)
	for rows.Next() {
		var catalogName, schemaName string
		if err := rows.Scan(&catalogName, &schemaName); err != nil {
			return nil, err
		}
		catalogMap[catalogName] = append(catalogMap[catalogName], SchemaData{Name: schemaName})
	}

	var catalogs []CatalogInfo
	for name, schemas := range catalogMap {
		catalogs = append(catalogs, CatalogInfo{
			Name:    name,
			Schemas: schemas,
		})
	}

	return catalogs, rows.Err()
}

func (t Tool) extractTables(ctx context.Context, catalogs []CatalogInfo) ([]CatalogInfo, error) {
	for i := range catalogs {
		for j := range catalogs[i].Schemas {
			tables, err := t.extractTablesForSchema(ctx, catalogs[i].Name, catalogs[i].Schemas[j].Name)
			if err != nil {
				// Continue with other schemas even if one fails
				continue
			}
			catalogs[i].Schemas[j].Tables = tables
		}
	}
	return catalogs, nil
}

func (t Tool) extractTablesForSchema(ctx context.Context, catalogName, schemaName string) ([]TableInfo, error) {
	query := `
		SELECT table_name, table_type
		FROM information_schema.tables
		WHERE table_catalog = ? AND table_schema = ?
		ORDER BY table_name
	`

	rows, err := t.Db.QueryContext(ctx, query, catalogName, schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		if err := rows.Scan(&table.Name, &table.Type); err != nil {
			continue
		}

		// Extract columns for each table
		columns, err := t.extractColumnsForTable(ctx, catalogName, schemaName, table.Name)
		if err == nil {
			table.Columns = columns
		}

		tables = append(tables, table)
	}

	return tables, rows.Err()
}

func (t Tool) extractColumnsForTable(ctx context.Context, catalogName, schemaName, tableName string) ([]ColumnInfo, error) {
	query := `
		SELECT 
			column_name,
			data_type,
			ordinal_position,
			is_nullable,
			column_default,
			column_comment
		FROM information_schema.columns
		WHERE table_catalog = ? 
			AND table_schema = ? 
			AND table_name = ?
		ORDER BY ordinal_position
	`

	rows, err := t.Db.QueryContext(ctx, query, catalogName, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var defaultValue, comment sql.NullString

		if err := rows.Scan(&col.Name, &col.DataType, &col.Position, &col.IsNullable, &defaultValue, &comment); err != nil {
			continue
		}

		if defaultValue.Valid {
			col.DefaultValue = &defaultValue.String
		}
		if comment.Valid {
			col.Comment = &comment.String
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

func (t Tool) extractClusterInfo(ctx context.Context) (*ClusterInfo, error) {
	query := `
		SELECT 
			node_id,
			http_uri,
			node_version,
			coordinator,
			state
		FROM system.runtime.nodes
		ORDER BY coordinator DESC, node_id
	`

	rows, err := t.Db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clusterInfo := &ClusterInfo{
		Coordinators: []NodeInfo{},
		Workers:      []NodeInfo{},
	}

	for rows.Next() {
		var node NodeInfo
		var isCoordinator bool

		if err := rows.Scan(&node.NodeID, &node.HttpURI, &node.NodeVersion, &isCoordinator, &node.State); err != nil {
			continue
		}

		if isCoordinator {
			clusterInfo.Coordinators = append(clusterInfo.Coordinators, node)
		} else {
			clusterInfo.Workers = append(clusterInfo.Workers, node)
		}

		if clusterInfo.Version == "" {
			clusterInfo.Version = node.NodeVersion
		}
	}

	clusterInfo.TotalNodes = len(clusterInfo.Coordinators) + len(clusterInfo.Workers)

	return clusterInfo, rows.Err()
}

func (t Tool) calculateStatistics(schema *SchemaInfo) Statistics {
	stats := Statistics{
		TablesByType: make(map[string]int),
	}

	for _, catalog := range schema.Catalogs {
		stats.TotalCatalogs++
		for _, schemaData := range catalog.Schemas {
			stats.TotalSchemas++
			for _, table := range schemaData.Tables {
				stats.TotalTables++
				stats.TablesByType[table.Type]++
				stats.TotalColumns += len(table.Columns)
			}
		}
	}

	return stats
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParamValues{}, nil
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
