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

package trace

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// tracer for toolbox server
var tracer = otel.Tracer("")

// setTracer sets the tracer with instrumentation name and instrumentation version
func SetTracer(versionString string) {
	tracer = otel.Tracer("github.com/googleapis/genai-toolbox/internal/opentel", trace.WithInstrumentationVersion(versionString))
}

// Tracer retrieves toolbox server tracer
func Tracer() trace.Tracer {
	return tracer
}
