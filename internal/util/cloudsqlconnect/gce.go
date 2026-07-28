// Copyright 2025 Google LLC
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
	"fmt"
	"strings"
)

// ParseConnectionName splits a Cloud SQL instance connection name
// ("project:region:instance") into its three components, rejecting any input
// that doesn't have exactly three non-empty parts.
func ParseConnectionName(connName string) (project, region, instance string, err error) {
	parts := strings.Split(connName, ":")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid connection name format %q: expected project:region:instance", connName)
	}
	if parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("invalid connection name %q: project, region, and instance must all be non-empty", connName)
	}
	return parts[0], parts[1], parts[2], nil
}

// ExtractNetworkName pulls the trailing element off a fully-qualified network
// or subnetwork resource path. Idempotent for inputs that are already a name.
func ExtractNetworkName(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		return path
	}
	return path[idx+1:]
}

// IsSameVPC reports whether the Cloud SQL VPC and the GCE VM VPC resolve to
// the same network name.
func IsSameVPC(sqlVPC, vmVPC string) bool {
	sqlNet := ExtractNetworkName(sqlVPC)
	return sqlNet != "" && sqlNet == vmVPC
}

// ToolSlug returns the slug used in tool kinds for a given engine
// ("postgres", "mysql", "mssql"). Used in user-facing error messages that
// point at the correct sibling tool when a user invokes the wrong one.
func ToolSlug(dt DatabaseType) string {
	switch dt {
	case PostgreSQL:
		return "postgres"
	case MySQL:
		return "mysql"
	case SQLServer:
		return "mssql"
	default:
		return string(dt)
	}
}

// AssertEngine returns nil when sqlInfo's DatabaseVersion matches expected,
// or an error pointing the caller at the correct sibling tool otherwise.
// Tools call this immediately after fetching the Cloud SQL instance so a
// Postgres-tool-on-MySQL-instance (or similar) surfaces a clear diagnostic
// instead of silently producing wrong code snippets.
func AssertEngine(expected DatabaseType, instanceName, databaseVersion string) error {
	got := ParseDatabaseType(databaseVersion)
	if got == expected {
		return nil
	}
	return fmt.Errorf(
		"instance %q is %s (DatabaseVersion=%s); use the cloud-sql-%s-connect-gce tool instead",
		instanceName, got, databaseVersion, ToolSlug(got),
	)
}
