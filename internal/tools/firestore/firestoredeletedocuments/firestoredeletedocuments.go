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
	"net/http"

	firestoreapi "cloud.google.com/go/firestore"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	fsUtil "github.com/googleapis/mcp-toolbox/internal/tools/firestore/util"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const resourceType string = "firestore-delete-documents"
const documentPathsKey string = "documentPaths"

func init() {
	if !tools.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", resourceType))
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
	DeleteDocuments(context.Context, []string) ([]any, error)
}

type Config struct {
	Name         string                 `yaml:"name" validate:"required"`
	Type         string                 `yaml:"type" validate:"required"`
	Source       string                 `yaml:"source" validate:"required"`
	Description  string                 `yaml:"description" validate:"required"`
	AuthRequired []string               `yaml:"authRequired"`
	Annotations  *tools.ToolAnnotations `yaml:"annotations,omitempty"`

	ScopesRequired []string `yaml:"scopesRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	documentPathsParameter := parameters.NewArrayParameter(documentPathsKey, "Array of relative document paths to delete from Firestore (e.g., 'users/userId' or 'users/userId/posts/postId'). Note: These are relative paths, NOT absolute paths like 'projects/{project_id}/databases/{database_id}/documents/...'", parameters.NewStringParameter("item", "Relative document path"))
	params := parameters.Parameters{documentPathsParameter}

	return Tool{
		BaseTool: tools.BaseTool{
			Name:             cfg.Name,
			Description:      cfg.Description,
			Metadata:         tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
			StaticParameters: params,
			ScopesRequired:   cfg.ScopesRequired,
			Annotations:      tools.GetAnnotationsOrDefault(cfg.Annotations, tools.NewDestructiveAnnotations),
		},
		cfg: cfg,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	tools.BaseTool
	cfg Config
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.cfg
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.cfg.Source, t.cfg.Name, t.cfg.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	mapParams := params.AsMap()
	documentPathsRaw, ok := mapParams[documentPathsKey].([]any)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("invalid or missing '%s' parameter; expected an array", documentPathsKey), nil)
	}
	if len(documentPathsRaw) == 0 {
		return nil, util.NewAgentError(fmt.Sprintf("'%s' parameter cannot be empty", documentPathsKey), nil)
	}

	// Use ConvertAnySliceToTyped to convert the slice
	typedSlice, err := parameters.ConvertAnySliceToTyped(documentPathsRaw, "string")
	if err != nil {
		return nil, util.NewAgentError(fmt.Sprintf("failed to convert document paths: %v", err), err)
	}

	documentPaths, ok := typedSlice.([]string)
	if !ok {
		return nil, util.NewAgentError("unexpected type conversion error for document paths", nil)
	}

	// Validate each document path
	for i, path := range documentPaths {
		if err := fsUtil.ValidateDocumentPath(path); err != nil {
			return nil, util.NewAgentError(fmt.Sprintf("invalid document path at index %d: %v", i, err), err)
		}
	}
	resp, err := source.DeleteDocuments(ctx, documentPaths)
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}
	return resp, nil
}

