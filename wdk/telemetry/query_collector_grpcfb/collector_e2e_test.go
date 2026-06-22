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

package query_collector_grpcfb_test

import (
	"context"
	"errors"
	"io"
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

	"piko.sh/piko"
	"piko.sh/piko/wdk/telemetry/query_collector_grpcfb"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

const (
	maxStatementLen = 4096
	maxErrorLen     = 512
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

func (s *sink) queryStats() []telemetry_grpcfb.QueryStat {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []telemetry_grpcfb.QueryStat
	for _, b := range s.batches {
		out = append(out, b.QueryStats...)
	}
	return out
}

type harness struct {
	snk    *sink
	client *telemetry_grpcfb.Client
	col    *query_collector_grpcfb.Collector
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

	client.Start(context.Background())

	return &harness{
		snk:    snk,
		client: client,
		col:    query_collector_grpcfb.New(client),
	}
}

func (h *harness) drain(ctx context.Context, t *testing.T) {
	t.Helper()
	require.NoError(t, h.client.Close(ctx))
}

func TestObserveQueryStreamsRedactedQueryStat(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	h.col.ObserveQuery(ctx, &piko.QueryObservation{
		Connection: "registry",
		Operation:  "SELECT",
		Statement:  "SELECT * FROM t WHERE name='alice' AND id=42",
		DurationMs: 7,
		Rows:       3,
	})
	h.drain(ctx, t)

	stats := h.snk.queryStats()
	require.Len(t, stats, 1)
	got := stats[0]

	assert.Equal(t, "registry", got.Connection)
	assert.Equal(t, "SELECT", got.Operation)
	assert.Equal(t, "SELECT * FROM t WHERE name='?' AND id=?", got.Statement)
	assert.NotContains(t, got.Statement, "alice")
	assert.NotContains(t, got.Statement, "42")
	assert.Equal(t, "ok", got.Status)
	assert.Empty(t, got.Error)
	assert.Equal(t, int64(7), got.DurationMs)
	assert.Equal(t, int64(3), got.Rows)
	assert.Equal(t, int64(1), got.Calls)
	assert.Positive(t, got.TsMs)
}

func TestObserveQueryStreamsRedactedErrorOnFailure(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	h.col.ObserveQuery(ctx, &piko.QueryObservation{
		Connection: "orchestrator",
		Operation:  "INSERT",
		Statement:  "INSERT INTO u (email) VALUES ('bob@example.com')",
		Err:        errors.New("duplicate key 'bob@example.com' violates constraint 13"),
	})
	h.drain(ctx, t)

	stats := h.snk.queryStats()
	require.Len(t, stats, 1)
	got := stats[0]

	assert.Equal(t, "error", got.Status)
	assert.NotEmpty(t, got.Error)
	assert.NotContains(t, got.Error, "bob@example.com")
	assert.NotContains(t, got.Error, "13")
	assert.NotContains(t, got.Statement, "bob@example.com")
	assert.Equal(t, int64(1), got.Calls)
}

func TestObserveQueryTruncatesOverLongStatement(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stmt := "SELECT " + strings.Repeat("é", maxStatementLen)
	h.col.ObserveQuery(ctx, &piko.QueryObservation{Statement: stmt})
	h.drain(ctx, t)

	stats := h.snk.queryStats()
	require.Len(t, stats, 1)
	got := stats[0]

	assert.LessOrEqual(t, len(got.Statement), maxStatementLen)
	assert.Less(t, len(got.Statement), len(stmt))
	assert.True(t, utf8.ValidString(got.Statement), "truncated statement must stay valid UTF-8")
}

func TestObserveQueryTruncatesOverLongError(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errMsg := "boom " + strings.Repeat("é", maxErrorLen)
	h.col.ObserveQuery(ctx, &piko.QueryObservation{
		Statement: "SELECT 1",
		Err:       errors.New(errMsg),
	})
	h.drain(ctx, t)

	stats := h.snk.queryStats()
	require.Len(t, stats, 1)
	got := stats[0]

	assert.Equal(t, "error", got.Status)
	assert.LessOrEqual(t, len(got.Error), maxErrorLen)
	assert.Less(t, len(got.Error), len(errMsg))
	assert.True(t, utf8.ValidString(got.Error), "truncated error must stay valid UTF-8")
}

func TestObserveQueryMarksTruncatedStatement(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stmt := "SELECT " + strings.Repeat("a", maxStatementLen)
	h.col.ObserveQuery(ctx, &piko.QueryObservation{Statement: stmt})
	h.drain(ctx, t)

	stats := h.snk.queryStats()
	require.Len(t, stats, 1)
	assert.Contains(t, stats[0].Attrs, telemetry_grpcfb.KV{Key: "truncated", Value: "statement"})
}

func TestObserveQueryMarksTruncatedError(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	h.col.ObserveQuery(ctx, &piko.QueryObservation{
		Statement: "SELECT 1",
		Err:       errors.New("boom " + strings.Repeat("a", maxErrorLen)),
	})
	h.drain(ctx, t)

	stats := h.snk.queryStats()
	require.Len(t, stats, 1)
	assert.Contains(t, stats[0].Attrs, telemetry_grpcfb.KV{Key: "truncated", Value: "error"})
}

func TestObserveQueryShortInputCarriesNoTruncationMarker(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	h.col.ObserveQuery(ctx, &piko.QueryObservation{Statement: "SELECT 1"})
	h.drain(ctx, t)

	stats := h.snk.queryStats()
	require.Len(t, stats, 1)
	assert.Empty(t, stats[0].Attrs)
}

func TestObserveQueryRecoversFromPanickingError(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assert.NotPanics(t, func() {
		h.col.ObserveQuery(ctx, &piko.QueryObservation{
			Statement: "SELECT 1",
			Err:       panickingError{},
		})
	})
	h.drain(ctx, t)

	assert.Empty(t, h.snk.queryStats())
}

type panickingError struct{}

func (panickingError) Error() string {
	panic("boom")
}

func TestObserveQueryNilObservationStreamsNothing(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assert.NotPanics(t, func() {
		h.col.ObserveQuery(ctx, nil)
	})
	h.drain(ctx, t)

	assert.Empty(t, h.snk.queryStats())
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
