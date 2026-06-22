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
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type captureServer struct {
	batches []*Batch
	mu      sync.Mutex
}

func (s *captureServer) Ingest(stream IngestStream) error {
	var frames, events int64
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
		frames++
		events += int64(b.EventCount())
	}
	return stream.SendAndClose(&IngestAck{OK: true, Frames: frames, Events: events})
}

func (s *captureServer) snapshot() []*Batch {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Batch, len(s.batches))
	copy(out, s.batches)
	return out
}

func dialBufconn(t *testing.T, lis *bufconn.Listener) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(Codec{})),
	)
	require.NoError(t, err)
	return conn
}

func TestStreamRoundTrip(t *testing.T) {
	lis := bufconn.Listen(1 << 20)
	srv := NewServer()
	recorder := &captureServer{}
	RegisterServer(srv, recorder)
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn := dialBufconn(t, lis)
	defer conn.Close()

	client := New(conn, Config{
		SiteID:        "site-1",
		APIKey:        "key-1",
		Source:        "host-a",
		FlushSize:     2,
		FlushInterval: time.Hour,
	})
	client.Start(context.Background())

	client.AddAnalytics(context.Background(), AnalyticsEvent{Kind: "pageview", Path: "/", StatusCode: 200, Properties: []KV{{Key: "k", Value: "v"}}})
	client.AddAnalytics(context.Background(), AnalyticsEvent{Kind: "action", ActionName: "save"})
	client.AddWatchdog(context.Background(), WatchdogEvent{EventType: "heap_threshold_exceeded", Priority: 2, Message: "heap high", Fields: []KV{{Key: "bytes", Value: "1024"}}})
	client.AddError(context.Background(), ErrorEvent{Fingerprint: "fp1", Type: "panic", Value: "boom"})
	client.AddProfile(context.Background(), ProfileMeta{ProfileType: "heap", SizeBytes: 4, Blob: []byte("PROF")})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.Close(ctx))

	batches := recorder.snapshot()
	require.Len(t, batches, 3)

	var nAnalytics, nWatchdog, nErrors, nProfiles int
	var seqs []int64
	for _, b := range batches {
		assert.Equal(t, "site-1", b.SiteID)
		assert.Equal(t, "key-1", b.APIKey)
		assert.Equal(t, "host-a", b.Source)
		assert.NotZero(t, b.SentAtMs, "batch missing sent_at")
		seqs = append(seqs, b.Seq)
		nAnalytics += len(b.Analytics)
		nWatchdog += len(b.Watchdog)
		nErrors += len(b.Errors)
		nProfiles += len(b.Profiles)
	}
	assert.Equal(t, 2, nAnalytics, "analytics count")
	assert.Equal(t, 1, nWatchdog, "watchdog count")
	assert.Equal(t, 1, nErrors, "errors count")
	assert.Equal(t, 1, nProfiles, "profiles count")

	for i, s := range seqs {
		assert.Equal(t, int64(i+1), s, "seq[%d]", i)
	}

	var foundProps, foundBlob bool
	for _, b := range batches {
		for _, a := range b.Analytics {
			if len(a.Properties) == 1 && a.Properties[0].Key == "k" && a.Properties[0].Value == "v" {
				foundProps = true
			}
		}
		for _, p := range b.Profiles {
			if string(p.Blob) == "PROF" {
				foundBlob = true
			}
		}
	}
	assert.True(t, foundProps, "analytics properties not preserved over the stream")
	assert.True(t, foundBlob, "inline profile blob not preserved over the stream")

	assert.Equal(t, int64(5), client.Sent())
	assert.Equal(t, int64(0), client.Dropped())
}

func TestStreamServerRejectsMalformedFrame(t *testing.T) {
	lis := bufconn.Listen(1 << 20)
	srv := NewServer()
	RegisterServer(srv, &captureServer{})
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn := dialBufconn(t, lis)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := conn.NewStream(ctx, &serviceDesc.Streams[0], ingestFullMethod, grpc.ForceCodec(Codec{}))
	require.NoError(t, err)

	if err := stream.SendMsg(rawFrame{0xff, 0xff, 0xff, 0xff}); err != nil {

		t.Logf("SendMsg returned: %v", err)
	}
	_ = stream.CloseSend()
	var ack IngestAck
	assert.Error(t, stream.RecvMsg(&ack), "expected the malformed frame to fail the RPC")

	client := New(conn, Config{SiteID: "s", FlushSize: 1, FlushInterval: time.Hour})
	client.Start(ctx)
	client.AddAnalytics(context.Background(), AnalyticsEvent{Kind: "pageview", Path: "/ok"})
	require.NoError(t, client.Close(ctx))
	assert.Equal(t, int64(1), client.Sent(), "server did not survive the malformed frame")
}

type rawFrame []byte

func (r rawFrame) Marshal() ([]byte, error) { return r, nil }
func (r rawFrame) Unmarshal(data []byte) error {
	return verifyMessage(data, batchFields)
}

type panicServer struct {
	delegate captureServer
	panicked bool
	mu       sync.Mutex
}

func (s *panicServer) Ingest(stream IngestStream) error {
	s.mu.Lock()
	first := !s.panicked
	s.panicked = true
	s.mu.Unlock()
	if first {
		panic("ingest boom")
	}
	return s.delegate.Ingest(stream)
}

func TestServerRecoversFromHandlerPanic(t *testing.T) {
	lis := bufconn.Listen(1 << 20)
	srv := NewServer()
	RegisterServer(srv, &panicServer{})
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn := dialBufconn(t, lis)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := conn.NewStream(ctx, &serviceDesc.Streams[0], ingestFullMethod, grpc.ForceCodec(Codec{}))
	require.NoError(t, err)
	_ = stream.SendMsg(&Batch{SiteID: "s", Analytics: []AnalyticsEvent{{Kind: "pageview"}}})
	_ = stream.CloseSend()
	var ack IngestAck
	err = stream.RecvMsg(&ack)
	require.Error(t, err, "panicking handler must fail the RPC")
	assert.Equal(t, codes.Internal, status.Code(err), "panic must surface as codes.Internal")

	client := New(conn, Config{SiteID: "s", FlushSize: 1, FlushInterval: time.Hour})
	client.Start(ctx)
	client.AddAnalytics(context.Background(), AnalyticsEvent{Kind: "pageview", Path: "/ok"})
	require.NoError(t, client.Close(ctx))
	assert.Equal(t, int64(1), client.Sent(), "server did not survive the handler panic")
}

func TestServerMaxReceiveMessageSizeRejectsOversized(t *testing.T) {
	const limit = 1024
	lis := bufconn.Listen(1 << 20)
	srv := NewServer(WithMaxReceiveMessageSize(limit))
	RegisterServer(srv, &captureServer{})
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn := dialBufconn(t, lis)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := conn.NewStream(ctx, &serviceDesc.Streams[0], ingestFullMethod, grpc.ForceCodec(Codec{}))
	require.NoError(t, err)

	big := &Batch{SiteID: "s", Logs: []LogLine{{Message: string(make([]byte, 4096))}}}
	_ = stream.SendMsg(big)
	_ = stream.CloseSend()
	var ack IngestAck
	err = stream.RecvMsg(&ack)
	require.Error(t, err, "oversized frame must be rejected")
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
}
