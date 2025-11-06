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
	"regexp"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/prebuiltconfigs"
	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/telemetry"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"

	// Import prompt packages for side effect of registration
	_ "github.com/googleapis/genai-toolbox/internal/prompts/custom"

	// Import tool packages for side effect of registration
	_ "github.com/googleapis/genai-toolbox/internal/tools/alloydb/alloydbcreatecluster"
	_ "github.com/googleapis/genai-toolbox/internal/tools/alloydb/alloydbcreateinstance"
	_ "github.com/googleapis/genai-toolbox/internal/tools/alloydb/alloydbcreateuser"
	_ "github.com/googleapis/genai-toolbox/internal/tools/alloydb/alloydbgetcluster"
	_ "github.com/googleapis/genai-toolbox/internal/tools/alloydb/alloydbgetinstance"
	_ "github.com/googleapis/genai-toolbox/internal/tools/alloydb/alloydbgetuser"
	_ "github.com/googleapis/genai-toolbox/internal/tools/alloydb/alloydblistclusters"
	_ "github.com/googleapis/genai-toolbox/internal/tools/alloydb/alloydblistinstances"
	_ "github.com/googleapis/genai-toolbox/internal/tools/alloydb/alloydblistusers"
	_ "github.com/googleapis/genai-toolbox/internal/tools/alloydb/alloydbwaitforoperation"
	_ "github.com/googleapis/genai-toolbox/internal/tools/alloydbainl"
	_ "github.com/googleapis/genai-toolbox/internal/tools/bigquery/bigqueryanalyzecontribution"
	_ "github.com/googleapis/genai-toolbox/internal/tools/bigquery/bigqueryconversationalanalytics"
	_ "github.com/googleapis/genai-toolbox/internal/tools/bigquery/bigqueryexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/bigquery/bigqueryforecast"
	_ "github.com/googleapis/genai-toolbox/internal/tools/bigquery/bigquerygetdatasetinfo"
	_ "github.com/googleapis/genai-toolbox/internal/tools/bigquery/bigquerygettableinfo"
	_ "github.com/googleapis/genai-toolbox/internal/tools/bigquery/bigquerylistdatasetids"
	_ "github.com/googleapis/genai-toolbox/internal/tools/bigquery/bigquerylisttableids"
	_ "github.com/googleapis/genai-toolbox/internal/tools/bigquery/bigquerysearchcatalog"
	_ "github.com/googleapis/genai-toolbox/internal/tools/bigquery/bigquerysql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/bigtable"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cassandra/cassandracql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/clickhouse/clickhouseexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/clickhouse/clickhouselistdatabases"
	_ "github.com/googleapis/genai-toolbox/internal/tools/clickhouse/clickhouselisttables"
	_ "github.com/googleapis/genai-toolbox/internal/tools/clickhouse/clickhousesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudmonitoring"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsql/cloudsqlcreatedatabase"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsql/cloudsqlcreateusers"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsql/cloudsqlgetinstances"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsql/cloudsqllistdatabases"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsql/cloudsqllistinstances"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsql/cloudsqlwaitforoperation"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsqlmssql/cloudsqlmssqlcreateinstance"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsqlmysql/cloudsqlmysqlcreateinstance"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsqlpg/cloudsqlpgcreateinstances"
	_ "github.com/googleapis/genai-toolbox/internal/tools/couchbase"
	_ "github.com/googleapis/genai-toolbox/internal/tools/dataform/dataformcompilelocal"
	_ "github.com/googleapis/genai-toolbox/internal/tools/dataplex/dataplexlookupentry"
	_ "github.com/googleapis/genai-toolbox/internal/tools/dataplex/dataplexsearchaspecttypes"
	_ "github.com/googleapis/genai-toolbox/internal/tools/dataplex/dataplexsearchentries"
	_ "github.com/googleapis/genai-toolbox/internal/tools/dgraph"
	_ "github.com/googleapis/genai-toolbox/internal/tools/firebird/firebirdexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/firebird/firebirdsql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/firestore/firestoreadddocuments"
	_ "github.com/googleapis/genai-toolbox/internal/tools/firestore/firestoredeletedocuments"
	_ "github.com/googleapis/genai-toolbox/internal/tools/firestore/firestoregetdocuments"
	_ "github.com/googleapis/genai-toolbox/internal/tools/firestore/firestoregetrules"
	_ "github.com/googleapis/genai-toolbox/internal/tools/firestore/firestorelistcollections"
	_ "github.com/googleapis/genai-toolbox/internal/tools/firestore/firestorequery"
	_ "github.com/googleapis/genai-toolbox/internal/tools/firestore/firestorequerycollection"
	_ "github.com/googleapis/genai-toolbox/internal/tools/firestore/firestoreupdatedocument"
	_ "github.com/googleapis/genai-toolbox/internal/tools/firestore/firestorevalidaterules"
	_ "github.com/googleapis/genai-toolbox/internal/tools/http"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookercreateprojectfile"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookeradddashboardelement"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookerconversationalanalytics"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookerdeleteprojectfile"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookerdevmode"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookergetdashboards"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookergetdimensions"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookergetexplores"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookergetfilters"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookergetlooks"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookergetmeasures"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookergetmodels"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookergetparameters"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookergetprojectfile"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookergetprojectfiles"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookergetprojects"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookerhealthanalyze"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookerhealthpulse"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookerhealthvacuum"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookermakedashboard"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookermakelook"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookerquery"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookerquerysql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookerqueryurl"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookerrunlook"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookerupdateprojectfile"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mongodb/mongodbaggregate"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mongodb/mongodbdeletemany"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mongodb/mongodbdeleteone"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mongodb/mongodbfind"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mongodb/mongodbfindone"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mongodb/mongodbinsertmany"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mongodb/mongodbinsertone"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mongodb/mongodbupdatemany"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mongodb/mongodbupdateone"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mssql/mssqlexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mssql/mssqllisttables"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mssql/mssqlsql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mysql/mysqlexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mysql/mysqllistactivequeries"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mysql/mysqllisttablefragmentation"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mysql/mysqllisttables"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mysql/mysqllisttablesmissinguniqueindexes"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mysql/mysqlsql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/neo4j/neo4jcypher"
	_ "github.com/googleapis/genai-toolbox/internal/tools/neo4j/neo4jexecutecypher"
	_ "github.com/googleapis/genai-toolbox/internal/tools/neo4j/neo4jschema"
	_ "github.com/googleapis/genai-toolbox/internal/tools/oceanbase/oceanbaseexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/oceanbase/oceanbasesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/oracle/oracleexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/oracle/oraclesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgresexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslistactivequeries"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslistavailableextensions"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslistinstalledextensions"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslisttables"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgressql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/redis"
	_ "github.com/googleapis/genai-toolbox/internal/tools/serverlessspark/serverlesssparkgetbatch"
	_ "github.com/googleapis/genai-toolbox/internal/tools/serverlessspark/serverlesssparklistbatches"
	_ "github.com/googleapis/genai-toolbox/internal/tools/spanner/spannerexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/spanner/spannerlisttables"
	_ "github.com/googleapis/genai-toolbox/internal/tools/spanner/spannersql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/sqlite/sqliteexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/sqlite/sqlitesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/tidb/tidbexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/tidb/tidbsql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/trino/trinoexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/trino/trinosql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/utility/wait"
	_ "github.com/googleapis/genai-toolbox/internal/tools/valkey"
	_ "github.com/googleapis/genai-toolbox/internal/tools/yugabytedbsql"

	"github.com/spf13/cobra"

	_ "github.com/googleapis/genai-toolbox/internal/sources/alloydbadmin"
	_ "github.com/googleapis/genai-toolbox/internal/sources/alloydbpg"
	_ "github.com/googleapis/genai-toolbox/internal/sources/bigquery"
	_ "github.com/googleapis/genai-toolbox/internal/sources/bigtable"
	_ "github.com/googleapis/genai-toolbox/internal/sources/cassandra"
	_ "github.com/googleapis/genai-toolbox/internal/sources/clickhouse"
	_ "github.com/googleapis/genai-toolbox/internal/sources/cloudmonitoring"
	_ "github.com/googleapis/genai-toolbox/internal/sources/cloudsqladmin"
	_ "github.com/googleapis/genai-toolbox/internal/sources/cloudsqlmssql"
	_ "github.com/googleapis/genai-toolbox/internal/sources/cloudsqlmysql"
	_ "github.com/googleapis/genai-toolbox/internal/sources/cloudsqlpg"
	_ "github.com/googleapis/genai-toolbox/internal/sources/couchbase"
	_ "github.com/googleapis/genai-toolbox/internal/sources/dataplex"
	_ "github.com/googleapis/genai-toolbox/internal/sources/dgraph"
	_ "github.com/googleapis/genai-toolbox/internal/sources/firebird"
	_ "github.com/googleapis/genai-toolbox/internal/sources/firestore"
	_ "github.com/googleapis/genai-toolbox/internal/sources/http"
	_ "github.com/googleapis/genai-toolbox/internal/sources/looker"
	_ "github.com/googleapis/genai-toolbox/internal/sources/mongodb"
	_ "github.com/googleapis/genai-toolbox/internal/sources/mssql"
	_ "github.com/googleapis/genai-toolbox/internal/sources/mysql"
	_ "github.com/googleapis/genai-toolbox/internal/sources/neo4j"
	_ "github.com/googleapis/genai-toolbox/internal/sources/oceanbase"
	_ "github.com/googleapis/genai-toolbox/internal/sources/oracle"
	_ "github.com/googleapis/genai-toolbox/internal/sources/postgres"
	_ "github.com/googleapis/genai-toolbox/internal/sources/redis"
	_ "github.com/googleapis/genai-toolbox/internal/sources/serverlessspark"
	_ "github.com/googleapis/genai-toolbox/internal/sources/spanner"
	_ "github.com/googleapis/genai-toolbox/internal/sources/sqlite"
	_ "github.com/googleapis/genai-toolbox/internal/sources/tidb"
	_ "github.com/googleapis/genai-toolbox/internal/sources/trino"
	_ "github.com/googleapis/genai-toolbox/internal/sources/valkey"
	_ "github.com/googleapis/genai-toolbox/internal/sources/yugabytedb"
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

	cfg            server.ServerConfig
	logger         log.Logger
	tools_file     string
	tools_files    []string
	tools_folder   string
	prebuiltConfig string
	inStream       io.Reader
	outStream      io.Writer
	errStream      io.Writer
}

