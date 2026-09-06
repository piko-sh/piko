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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/daemon/daemon_dto"
)

type recordedEntry struct {
	attrs   map[string]string
	message string
}

type recordingSink struct {
	entries []recordedEntry
	mu      sync.Mutex
}

func (*recordingSink) Enabled(context.Context, slog.Level) bool { return true }

func (s *recordingSink) Handle(_ context.Context, record slog.Record) error {
	flat := make(map[string]string, record.NumAttrs())
	record.Attrs(func(attribute slog.Attr) bool {
		flat[attribute.Key] = attribute.Value.String()
		return true
	})

	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, recordedEntry{attrs: flat, message: record.Message})

	return nil
}

func (s *recordingSink) WithAttrs([]slog.Attr) slog.Handler { return s }

func (s *recordingSink) WithGroup(string) slog.Handler { return s }

func (s *recordingSink) attrsFor(t *testing.T, message string) map[string]string {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, entry := range s.entries {
		if entry.message == message {
			return entry.attrs
		}
	}
	require.Failf(t, "message not logged", "no record with message %q", message)

	return nil
}

func newRecordingLogger() (Logger, *recordingSink) {
	sink := &recordingSink{}
	return New(slog.New(NewRequestContextHandler(sink)), "piko/context_test"), sink
}

func newRequest(clientIP string) (context.Context, *daemon_dto.PikoRequestCtx) {
	pctx := daemon_dto.AcquirePikoRequestCtx()
	pctx.ClientIP = clientIP
	return daemon_dto.WithPikoRequestCtx(context.Background(), pctx), pctx
}

func TestFrom_DetachedContextKeepsLoggingTheOriginalRequestAfterRelease(t *testing.T) {
	fallback, sink := newRecordingLogger()

	requestCtx, pctx := newRequest("10.0.0.1")
	_, requestLogger := From(requestCtx, fallback)
	requestLogger.Info("request handled")

	backgroundCtx := daemon_dto.DetachRequestContext(requestCtx)
	start := make(chan struct{})
	var waitGroup sync.WaitGroup
	waitGroup.Go(func() {
		<-start
		_, backgroundLogger := From(backgroundCtx, fallback)
		backgroundLogger.Error("background persistence failed")
	})

	daemon_dto.ReleasePikoRequestCtx(pctx)

	nextCtx, next := newRequest("192.168.9.9")
	defer daemon_dto.ReleasePikoRequestCtx(next)
	close(start)
	_, nextLogger := From(nextCtx, fallback)
	nextLogger.Info("next request")
	waitGroup.Wait()

	assert.Equal(t, "10.0.0.1", sink.attrsFor(t, "request handled")["client_ip"])
	assert.Equal(t, "10.0.0.1", sink.attrsFor(t, "background persistence failed")["client_ip"],
		"background work must keep the tags of the request that started it")
	assert.Equal(t, "192.168.9.9", sink.attrsFor(t, "next request")["client_ip"])
}

func TestFrom_ConcurrentCallsShareOneRequestLogger(t *testing.T) {
	fallback, _ := newRecordingLogger()
	requestCtx, pctx := newRequest("10.0.0.1")
	defer daemon_dto.ReleasePikoRequestCtx(pctx)

	const callers = 8
	loggers := make([]Logger, callers)
	var waitGroup sync.WaitGroup
	for caller := range callers {
		waitGroup.Go(func() {
			_, loggers[caller] = From(requestCtx, fallback)
		})
	}
	waitGroup.Wait()

	for caller, l := range loggers {
		assert.Same(t, pctx.CachedLoggerValue(), l, "caller %d received a logger that was not cached", caller)
	}
}

func TestFrom_WithoutCarrierStoresTheFallbackOnTheContext(t *testing.T) {
	fallback, _ := newRecordingLogger()

	ctx, l := From(context.Background(), fallback)

	assert.Same(t, fallback, l)
	_, again := From(ctx, nil)
	assert.Same(t, fallback, again, "the stored logger wins over the fallback")
}

func TestMustFrom(t *testing.T) {
	fallback, sink := newRecordingLogger()

	testCases := []struct {
		build     func(t *testing.T) context.Context
		name      string
		wantPanic bool
	}{
		{
			name: "returns the logger cached on the carrier",
			build: func(t *testing.T) context.Context {
				t.Helper()
				ctx, pctx := newRequest("10.0.0.1")
				t.Cleanup(func() { daemon_dto.ReleasePikoRequestCtx(pctx) })
				_, _ = From(ctx, fallback)
				return ctx
			},
		},
		{
			name: "binds a context logger to the detached carrier",
			build: func(t *testing.T) context.Context {
				t.Helper()
				ctx, pctx := newRequest("10.0.0.1")
				ctx = WithLogger(ctx, fallback.WithSpanContext(ctx))
				detached := daemon_dto.DetachRequestContext(ctx)
				daemon_dto.ReleasePikoRequestCtx(pctx)
				return detached
			},
		},
		{
			name: "returns a context logger without a carrier",
			build: func(t *testing.T) context.Context {
				t.Helper()
				return WithLogger(context.Background(), fallback)
			},
		},
		{
			name: "panics without any logger",
			build: func(t *testing.T) context.Context {
				t.Helper()
				ctx, pctx := newRequest("10.0.0.1")
				t.Cleanup(func() { daemon_dto.ReleasePikoRequestCtx(pctx) })
				return ctx
			},
			wantPanic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.build(t)
			if tc.wantPanic {
				assert.Panics(t, func() { MustFrom(ctx) })
				return
			}

			l := MustFrom(ctx)
			require.NotNil(t, l)
			l.Info(tc.name)
			if daemon_dto.PikoRequestCtxFromContext(ctx) != nil {
				assert.Equal(t, "10.0.0.1", sink.attrsFor(t, tc.name)["client_ip"])
			}
		})
	}
}

func TestHasLogger(t *testing.T) {
	fallback, _ := newRecordingLogger()

	cachedCtx, cachedCarrier := newRequest("10.0.0.1")
	defer daemon_dto.ReleasePikoRequestCtx(cachedCarrier)
	_, _ = From(cachedCtx, fallback)

	emptyCtx, emptyCarrier := newRequest("10.0.0.2")
	defer daemon_dto.ReleasePikoRequestCtx(emptyCarrier)

	testCases := []struct {
		ctx  context.Context
		name string
		want bool
	}{
		{name: "logger cached on the carrier", ctx: cachedCtx, want: true},
		{name: "carrier without a logger", ctx: emptyCtx, want: false},
		{name: "context logger", ctx: WithLogger(context.Background(), fallback), want: true},
		{name: "no logger", ctx: context.Background(), want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, HasLogger(tc.ctx))
		})
	}
}

func TestFrom_IgnoresAForeignValueCachedOnTheCarrier(t *testing.T) {
	fallback, _ := newRecordingLogger()
	requestCtx, pctx := newRequest("10.0.0.1")
	defer daemon_dto.ReleasePikoRequestCtx(pctx)
	pctx.StoreCachedLogger("not a logger")

	_, l := From(requestCtx, fallback)

	require.NotNil(t, l)
	assert.False(t, HasLogger(requestCtx), "a foreign value is not a logger")
}
