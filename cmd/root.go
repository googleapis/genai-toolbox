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
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudgda"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/cloudhealthcarefhirfetchpage"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/cloudhealthcarefhirpatienteverything"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/cloudhealthcarefhirpatientsearch"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/cloudhealthcaregetdataset"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/cloudhealthcaregetdicomstore"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/cloudhealthcaregetdicomstoremetrics"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/cloudhealthcaregetfhirresource"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/cloudhealthcaregetfhirstore"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/cloudhealthcaregetfhirstoremetrics"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/cloudhealthcarelistdicomstores"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/cloudhealthcarelistfhirstores"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/cloudhealthcareretrieverendereddicominstance"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/cloudhealthcaresearchdicominstances"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/cloudhealthcaresearchdicomseries"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/cloudhealthcaresearchdicomstudies"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudmonitoring"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsql/cloudsqlcloneinstance"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsql/cloudsqlcreatedatabase"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsql/cloudsqlcreateusers"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsql/cloudsqlgetinstances"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsql/cloudsqllistdatabases"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsql/cloudsqllistinstances"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsql/cloudsqlwaitforoperation"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsqlmssql/cloudsqlmssqlcreateinstance"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsqlmysql/cloudsqlmysqlcreateinstance"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsqlpg/cloudsqlpgcreateinstances"
	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsqlpg/cloudsqlpgupgradeprecheck"
	_ "github.com/googleapis/genai-toolbox/internal/tools/couchbase"
	_ "github.com/googleapis/genai-toolbox/internal/tools/dataform/dataformcompilelocal"
	_ "github.com/googleapis/genai-toolbox/internal/tools/dataplex/dataplexlookupentry"
	_ "github.com/googleapis/genai-toolbox/internal/tools/dataplex/dataplexsearchaspecttypes"
	_ "github.com/googleapis/genai-toolbox/internal/tools/dataplex/dataplexsearchentries"
	_ "github.com/googleapis/genai-toolbox/internal/tools/dgraph"
	_ "github.com/googleapis/genai-toolbox/internal/tools/elasticsearch/elasticsearchesql"
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
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookeradddashboardelement"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookeradddashboardfilter"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookerconversationalanalytics"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookercreateprojectfile"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookerdeleteprojectfile"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookerdevmode"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookergenerateembedurl"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookergetconnectiondatabases"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookergetconnections"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookergetconnectionschemas"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookergetconnectiontablecolumns"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookergetconnectiontables"
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
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookerrundashboard"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookerrunlook"
	_ "github.com/googleapis/genai-toolbox/internal/tools/looker/lookerupdateprojectfile"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mindsdb/mindsdbexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mindsdb/mindsdbsql"
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
	_ "github.com/googleapis/genai-toolbox/internal/tools/mysql/mysqlgetqueryplan"
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
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgresdatabaseoverview"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgresexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgresgetcolumncardinality"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslistactivequeries"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslistavailableextensions"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslistdatabasestats"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslistindexes"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslistinstalledextensions"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslistlocks"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslistpgsettings"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslistpublicationtables"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslistquerystats"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslistroles"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslistschemas"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslistsequences"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgresliststoredprocedure"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslisttables"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslisttablespaces"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslisttablestats"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslisttriggers"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslistviews"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgreslongrunningtransactions"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgresreplicationstats"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgres/postgressql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/redis/redis"
	_ "github.com/googleapis/genai-toolbox/internal/tools/redis/redisexecutecmd"
	_ "github.com/googleapis/genai-toolbox/internal/tools/serverlessspark/serverlesssparkcancelbatch"
	_ "github.com/googleapis/genai-toolbox/internal/tools/serverlessspark/serverlesssparkcreatepysparkbatch"
	_ "github.com/googleapis/genai-toolbox/internal/tools/serverlessspark/serverlesssparkcreatesparkbatch"
	_ "github.com/googleapis/genai-toolbox/internal/tools/serverlessspark/serverlesssparkgetbatch"
	_ "github.com/googleapis/genai-toolbox/internal/tools/serverlessspark/serverlesssparklistbatches"
	_ "github.com/googleapis/genai-toolbox/internal/tools/singlestore/singlestoreexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/singlestore/singlestoresql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/snowflake/snowflakeexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/snowflake/snowflakesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/spanner/spannerexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/spanner/spannerlistgraphs"
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
	flags.IntVar(&opts.Cfg.PollInterval, "poll-interval", 0, "Specifies the polling frequency (seconds) for configuration file updates.")

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

