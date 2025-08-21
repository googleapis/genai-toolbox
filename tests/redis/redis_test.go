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

package redis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
	"github.com/redis/go-redis/v9"
)

var (
	RedisSourceKind = "redis"
	RedisToolKind   = "redis"
	RedisAddress    = os.Getenv("REDIS_ADDRESS")
	RedisPass       = os.Getenv("REDIS_PASS")
)

func getRedisVars(t *testing.T) map[string]any {
	switch "" {
	case RedisAddress:
		t.Fatal("'REDIS_ADDRESS' not set")
	case RedisPass:
		t.Fatal("'REDIS_PASS' not set")
	}
	return map[string]any{
		"kind":     RedisSourceKind,
		"address":  []string{RedisAddress},
		"password": RedisPass,
	}
}

func initRedisClient(ctx context.Context, address, pass string) (*redis.Client, error) {
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

func TestRedisToolEndpoints(t *testing.T) {
	sourceConfig := getRedisVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	client, err := initRedisClient(ctx, RedisAddress, RedisPass)
	if err != nil {
		t.Fatalf("unable to create Redis connection: %s", err)
	}

	// set up data for param tool
	teardownDB := setupRedisDB(t, ctx, client)
	defer teardownDB(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetRedisValkeyToolsConfig(sourceConfig, RedisToolKind)

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

	select1Want, failInvocationWant, invokeParamWant, invokeIdNullWant, nullWant, mcpInvokeParamWant := tests.GetRedisValkeyWants()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant, invokeIdNullWant, nullWant, true, true)
	runExecuteSqlToolInvokeTest(t)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
}

type User struct {
	Name string `json:"name"`
	Id   int32  `json:"id"`
}

func setupRedisDB(t *testing.T, ctx context.Context, client *redis.Client) func(*testing.T) {
	keys := []string{"row1", "row2", "row3", "row4", "null"}
	user := User{
		Name: "Alice",
		Id:   1,
	}

	// Marshal the struct to JSON
	userJSON, err := json.Marshal(user)
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}
	commands := [][]any{
		{"HSET", keys[0], "id", 1, "name", "Alice"},
		{"HSET", keys[1], "id", 2, "name", "Jane"},
		{"HSET", keys[2], "id", 3, "name", "Sid"},
		{"HSET", keys[3], "id", 4, "name", nil},
		{"SET", keys[4], "null"},
		{"HSET", tests.ServiceAccountEmail, "name", "Alice"},
		{"JSON.SET", "user", "$", string(userJSON)},
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

func runExecuteSqlToolInvokeTest(t *testing.T) {
	// Get ID token
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-exec-cmd-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-cmd-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"cmd":["HGETALL", "row1"]}`)),
			want:          "{\"id\":\"1\",\"name\":\"Alice\"}",
			isErr:         false,
		},
		{
			name:          "invoke my-exec-cmd-tool null response",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-cmd-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"cmd": ["GET", "null"]}`)),
			want:          "\"null\"",
			isErr:         false,
		},
		{
			name:          "invoke my-exec-cmd-tool ping",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-cmd-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"cmd":["PING"]}`)),
			want:          "\"PONG\"",
			isErr:         false,
		},
		{
			name:          "invoke my-exec-cmd-tool push to list",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-cmd-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"cmd": ["RPUSH", "tasks", "task1", "task2", "task3"]}`)),
			want:          "3",
			isErr:         false,
		},
		{
			name:          "invoke my-exec-cmd-tool read a range in a list",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-cmd-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"cmd":["LRANGE", "tasks", "0" ,"-1"]}`)),
			want:          "[\"task1\",\"task2\",\"task3\"]",
			isErr:         false,
		},
		{
			name:          "invoke my-exec-cmd-tool json get",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-cmd-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"cmd":["JSON.GET", "user", "$"]}`)),
			want:          "\"[{\\\"name\\\":\\\"Alice\\\",\\\"id\\\":1}]\"",
			isErr:         false,
		},
		{
			name:          "Invoke my-auth-exec-cmd-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-exec-cmd-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"cmd":["HGETALL", "%s"]}`, tests.ServiceAccountEmail))),
			isErr:         false,
			want:          "[{\"name\":\"Alice\"}]",
		},
		{
			name:          "Invoke my-auth-exec-cmd-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-exec-cmd-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:   bytes.NewBuffer([]byte(`{"cmd":["HGETALL", "row1"]}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-exec-cmd-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-exec-cmd-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"cmd":["PING"]}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Check response body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if got != tc.want {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}
