// Copyright 2026 Google LLC
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
package lookercreateviewfromtable

import (
	"context"
	"fmt"
	"net/http"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/looker/lookercommon"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"

	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const resourceType string = "looker-create-view-from-table"

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
	UseClientAuthorization() bool
	GetAuthTokenHeaderName() string
	LookerApiSettings() *rtl.ApiSettings
	GetLookerSDK(string) (*v4.LookerSDK, error)
}

type Config struct {
	Name         string                 `yaml:"name" validate:"required"`
	Type         string                 `yaml:"type" validate:"required"`
	Source       string                 `yaml:"source" validate:"required"`
	Description  string                 `yaml:"description" validate:"required"`
	AuthRequired []string               `yaml:"authRequired"`
	Annotations  *tools.ToolAnnotations `yaml:"annotations,omitempty"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	projectIdParameter := parameters.NewStringParameter("project_id", "The id of the project to create the view in.")
	connectionParameter := parameters.NewStringParameter("connection", "The database connection name.")

	tableDef := parameters.NewMapParameter("table", "Table definition.", "")
	tablesParameter := parameters.NewArrayParameter("tables", `The tables to generate views for.
		Each item must be a map with:
		- schema (string, required)
		- table_name (string, required)
		- primary_key (string, optional)
		- base_view (boolean, optional)
		- columns (array of objects, optional): Each object must have 'column_name' (string).`, tableDef)

	folderNameParameter := parameters.NewStringParameterWithDefault("folder_name", "views", "The folder to place the view files in (e.g., 'views').")

	params := parameters.Parameters{projectIdParameter, connectionParameter, tablesParameter, folderNameParameter}

	annotations := cfg.Annotations
	if annotations == nil {
		readOnlyHint := false
		annotations = &tools.ToolAnnotations{
			ReadOnlyHint: &readOnlyHint,
		}
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, annotations)

	// finish tool setup
	return Tool{
		Config:     cfg,
		Parameters: params,
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   params.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	Parameters  parameters.Parameters `yaml:"parameters"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, util.NewClientServerError(fmt.Sprintf("error getting logger from context: %s", err), http.StatusInternalServerError, err)
	}

	sdk, err := source.GetLookerSDK(string(accessToken))
	if err != nil {
		return nil, util.NewClientServerError(fmt.Sprintf("error getting sdk: %v", err), http.StatusInternalServerError, err)
	}

	mapParams := params.AsMap()
	projectId, ok := mapParams["project_id"].(string)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("'project_id' must be a string, got %T", mapParams["project_id"]), nil)
	}
	connection, ok := mapParams["connection"].(string)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("'connection' must be a string, got %T", mapParams["connection"]), nil)
	}
	folderName, ok := mapParams["folder_name"].(string)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("'folder_name' must be a string, got %T", mapParams["folder_name"]), nil)
	}

	tablesSlice, ok := mapParams["tables"].([]any)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("'tables' must be an array, got %T", mapParams["tables"]), nil)
	}

	logger.DebugContext(ctx, "generating views with request", "tables", tablesSlice)

	var generatorTables []lookercommon.ProjectGeneratorTable
	for _, tRaw := range tablesSlice {
		t, ok := tRaw.(map[string]any)
		if !ok {
			return nil, util.NewClientServerError(fmt.Sprintf("expected map in tables list, got %T", tRaw), http.StatusInternalServerError, nil)
		}

		var schema, tableName string
		var primaryKey *string
		var baseView *bool
		var columns []lookercommon.ProjectGeneratorColumn

		if s, ok := t["schema"].(string); ok {
			schema = s
		}
		if tn, ok := t["table_name"].(string); ok {
			tableName = tn
		}
		// Enforce required fields for map input
		if schema == "" || tableName == "" {
			return nil, util.NewClientServerError("schema and table_name are required in table map", http.StatusInternalServerError, nil)
		}

		if pk, ok := t["primary_key"].(string); ok {
			primaryKey = &pk
		}
		if bv, ok := t["base_view"].(bool); ok {
			baseView = &bv
		}
		if colsRaw, ok := t["columns"].([]any); ok {
			for _, cRaw := range colsRaw {
				if cMap, ok := cRaw.(map[string]any); ok {
					if cName, ok := cMap["column_name"].(string); ok {
						columns = append(columns, lookercommon.ProjectGeneratorColumn{ColumnName: cName})
					}
				}
			}
		}

		if tableName == "" {
			continue // Skip invalid entries
		}

		generatorTables = append(generatorTables, lookercommon.ProjectGeneratorTable{
			Schema:     schema,
			TableName:  tableName,
			PrimaryKey: primaryKey,
			BaseView:   baseView,
			Columns:    columns,
		})
	}

	queryParams := lookercommon.ProjectGeneratorQueryParams{
		Connection:          connection,
		FileTypeForExplores: "none",
		FolderName:          folderName,
	}

	reqBody := lookercommon.ProjectGeneratorRequestBody{
		Tables: generatorTables,
	}

	logger.DebugContext(ctx, "generating views with request", "query", queryParams, "body", reqBody)

	err = lookercommon.CreateViewsFromTables(ctx, sdk, projectId, queryParams, reqBody, source.LookerApiSettings())
	if err != nil {
		return nil, util.NewClientServerError(fmt.Sprintf("error generating views: %s", err), http.StatusInternalServerError, err)
	}

	return map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Triggered view generation for project %s in folder %s", projectId, folderName),
	}, nil
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.Parameters, paramValues, embeddingModelsMap, nil)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return "", err
	}
	return source.GetAuthTokenHeaderName(), nil
}

func (t Tool) GetParameters() parameters.Parameters {
	return t.Parameters
}
