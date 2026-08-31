// Copyright 2026 PolitePixels Limited
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

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package logger_domain

import (
	"context"
	"log/slog"

	"piko.sh/piko/internal/daemon/daemon_dto"
)

// requestContextHandler is an slog.Handler wrapper that enriches every log record with
// per-request fields from PikoRequestCtx. When no PikoRequestCtx is present in the
// context (non-request code paths), the handler is a zero-cost passthrough.
//
// Fields added to every request-scoped record (when non-empty):
//   - request_id: the formatted request ID (server-generated or forwarded)
//   - client_ip: the real client IP after trusted-proxy extraction
//   - locale: the route locale (e.g., "en", "de")
//
// Fields added only at Error and above:
//   - route: the matched route pattern (e.g., "/blog/{slug}")
//   - browser, browser_version, os, device: coarse families derived from the User-Agent
//
// The raw User-Agent is never added.
type requestContextHandler struct {
	// inner is the wrapped handler that receives enriched records.
	inner slog.Handler
}

// Enabled delegates to the inner handler.
//
// Takes level (slog.Level) which is the level to check.
//
// Returns bool which indicates whether the level is enabled.
func (h *requestContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle enriches the log record with PikoRequestCtx fields then delegates to the inner
// handler.
//
// Takes record (slog.Record) which is the log record to enrich.
//
// Returns error when the inner handler fails.
//
//nolint:gocritic // slog.Handler requires value receiver
func (h *requestContextHandler) Handle(ctx context.Context, record slog.Record) error {
	if pctx := daemon_dto.PikoRequestCtxFromContext(ctx); pctx != nil {
		if rid := pctx.RequestID(); rid != "" {
			record.AddAttrs(slog.String("request_id", rid))
		}
		if pctx.ClientIP != "" {
			record.AddAttrs(slog.String("client_ip", pctx.ClientIP))
		}
		if pctx.Locale != "" {
			record.AddAttrs(slog.String("locale", pctx.Locale))
		}
		if record.Level >= slog.LevelError {
			addErrorContext(&record, pctx)
		}
	}
	return h.inner.Handle(ctx, record)
}

// addErrorContext attaches the diagnostic fields that are only worth their allocation on
// an error: the matched route, and the coarse client families derived from the
// User-Agent.
//
// Takes record (*slog.Record) which is enriched in place.
// Takes pctx (*daemon_dto.PikoRequestCtx) which carries the request-scoped values.
func addErrorContext(record *slog.Record, pctx *daemon_dto.PikoRequestCtx) {
	if pctx.MatchedPattern != "" {
		record.AddAttrs(slog.String("route", pctx.MatchedPattern))
	}
	if pctx.UserAgent == "" {
		return
	}
	class := pctx.UserAgentClass()
	if class.Browser != "" {
		record.AddAttrs(slog.String("browser", class.Browser))
	}
	if class.BrowserMajor != "" {
		record.AddAttrs(slog.String("browser_version", class.BrowserMajor))
	}
	if class.OS != "" {
		record.AddAttrs(slog.String("os", class.OS))
	}
	if class.Device != "" {
		record.AddAttrs(slog.String("device", class.Device))
	}
}

// WithAttrs returns a new handler wrapping the inner handler's WithAttrs result.
//
// Takes attrs ([]slog.Attr) which are the attributes to add.
//
// Returns slog.Handler which wraps the enriched inner handler.
func (h *requestContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &requestContextHandler{inner: h.inner.WithAttrs(attrs)}
}

// WithGroup returns a new handler wrapping the inner handler's WithGroup result.
//
// Takes name (string) which is the group name to apply.
//
// Returns slog.Handler which wraps the grouped inner handler.
func (h *requestContextHandler) WithGroup(name string) slog.Handler {
	return &requestContextHandler{inner: h.inner.WithGroup(name)}
}

// NewRequestContextHandler wraps the given handler so that every log record produced
// within an HTTP request is automatically enriched with request-scoped fields from
// PikoRequestCtx.
//
// Takes inner (slog.Handler) which is the handler to delegate to after enrichment.
//
// Returns slog.Handler which enriches records then delegates to inner.
func NewRequestContextHandler(inner slog.Handler) slog.Handler {
	return &requestContextHandler{inner: inner}
}
