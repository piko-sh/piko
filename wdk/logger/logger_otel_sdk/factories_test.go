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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
)

func TestOtelServiceFactoriesBuildSDKAdapters(t *testing.T) {
	t.Parallel()

	factories := OtelServiceFactories()
	require.NotNil(t, factories.SpanProcessorFactory)
	require.NotNil(t, factories.MetricsCollectorFactory)

	store := monitoring_domain.NewTelemetryStore()

	assert.IsType(t, &SpanProcessor{}, factories.SpanProcessorFactory(store))
	assert.IsType(t, &MetricsCollector{}, factories.MetricsCollectorFactory(store, time.Minute))
}
