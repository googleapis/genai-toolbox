// Copyright 2026 Google LLC
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

package util

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

// PopulateUrlParams injects bound URL query parameters into the data arguments
// and performs automatic type conversion for integer, boolean, and float parameters.
func PopulateUrlParams(ctx context.Context, data map[string]any, toolParams parameters.Parameters) map[string]any {
	urlParams, ok := util.UrlParamsFromContext(ctx)
	if !ok {
		return data
	}
	if data == nil {
		data = make(map[string]any)
	}
	for name, val := range urlParams {
		// Only inject if the client didn't supply it explicitly.
		if _, exists := data[name]; !exists {
			data[name] = val

			// Attempt type conversion for known parameters
			for _, p := range toolParams {
				if p.GetName() == name {
					switch p.GetType() {
					case "integer":
						if i, err := strconv.Atoi(val); err == nil {
							data[name] = i
						}
					case "boolean":
						if b, err := strconv.ParseBool(val); err == nil {
							data[name] = b
						}
					case "float":
						if f, err := strconv.ParseFloat(val, 64); err == nil {
							data[name] = f
						}
					case "array":
						var arr []any
						if err := json.Unmarshal([]byte(val), &arr); err == nil {
							data[name] = arr
						}
					case "map":
						var m map[string]any
						if err := json.Unmarshal([]byte(val), &m); err == nil {
							data[name] = m
						}
					}
					break
				}
			}
		}
	}
	return data
}
