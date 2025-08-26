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
	"time"

	"golang.org/x/crypto/ssh"
)

// Tunnel represents an SSH tunnel for database connections
type Tunnel struct {
	config    Config
	sshClient *ssh.Client
	listener  net.Listener
	localAddr string
	cancel    context.CancelFunc
}

// NewTunnel creates a new SSH tunnel with the given configuration
func NewTunnel(config Config) *Tunnel {
	return &Tunnel{config: config}
}

// Start establishes the SSH tunnel and returns the local address to connect to
func (t *Tunnel) Start(ctx context.Context, remoteHost, remotePort string) (string, error) {
	// Create SSH client connection
	var err error
	t.sshClient, err = t.createSSHClient()
	if err != nil {
		return "", fmt.Errorf("SSH connection failed: %w", err)
	}

	// Create local listener
	listenAddr := fmt.Sprintf("127.0.0.1:%d", t.config.LocalPort)
	t.listener, err = net.Listen("tcp", listenAddr)
	if err != nil {
		t.sshClient.Close()
		return "", fmt.Errorf("failed to create local listener: %w", err)
	}

	t.localAddr = t.listener.Addr().String()

	// Start forwarding goroutine
	ctx, t.cancel = context.WithCancel(ctx)
	go t.forward(ctx, remoteHost, remotePort)

	return t.localAddr, nil
}

// Stop closes the SSH tunnel and cleans up resources
func (t *Tunnel) Stop() error {
	if t.cancel != nil {
		t.cancel()
	}
	if t.listener != nil {
		t.listener.Close()
	}
	if t.sshClient != nil {
		t.sshClient.Close()
	}
	return nil
}

// createSSHClient creates and connects an SSH client
func (t *Tunnel) createSSHClient() (*ssh.Client, error) {
	timeout, err := time.ParseDuration(t.config.TimeoutWithDefault())
	if err != nil {
		return nil, fmt.Errorf("invalid timeout format: %w", err)
	}

	config := &ssh.ClientConfig{
		User:            t.config.User,
		Auth:            t.getAuthMethods(),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Implement proper host key verification
		Timeout:         timeout,
	}

	addr := fmt.Sprintf("%s:%s", t.config.Host, t.config.PortWithDefault())
	return ssh.Dial("tcp", addr, config)
}

// getAuthMethods returns the authentication methods based on configuration
func (t *Tunnel) getAuthMethods() []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	if t.config.Password != "" {
		methods = append(methods, ssh.Password(t.config.Password))
	}

	if t.config.PrivateKeyPath != "" {
		if key, err := t.loadPrivateKey(); err == nil {
			methods = append(methods, ssh.PublicKeys(key))
		}
	}

	return methods
}

// loadPrivateKey loads and parses a private key file
func (t *Tunnel) loadPrivateKey() (ssh.Signer, error) {
	keyBytes, err := os.ReadFile(t.config.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	if t.config.Passphrase != "" {
		return ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(t.config.Passphrase))
	}
	return ssh.ParsePrivateKey(keyBytes)
}

// forward handles incoming connections and forwards them through the SSH tunnel
func (t *Tunnel) forward(ctx context.Context, remoteHost, remotePort string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := t.listener.Accept()
			if err != nil {
				// Listener was likely closed
				return
			}
			go t.handleConnection(conn, remoteHost, remotePort)
		}
	}
}

// handleConnection handles a single connection through the tunnel
func (t *Tunnel) handleConnection(localConn net.Conn, remoteHost, remotePort string) {
	defer localConn.Close()

	// Connect to remote service through SSH tunnel
	remoteAddr := fmt.Sprintf("%s:%s", remoteHost, remotePort)
	remoteConn, err := t.sshClient.Dial("tcp", remoteAddr)
	if err != nil {
		return
	}
	defer remoteConn.Close()

	// Copy data bidirectionally
	done := make(chan struct{}, 2)
	
	go func() {
		io.Copy(remoteConn, localConn)
		done <- struct{}{}
	}()
	
	go func() {
		io.Copy(localConn, remoteConn)
		done <- struct{}{}
	}()

	// Wait for one direction to finish
	<-done
}