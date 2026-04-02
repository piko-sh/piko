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

package logger_otel_sdk

import (
	"time"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
)

// OtelServiceFactories returns the monitoring service factories that create
// real OTEL SDK span processors and metrics collectors. Pass the result to
// piko.WithMonitoringOtelFactories() to enable SDK-backed monitoring.
//
// Returns monitoring_domain.ServiceFactories which contains factories for
// creating SpanProcessor and MetricsCollector instances.
func OtelServiceFactories() monitoring_domain.ServiceFactories {
	return monitoring_domain.ServiceFactories{
		SpanProcessorFactory: func(store *monitoring_domain.TelemetryStore) monitoring_domain.SpanProcessor {
			return NewSpanProcessor(store)
		},
		MetricsCollectorFactory: func(store *monitoring_domain.TelemetryStore, interval time.Duration) monitoring_domain.MetricsCollectorAdapter {
			return NewMetricsCollector(store, interval)
		},
	}
}
