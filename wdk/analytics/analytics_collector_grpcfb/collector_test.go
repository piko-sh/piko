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

package analytics_collector_grpcfb_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	"piko.sh/piko/wdk/analytics"
	"piko.sh/piko/wdk/analytics/analytics_collector_grpcfb"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
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

func (s *sink) events() []telemetry_grpcfb.AnalyticsEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []telemetry_grpcfb.AnalyticsEvent
	for _, b := range s.batches {
		out = append(out, b.Analytics...)
	}
	return out
}

type harness struct {
	snk    *sink
	client *telemetry_grpcfb.Client
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	lis := bufconn.Listen(1 << 20)
	srv := telemetry_grpcfb.NewServer()
	snk := &sink{}
	telemetry_grpcfb.RegisterServer(srv, snk)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn := dial(t, lis)
	t.Cleanup(func() { _ = conn.Close() })

	client := telemetry_grpcfb.New(conn, telemetry_grpcfb.Config{
		SiteID:        "site-x",
		APIKey:        "key-x",
		FlushSize:     1,
		FlushInterval: time.Hour,
	})
	return &harness{snk: snk, client: client}
}

func hashPII(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func streamOne(t *testing.T, h *harness, col *analytics_collector_grpcfb.Collector, e *analytics.Event) telemetry_grpcfb.AnalyticsEvent {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	col.Start(ctx)

	require.NoError(t, col.Collect(ctx, e))

	require.NoError(t, col.Close(ctx))
	require.NoError(t, h.client.Close(ctx))

	got := h.snk.events()
	require.Len(t, got, 1)
	return got[0]
}

func TestCollectorStreamsAnalytics(t *testing.T) {
	h := newHarness(t)
	col := analytics_collector_grpcfb.New(h.client, analytics_collector_grpcfb.WithUserID())

	assert.Equal(t, "grpcfb", col.Name())

	got := streamOne(t, h, col, &analytics.Event{
		Type:       analytics.EventPageView,
		Timestamp:  time.Unix(1_700_000_000, 0),
		Hostname:   "example.com",
		Path:       "/blog",
		Method:     "GET",
		StatusCode: 200,
		Duration:   42 * time.Millisecond,
		UserID:     "u-9",
		Properties: map[string]string{"plan": "pro"},
	})

	h.snk.mu.Lock()
	require.NotEmpty(t, h.snk.batches)
	for _, b := range h.snk.batches {
		assert.Equal(t, "site-x", b.SiteID)
		assert.Equal(t, "key-x", b.APIKey)
	}
	h.snk.mu.Unlock()

	assert.Equal(t, "pageview", got.Kind)
	assert.Equal(t, "/blog", got.Path)
	assert.Equal(t, "GET", got.Method)
	assert.Equal(t, int32(200), got.StatusCode)
	assert.Equal(t, int64(42), got.DurationMs)
	assert.Equal(t, "u-9", got.UserID)
	assert.Equal(t, "example.com", got.Hostname)
	require.Len(t, got.Properties, 1)
	assert.Equal(t, "plan", got.Properties[0].Key)
	assert.Equal(t, "pro", got.Properties[0].Value)
	assert.Equal(t, time.Unix(1_700_000_000, 0).UnixMilli(), got.TimestampMs)
}

func TestCollectorAnonymisesByDefault(t *testing.T) {
	h := newHarness(t)
	col := analytics_collector_grpcfb.New(h.client)

	const (
		rawIP        = "203.0.113.7"
		rawUserAgent = "Mozilla/5.0 (X11; Linux x86_64) Firefox/123.0"
	)

	got := streamOne(t, h, col, &analytics.Event{
		Type:      analytics.EventPageView,
		Timestamp: time.Unix(1_700_000_000, 0),
		URL:       "https://example.com/reset?token=secret&next=/home",
		Referrer:  "https://search.example/results?q=private+search#frag",
		ClientIP:  rawIP,
		UserAgent: rawUserAgent,
		UserID:    "u-9",
	})

	assert.Equal(t, hashPII(rawIP), got.ClientIP)
	assert.Len(t, got.ClientIP, 64)
	assert.NotEqual(t, rawIP, got.ClientIP)
	assertHexLower(t, got.ClientIP)

	assert.Equal(t, hashPII(rawUserAgent), got.UserAgent)
	assert.Len(t, got.UserAgent, 64)
	assert.NotEqual(t, rawUserAgent, got.UserAgent)

	assert.Equal(t, "https://example.com/reset", got.URL)
	assert.Equal(t, "https://search.example/results", got.Referrer)
	assert.NotContains(t, got.URL, "secret")
	assert.NotContains(t, got.Referrer, "private")

	assert.Empty(t, got.UserID)
}

func TestCollectorRawWithOptions(t *testing.T) {
	h := newHarness(t)
	col := analytics_collector_grpcfb.New(h.client,
		analytics_collector_grpcfb.WithRawClientIP(),
		analytics_collector_grpcfb.WithRawUserAgent(),
		analytics_collector_grpcfb.WithURLQuery(),
		analytics_collector_grpcfb.WithUserID(),
	)

	const (
		rawIP        = "203.0.113.7"
		rawUserAgent = "Mozilla/5.0 (X11; Linux x86_64) Firefox/123.0"
		rawURL       = "https://example.com/reset?token=secret&next=/home"
		rawReferrer  = "https://search.example/results?q=private+search#frag"
		rawUserID    = "u-9"
	)

	got := streamOne(t, h, col, &analytics.Event{
		Type:      analytics.EventPageView,
		Timestamp: time.Unix(1_700_000_000, 0),
		URL:       rawURL,
		Referrer:  rawReferrer,
		ClientIP:  rawIP,
		UserAgent: rawUserAgent,
		UserID:    rawUserID,
	})

	assert.Equal(t, rawIP, got.ClientIP)
	assert.Equal(t, rawUserAgent, got.UserAgent)
	assert.Equal(t, rawURL, got.URL)
	assert.Equal(t, rawReferrer, got.Referrer)
	assert.Equal(t, rawUserID, got.UserID)
}

func TestCollectorCapsProperties(t *testing.T) {
	const (
		maxProperties       = 128
		maxPropertyMarkers  = 3
		maxPropertyValueLen = 1024
	)

	t.Run("count capped at maxProperties", func(t *testing.T) {
		h := newHarness(t)
		col := analytics_collector_grpcfb.New(h.client)

		props := make(map[string]string, maxProperties*2)
		for i := range maxProperties * 2 {
			props["k"+strconv.Itoa(i)] = "v"
		}

		got := streamOne(t, h, col, &analytics.Event{
			Type:       analytics.EventCustom,
			Timestamp:  time.Unix(1_700_000_000, 0),
			Properties: props,
		})

		const emitted = maxProperties - maxPropertyMarkers

		assert.LessOrEqual(t, len(got.Properties), maxProperties)
		assert.Equal(t, strconv.Itoa(len(props)-emitted),
			propertyValue(got, "client.properties_dropped"),
			"the count that did not fit is reported, not silently lost")
	})

	t.Run("oversized value truncated to valid UTF-8", func(t *testing.T) {
		h := newHarness(t)
		col := analytics_collector_grpcfb.New(h.client)

		const star = "⭐"
		oversized := strings.Repeat(star, (maxPropertyValueLen/len(star))+50)
		require.Greater(t, len(oversized), maxPropertyValueLen)

		got := streamOne(t, h, col, &analytics.Event{
			Type:       analytics.EventCustom,
			Timestamp:  time.Unix(1_700_000_000, 0),
			Properties: map[string]string{"big": oversized},
		})

		require.Len(t, got.Properties, 2, "the value plus its truncation marker")
		assert.Equal(t, "1", propertyValue(got, "client.properties_truncated"))
		value := propertyValue(got, "big")
		assert.LessOrEqual(t, len(value), maxPropertyValueLen)
		assert.True(t, utf8.ValidString(value), "truncated value must stay valid UTF-8")

		assert.Zero(t, len(value)%len(star))
	})
}

func TestCollectorNilEventIsNoOp(t *testing.T) {
	h := newHarness(t)
	col := analytics_collector_grpcfb.New(h.client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	col.Start(ctx)

	require.NoError(t, col.Collect(ctx, nil))

	require.NoError(t, col.Close(ctx))
	require.NoError(t, h.client.Close(ctx))

	assert.Empty(t, h.snk.events())
}

func TestCollectorName(t *testing.T) {
	col := analytics_collector_grpcfb.New(nil)
	assert.Equal(t, "grpcfb", col.Name())
}

func TestCollectorFlush(t *testing.T) {
	h := newHarness(t)
	col := analytics_collector_grpcfb.New(h.client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	col.Start(ctx)

	require.NoError(t, col.Collect(ctx, &analytics.Event{
		Type:      analytics.EventPageView,
		Timestamp: time.Unix(1_700_000_000, 0),
		Path:      "/x",
	}))
	require.NoError(t, col.Flush(ctx))

	require.NoError(t, h.client.Close(ctx))
	assert.Len(t, h.snk.events(), 1)

	var nilCol *analytics_collector_grpcfb.Collector
	require.NoError(t, nilCol.Flush(ctx))
}

func TestCollectorCloseShared(t *testing.T) {
	h := newHarness(t)
	col := analytics_collector_grpcfb.New(h.client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	col.Start(ctx)

	require.NoError(t, col.Collect(ctx, &analytics.Event{
		Type:      analytics.EventPageView,
		Timestamp: time.Unix(1_700_000_000, 0),
		Path:      "/shared",
	}))

	require.NoError(t, col.Close(ctx))
	require.NoError(t, h.client.Close(ctx))

	got := h.snk.events()
	require.Len(t, got, 1)
	assert.Equal(t, "/shared", got[0].Path)
}

func TestCollectorCloseOwned(t *testing.T) {
	lis := bufconn.Listen(1 << 20)
	srv := telemetry_grpcfb.NewServer()
	snk := &sink{}
	telemetry_grpcfb.RegisterServer(srv, snk)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	col, err := analytics_collector_grpcfb.Dial("passthrough:///bufnet",
		telemetry_grpcfb.Config{
			SiteID:        "site-x",
			APIKey:        "key-x",
			FlushSize:     1,
			FlushInterval: time.Hour,
		},
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(telemetry_grpcfb.Codec{})),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	col.Start(ctx)

	require.NoError(t, col.Collect(ctx, &analytics.Event{
		Type:      analytics.EventPageView,
		Timestamp: time.Unix(1_700_000_000, 0),
		Path:      "/owned",
	}))

	require.NoError(t, col.Close(ctx))

	got := snk.events()
	require.Len(t, got, 1)
	assert.Equal(t, "/owned", got[0].Path)
}

func TestCollectorNilSafety(t *testing.T) {
	ctx := context.Background()

	t.Run("nil collector", func(t *testing.T) {
		var col *analytics_collector_grpcfb.Collector
		assert.NotPanics(t, func() { col.Start(ctx) })
		require.NoError(t, col.Collect(ctx, &analytics.Event{Type: analytics.EventPageView}))
		require.NoError(t, col.Flush(ctx))
		require.NoError(t, col.Close(ctx))
	})

	t.Run("nil client", func(t *testing.T) {
		col := analytics_collector_grpcfb.New(nil)
		assert.NotPanics(t, func() { col.Start(ctx) })
		require.NoError(t, col.Collect(ctx, &analytics.Event{Type: analytics.EventPageView}))
		require.NoError(t, col.Flush(ctx))
		require.NoError(t, col.Close(ctx))
	})
}

func assertHexLower(t *testing.T, s string) {
	t.Helper()
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			assert.Failf(t, "non-hex character", "string %q contains non-hex rune %q", s, r)
			return
		}
	}
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

func propertyValue(event telemetry_grpcfb.AnalyticsEvent, key string) string {
	for _, pair := range event.Properties {
		if pair.Key == key {
			return pair.Value
		}
	}
	return ""
}
