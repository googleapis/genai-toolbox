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

package firestoredeletedocuments

import (
	"context"
	"fmt"

	firestoreapi "cloud.google.com/go/firestore"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	firestoreds "github.com/googleapis/genai-toolbox/internal/sources/firestore"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "firestore-delete-documents"
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

	documentPathsParameter := tools.NewArrayParameter(documentPathsKey, "Array of document paths to delete from Firestore.", tools.NewStringParameter("item", "Document path"))
	parameters := tools.Parameters{documentPathsParameter}

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

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	mapParams := params.AsMap()
	
	var documentPaths []string
	
	switch v := mapParams[documentPathsKey].(type) {
	case []string:
		documentPaths = v
	default:
		return nil, fmt.Errorf("invalid or missing '%s' parameter; expected an array of strings", documentPathsKey)
	}

	if len(documentPaths) == 0 {
		return nil, fmt.Errorf("'%s' parameter cannot be empty", documentPathsKey)
	}

	// Create a BulkWriter to handle multiple deletions efficiently
	bulkWriter := t.Client.BulkWriter(ctx)

	// Keep track of jobs for each document
	jobs := make([]*firestoreapi.BulkWriterJob, len(documentPaths))

	// Add all delete operations to the BulkWriter
	for i, path := range documentPaths {
		docRef := t.Client.Doc(path)
		job, err := bulkWriter.Delete(docRef)
		if err != nil {
			return nil, fmt.Errorf("failed to add delete operation for document %q: %w", path, err)
		}
		jobs[i] = job
	}

	// End the BulkWriter to execute all operations
	bulkWriter.End()

	// Collect results
	results := make([]any, len(documentPaths))
	for i, job := range jobs {
		docData := make(map[string]any)
		docData["path"] = documentPaths[i]

		// Wait for the job to complete and get the result
		_, err := job.Results()
		if err != nil {
			docData["success"] = false
			docData["error"] = err.Error()
		} else {
			docData["success"] = true
		}

		results[i] = docData
	}

	return results, nil
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
