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

package firestoregetdocuments

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

const kind string = "firestore-get-documents"
const documentPathsKey string = "documentPaths"

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

	documentPathsParameter := parameters.NewArrayParameter(documentPathsKey, "Array of relative document paths to retrieve from Firestore (e.g., 'users/userId' or 'users/userId/posts/postId'). Note: These are relative paths, NOT absolute paths like 'projects/{project_id}/databases/{database_id}/documents/...'", parameters.NewStringParameter("item", "Relative document path"))
	params := parameters.Parameters{documentPathsParameter}

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
	documentPathsRaw, ok := mapParams[documentPathsKey].([]any)
	if !ok {
		return nil, fmt.Errorf("invalid or missing '%s' parameter; expected an array", documentPathsKey)
	}

	if len(documentPathsRaw) == 0 {
		return nil, fmt.Errorf("'%s' parameter cannot be empty", documentPathsKey)
	}

	// Use ConvertAnySliceToTyped to convert the slice
	typedSlice, err := parameters.ConvertAnySliceToTyped(documentPathsRaw, "string")
	if err != nil {
		return nil, fmt.Errorf("failed to convert document paths: %w", err)
	}

	documentPaths, ok := typedSlice.([]string)
	if !ok {
		return nil, fmt.Errorf("unexpected type conversion error for document paths")
	}

	// Validate each document path
	for i, path := range documentPaths {
		if err := util.ValidateDocumentPath(path); err != nil {
			return nil, fmt.Errorf("invalid document path at index %d: %w", i, err)
		}
	}

	// Create document references from paths
	docRefs := make([]*firestoreapi.DocumentRef, len(documentPaths))
	for i, path := range documentPaths {
		docRefs[i] = t.Client.Doc(path)
	}

	// Get all documents
	snapshots, err := t.Client.GetAll(ctx, docRefs)
	if err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}

	// Convert snapshots to response data
	results := make([]any, len(snapshots))
	for i, snapshot := range snapshots {
		docData := make(map[string]any)
		docData["path"] = documentPaths[i]
		docData["exists"] = snapshot.Exists()

		if snapshot.Exists() {
			docData["data"] = snapshot.Data()
			docData["createTime"] = snapshot.CreateTime
			docData["updateTime"] = snapshot.UpdateTime
			docData["readTime"] = snapshot.ReadTime
		}

		results[i] = docData
	}

	return results, nil
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
