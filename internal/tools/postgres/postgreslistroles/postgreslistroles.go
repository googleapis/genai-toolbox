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

package postgreslistroles

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"github.com/jackc/pgx/v5/pgxpool"
)

const kind string = "postgres-list-roles"

const listRolesStatement = `
	WITH RoleDetails AS (
		SELECT
			r.rolname AS role_name,
			r.oid AS oid,
			r.rolconnlimit AS connection_limit,
			r.rolsuper AS is_superuser,
			r.rolinherit AS inherits_privileges,
			r.rolcreaterole AS can_create_roles,
			r.rolcreatedb AS can_create_db,
			r.rolcanlogin AS can_login,
			r.rolreplication AS is_replication_role, 
			r.rolbypassrls AS bypass_rls,          
			r.rolvaliduntil AS valid_until,         
			-- List of roles that belong to this role (Direct Members)
			ARRAY(
				SELECT m_r.rolname
				FROM pg_auth_members pam
				JOIN pg_roles m_r ON pam.member = m_r.oid
				WHERE pam.roleid = r.oid
			) AS direct_members,
			-- List of roles that this role belongs to (Member Of)
			ARRAY(
				SELECT g_r.rolname
				FROM pg_auth_members pam
				JOIN pg_roles g_r ON pam.roleid = g_r.oid
				WHERE pam.member = r.oid
			) AS member_of
		FROM pg_roles r
	-- Exclude system and internal roles
		WHERE r.rolname NOT LIKE 'cloudsql%' 
		AND r.rolname NOT LIKE 'alloydb_%'
		AND r.rolname NOT LIKE 'pg_%'
	)
	SELECT *
	FROM RoleDetails
	WHERE
		($1::text IS NULL OR role_name LIKE '%' || $1 || '%')
	ORDER BY role_name
	LIMIT COALESCE($2::int, 50);
`

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
	PostgresPool() *pgxpool.Pool
	RunSQL(context.Context, string, []any) (any, error)
}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	allParameters := parameters.Parameters{
		parameters.NewStringParameterWithDefault("role_name", "", "Optional: a text to filter results by role name. The input is used within a LIKE clause."),
		parameters.NewIntParameterWithDefault("limit", 50, "Optional: The maximum number of rows to return. Default is 10"),
	}

	description := cfg.Description
	if description == "" {
		description = "Lists all the user-created roles in the instance . It returns the role name, Object ID, the maximum number of concurrent connections the role can make, along with boolean indicators for: superuser status, privilege inheritance from member roles, ability to create roles, ability to create databases, ability to log in, replication privilege, and the ability to bypass row-level security, the password expiration timestamp, a list of direct members belonging to this role, and a list of other roles/groups that this role is a member of."
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, description, cfg.AuthRequired, allParameters, nil)

	// finish tool setup
	return Tool{
		Config:    cfg,
		allParams: allParameters,
		manifest: tools.Manifest{
			Description:  description,
			Parameters:   allParameters.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	allParams   parameters.Parameters `yaml:"allParams"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Kind)
	if err != nil {
		return nil, err
	}

	paramsMap := params.AsMap()

	newParams, err := parameters.GetParams(t.allParams, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract standard params %w", err)
	}
	sliceParams := newParams.AsSlice()
	return source.RunSQL(ctx, listRolesStatement, sliceParams)
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.allParams, data, claims)
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.allParams, paramValues, embeddingModelsMap, nil)
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

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	return false, nil
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}
