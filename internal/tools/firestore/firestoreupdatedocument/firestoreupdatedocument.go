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

package firestoreupdatedocument

import (
	"context"
	"fmt"
	"strings"

	firestoreapi "cloud.google.com/go/firestore"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	firestoreds "github.com/googleapis/genai-toolbox/internal/sources/firestore"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/firestore/util"
)

const kind string = "firestore-update-document"
const documentPathKey string = "documentPath"
const documentDataKey string = "documentData"
const updateMaskKey string = "updateMask"
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
	documentPathParameter := tools.NewStringParameter(
		documentPathKey,
		"The path of the document which needs to be updated",
	)

	documentDataParameter := tools.NewMapParameter(
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

	updateMaskParameter := tools.NewArrayParameterWithRequired(
		updateMaskKey,
		"The selective fields to update. If not provided, all fields in documentData will be updated. When provided, only the specified fields will be updated. Fields referenced in the mask but not present in documentData will be deleted from the document",
		false, // not required
		tools.NewStringParameter("field", "Field path to update or delete. Use dot notation to access nested fields within maps (e.g., 'address.city' to update the city field within an address map, or 'user.profile.name' for deeply nested fields). To delete a field, include it in the mask but omit it from documentData. Note: You cannot update individual array elements; you must update the entire array field"),
	)

	returnDataParameter := tools.NewBooleanParameterWithDefault(
		returnDocumentDataKey,
		false,
		"If set to true the output will have the data of the updated document. This flag if set to false will help avoid overloading the context of the agent.",
	)

	parameters := tools.Parameters{
		documentPathParameter,
		documentDataParameter,
		updateMaskParameter,
		returnDataParameter,
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
		Client:       s.FirestoreClient(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
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

	Client      *firestoreapi.Client
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) (any, error) {
	mapParams := params.AsMap()

	// Get document path
	documentPath, ok := mapParams[documentPathKey].(string)
	if !ok || documentPath == "" {
		return nil, fmt.Errorf("invalid or missing '%s' parameter", documentPathKey)
	}

	// Get document data
	documentDataRaw, ok := mapParams[documentDataKey]
	if !ok {
		return nil, fmt.Errorf("invalid or missing '%s' parameter", documentDataKey)
	}

	// Get update mask if provided
	var updatePaths []string
	if updateMaskRaw, ok := mapParams[updateMaskKey]; ok && updateMaskRaw != nil {
		if updateMaskArray, ok := updateMaskRaw.([]any); ok {
			// Use ConvertAnySliceToTyped to convert the slice
			typedSlice, err := tools.ConvertAnySliceToTyped(updateMaskArray, "string")
			if err != nil {
				return nil, fmt.Errorf("failed to convert update mask: %w", err)
			}
			updatePaths, ok = typedSlice.([]string)
			if !ok {
				return nil, fmt.Errorf("unexpected type conversion error for update mask")
			}
		}
	}

	// Get return document data flag
	returnData := false
	if val, ok := mapParams[returnDocumentDataKey].(bool); ok {
		returnData = val
	}

	// Get the document reference
	docRef := t.Client.Doc(documentPath)

	// Prepare update data
	var writeResult *firestoreapi.WriteResult
	var writeErr error

	if len(updatePaths) > 0 {
		// Use selective field update with update mask
		updates := make([]firestoreapi.Update, 0, len(updatePaths))

		// Convert document data without delete markers
		dataMap, err := util.JSONToFirestoreValue(documentDataRaw, t.Client)
		if err != nil {
			return nil, fmt.Errorf("failed to convert document data: %w", err)
		}

		// Ensure it's a map
		dataMapTyped, ok := dataMap.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("document data must be a map")
		}

		for _, path := range updatePaths {
			// Get the value for this path from the document data
			value, exists := getFieldValue(dataMapTyped, path)
			if !exists {
				// Field not in document data but in mask - delete it
				value = firestoreapi.Delete
			}

			updates = append(updates, firestoreapi.Update{
				Path:  path,
				Value: value,
			})
		}

		writeResult, writeErr = docRef.Update(ctx, updates)
	} else {
		// Update all fields in the document data (merge)
		documentData, err := util.JSONToFirestoreValue(documentDataRaw, t.Client)
		if err != nil {
			return nil, fmt.Errorf("failed to convert document data: %w", err)
		}
		writeResult, writeErr = docRef.Set(ctx, documentData, firestoreapi.MergeAll)
	}

	if writeErr != nil {
		return nil, fmt.Errorf("failed to update document: %w", writeErr)
	}

	// Build the response
	response := map[string]any{
		"documentPath": docRef.Path,
		"updateTime":   writeResult.UpdateTime.Format("2006-01-02T15:04:05.999999999Z"),
	}

	// Add document data if requested
	if returnData {
		// Fetch the updated document to return the current state
		snapshot, err := docRef.Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve updated document: %w", err)
		}

		// Convert the document data to simple JSON format
		simplifiedData := util.FirestoreValueToJSON(snapshot.Data())
		response["documentData"] = simplifiedData
	}

	return response, nil
}


// getFieldValue retrieves a value from a nested map using a dot-separated path
func getFieldValue(data map[string]interface{}, path string) (interface{}, bool) {
	// Split the path by dots for nested field access
	parts := strings.Split(path, ".")

	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - return the value
			if value, exists := current[part]; exists {
				return value, true
			}
			return nil, false
		}

		// Navigate deeper into the structure
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return nil, false
		}
	}

	return nil, false
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
