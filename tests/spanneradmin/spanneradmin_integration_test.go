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

package spanneradmin

import (
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

	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	SpannerProject = os.Getenv("SPANNER_PROJECT")
)

func getSpannerAdminVars(t *testing.T) map[string]any {
	if SpannerProject == "" {
		t.Fatal("'SPANNER_PROJECT' not set")
	}

	return map[string]any{
		"kind":           "spanner-admin",
		"defaultProject": SpannerProject,
	}
}

func TestSpannerAdminCreateInstance(t *testing.T) {
	sourceConfig := getSpannerAdminVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	shortUuid := strings.ReplaceAll(uuid.New().String(), "-", "")[:10]
	instanceId := "test-inst-" + shortUuid

	displayName := "Test Instance " + shortUuid
	instanceConfig := "regional-us-central1"
	nodeCount := 1
	edition := "ENTERPRISE"

	// Setup Admin Client for verification and cleanup
	adminClient, err := instance.NewInstanceAdminClient(ctx)
	if err != nil {
		t.Fatalf("unable to create Spanner instance admin client: %s", err)
	}
	defer adminClient.Close()

	// Teardown function
	defer func() {
		err := adminClient.DeleteInstance(ctx, &instancepb.DeleteInstanceRequest{
			Name: fmt.Sprintf("projects/%s/instances/%s", SpannerProject, instanceId),
		})
		if err != nil {
			// If it fails, it might not have been created, log it but don't fail if it's "not found"
			t.Logf("cleanup: failed to delete instance %s: %s", instanceId, err)
		} else {
			t.Logf("cleanup: deleted instance %s", instanceId)
		}
	}()

	// Construct Tools Config

	toolsConfig := map[string]any{
		"sources": map[string]any{
			"my-spanner-admin": sourceConfig,
		},
		"tools": map[string]any{
			"create-instance-tool": map[string]any{
				"kind":        "spanner-create-instance",
				"source":      "my-spanner-admin",
				"description": "Creates a Spanner instance.",
			},
		},
	}

	// Start Toolbox Server
	cmd, cleanup, err := tests.StartCmd(ctx, toolsConfig)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancelWait := context.WithTimeout(ctx, 10*time.Second)
	defer cancelWait()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	// Prepare Invocation Payload

	payload := map[string]any{
		"project":         SpannerProject,
		"instanceId":      instanceId,
		"displayName":     displayName,
		"config":          instanceConfig,
		"nodeCount":       nodeCount,
		"edition":         edition,
		"processingUnits": 0,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %s", err)
	}

	// Invoke Tool
	invokeUrl := "http://127.0.0.1:5000/api/tool/create-instance-tool/invoke"
	req, err := http.NewRequest(http.MethodPost, invokeUrl, bytes.NewBuffer(payloadBytes))
	if err != nil {
		t.Fatalf("unable to create request: %s", err)
	}
	req.Header.Add("Content-type", "application/json")

	t.Logf("Invoking create-instance-tool for instance: %s", instanceId)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unable to send request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Check Response
	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		t.Fatalf("error parsing response body")
	}

	// Verify Instance Exists via Admin Client
	t.Logf("Verifying instance %s exists...", instanceId)
	instanceName := fmt.Sprintf("projects/%s/instances/%s", SpannerProject, instanceId)
	gotInstance, err := adminClient.GetInstance(ctx, &instancepb.GetInstanceRequest{
		Name: instanceName,
	})
	if err != nil {
		t.Fatalf("failed to get instance from admin client: %s", err)
	}

	if gotInstance.Name != instanceName {
		t.Errorf("expected instance name %s, got %s", instanceName, gotInstance.Name)
	}
	if gotInstance.DisplayName != displayName {
		t.Errorf("expected display name %s, got %s", displayName, gotInstance.DisplayName)
	}
	if gotInstance.NodeCount != int32(nodeCount) {
		t.Errorf("expected node count %d, got %d", nodeCount, gotInstance.NodeCount)
	}
}
