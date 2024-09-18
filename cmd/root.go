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

package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	// versionString indicates the version of this library.
	//go:embed version.txt
	versionString string
	// metadataString indicates additional build or distribution metadata.
	metadataString string
)

func init() {
	versionString = semanticVersion()
}

// semanticVersion returns the version of the CLI including a compile-time metadata.
func semanticVersion() string {
	v := strings.TrimSpace(versionString)
	if metadataString != "" {
		v += "+" + metadataString
	}
	return v
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := NewCommand().Execute(); err != nil {
		exit := 1
		os.Exit(exit)
	}
}

// Command represents an invocation of the CLI.
type Command struct {
	*cobra.Command

	cfg        server.Config
	tools_file string
}

// NewCommand returns a Command object representing an invocation of the CLI.
func NewCommand() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "toolbox",
			Version: versionString,
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&cmd.cfg.Address, "address", "a", "127.0.0.1", "Address of the interface the server will listen on.")
	flags.IntVarP(&cmd.cfg.Port, "port", "p", 5000, "Port the server will listen on.")

	flags.StringVar(&cmd.tools_file, "tools_file", "tools.yaml", "File path specifying the tool configuration")

	// wrap RunE command so that we have access to original Command object
	cmd.RunE = func(*cobra.Command, []string) error { return run(cmd) }

	return cmd
}

// parseToolsFile parses the provided yaml into appropriate configs.
func parseToolsFile(raw []byte) (sources.Configs, tools.Configs, error) {
	tools_file := &struct {
		Sources sources.Configs `yaml:"sources"`
		Tools   tools.Configs   `yaml:"tools"`
	}{}
	// Parse contents
	err := yaml.Unmarshal(raw, tools_file)
	if err != nil {
		return nil, nil, err
	}
	return tools_file.Sources, tools_file.Tools, nil
}

func run(cmd *Command) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// Read tool file contents
	buf, err := os.ReadFile(cmd.tools_file)
	if err != nil {
		return fmt.Errorf("Unable to read tool file at %q: %w", cmd.tools_file, err)
	}
	cmd.cfg.SourceConfigs, cmd.cfg.ToolConfigs, err = parseToolsFile(buf)
	if err != nil {
		return fmt.Errorf("Unable to parse tool file at %q: %w", cmd.tools_file, err)
	}

	// run server
	s, err := server.NewServer(cmd.cfg)
	if err != nil {
		return fmt.Errorf("Toolbox failed to start with the following error: %w", err)
	}
	err = s.ListenAndServe(ctx)
	if err != nil {
		return fmt.Errorf("Toolbox crashed with the following error: %w", err)
	}

	return nil
}
