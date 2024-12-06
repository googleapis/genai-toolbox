// Copyright 2024 Google LLC
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

package cmd

import (
	"bytes"
	_ "embed"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	cloudsqlpgsrc "github.com/googleapis/genai-toolbox/internal/sources/cloudsqlpg"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/postgressql"
	"github.com/spf13/cobra"
)

func withDefaults(c server.ServerConfig) server.ServerConfig {
	if c.Address == "" {
		c.Address = "127.0.0.1"
	}
	if c.Port == 0 {
		c.Port = 5000
	}
	return c
}

func invokeCommand(args []string) (*Command, string, error) {
	c := NewCommand()

	// Keep the test output quiet
	c.SilenceUsage = true
	c.SilenceErrors = true

	// Capture output
	buf := new(bytes.Buffer)
	c.SetOut(buf)
	c.SetErr(buf)
	c.SetArgs(args)

	// Disable execute behavior
	c.RunE = func(*cobra.Command, []string) error {
		return nil
	}

	err := c.Execute()

	return c, buf.String(), err
}

func TestVersion(t *testing.T) {
	data, err := os.ReadFile("version.txt")
	if err != nil {
		t.Fatalf("failed to read version.txt: %v", err)
	}
	want := strings.TrimSpace(string(data))

	_, got, err := invokeCommand([]string{"--version"})
	if err != nil {
		t.Fatalf("error invoking command: %s", err)
	}

	if !strings.Contains(got, want) {
		t.Errorf("cli did not return correct version: want %q, got %q", want, got)
	}
}

func TestServerConfigFlags(t *testing.T) {
	tcs := []struct {
		desc string
		args []string
		want server.ServerConfig
	}{
		{
			desc: "default values",
			args: []string{},
			want: withDefaults(server.ServerConfig{}),
		},
		{
			desc: "address short",
			args: []string{"-a", "127.0.1.1"},
			want: withDefaults(server.ServerConfig{
				Address: "127.0.1.1",
			}),
		},
		{
			desc: "address long",
			args: []string{"--address", "0.0.0.0"},
			want: withDefaults(server.ServerConfig{
				Address: "0.0.0.0",
			}),
		},
		{
			desc: "port short",
			args: []string{"-p", "5052"},
			want: withDefaults(server.ServerConfig{
				Port: 5052,
			}),
		},
		{
			desc: "port long",
			args: []string{"--port", "5050"},
			want: withDefaults(server.ServerConfig{
				Port: 5050,
			}),
		},
		{
			desc: "logging format",
			args: []string{"--logging-format", "JSON"},
			want: withDefaults(server.ServerConfig{
				LoggingFormat: "JSON",
			}),
		},
		{
			desc: "debug logs",
			args: []string{"--log-level", "WARN"},
			want: withDefaults(server.ServerConfig{
				LogLevel: "WARN",
			}),
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			c, _, err := invokeCommand(tc.args)
			if err != nil {
				t.Fatalf("unexpected error invoking command: %s", err)
			}

			if !cmp.Equal(c.cfg, tc.want) {
				t.Fatalf("got %v, want %v", c.cfg, tc.want)
			}
		})
	}
}

func TestFailServerConfigFlags(t *testing.T) {
	tcs := []struct {
		desc string
		args []string
	}{
		{
			desc: "logging format",
			args: []string{"--logging-format", "fail"},
		},
		{
			desc: "debug logs",
			args: []string{"--log-level", "fail"},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, err := invokeCommand(tc.args)
			if err == nil {
				t.Fatalf("expected an error, but got nil")
			}
		})
	}
}

func TestDefaultLoggingFormat(t *testing.T) {
	c, _, err := invokeCommand([]string{})
	if err != nil {
		t.Fatalf("unexpected error invoking command: %s", err)
	}
	got := c.cfg.LoggingFormat.String()
	want := "standard"
	if got != want {
		t.Fatalf("unexpected default logging format flag: got %v, want %v", got, want)
	}
}

func TestDefaultLogLevel(t *testing.T) {
	c, _, err := invokeCommand([]string{})
	if err != nil {
		t.Fatalf("unexpected error invoking command: %s", err)
	}
	got := c.cfg.LogLevel.String()
	want := "info"
	if got != want {
		t.Fatalf("unexpected default log level flag: got %v, want %v", got, want)
	}
}

func TestToolFileFlag(t *testing.T) {
	tcs := []struct {
		desc string
		args []string
		want string
	}{
		{
			desc: "default value",
			args: []string{},
			want: "tools.yaml",
		},
		{
			desc: "foo file",
			args: []string{"--tools_file", "foo.yaml"},
			want: "foo.yaml",
		},
		{
			desc: "address long",
			args: []string{"--tools_file", "bar.yaml"},
			want: "bar.yaml",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			c, _, err := invokeCommand(tc.args)
			if err != nil {
				t.Fatalf("unexpected error invoking command: %s", err)
			}
			if c.tools_file != tc.want {
				t.Fatalf("got %v, want %v", c.cfg, tc.want)
			}
		})
	}
}

func TestParseToolFile(t *testing.T) {
	tcs := []struct {
		description  string
		in           string
		wantSources  server.SourceConfigs
		wantTools    server.ToolConfigs
		wantToolsets server.ToolsetConfigs
	}{
		{
			description: "basic example",
			in: `
			sources:
				my-pg-instance:
					kind: cloud-sql-postgres
					project: my-project
					region: my-region
					instance: my-instance
					database: my_db
			tools:
				example_tool:
					kind: postgres-sql
					source: my-pg-instance
					description: some description
					statement: |
						SELECT * FROM SQL_STATEMENT;
					parameters:
						- name: country
						  type: string
						  description: some description
			toolsets:
				example_toolset:
					- example_tool
			`,
			wantSources: server.SourceConfigs{
				"my-pg-instance": cloudsqlpgsrc.Config{
					Name:     "my-pg-instance",
					Kind:     cloudsqlpgsrc.SourceKind,
					Project:  "my-project",
					Region:   "my-region",
					Instance: "my-instance",
					IPType:   "public",
					Database: "my_db",
				},
			},
			wantTools: server.ToolConfigs{
				"example_tool": postgressql.Config{
					Name:        "example_tool",
					Kind:        postgressql.ToolKind,
					Source:      "my-pg-instance",
					Description: "some description",
					Statement:   "SELECT * FROM SQL_STATEMENT;\n",
					Parameters: []tools.Parameter{
						tools.NewStringParameter("country", "some description"),
					},
				},
			},
			wantToolsets: server.ToolsetConfigs{
				"example_toolset": tools.ToolsetConfig{
					Name:      "example_toolset",
					ToolNames: []string{"example_tool"},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.description, func(t *testing.T) {
			gotSources, gotTools, gotToolsets, err := parseToolsFile(testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("failed to parse input: %v", err)
			}
			if diff := cmp.Diff(tc.wantSources, gotSources); diff != "" {
				t.Fatalf("incorrect sources parse: diff %v", diff)
			}
			if diff := cmp.Diff(tc.wantTools, gotTools); diff != "" {
				t.Fatalf("incorrect tools parse: diff %v", diff)
			}
			if diff := cmp.Diff(tc.wantToolsets, gotToolsets); diff != "" {
				t.Fatalf("incorrect tools parse: diff %v", diff)
			}
		})
	}

}
