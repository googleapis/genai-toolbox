// Copyright 2025 Google LLC
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

package elasticsearch

import (
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/tools"
)

// RetrieveIndices extracts the indices from the provided parameters.
func RetrieveIndices(params tools.ParamValues) ([]string, error) {
	paramsMap := params.AsMap()
	index, ok := paramsMap["index"]
	if ok {
		if str, ok := index.(string); ok {
			return []string{str}, nil
		}
		return nil, fmt.Errorf("invalid type for index: expected string, got %T", index)
	}

	anyIndices, ok := paramsMap["indices"].([]any)
	if !ok {
		return nil, fmt.Errorf("missing required parameter: indices, got %T", paramsMap["indices"])
	}
	var indices []string
	for _, index := range anyIndices {
		if str, ok := index.(string); ok {
			indices = append(indices, str)
		} else {
			return nil, fmt.Errorf("invalid type for indices: expected []string, got %T", index)
		}
	}
	return indices, nil
}
