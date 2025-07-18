// Copyright 2024 Google LLC
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

package neo4jdbschema

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	neo4jsc "github.com/googleapis/genai-toolbox/internal/sources/neo4j"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/neo4j/cache"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const kind string = "neo4j-db-schema"

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
	Neo4jDriver() neo4j.DriverWithContext
	Neo4jDatabase() string
}

// validate compatible sources are still compatible
var _ compatibleSource = &neo4jsc.Source{}

var compatibleSources = [...]string{neo4jsc.SourceKind}

type Config struct {
	Name               string   `yaml:"name" validate:"required"`
	Kind               string   `yaml:"kind" validate:"required"`
	Source             string   `yaml:"source" validate:"required"`
	Description        string   `yaml:"description" validate:"required"`
	AuthRequired       []string `yaml:"authRequired"`
	CacheExpireMinutes int      `yaml:"cacheExpireMinutes,default=60"` // Cache expiration time in minutes
	DisableDbInfo      bool     `yaml:"disableDbInfo"`                 // If true, skips extracting database info (like version, edition)
	DisableErrors      bool     `yaml:"disableErrors"`                 // If true, skips collecting errors during schema extraction
	DisableIndexes     bool     `yaml:"disableIndexes"`                // If true, skips extracting indexes
	DisableConstraints bool     `yaml:"disableConstraints"`            // If true, skips extracting constraints
	DisableAPOCUsage   bool     `yaml:"disableAPOCUsage"`              // If true, skips using APOC procedures for schema extraction
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
	var s compatibleSource
	s, ok = rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	parameters := tools.Parameters{}
	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	// finish tool setup
	t := Tool{
		Name:         cfg.Name,
		Kind:         kind,
		AuthRequired: cfg.AuthRequired,
		Driver:       s.Neo4jDriver(),
		Database:     s.Neo4jDatabase(),
		cache:        cache.NewCache(),
		conf:         &cfg,
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

// SchemaInfo represents the complete database schema
type SchemaInfo struct {
	NodeLabels    []NodeLabel    `json:"nodeLabels"`
	Relationships []Relationship `json:"relationships"`
	Constraints   []Constraint   `json:"constraints"`
	Indexes       []Index        `json:"indexes"`
	DatabaseInfo  DatabaseInfo   `json:"databaseInfo"`
	Statistics    Statistics     `json:"statistics"`
	Errors        []string       `json:"errors,omitempty"`
}

// NodeLabel represents a node label with its properties
type NodeLabel struct {
	Name       string         `json:"name"`
	Properties []PropertyInfo `json:"properties"`
	Count      int64          `json:"count"`
}

// Relationship represents a relationship type with its properties
type Relationship struct {
	Type       string         `json:"type"`
	Properties []PropertyInfo `json:"properties"`
	StartNode  string         `json:"startNode,omitempty"`
	EndNode    string         `json:"endNode,omitempty"`
	Count      int64          `json:"count"`
}

// PropertyInfo represents a property with its data types
type PropertyInfo struct {
	Name      string   `json:"name"`
	Types     []string `json:"types"`
	Mandatory bool     `json:"-"`
	Unique    bool     `json:"-"`
	Indexed   bool     `json:"-"`
}

// Constraint represents a database constraint
type Constraint struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	EntityType string   `json:"entityType"`
	Label      string   `json:"label,omitempty"`
	Properties []string `json:"properties"`
}

// Index represents a database index
type Index struct {
	Name       string   `json:"name"`
	State      string   `json:"state"`
	Type       string   `json:"type"`
	EntityType string   `json:"entityType"`
	Label      string   `json:"label,omitempty"`
	Properties []string `json:"properties"`
}

// DatabaseInfo contains general database information
type DatabaseInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Edition string `json:"edition,omitempty"`
}

// Statistics contains database statistics
type Statistics struct {
	TotalNodes          int64            `json:"totalNodes"`
	TotalRelationships  int64            `json:"totalRelationships"`
	TotalProperties     int64            `json:"totalProperties"`
	NodesByLabel        map[string]int64 `json:"nodesByLabel"`
	RelationshipsByType map[string]int64 `json:"relationshipsByType"`
	PropertiesByLabel   map[string]int64 `json:"propertiesByLabel"`
	PropertiesByRelType map[string]int64 `json:"propertiesByRelType"`
}

