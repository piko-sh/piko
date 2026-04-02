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

package logger_state

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type discardHandler struct{}

func (h *discardHandler) Enabled(_ context.Context, _ slog.Level) bool  { return false }
func (h *discardHandler) Handle(_ context.Context, _ slog.Record) error { return nil }
func (h *discardHandler) WithAttrs(_ []slog.Attr) slog.Handler          { return h }
func (h *discardHandler) WithGroup(_ string) slog.Handler               { return h }

type trackingCloser struct {
	closed bool
}

func (c *trackingCloser) Close() error {
	c.closed = true
	return nil
}

func resetTestState(t *testing.T) {
	t.Helper()
	ResetState()
}

func TestHasExplicitHandlers(t *testing.T) {
	t.Run("returns false after reset", func(t *testing.T) {
		resetTestState(t)

		assert.False(t, HasExplicitHandlers())
	})

	t.Run("returns true after adding handler", func(t *testing.T) {
		resetTestState(t)

		AddHandler(&discardHandler{}, nil)

		assert.True(t, HasExplicitHandlers())
	})

	t.Run("returns false after clear", func(t *testing.T) {
		resetTestState(t)

		AddHandler(&discardHandler{}, nil)
		ClearAllHandlers()

		assert.False(t, HasExplicitHandlers())
	})
}

func TestAddHandler(t *testing.T) {
	t.Run("adds handler without closer", func(t *testing.T) {
		resetTestState(t)

		AddHandler(&discardHandler{}, nil)

		assert.True(t, HasExplicitHandlers())
	})

	t.Run("adds handler with closer", func(t *testing.T) {
		resetTestState(t)

		closer := &trackingCloser{}
		AddHandler(&discardHandler{}, closer)

		assert.True(t, HasExplicitHandlers())

		shutdown := GetShutdownFunc()
		err := shutdown(context.Background())
		require.NoError(t, err)
		assert.True(t, closer.closed)
	})

	t.Run("replaces default handler", func(t *testing.T) {
		resetTestState(t)

		assert.False(t, HasExplicitHandlers())
		AddHandler(&discardHandler{}, nil)
		assert.True(t, HasExplicitHandlers())
	})
}

func TestAddWrapper(t *testing.T) {
	resetTestState(t)

	called := false
	AddWrapper(func(h slog.Handler) slog.Handler {
		called = true
		return h
	})

	assert.True(t, called)
}

func TestClearAllHandlers(t *testing.T) {
	t.Run("removes all handlers", func(t *testing.T) {
		resetTestState(t)

		AddHandler(&discardHandler{}, nil)
		AddHandler(&discardHandler{}, nil)
		assert.True(t, HasExplicitHandlers())

		ClearAllHandlers()

		assert.False(t, HasExplicitHandlers())
	})

	t.Run("closes active closers", func(t *testing.T) {
		resetTestState(t)

		closer := &trackingCloser{}
		AddHandler(&discardHandler{}, closer)

		ClearAllHandlers()

		assert.True(t, closer.closed)
	})
}

func TestResetState(t *testing.T) {
	t.Run("resets to default state", func(t *testing.T) {
		AddHandler(&discardHandler{}, nil)
		assert.True(t, HasExplicitHandlers())

		ResetState()

		assert.False(t, HasExplicitHandlers())
	})

	t.Run("closes existing closers", func(t *testing.T) {
		closer := &trackingCloser{}
		AddHandler(&discardHandler{}, closer)

		ResetState()

		assert.True(t, closer.closed)
	})
}

func TestAddShutdownHook(t *testing.T) {
	resetTestState(t)

	hookCalled := false
	AddShutdownHook(func(_ context.Context) error {
		hookCalled = true
		return nil
	})

	shutdown := GetShutdownFunc()
	err := shutdown(context.Background())
	require.NoError(t, err)
	assert.True(t, hookCalled)
}

func TestGetShutdownFunc(t *testing.T) {
	t.Run("executes shutdown hooks", func(t *testing.T) {
		resetTestState(t)

		var order []int
		AddShutdownHook(func(_ context.Context) error {
			order = append(order, 1)
			return nil
		})
		AddShutdownHook(func(_ context.Context) error {
			order = append(order, 2)
			return nil
		})

		shutdown := GetShutdownFunc()
		err := shutdown(context.Background())

		require.NoError(t, err)
		assert.Equal(t, []int{1, 2}, order)
	})

	t.Run("closes active closers", func(t *testing.T) {
		resetTestState(t)

		closer := &trackingCloser{}
		AddHandler(&discardHandler{}, closer)

		shutdown := GetShutdownFunc()
		err := shutdown(context.Background())

		require.NoError(t, err)
		assert.True(t, closer.closed)
	})
}

func TestGetSharedHTTPClient(t *testing.T) {
	client := GetSharedHTTPClient()

	require.NotNil(t, client)

	client2 := GetSharedHTTPClient()
	assert.Same(t, client, client2)
}

func TestGetSentryInitOnce(t *testing.T) {
	once := GetSentryInitOnce()
	require.NotNil(t, once)

	once2 := GetSentryInitOnce()
	assert.Same(t, once, once2)
}

var (
	_ slog.Handler = (*discardHandler)(nil)
	_ io.Closer    = (*trackingCloser)(nil)
)
