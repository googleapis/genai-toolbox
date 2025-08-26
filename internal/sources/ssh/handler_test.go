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
	"testing"
)

func TestResolveConnection_NoSSH(t *testing.T) {
	ctx := context.Background()
	
	// Test with nil SSH config
	host, port, cleanup, err := ResolveConnection(ctx, "localhost", "5432", nil)
	if err != nil {
		t.Fatalf("ResolveConnection() error = %v, want nil", err)
	}
	defer cleanup()
	
	if host != "localhost" {
		t.Errorf("ResolveConnection() host = %v, want localhost", host)
	}
	if port != "5432" {
		t.Errorf("ResolveConnection() port = %v, want 5432", port)
	}
	
	// Test with disabled SSH config
	sshConfig := &Config{Enabled: false}
	host, port, cleanup, err = ResolveConnection(ctx, "db.example.com", "3306", sshConfig)
	if err != nil {
		t.Fatalf("ResolveConnection() error = %v, want nil", err)
	}
	defer cleanup()
	
	if host != "db.example.com" {
		t.Errorf("ResolveConnection() host = %v, want db.example.com", host)
	}
	if port != "3306" {
		t.Errorf("ResolveConnection() port = %v, want 3306", port)
	}
}

func TestResolveConnection_InvalidSSHConfig(t *testing.T) {
	ctx := context.Background()
	
	tests := []struct {
		name      string
		sshConfig *Config
		wantErr   string
	}{
		{
			name: "missing host",
			sshConfig: &Config{
				Enabled:  true,
				User:     "deploy",
				Password: "secret",
			},
			wantErr: "invalid SSH configuration: ssh.host is required when SSH is enabled",
		},
		{
			name: "missing user",
			sshConfig: &Config{
				Enabled:  true,
				Host:     "bastion.example.com",
				Password: "secret",
			},
			wantErr: "invalid SSH configuration: ssh.user is required when SSH is enabled",
		},
		{
			name: "missing authentication",
			sshConfig: &Config{
				Enabled: true,
				Host:    "bastion.example.com",
				User:    "deploy",
			},
			wantErr: "invalid SSH configuration: either ssh.password or ssh.private_key_path must be provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := ResolveConnection(ctx, "localhost", "5432", tt.sshConfig)
			if err == nil {
				t.Errorf("ResolveConnection() error = nil, want error")
				return
			}
			if err.Error() != tt.wantErr {
				t.Errorf("ResolveConnection() error = %v, want %v", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestResolveConnection_SSHConnectionFailure(t *testing.T) {
	ctx := context.Background()
	
	// Use invalid SSH configuration that will fail to connect
	sshConfig := &Config{
		Enabled:  true,
		Host:     "nonexistent-bastion.invalid",
		User:     "deploy",
		Password: "secret",
		Timeout:  "1s", // Short timeout for faster test
	}
	
	_, _, cleanup, err := ResolveConnection(ctx, "localhost", "5432", sshConfig)
	if err == nil {
		defer cleanup()
		t.Error("ResolveConnection() error = nil, want SSH connection error")
		return
	}
	
	// Should get an SSH connection failure
	if err.Error() == "" {
		t.Errorf("ResolveConnection() error message should not be empty")
	}
	
	// Error should mention SSH tunnel failure
	expectedSubstring := "failed to establish SSH tunnel"
	if len(err.Error()) < len(expectedSubstring) || err.Error()[:len(expectedSubstring)] != expectedSubstring {
		t.Errorf("ResolveConnection() error = %v, should start with %v", err.Error(), expectedSubstring)
	}
}