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

package firestoredeletedocuments_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/firestore/firestoredeletedocuments"
)

func TestParseFromYamlFirestoreDeleteDocuments(t *testing.T) {
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
				delete_docs_tool:
					kind: firestore-delete-documents
					source: my-firestore-instance
					description: Delete documents from Firestore by paths
			`,
			want: server.ToolConfigs{
				"delete_docs_tool": firestoredeletedocuments.Config{
					Name:         "delete_docs_tool",
					Kind:         "firestore-delete-documents",
					Source:       "my-firestore-instance",
					Description:  "Delete documents from Firestore by paths",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth requirements",
			in: `
			tools:
				secure_delete_docs:
					kind: firestore-delete-documents
					source: prod-firestore
					description: Delete documents with authentication
					authRequired:
						- google-auth-service
						- api-key-service
			`,
			want: server.ToolConfigs{
				"secure_delete_docs": firestoredeletedocuments.Config{
					Name:         "secure_delete_docs",
					Kind:         "firestore-delete-documents",
					Source:       "prod-firestore",
					Description:  "Delete documents with authentication",
					AuthRequired: []string{"google-auth-service", "api-key-service"},
				},
			},
		},
		{
			desc: "complex configuration",
			in: `
			tools:
				bulk_deleter:
					kind: firestore-delete-documents
					source: users-db-firestore
					description: |
						Delete multiple documents from Firestore.
						This tool can delete multiple documents in a single call using bulk operations.
					authRequired:
						- oauth2-service
			`,
			want: server.ToolConfigs{
				"bulk_deleter": firestoredeletedocuments.Config{
					Name:   "bulk_deleter",
					Kind:   "firestore-delete-documents",
					Source: "users-db-firestore",
					Description: `Delete multiple documents from Firestore.
This tool can delete multiple documents in a single call using bulk operations.
`,
					AuthRequired: []string{"oauth2-service"},
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
		delete_user_docs:
			kind: firestore-delete-documents
			source: users-firestore
			description: Delete user documents
			authRequired:
				- user-auth
		delete_product_docs:
			kind: firestore-delete-documents
			source: products-firestore
			description: Delete product documents
		delete_order_docs:
			kind: firestore-delete-documents
			source: orders-firestore
			description: Delete order documents
			authRequired:
				- user-auth
				- admin-auth
	`
	want := server.ToolConfigs{
		"delete_user_docs": firestoredeletedocuments.Config{
			Name:         "delete_user_docs",
			Kind:         "firestore-delete-documents",
			Source:       "users-firestore",
			Description:  "Delete user documents",
			AuthRequired: []string{"user-auth"},
		},
		"delete_product_docs": firestoredeletedocuments.Config{
			Name:         "delete_product_docs",
			Kind:         "firestore-delete-documents",
			Source:       "products-firestore",
			Description:  "Delete product documents",
			AuthRequired: []string{},
		},
		"delete_order_docs": firestoredeletedocuments.Config{
			Name:         "delete_order_docs",
			Kind:         "firestore-delete-documents",
			Source:       "orders-firestore",
			Description:  "Delete order documents",
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