// NewCommand returns a Command object representing an invocation of the CLI.
func NewCommand(opts ...Option) *Command {
	in := os.Stdin
	out := os.Stdout
	err := os.Stderr

	baseCmd := &cobra.Command{
		Use:           "toolbox",
		Version:       versionString,
		SilenceErrors: true,
	}
	cmd := &Command{
		Command:   baseCmd,
		inStream:  in,
		outStream: out,
		errStream: err,
	}

	for _, o := range opts {
		o(cmd)
	}

	// Do not print Usage on runtime error
	cmd.SilenceUsage = true

	// Set server version
	cmd.cfg.Version = versionString

	// set baseCmd in, out and err the same as cmd.
	baseCmd.SetIn(cmd.inStream)
	baseCmd.SetOut(cmd.outStream)
	baseCmd.SetErr(cmd.errStream)

	flags := cmd.Flags()
	flags.StringVarP(&cmd.cfg.Address, "address", "a", "127.0.0.1", "Address of the interface the server will listen on.")
	flags.IntVarP(&cmd.cfg.Port, "port", "p", 5000, "Port the server will listen on.")

	flags.StringVar(&cmd.tools_file, "tools_file", "", "File path specifying the tool configuration. Cannot be used with --prebuilt.")
	// deprecate tools_file
	_ = flags.MarkDeprecated("tools_file", "please use --tools-file instead")
	flags.StringVar(&cmd.tools_file, "tools-file", "", "File path specifying the tool configuration. Cannot be used with --prebuilt, --tools-files, or --tools-folder.")
	flags.StringSliceVar(&cmd.tools_files, "tools-files", []string{}, "Multiple file paths specifying tool configurations. Files will be merged. Cannot be used with --prebuilt, --tools-file, or --tools-folder.")
	flags.StringVar(&cmd.tools_folder, "tools-folder", "", "Directory path containing YAML tool configuration files. All .yaml and .yml files in the directory will be loaded and merged. Cannot be used with --prebuilt, --tools-file, or --tools-files.")
	flags.Var(&cmd.cfg.LogLevel, "log-level", "Specify the minimum level logged. Allowed: 'DEBUG', 'INFO', 'WARN', 'ERROR'.")
	flags.Var(&cmd.cfg.LoggingFormat, "logging-format", "Specify logging format to use. Allowed: 'standard' or 'JSON'.")
	flags.BoolVar(&cmd.cfg.TelemetryGCP, "telemetry-gcp", false, "Enable exporting directly to Google Cloud Monitoring.")
	flags.StringVar(&cmd.cfg.TelemetryOTLP, "telemetry-otlp", "", "Enable exporting using OpenTelemetry Protocol (OTLP) to the specified endpoint (e.g. 'http://127.0.0.1:4318')")
	flags.StringVar(&cmd.cfg.TelemetryServiceName, "telemetry-service-name", "toolbox", "Sets the value of the service.name resource attribute for telemetry data.")

	// Fetch prebuilt tools sources to customize the help description
	prebuiltHelp := fmt.Sprintf(
		"Use a prebuilt tool configuration by source type. Cannot be used with --tools-file. Allowed: '%s'.",
		strings.Join(prebuiltconfigs.GetPrebuiltSources(), "', '"),
	)
	flags.StringVar(&cmd.prebuiltConfig, "prebuilt", "", prebuiltHelp)
	flags.BoolVar(&cmd.cfg.Stdio, "stdio", false, "Listens via MCP STDIO instead of acting as a remote HTTP server.")
	flags.BoolVar(&cmd.cfg.DisableReload, "disable-reload", false, "Disables dynamic reloading of tools file.")
	flags.BoolVar(&cmd.cfg.UI, "ui", false, "Launches the Toolbox UI web server.")

	// wrap RunE command so that we have access to original Command object
	cmd.RunE = func(*cobra.Command, []string) error { return run(cmd) }

	return cmd
}

