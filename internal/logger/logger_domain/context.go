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

	"piko.sh/piko/internal/daemon/daemon_dto"
)

// loggerCtxKey is the context key for storing enriched loggers. Using a private struct
// type prevents key clashes with other packages.
type loggerCtxKey struct{}

var (
	// defaultContextLogger is lazily initialised on first use via resolveDefaultLogger. This
	// avoids init-order issues while providing a sensible fallback.
	defaultContextLogger Logger
)

// WithLogger stores a logger in the given context for later retrieval.
//
// Request-scoped data (such as request_id or user_id) then flows through the call stack
// without passing the logger as a parameter.
//
// Takes l (Logger) which is the logger to store in the context.
//
// Returns context.Context which contains the stored logger.
func WithLogger(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, l)
}

// From retrieves the logger from context, using the PikoRequestCtx cache on HTTP request
// paths for zero-allocation retrieval.
//
// Request path (PikoRequestCtx present):
//   - Hot path: returns the logger cached on the PikoRequestCtx.
//   - Cold path (first call per request): binds the fallback to the request context via
//     WithSpanContext and caches it on the PikoRequestCtx. When several goroutines of one
//     request race to cache, all of them return the logger that was stored first.
//
// Non-request path (no PikoRequestCtx):
//   - Falls back to loggerCtxKey{} lookup and context.WithValue storage.
//
// Takes fallback (Logger) which is the logger to use when none is in context. Pass nil to
// fall back to a global default logger.
//
// Returns context.Context which contains the logger for downstream retrieval.
// Returns Logger which is the context logger, fallback, or global default.
//
// Cost: O(1) on request hot path, O(n) context lookup on non-request paths.
func From(ctx context.Context, fallback Logger) (context.Context, Logger) {
	if ctx == nil {
		ctx = context.Background()
	}

	if pctx := daemon_dto.PikoRequestCtxFromContext(ctx); pctx != nil {
		if l, ok := pctx.CachedLoggerValue().(Logger); ok {
			return ctx, l
		}

		return ctx, cacheOnCarrier(pctx, resolveFallback(fallback).WithSpanContext(ctx))
	}

	if l, ok := ctx.Value(loggerCtxKey{}).(Logger); ok {
		return ctx, l
	}
	resolved := resolveFallback(fallback)
	return context.WithValue(ctx, loggerCtxKey{}, resolved), resolved
}

// MustFrom retrieves the logger from context, panicking if not present.
//
// Checks the logger cached on the PikoRequestCtx first (request paths), then falls back
// to loggerCtxKey{}. On a request path whose carrier has no cached logger yet, the
// loggerCtxKey{} logger is bound to ctx and cached, so a context detached for background
// work never logs through a logger bound to the original request.
//
// Use this in code paths where a logger MUST be in context (e.g., after middleware that
// guarantees it). Panicking early catches middleware misconfiguration during development.
//
// Returns Logger which is the context logger.
//
// Panics if no logger is stored in context.
func MustFrom(ctx context.Context) Logger {
	pctx := daemon_dto.PikoRequestCtxFromContext(ctx)
	if pctx != nil {
		if l, ok := pctx.CachedLoggerValue().(Logger); ok {
			return l
		}
	}
	l, ok := ctx.Value(loggerCtxKey{}).(Logger)
	if !ok {
		panic("logger: no logger in context - ensure middleware calls WithLogger")
	}
	if pctx != nil {
		return cacheOnCarrier(pctx, l.WithSpanContext(ctx))
	}
	return l
}

// HasLogger reports whether a logger is stored in the context.
//
// Checks the logger cached on the PikoRequestCtx first (request paths), then falls back
// to loggerCtxKey{}.
//
// Returns bool which is true when a logger is present, false otherwise.
func HasLogger(ctx context.Context) bool {
	if pctx := daemon_dto.PikoRequestCtxFromContext(ctx); pctx != nil {
		if _, ok := pctx.CachedLoggerValue().(Logger); ok {
			return true
		}
	}
	_, ok := ctx.Value(loggerCtxKey{}).(Logger)
	return ok
}

// cacheOnCarrier caches candidate on the request carrier unless another goroutine cached
// a logger first.
//
// Takes pctx (*daemon_dto.PikoRequestCtx) which is the request carrier.
// Takes candidate (Logger) which is the logger to cache.
//
// Returns Logger which is the logger now cached on the carrier.
func cacheOnCarrier(pctx *daemon_dto.PikoRequestCtx, candidate Logger) Logger {
	if stored, ok := pctx.StoreCachedLogger(candidate).(Logger); ok {
		return stored
	}
	return candidate
}

// resolveFallback returns the fallback logger if non-nil, otherwise lazily initialises
// and returns the global default context logger.
//
// Takes fallback (Logger) which is the preferred logger, or nil.
//
// Returns Logger which is the fallback or the global default.
func resolveFallback(fallback Logger) Logger {
	if fallback != nil {
		return fallback
	}
	if defaultContextLogger == nil {
		defaultContextLogger = GetLogger("piko/context")
	}
	return defaultContextLogger
}
