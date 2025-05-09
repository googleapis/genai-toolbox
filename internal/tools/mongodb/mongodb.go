package mongodb

import (
	"context"
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/sources"
	mongodbsrc "github.com/googleapis/genai-toolbox/internal/sources/mongodb"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"go.mongodb.org/mongo-driver/bson" // Import BSON for MongoDB operations
	"go.mongodb.org/mongo-driver/mongo"
	"gopkg.in/yaml.v3" // Import YAML for YAML parsing and generation
)

const ToolKind string = "mongodb-atlas"

type Config struct {
	Name         string           `yaml:"name" validate:"required"`
	Kind         string           `yaml:"kind" validate:"required"`
	Source       string           `yaml:"source" validate:"required"`
	Description  string           `yaml:"description" validate:"required"`
	Collection   string           `yaml:"collection" validate:"required"`
	Operation    string           `yaml:"operation" validate:"required"`
	Query        map[string]any   `yaml:"query" validate:"required"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return ToolKind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(*mongodbsrc.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool", ToolKind)
	}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: cfg.Parameters.McpManifest(),
	}

	// finish tool setup
	t := Tool{
		Name:         cfg.Name,
		Kind:         ToolKind,
		Parameters:   cfg.Parameters,
		Collection:   cfg.Collection,
		Operation:    cfg.Operation,
		Query:        cfg.Query,
		AuthRequired: cfg.AuthRequired,
		Client:       s.Client,
		Database:     s.DatabaseName(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: cfg.Parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`
	Operation    string           `yaml:"operation"`

	Client      *mongo.Client
	Database    string
	Collection  string
	Query       map[string]any
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	collection := t.Client.Database(t.Database).Collection(t.Collection)

	// Check the operation type

	switch t.Operation {

	case "find":
		// Extract the filter for the find operation
		filter := t.Query

		// Apply parameters to the query
		for key, value := range params.AsMap() {
			filter[key] = value
		}

		// Perform the find operation
		cursor, err := collection.Find(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("unable to execute query: %w", err)
		}
		defer cursor.Close(ctx)

		var results []any
		for cursor.Next(ctx) {
			var doc map[string]any
			if err := cursor.Decode(&doc); err != nil {
				return nil, fmt.Errorf("unable to parse document: %w", err)
			}
			results = append(results, doc)
		}

		if err := cursor.Err(); err != nil {
			return nil, fmt.Errorf("cursor error: %w", err)
		}

		return results, nil

	case "vectorSearch":
		// Extract the pipeline for the aggregate operation
		pipeline := make([]bson.M, 0)
		filter := t.Query

		for key, value := range params.AsMap() {
			filter[key] = value
		}

		// Add the $match stage to the pipeline

		// Add the $vectorsearch stage to the pipeline
		indexName, ok := params.AsMap()["indexName"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid or missing indexName for aggregate operation")
		}
		embeddings, ok := params.AsMap()["embeddings"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid or missing embeddings for aggregate operation")
		}
		path, ok := params.AsMap()["path"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid or missing path for aggregate operation")
		}
		pipeline = append(pipeline, bson.M{
			"$vectorSearch": bson.M{
				"index":         indexName,
				"queryVector":   embeddings,
				"path":          path,
				"numCandidates": 10,
				"limit":         10,
			},
		})

		// Perform the aggregate operation
		cursor, err := collection.Aggregate(ctx, pipeline)
		if err != nil {
			return nil, fmt.Errorf("unable to execute aggregate query: %w", err)
		}
		defer cursor.Close(ctx)

		var results []any
		for cursor.Next(ctx) {
			var doc map[string]any
			if err := cursor.Decode(&doc); err != nil {
				return nil, fmt.Errorf("unable to parse document: %w", err)
			}
			// delete the embedding field from the doc
			delete(doc, path)
			results = append(results, doc)
		}

		if err := cursor.Err(); err != nil {
			return nil, fmt.Errorf("cursor error: %w", err)
		}

		return results, nil
	default:
		return nil, fmt.Errorf("unsupported operation: %s", t.Operation)
	}

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
func (t Tool) GenerateAggregateYAML(pipeline []bson.M) (string, error) {
	// Create a map to hold the aggregate configuration
	config := make(map[string]interface{})
	config["name"] = t.Name
	config["kind"] = t.Kind
	config["description"] = t.manifest.Description
	config["collection"] = t.Collection
	config["operation"] = t.Operation
	config["query"] = t.Query
	config["authRequired"] = t.AuthRequired

	// Add the pipeline to the configuration
	config["pipeline"] = pipeline

	// Convert the configuration to YAML
	yamlBytes, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to generate YAML: %w", err)
	}

	return string(yamlBytes), nil
}
