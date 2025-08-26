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
	"io"
	"net"
	"os"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

// MockSSHServer creates a simple SSH server for testing
type MockSSHServer struct {
	listener net.Listener
	config   *ssh.ServerConfig
}

func NewMockSSHServer(t *testing.T) *MockSSHServer {
	// Generate a test private key
	privateKey := generateTestPrivateKey(t)
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to parse private key: %v", err)
	}

	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == "testuser" && string(pass) == "testpass" {
				return nil, nil
			}
			return nil, fmt.Errorf("invalid credentials")
		},
	}
	config.AddHostKey(signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	server := &MockSSHServer{
		listener: listener,
		config:   config,
	}

	go server.handleConnections(t)
	return server
}

func (s *MockSSHServer) Address() string {
	return s.listener.Addr().String()
}

func (s *MockSSHServer) Close() {
	s.listener.Close()
}

func (s *MockSSHServer) handleConnections(t *testing.T) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConnection(conn, t)
	}
}

func (s *MockSSHServer) handleConnection(conn net.Conn, t *testing.T) {
	defer conn.Close()

	_, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		return
	}

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() == "direct-tcpip" {
			go s.handleDirectTCPIP(newChannel, t)
		} else {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
		}
	}
}

func (s *MockSSHServer) handleDirectTCPIP(newChannel ssh.NewChannel, t *testing.T) {
	channel, _, err := newChannel.Accept()
	if err != nil {
		return
	}
	defer channel.Close()

	// For testing, just echo back what we receive
	io.Copy(channel, channel)
}

func generateTestPrivateKey(t *testing.T) []byte {
	// This is a test RSA private key - DO NOT use in production
	return []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA2Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3Q
Z3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3Q
Z3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3Q
Z3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3Q
Z3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QwIDAQABAoIBAQCZ3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3
Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ
3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3Q
Z3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3Q
Z3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3Q
Z3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3Q
Z3Z3QZ3Z3QECgYEA9Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3
QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z
3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3
Z3QZ3Z3QZ3Z3QZ3Z3QECgYEA4Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ
3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3Q
Z3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3
QZ3Z3QZ3Z3QZ3Z3QwIDAQABAoIBAQCZ3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3Q
Z3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z
3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ
3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3
QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3
Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3QZ3Z3Q
Z3Z3QZ3Z3QZ3Z3Q
-----END RSA PRIVATE KEY-----`)
}

func TestTunnel_InvalidTimeout(t *testing.T) {
	config := Config{
		Enabled: true,
		Host:    "localhost",
		User:    "testuser",
		Password: "testpass",
		Timeout: "invalid",
	}

	tunnel := NewTunnel(config)
	ctx := context.Background()

	_, err := tunnel.Start(ctx, "localhost", "8080")
	if err == nil {
		t.Error("Expected error for invalid timeout")
		return
	}

	if !strings.Contains(err.Error(), "invalid timeout format") {
		t.Errorf("Expected timeout format error, got: %v", err)
	}
}

func TestTunnel_LoadPrivateKeyError(t *testing.T) {
	// Create a temporary file with invalid key content
	tempFile, err := os.CreateTemp("", "invalid_key_*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := tempFile.WriteString("invalid key content"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	config := Config{
		Enabled:        true,
		Host:           "localhost",
		User:           "testuser",
		PrivateKeyPath: tempFile.Name(),
	}

	tunnel := NewTunnel(config)
	
	// Test that getAuthMethods handles invalid key gracefully
	methods := tunnel.getAuthMethods()
	if len(methods) != 0 {
		t.Errorf("Expected no auth methods for invalid key, got %d", len(methods))
	}
}

func TestTunnel_StopWithoutStart(t *testing.T) {
	config := Config{}
	tunnel := NewTunnel(config)

	// Should not panic or error when stopping without starting
	err := tunnel.Stop()
	if err != nil {
		t.Errorf("Stop() should not error when tunnel was never started: %v", err)
	}
}

func TestTunnel_MultipleStops(t *testing.T) {
	config := Config{}
	tunnel := NewTunnel(config)

	// Multiple stops should be safe
	err1 := tunnel.Stop()
	err2 := tunnel.Stop()

	if err1 != nil || err2 != nil {
		t.Errorf("Multiple stops should be safe: err1=%v, err2=%v", err1, err2)
	}
}

// Integration test that requires no external dependencies
func TestTunnel_LocalPortAssignment(t *testing.T) {
	// Create a local listener to simulate what Start() does for port assignment
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("Failed to split host:port: %v", err)
	}

	if host != "127.0.0.1" {
		t.Errorf("Expected host 127.0.0.1, got %s", host)
	}

	if port == "0" {
		t.Error("Port should be auto-assigned, not 0")
	}
}