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

package crypto_domain

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/provider/provider_domain"
)

var (
	// errProviderNameEmpty is returned when a crypto provider is registered
	// with an empty name.
	errProviderNameEmpty = errors.New("provider name cannot be empty")

	// errProviderNil is returned when a nil crypto provider is supplied during
	// registration.
	errProviderNil = errors.New("provider cannot be nil")
)

// RegisterProvider adds a new crypto provider with the given name.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes name (string) which identifies the provider for later retrieval.
// Takes provider (EncryptionProvider) which is the crypto backend to register.
//
// Returns error when the name is empty, the provider is nil, or a provider
// with the same name is already registered.
func (s *cryptoService) RegisterProvider(ctx context.Context, name string, provider EncryptionProvider) error {
	if name == "" {
		return errProviderNameEmpty
	}
	if provider == nil {
		return errProviderNil
	}

	if err := s.registry.RegisterProvider(ctx, name, provider); err != nil {
		return fmt.Errorf("registering crypto provider %q: %w", name, err)
	}

	_, l := logger_domain.From(ctx, log)
	l.Internal("Registered crypto provider",
		logger_domain.String("provider_name", name),
		logger_domain.String("provider_type", string(provider.Type())))

	return nil
}

// SetDefaultProvider sets the provider to use when no specific provider is
// named in a call.
//
// Takes name (string) which identifies the provider to set as default.
//
// Returns error when the named provider does not exist.
func (s *cryptoService) SetDefaultProvider(name string) error {
	return s.registry.SetDefaultProvider(context.Background(), name)
}

// GetProviders returns a sorted list of all registered provider names.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
//
// Returns []string which contains the provider names in alphabetical order.
func (s *cryptoService) GetProviders(ctx context.Context) []string {
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
func (s *cryptoService) HasProvider(name string) bool {
	return s.registry.HasProvider(name)
}

// ListProviders returns detailed information about all registered providers.
//
// Returns []provider_domain.ProviderInfo which contains provider metadata,
// health status, and capabilities.
func (s *cryptoService) ListProviders(ctx context.Context) []provider_domain.ProviderInfo {
	return s.registry.ListProviders(ctx)
}

// getProvider retrieves the default crypto provider.
//
// Returns EncryptionProvider which is the default provider.
// Returns error when no provider is registered or no default is configured.
func (s *cryptoService) getProvider(ctx context.Context) (EncryptionProvider, error) {
	providerName := s.registry.GetDefaultProvider()
	if providerName == "" {
		return nil, errors.New("no default crypto provider configured")
	}
	return s.registry.GetProvider(ctx, providerName)
}

// Close shuts down all providers in an orderly way.
//
// Returns error when the providers cannot be closed.
func (s *cryptoService) Close(ctx context.Context) error {
	return s.registry.CloseAll(ctx)
}
