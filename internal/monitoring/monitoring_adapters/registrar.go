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

package monitoring_adapters

import (
	"context"
	"time"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
)

// noopSpanProcessor is a span processor that does nothing. It is used as the
// default when no OTEL SDK factories are provided.
type noopSpanProcessor struct{}

// Shutdown is a no-op that always returns nil.
//
// Returns error which is always nil.
func (*noopSpanProcessor) Shutdown(context.Context) error { return nil }

// ForceFlush is a no-op that always returns nil.
//
// Returns error which is always nil.
func (*noopSpanProcessor) ForceFlush(context.Context) error { return nil }

// noopMetricReader is a metric reader that does nothing. It is used as the
// default when no OTEL SDK factories are provided.
type noopMetricReader struct{}

// Shutdown is a no-op that always returns nil.
//
// Returns error which is always nil.
func (*noopMetricReader) Shutdown(context.Context) error { return nil }

// noopMetricsCollector collects no metrics. It is used as the default when no
// OTEL SDK factories are provided.
type noopMetricsCollector struct{}

// Start is a no-op that returns immediately.
func (*noopMetricsCollector) Start(context.Context) {}

// Stop is a no-op that returns immediately.
func (*noopMetricsCollector) Stop() {}

// Reader returns a no-op metric reader.
//
// Returns monitoring_domain.MetricReader which is a no-op
// reader.
func (*noopMetricsCollector) Reader() monitoring_domain.MetricReader {
	return &noopMetricReader{}
}

// DefaultServiceFactories returns noop service factories. When the OTEL SDK is
// required, use piko.WithMonitoringOtelFactories() with the factories from
// logger_otel_sdk.OtelServiceFactories() instead.
//
// Returns monitoring_domain.ServiceFactories which contains noop factories.
func DefaultServiceFactories() monitoring_domain.ServiceFactories {
	return monitoring_domain.ServiceFactories{
		SpanProcessorFactory: func(_ *monitoring_domain.TelemetryStore) monitoring_domain.SpanProcessor {
			return &noopSpanProcessor{}
		},
		MetricsCollectorFactory: func(_ *monitoring_domain.TelemetryStore, _ time.Duration) monitoring_domain.MetricsCollectorAdapter {
			return &noopMetricsCollector{}
		},
	}
}
