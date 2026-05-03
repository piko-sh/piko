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

//go:build !bench

package logger_domain_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
)

type mockCloser struct {
	closeErr error
	mu       sync.Mutex
	closed   bool
}

func (m *mockCloser) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return m.closeErr
}

func (m *mockCloser) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func TestLifecycleManager_RegisterShutdownHook(t *testing.T) {
	lm := logger_domain.NewLifecycleManager()

	hookCalled := false
	lm.RegisterShutdownHook(func() {
		hookCalled = true
	})

	err := lm.Shutdown(context.Background())
	require.NoError(t, err)

	assert.True(t, hookCalled, "shutdown hook should be called")
}

func TestLifecycleManager_MultipleShutdownHooks(t *testing.T) {
	lm := logger_domain.NewLifecycleManager()

	var callOrder []int
	var mu sync.Mutex

	lm.RegisterShutdownHook(func() {
		mu.Lock()
		defer mu.Unlock()
		callOrder = append(callOrder, 1)
	})

	lm.RegisterShutdownHook(func() {
		mu.Lock()
		defer mu.Unlock()
		callOrder = append(callOrder, 2)
	})

	lm.RegisterShutdownHook(func() {
		mu.Lock()
		defer mu.Unlock()
		callOrder = append(callOrder, 3)
	})

	err := lm.Shutdown(context.Background())
	require.NoError(t, err)

	assert.Equal(t, []int{3, 2, 1}, callOrder, "hooks should be called in LIFO order")
}

func TestLifecycleManager_RegisterClosable(t *testing.T) {
	lm := logger_domain.NewLifecycleManager()

	closer := &mockCloser{}
	lm.RegisterClosable(closer)

	err := lm.Shutdown(context.Background())
	require.NoError(t, err)

	assert.True(t, closer.IsClosed(), "closable should be closed")
}

func TestLifecycleManager_MultipleClosables(t *testing.T) {
	lm := logger_domain.NewLifecycleManager()

	closer1 := &mockCloser{}
	closer2 := &mockCloser{}
	closer3 := &mockCloser{}

	lm.RegisterClosable(closer1)
	lm.RegisterClosable(closer2)
	lm.RegisterClosable(closer3)

	err := lm.Shutdown(context.Background())
	require.NoError(t, err)

	assert.True(t, closer1.IsClosed(), "closer1 should be closed")
	assert.True(t, closer2.IsClosed(), "closer2 should be closed")
	assert.True(t, closer3.IsClosed(), "closer3 should be closed")
}

func TestLifecycleManager_ClosableError(t *testing.T) {
	lm := logger_domain.NewLifecycleManager()

	closeError := errors.New("close failed")
	closer1 := &mockCloser{closeErr: closeError}
	closer2 := &mockCloser{}

	lm.RegisterClosable(closer1)
	lm.RegisterClosable(closer2)

	err := lm.Shutdown(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "close failed", "error should mention the close failure")

	assert.True(t, closer1.IsClosed(), "closer1 should be attempted")
	assert.True(t, closer2.IsClosed(), "closer2 should still be closed despite closer1 error")
}

func TestLifecycleManager_HooksBeforeClosables(t *testing.T) {
	lm := logger_domain.NewLifecycleManager()

	var executionOrder []string
	var mu sync.Mutex

	lm.RegisterShutdownHook(func() {
		mu.Lock()
		defer mu.Unlock()
		executionOrder = append(executionOrder, "hook")
	})

	lm.RegisterClosable(&closerWithCallback{
		callback: func() {
			mu.Lock()
			defer mu.Unlock()
			executionOrder = append(executionOrder, "closable")
		},
	})

	err := lm.Shutdown(context.Background())
	require.NoError(t, err)

	assert.Equal(t, []string{"hook", "closable"}, executionOrder,
		"hooks should execute before closables")
}

