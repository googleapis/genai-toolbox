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

package clickhouse

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const listTablesType string = "clickhouse-list-tables"
const databaseKey string = "database"

// validIdentifier matches the ClickHouse unquoted identifier grammar: a letter
// or underscore followed by letters, digits, or underscores. The `database`
// parameter is interpolated directly into a `SHOW TABLES FROM` statement and
// cannot be bound as a positional value, so restricting it to this character
// set is the only safe option short of refactoring to use ClickHouse's
// `system.tables` view.
var validIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func init() {
	if !tools.Register(listTablesType, newListTablesConfig) {
		panic(fmt.Sprintf("tool type %q already registered", listTablesType))
	}
}

func newListTablesConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type compatibleSource interface {
	RunSQL(context.Context, string, parameters.ParamValues) (any, error)
}

type Config struct {
	Name         string                 `yaml:"name" validate:"required"`
	Type         string                 `yaml:"type" validate:"required"`
	Source       string                 `yaml:"source" validate:"required"`
	Description  string                 `yaml:"description" validate:"required"`
	AuthRequired []string               `yaml:"authRequired"`
	Parameters   parameters.Parameters  `yaml:"parameters"`
	Annotations  *tools.ToolAnnotations `yaml:"annotations,omitempty"`

	ScopesRequired []string `yaml:"scopesRequired"`
}

var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return listTablesType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	databaseParameter := parameters.NewStringParameter(databaseKey, "The database to list tables from.")
	params := parameters.Parameters{databaseParameter}

	allParameters, paramManifest, _ := parameters.ProcessParameters(nil, params)

	t := Tool{
		Config:    cfg,
		AllParams: allParameters,
		manifest:  tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
	}
	return t, nil
}

var _ tools.Tool = Tool{}

type Tool struct {
	Config
	AllParams parameters.Parameters `yaml:"allParams"`
	manifest  tools.Manifest
}

func (t Tool) GetName() string {
	return t.Name
}

func (t Tool) GetDescription() string {
	return t.Description
}

func (t Tool) GetAuthRequired() []string {
	return t.AuthRequired
}

func (t Tool) GetAnnotations() *tools.ToolAnnotations {
	return tools.GetAnnotationsOrDefault(t.Annotations, tools.NewReadOnlyAnnotations)
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, token tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	mapParams := params.AsMap()
	database, ok := mapParams[databaseKey].(string)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("invalid or missing '%s' parameter; expected a string", databaseKey), nil)
	}

	// The database name is interpolated directly into the `SHOW TABLES FROM`
	// statement. Reject anything that is not a plain identifier so that a
	// caller cannot smuggle additional clauses (LIKE, LIMIT, FORMAT,
	// INTO OUTFILE, ...), quoted identifiers, or other expressions through
	// this parameter. Without this check, an MCP client (or a prompt-injected
	// LLM) could escape the intended scope of the tool and read arbitrary
	// system tables, switch the output format to one the calling layer cannot
	// parse safely, or chain a SHOW with an `INTO OUTFILE` write.
	if !validIdentifier.MatchString(database) {
		return nil, util.NewAgentError(fmt.Sprintf("invalid '%s' parameter %q: must be a plain identifier matching %s", databaseKey, database, validIdentifier.String()), nil)
	}
	query := fmt.Sprintf("SHOW TABLES FROM %s", database)

	out, err := source.RunSQL(ctx, query, nil)
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}

	res, ok := out.([]any)
	if !ok {
		return nil, util.NewClientServerError("unable to convert result to list", http.StatusInternalServerError, nil)
	}

	var tables []map[string]any
	for _, item := range res {
		tableMap, ok := item.(map[string]any)
		if !ok {
			return nil, util.NewClientServerError(fmt.Sprintf("unexpected type in result: got %T, want map[string]any", item), http.StatusInternalServerError, nil)
		}
		tableMap["database"] = database
		tables = append(tables, tableMap)
	}
	return tables, nil
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.AllParams, paramValues, embeddingModelsMap, nil)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
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
	return t.AllParams
}

func (t Tool) GetScopesRequired() []string {
	return t.ScopesRequired
}
