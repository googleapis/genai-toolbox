// Copyright 2024 Google LLC
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

package postgressql

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/alloydbpg"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqlpg"
	"github.com/googleapis/genai-toolbox/internal/sources/postgres"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const kind string = "postgres-sql"

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
	Name               string           `yaml:"name" validate:"required"`
	Kind               string           `yaml:"kind" validate:"required"`
	Source             string           `yaml:"source" validate:"required"`
	Description        string           `yaml:"description" validate:"required"`
	Statement          string           `yaml:"statement" validate:"required"`
	AuthRequired       []string         `yaml:"authRequired"`
	RequiredRoles      []string         `yaml:"requiredRoles"`
	RequiredPermissions []string         `yaml:"requiredPermissions"`
	AllowedOperations  []string         `yaml:"allowedOperations"`
	RestrictedTables   []string         `yaml:"restrictedTables"`
	MaxAffectedRows    int              `yaml:"maxAffectedRows"`
	Parameters         tools.Parameters `yaml:"parameters"`
	TemplateParameters tools.Parameters `yaml:"templateParameters"`
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

	allParameters, paramManifest, paramMcpManifest := tools.ProcessParameters(cfg.TemplateParameters, cfg.Parameters)

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: paramMcpManifest,
	}

	// finish tool setup
	t := Tool{
		Name:               cfg.Name,
		Kind:               kind,
		Parameters:         cfg.Parameters,
		TemplateParameters: cfg.TemplateParameters,
		AllParams:          allParameters,
		Statement:          cfg.Statement,
		AuthRequired:       cfg.AuthRequired,
		RequiredRoles:      cfg.RequiredRoles,
		RequiredPermissions: cfg.RequiredPermissions,
		AllowedOperations:  cfg.AllowedOperations,
		RestrictedTables:   cfg.RestrictedTables,
		MaxAffectedRows:    cfg.MaxAffectedRows,
		Pool:               s.PostgresPool(),
		manifest:           tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:        mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name               string           `yaml:"name"`
	Kind               string           `yaml:"kind"`
	AuthRequired       []string         `yaml:"authRequired"`
	RequiredRoles      []string         `yaml:"requiredRoles"`
	RequiredPermissions []string         `yaml:"requiredPermissions"`
	AllowedOperations  []string         `yaml:"allowedOperations"`
	RestrictedTables   []string         `yaml:"restrictedTables"`
	MaxAffectedRows    int              `yaml:"maxAffectedRows"`
	Parameters         tools.Parameters `yaml:"parameters"`
	TemplateParameters tools.Parameters `yaml:"templateParameters"`
	AllParams          tools.Parameters `yaml:"allParams"`

	Pool        *pgxpool.Pool
	Statement   string
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	// Print the database user, source, and database name for debugging
	var dbUser, dbSource, dbName, sqlStatement, traceID, spanID string
	if t.Pool != nil && t.Pool.Config() != nil && t.Pool.Config().ConnConfig != nil {
		dbUser = t.Pool.Config().ConnConfig.User
		dbName = t.Pool.Config().ConnConfig.Database
		fmt.Println("[DEBUG] Database user:", dbUser)
		fmt.Println("[DEBUG] Database name:", dbName)
	}
	// The source name is t.Name (tool name) or t.Kind, but the actual source name is not directly stored.
	dbSource = t.Name
	fmt.Println("[DEBUG] Source name (tool name):", dbSource)

	paramsMap := params.AsMap()
	sqlStatement, err := tools.ResolveTemplateParams(t.TemplateParameters, t.Statement, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract template params %w", err)
	}
	fmt.Println("[DEBUG] SQL statement:", sqlStatement)

	// Get trace/span ID from context if available
	if span := trace.SpanFromContext(ctx); span != nil {
		traceID = span.SpanContext().TraceID().String()
		spanID = span.SpanContext().SpanID().String()
		fmt.Println("[DEBUG] TraceID:", traceID)
		fmt.Println("[DEBUG] SpanID:", spanID)
		// Add all relevant attributes to the span
		span.SetAttributes(
			attribute.String("db.user", dbUser),
			attribute.String("db.source", dbSource),
			attribute.String("db.name", dbName),
			attribute.String("db.statement", sqlStatement),
			attribute.String("trace.id", traceID),
			attribute.String("span.id", spanID),
		)
	}

	// Set application_name for this session to include trace/span ID and tool name
	appName := fmt.Sprintf("genai-toolbox|trace_id=%s|span_id=%s|tool=%s", traceID, spanID, dbSource)
	_, err = t.Pool.Exec(ctx, fmt.Sprintf("SET application_name = '%s'", appName))
	if err != nil {
		fmt.Println("Failed to set application_name:", err)
		return nil, fmt.Errorf("unable to set application_name: %w", err)
	}

	// Hardcode for testing: set app.current_customer_id for RLS
	_, err = t.Pool.Exec(ctx, "SET app.current_customer_id = '00000000-0000-0000-0000-000000000001'")
	if err != nil {
		fmt.Println("Failed to set app.current_customer_id:", err)
		return nil, fmt.Errorf("unable to set app.current_customer_id: %w", err)
	}

	newParams, err := tools.GetParams(t.Parameters, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract standard params %w", err)
	}
	sliceParams := newParams.AsSlice()
	results, err := t.Pool.Query(ctx, sqlStatement, sliceParams...)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}

	fields := results.FieldDescriptions()

	var out []any
	for results.Next() {
		v, err := results.Values()
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}
		vMap := make(map[string]any)
		for i, f := range fields {
			vMap[f.Name] = v[i]
		}
		out = append(out, vMap)
	}

	return out, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.AllParams, data, claims)
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
