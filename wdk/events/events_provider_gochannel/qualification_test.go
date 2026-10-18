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

package events_provider_gochannel

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"
)

func TestPublicationSoak(t *testing.T) {
	durationText := os.Getenv("PIKO_PUBLICATION_SOAK")
	if durationText == "" {
		t.Skip("set PIKO_PUBLICATION_SOAK to run sustained qualification")
	}
	duration, err := time.ParseDuration(durationText)
	require.NoError(t, err)
	cfg := DefaultConfig()
	cfg.PublishQueueCapacity = 64
	cfg.OutputChannelBuffer = 1
	p, err := NewGoChannelProvider(cfg)
	require.NoError(t, err)
	defer p.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	sub, err := p.Subscriber().Subscribe(ctx, "slow")
	require.NoError(t, err)
	msg := message.NewMessage("test", make([]byte, 1024))
	for range cfg.PublishQueueCapacity {
		require.NoError(t, p.Publisher().Publish("slow", msg))
	}
	time.Sleep(10 * time.Millisecond)
	runtime.GC()
	baseline := runtime.NumGoroutine()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	baselineHeap := mem.HeapAlloc
	finish := time.Now().Add(duration)
	report := time.Now()
	attempts := 0
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for time.Now().Before(finish) {
		<-ticker.C
		for range 100 {
			require.ErrorIs(t, p.Publisher().Publish("slow", msg), ErrPublishOverloaded)
			attempts++
		}
		stats := p.PublicationStats()
		require.Equal(t, 64, stats.Messages)
		require.Equal(t, 1, stats.Topics)
		if time.Now().After(report) {
			runtime.GC()
			runtime.ReadMemStats(&mem)
			require.LessOrEqual(t, runtime.NumGoroutine(), baseline+8)
			require.LessOrEqual(t, mem.HeapAlloc, baselineHeap+(8<<20))
			t.Logf("attempts=%d messages=%d bytes=%d topics=%d goroutines=%d heap=%d", attempts, stats.Messages, stats.Bytes, stats.Topics, runtime.NumGoroutine(), mem.HeapAlloc)
			report = time.Now().Add(time.Minute)
		}
	}
	for range 64 {
		select {
		case m := <-sub:
			m.Ack()
		case <-time.After(time.Second):
			t.Fatal("drain stalled")
		}
	}
	require.Eventually(t, func() bool { return p.PublicationStats().Messages == 0 }, time.Second, time.Millisecond)
	require.NoError(t, p.Close())
	require.Zero(t, p.PublicationStats().Bytes)
	require.Zero(t, p.PublicationStats().Topics)
	t.Logf("qualification completed after %s, %d rejected publications, drained all accepted messages", duration, attempts)
}
