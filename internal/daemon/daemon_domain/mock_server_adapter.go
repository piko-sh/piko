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
	"net/http"
	"sync"
	"sync/atomic"
)

// MockServerAdapter is a test double for ServerAdapter where nil function fields return
// zero values and call counts are tracked atomically.
type MockServerAdapter struct {
	// ListenAndServeFunc is the function called by ListenAndServe when no context-aware
	// function is set.
	ListenAndServeFunc func(address string, handler http.Handler) error

	// ListenAndServeCtxFunc is the function called by ListenAndServe, and takes precedence
	// over ListenAndServeFunc.
	ListenAndServeCtxFunc func(ctx context.Context, address string, handler http.Handler) error

	// ShutdownFunc is the function called by Shutdown.
	ShutdownFunc func(ctx context.Context) error

	// boundChan closes when the mock listener is treated as bound.
	boundChan chan struct{}

	// ListenAndServeCallCount tracks how many times ListenAndServe was called.
	ListenAndServeCallCount atomic.Int64

	// ShutdownCallCount tracks how many times Shutdown was called.
	ShutdownCallCount atomic.Int64

	// boundMu guards boundChan and boundClosed.
	boundMu sync.Mutex

	// NeverBinds makes the mock behave like a server whose bind failed, so Bound never
	// closes. It is how a test drives the readiness gate.
	NeverBinds bool

	// boundClosed records that boundChan has been closed, so closing is idempotent.
	boundClosed bool
}

var (
	_ ServerAdapter = (*MockServerAdapter)(nil)
)

// ListenAndServe starts the HTTP server on the given address.
//
// Takes address (string) which is the network address to listen on.
// Takes handler (http.Handler) which serves incoming HTTP requests.
//
// Returns error, or nil when neither function is set.
func (m *MockServerAdapter) ListenAndServe(ctx context.Context, address string, handler http.Handler) error {
	m.ListenAndServeCallCount.Add(1)

	if !m.NeverBinds {
		m.markBound()
	}

	if m.ListenAndServeCtxFunc != nil {
		return m.ListenAndServeCtxFunc(ctx, address, handler)
	}
	if m.ListenAndServeFunc != nil {
		return m.ListenAndServeFunc(address, handler)
	}

	return nil
}

// Bound reports when the mock listener is bound.
//
// Returns <-chan struct{} which closes once ListenAndServe has been called, unless
// NeverBinds is set.
//
// Concurrency: safe for concurrent use; the channel is created under the mutex.
func (m *MockServerAdapter) Bound() <-chan struct{} {
	m.boundMu.Lock()
	defer m.boundMu.Unlock()

	return m.ensureBoundChanLocked()
}

// SetOnBound is a no-op for the mock adapter.
func (*MockServerAdapter) SetOnBound(_ func(address string)) {}

// Shutdown stops the service in a controlled way.
//
// Returns error, or nil if ShutdownFunc is nil.
func (m *MockServerAdapter) Shutdown(ctx context.Context) error {
	m.ShutdownCallCount.Add(1)
	if m.ShutdownFunc != nil {
		return m.ShutdownFunc(ctx)
	}
	return nil
}

// markBound signals that the mock listener is accepting connections.
//
// Closing is idempotent because a test may drive more than one bind.
//
// Concurrency: safe for concurrent use; guarded by the mutex.
func (m *MockServerAdapter) markBound() {
	m.boundMu.Lock()
	defer m.boundMu.Unlock()

	channel := m.ensureBoundChanLocked()
	if m.boundClosed {
		return
	}
	m.boundClosed = true
	close(channel)
}

// ensureBoundChanLocked returns the bound channel, creating it when Bound has not yet
// been called. The caller must hold the mutex.
//
// Returns chan struct{} which closes once the mock listener binds.
func (m *MockServerAdapter) ensureBoundChanLocked() chan struct{} {
	if m.boundChan == nil {
		m.boundChan = make(chan struct{})
	}

	return m.boundChan
}
