package yugabytedb_test

import (
	"testing"

	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/yugabytedb"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

// Basic config parse
func TestParseFromYamlYugabyteDB(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "only required fields",
			in: `
			sources:
				my-yb-instance:
					kind: yugabytedb
					name: my-yb-instance
					host: yb-host
					port: yb-port
					user: yb_user
					password: yb_pass
					database: yb_db
			`,
			want: server.SourceConfigs{
				"my-yb-instance": yugabytedb.Config{
					Name:     "my-yb-instance",
					Kind:     "yugabytedb",
					Host:     "yb-host",
					Port:     "yb-port",
					User:     "yb_user",
					Password: "yb_pass",
					Database: "yb_db",
				},
			},
		},
		{
			desc: "with loadBalance only",
			in: `
			sources:
				my-yb-instance:
					kind: yugabytedb
					name: my-yb-instance
					host: yb-host
					port: yb-port
					user: yb_user
					password: yb_pass
					database: yb_db
					loadBalance: true
			`,
			want: server.SourceConfigs{
				"my-yb-instance": yugabytedb.Config{
					Name:        "my-yb-instance",
					Kind:        "yugabytedb",
					Host:        "yb-host",
					Port:        "yb-port",
					User:        "yb_user",
					Password:    "yb_pass",
					Database:    "yb_db",
					LoadBalance: "true",
				},
			},
		},
		{
			desc: "loadBalance with topologyKeys",
			in: `
			sources:
				my-yb-instance:
					kind: yugabytedb
					name: my-yb-instance
					host: yb-host
					port: yb-port
					user: yb_user
					password: yb_pass
					database: yb_db
					loadBalance: true
					topologyKeys: zone1,zone2
			`,
			want: server.SourceConfigs{
				"my-yb-instance": yugabytedb.Config{
					Name:         "my-yb-instance",
					Kind:         "yugabytedb",
					Host:         "yb-host",
					Port:         "yb-port",
					User:         "yb_user",
					Password:     "yb_pass",
					Database:     "yb_db",
					LoadBalance:  "true",
					TopologyKeys: "zone1,zone2",
				},
			},
		},
		{
			desc: "with fallback only",
			in: `
			sources:
				my-yb-instance:
					kind: yugabytedb
					name: my-yb-instance
					host: yb-host
					port: yb-port
					user: yb_user
					password: yb_pass
					database: yb_db
					loadBalance: true
					topologyKeys: zone1
					fallbackToTopologyKeysOnly: true
			`,
			want: server.SourceConfigs{
				"my-yb-instance": yugabytedb.Config{
					Name:                       "my-yb-instance",
					Kind:                       "yugabytedb",
					Host:                       "yb-host",
					Port:                       "yb-port",
					User:                       "yb_user",
					Password:                   "yb_pass",
					Database:                   "yb_db",
					LoadBalance:                "true",
					TopologyKeys:               "zone1",
					FallBackToTopologyKeysOnly: "true",
				},
			},
		},
		{
			desc: "with refresh interval and reconnect delay",
			in: `
			sources:
				my-yb-instance:
					kind: yugabytedb
					name: my-yb-instance
					host: yb-host
					port: yb-port
					user: yb_user
					password: yb_pass
					database: yb_db
					loadBalance: true
					ybServersRefreshInterval: 20
					failedHostReconnectDelaySecs: 5
			`,
			want: server.SourceConfigs{
				"my-yb-instance": yugabytedb.Config{
					Name:                            "my-yb-instance",
					Kind:                            "yugabytedb",
					Host:                            "yb-host",
					Port:                            "yb-port",
					User:                            "yb_user",
					Password:                        "yb_pass",
					Database:                        "yb_db",
					LoadBalance:                     "true",
					YBServersRefreshInterval:        "20",
					FailedHostReconnectDelaySeconds: "5",
				},
			},
		},
		{
			desc: "all fields set",
			in: `
			sources:
				my-yb-instance:
					kind: yugabytedb
					name: my-yb-instance
					host: yb-host
					port: yb-port
					user: yb_user
					password: yb_pass
					database: yb_db
					loadBalance: true
					topologyKeys: zone1,zone2
					fallbackToTopologyKeysOnly: true
					ybServersRefreshInterval: 30
					failedHostReconnectDelaySecs: 10
			`,
			want: server.SourceConfigs{
				"my-yb-instance": yugabytedb.Config{
					Name:                            "my-yb-instance",
					Kind:                            "yugabytedb",
					Host:                            "yb-host",
					Port:                            "yb-port",
					User:                            "yb_user",
					Password:                        "yb_pass",
					Database:                        "yb_db",
					LoadBalance:                     "true",
					TopologyKeys:                    "zone1,zone2",
					FallBackToTopologyKeysOnly:      "true",
					YBServersRefreshInterval:        "30",
					FailedHostReconnectDelaySeconds: "10",
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
				t.Fatalf("incorrect parse (-want +got):\n%s", cmp.Diff(tc.want, got.Sources))
			}
		})
	}
}

func TestFailParseFromYamlYugabyteDB(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "extra field",
			in: `
			sources:
				my-yb-source:
					kind: yugabytedb
					name: my-yb-source
					host: yb-host
					port: yb-port
					database: yb_db
					user: yb_user
					password: yb_pass
					foo: bar
			`,
			err: "unable to parse source \"my-yb-source\" as \"yugabytedb\": [2:1] unknown field \"foo\"",
		},
		{
			desc: "missing required field (password)",
			in: `
			sources:
				my-yb-source:
					kind: yugabytedb
					name: my-yb-source
					host: yb-host
					port: yb-port
					database: yb_db
					user: yb_user
			`,
			err: "unable to parse source \"my-yb-source\" as \"yugabytedb\": Key: 'Config.Password' Error:Field validation for 'Password' failed on the 'required' tag",
		},
		{
			desc: "missing required field (host)",
			in: `
			sources:
				my-yb-source:
					kind: yugabytedb
					name: my-yb-source
					port: yb-port
					database: yb_db
					user: yb_user
					password: yb_pass
			`,
			err: "unable to parse source \"my-yb-source\" as \"yugabytedb\": Key: 'Config.Host' Error:Field validation for 'Host' failed on the 'required' tag",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err == nil {
				t.Fatalf("expected parsing to fail")
			}
			errStr := err.Error()
			if !strings.Contains(errStr, tc.err) {
				t.Fatalf("unexpected error:\nGot:  %q\nWant: %q", errStr, tc.err)
			}
		})
	}
}
