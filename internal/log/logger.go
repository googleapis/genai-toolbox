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

package log

import (
	"context"
)

// Logger is the interface used throughout the project for logging.
type Logger interface {
	// DebugContext is for reporting additional information about internal operations.
	DebugContext(ctx context.Context, format string, args ...interface{})
	// InfoContext is for reporting informational messages.
	InfoContext(ctx context.Context, format string, args ...interface{})
	// WarnContext is for reporting warning messages.
	WarnContext(ctx context.Context, format string, args ...interface{})
	// ErrorContext is for reporting errors.
	ErrorContext(ctx context.Context, format string, args ...interface{})
}
