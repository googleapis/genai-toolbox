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

package tools_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

func TestGetMcpManifestMetadata(t *testing.T) {
	trueVal := true
	falseVal := false

	authServices := []parameters.ParamAuthService{
		{
			Name:  "my-google-auth-service",
			Field: "auth_field",
		},
		{
			Name:  "other-auth-service",
			Field: "other_auth_field",
		}}
	tcs := []struct {
		desc            string
		name            string
		description     string
		authInvoke      []string
		params          parameters.Parameters
		annotations     *tools.ToolAnnotations
		wantMetadata    map[string]any
		wantAnnotations []byte
	}{
		{
			desc:         "basic manifest without metadata",
			name:         "basic",
			description:  "foo bar",
			authInvoke:   []string{},
			params:       parameters.Parameters{parameters.NewStringParameter("string-param", "string parameter")},
			annotations:  nil,
			wantMetadata: nil,
		},
		{
			desc:            "basic manifest without metadata with annotations",
			name:            "basic",
			description:     "foo bar",
			authInvoke:      []string{},
			params:          parameters.Parameters{parameters.NewStringParameter("string-param", "string parameter")},
			annotations:     &tools.ToolAnnotations{ReadOnlyHint: &trueVal, DestructiveHint: &falseVal},
			wantMetadata:    nil,
			wantAnnotations: []byte(`{"destructiveHint":false,"readOnlyHint":true}`),
		},
		{
			desc:         "with auth invoke metadata",
			name:         "basic",
			description:  "foo bar",
			authInvoke:   []string{"auth1", "auth2"},
			params:       parameters.Parameters{parameters.NewStringParameter("string-param", "string parameter")},
			annotations:  nil,
			wantMetadata: map[string]any{"toolbox/authInvoke": []string{"auth1", "auth2"}},
		},
		{
			desc:        "with auth param metadata",
			name:        "basic",
			description: "foo bar",
			authInvoke:  []string{},
			params:      parameters.Parameters{parameters.NewStringParameterWithAuth("string-param", "string parameter", authServices)},
			annotations: nil,
			wantMetadata: map[string]any{
				"toolbox/authParam": map[string][]string{
					"string-param": {"my-google-auth-service", "other-auth-service"},
				},
			},
		},
		{
			desc:        "with auth invoke and auth param metadata",
			name:        "basic",
			description: "foo bar",
			authInvoke:  []string{"auth1", "auth2"},
			params:      parameters.Parameters{parameters.NewStringParameterWithAuth("string-param", "string parameter", authServices)},
			annotations: nil,
			wantMetadata: map[string]any{
				"toolbox/authInvoke": []string{"auth1", "auth2"},
				"toolbox/authParam": map[string][]string{
					"string-param": {"my-google-auth-service", "other-auth-service"},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := tools.GetMcpManifest(tc.name, tc.description, tc.authInvoke, tc.params, tc.annotations)
			gotM := got.Metadata
			if diff := cmp.Diff(tc.wantMetadata, gotM); diff != "" {
				t.Fatalf("unexpected metadata (-want +got):\n%s", diff)
			}

			if got.Annotations != nil {
				annotations, _ := json.Marshal(got.Annotations)
				if diff := cmp.Diff(tc.wantAnnotations, annotations); diff != "" {
					t.Fatalf("unexpected annotations (-want +got):\n%s", diff)
				}
			}

		})
	}
}
