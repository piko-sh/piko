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

package cache_domain

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// logKeyProviderName is the log attribute key for cache provider names.
	logKeyProviderName = "provider_name"
)

var (
	errProviderNameEmpty = errors.New("provider name cannot be empty")

	errProviderNil = errors.New("provider cannot be nil")
)

var _ Service = (*service)(nil)

// service implements the Service interface, managing a registry of cache
// providers and creating configured cache instances on demand.
type service struct {
	// providers maps provider names to their Provider instances.
	providers map[string]any

	// defaultProvider is the name of the provider to use when none is specified.
	defaultProvider string

	// mu guards access to providers and defaultProvider.
	mu sync.RWMutex
}

// RegisterProvider adds a cache provider implementation to the service
// registry.
//
// Takes ctx (context.Context) which carries logging context for trace and
// request ID propagation.
// Takes name (string) which identifies the provider in the registry.
// Takes provider (any) which must be a Provider instance.
//
// Returns error when name is empty, provider is nil, provider is not a Provider
// instance, or a provider with that name already exists.
//
// Safe for concurrent use.
func (s *service) RegisterProvider(ctx context.Context, name string, provider any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return errProviderNameEmpty
	}
	if provider == nil {
		return errProviderNil
	}
	if _, exists := s.providers[name]; exists {
		return fmt.Errorf("provider with name '%s' is already registered", name)
	}

	p, ok := provider.(Provider)
	if !ok {
		return errors.New("provider must implement the Provider interface")
	}

	_, l := logger_domain.From(ctx, log)
	l.Internal("Registered cache provider",
		logger_domain.String(logKeyProviderName, name))
	s.providers[name] = p

	return nil
}

// GetProvider retrieves a registered Provider by name.
//
// Takes name (string) which identifies the provider to retrieve.
//
// Returns Provider which is the requested provider instance.
// Returns error when the provider is not found.
//
// Safe for concurrent use.
func (s *service) GetProvider(name string) (Provider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	providerAny, ok := s.providers[name]
	if !ok {
		return nil, fmt.Errorf("%w: '%s'", ErrProviderNotFound, name)
	}

	provider, ok := providerAny.(Provider)
	if !ok {
		return nil, fmt.Errorf("provider '%s' is not a valid Provider instance", name)
	}

	return provider, nil
}

// GetProviders returns a sorted list of all registered provider names.
//
// Returns []string which contains the provider names in alphabetical order.
//
// Safe for concurrent use.
func (s *service) GetProviders() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.providers))
	for name := range s.providers {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// SetDefaultProvider sets the default cache provider to use when none is
// specified.
//
// Takes ctx (context.Context) which carries logging context for trace and
// request ID propagation.
// Takes name (string) which is the provider name to set as default.
//
// Returns error when the provider name is not registered.
//
// Safe for concurrent use.
func (s *service) SetDefaultProvider(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.providers[name]; !exists {
		return fmt.Errorf("%w: '%s' not registered", ErrProviderNotFound, name)
	}

	s.defaultProvider = name
	_, l := logger_domain.From(ctx, log)
	l.Internal("Set default cache provider",
		logger_domain.String(logKeyProviderName, name))
	return nil
}

// GetDefaultProvider returns the name of the current default provider.
//
// Returns string which is the default provider name.
//
// Safe for concurrent use.
func (s *service) GetDefaultProvider() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaultProvider
}

// Close shuts down all registered providers and releases their resources.
//
// Takes ctx (context.Context) which carries logging context for trace and
// request ID propagation.
//
// Returns error when one or more providers fail to close.
//
// Safe for concurrent use.
func (s *service) Close(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)

	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []error
	for name, providerAny := range s.providers {
		if provider, ok := providerAny.(Provider); ok {
			if err := provider.Close(); err != nil {
				errs = append(errs, fmt.Errorf("error closing provider '%s': %w", name, err))
				l.Error("Failed to close cache provider",
					logger_domain.String(logKeyProviderName, name),
					logger_domain.Error(err))
			} else {
				l.Internal("Closed cache provider", logger_domain.String(logKeyProviderName, name))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing providers: %w", errors.Join(errs...))
	}

	return nil
}

// Name returns the service identifier and implements the
// healthprobe_domain.Probe interface.
//
// Returns string which is the constant "CacheService".
func (*service) Name() string {
	return "CacheService"
}

// Check implements the healthprobe_domain.Probe interface.
// It verifies cache providers are available.
//
// Returns healthprobe_dto.Status which contains the health state and provider
// count.
//
// Safe for concurrent use.
func (s *service) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	s.mu.RLock()
	providerCount := len(s.providers)
	s.mu.RUnlock()

	state := healthprobe_dto.StateHealthy
	message := fmt.Sprintf("Cache service operational with %d provider(s)", providerCount)

	if providerCount == 0 {
		message = "No cache providers configured"
	}

	return healthprobe_dto.Status{
		Name:      s.Name(),
		State:     state,
		Message:   message,
		Timestamp: time.Now(),
		Duration:  time.Since(startTime).String(),
	}
}

