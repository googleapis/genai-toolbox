//go:build integration && bigtable

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

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/bigtable"
	"github.com/google/uuid"
)

var (
	BIGTABLE_SOURCE_KIND = "bigtable"
	BIGTABLE_TOOL_KIND   = "bigtable-sql"
	BIGTABLE_PROJECT     = os.Getenv("BIGTABLE_PROJECT")
	BIGTABLE_INSTANCE    = os.Getenv("BIGTABLE_INSTANCE")
)

func getBigtableVars(t *testing.T) map[string]string {
	switch "" {
	case BIGTABLE_PROJECT:
		t.Fatal("'BIGTABLE_PROJECT' not set")
	case BIGTABLE_INSTANCE:
		t.Fatal("'BIGTABLE_INSTANCE' not set")
	}

	return map[string]string{
		"kind":     BIGTABLE_SOURCE_KIND,
		"project":  BIGTABLE_PROJECT,
		"instance": BIGTABLE_INSTANCE,
	}
}

type TestRow struct {
	RowKey     string
	ColumnName string
	Data       []byte
}

func TestBigtableToolEndpoints(t *testing.T) {
	sourceConfig := getBigtableVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	tableName := "bt_" + strings.Replace(uuid.New().String(), "-", "", -1)
	columnFamilyName, muts, rowKeys := getTestData()
	teardownTable1 := SetupBtTable(t, ctx, sourceConfig["project"], sourceConfig["instance"], tableName, columnFamilyName, muts, rowKeys)
	defer teardownTable1(t)

	// Write config into a file and pass it to command
	toolsFile := getToolsConfig(sourceConfig, tableName, "state")
	cmd, cleanup, err := StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := cmd.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`))
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	runBtToolGetTest(t)

	runBtInvokeTest(t)
}

func runBtToolGetTest(t *testing.T) {

	// Test tool get endpoint
	tcs := []struct {
		name string
		api  string
		want map[string]any
	}{
		{
			name: "get my-simple-tool",
			api:  "http://127.0.0.1:5000/api/tool/my-simple-tool/",
			want: map[string]any{
				"my-simple-tool": map[string]any{
					"description": "Simple tool to test end to end functionality.",
					"parameters": []any{
						map[string]any{
							"name":        "state",
							"type":        "string",
							"description": "state filter",
							"authSources": []any{}},
					},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(tc.api)
			if err != nil {
				t.Fatalf("error when sending a request: %s", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatalf("response status code is not 200")
			}

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["tools"]
			if !ok {
				t.Fatalf("unable to find tools in response body")
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func runBtInvokeTest(t *testing.T) {

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "provided parameters were invalid: parameter \"state\" is required",
			api:           "http://127.0.0.1:5000/api/tool/my-simple-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-simple-tool with filter",
			api:           "http://127.0.0.1:5000/api/tool/my-simple-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"state": "CA"}`)),
			want:          "[{state:CA}]",
			isErr:         false,
		},
		{
			name:          "invoke my-simple-tool - empty result",
			api:           "http://127.0.0.1:5000/api/tool/my-simple-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"state": "NY"}`)),
			want:          "null",
			isErr:         false,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr == true {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Check response body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}
			// Remove `\` and `"` for string comparison
			got = strings.ReplaceAll(got, "\\", "")
			want := strings.ReplaceAll(tc.want, "\\", "")
			got = strings.ReplaceAll(got, "\"", "")
			want = strings.ReplaceAll(want, "\"", "")
			if got != want {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}

func getToolsConfig(sourceConfig map[string]string, tableName string, filterField string) map[string]any {
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-bigtable-instance": sourceConfig,
		},
		"tools": map[string]any{
			"my-simple-tool": map[string]any{
				"kind":        BIGTABLE_TOOL_KIND,
				"source":      "my-bigtable-instance",
				"description": "Simple tool to test end to end functionality.",
				"statement":   fmt.Sprintf("SELECT address['%s'] as %s from `%s` WHERE address['%s'] = @%s;", filterField, filterField, tableName, filterField, filterField),
				"parameters": []map[string]any{
					{
						"name":        filterField,
						"type":        "string",
						"description": fmt.Sprintf("%s filter", filterField),
					},
				},
			},
		},
	}
	return toolsFile
}

