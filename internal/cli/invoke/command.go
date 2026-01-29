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

package invoke

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/server/resources"
	"github.com/spf13/cobra"
)

// Dependencies defines the runtime dependencies for the invoke command.
type Dependencies struct {
	// Cfg points to the server configuration
	Cfg *server.ServerConfig
	// Out is the stdout writer
	Out io.Writer
	// Err is the stderr writer
	Err io.Writer
	// LoadConfig is a function that loads and merges configuration
	LoadConfig func(context.Context) error
	// Setup initializes the environment (logger, telemetry, etc.)
	Setup func(context.Context) (context.Context, func(context.Context) error, error)
	// Version is the application version string
	Version string
}

func NewCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoke <tool-name> [params]",
		Short: "Execute a tool directly",
		Long: `Execute a tool directly with parameters.
Params must be a JSON string.
Example:
  toolbox invoke my-tool '{"param1": "value1"}'`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runInvoke(c, args, deps)
		},
	}
	return cmd
}

func runInvoke(cmd *cobra.Command, args []string, deps Dependencies) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	ctx, shutdown, err := deps.Setup(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = shutdown(ctx)
	}()

	// Load and merge tool configurations
	if err := deps.LoadConfig(ctx); err != nil {
		return err
	}

	// Initialize Resources
	sourcesMap, authServicesMap, embeddingModelsMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap, err := server.InitializeConfigs(ctx, *deps.Cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize resources: %w", err)
	}

	resourceMgr := resources.NewResourceManager(sourcesMap, authServicesMap, embeddingModelsMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap)

	// Execute Tool
	toolName := args[0]
	tool, ok := resourceMgr.GetTool(toolName)
	if !ok {
		return fmt.Errorf("tool %q not found", toolName)
	}

	var paramsInput string
	if len(args) > 1 {
		paramsInput = args[1]
	}

	params := make(map[string]any)
	if paramsInput != "" {
		if err := json.Unmarshal([]byte(paramsInput), &params); err != nil {
			return fmt.Errorf("params must be a valid JSON string: %w", err)
		}
	}

	parsedParams, err := tool.ParseParams(params, nil)
	if err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}

	// Client Auth not supported for ephemeral CLI call
	requiresAuth, err := tool.RequiresClientAuthorization(resourceMgr)
	if err != nil {
		return fmt.Errorf("failed to check auth requirements: %w", err)
	}
	if requiresAuth {
		return fmt.Errorf("client authorization is not supported")
	}

	result, err := tool.Invoke(ctx, resourceMgr, parsedParams, "")
	if err != nil {
		return fmt.Errorf("tool execution failed: %w", err)
	}

	// Print Result
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	fmt.Fprintln(deps.Out, string(output))

	return nil
}
