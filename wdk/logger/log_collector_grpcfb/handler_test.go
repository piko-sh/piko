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

package log_collector_grpcfb

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

func TestToErrorLiftsFields(t *testing.T) {
	r := slog.NewRecord(time.Unix(0, 0), slog.LevelError, "boom", 0)
	line := telemetry_grpcfb.LogLine{Message: "boom", Logger: "svc"}
	ev := New(nil).toError(&r, line, nil, extractedFields{culprit: "doThing", stack: "frame1\nframe2", errSuffix: "disk full"}, "")
	assert.Equal(t, "doThing", ev.Culprit)
	assert.Equal(t, "boom: disk full", ev.Value)
	assert.NotEmpty(t, ev.StackJSON, "StackJSON should be a JSON array of frames")
	assert.Equal(t, "svc", ev.Type)
	assert.True(t, ev.Handled)
}

func TestToErrorBoundsStackAndValue(t *testing.T) {
	r := slog.NewRecord(time.Unix(0, 0), slog.LevelError, strings.Repeat("a", maxErrorValueLen*2), 0)
	line := telemetry_grpcfb.LogLine{Message: r.Message}
	ev := New(nil).toError(&r, line, nil, extractedFields{stack: strings.Repeat("x\n", maxStackLen)}, "")
	assert.LessOrEqual(t, len(ev.Value), maxErrorValueLen)

	assert.Less(t, len(ev.StackJSON), 4*maxStackLen, "stack is bounded before JSON wrapping")
}

func TestRateLimiterBurst(t *testing.T) {
	l := newRateLimiter(0, 3, clock.RealClock())
	allowed := 0
	for range 10 {
		if first(l.allow()) {
			allowed++
		}
	}
	assert.Equal(t, 3, allowed, "burst allowance")

	var nilLimiter *rateLimiter
	assert.True(t, first(nilLimiter.allow()), "nil limiter must allow")
}

func TestRateLimiterRefill(t *testing.T) {
	mock := clock.NewMockClock(time.Unix(1_700_000_000, 0))
	l := newRateLimiter(10, 1, mock)
	assert.True(t, first(l.allow()), "first call spends the burst token")
	assert.False(t, first(l.allow()), "no tokens left without refill")
	mock.Advance(200 * time.Millisecond)
	assert.True(t, first(l.allow()), "refilled after advancing the clock")
}

func TestFingerprintDigitStable(t *testing.T) {
	a := fingerprint("svc", "fn", "request 123 failed")
	b := fingerprint("svc", "fn", "request 456 failed")
	assert.Equal(t, a, b, "fingerprint should be digit-stable")
}

func TestCapFieldUTF8Safe(t *testing.T) {
	long := strings.Repeat("é", maxFieldValueLen)
	got := capField(long)
	assert.LessOrEqual(t, len(got), maxFieldValueLen)
	assert.True(t, utf8.ValidString(got), "truncated value must remain valid UTF-8")
}

func TestCollectFieldsLiftsAndExtracts(t *testing.T) {
	h := &Handler{group: "grp"}
	r := slog.NewRecord(time.Unix(0, 0), slog.LevelError, "boom", 0)
	r.AddAttrs(
		slog.String("trace_id", "t-1"),
		slog.String("span_id", "s-1"),
		slog.String("logger", "mysvc"),
		slog.String("functionName", "doThing"),
		slog.String("error", "disk full"),
		slog.String("plan", "pro"),
	)
	var line telemetry_grpcfb.LogLine
	fields, extracted := h.collectFields(&r, &line)

	assert.Equal(t, "t-1", line.TraceID)
	assert.Equal(t, "s-1", line.SpanID)
	assert.Equal(t, "mysvc", line.Logger)
	assert.Equal(t, "doThing", extracted.culprit)
	assert.Equal(t, "disk full", extracted.errSuffix)

	keys := make(map[string]string, len(fields))
	for _, f := range fields {
		keys[f.Key] = f.Value
	}
	assert.Equal(t, "pro", keys["grp.plan"], "generic attrs are group-prefixed")
	assert.Equal(t, "disk full", keys["grp.error"], "error attr is also kept as a field")
	assert.NotContains(t, keys, "grp.trace_id", "lifted attrs are not duplicated as fields")
}

func TestEnabledRespectsLevel(t *testing.T) {
	h := New(nil).WithLevel(slog.LevelWarn)
	assert.False(t, h.Enabled(context.Background(), slog.LevelInfo))
	assert.True(t, h.Enabled(context.Background(), slog.LevelError))
}

func TestHandleNilClientIsNoop(t *testing.T) {
	h := New(nil)
	r := slog.NewRecord(time.Unix(0, 0), slog.LevelInfo, "hi", 0)
	require.NoError(t, h.Handle(context.Background(), r))
}

