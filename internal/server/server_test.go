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
	"strconv"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/server"
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
	cfg := server.Config{
		Address: addr,
		Port:    port,
	}
	s, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("Unable to initialize server!")
	}

	// start server in background
	errCh := make(chan error)
	go func() {
		err := s.ListenAndServe(ctx)
		defer close(errCh)
		if err != nil {
			errCh <- err
		}
	}()

	if !tryDial(net.JoinHostPort(addr, strconv.Itoa(port)), 10) {
		t.Fatalf("Unable to dial server!")
	}

}