func TestLifecycleManager_EmptyShutdown(t *testing.T) {
	lm := logger_domain.NewLifecycleManager()

	err := lm.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestLifecycleManager_MultipleShutdowns(t *testing.T) {
	lm := logger_domain.NewLifecycleManager()

	hookCalled := 0
	lm.RegisterShutdownHook(func() {
		hookCalled++
	})

	closer := &mockCloser{}
	lm.RegisterClosable(closer)

	err := lm.Shutdown(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, hookCalled, "hook should be called once")
	assert.True(t, closer.IsClosed(), "closer should be closed")

	closer.mu.Lock()
	closer.closed = false
	closer.mu.Unlock()

	err = lm.Shutdown(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, hookCalled, "hook should not be called again")
	assert.False(t, closer.IsClosed(), "closer should not be closed again")
}

func TestLifecycleManager_ThreadSafety(t *testing.T) {
	lm := logger_domain.NewLifecycleManager()

	var wg sync.WaitGroup
	hookCallCount := 0
	var mu sync.Mutex

	for range 10 {
		wg.Go(func() {
			lm.RegisterShutdownHook(func() {
				mu.Lock()
				defer mu.Unlock()
				hookCallCount++
			})
		})
	}

	closers := make([]*mockCloser, 10)
	for i := range 10 {
		closers[i] = &mockCloser{}
		index := i
		wg.Go(func() {
			lm.RegisterClosable(closers[index])
		})
	}

	wg.Wait()

	err := lm.Shutdown(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 10, hookCallCount, "all hooks should be called")
	for i, closer := range closers {
		assert.True(t, closer.IsClosed(), "closer %d should be closed", i)
	}
}

func TestLifecycleManager_Isolation(t *testing.T) {

	lm1 := logger_domain.NewLifecycleManager()
	lm2 := logger_domain.NewLifecycleManager()

	hook1Called := false
	hook2Called := false

	lm1.RegisterShutdownHook(func() {
		hook1Called = true
	})

	lm2.RegisterShutdownHook(func() {
		hook2Called = true
	})

	err := lm1.Shutdown(context.Background())
	require.NoError(t, err)

	assert.True(t, hook1Called, "lm1 hook should be called")
	assert.False(t, hook2Called, "lm2 hook should NOT be called")

	err = lm2.Shutdown(context.Background())
	require.NoError(t, err)

	assert.True(t, hook2Called, "lm2 hook should now be called")
}

func TestDefaultLifecycleManager_BackwardCompatibility(t *testing.T) {

	hookCalled := false
	logger_domain.RegisterShutdownHook(func() {
		hookCalled = true
	})

	closer := &mockCloser{}
	logger_domain.RegisterClosable(closer)

	err := logger_domain.Shutdown(context.Background())
	require.NoError(t, err)

	assert.True(t, hookCalled, "global shutdown hook should be called")
	assert.True(t, closer.IsClosed(), "global closable should be closed")
}

func TestNotificationHandler_UsesLifecycleManager(t *testing.T) {
	lm := logger_domain.NewLifecycleManager()

	transport := &mockTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)
	handler := logger_domain.NewNotificationHandlerWithOptions(
		baseHandler,
		transport,
		slog.LevelError,
		clock.RealClock(),
		lm,
	)

	require.NotNil(t, handler)

	err := lm.Shutdown(context.Background())
	require.NoError(t, err)

	assert.Empty(t, transport.batches, "no batches should be sent on empty shutdown")
}

func TestNotificationHandler_NoLifecycleManager(t *testing.T) {

	transport := &mockTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)
	handler := logger_domain.NewNotificationHandlerWithOptions(
		baseHandler,
		transport,
		slog.LevelError,
		clock.RealClock(),
		nil,
	)

	require.NotNil(t, handler)

	handler.Shutdown()

	assert.Empty(t, transport.batches, "no batches should be sent")
}

type closerWithCallback struct {
	callback func()
}

func (c *closerWithCallback) Close() error {
	c.callback()
	return nil
}

type mockTransport struct {
	batches []map[string]*logger_domain.GroupedError
}

func (m *mockTransport) SendGroupedErrors(ctx context.Context, batch map[string]*logger_domain.GroupedError) error {
	m.batches = append(m.batches, batch)
	return nil
}

func TestLifecycleManager_Shutdown_AllowsHooksToRegister(t *testing.T) {
	lm := logger_domain.NewLifecycleManager()

	innerCalled := false
	lm.RegisterShutdownHook(func() {
		lm.RegisterShutdownHook(func() {
			innerCalled = true
		})
	})

	closer := &mockCloser{}
	lm.RegisterShutdownHook(func() {
		lm.RegisterClosable(closer)
	})

	done := make(chan error, 1)
	go func() {
		done <- lm.Shutdown(context.Background())
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown deadlocked when a hook tried to register another hook")
	}

	assert.False(t, innerCalled,
		"hook registered during shutdown should not run in the same shutdown cycle")
	assert.False(t, closer.IsClosed(),
		"closable registered during shutdown should not be closed in the same cycle")

	require.NoError(t, lm.Shutdown(context.Background()))
	assert.True(t, innerCalled,
		"hook registered during shutdown should run on the next shutdown")
	assert.True(t, closer.IsClosed(),
		"closable registered during shutdown should be closed on the next shutdown")
}

func TestLifecycleManager_Shutdown_JoinsErrors(t *testing.T) {
	lm := logger_domain.NewLifecycleManager()

	first := errors.New("first close failure")
	second := errors.New("second close failure")
	lm.RegisterClosable(&mockCloser{closeErr: first})
	lm.RegisterClosable(&mockCloser{closeErr: second})

	err := lm.Shutdown(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, first, "joined error should wrap the first close failure")
	assert.ErrorIs(t, err, second, "joined error should wrap the second close failure")
}

func TestClearLifecycle(t *testing.T) {

	hookCalled := false
	logger_domain.DefaultLifecycleManager.RegisterShutdownHook(func() {
		hookCalled = true
	})

	closeCalled := false
	closer := &closerWithCallback{callback: func() {
		closeCalled = true
	}}
	logger_domain.DefaultLifecycleManager.RegisterClosable(closer)

	logger_domain.ClearLifecycle()

	assert.True(t, closeCalled, "closable should be closed during ClearLifecycle")

	_ = logger_domain.DefaultLifecycleManager.Shutdown(context.Background())
	assert.False(t, hookCalled, "hook should not be called after ClearLifecycle")
}
