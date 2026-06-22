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

package watchdog_grpcfb_test

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
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

	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/wdk/monitoring/watchdog_grpcfb"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

const (
	maxInlineProfile = 12 << 20
	maxMessageLen    = 4096
	maxFieldValueLen = 1024
	maxFields        = 128
)

type sink struct {
	batches []*telemetry_grpcfb.Batch
	mu      sync.Mutex
}

func (s *sink) Ingest(stream telemetry_grpcfb.IngestStream) error {
	for {
		b, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.batches = append(s.batches, b)
		s.mu.Unlock()
	}
	return stream.SendAndClose(&telemetry_grpcfb.IngestAck{OK: true})
}

func (s *sink) watchdogs() []telemetry_grpcfb.WatchdogEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []telemetry_grpcfb.WatchdogEvent
	for _, b := range s.batches {
		out = append(out, b.Watchdog...)
	}
	return out
}

func (s *sink) profiles() []telemetry_grpcfb.ProfileMeta {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []telemetry_grpcfb.ProfileMeta
	for _, b := range s.batches {
		out = append(out, b.Profiles...)
	}
	return out
}

type harness struct {
	sink   *sink
	client *telemetry_grpcfb.Client
}

func newHarness(t *testing.T, cfg telemetry_grpcfb.Config) *harness {
	t.Helper()

	lis := bufconn.Listen(1 << 20)
	srv := telemetry_grpcfb.NewServer()
	snk := &sink{}
	telemetry_grpcfb.RegisterServer(srv, snk)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn := dial(t, lis)
	t.Cleanup(func() { _ = conn.Close() })

	cfg.FlushSize = 1
	cfg.FlushInterval = time.Hour
	client := telemetry_grpcfb.New(conn, cfg)

	return &harness{sink: snk, client: client}
}

func dial(t *testing.T, lis *bufconn.Listener) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(telemetry_grpcfb.Codec{})),
	)
	require.NoError(t, err)
	return conn
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestNotifierStreamsEventsAndProfiles(t *testing.T) {
	h := newHarness(t, telemetry_grpcfb.Config{SiteID: "site-w", APIKey: "key-w"})
	n := watchdog_grpcfb.New(h.client)

	ctx := testContext(t)
	n.Start(ctx)

	require.NoError(t, n.Notify(ctx, monitoring_domain.WatchdogEvent{
		EventType: monitoring_domain.WatchdogEventHeapThresholdExceeded,
		Priority:  monitoring_domain.WatchdogPriorityHigh,
		Message:   "heap above threshold",
		Fields:    map[string]string{"heap_bytes": "1048576"},
	}))

	require.NoError(t, n.Upload(ctx, "heap", []byte("PROFILEBYTES"), map[string]string{
		"reason":     "threshold",
		"goroutines": "100",
	}))

	require.NoError(t, n.Close(ctx))
	require.NoError(t, h.client.Close(ctx))

	h.sink.mu.Lock()
	for _, b := range h.sink.batches {
		assert.Equal(t, "site-w", b.SiteID)
		assert.Equal(t, "key-w", b.APIKey)
	}
	h.sink.mu.Unlock()

	watchdogs := h.sink.watchdogs()
	require.Len(t, watchdogs, 1)
	got := watchdogs[0]
	assert.Equal(t, string(monitoring_domain.WatchdogEventHeapThresholdExceeded), got.EventType)
	assert.Equal(t, int32(monitoring_domain.WatchdogPriorityHigh), got.Priority)
	assert.Equal(t, "heap above threshold", got.Message)
	assert.Positive(t, got.TimestampMs)
	require.Len(t, got.Fields, 1)
	assert.Equal(t, "heap_bytes", got.Fields[0].Key)
	assert.Equal(t, "1048576", got.Fields[0].Value)

	profiles := h.sink.profiles()
	require.Len(t, profiles, 1)
	prof := profiles[0]
	assert.Equal(t, "heap", prof.ProfileType)
	assert.Equal(t, "PROFILEBYTES", string(prof.Blob))
	assert.Equal(t, int64(len("PROFILEBYTES")), prof.SizeBytes)
	assert.Equal(t, "gzip", prof.ContentEncoding)
	assert.Equal(t, "threshold", prof.Reason)
	assert.Positive(t, prof.TimestampMs)

	assert.Len(t, prof.Fields, 2)
}

