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
	"maps"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	// Importing the cmd/internal package also import packages for side effect of registration
	"github.com/googleapis/genai-toolbox/cmd/internal"
	"github.com/googleapis/genai-toolbox/cmd/internal/invoke"
	"github.com/googleapis/genai-toolbox/cmd/internal/skills"
	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/spf13/cobra"
)

var (
	// versionString stores the full semantic version, including build metadata.
	versionString string
	// versionNum indicates the numerical part fo the version
	//go:embed version.txt
	versionNum string
	// metadataString indicates additional build or distribution metadata.
	buildType string = "dev" // should be one of "dev", "binary", or "container"
	// commitSha is the git commit it was built from
	commitSha string
)

func init() {
	versionString = semanticVersion()
}

// semanticVersion returns the version of the CLI including a compile-time metadata.
func semanticVersion() string {
	metadataStrings := []string{buildType, runtime.GOOS, runtime.GOARCH}
	if commitSha != "" {
		metadataStrings = append(metadataStrings, commitSha)
	}
	v := strings.TrimSpace(versionNum) + "+" + strings.Join(metadataStrings, ".")
	return v
}

// GenerateCommand returns a new Command object with the specified IO streams
// This is used for integration test package
func GenerateCommand(out, err io.Writer) *cobra.Command {
	opts := internal.NewToolboxOptions(internal.WithIOStreams(out, err))
	return NewCommand(opts)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Initialize options
	opts := internal.NewToolboxOptions()

	if err := NewCommand(opts).Execute(); err != nil {
		exit := 1
		os.Exit(exit)
	}
}

// NewCommand returns a Command object representing an invocation of the CLI.
func NewCommand(opts *internal.ToolboxOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "toolbox",
		Version:       versionString,
		SilenceErrors: true,
	}

	// Do not print Usage on runtime error
	cmd.SilenceUsage = true

	// Set server version
	opts.Cfg.Version = versionString

	// set baseCmd in, out and err the same as cmd.
	cmd.SetIn(opts.IOStreams.In)
	cmd.SetOut(opts.IOStreams.Out)
	cmd.SetErr(opts.IOStreams.ErrOut)

	// setup flags that are common across all commands
	internal.PersistentFlags(cmd, opts)

	flags := cmd.Flags()

	flags.StringVarP(&opts.Cfg.Address, "address", "a", "127.0.0.1", "Address of the interface the server will listen on.")
	flags.IntVarP(&opts.Cfg.Port, "port", "p", 5000, "Port the server will listen on.")

	flags.StringVar(&opts.ToolsFile, "tools_file", "", "File path specifying the tool configuration. Cannot be used with --tools-files, or --tools-folder.")
	// deprecate tools_file
	_ = flags.MarkDeprecated("tools_file", "please use --tools-file instead")
	flags.BoolVar(&opts.Cfg.Stdio, "stdio", false, "Listens via MCP STDIO instead of acting as a remote HTTP server.")
	flags.BoolVar(&opts.Cfg.DisableReload, "disable-reload", false, "Disables dynamic reloading of tools file.")
	flags.BoolVar(&opts.Cfg.UI, "ui", false, "Launches the Toolbox UI web server.")
	// TODO: Insecure by default. Might consider updating this for v1.0.0
	flags.StringSliceVar(&opts.Cfg.AllowedOrigins, "allowed-origins", []string{"*"}, "Specifies a list of origins permitted to access this server. Defaults to '*'.")
	flags.StringSliceVar(&opts.Cfg.AllowedHosts, "allowed-hosts", []string{"*"}, "Specifies a list of hosts permitted to access this server. Defaults to '*'.")

	// wrap RunE command so that we have access to original Command object
	cmd.RunE = func(*cobra.Command, []string) error { return run(cmd, opts) }

	// Register subcommands for tool invocation
	cmd.AddCommand(invoke.NewCommand(opts))
	// Register subcommands for skill generation
	cmd.AddCommand(skills.NewCommand(opts))

	return cmd
}

func handleDynamicReload(ctx context.Context, toolsFile internal.ToolsFile, s *server.Server) error {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		panic(err)
	}

	sourcesMap, authServicesMap, embeddingModelsMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap, err := validateReloadEdits(ctx, toolsFile)
	if err != nil {
		errMsg := fmt.Errorf("unable to validate reloaded edits: %w", err)
		logger.WarnContext(ctx, errMsg.Error())
		return err
	}

	s.ResourceMgr.SetResources(sourcesMap, authServicesMap, embeddingModelsMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap)

	return nil
}

