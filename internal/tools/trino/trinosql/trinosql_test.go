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

package trinosql

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestToolName(t *testing.T) {
	tool := Tool{
		Name: "test-trino-sql-tool",
	}

	got := tool.Name
	want := "test-trino-sql-tool"

	if got != want {
		t.Errorf("Tool.Name = %v, want %v", got, want)
	}
}

func TestToolDescription(t *testing.T) {
	manifest := tools.Manifest{
		Description: "Execute parameterized SQL queries on Trino",
	}

	tool := Tool{
		Name:     "test-tool",
		manifest: manifest,
	}

	got := tool.Manifest().Description
	want := "Execute parameterized SQL queries on Trino"

	if got != want {
		t.Errorf("Tool.Manifest().Description = %v, want %v", got, want)
	}
}

func TestToolParameters(t *testing.T) {
	tableNameParameter := tools.NewStringParameter("table_name", "Name of the table to query")
	params := tools.Parameters{tableNameParameter}

	tool := Tool{
		Name:       "test-tool",
		Parameters: params,
		manifest:   tools.Manifest{Parameters: params.Manifest()},
	}

	got := tool.Manifest().Parameters
	expected := params.Manifest()

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("Tool parameters mismatch (-want +got):\n%s", diff)
	}
}