// Helper to check if a file has a newer ModTime than stored in the map
func checkModTime(path string, mTime time.Time, lastSeen map[string]time.Time) bool {
	if mTime.After(lastSeen[path]) {
		lastSeen[path] = mTime
		return true
	}
	return false
}

// Helper to scan watched files and check their modification times in polling system
func scanWatchedFiles(watchingFolder bool, folderToWatch string, watchedFiles map[string]bool, lastSeen map[string]time.Time) (map[string]bool, bool, error) {
	changed := false
	currentDiskFiles := make(map[string]bool)
	if watchingFolder {
		files, err := os.ReadDir(folderToWatch)
		if err != nil {
			return nil, changed, fmt.Errorf("error reading tools folder %w", err)
		}
		for _, f := range files {
			if !f.IsDir() && (strings.HasSuffix(f.Name(), ".yaml") || strings.HasSuffix(f.Name(), ".yml")) {
				fullPath := filepath.Join(folderToWatch, f.Name())
				currentDiskFiles[fullPath] = true
				if info, err := f.Info(); err == nil {
					if checkModTime(fullPath, info.ModTime(), lastSeen) {
						changed = true
					}
				}
			}
		}
	} else {
		for f := range watchedFiles {
			if info, err := os.Stat(f); err == nil {
				currentDiskFiles[f] = true
				if checkModTime(f, info.ModTime(), lastSeen) {
					changed = true
				}
			}
		}
	}
	return currentDiskFiles, changed, nil
}

// watchChanges checks for changes in the provided yaml tools file(s) or folder.
func watchChanges(ctx context.Context, watchDirs map[string]bool, watchedFiles map[string]bool, s *server.Server, pollTickerSecond int) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		panic(err)
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		logger.WarnContext(ctx, fmt.Sprintf("error setting up new watcher %s", err))
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

	lastSeen := make(map[string]time.Time)
	var pollTickerChan <-chan time.Time
	if pollTickerSecond > 0 {
		ticker := time.NewTicker(time.Duration(pollTickerSecond) * time.Second)
		defer ticker.Stop()
		pollTickerChan = ticker.C // Assign the channel
		logger.DebugContext(ctx, fmt.Sprintf("NFS polling enabled every %v", pollTickerSecond))

		// Pre-populate lastSeen to avoid an initial spurious reload
		_, _, err = scanWatchedFiles(watchingFolder, folderToWatch, watchedFiles, lastSeen)
		if err != nil {
			logger.WarnContext(ctx, err.Error())
		}
	} else {
		logger.DebugContext(ctx, "NFS polling disabled (interval is 0)")
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
		case <-pollTickerChan:
			// Get files that are currently on disk
			currentDiskFiles, changed, err := scanWatchedFiles(watchingFolder, folderToWatch, watchedFiles, lastSeen)
			if err != nil {
				logger.WarnContext(ctx, err.Error())
				continue
			}

			// Check for Deletions
			// If it was in lastSeen but is NOT in currentDiskFiles, it's
			// deleted; we will need to reload the server.
			for path := range lastSeen {
				if !currentDiskFiles[path] {
					logger.DebugContext(ctx, fmt.Sprintf("File deleted (detected via polling): %s", path))
					delete(lastSeen, path)
					changed = true
				}
			}
			if changed {
				logger.DebugContext(ctx, "File change detected via polling")
				// once this timer runs out, it will trigger debounce.C
				debounce.Reset(debounceDelay)
			}
		case err, ok := <-w.Errors:
			if !ok {
				logger.WarnContext(ctx, "file watcher was closed unexpectedly")
				return
			}
			if err != nil {
				logger.WarnContext(ctx, fmt.Sprintf("file watcher error %s", err))
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
					logger.WarnContext(ctx, fmt.Sprintf("error loading tools folder %s", err))
					continue
				}
			} else {
				logger.DebugContext(ctx, "Reloading tools file(s).")
				reloadedToolsFile, err = internal.LoadAndMergeToolsFiles(ctx, slices.Collect(maps.Keys(watchedFiles)))
				if err != nil {
					logger.WarnContext(ctx, fmt.Sprintf("error loading tools files %s", err))
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
		go watchChanges(ctx, watchDirs, watchedFiles, s, opts.Cfg.PollInterval)
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
