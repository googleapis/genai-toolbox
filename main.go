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

package main

import (
	"github.com/googleapis/genai-toolbox/cmd"

	// Import tool packages for side effect of registration
	_ "github.com/googleapis/genai-toolbox/internal/tools/alloydbainl"
	_ "github.com/googleapis/genai-toolbox/internal/tools/bigquery"
	_ "github.com/googleapis/genai-toolbox/internal/tools/bigtable"
	_ "github.com/googleapis/genai-toolbox/internal/tools/couchbase"
	_ "github.com/googleapis/genai-toolbox/internal/tools/dgraph"
	_ "github.com/googleapis/genai-toolbox/internal/tools/http"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mssqlexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mssqlsql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mysqlexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/mysqlsql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/neo4j"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgresexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/postgressql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/spanner"
	_ "github.com/googleapis/genai-toolbox/internal/tools/spannerexecutesql"
	_ "github.com/googleapis/genai-toolbox/internal/tools/sqlitesql"
)

func main() {
	cmd.Execute()
}
