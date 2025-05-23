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

// Helper function to replace parameters in the commands
func ReplaceCommandsParams(commands [][]string, params Parameters, paramValues ParamValues) ([][]any, error) {
	paramMap := paramValues.AsMapWithDollarPrefix()
	typeMap := make(map[string]string, len(params))
	for _, p := range params {
		placeholder := "$" + p.GetName()
		typeMap[placeholder] = p.GetType()
	}
	newCommands := make([][]any, len(commands))
	for i, cmd := range commands {
		newCmd := make([]any, len(cmd))
		for j, part := range cmd {
			v, ok := paramMap[part]
			if !ok {
				// Command part is not a Parameter placeholder
				newCmd[j] = part
				continue
			}
			if typeMap[part] == "array" {
				for _, item := range v.([]any) {
					// Nested arrays will only be expanded once
					// e.g., [A, [B, C]]  --> ["A", "[B C]"]
					newCmd = append(newCmd, fmt.Sprintf("%s", item))
				}
				continue
			}
			newCmd[j] = fmt.Sprintf("%s", v)
		}
		newCommands[i] = newCmd
	}
	return newCommands, nil
}
