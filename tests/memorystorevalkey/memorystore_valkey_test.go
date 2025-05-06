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

package memorystorevalkey

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/tests"
	"github.com/valkey-io/valkey-go"
)

var (
	MEMORYSTORE_VALKEY_SOURCE_KIND = "memorystore-valkey"
	MEMORYSTORE_VALKEY_TOOL_KIND   = "redis"
	MEMORYSTORE_VALKEY_ADDRESS     = os.Getenv("MEMORYSTORE_VALKEY_ADDRESS")
	MEMORYSTORE_VALKEY_DATABASE    = os.Getenv("MEMORYSTORE_VALKEY_DATABASE")
	MEMORYSTORE_VALKEY_PASS        = os.Getenv("MEMORYSTORE_VALKEY_PASS")
)

func getValkeyVars(t *testing.T) map[string]any {
	switch "" {
	case MEMORYSTORE_VALKEY_ADDRESS:
		t.Fatal("'MEMORYSTORE_VALKEY_ADDRESS' not set")
	case MEMORYSTORE_VALKEY_DATABASE:
		t.Fatal("'MEMORYSTORE_VALKEY_DATABASE' not set")
	case MEMORYSTORE_VALKEY_PASS:
		t.Fatal("'MEMORYSTORE_VALKEY_PASS' not set")
	}

	return map[string]any{
		"kind":     MEMORYSTORE_VALKEY_SOURCE_KIND,
		"address":  []string{MEMORYSTORE_VALKEY_ADDRESS},
		"database": MEMORYSTORE_VALKEY_DATABASE,
		"password": MEMORYSTORE_VALKEY_PASS,
		"useIAM":   true,
	}
}

func initMemorystoreValkeyClient(ctx context.Context, addr, pass string, db int) (valkey.Client, error) {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{addr},
		Password:    pass,
		SelectDB:    db,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	// Ping the server to check connectivity (using Do)
	pingCmd := client.B().Ping().Build()
	_, err = client.Do(ctx, pingCmd).ToString()
	if err != nil {
		return nil, fmt.Errorf("Failed to execute PING command: %v", err)
	}
	return client, nil
}

func TestMemorystoreValkeyToolEndpoints(t *testing.T) {
	sourceConfig := getValkeyVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	db, err := strconv.Atoi(MEMORYSTORE_VALKEY_DATABASE)
	if err != nil {
		t.Fatalf("unable to convert `VALKEY_DATABASE` str to int: %s", err)
	}
	client, err := initMemorystoreValkeyClient(ctx, MEMORYSTORE_VALKEY_ADDRESS, MEMORYSTORE_VALKEY_PASS, db)
	if err != nil {
		t.Fatalf("unable to create SQL Server connection pool: %s", err)
	}
	// set up data for param tool
	teardownDB := tests.SetupRedisDB(t, ctx, client)
	defer teardownDB(t)

	// Write config into a file and pass it to command
	authCmds, paramCmds := tests.GetValkeyRedisToolCmds()
	toolsFile := tests.GetValkeyRedisToolsConfig(sourceConfig, MEMORYSTORE_VALKEY_TOOL_KIND, paramCmds, authCmds)

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

	select1Want, failInvocationWant := tests.GetRedisWants()
	invokeParamWant, mcpInvokeParamWant := tests.GetNonSpannerInvokeParamWant()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
}
