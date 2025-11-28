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

package firestorelistcollections

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

const kind string = "firestore-list-collections"
const parentPathKey string = "parentPath"

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

	emptyString := ""
	parentPathParameter := parameters.NewStringParameterWithDefault(parentPathKey, emptyString, "Relative parent document path to list subcollections from (e.g., 'users/userId'). If not provided, lists root collections. Note: This is a relative path, NOT an absolute path like 'projects/{project_id}/databases/{database_id}/documents/...'")
	params := parameters.Parameters{parentPathParameter}

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

	var collectionRefs []*firestoreapi.CollectionRef
	var err error

	// Check if parentPath is provided
	parentPath, hasParent := mapParams[parentPathKey].(string)

	if hasParent && parentPath != "" {
		// Validate parent document path
		if err := util.ValidateDocumentPath(parentPath); err != nil {
			return nil, fmt.Errorf("invalid parent document path: %w", err)
		}

		// List subcollections of the specified document
		docRef := t.Client.Doc(parentPath)
		collectionRefs, err = docRef.Collections(ctx).GetAll()
		if err != nil {
			return nil, fmt.Errorf("failed to list subcollections of document %q: %w", parentPath, err)
		}
	} else {
		// List root collections
		collectionRefs, err = t.Client.Collections(ctx).GetAll()
		if err != nil {
			return nil, fmt.Errorf("failed to list root collections: %w", err)
		}
	}

	// Convert collection references to response data
	results := make([]any, len(collectionRefs))
	for i, collRef := range collectionRefs {
		collData := make(map[string]any)
		collData["id"] = collRef.ID
		collData["path"] = collRef.Path

		// If this is a subcollection, include parent information
		if collRef.Parent != nil {
			collData["parent"] = collRef.Parent.Path
		}

		results[i] = collData
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
