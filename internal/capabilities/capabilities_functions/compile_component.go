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
	"errors"
	"fmt"
	"io"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/compiler/compiler_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// CompileComponent returns a capability function that compiles a component
// using the provided compiler service. It requires a 'sourcePath' parameter
// to identify the component being compiled.
//
// Takes compiler (compiler_domain.CompilerService) which provides the
// compilation service for processing component source code.
//
// Returns capabilities_domain.CapabilityFunc which is the capability function
// that performs the compilation when invoked.
func CompileComponent(compiler compiler_domain.CompilerService) capabilities_domain.CapabilityFunc {
	return func(ctx context.Context, inputData io.Reader, params capabilities_domain.CapabilityParams) (io.Reader, error) {
		ctx, span, l := log.Span(ctx, "CompileComponent",
			logger_domain.String(logger_domain.KeyReference, "capability"),
		)
		defer span.End()

		l.Trace("Starting compile component capability execution")

		if err := ctx.Err(); err != nil {
			l.Warn("Context cancelled during execution", logger_domain.String(logger_domain.KeyError, err.Error()))
			span.RecordError(err)
			return nil, fmt.Errorf("compile component context cancelled: %w", err)
		}

		sourcePath, err := extractSourcePath(ctx, params, span)
		if err != nil {
			return nil, fmt.Errorf("extracting source path for compilation: %w", err)
		}
		span.SetAttributes(attribute.String("sourcePath", sourcePath))
		l = l.With(logger_domain.String("sourcePath", sourcePath))

		inputBytes, err := readInputData(ctx, span, inputData)
		if err != nil {
			return nil, fmt.Errorf("reading input data for compilation: %w", err)
		}
		span.SetAttributes(attribute.Int("inputSize", len(inputBytes)))
		l = l.With(logger_domain.Int("inputSize", len(inputBytes)))

		artefact, err := compileSource(ctx, compiler, sourcePath, inputBytes)
		if err != nil {
			span.SetAttributes(attribute.Int64("compilationDuration", 0))
			compilationErrorCount.Add(ctx, 1)
			return nil, capabilities_domain.NewFatalError(err)
		}

		outputJS, err := extractOutput(ctx, artefact, sourcePath, span)
		if err != nil {
			compilationErrorCount.Add(ctx, 1)
			return nil, capabilities_domain.NewFatalError(err)
		}

		recordSuccess(ctx, span, outputJS)
		return bytes.NewReader([]byte(outputJS)), nil
	}
}

// extractSourcePath checks and returns the required sourcePath parameter.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes params (capabilities_domain.CapabilityParams) which holds the
// parameters to read from.
// Takes span (trace.Span) which gives tracing context for error reporting.
//
// Returns string which is the checked source path.
// Returns error when the sourcePath parameter is missing or empty.
func extractSourcePath(ctx context.Context, params capabilities_domain.CapabilityParams, span trace.Span) (string, error) {
	ctx, l := logger_domain.From(ctx, log)
	sourcePath, ok := params["sourcePath"]
	if !ok || sourcePath == "" {
		err := errors.New("compile-component capability requires a 'sourcePath' parameter")
		l.ReportError(span, err, "Missing required sourcePath parameter")
		return "", err
	}
	return sourcePath, nil
}

// readInputData reads all data from an input stream into memory.
//
// Takes span (trace.Span) which receives error reports on failure.
// Takes inputData (io.Reader) which provides the source data to read.
//
// Returns []byte which contains the full input data.
// Returns error when the input stream cannot be read.
func readInputData(ctx context.Context, span trace.Span, inputData io.Reader) ([]byte, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Buffering input stream for compiler")

	var inputBytes []byte
	err := l.RunInSpan(ctx, "ReadInputData", func(_ context.Context, _ logger_domain.Logger) error {
		var err error
		inputBytes, err = io.ReadAll(inputData)
		if err != nil {
			return fmt.Errorf("reading all input data: %w", err)
		}
		return nil
	})
	if err != nil {
		l.ReportError(span, err, "Failed to read input stream")
		return nil, fmt.Errorf("failed to read component source stream: %w", err)
	}
	return inputBytes, nil
}

// compileSource runs the compiler and records how long it takes.
//
// Takes compiler (compiler_domain.CompilerService) which performs the
// compilation.
// Takes sourcePath (string) which identifies the source file being compiled.
// Takes inputBytes ([]byte) which contains the source content to compile.
//
// Returns *compiler_dto.CompiledArtefact which contains the compiled output.
// Returns error when the compiler service fails.
func compileSource(
	ctx context.Context,
	compiler compiler_domain.CompilerService,
	sourcePath string,
	inputBytes []byte,
) (*compiler_dto.CompiledArtefact, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Compiling component")

	var artefact *compiler_dto.CompiledArtefact
	startTime := time.Now()

	err := l.RunInSpan(ctx, "CompileSFCBytes", func(spanCtx context.Context, _ logger_domain.Logger) error {
		var err error
		artefact, err = compiler.CompileSFCBytes(spanCtx, sourcePath, inputBytes)
		if err != nil {
			return fmt.Errorf("compiling SFC bytes for %q: %w", sourcePath, err)
		}
		return nil
	})

	duration := time.Since(startTime)
	compilationDuration.Record(ctx, float64(duration.Milliseconds()))

	if err != nil {
		return nil, fmt.Errorf("compiler service failed for source '%s': %w", sourcePath, err)
	}
	return artefact, nil
}

// extractOutput finds and returns the compiled JavaScript from the artefact.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes artefact (*compiler_dto.CompiledArtefact) which contains the compiled
// files and metadata.
// Takes sourcePath (string) which identifies the source for error messages.
// Takes span (trace.Span) which records telemetry attributes.
//
// Returns string which is the JavaScript content from the entrypoint file.
// Returns error when the expected entrypoint file is not found in the
// artefact.
func extractOutput(
	ctx context.Context,
	artefact *compiler_dto.CompiledArtefact,
	sourcePath string,
	span trace.Span,
) (string, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Looking for output JS in compiled artefact",
		logger_domain.String("baseJSPath", artefact.BaseJSPath),
		logger_domain.Int("fileCount", len(artefact.Files)))
	span.SetAttributes(
		attribute.String("baseJSPath", artefact.BaseJSPath),
		attribute.Int("fileCount", len(artefact.Files)),
	)

	outputJS, ok := artefact.Files[artefact.BaseJSPath]
	if !ok {
		err := fmt.Errorf("compiled artefact for '%s' did not contain expected entrypoint file '%s'", sourcePath, artefact.BaseJSPath)
		l.ReportError(span, err, "Expected entrypoint file not found")
		return "", err
	}
	return outputJS, nil
}

// recordSuccess logs and records metrics for a successful compilation.
//
// Takes span (trace.Span) which receives the output size attribute and status.
// Takes outputJS (string) which is the compiled output to measure.
func recordSuccess(ctx context.Context, span trace.Span, outputJS string) {
	ctx, l := logger_domain.From(ctx, log)
	outputSize := len(outputJS)
	compiledComponentSize.Record(ctx, int64(outputSize))
	span.SetAttributes(attribute.Int("outputSize", outputSize))
	span.SetStatus(codes.Ok, "Component compilation successful")
	l.Trace("Component compilation successful", logger_domain.Int("outputSize", outputSize))
}
