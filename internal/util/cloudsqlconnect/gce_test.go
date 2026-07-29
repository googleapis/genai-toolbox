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

package cloudsqlconnect

import (
	"strings"
	"testing"
)

func TestParseConnectionName(t *testing.T) {
	tcs := []struct {
		desc        string
		input       string
		wantProject string
		wantRegion  string
		wantInst    string
		wantErr     bool
	}{
		{desc: "valid us", input: "my-project:us-central1:my-instance",
			wantProject: "my-project", wantRegion: "us-central1", wantInst: "my-instance"},
		{desc: "valid eu", input: "project-123:europe-west1:db-prod",
			wantProject: "project-123", wantRegion: "europe-west1", wantInst: "db-prod"},
		{desc: "no separators", input: "invalid-format", wantErr: true},
		{desc: "two parts", input: "only:two", wantErr: true},
		{desc: "all empty parts", input: "::", wantErr: true},
		{desc: "empty project", input: ":region:instance", wantErr: true},
		{desc: "empty region", input: "project::instance", wantErr: true},
		{desc: "empty instance", input: "project:region:", wantErr: true},
		{desc: "four parts", input: "a:b:c:d", wantErr: true},
		{desc: "empty string", input: "", wantErr: true},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			project, region, instance, err := ParseConnectionName(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got none", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if project != tc.wantProject || region != tc.wantRegion || instance != tc.wantInst {
				t.Fatalf("got (%q, %q, %q), want (%q, %q, %q)",
					project, region, instance, tc.wantProject, tc.wantRegion, tc.wantInst)
			}
		})
	}
}

func TestExtractNetworkNameRoundTrip(t *testing.T) {
	tcs := []struct{ in, want string }{
		{"projects/my-project/global/networks/default", "default"},
		{"projects/my-project/regions/us-central1/subnetworks/sn1", "sn1"},
		{"default", "default"},
		{"", ""},
	}
	for _, tc := range tcs {
		if got := ExtractNetworkName(tc.in); got != tc.want {
			t.Errorf("ExtractNetworkName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsSameVPC(t *testing.T) {
	tcs := []struct {
		desc        string
		sqlVPC      string
		vmVPC       string
		wantSameVPC bool
	}{
		{desc: "same name", sqlVPC: "default", vmVPC: "default", wantSameVPC: true},
		{desc: "fully qualified equals short", sqlVPC: "projects/p/global/networks/default", vmVPC: "default", wantSameVPC: true},
		{desc: "different names", sqlVPC: "vpc-a", vmVPC: "vpc-b", wantSameVPC: false},
		{desc: "empty sql", sqlVPC: "", vmVPC: "default", wantSameVPC: false},
		{desc: "empty vm", sqlVPC: "default", vmVPC: "", wantSameVPC: false},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			if got := IsSameVPC(tc.sqlVPC, tc.vmVPC); got != tc.wantSameVPC {
				t.Errorf("IsSameVPC(%q, %q) = %v, want %v", tc.sqlVPC, tc.vmVPC, got, tc.wantSameVPC)
			}
		})
	}
}

func TestToolSlug(t *testing.T) {
	tcs := []struct {
		dt   DatabaseType
		want string
	}{
		{PostgreSQL, "postgres"},
		{MySQL, "mysql"},
		{SQLServer, "mssql"},
		{DatabaseType("future-engine"), "future-engine"},
	}
	for _, tc := range tcs {
		if got := ToolSlug(tc.dt); got != tc.want {
			t.Errorf("ToolSlug(%q) = %q, want %q", tc.dt, got, tc.want)
		}
	}
}

func TestAssertEngine(t *testing.T) {
	tcs := []struct {
		desc     string
		expected DatabaseType
		version  string
		wantErr  bool
		wantHint string
	}{
		{desc: "postgres matches", expected: PostgreSQL, version: "POSTGRES_15"},
		{desc: "mysql matches", expected: MySQL, version: "MYSQL_8_0"},
		{desc: "sqlserver matches", expected: SQLServer, version: "SQLSERVER_2022_STANDARD"},
		{desc: "wrong engine surfaces sibling tool",
			expected: PostgreSQL, version: "MYSQL_8_0", wantErr: true,
			wantHint: "cloud-sql-mysql-connect-gce"},
		{desc: "wrong engine for sqlserver-tool-on-pg",
			expected: SQLServer, version: "POSTGRES_15", wantErr: true,
			wantHint: "cloud-sql-postgres-connect-gce"},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			err := AssertEngine(tc.expected, "my-instance", tc.version)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none")
				}
				if !strings.Contains(err.Error(), tc.wantHint) {
					t.Errorf("error %q should mention %q", err.Error(), tc.wantHint)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
