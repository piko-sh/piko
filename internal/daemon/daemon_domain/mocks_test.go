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

package daemon_domain

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockServerAdapterWithShutdownCh struct {
	*MockServerAdapter
	shutdownCh     chan struct{}
	shutdownClosed int64
}

func newMockServerAdapter() *mockServerAdapterWithShutdownCh {
	shutdownChannel := make(chan struct{})
	m := &mockServerAdapterWithShutdownCh{
		MockServerAdapter: &MockServerAdapter{},
		shutdownCh:        shutdownChannel,
	}
	m.ListenAndServeFunc = func(_ string, _ http.Handler) error {
		return http.ErrServerClosed
	}
	m.ShutdownFunc = func(_ context.Context) error {
		if atomic.CompareAndSwapInt64(&m.shutdownClosed, 0, 1) {
			close(m.shutdownCh)
		}
		return nil
	}
	return m
}

func testDaemonConfig() DaemonConfig {
	return DaemonConfig{
		NetworkPort:         "8080",
		NetworkAutoNextPort: false,
		HealthEnabled:       false,
		HealthPort:          "8081",
		HealthBindAddress:   "127.0.0.1",
		HealthAutoNextPort:  false,
		HealthLivePath:      "/healthz",
		HealthReadyPath:     "/readyz",
	}
}

func testDaemonConfigWithHealthProbe() DaemonConfig {
	config := testDaemonConfig()
	config.HealthEnabled = true
	return config
}

func waitForCondition(timeout time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func TestNewMockSignalNotifier_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	mock := NewMockSignalNotifier()

	require.NotNil(t, mock)
}

func TestMockSignalNotifier_NotifyContext(t *testing.T) {
	t.Parallel()

	t.Run("nil NotifyContextFunc returns cancellable context", func(t *testing.T) {
		t.Parallel()

		mock := NewMockSignalNotifier()
		parent := context.Background()

		ctx, cancel := mock.NotifyContext(parent)
		defer cancel()

		require.NotNil(t, ctx)
		require.NotNil(t, cancel)
		assert.NoError(t, ctx.Err())
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NotifyContextCallCount))
	})

	t.Run("delegates to NotifyContextFunc", func(t *testing.T) {
		t.Parallel()

		var capturedParent context.Context
		customCtx, customCancelCause := context.WithCancelCause(context.Background())
		defer customCancelCause(fmt.Errorf("test: cleanup"))
		customCancel := func() { customCancelCause(fmt.Errorf("test: cleanup")) }

		mock := NewMockSignalNotifier()
		mock.NotifyContextFunc = func(parent context.Context) (context.Context, context.CancelFunc) {
			capturedParent = parent
			return customCtx, customCancel
		}

		parent := context.Background()
		ctx, cancel := mock.NotifyContext(parent)

		assert.Equal(t, parent, capturedParent)
		assert.Equal(t, customCtx, ctx)
		assert.NotNil(t, cancel)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NotifyContextCallCount))
	})
}

func TestMockSignalNotifier_Trigger(t *testing.T) {
	t.Parallel()

	t.Run("cancels context from NotifyContext", func(t *testing.T) {
		t.Parallel()

		mock := NewMockSignalNotifier()
		ctx, cancel := mock.NotifyContext(context.Background())
		defer cancel()

		require.NoError(t, ctx.Err())

		mock.Trigger()

		assert.Error(t, ctx.Err())
		assert.True(t, mock.WasTriggered())
	})

	t.Run("is idempotent", func(t *testing.T) {
		t.Parallel()

		mock := NewMockSignalNotifier()
		_, cancel := mock.NotifyContext(context.Background())
		defer cancel()

		mock.Trigger()
		mock.Trigger()
		mock.Trigger()

		assert.True(t, mock.WasTriggered())
	})

	t.Run("is safe without prior NotifyContext call", func(t *testing.T) {
		t.Parallel()

		mock := NewMockSignalNotifier()

		mock.Trigger()

		assert.True(t, mock.WasTriggered())
	})
}

