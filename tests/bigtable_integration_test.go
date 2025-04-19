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
	"encoding/binary"
	"fmt"
	"log"
	"os"
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

func getBigtableVars(t *testing.T) map[string]any {
	switch "" {
	case BIGTABLE_PROJECT:
		t.Fatal("'BIGTABLE_PROJECT' not set")
	case BIGTABLE_INSTANCE:
		t.Fatal("'BIGTABLE_INSTANCE' not set")
	}

	return map[string]any{
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

	tableName := "param_table" + strings.Replace(uuid.New().String(), "-", "", -1)
	tableNameAuth := "auth_table_" + strings.Replace(uuid.New().String(), "-", "", -1)

	columnFamilyName := "cf"
	muts, rowKeys := getTestData(columnFamilyName)

	// Do not change the shape of statement without checking tests/common_test.go.
	// The structure and value of seed data has to match https://github.com/googleapis/genai-toolbox/blob/4dba0df12dc438eca3cb476ef52aa17cdf232c12/tests/common_test.go#L200-L251
	param_test_statement := fmt.Sprintf("SELECT TO_INT64(cf['id']) as id, CAST(cf['name'] AS string) as name, FROM %s WHERE TO_INT64(cf['id']) = @id OR CAST(cf['name'] AS string) = @name;", tableName)
	teardownTable1 := SetupBtTable(t, ctx, sourceConfig["project"].(string), sourceConfig["instance"].(string), tableName, columnFamilyName, muts, rowKeys)
	defer teardownTable1(t)

	// Do not change the shape of statement without checking tests/common_test.go.
	// The structure and value of seed data has to match https://github.com/googleapis/genai-toolbox/blob/4dba0df12dc438eca3cb476ef52aa17cdf232c12/tests/common_test.go#L200-L251
	auth_tool_statement := fmt.Sprintf("SELECT CAST(cf['name'] AS string) as name FROM %s WHERE CAST(cf['email'] AS string) = @email;", tableNameAuth)
	teardownTable2 := SetupBtTable(t, ctx, sourceConfig["project"].(string), sourceConfig["instance"].(string), tableNameAuth, columnFamilyName, muts, rowKeys)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := GetToolsConfig(sourceConfig, BIGTABLE_TOOL_KIND, param_test_statement, auth_tool_statement)
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

	RunToolGetTest(t)

	// Actual test parameters are set in https://github.com/googleapis/genai-toolbox/blob/52b09a67cb40ac0c5f461598b4673136699a3089/tests/tool_test.go#L250
	select_1_want := "[{$col1:1}]"
	RunToolInvokeTest(t, select_1_want)
}

func getTestData(columnFamilyName string) ([]*bigtable.Mutation, []string) {
	muts := []*bigtable.Mutation{}
	rowKeys := []string{}

	var ids [3][]byte
	for i := range ids {
		binary1 := new(bytes.Buffer)
		if err := binary.Write(binary1, binary.BigEndian, int64(i+1)); err != nil {
			log.Fatalf("Unable to encode id: %v", err)
		}
		ids[i] = binary1.Bytes()
	}

	now := bigtable.Time(time.Now())
	for rowKey, mutData := range map[string]map[string][]byte{
		// Do not change the test data without checking tests/common_test.go.
		// The structure and value of seed data has to match https://github.com/googleapis/genai-toolbox/blob/4dba0df12dc438eca3cb476ef52aa17cdf232c12/tests/common_test.go#L200-L251
		// Expected values are defined in https://github.com/googleapis/genai-toolbox/blob/52b09a67cb40ac0c5f461598b4673136699a3089/tests/tool_test.go#L229-L310
		"row-01": {
			"name":  []byte("Alice"),
			"email": []byte(SERVICE_ACCOUNT_EMAIL),
			"id":    ids[0],
		},
		"row-02": {
			"name":  []byte("Jane"),
			"email": []byte("janedoe@gmail.com"),
			"id":    ids[1],
		},
		"row-03": {
			"name": []byte("Sid"),
			"id":   ids[2],
		},
	} {
		mut := bigtable.NewMutation()
		for col, v := range mutData {
			mut.Set(columnFamilyName, col, now, v)
		}
		muts = append(muts, mut)
		rowKeys = append(rowKeys, rowKey)
	}
	return muts, rowKeys
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
		if err := adminClient.CreateColumnFamily(ctx, tableName, columnFamilyName); err != nil {
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
