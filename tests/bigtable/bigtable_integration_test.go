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

package bigtable

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
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	BigtableSourceKind = "bigtable"
	BigtableToolKind   = "bigtable-sql"
	BigtableProject    = os.Getenv("BIGTABLE_PROJECT")
	BigtableInstance   = os.Getenv("BIGTABLE_INSTANCE")
)

func getBigtableVars(t *testing.T) map[string]any {
	switch "" {
	case BigtableProject:
		t.Fatal("'BIGTABLE_PROJECT' not set")
	case BigtableInstance:
		t.Fatal("'BIGTABLE_INSTANCE' not set")
	}

	return map[string]any{
		"kind":     BigtableSourceKind,
		"project":  BigtableProject,
		"instance": BigtableInstance,
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

	tableName := "param_table" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "tmpl_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	columnFamilyName := "cf"
	muts, rowKeys := getTestData(columnFamilyName)

	// Do not change the shape of statement without checking tests/common_test.go.
	// The structure and value of seed data has to match https://github.com/googleapis/genai-toolbox/blob/4dba0df12dc438eca3cb476ef52aa17cdf232c12/tests/common_test.go#L200-L251
	paramTestStatement := fmt.Sprintf("SELECT TO_INT64(cf['id']) as id, CAST(cf['name'] AS string) as name, FROM %s WHERE TO_INT64(cf['id']) = @id OR CAST(cf['name'] AS string) = @name;", tableName)
	idParamTestStatement := fmt.Sprintf("SELECT TO_INT64(cf['id']) as id, CAST(cf['name'] AS string) as name, FROM %s WHERE TO_INT64(cf['id']) = @id;", tableName)
	nameParamTestStatement := fmt.Sprintf("SELECT TO_INT64(cf['id']) as id, CAST(cf['name'] AS string) as name, FROM %s WHERE CAST(cf['name'] AS string) = @name;", tableName)
	arrayTestStatement := fmt.Sprintf(
		"SELECT TO_INT64(cf['id']) AS id, CAST(cf['name'] AS string) AS name FROM %s WHERE TO_INT64(cf['id']) IN UNNEST(@idArray) AND CAST(cf['name'] AS string) IN UNNEST(@nameArray);",
		tableName,
	)
	teardownTable1 := setupBtTable(t, ctx, sourceConfig["project"].(string), sourceConfig["instance"].(string), tableName, columnFamilyName, muts, rowKeys)
	defer teardownTable1(t)

	// Do not change the shape of statement without checking tests/common_test.go.
	// The structure and value of seed data has to match https://github.com/googleapis/genai-toolbox/blob/4dba0df12dc438eca3cb476ef52aa17cdf232c12/tests/common_test.go#L200-L251
	authToolStatement := fmt.Sprintf("SELECT CAST(cf['name'] AS string) as name FROM %s WHERE CAST(cf['email'] AS string) = @email;", tableNameAuth)
	teardownTable2 := setupBtTable(t, ctx, sourceConfig["project"].(string), sourceConfig["instance"].(string), tableNameAuth, columnFamilyName, muts, rowKeys)
	defer teardownTable2(t)

	mutsTmpl, rowKeysTmpl := getTestDataTemplateParam(columnFamilyName)
	teardownTableTmpl := setupBtTable(t, ctx, sourceConfig["project"].(string), sourceConfig["instance"].(string), tableNameTemplateParam, columnFamilyName, mutsTmpl, rowKeysTmpl)
	defer teardownTableTmpl(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, BigtableToolKind, paramTestStatement, idParamTestStatement, nameParamTestStatement, arrayTestStatement, authToolStatement)
	toolsFile = addTemplateParamConfig(t, toolsFile)

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	// Get configs for tests
	// Actual test parameters are set in https://github.com/googleapis/genai-toolbox/blob/52b09a67cb40ac0c5f461598b4673136699a3089/tests/tool_test.go#L250
	select1Want := "[{\"$col1\":1}]"
	myToolById4Want := `[{"id":4,"name":""}]`
	mcpMyFailToolWant := `{"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"unable to prepare statement: rpc error: code = InvalidArgument desc = Syntax error: Unexpected identifier \"SELEC\" [at 1:1]"}],"isError":true}}`
	mcpSelect1Want := `{"jsonrpc":"2.0","id":"invoke my-auth-required-tool","result":{"content":[{"type":"text","text":"{\"$col1\":1}"}]}}`
	nameFieldArray := `["CAST(cf['name'] AS string) as name"]`
	nameColFilter := "CAST(cf['name'] AS string)"

	// Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want,
		tests.WithMyToolById4Want(myToolById4Want),
	)
	tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam,
		tests.WithNameFieldArray(nameFieldArray),
		tests.WithNameColFilter(nameColFilter),
		tests.DisableDdlTest(),
		tests.DisableInsertTest(),
	)
}

