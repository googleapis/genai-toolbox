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

package serve

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/cmd/internal"
	"github.com/spf13/cobra"
)

func serveCommand(ctx context.Context, args []string) (string, error) {
	parentCmd := &cobra.Command{Use: "toolbox"}

	buf := new(bytes.Buffer)
	opts := internal.NewToolboxOptions(internal.WithIOStreams(buf, buf))
	internal.PersistentFlags(parentCmd, opts)

	cmd := NewCommand(opts)
	parentCmd.AddCommand(cmd)
	parentCmd.SetArgs(args)
	// Inject the context into the Cobra command
	parentCmd.SetContext(ctx)

	err := parentCmd.Execute()
	return buf.String(), err
}

func TestServe(t *testing.T) {
	// context will automatically shutdown in 1 second.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	args := []string{"serve", "--port", "0"}
	output, err := serveCommand(ctx, args)
	if err != nil {
		t.Fatalf("expected graceful shutdown without error, got: %v", err)
	}

	if !strings.Contains(output, "Server ready to serve!") {
		t.Errorf("expected to find server ready message in output, got: %s", output)
	}

	if !strings.Contains(output, "Shutting down gracefully...") {
		t.Errorf("expected to find graceful shutdown message in output, got: %s", output)
	}
}
