// Copyright 2024 Google LLC
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

package alloydbpg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/alloydbconn"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	AlloyDBPostgresSourceKind            = "alloydb-postgres"
	AlloyDBPostgresToolKind              = "postgres-sql"
	AlloyDBPostgresCreateClusterToolKind = "alloydb-pg-create-cluster"
	AlloyDBPostgresProject               = os.Getenv("ALLOYDB_POSTGRES_PROJECT")
	AlloyDBPostgresRegion                = os.Getenv("ALLOYDB_POSTGRES_REGION")
	AlloyDBPostgresCluster               = os.Getenv("ALLOYDB_POSTGRES_CLUSTER")
	AlloyDBPostgresInstance              = os.Getenv("ALLOYDB_POSTGRES_INSTANCE")
	AlloyDBPostgresDatabase              = os.Getenv("ALLOYDB_POSTGRES_DATABASE")
	AlloyDBPostgresUser                  = os.Getenv("ALLOYDB_POSTGRES_USER")
	AlloyDBPostgresPass                  = os.Getenv("ALLOYDB_POSTGRES_PASS")
)

func getAlloyDBPgVars(t *testing.T) map[string]any {
	switch "" {
	case AlloyDBPostgresProject:
		t.Fatal("'ALLOYDB_POSTGRES_PROJECT' not set")
	case AlloyDBPostgresRegion:
		t.Fatal("'ALLOYDB_POSTGRES_REGION' not set")
	case AlloyDBPostgresCluster:
		t.Fatal("'ALLOYDB_POSTGRES_CLUSTER' not set")
	case AlloyDBPostgresInstance:
		t.Fatal("'ALLOYDB_POSTGRES_INSTANCE' not set")
	case AlloyDBPostgresDatabase:
		t.Fatal("'ALLOYDB_POSTGRES_DATABASE' not set")
	case AlloyDBPostgresUser:
		t.Fatal("'ALLOYDB_POSTGRES_USER' not set")
	case AlloyDBPostgresPass:
		t.Fatal("'ALLOYDB_POSTGRES_PASS' not set")
	}
	return map[string]any{
		"kind":     AlloyDBPostgresSourceKind,
		"project":  AlloyDBPostgresProject,
		"cluster":  AlloyDBPostgresCluster,
		"instance": AlloyDBPostgresInstance,
		"region":   AlloyDBPostgresRegion,
		"database": AlloyDBPostgresDatabase,
		"user":     AlloyDBPostgresUser,
		"password": AlloyDBPostgresPass,
	}
}

// Copied over from  alloydb_pg.go
func getAlloyDBDialOpts(ipType string) ([]alloydbconn.DialOption, error) {
	switch strings.ToLower(ipType) {
	case "private":
		return []alloydbconn.DialOption{alloydbconn.WithPrivateIP()}, nil
	case "public":
		return []alloydbconn.DialOption{alloydbconn.WithPublicIP()}, nil
	default:
		return nil, fmt.Errorf("invalid ipType %s", ipType)
	}
}

