// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package lookerhealthpulse

import (
	"context"
	"testing"
)

// Each test now matches a pulse action.

func TestRunPulse_CheckDBConnections(t *testing.T) {
	params := PulseParams{
		Action: "check_db_connections",
	}
	tool, err := NewTool(&mockApiSettings)
	if err != nil {
		t.Fatalf("NewTool error: %v", err)
	}
	result, err := tool.RunPulse(context.Background(), params)
	if err != nil {
		t.Fatalf("check_db_connections failed: %v", err)
	}
	_ = result // TODO: Assert expected result
}

func TestRunPulse_CheckDashboardPerformance(t *testing.T) {
	params := PulseParams{
		Action: "check_dashboard_performance",
	}
	tool, err := NewTool(&mockApiSettings)
	if err != nil {
		t.Fatalf("NewTool error: %v", err)
	}
	result, err := tool.RunPulse(context.Background(), params)
	if err != nil {
		t.Fatalf("check_dashboard_performance failed: %v", err)
	}
	_ = result
}

func TestRunPulse_CheckDashboardErrors(t *testing.T) {
	params := PulseParams{
		Action: "check_dashboard_errors",
	}
	tool, err := NewTool(&mockApiSettings)
	if err != nil {
		t.Fatalf("NewTool error: %v", err)
	}
	result, err := tool.RunPulse(context.Background(), params)
	if err != nil {
		t.Fatalf("check_dashboard_errors failed: %v", err)
	}
	_ = result
}

func TestRunPulse_CheckExplorePerformance(t *testing.T) {
	params := PulseParams{
		Action: "check_explore_performance",
	}
	tool, err := NewTool(&mockApiSettings)
	if err != nil {
		t.Fatalf("NewTool error: %v", err)
	}
	result, err := tool.RunPulse(context.Background(), params)
	if err != nil {
		t.Fatalf("check_explore_performance failed: %v", err)
	}
	_ = result
}

func TestRunPulse_CheckScheduleFailures(t *testing.T) {
	params := PulseParams{
		Action: "check_schedule_failures",
	}
	tool, err := NewTool(&mockApiSettings)
	if err != nil {
		t.Fatalf("NewTool error: %v", err)
	}
	result, err := tool.RunPulse(context.Background(), params)
	if err != nil {
		t.Fatalf("check_schedule_failures failed: %v", err)
	}
	_ = result
}

func TestRunPulse_CheckLegacyFeatures(t *testing.T) {
	params := PulseParams{
		Action: "check_legacy_features",
	}
	tool, err := NewTool(&mockApiSettings)
	if err != nil {
		t.Fatalf("NewTool error: %v", err)
	}
	result, err := tool.RunPulse(context.Background(), params)
	if err != nil {
		t.Fatalf("check_legacy_features failed: %v", err)
	}
	_ = result
}