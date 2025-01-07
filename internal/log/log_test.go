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
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSeverityToLevel(t *testing.T) {
	tcs := []struct {
		name string
		in   string
		want slog.Level
	}{
		{
			name: "test debug",
			in:   "Debug",
			want: slog.LevelDebug,
		},
		{
			name: "test info",
			in:   "Info",
			want: slog.LevelInfo,
		},
		{
			name: "test warn",
			in:   "Warn",
			want: slog.LevelWarn,
		},
		{
			name: "test error",
			in:   "Error",
			want: slog.LevelError,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SeverityToLevel(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got != tc.want {
				t.Fatalf("incorrect level to severity: got %v, want %v", got, tc.want)
			}

		})
	}
}

func TestSeverityToLevelError(t *testing.T) {
	_, err := SeverityToLevel("fail")
	if err == nil {
		t.Fatalf("expected error on incorrect level")
	}
}

func TestLevelToSeverity(t *testing.T) {
	tcs := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "test debug",
			in:   slog.LevelDebug.String(),
			want: "DEBUG",
		},
		{
			name: "test info",
			in:   slog.LevelInfo.String(),
			want: "INFO",
		},
		{
			name: "test warn",
			in:   slog.LevelWarn.String(),
			want: "WARN",
		},
		{
			name: "test error",
			in:   slog.LevelError.String(),
			want: "ERROR",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := levelToSeverity(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got != tc.want {
				t.Fatalf("incorrect level to severity: got %v, want %v", got, tc.want)
			}

		})
	}
}

func TestLevelToSeverityError(t *testing.T) {
	_, err := levelToSeverity("fail")
	if err == nil {
		t.Fatalf("expected error on incorrect slog level")
	}
}

func runLogger(logger Logger, logMsg string) {
	ctx := context.Background()
	switch logMsg {
	case "info":
		logger.InfoContext(ctx, "log info")
	case "debug":
		logger.DebugContext(ctx, "log debug")
	case "warn":
		logger.WarnContext(ctx, "log warn")
	case "error":
		logger.ErrorContext(ctx, "log error")
	}
}

func TestStdLogger(t *testing.T) {
	tcs := []struct {
		name     string
		logLevel string
		logMsg   string
		wantOut  string
		wantErr  string
	}{
		{
			name:     "debug logger logging debug",
			logLevel: "debug",
			logMsg:   "debug",
			wantOut:  "DEBUG \"log debug\" \n",
			wantErr:  "",
		},
		{
			name:     "info logger logging debug",
			logLevel: "info",
			logMsg:   "debug",
			wantOut:  "",
			wantErr:  "",
		},
		{
			name:     "warn logger logging debug",
			logLevel: "warn",
			logMsg:   "debug",
			wantOut:  "",
			wantErr:  "",
		},
		{
			name:     "error logger logging debug",
			logLevel: "error",
			logMsg:   "debug",
			wantOut:  "",
			wantErr:  "",
		},
		{
			name:     "debug logger logging info",
			logLevel: "debug",
			logMsg:   "info",
			wantOut:  "INFO \"log info\" \n",
			wantErr:  "",
		},
		{
			name:     "info logger logging info",
			logLevel: "info",
			logMsg:   "info",
			wantOut:  "INFO \"log info\" \n",
			wantErr:  "",
		},
		{
			name:     "warn logger logging info",
			logLevel: "warn",
			logMsg:   "info",
			wantOut:  "",
			wantErr:  "",
		},
		{
			name:     "error logger logging info",
			logLevel: "error",
			logMsg:   "info",
			wantOut:  "",
			wantErr:  "",
		},
		{
			name:     "debug logger logging warn",
			logLevel: "debug",
			logMsg:   "warn",
			wantOut:  "",
			wantErr:  "WARN \"log warn\" \n",
		},
		{
			name:     "info logger logging warn",
			logLevel: "info",
			logMsg:   "warn",
			wantOut:  "",
			wantErr:  "WARN \"log warn\" \n",
		},
		{
			name:     "warn logger logging warn",
			logLevel: "warn",
			logMsg:   "warn",
			wantOut:  "",
			wantErr:  "WARN \"log warn\" \n",
		},
		{
			name:     "error logger logging warn",
			logLevel: "error",
			logMsg:   "warn",
			wantOut:  "",
			wantErr:  "",
		},
		{
			name:     "debug logger logging error",
			logLevel: "debug",
			logMsg:   "error",
			wantOut:  "",
			wantErr:  "ERROR \"log error\" \n",
		},
		{
			name:     "info logger logging error",
			logLevel: "info",
			logMsg:   "error",
			wantOut:  "",
			wantErr:  "ERROR \"log error\" \n",
		},
		{
			name:     "warn logger logging error",
			logLevel: "warn",
			logMsg:   "error",
			wantOut:  "",
			wantErr:  "ERROR \"log error\" \n",
		},
		{
			name:     "error logger logging error",
			logLevel: "error",
			logMsg:   "error",
			wantOut:  "",
			wantErr:  "ERROR \"log error\" \n",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			outW := new(bytes.Buffer)
			errW := new(bytes.Buffer)

			logger, err := NewStdLogger(outW, errW, tc.logLevel)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			runLogger(logger, tc.logMsg)

			outWString := outW.String()
			spaceIndexOut := strings.Index(outWString, " ")
			gotOut := outWString[spaceIndexOut+1:]

			errWString := errW.String()
			spaceIndexErr := strings.Index(errWString, " ")
			gotErr := errWString[spaceIndexErr+1:]

			if diff := cmp.Diff(gotOut, tc.wantOut); diff != "" {
				t.Fatalf("incorrect log: diff %v", diff)
			}
			if diff := cmp.Diff(gotErr, tc.wantErr); diff != "" {
				t.Fatalf("incorrect log: diff %v", diff)
			}
		})
	}
}

