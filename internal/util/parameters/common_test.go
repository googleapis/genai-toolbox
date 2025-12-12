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

package parameters_test

import (
	"strings"
	"testing"
	"text/template"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

func TestPopulateTemplate(t *testing.T) {
	tcs := []struct {
		name           string
		templateName   string
		templateString string
		data           map[string]any
		want           string
		wantErr        bool
	}{
		{
			name:           "simple string substitution",
			templateName:   "test",
			templateString: "Hello {{.name}}!",
			data:           map[string]any{"name": "World"},
			want:           "Hello World!",
			wantErr:        false,
		},
		{
			name:           "multiple substitutions",
			templateName:   "test",
			templateString: "{{.greeting}} {{.name}}, you are {{.age}} years old",
			data:           map[string]any{"greeting": "Hello", "name": "Alice", "age": 30},
			want:           "Hello Alice, you are 30 years old",
			wantErr:        false,
		},
		{
			name:           "empty template",
			templateName:   "test",
			templateString: "",
			data:           map[string]any{},
			want:           "",
			wantErr:        false,
		},
		{
			name:           "no substitutions",
			templateName:   "test",
			templateString: "Plain text without templates",
			data:           map[string]any{},
			want:           "Plain text without templates",
			wantErr:        false,
		},
		{
			name:           "invalid template syntax",
			templateName:   "test",
			templateString: "{{.name",
			data:           map[string]any{"name": "World"},
			want:           "",
			wantErr:        true,
		},
		{
			name:           "missing field",
			templateName:   "test",
			templateString: "{{.missing}}",
			data:           map[string]any{"name": "World"},
			want:           "<no value>",
			wantErr:        false,
		},
		{
			name:           "invalid function call",
			templateName:   "test",
			templateString: "{{.name.invalid}}",
			data:           map[string]any{"name": "World"},
			want:           "",
			wantErr:        true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parameters.PopulateTemplate(tc.templateName, tc.templateString, tc.data)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect result (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPopulateTemplateWithFunc(t *testing.T) {
	// Custom function for testing
	customFuncs := template.FuncMap{
		"upper": strings.ToUpper,
		"add": func(a, b int) int {
			return a + b
		},
	}

	tcs := []struct {
		name           string
		templateName   string
		templateString string
		data           map[string]any
		funcMap        template.FuncMap
		want           string
		wantErr        bool
	}{
		{
			name:           "with custom upper function",
			templateName:   "test",
			templateString: "{{upper .text}}",
			data:           map[string]any{"text": "hello"},
			funcMap:        customFuncs,
			want:           "HELLO",
			wantErr:        false,
		},
		{
			name:           "with custom add function",
			templateName:   "test",
			templateString: "Result: {{add .x .y}}",
			data:           map[string]any{"x": 5, "y": 3},
			funcMap:        customFuncs,
			want:           "Result: 8",
			wantErr:        false,
		},
		{
			name:           "nil funcMap",
			templateName:   "test",
			templateString: "Hello {{.name}}",
			data:           map[string]any{"name": "World"},
			funcMap:        nil,
			want:           "Hello World",
			wantErr:        false,
		},
		{
			name:           "combine custom function with regular substitution",
			templateName:   "test",
			templateString: "{{upper .greeting}} {{.name}}!",
			data:           map[string]any{"greeting": "hello", "name": "Alice"},
			funcMap:        customFuncs,
			want:           "HELLO Alice!",
			wantErr:        false,
		},
		{
			name:           "undefined function",
			templateName:   "test",
			templateString: "{{undefined .text}}",
			data:           map[string]any{"text": "hello"},
			funcMap:        nil,
			want:           "",
			wantErr:        true,
		},
		{
			name:           "wrong number of arguments",
			templateName:   "test",
			templateString: "{{upper}}",
			data:           map[string]any{},
			funcMap:        template.FuncMap{"upper": strings.ToUpper},
			want:           "",
			wantErr:        true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parameters.PopulateTemplateWithFunc(tc.templateName, tc.templateString, tc.data, tc.funcMap)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect result (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPopulateTemplateWithJSON(t *testing.T) {
	tcs := []struct {
		name           string
		templateName   string
		templateString string
		data           map[string]any
		want           string
		wantErr        bool
	}{
		{
			name:           "json string",
			templateName:   "test",
			templateString: "Data: {{json .value}}",
			data:           map[string]any{"value": "hello"},
			want:           `Data: "hello"`,
			wantErr:        false,
		},
		{
			name:           "json number",
			templateName:   "test",
			templateString: "Number: {{json .num}}",
			data:           map[string]any{"num": 42},
			want:           "Number: 42",
			wantErr:        false,
		},
		{
			name:           "json boolean",
			templateName:   "test",
			templateString: "Bool: {{json .flag}}",
			data:           map[string]any{"flag": true},
			want:           "Bool: true",
			wantErr:        false,
		},
		{
			name:           "json array",
			templateName:   "test",
			templateString: "Array: {{json .items}}",
			data:           map[string]any{"items": []any{"a", "b", "c"}},
			want:           `Array: ["a","b","c"]`,
			wantErr:        false,
		},
		{
			name:           "json object",
			templateName:   "test",
			templateString: "Object: {{json .obj}}",
			data:           map[string]any{"obj": map[string]any{"name": "Alice", "age": 30}},
			want:           `Object: {"age":30,"name":"Alice"}`,
			wantErr:        false,
		},
		{
			name:           "json null",
			templateName:   "test",
			templateString: "Null: {{json .nullValue}}",
			data:           map[string]any{"nullValue": nil},
			want:           "Null: null",
			wantErr:        false,
		},
		{
			name:           "combine json with regular substitution",
			templateName:   "test",
			templateString: "User {{.name}} has data: {{json .data}}",
			data:           map[string]any{"name": "Bob", "data": map[string]any{"id": 123}},
			want:           `User Bob has data: {"id":123}`,
			wantErr:        false,
		},
		{
			name:           "missing field for json",
			templateName:   "test",
			templateString: "{{json .missing}}",
			data:           map[string]any{},
			want:           "null",
			wantErr:        false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parameters.PopulateTemplateWithJSON(tc.templateName, tc.templateString, tc.data)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect result (-want +got):\n%s", diff)
			}
		})
	}
}