func getTestData() (string, []*bigtable.Mutation, []string) {
	columnFamilyName := "address"
	muts := []*bigtable.Mutation{}
	rowKeys := []string{}
	v1Timestamp := bigtable.Time(time.Now().Add(1 * time.Minute))
	v2Timestamp := bigtable.Time(time.Now())

	type cell struct {
		Ts    bigtable.Timestamp // Using bigtable.Timestamp as an example
		Value []byte
	}

	for rowKey, mutData := range map[string]map[string]any{
		"row-01": {
			"state": []cell{
				{
					Value: []byte("WA"),
				},
				{
					Ts:    v2Timestamp,
					Value: []byte("CA"),
				},
			},
			"city": []cell{
				{
					Ts:    v2Timestamp,
					Value: []byte("San Francisco"),
				},
			},
		},
		"row-02": {
			"state": []cell{
				{
					Ts:    v1Timestamp,
					Value: []byte("AZ"),
				},
			},
			"city": []cell{
				{
					Ts:    v1Timestamp,
					Value: []byte("Phoenix"),
				},
			},
		},
	} {
		mut := bigtable.NewMutation()
		for col, v := range mutData {
			cells, ok := v.([]cell)
			if ok {
				for _, cell := range cells {
					mut.Set(columnFamilyName, col, cell.Ts, cell.Value)
				}
			}
		}
		muts = append(muts, mut)
		rowKeys = append(rowKeys, rowKey)
	}
	return columnFamilyName, muts, rowKeys
}

func SetupBtTable(t *testing.T, ctx context.Context, projectId string, instance string, tableName string, columnFamilyName string, muts []*bigtable.Mutation, rowKeys []string) func(*testing.T) {
	// Creating clients
	adminClient, err := bigtable.NewAdminClient(ctx, projectId, instance)
	if err != nil {
		t.Fatalf("NewAdminClient: %v", err)
	}

	client, err := bigtable.NewClient(ctx, projectId, instance)
	if err != nil {
		log.Fatalf("Could not create data operations client: %v", err)
	}
	defer client.Close()

	// Creating tables
	tables, err := adminClient.Tables(ctx)
	if err != nil {
		log.Fatalf("Could not fetch table list: %v", err)
	}

	if !slices.Contains(tables, tableName) {
		log.Printf("Creating table %s", tableName)
		if err := adminClient.CreateTable(ctx, tableName); err != nil {
			log.Fatalf("Could not create table %s: %v", tableName, err)
		}
	}

	tblInfo, err := adminClient.TableInfo(ctx, tableName)
	if err != nil {
		log.Fatalf("Could not read info for table %s: %v", tableName, err)
	}

	// Creating column family
	if !slices.Contains(tblInfo.Families, columnFamilyName) {
		if err := adminClient.CreateColumnFamilyWithConfig(ctx, tableName, columnFamilyName, bigtable.Family{ValueType: bigtable.StringType{}}); err != nil {

			log.Fatalf("Could not create column family %s: %v", columnFamilyName, err)
		}
	}

	tbl := client.Open(tableName)
	rowErrs, err := tbl.ApplyBulk(ctx, rowKeys, muts)
	if err != nil {
		log.Fatalf("Could not apply bulk row mutation: %v", err)
	}
	if rowErrs != nil {
		for _, rowErr := range rowErrs {
			log.Printf("Error writing row: %v", rowErr)
		}
		log.Fatalf("Could not write some rows")
	}

	// Writing data
	return func(t *testing.T) {
		// tear down test
		if err = adminClient.DeleteTable(ctx, tableName); err != nil {
			log.Fatalf("Teardown failed. Could not delete table %s: %v", tableName, err)
		}
		defer adminClient.Close()
	}
}
