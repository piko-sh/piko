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

package retry_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/retry"
)

func TestErrorClassifier_IsRetryable_Nil(t *testing.T) {
	c := retry.NewErrorClassifier()
	assert.False(t, c.IsRetryable(nil))
}

func TestErrorClassifier_IsRetryable_DefaultPermanentErrors(t *testing.T) {
	c := retry.NewErrorClassifier()

	testCases := []struct {
		err  error
		name string
	}{
		{name: "context.Canceled", err: context.Canceled},
		{name: "context.DeadlineExceeded", err: context.DeadlineExceeded},
		{name: "wrapped context.Canceled", err: fmt.Errorf("op failed: %w", context.Canceled)},
		{name: "wrapped DeadlineExceeded", err: fmt.Errorf("timed out: %w", context.DeadlineExceeded)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.False(t, c.IsRetryable(tc.err),
				"error %q should be permanent (not retryable)", tc.err)
		})
	}
}

func TestErrorClassifier_IsRetryable_CustomPermanentErrors(t *testing.T) {
	errNotFound := errors.New("not found")
	errForbidden := errors.New("forbidden")

	c := retry.NewErrorClassifier(
		retry.WithPermanentErrors(errNotFound, errForbidden),
	)

	t.Run("custom permanent error", func(t *testing.T) {
		assert.False(t, c.IsRetryable(errNotFound))
		assert.False(t, c.IsRetryable(errForbidden))
	})

	t.Run("wrapped custom permanent error", func(t *testing.T) {
		assert.False(t, c.IsRetryable(fmt.Errorf("query: %w", errNotFound)))
	})

	t.Run("default permanent errors still work", func(t *testing.T) {
		assert.False(t, c.IsRetryable(context.Canceled))
	})
}

type mockNetError struct {
	timeout bool
}

func (e *mockNetError) Error() string   { return "mock net error" }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return false }

var _ net.Error = (*mockNetError)(nil)

func TestErrorClassifier_IsRetryable_NetworkTimeout(t *testing.T) {
	c := retry.NewErrorClassifier()

	t.Run("network timeout is retryable", func(t *testing.T) {
		err := &mockNetError{timeout: true}
		assert.True(t, c.IsRetryable(err))
	})

	t.Run("non-timeout network error is not retryable", func(t *testing.T) {
		err := &mockNetError{timeout: false}
		assert.False(t, c.IsRetryable(err))
	})
}

func TestErrorClassifier_IsRetryable_SyscallErrors(t *testing.T) {
	c := retry.NewErrorClassifier()

	retryableSyscalls := []struct {
		name  string
		errno syscall.Errno
	}{
		{name: "ECONNREFUSED", errno: syscall.ECONNREFUSED},
		{name: "ECONNRESET", errno: syscall.ECONNRESET},
		{name: "ETIMEDOUT", errno: syscall.ETIMEDOUT},
		{name: "EHOSTUNREACH", errno: syscall.EHOSTUNREACH},
		{name: "ENETUNREACH", errno: syscall.ENETUNREACH},
		{name: "ECONNABORTED", errno: syscall.ECONNABORTED},
	}

	for _, tc := range retryableSyscalls {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, c.IsRetryable(tc.errno),
				"syscall error %v should be retryable", tc.errno)
		})
	}

	t.Run("wrapped syscall error is retryable", func(t *testing.T) {
		wrapped := fmt.Errorf("dial tcp: %w", syscall.ECONNREFUSED)
		assert.True(t, c.IsRetryable(wrapped))
	})

	t.Run("non-retryable syscall error", func(t *testing.T) {
		assert.False(t, c.IsRetryable(syscall.ENOENT),
			"ENOENT should not be retryable")
	})
}

func TestErrorClassifier_IsRetryable_MessagePatterns(t *testing.T) {
	c := retry.NewErrorClassifier()

	retryable := []string{
		"connection refused",
		"connection reset by peer",
		"connection timeout occurred",
		"timeout waiting for response",
		"temporary failure in name resolution",
		"too many requests",
		"rate limit exceeded",
		"throttle applied",
		"HTTP 500 internal server error",
		"HTTP 502 bad gateway",
		"HTTP 503 service unavailable",
		"HTTP 504 gateway timeout",
		"CONNECTION REFUSED",
	}

	for _, message := range retryable {
		t.Run(message, func(t *testing.T) {
			err := errors.New(message)
			assert.True(t, c.IsRetryable(err),
				"error %q should be retryable", message)
		})
	}
}

func TestErrorClassifier_IsRetryable_NonRetryableMessages(t *testing.T) {
	c := retry.NewErrorClassifier()

	nonRetryable := []string{
		"some generic error",
		"object not found",
		"invalid parameter",
		"unknown error",
		"invalid json payload",
		"authentication failed",
		"permission denied",
	}

	for _, message := range nonRetryable {
		t.Run(message, func(t *testing.T) {
			err := errors.New(message)
			assert.False(t, c.IsRetryable(err),
				"error %q should not be retryable", message)
		})
	}
}

func TestErrorClassifier_IsRetryable_CustomPatterns(t *testing.T) {
	c := retry.NewErrorClassifier(
		retry.WithRetryablePatterns("slack api", "discord webhook", "webhook error"),
	)

	t.Run("custom patterns are matched", func(t *testing.T) {
		assert.True(t, c.IsRetryable(errors.New("slack api error")))
		assert.True(t, c.IsRetryable(errors.New("discord webhook failed")))
		assert.True(t, c.IsRetryable(errors.New("webhook error occurred")))
	})

	t.Run("default patterns still work", func(t *testing.T) {
		assert.True(t, c.IsRetryable(errors.New("connection refused")))
	})
}

func TestIsNetworkTimeout(t *testing.T) {
	t.Run("timeout is true", func(t *testing.T) {
		assert.True(t, retry.IsNetworkTimeout(&mockNetError{timeout: true}))
	})

	t.Run("non-timeout is false", func(t *testing.T) {
		assert.False(t, retry.IsNetworkTimeout(&mockNetError{timeout: false}))
	})

	t.Run("non-net error is false", func(t *testing.T) {
		assert.False(t, retry.IsNetworkTimeout(errors.New("not a net error")))
	})
}

func TestIsSyscallRetryable(t *testing.T) {
	t.Run("ECONNREFUSED is retryable", func(t *testing.T) {
		assert.True(t, retry.IsSyscallRetryable(syscall.ECONNREFUSED))
	})

	t.Run("ENOENT is not retryable", func(t *testing.T) {
		assert.False(t, retry.IsSyscallRetryable(syscall.ENOENT))
	})

	t.Run("non-syscall error is not retryable", func(t *testing.T) {
		assert.False(t, retry.IsSyscallRetryable(errors.New("not a syscall")))
	})
}
