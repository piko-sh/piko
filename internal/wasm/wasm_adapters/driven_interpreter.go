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
	"fmt"
	"net/url"
	"runtime/debug"

	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/internal/wasm/wasm_domain"
	"piko.sh/piko/internal/wasm/wasm_dto"
)

// InterpreterAdapter implements InterpreterPort to execute generated Go code inside WASM.
type InterpreterAdapter struct {
	// interpreterFactory creates new interpreter instances.
	interpreterFactory wasm_domain.InterpreterFactoryPort
}

var (
	_ wasm_domain.InterpreterPort = (*InterpreterAdapter)(nil)
)

// InterpreterAdapterOption configures an InterpreterAdapter.
type InterpreterAdapterOption func(*InterpreterAdapter)

// NewInterpreterAdapter creates a new interpreter adapter for WASM.
//
// Takes opts (...InterpreterAdapterOption) which configure the adapter.
//
// Returns *InterpreterAdapter which is ready for interpreting generated code.
func NewInterpreterAdapter(opts ...InterpreterAdapterOption) *InterpreterAdapter {
	a := &InterpreterAdapter{
		interpreterFactory: nil,
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// Interpret executes generated Go code and returns the template AST.
//
// A panic raised while compiling, initialising or building the template is recovered and
// reported as a failed response; its stack is logged once and kept out of the response.
//
// Takes request (*wasm_dto.InterpretRequest) which contains the generated code and
// configuration.
//
// Returns *wasm_dto.InterpretResponse which contains the template AST and metadata.
// Returns error which is always nil because failures are reported inside the response.
func (a *InterpreterAdapter) Interpret(ctx context.Context, request *wasm_dto.InterpretRequest) (response *wasm_dto.InterpretResponse, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			_, l := logger_domain.From(ctx, log)
			l.Warn("Interpreter panicked while running a playground program",
				logger_domain.String("recovered", fmt.Sprintf("%v", recovered)),
				logger_domain.String("stack", string(debug.Stack())),
			)
			response = newFailedInterpretResponse(fmt.Sprintf("interpreter panicked: %v", recovered), nil)
			err = nil
		}
	}()

	if a.interpreterFactory == nil {
		return newFailedInterpretResponse("interpreter factory not configured", nil), nil
	}

	clearRegisteredBuilders(request)

	interpreter := a.interpreterFactory.NewInterpreter()
	if err := interpreter.CompileAndExecute(ctx, request.GeneratedCode, request.PackagePath, request.Dependencies); err != nil {
		return newFailedInterpretResponse(fmt.Sprintf("batch compilation failed: %v", err), nil), nil
	}

	return a.buildASTResponse(ctx, request)
}

// buildASTResponse retrieves the registered AST function and produces an
// InterpretResponse.
//
// Takes ctx (context.Context) for the request context.
// Takes request (*wasm_dto.InterpretRequest) which contains the package path, request
// URL, and props.
//
// Returns *wasm_dto.InterpretResponse which contains the template AST and metadata.
// Returns error which is always nil because errors are reported inside the response
// struct.
func (*InterpreterAdapter) buildASTResponse(ctx context.Context, request *wasm_dto.InterpretRequest) (*wasm_dto.InterpretResponse, error) {
	astFunc, found := templater_domain.GetASTFunc(request.PackagePath)
	if !found {
		return newFailedInterpretResponse(fmt.Sprintf("BuildAST not registered for package path: %s", request.PackagePath), nil), nil
	}

	requestData := buildMockRequestData(ctx, request.RequestURL)
	defer requestData.Release()

	ast, metadata, runtimeDiags := astFunc(requestData, request.Props)

	diagnostics := convertRuntimeDiagnostics(runtimeDiags)

	if ast == nil && len(diagnostics) > 0 {
		return newFailedInterpretResponse("BuildAST returned nil AST", diagnostics), nil
	}

	return &wasm_dto.InterpretResponse{
		Success:     true,
		AST:         ast,
		Metadata:    &metadata,
		Error:       "",
		Diagnostics: diagnostics,
	}, nil
}

// WithInterpreterFactory sets the interpreter factory for the adapter.
//
// Takes factory (wasm_domain.InterpreterFactoryPort) which creates new interpreter
// instances.
//
// Returns InterpreterAdapterOption which configures the adapter.
func WithInterpreterFactory(factory wasm_domain.InterpreterFactoryPort) InterpreterAdapterOption {
	return func(a *InterpreterAdapter) {
		a.interpreterFactory = factory
	}
}

// clearRegisteredBuilders removes the template functions registered for the request's
// packages by earlier programs, so after the program runs only its own builders can be
// found.
//
// Takes request (*wasm_dto.InterpretRequest) which names the main package and its
// dependencies.
func clearRegisteredBuilders(request *wasm_dto.InterpretRequest) {
	templater_domain.Unregister(request.PackagePath)
	for dependencyPath := range request.Dependencies {
		templater_domain.Unregister(dependencyPath)
	}
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
// Takes diagnostics ([]*generator_dto.RuntimeDiagnostic) which are the diagnostics from
// BuildAST execution.
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
			Code:     d.Code,
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

// newFailedInterpretResponse builds an unsuccessful interpret response.
//
// Takes message (string) which describes the failure.
// Takes diagnostics ([]wasm_dto.Diagnostic) which are any diagnostics gathered before the
// failure, or nil.
//
// Returns *wasm_dto.InterpretResponse which carries no AST or metadata.
func newFailedInterpretResponse(message string, diagnostics []wasm_dto.Diagnostic) *wasm_dto.InterpretResponse {
	return &wasm_dto.InterpretResponse{
		Success:     false,
		AST:         nil,
		Metadata:    nil,
		Error:       message,
		Diagnostics: diagnostics,
	}
}
