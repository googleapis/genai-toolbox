// Copyright 2026 Google LLC
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

package alloydb

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

func TestAlloyDBListTools(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	toolsFile := getAlloyDBToolsConfig()

	// Start the toolbox server
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile)
	if err != nil {
		t.Fatalf("command initialization returned an error: %v", err)
	}
	defer cleanup()

	waitCtx, cancelWait := context.WithTimeout(ctx, 20*time.Second)
	defer cancelWait()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %v", err)
	}

	// Verify list of tools
	expectedTools := []tests.MCPToolManifest{
		{
			Name:        "alloydb-list-clusters",
			Description: "Lists all AlloyDB clusters in a given project and location.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project": map[string]any{
						"description": "The GCP project ID to list clusters for.",
						"type":        "string",
					},
					"location": map[string]any{
						"default":     "-",
						"description": "Optional: The location to list clusters in (e.g., 'us-central1'). Use '-' to list clusters across all locations.(Default: '-')",
						"type":        "string",
					},
				},
				"required": []any{"project"},
			},
		},
		{
			Name:        "alloydb-list-users",
			Description: "Lists all AlloyDB users within a specific cluster.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cluster": map[string]any{
						"description": "The ID of the cluster to list users from.",
						"type":        "string",
					},
					"location": map[string]any{
						"description": "The location of the cluster (e.g., 'us-central1').",
						"type":        "string",
					},
					"project": map[string]any{
						"description": "The GCP project ID.",
						"type":        "string",
					},
				},
				"required": []any{"project", "location", "cluster"},
			},
		},
	}

	tests.RunMCPToolsListMethod(t, expectedTools)
}

func TestAlloyDBCallTool(t *testing.T) {
	vars := getAlloyDBVars(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	toolsFile := getAlloyDBToolsConfig()

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile)
	if err != nil {
		t.Fatalf("command initialization returned an error: %v", err)
	}
	defer cleanup()

	waitCtx, cancelWait := context.WithTimeout(ctx, 20*time.Second)
	defer cancelWait()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %v", err)
	}

	// Test calling "alloydb-list-clusters"
	args := map[string]any{
		"project":  vars["project"],
		"location": vars["location"],
	}

	wantContains := fmt.Sprintf(`"name":"projects/%s/locations/%s/clusters/%s"`, vars["project"], vars["location"], vars["cluster"])

	tests.RunMCPCustomToolCallMethod(t, "alloydb-list-clusters", args, wantContains)

	// Negative cases from legacy runAlloyDBMCPToolCallMethod
	t.Run("MCP Invoke my-fail-tool missing project", func(t *testing.T) {
		statusCode, mcpResp, err := tests.InvokeMCPTool(t, "my-fail-tool", map[string]any{"location": vars["location"]}, nil)
		if err != nil {
			t.Fatalf("native error executing %s: %s", "my-fail-tool", err)
		}
		if statusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", statusCode)
		}
		tests.AssertMCPError(t, mcpResp, `parameter "project" is required`)
	})

	t.Run("MCP Invoke invalid tool", func(t *testing.T) {
		statusCode, mcpResp, err := tests.InvokeMCPTool(t, "non-existent-tool", map[string]any{}, nil)
		if err != nil {
			t.Fatalf("native error executing %s: %s", "non-existent-tool", err)
		}
		if statusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", statusCode)
		}
		tests.AssertMCPError(t, mcpResp, `tool with name "non-existent-tool" does not exist`)
	})

	t.Run("MCP Invoke tool without required parameters", func(t *testing.T) {
		statusCode, mcpResp, err := tests.InvokeMCPTool(t, "alloydb-list-clusters", map[string]any{"location": vars["location"]}, nil)
		if err != nil {
			t.Fatalf("native error executing %s: %s", "alloydb-list-clusters", err)
		}
		if statusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", statusCode)
		}
		tests.AssertMCPError(t, mcpResp, `parameter "project" is required`)
	})
}

func TestAlloyDBListClusters(t *testing.T) {
	vars := getAlloyDBVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	toolsFile := getAlloyDBToolsConfig()

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile)
	if err != nil {
		t.Fatalf("command initialization returned an error: %v", err)
	}
	defer cleanup()

	waitCtx, cancelWait := context.WithTimeout(ctx, 20*time.Second)
	defer cancelWait()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %v", err)
	}

	wantForAllLocations := []string{
		fmt.Sprintf("projects/%s/locations/us-central1/clusters/alloydb-ai-nl-testing", vars["project"]),
		fmt.Sprintf("projects/%s/locations/us-central1/clusters/alloydb-pg-testing", vars["project"]),
	}

	t.Run("list clusters for all locations", func(t *testing.T) {
		args := map[string]any{"project": vars["project"], "location": "-"}
		statusCode, mcpResp, err := tests.InvokeMCPTool(t, "alloydb-list-clusters", args, nil)
		if err != nil {
			t.Fatalf("native error executing: %s", err)
		}
		if statusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", statusCode)
		}
		if mcpResp.Result.IsError {
			t.Fatalf("returned error result: %v", mcpResp.Result)
		}

		got := mcpResp.Result.Content[0].Text
		for _, want := range wantForAllLocations {
			if !regexp.MustCompile(want).MatchString(got) {
				t.Errorf("Expected substring not found: %q", want)
			}
		}
	})

	t.Run("list clusters missing project", func(t *testing.T) {
		args := map[string]any{"location": vars["location"]}
		statusCode, mcpResp, err := tests.InvokeMCPTool(t, "alloydb-list-clusters", args, nil)
		if err != nil {
			t.Fatalf("native error executing: %s", err)
		}
		if statusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", statusCode)
		}
		tests.AssertMCPError(t, mcpResp, `parameter "project" is required`)
	})
}
