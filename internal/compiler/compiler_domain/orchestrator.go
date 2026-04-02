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

package compiler_domain

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/compiler/compiler_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// moduleNameKey is the context key for storing the module name during
// compilation.
type moduleNameKey struct{}

// compilerOrchestrator coordinates the SFC compilation pipeline.
// It implements CompilerService.
type compilerOrchestrator struct {
	// inputReader reads SFC source data by identifier.
	inputReader InputReaderPort

	// sfcCompiler converts raw SFC bytes into compiled artefacts.
	sfcCompiler SFCCompiler

	// moduleName is the Go module name from go.mod, such as
	// "github.com/org/repo". Used to resolve @/ aliases in asset paths.
	moduleName string

	// transformSteps holds the ordered list of transformations to apply to
	// compiled artefacts.
	transformSteps []TransformationPort
}

// OrchestratorOption is a functional option for configuring the compiler
// orchestrator.
type OrchestratorOption func(*compilerOrchestrator)

// CompileSingle compiles a single SFC by reading it from the input reader
// using the given source ID.
//
// Takes sourceID (string) which identifies the SFC to compile.
//
// Returns *compiler_dto.CompiledArtefact which contains the compiled output.
// Returns error when the SFC cannot be read or compilation fails.
func (o *compilerOrchestrator) CompileSingle(ctx context.Context, sourceID string) (*compiler_dto.CompiledArtefact, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "CompilerOrchestrator.CompileSingle",
		logger_domain.String("sourceID", sourceID),
	)
	defer span.End()

	OrchestratorCompilationCount.Add(ctx, 1)

	l.Trace("Reading SFC from input reader")

	var rawData []byte
	err := l.RunInSpan(ctx, "ReadSFC", func(ctx context.Context, _ logger_domain.Logger) error {
		var err error
		rawData, err = o.inputReader.ReadSFC(ctx, sourceID)
		if err != nil {
			return fmt.Errorf("reading SFC %q: %w", sourceID, err)
		}
		return nil
	})

	if err != nil {
		l.ReportError(span, err, "Failed to read SFC")
		return nil, fmt.Errorf("failed to read SFC: %w", err)
	}

	l.Trace("Successfully read SFC, proceeding to compile",
		logger_domain.Int("dataSize", len(rawData)))

	return o.CompileSFCBytes(ctx, sourceID, rawData)
}

// CompileSFCBytes compiles raw SFC bytes into a compiled artefact with the
// given source ID.
//
// Takes sourceID (string) which identifies the source of the SFC content.
// Takes rawSFC ([]byte) which contains the raw single-file component bytes.
//
// Returns *compiler_dto.CompiledArtefact which is the compiled and transformed
// artefact.
// Returns error when compilation or any transformation step fails.
func (o *compilerOrchestrator) CompileSFCBytes(ctx context.Context, sourceID string, rawSFC []byte) (*compiler_dto.CompiledArtefact, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "CompilerOrchestrator.CompileSFCBytes",
		logger_domain.String("sourceID", sourceID),
		logger_domain.Int("rawSFCSize", len(rawSFC)),
	)
	defer span.End()

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		OrchestratorCompilationDuration.Record(ctx, float64(duration.Milliseconds()))
	}()

	l.Trace("Compiling SFC bytes")

	if o.moduleName != "" {
		ctx = WithModuleName(ctx, o.moduleName)
		l.Trace("Added module name to context", logger_domain.String("moduleName", o.moduleName))
	}

	artefact, err := o.sfcCompiler.CompileSFC(ctx, sourceID, rawSFC)
	if err != nil {
		l.Trace("Failed to compile SFC", logger_domain.Error(err))
		OrchestratorCompilationErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("compile error: %w", err)
	}

	l.Trace("Setting source identifier", logger_domain.String("sourceID", sourceID))
	artefact.SourceIdentifier = sourceID

	artefact, err = o.applyTransformationSteps(ctx, span, artefact)
	if err != nil {
		return nil, fmt.Errorf("applying transformation steps: %w", err)
	}

	l.Trace("Successfully compiled and transformed SFC")
	return artefact, nil
}