// APOCSchemaResult represents the result from apoc.meta.schema()
type APOCSchemaResult struct {
	Value map[string]APOCEntity `json:"value"`
}

// APOCEntity represents a node or relationship in APOC schema
type APOCEntity struct {
	Type          string                          `json:"type"`
	Count         int64                           `json:"count"`
	Labels        []string                        `json:"labels,omitempty"`
	Properties    map[string]APOCProperty         `json:"properties"`
	Relationships map[string]APOCRelationshipInfo `json:"relationships,omitempty"`
}

// APOCProperty represents property info from APOC
type APOCProperty struct {
	Type      string `json:"type"`
	Indexed   bool   `json:"indexed"`
	Unique    bool   `json:"unique"`
	Existence bool   `json:"existence"`
}

// APOCRelationshipInfo represents relationship info from APOC
type APOCRelationshipInfo struct {
	Count      int64                   `json:"count"`
	Direction  string                  `json:"direction"`
	Labels     []string                `json:"labels"`
	Properties map[string]APOCProperty `json:"properties"`
}

type Tool struct {
	Name         string   `yaml:"name"`
	Kind         string   `yaml:"kind"`
	AuthRequired []string `yaml:"authRequired"`
	Driver       neo4j.DriverWithContext
	Database     string
	cache        *cache.Cache
	conf         *Config
	manifest     tools.Manifest
	mcpManifest  tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) (any, error) {
	var schema *SchemaInfo

	// Check if the schema was already cached and if so, return it
	cachedSchema, ok := t.cache.Get("schema")
	if ok {
		if schema, ok = cachedSchema.(*SchemaInfo); ok {
			return []any{schema}, nil
		}
	}

	var err error
	schema, err = t.extractSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract database schema: %w", err)
	}

	// Cache the schema for future use
	t.cache.Set("schema", schema, time.Duration(t.conf.CacheExpireMinutes)*time.Minute)

	return schema, nil
}

