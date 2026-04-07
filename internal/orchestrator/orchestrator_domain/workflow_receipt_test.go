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

package orchestrator_domain

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewWorkflowReceipt(t *testing.T) {
	t.Parallel()

	workflowID := "workflow-123"
	receipt := newWorkflowReceipt(workflowID)

	require.NotNil(t, receipt, "newWorkflowReceipt returned nil")
	if receipt.WorkflowID != workflowID {
		t.Errorf("WorkflowID: expected %s, got %s", workflowID, receipt.WorkflowID)
	}
	if receipt.doneCh == nil {
		t.Error("doneCh should be initialised")
	}
}

func TestWorkflowReceipt_Done_ReturnsChannel(t *testing.T) {
	t.Parallel()

	receipt := newWorkflowReceipt("workflow-123")
	doneChannel := receipt.Done()

	if doneChannel == nil {
		t.Error("Done() returned nil channel")
	}
}

func TestWorkflowReceipt_Resolve_WithNilError(t *testing.T) {
	t.Parallel()

	receipt := newWorkflowReceipt("workflow-123")
	receipt.resolve(nil)

	select {
	case err, ok := <-receipt.Done():
		if ok {
			t.Error("channel should be closed (ok should be false)")
		}
		if err != nil {
			t.Errorf("error should be nil, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timed out waiting for channel to close")
	}
}

func TestWorkflowReceipt_Resolve_WithError(t *testing.T) {
	t.Parallel()

	receipt := newWorkflowReceipt("workflow-123")
	expectedErr := errors.New("workflow failed")
	receipt.resolve(expectedErr)

	select {
	case err := <-receipt.Done():
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timed out waiting for error")
	}

	select {
	case _, ok := <-receipt.Done():
		if ok {
			t.Error("channel should be closed after error is received")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timed out waiting for channel close")
	}
}

func TestWorkflowReceipt_Resolve_IsIdempotent(t *testing.T) {
	t.Parallel()

	receipt := newWorkflowReceipt("workflow-123")
	err1 := errors.New("first error")
	err2 := errors.New("second error")

	receipt.resolve(err1)
	receipt.resolve(err2)
	receipt.resolve(nil)

	select {
	case err := <-receipt.Done():
		if !errors.Is(err, err1) {
			t.Errorf("expected first error %v, got %v", err1, err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timed out waiting for error")
	}
}

func TestWorkflowReceipt_Wait_ReturnsOnSuccess(t *testing.T) {
	t.Parallel()

	receipt := newWorkflowReceipt("workflow-123")

	go func() {
		time.Sleep(10 * time.Millisecond)
		receipt.resolve(nil)
	}()

	err := receipt.Wait(context.Background())

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestWorkflowReceipt_Wait_ReturnsError(t *testing.T) {
	t.Parallel()

	receipt := newWorkflowReceipt("workflow-123")
	expectedErr := errors.New("task failed")

	go func() {
		time.Sleep(10 * time.Millisecond)
		receipt.resolve(expectedErr)
	}()

	err := receipt.Wait(context.Background())

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestWorkflowReceipt_Wait_RespectsContextCancellation(t *testing.T) {
	t.Parallel()

	receipt := newWorkflowReceipt("workflow-123")
	ctx, cancel := context.WithCancelCause(context.Background())

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel(fmt.Errorf("test: simulating cancelled context"))
	}()

	err := receipt.Wait(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestWorkflowReceipt_Wait_RespectsContextDeadline(t *testing.T) {
	t.Parallel()

	receipt := newWorkflowReceipt("workflow-123")
	ctx, cancel := context.WithTimeoutCause(context.Background(), 10*time.Millisecond, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	err := receipt.Wait(ctx)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestWorkflowReceipt_Wait_ImmediateIfAlreadyResolved(t *testing.T) {
	t.Parallel()

	receipt := newWorkflowReceipt("workflow-123")
	expectedErr := errors.New("pre-resolved error")
	receipt.resolve(expectedErr)

	start := time.Now()
	err := receipt.Wait(context.Background())
	duration := time.Since(start)

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if duration > 10*time.Millisecond {
		t.Errorf("Wait took too long for pre-resolved receipt: %v", duration)
	}
}

func TestWorkflowReceipt_ConcurrentResolve(t *testing.T) {
	t.Parallel()

	receipt := newWorkflowReceipt("workflow-123")

	done := make(chan struct{})
	for i := range 10 {
		go func(i int) {
			receipt.resolve(errors.New("error"))
			done <- struct{}{}
		}(i)
	}

	for range 10 {
		<-done
	}

	select {
	case _, ok := <-receipt.Done():
		_ = ok
	case <-time.After(100 * time.Millisecond):
		t.Error("timed out")
	}
}

func TestWorkflowReceipt_ConcurrentWait(t *testing.T) {
	t.Parallel()

	receipt := newWorkflowReceipt("workflow-123")
	expectedErr := errors.New("concurrent error")

	results := make(chan error, 5)
	for range 5 {
		go func() {
			results <- receipt.Wait(context.Background())
		}()
	}

	time.Sleep(10 * time.Millisecond)

	receipt.resolve(expectedErr)

	errorCount := 0
	nilCount := 0
	for i := range 5 {
		select {
		case err := <-results:
			if errors.Is(err, expectedErr) {
				errorCount++
			} else if err == nil {
				nilCount++
			} else {
				t.Errorf("waiter %d: unexpected error %v", i, err)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("waiter %d timed out", i)
		}
	}

	if errorCount != 1 {
		t.Errorf("expected exactly 1 waiter to receive the error, got %d", errorCount)
	}
	if nilCount != 4 {
		t.Errorf("expected 4 waiters to receive nil, got %d", nilCount)
	}
}
