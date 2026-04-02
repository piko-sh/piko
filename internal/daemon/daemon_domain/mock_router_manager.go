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

	"piko.sh/piko/internal/templater/templater_domain"
)

// MockRouterManager is a test double for RouterManager where nil function
// fields return zero values and call counts are tracked atomically.
type MockRouterManager struct {
	// ReloadRoutesFunc is the function called by
	// ReloadRoutes.
	ReloadRoutesFunc func(ctx context.Context, store templater_domain.ManifestStoreView) error

	// ServeHTTPFunc is the function called by ServeHTTP.
	ServeHTTPFunc func(w http.ResponseWriter, r *http.Request)

	// ReloadRoutesCallCount tracks how many times
	// ReloadRoutes was called.
	ReloadRoutesCallCount int64

	// ServeHTTPCallCount tracks how many times ServeHTTP
	// was called.
	ServeHTTPCallCount int64
}

var _ RouterManager = (*MockRouterManager)(nil)

// ReloadRoutes refreshes the routing configuration from the manifest store.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes store (templater_domain.ManifestStoreView) which
// provides the route definitions.
//
// Returns error, or nil if ReloadRoutesFunc is nil.
func (m *MockRouterManager) ReloadRoutes(ctx context.Context, store templater_domain.ManifestStoreView) error {
	atomic.AddInt64(&m.ReloadRoutesCallCount, 1)
	if m.ReloadRoutesFunc != nil {
		return m.ReloadRoutesFunc(ctx, store)
	}
	return nil
}

// ServeHTTP handles an HTTP request.
//
// Takes w (http.ResponseWriter) which is the response writer.
// Takes r (*http.Request) which is the incoming HTTP request.
func (m *MockRouterManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&m.ServeHTTPCallCount, 1)
	if m.ServeHTTPFunc != nil {
		m.ServeHTTPFunc(w, r)
	}
}
