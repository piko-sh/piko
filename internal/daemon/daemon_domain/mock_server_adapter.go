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
	"sync/atomic"
)

// MockServerAdapter is a test double for ServerAdapter where nil function
// fields return zero values and call counts are tracked atomically.
type MockServerAdapter struct {
	// ListenAndServeFunc is the function called by
	// ListenAndServe.
	ListenAndServeFunc func(address string, handler http.Handler) error

	// ShutdownFunc is the function called by Shutdown.
	ShutdownFunc func(ctx context.Context) error

	// ListenAndServeCallCount tracks how many times
	// ListenAndServe was called.
	ListenAndServeCallCount int64

	// ShutdownCallCount tracks how many times Shutdown
	// was called.
	ShutdownCallCount int64
}

var _ ServerAdapter = (*MockServerAdapter)(nil)

// ListenAndServe starts the HTTP server on the given address.
//
// Takes address (string) which is the network address to listen on.
// Takes handler (http.Handler) which serves incoming HTTP requests.
//
// Returns error, or nil if ListenAndServeFunc is nil.
func (m *MockServerAdapter) ListenAndServe(address string, handler http.Handler) error {
	atomic.AddInt64(&m.ListenAndServeCallCount, 1)
	if m.ListenAndServeFunc != nil {
		return m.ListenAndServeFunc(address, handler)
	}
	return nil
}

// SetOnBound is a no-op for the mock adapter.
func (*MockServerAdapter) SetOnBound(_ func(address string)) {}

// Shutdown stops the service in a controlled way.
//
// Returns error, or nil if ShutdownFunc is nil.
func (m *MockServerAdapter) Shutdown(ctx context.Context) error {
	atomic.AddInt64(&m.ShutdownCallCount, 1)
	if m.ShutdownFunc != nil {
		return m.ShutdownFunc(ctx)
	}
	return nil
}
