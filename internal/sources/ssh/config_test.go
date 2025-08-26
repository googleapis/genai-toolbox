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
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "disabled SSH config is valid",
			config: Config{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "valid SSH config with password",
			config: Config{
				Enabled:  true,
				Host:     "bastion.example.com",
				User:     "deploy",
				Password: "secret",
			},
			wantErr: false,
		},
		{
			name: "valid SSH config with private key",
			config: Config{
				Enabled:        true,
				Host:           "bastion.example.com",
				User:           "deploy",
				PrivateKeyPath: "/path/to/key",
			},
			wantErr: false,
		},
		{
			name: "valid SSH config with both password and key",
			config: Config{
				Enabled:        true,
				Host:           "bastion.example.com",
				User:           "deploy",
				Password:       "secret",
				PrivateKeyPath: "/path/to/key",
			},
			wantErr: false,
		},
		{
			name: "missing host",
			config: Config{
				Enabled:  true,
				User:     "deploy",
				Password: "secret",
			},
			wantErr: true,
			errMsg:  "ssh.host is required when SSH is enabled",
		},
		{
			name: "missing user",
			config: Config{
				Enabled:  true,
				Host:     "bastion.example.com",
				Password: "secret",
			},
			wantErr: true,
			errMsg:  "ssh.user is required when SSH is enabled",
		},
		{
			name: "missing both password and private key",
			config: Config{
				Enabled: true,
				Host:    "bastion.example.com",
				User:    "deploy",
			},
			wantErr: true,
			errMsg:  "either ssh.password or ssh.private_key_path must be provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("Config.Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestConfig_PortWithDefault(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name:     "default port when empty",
			config:   Config{},
			expected: "22",
		},
		{
			name: "custom port",
			config: Config{
				Port: "2222",
			},
			expected: "2222",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.PortWithDefault(); got != tt.expected {
				t.Errorf("Config.PortWithDefault() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_TimeoutWithDefault(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name:     "default timeout when empty",
			config:   Config{},
			expected: "30s",
		},
		{
			name: "custom timeout",
			config: Config{
				Timeout: "60s",
			},
			expected: "60s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.TimeoutWithDefault(); got != tt.expected {
				t.Errorf("Config.TimeoutWithDefault() = %v, want %v", got, tt.expected)
			}
		})
	}
}