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

package tools

import (
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/sources"
)

const CloudSQLPgSQLGenericKind string = "cloud-sql-postgres-generic"

// validate interface
var _ Config = CloudSQLPgGenericConfig{}

type CloudSQLPgGenericConfig struct {
	Name        string               `yaml:"name"`
	Kind        string               `yaml:"kind"`
	Source      string               `yaml:"source"`
	Description string               `yaml:"description"`
	Statement   string               `yaml:"statement"`
	Parameters  map[string]Parameter `yaml:"parameters"`
}

func (r CloudSQLPgGenericConfig) toolKind() string {
	return CloudSQLPgSQLGenericKind
}

func (r CloudSQLPgGenericConfig) Initialize(srcs map[string]sources.Source) (Tool, error) {
	// verify source exists
	rawS, ok := srcs[r.Source]
	if !ok {
		return nil, fmt.Errorf("No source named %q configured!", r.Source)
	}

	// verify the source is the right kind
	s, ok := rawS.(sources.CloudSQLPgSource)
	if !ok {
		return nil, fmt.Errorf("Sources for %q tools must be of kind %q!", CloudSQLPgSQLGenericKind, sources.CloudSQLPgKind)
	}

	// finish tool setup
	t := CloudSQLPgGenericTool{
		Name:   r.Name,
		Kind:   CloudSQLPgSQLGenericKind,
		Source: s,
	}
	return t, nil
}

// validate interface
var _ Tool = CloudSQLPgGenericTool{}

type CloudSQLPgGenericTool struct {
	Name   string `yaml:"name"`
	Kind   string `yaml:"kind"`
	Source sources.CloudSQLPgSource
}

func (t CloudSQLPgGenericTool) Invoke() (string, error) {
	return fmt.Sprintf("Stub tool call for %q!", t.Name), nil
}

func (t CloudSQLPgGenericConfig) DescribeTool() ToolManifest {
	toolManifest := ToolManifest{
		Description: t.Description,
		Parameters:  t.Parameters,
	}
	return toolManifest
}
