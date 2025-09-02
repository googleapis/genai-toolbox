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

package firestorequerycollectionparameterizable_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/firestore/firestorequerycollectionparameterizable"
)

func TestParseFromYamlFirestoreQueryCollectionParameterizable(t *testing.T) {
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
			desc: "basic example with parameterized collection path",
			in: `
			tools:
				query_users_tool:
					kind: firestore-query-collection-parameterizable
					source: my-firestore-instance
					description: Query users collection with parameterized path
					collectionPath: "users/{{.userId}}/documents"
					parameters:
						- name: userId
						  type: string
						  description: The user ID to query documents for
						  required: true
			`,
			want: server.ToolConfigs{
				"query_users_tool": firestorequerycollectionparameterizable.Config{
					Name:           "query_users_tool",
					Kind:           "firestore-query-collection-parameterizable",
					Source:         "my-firestore-instance",
					Description:    "Query users collection with parameterized path",
					CollectionPath: "users/{{.userId}}/documents",
					AuthRequired:   []string{},
					Parameters: tools.Parameters{
						tools.NewStringParameterWithRequired("userId", "The user ID to query documents for", true),
					},
				},
			},
		},
		{
			desc: "with parameterized filters",
			in: `
			tools:
				query_products_tool:
					kind: firestore-query-collection-parameterizable
					source: prod-firestore
					description: Query products with dynamic filters
					collectionPath: "products"
					filters: |
						{
							"and": [
								{"field": "category", "op": "==", "value": {"stringValue": "{{.category}}"}},
								{"field": "price", "op": "<=", "value": {"doubleValue": {{.maxPrice}}}}
							]
						}
					parameters:
						- name: category
						  type: string
						  description: Product category to filter by
						  required: true
						- name: maxPrice
						  type: float
						  description: Maximum price for products
						  required: true
			`,
			want: server.ToolConfigs{
				"query_products_tool": firestorequerycollectionparameterizable.Config{
					Name:           "query_products_tool",
					Kind:           "firestore-query-collection-parameterizable",
					Source:         "prod-firestore",
					Description:    "Query products with dynamic filters",
					CollectionPath: "products",
					Filters: `{
  "and": [
    {"field": "category", "op": "==", "value": {"stringValue": "{{.category}}"}},
    {"field": "price", "op": "<=", "value": {"doubleValue": {{.maxPrice}}}}
  ]
}
`,
					AuthRequired: []string{},
					Parameters: tools.Parameters{
						tools.NewStringParameterWithRequired("category", "Product category to filter by", true),
						tools.NewFloatParameterWithRequired("maxPrice", "Maximum price for products", true),
					},
				},
			},
		},
		{
			desc: "with select fields and orderBy",
			in: `
			tools:
				query_orders_tool:
					kind: firestore-query-collection-parameterizable
					source: orders-firestore
					description: Query orders with field selection
					collectionPath: "orders"
					select:
						- orderId
						- customerName
						- totalAmount
					orderBy:
						field: "{{.sortField}}"
						direction: "DESCENDING"
					limit: 50
					parameters:
						- name: sortField
						  type: string
						  description: Field to sort by
						  required: true
			`,
			want: server.ToolConfigs{
				"query_orders_tool": firestorequerycollectionparameterizable.Config{
					Name:           "query_orders_tool",
					Kind:           "firestore-query-collection-parameterizable",
					Source:         "orders-firestore",
					Description:    "Query orders with field selection",
					CollectionPath: "orders",
					Select:         []string{"orderId", "customerName", "totalAmount"},
					OrderBy: map[string]any{
						"field":     "{{.sortField}}",
						"direction": "DESCENDING",
					},
					Limit:        50,
					AuthRequired: []string{},
					Parameters: tools.Parameters{
						tools.NewStringParameterWithRequired("sortField", "Field to sort by", true),
					},
				},
			},
		},
		{
			desc: "with auth requirements and complex filters",
			in: `
			tools:
				secure_query_tool:
					kind: firestore-query-collection-parameterizable
					source: secure-firestore
					description: Query with authentication and complex filters
					collectionPath: "{{.collection}}"
					filters: |
						{
							"or": [
								{
									"and": [
										{"field": "status", "op": "==", "value": {"stringValue": "{{.status}}"}},
										{"field": "priority", "op": ">=", "value": {"integerValue": "{{.minPriority}}"}}
									]
								},
								{"field": "urgent", "op": "==", "value": {"booleanValue": true}}
							]
						}
					analyzeQuery: true
					authRequired:
						- google-auth-service
						- api-key-service
					parameters:
						- name: collection
						  type: string
						  description: Collection name to query
						  required: true
						- name: status
						  type: string
						  description: Status to filter by
						  required: true
						- name: minPriority
						  type: integer
						  description: Minimum priority level
						  default: 1
			`,
			want: server.ToolConfigs{
				"secure_query_tool": firestorequerycollectionparameterizable.Config{
					Name:           "secure_query_tool",
					Kind:           "firestore-query-collection-parameterizable",
					Source:         "secure-firestore",
					Description:    "Query with authentication and complex filters",
					CollectionPath: "{{.collection}}",
					Filters: `{
  "or": [
    {
      "and": [
        {"field": "status", "op": "==", "value": {"stringValue": "{{.status}}"}},
        {"field": "priority", "op": ">=", "value": {"integerValue": "{{.minPriority}}"}}
      ]
    },
    {"field": "urgent", "op": "==", "value": {"booleanValue": true}}
  ]
}
`,
					AnalyzeQuery: true,
					AuthRequired: []string{"google-auth-service", "api-key-service"},
					Parameters: tools.Parameters{
						tools.NewStringParameterWithRequired("collection", "Collection name to query", true),
						tools.NewStringParameterWithRequired("status", "Status to filter by", true),
						tools.NewIntParameterWithDefault("minPriority", 1, "Minimum priority level"),
					},
				},
			},
		},
		{
			desc: "with Firestore native JSON value types and template parameters",
			in: `
			tools:
				query_with_typed_values:
					kind: firestore-query-collection-parameterizable
					source: typed-firestore
					description: Query with Firestore native JSON value types
					collectionPath: "countries"
					filters: |
						{
							"or": [
								{"field": "continent", "op": "==", "value": {"stringValue": "{{.continent}}"}},
								{
									"and": [
										{"field": "area", "op": ">", "value": {"integerValue": "2000000"}},
										{"field": "area", "op": "<", "value": {"integerValue": "3000000"}},
										{"field": "population", "op": ">=", "value": {"integerValue": "{{.minPopulation}}"}},
										{"field": "gdp", "op": ">", "value": {"doubleValue": {{.minGdp}}}},
										{"field": "isActive", "op": "==", "value": {"booleanValue": {{.isActive}}}},
										{"field": "lastUpdated", "op": ">=", "value": {"timestampValue": "{{.startDate}}"}}
									]
								}
							]
						}
					parameters:
						- name: continent
						  type: string
						  description: Continent to filter by
						  required: true
						- name: minPopulation
						  type: string
						  description: Minimum population as string
						  required: true
						- name: minGdp
						  type: float
						  description: Minimum GDP value
						  required: true
						- name: isActive
						  type: boolean
						  description: Filter by active status
						  required: true
						- name: startDate
						  type: string
						  description: Start date in RFC3339 format
						  required: true
			`,
			want: server.ToolConfigs{
				"query_with_typed_values": firestorequerycollectionparameterizable.Config{
					Name:           "query_with_typed_values",
					Kind:           "firestore-query-collection-parameterizable",
					Source:         "typed-firestore",
					Description:    "Query with Firestore native JSON value types",
					CollectionPath: "countries",
					Filters: `{
  "or": [
    {"field": "continent", "op": "==", "value": {"stringValue": "{{.continent}}"}},
    {
      "and": [
        {"field": "area", "op": ">", "value": {"integerValue": "2000000"}},
        {"field": "area", "op": "<", "value": {"integerValue": "3000000"}},
        {"field": "population", "op": ">=", "value": {"integerValue": "{{.minPopulation}}"}},
        {"field": "gdp", "op": ">", "value": {"doubleValue": {{.minGdp}}}},
        {"field": "isActive", "op": "==", "value": {"booleanValue": {{.isActive}}}},
        {"field": "lastUpdated", "op": ">=", "value": {"timestampValue": "{{.startDate}}"}}
      ]
    }
  ]
}
`,
					AuthRequired: []string{},
					Parameters: tools.Parameters{
						tools.NewStringParameterWithRequired("continent", "Continent to filter by", true),
						tools.NewStringParameterWithRequired("minPopulation", "Minimum population as string", true),
						tools.NewFloatParameterWithRequired("minGdp", "Minimum GDP value", true),
						tools.NewBooleanParameterWithRequired("isActive", "Filter by active status", true),
						tools.NewStringParameterWithRequired("startDate", "Start date in RFC3339 format", true),
					},
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

func TestParseFromYamlMultipleParameterizableTools(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	in := `
	tools:
		query_user_posts:
			kind: firestore-query-collection-parameterizable
			source: social-firestore
			description: Query user posts with filtering
			collectionPath: "users/{{.userId}}/posts"
			filters: |
				{
					"and": [
						{"field": "visibility", "op": "==", "value": {"stringValue": "{{.visibility}}"}},
						{"field": "createdAt", "op": ">=", "value": {"timestampValue": "{{.startDate}}"}}
					]
				}
			select:
				- title
				- content
				- likes
			orderBy:
				field: createdAt
				direction: "{{.sortOrder}}"
			limit: 20
			parameters:
				- name: userId
				  type: string
				  description: User ID whose posts to query
				  required: true
				- name: visibility
				  type: string
				  description: Post visibility (public, private, friends)
				  required: true
				- name: startDate
				  type: string
				  description: Start date for posts
				  required: true
				- name: sortOrder
				  type: string
				  description: Sort order (ASCENDING or DESCENDING)
				  default: "DESCENDING"
		query_inventory:
			kind: firestore-query-collection-parameterizable
			source: inventory-firestore
			description: Query inventory items
			collectionPath: "warehouses/{{.warehouseId}}/inventory"
			filters: |
				{
					"field": "quantity", "op": "<", "value": {"integerValue": "{{.threshold}}"}}
			parameters:
				- name: warehouseId
				  type: string
				  description: Warehouse ID to check inventory
				  required: true
				- name: threshold
				  type: integer
				  description: Quantity threshold for low stock
				  required: true
		query_transactions:
			kind: firestore-query-collection-parameterizable
			source: finance-firestore
			description: Query financial transactions
			collectionPath: "accounts/{{.accountId}}/transactions"
			filters: |
				{
					"or": [
						{"field": "type", "op": "==", "value": {"stringValue": "{{.transactionType}}"}},
						{"field": "amount", "op": ">", "value": {"doubleValue": {{.minAmount}}}}
					]
				}
			analyzeQuery: true
			authRequired:
				- finance-auth
			parameters:
				- name: accountId
				  type: string
				  description: Account ID for transactions
				  required: true
				- name: transactionType
				  type: string
				  description: Type of transaction
				  default: "all"
				- name: minAmount
				  type: float
				  description: Minimum transaction amount
				  default: 0
	`
	want := server.ToolConfigs{
		"query_user_posts": firestorequerycollectionparameterizable.Config{
			Name:           "query_user_posts",
			Kind:           "firestore-query-collection-parameterizable",
			Source:         "social-firestore",
			Description:    "Query user posts with filtering",
			CollectionPath: "users/{{.userId}}/posts",
			Filters: `{
  "and": [
    {"field": "visibility", "op": "==", "value": {"stringValue": "{{.visibility}}"}},
    {"field": "createdAt", "op": ">=", "value": {"timestampValue": "{{.startDate}}"}}
  ]
}
`,
			Select: []string{"title", "content", "likes"},
			OrderBy: map[string]any{
				"field":     "createdAt",
				"direction": "{{.sortOrder}}",
			},
			Limit:        20,
			AuthRequired: []string{},
			Parameters: tools.Parameters{
				tools.NewStringParameterWithRequired("userId", "User ID whose posts to query", true),
				tools.NewStringParameterWithRequired("visibility", "Post visibility (public, private, friends)", true),
				tools.NewStringParameterWithRequired("startDate", "Start date for posts", true),
				tools.NewStringParameterWithDefault("sortOrder", "DESCENDING", "Sort order (ASCENDING or DESCENDING)"),
			},
		},
		"query_inventory": firestorequerycollectionparameterizable.Config{
			Name:           "query_inventory",
			Kind:           "firestore-query-collection-parameterizable",
			Source:         "inventory-firestore",
			Description:    "Query inventory items",
			CollectionPath: "warehouses/{{.warehouseId}}/inventory",
			Filters: `{
  "field": "quantity", "op": "<", "value": {"integerValue": "{{.threshold}}"}}
`,
			AuthRequired: []string{},
			Parameters: tools.Parameters{
				tools.NewStringParameterWithRequired("warehouseId", "Warehouse ID to check inventory", true),
				tools.NewIntParameterWithRequired("threshold", "Quantity threshold for low stock", true),
			},
		},
		"query_transactions": firestorequerycollectionparameterizable.Config{
			Name:           "query_transactions",
			Kind:           "firestore-query-collection-parameterizable",
			Source:         "finance-firestore",
			Description:    "Query financial transactions",
			CollectionPath: "accounts/{{.accountId}}/transactions",
			Filters: `{
  "or": [
    {"field": "type", "op": "==", "value": {"stringValue": "{{.transactionType}}"}},
    {"field": "amount", "op": ">", "value": {"doubleValue": {{.minAmount}}}}
  ]
}
`,
			AnalyzeQuery: true,
			AuthRequired: []string{"finance-auth"},
			Parameters: tools.Parameters{
				tools.NewStringParameterWithRequired("accountId", "Account ID for transactions", true),
				tools.NewStringParameterWithDefault("transactionType", "all", "Type of transaction"),
				tools.NewFloatParameterWithDefault("minAmount", 0, "Minimum transaction amount"),
			},
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
