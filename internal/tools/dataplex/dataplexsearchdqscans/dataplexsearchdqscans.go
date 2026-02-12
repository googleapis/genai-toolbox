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

package dataplexsearchdqscans

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/dataplex/apiv1/dataplexpb"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const resourceType string = "dataplex-search-dq-scans"

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
	SearchDataQualityScans(context.Context, string, int, string) ([]*dataplexpb.DataScan, error)
}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Type         string   `yaml:"type" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	filter := parameters.NewStringParameterWithDefault("filter", "", "Optional. Filter string to search/filter data quality scans. E.g. \"display_name = \\\"my-scan\\\"\"")
	dataScanID := parameters.NewStringParameterWithDefault("data_scan_id", "", "Optional. The ID of the data scan to filter by.")
	tableName := parameters.NewStringParameterWithDefault("table_name", "", "Optional. The name of the table to filter by.")
	pageSize := parameters.NewIntParameterWithDefault("pageSize", 10, "Number of returned data quality scans in the page.")
	orderBy := parameters.NewStringParameterWithDefault("orderBy", "", "Specifies the ordering of results.")
	params := parameters.Parameters{filter, dataScanID, tableName, pageSize, orderBy}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, nil)

	t := Tool{
		Config:     cfg,
		Parameters: params,
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   params.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

type Tool struct {
	Config
	Parameters  parameters.Parameters
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return nil, err
	}
	paramsMap := params.AsMap()
	filter, _ := paramsMap["filter"].(string)
	dataScanID, _ := paramsMap["data_scan_id"].(string)
	tableName, _ := paramsMap["table_name"].(string)
	pageSize, _ := paramsMap["pageSize"].(int)
	orderBy, _ := paramsMap["orderBy"].(string)

	var filters []string
	if filter != "" {
		filters = append(filters, filter)
	}
	if dataScanID != "" {
		// assuming data_scan_id usually maps to name or id filter if supported, or display_name
		// referencing user request "parameters could be datascan name"
		// The generic filter "resource.name" or "display_name" is often used.
		// Let's assume display_name for user convenience or name if full resource name.
		// If it's just ID, we might need wildcard?
		// Actually, `id` might be part of the resource name.
		// Let's use `display_name` as it's more likely what user means by "name".
		// Or if they mean ID, it might be `resource.name : id`.
		// Let's try `display_name = "ID"` first as safe bet or just append to filter.
		filters = append(filters, fmt.Sprintf("display_name = %q", dataScanID))
	}
	if tableName != "" {
		// "data.entity" is typically used for table in DataScan filters
		filters = append(filters, fmt.Sprintf("data.entity = %q", tableName))
	}

	finalFilter := strings.Join(filters, " AND ")

	return source.SearchDataQualityScans(ctx, finalFilter, pageSize, orderBy)
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.Parameters, paramValues, embeddingModelsMap, nil)
}

func (t Tool) Manifest() tools.Manifest {
	// Returns the tool manifest
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	// Returns the tool MCP manifest
	return t.mcpManifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	return false, nil
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}

func (t Tool) GetParameters() parameters.Parameters {
	return t.Parameters
}
