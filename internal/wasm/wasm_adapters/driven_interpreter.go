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

//go:build js && wasm

package wasm_adapters

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"

	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/internal/wasm/wasm_domain"
	"piko.sh/piko/internal/wasm/wasm_dto"
)

// InterpreterAdapter implements InterpreterPort to execute generated
// Go code inside WASM.
//
// It supports both incremental Eval and Piko's internal bytecode
// interpreter (via batch compilation). It uses interface-based
// dependencies to avoid importing interpreter implementations
// directly, keeping them isolated to the wdk package and cmd/wasm.
type InterpreterAdapter struct {
	// symbolLoader loads symbols into interpreters.
	symbolLoader wasm_domain.SymbolLoaderPort

	// interpreterFactory creates new interpreter instances.
	interpreterFactory wasm_domain.InterpreterFactoryPort
}

var _ wasm_domain.InterpreterPort = (*InterpreterAdapter)(nil)

// InterpreterAdapterOption configures an InterpreterAdapter.
type InterpreterAdapterOption func(*InterpreterAdapter)

// interpreterInstance wraps the interpreter to provide a consistent interface.
type interpreterInstance struct {
	// interp holds the interpreter instance.
	interp any
}

// Eval evaluates Go source code.
//
// Takes ctx (context.Context) for cancellation and deadlines.
// Takes code (string) which is the Go source code to evaluate.
//
// Returns any which is the result of evaluating the code.
// Returns error when the interpreter does not support evaluation or the code
// is invalid.
func (w *interpreterInstance) Eval(ctx context.Context, code string) (any, error) {
	if evaluator, ok := w.interp.(interface {
		Eval(string) (reflect.Value, error)
	}); ok {
		value, err := evaluator.Eval(code)
		if err != nil {
			return nil, fmt.Errorf("evaluating code: %w", err)
		}
		return value.Interface(), nil
	}
	if evaluator, ok := w.interp.(interface{ Eval(string) (any, error) }); ok {
		return evaluator.Eval(code)
	}
	if evaluator, ok := w.interp.(interface {
		Eval(context.Context, string) (any, error)
	}); ok {
		return evaluator.Eval(ctx, code)
	}
	return nil, errors.New("interpreter does not implement Eval")
}

// SetBuildContext sets the build context for the interpreter.
//
// Takes buildCtx (any) which provides the build context configuration.
func (w *interpreterInstance) SetBuildContext(buildCtx any) {
	if setter, ok := w.interp.(interface{ SetBuildContext(any) }); ok {
		setter.SetBuildContext(buildCtx)
	}
}

// SetSourcecodeFilesystem sets the virtual filesystem for source code access.
//
// Takes filesystem (any) which provides the filesystem implementation to use.
func (w *interpreterInstance) SetSourcecodeFilesystem(filesystem any) {
	if setter, ok := w.interp.(interface{ SetSourcecodeFilesystem(any) }); ok {
		setter.SetSourcecodeFilesystem(filesystem)
	}
}

// RegisterPackageAlias registers a package alias.
//
// Takes canonical (string) which is the full package path.
// Takes alias (string) which is the short name to use for the package.
//
// Returns error when the underlying interpreter fails to register the alias.
func (w *interpreterInstance) RegisterPackageAlias(canonical, alias string) error {
	if registrar, ok := w.interp.(interface{ RegisterPackageAlias(string, string) error }); ok {
		return registrar.RegisterPackageAlias(canonical, alias)
	}
	return nil
}

// Reset clears the interpreter state if the underlying interpreter supports it.
func (w *interpreterInstance) Reset() {
	if resetter, ok := w.interp.(interface{ Reset() }); ok {
		resetter.Reset()
	}
}

// Clone creates a copy of the interpreter.
//
// Returns templater_domain.InterpreterPort which is a copy of the interpreter,
// or nil if the underlying interpreter does not support cloning.
func (w *interpreterInstance) Clone() templater_domain.InterpreterPort {
	if cloner, ok := w.interp.(interface{ Clone() any }); ok {
		return &interpreterInstance{interp: cloner.Clone()}
	}
	return nil
}

