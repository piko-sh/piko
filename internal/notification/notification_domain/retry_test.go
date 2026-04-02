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

package notification_domain

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestIsRetryableError_Nil(t *testing.T) {
	if IsRetryableError(nil) {
		t.Error("expected nil error to not be retryable")
	}
}

func TestIsRetryableError_PermanentErrors(t *testing.T) {
	permanentErrors := []struct {
		err  error
		name string
	}{
		{name: "context.Canceled", err: context.Canceled},
		{name: "context.DeadlineExceeded", err: context.DeadlineExceeded},
		{name: "ErrProviderNotFound", err: ErrProviderNotFound},
		{name: "ErrNoProviders", err: ErrNoProviders},
		{name: "ErrNoDefaultProvider", err: ErrNoDefaultProvider},
		{name: "ErrEmptyMessage", err: ErrEmptyMessage},
		{name: "ErrEmptyTitle", err: ErrEmptyTitle},
		{name: "ErrUnsupportedContentType", err: ErrUnsupportedContentType},
		{name: "ErrMessageTooLong", err: ErrMessageTooLong},
		{name: "wrapped context.Canceled", err: fmt.Errorf("wrapped: %w", context.Canceled)},
		{name: "wrapped ErrProviderNotFound", err: fmt.Errorf("wrapped: %w", ErrProviderNotFound)},
	}

	for _, tc := range permanentErrors {
		t.Run(tc.name, func(t *testing.T) {
			if IsRetryableError(tc.err) {
				t.Errorf("expected %q to be permanent (not retryable)", tc.err)
			}
		})
	}
}

func TestIsRetryableError_NotificationPatterns(t *testing.T) {
	patterns := []string{
		"slack api error",
		"discord webhook failed",
		"webhook error occurred",
		"circuit breaker open",
	}

	for _, message := range patterns {
		t.Run(message, func(t *testing.T) {
			err := errors.New(message)
			if !IsRetryableError(err) {
				t.Errorf("expected %q to be retryable", message)
			}
		})
	}
}

func TestIsRetryableError_NonRetryableMessage(t *testing.T) {
	nonRetryable := []string{
		"unknown error",
		"invalid json payload",
		"authentication failed",
		"permission denied",
	}

	for _, message := range nonRetryable {
		t.Run(message, func(t *testing.T) {
			err := errors.New(message)
			if IsRetryableError(err) {
				t.Errorf("expected %q to not be retryable", message)
			}
		})
	}
}
