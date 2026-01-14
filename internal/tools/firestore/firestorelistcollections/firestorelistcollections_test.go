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

package firestorelistcollections_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/firestore/firestorelistcollections"
)

func TestParseFromYamlFirestoreListCollections(t *testing.T) {
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
				list_collections_tool:
					kind: firestore-list-collections
					source: my-firestore-instance
					description: List collections in Firestore
			`,
			want: server.ToolConfigs{
				"list_collections_tool": firestorelistcollections.Config{
					Name:         "list_collections_tool",
					Type:         "firestore-list-collections",
					Source:       "my-firestore-instance",
					Description:  "List collections in Firestore",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth requirements",
			in: `
			tools:
				secure_list_collections:
					kind: firestore-list-collections
					source: prod-firestore
					description: List collections with authentication
					authRequired:
						- google-auth-service
						- api-key-service
			`,
			want: server.ToolConfigs{
				"secure_list_collections": firestorelistcollections.Config{
					Name:         "secure_list_collections",
					Type:         "firestore-list-collections",
					Source:       "prod-firestore",
					Description:  "List collections with authentication",
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
		list_user_collections:
			kind: firestore-list-collections
			source: users-firestore
			description: List user-related collections
			authRequired:
				- user-auth
		list_product_collections:
			kind: firestore-list-collections
			source: products-firestore
			description: List product-related collections
		list_admin_collections:
			kind: firestore-list-collections
			source: admin-firestore
			description: List administrative collections
			authRequired:
				- user-auth
				- admin-auth
	`
	want := server.ToolConfigs{
		"list_user_collections": firestorelistcollections.Config{
			Name:         "list_user_collections",
			Type:         "firestore-list-collections",
			Source:       "users-firestore",
			Description:  "List user-related collections",
			AuthRequired: []string{"user-auth"},
		},
		"list_product_collections": firestorelistcollections.Config{
			Name:         "list_product_collections",
			Type:         "firestore-list-collections",
			Source:       "products-firestore",
			Description:  "List product-related collections",
			AuthRequired: []string{},
		},
		"list_admin_collections": firestorelistcollections.Config{
			Name:         "list_admin_collections",
			Type:         "firestore-list-collections",
			Source:       "admin-firestore",
			Description:  "List administrative collections",
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