// validateReloadEdits checks that the reloaded tools file configs can initialized without failing
func validateReloadEdits(
	ctx context.Context, toolsFile internal.ToolsFile,
) (map[string]sources.Source, map[string]auth.AuthService, map[string]embeddingmodels.EmbeddingModel, map[string]tools.Tool, map[string]tools.Toolset, map[string]prompts.Prompt, map[string]prompts.Promptset, error,
) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		panic(err)
	}

	instrumentation, err := util.InstrumentationFromContext(ctx)
	if err != nil {
		panic(err)
	}

	logger.DebugContext(ctx, "Attempting to parse and validate reloaded tools file.")

	ctx, span := instrumentation.Tracer.Start(ctx, "toolbox/server/reload")
	defer span.End()

	reloadedConfig := server.ServerConfig{
		Version:               versionString,
		SourceConfigs:         toolsFile.Sources,
		AuthServiceConfigs:    toolsFile.AuthServices,
		EmbeddingModelConfigs: toolsFile.EmbeddingModels,
		ToolConfigs:           toolsFile.Tools,
		ToolsetConfigs:        toolsFile.Toolsets,
		PromptConfigs:         toolsFile.Prompts,
	}

	sourcesMap, authServicesMap, embeddingModelsMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap, err := server.InitializeConfigs(ctx, reloadedConfig)
	if err != nil {
		errMsg := fmt.Errorf("unable to initialize reloaded configs: %w", err)
		logger.WarnContext(ctx, errMsg.Error())
		return nil, nil, nil, nil, nil, nil, nil, err
	}

	return sourcesMap, authServicesMap, embeddingModelsMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap, nil
}

// watchChanges checks for changes in the provided yaml tools file(s) or folder.
func watchChanges(ctx context.Context, watchDirs map[string]bool, watchedFiles map[string]bool, s *server.Server) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		panic(err)
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		logger.WarnContext(ctx, "error setting up new watcher %s", err)
		return
	}

	defer w.Close()

	watchingFolder := false
	var folderToWatch string

	// if watchedFiles is empty, indicates that user passed entire folder instead
	if len(watchedFiles) == 0 {
		watchingFolder = true

		// validate that watchDirs only has single element
		if len(watchDirs) > 1 {
			logger.WarnContext(ctx, "error setting watcher, expected single tools folder if no file(s) are defined.")
			return
		}

		for onlyKey := range watchDirs {
			folderToWatch = onlyKey
			break
		}
	}

	for dir := range watchDirs {
		err := w.Add(dir)
		if err != nil {
			logger.WarnContext(ctx, fmt.Sprintf("Error adding path %s to watcher: %s", dir, err))
			break
		}
		logger.DebugContext(ctx, fmt.Sprintf("Added directory %s to watcher.", dir))
	}

	// debounce timer is used to prevent multiple writes triggering multiple reloads
	debounceDelay := 100 * time.Millisecond
	debounce := time.NewTimer(1 * time.Minute)
	debounce.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.DebugContext(ctx, "file watcher context cancelled")
			return
		case err, ok := <-w.Errors:
			if !ok {
				logger.WarnContext(ctx, "file watcher was closed unexpectedly")
				return
			}
			if err != nil {
				logger.WarnContext(ctx, "file watcher error %s", err)
				return
			}

		case e, ok := <-w.Events:
			if !ok {
				logger.WarnContext(ctx, "file watcher already closed")
				return
			}

			// only check for events which indicate user saved a new tools file
			// multiple operations checked due to various file update methods across editors
			if !e.Has(fsnotify.Write | fsnotify.Create | fsnotify.Rename) {
				continue
			}

			cleanedFilename := filepath.Clean(e.Name)
			logger.DebugContext(ctx, fmt.Sprintf("%s event detected in %s", e.Op, cleanedFilename))

			folderChanged := watchingFolder &&
				(strings.HasSuffix(cleanedFilename, ".yaml") || strings.HasSuffix(cleanedFilename, ".yml"))

			if folderChanged || watchedFiles[cleanedFilename] {
				// indicates the write event is on a relevant file
				debounce.Reset(debounceDelay)
			}

		case <-debounce.C:
			debounce.Stop()
			var reloadedToolsFile internal.ToolsFile

			if watchingFolder {
				logger.DebugContext(ctx, "Reloading tools folder.")
				reloadedToolsFile, err = internal.LoadAndMergeToolsFolder(ctx, folderToWatch)
				if err != nil {
					logger.WarnContext(ctx, "error loading tools folder %s", err)
					continue
				}
			} else {
				logger.DebugContext(ctx, "Reloading tools file(s).")
				reloadedToolsFile, err = internal.LoadAndMergeToolsFiles(ctx, slices.Collect(maps.Keys(watchedFiles)))
				if err != nil {
					logger.WarnContext(ctx, "error loading tools files %s", err)
					continue
				}
			}

			err = handleDynamicReload(ctx, reloadedToolsFile, s)
			if err != nil {
				errMsg := fmt.Errorf("unable to parse reloaded tools file at %q: %w", reloadedToolsFile, err)
				logger.WarnContext(ctx, errMsg.Error())
				continue
			}
		}
	}
}

