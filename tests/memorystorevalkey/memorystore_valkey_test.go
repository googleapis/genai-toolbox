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
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/tests"
	"github.com/valkey-io/valkey-go"
)

var (
	MEMORYSTORE_VALKEY_SOURCE_KIND = "memorystore-valkey"
	MEMORYSTORE_VALKEY_TOOL_KIND   = "valkey"
	MEMORYSTORE_VALKEY_ADDRESS     = os.Getenv("MEMORYSTORE_VALKEY_ADDRESS")
	MEMORYSTORE_VALKEY_DATABASE    = os.Getenv("MEMORYSTORE_VALKEY_DATABASE")
)

func getValkeyVars(t *testing.T) map[string]any {
	switch "" {
	case MEMORYSTORE_VALKEY_ADDRESS:
		t.Fatal("'MEMORYSTORE_VALKEY_ADDRESS' not set")
	case MEMORYSTORE_VALKEY_DATABASE:
		t.Fatal("'MEMORYSTORE_VALKEY_DATABASE' not set")
	}

	return map[string]any{
		"kind":     MEMORYSTORE_VALKEY_SOURCE_KIND,
		"address":  []string{MEMORYSTORE_VALKEY_ADDRESS},
		"database": MEMORYSTORE_VALKEY_DATABASE,
		"useIAM":   true,
	}
}

func initMemorystoreValkeyClient(ctx context.Context, addr string, db int) (valkey.Client, error) {

	//Pass in an access token getter fn for IAM auth
	authFn := func(authCtx valkey.AuthCredentialsContext) (valkey.AuthCredentials, error) {
		token, err := sources.GetIAMAccessToken(ctx)
		if err != nil {
			log.Printf("AuthCredentialsFn: Error fetching token: %v", err)
			return valkey.AuthCredentials{}, err
		}
		return valkey.AuthCredentials{
			Username: "",
			Password: token,
		}, nil
	}
	dialer := net.Dialer{Timeout: time.Minute}
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress:       []string{addr},
		Dialer:            dialer,
		SelectDB:          db,
		AuthCredentialsFn: authFn,
		ForceSingleClient: true,
	})

	if err != nil {
		log.Fatalf("error creating client: %v", err)
	}

	// Ping the server to check connectivity (using Do)
	pingCmd := client.B().Ping().Build()
	_, err = client.Do(ctx, pingCmd).ToString()
	if err != nil {
		log.Fatalf("Failed to execute PING command: %v", err)
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
	client, err := initMemorystoreValkeyClient(ctx, MEMORYSTORE_VALKEY_ADDRESS, db)
	if err != nil {
		t.Fatalf("unable to create Valkey connection: %s", err)
	}

	// set up data for param tool
	teardownDB := setupValkeyDB(t, ctx, client)
	defer teardownDB(t)

	// Write config into a file and pass it to command
	authCmds, paramCmds := tests.GetRedisValkeyToolCmds()
	toolsFile := tests.GetRedisValkeyToolsConfig(sourceConfig, MEMORYSTORE_VALKEY_TOOL_KIND, paramCmds, authCmds)

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

	select1Want, failInvocationWant, invokeParamWant, invokeAuthWant, mcpInvokeParamWant := tests.GetRedisValkeyWants()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant, invokeAuthWant)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
}

func setupValkeyDB(t *testing.T, ctx context.Context, client valkey.Client) func(*testing.T) {
	keys := []string{"row1", "row2", "row3"}
	commands := [][]string{
		{"HSET", keys[0], "name", "Alice", "id", "1"},
		{"HSET", keys[1], "name", "Jane", "id", "2"},
		{"HSET", keys[2], "name", "Sid", "id", "3"},
		{"HSET", tests.SERVICE_ACCOUNT_EMAIL, "name", `{"name":"Alice"}`},
	}
	builtCmds := make(valkey.Commands, len(commands))

	for i, cmd := range commands {
		builtCmds[i] = client.B().Arbitrary(cmd...).Build()
	}

	responses := client.DoMulti(ctx, builtCmds...)
	for _, resp := range responses {
		if err := resp.Error(); err != nil {
			t.Fatalf("unable to insert test data: %s", err)
		}
	}

	return func(t *testing.T) {
		// tear down test
		_, err := client.Do(ctx, client.B().Del().Key(keys...).Build()).AsInt64()
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}

}
