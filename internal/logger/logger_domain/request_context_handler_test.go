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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/daemon/daemon_dto"
)

type attrSink struct {
	last  map[string]string
	count int
}

func (s *attrSink) Enabled(context.Context, slog.Level) bool { return true }

func (s *attrSink) Handle(_ context.Context, r slog.Record) error {
	flat := make(map[string]string, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		flat[a.Key] = a.Value.String()
		return true
	})
	s.last = flat
	s.count++
	return nil
}

func (s *attrSink) WithAttrs([]slog.Attr) slog.Handler { return s }

func (s *attrSink) WithGroup(string) slog.Handler { return s }

func requestCtx() context.Context {
	pctx := &daemon_dto.PikoRequestCtx{
		ForwardedRequestID: "req-abc123",
		ClientIP:           "203.0.113.7",
		Locale:             "en",
		MatchedPattern:     "/blog/{slug}",
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
			"(KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	}
	return daemon_dto.WithPikoRequestCtx(context.Background(), pctx)
}

func handle(t *testing.T, level slog.Level) map[string]string {
	t.Helper()
	sink := &attrSink{}
	handler := NewRequestContextHandler(sink)
	record := slog.NewRecord(time.Now(), level, "msg", 0)

	require.NoError(t, handler.Handle(requestCtx(), record))
	require.Equal(t, 1, sink.count, "the record reaches the inner handler exactly once")

	return sink.last
}

func TestRequestContextHandler_InfoStaysCheap(t *testing.T) {
	got := handle(t, slog.LevelInfo)

	for _, key := range []string{"request_id", "client_ip", "locale"} {
		assert.Contains(t, got, key, "Info records carry the always-on fields")
	}
	for _, key := range []string{"route", "browser", "browser_version", "os", "device"} {
		assert.NotContains(t, got, key, "the diagnostic fields appear only at Error and above")
	}
	assert.LessOrEqual(t, len(got), 3, "the enrichment stays within the inline attribute budget")
}

func TestRequestContextHandler_ErrorCarriesDiagnostics(t *testing.T) {
	got := handle(t, slog.LevelError)

	want := map[string]string{
		"client_ip":       "203.0.113.7",
		"locale":          "en",
		"route":           "/blog/{slug}",
		"browser":         "Chrome",
		"browser_version": "131",
		"os":              "macOS",
		"device":          "desktop",
	}
	for key, value := range want {
		assert.Equal(t, value, got[key], "record carries %q", key)
	}
}

func TestRequestContextHandler_NeverLogsRawUserAgent(t *testing.T) {
	got := handle(t, slog.LevelError)

	for key, value := range got {
		require.NotContains(t, value, "AppleWebKit", "record[%q] carries the raw User-Agent", key)
		require.NotContains(t, value, "Mozilla/", "record[%q] carries the raw User-Agent", key)
	}
	require.NotContains(t, got, "user_agent")
}

func TestRequestContextHandler_NoCarrierIsPassthrough(t *testing.T) {
	sink := &attrSink{}
	handler := NewRequestContextHandler(sink)
	record := slog.NewRecord(time.Now(), slog.LevelError, "msg", 0)

	require.NoError(t, handler.Handle(context.Background(), record))
	require.Empty(t, sink.last, "a record produced outside a request is passed through unchanged")
}

func TestRequestContextHandler_OmitsUnknownFamilies(t *testing.T) {
	sink := &attrSink{}
	handler := NewRequestContextHandler(sink)
	pctx := &daemon_dto.PikoRequestCtx{MatchedPattern: "/health"}
	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), pctx)
	record := slog.NewRecord(time.Now(), slog.LevelError, "msg", 0)

	require.NoError(t, handler.Handle(ctx, record))

	for _, key := range []string{"browser", "browser_version", "os", "device"} {
		assert.NotContains(t, sink.last, key, "an undetermined family is absent, not blank")
	}
	assert.Equal(t, "/health", sink.last["route"])
}

type discardSink struct{}

func (discardSink) Enabled(context.Context, slog.Level) bool { return true }

func (discardSink) Handle(context.Context, slog.Record) error { return nil }

func (d discardSink) WithAttrs([]slog.Attr) slog.Handler { return d }

func (d discardSink) WithGroup(string) slog.Handler { return d }

func BenchmarkRequestContextHandler_Info(b *testing.B) {
	h := NewRequestContextHandler(discardSink{})
	ctx := requestCtx()
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)

	b.ReportAllocs()
	for range b.N {
		_ = h.Handle(ctx, rec)
	}
}

func TestRequestContextHandler_InfoDoesNotAllocate(t *testing.T) {
	handler := NewRequestContextHandler(discardSink{})
	ctx := requestCtx()

	allocations := testing.AllocsPerRun(100, func() {
		record := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
		record.AddAttrs(slog.String("order", "o-1"), slog.Int("items", 3))
		_ = handler.Handle(ctx, record)
	})

	assert.Zero(t, allocations,
		"three enrichment fields plus two caller attributes stay within the inline array")
}

func TestRequestContextHandler_ErrorMayAllocate(t *testing.T) {
	ctx := requestCtx()

	sink := &attrSink{}
	errorHandler := NewRequestContextHandler(sink)
	require.NoError(t, errorHandler.Handle(ctx, slog.NewRecord(time.Now(), slog.LevelError, "msg", 0)))
	assert.Greater(t, len(sink.last), 3, "Error records carry the diagnostic fields as well")
}
