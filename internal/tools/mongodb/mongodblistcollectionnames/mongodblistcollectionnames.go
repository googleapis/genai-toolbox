// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mongodblistcollectionnames

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	mongosrc "github.com/googleapis/genai-toolbox/internal/sources/mongodb"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const kind string = "mongodb-list-collection-names"

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

type Config struct {
	Name         string           `yaml:"name" validate:"required"`
	Kind         string           `yaml:"kind" validate:"required"`
	Source       string           `yaml:"source" validate:"required"`
	AuthRequired []string         `yaml:"authRequired" validate:"required"`
	Description  string           `yaml:"description" validate:"required"`
	Database     string           `yaml:"database"`
	Params       tools.Parameters `yaml:"params"`
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
	s, ok := rawS.(*mongosrc.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `mongodb`", kind)
	}

	// Check for duplicate parameters
	err := tools.CheckDuplicateParameters(cfg.Params)
	if err != nil {
		return nil, err
	}

	// Determine if database parameter is needed
	var allParams tools.Parameters
	if cfg.Database == "" {
		// Database not specified in config, so add it as a parameter
		databaseParam := tools.Parameter{
			Name:        "database",
			Description: "The name of the database to list collections from",
			Type:        "string",
			Required:    true,
		}
		allParams = append(tools.Parameters{databaseParam}, cfg.Params...)
	} else {
		// Database is specified in config, use provided params
		allParams = cfg.Params
	}

	// Create Toolbox manifest
	paramManifest := allParams.Manifest()
	if paramManifest == nil {
		paramManifest = make([]tools.ParameterManifest, 0)
	}

	// Create MCP manifest
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParams)

	// finish tool setup
	return Tool{
		Name:          cfg.Name,
		Kind:          kind,
		Description:   cfg.Description,
		AuthRequired:  cfg.AuthRequired,
		Database:      cfg.Database,
		AllParams:     allParams,
		client:        s.Client,
		manifest:      tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:   mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string `yaml:"name"`
	Kind         string `yaml:"kind"`
	Description  string `yaml:"description"`
	AuthRequired []string
	Database     string
	AllParams    tools.Parameters

	client      *mongo.Client
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	// Determine database name
	var databaseName string
	if t.Database != "" {
		databaseName = t.Database
	} else {
		// Get database name from parameters
		paramsMap := params.AsMap()
		dbParam, ok := paramsMap["database"]
		if !ok {
			return nil, fmt.Errorf("database parameter is required")
		}
		databaseName, ok = dbParam.(string)
		if !ok {
			return nil, fmt.Errorf("database parameter must be a string")
		}
	}

	// List all collections in the database
	database := t.client.Database(databaseName)
	collectionNames, err := database.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("error listing collections in database %q: %w", databaseName, err)
	}

	return collectionNames, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.AllParams, data, claims)
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
	return false
}