type ToolsFile struct {
	Sources      server.SourceConfigs      `yaml:"sources"`
	AuthSources  server.AuthServiceConfigs `yaml:"authSources"` // Deprecated: Kept for compatibility.
	AuthServices server.AuthServiceConfigs `yaml:"authServices"`
	Tools        server.ToolConfigs        `yaml:"tools"`
	Toolsets     server.ToolsetConfigs     `yaml:"toolsets"`
	Prompts      server.PromptConfigs      `yaml:"prompts"`
	Promptsets   server.PromptsetConfigs   `yaml:"promptsets"`
}

// parseEnv replaces environment variables ${ENV_NAME} with their values.
// also support ${ENV_NAME:default_value}.
func parseEnv(input string) (string, error) {
	re := regexp.MustCompile(`\$\{(\w+)(:(\w*))?\}`)

	var err error
	output := re.ReplaceAllStringFunc(input, func(match string) string {
		parts := re.FindStringSubmatch(match)

		// extract the variable name
		variableName := parts[1]
		if value, found := os.LookupEnv(variableName); found {
			return value
		}
		if parts[2] != "" {
			return parts[3]
		}
		err = fmt.Errorf("environment variable not found: %q", variableName)
		return ""
	})
	return output, err
}

// parseToolsFile parses the provided yaml into appropriate configs.
func parseToolsFile(ctx context.Context, raw []byte) (ToolsFile, error) {
	var toolsFile ToolsFile
	// Replace environment variables if found
	output, err := parseEnv(string(raw))
	if err != nil {
		return toolsFile, fmt.Errorf("error parsing environment variables: %s", err)
	}
	raw = []byte(output)

	// Parse contents
	err = yaml.UnmarshalContext(ctx, raw, &toolsFile, yaml.Strict())
	if err != nil {
		return toolsFile, err
	}
	return toolsFile, nil
}

