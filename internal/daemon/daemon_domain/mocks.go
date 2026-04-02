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
	"runtime"
	"sync"
	"sync/atomic"
)

// MockSignalNotifier is a test double for SignalNotifier where nil function
// fields use default behaviour and call counts are tracked atomically.
type MockSignalNotifier struct {
	// cancelFunc stores the cancel function set by NotifyContext
	// so that Trigger can cancel the derived context.
	cancelFunc atomic.Value

	// notifyContextCalled is closed on the first call to NotifyContext,
	// giving tests a channel-based wait instead of polling.
	notifyContextCalled chan struct{}

	// NotifyContextFunc is the function called by NotifyContext.
	NotifyContextFunc func(parent context.Context) (context.Context, context.CancelFunc)

	// NotifyContextCallCount tracks how many times NotifyContext was called.
	NotifyContextCallCount int64

	// triggered tracks whether Trigger was called, using
	// atomic compare-and-swap to ensure at-most-once semantics.
	triggered int64

	// notifyContextOnce ensures the notifyContextCalled channel is closed
	// exactly once, even from a zero-value MockSignalNotifier.
	notifyContextOnce sync.Once
}

// Compile-time interface check.
var _ SignalNotifier = (*MockSignalNotifier)(nil)

// NewMockSignalNotifier creates a new MockSignalNotifier for testing.
//
// Returns *MockSignalNotifier which is the initialised mock.
func NewMockSignalNotifier() *MockSignalNotifier {
	return &MockSignalNotifier{
		notifyContextCalled: make(chan struct{}),
	}
}

// NotifyContext returns a context that can be cancelled by calling Trigger.
// If NotifyContextFunc is set, it delegates to that function instead.
//
// Takes parent (context.Context) which is the parent context to derive from.
//
// Returns context.Context which is the derived context that will be cancelled.
// Returns context.CancelFunc which cancels the returned context.
func (n *MockSignalNotifier) NotifyContext(parent context.Context) (context.Context, context.CancelFunc) {
	atomic.AddInt64(&n.NotifyContextCallCount, 1)
	n.notifyContextOnce.Do(func() {
		if n.notifyContextCalled != nil {
			close(n.notifyContextCalled)
		}
	})

	if n.NotifyContextFunc != nil {
		return n.NotifyContextFunc(parent)
	}

	ctx, cancel := context.WithCancel(parent)
	n.cancelFunc.Store(cancel)
	return ctx, cancel
}

// Trigger simulates receiving a shutdown signal by cancelling the context. Use
// it in tests to trigger graceful shutdown without real OS signals.
func (n *MockSignalNotifier) Trigger() {
	if atomic.CompareAndSwapInt64(&n.triggered, 0, 1) {
		if cancelFunction, ok := n.cancelFunc.Load().(context.CancelFunc); ok && cancelFunction != nil {
			cancelFunction()
		}
	}
}

// WasTriggered returns whether Trigger was called.
//
// Returns bool which is true if Trigger was called at least once.
func (n *MockSignalNotifier) WasTriggered() bool {
	return atomic.LoadInt64(&n.triggered) == 1
}

// AwaitNotifyContext returns a channel that is closed when NotifyContext is
// called for the first time. Tests should select on this instead of polling
// NotifyContextCalled, eliminating wall-clock timing dependencies.
//
// Returns <-chan struct{} which is closed once NotifyContext has been called.
func (n *MockSignalNotifier) AwaitNotifyContext() <-chan struct{} {
	if n.notifyContextCalled == nil {
		// Zero-value MockSignalNotifier: fall back to a polling check.
		ch := make(chan struct{})
		go func() {
			for !n.NotifyContextCalled() {
				runtime.Gosched()
			}
			close(ch)
		}()
		return ch
	}
	return n.notifyContextCalled
}

// Reset clears the mock state so it can be reused in tests.
func (n *MockSignalNotifier) Reset() {
	atomic.StoreInt64(&n.triggered, 0)
	n.cancelFunc.Store(context.CancelFunc(nil))
	atomic.StoreInt64(&n.NotifyContextCallCount, 0)
	n.notifyContextCalled = make(chan struct{})
	n.notifyContextOnce = sync.Once{}
}

// NotifyContextCalled returns whether NotifyContext has been called at least once.
//
// Returns bool which is true if NotifyContext was called at least once.
func (n *MockSignalNotifier) NotifyContextCalled() bool {
	return atomic.LoadInt64(&n.NotifyContextCallCount) > 0
}
