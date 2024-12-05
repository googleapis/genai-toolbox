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

type IPType string

func (i *IP_type) String() string {
	if string(*i) != "" {
		return strings.ToLower(string(*i))
	}
	return "public"
}

func (i *IP_type) UnmarshalYAML(node *yaml.Node) error {
	var ip_type string
	if err := node.Decode(&ip_type); err != nil {
		return err
	}
	switch ip_type {
	case "private", "public":
		*i = IP_type(ip_type)
		return nil
	default:
		return fmt.Errorf(`ip_type invalid: must be one of "public", or "private"`)
	}
}
