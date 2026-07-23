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
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

// VectorFieldConfig defines a plain-text input that should be embedded and stored
// at the specified Firestore field path during writes.
type VectorFieldConfig struct {
	Name           string `yaml:"name" validate:"required"`
	Description    string `yaml:"description"`
	FieldPath      string `yaml:"fieldPath" validate:"required"`
	EmbeddedBy     string `yaml:"embeddedBy" validate:"required"`
	Required       bool   `yaml:"required"`
	ValueFromParam string `yaml:"valueFromParam"`
}

// VectorFieldRuntime captures the runtime metadata required to process vector inputs.
type VectorFieldRuntime struct {
	ParameterName string
	FieldPath     string
}

// BuildVectorFieldParameter converts the config into a string parameter definition and runtime metadata.
func BuildVectorFieldParameter(cfg VectorFieldConfig) (*parameters.StringParameter, VectorFieldRuntime, error) {
	if cfg.Name == "" {
		return nil, VectorFieldRuntime{}, fmt.Errorf("vector field parameter requires a name")
	}
	if cfg.Description == "" && cfg.ValueFromParam == "" {
		return nil, VectorFieldRuntime{}, fmt.Errorf("vector field %q must specify description", cfg.Name)
	}
	if cfg.FieldPath == "" {
		return nil, VectorFieldRuntime{}, fmt.Errorf("vector field %q must specify fieldPath", cfg.Name)
	}
	if cfg.EmbeddedBy == "" {
		return nil, VectorFieldRuntime{}, fmt.Errorf("vector field %q must specify embeddedBy", cfg.Name)
	}

	param := parameters.NewStringParameterWithRequired(cfg.Name, cfg.Description, cfg.Required)
	param.EmbeddedBy = cfg.EmbeddedBy
	param.ValueFromParam = cfg.ValueFromParam

	runtime := VectorFieldRuntime{
		ParameterName: cfg.Name,
		FieldPath:     cfg.FieldPath,
	}

	return param, runtime, nil
}

