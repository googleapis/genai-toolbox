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
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// ValueTextHandler is a [Handler] that writes Records to an [io.Writer] with values separated by spaces.
type ValueTextHandler struct {
	h   slog.Handler
	mu  *sync.Mutex
	out io.Writer
}

// NewValueTextHandler creates a [ValueTextHandler] that writes to out, using the given options.
func NewValueTextHandler(out io.Writer, opts *slog.HandlerOptions) *ValueTextHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &ValueTextHandler{
		out: out,
		h: slog.NewTextHandler(out, &slog.HandlerOptions{
			Level:       opts.Level,
			AddSource:   opts.AddSource,
			ReplaceAttr: nil,
		}),
		mu: &sync.Mutex{},
	}
}

func (h *ValueTextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

func (h *ValueTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ValueTextHandler{h: h.h.WithAttrs(attrs), out: h.out, mu: h.mu}
}

func (h *ValueTextHandler) WithGroup(name string) slog.Handler {
	return &ValueTextHandler{h: h.h.WithGroup(name), out: h.out, mu: h.mu}
}

// Handle formats its argument [Record] as a single line of space-separated values.
// Example output format: 2024-11-12T15:08:11.451377-08:00 INFO "Initialized 0 sources.\n"
func (h *ValueTextHandler) Handle(ctx context.Context, r slog.Record) error {
	buf := make([]byte, 0, 1024)

	// time
	if !r.Time.IsZero() {
		buf = h.appendAttr(buf, slog.Time(slog.TimeKey, r.Time))
	}
	// level
	buf = h.appendAttr(buf, slog.Any(slog.LevelKey, r.Level))
	// message
	buf = h.appendAttr(buf, slog.String(slog.MessageKey, r.Message))

	r.Attrs(func(a slog.Attr) bool {
		buf = h.appendAttr(buf, a)
		return true
	})
	buf = append(buf, "\n"...)

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf)
	return err
}

// appendAttr is responsible for formatting a single attribute
func (h *ValueTextHandler) appendAttr(buf []byte, a slog.Attr) []byte {
	// Resolve the Attr's value before doing anything else.
	a.Value = a.Value.Resolve()
	// Ignore empty Attrs.
	if a.Equal(slog.Attr{}) {
		return buf
	}
	switch a.Value.Kind() {
	case slog.KindString:
		// Quote string values, to make them easy to parse.
		buf = fmt.Appendf(buf, "%q ", a.Value.String())
	case slog.KindTime:
		// Write times in a standard way, without the monotonic time.
		buf = fmt.Appendf(buf, "%s ", a.Value.Time().Format(time.RFC3339Nano))
	case slog.KindGroup:
		attrs := a.Value.Group()
		// Ignore empty groups.
		if len(attrs) == 0 {
			return buf
		}
		for _, ga := range attrs {
			buf = h.appendAttr(buf, ga)
		}
	default:
		buf = fmt.Appendf(buf, "%s ", a.Value)
	}

	return buf
}

// spanContextLogHandler is an slog.Handler which adds attributes from the span
// context.
type spanContextLogHandler struct {
	slog.Handler
}

// handlerWithSpanContext adds attributes from the span context.
func handlerWithSpanContext(handler slog.Handler) *spanContextLogHandler {
	return &spanContextLogHandler{Handler: handler}
}

// Handle overrides slog.Handler's Handle method. This adds attributes from the
// span context to the slog.Record.
func (t *spanContextLogHandler) Handle(ctx context.Context, record slog.Record) error {
	// Get the SpanContext from the golang Context.
	if s := trace.SpanContextFromContext(ctx); s.IsValid() {
		// Add trace context attributes following Cloud Logging structured log format described
		// in https://cloud.google.com/logging/docs/structured-logging#special-payload-fields
		record.AddAttrs(
			slog.Any("logging.googleapis.com/trace", s.TraceID()),
		)
		record.AddAttrs(
			slog.Any("logging.googleapis.com/spanId", s.SpanID()),
		)
		record.AddAttrs(
			slog.Bool("logging.googleapis.com/trace_sampled", s.TraceFlags().IsSampled()),
		)
	}
	return t.Handler.Handle(ctx, record)
}
