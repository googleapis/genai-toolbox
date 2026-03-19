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
	"fmt"
	"strings"

	"github.com/googleapis/genai-toolbox/internal/prebuiltconfigs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// PersistentFlags sets up flags that are available for all commands and
// subcommands
// It is also used to set up persistent flags during subcommand unit tests
func PersistentFlags(parentCmd *cobra.Command, opts *ToolboxOptions) {
	persistentFlags := parentCmd.PersistentFlags()
	persistentFlags.Var(&opts.Cfg.LogLevel, "log-level", "Specify the minimum level logged. Allowed: 'DEBUG', 'INFO', 'WARN', 'ERROR'.")
	persistentFlags.Var(&opts.Cfg.LoggingFormat, "logging-format", "Specify logging format to use. Allowed: 'standard' or 'JSON'.")
	persistentFlags.BoolVar(&opts.Cfg.TelemetryGCP, "telemetry-gcp", false, "Enable exporting directly to Google Cloud Monitoring.")
	persistentFlags.StringVar(&opts.Cfg.TelemetryOTLP, "telemetry-otlp", "", "Enable exporting using OpenTelemetry Protocol (OTLP) to the specified endpoint (e.g. 'http://127.0.0.1:4318')")
	persistentFlags.StringVar(&opts.Cfg.TelemetryServiceName, "telemetry-service-name", "toolbox", "Sets the value of the service.name resource attribute for telemetry data.")
	persistentFlags.StringSliceVar(&opts.Cfg.UserAgentMetadata, "user-agent-metadata", []string{}, "Appends additional metadata to the User-Agent.")
}

// ConfigFileFlags defines flags related to the configuration file.
// It should be applied to any command that requires configuration loading.
func ConfigFileFlags(flags *pflag.FlagSet, opts *ToolboxOptions) {
	flags.StringVar(&opts.ToolsFile, "tools-file", "", "File path specifying the tool configuration. Cannot be used with --tools-files, or --tools-folder.")
	flags.StringSliceVar(&opts.ToolsFiles, "tools-files", []string{}, "Multiple file paths specifying tool configurations. Files will be merged. Cannot be used with --tools-file, or --tools-folder.")
	flags.StringVar(&opts.ToolsFolder, "tools-folder", "", "Directory path containing YAML tool configuration files. All .yaml and .yml files in the directory will be loaded and merged. Cannot be used with --tools-file, or --tools-files.")
	// Fetch prebuilt tools sources to customize the help description
	prebuiltHelp := fmt.Sprintf(
		"Use a prebuilt tool configuration by source type. Allowed: '%s'. Can be specified multiple times.",
		strings.Join(prebuiltconfigs.GetPrebuiltSources(), "', '"),
	)
	flags.StringSliceVar(&opts.PrebuiltConfigs, "prebuilt", []string{}, prebuiltHelp)
}

// ServeFlags defines flags for starting and configuring the server.
func ServeFlags(flags *pflag.FlagSet, opts *ToolboxOptions) {
	flags.StringVarP(&opts.Cfg.Address, "address", "a", "127.0.0.1", "Address of the interface the server will listen on.")
	flags.IntVarP(&opts.Cfg.Port, "port", "p", 5000, "Port the server will listen on.")
	flags.BoolVar(&opts.Cfg.Stdio, "stdio", false, "Listens via MCP STDIO instead of acting as a remote HTTP server.")
	flags.BoolVar(&opts.Cfg.UI, "ui", false, "Launches the Toolbox UI web server.")

	flags.StringSliceVar(&opts.Cfg.AllowedOrigins, "allowed-origins", []string{"*"}, "Specifies a list of origins permitted to access this server. Defaults to '*'.")
	flags.StringSliceVar(&opts.Cfg.AllowedHosts, "allowed-hosts", []string{"*"}, "Specifies a list of hosts permitted to access this server. Defaults to '*'.")
}