func (t Tool) ParseParams(data map[string]any, claimsMap map[string]map[string]any) (tools.ParamValues, error) {
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

// extractSchema extracts the complete schema from Neo4j
func (t Tool) extractSchema(ctx context.Context) (*SchemaInfo, error) {
	schema := &SchemaInfo{}
	var mu sync.Mutex

	// Define extraction tasks
	type extractionTask struct {
		name     string
		fn       func() error
		disabled bool
	}

	tasks := []extractionTask{
		{
			name:     "database info",
			disabled: t.conf.DisableDbInfo,
			fn: func() error {
				dbInfo, err := t.extractDatabaseInfo(ctx)
				if err != nil {
					return fmt.Errorf("failed to extract database info: %w", err)
				}

				mu.Lock()
				defer mu.Unlock()
				schema.DatabaseInfo = *dbInfo
				return nil
			},
		},
		{
			name: "APOC schema",
			fn: func() error {
				var nodeLabels []NodeLabel
				var relationships []Relationship
				var stats *Statistics
				var err error

				if !t.conf.DisableAPOCUsage {
					nodeLabels, relationships, stats, err = t.GetAPOCSchema(ctx)
				} else {
					nodeLabels, relationships, stats, err = t.GetSchemaWithoutAPOC(ctx, 100)
				}
				if err != nil {
					return fmt.Errorf("failed to get schema: %w", err)
				}

				mu.Lock()
				defer mu.Unlock()
				schema.NodeLabels = nodeLabels
				schema.Statistics = *stats

				// With low sampling rate, we may not have all relationships and it is better to extract them separately
				if !t.conf.DisableAPOCUsage {
					schema.Relationships = relationships
				}
				return nil
			},
		},
		{
			name:     "constraints",
			disabled: t.conf.DisableConstraints,
			fn: func() error {
				constraints, err := t.extractConstraints(ctx)
				if err != nil {
					return fmt.Errorf("failed to extract constraints: %w", err)
				}

				mu.Lock()
				defer mu.Unlock()
				schema.Constraints = constraints
				return nil
			},
		},
		{
			name:     "indexes",
			disabled: t.conf.DisableIndexes,
			fn: func() error {
				indexes, err := t.extractIndexes(ctx)
				if err != nil {
					return fmt.Errorf("failed to extract indexes: %w", err)
				}

				mu.Lock()
				defer mu.Unlock()
				schema.Indexes = indexes
				return nil
			},
		},
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(tasks))

	// Execute all tasks concurrently
	for _, task := range tasks {
		if task.disabled {
			continue
		}
		wg.Add(1)
		go func(t extractionTask) {
			defer wg.Done()
			if err := t.fn(); err != nil {
				errCh <- err
				return
			}
		}(task)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Collect any errors that occurred
	close(errCh)
	if t.conf.DisableErrors {
		return schema, nil // If errors are disabled, return the schema without errors
	}

	for err := range errCh {
		if err != nil {
			schema.Errors = append(schema.Errors, err.Error())
		}
	}
	return schema, nil
}

// GetAPOCSchema calls apoc.meta.schema() to get schema information with low sampling.
// It means that it may not capture all properties or relationships, but it provides
// a good overview of the database structure.
func (t Tool) GetAPOCSchema(ctx context.Context) ([]NodeLabel, []Relationship, *Statistics, error) {
	// Declare result structures variables
	var nodeLabels []NodeLabel
	var relationships []Relationship
	var stats = &Statistics{
		NodesByLabel:        make(map[string]int64),
		RelationshipsByType: make(map[string]int64),
		PropertiesByLabel:   make(map[string]int64),
		PropertiesByRelType: make(map[string]int64),
	}

	// Shared mutex for concurrent access
	var mu sync.Mutex
	var firstErr error

	// Create a context with cancellation to stop all goroutines on first error
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	// Helper to handle errors
	handleError := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		if firstErr == nil {
			firstErr = err
			cancel() // Cancel context to stop other operations
		}
	}

	// Define Task structure
	type Task struct {
		name string
		fn   func(session neo4j.SessionWithContext) error
	}

	tasks := []Task{
		{
			name: "apoc-schema",
			fn: func(session neo4j.SessionWithContext) error {
				// Run an APOC schema query with sampling
				result, err := session.Run(ctx, "CALL apoc.meta.schema({sample: 10}) YIELD value RETURN value", nil)
				if err != nil {
					return fmt.Errorf("failed to run APOC schema query: %w", err)
				}

				if !result.Next(ctx) {
					return fmt.Errorf("no results returned from APOC schema query")
				}

				// The result is a map of entity names to their metadata
				schemaMap, ok := result.Record().Values[0].(map[string]any)
				if !ok {
					return fmt.Errorf("unexpected result format from APOC schema query: %T", result.Record().Values[0])
				}

				// Marshal the schema map to APOCSchemaResult structure
				var apocSchemaResult *APOCSchemaResult
				apocSchemaResult, err = mapToAPOCSchema(schemaMap)
				if err != nil {
					return fmt.Errorf("failed to convert schema map to APOCSchemaResult: %w", err)
				}

				mu.Lock()
				defer mu.Unlock()

				var apocStats *Statistics
				nodeLabels, apocStats = processAPOCSchema(apocSchemaResult)
				stats.TotalNodes = apocStats.TotalNodes
				stats.TotalProperties += apocStats.TotalProperties
				stats.NodesByLabel = apocStats.NodesByLabel
				stats.PropertiesByLabel = apocStats.PropertiesByLabel
				return nil
			},
		},
		{
			name: "apoc-relationships",
			fn: func(session neo4j.SessionWithContext) error {
				// Retrieve a distinct list of relationship types with their properties, counts and sources and targets
				result, err := session.Run(ctx, `
					MATCH (startNode)-[rel]->(endNode)
					WITH 
					  labels(startNode)[0] AS startNode,
					  type(rel) AS relType,
					  apoc.meta.cypher.types(rel) AS relProperties,
					  labels(endNode)[0] AS endNode,
					  count(*) AS count
					RETURN  relType, startNode, endNode, relProperties, count
				`, nil)

				if err != nil {
					return fmt.Errorf("failed to extract relationships: %w", err)
				}

				for result.Next(ctx) {
					record := result.Record()
					relType := record.Values[0].(string)
					startNode := record.Values[1].(string)
					endNode := record.Values[2].(string)
					properties := record.Values[3].(map[string]any)
					count := record.Values[4].(int64)

					if relType == "" || count == 0 {
						continue // Skip empty relationship types or those with zero counts
					}

					relationship := Relationship{
						Type:       relType,
						StartNode:  startNode,
						EndNode:    endNode,
						Count:      count,
						Properties: []PropertyInfo{},
					}

					for prop, propType := range properties {
						relationship.Properties = append(relationship.Properties, PropertyInfo{
							Name:  prop,
							Types: []string{propType.(string)},
						})
					}

					mu.Lock()
					relationships = append(relationships, relationship)
					stats.RelationshipsByType[relType] += count
					stats.TotalRelationships += count

					cnt := int64(len(relationship.Properties))
					stats.TotalProperties += cnt
					stats.PropertiesByRelType[relType] += cnt
					mu.Unlock()
				}

				// set stats maps to nil if they are empty
				mu.Lock()
				defer mu.Unlock()

				if len(stats.RelationshipsByType) == 0 {
					stats.RelationshipsByType = nil
				}
				if len(stats.PropertiesByRelType) == 0 {
					stats.PropertiesByRelType = nil
				}
				return nil
			},
		},
	}

	// Execute all tasks concurrently
	var wg sync.WaitGroup
	wg.Add(len(tasks))

	for _, task := range tasks {
		go func(task Task) {
			defer wg.Done()

			// Create a new session for this Task
			session := t.Driver.NewSession(ctx, neo4j.SessionConfig{
				DatabaseName: t.Database,
			})
			defer session.Close(ctx)

			// Execute the Task
			if err := task.fn(session); err != nil {
				handleError(fmt.Errorf("task %s failed: %w", task.name, err))
			}
		}(task)
	}

	wg.Wait()

	// Check if any errors occurred
	if firstErr != nil {
		return nil, nil, nil, firstErr
	}

	return nodeLabels, relationships, stats, nil
}

// GetSchemaWithoutAPOC collects schema information using native Cypher queries and data sampling.
// It is designed as a replacement for APOC-based schema extraction and uses goroutines for performance.
// It returns the discovered node labels, relationships, basic statistics, and any error that occurred.
func (t Tool) GetSchemaWithoutAPOC(ctx context.Context, sampleSize int) ([]NodeLabel, []Relationship, *Statistics, error) {
	// Pre-allocate result structures
	nodePropsMap := make(map[string]map[string]map[string]bool, 32) // label -> property -> types set
	relPropsMap := make(map[string]map[string]map[string]bool, 32)  // relType -> property -> types set
	nodeCounts := make(map[string]int64, 32)
	relCounts := make(map[string]int64, 32)
	relConnectivity := make(map[string]struct {
		startNode string
		endNode   string
		count     int64
	}, 32)

	// Shared mutex for concurrent access
	var mu sync.Mutex
	var firstErr error

	// Create a context with cancellation to stop all goroutines on first error
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	// Helper to handle errors
	handleError := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		if firstErr == nil {
			firstErr = err
			cancel() // Cancel context to stop other operations
		}
	}

	// Define Task structure
	type Task struct {
		name string
		fn   func(session neo4j.SessionWithContext) error
	}

	// Define all tasks
	tasks := []Task{
		{
			name: "node-schema",
			fn: func(session neo4j.SessionWithContext) error {
				// Get node counts first for all labels
				countQuery := `
					MATCH (n)
					UNWIND labels(n) AS label
					RETURN label, count(*) AS count
					ORDER BY count DESC
				`

				countResult, err := session.Run(ctx, countQuery, nil)
				if err != nil {
					return fmt.Errorf("node count query failed: %w", err)
				}

				labelsList := make([]string, 0)
				mu.Lock()
				for countResult.Next(ctx) {
					record := countResult.Record()
					label := record.Values[0].(string)
					count := record.Values[1].(int64)
					nodeCounts[label] = count
					labelsList = append(labelsList, label)
				}
				mu.Unlock()

				if err = countResult.Err(); err != nil {
					return fmt.Errorf("node count result error: %w", err)
				}

				// Sample properties for each label
				for _, label := range labelsList {
					propQuery := fmt.Sprintf(`
						MATCH (n:%s)
						WITH n LIMIT $sampleSize
						UNWIND keys(n) AS key
						WITH key, n[key] AS value
						WHERE value IS NOT NULL
						RETURN key, COLLECT(DISTINCT valueType(value)) AS types
					`, label)

					var propResult neo4j.ResultWithContext
					propResult, err = session.Run(ctx, propQuery, map[string]any{"sampleSize": sampleSize})
					if err != nil {
						return fmt.Errorf("node properties query for label %s failed: %w", label, err)
					}

					mu.Lock()
					if nodePropsMap[label] == nil {
						nodePropsMap[label] = make(map[string]map[string]bool)
					}

					for propResult.Next(ctx) {
						record := propResult.Record()
						key := record.Values[0].(string)
						types := record.Values[1].([]any)

						if nodePropsMap[label][key] == nil {
							nodePropsMap[label][key] = make(map[string]bool)
						}

						for _, tp := range types {
							nodePropsMap[label][key][tp.(string)] = true
						}
					}
					mu.Unlock()

					if err = propResult.Err(); err != nil {
						return fmt.Errorf("node properties result error for label %s: %w", label, err)
					}
				}

				return nil
			},
		},
		{
			name: "relationship-schema",
			fn: func(session neo4j.SessionWithContext) error {
				// Get relationship counts and connectivity in one query
				relQuery := `
					MATCH (start)-[r]->(end)
					WITH type(r) AS relType, 
					     labels(start) AS startLabels, 
					     labels(end) AS endLabels,
					     count(*) AS count
					RETURN relType, 
					       CASE WHEN size(startLabels) > 0 THEN startLabels[0] ELSE null END AS startLabel,
					       CASE WHEN size(endLabels) > 0 THEN endLabels[0] ELSE null END AS endLabel,
					       sum(count) AS totalCount
					ORDER BY totalCount DESC
				`

				relResult, err := session.Run(ctx, relQuery, nil)
				if err != nil {
					return fmt.Errorf("relationship count query failed: %w", err)
				}

				relTypesList := make([]string, 0)
				mu.Lock()
				for relResult.Next(ctx) {
					record := relResult.Record()
					relType := record.Values[0].(string)
					startLabel := ""
					endLabel := ""
					if record.Values[1] != nil {
						startLabel = record.Values[1].(string)
					}
					if record.Values[2] != nil {
						endLabel = record.Values[2].(string)
					}
					count := record.Values[3].(int64)

					relCounts[relType] = count
					relTypesList = append(relTypesList, relType)

					// Store the most common pattern
					if existing, ok := relConnectivity[relType]; !ok || count > existing.count {
						relConnectivity[relType] = struct {
							startNode string
							endNode   string
							count     int64
						}{
							startNode: startLabel,
							endNode:   endLabel,
							count:     count,
						}
					}
				}
				mu.Unlock()

				if err = relResult.Err(); err != nil {
					return fmt.Errorf("relationship count result error: %w", err)
				}

				// Sample properties for each relationship type
				for _, relType := range relTypesList {
					propQuery := fmt.Sprintf(`
						MATCH ()-[r:%s]->()
						WITH r LIMIT $sampleSize
						WHERE size(keys(r)) > 0
						UNWIND keys(r) AS key
						WITH key, r[key] AS value
						WHERE value IS NOT NULL
						RETURN key, COLLECT(DISTINCT valueType(value)) AS types
					`, relType)

					var propResult neo4j.ResultWithContext
					propResult, err = session.Run(ctx, propQuery, map[string]any{"sampleSize": sampleSize})
					if err != nil {
						return fmt.Errorf("relationship properties query for type %s failed: %w", relType, err)
					}

					mu.Lock()
					if relPropsMap[relType] == nil {
						relPropsMap[relType] = make(map[string]map[string]bool)
					}

					for propResult.Next(ctx) {
						record := propResult.Record()
						key := record.Values[0].(string)
						types := record.Values[1].([]any)

						if relPropsMap[relType][key] == nil {
							relPropsMap[relType][key] = make(map[string]bool)
						}

						for _, t := range types {
							relPropsMap[relType][key][t.(string)] = true
						}
					}
					mu.Unlock()

					if err = propResult.Err(); err != nil {
						return fmt.Errorf("relationship properties result error for type %s: %w", relType, err)
					}
				}

				return nil
			},
		},
	}

	// Execute all tasks concurrently
	var wg sync.WaitGroup
	wg.Add(len(tasks))

	for _, task := range tasks {
		go func(task Task) {
			defer wg.Done()

			// Create a new session for this Task
			session := t.Driver.NewSession(ctx, neo4j.SessionConfig{
				DatabaseName: t.Database,
			})
			defer session.Close(ctx)

			// Execute the Task
			if err := task.fn(session); err != nil {
				handleError(fmt.Errorf("task %s failed: %w", task.name, err))
			}
		}(task)
	}

	// Wait for all tasks to complete
	wg.Wait()

	// Check if any error occurred
	if firstErr != nil {
		return nil, nil, nil, firstErr
	}

	// Process the collected data into the final format
	nodeLabels, relationships, stats := processNoneAPOCSchema(nodeCounts, nodePropsMap, relCounts, relPropsMap, relConnectivity)

	return nodeLabels, relationships, stats, nil
}