func resolveWatcherInputs(toolsFile string, toolsFiles []string, toolsFolder string) (map[string]bool, map[string]bool) {
	var relevantFiles []string

	// map for efficiently checking if a file is relevant
	watchedFiles := make(map[string]bool)

	// dirs that will be added to watcher (fsnotify prefers watching directory then filtering for file)
	watchDirs := make(map[string]bool)

	if len(toolsFiles) > 0 {
		relevantFiles = toolsFiles
	} else if toolsFolder != "" {
		watchDirs[filepath.Clean(toolsFolder)] = true
	} else {
		relevantFiles = []string{toolsFile}
	}

	// extract parent dir for relevant files and dedup
	for _, f := range relevantFiles {
		cleanFile := filepath.Clean(f)
		watchedFiles[cleanFile] = true
		watchDirs[filepath.Dir(cleanFile)] = true
	}

	return watchDirs, watchedFiles
}

func run(cmd *cobra.Command, opts *internal.ToolboxOptions) error {
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
			opts.Logger.DebugContext(sCtx, "Received SIGINT signal to shutdown.")
		case syscall.SIGTERM:
			opts.Logger.DebugContext(sCtx, "Sending SIGTERM signal to shutdown.")
		}
		cancel()
	}(ctx)

	ctx, shutdown, err := opts.Setup(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = shutdown(ctx)
	}()

	isCustomConfigured, err := opts.LoadConfig(ctx)
	if err != nil {
		return err
	}

	// start server
	s, err := server.NewServer(ctx, opts.Cfg)
	if err != nil {
		errMsg := fmt.Errorf("toolbox failed to initialize: %w", err)
		opts.Logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	// run server in background
	srvErr := make(chan error)
	if opts.Cfg.Stdio {
		go func() {
			defer close(srvErr)
			err = s.ServeStdio(ctx, opts.IOStreams.In, opts.IOStreams.Out)
			if err != nil {
				srvErr <- err
			}
		}()
	} else {
		err = s.Listen(ctx)
		if err != nil {
			errMsg := fmt.Errorf("toolbox failed to start listener: %w", err)
			opts.Logger.ErrorContext(ctx, errMsg.Error())
			return errMsg
		}
		opts.Logger.InfoContext(ctx, "Server ready to serve!")
		if opts.Cfg.UI {
			opts.Logger.InfoContext(ctx, fmt.Sprintf("Toolbox UI is up and running at: http://%s:%d/ui", opts.Cfg.Address, opts.Cfg.Port))
		}

		go func() {
			defer close(srvErr)
			err = s.Serve(ctx)
			if err != nil {
				srvErr <- err
			}
		}()
	}

	if isCustomConfigured && !opts.Cfg.DisableReload {
		watchDirs, watchedFiles := resolveWatcherInputs(opts.ToolsFile, opts.ToolsFiles, opts.ToolsFolder)
		// start watching the file(s) or folder for changes to trigger dynamic reloading
		go watchChanges(ctx, watchDirs, watchedFiles, s)
	}

	// wait for either the server to error out or the command's context to be canceled
	select {
	case err := <-srvErr:
		if err != nil {
			errMsg := fmt.Errorf("toolbox crashed with the following error: %w", err)
			opts.Logger.ErrorContext(ctx, errMsg.Error())
			return errMsg
		}
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		opts.Logger.WarnContext(shutdownContext, "Shutting down gracefully...")
		err := s.Shutdown(shutdownContext)
		if err == context.DeadlineExceeded {
			return fmt.Errorf("graceful shutdown timed out... forcing exit")
		}
	}

	return nil
}
