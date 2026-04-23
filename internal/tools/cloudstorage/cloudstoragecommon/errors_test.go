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

package cloudstoragecommon_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googleapis/mcp-toolbox/internal/tools/cloudstorage/cloudstoragecommon"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"google.golang.org/api/googleapi"
)

func gcpErr(code int) error {
	return &googleapi.Error{Code: code, Message: fmt.Sprintf("synthetic %d", code)}
}

func TestProcessGCSError_NilPassthrough(t *testing.T) {
	if got := cloudstoragecommon.ProcessGCSError(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestProcessGCSError_Categorization(t *testing.T) {
	tcs := []struct {
		desc     string
		in       error
		category util.ErrorCategory
		// code is checked only for server errors.
		code int
	}{
		{desc: "bucket not exist sentinel", in: storage.ErrBucketNotExist, category: util.CategoryAgent},
		{desc: "object not exist sentinel", in: storage.ErrObjectNotExist, category: util.CategoryAgent},
		{desc: "wrapped object not exist", in: fmt.Errorf("wrapped: %w", storage.ErrObjectNotExist), category: util.CategoryAgent},
		{desc: "read size limit exceeded", in: fmt.Errorf("big: %w", cloudstoragecommon.ErrReadSizeLimitExceeded), category: util.CategoryAgent},
		{desc: "binary (non-UTF-8) content", in: fmt.Errorf("obj: %w", cloudstoragecommon.ErrBinaryContent), category: util.CategoryAgent},
		{desc: "400 bad request", in: gcpErr(http.StatusBadRequest), category: util.CategoryAgent},
		{desc: "401 unauthorized", in: gcpErr(http.StatusUnauthorized), category: util.CategoryServer, code: http.StatusUnauthorized},
		{desc: "403 forbidden", in: gcpErr(http.StatusForbidden), category: util.CategoryServer, code: http.StatusForbidden},
		{desc: "404 not found", in: gcpErr(http.StatusNotFound), category: util.CategoryAgent},
		{desc: "409 conflict", in: gcpErr(http.StatusConflict), category: util.CategoryAgent},
		{desc: "412 precondition failed", in: gcpErr(http.StatusPreconditionFailed), category: util.CategoryAgent},
		{desc: "416 range not satisfiable", in: gcpErr(http.StatusRequestedRangeNotSatisfiable), category: util.CategoryAgent},
		{desc: "429 rate limited", in: gcpErr(http.StatusTooManyRequests), category: util.CategoryServer, code: http.StatusTooManyRequests},
		{desc: "500 internal", in: gcpErr(http.StatusInternalServerError), category: util.CategoryServer, code: http.StatusBadGateway},
		{desc: "503 unavailable", in: gcpErr(http.StatusServiceUnavailable), category: util.CategoryServer, code: http.StatusBadGateway},
		{desc: "418 teapot (unmapped 4xx)", in: gcpErr(http.StatusTeapot), category: util.CategoryServer, code: http.StatusInternalServerError},
		{desc: "context cancelled", in: context.Canceled, category: util.CategoryServer, code: http.StatusGatewayTimeout},
		{desc: "context deadline exceeded", in: context.DeadlineExceeded, category: util.CategoryServer, code: http.StatusGatewayTimeout},
		{desc: "unrecognized error", in: errors.New("kaboom"), category: util.CategoryServer, code: http.StatusInternalServerError},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := cloudstoragecommon.ProcessGCSError(tc.in)
			if got == nil {
				t.Fatalf("expected non-nil ToolboxError, got nil")
			}
			if got.Category() != tc.category {
				t.Errorf("category = %v, want %v", got.Category(), tc.category)
			}
			if tc.category == util.CategoryServer {
				cse, ok := got.(*util.ClientServerError)
				if !ok {
					t.Fatalf("expected *ClientServerError, got %T", got)
				}
				if cse.Code != tc.code {
					t.Errorf("code = %d, want %d", cse.Code, tc.code)
				}
			} else {
				if _, ok := got.(*util.AgentError); !ok {
					t.Fatalf("expected *AgentError, got %T", got)
				}
			}
			if !errors.Is(got, tc.in) {
				t.Errorf("errors.Is chain broken: got %v not wrapping original %v", got, tc.in)
			}
		})
	}
}
