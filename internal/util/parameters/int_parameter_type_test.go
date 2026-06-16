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

package parameters_test

import (
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

// TestIntParameterAllowedExcludedReportIntegerType guards against the int
// constructors reporting the wrong parameter type. The allowed/excluded value
// variants previously copied the string constructor and set Type to "string".
func TestIntParameterAllowedExcludedReportIntegerType(t *testing.T) {
	for _, p := range []parameters.Parameter{
		parameters.NewIntParameterWithAllowedValues("my_int", "an int param", []any{1}),
		parameters.NewIntParameterWithExcludedValues("my_int", "an int param", []any{1}),
	} {
		if got := p.GetType(); got != parameters.TypeInt {
			t.Errorf("GetType() = %q, want %q", got, parameters.TypeInt)
		}
		if got := p.Manifest().Type; got != parameters.TypeInt {
			t.Errorf("Manifest().Type = %q, want %q", got, parameters.TypeInt)
		}
	}
}
