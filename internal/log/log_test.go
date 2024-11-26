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
	"log/slog"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/toolbox"
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
			got, err := severityToLevel(tc.in)
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
	_, err := severityToLevel("fail")
	if err == nil {
		t.Fatalf("expected error on incorrect level")
	}
}

func runLogger(logger toolbox.Logger, logMsg string) {
	switch logMsg {
	case "info":
		logger.Info("log info")
	case "debug":
		logger.Debug("log debug")
	case "warn":
		logger.Warn("log warn")
	case "error":
		logger.Error("log error")
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
