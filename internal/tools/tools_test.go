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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestGetMcpManifestMetadata(t *testing.T) {
	authServices := []tools.ParamAuthService{
		{
			Name:  "my-google-auth-service",
			Field: "auth_field",
		},
		{
			Name:  "other-auth-service",
			Field: "other_auth_field",
		}}
	tcs := []struct {
		desc         string
		name         string
		description  string
		authInvoke   []string
		params       tools.Parameters
		wantMetadata map[string]any
	}{
		{
			desc:         "basic manifest without metadata",
			name:         "basic",
			description:  "foo bar",
			authInvoke:   []string{},
			params:       tools.Parameters{tools.NewStringParameter("string-param", "string parameter")},
			wantMetadata: nil,
		},
		{
			desc:         "with auth invoke metadata",
			name:         "basic",
			description:  "foo bar",
			authInvoke:   []string{"auth1", "auth2"},
			params:       tools.Parameters{tools.NewStringParameter("string-param", "string parameter")},
			wantMetadata: map[string]any{"toolbox/authInvoke": []string{"auth1", "auth2"}},
		},
		{
			desc:        "with auth param metadata",
			name:        "basic",
			description: "foo bar",
			authInvoke:  []string{},
			params:      tools.Parameters{tools.NewStringParameterWithAuth("string-param", "string parameter", authServices)},
			wantMetadata: map[string]any{
				"toolbox/authParam": map[string][]string{
					"string-param": []string{"my-google-auth-service", "other-auth-service"},
				},
			},
		},
		{
			desc:        "with auth invoke and auth param metadata",
			name:        "basic",
			description: "foo bar",
			authInvoke:  []string{"auth1", "auth2"},
			params:      tools.Parameters{tools.NewStringParameterWithAuth("string-param", "string parameter", authServices)},
			wantMetadata: map[string]any{
				"toolbox/authInvoke": []string{"auth1", "auth2"},
				"toolbox/authParam": map[string][]string{
					"string-param": []string{"my-google-auth-service", "other-auth-service"},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := tools.GetMcpManifest(tc.name, tc.description, tc.authInvoke, tc.params)
			gotM := got.Metadata
			if diff := cmp.Diff(tc.wantMetadata, gotM); diff != "" {
				t.Fatalf("unexpected metadata (-want +got):\n%s", diff)
			}
		})
	}
}
