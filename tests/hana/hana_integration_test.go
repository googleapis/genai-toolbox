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

package hana

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	_ "github.com/SAP/go-hdb/driver"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	HanaSourceKind = "hana"
	HanaToolKind   = "hana-sql"
	HanaDatabase   = os.Getenv("HANA_DATABASE")
	HanaHost       = os.Getenv("HANA_HOST")
	HanaPort       = os.Getenv("HANA_PORT")
	HanaUser       = os.Getenv("HANA_USER")
	HanaPass       = os.Getenv("HANA_PASSWORD")
)

func getHanaVars(t *testing.T) map[string]any {
	switch "" {
	case HanaDatabase:
		t.Fatal("'HANA_DATABASE' not set")
	case HanaHost:
		t.Fatal("'HANA_HOST' not set")
	case HanaPort:
		t.Fatal("'HANA_PORT' not set")
	case HanaUser:
		t.Fatal("'HANA_USER' not set")
	case HanaPass:
		t.Fatal("'HANA_PASSWORD' not set")
	}

	return map[string]any{
		"kind":     HanaSourceKind,
		"host":     HanaHost,
		"port":     HanaPort,
		"database": HanaDatabase,
		"user":     HanaUser,
		"password": HanaPass,
	}
}

// initHanaConnection creates a connection using the go-hdb driver.
func initHanaConnection(host, port, user, pass, dbname string) (*sql.DB, error) {
	// Construct DSN: hdb://user:pass@host:port?databaseName=<dbname>
	dsnURL := &url.URL{
		Scheme: "hdb",
		User:   url.UserPassword(user, pass),
		Host:   fmt.Sprintf("%s:%s", host, port),
	}

	qs := url.Values{}
	if dbname != "" {
		qs.Add("databaseName", dbname)
	}
	if len(qs) > 0 {
		dsnURL.RawQuery = qs.Encode()
	}

	db, err := sql.Open("hdb", dsnURL.String())
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	return db, nil
}

func TestHanaToolEndpoints(t *testing.T) {
	sourceConfig := getHanaVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	db, err := initHanaConnection(HanaHost, HanaPort, HanaUser, HanaPass, HanaDatabase)
	if err != nil {
		t.Fatalf("unable to create HANA connection: %s", err)
	}
	defer db.Close()

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("unable to ping HANA database: %s", err)
	}

	// create table name with UUID
	tableNameParam := "PARAM_TABLE_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "AUTH_TABLE_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "TEMPLATE_PARAM_TABLE_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := tests.GetHanaParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupHanaTable(t, ctx, db, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetHanaAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupHanaTable(t, ctx, db, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// set up data for template param tool
	createTemplateParamTableStmt, insertTemplateParamTableStmt, templateParamToolStmt, templateParamTestParams := tests.GetHanaTemplateParamToolInfo(tableNameTemplateParam)
	teardownTable3 := tests.SetupHanaTable(t, ctx, db, createTemplateParamTableStmt, insertTemplateParamTableStmt, tableNameTemplateParam, templateParamTestParams)
	defer teardownTable3(t)

	ctxWithLogger, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Test hana-sql tool with parameters
	args = []string{
		tests.CreateSource("hana-source", sourceConfig),
		tests.CreateToolWithParams("hana-param-tool", "hana-sql", "hana-source", "Test HANA param tool", paramToolStmt, paramTestParams),
		tests.CreateToolWithParams("hana-id-param-tool", "hana-sql", "hana-source", "Test HANA ID param tool", idParamToolStmt, paramTestParams[:1]),
		tests.CreateToolWithParams("hana-name-param-tool", "hana-sql", "hana-source", "Test HANA name param tool", nameParamToolStmt, paramTestParams[1:2]),
		tests.CreateToolWithAuth("hana-auth-tool", "hana-sql", "hana-source", "Test HANA auth tool", authToolStmt, []string{"google"}, authTestParams),
		tests.CreateToolWithArrayParams("hana-array-tool", "hana-sql", "hana-source", "Test HANA array tool", arrayToolStmt, paramTestParams),
		tests.CreateToolWithTemplateParams("hana-template-param-tool", "hana-sql", "hana-source", "Test HANA template param tool", templateParamToolStmt, templateParamTestParams),
	}

	tests.RunServerTest(t, ctxWithLogger, args,
		[]tests.ToolTest{
			{
				Name:   "hana-param-tool",
				Params: map[string]any{"id": 1, "name": "param1"},
			},
			{
				Name:   "hana-id-param-tool",
				Params: map[string]any{"id": 2},
			},
			{
				Name:   "hana-name-param-tool",
				Params: map[string]any{"name": "param2"},
			},
			{
				Name:          "hana-auth-tool",
				Params:        map[string]any{"id": 1},
				AuthServices:  []string{"google"},
				StatusPattern: regexp.MustCompile(`401`),
			},
			{
				Name:   "hana-array-tool",
				Params: map[string]any{"ids": []int{1, 2}},
			},
			{
				Name:   "hana-template-param-tool",
				Params: map[string]any{"tableName": tableNameTemplateParam},
			},
		})

	// Test hana-execute-sql tool
	args = []string{
		tests.CreateSource("hana-source", sourceConfig),
		tests.CreateTool("hana-execute-sql-tool", "hana-execute-sql", "hana-source", "Test HANA execute SQL tool"),
	}

	tests.RunServerTest(t, ctxWithLogger, args,
		[]tests.ToolTest{
			{
				Name:   "hana-execute-sql-tool",
				Params: map[string]any{"sql": fmt.Sprintf("SELECT COUNT(*) as row_count FROM %s", tableNameParam)},
			},
		})
}
