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

package safeerror

import (
	"errors"
	"fmt"
	"testing"
)

var errSentinel = errors.New("sentinel")

func TestNewError(t *testing.T) {
	t.Parallel()

	cause := errors.New("database connection refused")
	err := NewError("something went wrong", cause)

	if err.Error() != "database connection refused" {
		t.Errorf("Error() = %q, want %q", err.Error(), "database connection refused")
	}

	var safeErr Error
	if !errors.As(err, &safeErr) {
		t.Fatal("expected error to implement Error interface")
	}

	if safeErr.SafeMessage() != "something went wrong" {
		t.Errorf("SafeMessage() = %q, want %q", safeErr.SafeMessage(), "something went wrong")
	}
}

func TestErrorf(t *testing.T) {
	t.Parallel()

	err := Errorf("could not load user", "querying user %s in table %s: %w", "42", "users", errSentinel)

	expected := fmt.Sprintf("querying user %s in table %s: %s", "42", "users", errSentinel.Error())
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}

	var safeErr Error
	if !errors.As(err, &safeErr) {
		t.Fatal("expected error to implement Error interface")
	}

	if safeErr.SafeMessage() != "could not load user" {
		t.Errorf("SafeMessage() = %q, want %q", safeErr.SafeMessage(), "could not load user")
	}
}

func TestUnwrapPreservesSentinel(t *testing.T) {
	t.Parallel()

	err := NewError("safe message", errSentinel)

	if !errors.Is(err, errSentinel) {
		t.Error("expected errors.Is to find sentinel through NewError wrapping")
	}
}

func TestUnwrapPreservesSentinelThroughErrorf(t *testing.T) {
	t.Parallel()

	err := Errorf("safe message", "context: %w", errSentinel)

	if !errors.Is(err, errSentinel) {
		t.Error("expected errors.Is to find sentinel through Errorf wrapping")
	}
}

func TestUnwrapPreservesSentinelThroughFmtErrorf(t *testing.T) {
	t.Parallel()

	inner := NewError("safe message", errSentinel)
	outer := fmt.Errorf("wrapping context: %w", inner)

	if !errors.Is(outer, errSentinel) {
		t.Error("expected errors.Is to find sentinel through fmt.Errorf + NewError chain")
	}

	if _, ok := errors.AsType[Error](outer); !ok {
		t.Error("expected errors.As to find Error through fmt.Errorf wrapping")
	}
}

func TestExtractSafeMessage_DevelopmentMode(t *testing.T) {
	t.Parallel()

	err := NewError("safe message", errors.New("internal database failure"))

	result := ExtractSafeMessage(err, true)
	if result != "internal database failure" {
		t.Errorf("ExtractSafeMessage(dev) = %q, want %q", result, "internal database failure")
	}
}

func TestExtractSafeMessage_ProductionWithSafeError(t *testing.T) {
	t.Parallel()

	err := NewError("something went wrong", errors.New("internal database failure"))

	result := ExtractSafeMessage(err, false)
	if result != "something went wrong" {
		t.Errorf("ExtractSafeMessage(prod, SafeError) = %q, want %q", result, "something went wrong")
	}
}

func TestExtractSafeMessage_ProductionWithPlainError(t *testing.T) {
	t.Parallel()

	err := errors.New("internal database failure")

	result := ExtractSafeMessage(err, false)
	if result != "An internal error occurred" {
		t.Errorf("ExtractSafeMessage(prod, plain) = %q, want %q", result, "An internal error occurred")
	}
}

func TestExtractSafeMessage_ProductionWithWrappedSafeError(t *testing.T) {
	t.Parallel()

	inner := NewError("user not found", errors.New("sql: no rows"))
	outer := fmt.Errorf("rendering page: %w", inner)

	result := ExtractSafeMessage(outer, false)
	if result != "user not found" {
		t.Errorf("ExtractSafeMessage(prod, wrapped) = %q, want %q", result, "user not found")
	}
}

var _ Error = (*safeError)(nil)
