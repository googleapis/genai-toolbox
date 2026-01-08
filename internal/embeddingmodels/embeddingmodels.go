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
	"strconv"
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

// FormatVectorForPgvector converts a slice of floats into a PostgreSQL vector literal string: '[x, y, z]'
func FormatVectorForPgvector(vectorFloats []float32) any {
	if len(vectorFloats) == 0 {
		return "[]"
	}

	// Pre-allocate the builder.
	var b strings.Builder
	b.Grow(len(vectorFloats) * 10)

	b.WriteByte('[')
	for i, f := range vectorFloats {
		if i > 0 {
			b.WriteString(", ")
		}
		b.Write(strconv.AppendFloat(nil, float64(f), 'g', -1, 32))
	}
	b.WriteByte(']')

	return b.String()
}

var _ VectorFormatter = FormatVectorForPgvector