func TestNotifyTruncatesMessageAndCapsFields(t *testing.T) {
	h := newHarness(t, telemetry_grpcfb.Config{SiteID: "site-trunc"})
	n := watchdog_grpcfb.New(h.client)

	ctx := testContext(t)
	n.Start(ctx)

	longMessage := strings.Repeat("m", maxMessageLen+512)
	fields := make(map[string]string, maxFields+50)
	for i := range maxFields + 50 {

		fields["field-"+strconv.Itoa(i)] = strings.Repeat("v", maxFieldValueLen+128)
	}

	require.NoError(t, n.Notify(ctx, monitoring_domain.WatchdogEvent{
		EventType: monitoring_domain.WatchdogEventHeapThresholdExceeded,
		Priority:  monitoring_domain.WatchdogPriorityCritical,
		Message:   longMessage,
		Fields:    fields,
	}))

	require.NoError(t, h.client.Close(ctx))

	watchdogs := h.sink.watchdogs()
	require.Len(t, watchdogs, 1)
	got := watchdogs[0]

	assert.LessOrEqual(t, len(got.Message), maxMessageLen)
	assert.True(t, utf8.ValidString(got.Message))
	assert.Equal(t, strings.Repeat("m", maxMessageLen), got.Message)

	assert.Len(t, got.Fields, maxFields)
	for _, f := range got.Fields {
		assert.LessOrEqual(t, len(f.Value), maxFieldValueLen)
		assert.True(t, utf8.ValidString(f.Value))
	}
}

func TestNotifyTruncatesMultiByteMessageAtRuneBoundary(t *testing.T) {
	h := newHarness(t, telemetry_grpcfb.Config{})
	n := watchdog_grpcfb.New(h.client)

	ctx := testContext(t)
	n.Start(ctx)

	message := strings.Repeat("é", maxMessageLen)
	require.NoError(t, n.Notify(ctx, monitoring_domain.WatchdogEvent{
		EventType: monitoring_domain.WatchdogEventHeapThresholdExceeded,
		Message:   message,
	}))
	require.NoError(t, h.client.Close(ctx))

	watchdogs := h.sink.watchdogs()
	require.Len(t, watchdogs, 1)
	got := watchdogs[0].Message
	assert.LessOrEqual(t, len(got), maxMessageLen)
	assert.True(t, utf8.ValidString(got), "truncation must not sever a multi-byte rune")
}

func TestUploadCarriesBlobInlineForSmallData(t *testing.T) {
	h := newHarness(t, telemetry_grpcfb.Config{})
	n := watchdog_grpcfb.New(h.client)

	ctx := testContext(t)
	n.Start(ctx)

	data := []byte("a small gzip-ish blob")
	require.NoError(t, n.Upload(ctx, "cpu", data, map[string]string{"reason": "manual"}))
	require.NoError(t, h.client.Close(ctx))

	profiles := h.sink.profiles()
	require.Len(t, profiles, 1)
	prof := profiles[0]
	assert.Equal(t, "cpu", prof.ProfileType)
	assert.Equal(t, data, prof.Blob)
	assert.Equal(t, int64(len(data)), prof.SizeBytes)
	assert.Equal(t, "manual", prof.Reason)
	for _, f := range prof.Fields {
		assert.NotEqual(t, "blob_omitted", f.Key, "small blobs must travel inline")
	}
}

func TestUploadOmitsOversizeBlob(t *testing.T) {
	h := newHarness(t, telemetry_grpcfb.Config{})
	n := watchdog_grpcfb.New(h.client)

	ctx := testContext(t)
	n.Start(ctx)

	data := make([]byte, maxInlineProfile+1)
	require.NoError(t, n.Upload(ctx, "heap", data, map[string]string{"reason": "huge"}))
	require.NoError(t, h.client.Close(ctx))

	profiles := h.sink.profiles()
	require.Len(t, profiles, 1)
	prof := profiles[0]
	assert.Empty(t, prof.Blob, "oversize blob must be dropped")
	assert.Equal(t, int64(len(data)), prof.SizeBytes, "reported size still reflects the original bytes")

	var omitted string
	var found bool
	for _, f := range prof.Fields {
		if f.Key == "blob_omitted" {
			omitted = f.Value
			found = true
		}
	}
	require.True(t, found, "an omitted oversize blob must be noted in the fields")
	assert.Equal(t, "oversize", omitted)
}

