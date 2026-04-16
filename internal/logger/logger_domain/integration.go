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

package logger_domain

import (
	"context"
	"log/slog"
	"sync"

	"go.opentelemetry.io/otel/propagation"
)

// Integration represents an external logging or tracing service integration.
//
// Integrations are optional and must be explicitly imported to be available.
// When an integration package is imported, it auto-registers itself via init().
// The logger framework then discovers registered integrations at runtime.
//
// This design allows the core logger domain to remain agnostic of specific
// integrations (Sentry, Datadog, Honeycomb, etc.) while supporting arbitrary
// third-party integrations.
type Integration interface {
	// Type returns the integration type name as used in config files. This must
	// match the "type" field in IntegrationConfig, for example "sentry", "datadog",
	// or "honeycomb".
	//
	// Returns string which is the integration type identifier.
	Type() string

	// CreateHandler creates an slog.Handler from the integration-specific config.
	//
	// The config parameter is the typed config struct from logger_dto (e.g.,
	// *SentryConfig, *DatadogConfig). Each integration implementation should
	// type-assert to its expected config type.
	//
	// Returns the configured slog.Handler, or nil if the integration doesn't
	// provide a log handler. Returns an error if initialisation fails.
	CreateHandler(config any) (slog.Handler, error)
}

// SpanProcessor receives span lifecycle events from the OTEL SDK.
//
// Concrete implementations (e.g. from otel/sdk/trace) satisfy this interface
// via structural typing, allowing the logger domain to remain SDK-free.
type SpanProcessor interface {
	// Shutdown flushes remaining spans and releases resources.
	//
	// Takes ctx (context.Context) which controls the shutdown deadline.
	//
	// Returns error when the shutdown fails or the context expires.
	Shutdown(ctx context.Context) error

	// ForceFlush immediately exports all buffered span data.
	//
	// Takes ctx (context.Context) which controls the flush deadline.
	//
	// Returns error when the flush fails or the context expires.
	ForceFlush(ctx context.Context) error
}

// OtelIntegration extends Integration for services that provide OpenTelemetry
// components for distributed tracing.
//
// Integrations implementing this interface will have their OTel components
// added to the global tracer during OTEL setup.
type OtelIntegration interface {
	Integration

	// OtelComponents returns the OpenTelemetry span processor and propagator
	// for this integration.
	//
	// Called during OTEL setup to enable distributed tracing through the
	// external service. Either or both return values may be nil if the
	// integration doesn't provide that component.
	OtelComponents() (processor SpanProcessor, propagator propagation.TextMapPropagator)
}

var (
	// integrationsMu guards concurrent access to integrations.
	integrationsMu sync.RWMutex

	// integrations holds the registered logger integrations keyed by type name.
	integrations = make(map[string]Integration)
)

// RegisterIntegration adds an integration to the registry.
//
// This is usually called by an integration package's init() function
// to register itself when the package is imported.
//
// When an integration with the same type is already registered, it is
// replaced with the new one.
//
// Takes integration (Integration) which is the integration to register.
//
// Safe for concurrent use by multiple goroutines.
func RegisterIntegration(integration Integration) {
	integrationsMu.Lock()
	defer integrationsMu.Unlock()
	integrations[integration.Type()] = integration
}

// GetIntegration returns the registered integration for the given type name,
// or nil if no integration with that type is registered.
//
// Use this to check if an integration is available before trying to use it.
//
// Takes typeName (string) which specifies the integration type to look up.
//
// Returns Integration which is the registered integration, or nil if not found.
//
// Safe for concurrent use by multiple goroutines.
func GetIntegration(typeName string) Integration {
	integrationsMu.RLock()
	defer integrationsMu.RUnlock()
	return integrations[typeName]
}

// IsIntegrationAvailable reports whether an integration with the given type
// name is registered.
//
// Takes typeName (string) which specifies the integration type to look up.
//
// Returns bool which is true if the integration exists, false otherwise.
func IsIntegrationAvailable(typeName string) bool {
	return GetIntegration(typeName) != nil
}

// GetEnabledOtelIntegrations returns all registered integrations that implement
// OtelIntegration and are in the provided list of enabled types.
//
// This is used during OTEL setup to collect components from all enabled
// integrations.
//
// Takes enabledTypes ([]string) which specifies the integration type names to
// retrieve.
//
// Returns []OtelIntegration which contains the matching integrations that
// implement OtelIntegration.
//
// Safe for concurrent use by multiple goroutines.
func GetEnabledOtelIntegrations(enabledTypes []string) []OtelIntegration {
	integrationsMu.RLock()
	defer integrationsMu.RUnlock()

	var result []OtelIntegration
	for _, typeName := range enabledTypes {
		if integration, ok := integrations[typeName]; ok {
			if otelInt, ok := integration.(OtelIntegration); ok {
				result = append(result, otelInt)
			}
		}
	}
	return result
}
