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

package clickhouse

import (
	"context"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"go.opentelemetry.io/otel"
)

func TestConfig_SourceConfigKind(t *testing.T) {
	config := Config{}
	if config.SourceConfigKind() != SourceKind {
		t.Errorf("Expected %s, got %s", SourceKind, config.SourceConfigKind())
	}
}

func TestNewConfig(t *testing.T) {
	yamlContent := `
name: test-clickhouse
kind: clickhouse
host: localhost
port: "9000"
user: default
password: ""
database: default
protocol: native
secure: false
compression: lz4
`

	decoder := yaml.NewDecoder(strings.NewReader(yamlContent))
	config, err := newConfig(context.Background(), "test-clickhouse", decoder)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	clickhouseConfig, ok := config.(Config)
	if !ok {
		t.Fatalf("Expected Config type, got %T", config)
	}

	if clickhouseConfig.Name != "test-clickhouse" {
		t.Errorf("Expected name 'test-clickhouse', got %s", clickhouseConfig.Name)
	}
	if clickhouseConfig.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got %s", clickhouseConfig.Host)
	}
	if clickhouseConfig.Port != "9000" {
		t.Errorf("Expected port '9000', got %s", clickhouseConfig.Port)
	}
	if clickhouseConfig.User != "default" {
		t.Errorf("Expected user 'default', got %s", clickhouseConfig.User)
	}
	if clickhouseConfig.Database != "default" {
		t.Errorf("Expected database 'default', got %s", clickhouseConfig.Database)
	}
	if clickhouseConfig.Protocol != "native" {
		t.Errorf("Expected protocol 'native', got %s", clickhouseConfig.Protocol)
	}
	if clickhouseConfig.Secure != false {
		t.Errorf("Expected secure false, got %t", clickhouseConfig.Secure)
	}
	if clickhouseConfig.Compression != "lz4" {
		t.Errorf("Expected compression 'lz4', got %s", clickhouseConfig.Compression)
	}
}

func TestSource_SourceKind(t *testing.T) {
	source := &Source{}
	if source.SourceKind() != SourceKind {
		t.Errorf("Expected %s, got %s", SourceKind, source.SourceKind())
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		protocol    string
		compression string
		expectError bool
	}{
		{
			name:        "valid native protocol",
			protocol:    "native",
			compression: "lz4",
			expectError: false,
		},
		{
			name:        "valid http protocol",
			protocol:    "http",
			compression: "gzip",
			expectError: false,
		},
		{
			name:        "invalid protocol",
			protocol:    "invalid",
			compression: "lz4",
			expectError: true,
		},
		{
			name:        "invalid compression",
			protocol:    "native",
			compression: "invalid",
			expectError: true,
		},
		{
			name:        "empty values use defaults",
			protocol:    "",
			compression: "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.protocol, tt.compression)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestInitClickHouseConnectionPool_DSNGeneration(t *testing.T) {
	tracer := otel.Tracer("test")
	ctx := context.Background()

	tests := []struct {
		name        string
		host        string
		port        string
		user        string
		pass        string
		dbname      string
		protocol    string
		secure      bool
		compression string
		shouldErr   bool
	}{
		{
			name:        "native protocol",
			host:        "localhost",
			port:        "9000",
			user:        "default",
			pass:        "",
			dbname:      "default",
			protocol:    "native",
			secure:      false,
			compression: "lz4",
			shouldErr:   true, // will fail to connect but DSN should be valid
		},
		{
			name:        "http protocol",
			host:        "localhost",
			port:        "8123",
			user:        "default",
			pass:        "",
			dbname:      "default",
			protocol:    "http",
			secure:      false,
			compression: "lz4",
			shouldErr:   true, // will fail to connect but DSN should be valid
		},
		{
			name:        "https protocol",
			host:        "localhost",
			port:        "8443",
			user:        "default",
			pass:        "",
			dbname:      "default",
			protocol:    "https",
			secure:      true,
			compression: "gzip",
			shouldErr:   true, // will fail to connect but DSN should be valid
		},
		{
			name:        "special characters in password",
			host:        "localhost",
			port:        "9000",
			user:        "test@user",
			pass:        "pass@word:with/special&chars",
			dbname:      "default",
			protocol:    "native",
			secure:      false,
			compression: "lz4",
			shouldErr:   true, // will fail to connect but DSN should be valid
		},
		{
			name:        "invalid protocol should fail",
			host:        "localhost",
			port:        "9000",
			user:        "default",
			pass:        "",
			dbname:      "default",
			protocol:    "invalid",
			secure:      false,
			compression: "lz4",
			shouldErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := initClickHouseConnectionPool(ctx, tracer, "test", tt.host, tt.port, tt.user, tt.pass, tt.dbname, tt.protocol, tt.secure, tt.compression)

			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if pool != nil {
				pool.Close()
			}
		})
	}
}
