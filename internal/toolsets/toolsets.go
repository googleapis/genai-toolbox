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

package toolsets

import (
	"github.com/googleapis/genai-toolbox/internal/tools"
	"gopkg.in/yaml.v3"
)

type Config interface {
	Initialize() (Toolset, error)
}

// SourceConfigs is a type used to allow unmarshal of the data source config map
type Configs map[string]Config

// validate interface
var _ yaml.Unmarshaler = &Configs{}

func (c *Configs) UnmarshalYAML(node *yaml.Node) error {
	*c = make(Configs)

	var rawToolsets map[string][]string
	if err := node.Decode(&rawToolsets); err != nil {
		return err
	}
	return nil
}

type ToolsetManifest struct {
	ServerVersion string                        `json:"serverVersion"`
	Tools         map[string]tools.ToolManifest `json:"tools"`
}
type Toolset interface {
	Describe() (ToolsetManifest, error)
}