// Copied over from  alloydb_pg.go
func initAlloyDBPgConnectionPool(project, region, cluster, instance, ipType, user, pass, dbname string) (*pgxpool.Pool, error) {
	// Configure the driver to connect to the database
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, pass, dbname)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection uri: %w", err)
	}

	// Create a new dialer with options
	dialOpts, err := getAlloyDBDialOpts(ipType)
	if err != nil {
		return nil, err
	}
	d, err := alloydbconn.NewDialer(context.Background(), alloydbconn.WithDefaultDialOptions(dialOpts...))
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection uri: %w", err)
	}

	// Tell the driver to use the AlloyDB Go Connector to create connections
	i := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", project, region, cluster, instance)
	config.ConnConfig.DialFunc = func(ctx context.Context, _ string, instance string) (net.Conn, error) {
		return d.Dial(ctx, i)
	}

	// Interact with the driver directly as you normally would
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func TestAlloyDBPgToolEndpoints(t *testing.T) {
	sourceConfig := getAlloyDBPgVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initAlloyDBPgConnectionPool(AlloyDBPostgresProject, AlloyDBPostgresRegion, AlloyDBPostgresCluster, AlloyDBPostgresInstance, "public", AlloyDBPostgresUser, AlloyDBPostgresPass, AlloyDBPostgresDatabase)
	if err != nil {
		t.Fatalf("unable to create AlloyDB connection pool: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := tests.GetPostgresSQLParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupPostgresSQLTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetPostgresSQLAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupPostgresSQLTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, AlloyDBPostgresToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddPgExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetPostgresSQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, AlloyDBPostgresToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

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
	select1Want, failInvocationWant, createTableStatement, mcpSelect1Want := tests.GetPostgresWants()

	// Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want)
	tests.RunMCPToolCallMethod(t, failInvocationWant, mcpSelect1Want)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam)
}

// Test connection with different IP type
func TestAlloyDBPgIpConnection(t *testing.T) {
	sourceConfig := getAlloyDBPgVars(t)

	tcs := []struct {
		name   string
		ipType string
	}{
		{
			name:   "public ip",
			ipType: "public",
		},
		{
			name:   "private ip",
			ipType: "private",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			sourceConfig["ipType"] = tc.ipType
			err := tests.RunSourceConnectionTest(t, sourceConfig, AlloyDBPostgresToolKind)
			if err != nil {
				t.Fatalf("Connection test failure: %s", err)
			}
		})
	}
}

// Test IAM connection
func TestAlloyDBPgIAMConnection(t *testing.T) {
	getAlloyDBPgVars(t)
	// service account email used for IAM should trim the suffix
	serviceAccountEmail := strings.TrimSuffix(tests.ServiceAccountEmail, ".gserviceaccount.com")

	noPassSourceConfig := map[string]any{
		"kind":     AlloyDBPostgresSourceKind,
		"project":  AlloyDBPostgresProject,
		"cluster":  AlloyDBPostgresCluster,
		"instance": AlloyDBPostgresInstance,
		"region":   AlloyDBPostgresRegion,
		"database": AlloyDBPostgresDatabase,
		"user":     serviceAccountEmail,
	}

	noUserSourceConfig := map[string]any{
		"kind":     AlloyDBPostgresSourceKind,
		"project":  AlloyDBPostgresProject,
		"cluster":  AlloyDBPostgresCluster,
		"instance": AlloyDBPostgresInstance,
		"region":   AlloyDBPostgresRegion,
		"database": AlloyDBPostgresDatabase,
		"password": "random",
	}

	noUserNoPassSourceConfig := map[string]any{
		"kind":     AlloyDBPostgresSourceKind,
		"project":  AlloyDBPostgresProject,
		"cluster":  AlloyDBPostgresCluster,
		"instance": AlloyDBPostgresInstance,
		"region":   AlloyDBPostgresRegion,
		"database": AlloyDBPostgresDatabase,
	}
	tcs := []struct {
		name         string
		sourceConfig map[string]any
		isErr        bool
	}{
		{
			name:         "no user no pass",
			sourceConfig: noUserNoPassSourceConfig,
			isErr:        false,
		},
		{
			name:         "no password",
			sourceConfig: noPassSourceConfig,
			isErr:        false,
		},
		{
			name:         "no user",
			sourceConfig: noUserSourceConfig,
			isErr:        true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := tests.RunSourceConnectionTest(t, tc.sourceConfig, AlloyDBPostgresToolKind)
			if err != nil {
				if tc.isErr {
					return
				}
				t.Fatalf("Connection test failure: %s", err)
			}
			if tc.isErr {
				t.Fatalf("Expected error but test passed.")
			}
		})
	}
}

// HTTP handler for mock server
type handler struct {
	mu       sync.Mutex
	response mockResponse
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.response.body == "" {
		return
	}

	w.WriteHeader(h.response.statusCode)
	if _, err := w.Write([]byte(h.response.body)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *handler) setResponse(res mockResponse) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.response = res
}

type mockResponse struct {
	statusCode int
	body       string
}

