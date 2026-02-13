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

package skills

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/googleapis/genai-toolbox/cmd/internal"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/server/resources"
	"github.com/googleapis/genai-toolbox/internal/tools"

	"github.com/spf13/cobra"
)

// skillsCmd is the command for generating skills.
type skillsCmd struct {
	*cobra.Command
	name        string
	description string
	toolset     string
	outputDir   string
}

// NewCommand creates a new Command.
func NewCommand(opts *internal.ToolboxOptions) *cobra.Command {
	cmd := &skillsCmd{}
	cmd.Command = &cobra.Command{
		Use:   "skills-generate",
		Short: "Generate skills from tool configurations",
		RunE: func(c *cobra.Command, args []string) error {
			return run(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&cmd.name, "name", "", "Name of the generated skill.")
	cmd.Flags().StringVar(&cmd.description, "description", "", "Description of the generated skill")
	cmd.Flags().StringVar(&cmd.toolset, "toolset", "", "Name of the toolset to convert into a skill. If not provided, all tools will be included.")
	cmd.Flags().StringVar(&cmd.outputDir, "output-dir", "skills", "Directory to output generated skills")

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("description")
	return cmd.Command
}

func run(cmd *skillsCmd, opts *internal.ToolboxOptions) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	ctx, shutdown, err := opts.Setup(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = shutdown(ctx)
	}()

	_, err = opts.LoadConfig(ctx)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cmd.outputDir, 0755); err != nil {
		errMsg := fmt.Errorf("error creating output directory: %w", err)
		opts.Logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	opts.Logger.InfoContext(ctx, fmt.Sprintf("Generating skill '%s'...", cmd.name))

	// Initialize toolbox and collect tools
	allTools, err := cmd.collectTools(ctx, opts)
	if err != nil {
		errMsg := fmt.Errorf("error collecting tools: %w", err)
		opts.Logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	if len(allTools) == 0 {
		opts.Logger.InfoContext(ctx, "No tools found to generate.")
		return nil
	}

	// Generate the combined skill directory
	skillPath := filepath.Join(cmd.outputDir, cmd.name)
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		errMsg := fmt.Errorf("error creating skill directory: %w", err)
		opts.Logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	// Generate assets directory
	assetsPath := filepath.Join(skillPath, "assets")
	if err := os.MkdirAll(assetsPath, 0755); err != nil {
		errMsg := fmt.Errorf("error creating assets dir: %w", err)
		opts.Logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	// Generate scripts directory
	scriptsPath := filepath.Join(skillPath, "scripts")
	if err := os.MkdirAll(scriptsPath, 0755); err != nil {
		errMsg := fmt.Errorf("error creating scripts dir: %w", err)
		opts.Logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	// Iterate over keys to ensure deterministic order
	var toolNames []string
	for name := range allTools {
		toolNames = append(toolNames, name)
	}
	sort.Strings(toolNames)

	for _, toolName := range toolNames {
		// Generate YAML config in asset directory
		minimizedContent, err := generateToolConfigYAML(opts.Cfg, toolName)
		if err != nil {
			errMsg := fmt.Errorf("error generating filtered config for %s: %w", toolName, err)
			opts.Logger.ErrorContext(ctx, errMsg.Error())
			return errMsg
		}

		specificToolsFileName := fmt.Sprintf("%s.yaml", toolName)
		if minimizedContent != nil {
			destPath := filepath.Join(assetsPath, specificToolsFileName)
			if err := os.WriteFile(destPath, minimizedContent, 0644); err != nil {
				errMsg := fmt.Errorf("error writing filtered config for %s: %w", toolName, err)
				opts.Logger.ErrorContext(ctx, errMsg.Error())
				return errMsg
			}
		}

		// Generate wrapper script in scripts directory
		scriptContent, err := generateScriptContent(toolName, specificToolsFileName)
		if err != nil {
			errMsg := fmt.Errorf("error generating script content for %s: %w", toolName, err)
			opts.Logger.ErrorContext(ctx, errMsg.Error())
			return errMsg
		}

		scriptFilename := filepath.Join(scriptsPath, fmt.Sprintf("%s.js", toolName))
		if err := os.WriteFile(scriptFilename, []byte(scriptContent), 0755); err != nil {
			errMsg := fmt.Errorf("error writing script %s: %w", scriptFilename, err)
			opts.Logger.ErrorContext(ctx, errMsg.Error())
			return errMsg
		}
	}

	// Generate SKILL.md
	skillContent, err := generateSkillMarkdown(cmd.name, cmd.description, allTools)
	if err != nil {
		errMsg := fmt.Errorf("error generating SKILL.md content: %w", err)
		opts.Logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}
	skillMdPath := filepath.Join(skillPath, "SKILL.md")
	if err := os.WriteFile(skillMdPath, []byte(skillContent), 0644); err != nil {
		errMsg := fmt.Errorf("error writing SKILL.md: %w", err)
		opts.Logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	opts.Logger.InfoContext(ctx, fmt.Sprintf("Successfully generated skill '%s' with %d tools.", cmd.name, len(allTools)))

	return nil
}

func (c *skillsCmd) collectTools(ctx context.Context, opts *internal.ToolboxOptions) (map[string]tools.Tool, error) {
	// Initialize Resources
	sourcesMap, authServicesMap, embeddingModelsMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap, err := server.InitializeConfigs(ctx, opts.Cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize resources: %w", err)
	}

	resourceMgr := resources.NewResourceManager(sourcesMap, authServicesMap, embeddingModelsMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap)

	result := make(map[string]tools.Tool)

	if c.toolset == "" {
		return toolsMap, nil
	}

	ts, ok := resourceMgr.GetToolset(c.toolset)
	if !ok {
		return nil, fmt.Errorf("toolset %q not found", c.toolset)
	}

	for _, t := range ts.Tools {
		if t != nil {
			tool := *t
			result[tool.McpManifest().Name] = tool
		}
	}

	return result, nil
}
