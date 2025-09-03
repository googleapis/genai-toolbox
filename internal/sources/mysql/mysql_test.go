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

package mysql_test

import (
	"context"
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/mysql"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlCloudSQLMySQL(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-mysql-instance:
					kind: mysql
					host: 0.0.0.0
					port: my-port
					database: my_db
					user: my_user
					password: my_pass
			`,
			want: server.SourceConfigs{
				"my-mysql-instance": mysql.Config{
					Name:     "my-mysql-instance",
					Kind:     mysql.SourceKind,
					Host:     "0.0.0.0",
					Port:     "my-port",
					Database: "my_db",
					User:     "my_user",
					Password: "my_pass",
				},
			},
		},
		{
			desc: "with query timeout",
			in: `
			sources:
				my-mysql-instance:
					kind: mysql
					host: 0.0.0.0
					port: my-port
					database: my_db
					user: my_user
					password: my_pass
					queryTimeout: 45s
			`,
			want: server.SourceConfigs{
				"my-mysql-instance": mysql.Config{
					Name:         "my-mysql-instance",
					Kind:         mysql.SourceKind,
					Host:         "0.0.0.0",
					Port:         "my-port",
					Database:     "my_db",
					User:         "my_user",
					Password:     "my_pass",
					QueryTimeout: "45s",
				},
			},
		},
		{
			desc: "with query params",
			in: `
			sources:
				my-mysql-instance:
					kind: mysql
					host: 0.0.0.0
					port: my-port
					database: my_db
					user: my_user
					password: my_pass
					queryParams:
						tls: preferred
						charset: utf8mb4
			`,
			want: server.SourceConfigs{
				"my-mysql-instance": mysql.Config{
					Name:     "my-mysql-instance",
					Kind:     mysql.SourceKind,
					Host:     "0.0.0.0",
					Port:     "my-port",
					Database: "my_db",
					User:     "my_user",
					Password: "my_pass",
					QueryParams: map[string]string{
						"tls":     "preferred",
						"charset": "utf8mb4",
					},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got.Sources, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("mismatch (-want +got):\n%s", diff)
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
				my-mysql-instance:
					kind: mysql
					host: 0.0.0.0
					port: my-port
					database: my_db
					user: my_user
					password: my_pass
					foo: bar
			`,
			err: "unknown field \"foo\"",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-mysql-instance:
					kind: mysql
					port: my-port
					database: my_db
					user: my_user
					password: my_pass
			`,
			err: "Field validation for 'Host' failed",
		},
		{
			desc: "invalid query params type",
			in: `
			sources:
				my-mysql-instance:
					kind: mysql
					host: 0.0.0.0
					port: 3306
					database: my_db
					user: my_user
					password: my_pass
					queryParams: not-a-map
			`,
			err: "string was used where mapping is expected",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			errStr := err.Error()
			if !strings.Contains(errStr, tc.err) {
				t.Fatalf("unexpected error: got %q, want substring %q", errStr, tc.err)
			}
		})
	}
}

// TestFailInitialization test error during initialization without attempting a DB connection.
func TestFailInitialization(t *testing.T) {
	t.Parallel()

	cfg := mysql.Config{
		Name:         "instance",
		Kind:         "mysql",
		Host:         "localhost",
		Port:         "3306",
		Database:     "db",
		User:         "user",
		Password:     "pass",
		QueryTimeout: "abc", // invalid duration
	}
	_, err := cfg.Initialize(context.Background(), noop.NewTracerProvider().Tracer("test"))
	if err == nil {
		t.Fatalf("expected error for invalid queryTimeout, got nil")
	}
	if !strings.Contains(err.Error(), "invalid queryTimeout") {
		t.Fatalf("unexpected error: %v", err)
	}
}