func TestWithAttrsPresetsFields(t *testing.T) {
	h, ok := New(nil).WithAttrs([]slog.Attr{slog.String("preset", "v")}).(*Handler)
	require.True(t, ok)
	r := slog.NewRecord(time.Unix(0, 0), slog.LevelInfo, "m", 0)
	var line telemetry_grpcfb.LogLine
	fields, _ := h.collectFields(&r, &line)
	require.Len(t, fields, 1)
	assert.Equal(t, "preset", fields[0].Key)
	assert.Equal(t, "v", fields[0].Value)
}

func TestWithGroupNamespacesRecordAttrs(t *testing.T) {
	h, ok := New(nil).WithGroup("g").(*Handler)
	require.True(t, ok)
	r := slog.NewRecord(time.Unix(0, 0), slog.LevelInfo, "m", 0)
	r.AddAttrs(slog.String("plan", "pro"))
	var line telemetry_grpcfb.LogLine
	fields, _ := h.collectFields(&r, &line)
	require.Len(t, fields, 1)
	assert.Equal(t, "g.plan", fields[0].Key)
}

type logSink struct {
	lines []telemetry_grpcfb.LogLine
	errs  []telemetry_grpcfb.ErrorEvent
	mu    sync.Mutex
}

func (s *logSink) Ingest(stream telemetry_grpcfb.IngestStream) error {
	for {
		b, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.lines = append(s.lines, b.Logs...)
		s.errs = append(s.errs, b.Errors...)
		s.mu.Unlock()
	}
	return stream.SendAndClose(&telemetry_grpcfb.IngestAck{OK: true})
}

func (s *logSink) snapshot() ([]telemetry_grpcfb.LogLine, []telemetry_grpcfb.ErrorEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]telemetry_grpcfb.LogLine(nil), s.lines...), append([]telemetry_grpcfb.ErrorEvent(nil), s.errs...)
}

func startSink(t *testing.T) (*logSink, *bufconn.Listener, func()) {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := telemetry_grpcfb.NewServer()
	snk := &logSink{}
	telemetry_grpcfb.RegisterServer(srv, snk)
	go func() { _ = srv.Serve(lis) }()
	return snk, lis, srv.Stop
}

func bufDialOpts(lis *bufconn.Listener) []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
}

func dialSink(t *testing.T, lis *bufconn.Listener) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient("passthrough:///bufnet", append(bufDialOpts(lis),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(telemetry_grpcfb.Codec{})))...)
	require.NoError(t, err)
	return conn
}

func TestHandleForwardsLogAndError(t *testing.T) {
	snk, lis, stop := startSink(t)
	defer stop()
	conn := dialSink(t, lis)
	defer conn.Close()

	client := telemetry_grpcfb.New(conn, telemetry_grpcfb.Config{SiteID: "s", FlushInterval: time.Hour})
	client.Start(context.Background())
	h := New(client)

	rec := slog.NewRecord(time.Unix(0, 0), slog.LevelError, "disk failed", 0)
	rec.AddAttrs(slog.String("logger", "store"), slog.String("error", "no space"))
	require.NoError(t, h.Handle(context.Background(), rec))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.Close(ctx))

	lines, errs := snk.snapshot()
	require.Len(t, lines, 1)
	assert.Equal(t, "disk failed", lines[0].Message)
	require.Len(t, errs, 1)
	assert.Equal(t, "disk failed: no space", errs[0].Value)
	assert.Equal(t, "store", errs[0].Type)
}

func TestWithErrorEventsDisablesErrorForwarding(t *testing.T) {
	snk, lis, stop := startSink(t)
	defer stop()
	conn := dialSink(t, lis)
	defer conn.Close()

	client := telemetry_grpcfb.New(conn, telemetry_grpcfb.Config{SiteID: "s", FlushInterval: time.Hour})
	client.Start(context.Background())
	h := New(client).WithErrorEvents(false)

	rec := slog.NewRecord(time.Unix(0, 0), slog.LevelError, "boom", 0)
	require.NoError(t, h.Handle(context.Background(), rec))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.Close(ctx))

	lines, errs := snk.snapshot()
	assert.Len(t, lines, 1)
	assert.Empty(t, errs, "ErrorEvents must be suppressed when WithErrorEvents(false)")
}

func TestDialOwnsAndClosesClient(t *testing.T) {
	_, lis, stop := startSink(t)
	defer stop()

	h, err := Dial("passthrough:///bufnet", telemetry_grpcfb.Config{SiteID: "s", FlushInterval: time.Hour}, bufDialOpts(lis)...)
	require.NoError(t, err)
	require.True(t, h.owns, "Dial-built handler must own its client")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rec := slog.NewRecord(time.Unix(0, 0), slog.LevelInfo, "hello", 0)
	require.NoError(t, h.Handle(ctx, rec))
	require.NoError(t, h.Close(ctx))
}

func TestCloseSharedHandlerIsNoop(t *testing.T) {
	client := telemetry_grpcfb.New(nil, telemetry_grpcfb.Config{SiteID: "s"})
	h := New(client)
	require.NoError(t, h.Close(context.Background()))
}
