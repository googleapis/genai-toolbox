// Copyright © 2025, Oracle and/or its affiliates.

package oracle_test

import (
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/oracle"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlOracle(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "connection string and useOCI=true",
			in: `
            sources:
                my-oracle-cs:
                    kind: oracle
                    connectionString: "my-host:1521/XEPDB1"
                    user: my_user
                    password: my_pass
                    useOCI: true
            `,
			want: server.SourceConfigs{
				"my-oracle-cs": oracle.Config{
					Name:             "my-oracle-cs",
					Kind:             oracle.SourceKind,
					ConnectionString: "my-host:1521/XEPDB1",
					User:             "my_user",
					Password:         "my_pass",
					UseOCI:           true,
				},
			},
		},
		{
			desc: "host/port/serviceName and default useOCI=false",
			in: `
            sources:
                my-oracle-host:
                    kind: oracle
                    host: my-host
                    port: 1521
                    serviceName: ORCLPDB
                    user: my_user
                    password: my_pass
            `,
			want: server.SourceConfigs{
				"my-oracle-host": oracle.Config{
					Name:        "my-oracle-host",
					Kind:        oracle.SourceKind,
					Host:        "my-host",
					Port:        1521,
					ServiceName: "ORCLPDB",
					User:        "my_user",
					Password:    "my_pass",
					UseOCI:      false,
				},
			},
		},
		{
			desc: "tnsAlias and TnsAdmin specified with explicit useOCI=true",
			in: `
            sources:
                my-oracle-tns-oci:
                    kind: oracle
                    tnsAlias: FINANCE_DB
                    tnsAdmin: /opt/oracle/network/admin
                    user: my_user
                    password: my_pass
                    useOCI: true 
            `,
			want: server.SourceConfigs{
				"my-oracle-tns-oci": oracle.Config{
					Name:     "my-oracle-tns-oci",
					Kind:     oracle.SourceKind,
					TnsAlias: "FINANCE_DB",
					TnsAdmin: "/opt/oracle/network/admin",
					User:     "my_user",
					Password: "my_pass",
					UseOCI:   true,
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}

			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if !cmp.Equal(tc.want, got.Sources) {
				t.Fatalf("incorrect parse:\nwant: %v\ngot:  %v\ndiff: %s", tc.want, got.Sources, cmp.Diff(tc.want, got.Sources))
			}
		})
	}
}

func TestFailParseFromYamlOracle(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "extra field",
			in: `
			sources:
				my-oracle-instance:
					kind: oracle
					host: my-host
					serviceName: ORCL
					user: my_user
					password: my_pass
					extraField: value
			`,
			err: "unable to parse source \"my-oracle-instance\" as \"oracle\": [1:1] unknown field \"extraField\"\n>  1 | extraField: value\n       ^\n   2 | host: my-host\n   3 | kind: oracle\n   4 | password: my_pass\n   5 | ",
		},
		{
			desc: "missing required password field",
			in: `
            sources:
                my-oracle-instance:
                    kind: oracle
                    host: my-host
                    serviceName: ORCL
                    user: my_user
            `,
			err: "unable to parse source \"my-oracle-instance\" as \"oracle\": Key: 'Config.Password' Error:Field validation for 'Password' failed on the 'required' tag",
		},
		{
			desc: "missing connection method fields (validate fails)",
			in: `
            sources:
                my-oracle-instance:
                    kind: oracle
                    user: my_user
                    password: my_pass
            `,
			err: "unable to parse source \"my-oracle-instance\" as \"oracle\": invalid Oracle configuration: must provide one of: 'tns_alias', 'connection_string', or both 'host' and 'service_name'",
		},
		{
			desc: "multiple connection methods provided (validate fails)",
			in: `
            sources:
                my-oracle-instance:
                    kind: oracle
                    host: my-host
                    serviceName: ORCL
                    connectionString: "my-host:1521/XEPDB1"
                    user: my_user
                    password: my_pass
            `,
			err: "unable to parse source \"my-oracle-instance\" as \"oracle\": invalid Oracle configuration: provide only one connection method: 'tns_alias', 'connection_string', or 'host'+'service_name'",
		},
		{
			desc: "fail on tnsAdmin with useOCI=false",
			in: `
			sources:
				my-oracle-fail:
					kind: oracle
					tnsAlias: FINANCE_DB
					tnsAdmin: /opt/oracle/network/admin
					user: my_user
					password: my_pass
					useOCI: false
			`,
			err: "unable to parse source \"my-oracle-fail\" as \"oracle\": invalid Oracle configuration: `tnsAdmin` can only be used when `UseOCI` is true, or use `walletLocation` instead",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}

			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			errStr := strings.ReplaceAll(err.Error(), "\r", "")

			if errStr != tc.err {
				t.Fatalf("unexpected error:\ngot:\n%q\nwant:\n%q\n", errStr, tc.err)
			}
		})
	}
}
