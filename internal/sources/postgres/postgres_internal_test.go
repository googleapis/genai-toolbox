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

package postgres

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestApplyConnectTimeout(t *testing.T) {
	newConfig := func() *pgxpool.Config {
		config, err := pgxpool.ParseConfig("postgres://user:pass@localhost:5432/db")
		if err != nil {
			t.Fatalf("failed to parse base config: %v", err)
		}
		return config
	}

	t.Run("sets the timeout when configured", func(t *testing.T) {
		config := newConfig()
		if err := applyConnectTimeout(config, "5s"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := config.ConnConfig.ConnectTimeout; got != 5*time.Second {
			t.Errorf("ConnectTimeout = %v, want %v", got, 5*time.Second)
		}
	})

	t.Run("leaves the timeout unset when empty", func(t *testing.T) {
		config := newConfig()
		if err := applyConnectTimeout(config, ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := config.ConnConfig.ConnectTimeout; got != 0 {
			t.Errorf("ConnectTimeout = %v, want 0 (no timeout)", got)
		}
	})

	t.Run("returns an error for an invalid duration", func(t *testing.T) {
		config := newConfig()
		if err := applyConnectTimeout(config, "not-a-duration"); err == nil {
			t.Error("expected an error for an invalid duration, got nil")
		}
	})

	t.Run("returns an error for a non-positive duration", func(t *testing.T) {
		config := newConfig()
		if err := applyConnectTimeout(config, "0s"); err == nil {
			t.Error("expected an error for a 0s duration, got nil")
		}
		if err := applyConnectTimeout(config, "-5s"); err == nil {
			t.Error("expected an error for a negative duration, got nil")
		}
	})
}
