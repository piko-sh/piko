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
	"fmt"
	"io"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

// compressionConfig holds the settings for a compression capability.
type compressionConfig struct {
	// parseLevel extracts the compression level from the given capability parameters.
	parseLevel func(params capabilities_domain.CapabilityParams, defaultLevel int) int

	// factory returns a writerFactory for the given compression level.
	factory func(level int) writerFactory

	// spanName is used for tracing spans and log messages.
	spanName string

	// defaultLevel is the compression level used when no level is given.
	defaultLevel int
}

// createCompressionCapability returns a capability function that performs
// streaming compression using the given settings.
//
// Takes config (compressionConfig) which specifies the compression settings
// including span name, level parser, default level, and compressor factory.
//
// Returns capabilities_domain.CapabilityFunc which wraps the compression
// logic for use with the capability system.
func createCompressionCapability(config compressionConfig) capabilities_domain.CapabilityFunc {
	return func(ctx context.Context, inputData io.Reader, params capabilities_domain.CapabilityParams) (io.Reader, error) {
		ctx, span, l := log.Span(ctx, config.spanName,
			logger_domain.String(logger_domain.KeyReference, "capability"),
			logger_domain.String("mode", "streaming"),
		)
		defer span.End()

		if err := ctx.Err(); err != nil {
			l.Warn("Context cancelled before execution", logger_domain.Error(err))
			span.RecordError(err)
			span.SetStatus(codes.Error, "Context cancelled")
			return nil, fmt.Errorf("context cancelled before %s: %w", config.spanName, err)
		}

		level := config.parseLevel(params, config.defaultLevel)
		span.SetAttributes(attribute.Int("compressionLevel", level))
		l.Trace("Using compression level", logger_domain.Int("level", level))

		outputStream := processStream(ctx, inputData, config.factory(level))

		span.SetStatus(codes.Ok, config.spanName+" stream initiated")
		l.Trace(config.spanName + " stream prepared")

		return outputStream, nil
	}
}
