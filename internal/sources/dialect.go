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

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type Dialect string

func (i *Dialect) String() string {
	if string(*i) != "" {
		return strings.ToLower(string(*i))
	}
	return "google_standard_sql"
}

func (i *Dialect) UnmarshalYAML(node *yaml.Node) error {
	var dialect string
	if err := node.Decode(&dialect); err != nil {
		return err
	}
	switch strings.ToLower(dialect) {
	case "google_standard_sql", "postgresql":
		*i = Dialect(strings.ToLower(dialect))
		return nil
	default:
		return fmt.Errorf(`dialect invalid: must be one of "google_standard_sql", or "postgresql"`)
	}
}
