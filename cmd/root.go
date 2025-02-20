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
	"os/signal"
	"strings"
	"syscall"
	"time"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/telemetry"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/spf13/cobra"
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

	// Set server version
	cmd.cfg.Version = versionString

	// set baseCmd out and err the same as cmd.
	baseCmd.SetOut(cmd.outStream)
	baseCmd.SetErr(cmd.errStream)

	flags := cmd.Flags()
	flags.StringVarP(&cmd.cfg.Address, "address", "a", "127.0.0.1", "Address of the interface the server will listen on.")
	flags.IntVarP(&cmd.cfg.Port, "port", "p", 5000, "Port the server will listen on.")

	flags.StringVar(&cmd.tools_file, "tools_file", "tools.yaml", "File path specifying the tool configuration.")
	flags.Var(&cmd.cfg.LogLevel, "log-level", "Specify the minimum level logged. Allowed: 'DEBUG', 'INFO', 'WARN', 'ERROR'.")
	flags.Var(&cmd.cfg.LoggingFormat, "logging-format", "Specify logging format to use. Allowed: 'standard' or 'JSON'.")
	flags.BoolVar(&cmd.cfg.TelemetryGCP, "telemetry-gcp", false, "Enable exporting directly to Google Cloud Monitoring.")
	flags.StringVar(&cmd.cfg.TelemetryOTLP, "telemetry-otlp", "", "Enable exporting using OpenTelemetry Protocol (OTLP) to the specified endpoint (e.g. 'http://127.0.0.1:4318')")
	flags.StringVar(&cmd.cfg.TelemetryServiceName, "telemetry-service-name", "toolbox", "Sets the value of the service.name resource attribute for telemetry data.")

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
func parseToolsFile(ctx context.Context, raw []byte) (ToolsFile, error) {
	var toolsFile ToolsFile
	// Parse contents
	err := yaml.UnmarshalContext(ctx, raw, &toolsFile, yaml.Strict())
	if err != nil {
		return toolsFile, err
	}
	return toolsFile, nil
}

func run(cmd *Command) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// watch for sigterm / sigint signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	go func(sCtx context.Context) {
		var s os.Signal
		select {
		case <-sCtx.Done():
			// this should only happen when the context supplied when testing is canceled
			return
		case s = <-signals:
		}
		switch s {
		case syscall.SIGINT:
			cmd.logger.DebugContext(sCtx, "Received SIGINT signal to shutdown.")
		case syscall.SIGTERM:
			cmd.logger.DebugContext(sCtx, "Sending SIGTERM signal to shutdown.")
		}
		cancel()
	}(ctx)

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

	ctx = context.WithValue(ctx, util.LoggerKey, cmd.logger)

	// Set up OpenTelemetry
	otelShutdown, err := telemetry.SetupOTel(ctx, cmd.Command.Version, cmd.cfg.TelemetryOTLP, cmd.cfg.TelemetryGCP, cmd.cfg.TelemetryServiceName)
	if err != nil {
		errMsg := fmt.Errorf("error setting up OpenTelemetry: %w", err)
		cmd.logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}
	defer func() {
		err := otelShutdown(ctx)
		if err != nil {
			errMsg := fmt.Errorf("error shutting down OpenTelemetry: %w", err)
			cmd.logger.ErrorContext(ctx, errMsg.Error())
		}
	}()

	// Read tool file contents
	buf, err := os.ReadFile(cmd.tools_file)
	if err != nil {
		errMsg := fmt.Errorf("unable to read tool file at %q: %w", cmd.tools_file, err)
		cmd.logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}
	toolsFile, err := parseToolsFile(ctx, buf)
	cmd.cfg.SourceConfigs, cmd.cfg.AuthSourceConfigs, cmd.cfg.ToolConfigs, cmd.cfg.ToolsetConfigs = toolsFile.Sources, toolsFile.AuthSources, toolsFile.Tools, toolsFile.Toolsets
	if err != nil {
		errMsg := fmt.Errorf("unable to parse tool file at %q: %w", cmd.tools_file, err)
		cmd.logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	// start server
	s, err := server.NewServer(ctx, cmd.cfg, cmd.logger)
	if err != nil {
		errMsg := fmt.Errorf("toolbox failed to initialize: %w", err)
		cmd.logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	err = s.Listen(ctx)
	if err != nil {
		errMsg := fmt.Errorf("toolbox failed to start listener: %w", err)
		cmd.logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}
	cmd.logger.InfoContext(ctx, "Server ready to serve!")

	// run server in background
	srvErr := make(chan error)
	go func() {
		defer close(srvErr)
		err = s.Serve()
		if err != nil {
			srvErr <- err
		}
	}()

	// wait for either the server to error out or the command's context to be canceled
	select {
	case err := <-srvErr:
		if err != nil {
			errMsg := fmt.Errorf("toolbox crashed with the following error: %w", err)
			cmd.logger.ErrorContext(ctx, errMsg.Error())
			return errMsg
		}
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd.logger.WarnContext(shutdownContext, "Shutting down gracefully...")
		err := s.Shutdown(shutdownContext)
		if err == context.DeadlineExceeded {
			return fmt.Errorf("graceful shutdown timed out... forcing exit.")
		}
	}

	return nil
}
