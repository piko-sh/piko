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

package storage_domain

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/shutdown"
)

const (
	// logFieldTransformer is the log field key for the transformer name.
	logFieldTransformer = "transformer"

	// logFieldTransformerType is the log field key for the transformer type.
	logFieldTransformerType = "type"

	// logFieldPriority is the log field key for transformer priority.
	logFieldPriority = "priority"
)

var (
	errProviderNameEmpty = errors.New("provider name cannot be empty")

	errProviderNil = errors.New("provider cannot be nil")
)

// RegisterProvider adds a storage provider with the given name.
// It wraps the provider with retry and circuit breaker decorators for
// resilience, and registers the provider for graceful shutdown.
//
// Takes ctx (context.Context) for cancellation and logging propagation.
// Takes name (string) which identifies the provider for later retrieval.
// Takes provider (StorageProviderPort) which is the storage backend to
// register.
//
// Returns error when the name is empty, the provider is nil, or a provider
// with the same name is already registered.
//
// Safe for concurrent use. The default provider must be set separately with
// SetDefaultProvider.
func (s *service) RegisterProvider(ctx context.Context, name string, provider StorageProviderPort) error {
	if name == "" {
		return errProviderNameEmpty
	}
	if provider == nil {
		return errProviderNil
	}

	s.mu.Lock()
	var wrappedProvider StorageProviderPort = NewTransformerWrapper(provider, s.transformerRegistry, nil, name)

	shouldWrapRetry := s.config.EnableRetry && !provider.SupportsRetry()
	if shouldWrapRetry {
		wrappedProvider = newRetryableOperation(s.config.RetryConfig, wrappedProvider, name, s.clock)
	}

	shouldWrapCircuitBreaker := s.config.EnableCircuitBreaker && !provider.SupportsCircuitBreaking()
	if shouldWrapCircuitBreaker {
		wrappedProvider = NewCircuitBreakerWrapper(ctx, wrappedProvider, s.config.CircuitBreakerConfig, name)
	}
	s.mu.Unlock()

	if err := s.registry.RegisterProvider(ctx, name, wrappedProvider); err != nil {
		return fmt.Errorf("registering storage provider %q: %w", name, err)
	}

	shutdownName := fmt.Sprintf("storage-provider-%s", name)
	shutdown.Register(ctx, shutdownName, func(ctx context.Context) error {
		ctx, l := logger_domain.From(ctx, log)
		l.Internal("Closing storage provider", logger_domain.String(logFieldProvider, name))
		return provider.Close(ctx)
	})

	_, l := logger_domain.From(ctx, log)
	l.Internal("Registered storage provider with resilience layers",
		logger_domain.String(logFieldProvider, name),
		logger_domain.Bool("retry_wrapped", shouldWrapRetry),
		logger_domain.Bool("retry_provider_native", provider.SupportsRetry()),
		logger_domain.Bool("circuit_breaker_wrapped", shouldWrapCircuitBreaker),
		logger_domain.Bool("circuit_breaker_provider_native", provider.SupportsCircuitBreaking()))

	return nil
}

// SetDefaultProvider sets the provider to use when no specific provider is
// named in a call.
//
// Takes name (string) which identifies the provider to set as default.
//
// Returns error when the named provider does not exist.
func (s *service) SetDefaultProvider(name string) error {
	if err := s.registry.SetDefaultProvider(context.Background(), name); err != nil {
		return fmt.Errorf("setting default storage provider to %q: %w", name, err)
	}
	return nil
}

// GetProviders returns a sorted list of all registered provider names.
//
// Takes ctx (context.Context) for cancellation and logging propagation.
//
// Returns []string which contains the provider names in alphabetical order.
func (s *service) GetProviders(ctx context.Context) []string {
	providers := s.registry.ListProviders(ctx)
	names := make([]string, 0, len(providers))
	for _, p := range providers {
		names = append(names, p.Name)
	}
	slices.Sort(names)
	return names
}

// HasProvider checks if a provider with the given name has been registered.
//
// Takes name (string) which specifies the provider name to look up.
//
// Returns bool which is true if the provider exists, false otherwise.
func (s *service) HasProvider(name string) bool {
	return s.registry.HasProvider(name)
}

// ListProviders returns detailed information about all registered providers.
//
// Returns []provider_domain.ProviderInfo which contains provider metadata,
// health status, and capabilities.
func (s *service) ListProviders(ctx context.Context) []provider_domain.ProviderInfo {
	return s.registry.ListProviders(ctx)
}

