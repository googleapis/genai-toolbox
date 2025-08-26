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
	"net"
)

// ResolveConnection handles SSH tunneling and returns effective host/port to connect to.
// If SSH is disabled, returns original host/port unchanged.
// If SSH is enabled, establishes tunnel and returns localhost/localport.
// The cleanup function must be called to properly close the tunnel.
func ResolveConnection(ctx context.Context, host, port string, sshConfig *Config) (effectiveHost, effectivePort string, cleanup func() error, err error) {
	// No SSH config or disabled - return original values
	if sshConfig == nil || !bool(sshConfig.Enabled) {
		return host, port, func() error { return nil }, nil
	}

	// Validate SSH config
	if err := sshConfig.Validate(); err != nil {
		return "", "", nil, fmt.Errorf("invalid SSH configuration: %w", err)
	}

	// Create and start SSH tunnel
	tunnel := NewTunnel(*sshConfig)
	localAddr, err := tunnel.Start(ctx, host, port)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to establish SSH tunnel: %w", err)
	}

	// Parse local address to get host and port
	localHost, localPort, err := net.SplitHostPort(localAddr)
	if err != nil {
		tunnel.Stop()
		return "", "", nil, fmt.Errorf("failed to parse tunnel address: %w", err)
	}

	// Return tunnel address and cleanup function
	return localHost, localPort, tunnel.Stop, nil
}