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

// Package cloudstoragecommon holds helpers shared across the Cloud Storage
// tool implementations, chiefly error classification.
package cloudstoragecommon

import (
	"context"
	"errors"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"google.golang.org/api/googleapi"
)

// ErrReadSizeLimitExceeded is returned by the source when an object/range
// would exceed the source's configured byte limit. ProcessGCSError maps this
// to an Agent error because the LLM can fix the call by narrowing the 'range'
// parameter.
var ErrReadSizeLimitExceeded = errors.New("cloud storage read size limit exceeded")

// ErrBinaryContent is returned by the source when an object's bytes are not
// valid UTF-8. The MCP tool result channel only carries text today, so binary
// payloads cannot be faithfully round-tripped; ProcessGCSError maps this to an
// Agent error so the LLM knows to stop asking for this object.
//
// TODO: when the toolbox supports non-text MCP content (embedded resources,
// images, blobs), remove this guard and return binary payloads directly.
var ErrBinaryContent = errors.New("cloud storage object is not valid UTF-8 text")

// ProcessGCSError classifies an error from the Cloud Storage Go client into
// either an Agent Error (the LLM can self-correct by changing its input — bad
// request, missing bucket/object, unsatisfiable range) or a Server Error
// (infrastructure failure — auth, IAM denial, quota, 5xx, network
// cancellation). See DEVELOPER.md "Tool Invocation & Error Handling" for the
// wider rationale.
func ProcessGCSError(err error) util.ToolboxError {
	if err == nil {
		return nil
	}

	// Transport-level cancellation/timeout — treat as infrastructure. These
	// checks come first because a wrapped googleapi.Error on top of a
	// cancelled context should still surface as a server error.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return util.NewClientServerError(
			"cloud storage request cancelled or timed out",
			http.StatusGatewayTimeout, err)
	}

	// Read-size cap tripped — agent can narrow the 'range' parameter.
	if errors.Is(err, ErrReadSizeLimitExceeded) {
		return util.NewAgentError(
			"object is too large to read in one call; narrow the 'range' parameter",
			err)
	}

	// Non-UTF-8 object — MCP only carries text today, so the agent should
	// stop trying to read this object.
	if errors.Is(err, ErrBinaryContent) {
		return util.NewAgentError(
			"object contains non-text (binary) bytes and cannot be returned; only UTF-8 text objects are supported",
			err)
	}

	// GCS sentinel errors — "not found" flavours are agent-fixable.
	if errors.Is(err, storage.ErrBucketNotExist) {
		return util.NewAgentError("cloud storage bucket does not exist", err)
	}
	if errors.Is(err, storage.ErrObjectNotExist) {
		return util.NewAgentError("cloud storage object does not exist", err)
	}

	// HTTP-layer errors from googleapis.
	var gErr *googleapi.Error
	if errors.As(err, &gErr) {
		switch gErr.Code {
		case http.StatusBadRequest:
			return util.NewAgentError("cloud storage rejected the request as invalid", err)
		case http.StatusUnauthorized:
			return util.NewClientServerError(
				"cloud storage authentication failed",
				http.StatusUnauthorized, err)
		case http.StatusForbidden:
			return util.NewClientServerError(
				"cloud storage permission denied",
				http.StatusForbidden, err)
		case http.StatusNotFound:
			return util.NewAgentError("cloud storage resource not found", err)
		case http.StatusConflict:
			return util.NewAgentError("cloud storage conflict", err)
		case http.StatusPreconditionFailed:
			return util.NewAgentError("cloud storage precondition failed", err)
		case http.StatusRequestedRangeNotSatisfiable:
			return util.NewAgentError("requested byte range is not satisfiable for this object", err)
		case http.StatusTooManyRequests:
			return util.NewClientServerError(
				"cloud storage request rate limit exceeded",
				http.StatusTooManyRequests, err)
		}
		if gErr.Code >= 500 {
			return util.NewClientServerError(
				"cloud storage server error",
				http.StatusBadGateway, err)
		}
		return util.NewClientServerError(
			"unexpected cloud storage error",
			http.StatusInternalServerError, err)
	}

	// Fallback — unrecognized error (likely transport-level).
	return util.NewClientServerError(
		"cloud storage request failed",
		http.StatusInternalServerError, err)
}
