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

package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	MysqlSourceKind = "mysql"
	MysqlToolKind   = "mysql-sql"
	MysqlDatabase   = os.Getenv("MYSQL_DATABASE")
	MysqlHost       = os.Getenv("MYSQL_HOST")
	MysqlPort       = os.Getenv("MYSQL_PORT")
	MysqlUser       = os.Getenv("MYSQL_USER")
	MysqlPass       = os.Getenv("MYSQL_PASS")
)

func getMySQLVars(t *testing.T) map[string]any {
	switch "" {
	case MysqlDatabase:
		t.Fatal("'MYSQL_DATABASE' not set")
	case MysqlHost:
		t.Fatal("'MYSQL_HOST' not set")
	case MysqlPort:
		t.Fatal("'MYSQL_PORT' not set")
	case MysqlUser:
		t.Fatal("'MYSQL_USER' not set")
	case MysqlPass:
		t.Fatal("'MYSQL_PASS' not set")
	}

	return map[string]any{
		"kind":     MysqlSourceKind,
		"host":     MysqlHost,
		"port":     MysqlPort,
		"database": MysqlDatabase,
		"user":     MysqlUser,
		"password": MysqlPass,
	}
}

// Copied over from mysql.go
func initMySQLConnectionPool(host, port, user, pass, dbname string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, dbname)

	// Interact with the driver directly as you normally would
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	return pool, nil
}

func TestMysqlToolEndpoints(t *testing.T) {
	sourceConfig := getMySQLVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initMySQLConnectionPool(MysqlHost, MysqlPort, MysqlUser, MysqlPass, MysqlDatabase)
	if err != nil {
		t.Fatalf("unable to create MySQL connection pool: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createStatement1, insertStatement1, toolStatement1, params1 := tests.GetMysqlParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupMySQLTable(t, ctx, pool, createStatement1, insertStatement1, tableNameParam, params1)
	defer teardownTable1(t)

	// set up data for auth tool
	createStatement2, insertStatement2, toolStatement2, params2 := tests.GetMysqlAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupMySQLTable(t, ctx, pool, createStatement2, insertStatement2, tableNameAuth, params2)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, MysqlToolKind, toolStatement1, toolStatement2)
	toolsFile = tests.AddMySqlExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetMysqlTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, MysqlToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

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

	select1Want, failInvocationWant, createTableStatement := tests.GetMysqlWants()
	invokeParamWant, mcpInvokeParamWant := tests.GetNonSpannerInvokeParamWant()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.NewTemplateParameterTestConfig())
}
