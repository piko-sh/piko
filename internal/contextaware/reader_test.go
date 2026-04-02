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

package contextaware_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"piko.sh/piko/internal/contextaware"
)

func TestNewReader_DelegatesToUnderlying(t *testing.T) {
	data := []byte("hello world")
	r := contextaware.NewReader(context.Background(), bytes.NewReader(data))

	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("got %q, want %q", got, data)
	}
}

func TestNewReader_ReturnsCancelledContextError(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: cancelled"))

	r := contextaware.NewReader(ctx, bytes.NewReader([]byte("data")))

	buffer := make([]byte, 4)
	n, err := r.Read(buffer)
	if n != 0 {
		t.Fatalf("expected 0 bytes, got %d", n)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestNewReader_ReturnsDeadlineExceededError(t *testing.T) {
	ctx, cancel := context.WithTimeoutCause(context.Background(), 0,
		fmt.Errorf("test: deadline exceeded"))
	defer cancel()

	r := contextaware.NewReader(ctx, bytes.NewReader([]byte("data")))

	buffer := make([]byte, 4)
	n, err := r.Read(buffer)
	if n != 0 {
		t.Fatalf("expected 0 bytes, got %d", n)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestNewReader_ActiveContextReadsNormally(t *testing.T) {
	data := []byte("abcdef")
	r := contextaware.NewReader(context.Background(), bytes.NewReader(data))

	buffer := make([]byte, 3)
	n, err := r.Read(buffer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 3 {
		t.Fatalf("expected 3 bytes, got %d", n)
	}
	if !bytes.Equal(buffer[:n], []byte("abc")) {
		t.Fatalf("got %q, want %q", buffer[:n], "abc")
	}
}
