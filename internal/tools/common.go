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
	"regexp"
)

var validName = regexp.MustCompile(`^[a-zA-Z0-9_-]*$`)

func IsValidName(s string) bool {
	return validName.MatchString(s)
}

func ConvertAnySliceToTyped(s []any, itemType string) (any, error) {
	var typedSlice any
	switch itemType {
	case "string":
		tempSlice := make([]string, len(s))
		for j, item := range s {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("expected item at index %d to be string, got %T", j, item)
			}
			tempSlice[j] = s
		}
		typedSlice = tempSlice
	case "integer":
		tempSlice := make([]int64, len(s))
		for j, item := range s {
			i, ok := item.(int)
			if !ok {
				return nil, fmt.Errorf("expected item at index %d to be integer, got %T", j, item)
			}
			tempSlice[j] = int64(i)
		}
		typedSlice = tempSlice
	case "float":
		tempSlice := make([]float64, len(s))
		for j, item := range s {
			f, ok := item.(float64)
			if !ok {
				return nil, fmt.Errorf("expected item at index %d to be float, got %T", j, item)
			}
			tempSlice[j] = f
		}
		typedSlice = tempSlice
	case "boolean":
		tempSlice := make([]bool, len(s))
		for j, item := range s {
			b, ok := item.(bool)
			if !ok {
				return nil, fmt.Errorf("expected item at index %d to be boolean, got %T", j, item)
			}
			tempSlice[j] = b
		}
		typedSlice = tempSlice
	}
	return typedSlice, nil
}
