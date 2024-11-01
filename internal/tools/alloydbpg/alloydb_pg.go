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

package alloydbpg

import (
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/alloydbpg"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/postgres"
)

const ToolKind string = "alloydb-postgres-generic"

// validate interface
var _ tools.Config = GenericConfig{}

type GenericConfig struct {
	Name        string           `yaml:"name"`
	Kind        string           `yaml:"kind"`
	Source      string           `yaml:"source"`
	Description string           `yaml:"description"`
	Statement   string           `yaml:"statement"`
	Parameters  tools.Parameters `yaml:"parameters"`
}

func (cfg GenericConfig) ToolKind() string {
	return ToolKind
}

func (cfg GenericConfig) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is the right kind
	s, ok := rawS.(alloydbpg.Source)
	if !ok {
		return nil, fmt.Errorf("sources for %q tools must be of kind %q", ToolKind, alloydbpg.SourceKind)
	}

	// finish tool setup
	t := GenericTool{
		GenericTool: postgres.NewGenericTool(cfg.Name, cfg.Statement, cfg.Description, s.Pool, cfg.Parameters),
	}
	return t, nil
}

// validate interface
var _ tools.Tool = GenericTool{}

type GenericTool struct {
	postgres.GenericTool
}