// Unwrap returns the underlying interpreter.
//
// Returns any which is the wrapped interpreter instance.
func (w *interpreterInstance) Unwrap() any {
	return w.interp
}

// batchCompiler is an optional interface that interpreter instances may
// implement to support batch compilation of multiple packages at once.
// The Piko bytecode interpreter implements this via wasmServiceWrapper.
type batchCompiler interface {
	// CompileAndExecuteWASM compiles and executes all packages
	// at once within the WASM environment.
	CompileAndExecuteWASM(ctx context.Context, mainCode, packagePath string, dependencies map[string]string) error
}

// Interpret executes generated Go code and returns the template AST.
//
// When the interpreter supports batch compilation (batchCompiler),
// it uses CompileAndExecuteWASM to compile all packages at once.
// Otherwise it falls back to the incremental Eval-based path.
//
// Takes request (*wasm_dto.InterpretRequest) which contains the generated code
// and configuration.
//
// Returns *wasm_dto.InterpretResponse which contains the template AST and
// metadata.
// Returns error when interpretation fails.
func (a *InterpreterAdapter) Interpret(ctx context.Context, request *wasm_dto.InterpretRequest) (*wasm_dto.InterpretResponse, error) {
	if a.interpreterFactory == nil {
		return &wasm_dto.InterpretResponse{
			Success: false,
			Error:   "interpreter factory not configured",
		}, nil
	}

	rawInterp := a.interpreterFactory.NewInterpreter()

	if a.symbolLoader != nil {
		if err := a.symbolLoader.Use(rawInterp); err != nil {
			return &wasm_dto.InterpretResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to load symbols: %v", err),
			}, nil
		}
	}

	if bc, ok := rawInterp.(batchCompiler); ok {
		if err := bc.CompileAndExecuteWASM(ctx, request.GeneratedCode, request.PackagePath, request.Dependencies); err != nil {
			return &wasm_dto.InterpretResponse{
				Success: false,
				Error:   fmt.Sprintf("batch compilation failed: %v", err),
			}, nil
		}

		return a.buildASTResponse(ctx, request)
	}

	wrapper := &interpreterInstance{interp: rawInterp}

	vfs := NewInterpreterVFS(map[string]string{
		request.PackagePath + "/main.go": request.GeneratedCode,
	})

	for packagePath, content := range request.Dependencies {
		vfs.AddFile(packagePath+"/generated.go", content)
	}

	wrapper.SetSourcecodeFilesystem(vfs)

	_, err := wrapper.Eval(ctx, request.GeneratedCode)
	if err != nil {
		return &wasm_dto.InterpretResponse{
			Success: false,
			Error:   fmt.Sprintf("evaluation failed: %v", err),
		}, nil
	}

	return a.buildASTResponse(ctx, request)
}

// buildASTResponse retrieves the registered AST function and produces
// an InterpretResponse. This is shared between the batch compilation
// and Eval code paths.
//
// Takes ctx (context.Context) for the request context.
// Takes request (*wasm_dto.InterpretRequest) which contains the package
// path, request URL, and props.
//
// Returns *wasm_dto.InterpretResponse which contains the template AST
// and metadata.
// Returns error which is always nil because errors are reported
// inside the response struct.
func (a *InterpreterAdapter) buildASTResponse(ctx context.Context, request *wasm_dto.InterpretRequest) (*wasm_dto.InterpretResponse, error) {
	astFunc, found := templater_domain.GetASTFunc(request.PackagePath)
	if !found {
		return &wasm_dto.InterpretResponse{
			Success: false,
			Error:   fmt.Sprintf("BuildAST not registered for package path: %s", request.PackagePath),
		}, nil
	}

	requestData := buildMockRequestData(ctx, request.RequestURL)
	defer requestData.Release()

	ast, metadata, runtimeDiags := astFunc(requestData, request.Props)

	diagnostics := convertRuntimeDiagnostics(runtimeDiags)

	if ast == nil && len(diagnostics) > 0 {
		return &wasm_dto.InterpretResponse{
			Success:     false,
			Error:       "BuildAST returned nil AST",
			Diagnostics: diagnostics,
		}, nil
	}

	return &wasm_dto.InterpretResponse{
		Success:     true,
		AST:         ast,
		Metadata:    &metadata,
		Diagnostics: diagnostics,
	}, nil
}

