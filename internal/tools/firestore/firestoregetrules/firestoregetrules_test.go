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

package firestoregetrules_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/firestore/firestoregetrules"
)

func TestParseFromYamlFirestoreGetRules(t *testing.T) {
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
			kind: tools
			name: get_rules_tool
			type: firestore-get-rules
			source: my-firestore-instance
			description: Retrieves the active Firestore security rules for the current project
			`,
			want: server.ToolConfigs{
				"get_rules_tool": firestoregetrules.Config{
					Name:         "get_rules_tool",
					Type:         "firestore-get-rules",
					Source:       "my-firestore-instance",
					Description:  "Retrieves the active Firestore security rules for the current project",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth requirements",
			in: `
			kind: tools
			name: secure_get_rules
			type: firestore-get-rules
			source: prod-firestore
			description: Get Firestore security rules with authentication
			authRequired:
				- google-auth-service
				- admin-service
			`,
			want: server.ToolConfigs{
				"secure_get_rules": firestoregetrules.Config{
					Name:         "secure_get_rules",
					Type:         "firestore-get-rules",
					Source:       "prod-firestore",
					Description:  "Get Firestore security rules with authentication",
					AuthRequired: []string{"google-auth-service", "admin-service"},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, got, _, _, err := server.UnmarshalResourceConfig(ctx, testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
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
	kind: tools
	name: get_dev_rules
	type: firestore-get-rules
	source: dev-firestore
	description: Get development Firestore rules
	authRequired:
		- dev-auth
---
	kind: tools
	name: get_staging_rules
	type: firestore-get-rules
	source: staging-firestore
	description: Get staging Firestore rules
---
	kind: tools
	name: get_prod_rules
	type: firestore-get-rules
	source: prod-firestore
	description: Get production Firestore rules
	authRequired:
		- prod-auth
		- admin-auth
	`
	want := server.ToolConfigs{
		"get_dev_rules": firestoregetrules.Config{
			Name:         "get_dev_rules",
			Type:         "firestore-get-rules",
			Source:       "dev-firestore",
			Description:  "Get development Firestore rules",
			AuthRequired: []string{"dev-auth"},
		},
		"get_staging_rules": firestoregetrules.Config{
			Name:         "get_staging_rules",
			Type:         "firestore-get-rules",
			Source:       "staging-firestore",
			Description:  "Get staging Firestore rules",
			AuthRequired: []string{},
		},
		"get_prod_rules": firestoregetrules.Config{
			Name:         "get_prod_rules",
			Type:         "firestore-get-rules",
			Source:       "prod-firestore",
			Description:  "Get production Firestore rules",
			AuthRequired: []string{"prod-auth", "admin-auth"},
		},
	}

	_, _, _, got, _, _, err := server.UnmarshalResourceConfig(ctx, testutils.FormatYaml(in))
	if err != nil {
		t.Fatalf("unable to unmarshal: %s", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("incorrect parse: diff %v", diff)
	}
}
