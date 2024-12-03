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
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// StdLogger is the standard logger
type StdLogger struct {
	outLogger *slog.Logger
	errLogger *slog.Logger
}

// NewStdLogger create a Logger that uses out and err for informational and error messages.
func NewStdLogger(outW, errW io.Writer, logLevel string) (Logger, error) {
	//Set log level
	var programLevel = new(slog.LevelVar)
	slogLevel, err := severityToLevel(logLevel)
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

// Debug logs debug messages
func (sl *StdLogger) Debug(msg string, keysAndValues ...interface{}) {
	sl.outLogger.Debug(msg, keysAndValues...)
}

// Info logs debug messages
func (sl *StdLogger) Info(msg string, keysAndValues ...interface{}) {
	sl.outLogger.Info(msg, keysAndValues...)
}

// Warn logs warning messages
func (sl *StdLogger) Warn(msg string, keysAndValues ...interface{}) {
	sl.errLogger.Warn(msg, keysAndValues...)
}

// Error logs error messages
func (sl *StdLogger) Error(msg string, keysAndValues ...interface{}) {
	sl.errLogger.Error(msg, keysAndValues...)
}

const (
	Debug = "DEBUG"
	Info  = "INFO"
	Warn  = "WARN"
	Error = "ERROR"
)

// Returns severity level based on string.
func severityToLevel(s string) (slog.Level, error) {
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
	slogLevel, err := severityToLevel(logLevel)
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
	outHandler := slog.NewJSONHandler(outW, &slog.HandlerOptions{
		AddSource:   true,
		Level:       programLevel,
		ReplaceAttr: replace,
	})
	errHandler := slog.NewJSONHandler(errW, &slog.HandlerOptions{
		AddSource:   true,
		Level:       programLevel,
		ReplaceAttr: replace,
	})

	return &StructuredLogger{outLogger: slog.New(outHandler), errLogger: slog.New(errHandler)}, nil
}

// Debug logs debug messages
func (sl *StructuredLogger) Debug(msg string, keysAndValues ...interface{}) {
	sl.outLogger.Debug(msg, keysAndValues...)
}

// Info logs info messages
func (sl *StructuredLogger) Info(msg string, keysAndValues ...interface{}) {
	sl.outLogger.Info(msg, keysAndValues...)
}

// Warn logs warning messages
func (sl *StructuredLogger) Warn(msg string, keysAndValues ...interface{}) {
	sl.errLogger.Warn(msg, keysAndValues...)
}

// Error logs error messages
func (sl *StructuredLogger) Error(msg string, keysAndValues ...interface{}) {
	sl.errLogger.Error(msg, keysAndValues...)
}
