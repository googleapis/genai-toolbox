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

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	neo4jsc "github.com/googleapis/genai-toolbox/internal/sources/neo4j"
	"github.com/googleapis/genai-toolbox/internal/tools"
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
	Name                 string   `yaml:"name" validate:"required"`
	Kind                 string   `yaml:"kind" validate:"required"`
	Source               string   `yaml:"source" validate:"required"`
	Description          string   `yaml:"description" validate:"required"`
	AuthRequired         []string `yaml:"authRequired"`
	DisableDbInfo        bool     `yaml:"disableDbInfo"`        // If true, skips extracting database info (like version, edition)
	DisableErrors        bool     `yaml:"disableErrors"`        // If true, skips collecting errors during schema extraction
	DisableIndexes       bool     `yaml:"disableIndexes"`       // If true, skips extracting indexes
	DisableConstraints   bool     `yaml:"disableConstraints"`   // If true, skips extracting constraints
	DisableRelationships bool     `yaml:"disableRelationships"` // If true, skips extracting relationships outside of APOC schema
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

	Driver      neo4j.DriverWithContext
	Database    string
	conf        *Config
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	schema, err := t.extractSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract database schema: %w", err)
	}
	return []any{schema}, nil
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
	var lock sync.Mutex

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

				lock.Lock()
				defer lock.Unlock()
				schema.DatabaseInfo = *dbInfo
				return nil
			},
		},
		{
			name: "APOC schema",
			fn: func() error {
				apocSchema, err := t.GetAPOCSchema(ctx)
				if err != nil {
					return fmt.Errorf("failed to get APOC schema: %w", err)
				}
				nodeLabels, relationships, stats := processAPOCSchema(apocSchema)

				lock.Lock()
				defer lock.Unlock()
				schema.NodeLabels = nodeLabels
				schema.Statistics = *stats

				// If outside APOC relationships extraction is disabled, we are adding APOC relationships to the schema
				if t.conf.DisableRelationships {
					schema.Relationships = relationships
				}
				return nil
			},
		},
		{
			name:     "relationships",
			disabled: t.conf.DisableRelationships,
			fn: func() error {
				relationships, err := t.extractRelationships(ctx)
				if err != nil {
					return fmt.Errorf("failed to extract relationships: %w", err)
				}
				schema.Relationships = relationships
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

				lock.Lock()
				defer lock.Unlock()
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

				lock.Lock()
				defer lock.Unlock()
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
func (t Tool) GetAPOCSchema(ctx context.Context) (*APOCSchemaResult, error) {
	session := t.Driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: t.Database})
	defer session.Close(ctx)

	result, err := session.Run(ctx, "CALL apoc.meta.schema({sample: 10}) YIELD value RETURN value", nil)
	if err != nil {
		return nil, err
	}

	if !result.Next(ctx) {
		return nil, fmt.Errorf("no schema result returned")
	}

	// The result is a map of entity names to their metadata
	schemaMap, ok := result.Record().Values[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected schema format")
	}

	// Marshal the schema map to APOCSchemaResult structure
	apocSchemaResult, err := mapToAPOCSchema(schemaMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert schema map to APOCSchemaResult: %w", err)
	}

	return apocSchemaResult, result.Err()
}

// extractRelationships extracts active relationships from database to fill gaps in APOC schema
func (t Tool) extractRelationships(ctx context.Context) ([]Relationship, error) {
	session := t.Driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: t.Database})
	defer session.Close(ctx)

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
		return nil, fmt.Errorf("failed to extract relationships: %w", err)
	}

	var relationships []Relationship
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

		relationships = append(relationships, relationship)
	}
	return relationships, nil
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
