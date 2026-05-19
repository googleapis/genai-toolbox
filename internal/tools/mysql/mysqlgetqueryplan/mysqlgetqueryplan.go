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

package mysqlgetqueryplan

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/orderedmap"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const resourceType string = "mysql-get-query-plan"

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
	MySQLPool() *sql.DB
	RunSQL(context.Context, string, []any) (any, error)
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
	sqlParameter := parameters.NewStringParameter("sql_statement", "The sql statement to explain.")
	params := parameters.Parameters{sqlParameter}

	// finish tool setup
	t := Tool{
		Config:     cfg,
		Parameters: params,
		manifest:   tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	Parameters parameters.Parameters `yaml:"parameters"`
	manifest   tools.Manifest
}

// explainableStatements is the set of statement types MySQL's EXPLAIN
// FORMAT=JSON accepts. It excludes ANALYZE (which executes the query) and
// FOR (EXPLAIN FOR CONNECTION does not support FORMAT=JSON).
var explainableStatements = map[string]bool{
	"SELECT":  true,
	"DELETE":  true,
	"INSERT":  true,
	"REPLACE": true,
	"UPDATE":  true,
	"TABLE":   true,
	"WITH":    true,
	"VALUES":  true,
}

// ValidateSQLStatement allows a single explainable statement. It rejects
// multi-statement input while tolerating string literals, comments, leading
// parentheses (e.g. parenthesized UNION) and a single trailing semicolon.
func ValidateSQLStatement(sqlStr string) error {
	keyword, multiStatement := scanStatement(sqlStr)
	if multiStatement {
		return fmt.Errorf("sql_statement must be a single statement")
	}
	if keyword == "" {
		return fmt.Errorf("sql_statement must not be empty")
	}
	if !explainableStatements[strings.ToUpper(keyword)] {
		return fmt.Errorf("sql_statement must begin with a DML keyword (SELECT, INSERT, UPDATE, DELETE, REPLACE, TABLE, WITH, VALUES); got %q", strings.ToUpper(keyword))
	}
	return nil
}

// scanStatement walks sqlStr while tracking string-literal and comment state.
// It returns the first statement keyword and whether content follows a
// top-level semicolon (i.e. a second statement).
func scanStatement(sqlStr string) (keyword string, multiStatement bool) {
	r := []rune(sqlStr)
	n := len(r)
	terminated := false
	for i := 0; i < n; {
		c := r[i]
		switch {
		case unicode.IsSpace(c):
			i++
		case c == '-' && i+1 < n && r[i+1] == '-', c == '#':
			for i < n && r[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < n && r[i+1] == '*':
			i += 2
			for i+1 < n && !(r[i] == '*' && r[i+1] == '/') {
				i++
			}
			i += 2
		case c == '\'' || c == '"' || c == '`':
			q := c
			i++
			for i < n {
				if r[i] == '\\' && q != '`' {
					i += 2
					continue
				}
				if r[i] == q {
					if i+1 < n && r[i+1] == q {
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
		case c == '(':
			i++
		case c == ';':
			terminated = true
			i++
		default:
			if terminated {
				return keyword, true
			}
			if keyword == "" && (unicode.IsLetter(c) || c == '_') {
				start := i
				for i < n && (unicode.IsLetter(r[i]) || unicode.IsDigit(r[i]) || r[i] == '_') {
					i++
				}
				keyword = string(r[start:i])
				continue
			}
			i++
		}
	}
	return keyword, false
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	paramsMap := params.AsMap()
	sqlStr, ok := paramsMap["sql_statement"].(string)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("unable to get cast %s", paramsMap["sql_statement"]), nil)
	}

	if err := ValidateSQLStatement(sqlStr); err != nil {
		return nil, util.NewAgentError(err.Error(), nil)
	}

	// Log the query executed for debugging.
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, util.NewClientServerError("error getting logger", http.StatusInternalServerError, err)
	}
	logger.DebugContext(ctx, fmt.Sprintf("executing `%s` tool query: %s", resourceType, sqlStr))

	query := fmt.Sprintf("EXPLAIN FORMAT=JSON %s", sqlStr)
	result, err := source.RunSQL(ctx, query, nil)
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}
	// extract and return only the query plan object
	resSlice, ok := result.([]any)
	if !ok || len(resSlice) == 0 {
		return nil, util.NewClientServerError("no query plan returned", http.StatusInternalServerError, nil)
	}
	row, ok := resSlice[0].(orderedmap.Row)
	if !ok || len(row.Columns) == 0 {
		return nil, util.NewClientServerError("no query plan returned in row", http.StatusInternalServerError, nil)
	}
	plan, ok := row.Columns[0].Value.(string)
	if !ok {
		return nil, util.NewClientServerError("unable to convert plan object to string", http.StatusInternalServerError, nil)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(plan), &out); err != nil {
		return nil, util.NewClientServerError("failed to unmarshal query plan json", http.StatusInternalServerError, err)
	}
	return out, nil
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.Parameters, paramValues, embeddingModelsMap, nil)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) RequiresClientAuthorization(_ tools.SourceProvider) (bool, error) {
	return false, nil
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

func (t Tool) GetAuthTokenHeaderName(_ tools.SourceProvider) (string, error) {
	return "Authorization", nil
}

func (t Tool) GetParameters() parameters.Parameters {
	return t.Parameters
}

func (t Tool) GetScopesRequired() []string {
	return t.ScopesRequired
}