func convertToBytes(v int) []byte {
	binary1 := new(bytes.Buffer)
	if err := binary.Write(binary1, binary.BigEndian, int64(v)); err != nil {
		log.Fatalf("Unable to encode id: %v", err)
	}
	return binary1.Bytes()
}

func getTestData(columnFamilyName string) ([]*bigtable.Mutation, []string) {
	muts := []*bigtable.Mutation{}
	rowKeys := []string{}

	var ids [4][]byte
	for i := range ids {
		ids[i] = convertToBytes(i + 1)
	}

	now := bigtable.Time(time.Now())
	for rowKey, mutData := range map[string]map[string][]byte{
		// Do not change the test data without checking tests/common_test.go.
		// The structure and value of seed data has to match https://github.com/googleapis/genai-toolbox/blob/4dba0df12dc438eca3cb476ef52aa17cdf232c12/tests/common_test.go#L200-L251
		// Expected values are defined in https://github.com/googleapis/genai-toolbox/blob/52b09a67cb40ac0c5f461598b4673136699a3089/tests/tool_test.go#L229-L310
		"row-01": {
			"name":  []byte("Alice"),
			"email": []byte(tests.ServiceAccountEmail),
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
		"row-04": {
			"name": nil,
			"id":   ids[3],
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

func getTestDataTemplateParam(columnFamilyName string) ([]*bigtable.Mutation, []string) {
	muts := []*bigtable.Mutation{}
	rowKeys := []string{}

	var ids [2][]byte
	for i := range ids {
		ids[i] = convertToBytes(i + 1)
	}

	now := bigtable.Time(time.Now())
	for rowKey, mutData := range map[string]map[string][]byte{
		// Do not change the test data without checking tests/common_test.go.
		// The structure and value of seed data has to match https://github.com/googleapis/genai-toolbox/blob/4dba0df12dc438eca3cb476ef52aa17cdf232c12/tests/common_test.go#L200-L251
		// Expected values are defined in https://github.com/googleapis/genai-toolbox/blob/52b09a67cb40ac0c5f461598b4673136699a3089/tests/tool_test.go#L229-L310
		"row-01": {
			"name": []byte("Alex"),
			"age":  convertToBytes(21),
			"id":   ids[0],
		},
		"row-02": {
			"name": []byte("Alice"),
			"age":  convertToBytes(100),
			"id":   ids[1],
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

func setupBtTable(t *testing.T, ctx context.Context, projectId string, instance string, tableName string, columnFamilyName string, muts []*bigtable.Mutation, rowKeys []string) func(*testing.T) {
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

func addTemplateParamConfig(t *testing.T, config map[string]any) map[string]any {
	toolsMap, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}
	toolsMap["select-templateParams-tool"] = map[string]any{
		"kind":        "bigtable-sql",
		"source":      "my-instance",
		"description": "Create table tool with template parameters",
		"statement":   "SELECT TO_INT64(cf['age']) as age, TO_INT64(cf['id']) as id, CAST(cf['name'] AS string) as name, FROM {{.tableName}};",
		"templateParameters": []parameters.Parameter{
			parameters.NewStringParameter("tableName", "some description"),
		},
	}
	toolsMap["select-templateParams-combined-tool"] = map[string]any{
		"kind":        "bigtable-sql",
		"source":      "my-instance",
		"description": "Create table tool with template parameters",
		"statement":   "SELECT TO_INT64(cf['age']) as age, TO_INT64(cf['id']) as id, CAST(cf['name'] AS string) as name, FROM {{.tableName}} WHERE TO_INT64(cf['id']) = @id;",
		"parameters":  []parameters.Parameter{parameters.NewIntParameter("id", "the id of the user")},
		"templateParameters": []parameters.Parameter{
			parameters.NewStringParameter("tableName", "some description"),
		},
	}
	toolsMap["select-fields-templateParams-tool"] = map[string]any{
		"kind":        "bigtable-sql",
		"source":      "my-instance",
		"description": "Create table tool with template parameters",
		"statement":   "SELECT {{array .fields}}, FROM {{.tableName}};",
		"templateParameters": []parameters.Parameter{
			parameters.NewStringParameter("tableName", "some description"),
			parameters.NewArrayParameter("fields", "The fields to select from", parameters.NewStringParameter("field", "A field that will be returned from the query.")),
		},
	}
	toolsMap["select-filter-templateParams-combined-tool"] = map[string]any{
		"kind":        "bigtable-sql",
		"source":      "my-instance",
		"description": "Create table tool with template parameters",
		"statement":   "SELECT TO_INT64(cf['age']) as age, TO_INT64(cf['id']) as id, CAST(cf['name'] AS string) as name, FROM {{.tableName}} WHERE {{.columnFilter}} = @name;",
		"parameters":  []parameters.Parameter{parameters.NewStringParameter("name", "the name of the user")},
		"templateParameters": []parameters.Parameter{
			parameters.NewStringParameter("tableName", "some description"),
			parameters.NewStringParameter("columnFilter", "some description"),
		},
	}
	config["tools"] = toolsMap
	return config
}
