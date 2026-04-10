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

package internal

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/googleapis/mcp-toolbox/internal/log"
	"github.com/googleapis/mcp-toolbox/internal/prebuiltconfigs"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/telemetry"
	"github.com/googleapis/mcp-toolbox/internal/util"
)

type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

// ToolboxOptions holds dependencies shared by all commands.
type ToolboxOptions struct {
	IOStreams       IOStreams
	Logger          log.Logger
	Cfg             server.ServerConfig
	Config          string
	Configs         []string
	ConfigFolder    string
	PrebuiltConfigs []string
	VersionNum      string
}

// Option defines a function that modifies the ToolboxOptions struct.
type Option func(*ToolboxOptions)

// NewToolboxOptions creates a new instance with defaults, then applies any
// provided options.
func NewToolboxOptions(opts ...Option) *ToolboxOptions {
	o := &ToolboxOptions{
		IOStreams: IOStreams{
			In:     os.Stdin,
			Out:    os.Stdout,
			ErrOut: os.Stderr,
		},
	}

	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Apply allows you to update an EXISTING ToolboxOptions instance.
// This is useful for "late binding".
func (o *ToolboxOptions) Apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithIOStreams updates the IO streams.
func WithIOStreams(out, err io.Writer) Option {
	return func(o *ToolboxOptions) {
		o.IOStreams.Out = out
		o.IOStreams.ErrOut = err
	}
}

// Setup create logger and telemetry instrumentations.
func (opts *ToolboxOptions) Setup(ctx context.Context) (context.Context, func(context.Context) error, error) {
	// If stdio, set logger's out stream (usually DEBUG and INFO logs) to
	// errStream
	loggerOut := opts.IOStreams.Out
	if opts.Cfg.Stdio {
		loggerOut = opts.IOStreams.ErrOut
	}

	// Handle logger separately from config
	logger, err := log.NewLogger(opts.Cfg.LoggingFormat.String(), opts.Cfg.LogLevel.String(), loggerOut, opts.IOStreams.ErrOut)
	if err != nil {
		return ctx, nil, fmt.Errorf("unable to initialize logger: %w", err)
	}

	ctx = util.WithLogger(ctx, logger)
	opts.Logger = logger

	// Set up OpenTelemetry
	otelShutdown, err := telemetry.SetupOTel(ctx, opts.Cfg.Version, opts.Cfg.TelemetryOTLP, opts.Cfg.TelemetryGCP, opts.Cfg.TelemetryServiceName)
	if err != nil {
		errMsg := fmt.Errorf("error setting up OpenTelemetry: %w", err)
		logger.ErrorContext(ctx, errMsg.Error())
		return ctx, nil, errMsg
	}

	shutdownFunc := func(ctx context.Context) error {
		err := otelShutdown(ctx)
		if err != nil {
			errMsg := fmt.Errorf("error shutting down OpenTelemetry: %w", err)
			logger.ErrorContext(ctx, errMsg.Error())
			return err
		}
		return nil
	}

	instrumentation, err := telemetry.CreateTelemetryInstrumentation(opts.Cfg.Version)
	if err != nil {
		errMsg := fmt.Errorf("unable to create telemetry instrumentation: %w", err)
		logger.ErrorContext(ctx, errMsg.Error())
		return ctx, shutdownFunc, errMsg
	}

	ctx = util.WithInstrumentation(ctx, instrumentation)

	return ctx, shutdownFunc, nil
}

// GetCustomConfigFiles retrieves the list of custom config file paths
func (opts *ToolboxOptions) GetCustomConfigFiles(ctx context.Context) ([]string, bool, error) {
	// Determine if Custom Files should be loaded
	// Check for explicit custom flags
	isCustomConfigured := opts.Config != "" || len(opts.Configs) > 0 || opts.ConfigFolder != ""

	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, isCustomConfigured, err
	}

	// Load Custom Configurations
	if isCustomConfigured {
		// Enforce exclusivity among custom flags (tools-file vs tools-files vs tools-folder)
		if (opts.Config != "" && len(opts.Configs) > 0) ||
			(opts.Config != "" && opts.ConfigFolder != "") ||
			(len(opts.Configs) > 0 && opts.ConfigFolder != "") {
			errMsg := fmt.Errorf("--config/--tools-file, --configs/--tools-files, and --config-folder/--tools-folder flags cannot be used simultaneously")
			logger.ErrorContext(ctx, errMsg.Error())
			return nil, isCustomConfigured, errMsg
		}

		if len(opts.Configs) > 0 {
			// Use tools-files
			logger.InfoContext(ctx, fmt.Sprintf("retrieving %d tool configuration files", len(opts.Configs)))
			return opts.Configs, isCustomConfigured, nil
		} else if opts.ConfigFolder != "" {
			// Use tools-folder
			allFiles, err := GetPathsFromConfigFolder(ctx, opts.ConfigFolder)
			return allFiles, isCustomConfigured, err
		} else {
			// use tools-file
			return []string{opts.Config}, isCustomConfigured, nil
		}
	}

	// Determine if default 'tools.yaml' should be used (No prebuilt AND No custom flags)
	useDefaultConfig := len(opts.PrebuiltConfigs) == 0
	if useDefaultConfig {
		// else we will add the default path regardless
		return []string{"tools.yaml"}, true, nil
	}

	// no custom config files are found
	// server are likely using prebuilt configs
	return []string{}, false, nil
}

