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

package dgraph

import (
	"encoding/json"
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/dgraph"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const ToolKind string = "dgraph-dql"

type compatibleSource interface {
	DgraphClient() *dgraph.DgraphClient
}

// validate compatible sources are still compatible
var _ compatibleSource = &dgraph.Source{}

var compatibleSources = [...]string{dgraph.SourceKind}

type Config struct {
	Name        string           `yaml:"name"`
	Kind        string           `yaml:"kind"`
	Source      string           `yaml:"source"`
	Description string           `yaml:"description"`
	Statement   string           `yaml:"statement"`
	IsQuery     bool             `yaml:"isQuery"`
	Timeout     string           `yaml:"timeout"`
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
		Name:         cfg.Name,
		Kind:         ToolKind,
		Parameters:   cfg.Parameters,
		Statement:    cfg.Statement,
		DgraphClient: s.DgraphClient(),
		IsQuery:      cfg.IsQuery,
		Timeout:      cfg.Timeout,
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: cfg.Parameters.Manifest()},
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
	DgraphClient *dgraph.DgraphClient
	IsQuery      bool
	Timeout      string
	Statement    string
	manifest     tools.Manifest
}

func (t Tool) Invoke(params tools.ParamValues) (string, error) {
	paramsMap := params.AsMapWithDollarPrefix()

	resp, err := t.DgraphClient.ExecuteQuery(t.Statement, paramsMap, t.IsQuery, t.Timeout)
	if err != nil {
		return "", err
	}

	if err := dgraph.CheckError(resp); err != nil {
		return "", err
	}

	var result struct {
		Data map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("error parsing JSON: %v", err)
	}

	return fmt.Sprintf(
		"Stub tool call for %q! Parameters parsed: %q \n Output: %v",
		t.Name, paramsMap, result.Data,
	), nil
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
