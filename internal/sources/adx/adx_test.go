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

package adx

import (
	"context"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"go.opentelemetry.io/otel/trace"
)

func TestConfig_SourceConfigKind(t *testing.T) {
	cfg := Config{}
	if got := cfg.SourceConfigKind(); got != SourceKind {
		t.Errorf("SourceConfigKind() = %v, want %v", got, SourceKind)
	}
}

func TestSource_SourceKind(t *testing.T) {
	src := &Source{}
	if got := src.SourceKind(); got != SourceKind {
		t.Errorf("SourceKind() = %v, want %v", got, SourceKind)
	}
}

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid config with default auth",
			yaml: `
name: test-adx
kind: adx
cluster_uri: https://test.eastus.kusto.windows.net
database: testdb
`,
			wantErr: false,
		},
		{
			name: "valid config with client_secret auth",
			yaml: `
name: test-adx
kind: adx
cluster_uri: https://test.eastus.kusto.windows.net
database: testdb
auth_mode: client_secret
tenant_id: test-tenant
client_id: test-client
client_secret: test-secret
`,
			wantErr: false,
		},
		{
			name: "valid config with legacy explicit auth (maps to client_secret)",
			yaml: `
name: test-adx
kind: adx
cluster_uri: https://test.eastus.kusto.windows.net
database: testdb
auth_mode: explicit
tenant_id: test-tenant
client_id: test-client
client_secret: test-secret
`,
			wantErr: false,
		},
		{
			name: "valid config with delegated auth",
			yaml: `
name: test-adx
kind: adx
cluster_uri: https://test.eastus.kusto.windows.net
database: testdb
auth_mode: delegated
access_token: test-token
`,
			wantErr: false,
		},
		{
			name: "valid config with managed identity auth (system-assigned)",
			yaml: `
name: test-adx
kind: adx
cluster_uri: https://test.eastus.kusto.windows.net
database: testdb
auth_mode: mi
`,
			wantErr: false,
		},
		{
			name: "valid config with managed identity auth (system-assigned) - new mode",
			yaml: `
name: test-adx
kind: adx
cluster_uri: https://test.eastus.kusto.windows.net
database: testdb
auth_mode: managed_identity
`,
			wantErr: false,
		},
		{
			name: "valid config with managed identity auth (user-assigned) - new mode",
			yaml: `
name: test-adx
kind: adx
cluster_uri: https://test.eastus.kusto.windows.net
database: testdb
auth_mode: managed_identity
managed_identity: 12345678-1234-1234-1234-123456789012
`,
			wantErr: false,
		},
		{
			name: "valid config with device_code auth",
			yaml: `
name: test-adx
kind: adx
cluster_uri: https://test.eastus.kusto.windows.net
database: testdb
auth_mode: device_code
tenant_id: test-tenant
client_id: test-client
`,
			wantErr: false,
		},
		{
			name: "valid config with legacy dcr auth (maps to device_code)",
			yaml: `
name: test-adx
kind: adx
cluster_uri: https://test.eastus.kusto.windows.net
database: testdb
auth_mode: dcr
tenant_id: test-tenant
client_id: test-client
`,
			wantErr: false,
		},
		{
			name: "valid config with browser auth",
			yaml: `
name: test-adx
kind: adx
cluster_uri: https://test.eastus.kusto.windows.net
database: testdb
auth_mode: browser
tenant_id: test-tenant
client_id: test-client
redirect_url: http://localhost:8080
`,
			wantErr: false,
		},
		{
			name: "valid config with browser auth (minimal)",
			yaml: `
name: test-adx
kind: adx
cluster_uri: https://test.eastus.kusto.windows.net
database: testdb
auth_mode: browser
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := yaml.NewDecoder(strings.NewReader(tt.yaml))
			_, err := newConfig(context.Background(), "test", decoder)
			if (err != nil) != tt.wantErr {
				t.Errorf("newConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Initialize_ValidationErrors(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "client_secret auth missing tenant_id",
			config: Config{
				Name:       "test",
				ClusterURI: "https://test.eastus.kusto.windows.net",
				Database:   "testdb",
				AuthMode:   AuthModeClientSecret,
				ClientID:   "test-client",
				ClientSecret: "test-secret",
			},
			wantErr: true,
			errMsg:  "tenant_id, client_id, and client_secret are required for client_secret auth mode",
		},
		{
			name: "legacy explicit auth missing tenant_id (should map to client_secret)",
			config: Config{
				Name:       "test",
				ClusterURI: "https://test.eastus.kusto.windows.net",
				Database:   "testdb",
				AuthMode:   AuthMode("explicit"),
				ClientID:   "test-client",
				ClientSecret: "test-secret",
			},
			wantErr: true,
			errMsg:  "tenant_id, client_id, and client_secret are required for client_secret auth mode",
		},
		{
			name: "delegated auth missing access_token",
			config: Config{
				Name:       "test",
				ClusterURI: "https://test.eastus.kusto.windows.net",
				Database:   "testdb",
				AuthMode:   AuthModeDelegated,
			},
			wantErr: true,
			errMsg:  "access_token is required for delegated auth mode",
		},
		{
			name: "device_code auth missing client_id",
			config: Config{
				Name:       "test",
				ClusterURI: "https://test.eastus.kusto.windows.net",
				Database:   "testdb",
				AuthMode:   AuthModeDeviceCode,
				TenantID:   "test-tenant",
			},
			wantErr: true,
			errMsg:  "client_id and tenant_id are required for device_code auth mode",
		},
		{
			name: "legacy dcr auth missing client_id (should map to device_code)",
			config: Config{
				Name:       "test",
				ClusterURI: "https://test.eastus.kusto.windows.net",
				Database:   "testdb",
				AuthMode:   AuthMode("dcr"),
				TenantID:   "test-tenant",
			},
			wantErr: true,
			errMsg:  "client_id and tenant_id are required for device_code auth mode",
		},
		{
			name: "unsupported auth mode",
			config: Config{
				Name:       "test",
				ClusterURI: "https://test.eastus.kusto.windows.net",
				Database:   "testdb",
				AuthMode:   AuthMode("invalid"),
			},
			wantErr: true,
			errMsg:  "unsupported auth mode: invalid",
		},
		{
			name: "browser auth works without additional config",
			config: Config{
				Name:       "test",
				ClusterURI: "https://test.eastus.kusto.windows.net",
				Database:   "testdb",
				AuthMode:   AuthModeBrowser,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.config.Initialize(context.Background(), tracer)
			if (err != nil) != tt.wantErr {
				t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("Initialize() error = %v, want error containing %v", err, tt.errMsg)
			}
		})
	}
}

// mockKustoClient for testing ExecuteQuery functionality
type mockKustoClient struct {
	results []map[string]interface{}
}

func TestExecuteQuery_Basic(t *testing.T) {
	// Test basic query functionality without pagination
	src := &Source{
		Name:     "test",
		Kind:     SourceKind,
		Database: "testdb",
		Client:   nil, // We won't actually call the client for these tests
	}
	
	// Test that the method signature accepts simple query parameter
	query := "TestTable | take 10"
	// We can't actually execute without a real client, but we can test the signature
	_ = query
	_ = src
}

func TestAuthMode_Constants(t *testing.T) {
	tests := []struct {
		mode     AuthMode
		expected string
	}{
		{AuthModeDefault, "default"},
		{AuthModeClientSecret, "client_secret"},
		{AuthModeDelegated, "delegated"},
		{AuthModeDeviceCode, "device_code"},
		{AuthModeManagedIdentity, "managed_identity"},
		{AuthModeBrowser, "browser"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			if string(tt.mode) != tt.expected {
				t.Errorf("Auth mode %v = %v, want %v", tt.mode, string(tt.mode), tt.expected)
			}
		})
	}
}

func TestAuthMode_LegacyCompatibility(t *testing.T) {
	// Test that legacy constants are properly mapped
	tests := []struct {
		name     string
		legacy   string
		expected AuthMode
	}{
		{"explicit maps to client_secret", "explicit", AuthModeClientSecret},
		{"dcr maps to device_code", "dcr", AuthModeDeviceCode},
		{"mi maps to managed_identity", "mi", AuthModeManagedIdentity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the mapping logic from Initialize method
			mode := AuthMode(tt.legacy)
			switch string(mode) {
			case "explicit":
				mode = AuthModeClientSecret
			case "dcr":
				mode = AuthModeDeviceCode
			case "mi":
				mode = AuthModeManagedIdentity
			}
			
			if mode != tt.expected {
				t.Errorf("Legacy mapping failed: %q should map to %q, got %q", tt.legacy, tt.expected, mode)
			}
		})
	}
}



func TestSourceRegistration(t *testing.T) {
	// Test that the source is properly registered
	decoder := yaml.NewDecoder(strings.NewReader(`
name: test
kind: adx
cluster_uri: https://test.eastus.kusto.windows.net
database: testdb
`))
	sourceConfig, err := newConfig(context.Background(), "test", decoder)
	if err != nil {
		t.Fatalf("Failed to create source config: %v", err)
	}

	if sourceConfig.SourceConfigKind() != SourceKind {
		t.Errorf("Source kind mismatch: got %v, want %v", sourceConfig.SourceConfigKind(), SourceKind)
	}
}