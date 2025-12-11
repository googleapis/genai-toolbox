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

package embeddingmodels

import (
	"context"
	"fmt"
	"strings"
)

// EmbeddingModelConfig is the interface for configuring embedding models.
type EmbeddingModelConfig interface {
	EmbeddingModelConfigKind() string
	Initialize(context.Context) (EmbeddingModel, error)
}

type EmbeddingModel interface {
	EmbeddingModelKind() string
	ToConfig() EmbeddingModelConfig
	EmbedParameters(context.Context, []string) ([][]float32, error)
}

type VectorFormatter func(vectorFloats []float32) any

// formatVectorForPgvector converts a slice of floats into a PostgreSQL vector literal string: '[x, y, z]'
func FormatVectorForPgvector(vectorFloats []float32) any {
	str := make([]string, len(vectorFloats))
	for i, f := range vectorFloats {
		str[i] = fmt.Sprintf("%g", f)
	}
	// Join with comma and enclose in square brackets '[]'
	return fmt.Sprintf("[%s]", strings.Join(str, ", "))
}

var _ VectorFormatter = FormatVectorForPgvector
