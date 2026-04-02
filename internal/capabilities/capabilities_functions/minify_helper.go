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

package capabilities_functions

import (
	"context"
	"io"
	"time"

	"github.com/tdewolff/minify/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

// minifyConfig holds the settings for a minification capability.
type minifyConfig struct {
	// minifier performs content minification for the configured MIME type.
	minifier *minify.M

	// spanName identifies the span for distributed tracing.
	spanName string

	// mimeType is the MIME type of the content to minify.
	mimeType string

	// contentType specifies the media type for logging, e.g. "JavaScript" or "SVG".
	contentType string
}

// createMinifyCapability returns a capability function that minifies content
// using the provided tdewolff/minify settings.
//
// Takes config (minifyConfig) which specifies the minifier settings and content
// type.
//
// Returns capabilities_domain.CapabilityFunc which wraps the minification
// logic as a streaming capability.
func createMinifyCapability(config minifyConfig) capabilities_domain.CapabilityFunc {
	return func(ctx context.Context, inputData io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
		ctx, span, l := log.Span(ctx, config.spanName,
			logger_domain.String(logger_domain.KeyReference, "capability"),
			logger_domain.String("mode", "streaming"),
		)
		defer span.End()

		l.Trace("Preparing " + config.contentType + " minification stream")

		select {
		case <-ctx.Done():
			l.Warn("Context cancelled during execution setup", logger_domain.String(logger_domain.KeyError, context.Cause(ctx).Error()))
			span.RecordError(context.Cause(ctx))
			span.SetStatus(codes.Error, "Context cancelled")
			return nil, ctx.Err()
		default:
		}

		startTime := time.Now()
		minifiedStream := config.minifier.Reader(config.mimeType, inputData)
		duration := time.Since(startTime)

		minificationDuration.Record(ctx, float64(duration.Milliseconds()))
		span.SetAttributes(
			attribute.Int64("durationMs", duration.Milliseconds()),
			attribute.String("status", "stream_prepared"),
		)
		span.SetStatus(codes.Ok, config.contentType+" minification stream prepared")

		l.Trace(config.contentType + " minification stream prepared successfully")
		return newFatalReader(minifiedStream), nil
	}
}
