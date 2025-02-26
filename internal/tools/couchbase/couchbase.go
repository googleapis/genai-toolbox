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

package couchbase

import (
	"encoding/json"
	"fmt"

	"github.com/couchbase/gocb/v2"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/couchbase"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const ToolKind string = "couchbase-sql"

type compatibleSource interface {
	CouchbaseScope() *gocb.Scope
}

// validate compatible sources are still compatible
var _ compatibleSource = &couchbase.Source{}

var compatibleSources = [...]string{couchbase.SourceKind}

type Config struct {
	Name        string           `yaml:"name" validate:"required"`
	Kind        string           `yaml:"kind" validate:"required"`
	Source      string           `yaml:"source" validate:"required"`
	Description string           `yaml:"description" validate:"required"`
	Statement   string           `yaml:"statement" validate:"required"`
	Parameters  tools.Parameters `yaml:"parameters"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return ToolKind
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
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", ToolKind, compatibleSources)
	}

	// finish tool setup
	t := Tool{
		Name:       cfg.Name,
		Kind:       ToolKind,
		Parameters: cfg.Parameters,
		Statement:  cfg.Statement,
		Scope:      s.CouchbaseScope(),
		manifest:   tools.Manifest{Description: cfg.Description, Parameters: cfg.Parameters.Manifest()},
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	Parameters   tools.Parameters `yaml:"parameters"`
	AuthRequired []string         `yaml:"authRequired"`

	Scope     *gocb.Scope
	Statement string
	manifest  tools.Manifest
}

func (t Tool) Invoke(params tools.ParamValues) ([]any, error) {
	namedParams := params.AsMap()
	results, err := t.Scope.Query(t.Statement, &gocb.QueryOptions{
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: namedParams,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}

	var out []any
	for results.Next() {
		var result json.RawMessage
		err := results.Row(&result)
		if err != nil {
			return nil, fmt.Errorf("error processing row: %w", err)
		}
		out = append(out, result)
	}
	return out, nil
}

func (t Tool) ParseParams(data map[string]any, claimsMap map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data, claimsMap)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) Authorized(verifiedAuthSources []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthSources)
}
