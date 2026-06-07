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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/test/bufconn"
)

func oversizedBatch() *Batch {
	logs := make([]LogLine, 20)
	for i := range logs {
		logs[i] = LogLine{Message: string(make([]byte, maxStringLen))}
	}
	return &Batch{SiteID: "s", Logs: logs}
}

func TestOversizedBatchDroppedNotStreamed(t *testing.T) {
	lis := bufconn.Listen(1 << 20)
	srv := NewServer()
	recorder := &captureServer{}
	RegisterServer(srv, recorder)
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn := dialBufconn(t, lis)
	defer conn.Close()

	client := New(conn, Config{SiteID: "s", FlushInterval: time.Hour})
	client.Start(context.Background())

	client.enqueue(context.Background(), oversizedBatch())
	client.enqueue(context.Background(), &Batch{SiteID: "s", Analytics: []AnalyticsEvent{{Kind: "p"}}})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.Close(ctx))

	assert.Equal(t, int64(1), client.Sent(), "the in-budget batch must still be delivered")
	assert.Equal(t, int64(20), client.Dropped(), "the oversized batch's events must be dropped")
	batches := recorder.snapshot()
	for _, b := range batches {
		assert.Empty(t, b.Logs, "the oversized log batch must never reach the server")
	}
}

func TestEnqueueDropsWhenQueueFull(t *testing.T) {
	client := New(nil, Config{SiteID: "s", MaxQueuedBatches: 1})

	client.enqueue(context.Background(), &Batch{Analytics: []AnalyticsEvent{{Kind: "a"}}})
	client.enqueue(context.Background(), &Batch{Analytics: []AnalyticsEvent{{Kind: "b"}}})
	client.enqueue(context.Background(), &Batch{Analytics: []AnalyticsEvent{{Kind: "c"}}})
	assert.Equal(t, int64(2), client.Dropped(), "two batches should overflow the depth-1 queue")
	assert.True(t, client.dropWarned.Load(), "queue-full drop must warn once")
}