func TestUploadAtInlineBoundaryKeepsBlob(t *testing.T) {
	h := newHarness(t, telemetry_grpcfb.Config{})
	n := watchdog_grpcfb.New(h.client)

	ctx := testContext(t)
	n.Start(ctx)

	data := make([]byte, maxInlineProfile)
	for i := range data {
		data[i] = byte(i)
	}
	require.NoError(t, n.Upload(ctx, "heap", data, nil))
	require.NoError(t, h.client.Close(ctx))

	profiles := h.sink.profiles()
	require.Len(t, profiles, 1)
	prof := profiles[0]
	require.Len(t, prof.Blob, maxInlineProfile)
	assert.Equal(t, data, prof.Blob)
	for _, f := range prof.Fields {
		assert.NotEqual(t, "blob_omitted", f.Key)
	}
}

func TestUploadEmptyProfileTypeReturnsError(t *testing.T) {
	h := newHarness(t, telemetry_grpcfb.Config{})
	n := watchdog_grpcfb.New(h.client)

	ctx := testContext(t)
	n.Start(ctx)

	err := n.Upload(ctx, "", []byte("ignored"), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, watchdog_grpcfb.ErrEmptyProfileType)

	require.NoError(t, h.client.Close(ctx))

	assert.Empty(t, h.sink.profiles())
}

func TestUploadEmptyDataCarriesNoBlob(t *testing.T) {
	h := newHarness(t, telemetry_grpcfb.Config{})
	n := watchdog_grpcfb.New(h.client)

	ctx := testContext(t)
	n.Start(ctx)

	require.NoError(t, n.Upload(ctx, "block", nil, map[string]string{"reason": "empty"}))
	require.NoError(t, h.client.Close(ctx))

	profiles := h.sink.profiles()
	require.Len(t, profiles, 1)
	prof := profiles[0]
	assert.Empty(t, prof.Blob)
	assert.Equal(t, int64(0), prof.SizeBytes)
	for _, f := range prof.Fields {
		assert.NotEqual(t, "blob_omitted", f.Key, "empty data is not oversize")
	}
}

func TestCappedFieldsCountAndValueTruncation(t *testing.T) {
	tests := []struct {
		name         string
		fieldCount   int
		valueLen     int
		wantCount    int
		wantValueCap bool
	}{
		{
			name:       "below all caps",
			fieldCount: 3,
			valueLen:   16,
			wantCount:  3,
		},
		{
			name:         "value over cap is truncated",
			fieldCount:   2,
			valueLen:     maxFieldValueLen + 200,
			wantCount:    2,
			wantValueCap: true,
		},
		{
			name:       "count over cap is bounded",
			fieldCount: maxFields + 64,
			valueLen:   8,
			wantCount:  maxFields,
		},
		{
			name:         "both caps engaged",
			fieldCount:   maxFields + 10,
			valueLen:     maxFieldValueLen + 1,
			wantCount:    maxFields,
			wantValueCap: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, telemetry_grpcfb.Config{})
			n := watchdog_grpcfb.New(h.client)

			ctx := testContext(t)
			n.Start(ctx)

			fields := make(map[string]string, tc.fieldCount)
			for i := range tc.fieldCount {
				fields["field-"+strconv.Itoa(i)] = strings.Repeat("v", tc.valueLen)
			}

			require.NoError(t, n.Notify(ctx, monitoring_domain.WatchdogEvent{
				EventType: monitoring_domain.WatchdogEventHeapThresholdExceeded,
				Message:   "msg",
				Fields:    fields,
			}))
			require.NoError(t, h.client.Close(ctx))

			watchdogs := h.sink.watchdogs()
			require.Len(t, watchdogs, 1)
			got := watchdogs[0].Fields
			assert.Len(t, got, tc.wantCount)
			for _, f := range got {
				assert.True(t, utf8.ValidString(f.Value))
				if tc.wantValueCap {
					assert.LessOrEqual(t, len(f.Value), maxFieldValueLen)
				}
			}
		})
	}
}

func TestNotifyNilFieldsYieldsNoFields(t *testing.T) {
	h := newHarness(t, telemetry_grpcfb.Config{})
	n := watchdog_grpcfb.New(h.client)

	ctx := testContext(t)
	n.Start(ctx)

	require.NoError(t, n.Notify(ctx, monitoring_domain.WatchdogEvent{
		EventType: monitoring_domain.WatchdogEventHeapThresholdExceeded,
		Message:   "no fields",
	}))
	require.NoError(t, h.client.Close(ctx))

	watchdogs := h.sink.watchdogs()
	require.Len(t, watchdogs, 1)
	assert.Empty(t, watchdogs[0].Fields)
}

