//go:build integration

//
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

// Package tests contains end to end tests meant to verify the Toolbox Server
// works as expected when executed as a binary.

package tests

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/googleapis/genai-toolbox/cmd"
)

// tmpFileWithCleanup creates a temporary file with the content and returns the path and
// a function to clean it up, or any errors encountered instead
func tmpFileWithCleanup(content []byte) (string, func(), error) {
	// create a random file in the temp dir
	f, err := os.CreateTemp("", "*") // * indicates random string
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.Remove(f.Name()) }

	if _, err := f.Write(content); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", nil, err
	}
	return f.Name(), cleanup, err
}

// CmdExec represents an invocation of a toolbox command.
type CmdExec struct {
	Out io.ReadCloser

	cmd     *cmd.Command
	cancel  context.CancelFunc
	closers []io.Closer
	done    chan bool // closed once the cmd is completed
	err     error
}

// StartCmd returns a CmdExec representing a running instance of a toolbox command.
func StartCmd(ctx context.Context, toolsFile map[string]any, args ...string) (*CmdExec, func(), error) {
	b, err := yaml.Marshal(toolsFile)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to marshal tools file: %s", err)
	}
	path, cleanup, err := tmpFileWithCleanup(b)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to write tools file: %s", err)
	}
	args = append(args, "--tools_file", path)

	ctx, cancel := context.WithCancel(ctx)
	// Open a pipe for tracking the output from the cmd
	pr, pw, err := os.Pipe()
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("unable to open stdout pipe: %w", err)
	}

	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("unable to initiate logger: %w", err)
	}
	c := cmd.NewCommand(cmd.WithStreams(pw, pw))
	c.SetArgs(args)

	t := &CmdExec{
		Out:     pr,
		cmd:     c,
		cancel:  cancel,
		closers: []io.Closer{pr, pw},
		done:    make(chan bool),
	}

	// Start the command in the background
	go func() {
		defer close(t.done)
		defer cancel()
		t.err = c.ExecuteContext(ctx)
	}()
	return t, cleanup, nil

}

// Stop sends the TERM signal to the cmd and returns.
func (c *CmdExec) Stop() {
	c.cancel()
}

// Waits until the execution is completed and returns any error from the result.
func (c *CmdExec) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.done:
		return c.err
	}
}

// Done returns true if the command has exited.
func (c *CmdExec) Done() bool {
	select {
	case <-c.done:
		return true
	default:
	}
	return false
}

// Close releases any resources associated with the instance.
func (c *CmdExec) Close() {
	c.cancel()
	for _, c := range c.closers {
		c.Close()
	}
}

// WaitForString waits until the server logs a single line that matches the provided regex.
// returns the output of whatever the server sent so far.
func (c *CmdExec) WaitForString(ctx context.Context, re *regexp.Regexp) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	in := bufio.NewReader(c.Out)

	// read lines in background, sending result of each read over a channel
	// this allows us to use in.ReadString without blocking
	type result struct {
		s   string
		err error
	}
	output := make(chan result)
	go func() {
		defer close(output)
		for {
			select {
			case <-ctx.Done():
				// if the context is canceled, the orig thread will send back the error
				// so we can just exit the goroutine here
				return
			default:
				// otherwise read a line from the output
				s, err := in.ReadString('\n')
				if err != nil {
					output <- result{err: err}
					return
				}
				output <- result{s: s}
				// if that last string matched, exit the goroutine
				if re.MatchString(s) {
					return
				}
			}
		}
	}()

	// collect the output until the ctx is canceled, an error was hit,
	// or match was found (which is indicated the channel is closed)
	var sb strings.Builder
	for {
		select {
		case <-ctx.Done():
			// if ctx is done, return that error
			return sb.String(), ctx.Err()
		case o, ok := <-output:
			if !ok {
				// match was found!
				return sb.String(), nil
			}
			if o.err != nil {
				// error was found!
				return sb.String(), o.err
			}
			sb.WriteString(o.s)
		}
	}
}

func RunToolInvocationWithParamsTest(t *testing.T, sourceConfig map[string]any, toolKind string, tableName string) {
	// Write config into a file and pass it to command
	var statement string
	switch toolKind {
	case "postgres-sql":
		statement = fmt.Sprintf("SELECT * FROM %s WHERE id = $1 OR name = $2;", tableName)
	default:
		t.Fatalf("invalid tool kind: %s", toolKind)
	}

	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"tools": map[string]any{
			"my-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with params.",
				"statement":   statement,
				"parameters": []any{
					map[string]any{
						"name":        "id",
						"type":        "integer",
						"description": "user ID",
					},
					map[string]any{
						"name":        "name",
						"type":        "string",
						"description": "user name",
					},
				},
			},
		},
	}

	// Initialize a test command
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	cmd, cleanup, err := StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	out, err := cmd.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`))
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	// Test Tool invocation with parameters
	invokeTcs := []struct {
		name string
		api  string

		requestBody io.Reader
		want        string
		isErr       bool
	}{
		{
			name:        "Invoke my-tool with parameters",
			api:         "http://127.0.0.1:5000/api/tool/my-tool/invoke",
			requestBody: bytes.NewBuffer([]byte(`{"id": 3, "name": "Alice"}`)),
			isErr:       false,
			want:        "Stub tool call for \"my-tool\"! Parameters parsed: [{\"id\" '\\x03'} {\"name\" \"Alice\"}] \n Output: [%!s(int32=1) Alice][%!s(int32=3) Sid]",
		},
		{
			name:        "Invoke my-tool without parameters",
			api:         "http://127.0.0.1:5000/api/tool/my-tool/invoke",
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			isErr:       true,
		},
		{
			name:        "Invoke my-tool without insufficient parameters",
			api:         "http://127.0.0.1:5000/api/tool/my-tool/invoke",
			requestBody: bytes.NewBuffer([]byte(`{"id": 1}`)),
			isErr:       true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request with parameters
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}

			if resp.StatusCode != http.StatusOK {
				if tc.isErr == true {
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

func RunSourceConnectionTest(t *testing.T, sourceConfig map[string]any, toolKind string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"tools": map[string]any{
			"my-simple-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
				"statement":   "SELECT 1;",
			},
		},
	}
	cmd, cleanup, err := StartCmd(ctx, toolsFile, args...)
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
}
