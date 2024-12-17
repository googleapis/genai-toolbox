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
	"io"
	"os"
	"strings"

	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/opentel"
	"github.com/googleapis/genai-toolbox/internal/server"
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

	cfg        server.ServerConfig
	logger     log.Logger
	tools_file string
	outStream  io.Writer
	errStream  io.Writer
}

// NewCommand returns a Command object representing an invocation of the CLI.
func NewCommand(opts ...Option) *Command {
	out := os.Stdout
	err := os.Stderr

	baseCmd := &cobra.Command{
		Use:           "toolbox",
		Version:       versionString,
		SilenceErrors: true,
	}
	cmd := &Command{
		Command:   baseCmd,
		outStream: out,
		errStream: err,
	}

	for _, o := range opts {
		o(cmd)
	}

	// set baseCmd out and err the same as cmd.
	baseCmd.SetOut(cmd.outStream)
	baseCmd.SetErr(cmd.errStream)

	flags := cmd.Flags()
	flags.StringVarP(&cmd.cfg.Address, "address", "a", "127.0.0.1", "Address of the interface the server will listen on.")
	flags.IntVarP(&cmd.cfg.Port, "port", "p", 5000, "Port the server will listen on.")

	flags.StringVar(&cmd.tools_file, "tools_file", "tools.yaml", "File path specifying the tool configuration.")
	flags.Var(&cmd.cfg.LogLevel, "log-level", "Specify the minimum level logged. Allowed: 'DEBUG', 'INFO', 'WARN', 'ERROR'.")
	flags.Var(&cmd.cfg.LoggingFormat, "logging-format", "Specify logging format to use. Allowed: 'standard' or 'JSON'.")

	// wrap RunE command so that we have access to original Command object
	cmd.RunE = func(*cobra.Command, []string) error { return run(cmd) }

	return cmd
}

type ToolsFile struct {
	Sources     server.SourceConfigs     `yaml:"sources"`
	AuthSources server.AuthSourceConfigs `yaml:"authSources"`
	Tools       server.ToolConfigs       `yaml:"tools"`
	Toolsets    server.ToolsetConfigs    `yaml:"toolsets"`
}

// parseToolsFile parses the provided yaml into appropriate configs.
func parseToolsFile(raw []byte) (ToolsFile, error) {
	var toolsFile ToolsFile
	// Parse contents
	err := yaml.Unmarshal(raw, &toolsFile)
	if err != nil {
		return toolsFile, err
	}
	return toolsFile, nil
}

func run(cmd *Command) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// Handle logger separately from config
	switch strings.ToLower(cmd.cfg.LoggingFormat.String()) {
	case "json":
		logger, err := log.NewStructuredLogger(cmd.outStream, cmd.errStream, cmd.cfg.LogLevel.String())
		if err != nil {
			return fmt.Errorf("unable to initialize logger: %w", err)
		}
		cmd.logger = logger
	case "standard":
		logger, err := log.NewStdLogger(cmd.outStream, cmd.errStream, cmd.cfg.LogLevel.String())
		if err != nil {
			return fmt.Errorf("unable to initialize logger: %w", err)
		}
		cmd.logger = logger
	default:
		return fmt.Errorf("logging format invalid.")
	}

	// Set up OpenTelemetry
	otelShutdown, err := opentel.SetupOTelSDK(ctx, cmd.Command.Version)
	if err != nil {
		errMsg := fmt.Errorf("error setting up OpenTelemetry: %w", err)
		cmd.logger.Error(errMsg.Error())
		return errMsg
	}
	defer func() {
		err := otelShutdown(context.Background())
		if err != nil {
			errMsg := fmt.Errorf("error shutting down OpenTelemetry: %w", err)
			cmd.logger.Error(errMsg.Error())
		}
	}()

	// Read tool file contents
	buf, err := os.ReadFile(cmd.tools_file)
	if err != nil {
		errMsg := fmt.Errorf("unable to read tool file at %q: %w", cmd.tools_file, err)
		cmd.logger.Error(errMsg.Error())
		return errMsg
	}
	toolsFile, err := parseToolsFile(buf)
	cmd.cfg.SourceConfigs, cmd.cfg.AuthSourceConfigs, cmd.cfg.ToolConfigs, cmd.cfg.ToolsetConfigs = toolsFile.Sources, toolsFile.AuthSources, toolsFile.Tools, toolsFile.Toolsets
	if err != nil {
		errMsg := fmt.Errorf("unable to parse tool file at %q: %w", cmd.tools_file, err)
		cmd.logger.Error(errMsg.Error())
		return errMsg
	}

	// run server
	s, err := server.NewServer(cmd.cfg, cmd.logger)
	if err != nil {
		errMsg := fmt.Errorf("toolbox failed to start with the following error: %w", err)
		cmd.logger.Error(errMsg.Error())
		return errMsg
	}
	l, err := s.Listen(ctx)
	if err != nil {
		errMsg := fmt.Errorf("toolbox failed to mount listener: %w", err)
		cmd.logger.Error(errMsg.Error())
		return errMsg
	}
	cmd.logger.Info("Server ready to serve")
	err = s.Serve(l)
	if err != nil {
		errMsg := fmt.Errorf("toolbox crashed with the following error: %w", err)
		cmd.logger.Error(errMsg.Error())
		return errMsg
	}

	return nil
}
