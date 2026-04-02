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

package provider_otter

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/provider/provider_domain"
)

// OtterProvider implements the cache.Provider interface for Otter in-memory
// caches. Unlike connection-pooled providers, Otter creates an independent
// cache instance per namespace since there are no shared resources.
type OtterProvider struct {
	// namespaces stores all created cache instances, keyed by namespace name.
	namespaces map[string]any

	// mu guards access to the namespaces map.
	mu sync.RWMutex
}

var _ cache_domain.Provider = (*OtterProvider)(nil)
var _ provider_domain.ProviderMetadata = (*OtterProvider)(nil)

// NewOtterProvider creates a new Otter cache provider.
// Otter is an in-memory cache and needs no global settings.
//
// Returns *OtterProvider which is ready for use with namespace registration.
func NewOtterProvider() *OtterProvider {
	return &OtterProvider{
		namespaces: make(map[string]any),
		mu:         sync.RWMutex{},
	}
}

// GetProviderType returns the provider implementation type.
//
// Returns string which is "otter".
func (*OtterProvider) GetProviderType() string {
	return "otter"
}

// GetProviderMetadata returns metadata about the Otter cache provider.
//
// Returns map[string]any which describes the provider capabilities.
//
// Safe for concurrent use.
func (p *OtterProvider) GetProviderMetadata() map[string]any {
	p.mu.RLock()
	n := len(p.namespaces)
	p.mu.RUnlock()

	return map[string]any{
		"backend":    "otter (in-memory)",
		"namespaces": n,
	}
}

// CreateNamespaceTyped creates a new Otter cache instance for the given
// namespace, or returns an existing one if already created.
//
// Each namespace gets its own independent Otter cache with no shared resources.
// This is a non-generic method that uses type erasure; call via
// CreateNamespace[K,V]() for type safety.
//
// Takes namespace (string) which identifies the cache; defaults to "default"
// if empty.
// Takes options (any) which configures the cache behaviour.
//
// Returns any which is the cache instance for the namespace.
// Returns error when the cache cannot be created.
//
// Safe for concurrent use; uses a mutex to protect namespace registration.
func (p *OtterProvider) CreateNamespaceTyped(namespace string, options any) (any, error) {
	if namespace == "" {
		namespace = "default"
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if existing, exists := p.namespaces[namespace]; exists {
		_, l := logger_domain.From(context.Background(), log)
		l.Internal("Reusing existing Otter namespace",
			logger_domain.String("namespace", namespace))
		return existing, nil
	}

	cache, err := createOtterCacheGeneric(context.Background(), namespace, options)
	if err != nil {
		return nil, fmt.Errorf("creating otter cache for namespace %q: %w", namespace, err)
	}

	p.namespaces[namespace] = cache

	return cache, nil
}

// ListNamespaces returns a snapshot of the namespaces map. The values are
// type-erased cache instances that may implement EstimatedSize() int.
//
// Returns map[string]any which maps namespace names to cache instances.
//
// Safe for concurrent use.
func (p *OtterProvider) ListNamespaces() map[string]any {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]any, len(p.namespaces))
	maps.Copy(result, p.namespaces)

	return result
}

// Close releases all resources managed by this provider.
// For Otter, this closes all cache instances.
//
// Returns error when closing fails, though currently always returns nil.
//
// Safe for concurrent use; holds the provider's mutex during cleanup.
func (p *OtterProvider) Close() error {
	_, l := logger_domain.From(context.Background(), log)

	p.mu.Lock()
	defer p.mu.Unlock()

	for namespace, cacheAny := range p.namespaces {
		if closer, ok := cacheAny.(interface{ Close() }); ok {
			closer.Close()
			l.Internal("Closed Otter namespace", logger_domain.String("namespace", namespace))
		}
	}

	p.namespaces = make(map[string]any)

	l.Internal("Closed Otter provider")
	return nil
}

// Name returns the unique name identifying this cache provider.
//
// Returns string which is the provider identifier.
func (*OtterProvider) Name() string {
	return "otter"
}

// Check implements the healthprobe_domain.Probe interface, returning the
// health status for the specified check type.
//
// When checkType is liveness, returns healthy since Otter is in-memory.
// When checkType is readiness, verifies the namespaces map is accessible.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies whether to
// perform a liveness or readiness check.
//
// Returns healthprobe_dto.Status which contains the health state and details.
//
// Safe for concurrent use; uses a read lock when accessing namespaces.
func (p *OtterProvider) Check(_ context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	if checkType == healthprobe_dto.CheckTypeLiveness {
		return healthprobe_dto.Status{
			Name:      p.Name(),
			State:     healthprobe_dto.StateHealthy,
			Message:   "Otter in-memory cache provider operational",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime).String(),
		}
	}

	p.mu.RLock()
	namespaceCount := len(p.namespaces)
	p.mu.RUnlock()

	return healthprobe_dto.Status{
		Name:      p.Name(),
		State:     healthprobe_dto.StateHealthy,
		Message:   fmt.Sprintf("Otter cache operational with %d namespace(s)", namespaceCount),
		Timestamp: time.Now(),
		Duration:  time.Since(startTime).String(),
	}
}
