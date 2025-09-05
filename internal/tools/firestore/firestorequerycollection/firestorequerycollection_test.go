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

package firestorequerycollection_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/firestore/firestorequerycollection"
)

func TestParseFromYamlFirestoreQueryCollection(t *testing.T) {
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
				query_users_tool:
					kind: firestore-query-collection
					source: my-firestore-instance
					description: Query users collection with filters and ordering
			`,
			want: server.ToolConfigs{
				"query_users_tool": firestorequerycollection.Config{
					Name:         "query_users_tool",
					Type:         "firestore-query-collection",
					Source:       "my-firestore-instance",
					Description:  "Query users collection with filters and ordering",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth requirements",
			in: `
			tools:
				secure_query_tool:
					kind: firestore-query-collection
					source: prod-firestore
					description: Query collections with authentication
					authRequired:
						- google-auth-service
						- api-key-service
			`,
			want: server.ToolConfigs{
				"secure_query_tool": firestorequerycollection.Config{
					Name:         "secure_query_tool",
					Type:         "firestore-query-collection",
					Source:       "prod-firestore",
					Description:  "Query collections with authentication",
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
		query_users:
			kind: firestore-query-collection
			source: users-firestore
			description: Query user documents with filtering
			authRequired:
				- user-auth
		query_products:
			kind: firestore-query-collection
			source: products-firestore
			description: Query product catalog
		query_orders:
			kind: firestore-query-collection
			source: orders-firestore
			description: Query customer orders with complex filters
			authRequired:
				- user-auth
				- admin-auth
	`
	want := server.ToolConfigs{
		"query_users": firestorequerycollection.Config{
			Name:         "query_users",
			Type:         "firestore-query-collection",
			Source:       "users-firestore",
			Description:  "Query user documents with filtering",
			AuthRequired: []string{"user-auth"},
		},
		"query_products": firestorequerycollection.Config{
			Name:         "query_products",
			Type:         "firestore-query-collection",
			Source:       "products-firestore",
			Description:  "Query product catalog",
			AuthRequired: []string{},
		},
		"query_orders": firestorequerycollection.Config{
			Name:         "query_orders",
			Type:         "firestore-query-collection",
			Source:       "orders-firestore",
			Description:  "Query customer orders with complex filters",
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