// LoadConfig checks and merge files that should be loaded into the server
func (opts *ToolboxOptions) LoadConfig(ctx context.Context, parser *ConfigParser) (bool, error) {
	// get all the file paths for custom config file
	filesPaths, isCustomConfigured, err := opts.GetCustomConfigFiles(ctx)
	if err != nil {
		return isCustomConfigured, err
	}

	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return isCustomConfigured, err
	}

	var allConfigs []Config

	// Load Prebuilt Configuration
	if len(opts.PrebuiltConfigs) > 0 {
		slices.Sort(opts.PrebuiltConfigs)
		sourcesList := strings.Join(opts.PrebuiltConfigs, ", ")
		logMsg := fmt.Sprintf("Using prebuilt tool configurations for: %s", sourcesList)
		logger.InfoContext(ctx, logMsg)

		for _, configName := range opts.PrebuiltConfigs {
			buf, err := prebuiltconfigs.Get(configName)
			if err != nil {
				logger.ErrorContext(ctx, err.Error())
				return isCustomConfigured, err
			}

			// Parse into Config struct
			parsed, err := parser.ParseConfig(ctx, buf)
			if err != nil {
				errMsg := fmt.Errorf("unable to parse prebuilt tool configuration for '%s': %w", configName, err)
				logger.ErrorContext(ctx, errMsg.Error())
				return isCustomConfigured, errMsg
			}
			allConfigs = append(allConfigs, parsed)
		}
	}

	// Load Custom Configurations
	if isCustomConfigured {
		customTools, err := parser.LoadAndMergeConfigs(ctx, filesPaths)
		if err != nil {
			logger.ErrorContext(ctx, err.Error())
			return isCustomConfigured, err
		}
		allConfigs = append(allConfigs, customTools)
	}

	// Modify version string based on loaded configurations
	if len(opts.PrebuiltConfigs) > 0 {
		tag := "prebuilt"
		if isCustomConfigured {
			tag = "custom"
		}
		// prebuiltConfigs is already sorted above
		for _, configName := range opts.PrebuiltConfigs {
			opts.Cfg.Version += fmt.Sprintf("+%s.%s", tag, configName)
		}
	}

	// Merge Everything
	// This will error if custom tools collide with prebuilt tools
	finalConfig, err := mergeConfigs(allConfigs...)
	if err != nil {
		logger.ErrorContext(ctx, err.Error())
		return isCustomConfigured, err
	}

	opts.Cfg.SourceConfigs = finalConfig.Sources
	opts.Cfg.AuthServiceConfigs = finalConfig.AuthServices
	opts.Cfg.EmbeddingModelConfigs = finalConfig.EmbeddingModels
	opts.Cfg.ToolConfigs = finalConfig.Tools
	opts.Cfg.ToolsetConfigs = finalConfig.Toolsets
	opts.Cfg.PromptConfigs = finalConfig.Prompts

	return isCustomConfigured, nil
}