// extractDatabaseInfo extracts general database information
func (t Tool) extractDatabaseInfo(ctx context.Context) (*DatabaseInfo, error) {
	session := t.Driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: t.Database})
	defer session.Close(ctx)

	result, err := session.Run(ctx, "CALL dbms.components() YIELD name, versions, edition", nil)
	if err != nil {
		return nil, err
	}

	dbInfo := &DatabaseInfo{}
	if result.Next(ctx) {
		record := result.Record()
		dbInfo.Name = record.Values[0].(string)
		versions := record.Values[1].([]any)
		if len(versions) > 0 {
			dbInfo.Version = versions[0].(string)
		}
		dbInfo.Edition = record.Values[2].(string)
	}

	return dbInfo, result.Err()
}

// extractConstraints extracts all constraints with their names
func (t Tool) extractConstraints(ctx context.Context) ([]Constraint, error) {
	session := t.Driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: t.Database})
	defer session.Close(ctx)

	result, err := session.Run(ctx, "SHOW CONSTRAINTS", nil)
	if err != nil {
		return nil, err
	}

	var constraints []Constraint
	for result.Next(ctx) {
		record := result.Record()
		constraint := Constraint{
			Name:       getStringValue(record.AsMap()["name"]),
			Type:       getStringValue(record.AsMap()["type"]),
			EntityType: getStringValue(record.AsMap()["entityType"]),
		}

		if labels, ok := record.AsMap()["labelsOrTypes"].([]any); ok && len(labels) > 0 {
			constraint.Label = labels[0].(string)
		}

		if props, ok := record.AsMap()["properties"].([]any); ok {
			constraint.Properties = convertToStringSlice(props)
		}

		constraints = append(constraints, constraint)
	}

	return constraints, result.Err()
}

// extractIndexes extracts all indexes with their names
func (t Tool) extractIndexes(ctx context.Context) ([]Index, error) {
	session := t.Driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: t.Database})
	defer session.Close(ctx)

	result, err := session.Run(ctx, "SHOW INDEXES", nil)
	if err != nil {
		return nil, err
	}

	var indexes []Index
	for result.Next(ctx) {
		record := result.Record()
		index := Index{
			Name:       getStringValue(record.AsMap()["name"]),
			State:      getStringValue(record.AsMap()["state"]),
			Type:       getStringValue(record.AsMap()["type"]),
			EntityType: getStringValue(record.AsMap()["entityType"]),
		}

		if labels, ok := record.AsMap()["labelsOrTypes"].([]any); ok && len(labels) > 0 {
			index.Label = labels[0].(string)
		}

		if props, ok := record.AsMap()["properties"].([]any); ok {
			index.Properties = convertToStringSlice(props)
		}

		indexes = append(indexes, index)
	}

	return indexes, result.Err()
}
