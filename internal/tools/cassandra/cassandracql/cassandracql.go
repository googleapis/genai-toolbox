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

package cassandracql

import (
	"context"
	"fmt"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/cassandra"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind string = "cassandra-cql"

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
	CassandraSession() *gocql.Session
}

var _ compatibleSource = &cassandra.Source{}

var compatibleSources = [...]string{cassandra.SourceKind}

type Config struct {
	Name               string                `yaml:"name" validate:"required"`
	Kind               string                `yaml:"kind" validate:"required"`
	Source             string                `yaml:"source" validate:"required"`
	Description        string                `yaml:"description" validate:"required"`
	Statement          string                `yaml:"statement" validate:"required"`
	AuthRequired       []string              `yaml:"authRequired"`
	Parameters         parameters.Parameters `yaml:"parameters"`
	TemplateParameters parameters.Parameters `yaml:"templateParameters"`
}

// Initialize implements tools.ToolConfig.
func (c Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[c.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", c.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	allParameters, paramManifest, err := parameters.ProcessParameters(c.TemplateParameters, c.Parameters)
	if err != nil {
		return nil, err
	}

	mcpManifest := tools.GetMcpManifest(c.Name, c.Description, c.AuthRequired, allParameters, nil)

	t := Tool{
		Config:      c,
		AllParams:   allParameters,
		Session:     s.CassandraSession(),
		manifest:    tools.Manifest{Description: c.Description, Parameters: paramManifest, AuthRequired: c.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// ToolConfigKind implements tools.ToolConfig.
func (c Config) ToolConfigKind() string {
	return kind
}

var _ tools.ToolConfig = Config{}

type Tool struct {
	Config
	AllParams parameters.Parameters `yaml:"allParams"`

	Session     *gocql.Session
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

// RequiresClientAuthorization implements tools.Tool.
func (t Tool) RequiresClientAuthorization() bool {
	return false
}

// Authorized implements tools.Tool.
func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

// Invoke implements tools.Tool.
func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()
	newStatement, err := parameters.ResolveTemplateParams(t.TemplateParameters, t.Statement, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract template params %w", err)
	}

	newParams, err := parameters.GetParams(t.Parameters, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract standard params %w", err)
	}
	sliceParams := newParams.AsSlice()
	iter := t.Session.Query(newStatement, sliceParams...).IterContext(ctx)

	// Create a slice to store the out
	var out []map[string]interface{}

	// Scan results into a map and append to the slice
	for {
		row := make(map[string]interface{}) // Create a new map for each row
		if !iter.MapScan(row) {
			break // No more rows
		}
		out = append(out, row)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("unable to parse rows: %w", err)
	}
	return out, nil
}

// Manifest implements tools.Tool.
func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

// McpManifest implements tools.Tool.
func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

// ParseParams implements tools.Tool.
func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.AllParams, data, claims)
}

var _ tools.Tool = Tool{}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