// getProvider retrieves a storage provider by name.
//
// Takes name (string) which specifies the provider to retrieve. If empty, the
// default provider is used.
//
// Returns StorageProviderPort which is the requested storage provider.
// Returns error when no provider is specified and no default is set, or when
// the named provider does not exist.
func (s *service) getProvider(ctx context.Context, name string) (StorageProviderPort, error) {
	providerName := name
	if providerName == "" {
		providerName = s.registry.GetDefaultProvider()
	}
	if providerName == "" {
		return nil, errors.New("no default provider configured and no provider specified")
	}
	return s.registry.GetProvider(ctx, providerName)
}

// RegisterTransformer adds a stream transformer to the service registry.
//
// Takes ctx (context.Context) for cancellation and logging propagation.
// Takes transformer (StreamTransformerPort) which is the transformer to add.
//
// Returns error when transformer is nil or registration fails.
//
// Safe for concurrent use.
func (s *service) RegisterTransformer(ctx context.Context, transformer StreamTransformerPort) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if transformer == nil {
		return errTransformerNil
	}

	if err := s.transformerRegistry.Register(transformer); err != nil {
		return fmt.Errorf("failed to register transformer: %w", err)
	}

	_, l := logger_domain.From(ctx, log)
	l.Internal("Registered stream transformer",
		logger_domain.String(logFieldTransformer, transformer.Name()),
		logger_domain.String(logFieldTransformerType, string(transformer.Type())),
		logger_domain.Int(logFieldPriority, transformer.Priority()))

	return nil
}

// GetTransformers returns a sorted list of all registered transformer names.
//
// Returns []string which contains the names of all registered transformers.
//
// Safe for concurrent use.
func (s *service) GetTransformers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.transformerRegistry.GetNames()
}

// HasTransformer checks if a transformer with the given name has been
// registered.
//
// Takes name (string) which is the transformer name to look up.
//
// Returns bool which is true if the transformer exists, false otherwise.
//
// Safe for concurrent use.
func (s *service) HasTransformer(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.transformerRegistry.Has(name)
}

// RegisterDispatcher registers and starts a storage dispatcher with the service.
//
// Takes ctx (context.Context) for cancellation and logging propagation.
// Takes dispatcher (StorageDispatcherPort) which is the dispatcher to register
// and start.
//
// Returns error when dispatcher is nil or fails to start.
//
// Safe for concurrent use.
func (s *service) RegisterDispatcher(ctx context.Context, dispatcher StorageDispatcherPort) error {
	if dispatcher == nil {
		return errDispatcherNil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.dispatcher = dispatcher

	ctx, l := logger_domain.From(ctx, log)
	if err := dispatcher.Start(ctx); err != nil {
		return fmt.Errorf("failed to start storage dispatcher: %w", err)
	}

	l.Internal("Storage dispatcher registered and started")
	return nil
}

// FlushDispatcher forces a flush of all queued operations in the dispatcher.
//
// Returns error when no dispatcher is registered or the flush fails.
//
// Safe for concurrent use. Uses a read lock to access the dispatcher.
func (s *service) FlushDispatcher(ctx context.Context) error {
	s.mu.RLock()
	dispatcher := s.dispatcher
	s.mu.RUnlock()

	if dispatcher == nil {
		return errNoDispatcher
	}

	if err := dispatcher.Flush(ctx); err != nil {
		return fmt.Errorf("flushing storage dispatcher: %w", err)
	}
	return nil
}

// GetStats returns a snapshot of current service statistics.
//
// Returns ServiceStats which contains the current operation counts and timing
// data.
func (s *service) GetStats(_ context.Context) ServiceStats {
	stats := ServiceStats{
		StartTime:            s.stats.StartTime,
		TotalOperations:      atomic.LoadInt64(&s.stats.TotalOperations),
		SuccessfulOperations: atomic.LoadInt64(&s.stats.SuccessfulOperations),
		FailedOperations:     atomic.LoadInt64(&s.stats.FailedOperations),
		RetryAttempts:        atomic.LoadInt64(&s.stats.RetryAttempts),
		CacheHits:            atomic.LoadInt64(&s.stats.CacheHits),
		CacheMisses:          atomic.LoadInt64(&s.stats.CacheMisses),
		DLQEntries:           atomic.LoadInt64(&s.stats.DLQEntries),
	}

	return stats
}