func TestMockSignalNotifier_WasTriggered_Subtests(t *testing.T) {
	t.Parallel()

	t.Run("returns false before Trigger", func(t *testing.T) {
		t.Parallel()

		mock := NewMockSignalNotifier()

		assert.False(t, mock.WasTriggered())
	})

	t.Run("returns true after Trigger", func(t *testing.T) {
		t.Parallel()

		mock := NewMockSignalNotifier()
		mock.Trigger()

		assert.True(t, mock.WasTriggered())
	})
}

func TestMockSignalNotifier_Reset_Subtests(t *testing.T) {
	t.Parallel()

	t.Run("clears triggered state", func(t *testing.T) {
		t.Parallel()

		mock := NewMockSignalNotifier()
		_, cancel := mock.NotifyContext(context.Background())
		defer cancel()

		mock.Trigger()
		require.True(t, mock.WasTriggered())

		mock.Reset()

		assert.False(t, mock.WasTriggered())
	})

	t.Run("clears call count", func(t *testing.T) {
		t.Parallel()

		mock := NewMockSignalNotifier()
		_, cancel := mock.NotifyContext(context.Background())
		defer cancel()

		require.Equal(t, int64(1), atomic.LoadInt64(&mock.NotifyContextCallCount))

		mock.Reset()

		assert.Equal(t, int64(0), atomic.LoadInt64(&mock.NotifyContextCallCount))
	})

	t.Run("allows re-trigger after reset", func(t *testing.T) {
		t.Parallel()

		mock := NewMockSignalNotifier()
		ctx1, cancel1 := mock.NotifyContext(context.Background())
		defer cancel1()

		mock.Trigger()
		require.Error(t, ctx1.Err())

		mock.Reset()

		ctx2, cancel2 := mock.NotifyContext(context.Background())
		defer cancel2()

		require.NoError(t, ctx2.Err())

		mock.Trigger()
		assert.Error(t, ctx2.Err())
	})
}

func TestMockSignalNotifier_NotifyContextCalled(t *testing.T) {
	t.Parallel()

	t.Run("returns false before any calls", func(t *testing.T) {
		t.Parallel()

		mock := NewMockSignalNotifier()

		assert.False(t, mock.NotifyContextCalled())
	})

	t.Run("returns true after NotifyContext is called", func(t *testing.T) {
		t.Parallel()

		mock := NewMockSignalNotifier()
		_, cancel := mock.NotifyContext(context.Background())
		defer cancel()

		assert.True(t, mock.NotifyContextCalled())
	})
}

func TestMockSignalNotifier_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockSignalNotifier

	ctx, cancel := mock.NotifyContext(context.Background())
	defer cancel()

	require.NotNil(t, ctx)
	require.NotNil(t, cancel)
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NotifyContextCallCount))
	assert.False(t, mock.WasTriggered())
	assert.True(t, mock.NotifyContextCalled())
}

func TestMockSignalNotifier_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := NewMockSignalNotifier()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for range goroutines {
		go func() {
			defer wg.Done()
			ctx, cancel := mock.NotifyContext(context.Background())
			defer cancel()
			_ = ctx.Err()
		}()
		go func() {
			defer wg.Done()
			_ = mock.WasTriggered()
		}()
		go func() {
			defer wg.Done()
			_ = mock.NotifyContextCalled()
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.NotifyContextCallCount))
}

func TestMockSignalNotifier_ImplementsSignalNotifier(t *testing.T) {
	t.Parallel()

	mock := NewMockSignalNotifier()
	var _ SignalNotifier = mock
}

type MockDrainSignaller struct {
	SignalDrainCallCount int64
}

func (m *MockDrainSignaller) SignalDrain() {
	atomic.AddInt64(&m.SignalDrainCallCount, 1)
}

func TestMockDrainSignaller_ImplementsDrainSignaller(t *testing.T) {
	t.Parallel()

	var _ DrainSignaller = &MockDrainSignaller{}
}

func TestMockDrainSignaller_TracksCallCount(t *testing.T) {
	t.Parallel()

	mock := &MockDrainSignaller{}

	assert.Equal(t, int64(0), atomic.LoadInt64(&mock.SignalDrainCallCount))

	mock.SignalDrain()
	mock.SignalDrain()

	assert.Equal(t, int64(2), atomic.LoadInt64(&mock.SignalDrainCallCount))
}
