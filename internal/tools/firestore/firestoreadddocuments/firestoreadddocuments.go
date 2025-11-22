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

package firestoreadddocuments

import (
	"context"
	"fmt"

	firestoreapi "cloud.google.com/go/firestore"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	firestoreds "github.com/googleapis/genai-toolbox/internal/sources/firestore"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/firestore/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind string = "firestore-add-documents"
const collectionPathKey string = "collectionPath"
const documentDataKey string = "documentData"
const returnDocumentDataKey string = "returnData"

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
	FirestoreClient() *firestoreapi.Client
}

// validate compatible sources are still compatible
var _ compatibleSource = &firestoreds.Source{}

var compatibleSources = [...]string{firestoreds.SourceKind}

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

	// Create parameters
	collectionPathParameter := parameters.NewStringParameter(
		collectionPathKey,
		"The relative path of the collection where the document will be added to (e.g., 'users' or 'users/userId/posts'). Note: This is a relative path, NOT an absolute path like 'projects/{project_id}/databases/{database_id}/documents/...'",
	)

	documentDataParameter := parameters.NewMapParameter(
		documentDataKey,
		`The document data in Firestore's native JSON format. Each field must be wrapped with a type indicator:
- Strings: {"stringValue": "text"}
- Integers: {"integerValue": "123"} or {"integerValue": 123}
- Doubles: {"doubleValue": 123.45}
- Booleans: {"booleanValue": true}
- Timestamps: {"timestampValue": "2025-01-07T10:00:00Z"}
- GeoPoints: {"geoPointValue": {"latitude": 34.05, "longitude": -118.24}}
- Arrays: {"arrayValue": {"values": [{"stringValue": "item1"}, {"integerValue": "2"}]}}
- Maps: {"mapValue": {"fields": {"key1": {"stringValue": "value1"}, "key2": {"booleanValue": true}}}}
- Null: {"nullValue": null}
- Bytes: {"bytesValue": "base64EncodedString"}
- References: {"referenceValue": "collection/document"}`,
		"", // Empty string for generic map that accepts any value type
	)

	returnDataParameter := parameters.NewBooleanParameterWithDefault(
		returnDocumentDataKey,
		false,
		"If set to true the output will have the data of the created document. This flag if set to false will help avoid overloading the context of the agent.",
	)

	params := parameters.Parameters{
		collectionPathParameter,
		documentDataParameter,
		returnDataParameter,
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, nil)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		Parameters:  params,
		Client:      s.FirestoreClient(),
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	Parameters parameters.Parameters `yaml:"parameters"`

	Client      *firestoreapi.Client
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	mapParams := params.AsMap()

	// Get collection path
	collectionPath, ok := mapParams[collectionPathKey].(string)
	if !ok || collectionPath == "" {
		return nil, fmt.Errorf("invalid or missing '%s' parameter", collectionPathKey)
	}

	// Validate collection path
	if err := util.ValidateCollectionPath(collectionPath); err != nil {
		return nil, fmt.Errorf("invalid collection path: %w", err)
	}

	// Get document data
	documentDataRaw, ok := mapParams[documentDataKey]
	if !ok {
		return nil, fmt.Errorf("invalid or missing '%s' parameter", documentDataKey)
	}

	// Convert the document data from JSON format to Firestore format
	// The client is passed to handle referenceValue types
	documentData, err := util.JSONToFirestoreValue(documentDataRaw, t.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to convert document data: %w", err)
	}

	// Get return document data flag
	returnData := false
	if val, ok := mapParams[returnDocumentDataKey].(bool); ok {
		returnData = val
	}

	// Get the collection reference
	collection := t.Client.Collection(collectionPath)

	// Add the document to the collection
	docRef, writeResult, err := collection.Add(ctx, documentData)
	if err != nil {
		return nil, fmt.Errorf("failed to add document: %w", err)
	}

	// Build the response
	response := map[string]any{
		"documentPath": docRef.Path,
		"createTime":   writeResult.UpdateTime.Format("2006-01-02T15:04:05.999999999Z"),
	}

	// Add document data if requested
	if returnData {
		// Convert the document data back to simple JSON format
		simplifiedData := util.FirestoreValueToJSON(documentData)
		response["documentData"] = simplifiedData
	}

	return response, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Parameters, data, claims)
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

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
