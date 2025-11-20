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

package postgresreplicationstats

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/alloydbpg"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqlpg"
	"github.com/googleapis/genai-toolbox/internal/sources/postgres"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"github.com/jackc/pgx/v5/pgxpool"
)

const kind string = "postgres-replication-stats"

const replicationStats = `
	SELECT
		pid,
		usename,
        application_name,
		backend_xmin,
        client_addr,
        state,
        sync_state,
        pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), sent_lsn)) AS sent_lag,
        pg_size_pretty(pg_wal_lsn_diff(sent_lsn, write_lsn)) AS write_lag,
        pg_size_pretty(pg_wal_lsn_diff(write_lsn, flush_lsn)) AS flush_lag,
        pg_size_pretty(pg_wal_lsn_diff(flush_lsn, replay_lsn)) AS replay_lag,
        pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn)) AS total_lag
    FROM
        pg_stat_replication;
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
}

// validate compatible sources are still compatible
var _ compatibleSource = &alloydbpg.Source{}
var _ compatibleSource = &cloudsqlpg.Source{}
var _ compatibleSource = &postgres.Source{}

var compatibleSources = [...]string{alloydbpg.SourceKind, cloudsqlpg.SourceKind, postgres.SourceKind}

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

	allParameters := parameters.Parameters{}
	paramManifest := allParameters.Manifest()

	if cfg.Description == "" {
		cfg.Description = "Lists each replica's process ID, user name, application name, backend_xmin (standby's xmin horizon reported by hot_standby_feedback), client IP address, connection state, and sync_state, along with lag sizes in bytes for sent_lag (primary to sent), write_lag (sent to written), flush_lag (written to flushed), replay_lag (flushed to replayed), and the overall total_lag (primary to replayed)."
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	// finish tool setup
	return Tool{
		name:         cfg.Name,
		kind:         cfg.Kind,
		authRequired: cfg.AuthRequired,
		allParams:    allParameters,
		pool:         s.PostgresPool(),
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   paramManifest,
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	name         string                `yaml:"name"`
	kind         string                `yaml:"kind"`
	authRequired []string              `yaml:"authRequired"`
	allParams    parameters.Parameters `yaml:"allParams"`
	pool         *pgxpool.Pool
	manifest     tools.Manifest
	mcpManifest  tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	newParams, err := parameters.GetParams(t.allParams, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract standard params %w", err)
	}
	sliceParams := newParams.AsSlice()

	results, err := t.pool.Query(ctx, replicationStats, sliceParams...)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer results.Close()

	fields := results.FieldDescriptions()
	var out []map[string]any

	for results.Next() {
		values, err := results.Values()
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}
		rowMap := make(map[string]any)
		for i, field := range fields {
			rowMap[string(field.Name)] = values[i]
		}
		out = append(out, rowMap)
	}

	return out, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.allParams, data, claims)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.authRequired, verifiedAuthServices)
}

func (t Tool) RequiresClientAuthorization() bool {
	return false
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