func TestFlushStreamsQueuedEvents(t *testing.T) {
	h := newHarness(t, telemetry_grpcfb.Config{})
	n := watchdog_grpcfb.New(h.client)

	ctx := testContext(t)
	n.Start(ctx)

	require.NoError(t, n.Notify(ctx, monitoring_domain.WatchdogEvent{
		EventType: monitoring_domain.WatchdogEventHeapThresholdExceeded,
		Message:   "flush me",
	}))
	require.NoError(t, n.Flush(ctx))

	require.NoError(t, h.client.Close(ctx))

	assert.Len(t, h.sink.watchdogs(), 1)
}

func TestCloseSharedClientOnlyFlushes(t *testing.T) {
	h := newHarness(t, telemetry_grpcfb.Config{})
	n := watchdog_grpcfb.New(h.client)

	ctx := testContext(t)
	n.Start(ctx)

	require.NoError(t, n.Notify(ctx, monitoring_domain.WatchdogEvent{
		EventType: monitoring_domain.WatchdogEventHeapThresholdExceeded,
		Message:   "first",
	}))

	require.NoError(t, n.Close(ctx))

	require.NoError(t, n.Notify(ctx, monitoring_domain.WatchdogEvent{
		EventType: monitoring_domain.WatchdogEventHeapThresholdExceeded,
		Message:   "second",
	}))

	require.NoError(t, h.client.Close(ctx))

	assert.Len(t, h.sink.watchdogs(), 2, "the shared client survives the notifier's Close")
}

func TestCloseOwnedClientDrainsAndCloses(t *testing.T) {
	lis := bufconn.Listen(1 << 20)
	srv := telemetry_grpcfb.NewServer()
	snk := &sink{}
	telemetry_grpcfb.RegisterServer(srv, snk)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	ctx := testContext(t)

	n, err := watchdog_grpcfb.Dial("passthrough:///bufnet",
		telemetry_grpcfb.Config{FlushSize: 1, FlushInterval: time.Hour},
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(telemetry_grpcfb.Codec{})),
	)
	require.NoError(t, err)

	n.Start(ctx)
	require.NoError(t, n.Notify(ctx, monitoring_domain.WatchdogEvent{
		EventType: monitoring_domain.WatchdogEventHeapThresholdExceeded,
		Message:   "owned",
	}))

	require.NoError(t, n.Close(ctx))

	assert.Len(t, snk.watchdogs(), 1)
}

func TestDialEmptyTargetReturnsError(t *testing.T) {
	n, err := watchdog_grpcfb.Dial("", telemetry_grpcfb.Config{})
	require.Error(t, err)
	assert.Nil(t, n)
	assert.ErrorIs(t, err, telemetry_grpcfb.ErrEmptyTarget)
}

func TestNilClientIsSafe(t *testing.T) {
	ctx := testContext(t)

	n := &watchdog_grpcfb.Notifier{}

	assert.NotPanics(t, func() { n.Start(ctx) })
	assert.NotPanics(t, func() {
		require.NoError(t, n.Notify(ctx, monitoring_domain.WatchdogEvent{
			EventType: monitoring_domain.WatchdogEventHeapThresholdExceeded,
			Message:   "ignored",
			Fields:    map[string]string{"k": "v"},
		}))
	})
	assert.NotPanics(t, func() {
		require.NoError(t, n.Upload(ctx, "heap", []byte("ignored"), map[string]string{"reason": "x"}))
	})
	assert.NotPanics(t, func() { require.NoError(t, n.Flush(ctx)) })
	assert.NotPanics(t, func() { require.NoError(t, n.Close(ctx)) })
}

func TestNilClientUploadStillRejectsEmptyType(t *testing.T) {
	ctx := testContext(t)
	n := &watchdog_grpcfb.Notifier{}

	err := n.Upload(ctx, "", nil, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, watchdog_grpcfb.ErrEmptyProfileType)
}

func TestNilNotifierReceiverIsSafe(t *testing.T) {
	ctx := testContext(t)

	var n *watchdog_grpcfb.Notifier

	assert.NotPanics(t, func() { n.Start(ctx) })
	assert.NotPanics(t, func() { require.NoError(t, n.Notify(ctx, monitoring_domain.WatchdogEvent{})) })
	assert.NotPanics(t, func() { require.NoError(t, n.Upload(ctx, "heap", nil, nil)) })
	assert.NotPanics(t, func() { require.NoError(t, n.Flush(ctx)) })
	assert.NotPanics(t, func() { require.NoError(t, n.Close(ctx)) })
}
