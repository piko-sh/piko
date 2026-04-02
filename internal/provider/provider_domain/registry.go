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
	"context"
	"time"
)

// ProviderRegistry manages provider registration, discovery, and lifecycle.
// It implements the ProviderRegistry interface from the provider domain.
//
// The generic type T is the provider interface (such as EmailProviderPort or
// StorageProviderPort). The registry provides type-safe provider registration
// with duplicate detection, default provider selection, provider discovery
// with metadata, and graceful shutdown with provider cleanup.
//
// Thread-safety: All implementations must be safe for concurrent use.
type ProviderRegistry[T any] interface {
	// RegisterProvider adds a named provider to the registry.
	//
	// Takes ctx (context.Context) which carries logging context for
	// trace/request ID propagation.
	// Takes name (string) which identifies the provider.
	// Takes provider (T) which is the provider to register.
	//
	// Returns error when name is empty or a provider with this name already exists.
	//
	// Note: The default provider must be set with SetDefaultProvider.
	RegisterProvider(ctx context.Context, name string, provider T) error

	// SetDefaultProvider marks a provider as the default.
	//
	// Takes ctx (context.Context) which carries logging context for
	// trace/request ID propagation.
	// Takes name (string) which identifies the provider to set as default.
	//
	// Returns error when the provider is not found.
	SetDefaultProvider(ctx context.Context, name string) error

	// GetDefaultProvider returns the name of the default provider.
	//
	// Returns string which is empty if no providers are registered.
	GetDefaultProvider() string

	// GetProvider retrieves a provider by its name.
	//
	// Takes name (string) which is the name of the provider to find.
	//
	// Returns T which is the provider if found.
	// Returns error when the provider is not found.
	GetProvider(ctx context.Context, name string) (T, error)

	// ListProviders returns details about all registered providers.
	//
	// Returns []ProviderInfo which contains the name, type, and capabilities of
	// each provider.
	ListProviders(ctx context.Context) []ProviderInfo

	// HasProvider checks whether a provider with the given name is registered.
	//
	// Takes name (string) which is the provider name to look for.
	//
	// Returns bool which is true if the provider exists, false otherwise.
	HasProvider(name string) bool

	// CloseAll gracefully closes all registered providers during application
	// shutdown. Providers that implement io.Closer will have Close called.
	//
	// Returns error when any provider fails to close.
	CloseAll(ctx context.Context) error
}

// ProviderInfo contains metadata about a registered provider.
// Returned by ListProviders() for discovery and monitoring.
type ProviderInfo struct {
	// Capabilities contains provider-specific metadata. It is populated from the
	// ProviderMetadata interface if the provider implements it.
	Capabilities map[string]any `json:"capabilities,omitempty"`

	// RegisteredAt is when the provider was registered.
	RegisteredAt time.Time `json:"registered_at"`

	// Name is the unique identifier for the provider, set during registration.
	Name string `json:"name"`

	// ProviderType is the implementation type such as "smtp", "ses", "s3", or
	// "disk". Populated from ProviderMetadata interface if implemented.
	ProviderType string `json:"provider_type"`

	// IsDefault indicates whether this is the default provider.
	IsDefault bool `json:"is_default"`
}
