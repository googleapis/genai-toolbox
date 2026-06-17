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

package http_test

import (
	"bytes"
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/log"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	sourcehttp "github.com/googleapis/mcp-toolbox/internal/sources/http"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	http "github.com/googleapis/mcp-toolbox/internal/tools/http"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

// mockSourceProvider implements tools.SourceProvider backed by a real Source.
type mockSourceProvider struct {
	sources map[string]sources.Source
}

func (m *mockSourceProvider) GetSource(name string) (sources.Source, bool) {
	s, ok := m.sources[name]
	return s, ok
}

func TestParseFromYamlHTTP(t *testing.T) {
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
			kind: tool
			name: example_tool
			type: http
			source: my-instance
			method: GET
			description: some description
			path: search
			`,
			want: server.ToolConfigs{
				"example_tool": http.Config{
					ConfigBase: tools.ConfigBase{
						Name:         "example_tool",
						Description:  "some description",
						AuthRequired: []string{},
					},
					Type:   "http",
					Source: "my-instance",
					Method: "GET",
					Path:   "search",
				},
			},
		},
		{
			desc: "advanced example",
			in: `
			kind: tool
			name: example_tool
			type: http
			source: my-instance
			method: GET
			path: "{{.pathParam}}?name=alice&pet=cat"
			description: some description
			authRequired:
				- my-google-auth-service
				- other-auth-service
			queryParams:
				- name: country
				  type: string
				  description: some description
				  authServices:
					- name: my-google-auth-service
					  field: user_id
					- name: other-auth-service
					  field: user_id
			pathParams:
			    - name: pathParam
			      type: string
			      description: path param
			requestBody: |
					{
						"age": {{.age}},
						"city": "{{.city}}",
						"food": {{.food}}
					}
			bodyParams:
				- name: age
				  type: integer
				  description: age num
				- name: city
				  type: string
				  description: city string
			headers:
				Authorization: API_KEY
				Content-Type: application/json
			headerParams:
				- name: Language
				  type: string
				  description: language string
			`,
			want: server.ToolConfigs{
				"example_tool": http.Config{
					ConfigBase: tools.ConfigBase{
						Name:         "example_tool",
						Description:  "some description",
						AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
					},
					Type:   "http",
					Source: "my-instance",
					Method: "GET",
					Path:   "{{.pathParam}}?name=alice&pet=cat",
					QueryParams: []parameters.Parameter{
						parameters.NewStringParameterWithAuth("country", "some description",
							[]parameters.ParamAuthService{{Name: "my-google-auth-service", Field: "user_id"},
								{Name: "other-auth-service", Field: "user_id"}}),
					},
					PathParams: parameters.Parameters{
						&parameters.StringParameter{
							CommonParameter: parameters.CommonParameter{Name: "pathParam", Type: "string", Desc: "path param"},
						},
					},
					RequestBody: `{
  "age": {{.age}},
  "city": "{{.city}}",
  "food": {{.food}}
}
`,
					BodyParams:   []parameters.Parameter{parameters.NewIntParameter("age", "age num"), parameters.NewStringParameter("city", "city string")},
					Headers:      map[string]string{"Authorization": "API_KEY", "Content-Type": "application/json"},
					HeaderParams: []parameters.Parameter{parameters.NewStringParameter("Language", "language string")},
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

func TestFailParseFromYamlHTTP(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "Invalid method",
			in: `
			kind: tool
			name: example_tool
			type: http
			source: my-instance
			method: GOT
			path: "search?name=alice&pet=cat"
			description: some description
			authRequired:
				- my-google-auth-service
				- other-auth-service
			queryParams:
				- name: country
				  type: string
				  description: some description
				  authServices:
					- name: my-google-auth-service
					  field: user_id
					- name: other-auth-service
					  field: user_id
			requestBody: |
					{
						"age": {{.age}},
						"city": "{{.city}}"
					}
			bodyParams:
				- name: age
				  type: integer
				  description: age num
				- name: city
				  type: string
				  description: city string
			headers:
				Authorization: API_KEY
				Content-Type: application/json
			headerParams:
				- name: Language
				  type: string
				  description: language string
			`,
			err: `GOT is not a valid http method`,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, _, _, _, err := server.UnmarshalResourceConfig(ctx, testutils.FormatYaml(tc.in))
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			errStr := err.Error()
			if !strings.Contains(errStr, tc.err) {
				t.Fatalf("unexpected error string: got %q, want substring %q", errStr, tc.err)
			}
		})
	}

}

func newTestContext(t *testing.T) context.Context {
	t.Helper()
	logger, err := log.NewLogger("standard", log.Debug, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	return util.WithLogger(context.Background(), logger)
}

func TestInvokeForwardAuthorizationHeader(t *testing.T) {
	tcs := []struct {
		desc                       string
		forwardAuthorizationHeader bool
		accessToken                tools.AccessToken
		staticHeader               string // pre-configured Authorization header on the tool
		wantAuthHeader             string // what the upstream server should receive
	}{
		{
			desc:                       "forwards token when enabled and token present",
			forwardAuthorizationHeader: true,
			accessToken:                "Bearer caller-token",
			wantAuthHeader:             "Bearer caller-token",
		},
		{
			desc:                       "does not forward when disabled",
			forwardAuthorizationHeader: false,
			accessToken:                "Bearer caller-token",
			wantAuthHeader:             "",
		},
		{
			desc:                       "does not forward when token is empty",
			forwardAuthorizationHeader: true,
			accessToken:                "",
			wantAuthHeader:             "",
		},
		{
			desc:                       "forwarded token overrides static tool header",
			forwardAuthorizationHeader: true,
			accessToken:                "Bearer caller-token",
			staticHeader:               "Bearer static-key",
			wantAuthHeader:             "Bearer caller-token",
		},
		{
			desc:                       "static header preserved when forwarding disabled",
			forwardAuthorizationHeader: false,
			accessToken:                "Bearer caller-token",
			staticHeader:               "Bearer static-key",
			wantAuthHeader:             "Bearer static-key",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			var receivedAuth string
			upstream := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
				receivedAuth = r.Header.Get("Authorization")
				w.WriteHeader(nethttp.StatusOK)
				_, _ = w.Write([]byte(`"ok"`))
			}))
			defer upstream.Close()

			ctx := newTestContext(t)

			sourceCfg := sourcehttp.Config{
				Name:                       "test-source",
				Type:                       sourcehttp.SourceType,
				BaseURL:                    upstream.URL + "/",
				Timeout:                    "5s",
				ForwardAuthorizationHeader: tc.forwardAuthorizationHeader,
			}
			initializedSource, err := sourceCfg.Initialize(ctx, nil)
			if err != nil {
				t.Fatalf("failed to initialize source: %v", err)
			}

			toolHeaders := map[string]string{}
			if tc.staticHeader != "" {
				toolHeaders["Authorization"] = tc.staticHeader
			}

			toolCfg := http.Config{
				Name:        "test_tool",
				Type:        "http",
				Source:      "test-source",
				Description: "test tool",
				Method:      "GET",
				Path:        "ping",
				Headers:     toolHeaders,
			}
			srcs := map[string]sources.Source{"test-source": initializedSource}
			tool, err := toolCfg.Initialize(srcs)
			if err != nil {
				t.Fatalf("failed to initialize tool: %v", err)
			}

			provider := &mockSourceProvider{sources: srcs}
			_, toolboxErr := tool.Invoke(ctx, provider, parameters.ParamValues{}, tc.accessToken)
			if toolboxErr != nil {
				t.Fatalf("unexpected invoke error: %v", toolboxErr)
			}

			if receivedAuth != tc.wantAuthHeader {
				t.Errorf("Authorization header: got %q, want %q", receivedAuth, tc.wantAuthHeader)
			}
		})
	}
}