// applyTransformationSteps applies all registered transformation steps
// to the artefact.
//
// Takes span (trace.Span) which is the parent span for tracing.
// Takes artefact (*compiler_dto.CompiledArtefact) which will be transformed.
//
// Returns *compiler_dto.CompiledArtefact which is the transformed artefact.
// Returns error when any transformation step fails.
func (o *compilerOrchestrator) applyTransformationSteps(ctx context.Context, span trace.Span, artefact *compiler_dto.CompiledArtefact) (*compiler_dto.CompiledArtefact, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Applying transformation steps", logger_domain.Int("stepCount", len(o.transformSteps)))

	for i, step := range o.transformSteps {
		stepIndex := i
		OrchestratorTransformationCount.Add(ctx, 1)

		transformStartTime := time.Now()
		err := l.RunInSpan(ctx, fmt.Sprintf("TransformStep%d", stepIndex), func(ctx context.Context, _ logger_domain.Logger) error {
			var err error
			artefact, err = step.Transform(ctx, artefact)
			if err != nil {
				return fmt.Errorf("transform step %d: %w", stepIndex, err)
			}
			return nil
		})
		transformDuration := time.Since(transformStartTime)
		OrchestratorTransformationDuration.Record(ctx, float64(transformDuration.Milliseconds()))

		if err != nil {
			l.ReportError(span, err, "Transformation step failed",
				logger_domain.Int("stepIndex", stepIndex))
			OrchestratorTransformationErrorCount.Add(ctx, 1)
			return nil, fmt.Errorf("transformation error: %w", err)
		}
	}

	return artefact, nil
}

// WithModuleName returns a new context with the module name attached.
//
// Takes moduleName (string) which is the name of the Go module.
//
// Returns context.Context which contains the module name value.
func WithModuleName(ctx context.Context, moduleName string) context.Context {
	return context.WithValue(ctx, moduleNameKey{}, moduleName)
}

// GetModuleName returns the module name stored in the context.
//
// Returns string which is the module name, or empty if not set.
func GetModuleName(ctx context.Context) string {
	if v := ctx.Value(moduleNameKey{}); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// WithSFCCompiler sets a custom SFCCompiler for the orchestrator. This is
// useful for testing with mock compilers.
//
// Takes c (SFCCompiler) which provides the compiler to use.
//
// Returns OrchestratorOption which sets up the orchestrator to use the given
// compiler.
func WithSFCCompiler(c SFCCompiler) OrchestratorOption {
	return func(o *compilerOrchestrator) {
		o.sfcCompiler = c
	}
}

// WithOrchestratorModuleName sets the Go module name for @/ alias resolution.
// The module name should come from the resolver's GetModuleName() method.
//
// Takes moduleName (string) which is the Go module name, for example
// "github.com/org/repo".
//
// Returns OrchestratorOption which configures the orchestrator to use the
// given module name for path resolution.
func WithOrchestratorModuleName(moduleName string) OrchestratorOption {
	return func(o *compilerOrchestrator) {
		o.moduleName = moduleName
	}
}

// NewCompilerOrchestrator creates a new compiler orchestrator with the given
// input reader and transformation steps.
//
// Takes inputReader (InputReaderPort) which provides the source input.
// Takes transformSteps ([]TransformationPort) which defines the pipeline.
// Takes opts (...OrchestratorOption) which configures optional behaviour.
//
// Returns CompilerService which is the configured orchestrator ready for use.
func NewCompilerOrchestrator(inputReader InputReaderPort, transformSteps []TransformationPort, opts ...OrchestratorOption) CompilerService {
	o := &compilerOrchestrator{
		inputReader:    inputReader,
		sfcCompiler:    NewSFCCompiler(),
		transformSteps: transformSteps,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
