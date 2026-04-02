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

package driver_handlers

import (
	"context"
	"io"
	"sync"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// OtelProviderResult holds the outputs from creating OTEL SDK providers.
type OtelProviderResult struct {
	// TracerProvider holds the configured OTEL trace provider.
	TracerProvider trace.TracerProvider

	// MeterProvider holds the configured OTEL meter provider.
	MeterProvider metric.MeterProvider

	// ShutdownFunc holds a function that shuts down the providers gracefully.
	ShutdownFunc func(context.Context) error

	// Closers holds resources that must be closed during shutdown.
	Closers []io.Closer
}

// OtelProviderFactory creates real OTEL SDK trace and metric providers. When no
// factory is registered, SetupOtel uses noop providers instead.
type OtelProviderFactory func(
	ctx context.Context,
	config OtelSetupConfig,
	additionalProcessors []any,
	additionalReaders []any,
) (OtelProviderResult, error)

var (
	providerFactoryMu sync.RWMutex

	providerFactory OtelProviderFactory
)

// RegisterOtelProviderFactory sets the factory for creating OTEL SDK providers.
//
// Typically called from init() in the wdk/logger/logger_otel_sdk module.
//
// Takes factory (OtelProviderFactory) which creates the SDK providers.
//
// Safe for concurrent use. Protected by a package-level mutex.
func RegisterOtelProviderFactory(factory OtelProviderFactory) {
	providerFactoryMu.Lock()
	defer providerFactoryMu.Unlock()
	providerFactory = factory
}

// getOtelProviderFactory returns the registered provider factory, or nil if
// none has been registered.
//
// Returns OtelProviderFactory which creates SDK providers, or nil.
//
// Safe for concurrent use. Protected by a package-level mutex.
func getOtelProviderFactory() OtelProviderFactory {
	providerFactoryMu.RLock()
	defer providerFactoryMu.RUnlock()
	return providerFactory
}
