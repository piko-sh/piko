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

package goroutine

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeCall_RecoversPanic(t *testing.T) {
	t.Parallel()

	err := SafeCall(context.Background(), "test.component", func() error {
		panic("boom")
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic in test.component: boom")

	panicErr, ok := errors.AsType[*PanicError](err)
	require.True(t, ok)
	assert.Equal(t, "test.component", panicErr.Component)
	assert.Equal(t, "boom", panicErr.Value)
	assert.NotEmpty(t, panicErr.Stack)
}

func TestSafeCall_NoPanic(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("normal error")
	err := SafeCall(context.Background(), "test.component", func() error {
		return sentinel
	})

	assert.ErrorIs(t, err, sentinel)
}

func TestSafeCall_NoPanicNilError(t *testing.T) {
	t.Parallel()

	err := SafeCall(context.Background(), "test.component", func() error {
		return nil
	})

	assert.NoError(t, err)
}

func TestSafeCall1_RecoversPanic(t *testing.T) {
	t.Parallel()

	result, err := SafeCall1(context.Background(), "test.provider", func() (string, error) {
		panic("provider crashed")
	})

	require.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "panic in test.provider: provider crashed")

	panicErr, ok := errors.AsType[*PanicError](err)
	require.True(t, ok)
	assert.Equal(t, "test.provider", panicErr.Component)
}

func TestSafeCall1_NoPanic(t *testing.T) {
	t.Parallel()

	result, err := SafeCall1(context.Background(), "test.provider", func() (string, error) {
		return "hello", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestSafeCall1_NoPanicWithError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("provider error")
	result, err := SafeCall1(context.Background(), "test.provider", func() (int, error) {
		return 0, sentinel
	})

	assert.ErrorIs(t, err, sentinel)
	assert.Zero(t, result)
}

func TestSafeCall2_RecoversPanic(t *testing.T) {
	t.Parallel()

	r1, r2, err := SafeCall2(context.Background(), "test.cache", func() (string, bool, error) {
		panic("cache panicked")
	})

	require.Error(t, err)
	assert.Empty(t, r1)
	assert.False(t, r2)
	assert.Contains(t, err.Error(), "panic in test.cache: cache panicked")

	_, ok := errors.AsType[*PanicError](err)
	require.True(t, ok)
}

func TestSafeCall2_NoPanic(t *testing.T) {
	t.Parallel()

	r1, r2, err := SafeCall2(context.Background(), "test.cache", func() (string, bool, error) {
		return "value", true, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "value", r1)
	assert.True(t, r2)
}

func TestSafeCallValue_RecoversPanic(t *testing.T) {
	t.Parallel()

	result := SafeCallValue(context.Background(), "test.capability", func() bool {
		panic("capability panicked")
	})

	assert.False(t, result)
}

func TestSafeCallValue_NoPanic(t *testing.T) {
	t.Parallel()

	result := SafeCallValue(context.Background(), "test.capability", func() bool {
		return true
	})

	assert.True(t, result)
}

func TestSafeCallValue_ReturnsZeroOnPanic(t *testing.T) {
	t.Parallel()

	result := SafeCallValue(context.Background(), "test.size", func() int {
		panic("size panicked")
	})

	assert.Zero(t, result)
}

func TestSafeCallValue_StringReturnsEmptyOnPanic(t *testing.T) {
	t.Parallel()

	result := SafeCallValue(context.Background(), "test.model", func() string {
		panic("model panicked")
	})

	assert.Empty(t, result)
}

func TestPanicError_ErrorsAs(t *testing.T) {
	t.Parallel()

	err := SafeCall(context.Background(), "test.as", func() error {
		panic("for errors.As")
	})

	panicErr, ok := errors.AsType[*PanicError](err)
	require.True(t, ok)
	assert.Equal(t, "test.as", panicErr.Component)
	assert.Equal(t, "for errors.As", panicErr.Value)
	assert.Contains(t, panicErr.Stack, "goroutine")
}

func TestPanicError_NotConfusedWithNormalError(t *testing.T) {
	t.Parallel()

	err := SafeCall(context.Background(), "test.normal", func() error {
		return errors.New("normal error")
	})

	_, ok := errors.AsType[*PanicError](err)
	assert.False(t, ok)
}

func TestProviderTimeout_DetectedWhenCallerCtxAlive(t *testing.T) {
	t.Parallel()

	err := SafeCall(context.Background(), "test.redis", func() error {
		return context.DeadlineExceeded
	})

	require.Error(t, err)

	timeoutErr, ok := errors.AsType[*ProviderTimeoutError](err)
	require.True(t, ok)
	assert.Equal(t, "test.redis", timeoutErr.Component)
	assert.ErrorIs(t, timeoutErr.Err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "provider timeout in test.redis")
}

func TestProviderTimeout_NotDetectedWhenCallerCtxDead(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("our timeout"))

	err := SafeCall(ctx, "test.redis", func() error {
		return context.DeadlineExceeded
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	_, ok := errors.AsType[*ProviderTimeoutError](err)
	assert.False(t, ok)
}

func TestProviderCancellation_DetectedWhenCallerCtxAlive(t *testing.T) {
	t.Parallel()

	err := SafeCall(context.Background(), "test.http", func() error {
		return context.Canceled
	})

	require.Error(t, err)

	timeoutErr, ok := errors.AsType[*ProviderTimeoutError](err)
	require.True(t, ok)
	assert.Equal(t, "test.http", timeoutErr.Component)
	assert.ErrorIs(t, timeoutErr.Err, context.Canceled)
}

func TestProviderTimeout_WrappedErrorStillMatchesIs(t *testing.T) {
	t.Parallel()

	err := SafeCall(context.Background(), "test.cache", func() error {
		return context.DeadlineExceeded
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestProviderTimeout_ErrorsAs(t *testing.T) {
	t.Parallel()

	err := SafeCall(context.Background(), "test.storage", func() error {
		return context.DeadlineExceeded
	})

	timeoutErr, ok := errors.AsType[*ProviderTimeoutError](err)
	require.True(t, ok)
	assert.Equal(t, "test.storage", timeoutErr.Component)
}

func TestProviderTimeout_NormalErrorNotWrapped(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("connection refused")
	err := SafeCall(context.Background(), "test.redis", func() error {
		return sentinel
	})

	assert.ErrorIs(t, err, sentinel)

	_, ok := errors.AsType[*ProviderTimeoutError](err)
	assert.False(t, ok)
}

func TestProviderTimeout_SafeCall1(t *testing.T) {
	t.Parallel()

	result, err := SafeCall1(context.Background(), "test.provider", func() (string, error) {
		return "", context.DeadlineExceeded
	})

	require.Error(t, err)
	assert.Empty(t, result)

	timeoutErr, ok := errors.AsType[*ProviderTimeoutError](err)
	require.True(t, ok)
	assert.Equal(t, "test.provider", timeoutErr.Component)
}

func TestProviderTimeout_SafeCall2(t *testing.T) {
	t.Parallel()

	r1, r2, err := SafeCall2(context.Background(), "test.cache", func() (string, bool, error) {
		return "", false, context.DeadlineExceeded
	})

	require.Error(t, err)
	assert.Empty(t, r1)
	assert.False(t, r2)

	timeoutErr, ok := errors.AsType[*ProviderTimeoutError](err)
	require.True(t, ok)
	assert.Equal(t, "test.cache", timeoutErr.Component)
}

func TestProviderTimeout_WrappedDeadline(t *testing.T) {
	t.Parallel()

	err := SafeCall(context.Background(), "test.redis", func() error {
		return fmt.Errorf("redis: %w", context.DeadlineExceeded)
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	timeoutErr, ok := errors.AsType[*ProviderTimeoutError](err)
	require.True(t, ok)
	assert.Equal(t, "test.redis", timeoutErr.Component)
}
