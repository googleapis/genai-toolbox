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

	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/server/resources"

	"github.com/spf13/cobra"
)

// RootCommand defines the interface for required by skills-generate subcommand.
// This allows subcommands to access shared resources and functionality without
// direct coupling to the root command's implementation.
type RootCommand interface {
	// Config returns a copy of the current server configuration.
	Config() server.ServerConfig

	// LoadConfig loads and merges the configuration from files, folders, and prebuilts.
	LoadConfig(ctx context.Context) error

	// Setup initializes the runtime environment, including logging and telemetry.
	// It returns the updated context and a shutdown function to be called when finished.
	Setup(ctx context.Context) (context.Context, func(context.Context) error, error)

	// Logger returns the logger instance.
	Logger() log.Logger
}

// Command is the command for generating skills.
type Command struct {
	*cobra.Command
	rootCmd     RootCommand
	name        string
	description string
	toolset     string
	outputDir   string
}

// Parameter represents a parameter of a tool.
type Parameter struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
	Default     interface{} `json:"default"`
	Required    bool        `json:"required"`
}

// Tool represents a tool.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  []Parameter `json:"parameters"`
}

// Config represents the structure of the tools.yaml file.
type Config struct {
	Sources map[string]interface{}            `yaml:"sources,omitempty"`
	Tools   map[string]map[string]interface{} `yaml:"tools"`
}

// serverConfig holds the configuration used to start the toolbox server.
type serverConfig struct {
	prebuiltConfigs []string
	toolsFile       string
}

// NewCommand creates a new Command.
func NewCommand(rootCmd RootCommand) *cobra.Command {
	cmd := &Command{
		rootCmd: rootCmd,
	}
	cmd.Command = &cobra.Command{
		Use:   "skills-generate",
		Short: "Generate skills from tool configurations",
		RunE: func(c *cobra.Command, args []string) error {
			return cmd.run(c)
		},
	}

	cmd.Flags().StringVar(&cmd.name, "name", "", "Name of the generated skill.")
	cmd.Flags().StringVar(&cmd.description, "description", "", "Description of the generated skill")
	cmd.Flags().StringVar(&cmd.toolset, "toolset", "", "Name of the toolset (and generated skill folder). If provided, only tools in this toolset are generated.")
	cmd.Flags().StringVar(&cmd.outputDir, "output-dir", "skills", "Directory to output generated skills")

	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("description")
	return cmd.Command
}

