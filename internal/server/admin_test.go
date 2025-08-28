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

package server

import (
    "reflect"
    "encoding/json"
	"context"
	"net/http"
	"testing"
    "slices"

	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

var _ sources.Source = &MockSource{}
var _ sources.SourceConfig = &MockSourceConfig{}

// MockSource is used to mock sources in tests
type MockSource struct {
	name string
	kind string
}

func (s MockSource) SourceKind() string {
	return s.kind
}

type MockSourceConfig struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	Project  string `yaml:"project"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

func (sc MockSourceConfig) SourceConfigKind() string {
	return "mock-source"
}

func (sc MockSourceConfig) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	s := MockSource{
		name: sc.Name,
		kind: sc.Kind,
	}
	return s, nil
}

var sourceConfig1 = MockSourceConfig{
	Name:     "source1",
	Kind:     "mock-source",
	Project:  "my-project",
	User:     "my-user",
	Password: "my-password",
	Database: "my-db",
}

type MockAuthService struct {
	name     string
	kind     string
	clientID string
}

func (as MockAuthService) AuthServiceKind() string {
	return as.kind
}

func (as MockAuthService) GetName() string {
	return as.name
}

func (as MockAuthService) GetClaimsFromHeader(context.Context, http.Header) (map[string]any, error) {
	return map[string]any{"foo": "bar"}, nil
}

type MockASConfig struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	ClientID string `yaml:"clientId"`
}

func (ac MockASConfig) AuthServiceConfigKind() string {
	return "mock-auth-service"
}

func (ac MockASConfig) Initialize() (auth.AuthService, error) {
	a := MockAuthService{
		name:     ac.Name,
		kind:     ac.Kind,
		clientID: ac.ClientID,
	}
	return a, nil
}

var authService1 = MockASConfig{
	Name:     "auth-service1",
	Kind:     "mock-auth-service",
	ClientID: "foo",
}

func TestAdminGetResourceEndpoint(t *testing.T) {
	source1, _ := sourceConfig1.Initialize(context.Background(), nil)
	mockSources := []MockSource{source1.(MockSource)}
	as1, _ := authService1.Initialize()
	mockAuthServices := []MockAuthService{as1.(MockAuthService)}
	mockTools := []MockTool{tool1, tool2}
	sourcesMap, authServicesMap, toolsMap, toolsets := setUpResources(t, mockSources, mockAuthServices, mockTools)
	r, shutdown := setUpServer(t, "admin", sourcesMap, authServicesMap, toolsMap, toolsets)
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	// wantResponse is a struct for checks against test cases
	type wantResponse struct {
		statusCode    int
		isErr         bool
		errString     string
		resourcesList []string
	}

	testCases := []struct {
		name string
		url  string
		want wantResponse
	}{
		{
			name: "get source",
			url:  "/source",
			want: wantResponse{
				statusCode:    http.StatusOK,
				resourcesList: []string{"source1"},
			},
		},
		{
			name: "get auth services",
			url:  "/authservice",
			want: wantResponse{
				statusCode:    http.StatusOK,
				resourcesList: []string{"auth-service1"},
			},
		},
		{
			name: "get tool",
			url:  "/tool",
			want: wantResponse{
				statusCode:    http.StatusOK,
				resourcesList: []string{"no_params", "some_params"},
			},
		},
		{
			name: "get toolset",
			url:  "/toolset",
			want: wantResponse{
				statusCode:    http.StatusOK,
				resourcesList: []string{"", "tool1_only", "tool2_only"},
			},
		},
		{
			name: "get invalid",
			url:  "/invalid",
			want: wantResponse{
				statusCode:    http.StatusNotFound,
                isErr: true,
                errString: `invalid resource invalid, please provide one of "source", "authservice", "tool", or "toolset"`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body, err := runRequest(ts, http.MethodGet, tc.url, nil, nil)
			if err != nil {
				t.Fatalf("unexpected error during request: %s", err)
			}

			if contentType := resp.Header.Get("Content-type"); contentType != "application/json" {
				t.Fatalf("unexpected content-type header: want %s, got %s", "application/json", contentType)
			}

			if tc.want.statusCode != resp.StatusCode {
				t.Fatalf("unexpected status code: want %d, got %d", tc.want.statusCode, resp.StatusCode)
			}

			if tc.want.isErr {
                var res errResponse
                err = json.Unmarshal(body, &res)
                if err != nil {
                    t.Fatalf("error unmarshaling body: %s", err)
                }
				if tc.want.errString != res.ErrorText {
					t.Fatalf("unexpected error message: want %s, got %s", tc.want.errString, res.ErrorText)
				}
				return
			}

            var res []string
            err = json.Unmarshal(body, &res)
            if err != nil {
                t.Fatalf("error unmarshaling body: %s", err)
            }
            slices.Sort(res)
            if !reflect.DeepEqual(tc.want.resourcesList, res) {
				t.Fatalf("unexpected response: want %+v, got %+v", tc.want.resourcesList, res)
			}
		})
	}
}
