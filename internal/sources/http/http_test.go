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
	"github.com/googleapis/mcp-toolbox/internal/sources/http"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/util"
)

func TestParseFromYamlHttp(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			kind: source
			name: my-http-instance
			type: http
			baseUrl: http://test_server/
			`,
			want: map[string]sources.SourceConfig{
				"my-http-instance": http.Config{
					Name:                   "my-http-instance",
					Type:                   http.SourceType,
					BaseURL:                "http://test_server/",
					Timeout:                "30s",
					DisableSslVerification: false,
				},
			},
		},
		{
			desc: "advanced example",
			in: `
			kind: source
			name: my-http-instance
			type: http
			baseUrl: http://test_server/
			timeout: 10s
			headers:
				Authorization: test_header
				Custom-Header: custom
			queryParams:
				api-key: test_api_key
				param: param-value
			returnFullError: true
			disableSslVerification: true
			`,
			want: map[string]sources.SourceConfig{
				"my-http-instance": http.Config{
					Name:                   "my-http-instance",
					Type:                   http.SourceType,
					BaseURL:                "http://test_server/",
					Timeout:                "10s",
					DefaultHeaders:         map[string]string{"Authorization": "test_header", "Custom-Header": "custom"},
					QueryParams:            map[string]string{"api-key": "test_api_key", "param": "param-value"},
					ReturnFullError:        true,
					DisableSslVerification: true,
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got, _, _, _, _, _, err := server.UnmarshalResourceConfig(context.Background(), testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if !cmp.Equal(tc.want, got) {
				t.Fatalf("incorrect parse: want %v, got %v", tc.want, got)
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
			kind: source
			name: my-http-instance
			type: http
			baseUrl: http://test_server/
			timeout: 10s
			headers:
				Authorization: test_header
			queryParams:
				api-key: test_api_key
			project: test-project
			`,
			err: "error unmarshaling source: unable to parse source \"my-http-instance\" as \"http\": [5:1] unknown field \"project\"\n   2 | headers:\n   3 |   Authorization: test_header\n   4 | name: my-http-instance\n>  5 | project: test-project\n       ^\n   6 | queryParams:\n   7 |   api-key: test_api_key\n   8 | timeout: 10s\n   9 | ",
		},
		{
			desc: "missing required field",
			in: `
			kind: source
			name: my-http-instance
			baseUrl: http://test_server/
			`,
			err: "error unmarshaling source: missing 'type' field or it is not a string",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, _, _, _, err := server.UnmarshalResourceConfig(context.Background(), testutils.FormatYaml(tc.in))
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

func TestRunRequestSanitizesErrorBodyByDefault(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.WriteHeader(nethttp.StatusBadRequest)
		_, _ = w.Write([]byte("sensitive details"))
	}))
	defer server.Close()

	logger, err := log.NewLogger("standard", log.Debug, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	ctx := util.WithLogger(context.Background(), logger)

	sourceConfig := http.Config{
		Name:    "test-http",
		Type:    http.SourceType,
		BaseURL: server.URL,
		Timeout: "30s",
	}
	initialized, err := sourceConfig.Initialize(ctx, nil)
	if err != nil {
		t.Fatalf("failed to initialize source: %v", err)
	}
	source := initialized.(*http.Source)

	req, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	_, err = source.RunRequest(ctx, req)
	if err == nil {
		t.Fatalf("expected error for non-2xx response")
	}
	if strings.Contains(err.Error(), "sensitive details") {
		t.Fatalf("expected sanitized error message, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "unexpected status code: 400") {
		t.Fatalf("expected status code in error message, got %q", err.Error())
	}
}

func TestRunRequestIncludesErrorBodyWhenEnabled(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.WriteHeader(nethttp.StatusInternalServerError)
		_, _ = w.Write([]byte("sensitive details"))
	}))
	defer server.Close()

	logger, err := log.NewLogger("standard", log.Debug, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	ctx := util.WithLogger(context.Background(), logger)

	sourceConfig := http.Config{
		Name:            "test-http",
		Type:            http.SourceType,
		BaseURL:         server.URL,
		Timeout:         "30s",
		ReturnFullError: true,
	}
	initialized, err := sourceConfig.Initialize(ctx, nil)
	if err != nil {
		t.Fatalf("failed to initialize source: %v", err)
	}
	source := initialized.(*http.Source)

	req, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	_, err = source.RunRequest(ctx, req)
	if err == nil {
		t.Fatalf("expected error for non-2xx response")
	}
	if !strings.Contains(err.Error(), "response body: sensitive details") {
		t.Fatalf("expected response body in error message, got %q", err.Error())
	}
}
