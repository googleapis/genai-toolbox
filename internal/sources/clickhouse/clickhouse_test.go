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

func TestConfigSourceConfigKind(t *testing.T) {
	config := Config{}
	if config.SourceConfigKind() != SourceKind {
		t.Errorf("Expected %s, got %s", SourceKind, config.SourceConfigKind())
	}
}

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected Config
	}{
		{
			name: "all fields specified",
			yaml: `
name: test-clickhouse
kind: clickhouse
host: localhost
port: "8443"
user: default
password: "mypass"
database: mydb
protocol: https
secure: true
`,
			expected: Config{
				Name:     "test-clickhouse",
				Kind:     "clickhouse",
				Host:     "localhost",
				Port:     "8443",
				User:     "default",
				Password: "mypass",
				Database: "mydb",
				Protocol: "https",
				Secure:   true,
			},
		},
		{
			name: "minimal configuration with defaults",
			yaml: `
name: minimal-clickhouse
kind: clickhouse
host: 127.0.0.1
port: "8123"
user: testuser
database: testdb
`,
			expected: Config{
				Name:     "minimal-clickhouse",
				Kind:     "clickhouse",
				Host:     "127.0.0.1",
				Port:     "8123",
				User:     "testuser",
				Password: "",
				Database: "testdb",
				Protocol: "",
				Secure:   false,
			},
		},
		{
			name: "http protocol",
			yaml: `
name: http-clickhouse
kind: clickhouse
host: clickhouse.example.com
port: "8123"
user: analytics
password: "securepass"
database: analytics_db
protocol: http
secure: false
`,
			expected: Config{
				Name:     "http-clickhouse",
				Kind:     "clickhouse",
				Host:     "clickhouse.example.com",
				Port:     "8123",
				User:     "analytics",
				Password: "securepass",
				Database: "analytics_db",
				Protocol: "http",
				Secure:   false,
			},
		},
		{
			name: "https with secure connection",
			yaml: `
name: secure-clickhouse
kind: clickhouse
host: secure.clickhouse.io
port: "8443"
user: secureuser
password: "verysecure"
database: production
protocol: https
secure: true
`,
			expected: Config{
				Name:     "secure-clickhouse",
				Kind:     "clickhouse",
				Host:     "secure.clickhouse.io",
				Port:     "8443",
				User:     "secureuser",
				Password: "verysecure",
				Database: "production",
				Protocol: "https",
				Secure:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := yaml.NewDecoder(strings.NewReader(tt.yaml))
			config, err := newConfig(context.Background(), tt.expected.Name, decoder)
			if err != nil {
				t.Fatalf("Failed to create config: %v", err)
			}

			clickhouseConfig, ok := config.(Config)
			if !ok {
				t.Fatalf("Expected Config type, got %T", config)
			}

			if clickhouseConfig.Name != tt.expected.Name {
				t.Errorf("Name: expected %q, got %q", tt.expected.Name, clickhouseConfig.Name)
			}
			if clickhouseConfig.Kind != tt.expected.Kind {
				t.Errorf("Kind: expected %q, got %q", tt.expected.Kind, clickhouseConfig.Kind)
			}
			if clickhouseConfig.Host != tt.expected.Host {
				t.Errorf("Host: expected %q, got %q", tt.expected.Host, clickhouseConfig.Host)
			}
			if clickhouseConfig.Port != tt.expected.Port {
				t.Errorf("Port: expected %q, got %q", tt.expected.Port, clickhouseConfig.Port)
			}
			if clickhouseConfig.User != tt.expected.User {
				t.Errorf("User: expected %q, got %q", tt.expected.User, clickhouseConfig.User)
			}
			if clickhouseConfig.Password != tt.expected.Password {
				t.Errorf("Password: expected %q, got %q", tt.expected.Password, clickhouseConfig.Password)
			}
			if clickhouseConfig.Database != tt.expected.Database {
				t.Errorf("Database: expected %q, got %q", tt.expected.Database, clickhouseConfig.Database)
			}
			if clickhouseConfig.Protocol != tt.expected.Protocol {
				t.Errorf("Protocol: expected %q, got %q", tt.expected.Protocol, clickhouseConfig.Protocol)
			}
			if clickhouseConfig.Secure != tt.expected.Secure {
				t.Errorf("Secure: expected %v, got %v", tt.expected.Secure, clickhouseConfig.Secure)
			}
		})
	}
}

func TestNewConfigInvalidYAML(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectError bool
	}{
		{
			name: "invalid yaml syntax",
			yaml: `
name: test-clickhouse
kind: clickhouse
host: [invalid
`,
			expectError: true,
		},
		{
			name: "missing required fields",
			yaml: `
name: test-clickhouse
kind: clickhouse
`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := yaml.NewDecoder(strings.NewReader(tt.yaml))
			_, err := newConfig(context.Background(), "test-clickhouse", decoder)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
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
		expectError bool
	}{
		{
			name:        "valid https protocol",
			protocol:    "https",
			expectError: false,
		},
		{
			name:        "valid http protocol",
			protocol:    "http",
			expectError: false,
		},
		{
			name:        "invalid protocol",
			protocol:    "invalid",
			expectError: true,
		},
		{
			name:        "invalid protocol - native not supported",
			protocol:    "native",
			expectError: true,
		},
		{
			name:        "empty values use defaults",
			protocol:    "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.protocol)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestInitClickHouseConnectionPoolDSNGeneration(t *testing.T) {
	tracer := otel.Tracer("test")
	ctx := context.Background()

	tests := []struct {
		name      string
		host      string
		port      string
		user      string
		pass      string
		dbname    string
		protocol  string
		secure    bool
		shouldErr bool
	}{
		{
			name:      "http protocol with defaults",
			host:      "localhost",
			port:      "8123",
			user:      "default",
			pass:      "",
			dbname:    "default",
			protocol:  "http",
			secure:    false,
			shouldErr: true,
		},
		{
			name:      "https protocol with secure",
			host:      "localhost",
			port:      "8443",
			user:      "default",
			pass:      "",
			dbname:    "default",
			protocol:  "https",
			secure:    true,
			shouldErr: true,
		},
		{
			name:      "special characters in password",
			host:      "localhost",
			port:      "8443",
			user:      "test@user",
			pass:      "pass@word:with/special&chars",
			dbname:    "default",
			protocol:  "https",
			secure:    true,
			shouldErr: true,
		},
		{
			name:      "invalid protocol should fail",
			host:      "localhost",
			port:      "9000",
			user:      "default",
			pass:      "",
			dbname:    "default",
			protocol:  "invalid",
			secure:    false,
			shouldErr: true,
		},
		{
			name:      "empty protocol defaults to https",
			host:      "localhost",
			port:      "8443",
			user:      "user",
			pass:      "pass",
			dbname:    "testdb",
			protocol:  "",
			secure:    true,
			shouldErr: true,
		},
		{
			name:      "http with secure flag should upgrade to https",
			host:      "example.com",
			port:      "8443",
			user:      "user",
			pass:      "pass",
			dbname:    "db",
			protocol:  "http",
			secure:    true,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := initClickHouseConnectionPool(ctx, tracer, "test", tt.host, tt.port, tt.user, tt.pass, tt.dbname, tt.protocol, tt.secure)

			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if pool != nil {
				pool.Close()
			}
		})
	}
}
