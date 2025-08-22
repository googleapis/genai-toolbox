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
package mongodbinsertone

import (
	"context"
	"errors"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	mongosrc "github.com/googleapis/genai-toolbox/internal/sources/mongodb"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const kind string = "mongodb-insert-one"

const dataParamsKey = "data"

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
	Database     string   `yaml:"database" validate:"required"`
	Collection   string   `yaml:"collection" validate:"required"`
	Canonical    bool     `yaml:"canonical" validate:"required"` //i want to force the user to choose
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

	payloadParams := tools.NewStringParameterWithRequired(dataParamsKey, "the JSON payload to insert, should be a JSON object", true)

	allParameters := tools.Parameters{payloadParams}

	// Create Toolbox manifest
	paramManifest := allParameters.Manifest()

	if paramManifest == nil {
		paramManifest = make([]tools.ParameterManifest, 0)
	}

	// Create MCP manifest
	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: allParameters.McpManifest(),
	}

	// finish tool setup
	return Tool{
		Name:          cfg.Name,
		Kind:          kind,
		AuthRequired:  cfg.AuthRequired,
		Collection:    cfg.Collection,
		Canonical:     cfg.Canonical,
		PayloadParams: allParameters,
		database:      s.Client.Database(cfg.Database),
		manifest:      tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:   mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name          string           `yaml:"name"`
	Kind          string           `yaml:"kind"`
	AuthRequired  []string         `yaml:"authRequired"`
	Description   string           `yaml:"description"`
	Collection    string           `yaml:"collection"`
	Canonical     bool             `yaml:"canonical" validation:"required"`
	PayloadParams tools.Parameters `yaml:"payloadParams" validate:"required"`

	database    *mongo.Database
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	if len(params) == 0 {
		return nil, errors.New("no input found")
	}
	// use the first, assume it's a string
	var jsonData, ok = params[0].Value.(string)
	if !ok {
		return nil, errors.New("no input found")
	}

	var data any
	err := bson.UnmarshalExtJSON([]byte(jsonData), t.Canonical, &data)
	if err != nil {
		return nil, err
	}

	res, err := t.database.Collection(t.Collection).InsertOne(ctx, data, options.InsertOne())
	if err != nil {
		return nil, err
	}

	return res.InsertedID, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.PayloadParams, data, claims)
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
