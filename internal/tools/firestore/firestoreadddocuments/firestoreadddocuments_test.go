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

package firestoreadddocuments_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/firestore/firestoreadddocuments"
)

func TestParseFromYamlFirestoreAddDocuments(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		desc string
		in   string
		want server.ToolConfigs
	}{
		{
			desc: "basic example",
			in: `
			tools:
				add_docs_tool:
					kind: firestore-add-documents
					source: my-firestore-instance
					description: Add documents to Firestore collections
			`,
			want: server.ToolConfigs{
				"add_docs_tool": firestoreadddocuments.Config{
					Name:         "add_docs_tool",
					Kind:         "firestore-add-documents",
					Source:       "my-firestore-instance",
					Description:  "Add documents to Firestore collections",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth requirements",
			in: `
			tools:
				secure_add_docs:
					kind: firestore-add-documents
					source: prod-firestore
					description: Add documents with authentication
					authRequired:
						- google-auth-service
						- api-key-service
			`,
			want: server.ToolConfigs{
				"secure_add_docs": firestoreadddocuments.Config{
					Name:         "secure_add_docs",
					Kind:         "firestore-add-documents",
					Source:       "prod-firestore",
					Description:  "Add documents with authentication",
					AuthRequired: []string{"google-auth-service", "api-key-service"},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Tools server.ToolConfigs `yaml:"tools"`
			}{}
			// Parse contents
			err := yaml.UnmarshalContext(ctx, testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got.Tools); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}

func TestParseFromYamlMultipleTools(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	in := `
	tools:
		add_user_docs:
			kind: firestore-add-documents
			source: users-firestore
			description: Add user documents
			authRequired:
				- user-auth
		add_product_docs:
			kind: firestore-add-documents
			source: products-firestore
			description: Add product documents
		add_order_docs:
			kind: firestore-add-documents
			source: orders-firestore
			description: Add order documents
			authRequired:
				- user-auth
				- admin-auth
	`
	want := server.ToolConfigs{
		"add_user_docs": firestoreadddocuments.Config{
			Name:         "add_user_docs",
			Kind:         "firestore-add-documents",
			Source:       "users-firestore",
			Description:  "Add user documents",
			AuthRequired: []string{"user-auth"},
		},
		"add_product_docs": firestoreadddocuments.Config{
			Name:         "add_product_docs",
			Kind:         "firestore-add-documents",
			Source:       "products-firestore",
			Description:  "Add product documents",
			AuthRequired: []string{},
		},
		"add_order_docs": firestoreadddocuments.Config{
			Name:         "add_order_docs",
			Kind:         "firestore-add-documents",
			Source:       "orders-firestore",
			Description:  "Add order documents",
			AuthRequired: []string{"user-auth", "admin-auth"},
		},
	}

	got := struct {
		Tools server.ToolConfigs `yaml:"tools"`
	}{}
	// Parse contents
	err = yaml.UnmarshalContext(ctx, testutils.FormatYaml(in), &got)
	if err != nil {
		t.Fatalf("unable to unmarshal: %s", err)
	}
	if diff := cmp.Diff(want, got.Tools); diff != "" {
		t.Fatalf("incorrect parse: diff %v", diff)
	}
}
