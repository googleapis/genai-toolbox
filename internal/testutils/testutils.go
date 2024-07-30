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

package testutils

import (
	"strings"
)

// formatYaml is a utility function for stripping out tabs in multiline strings
func FormatYaml(in string) []byte {
	// removes any leading indentation(tabs)
	in = strings.ReplaceAll(in, "\n\t", "\n ")
	// converts remaining indentation
	in = strings.ReplaceAll(in, "\t", "  ")
	return []byte(in)
}
