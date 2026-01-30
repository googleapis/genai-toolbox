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
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// NewLogger creates a new logger based on the provided format and level.
func NewLogger(format, level string, out, err io.Writer) (Logger, error) {
	switch strings.ToLower(format) {
	case "json":
		return NewStructuredLogger(out, err, level)
	case "standard":
		return NewStdLogger(out, err, level)
	default:
		return nil, fmt.Errorf("logging format invalid: %s", format)
	}
}

// StdLogger is the standard logger
type StdLogger struct {
	outLogger *slog.Logger
	errLogger *slog.Logger
}

// NewStdLogger create a Logger that uses out and err for informational and error messages.
func NewStdLogger(outW, errW io.Writer, logLevel string) (Logger, error) {
	//Set log level
	var programLevel = new(slog.LevelVar)
	slogLevel, err := SeverityToLevel(logLevel)
	if err != nil {
		return nil, err
	}
	programLevel.Set(slogLevel)

	handlerOptions := &slog.HandlerOptions{Level: programLevel}

	return &StdLogger{
		outLogger: slog.New(NewValueTextHandler(outW, handlerOptions)),
		errLogger: slog.New(NewValueTextHandler(errW, handlerOptions)),
	}, nil
}

// DebugContext logs debug messages
func (sl *StdLogger) DebugContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	sl.outLogger.DebugContext(ctx, msg, keysAndValues...)
}

// InfoContext logs debug messages
func (sl *StdLogger) InfoContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	sl.outLogger.InfoContext(ctx, msg, keysAndValues...)
}

// WarnContext logs warning messages
func (sl *StdLogger) WarnContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	sl.errLogger.WarnContext(ctx, msg, keysAndValues...)
}

// ErrorContext logs error messages
func (sl *StdLogger) ErrorContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	sl.errLogger.ErrorContext(ctx, msg, keysAndValues...)
}

const (
	Debug = "DEBUG"
	Info  = "INFO"
	Warn  = "WARN"
	Error = "ERROR"
)

// Returns severity level based on string.
func SeverityToLevel(s string) (slog.Level, error) {
	switch strings.ToUpper(s) {
	case Debug:
		return slog.LevelDebug, nil
	case Info:
		return slog.LevelInfo, nil
	case Warn:
		return slog.LevelWarn, nil
	case Error:
		return slog.LevelError, nil
	default:
		return slog.Level(-5), fmt.Errorf("invalid log level")
	}
}

// Returns severity string based on level.
func levelToSeverity(s string) (string, error) {
	switch s {
	case slog.LevelDebug.String():
		return Debug, nil
	case slog.LevelInfo.String():
		return Info, nil
	case slog.LevelWarn.String():
		return Warn, nil
	case slog.LevelError.String():
		return Error, nil
	default:
		return "", fmt.Errorf("invalid slog level")
	}
}

type StructuredLogger struct {
	outLogger *slog.Logger
	errLogger *slog.Logger
}

// NewStructuredLogger create a Logger that logs messages using JSON.
func NewStructuredLogger(outW, errW io.Writer, logLevel string) (Logger, error) {
	//Set log level
	var programLevel = new(slog.LevelVar)
	slogLevel, err := SeverityToLevel(logLevel)
	if err != nil {
		return nil, err
	}
	programLevel.Set(slogLevel)

	replace := func(groups []string, a slog.Attr) slog.Attr {
		switch a.Key {
		case slog.LevelKey:
			value := a.Value.String()
			sev, _ := levelToSeverity(value)
			return slog.Attr{
				Key:   "severity",
				Value: slog.StringValue(sev),
			}
		case slog.MessageKey:
			return slog.Attr{
				Key:   "message",
				Value: a.Value,
			}
		case slog.SourceKey:
			return slog.Attr{
				Key:   "logging.googleapis.com/sourceLocation",
				Value: a.Value,
			}
		case slog.TimeKey:
			return slog.Attr{
				Key:   "timestamp",
				Value: a.Value,
			}
		}
		return a
	}

	// Configure structured logs to adhere to Cloud LogEntry format
	// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry
	outHandler := handlerWithSpanContext(slog.NewJSONHandler(outW, &slog.HandlerOptions{
		AddSource:   true,
		Level:       programLevel,
		ReplaceAttr: replace,
	}))
	errHandler := handlerWithSpanContext(slog.NewJSONHandler(errW, &slog.HandlerOptions{
		AddSource:   true,
		Level:       programLevel,
		ReplaceAttr: replace,
	}))

	return &StructuredLogger{outLogger: slog.New(outHandler), errLogger: slog.New(errHandler)}, nil
}

// DebugContext logs debug messages
func (sl *StructuredLogger) DebugContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	sl.outLogger.DebugContext(ctx, msg, keysAndValues...)
}

// InfoContext logs info messages
func (sl *StructuredLogger) InfoContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	sl.outLogger.InfoContext(ctx, msg, keysAndValues...)
}

// WarnContext logs warning messages
func (sl *StructuredLogger) WarnContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	sl.errLogger.WarnContext(ctx, msg, keysAndValues...)
}

// ErrorContext logs error messages
func (sl *StructuredLogger) ErrorContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	sl.errLogger.ErrorContext(ctx, msg, keysAndValues...)
}
