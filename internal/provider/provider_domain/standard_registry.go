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

package provider_domain

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
)

// errProviderNameEmpty is returned when a provider is registered with an
// empty name.
var errProviderNameEmpty = errors.New("provider name cannot be empty")

// StandardRegistry is the production implementation of ProviderRegistry.
// Thread-safe with auto-cleanup on shutdown.
//
// Generic type T is the provider interface (EmailProviderPort,
// StorageProviderPort, etc.)
//
// Features: - Thread-safe concurrent access (read-write mutex) - Graceful
// shutdown with provider cleanup - Provider discovery with metadata
type StandardRegistry[T any] struct {
	// providers maps provider names to their registration info.
	providers map[string]*registrationInfo[T]

	// defaultName is the fallback name used when no name is provided.
	defaultName string

	// serviceName identifies the service for logging purposes (e.g., "email",
	// "storage").
	serviceName string

	// mu guards concurrent access to the registry.
	mu sync.RWMutex
}

// registrationInfo holds a provider instance and the time it was registered.
type registrationInfo[T any] struct {
	// provider is the underlying value that implements the registered interface.
	provider T

	// registeredAt is when the path was registered.
	registeredAt time.Time
}

// RegisterProvider registers a new provider under the given name. It
// implements ProviderRegistry.RegisterProvider.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes name (string) which is the unique identifier for the provider.
// Takes provider (T) which is the provider instance to register.
//
// Returns error when the name is empty or a provider with that name already
// exists.
//
// Safe for concurrent use; protected by a mutex.
func (r *StandardRegistry[T]) RegisterProvider(ctx context.Context, name string, provider T) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if name == "" {
		return errProviderNameEmpty
	}

	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %q already registered", name)
	}

	r.providers[name] = &registrationInfo[T]{
		provider:     provider,
		registeredAt: time.Now(),
	}

	_, l := logger_domain.From(ctx, log)
	l.Internal("Provider registered",
		logger_domain.String("service", r.serviceName),
		logger_domain.String("provider_name", name))

	return nil
}

// SetDefaultProvider sets the named provider as the default for this registry.
// Implements ProviderRegistry.SetDefaultProvider.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes name (string) which identifies the provider to set as default.
//
// Returns error when the named provider is not found in the registry.
//
// Safe for concurrent use; protected by a mutex.
func (r *StandardRegistry[T]) SetDefaultProvider(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; !exists {
		return fmt.Errorf("%s provider %q not found", r.serviceName, name)
	}

	r.defaultName = name

	_, l := logger_domain.From(ctx, log)
	l.Internal("Default provider set",
		logger_domain.String("service", r.serviceName),
		logger_domain.String("provider_name", name))

	return nil
}

// GetDefaultProvider returns the name of the default provider. Implements
// ProviderRegistry.GetDefaultProvider.
//
// Returns string which is the name of the default provider, or empty if none
// is set.
//
// Safe for concurrent use.
func (r *StandardRegistry[T]) GetDefaultProvider() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaultName
}

// GetProvider retrieves a provider by name from the registry. Implements
// ProviderRegistry.GetProvider.
//
// Takes name (string) which specifies the provider to retrieve.
//
// Returns T which is the requested provider instance.
// Returns error when the named provider is not registered.
//
// Safe for concurrent use; uses a read lock to access the provider map.
func (r *StandardRegistry[T]) GetProvider(_ context.Context, name string) (T, error) {
	r.mu.RLock()
	info, exists := r.providers[name]
	r.mu.RUnlock()

	if !exists {
		var zero T
		return zero, fmt.Errorf("%s provider %q not found", r.serviceName, name)
	}

	return info.provider, nil
}

// HasProvider reports whether a provider with the given name is registered.
// Implements ProviderRegistry.HasProvider.
//
// Takes name (string) which is the provider name to check.
//
// Returns bool which is true if the provider exists, false otherwise.
//
// Safe for concurrent use; protected by a read lock.
func (r *StandardRegistry[T]) HasProvider(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.providers[name]
	return exists
}

// ListProviders returns information about all registered providers.
// Implements ProviderRegistry.ListProviders.
//
// Returns []ProviderInfo which contains metadata for each registered provider.
//
// Safe for concurrent use. Protected by a read lock.
func (r *StandardRegistry[T]) ListProviders(_ context.Context) []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ProviderInfo, 0, len(r.providers))

	for name, info := range r.providers {
		providerInfo := ProviderInfo{
			Name:         name,
			ProviderType: "unknown",
			IsDefault:    name == r.defaultName,
			Capabilities: nil,
			RegisteredAt: info.registeredAt,
		}

		if metadata, ok := any(info.provider).(ProviderMetadata); ok {
			providerInfo.ProviderType = metadata.GetProviderType()
			providerInfo.Capabilities = metadata.GetProviderMetadata()
		}

		result = append(result, providerInfo)
	}

	slices.SortFunc(result, func(a, b ProviderInfo) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return result
}

// CloseAll closes all registered providers that implement a Close method.
// Implements ProviderRegistry.CloseAll.
//
// Returns error when one or more providers fail to close.
//
// Safe for concurrent use. Takes a read lock while copying the provider list,
// then releases it before closing providers.
func (r *StandardRegistry[T]) CloseAll(ctx context.Context) error {
	r.mu.RLock()
	providers := make([]T, 0, len(r.providers))
	for _, info := range r.providers {
		providers = append(providers, info.provider)
	}
	r.mu.RUnlock()

	var errs []error
	for _, provider := range providers {
		if closer, ok := any(provider).(interface{ Close(context.Context) error }); ok {
			if err := closer.Close(ctx); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing %s providers: %w", r.serviceName, errors.Join(errs...))
	}
	return nil
}

// NewStandardRegistry creates a new provider registry for a service.
//
// Takes serviceName (string) which identifies the service for logging and
// error messages (e.g., "email", "storage").
//
// Returns *StandardRegistry[T] which is the initialised registry ready
// for provider registration.
func NewStandardRegistry[T any](serviceName string) *StandardRegistry[T] {
	return &StandardRegistry[T]{
		mu:          sync.RWMutex{},
		providers:   make(map[string]*registrationInfo[T]),
		defaultName: "",
		serviceName: serviceName,
	}
}
