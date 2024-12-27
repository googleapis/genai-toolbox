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

package server_test

import (
	"context"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/telemetry"
)

// tryDial is a utility function that dials an address up to 'attempts' number of times.
func tryDial(addr string, attempts int) bool {
	for i := 0; i < attempts; i++ {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		_ = conn.Close()
		return true
	}
	return false
}

func TestServe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr, port := "127.0.0.1", 5000
	cfg := server.ServerConfig{
		Address: addr,
		Port:    port,
	}

	customMetric, otelShutdown, err := telemetry.SetupOTel(ctx, "0.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	defer func() {
		err := otelShutdown(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}()
	testLogger, err := log.NewStdLogger(os.Stdout, os.Stderr, "info")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	s, err := server.NewServer(cfg, testLogger, customMetric)
	if err != nil {
		t.Fatalf("unable to initialize server! %v", err)
	}

	// start server in background
	errCh := make(chan error)
	go func() {
		l, err := s.Listen(ctx)
		defer close(errCh)
		if err != nil {
			errCh <- err
		}
		err = s.Serve(l)
		if err != nil {
			errCh <- err
		}
	}()

	if !tryDial(net.JoinHostPort(addr, strconv.Itoa(port)), 10) {
		t.Fatalf("unable to dial server!")
	}

}
