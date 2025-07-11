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

package firestoregetdocuments_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/firestore/firestoregetdocuments"
)

func TestParseFromYamlFirestoreGetDocuments(t *testing.T) {
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
				get_docs_tool:
					kind: firestore-get-documents
					source: my-firestore-instance
					description: Retrieve documents from Firestore by paths
			`,
			want: server.ToolConfigs{
				"get_docs_tool": firestoregetdocuments.Config{
					Name:         "get_docs_tool",
					Kind:         "firestore-get-documents",
					Source:       "my-firestore-instance",
					Description:  "Retrieve documents from Firestore by paths",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth requirements",
			in: `
			tools:
				secure_get_docs:
					kind: firestore-get-documents
					source: prod-firestore
					description: Get documents with authentication
					authRequired:
						- google-auth-service
						- api-key-service
			`,
			want: server.ToolConfigs{
				"secure_get_docs": firestoregetdocuments.Config{
					Name:         "secure_get_docs",
					Kind:         "firestore-get-documents",
					Source:       "prod-firestore",
					Description:  "Get documents with authentication",
					AuthRequired: []string{"google-auth-service", "api-key-service"},
				},
			},
		},
		{
			desc: "complex configuration",
			in: `
			tools:
				user_data_retriever:
					kind: firestore-get-documents
					source: users-db-firestore
					description: |
						Retrieve user profile documents from Firestore.
						This tool can fetch multiple documents in a single call.
					authRequired:
						- oauth2-service
			`,
			want: server.ToolConfigs{
				"user_data_retriever": firestoregetdocuments.Config{
					Name:   "user_data_retriever",
					Kind:   "firestore-get-documents",
					Source: "users-db-firestore",
					Description: `Retrieve user profile documents from Firestore.
This tool can fetch multiple documents in a single call.
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
		get_user_docs:
			kind: firestore-get-documents
			source: users-firestore
			description: Get user documents
			authRequired:
				- user-auth
		get_product_docs:
			kind: firestore-get-documents
			source: products-firestore
			description: Get product documents
		get_order_docs:
			kind: firestore-get-documents
			source: orders-firestore
			description: Get order documents
			authRequired:
				- user-auth
				- admin-auth
	`
	want := server.ToolConfigs{
		"get_user_docs": firestoregetdocuments.Config{
			Name:         "get_user_docs",
			Kind:         "firestore-get-documents",
			Source:       "users-firestore",
			Description:  "Get user documents",
			AuthRequired: []string{"user-auth"},
		},
		"get_product_docs": firestoregetdocuments.Config{
			Name:         "get_product_docs",
			Kind:         "firestore-get-documents",
			Source:       "products-firestore",
			Description:  "Get product documents",
			AuthRequired: []string{},
		},
		"get_order_docs": firestoregetdocuments.Config{
			Name:         "get_order_docs",
			Kind:         "firestore-get-documents",
			Source:       "orders-firestore",
			Description:  "Get order documents",
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
