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

package ssh

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/goccy/go-yaml"
)

// parseEnv replaces environment variables ${ENV_NAME} or ${ENV_NAME:-default} with their values.
// This is a copy of the function from cmd/root.go for testing purposes.
func parseEnv(input string) (string, error) {
	// Support both ${VAR} and ${VAR:-default} syntax
	re := regexp.MustCompile(`\$\{([^}:]+)(?::-([^}]*))?\}`)

	var err error
	output := re.ReplaceAllStringFunc(input, func(match string) string {
		parts := re.FindStringSubmatch(match)

		// extract the variable name and optional default value
		variableName := parts[1]
		defaultValue := ""
		if len(parts) > 2 {
			defaultValue = parts[2]
		}

		if value, found := os.LookupEnv(variableName); found {
			return value
		}
		
		// If no default value and variable not found, return error
		if len(parts) <= 2 {
			err = fmt.Errorf("environment variable not found: %q", variableName)
			return ""
		}
		
		// Return default value
		return defaultValue
	})
	return output, err
}

// TestBooleanFromEnvironmentVariable tests how YAML parsing handles boolean values from environment variables
func TestBooleanFromEnvironmentVariable(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
		wantErr  bool
	}{
		{
			name:     "string 'true' becomes boolean true",
			envValue: "true",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "string 'false' becomes boolean false",
			envValue: "false",
			expected: false,
			wantErr:  false,
		},
		{
			name:     "string 'True' becomes boolean true (case insensitive)",
			envValue: "True",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "string 'FALSE' becomes boolean false (case insensitive)",
			envValue: "FALSE",
			expected: false,
			wantErr:  false,
		},
		{
			name:     "string 'yes' becomes boolean true",
			envValue: "yes",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "string 'no' becomes boolean false",
			envValue: "no",
			expected: false,
			wantErr:  false,
		},
		{
			name:     "string '1' becomes boolean true",
			envValue: "1",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "string '0' becomes boolean false",
			envValue: "0",
			expected: false,
			wantErr:  false,
		},
		{
			name:     "empty string becomes boolean false",
			envValue: "",
			expected: false,
			wantErr:  false,
		},
		{
			name:     "invalid boolean string causes parsing error",
			envValue: "invalid",
			expected: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			os.Setenv("TEST_SSH_ENABLED", tt.envValue)
			defer os.Unsetenv("TEST_SSH_ENABLED")

			// Test YAML with environment variable substitution
			yamlContent := `
enabled: ${TEST_SSH_ENABLED}
host: test.example.com
user: testuser
password: testpass
`

			// Process environment variables first
			processedYaml, parseErr := parseEnv(yamlContent)
			if parseErr != nil {
				t.Errorf("Environment variable processing error = %v", parseErr)
				return
			}

			var config Config
			err := yaml.UnmarshalContext(context.Background(), []byte(processedYaml), &config)

			if (err != nil) != tt.wantErr {
				t.Errorf("YAML unmarshaling error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && bool(config.Enabled) != tt.expected {
				t.Errorf("Config.Enabled = %v, want %v", bool(config.Enabled), tt.expected)
			}
		})
	}
}

// TestEnvironmentVariableDefaults tests default values in environment variable substitution
func TestEnvironmentVariableDefaults(t *testing.T) {
	tests := []struct {
		name         string
		yamlTemplate string
		envVar       string
		envValue     string
		expected     bool
	}{
		{
			name: "default false when environment variable not set",
			yamlTemplate: `
enabled: ${SSH_ENABLED:-false}
host: test.example.com
user: testuser
password: testpass
`,
			envVar:   "SSH_ENABLED",
			envValue: "", // Unset
			expected: false,
		},
		{
			name: "default true when environment variable not set",
			yamlTemplate: `
enabled: ${SSH_ENABLED:-true}
host: test.example.com
user: testuser
password: testpass
`,
			envVar:   "SSH_ENABLED",
			envValue: "", // Unset
			expected: true,
		},
		{
			name: "environment variable overrides default",
			yamlTemplate: `
enabled: ${SSH_ENABLED:-false}
host: test.example.com
user: testuser
password: testpass
`,
			envVar:   "SSH_ENABLED",
			envValue: "true",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set or unset environment variable
			if tt.envValue != "" {
				os.Setenv(tt.envVar, tt.envValue)
			} else {
				os.Unsetenv(tt.envVar)
			}
			defer os.Unsetenv(tt.envVar)

			// Process environment variables first
			processedYaml, parseErr := parseEnv(tt.yamlTemplate)
			if parseErr != nil {
				t.Errorf("Environment variable processing error = %v", parseErr)
				return
			}

			var config Config
			err := yaml.UnmarshalContext(context.Background(), []byte(processedYaml), &config)
			if err != nil {
				t.Errorf("YAML unmarshaling error = %v", err)
				return
			}

			if bool(config.Enabled) != tt.expected {
				t.Errorf("Config.Enabled = %v, want %v", bool(config.Enabled), tt.expected)
			}
		})
	}
}