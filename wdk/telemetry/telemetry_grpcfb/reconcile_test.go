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

package telemetry_grpcfb

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type rejectingServer struct{}

func (rejectingServer) Ingest(stream IngestStream) error {
	for {
		_, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
	}
	return status.Error(codes.Unauthenticated, "bad delivery key")
}

type shortAckServer struct {
	accept int64
}

func (s shortAckServer) Ingest(stream IngestStream) error {
	var frames int64
	for {
		_, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		frames++
	}
	return stream.SendAndClose(&IngestAck{OK: true, Frames: frames, Events: s.accept})
}

type notOKServer struct{}

func (notOKServer) Ingest(stream IngestStream) error {
	for {
		_, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
	}
	return stream.SendAndClose(&IngestAck{OK: false, Message: "quota exceeded"})
}

func startServer(t *testing.T, srvImpl Server) *bufconn.Listener {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := NewServer()
	RegisterServer(srv, srvImpl)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	return lis
}

func TestReconcileNormalAck(t *testing.T) {
	lis := startServer(t, &captureServer{})
	conn := dialBufconn(t, lis)
	defer conn.Close()

	client := New(conn, Config{SiteID: "s", FlushSize: 2, FlushInterval: time.Hour})
	client.Start(context.Background())
	client.AddAnalytics(context.Background(), AnalyticsEvent{Kind: "pageview", Path: "/a"})
	client.AddAnalytics(context.Background(), AnalyticsEvent{Kind: "pageview", Path: "/b"})
	client.AddAnalytics(context.Background(), AnalyticsEvent{Kind: "pageview", Path: "/c"})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.Close(ctx))

	assert.Equal(t, int64(3), client.Sent(), "Sent must equal the acked event count")
	assert.Equal(t, int64(0), client.Dropped())
	assert.Equal(t, int64(0), client.Rejected())
}

func TestReconcileUnauthenticated(t *testing.T) {
	lis := startServer(t, rejectingServer{})
	conn := dialBufconn(t, lis)
	defer conn.Close()

	client := New(conn, Config{
		SiteID:        "s",
		APIKey:        "wrong",
		FlushSize:     2,
		FlushInterval: time.Hour,
	})
	client.Start(context.Background())
	client.AddAnalytics(context.Background(), AnalyticsEvent{Kind: "pageview", Path: "/a"})
	client.AddAnalytics(context.Background(), AnalyticsEvent{Kind: "pageview", Path: "/b"})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.Close(ctx))

	assert.Equal(t, int64(0), client.Sent(), "rejected events must never count as Sent")
	assert.GreaterOrEqual(t, client.Rejected(), int64(2), "rejected events must be surfaced")
	assert.GreaterOrEqual(t, client.Dropped(), int64(2), "rejected events are also dropped")
	assert.Equal(t, client.Rejected(), client.Dropped(), "every rejected event is a dropped event")
}

func TestReconcileShortAck(t *testing.T) {
	lis := startServer(t, shortAckServer{accept: 1})
	conn := dialBufconn(t, lis)
	defer conn.Close()

	client := New(conn, Config{
		SiteID:        "s",
		FlushSize:     3,
		FlushInterval: time.Hour,
	})
	client.Start(context.Background())
	client.AddAnalytics(context.Background(), AnalyticsEvent{Kind: "pageview", Path: "/a"})
	client.AddAnalytics(context.Background(), AnalyticsEvent{Kind: "pageview", Path: "/b"})
	client.AddAnalytics(context.Background(), AnalyticsEvent{Kind: "pageview", Path: "/c"})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.Close(ctx))

	assert.Equal(t, int64(1), client.Sent(), "only acked events count as Sent")
	assert.Equal(t, int64(2), client.Dropped(), "the ack shortfall is dropped")
	assert.Equal(t, int64(0), client.Rejected(), "a short ack is not a rejection")
}

func TestReconcileNotOK(t *testing.T) {
	lis := startServer(t, notOKServer{})
	conn := dialBufconn(t, lis)
	defer conn.Close()

	client := New(conn, Config{
		SiteID:        "s",
		FlushSize:     2,
		FlushInterval: time.Hour,
	})
	client.Start(context.Background())
	client.AddAnalytics(context.Background(), AnalyticsEvent{Kind: "pageview", Path: "/a"})
	client.AddAnalytics(context.Background(), AnalyticsEvent{Kind: "pageview", Path: "/b"})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.Close(ctx))

	assert.Equal(t, int64(0), client.Sent())
	assert.Equal(t, int64(2), client.Rejected())
	assert.Equal(t, int64(2), client.Dropped())
}
