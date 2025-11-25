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
package lookergetconnectiontablecolumns

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	lookersrc "github.com/googleapis/genai-toolbox/internal/sources/looker"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/looker/lookercommon"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"

	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const kind string = "looker-get-connection-table-columns"

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
	Name         string                 `yaml:"name" validate:"required"`
	Kind         string                 `yaml:"kind" validate:"required"`
	Source       string                 `yaml:"source" validate:"required"`
	Description  string                 `yaml:"description" validate:"required"`
	AuthRequired []string               `yaml:"authRequired"`
	Annotations  *tools.ToolAnnotations `yaml:"annotations,omitempty"`
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
	s, ok := rawS.(*lookersrc.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `looker`", kind)
	}

	connParameter := parameters.NewStringParameter("conn", "The connection containing the tables.")
	dbParameter := parameters.NewStringParameterWithRequired("db", "The optional database to search", false)
	schemaParameter := parameters.NewStringParameter("schema", "The schema containing the tables.")
	tablesParameter := parameters.NewStringParameter("tables", "A comma separated list of tables containing the columns.")
	params := parameters.Parameters{connParameter, dbParameter, schemaParameter, tablesParameter}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, cfg.Annotations)

	// finish tool setup
	return Tool{
		Config:              cfg,
		Parameters:          params,
		UseClientOAuth:      s.UseClientAuthorization(),
		AuthTokenHeaderName: s.GetAuthTokenHeaderName(),
		Client:              s.Client,
		ApiSettings:         s.ApiSettings,
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
	UseClientOAuth      bool
	AuthTokenHeaderName string
	Client              *v4.LookerSDK
	ApiSettings         *rtl.ApiSettings
	Parameters          parameters.Parameters `yaml:"parameters"`
	manifest            tools.Manifest
	mcpManifest         tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}
	mapParams := params.AsMap()
	conn, ok := mapParams["conn"].(string)
	if !ok {
		return nil, fmt.Errorf("'conn' must be a string, got %T", mapParams["conn"])
	}
	db, _ := mapParams["db"].(string)
	schema, ok := mapParams["schema"].(string)
	if !ok {
		return nil, fmt.Errorf("'schema' must be a string, got %T", mapParams["schema"])
	}
	tables, ok := mapParams["tables"].(string)
	if !ok {
		return nil, fmt.Errorf("'tables' must be a string, got %T", mapParams["tables"])
	}

	sdk, err := lookercommon.GetLookerSDK(t.UseClientOAuth, t.ApiSettings, t.Client, accessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting sdk: %w", err)
	}
	req := v4.RequestConnectionColumns{
		ConnectionName: conn,
		SchemaName:     &schema,
		TableNames:     &tables,
	}
	if db != "" {
		req.Database = &db
	}
	resp, err := sdk.ConnectionColumns(req, t.ApiSettings)
	if err != nil {
		return nil, fmt.Errorf("error making get_connection_table_columns request: %s", err)
	}
	var data []any
	for _, t := range resp {
		vMap := make(map[string]any)
		vMap["table_name"] = *t.Name
		vMap["sql_escaped_table_name"] = *t.SqlEscapedName
		vMap["schema_name"] = *t.SchemaName
		var columnData []any
		for _, c := range *t.Columns {
			vMap2 := make(map[string]any)
			vMap2["column_name"] = *c.Name
			vMap2["sql_escaped_column_name"] = *c.SqlEscapedName
			vMap2["data_type_database"] = *c.DataTypeDatabase
			vMap2["data_type_looker"] = *c.DataTypeLooker
			columnData = append(columnData, vMap2)
		}
		vMap["columns"] = columnData
		data = append(data, vMap)
	}
	logger.DebugContext(ctx, "data = ", data)

	return data, nil
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
	return t.UseClientOAuth
}

func (t Tool) GetAuthTokenHeaderName() string {
	return t.AuthTokenHeaderName
}
