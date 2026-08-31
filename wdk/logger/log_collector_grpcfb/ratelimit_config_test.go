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
	"log/slog"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

func limited(t *testing.T, perSecond, burst float64) *Handler {
	t.Helper()

	client := telemetry_grpcfb.New(nil, telemetry_grpcfb.Config{SiteID: "s", FlushInterval: time.Hour})

	return New(client).WithRateLimit(perSecond, burst)
}

func TestDropped_CountsWhatTheBucketShed(t *testing.T) {
	h := limited(t, 0.0001, 3)
	require.Zero(t, h.Dropped(), "nothing shed before anything is logged")

	ctx := context.Background()
	for range 10 {
		rec := slog.NewRecord(time.Unix(0, 0), slog.LevelInfo, "chatter", 0)
		require.NoError(t, h.Handle(ctx, rec))
	}

	assert.Equal(t, int64(7), h.Dropped())
}

func TestWithRateLimit_ZeroDisablesShedding(t *testing.T) {
	h := limited(t, 0, 0)
	ctx := context.Background()
	for range 50 {
		rec := slog.NewRecord(time.Unix(0, 0), slog.LevelInfo, "chatter", 0)
		require.NoError(t, h.Handle(ctx, rec))
	}
	assert.Zero(t, h.Dropped(), "an unlimited handler sheds nothing")
}

func TestWithRateLimit_RaisesTheCeiling(t *testing.T) {
	tight := limited(t, 1, 5)
	raised := limited(t, 10_000, 10_000)
	ctx := context.Background()

	send := func(h *Handler, n int) {
		for range n {
			rec := slog.NewRecord(time.Unix(0, 0), slog.LevelInfo, "chatter", 0)
			require.NoError(t, h.Handle(ctx, rec))
		}
	}
	send(tight, 100)
	send(raised, 100)

	assert.Positive(t, tight.Dropped())
	assert.Zero(t, raised.Dropped(), "a raised ceiling forwards what a Trace-level app produces")
}

func TestWithRateLimit_BurstIsClampedUpToTheRate(t *testing.T) {

	h := limited(t, 50, 1)
	ctx := context.Background()
	for range 50 {
		rec := slog.NewRecord(time.Unix(0, 0), slog.LevelInfo, "chatter", 0)
		require.NoError(t, h.Handle(ctx, rec))
	}
	assert.Zero(t, h.Dropped())
}

func TestRateLimit_ErrorsAreNeverShed(t *testing.T) {
	h := limited(t, 0.0001, 1)
	ctx := context.Background()
	for range 20 {
		rec := slog.NewRecord(time.Unix(0, 0), slog.LevelError, "boom", 0)
		require.NoError(t, h.Handle(ctx, rec))
	}
	assert.Zero(t, h.Dropped())
}

func TestDropped_IsSharedAcrossDerivedHandlers(t *testing.T) {
	root := limited(t, 0.0001, 2)
	child, ok := root.WithAttrs([]slog.Attr{slog.String("k", "v")}).(*Handler)
	require.True(t, ok)

	ctx := context.Background()
	for range 6 {
		rec := slog.NewRecord(time.Unix(0, 0), slog.LevelInfo, "chatter", 0)
		require.NoError(t, child.Handle(ctx, rec))
	}

	assert.Equal(t, root.Dropped(), child.Dropped())
	assert.Positive(t, root.Dropped(), "the parent sees what the child shed")
}

func TestRateLimiter_NilShedsNothing(t *testing.T) {
	var l *rateLimiter
	assert.True(t, first(l.allow()))
	assert.Zero(t, l.droppedCount())
}

func TestRateLimiter_RefillsOverTime(t *testing.T) {
	mock := clock.NewMockClock(time.Unix(1_700_000_000, 0))
	l := newRateLimiter(10, 2, mock)

	assert.True(t, first(l.allow()))
	assert.True(t, first(l.allow()))
	assert.False(t, first(l.allow()), "burst exhausted")
	assert.Equal(t, int64(1), l.droppedCount())

	mock.Advance(time.Second)
	assert.True(t, first(l.allow()), "a second at 10/s refills the bucket")
	assert.Equal(t, int64(1), l.droppedCount(), "a successful allow does not change the count")
}

func TestWithRateLimit_RejectsANonFiniteRate(t *testing.T) {
	cases := map[string][2]float64{
		"NaN rate":          {math.NaN(), 400},
		"NaN burst":         {200, math.NaN()},
		"NaN both":          {math.NaN(), math.NaN()},
		"infinite rate":     {math.Inf(1), 400},
		"infinite burst":    {200, math.Inf(1)},
		"infinite both":     {math.Inf(1), math.Inf(1)},
		"negative infinite": {math.Inf(-1), 400},
	}

	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			handler := limited(t, args[0], args[1])
			ctx := context.Background()

			for range 50 {
				record := slog.NewRecord(time.Unix(0, 0), slog.LevelInfo, "chatter", 0)
				require.NoError(t, handler.Handle(ctx, record))
			}

			assert.Zero(t, handler.Dropped(),
				"an unusable rate disables limiting rather than shedding everything")
		})
	}
}

func TestWithClock_DrivesTheLimiter(t *testing.T) {

	mockClock := clock.NewMockClock(time.Unix(1_700_000_000, 0))
	handler := limited(t, 1, 2).WithClock(mockClock)
	ctx := context.Background()

	send := func() {
		record := slog.NewRecord(time.Unix(0, 0), slog.LevelInfo, "chatter", 0)
		require.NoError(t, handler.Handle(ctx, record))
	}

	send()
	send()
	send()
	require.Equal(t, int64(1), handler.Dropped(), "the burst of two is spent")

	mockClock.Advance(time.Second)
	send()
	assert.Equal(t, int64(1), handler.Dropped(), "a second at one a second refills a token")
}

func first(allowed, _ bool) bool {
	return allowed
}
