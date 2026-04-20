// Copyright 2026 Google LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package util

import (
	"errors"
	"net/http"
	"testing"

	"google.golang.org/api/googleapi"
)

func TestProcessGcpErrorTreatsBadRequestAsAgentError(t *testing.T) {
	err := ProcessGcpError(&googleapi.Error{
		Code:    http.StatusBadRequest,
		Message: "Bad Request",
		Errors: []googleapi.ErrorItem{{
			Reason:  "invalidQuery",
			Message: "No matching signature for operator >=",
		}},
	})

	var agentErr *AgentError
	if !errors.As(err, &agentErr) {
		t.Fatalf("expected AgentError, got %T", err)
	}
}

func TestProcessGcpErrorPreservesAuthFailuresAsServerErrors(t *testing.T) {
	tcs := []struct {
		name string
		code int
	}{
		{name: "unauthorized", code: http.StatusUnauthorized},
		{name: "forbidden", code: http.StatusForbidden},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := ProcessGcpError(&googleapi.Error{
				Code:    tc.code,
				Message: http.StatusText(tc.code),
			})

			var clientServerErr *ClientServerError
			if !errors.As(err, &clientServerErr) {
				t.Fatalf("expected ClientServerError, got %T", err)
			}
			if clientServerErr.Code != tc.code {
				t.Fatalf("expected code %d, got %d", tc.code, clientServerErr.Code)
			}
		})
	}
}
