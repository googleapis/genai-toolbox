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

package sqlcommenter

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/googleapis/mcp-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
)

// AppendComment appends a SQLCommenter-format comment to the given SQL statement.
// It gathers attributes from the context (trace, server, client, tool metadata)
// and the provided dbSystemName, then appends them as key='value' pairs sorted
// alphabetically.
func AppendComment(ctx context.Context, statement string, dbSystemName string) string {
	// Only append SQL comments when sql-commenter is enabled
	if !util.SQLCommenterEnabledFromContext(ctx) {
		return statement
	}

	pairs := collectAttributes(ctx, dbSystemName)
	if len(pairs) == 0 {
		return statement
	}

	// Sort keys alphabetically
	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build comment in SQLCommenter format: key='url_encoded_value'
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		encodedKey := url.QueryEscape(k)
		encodedVal := url.QueryEscape(pairs[k])
		parts = append(parts, fmt.Sprintf("%s='%s'", encodedKey, encodedVal))
	}

	comment := strings.Join(parts, ",")
	return "/*" + comment + "*/ " + statement
}

// collectAttributes gathers all available SQLCommenter attributes from context.
func collectAttributes(ctx context.Context, dbSystemName string) map[string]string {
	attrs := make(map[string]string)

	// traceparent from OTel span context
	spanCtx := trace.SpanFromContext(ctx).SpanContext()
	if spanCtx.IsValid() {
		traceparent := fmt.Sprintf("00-%s-%s-%s",
			spanCtx.TraceID().String(),
			spanCtx.SpanID().String(),
			spanCtx.TraceFlags().String(),
		)
		attrs["traceparent"] = traceparent
	}

	// server from UserAgent context
	if ua, err := util.UserAgentFromContext(ctx); err == nil && ua != "" {
		attrs["server"] = ua
	}

	// db.system.name from parameter
	if dbSystemName != "" {
		attrs["db.system.name"] = dbSystemName
	}

	// tool.name from GenAIMetricAttrs
	if genAI := util.GenAIMetricAttrsFromContext(ctx); genAI != nil {
		if genAI.ToolName != "" {
			attrs["tool.name"] = genAI.ToolName
		}
	}

	// Client attributes from TelemetryAttributes
	if ta := util.TelemetryAttributesFromContext(ctx); ta != nil {
		// Combined client = name/version
		if ta.ClientName != "" && ta.ClientVersion != "" {
			attrs["client"] = ta.ClientName + "/" + ta.ClientVersion
		} else if ta.ClientName != "" {
			attrs["client"] = ta.ClientName
		} else if ta.ClientVersion != "" {
			attrs["client"] = ta.ClientVersion
		}

		if ta.ClientModel != "" {
			attrs["client.model"] = ta.ClientModel
		}
		if ta.ClientUserID != "" {
			attrs["client.user.id"] = ta.ClientUserID
		}
		if ta.ClientAgentID != "" {
			attrs["client.agent.id"] = ta.ClientAgentID
		}
	}

	return attrs
}
