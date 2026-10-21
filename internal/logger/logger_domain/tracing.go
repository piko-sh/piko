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
	"sync/atomic"

	"go.opentelemetry.io/otel/trace"
)

var (
	// tracingEnabled records whether a tracer provider that records spans is installed.
	tracingEnabled atomic.Bool
)

// SetTracingEnabled records whether spans are recorded, so hot paths can skip creating
// spans nobody records. The OTEL driver sets it when it installs a real tracer provider.
//
// Takes enabled (bool) which is true when spans are recorded.
func SetTracingEnabled(enabled bool) {
	tracingEnabled.Store(enabled)
}

// TracingEnabled reports whether a tracer provider that records spans is installed.
//
// Returns bool which is true when spans are recorded.
func TracingEnabled() bool {
	return tracingEnabled.Load()
}

// SpansActive reports whether tracing is enabled or ctx contains a recording span.
//
// Returns bool which is true when a span would be recorded.
func SpansActive(ctx context.Context) bool {
	return tracingEnabled.Load() || trace.SpanFromContext(ctx).IsRecording()
}
