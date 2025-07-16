package kuzudb_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/kuzudb"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlKuzudb(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
            sources:
                my-kuzu-db:
                    kind: kuzudb
                    database: /path/to/database.db
            `,
			want: server.SourceConfigs{
				"my-kuzu-db": kuzudb.KuzuDbConfig{
					Name:     "my-kuzu-db",
					Kind:     kuzudb.KuzuDbKind,
					Database: "/path/to/database.db",
				},
			},
		},
		{
			desc: "with configuration",
			in: `
			sources:
				my-kuzu-db:
					kind: kuzudb
					database: /path/to/database.db
					maxNumThreads: 10
					readOnly: true
            `,
			want: server.SourceConfigs{
				"my-kuzu-db": kuzudb.KuzuDbConfig{
					Name:              "my-kuzu-db",
					Kind:              kuzudb.KuzuDbKind,
					Database:          "/path/to/database.db",
					MaxNumThreads:     10,
					ReadOnly:          true,
					MaxDbSize:         0,
					BufferPoolSize:    0,
					EnableCompression: false,
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
