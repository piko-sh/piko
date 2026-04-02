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

package monitoring_transport_grpc

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/clock"
)

func TestRunWatchLoop_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	err := runWatchLoop(ctx, 100, func() error {
		return nil
	}, "test", nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRunWatchLoop_SendUpdateError(t *testing.T) {
	t.Parallel()

	mock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	baseline := mock.TimerCount()

	sendErr := errors.New("stream broken")
	errCh := make(chan error, 1)
	go func() {
		errCh <- runWatchLoop(context.Background(), 100, func() error {
			return sendErr
		}, "metric", mock)
	}()

	require.True(t, mock.AwaitTimerSetup(baseline, time.Second))
	mock.Advance(100 * time.Millisecond)

	err := <-errCh
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sending metric update")
	assert.ErrorIs(t, err, sendErr)
}

func TestRunWatchLoop_MinIntervalEnforcement(t *testing.T) {
	t.Parallel()

	mock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	baseline := mock.TimerCount()

	ctx, cancel := context.WithCancelCause(context.Background())
	callCount := 0
	done := make(chan struct{})

	go func() {
		_ = runWatchLoop(ctx, 10, func() error {
			callCount++
			return nil
		}, "test", mock)
		close(done)
	}()

	require.True(t, mock.AwaitTimerSetup(baseline, time.Second))

	mock.Advance(200 * time.Millisecond)

	cancel(fmt.Errorf("test: cleanup"))
	<-done

	assert.Equal(t, 0, callCount)
}

func TestRunWatchLoop_ValidInterval(t *testing.T) {
	t.Parallel()

	mock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	baseline := mock.TimerCount()

	ctx, cancel := context.WithCancelCause(context.Background())
	called := make(chan struct{}, 10)
	done := make(chan struct{})

	go func() {
		_ = runWatchLoop(ctx, 150, func() error {
			called <- struct{}{}
			return nil
		}, "test", mock)
		close(done)
	}()

	require.True(t, mock.AwaitTimerSetup(baseline, time.Second))

	for range 3 {
		mock.Advance(150 * time.Millisecond)
		select {
		case <-called:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for update callback")
		}
	}

	cancel(fmt.Errorf("test: cleanup"))
	<-done

	assert.Len(t, called, 0, "all callbacks should have been consumed")
}
