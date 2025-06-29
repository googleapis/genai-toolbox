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

package mssql

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

	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	MssqlSourceKind = "mssql"
	MssqlToolKind   = "mssql-sql"
	MssqlDatabase   = os.Getenv("MSSQL_DATABASE")
	MssqlHost       = os.Getenv("MSSQL_HOST")
	MssqlPort       = os.Getenv("MSSQL_PORT")
	MssqlUser       = os.Getenv("MSSQL_USER")
	MssqlPass       = os.Getenv("MSSQL_PASS")
)

func getMsSQLVars(t *testing.T) map[string]any {
	switch "" {
	case MssqlDatabase:
		t.Fatal("'MSSQL_DATABASE' not set")
	case MssqlHost:
		t.Fatal("'MSSQL_HOST' not set")
	case MssqlPort:
		t.Fatal("'MSSQL_PORT' not set")
	case MssqlUser:
		t.Fatal("'MSSQL_USER' not set")
	case MssqlPass:
		t.Fatal("'MSSQL_PASS' not set")
	}

	return map[string]any{
		"kind":     MssqlSourceKind,
		"host":     MssqlHost,
		"port":     MssqlPort,
		"database": MssqlDatabase,
		"user":     MssqlUser,
		"password": MssqlPass,
	}
}

// Copied over from mssql.go
func initMssqlConnection(host, port, user, pass, dbname string) (*sql.DB, error) {
	// Create dsn
	query := url.Values{}
	query.Add("database", dbname)
	url := &url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(user, pass),
		Host:     fmt.Sprintf("%s:%s", host, port),
		RawQuery: query.Encode(),
	}

	// Open database connection
	db, err := sql.Open("sqlserver", url.String())
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	return db, nil
}

func TestMssqlToolEndpoints(t *testing.T) {
	sourceConfig := getMsSQLVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initMssqlConnection(MssqlHost, MssqlPort, MssqlUser, MssqlPass, MssqlDatabase)
	if err != nil {
		t.Fatalf("unable to create SQL Server connection pool: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createStatement1, insertStatement1, toolStatement1, params1 := tests.GetMssqlParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupMsSQLTable(t, ctx, pool, createStatement1, insertStatement1, tableNameParam, params1)
	defer teardownTable1(t)

	// set up data for auth tool
	createStatement2, insertStatement2, toolStatement2, params2 := tests.GetMssqlAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupMsSQLTable(t, ctx, pool, createStatement2, insertStatement2, tableNameAuth, params2)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, MssqlToolKind, toolStatement1, toolStatement2)
	toolsFile = tests.AddMssqlExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetMssqlTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, MssqlToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
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

	tests.RunToolGetTest(t)

	select1Want, failInvocationWant, createTableStatement := tests.GetMssqlWants()
	invokeParamWant, mcpInvokeParamWant := tests.GetNonSpannerInvokeParamWant()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.NewTemplateParameterTestConfig())
}
