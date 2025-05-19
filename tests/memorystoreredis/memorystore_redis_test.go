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

package memorystoreredis

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/tests"
	"github.com/redis/go-redis/v9"
)

var (
	REDIS_SOURCE_KIND     = "memorystore-redis"
	REDIS_TOOL_KIND       = "redis"
	REDIS_ADDRESS         = os.Getenv("MEMORYSTORE_REDIS_ADDRESS")
	REDIS_PASS            = os.Getenv("MEMORYSTORE_REDIS_PASS")
	SERVICE_ACCOUNT_EMAIL = os.Getenv("SERVICE_ACCOUNT_EMAIL")
)

func getRedisVars(t *testing.T) map[string]any {
	switch "" {
	case REDIS_ADDRESS:
		t.Fatal("'REDIS_ADDRESS' not set")
	}
	return map[string]any{
		"kind":           REDIS_SOURCE_KIND,
		"address":        REDIS_ADDRESS,
		"password":       REDIS_PASS,
		"clusterEnabled": false,
	}
}

func initMemorystoreRedisClient(ctx context.Context, address, pass string) (*redis.Client, error) {
	// Create a new Redis client
	standaloneClient := redis.NewClient(&redis.Options{
		Addr:            address,
		PoolSize:        10,
		ConnMaxIdleTime: 60 * time.Second,
		MinIdleConns:    1,
		Password:        pass,
	})
	_, err := standaloneClient.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("unable to connect to redis: %s", err)
	}
	return standaloneClient, nil
}

func TestMemorystoreRedisToolEndpoints(t *testing.T) {
	sourceConfig := getRedisVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	client, err := initMemorystoreRedisClient(ctx, REDIS_ADDRESS, REDIS_PASS)
	if err != nil {
		t.Fatalf("unable to create Redis connection: %s", err)
	}

	// set up data for param tool
	teardownDB := setupRedisDB(t, ctx, client)
	defer teardownDB(t)

	param_cmds, auth_cmds := tests.GetRedisValkeyToolCmds()

	// Write config into a file and pass it to command
	toolsFile := tests.GetRedisValkeyToolsConfig(sourceConfig, REDIS_TOOL_KIND, param_cmds, auth_cmds)

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

func setupRedisDB(t *testing.T, ctx context.Context, client *redis.Client) func(*testing.T) {
	keys := []string{"row1", "row2", "row3"}
	commands := [][]any{
		{"HSET", keys[0], "name", "Alice", "id", "1"},
		{"HSET", keys[1], "name", "Jane", "id", "2"},
		{"HSET", keys[2], "name", "Sid", "id", "3"},
		{"HSET", SERVICE_ACCOUNT_EMAIL, "name", `{"name":"Alice"}`},
	}
	for _, c := range commands {
		resp := client.Do(ctx, c...)
		if err := resp.Err(); err != nil {
			t.Fatalf("unable to insert test data: %s", err)
		}
	}

	return func(t *testing.T) {
		// tear down test
		_, err := client.Del(ctx, keys...).Result()
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}

}
