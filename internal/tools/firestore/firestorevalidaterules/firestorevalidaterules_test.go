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

package firestorevalidaterules_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/firestore/firestorevalidaterules"
)

func TestParseFromYamlFirestoreValidateRules(t *testing.T) {
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
				validate_rules_tool:
					kind: firestore-validate-rules
					source: my-firestore-instance
					description: Validate Firestore security rules
			`,
			want: server.ToolConfigs{
				"validate_rules_tool": firestorevalidaterules.Config{
					Name:         "validate_rules_tool",
					Type:         "firestore-validate-rules",
					Source:       "my-firestore-instance",
					Description:  "Validate Firestore security rules",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth requirements",
			in: `
			tools:
				secure_validate_rules:
					kind: firestore-validate-rules
					source: prod-firestore
					description: Validate rules with authentication
					authRequired:
						- google-auth-service
						- api-key-service
			`,
			want: server.ToolConfigs{
				"secure_validate_rules": firestorevalidaterules.Config{
					Name:         "secure_validate_rules",
					Type:         "firestore-validate-rules",
					Source:       "prod-firestore",
					Description:  "Validate rules with authentication",
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
		validate_dev_rules:
			kind: firestore-validate-rules
			source: dev-firestore
			description: Validate development environment rules
			authRequired:
				- dev-auth
		validate_staging_rules:
			kind: firestore-validate-rules
			source: staging-firestore
			description: Validate staging environment rules
		validate_prod_rules:
			kind: firestore-validate-rules
			source: prod-firestore
			description: Validate production environment rules
			authRequired:
				- prod-auth
				- admin-auth
	`
	want := server.ToolConfigs{
		"validate_dev_rules": firestorevalidaterules.Config{
			Name:         "validate_dev_rules",
			Type:         "firestore-validate-rules",
			Source:       "dev-firestore",
			Description:  "Validate development environment rules",
			AuthRequired: []string{"dev-auth"},
		},
		"validate_staging_rules": firestorevalidaterules.Config{
			Name:         "validate_staging_rules",
			Type:         "firestore-validate-rules",
			Source:       "staging-firestore",
			Description:  "Validate staging environment rules",
			AuthRequired: []string{},
		},
		"validate_prod_rules": firestorevalidaterules.Config{
			Name:         "validate_prod_rules",
			Type:         "firestore-validate-rules",
			Source:       "prod-firestore",
			Description:  "Validate production environment rules",
			AuthRequired: []string{"prod-auth", "admin-auth"},
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