// WithSymbolLoader sets the symbol loader for the interpreter adapter.
//
// Takes loader (wasm_domain.SymbolLoaderPort) which loads symbols into
// the interpreter.
//
// Returns InterpreterAdapterOption which configures the adapter.
func WithSymbolLoader(loader wasm_domain.SymbolLoaderPort) InterpreterAdapterOption {
	return func(a *InterpreterAdapter) {
		a.symbolLoader = loader
	}
}

// WithInterpreterFactory sets the interpreter factory for the adapter.
//
// Takes factory (wasm_domain.InterpreterFactoryPort) which creates new
// interpreter instances.
//
// Returns InterpreterAdapterOption which configures the adapter.
func WithInterpreterFactory(factory wasm_domain.InterpreterFactoryPort) InterpreterAdapterOption {
	return func(a *InterpreterAdapter) {
		a.interpreterFactory = factory
	}
}

// NewInterpreterAdapter creates a new interpreter adapter for WASM.
//
// Takes opts (...InterpreterAdapterOption) which configure the adapter.
//
// Returns *InterpreterAdapter which is ready for interpreting generated code.
func NewInterpreterAdapter(opts ...InterpreterAdapterOption) *InterpreterAdapter {
	a := &InterpreterAdapter{}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// buildMockRequestData creates a RequestData instance for WASM execution.
//
// Takes requestURL (string) which is the URL for the mock request.
//
// Returns *templater_dto.RequestData which is configured for the given URL.
func buildMockRequestData(ctx context.Context, requestURL string) *templater_dto.RequestData {
	builder := templater_dto.NewRequestDataBuilder().
		WithContext(ctx).
		WithMethod("GET")

	if requestURL != "" {
		if parsedURL, err := url.Parse(requestURL); err == nil {
			builder.WithURL(parsedURL)
			builder.WithHost(parsedURL.Host)
			for key, values := range parsedURL.Query() {
				builder.AddQueryParam(key, values)
			}
		}
	}

	return builder.Build()
}

// convertRuntimeDiagnostics converts generator runtime diagnostics to wasm DTOs.
//
// Takes diagnostics ([]*generator_dto.RuntimeDiagnostic) which are the diagnostics
// from BuildAST execution.
//
// Returns []wasm_dto.Diagnostic which contains the converted diagnostics.
func convertRuntimeDiagnostics(diagnostics []*generator_dto.RuntimeDiagnostic) []wasm_dto.Diagnostic {
	if len(diagnostics) == 0 {
		return nil
	}

	result := make([]wasm_dto.Diagnostic, 0, len(diagnostics))
	for _, d := range diagnostics {
		if d == nil {
			continue
		}
		result = append(result, wasm_dto.Diagnostic{
			Severity: severityToString(d.Severity),
			Message:  d.Message,
			Location: wasm_dto.Location{
				FilePath: d.SourcePath,
				Line:     d.Line,
				Column:   d.Column,
			},
		})
	}
	return result
}

// severityToString converts a generator severity constant to a string.
//
// Takes severity (generator_dto.Severity) which is the severity level.
//
// Returns string which is the human-readable severity name.
func severityToString(severity generator_dto.Severity) string {
	switch severity {
	case generator_dto.Debug:
		return "debug"
	case generator_dto.Info:
		return "info"
	case generator_dto.Warning:
		return "warning"
	case generator_dto.Error:
		return "error"
	default:
		return "unknown"
	}
}
