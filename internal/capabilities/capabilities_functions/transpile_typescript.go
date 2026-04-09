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
	"bytes"
	"context"
	"fmt"
	"io"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

// TranspileTypeScript returns a capability function that transpiles TypeScript
// source code to JavaScript. It strips type annotations while preserving
// runtime code, outputting ES module format.
//
// The optional "sourcePath" parameter is used for error messages. If not
// provided, the filename defaults to "input.ts".
//
// Returns capabilities_domain.CapabilityFunc which performs TypeScript
// transpilation when invoked.
func TranspileTypeScript() capabilities_domain.CapabilityFunc {
	transpiler := generator_domain.NewJSTranspiler()

	return func(ctx context.Context, inputData io.Reader, params capabilities_domain.CapabilityParams) (io.Reader, error) {
		ctx, span, l := log.Span(ctx, "TranspileTypeScript",
			logger_domain.String(logger_domain.KeyReference, "capability"),
		)
		defer span.End()

		if err := ctx.Err(); err != nil {
			l.Warn("Context cancelled before transpilation", logger_domain.String(logger_domain.KeyError, err.Error()))
			span.RecordError(err)
			return nil, fmt.Errorf("transpile typescript context cancelled: %w", err)
		}

		sourcePath := "input.ts"
		if sourcePathParam, ok := params["sourcePath"]; ok && sourcePathParam != "" {
			sourcePath = sourcePathParam
		}
		span.SetAttributes(attribute.String("sourcePath", sourcePath))

		inputBytes, err := io.ReadAll(inputData)
		if err != nil {
			l.ReportError(span, err, "Failed to read TypeScript input")
			return nil, fmt.Errorf("reading typescript input: %w", err)
		}
		span.SetAttributes(attribute.Int("inputSize", len(inputBytes)))

		result, err := transpiler.Transpile(ctx, string(inputBytes), generator_domain.TranspileOptions{
			Filename: sourcePath,
			Minify:   false,
		})
		if err != nil {
			l.ReportError(span, err, "TypeScript transpilation failed")
			return nil, capabilities_domain.NewFatalError(
				fmt.Errorf("transpiling typescript file %q: %w", sourcePath, err),
			)
		}

		span.SetAttributes(attribute.Int("outputSize", len(result.Code)))
		span.SetStatus(codes.Ok, "TypeScript transpilation successful")
		l.Trace("TypeScript transpilation successful",
			logger_domain.Int("inputSize", len(inputBytes)),
			logger_domain.Int("outputSize", len(result.Code)))

		return bytes.NewReader([]byte(result.Code)), nil
	}
}
