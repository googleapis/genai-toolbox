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
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/googleapis/genai-toolbox/internal/auth/google"
	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/prebuiltconfigs"
	"github.com/googleapis/genai-toolbox/internal/server"
	cloudsqlpgsrc "github.com/googleapis/genai-toolbox/internal/sources/cloudsqlpg"
	httpsrc "github.com/googleapis/genai-toolbox/internal/sources/http"
	"github.com/googleapis/genai-toolbox/internal/telemetry"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/http"
	"github.com/googleapis/genai-toolbox/internal/tools/postgres/postgressql"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/spf13/cobra"
)

func withDefaults(c server.ServerConfig) server.ServerConfig {
	data, _ := os.ReadFile("version.txt")
	version := strings.TrimSpace(string(data)) // Preserving 'data', new var for clarity
	c.Version = version + "+" + strings.Join([]string{"dev", runtime.GOOS, runtime.GOARCH}, ".")

	if c.Address == "" {
		c.Address = "127.0.0.1"
	}
	if c.Port == 0 {
		c.Port = 5000
	}
	if c.TelemetryServiceName == "" {
		c.TelemetryServiceName = "toolbox"
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
		{
			desc: "telemetry gcp",
			args: []string{"--telemetry-gcp"},
			want: withDefaults(server.ServerConfig{
				TelemetryGCP: true,
			}),
		},
		{
			desc: "telemetry otlp",
			args: []string{"--telemetry-otlp", "http://127.0.0.1:4553"},
			want: withDefaults(server.ServerConfig{
				TelemetryOTLP: "http://127.0.0.1:4553",
			}),
		},
		{
			desc: "telemetry service name",
			args: []string{"--telemetry-service-name", "toolbox-custom"},
			want: withDefaults(server.ServerConfig{
				TelemetryServiceName: "toolbox-custom",
			}),
		},
		{
			desc: "stdio",
			args: []string{"--stdio"},
			want: withDefaults(server.ServerConfig{
				Stdio: true,
			}),
		},
		{
			desc: "disable reload",
			args: []string{"--disable-reload"},
			want: withDefaults(server.ServerConfig{
				DisableReload: true,
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

func TestParseEnv(t *testing.T) {
	tcs := []struct {
		desc      string
		env       map[string]string
		in        string
		want      string
		err       bool
		errString string
	}{
		{
			desc:      "without default without env",
			in:        "${FOO}",
			want:      "",
			err:       true,
			errString: `environment variable not found: "FOO"`,
		},
		{
			desc: "without default with env",
			env: map[string]string{
				"FOO": "bar",
			},
			in:   "${FOO}",
			want: "bar",
		},
		{
			desc: "with empty default",
			in:   "${FOO:}",
			want: "",
		},
		{
			desc: "with default",
			in:   "${FOO:bar}",
			want: "bar",
		},
		{
			desc: "with default with env",
			env: map[string]string{
				"FOO": "hello",
			},
			in:   "${FOO:bar}",
			want: "hello",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.env != nil {
				for k, v := range tc.env {
					t.Setenv(k, v)
				}
			}
			got, err := parseEnv(tc.in)
			if tc.err {
				if err == nil {
					t.Fatalf("expected error not found")
				}
				if tc.errString != err.Error() {
					t.Fatalf("incorrect error string: got %s, want %s", err, tc.errString)
				}
			}
			if tc.want != got {
				t.Fatalf("unexpected want: got %s, want %s", got, tc.want)
			}
		})
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
			want: "",
		},
		{
			desc: "foo file",
			args: []string{"--tools-file", "foo.yaml"},
			want: "foo.yaml",
		},
		{
			desc: "address long",
			args: []string{"--tools-file", "bar.yaml"},
			want: "bar.yaml",
		},
		{
			desc: "deprecated flag",
			args: []string{"--tools_file", "foo.yaml"},
			want: "foo.yaml",
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

func TestToolsFilesFlag(t *testing.T) {
	tcs := []struct {
		desc string
		args []string
		want []string
	}{
		{
			desc: "no value",
			args: []string{},
			want: []string{},
		},
		{
			desc: "single file",
			args: []string{"--tools-files", "foo.yaml"},
			want: []string{"foo.yaml"},
		},
		{
			desc: "multiple files",
			args: []string{"--tools-files", "foo.yaml,bar.yaml"},
			want: []string{"foo.yaml", "bar.yaml"},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			c, _, err := invokeCommand(tc.args)
			if err != nil {
				t.Fatalf("unexpected error invoking command: %s", err)
			}
			if diff := cmp.Diff(c.tools_files, tc.want); diff != "" {
				t.Fatalf("got %v, want %v", c.tools_files, tc.want)
			}
		})
	}
}

func TestToolsFolderFlag(t *testing.T) {
	tcs := []struct {
		desc string
		args []string
		want string
	}{
		{
			desc: "no value",
			args: []string{},
			want: "",
		},
		{
			desc: "folder set",
			args: []string{"--tools-folder", "test-folder"},
			want: "test-folder",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			c, _, err := invokeCommand(tc.args)
			if err != nil {
				t.Fatalf("unexpected error invoking command: %s", err)
			}
			if c.tools_folder != tc.want {
				t.Fatalf("got %v, want %v", c.tools_folder, tc.want)
			}
		})
	}
}

func TestPrebuiltFlag(t *testing.T) {
	tcs := []struct {
		desc string
		args []string
		want string
	}{
		{
			desc: "default value",
			args: []string{},
			want: "",
		},
		{
			desc: "custom pre built flag",
			args: []string{"--tools-file", "alloydb"},
			want: "alloydb",
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

func TestParseToolFile(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		description   string
		in            string
		wantToolsFile ToolsFile
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
					user: my_user
					password: my_pass
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
			wantToolsFile: ToolsFile{
				Sources: server.SourceConfigs{
					"my-pg-instance": cloudsqlpgsrc.Config{
						Name:     "my-pg-instance",
						Kind:     cloudsqlpgsrc.SourceKind,
						Project:  "my-project",
						Region:   "my-region",
						Instance: "my-instance",
						IPType:   "public",
						Database: "my_db",
						User:     "my_user",
						Password: "my_pass",
					},
				},
				Tools: server.ToolConfigs{
					"example_tool": postgressql.Config{
						Name:        "example_tool",
						Kind:        "postgres-sql",
						Source:      "my-pg-instance",
						Description: "some description",
						Statement:   "SELECT * FROM SQL_STATEMENT;\n",
						Parameters: []tools.Parameter{
							tools.NewStringParameter("country", "some description"),
						},
						AuthRequired: []string{},
					},
				},
				Toolsets: server.ToolsetConfigs{
					"example_toolset": tools.ToolsetConfig{
						Name:      "example_toolset",
						ToolNames: []string{"example_tool"},
					},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.description, func(t *testing.T) {
			toolsFile, err := parseToolsFile(ctx, testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("failed to parse input: %v", err)
			}
			if diff := cmp.Diff(tc.wantToolsFile.Sources, toolsFile.Sources); diff != "" {
				t.Fatalf("incorrect sources parse: diff %v", diff)
			}
			if diff := cmp.Diff(tc.wantToolsFile.AuthServices, toolsFile.AuthServices); diff != "" {
				t.Fatalf("incorrect authServices parse: diff %v", diff)
			}
			if diff := cmp.Diff(tc.wantToolsFile.Tools, toolsFile.Tools); diff != "" {
				t.Fatalf("incorrect tools parse: diff %v", diff)
			}
			if diff := cmp.Diff(tc.wantToolsFile.Toolsets, toolsFile.Toolsets); diff != "" {
				t.Fatalf("incorrect tools parse: diff %v", diff)
			}
		})
	}

}

func TestParseToolFileWithAuth(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		description   string
		in            string
		wantToolsFile ToolsFile
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
					user: my_user
					password: my_pass
			authServices:
				my-google-service:
					kind: google
					clientId: my-client-id
				other-google-service:
					kind: google
					clientId: other-client-id

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
						- name: id
						  type: integer
						  description: user id
						  authServices:
							- name: my-google-service
								field: user_id
						- name: email
							type: string
							description: user email
							authServices:
							- name: my-google-service
							  field: email
							- name: other-google-service
							  field: other_email

			toolsets:
				example_toolset:
					- example_tool
			`,
			wantToolsFile: ToolsFile{
				Sources: server.SourceConfigs{
					"my-pg-instance": cloudsqlpgsrc.Config{
						Name:     "my-pg-instance",
						Kind:     cloudsqlpgsrc.SourceKind,
						Project:  "my-project",
						Region:   "my-region",
						Instance: "my-instance",
						IPType:   "public",
						Database: "my_db",
						User:     "my_user",
						Password: "my_pass",
					},
				},
				AuthServices: server.AuthServiceConfigs{
					"my-google-service": google.Config{
						Name:     "my-google-service",
						Kind:     google.AuthServiceKind,
						ClientID: "my-client-id",
					},
					"other-google-service": google.Config{
						Name:     "other-google-service",
						Kind:     google.AuthServiceKind,
						ClientID: "other-client-id",
					},
				},
				Tools: server.ToolConfigs{
					"example_tool": postgressql.Config{
						Name:         "example_tool",
						Kind:         "postgres-sql",
						Source:       "my-pg-instance",
						Description:  "some description",
						Statement:    "SELECT * FROM SQL_STATEMENT;\n",
						AuthRequired: []string{},
						Parameters: []tools.Parameter{
							tools.NewStringParameter("country", "some description"),
							tools.NewIntParameterWithAuth("id", "user id", []tools.ParamAuthService{{Name: "my-google-service", Field: "user_id"}}),
							tools.NewStringParameterWithAuth("email", "user email", []tools.ParamAuthService{{Name: "my-google-service", Field: "email"}, {Name: "other-google-service", Field: "other_email"}}),
						},
					},
				},
				Toolsets: server.ToolsetConfigs{
					"example_toolset": tools.ToolsetConfig{
						Name:      "example_toolset",
						ToolNames: []string{"example_tool"},
					},
				},
			},
		},
		{
			description: "basic example with authSources",
			in: `
			sources:
				my-pg-instance:
					kind: cloud-sql-postgres
					project: my-project
					region: my-region
					instance: my-instance
					database: my_db
					user: my_user
					password: my_pass
			authSources:
				my-google-service:
					kind: google
					clientId: my-client-id
				other-google-service:
					kind: google
					clientId: other-client-id

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
						- name: id
						  type: integer
						  description: user id
						  authSources:
							- name: my-google-service
								field: user_id
						- name: email
							type: string
							description: user email
							authSources:
							- name: my-google-service
							  field: email
							- name: other-google-service
							  field: other_email

			toolsets:
				example_toolset:
					- example_tool
			`,
			wantToolsFile: ToolsFile{
				Sources: server.SourceConfigs{
					"my-pg-instance": cloudsqlpgsrc.Config{
						Name:     "my-pg-instance",
						Kind:     cloudsqlpgsrc.SourceKind,
						Project:  "my-project",
						Region:   "my-region",
						Instance: "my-instance",
						IPType:   "public",
						Database: "my_db",
						User:     "my_user",
						Password: "my_pass",
					},
				},
				AuthSources: server.AuthServiceConfigs{
					"my-google-service": google.Config{
						Name:     "my-google-service",
						Kind:     google.AuthServiceKind,
						ClientID: "my-client-id",
					},
					"other-google-service": google.Config{
						Name:     "other-google-service",
						Kind:     google.AuthServiceKind,
						ClientID: "other-client-id",
					},
				},
				Tools: server.ToolConfigs{
					"example_tool": postgressql.Config{
						Name:         "example_tool",
						Kind:         "postgres-sql",
						Source:       "my-pg-instance",
						Description:  "some description",
						Statement:    "SELECT * FROM SQL_STATEMENT;\n",
						AuthRequired: []string{},
						Parameters: []tools.Parameter{
							tools.NewStringParameter("country", "some description"),
							tools.NewIntParameterWithAuth("id", "user id", []tools.ParamAuthService{{Name: "my-google-service", Field: "user_id"}}),
							tools.NewStringParameterWithAuth("email", "user email", []tools.ParamAuthService{{Name: "my-google-service", Field: "email"}, {Name: "other-google-service", Field: "other_email"}}),
						},
					},
				},
				Toolsets: server.ToolsetConfigs{
					"example_toolset": tools.ToolsetConfig{
						Name:      "example_toolset",
						ToolNames: []string{"example_tool"},
					},
				},
			},
		},
		{
			description: "basic example with authRequired",
			in: `
			sources:
				my-pg-instance:
					kind: cloud-sql-postgres
					project: my-project
					region: my-region
					instance: my-instance
					database: my_db
					user: my_user
					password: my_pass
			authServices:
				my-google-service:
					kind: google
					clientId: my-client-id
				other-google-service:
					kind: google
					clientId: other-client-id

			tools:
				example_tool:
					kind: postgres-sql
					source: my-pg-instance
					description: some description
					statement: |
						SELECT * FROM SQL_STATEMENT;
					authRequired:
						- my-google-service
					parameters:
						- name: country
						  type: string
						  description: some description
						- name: id
						  type: integer
						  description: user id
						  authServices:
							- name: my-google-service
								field: user_id
						- name: email
							type: string
							description: user email
							authServices:
							- name: my-google-service
							  field: email
							- name: other-google-service
							  field: other_email

			toolsets:
				example_toolset:
					- example_tool
			`,
			wantToolsFile: ToolsFile{
				Sources: server.SourceConfigs{
					"my-pg-instance": cloudsqlpgsrc.Config{
						Name:     "my-pg-instance",
						Kind:     cloudsqlpgsrc.SourceKind,
						Project:  "my-project",
						Region:   "my-region",
						Instance: "my-instance",
						IPType:   "public",
						Database: "my_db",
						User:     "my_user",
						Password: "my_pass",
					},
				},
				AuthServices: server.AuthServiceConfigs{
					"my-google-service": google.Config{
						Name:     "my-google-service",
						Kind:     google.AuthServiceKind,
						ClientID: "my-client-id",
					},
					"other-google-service": google.Config{
						Name:     "other-google-service",
						Kind:     google.AuthServiceKind,
						ClientID: "other-client-id",
					},
				},
				Tools: server.ToolConfigs{
					"example_tool": postgressql.Config{
						Name:         "example_tool",
						Kind:         "postgres-sql",
						Source:       "my-pg-instance",
						Description:  "some description",
						Statement:    "SELECT * FROM SQL_STATEMENT;\n",
						AuthRequired: []string{"my-google-service"},
						Parameters: []tools.Parameter{
							tools.NewStringParameter("country", "some description"),
							tools.NewIntParameterWithAuth("id", "user id", []tools.ParamAuthService{{Name: "my-google-service", Field: "user_id"}}),
							tools.NewStringParameterWithAuth("email", "user email", []tools.ParamAuthService{{Name: "my-google-service", Field: "email"}, {Name: "other-google-service", Field: "other_email"}}),
						},
					},
				},
				Toolsets: server.ToolsetConfigs{
					"example_toolset": tools.ToolsetConfig{
						Name:      "example_toolset",
						ToolNames: []string{"example_tool"},
					},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.description, func(t *testing.T) {
			toolsFile, err := parseToolsFile(ctx, testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("failed to parse input: %v", err)
			}
			if diff := cmp.Diff(tc.wantToolsFile.Sources, toolsFile.Sources); diff != "" {
				t.Fatalf("incorrect sources parse: diff %v", diff)
			}
			if diff := cmp.Diff(tc.wantToolsFile.AuthServices, toolsFile.AuthServices); diff != "" {
				t.Fatalf("incorrect authServices parse: diff %v", diff)
			}
			if diff := cmp.Diff(tc.wantToolsFile.Tools, toolsFile.Tools); diff != "" {
				t.Fatalf("incorrect tools parse: diff %v", diff)
			}
			if diff := cmp.Diff(tc.wantToolsFile.Toolsets, toolsFile.Toolsets); diff != "" {
				t.Fatalf("incorrect tools parse: diff %v", diff)
			}
		})
	}

}

func TestEnvVarReplacement(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	t.Setenv("TestHeader", "ACTUAL_HEADER")
	t.Setenv("API_KEY", "ACTUAL_API_KEY")
	t.Setenv("clientId", "ACTUAL_CLIENT_ID")
	t.Setenv("clientId2", "ACTUAL_CLIENT_ID_2")
	t.Setenv("toolset_name", "ACTUAL_TOOLSET_NAME")
	t.Setenv("cat_string", "cat")
	t.Setenv("food_string", "food")
	t.Setenv("TestHeader", "ACTUAL_HEADER")

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		description   string
		in            string
		wantToolsFile ToolsFile
	}{
		{
			description: "file with env var example",
			in: `
			sources:
				my-http-instance:
					kind: http
					baseUrl: http://test_server/
					timeout: 10s
					headers:
						Authorization: ${TestHeader}
					queryParams:
						api-key: ${API_KEY}
			authServices:
				my-google-service:
					kind: google
					clientId: ${clientId}
				other-google-service:
					kind: google
					clientId: ${clientId2}

			tools:
				example_tool:
					kind: http
					source: my-instance
					method: GET
					path: "search?name=alice&pet=${cat_string}"
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
								"city": "{{.city}}",
								"food": "${food_string}",
								"other": "$OTHER"
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

			toolsets:
				${toolset_name}:
					- example_tool
			`,
			wantToolsFile: ToolsFile{
				Sources: server.SourceConfigs{
					"my-http-instance": httpsrc.Config{
						Name:           "my-http-instance",
						Kind:           httpsrc.SourceKind,
						BaseURL:        "http://test_server/",
						Timeout:        "10s",
						DefaultHeaders: map[string]string{"Authorization": "ACTUAL_HEADER"},
						QueryParams:    map[string]string{"api-key": "ACTUAL_API_KEY"},
					},
				},
				AuthServices: server.AuthServiceConfigs{
					"my-google-service": google.Config{
						Name:     "my-google-service",
						Kind:     google.AuthServiceKind,
						ClientID: "ACTUAL_CLIENT_ID",
					},
					"other-google-service": google.Config{
						Name:     "other-google-service",
						Kind:     google.AuthServiceKind,
						ClientID: "ACTUAL_CLIENT_ID_2",
					},
				},
				Tools: server.ToolConfigs{
					"example_tool": http.Config{
						Name:         "example_tool",
						Kind:         "http",
						Source:       "my-instance",
						Method:       "GET",
						Path:         "search?name=alice&pet=cat",
						Description:  "some description",
						AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
						QueryParams: []tools.Parameter{
							tools.NewStringParameterWithAuth("country", "some description",
								[]tools.ParamAuthService{{Name: "my-google-auth-service", Field: "user_id"},
									{Name: "other-auth-service", Field: "user_id"}}),
						},
						RequestBody: `{
  "age": {{.age}},
  "city": "{{.city}}",
  "food": "food",
  "other": "$OTHER"
}
`,
						BodyParams:   []tools.Parameter{tools.NewIntParameter("age", "age num"), tools.NewStringParameter("city", "city string")},
						Headers:      map[string]string{"Authorization": "API_KEY", "Content-Type": "application/json"},
						HeaderParams: []tools.Parameter{tools.NewStringParameter("Language", "language string")},
					},
				},
				Toolsets: server.ToolsetConfigs{
					"ACTUAL_TOOLSET_NAME": tools.ToolsetConfig{
						Name:      "ACTUAL_TOOLSET_NAME",
						ToolNames: []string{"example_tool"},
					},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.description, func(t *testing.T) {
			toolsFile, err := parseToolsFile(ctx, testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("failed to parse input: %v", err)
			}
			if diff := cmp.Diff(tc.wantToolsFile.Sources, toolsFile.Sources); diff != "" {
				t.Fatalf("incorrect sources parse: diff %v", diff)
			}
			if diff := cmp.Diff(tc.wantToolsFile.AuthServices, toolsFile.AuthServices); diff != "" {
				t.Fatalf("incorrect authServices parse: diff %v", diff)
			}
			if diff := cmp.Diff(tc.wantToolsFile.Tools, toolsFile.Tools); diff != "" {
				t.Fatalf("incorrect tools parse: diff %v", diff)
			}
			if diff := cmp.Diff(tc.wantToolsFile.Toolsets, toolsFile.Toolsets); diff != "" {
				t.Fatalf("incorrect tools parse: diff %v", diff)
			}
		})
	}

}

// normalizeFilepaths is a helper function to allow same filepath formats for Mac and Windows.
// this prevents needing multiple "want" cases for TestResolveWatcherInputs
func normalizeFilepaths(m map[string]bool) map[string]bool {
	newMap := make(map[string]bool)
	for k, v := range m {
		newMap[filepath.ToSlash(k)] = v
	}
	return newMap
}

func TestResolveWatcherInputs(t *testing.T) {
	tcs := []struct {
		description      string
		toolsFile        string
		toolsFiles       []string
		toolsFolder      string
		wantWatchDirs    map[string]bool
		wantWatchedFiles map[string]bool
	}{
		{
			description:      "single tools file",
			toolsFile:        "tools_folder/example_tools.yaml",
			toolsFiles:       []string{},
			toolsFolder:      "",
			wantWatchDirs:    map[string]bool{"tools_folder": true},
			wantWatchedFiles: map[string]bool{"tools_folder/example_tools.yaml": true},
		},
		{
			description:      "default tools file (root dir)",
			toolsFile:        "tools.yaml",
			toolsFiles:       []string{},
			toolsFolder:      "",
			wantWatchDirs:    map[string]bool{".": true},
			wantWatchedFiles: map[string]bool{"tools.yaml": true},
		},
		{
			description:   "multiple files in different folders",
			toolsFile:     "",
			toolsFiles:    []string{"tools_folder/example_tools.yaml", "tools_folder2/example_tools.yaml"},
			toolsFolder:   "",
			wantWatchDirs: map[string]bool{"tools_folder": true, "tools_folder2": true},
			wantWatchedFiles: map[string]bool{
				"tools_folder/example_tools.yaml":  true,
				"tools_folder2/example_tools.yaml": true,
			},
		},
		{
			description:   "multiple files in same folder",
			toolsFile:     "",
			toolsFiles:    []string{"tools_folder/example_tools.yaml", "tools_folder/example_tools2.yaml"},
			toolsFolder:   "",
			wantWatchDirs: map[string]bool{"tools_folder": true},
			wantWatchedFiles: map[string]bool{
				"tools_folder/example_tools.yaml":  true,
				"tools_folder/example_tools2.yaml": true,
			},
		},
		{
			description: "multiple files in different levels",
			toolsFile:   "",
			toolsFiles: []string{
				"tools_folder/example_tools.yaml",
				"tools_folder/special_tools/example_tools2.yaml"},
			toolsFolder:   "",
			wantWatchDirs: map[string]bool{"tools_folder": true, "tools_folder/special_tools": true},
			wantWatchedFiles: map[string]bool{
				"tools_folder/example_tools.yaml":                true,
				"tools_folder/special_tools/example_tools2.yaml": true,
			},
		},
		{
			description:      "tools folder",
			toolsFile:        "",
			toolsFiles:       []string{},
			toolsFolder:      "tools_folder",
			wantWatchDirs:    map[string]bool{"tools_folder": true},
			wantWatchedFiles: map[string]bool{},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.description, func(t *testing.T) {
			gotWatchDirs, gotWatchedFiles := resolveWatcherInputs(tc.toolsFile, tc.toolsFiles, tc.toolsFolder)

			normalizedGotWatchDirs := normalizeFilepaths(gotWatchDirs)
			normalizedGotWatchedFiles := normalizeFilepaths(gotWatchedFiles)

			if diff := cmp.Diff(tc.wantWatchDirs, normalizedGotWatchDirs); diff != "" {
				t.Errorf("incorrect watchDirs: diff %v", diff)
			}
			if diff := cmp.Diff(tc.wantWatchedFiles, normalizedGotWatchedFiles); diff != "" {
				t.Errorf("incorrect watchedFiles: diff %v", diff)
			}

		})
	}
}

// helper function for testing file detection in dynamic reloading
func tmpFileWithCleanup(content []byte) (string, func(), error) {
	f, err := os.CreateTemp("", "*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.Remove(f.Name()) }

	if _, err := f.Write(content); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", nil, err
	}
	return f.Name(), cleanup, err
}

func TestSingleEdit(t *testing.T) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), time.Minute)
	defer cancelCtx()

	pr, pw := io.Pipe()
	defer pw.Close()
	defer pr.Close()

	fileToWatch, cleanup, err := tmpFileWithCleanup([]byte("initial content"))
	if err != nil {
		t.Fatalf("error editing tools file %s", err)
	}
	defer cleanup()

	logger, err := log.NewStdLogger(pw, pw, "DEBUG")
	if err != nil {
		t.Fatalf("failed to setup logger %s", err)
	}
	ctx = util.WithLogger(ctx, logger)

	instrumentation, err := telemetry.CreateTelemetryInstrumentation(versionString)
	if err != nil {
		t.Fatalf("failed to setup instrumentation %s", err)
	}
	ctx = util.WithInstrumentation(ctx, instrumentation)

	mockServer := &server.Server{}

	cleanFileToWatch := filepath.Clean(fileToWatch)
	watchDir := filepath.Dir(cleanFileToWatch)

	watchedFiles := map[string]bool{cleanFileToWatch: true}
	watchDirs := map[string]bool{watchDir: true}

	go watchChanges(ctx, watchDirs, watchedFiles, mockServer)

	// escape backslash so regex doesn't fail on windows filepaths
	regexEscapedPathFile := strings.ReplaceAll(cleanFileToWatch, `\`, `\\\\*\\`)
	regexEscapedPathFile = path.Clean(regexEscapedPathFile)

	regexEscapedPathDir := strings.ReplaceAll(watchDir, `\`, `\\\\*\\`)
	regexEscapedPathDir = path.Clean(regexEscapedPathDir)

	begunWatchingDir := regexp.MustCompile(fmt.Sprintf(`DEBUG "Added directory %s to watcher."`, regexEscapedPathDir))
	_, err = testutils.WaitForString(ctx, begunWatchingDir, pr)
	if err != nil {
		t.Fatalf("timeout or error waiting for watcher to start: %s", err)
	}

	err = os.WriteFile(fileToWatch, []byte("modification"), 0777)
	if err != nil {
		t.Fatalf("error writing to file: %v", err)
	}

	// only check substring of DEBUG message due to some OS/editors firing different operations
	detectedFileChange := regexp.MustCompile(fmt.Sprintf(`event detected in %s"`, regexEscapedPathFile))
	_, err = testutils.WaitForString(ctx, detectedFileChange, pr)
	if err != nil {
		t.Fatalf("timeout or error waiting for file to detect write: %s", err)
	}
}

func TestPrebuiltTools(t *testing.T) {
	// Get prebuilt configs
	alloydb_admin_config, _ := prebuiltconfigs.Get("alloydb-postgres-admin")
	alloydb_config, _ := prebuiltconfigs.Get("alloydb-postgres")
	bigquery_config, _ := prebuiltconfigs.Get("bigquery")
	clickhouse_config, _ := prebuiltconfigs.Get("clickhouse")
	cloudsqlpg_config, _ := prebuiltconfigs.Get("cloud-sql-postgres")
	cloudsqlmysql_config, _ := prebuiltconfigs.Get("cloud-sql-mysql")
	cloudsqlmssql_config, _ := prebuiltconfigs.Get("cloud-sql-mssql")
	dataplex_config, _ := prebuiltconfigs.Get("dataplex")
	firestoreconfig, _ := prebuiltconfigs.Get("firestore")
	mysql_config, _ := prebuiltconfigs.Get("mysql")
	mssql_config, _ := prebuiltconfigs.Get("mssql")
	looker_config, _ := prebuiltconfigs.Get("looker")
	postgresconfig, _ := prebuiltconfigs.Get("postgres")
	spanner_config, _ := prebuiltconfigs.Get("spanner")
	spannerpg_config, _ := prebuiltconfigs.Get("spanner-postgres")
	neo4jconfig, _ := prebuiltconfigs.Get("neo4j")

	// Set environment variables
	t.Setenv("API_KEY", "your_api_key")

	t.Setenv("BIGQUERY_PROJECT", "your_gcp_project_id")
	t.Setenv("DATAPLEX_PROJECT", "your_gcp_project_id")
	t.Setenv("FIRESTORE_PROJECT", "your_gcp_project_id")
	t.Setenv("FIRESTORE_DATABASE", "your_firestore_db_name")

	t.Setenv("SPANNER_PROJECT", "your_gcp_project_id")
	t.Setenv("SPANNER_INSTANCE", "your_spanner_instance")
	t.Setenv("SPANNER_DATABASE", "your_spanner_db")

	t.Setenv("ALLOYDB_POSTGRES_PROJECT", "your_gcp_project_id")
	t.Setenv("ALLOYDB_POSTGRES_REGION", "your_gcp_region")
	t.Setenv("ALLOYDB_POSTGRES_CLUSTER", "your_alloydb_cluster")
	t.Setenv("ALLOYDB_POSTGRES_INSTANCE", "your_alloydb_instance")
	t.Setenv("ALLOYDB_POSTGRES_DATABASE", "your_alloydb_db")
	t.Setenv("ALLOYDB_POSTGRES_USER", "your_alloydb_user")
	t.Setenv("ALLOYDB_POSTGRES_PASSWORD", "your_alloydb_password")

	t.Setenv("CLICKHOUSE_PROTOCOL", "your_clickhouse_protocol")
	t.Setenv("CLICKHOUSE_DATABASE", "your_clickhouse_database")
	t.Setenv("CLICKHOUSE_PASSWORD", "your_clickhouse_password")
	t.Setenv("CLICKHOUSE_USER", "your_clickhouse_user")
	t.Setenv("CLICKHOUSE_HOST", "your_clickhosue_host")
	t.Setenv("CLICKHOUSE_PORT", "8123")

	t.Setenv("CLOUD_SQL_POSTGRES_PROJECT", "your_pg_project")
	t.Setenv("CLOUD_SQL_POSTGRES_INSTANCE", "your_pg_instance")
	t.Setenv("CLOUD_SQL_POSTGRES_DATABASE", "your_pg_db")
	t.Setenv("CLOUD_SQL_POSTGRES_REGION", "your_pg_region")
	t.Setenv("CLOUD_SQL_POSTGRES_USER", "your_pg_user")
	t.Setenv("CLOUD_SQL_POSTGRES_PASS", "your_pg_pass")

	t.Setenv("CLOUD_SQL_MYSQL_PROJECT", "your_gcp_project_id")
	t.Setenv("CLOUD_SQL_MYSQL_REGION", "your_gcp_region")
	t.Setenv("CLOUD_SQL_MYSQL_INSTANCE", "your_instance")
	t.Setenv("CLOUD_SQL_MYSQL_DATABASE", "your_cloudsql_mysql_db")
	t.Setenv("CLOUD_SQL_MYSQL_USER", "your_cloudsql_mysql_user")
	t.Setenv("CLOUD_SQL_MYSQL_PASSWORD", "your_cloudsql_mysql_password")

	t.Setenv("CLOUD_SQL_MSSQL_PROJECT", "your_gcp_project_id")
	t.Setenv("CLOUD_SQL_MSSQL_REGION", "your_gcp_region")
	t.Setenv("CLOUD_SQL_MSSQL_INSTANCE", "your_cloudsql_mssql_instance")
	t.Setenv("CLOUD_SQL_MSSQL_DATABASE", "your_cloudsql_mssql_db")
	t.Setenv("CLOUD_SQL_MSSQL_IP_ADDRESS", "127.0.0.1")
	t.Setenv("CLOUD_SQL_MSSQL_USER", "your_cloudsql_mssql_user")
	t.Setenv("CLOUD_SQL_MSSQL_PASSWORD", "your_cloudsql_mssql_password")
	t.Setenv("CLOUD_SQL_POSTGRES_PASSWORD", "your_cloudsql_pg_password")

	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_DATABASE", "your_postgres_db")
	t.Setenv("POSTGRES_USER", "your_postgres_user")
	t.Setenv("POSTGRES_PASSWORD", "your_postgres_password")

	t.Setenv("MYSQL_HOST", "localhost")
	t.Setenv("MYSQL_PORT", "3306")
	t.Setenv("MYSQL_DATABASE", "your_mysql_db")
	t.Setenv("MYSQL_USER", "your_mysql_user")
	t.Setenv("MYSQL_PASSWORD", "your_mysql_password")

	t.Setenv("MSSQL_HOST", "localhost")
	t.Setenv("MSSQL_PORT", "1433")
	t.Setenv("MSSQL_DATABASE", "your_mssql_db")
	t.Setenv("MSSQL_USER", "your_mssql_user")
	t.Setenv("MSSQL_PASSWORD", "your_mssql_password")

	t.Setenv("LOOKER_BASE_URL", "https://your_company.looker.com")
	t.Setenv("LOOKER_CLIENT_ID", "your_looker_client_id")
	t.Setenv("LOOKER_CLIENT_SECRET", "your_looker_client_secret")
	t.Setenv("LOOKER_VERIFY_SSL", "true")

	t.Setenv("NEO4J_URI", "bolt://localhost:7687")
	t.Setenv("NEO4J_DATABASE", "neo4j")
	t.Setenv("NEO4J_USERNAME", "your_neo4j_user")
	t.Setenv("NEO4J_PASSWORD", "your_neo4j_password")

	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		name        string
		in          []byte
		wantToolset server.ToolsetConfigs
	}{
		{
			name: "alloydb postgres admin prebuilt tools",
			in:   alloydb_admin_config,
			wantToolset: server.ToolsetConfigs{
				"alloydb-postgres-admin-tools": tools.ToolsetConfig{
					Name:      "alloydb-postgres-admin-tools",
					ToolNames: []string{"alloydb-create-cluster", "alloydb-operations-get", "alloydb-create-instance", "alloydb-list-clusters", "alloydb-list-instances", "alloydb-list-users", "alloydb-create-user"},
				},
			},
		},
		{
			name: "alloydb prebuilt tools",
			in:   alloydb_config,
			wantToolset: server.ToolsetConfigs{
				"alloydb-postgres-database-tools": tools.ToolsetConfig{
					Name:      "alloydb-postgres-database-tools",
					ToolNames: []string{"execute_sql", "list_tables"},
				},
			},
		},
		{
			name: "bigquery prebuilt tools",
			in:   bigquery_config,
			wantToolset: server.ToolsetConfigs{
				"bigquery-database-tools": tools.ToolsetConfig{
					Name:      "bigquery-database-tools",
					ToolNames: []string{"analyze_contribution", "ask_data_insights", "execute_sql", "forecast", "get_dataset_info", "get_table_info", "list_dataset_ids", "list_table_ids"},
				},
			},
		},
		{
			name: "clickhouse prebuilt tools",
			in:   clickhouse_config,
			wantToolset: server.ToolsetConfigs{
				"clickhouse-database-tools": tools.ToolsetConfig{
					Name:      "clickhouse-database-tools",
					ToolNames: []string{"execute_sql", "list_databases"},
				},
			},
		},
		{
			name: "cloudsqlpg prebuilt tools",
			in:   cloudsqlpg_config,
			wantToolset: server.ToolsetConfigs{
				"cloud-sql-postgres-database-tools": tools.ToolsetConfig{
					Name:      "cloud-sql-postgres-database-tools",
					ToolNames: []string{"execute_sql", "list_tables"},
				},
			},
		},
		{
			name: "cloudsqlmysql prebuilt tools",
			in:   cloudsqlmysql_config,
			wantToolset: server.ToolsetConfigs{
				"cloud-sql-mysql-database-tools": tools.ToolsetConfig{
					Name:      "cloud-sql-mysql-database-tools",
					ToolNames: []string{"execute_sql", "list_tables"},
				},
			},
		},
		{
			name: "cloudsqlmssql prebuilt tools",
			in:   cloudsqlmssql_config,
			wantToolset: server.ToolsetConfigs{
				"cloud-sql-mssql-database-tools": tools.ToolsetConfig{
					Name:      "cloud-sql-mssql-database-tools",
					ToolNames: []string{"execute_sql", "list_tables"},
				},
			},
		},
		{
			name: "dataplex prebuilt tools",
			in:   dataplex_config,
			wantToolset: server.ToolsetConfigs{
				"dataplex-tools": tools.ToolsetConfig{
					Name:      "dataplex-tools",
					ToolNames: []string{"dataplex_search_entries", "dataplex_lookup_entry", "dataplex_search_aspect_types"},
				},
			},
		},
		{
			name: "firestore prebuilt tools",
			in:   firestoreconfig,
			wantToolset: server.ToolsetConfigs{
				"firestore-database-tools": tools.ToolsetConfig{
					Name:      "firestore-database-tools",
					ToolNames: []string{"firestore-get-documents", "firestore-add-documents", "firestore-update-document", "firestore-list-collections", "firestore-delete-documents", "firestore-query-collection", "firestore-get-rules", "firestore-validate-rules"},
				},
			},
		},
		{
			name: "mysql prebuilt tools",
			in:   mysql_config,
			wantToolset: server.ToolsetConfigs{
				"mysql-database-tools": tools.ToolsetConfig{
					Name:      "mysql-database-tools",
					ToolNames: []string{"execute_sql", "list_tables"},
				},
			},
		},
		{
			name: "mssql prebuilt tools",
			in:   mssql_config,
			wantToolset: server.ToolsetConfigs{
				"mssql-database-tools": tools.ToolsetConfig{
					Name:      "mssql-database-tools",
					ToolNames: []string{"execute_sql", "list_tables"},
				},
			},
		},
		{
			name: "looker prebuilt tools",
			in:   looker_config,
			wantToolset: server.ToolsetConfigs{
				"looker-tools": tools.ToolsetConfig{
					Name:      "looker-tools",
					ToolNames: []string{"get_models", "get_explores", "get_dimensions", "get_measures", "get_filters", "get_parameters", "query", "query_sql", "query_url", "get_looks", "run_look", "make_look", "get_dashboards", "make_dashboard", "add_dashboard_element"},
				},
			},
		},
		{
			name: "postgres prebuilt tools",
			in:   postgresconfig,
			wantToolset: server.ToolsetConfigs{
				"postgres-database-tools": tools.ToolsetConfig{
					Name:      "postgres-database-tools",
					ToolNames: []string{"execute_sql", "list_tables"},
				},
			},
		},
		{
			name: "spanner prebuilt tools",
			in:   spanner_config,
			wantToolset: server.ToolsetConfigs{
				"spanner-database-tools": tools.ToolsetConfig{
					Name:      "spanner-database-tools",
					ToolNames: []string{"execute_sql", "execute_sql_dql", "list_tables"},
				},
			},
		},
		{
			name: "spanner pg prebuilt tools",
			in:   spannerpg_config,
			wantToolset: server.ToolsetConfigs{
				"spanner-postgres-database-tools": tools.ToolsetConfig{
					Name:      "spanner-postgres-database-tools",
					ToolNames: []string{"execute_sql", "execute_sql_dql", "list_tables"},
				},
			},
		},
		{
			name: "neo4j prebuilt tools",
			in:   neo4jconfig,
			wantToolset: server.ToolsetConfigs{
				"neo4j-database-tools": tools.ToolsetConfig{
					Name:      "neo4j-database-tools",
					ToolNames: []string{"execute_cypher", "get_schema"},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			toolsFile, err := parseToolsFile(ctx, tc.in)
			if err != nil {
				t.Fatalf("failed to parse input: %v", err)
			}
			if diff := cmp.Diff(tc.wantToolset, toolsFile.Toolsets); diff != "" {
				t.Fatalf("incorrect tools parse: diff %v", diff)
			}
		})
	}
}

func TestUpdateLogLevel(t *testing.T) {
	tcs := []struct {
		desc     string
		stdio    bool
		logLevel string
		want     bool
	}{
		{
			desc:     "no stdio",
			stdio:    false,
			logLevel: "info",
			want:     false,
		},
		{
			desc:     "stdio with info log",
			stdio:    true,
			logLevel: "info",
			want:     true,
		},
		{
			desc:     "stdio with debug log",
			stdio:    true,
			logLevel: "debug",
			want:     true,
		},
		{
			desc:     "stdio with warn log",
			stdio:    true,
			logLevel: "warn",
			want:     false,
		},
		{
			desc:     "stdio with error log",
			stdio:    true,
			logLevel: "error",
			want:     false,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := updateLogLevel(tc.stdio, tc.logLevel)
			if got != tc.want {
				t.Fatalf("incorrect indication to update log level: got %t, want %t", got, tc.want)
			}
		})
	}
}
