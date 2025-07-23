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

package trinoexecutesql

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestToolParameters(t *testing.T) {
	sqlParameter := tools.NewStringParameter("sql", "The SQL query to execute against the Trino database.")
	expected := tools.Parameters{sqlParameter}

	tool := Tool{
		Name:       "test-tool",
		Parameters: expected,
		manifest:   tools.Manifest{Parameters: expected.Manifest()},
	}

	params := tool.Manifest().Parameters
	expectedManifest := expected.Manifest()

	if diff := cmp.Diff(expectedManifest, params); diff != "" {
		t.Errorf("Tool parameters mismatch (-want +got):\n%s", diff)
	}
}

func TestToolName(t *testing.T) {
	tool := Tool{
		Name: "test-trino-tool",
	}

	got := tool.Name
	want := "test-trino-tool"

	if got != want {
		t.Errorf("Tool.Name = %v, want %v", got, want)
	}
}

func TestToolDescription(t *testing.T) {
	manifest := tools.Manifest{
		Description: "Execute SQL queries on Trino",
	}

	tool := Tool{
		Name:     "test-tool",
		manifest: manifest,
	}

	got := tool.Manifest().Description
	want := "Execute SQL queries on Trino"

	if got != want {
		t.Errorf("Tool.Manifest().Description = %v, want %v", got, want)
	}
}
