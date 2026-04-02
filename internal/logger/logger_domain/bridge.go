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
)

// SpanLifecycleHook provides callbacks for span lifecycle events, allowing
// external systems to hook into span creation and error reporting for
// observability integration.
type SpanLifecycleHook interface {
	// OnSpanStart begins a new tracing span with the given name and attributes.
	//
	// Takes spanName (string) which identifies the span.
	// Takes attrs ([]slog.Attr) which provides additional span metadata.
	//
	// Returns newCtx (context.Context) which carries the span for child
	// operations.
	// Returns finisher (func()) which must be called to end the span.
	OnSpanStart(ctx context.Context, spanName string, attrs []slog.Attr) (newCtx context.Context, finisher func())

	// OnReportError handles reporting of errors during processing.
	//
	// Takes err (error) which is the error that occurred.
	// Takes message (string) which provides additional context about the error.
	// Takes attrs ([]slog.Attr) which contains structured logging attributes.
	OnReportError(ctx context.Context, err error, message string, attrs []slog.Attr)
}
