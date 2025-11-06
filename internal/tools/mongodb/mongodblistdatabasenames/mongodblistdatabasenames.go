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

package mongodblistdatabasenames

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

const kind string = "mongodb-list-database-names"

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
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	AuthRequired []string `yaml:"authRequired" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
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

	// Create Toolbox manifest
	paramManifest := make([]tools.ParameterManifest, 0)

	// Create MCP manifest
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, nil)

	// finish tool setup
	return Tool{
		Name:         cfg.Name,
		Kind:         kind,
		Description:  cfg.Description,
		AuthRequired: cfg.AuthRequired,
		client:       s.Client,
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string `yaml:"name"`
	Kind         string `yaml:"kind"`
	Description  string `yaml:"description"`
	AuthRequired []string

	client      *mongo.Client
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	// List all databases
	databases, err := t.client.ListDatabases(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("error listing databases: %w", err)
	}

	// Extract database names
	var databaseNames []string
	for _, db := range databases.Databases {
		databaseNames = append(databaseNames, db.Name)
	}

	return databaseNames, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(nil, data, claims)
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
