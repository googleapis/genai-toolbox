// Copyright 2026 Google LLC
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

package spanner

import (
	"context"
	"testing"
	"time"
)

// Dummy function to simulate a long-running test operation
func doWork(ctx context.Context) error {
	select {
	case <-time.After(2 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Dummy cleanup that takes the original context
func doCleanup(t *testing.T, ctx context.Context) {
	// Simulate an RPC call that needs a working context
	select {
	case <-time.After(1 * time.Millisecond):
		t.Log("Cleanup succeeded")
	case <-ctx.Done():
		t.Errorf("Teardown failed: %v", ctx.Err())
	}
}

func TestContextTimeoutReproducer(t *testing.T) {
	// Simulate the test setup
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Simulate setup returning a teardown function
	teardown := func(t *testing.T) {
		doCleanup(t, context.WithoutCancel(ctx))
	}
	defer teardown(t)

	// Simulate the test doing work
	err := doWork(ctx)
	if err != nil {
		t.Logf("doWork failed as expected: %v", err)
	}
}
