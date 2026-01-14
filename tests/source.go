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

// Package tests contains end to end tests meant to verify the Toolbox Server
// works as expected when executed as a binary.

package tests

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

// RunSourceConnection test for source connection
func RunSourceConnectionTest(t *testing.T, sourceConfig map[string]any, toolType string) error {
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
				"kind":        toolType,
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
				"statement":   "SELECT 1;",
			},
		},
	}
	cmd, cleanup, err := StartCmd(ctx, toolsFile, args...)
	if err != nil {
		return fmt.Errorf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		return fmt.Errorf("toolbox didn't start successfully: %s", err)
	}
	return nil
}

// GetCloudSQLDialOpts returns cloud sql connector's dial option for ip type.
func GetCloudSQLDialOpts(ipType string) ([]cloudsqlconn.DialOption, error) {
	switch strings.ToLower(ipType) {
	case "private":
		return []cloudsqlconn.DialOption{cloudsqlconn.WithPrivateIP()}, nil
	case "public":
		return []cloudsqlconn.DialOption{cloudsqlconn.WithPublicIP()}, nil
	default:
		return nil, fmt.Errorf("invalid ipType %s", ipType)
	}
}
