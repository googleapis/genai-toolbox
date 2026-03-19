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
	"strings"

	"github.com/googleapis/genai-toolbox/cmd/internal"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/server/resources"
	"github.com/googleapis/genai-toolbox/internal/tools"

	"github.com/spf13/cobra"
)

// skillsCmd is the command for generating skills.
type skillsCmd struct {
	*cobra.Command
	name            string
	description     string
	toolset         string
	outputDir       string
	licenseHeader   string
	additionalNotes string
}

// NewCommand creates a new Command.
func NewCommand(opts *internal.ToolboxOptions) *cobra.Command {
	cmd := &skillsCmd{}
	cmd.Command = &cobra.Command{
		Use:   "skills-generate",
		Short: "Generate skills from tool configurations",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			return run(cmd, opts)
		},
	}

	flags := cmd.Flags()
	internal.ConfigFileFlags(flags, opts)
	flags.StringVar(&cmd.name, "name", "", "Name of the generated skill.")
	flags.StringVar(&cmd.description, "description", "", "Description of the generated skill")
	flags.StringVar(&cmd.toolset, "toolset", "", "Name of the toolset to convert into a skill. If not provided, all tools will be included.")
	flags.StringVar(&cmd.outputDir, "output-dir", "skills", "Directory to output generated skills")
	flags.StringVar(&cmd.licenseHeader, "license-header", "", "Optional license header to prepend to generated node scripts.")
	flags.StringVar(&cmd.additionalNotes, "additional-notes", "", "Additional notes to add under the Usage section of the generated SKILL.md")
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

	parser := internal.ToolsFileParser{}
	_, err = opts.LoadConfig(ctx, &parser)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cmd.outputDir, 0755); err != nil {
		errMsg := fmt.Errorf("error creating output directory: %w", err)
		opts.Logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	opts.Logger.InfoContext(ctx, "Generating skillagent skills...")

	// Group the collected tools by toolset they belong to
	skillsToTools, err := cmd.collectTools(ctx, opts)
	if err != nil {
		errMsg := fmt.Errorf("error collecting skill tools: %w", err)
		opts.Logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	if len(skillsToTools) == 0 {
		opts.Logger.InfoContext(ctx, "No tools found to generate.")
		return nil
	}

	// Iterate over keys to ensure deterministic order
	var skillNames []string
	for name := range skillsToTools {
		skillNames = append(skillNames, name)
	}
	sort.Strings(skillNames)

	for _, skillName := range skillNames {
		allTools := skillsToTools[skillName]
		if len(allTools) == 0 {
			opts.Logger.InfoContext(ctx, fmt.Sprintf("No tools found for skill '%s', skipping.", skillName))
			continue
		}

		// Generate the combined skill directory
		skillPath := filepath.Join(cmd.outputDir, skillName)
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

		var jsConfigArgs []string
		if len(opts.PrebuiltConfigs) > 0 {
			for _, pc := range opts.PrebuiltConfigs {
				jsConfigArgs = append(jsConfigArgs, `"--prebuilt"`, fmt.Sprintf(`"%s"`, pc))
			}
		}

		if opts.ToolsFolder != "" {
			folderName := filepath.Base(opts.ToolsFolder)
			destFolder := filepath.Join(assetsPath, folderName)
			if err := copyDir(opts.ToolsFolder, destFolder); err != nil {
				return err
			}
			jsConfigArgs = append(jsConfigArgs, `"--tools-folder"`, fmt.Sprintf(`path.join(__dirname, "..", "assets", %q)`, folderName))
		} else if len(opts.ToolsFiles) > 0 {
			for _, f := range opts.ToolsFiles {
				baseName := filepath.Base(f)
				destPath := filepath.Join(assetsPath, baseName)
				if err := copyFile(f, destPath); err != nil {
					return err
				}
				jsConfigArgs = append(jsConfigArgs, `"--tools-files"`, fmt.Sprintf(`path.join(__dirname, "..", "assets", %q)`, baseName))
			}
		} else if opts.ToolsFile != "" {
			baseName := filepath.Base(opts.ToolsFile)
			destPath := filepath.Join(assetsPath, baseName)
			if err := copyFile(opts.ToolsFile, destPath); err != nil {
				return err
			}
			jsConfigArgs = append(jsConfigArgs, `"--tools-file"`, fmt.Sprintf(`path.join(__dirname, "..", "assets", %q)`, baseName))
		}

		configArgsStr := strings.Join(jsConfigArgs, ", ")

		// Iterate over keys to ensure deterministic order
		var toolNames []string
		for name := range allTools {
			toolNames = append(toolNames, name)
		}
		sort.Strings(toolNames)

		for _, toolName := range toolNames {
			// Generate wrapper script in scripts directory
			scriptContent, err := generateScriptContent(toolName, configArgsStr, cmd.licenseHeader)
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
		skillContent, err := generateSkillMarkdown(skillName, cmd.description, cmd.additionalNotes, allTools, parser.EnvVars)
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

		opts.Logger.InfoContext(ctx, fmt.Sprintf("Successfully generated skill '%s' with %d tools.", skillName, len(allTools)))
	}

	return nil
}

func (c *skillsCmd) collectTools(ctx context.Context, opts *internal.ToolboxOptions) (map[string]map[string]tools.Tool, error) {
	// Initialize Resources
	sourcesMap, authServicesMap, embeddingModelsMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap, err := server.InitializeConfigs(ctx, opts.Cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize resources: %w", err)
	}

	resourceMgr := resources.NewResourceManager(sourcesMap, authServicesMap, embeddingModelsMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap)

	skillsToTools := make(map[string]map[string]tools.Tool)

	getToolsFromToolset := func(ts tools.Toolset) map[string]tools.Tool {
		toolsetTools := make(map[string]tools.Tool)
		for _, t := range ts.Tools {
			if t != nil {
				tool := *t
				toolsetTools[tool.McpManifest().Name] = tool
			}
		}
		return toolsetTools
	}

	if c.toolset != "" {
		ts, ok := resourceMgr.GetToolset(c.toolset)
		if !ok {
			return nil, fmt.Errorf("toolset %q not found", c.toolset)
		}

		skillsToTools[c.name] = getToolsFromToolset(ts)
		return skillsToTools, nil
	}

	if len(toolsetsMap) <= 1 {
		// Default to all tools if no toolset found
		skillsToTools[c.name] = toolsMap
		return skillsToTools, nil
	}

	// One skill per toolset
	for tsName, ts := range toolsetsMap {
		if tsName == "" {
			continue
		}
		skillName := fmt.Sprintf("%s-%s", c.name, tsName)
		skillsToTools[skillName] = getToolsFromToolset(ts)
	}

	return skillsToTools, nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}
		return copyFile(path, destPath)
	})
}
