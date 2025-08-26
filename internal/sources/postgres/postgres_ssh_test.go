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

package postgres

import (
	"context"
	"strings"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/sources/ssh"
	"go.opentelemetry.io/otel/trace"
)

func TestConfig_InitializeWithoutSSH(t *testing.T) {
	config := Config{
		Name:     "test-postgres",
		Kind:     "postgres",
		Host:     "localhost",
		Port:     "5432",
		User:     "testuser",
		Password: "testpass",
		Database: "testdb",
		SSH:      nil, // No SSH configuration
	}

	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("test")

	// This will fail because we don't have a real PostgreSQL server,
	// but we can verify SSH handling works correctly
	_, err := config.Initialize(ctx, tracer)
	
	// Should get a connection error, not SSH error
	if err != nil && strings.Contains(err.Error(), "SSH") {
		t.Errorf("Should not get SSH error when SSH is disabled, got: %v", err)
	}
}

func TestConfig_InitializeWithDisabledSSH(t *testing.T) {
	config := Config{
		Name:     "test-postgres",
		Kind:     "postgres",
		Host:     "localhost",
		Port:     "5432",
		User:     "testuser",
		Password: "testpass",
		Database: "testdb",
		SSH: &ssh.Config{
			Enabled: false,
		},
	}

	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("test")

	// This will fail because we don't have a real PostgreSQL server,
	// but we can verify SSH handling works correctly
	_, err := config.Initialize(ctx, tracer)
	
	// Should get a connection error, not SSH error
	if err != nil && strings.Contains(err.Error(), "SSH") {
		t.Errorf("Should not get SSH error when SSH is disabled, got: %v", err)
	}
}

func TestConfig_InitializeWithInvalidSSH(t *testing.T) {
	tests := []struct {
		name      string
		sshConfig *ssh.Config
		wantErr   string
	}{
		{
			name: "missing SSH host",
			sshConfig: &ssh.Config{
				Enabled:  true,
				User:     "deploy",
				Password: "secret",
			},
			wantErr: "ssh.host is required when SSH is enabled",
		},
		{
			name: "missing SSH user",
			sshConfig: &ssh.Config{
				Enabled:  true,
				Host:     "bastion.example.com",
				Password: "secret",
			},
			wantErr: "ssh.user is required when SSH is enabled",
		},
		{
			name: "missing SSH authentication",
			sshConfig: &ssh.Config{
				Enabled: true,
				Host:    "bastion.example.com",
				User:    "deploy",
			},
			wantErr: "either ssh.password or ssh.private_key_path must be provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Name:     "test-postgres",
				Kind:     "postgres",
				Host:     "localhost",
				Port:     "5432",
				User:     "testuser",
				Password: "testpass",
				Database: "testdb",
				SSH:      tt.sshConfig,
			}

			ctx := context.Background()
			tracer := trace.NewNoopTracerProvider().Tracer("test")

			_, err := config.Initialize(ctx, tracer)
			if err == nil {
				t.Errorf("Expected SSH validation error")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error containing '%v', got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestConfig_InitializeWithSSHConnectionFailure(t *testing.T) {
	config := Config{
		Name:     "test-postgres",
		Kind:     "postgres",
		Host:     "localhost",
		Port:     "5432",
		User:     "testuser",
		Password: "testpass",
		Database: "testdb",
		SSH: &ssh.Config{
			Enabled:  true,
			Host:     "nonexistent-bastion.invalid",
			User:     "deploy",
			Password: "secret",
			Timeout:  "1s", // Short timeout for faster test
		},
	}

	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("test")

	_, err := config.Initialize(ctx, tracer)
	if err == nil {
		t.Error("Expected SSH connection error")
		return
	}

	// Should get an SSH connection failure
	if !strings.Contains(err.Error(), "failed to establish SSH tunnel") {
		t.Errorf("Expected SSH tunnel error, got: %v", err)
	}
}

func TestSource_Close(t *testing.T) {
	// Create a source with a mock cleanup function
	cleanupCalled := false
	source := &Source{
		Name: "test-source",
		Kind: SourceKind,
		Pool: nil, // No actual pool for this test
		cleanup: func() error {
			cleanupCalled = true
			return nil
		},
	}

	err := source.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	if !cleanupCalled {
		t.Error("Cleanup function should have been called")
	}
}

func TestSource_CloseWithoutCleanup(t *testing.T) {
	// Create a source without cleanup function
	source := &Source{
		Name:    "test-source",
		Kind:    SourceKind,
		Pool:    nil, // No actual pool for this test
		cleanup: nil, // No cleanup function
	}

	err := source.Close()
	if err != nil {
		t.Errorf("Close() should not return error when cleanup is nil: %v", err)
	}
}

func TestSource_CloseWithCleanupError(t *testing.T) {
	// Create a source with a cleanup function that returns an error
	expectedErr := "cleanup failed"
	source := &Source{
		Name: "test-source",
		Kind: SourceKind,
		Pool: nil, // No actual pool for this test
		cleanup: func() error {
			return &testError{msg: expectedErr}
		},
	}

	err := source.Close()
	if err == nil {
		t.Error("Close() should return error when cleanup fails")
		return
	}

	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Expected error containing '%v', got: %v", expectedErr, err)
	}
}

// Helper error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}