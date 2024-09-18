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

package sources

const CloudSQLPgKind string = "cloud-sql-postgres"

// validate interface
var _ Config = CloudSQLPgConfig{}

type CloudSQLPgConfig struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	Project  string `yaml:"project"`
	Region   string `yaml:"region"`
	Instance string `yaml:"instance"`
	Database string `yaml:"database"`
}

func (r CloudSQLPgConfig) sourceKind() string {
	return CloudSQLPgKind
}

func (r CloudSQLPgConfig) Initialize() (Source, error) {
	s := CloudSQLPgSource{
		Name: r.Name,
		Kind: CloudSQLPgKind,
	}
	return s, nil
}

var _ Source = CloudSQLPgSource{}

type CloudSQLPgSource struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
}
