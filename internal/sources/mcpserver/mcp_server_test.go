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

package mcpserver_test

import (
	"context"
	"fmt"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/mcpserver"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYaml(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-mcp-server:
					kind: mcp-server
					endpoint: http://127.0.0.1:8080/mcp
					specVersion: 2025-03-26
					transport: http
			`,
			want: map[string]sources.SourceConfig{
				"my-mcp-server": mcpserver.Config{
					Name:        "my-mcp-server",
					Kind:        mcpserver.SourceKind,
					Endpoint:    "http://127.0.0.1:8080/mcp",
					SpecVersion: mcpserver.MAR_2025,
					Transport:   mcpserver.HTTP,
					AuthMethod:  mcpserver.None,
					AuthSecret:  "",
				},
			},
		},
		{
			desc: "defaults example",
			in: `
			sources:
				my-mcp-server:
					kind: mcp-server
					endpoint: http://127.0.0.1:8080/mcp
			`,
			want: map[string]sources.SourceConfig{
				"my-mcp-server": mcpserver.Config{
					Name:        "my-mcp-server",
					Kind:        mcpserver.SourceKind,
					Endpoint:    "http://127.0.0.1:8080/mcp",
					SpecVersion: mcpserver.MAR_2025,
					Transport:   mcpserver.HTTP,
					AuthMethod:  mcpserver.None,
					AuthSecret:  "",
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if !cmp.Equal(tc.want, got.Sources) {
				t.Fatalf("incorrect parse: want %v, got %v", tc.want, got.Sources)
			}
		})
	}
}

func TestParseSpecVersionFromYaml(t *testing.T) {
	tcs := []mcpserver.SpecVersion{mcpserver.NOV_2024, mcpserver.MAR_2025, mcpserver.JUN_2025}
	for _, tc := range tcs {
		t.Run(string(tc), func(t *testing.T) {
			template := `
			sources:
				my-mcp-server:
					kind: mcp-server
					endpoint: http://127.0.0.1:8080/mcp
					specVersion: %s
			`
			in := fmt.Sprintf(template, string(tc))
			want := map[string]sources.SourceConfig{
				"my-mcp-server": mcpserver.Config{
					Name:        "my-mcp-server",
					Kind:        mcpserver.SourceKind,
					Endpoint:    "http://127.0.0.1:8080/mcp",
					SpecVersion: tc,
					Transport:   mcpserver.HTTP,
					AuthMethod:  mcpserver.None,
				},
			}
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}

			wantCfg, wantOk := want["my-mcp-server"].(mcpserver.Config)
			gotCfg, gotOk := got.Sources["my-mcp-server"].(mcpserver.Config)
			if !wantOk || !gotOk {
				t.Fatalf("type assertion failed: wantOk=%v gotOk=%v", wantOk, gotOk)
			}
			if !cmp.Equal(wantCfg, gotCfg, cmpopts.IgnoreUnexported(mcpserver.Config{})) {
				t.Fatalf("incorrect parse: want %v, got %v", wantCfg, gotCfg)
			}
		})
	}
}

func TestParseTransportTypeFromYaml(t *testing.T) {
	tcs := []mcpserver.TransportType{mcpserver.STDIO, mcpserver.SSE, mcpserver.HTTP}
	for _, tc := range tcs {
		t.Run(string(tc), func(t *testing.T) {
			template := `
			sources:
				my-mcp-server:
					kind: mcp-server
					endpoint: http://127.0.0.1:8080/mcp
					transport: %s
			`
			in := fmt.Sprintf(template, tc)
			want := map[string]sources.SourceConfig{
				"my-mcp-server": mcpserver.Config{
					Name:        "my-mcp-server",
					Kind:        mcpserver.SourceKind,
					Endpoint:    "http://127.0.0.1:8080/mcp",
					SpecVersion: mcpserver.MAR_2025,
					Transport:   tc,
					AuthMethod:  mcpserver.None,
				},
			}
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}

			wantCfg, wantOk := want["my-mcp-server"].(mcpserver.Config)
			gotCfg, gotOk := got.Sources["my-mcp-server"].(mcpserver.Config)
			if !wantOk || !gotOk {
				t.Fatalf("type assertion failed: wantOk=%v gotOk=%v", wantOk, gotOk)
			}
			if !cmp.Equal(wantCfg, gotCfg, cmpopts.IgnoreUnexported(mcpserver.Config{})) {
				t.Fatalf("incorrect parse: want %v, got %v", wantCfg, gotCfg)
			}
		})
	}
}

func TestParseAuthMethodFromYaml(t *testing.T) {
	tcs := []mcpserver.AuthMethod{mcpserver.None, mcpserver.ApiKey, mcpserver.Bearer}
	for _, tc := range tcs {
		t.Run(string(tc), func(t *testing.T) {
			template := `
			sources:
				my-mcp-server:
					kind: mcp-server
					endpoint: http://127.0.0.1:8080/mcp
					authMethod: %s
			`
			in := fmt.Sprintf(template, tc)
			want := map[string]sources.SourceConfig{
				"my-mcp-server": mcpserver.Config{
					Name:        "my-mcp-server",
					Kind:        mcpserver.SourceKind,
					Endpoint:    "http://127.0.0.1:8080/mcp",
					SpecVersion: mcpserver.MAR_2025,
					Transport:   mcpserver.HTTP,
					AuthMethod:  tc,
				},
			}
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}

			wantCfg, wantOk := want["my-mcp-server"].(mcpserver.Config)
			gotCfg, gotOk := got.Sources["my-mcp-server"].(mcpserver.Config)
			if !wantOk || !gotOk {
				t.Fatalf("type assertion failed: wantOk=%v gotOk=%v", wantOk, gotOk)
			}
			if !cmp.Equal(wantCfg, gotCfg, cmpopts.IgnoreUnexported(mcpserver.Config{})) {
				t.Fatalf("incorrect parse: want %v, got %v", wantCfg, gotCfg)
			}
		})
	}
}

func TestFailParseFromYaml(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "extra field",
			in: `
			sources:
				my-mcp-server:
					kind: mcp-server
					endpoint: http://127.0.0.1:8080/mcp
					foo: bar
			`,
			err: "unable to parse source \"my-mcp-server\" as \"mcp-server\": [2:1] unknown field \"foo\"\n   1 | endpoint: http://127.0.0.1:8080/mcp\n>  2 | foo: bar\n       ^\n   3 | kind: mcp-server",
		},
		{
			desc: "missing required endpoint",
			in: `
			sources:
				my-neo4j-instance:
					kind: mcp-server
			`,
			err: "unable to parse source \"my-neo4j-instance\" as \"mcp-server\": Key: 'Config.Endpoint' Error:Field validation for 'Endpoint' failed on the 'required' tag",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			errStr := err.Error()
			if errStr != tc.err {
				t.Fatalf("unexpected error: got %q, want %q", errStr, tc.err)
			}
		})
	}
}

func TestFailInitializeFromYaml(t *testing.T) {
	tcs := []struct {
		desc string
		in   mcpserver.Config
		err  string
	}{
		{
			desc: "endpoint with invalid URI",
			in: mcpserver.Config{
				Name:        "my-mcp-server",
				Kind:        mcpserver.SourceKind,
				AuthMethod:  mcpserver.None,
				SpecVersion: mcpserver.MAR_2025,
				Transport:   mcpserver.HTTP,
				Endpoint:    "invalid-uri",
			},
			err: "failed when parsing endpoint: parse \"invalid-uri\": invalid URI for request",
		},
		{
			desc: "apiKey authMethod with missing secret",
			in: mcpserver.Config{
				Name:        "my-mcp-server",
				Kind:        mcpserver.SourceKind,
				SpecVersion: mcpserver.MAR_2025,
				Transport:   mcpserver.HTTP,
				Endpoint:    "http://127.0.0.1:8080/mcp",
				AuthMethod:  mcpserver.ApiKey,
				AuthSecret:  "", // empty secret
			},
			err: "authSecret is required when authMethod is set",
		},
		{
			desc: "bearer authMethod with missing secret",
			in: mcpserver.Config{
				Name:        "my-mcp-server",
				Kind:        mcpserver.SourceKind,
				SpecVersion: mcpserver.MAR_2025,
				Transport:   mcpserver.HTTP,
				Endpoint:    "http://127.0.0.1:8080/mcp",
				AuthMethod:  mcpserver.Bearer,
				AuthSecret:  "", // empty secret
			},
			err: "authSecret is required when authMethod is set",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, err := tc.in.Initialize(context.Background(), nil)
			if err == nil {
				t.Fatalf("expect Initialize to fail")
			}
			errStr := err.Error()
			if errStr != tc.err {
				t.Fatalf("unexpected error: got %q, want %q", errStr, tc.err)
			}
		})
	}
}
