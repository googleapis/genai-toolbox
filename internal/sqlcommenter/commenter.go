// Copyright 2025 Google LLC
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

// Package sqlcommenter appends SQLCommenter-formatted comments to SQL statements
// so that Cloud SQL Insights can group and filter queries by agent tool, service,
// and framework. For BigQuery it exposes helpers to build Job Labels instead,
// because BigQuery surfaces observability data through job metadata rather than
// SQL comments.
//
// SQLCommenter spec: https://google.github.io/sqlcommenter/
// Cloud SQL Insights recognised keys: action, application, controller,
// db_driver, framework, route.
package sqlcommenter

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// Context key type — unexported to avoid collisions with other packages.
type contextKey int

const (
	keyToolName  contextKey = iota
	keyDBDriver             // e.g. "pgx" or "mysql"
	keyAgentName            // MCP clientInfo.name captured at initialize time
)

// WithToolName stores the MCP tool name in the context so that AppendComment
// can retrieve it without callers having to thread it through every call.
func WithToolName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, keyToolName, name)
}

// WithDBDriver stores the database driver identifier (e.g. "pgx", "mysql") in
// the context.
func WithDBDriver(ctx context.Context, driver string) context.Context {
	return context.WithValue(ctx, keyDBDriver, driver)
}

// WithAgentName stores the MCP agent name in the context. It is set
// automatically by the server layer from clientInfo.name in the MCP
// initialize handshake, so individual tools do not need to call this.
// It is used as the "controller" tag, overriding TOOLBOX_AGENT_NAME.
func WithAgentName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, keyAgentName, name)
}

// appName returns the value of TOOLBOX_APP_NAME, falling back to "mcp-toolbox"
// when the variable is not set.
func appName() string {
	if v := os.Getenv("TOOLBOX_APP_NAME"); v != "" {
		return v
	}
	return "mcp-toolbox"
}

// agentName returns the value of TOOLBOX_AGENT_NAME, which is used as the
// "controller" tag in SQLCommenter comments. This identifies the AI agent
// (e.g. "sales-agent", "support-bot") rather than the individual tool action.
// Falls back to an empty string when unset, which causes the tag to be omitted.
func agentName() string {
	return os.Getenv("TOOLBOX_AGENT_NAME")
}

// encode percent-encodes a value according to the SQLCommenter spec (RFC 3986).
func encode(v string) string {
	return strings.ReplaceAll(url.QueryEscape(v), "+", "%20")
}

// AppendComment appends a SQLCommenter comment to sql.  The comment carries the
// tool name (action), application name (application), database driver
// (db_driver), and framework.
//
// If the SQL statement already ends with a semicolon the comment is inserted
// before it so that the statement remains syntactically valid.
//
// Example output:
//
//	SELECT * FROM orders WHERE id = $1
//	/*action='get-orders',application='my-service',db_driver='pgx',framework='mcp-toolbox'*/
func AppendComment(ctx context.Context, sql string) string {
	tags := map[string]string{
		"framework":   "mcp-toolbox",
		"application": appName(),
	}

	if v, ok := ctx.Value(keyToolName).(string); ok && v != "" {
		tags["action"] = v
	}
	// controller resolution priority (highest → lowest):
	//   1. Dynamic agent name from MCP clientInfo.name (set by server layer)
	//   2. Static TOOLBOX_AGENT_NAME env var
	//   3. Tool name (action) as fallback
	switch {
	case ctx.Value(keyAgentName) != nil:
		if v := ctx.Value(keyAgentName).(string); v != "" {
			tags["controller"] = v
		}
	case agentName() != "":
		tags["controller"] = agentName()
	default:
		if v, ok := ctx.Value(keyToolName).(string); ok && v != "" {
			tags["controller"] = v
		}
	}
	if v, ok := ctx.Value(keyDBDriver).(string); ok && v != "" {
		tags["db_driver"] = v
	}

	// Add traceparent for OpenTelemetry distributed tracing correlation
	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		tags["traceparent"] = fmt.Sprintf("00-%s-%s-%s",
			spanCtx.TraceID().String(),
			spanCtx.SpanID().String(),
			spanCtx.TraceFlags().String(),
		)
	}

	// Build key=value pairs in deterministic (sorted) order.
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(tags))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s='%s'", encode(k), encode(tags[k])))
	}

	comment := "/*" + strings.Join(parts, ",") + "*/"

	// Preserve a trailing semicolon — place the comment before it.
	trimmed := strings.TrimRight(sql, " \t\r\n")
	if strings.HasSuffix(trimmed, ";") {
		return trimmed[:len(trimmed)-1] + "\n" + comment + ";"
	}
	return sql + "\n" + comment
}


// JobLabels returns a map of BigQuery Job Labels built from the context values
// set by WithToolName / WithDBDriver, plus fixed framework/application keys.
// Pass the result to query.Labels before calling query.Run().
//
// Key resolution matches AppendComment to ensure consistent observability.
func JobLabels(ctx context.Context) map[string]string {
	labels := map[string]string{
		"framework":   "mcp-toolbox",
		"application": sanitizeLabelValue(appName()),
		"db_driver":   "bigquery",
	}

	if v, ok := ctx.Value(keyToolName).(string); ok && v != "" {
		labels["action"] = sanitizeLabelValue(v)
	}

	// controller resolution priority (highest → lowest):
	//   1. Dynamic agent name from MCP clientInfo.name (set by server layer)
	//   2. Static TOOLBOX_AGENT_NAME env var
	//   3. Tool name (action) as fallback
	switch {
	case ctx.Value(keyAgentName) != nil:
		if v := ctx.Value(keyAgentName).(string); v != "" {
			labels["controller"] = sanitizeLabelValue(v)
		}
	case agentName() != "":
		labels["controller"] = sanitizeLabelValue(agentName())
	default:
		if v, ok := ctx.Value(keyToolName).(string); ok && v != "" {
			labels["controller"] = sanitizeLabelValue(v)
		}
	}

	// Add traceparent for BigQuery Job Label correlation
	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		labels["traceparent"] = sanitizeLabelValue(fmt.Sprintf("00-%s-%s-%s",
			spanCtx.TraceID().String(),
			spanCtx.SpanID().String(),
			spanCtx.TraceFlags().String(),
		))
	}

	return labels
}

// sanitizeLabelValue converts a string to a valid BigQuery label value.
// BigQuery label values must be lowercase letters, numbers, hyphens, and
// underscores; at most 63 characters.
func sanitizeLabelValue(v string) string {
	v = strings.ToLower(v)
	var b strings.Builder
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	s := b.String()
	if len(s) > 63 {
		s = s[:63]
	}
	return s
}