// resolveProvider finds a provider by name, or uses the default if no name
// is given.
//
// Takes providerName (string) which is the name of the provider to find, or
// an empty string to use the default.
//
// Returns string which is the resolved provider name.
// Returns any which is the provider instance.
// Returns error when no provider name is given and no default is set, or when
// the named provider does not exist.
//
// Safe for concurrent use; holds a read lock while reading the provider map.
func (s *service) resolveProvider(providerName string) (string, any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if providerName == "" {
		providerName = s.defaultProvider
	}
	if providerName == "" {
		return "", nil, fmt.Errorf("%w: no default provider configured and no provider specified", ErrProviderNotFound)
	}

	providerAny, ok := s.providers[providerName]
	if !ok {
		return "", nil, fmt.Errorf("%w: '%s'", ErrProviderNotFound, providerName)
	}

	return providerName, providerAny, nil
}

// CreateNamespace creates a new cache instance using the namespace pattern
// and is a standalone function (rather than a method) because Go methods
// cannot have generic type parameters.
//
// Takes service (Service) which provides access to registered cache providers.
// Takes providerName (string) which identifies the cache provider to use.
// Takes namespace (string) which is the logical namespace for the cache
// instance.
// Takes options (cache_dto.Options[K, V]) which configures the cache
// behaviour.
//
// Returns Cache[K, V] which is the configured cache instance.
// Returns error when the provider is not found, options are invalid, or
// namespace creation fails.
func CreateNamespace[K comparable, V any](ctx context.Context, service Service, providerName, namespace string, options cache_dto.Options[K, V]) (Cache[K, V], error) {
	provider, err := service.GetProvider(providerName)
	if err != nil {
		return nil, fmt.Errorf("getting cache provider %q: %w", providerName, err)
	}

	if err := ValidateOptions(options); err != nil {
		return nil, fmt.Errorf("validating cache options for namespace %q: %w", namespace, err)
	}

	cacheAny, err := goroutine.SafeCall1(ctx, "cache.CreateNamespaceTyped", func() (any, error) {
		return provider.CreateNamespaceTyped(namespace, options)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace '%s' for provider '%s': %w", namespace, providerName, err)
	}

	cache, ok := cacheAny.(Cache[K, V])
	if !ok {
		return nil, fmt.Errorf("provider '%s' returned invalid cache type for namespace '%s'", providerName, namespace)
	}

	return cache, nil
}

// NewCache creates a new, configured cache instance by selecting the
// appropriate provider and returning a generic Cache interface, implemented
// as a standalone function because Go methods cannot have generic type
// parameters.
//
// Takes cacheService (Service) which provides access to registered
// cache providers.
// Takes options (cache_dto.Options[K, V]) which configures the cache
// instance.
//
// Returns Cache[K, V] which is the configured cache instance.
// Returns error when the service implementation is invalid, options
// validation fails, or provider creation fails.
func NewCache[K comparable, V any](cacheService Service, options cache_dto.Options[K, V]) (Cache[K, V], error) {
	s, ok := cacheService.(*service)
	if !ok {
		return nil, errors.New("invalid service implementation")
	}

	if err := ValidateOptions(options); err != nil {
		return nil, fmt.Errorf("validating cache options: %w", err)
	}

	providerName, providerAny, err := s.resolveProvider(options.Provider)
	if err != nil {
		return nil, fmt.Errorf("resolving cache provider: %w", err)
	}

	provider, ok := providerAny.(Provider)
	if !ok {
		return nil, fmt.Errorf("provider '%s' is not a valid Provider instance", providerName)
	}

	return createCacheFromProvider[K, V](provider, providerName, options)
}

// NewService creates a new cache service.
//
// Takes defaultProvider (string) which sets the provider to use when no
// provider is given in the cache options.
//
// Returns Service which is the configured cache service ready for use.
func NewService(defaultProvider string) Service {
	return &service{
		providers:       make(map[string]any),
		defaultProvider: defaultProvider,
	}
}

// createCacheFromProvider creates a cache using the new-style Provider
// interface.
//
// Takes provider (Provider) which is the cache provider implementation.
// Takes providerName (string) which identifies the provider for error
// messages.
// Takes options (cache_dto.Options[K, V]) which configures the cache
// instance.
//
// Returns Cache[K, V] which is the created cache instance.
// Returns error when namespace creation fails or the provider returns an
// invalid type.
func createCacheFromProvider[K comparable, V any](
	provider Provider,
	providerName string,
	options cache_dto.Options[K, V],
) (Cache[K, V], error) {
	namespace := cmp.Or(options.Namespace, "default")

	cacheAny, err := goroutine.SafeCall1(context.Background(), "cache.CreateNamespaceTyped", func() (any, error) {
		return provider.CreateNamespaceTyped(namespace, options)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace '%s' for provider '%s': %w", namespace, providerName, err)
	}

	cache, ok := cacheAny.(Cache[K, V])
	if !ok {
		return nil, fmt.Errorf("provider '%s' returned invalid cache type for namespace '%s'", providerName, namespace)
	}

	return cache, nil
}
