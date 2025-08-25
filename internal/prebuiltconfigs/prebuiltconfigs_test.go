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

package prebuiltconfigs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var expectedToolSources = []string{
	"alloydb-postgres-admin",
	"alloydb-postgres",
	"bigquery",
  "clickhouse",
	"cloud-sql-mssql",
	"cloud-sql-mysql",
	"cloud-sql-postgres",
	"dataplex",
	"firestore",
	"looker",
	"mssql",
	"mysql",
	"oceanbase",
	"postgres",
	"spanner-postgres",
	"spanner",
}

func TestGetPrebuiltSources(t *testing.T) {
	t.Run("Test Get Prebuilt Sources", func(t *testing.T) {
		sources := GetPrebuiltSources()
		if diff := cmp.Diff(expectedToolSources, sources); diff != "" {
			t.Fatalf("incorrect sources parse: diff %v", diff)
		}

	})
}

func TestLoadPrebuiltToolYAMLs(t *testing.T) {
	test_name := "test load prebuilt configs"
	expectedKeys := expectedToolSources
	t.Run(test_name, func(t *testing.T) {
		configsMap, keys, err := loadPrebuiltToolYAMLs()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		foundExpectedKeys := make(map[string]bool)

		if len(expectedKeys) != len(configsMap) {
			t.Fatalf("Failed to load all prebuilt tools.")
		}

		for _, expectedKey := range expectedKeys {
			_, ok := configsMap[expectedKey]
			if !ok {
				t.Fatalf("Prebuilt tools for '%s' was NOT FOUND in the loaded map.", expectedKey)
			} else {
				foundExpectedKeys[expectedKey] = true // Mark as found
			}
		}

		t.Log(expectedKeys)
		t.Log(keys)

		if diff := cmp.Diff(expectedKeys, keys); diff != "" {
			t.Fatalf("incorrect sources parse: diff %v", diff)
		}

	})
}

func TestGetPrebuiltTool(t *testing.T) {
	alloydb_admin_config, _ := Get("alloydb-postgres-admin")
	alloydb_config, _ := Get("alloydb-postgres")
	bigquery_config, _ := Get("bigquery")
	clickhouse_config, _ := Get("clickhouse")
	cloudsqlpg_config, _ := Get("cloud-sql-postgres")
	cloudsqlmysql_config, _ := Get("cloud-sql-mysql")
	cloudsqlmssql_config, _ := Get("cloud-sql-mssql")
	dataplex_config, _ := Get("dataplex")
	firestoreconfig, _ := Get("firestore")
	mysql_config, _ := Get("mysql")
	mssql_config, _ := Get("mssql")
	oceanbase_config, _ := Get("oceanbase")
	postgresconfig, _ := Get("postgres")
	spanner_config, _ := Get("spanner")
	spannerpg_config, _ := Get("spanner-postgres")
	if len(alloydb_admin_config) <= 0 {
		t.Fatalf("unexpected error: could not fetch alloydb prebuilt tools yaml")
	}
	if len(alloydb_config) <= 0 {
		t.Fatalf("unexpected error: could not fetch alloydb prebuilt tools yaml")
	}
	if len(bigquery_config) <= 0 {
		t.Fatalf("unexpected error: could not fetch bigquery prebuilt tools yaml")
	}
	if len(clickhouse_config) <= 0 {
		t.Fatalf("unexpected error: could not fetch clickhouse prebuilt tools yaml")
	}
	if len(cloudsqlpg_config) <= 0 {
		t.Fatalf("unexpected error: could not fetch cloud sql pg prebuilt tools yaml")
	}
	if len(cloudsqlmysql_config) <= 0 {
		t.Fatalf("unexpected error: could not fetch cloud sql mysql prebuilt tools yaml")
	}
	if len(cloudsqlmssql_config) <= 0 {
		t.Fatalf("unexpected error: could not fetch cloud sql mssql prebuilt tools yaml")
	}
	if len(dataplex_config) <= 0 {
		t.Fatalf("unexpected error: could not fetch dataplex prebuilt tools yaml")
	}
	if len(firestoreconfig) <= 0 {
		t.Fatalf("unexpected error: could not fetch firestore prebuilt tools yaml")
	}
	if len(mysql_config) <= 0 {
		t.Fatalf("unexpected error: could not fetch mysql prebuilt tools yaml")
	}
	if len(mssql_config) <= 0 {
		t.Fatalf("unexpected error: could not fetch mssql prebuilt tools yaml")
	}
	if len(oceanbase_config) <= 0 {
		t.Fatalf("unexpected error: could not fetch oceanbase prebuilt tools yaml")
	}
	if len(postgresconfig) <= 0 {
		t.Fatalf("unexpected error: could not fetch postgres prebuilt tools yaml")
	}
	if len(spanner_config) <= 0 {
		t.Fatalf("unexpected error: could not fetch spanner prebuilt tools yaml")
	}
	if len(spannerpg_config) <= 0 {
		t.Fatalf("unexpected error: could not fetch spanner pg prebuilt tools yaml")
	}
}

func TestFailGetPrebuiltTool(t *testing.T) {
	_, err := Get("sql")
	if err == nil {
		t.Fatalf("unexpected an error but got nil.")
	}
}
