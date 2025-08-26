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

//go:build integration

package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/sources/postgres"
	"github.com/googleapis/genai-toolbox/internal/sources/ssh"
	"go.opentelemetry.io/otel/trace"
)

// TestPostgresSSHIntegration tests PostgreSQL connection through SSH tunnel
// This test requires environment variables to be set for SSH and PostgreSQL configuration
func TestPostgresSSHIntegration(t *testing.T) {
	// Skip if not running integration tests with SSH setup
	if os.Getenv("POSTGRES_SSH_TEST") != "true" {
		t.Skip("Skipping SSH integration test - set POSTGRES_SSH_TEST=true to run")
	}

	// Required environment variables for SSH tunnel
	sshHost := os.Getenv("SSH_HOST")
	sshUser := os.Getenv("SSH_USER")
	sshPassword := os.Getenv("SSH_PASSWORD")
	sshKeyPath := os.Getenv("SSH_PRIVATE_KEY_PATH")

	// Required environment variables for PostgreSQL
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DATABASE")

	if sshHost == "" || sshUser == "" {
		t.Skip("SSH_HOST and SSH_USER must be set for SSH integration tests")
	}

	if sshPassword == "" && sshKeyPath == "" {
		t.Skip("Either SSH_PASSWORD or SSH_PRIVATE_KEY_PATH must be set")
	}

	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		t.Skip("PostgreSQL environment variables must be set (POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DATABASE)")
	}

	// Create SSH configuration
	sshConfig := &ssh.Config{
		Enabled: true,
		Host:    sshHost,
		Port:    "22",
		User:    sshUser,
		Timeout: "30s",
	}

	// Use password or private key authentication
	if sshPassword != "" {
		sshConfig.Password = sshPassword
	} else {
		sshConfig.PrivateKeyPath = sshKeyPath
	}

	// Create PostgreSQL configuration with SSH
	config := postgres.Config{
		Name:     "postgres-ssh-integration-test",
		Kind:     "postgres",
		Host:     dbHost,
		Port:     dbPort,
		User:     dbUser,
		Password: dbPassword,
		Database: dbName,
		SSH:      sshConfig,
	}

	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("integration-test")

	// Initialize PostgreSQL source with SSH tunnel
	source, err := config.Initialize(ctx, tracer)
	if err != nil {
		t.Fatalf("Failed to initialize PostgreSQL source with SSH: %v", err)
	}

	// Ensure cleanup
	defer func() {
		if pgSource, ok := source.(*postgres.Source); ok {
			pgSource.Close()
		}
	}()

	// Verify source was created successfully
	if source.SourceKind() != "postgres" {
		t.Errorf("Expected source kind 'postgres', got '%s'", source.SourceKind())
	}

	// Test database connectivity through SSH tunnel
	pgSource, ok := source.(*postgres.Source)
	if !ok {
		t.Fatalf("Expected *postgres.Source, got %T", source)
	}

	pool := pgSource.PostgresPool()
	if pool == nil {
		t.Fatal("PostgreSQL pool should not be nil")
	}

	// Execute a simple query to verify connectivity
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Failed to acquire connection from pool: %v", err)
	}
	defer conn.Release()

	var result int
	err = conn.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("Failed to execute test query: %v", err)
	}

	if result != 1 {
		t.Errorf("Expected query result 1, got %d", result)
	}

	t.Logf("Successfully connected to PostgreSQL through SSH tunnel and executed query")
}

// TestPostgresWithoutSSH tests that PostgreSQL works normally without SSH configuration
func TestPostgresWithoutSSH(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("POSTGRES_DIRECT_TEST") != "true" {
		t.Skip("Skipping direct PostgreSQL test - set POSTGRES_DIRECT_TEST=true to run")
	}

	// Required environment variables for direct PostgreSQL connection
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DATABASE")

	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		t.Skip("PostgreSQL environment variables must be set")
	}

	// Create PostgreSQL configuration without SSH
	config := postgres.Config{
		Name:     "postgres-direct-integration-test",
		Kind:     "postgres",
		Host:     dbHost,
		Port:     dbPort,
		User:     dbUser,
		Password: dbPassword,
		Database: dbName,
		SSH:      nil, // No SSH configuration
	}

	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("integration-test")

	// Initialize PostgreSQL source without SSH
	source, err := config.Initialize(ctx, tracer)
	if err != nil {
		t.Fatalf("Failed to initialize PostgreSQL source: %v", err)
	}

	// Ensure cleanup
	defer func() {
		if pgSource, ok := source.(*postgres.Source); ok {
			pgSource.Close()
		}
	}()

	// Test database connectivity
	pgSource, ok := source.(*postgres.Source)
	if !ok {
		t.Fatalf("Expected *postgres.Source, got %T", source)
	}

	pool := pgSource.PostgresPool()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Failed to acquire connection: %v", err)
	}
	defer conn.Release()

	var result int
	err = conn.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("Failed to execute test query: %v", err)
	}

	if result != 1 {
		t.Errorf("Expected query result 1, got %d", result)
	}

	t.Logf("Successfully connected to PostgreSQL directly and executed query")
}

// TestSSHConnectionFailure tests behavior when SSH connection fails
func TestSSHConnectionFailure(t *testing.T) {
	// This test doesn't require real infrastructure
	config := postgres.Config{
		Name:     "postgres-ssh-failure-test",
		Kind:     "postgres",
		Host:     "localhost",
		Port:     "5432",
		User:     "testuser",
		Password: "testpass",
		Database: "testdb",
		SSH: &ssh.Config{
			Enabled:  true,
			Host:     "nonexistent-bastion.invalid",
			User:     "testuser",
			Password: "testpass",
			Timeout:  "1s", // Short timeout for faster test
		},
	}

	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("integration-test")

	// Should fail to initialize due to SSH connection failure
	_, err := config.Initialize(ctx, tracer)
	if err == nil {
		t.Fatal("Expected SSH connection failure")
	}

	// Error should mention SSH tunnel failure
	expected := "failed to establish SSH tunnel"
	if len(err.Error()) < len(expected) || err.Error()[:len(expected)] != expected {
		t.Errorf("Expected error to start with '%s', got: %v", expected, err)
	}

	t.Logf("Correctly handled SSH connection failure: %v", err)
}

// Example of how to run these tests:
func ExampleRunSSHIntegrationTest() {
	// Set environment variables for SSH configuration:
	// export SSH_HOST=bastion.example.com
	// export SSH_USER=deploy  
	// export SSH_PASSWORD=secretpassword
	// # OR
	// export SSH_PRIVATE_KEY_PATH=/path/to/private/key
	
	// Set environment variables for PostgreSQL:
	// export POSTGRES_HOST=db.internal.example.com
	// export POSTGRES_PORT=5432
	// export POSTGRES_USER=dbuser
	// export POSTGRES_PASSWORD=dbpassword  
	// export POSTGRES_DATABASE=testdb
	
	// Enable the test:
	// export POSTGRES_SSH_TEST=true
	
	// Run the test:
	// go test -tags=integration -v ./tests/postgres/
	
	fmt.Println("SSH integration test setup complete")
}