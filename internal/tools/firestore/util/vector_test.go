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

package util

import (
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBuildVectorFieldParameter(t *testing.T) {
	cfg := VectorFieldConfig{
		Name:        "content",
		Description: "text to embed",
		FieldPath:   "embedding",
		EmbeddedBy:  "gemini",
		Required:    true,
	}

	param, runtime, err := BuildVectorFieldParameter(cfg)
	if err != nil {
		t.Fatalf("BuildVectorFieldParameter() error = %v", err)
	}
	if param == nil {
		t.Fatalf("expected parameter to be non-nil")
	}
	if param.EmbeddedBy != "gemini" {
		t.Fatalf("EmbeddedBy = %q, want %q", param.EmbeddedBy, "gemini")
	}
	if !param.GetRequired() {
		t.Fatalf("expected parameter to be required")
	}
	if runtime.ParameterName != "content" || runtime.FieldPath != "embedding" {
		t.Fatalf("unexpected runtime: %+v", runtime)
	}
}

func TestBuildVectorFieldParameterRequiresDescription(t *testing.T) {
	cfg := VectorFieldConfig{
		Name:       "content",
		FieldPath:  "embedding",
		EmbeddedBy: "model",
	}
	if _, _, err := BuildVectorFieldParameter(cfg); err == nil {
		t.Fatalf("expected error when description missing")
	}
}

func TestExtractVectorFieldValues(t *testing.T) {
	params := map[string]any{
		"content": []float32{0.1, 0.2},
	}
	runtime := []VectorFieldRuntime{{ParameterName: "content", FieldPath: "embedding"}}
	got, err := ExtractVectorFieldValues(params, runtime)
	if err != nil {
		t.Fatalf("ExtractVectorFieldValues() error = %v", err)
	}
	embedding, ok := got["embedding"]
	if !ok {
		t.Fatalf("expected embedding field to be present")
	}
	if len(embedding) != 2 {
		t.Fatalf("expected 2 vector values, got %d", len(embedding))
	}
	if math.Abs(embedding[0]-0.1) > 1e-6 || math.Abs(embedding[1]-0.2) > 1e-6 {
		t.Fatalf("unexpected vector values: %v", embedding)
	}
}

func TestUpsertVectorValuesNested(t *testing.T) {
	target := map[string]any{
		"existing": "value",
	}
	vectors := map[string][]float64{
		"metadata.embedding": []float64{0.5, -0.5},
	}
	if err := UpsertVectorValues(target, vectors); err != nil {
		t.Fatalf("UpsertVectorValues() error = %v", err)
	}
	nested, ok := target["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested map to exist")
	}
	if diff := cmp.Diff([]float64{0.5, -0.5}, nested["embedding"]); diff != "" {
		t.Fatalf("embedding mismatch (-want +got):\n%s", diff)
	}
}

func TestEnsureFieldPaths(t *testing.T) {
	mask := []string{"title"}
	vectors := map[string][]float64{"embedding": []float64{1, 2}}
	got := EnsureFieldPaths(mask, vectors)
	want := []string{"title", "embedding"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("EnsureFieldPaths mismatch (-want +got):\n%s", diff)
	}
}

func TestBuildVectorQueryRuntimeRequiresDescription(t *testing.T) {
	cfg := &VectorQueryConfig{
		Name:            "query",
		FieldPath:       "embedding",
		EmbeddedBy:      "model",
		DistanceMeasure: "COSINE",
	}
	if _, _, err := BuildVectorQueryRuntime(cfg); err == nil {
		t.Fatalf("expected error when description missing")
	}
}

func TestExtractVectorQueryValue(t *testing.T) {
	runtime := &VectorQueryRuntime{
		ParameterName: "query",
	}
	params := map[string]any{
		"query": []float32{0.1, -0.2},
	}
	values, ok, err := ExtractVectorQueryValue(params, runtime)
	if err != nil {
		t.Fatalf("ExtractVectorQueryValue() error = %v", err)
	}
	if !ok {
		t.Fatalf("expected vector query to be detected")
	}
	if len(values) != 2 || math.Abs(values[0]-0.1) > 1e-6 || math.Abs(values[1]+0.2) > 1e-6 {
		t.Fatalf("unexpected vector values: %v", values)
	}
}

func TestExtractVectorQueryValueMissing(t *testing.T) {
	values, ok, err := ExtractVectorQueryValue(map[string]any{}, &VectorQueryRuntime{ParameterName: "query"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false when parameter missing, got true with %v", values)
	}
}

func TestBuildFindNearestOptions(t *testing.T) {
	threshold := 0.42
	runtime := &VectorQueryRuntime{
		DistanceResultField: "score",
		DistanceThreshold:   &threshold,
	}
	got := BuildFindNearestOptions(runtime)
	if got == nil {
		t.Fatalf("expected options to be created")
	}
	if got.DistanceResultField != "score" {
		t.Fatalf("DistanceResultField = %q, want %q", got.DistanceResultField, "score")
	}
	if got.DistanceThreshold == nil || math.Abs(*got.DistanceThreshold-threshold) > 1e-9 {
		t.Fatalf("DistanceThreshold mismatch: got %v want %v", got.DistanceThreshold, threshold)
	}

	nilOpts := BuildFindNearestOptions(&VectorQueryRuntime{})
	if nilOpts != nil {
		t.Fatalf("expected nil options when no fields set")
	}
}