func TestAlloyDBPgCreateCluster(t *testing.T) {
	h := &handler{}
	server := httptest.NewServer(h)
	defer server.Close()

	toolsFile := getAlloyDBPgCreateToolsConfig(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile)
	if err != nil {
		t.Fatalf("command initialization failed: %v", err)
	}
	defer cleanup()

	waitCtx, waitCancel := context.WithTimeout(ctx, 10*time.Second)
	defer waitCancel()
	_, err = testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Fatalf("toolbox server didn't start successfully: %s", err)
	}

	tcs := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		mockResponse   mockResponse 
		wantResult     map[string]any
	}{
		{
			name:           "create cluster success",
			requestBody:    fmt.Sprintf(`{"project": "%s", "location": "%s", "clusterId": "test", "password": "p"}`, "test-project", "test-location"),
			wantStatusCode: http.StatusOK,
			mockResponse:   mockResponse{statusCode: http.StatusOK, body: `{"done":false,"metadata":{"@type":"type.googleapis.com/google.cloud.alloydb.v1.OperationMetadata","apiVersion":"v1","createTime":"2025-09-04T05:38:38.055667814Z","requestedCancellation":false,"target":"projects/test-project/locations/test-location/clusters/test-create-cluster","verb":"create"},"name":"projects/test-project/locations/test-location/operations/test-operation"}`},
			wantResult:     map[string]any{"done": false, "metadata": map[string]any{"@type": "type.googleapis.com/google.cloud.alloydb.v1.OperationMetadata", "apiVersion": "v1", "createTime": "2025-09-04T05:38:38.055667814Z", "requestedCancellation": false, "target": "projects/test-project/locations/test-location/clusters/test-create-cluster", "verb": "create"}, "name": "projects/test-project/locations/test-location/operations/test-operation"},
		},
		{
			name:           "create cluster failure",
			requestBody:    fmt.Sprintf(`{"project": "%s", "location": "%s", "clusterId": "test", "password": "p"}`, AlloyDBPostgresProject, AlloyDBPostgresRegion),
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{statusCode: http.StatusInternalServerError, body: `{"error": "failed"}`},
		},
		{
			name:           "create cluster with missing project",
			requestBody:    `{"location": "l1", "clusterId": "c1", "password": "p1"}`,
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{statusCode: http.StatusBadRequest, body: ""},
		},
		{
			name:           "fcreate cluster with missing clusterId",
			requestBody:    `{"project": "p1", "location": "l1", "password": "p1"}`,
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{statusCode: http.StatusBadRequest, body: ""},
		},
		{
			name:           "create cluster with missing password",
			requestBody:    `{"project": "p1", "location": "l1", "clusterId": "c1"}`,
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{statusCode: http.StatusBadRequest, body: ""},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h.setResponse(tc.mockResponse)

			api := "http://127.0.0.1:5000/api/tool/alloydb-pg-create-cluster/invoke"
			req, err := http.NewRequest(http.MethodPost, api, bytes.NewBufferString(tc.requestBody))
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatusCode {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected status %d, got %d: %s", tc.wantStatusCode, resp.StatusCode, string(bodyBytes))
			}

			if tc.wantResult != nil {
				var result struct {
					Result string `json:"result"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				var got map[string]any
				if err := json.Unmarshal([]byte(result.Result), &got); err != nil {
					t.Fatalf("failed to unmarshal nested result: %v", err)
				}

				if !reflect.DeepEqual(got, tc.wantResult) {
					t.Fatalf("unexpected result: got %+v, want %+v", got, tc.wantResult)
				}
			}
		})
	}
}

func getAlloyDBPgCreateToolsConfig(baseURL string) map[string]any {
	return map[string]any{
		"tools": map[string]any{
			"alloydb-pg-create-cluster": map[string]any{
				"kind":        AlloyDBPostgresCreateClusterToolKind,
				"description": "Create a new AlloyDB cluster. This is a long-running operation, but the API call returns quickly. This will return operation id to be used by get operations tool. Take all parameters from user in one go.",
				"baseURL":     baseURL,
			},
		},
	}
}