// mergeToolsFiles merges multiple ToolsFile structs into one.
// Detects and raises errors for resource conflicts in sources, authServices, tools, and toolsets.
// All resource names (sources, authServices, tools, toolsets) must be unique across all files.
func mergeToolsFiles(files ...ToolsFile) (ToolsFile, error) {
	merged := ToolsFile{
		Sources:      make(server.SourceConfigs),
		AuthServices: make(server.AuthServiceConfigs),
		Tools:        make(server.ToolConfigs),
		Toolsets:     make(server.ToolsetConfigs),
		Prompts:      make(server.PromptConfigs),
		Promptsets:   make(server.PromptsetConfigs),
	}

	var conflicts []string

	for fileIndex, file := range files {
		// Check for conflicts and merge sources
		for name, source := range file.Sources {
			if _, exists := merged.Sources[name]; exists {
				conflicts = append(conflicts, fmt.Sprintf("source '%s' (file #%d)", name, fileIndex+1))
			} else {
				merged.Sources[name] = source
			}
		}

		// Check for conflicts and merge authSources (deprecated, but still support)
		for name, authSource := range file.AuthSources {
			if _, exists := merged.AuthSources[name]; exists {
				conflicts = append(conflicts, fmt.Sprintf("authSource '%s' (file #%d)", name, fileIndex+1))
			} else {
				merged.AuthSources[name] = authSource
			}
		}

		// Check for conflicts and merge authServices
		for name, authService := range file.AuthServices {
			if _, exists := merged.AuthServices[name]; exists {
				conflicts = append(conflicts, fmt.Sprintf("authService '%s' (file #%d)", name, fileIndex+1))
			} else {
				merged.AuthServices[name] = authService
			}
		}

		// Check for conflicts and merge tools
		for name, tool := range file.Tools {
			if _, exists := merged.Tools[name]; exists {
				conflicts = append(conflicts, fmt.Sprintf("tool '%s' (file #%d)", name, fileIndex+1))
			} else {
				merged.Tools[name] = tool
			}
		}

		// Check for conflicts and merge toolsets
		for name, toolset := range file.Toolsets {
			if _, exists := merged.Toolsets[name]; exists {
				conflicts = append(conflicts, fmt.Sprintf("toolset '%s' (file #%d)", name, fileIndex+1))
			} else {
				merged.Toolsets[name] = toolset
			}
		}

		// Check for conflicts and merge prompts
		for name, prompt := range file.Prompts {
			if _, exists := merged.Prompts[name]; exists {
				conflicts = append(conflicts, fmt.Sprintf("prompt '%s' (file #%d)", name, fileIndex+1))
			} else {
				merged.Prompts[name] = prompt
			}
		}

		// Check for conflicts and merge promptsets
		for name, promptset := range file.Promptsets {
			if _, exists := merged.Promptsets[name]; exists {
				conflicts = append(conflicts, fmt.Sprintf("promptset '%s' (file #%d)", name, fileIndex+1))
			} else {
				merged.Promptsets[name] = promptset
			}
		}
	}

	// If conflicts were detected, return an error
	if len(conflicts) > 0 {
		return ToolsFile{}, fmt.Errorf("resource conflicts detected:\n  - %s\n\nPlease ensure each source, authService, tool, toolset, prompt and promptset has a unique name across all files", strings.Join(conflicts, "\n  - "))
	}

	return merged, nil
}

