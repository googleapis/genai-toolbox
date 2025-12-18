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
	"context"
	"fmt"
	"strings"
)

type IPType string

func (i *IPType) String() string {
	if string(*i) != "" {
		return strings.ToLower(string(*i))
	}
	return "public"
}

func (i *IPType) UnmarshalYAML(ctx context.Context, unmarshal func(interface{}) error) error {
	var ipType string
	if err := unmarshal(&ipType); err != nil {
		return err
	}
	switch strings.ToLower(ipType) {
	case "private", "public", "psc":
		*i = IPType(strings.ToLower(ipType))
		return nil
	default:
		return fmt.Errorf(`ipType invalid: must be one of "public", "private", or "psc"`)
	}
}