func TestStructuredLoggerDebugLog(t *testing.T) {
	tcs := []struct {
		name     string
		logLevel string
		logMsg   string
		wantOut  map[string]string
		wantErr  map[string]string
	}{
		{
			name:     "debug logger logging debug",
			logLevel: "debug",
			logMsg:   "debug",
			wantOut: map[string]string{
				"severity": "DEBUG",
				"message":  "log debug",
			},
			wantErr: map[string]string{},
		},
		{
			name:     "info logger logging debug",
			logLevel: "info",
			logMsg:   "debug",
			wantOut:  map[string]string{},
			wantErr:  map[string]string{},
		},
		{
			name:     "warn logger logging debug",
			logLevel: "warn",
			logMsg:   "debug",
			wantOut:  map[string]string{},
			wantErr:  map[string]string{},
		},
		{
			name:     "error logger logging debug",
			logLevel: "error",
			logMsg:   "debug",
			wantOut:  map[string]string{},
			wantErr:  map[string]string{},
		},
		{
			name:     "debug logger logging info",
			logLevel: "debug",
			logMsg:   "info",
			wantOut: map[string]string{
				"severity": "INFO",
				"message":  "log info",
			},
			wantErr: map[string]string{},
		},
		{
			name:     "info logger logging info",
			logLevel: "info",
			logMsg:   "info",
			wantOut: map[string]string{
				"severity": "INFO",
				"message":  "log info",
			},
			wantErr: map[string]string{},
		},
		{
			name:     "warn logger logging info",
			logLevel: "warn",
			logMsg:   "info",
			wantOut:  map[string]string{},
			wantErr:  map[string]string{},
		},
		{
			name:     "error logger logging info",
			logLevel: "error",
			logMsg:   "info",
			wantOut:  map[string]string{},
			wantErr:  map[string]string{},
		},
		{
			name:     "debug logger logging warn",
			logLevel: "debug",
			logMsg:   "warn",
			wantOut:  map[string]string{},
			wantErr: map[string]string{
				"severity": "WARN",
				"message":  "log warn",
			},
		},
		{
			name:     "info logger logging warn",
			logLevel: "info",
			logMsg:   "warn",
			wantOut:  map[string]string{},
			wantErr: map[string]string{
				"severity": "WARN",
				"message":  "log warn",
			},
		},
		{
			name:     "warn logger logging warn",
			logLevel: "warn",
			logMsg:   "warn",
			wantOut:  map[string]string{},
			wantErr: map[string]string{
				"severity": "WARN",
				"message":  "log warn",
			},
		},
		{
			name:     "error logger logging warn",
			logLevel: "error",
			logMsg:   "warn",
			wantOut:  map[string]string{},
			wantErr:  map[string]string{},
		},
		{
			name:     "debug logger logging error",
			logLevel: "debug",
			logMsg:   "error",
			wantOut:  map[string]string{},
			wantErr: map[string]string{
				"severity": "ERROR",
				"message":  "log error",
			},
		},
		{
			name:     "info logger logging error",
			logLevel: "info",
			logMsg:   "error",
			wantOut:  map[string]string{},
			wantErr: map[string]string{
				"severity": "ERROR",
				"message":  "log error",
			},
		},
		{
			name:     "warn logger logging error",
			logLevel: "warn",
			logMsg:   "error",
			wantOut:  map[string]string{},
			wantErr: map[string]string{
				"severity": "ERROR",
				"message":  "log error",
			},
		},
		{
			name:     "error logger logging error",
			logLevel: "error",
			logMsg:   "error",
			wantOut:  map[string]string{},
			wantErr: map[string]string{
				"severity": "ERROR",
				"message":  "log error",
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			outW := new(bytes.Buffer)
			errW := new(bytes.Buffer)

			logger, err := NewStructuredLogger(outW, errW, tc.logLevel)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			runLogger(logger, tc.logMsg)

			if len(tc.wantOut) != 0 {
				got := make(map[string]interface{})

				if err := json.Unmarshal(outW.Bytes(), &got); err != nil {
					t.Fatalf("failed to parse writer")
				}

				if got["severity"] != tc.wantOut["severity"] {
					t.Fatalf("incorrect severity: got %v, want %v", got["severity"], tc.wantOut["severity"])
				}

			} else {
				if outW.String() != "" {
					t.Fatalf("incorrect log. got %v, want %v", outW.String(), "")
				}
			}

			if len(tc.wantErr) != 0 {
				got := make(map[string]interface{})

				if err := json.Unmarshal(errW.Bytes(), &got); err != nil {
					t.Fatalf("failed to parse writer")
				}

				if got["severity"] != tc.wantErr["severity"] {
					t.Fatalf("incorrect severity: got %v, want %v", got["severity"], tc.wantErr["severity"])
				}

			} else {
				if errW.String() != "" {
					t.Fatalf("incorrect log. got %v, want %v", errW.String(), "")
				}
			}
		})
	}
}
