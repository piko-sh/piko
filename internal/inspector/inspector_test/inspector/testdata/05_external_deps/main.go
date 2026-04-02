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

// This file tests the ability to inspect types from external dependencies.
package main

import (
	// Standard library import
	"context"

	// Third-party imports
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

// Response struct using external types.
type Response struct {
	// A simple, well-known struct from an external package.
	RequestID uuid.UUID

	// A more complex type (an interface) from an external package.
	CurrentSpan trace.Span

	// A map using an external type as its value.
	SpanContexts map[string]trace.SpanContext
}

// Props struct to test passing external types as props.
type Props struct {
	Tracer trace.Tracer
}

// GetTraceID is a function that uses an external type in its signature.
// This tests method lookups and return type resolution.
func (r *Response) GetTraceID(ctx context.Context) string {
	if r.CurrentSpan != nil {
		return r.CurrentSpan.SpanContext().TraceID().String()
	}
	return ""
}