// ExtractVectorFieldValues renders vector field parameter values into float64 slices keyed by field path.
func ExtractVectorFieldValues(paramMap map[string]any, fields []VectorFieldRuntime) (map[string][]float64, error) {
	if len(fields) == 0 {
		return nil, nil
	}

	result := make(map[string][]float64, len(fields))
	for _, field := range fields {
		value, ok := paramMap[field.ParameterName]
		if !ok || value == nil {
			continue
		}

		vectorValues, err := ConvertVectorValue(value)
		if err != nil {
			return nil, fmt.Errorf("parameter %q: %w", field.ParameterName, err)
		}
		if len(vectorValues) == 0 {
			continue
		}
		result[field.FieldPath] = vectorValues
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

// ConvertVectorValue casts embedding outputs into a []float64 slice that Firestore accepts.
func ConvertVectorValue(val any) ([]float64, error) {
	switch v := val.(type) {
	case []float32:
		out := make([]float64, len(v))
		for i, item := range v {
			out[i] = float64(item)
		}
		return out, nil
	case []float64:
		return v, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("expected embedding result to be []float32 or []float64, got %T", val)
	}
}

// UpsertVectorValues mutates the provided map to include the given vectors at their field paths.
func UpsertVectorValues(target map[string]any, vectors map[string][]float64) error {
	if len(vectors) == 0 {
		return nil
	}

	for fieldPath, vector := range vectors {
		if err := setNestedField(target, fieldPath, vector); err != nil {
			return fmt.Errorf("failed to set field %q: %w", fieldPath, err)
		}
	}
	return nil
}

// EnsureFieldPaths appends any missing vector field paths to the provided mask slice.
func EnsureFieldPaths(mask []string, vectors map[string][]float64) []string {
	if len(vectors) == 0 {
		return mask
	}

	existing := make(map[string]struct{}, len(mask))
	for _, path := range mask {
		existing[path] = struct{}{}
	}

	for fieldPath := range vectors {
		if _, ok := existing[fieldPath]; ok {
			continue
		}
		mask = append(mask, fieldPath)
	}
	return mask
}

// setNestedField assigns the provided value within a map according to a dot-delimited field path.
func setNestedField(target map[string]any, fieldPath string, value any) error {
	if target == nil {
		return fmt.Errorf("target map cannot be nil")
	}
	segments := strings.Split(fieldPath, ".")
	if len(segments) == 0 {
		return fmt.Errorf("invalid field path: %q", fieldPath)
	}

	current := target
	for idx, segment := range segments {
		if segment == "" {
			return fmt.Errorf("field path %q contains an empty segment", fieldPath)
		}

		if idx == len(segments)-1 {
			current[segment] = value
			return nil
		}

		next, ok := current[segment]
		if !ok || next == nil {
			child := make(map[string]any)
			current[segment] = child
			current = child
			continue
		}

		nextMap, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("field %q is not a map; cannot create child field %q", strings.Join(segments[:idx+1], "."), segments[idx+1])
		}
		current = nextMap
	}

	return nil
}

// VectorQueryConfig describes a similarity search input.
type VectorQueryConfig struct {
	Name                string   `yaml:"name" validate:"required"`
	Description         string   `yaml:"description"`
	FieldPath           string   `yaml:"fieldPath" validate:"required"`
	EmbeddedBy          string   `yaml:"embeddedBy" validate:"required"`
	DistanceMeasure     string   `yaml:"distanceMeasure" validate:"required"`
	DistanceResultField string   `yaml:"distanceResultField"`
	DistanceThreshold   *float64 `yaml:"distanceThreshold"`
	Required            bool     `yaml:"required"`
}

// VectorQueryRuntime stores the computed runtime settings for vector search.
type VectorQueryRuntime struct {
	ParameterName       string
	FieldPath           string
	Measure             firestore.DistanceMeasure
	DistanceResultField string
	DistanceThreshold   *float64
}

// ExtractVectorQueryValue ensures the embedded vector parameter exists and returns its values.
func ExtractVectorQueryValue(paramMap map[string]any, runtime *VectorQueryRuntime) ([]float64, bool, error) {
	if runtime == nil {
		return nil, false, nil
	}
	value, ok := paramMap[runtime.ParameterName]
	if !ok || value == nil {
		return nil, false, nil
	}
	vectorValues, err := ConvertVectorValue(value)
	if err != nil {
		return nil, false, fmt.Errorf("parameter %q: %w", runtime.ParameterName, err)
	}
	if len(vectorValues) == 0 {
		return nil, false, fmt.Errorf("parameter %q: embedding result is empty", runtime.ParameterName)
	}
	return vectorValues, true, nil
}

// BuildVectorQueryRuntime creates parameter and runtime config for a vector query.
func BuildVectorQueryRuntime(cfg *VectorQueryConfig) (*parameters.StringParameter, *VectorQueryRuntime, error) {
	if cfg == nil {
		return nil, nil, nil
	}
	if cfg.Name == "" {
		return nil, nil, fmt.Errorf("vector query parameter requires a name")
	}
	if cfg.Description == "" {
		return nil, nil, fmt.Errorf("vector query %q must specify description", cfg.Name)
	}
	if cfg.FieldPath == "" {
		return nil, nil, fmt.Errorf("vector query %q must specify fieldPath", cfg.Name)
	}
	if cfg.EmbeddedBy == "" {
		return nil, nil, fmt.Errorf("vector query %q must specify embeddedBy", cfg.Name)
	}
	measure, err := parseDistanceMeasure(cfg.DistanceMeasure)
	if err != nil {
		return nil, nil, fmt.Errorf("vector query %q: %w", cfg.Name, err)
	}

	param := parameters.NewStringParameterWithRequired(cfg.Name, cfg.Description, cfg.Required)
	param.EmbeddedBy = cfg.EmbeddedBy

	runtime := &VectorQueryRuntime{
		ParameterName:       cfg.Name,
		FieldPath:           cfg.FieldPath,
		Measure:             measure,
		DistanceResultField: cfg.DistanceResultField,
		DistanceThreshold:   cfg.DistanceThreshold,
	}
	return param, runtime, nil
}

func parseDistanceMeasure(value string) (firestore.DistanceMeasure, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "EUCLIDEAN":
		return firestore.DistanceMeasureEuclidean, nil
	case "COSINE":
		return firestore.DistanceMeasureCosine, nil
	case "DOT_PRODUCT":
		return firestore.DistanceMeasureDotProduct, nil
	default:
		return 0, fmt.Errorf("unsupported distanceMeasure %q (supported: EUCLIDEAN, COSINE, DOT_PRODUCT)", value)
	}
}

// BuildFindNearestOptions converts the runtime settings into Firestore options.
func BuildFindNearestOptions(runtime *VectorQueryRuntime) *firestore.FindNearestOptions {
	if runtime == nil {
		return nil
	}

	opts := &firestore.FindNearestOptions{}
	if runtime.DistanceThreshold != nil {
		opts.DistanceThreshold = runtime.DistanceThreshold
	}
	if runtime.DistanceResultField != "" {
		opts.DistanceResultField = runtime.DistanceResultField
	}
	if opts.DistanceThreshold == nil && opts.DistanceResultField == "" {
		return nil
	}
	return opts
}
