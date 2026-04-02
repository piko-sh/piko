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

package provider_mock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

// MockProvider is a test implementation of the cache.Provider interface.
// It creates simple mock caches for each namespace.
type MockProvider struct {
	// namespaces maps namespace names to their MockAdapter instances.
	namespaces map[string]any

	// mu guards access to the provider's mutable state.
	mu sync.RWMutex

	// closed indicates whether the provider has been shut down.
	closed bool
}

var _ cache_domain.Provider = (*MockProvider)(nil)

// NewMockProvider creates a new mock provider for testing.
//
// Returns *MockProvider which is ready for use in tests.
func NewMockProvider() *MockProvider {
	return &MockProvider{
		namespaces: make(map[string]any),
		mu:         sync.RWMutex{},
		closed:     false,
	}
}

// CreateNamespaceTyped creates a new mock cache instance for the given
// namespace. This is a non-generic method that uses type erasure; call via
// CreateNamespace[K,V]() for type safety.
//
// Takes namespace (string) which identifies the cache namespace.
// Takes options (any) which provides type-erased configuration options.
//
// Returns any which is the created mock cache instance.
// Returns error when cache creation fails.
func (p *MockProvider) CreateNamespaceTyped(namespace string, options any) (any, error) {
	return createMockCache(p, namespace, options)
}

// Close releases all resources managed by this provider.
//
// Returns error when the provider cannot be closed.
//
// Safe for concurrent use.
func (p *MockProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.closed = true
	p.namespaces = make(map[string]any)

	return nil
}

// Name returns the provider's identifier.
//
// Returns string which is the constant "mock".
func (*MockProvider) Name() string {
	return "mock"
}

// Check implements the healthprobe_domain.Probe interface.
// Reports unhealthy if the provider has been closed, otherwise healthy.
//
// Returns healthprobe_dto.Status which contains the health state and details.
//
// Safe for concurrent use.
func (p *MockProvider) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	p.mu.RLock()
	isClosed := p.closed
	namespaceCount := len(p.namespaces)
	p.mu.RUnlock()

	if isClosed {
		return healthprobe_dto.Status{
			Name:      p.Name(),
			State:     healthprobe_dto.StateUnhealthy,
			Message:   "Mock cache provider is closed",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime).String(),
		}
	}

	return healthprobe_dto.Status{
		Name:      p.Name(),
		State:     healthprobe_dto.StateHealthy,
		Message:   fmt.Sprintf("Mock cache provider operational with %d namespace(s)", namespaceCount),
		Timestamp: time.Now(),
		Duration:  time.Since(startTime).String(),
	}
}

// IsClosed returns whether the provider has been closed (for testing).
//
// Returns bool which is true if the provider has been closed.
//
// Safe for concurrent use.
func (p *MockProvider) IsClosed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.closed
}
