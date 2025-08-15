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

package trino

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
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	TrinoSourceKind = "trino"
	TrinoToolKind   = "trino-sql"
	TrinoHost       = os.Getenv("TRINO_HOST")
	TrinoPort       = os.Getenv("TRINO_PORT")
	TrinoUser       = os.Getenv("TRINO_USER")
	TrinoPass       = os.Getenv("TRINO_PASS")
	TrinoCatalog    = os.Getenv("TRINO_CATALOG")
	TrinoSchema     = os.Getenv("TRINO_SCHEMA")
)

func getTrinoVars(t *testing.T) map[string]any {
	switch "" {
	case TrinoHost:
		t.Fatal("'TRINO_HOST' not set")
	case TrinoPort:
		t.Fatal("'TRINO_PORT' not set")
	case TrinoUser:
		t.Fatal("'TRINO_USER' not set")
	case TrinoCatalog:
		t.Fatal("'TRINO_CATALOG' not set")
	case TrinoSchema:
		t.Fatal("'TRINO_SCHEMA' not set")
	}

	return map[string]any{
		"kind":     TrinoSourceKind,
		"host":     TrinoHost,
		"port":     TrinoPort,
		"user":     TrinoUser,
		"password": TrinoPass,
		"catalog":  TrinoCatalog,
		"schema":   TrinoSchema,
	}
}

// initTrinoConnectionPool creates a Trino connection pool (copied from trino.go)
func initTrinoConnectionPool(host, port, user, pass, catalog, schema string) (*sql.DB, error) {
	dsn, err := buildTrinoDSN(host, port, user, pass, catalog, schema, "", "", false, false)
	if err != nil {
		return nil, fmt.Errorf("failed to build DSN: %w", err)
	}

	db, err := sql.Open("trino", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// buildTrinoDSN builds a Trino DSN string (simplified version from trino.go)
func buildTrinoDSN(host, port, user, password, catalog, schema, queryTimeout, accessToken string, kerberosEnabled, sslEnabled bool) (string, error) {
	scheme := "http"
	if sslEnabled {
		scheme = "https"
	}

	dsn := fmt.Sprintf("%s://%s@%s:%s?catalog=%s&schema=%s", scheme, user, host, port, catalog, schema)

	if password != "" {
		dsn = fmt.Sprintf("%s://%s:%s@%s:%s?catalog=%s&schema=%s", scheme, user, password, host, port, catalog, schema)
	}

	if queryTimeout != "" {
		dsn += "&queryTimeout=" + queryTimeout
	}

	if accessToken != "" {
		dsn += "&accessToken=" + accessToken
	}

	if kerberosEnabled {
		dsn += "&KerberosEnabled=true"
	}

	return dsn, nil
}

func TestTrinoToolEndpoints(t *testing.T) {
	sourceConfig := getTrinoVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initTrinoConnectionPool(TrinoHost, TrinoPort, TrinoUser, TrinoPass, TrinoCatalog, TrinoSchema)
	if err != nil {
		t.Fatalf("unable to create Trino connection pool: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := tests.GetTrinoParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupTrinoTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetTrinoAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupTrinoTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, TrinoToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddTrinoExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetTrinoTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, TrinoToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

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

	tests.RunToolGetTest(t)

	select1Want, failInvocationWant, createTableStatement := tests.GetTrinoWants()
	invokeParamWant, invokeIdNullWant, nullWant, mcpInvokeParamWant := tests.GetNonSpannerInvokeParamWant()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant, invokeIdNullWant, nullWant, true, false)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.NewTemplateParameterTestConfig())
}
