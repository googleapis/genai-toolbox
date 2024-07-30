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

const CloudSQLPgSQLGenericKind string = "cloud-sql-postgres-generic"

// validate interface
var _ Config = CloudSQLPgGenericConfig{}

type CloudSQLPgGenericConfig struct {
	Kind        string               `yaml:"kind"`
	Source      string               `yaml:"source"`
	Description string               `yaml:"description"`
	Statement   string               `yaml:"statement"`
	Parameters  map[string]Parameter `yaml:"parameters"`
}

func (r CloudSQLPgGenericConfig) toolKind() string {
	return CloudSQLPgSQLGenericKind
}
