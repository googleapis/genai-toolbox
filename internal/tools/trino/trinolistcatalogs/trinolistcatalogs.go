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

package trinolistcatalogs

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/trino"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "trino-list-catalogs"

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
	TrinoDB() *sql.DB
}

// validate compatible sources are still compatible
var _ compatibleSource = &trino.Source{}

var compatibleSources = [...]string{trino.SourceKind}

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

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: tools.Parameters{}.McpManifest(),
	}

	// finish tool setup
	t := Tool{
		Name:         cfg.Name,
		Kind:         kind,
		AuthRequired: cfg.AuthRequired,
		Db:           s.TrinoDB(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: tools.Parameters{}.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

// CatalogInfo represents information about a Trino catalog
type CatalogInfo struct {
	CatalogName string `json:"catalogName"`
	SchemaCount int    `json:"schemaCount,omitempty"`
}

// CatalogsResponse represents the response containing all catalogs
type CatalogsResponse struct {
	Catalogs   []CatalogInfo `json:"catalogs"`
	TotalCount int           `json:"totalCount"`
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string   `yaml:"name"`
	Kind         string   `yaml:"kind"`
	AuthRequired []string `yaml:"authRequired"`

	Db          *sql.DB
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	// Query to get all catalogs with schema counts
	query := `
		SELECT 
			catalog_name,
			COUNT(DISTINCT schema_name) as schema_count
		FROM information_schema.schemata
		GROUP BY catalog_name
		ORDER BY catalog_name
	`

	rows, err := t.Db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list catalogs: %w", err)
	}
	defer rows.Close()

	var catalogs []CatalogInfo
	for rows.Next() {
		var catalog CatalogInfo
		if err := rows.Scan(&catalog.CatalogName, &catalog.SchemaCount); err != nil {
			return nil, fmt.Errorf("failed to scan catalog row: %w", err)
		}
		catalogs = append(catalogs, catalog)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating catalog rows: %w", err)
	}

	response := CatalogsResponse{
		Catalogs:   catalogs,
		TotalCount: len(catalogs),
	}

	return response, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParamValues{}, nil
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