// loadAndMergeToolsFiles loads multiple YAML files and merges them
func loadAndMergeToolsFiles(ctx context.Context, filePaths []string) (ToolsFile, error) {
	var toolsFiles []ToolsFile

	for _, filePath := range filePaths {
		buf, err := os.ReadFile(filePath)
		if err != nil {
			return ToolsFile{}, fmt.Errorf("unable to read tool file at %q: %w", filePath, err)
		}

		toolsFile, err := parseToolsFile(ctx, buf)
		if err != nil {
			return ToolsFile{}, fmt.Errorf("unable to parse tool file at %q: %w", filePath, err)
		}

		toolsFiles = append(toolsFiles, toolsFile)
	}

	mergedFile, err := mergeToolsFiles(toolsFiles...)
	if err != nil {
		return ToolsFile{}, fmt.Errorf("unable to merge tools files: %w", err)
	}

	return mergedFile, nil
}

// loadAndMergeToolsFolder loads all YAML files from a directory and merges them
func loadAndMergeToolsFolder(ctx context.Context, folderPath string) (ToolsFile, error) {
	// Check if directory exists
	info, err := os.Stat(folderPath)
	if err != nil {
		return ToolsFile{}, fmt.Errorf("unable to access tools folder at %q: %w", folderPath, err)
	}
	if !info.IsDir() {
		return ToolsFile{}, fmt.Errorf("path %q is not a directory", folderPath)
	}

	// Find all YAML files in the directory
	pattern := filepath.Join(folderPath, "*.yaml")
	yamlFiles, err := filepath.Glob(pattern)
	if err != nil {
		return ToolsFile{}, fmt.Errorf("error finding YAML files in %q: %w", folderPath, err)
	}

	// Also find .yml files
	ymlPattern := filepath.Join(folderPath, "*.yml")
	ymlFiles, err := filepath.Glob(ymlPattern)
	if err != nil {
		return ToolsFile{}, fmt.Errorf("error finding YML files in %q: %w", folderPath, err)
	}

	// Combine both file lists
	allFiles := append(yamlFiles, ymlFiles...)

	if len(allFiles) == 0 {
		return ToolsFile{}, fmt.Errorf("no YAML files found in directory %q", folderPath)
	}

	// Use existing loadAndMergeToolsFiles function
	return loadAndMergeToolsFiles(ctx, allFiles)
}

