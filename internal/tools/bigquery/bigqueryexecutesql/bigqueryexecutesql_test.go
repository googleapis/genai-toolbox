// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package bigqueryexecutesql

import (
	"context"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
)

func TestParseFromYaml(t *testing.T) {
	t.Parallel()
	const (
		basicYAML = `
name: bigquery-execute-sql-tool
kind: bigquery-execute-sql
source: bq
description: test
authRequired:
  - gcp
`
	)
	tests := []struct {
		name    string
		input   string
		isError bool
		want    *Config
	}{
		{
			name:  "basic example",
			input: basicYAML,
			want: &Config{
				Name:         "bigquery-execute-sql-tool",
				Kind:         "bigquery-execute-sql",
				Source:       "bq",
				Description:  "test",
				AuthRequired: []string{"gcp"},
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got Config
			err := yaml.Unmarshal([]byte(tc.input), &got)
			if tc.isError {
				if err == nil {
					t.Errorf("yaml.Unmarshal got nil error, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("yaml.Unmarshal got unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, &got); diff != "" {
				t.Errorf("yaml.Unmarshal() returned diff (-want +got):\n%s", diff)
			}
		})
	}
}

type mockQuery struct {
	labels map[string]string
}

func (q *mockQuery) Run(ctx context.Context) (*bigquery.Job, error) {
	return nil, nil
}

func (q *mockQuery) Read(ctx context.Context) (*bigquery.RowIterator, error) {
	return nil, nil
}

type mockBigQueryClient struct {
	*bigquery.Client
	t         *testing.T
	projectID string
	location  string
}

func (c *mockBigQueryClient) Query(sql string) *bigquery.Query {
	q := &bigquery.Query{}
	return q
}

func (c *mockBigQueryClient) Close() error {
	return nil
}

func (c *mockBigQueryClient) Project() string {
	return c.projectID
}

func (c *mockBigQueryClient) Location() string {
	return c.location
}