func (c *Command) run(cmd *cobra.Command) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	ctx, shutdown, err := c.rootCmd.Setup(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = shutdown(ctx)
	}()

	logger := c.rootCmd.Logger()

	toolsFile, err := cmd.Flags().GetString("tools-file")
	if err != nil {
		errMsg := fmt.Errorf("error getting tools-file flag: %w", err)
		logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}
	prebuiltConfigs, err := cmd.Flags().GetStringSlice("prebuilt")
	if err != nil {
		errMsg := fmt.Errorf("error getting prebuilt flag: %w", err)
		logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	// Load and merge tool configurations
	if err := c.rootCmd.LoadConfig(ctx); err != nil {
		return err
	}

	if len(prebuiltConfigs) == 0 && toolsFile == "" {
		logger.InfoContext(ctx, "No configurations found to process. Use --tools-file or --prebuilt.")
		return nil
	}
	if err := os.MkdirAll(c.outputDir, 0755); err != nil {
		errMsg := fmt.Errorf("error creating output directory: %w", err)
		logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	logger.InfoContext(ctx, fmt.Sprintf("Generating skill '%s'...", c.name))

	config := serverConfig{
		prebuiltConfigs: prebuiltConfigs,
		toolsFile:       toolsFile,
	}

	// Initialize toolbox and collect tools
	allTools, err := c.collectTools(ctx)
	if err != nil {
		errMsg := fmt.Errorf("error collecting tools: %w", err)
		logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	if len(allTools) == 0 {
		logger.InfoContext(ctx, "No tools found to generate.")
		return nil
	}

	// Generate the combined skill
	skillPath := filepath.Join(c.outputDir, c.name)
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		errMsg := fmt.Errorf("error creating skill directory: %w", err)
		logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	// Generate assets directory if needed
	assetsPath := filepath.Join(skillPath, "assets")
	if toolsFile != "" {
		if err := os.MkdirAll(assetsPath, 0755); err != nil {
			errMsg := fmt.Errorf("error creating assets dir: %w", err)
			logger.ErrorContext(ctx, errMsg.Error())
			return errMsg
		}
	}

	// Generate scripts
	scriptsPath := filepath.Join(skillPath, "scripts")
	if err := os.MkdirAll(scriptsPath, 0755); err != nil {
		errMsg := fmt.Errorf("error creating scripts dir: %w", err)
		logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	for _, tool := range allTools {
		specificToolsFileName := ""
		if toolsFile != "" {
			minimizedContent, err := generateFilteredConfig(toolsFile, tool.Name)
			if err != nil {
				logger.ErrorContext(ctx, fmt.Sprintf("Error generating filtered config for %s: %v", tool.Name, err))
			}

			if minimizedContent != nil {
				specificToolsFileName = fmt.Sprintf("%s.yaml", tool.Name)
				destPath := filepath.Join(assetsPath, specificToolsFileName)
				if err := os.WriteFile(destPath, minimizedContent, 0644); err != nil {
					logger.ErrorContext(ctx, fmt.Sprintf("Error writing filtered config for %s: %v", tool.Name, err))
				}
			}
		}

		scriptContent, err := generateScriptContent(tool.Name, config, specificToolsFileName)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Error generating script content for %s: %v", tool.Name, err))
		} else {
			scriptFilename := filepath.Join(scriptsPath, fmt.Sprintf("%s.js", tool.Name))
			if err := os.WriteFile(scriptFilename, []byte(scriptContent), 0755); err != nil {
				logger.ErrorContext(ctx, fmt.Sprintf("Error writing script %s: %v", scriptFilename, err))
			}
		}
	}

	// Generate SKILL.md
	skillContent, err := generateSkillMarkdown(c.name, c.description, allTools)
	if err != nil {
		errMsg := fmt.Errorf("error generating SKILL.md content: %w", err)
		logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}
	skillMdPath := filepath.Join(skillPath, "SKILL.md")
	if err := os.WriteFile(skillMdPath, []byte(skillContent), 0644); err != nil {
		errMsg := fmt.Errorf("error writing SKILL.md: %w", err)
		logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	logger.InfoContext(ctx, fmt.Sprintf("Successfully generated skill '%s' with %d tools.", c.name, len(allTools)))

	return nil
}

func (c *Command) collectTools(ctx context.Context) (map[string]Tool, error) {
	// Initialize Resources
	sourcesMap, authServicesMap, embeddingModelsMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap, err := server.InitializeConfigs(ctx, c.rootCmd.Config())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize resources: %w", err)
	}

	resourceMgr := resources.NewResourceManager(sourcesMap, authServicesMap, embeddingModelsMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap)

	result := make(map[string]Tool)

	var toolsToProcess []string

	if c.toolset != "" {
		ts, ok := resourceMgr.GetToolset(c.toolset)
		if !ok {
			return nil, fmt.Errorf("toolset %q not found", c.toolset)
		}
		toolsToProcess = ts.ToolNames
	} else {
		// All tools
		for name := range toolsMap {
			toolsToProcess = append(toolsToProcess, name)
		}
	}

	for _, toolName := range toolsToProcess {
		t, ok := resourceMgr.GetTool(toolName)
		if !ok {
			// Should happen only if toolset refers to non-existent tool, but good to check
			continue
		}

		params := []Parameter{}
		for _, p := range t.GetParameters() {
			manifest := p.Manifest()
			params = append(params, Parameter{
				Name:        p.GetName(),
				Description: manifest.Description, // Use description from manifest
				Type:        p.GetType(),
				Default:     p.GetDefault(),
				Required:    p.GetRequired(),
			})
		}

		manifest := t.Manifest()
		result[toolName] = Tool{
			Name:        toolName,
			Description: manifest.Description,
			Parameters:  params,
		}
	}

	return result, nil
}