func handleDynamicReload(ctx context.Context, toolsFile ToolsFile, s *server.Server) error {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		panic(err)
	}

	sourcesMap, authServicesMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap, err := validateReloadEdits(ctx, toolsFile)
	if err != nil {
		errMsg := fmt.Errorf("unable to validate reloaded edits: %w", err)
		logger.WarnContext(ctx, errMsg.Error())
		return err
	}

	s.ResourceMgr.SetResources(sourcesMap, authServicesMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap)

	return nil
}

// validateReloadEdits checks that the reloaded tools file configs can initialized without failing
func validateReloadEdits(
	ctx context.Context, toolsFile ToolsFile,
) (map[string]sources.Source, map[string]auth.AuthService, map[string]tools.Tool, map[string]tools.Toolset, map[string]prompts.Prompt, map[string]prompts.Promptset, error,
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
		Version:            versionString,
		SourceConfigs:      toolsFile.Sources,
		AuthServiceConfigs: toolsFile.AuthServices,
		ToolConfigs:        toolsFile.Tools,
		ToolsetConfigs:     toolsFile.Toolsets,
		PromptConfigs:      toolsFile.Prompts,
		PromptsetConfigs:   toolsFile.Promptsets,
	}

	sourcesMap, authServicesMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap, err := server.InitializeConfigs(ctx, reloadedConfig)
	if err != nil {
		errMsg := fmt.Errorf("unable to initialize reloaded configs: %w", err)
		logger.WarnContext(ctx, errMsg.Error())
		return nil, nil, nil, nil, nil, nil, err
	}

	return sourcesMap, authServicesMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap, nil
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
			var reloadedToolsFile ToolsFile

			if watchingFolder {
				logger.DebugContext(ctx, "Reloading tools folder.")
				reloadedToolsFile, err = loadAndMergeToolsFolder(ctx, folderToWatch)
				if err != nil {
					logger.WarnContext(ctx, "error loading tools folder %s", err)
					continue
				}
			} else {
				logger.DebugContext(ctx, "Reloading tools file(s).")
				reloadedToolsFile, err = loadAndMergeToolsFiles(ctx, slices.Collect(maps.Keys(watchedFiles)))
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

	// If stdio, set logger's out stream (usually DEBUG and INFO logs) to errStream
	loggerOut := cmd.outStream
	if cmd.cfg.Stdio {
		loggerOut = cmd.errStream
	}

	// Handle logger separately from config
	switch strings.ToLower(cmd.cfg.LoggingFormat.String()) {
	case "json":
		logger, err := log.NewStructuredLogger(loggerOut, cmd.errStream, cmd.cfg.LogLevel.String())
		if err != nil {
			return fmt.Errorf("unable to initialize logger: %w", err)
		}
		cmd.logger = logger
	case "standard":
		logger, err := log.NewStdLogger(loggerOut, cmd.errStream, cmd.cfg.LogLevel.String())
		if err != nil {
			return fmt.Errorf("unable to initialize logger: %w", err)
		}
		cmd.logger = logger
	default:
		return fmt.Errorf("logging format invalid")
	}

	ctx = util.WithLogger(ctx, cmd.logger)

	// Set up OpenTelemetry
	otelShutdown, err := telemetry.SetupOTel(ctx, cmd.cfg.Version, cmd.cfg.TelemetryOTLP, cmd.cfg.TelemetryGCP, cmd.cfg.TelemetryServiceName)
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

	var toolsFile ToolsFile

	if cmd.prebuiltConfig != "" {
		// Make sure --prebuilt and --tools-file/--tools-files/--tools-folder flags are mutually exclusive
		if cmd.tools_file != "" || len(cmd.tools_files) > 0 || cmd.tools_folder != "" {
			errMsg := fmt.Errorf("--prebuilt and --tools-file/--tools-files/--tools-folder flags cannot be used simultaneously")
			cmd.logger.ErrorContext(ctx, errMsg.Error())
			return errMsg
		}
		// Use prebuilt tools
		buf, err := prebuiltconfigs.Get(cmd.prebuiltConfig)
		if err != nil {
			cmd.logger.ErrorContext(ctx, err.Error())
			return err
		}
		logMsg := fmt.Sprint("Using prebuilt tool configuration for ", cmd.prebuiltConfig)
		cmd.logger.InfoContext(ctx, logMsg)
		// Append prebuilt.source to Version string for the User Agent
		cmd.cfg.Version += "+prebuilt." + cmd.prebuiltConfig

		toolsFile, err = parseToolsFile(ctx, buf)
		if err != nil {
			errMsg := fmt.Errorf("unable to parse prebuilt tool configuration: %w", err)
			cmd.logger.ErrorContext(ctx, errMsg.Error())
			return errMsg
		}
	} else if len(cmd.tools_files) > 0 {
		// Make sure --tools-file, --tools-files, and --tools-folder flags are mutually exclusive
		if cmd.tools_file != "" || cmd.tools_folder != "" {
			errMsg := fmt.Errorf("--tools-file, --tools-files, and --tools-folder flags cannot be used simultaneously")
			cmd.logger.ErrorContext(ctx, errMsg.Error())
			return errMsg
		}

		// Use multiple tools files
		cmd.logger.InfoContext(ctx, fmt.Sprintf("Loading and merging %d tool configuration files", len(cmd.tools_files)))
		var err error
		toolsFile, err = loadAndMergeToolsFiles(ctx, cmd.tools_files)
		if err != nil {
			cmd.logger.ErrorContext(ctx, err.Error())
			return err
		}
	} else if cmd.tools_folder != "" {
		// Make sure --tools-folder and other flags are mutually exclusive
		if cmd.tools_file != "" || len(cmd.tools_files) > 0 {
			errMsg := fmt.Errorf("--tools-file, --tools-files, and --tools-folder flags cannot be used simultaneously")
			cmd.logger.ErrorContext(ctx, errMsg.Error())
			return errMsg
		}

		// Use tools folder
		cmd.logger.InfoContext(ctx, fmt.Sprintf("Loading and merging all YAML files from directory: %s", cmd.tools_folder))
		var err error
		toolsFile, err = loadAndMergeToolsFolder(ctx, cmd.tools_folder)
		if err != nil {
			cmd.logger.ErrorContext(ctx, err.Error())
			return err
		}
	} else {
		// Set default value of tools-file flag to tools.yaml
		if cmd.tools_file == "" {
			cmd.tools_file = "tools.yaml"
		}

		// Read single tool file contents
		buf, err := os.ReadFile(cmd.tools_file)
		if err != nil {
			errMsg := fmt.Errorf("unable to read tool file at %q: %w", cmd.tools_file, err)
			cmd.logger.ErrorContext(ctx, errMsg.Error())
			return errMsg
		}

		toolsFile, err = parseToolsFile(ctx, buf)
		if err != nil {
			errMsg := fmt.Errorf("unable to parse tool file at %q: %w", cmd.tools_file, err)
			cmd.logger.ErrorContext(ctx, errMsg.Error())
			return errMsg
		}
	}

	cmd.cfg.SourceConfigs, cmd.cfg.AuthServiceConfigs, cmd.cfg.ToolConfigs, cmd.cfg.ToolsetConfigs, cmd.cfg.PromptConfigs, cmd.cfg.PromptsetConfigs = toolsFile.Sources, toolsFile.AuthServices, toolsFile.Tools, toolsFile.Toolsets, toolsFile.Prompts, toolsFile.Promptsets

	authSourceConfigs := toolsFile.AuthSources
	if authSourceConfigs != nil {
		cmd.logger.WarnContext(ctx, "`authSources` is deprecated, use `authServices` instead")
		cmd.cfg.AuthServiceConfigs = authSourceConfigs
	}

	instrumentation, err := telemetry.CreateTelemetryInstrumentation(versionString)
	if err != nil {
		errMsg := fmt.Errorf("unable to create telemetry instrumentation: %w", err)
		cmd.logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	ctx = util.WithInstrumentation(ctx, instrumentation)

	// start server
	s, err := server.NewServer(ctx, cmd.cfg)
	if err != nil {
		errMsg := fmt.Errorf("toolbox failed to initialize: %w", err)
		cmd.logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	// run server in background
	srvErr := make(chan error)
	if cmd.cfg.Stdio {
		go func() {
			defer close(srvErr)
			err = s.ServeStdio(ctx, cmd.inStream, cmd.outStream)
			if err != nil {
				srvErr <- err
			}
		}()
	} else {
		err = s.Listen(ctx)
		if err != nil {
			errMsg := fmt.Errorf("toolbox failed to start listener: %w", err)
			cmd.logger.ErrorContext(ctx, errMsg.Error())
			return errMsg
		}
		cmd.logger.InfoContext(ctx, "Server ready to serve!")
		if cmd.cfg.UI {
			cmd.logger.InfoContext(ctx, fmt.Sprintf("Toolbox UI is up and running at: http://%s:%d/ui", cmd.cfg.Address, cmd.cfg.Port))
		}

		go func() {
			defer close(srvErr)
			err = s.Serve(ctx)
			if err != nil {
				srvErr <- err
			}
		}()
	}

	watchDirs, watchedFiles := resolveWatcherInputs(cmd.tools_file, cmd.tools_files, cmd.tools_folder)

	if !cmd.cfg.DisableReload {
		// start watching the file(s) or folder for changes to trigger dynamic reloading
		go watchChanges(ctx, watchDirs, watchedFiles, s)
	}

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
			return fmt.Errorf("graceful shutdown timed out... forcing exit")
		}
	}

	return nil
}
