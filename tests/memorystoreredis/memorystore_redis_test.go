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
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	REDIS_SOURCE_KIND = "memorystore-redis"
	REDIS_TOOL_KIND   = "redis"
	REDIS_ADDRESS     = os.Getenv("MEMORYSTORE_REDIS_ADDRESS")
	REDIS_DATABASE    = os.Getenv("MEMORYSTORE_REDIS_DATABASE")
	REDIS_PASS        = os.Getenv("MEMORYSTORE_REDIS_PASS")
)

func getRedisVars(t *testing.T) map[string]any {
	switch "" {
	case REDIS_ADDRESS:
		t.Fatal("'REDIS_ADDRESS' not set")
	case REDIS_DATABASE:
		t.Fatal("'REDIS_DATABASE' not set")
	case REDIS_PASS:
		t.Fatal("'REDIS_PASS' not set")
	}

	return map[string]any{
		"kind":     REDIS_SOURCE_KIND,
		"address":  REDIS_ADDRESS,
		"database": REDIS_DATABASE,
		"password": REDIS_PASS,
	}
}

type RedisClient interface {
	Do(context.Context, ...any) *redis.Cmd
}

func initMemorystoreRedisClient(ctx context.Context, address string) (RedisClient, error) {
	var client RedisClient
	var err error

	// Create a new Redis client
	standaloneClient := redis.NewClient(&redis.Options{
		Addr:            address,
		PoolSize:        10,
		ConnMaxIdleTime: 60 * time.Second,
		MinIdleConns:    1,
	})
	_, err = standaloneClient.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("unable to connect to redis: %s", err)
	}
	return client, nil
}

func TestMemorystoreRedisToolEndpoints(t *testing.T) {
	//sourceConfig := getRedisVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	//var args []string

	//db, err := strconv.Atoi(REDIS_DATABASE)
	// if err != nil {
	// 	t.Fatalf("unable to convert `REDIS_DATABASE` str to int: %s", err)
	// }
	_, err := initMemorystoreRedisClient(ctx, REDIS_ADDRESS)
	if err != nil {
		t.Fatalf("unable to create Redis connection: %s", err)
	}
	t.Fatalf("success")
	// set up data for param tool
	// teardownDB := tests.SetupRedisDB(t, ctx, client)
	// defer teardownDB(t)

	// // Write config into a file and pass it to command
	// toolsFile := tests.GetToolsConfig(sourceConfig, REDIS_TOOL_KIND, tool_statement1, tool_statement2)

	// cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	// if err != nil {
	// 	t.Fatalf("command initialization returned an error: %s", err)
	// }
	// defer cleanup()

	// waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	// defer cancel()
	// out, err := cmd.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`))
	// if err != nil {
	// 	t.Logf("toolbox command logs: \n%s", out)
	// 	t.Fatalf("toolbox didn't start successfully: %s", err)
	// }

	// tests.RunToolGetTest(t)

	// select1Want, failInvocationWant := tests.GetRedisWants()
	// invokeParamWant, mcpInvokeParamWant := tests.GetNonSpannerInvokeParamWant()
	// tests.RunToolInvokeTest(t, select1Want, invokeParamWant)
	// tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
}
