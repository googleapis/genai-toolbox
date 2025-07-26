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

package firebirdexecutesql

import (
	"context"
	"database/sql"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/firebird"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "firebird-execute-sql"

func init() {
	if !tools.Register(kind, newConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", kind))
	}
}

// newConfig cria uma nova instância de Config a partir do decoder YAML.
func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

// compatibleSource define o que a ferramenta espera de uma fonte de dados.
type compatibleSource interface {
	FirebirdDB() *sql.DB
}

// Garante que a fonte de dados do Firebird implementa a interface.
var _ compatibleSource = &firebird.Source{}

var compatibleSources = [...]string{firebird.SourceKind}

// Config define a estrutura da ferramenta no arquivo YAML.
type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"` // Adicione esta linha
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

// Garante que Config implementa a interface ToolConfig.
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

// Initialize cria e configura a instância da ferramenta executável.
func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// Verifica se a fonte existe.
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// Verifica se a fonte é compatível.
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	// O teste genérico envia o parâmetro com o nome "sql".
	sqlParameter := tools.NewStringParameter("sql", "The sql to execute.")
	parameters := tools.Parameters{sqlParameter}

	// Usa a função helper para gerar os manifestos.
	_, paramManifest, paramMcpManifest := tools.ProcessParameters(nil, parameters)

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: paramMcpManifest,
	}

	// Conclui a configuração da ferramenta.
	t := &Tool{
		Name:         cfg.Name,
		Parameters:   parameters,
		AuthRequired: cfg.AuthRequired,
		Db:           s.FirebirdDB(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

// Garante que Tool implementa a interface Tool.
var _ tools.Tool = &Tool{}

// Tool é a implementação da ferramenta executável.
type Tool struct {
	Name         string
	Parameters   tools.Parameters
	AuthRequired []string
	Db           *sql.DB
	manifest     tools.Manifest
	mcpManifest  tools.McpManifest
}

// Invoke executa a lógica da ferramenta.
func (t *Tool) Invoke(ctx context.Context, params tools.ParamValues) (any, error) {
	// O ParseParams genérico cria um slice de valores.
	// Pegamos o primeiro (e único) valor.
	sliceParams := params.AsSlice()
	if len(sliceParams) == 0 {
		return nil, fmt.Errorf("missing required 'sql' parameter")
	}
	sql, ok := sliceParams[0].(string)
	if !ok {
		return nil, fmt.Errorf("parameter 'sql' is not a valid string, got %T", sliceParams[0])
	}

	rows, err := t.Db.QueryContext(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve column information: %w", err)
	}

	// Adaptação para Firebird: comandos DDL não retornam colunas.
	// O teste espera 'null' nesses casos.
	if len(cols) == 0 {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("query execution failed: %w", err)
		}
		return nil, nil
	}

	var out []any
	for rows.Next() {
		// Adaptação para database/sql: usa Scan em vez de Values.
		values := make([]any, len(cols))
		scanArgs := make([]any, len(values))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}

		vMap := make(map[string]any)
		for i, colName := range cols {
			// O driver do Firebird pode retornar bytes, então convertemos para string.
			if b, ok := values[i].([]byte); ok {
				vMap[colName] = string(b)
			} else {
				vMap[colName] = values[i]
			}
		}
		out = append(out, vMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return out, nil
}

// ParseParams delega a análise para a função helper genérica.
func (t *Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data, claims)
}

func (t *Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t *Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t *Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}
