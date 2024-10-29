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
	Name        string     `yaml:"name"`
	Kind        string     `yaml:"kind"`
	Source      string     `yaml:"source"`
	Description string     `yaml:"description"`
	Statement   string     `yaml:"statement"`
	Parameters  Parameters `yaml:"parameters"`
}

func (cfg CloudSQLPgGenericConfig) toolKind() string {
	return CloudSQLPgSQLGenericKind
}

func (cfg CloudSQLPgGenericConfig) Initialize(srcs map[string]sources.Source) (Tool, error) {
	// verify source exists
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is the right kind
	s, ok := rawS.(sources.CloudSQLPgSource)
	if !ok {
		return nil, fmt.Errorf("sources for %q tools must be of kind %q", CloudSQLPgSQLGenericKind, sources.CloudSQLPgKind)
	}

	// finish tool setup
	t := CloudSQLPgGenericTool{
		PostgresGenericTool: PostgresGenericTool{
			Name:       cfg.Name,
			Kind:       CloudSQLPgSQLGenericKind,
			Pool:       s.Pool,
			Statement:  cfg.Statement,
			Parameters: cfg.Parameters,
			manifest:   ToolManifest{cfg.Description, generateManifests(cfg.Parameters)},
		},
	}
	return t, nil
}

// validate interface
var _ Tool = CloudSQLPgGenericTool{}

type CloudSQLPgGenericTool struct {
	PostgresGenericTool
}
