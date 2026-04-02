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

package wasm_domain

import (
	"context"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/internal/wasm/wasm_dto"
)

// WASMService defines the main interface for WASM runtime operations.
// This is the driving port that JavaScript calls into.
type WASMService interface {
	// Analyse parses and checks Go source code and returns type
	// information.
	//
	// Takes request (*wasm_dto.AnalyseRequest) which contains the
	// source code to check.
	//
	// Returns *wasm_dto.AnalyseResponse which contains the type information.
	// Returns error when parsing or analysis fails.
	Analyse(ctx context.Context, request *wasm_dto.AnalyseRequest) (*wasm_dto.AnalyseResponse, error)

	// GetCompletions returns code completion suggestions for a given position.
	//
	// Takes request (*wasm_dto.CompletionRequest) which specifies the position and
	// context for completions.
	//
	// Returns *wasm_dto.CompletionResponse which contains the suggested
	// completions.
	// Returns error when the completion request fails.
	GetCompletions(ctx context.Context, request *wasm_dto.CompletionRequest) (*wasm_dto.CompletionResponse, error)

	// GetHover returns hover information for a given position in a file.
	//
	// Takes request (*wasm_dto.HoverRequest) which specifies the file and position.
	//
	// Returns *wasm_dto.HoverResponse which contains the hover information.
	// Returns error when the hover lookup fails.
	GetHover(ctx context.Context, request *wasm_dto.HoverRequest) (*wasm_dto.HoverResponse, error)

	// Validate checks Go source code for errors.
	//
	// Takes request (*wasm_dto.ValidateRequest) which contains the source code to
	// check.
	//
	// Returns *wasm_dto.ValidateResponse which contains the validation results.
	// Returns error when validation fails.
	Validate(ctx context.Context, request *wasm_dto.ValidateRequest) (*wasm_dto.ValidateResponse, error)

	// ParseTemplate parses a PK template and returns its structure.
	//
	// Takes request (*wasm_dto.ParseTemplateRequest) which contains the template to
	// parse.
	//
	// Returns *wasm_dto.ParseTemplateResponse which holds the parsed structure.
	// Returns error when parsing fails.
	ParseTemplate(ctx context.Context, request *wasm_dto.ParseTemplateRequest) (*wasm_dto.ParseTemplateResponse, error)

	// RenderPreview renders a template preview with the given settings.
	//
	// Takes request (*wasm_dto.RenderPreviewRequest) which contains the template and
	// optional properties to use.
	//
	// Returns *wasm_dto.RenderPreviewResponse which contains the rendered output.
	// Returns error when rendering fails.
	RenderPreview(ctx context.Context, request *wasm_dto.RenderPreviewRequest) (*wasm_dto.RenderPreviewResponse, error)

	// GetRuntimeInfo returns details about the WASM runtime.
	//
	// Returns *wasm_dto.RuntimeInfo which contains the runtime details.
	// Returns error when the runtime information cannot be retrieved.
	GetRuntimeInfo(ctx context.Context) (*wasm_dto.RuntimeInfo, error)

	// Generate produces code artefacts from in-memory sources.
	//
	// Takes ctx (context.Context) which is the request context.
	// Takes request (*wasm_dto.GenerateFromSourcesRequest) which contains the source
	// files and configuration.
	//
	// Returns *wasm_dto.GenerateFromSourcesResponse which contains the generated
	// artefacts and manifest.
	// Returns error when the generator is not configured.
	Generate(ctx context.Context, request *wasm_dto.GenerateFromSourcesRequest) (*wasm_dto.GenerateFromSourcesResponse, error)

	// Render produces HTML from in-memory sources.
	// This method only supports static templates (no Go code execution).
	//
	// Takes ctx (context.Context) which is the request context.
	// Takes request (*wasm_dto.RenderFromSourcesRequest) which contains the source
	// files and configuration.
	//
	// Returns *wasm_dto.RenderFromSourcesResponse which contains the rendered HTML.
	// Returns error when the renderer is not configured.
	Render(ctx context.Context, request *wasm_dto.RenderFromSourcesRequest) (*wasm_dto.RenderFromSourcesResponse, error)
}

// StdlibLoaderPort defines the interface for loading pre-bundled stdlib type data.
type StdlibLoaderPort interface {
	// Load returns the pre-bundled stdlib TypeData.
	// The data is loaded from an embedded FlatBuffer file.
	//
	// Returns *TypeData which contains the standard library type information.
	// Returns error when loading fails.
	Load() (*inspector_dto.TypeData, error)

	// GetPackageList returns the list of standard library packages.
	//
	// Returns []string which contains the available package names.
	GetPackageList() []string
}

// JSInteropPort defines the interface for JavaScript interoperability.
// It abstracts syscall/js to allow testing without a JavaScript runtime.
type JSInteropPort interface {
	// RegisterFunction registers a Go function to be callable from JavaScript.
	//
	// Takes name (string) which is the name used to call the function from
	// JavaScript.
	// Takes handler (func(arguments []any) (any, error)) which is the Go function to
	// register.
	RegisterFunction(name string, handler func(arguments []any) (any, error))

	// Log writes a message to the JavaScript console.
	//
	// Takes level (string) which specifies the log level.
	// Takes message (string) which is the text to log.
	// Takes arguments (...any) which provides values for format placeholders.
	Log(level string, message string, arguments ...any)

	// MarshalToJS converts a Go value to a JavaScript-compatible form.
	//
	// Takes v (any) which is the Go value to convert.
	//
	// Returns any which is the JavaScript-compatible representation.
	// Returns error when the conversion fails.
	MarshalToJS(v any) (any, error)

	// UnmarshalFromJS converts a JavaScript value to a Go type.
	//
	// Takes jsValue (any) which is the JavaScript value to convert.
	// Takes target (any) which is a pointer to the Go value to populate.
	//
	// Returns error when the conversion fails.
	UnmarshalFromJS(jsValue any, target any) error
}

// ConsolePort provides console output for WASM modules.
// It replaces standard logging with JavaScript console output.
type ConsolePort interface {
	// Debug logs a message at debug level.
	//
	// Takes message (string) which is the message to log.
	// Takes arguments (...any) which are values for formatting the message.
	Debug(message string, arguments ...any)

	// Info logs a message at the info level.
	//
	// Takes message (string) which is the message to log.
	// Takes arguments (...any) which are optional values to format into the message.
	Info(message string, arguments ...any)

	// Warn logs a warning message with optional formatting arguments.
	//
	// Takes message (string) which is the warning message or format string.
	// Takes arguments (...any) which are optional values to format into the message.
	Warn(message string, arguments ...any)

	// Error logs an error message.
	//
	// Takes message (string) which is the message to log.
	// Takes arguments (...any) which are values to format into the message.
	Error(message string, arguments ...any)
}

// GeneratorPort provides code generation capabilities for WASM.
// It allows generating Go code from in-memory PK template sources without
// requiring file system access.
type GeneratorPort interface {
	// Generate produces code artefacts from in-memory sources.
	//
	// Takes ctx (context.Context) which is the request context.
	// Takes request (*wasm_dto.GenerateFromSourcesRequest) which contains the source
	// files and configuration.
	//
	// Returns *wasm_dto.GenerateFromSourcesResponse which contains the generated
	// artefacts and manifest.
	// Returns error when generation fails.
	Generate(ctx context.Context, request *wasm_dto.GenerateFromSourcesRequest) (*wasm_dto.GenerateFromSourcesResponse, error)
}

const (
	// defaultMaxSourceSize is the default limit for source code size, set to 1 MiB.
	defaultMaxSourceSize = 1024 * 1024

	// defaultPlaygroundModuleName is the default module name used in playground mode.
	defaultPlaygroundModuleName = "playground"
)

// Config holds settings for the WASM orchestrator.
type Config struct {
	// DefaultModuleName is the module name used when none is given.
	DefaultModuleName string

	// StdlibPackages lists the standard library packages to include.
	// If empty, uses the default set.
	StdlibPackages []string

	// MaxSourceSize is the maximum size of source code in bytes.
	MaxSourceSize int

	// EnableMetrics enables the collection of OpenTelemetry metrics.
	EnableMetrics bool
}

// Option is a function that configures the WASM orchestrator.
type Option func(*Orchestrator)

// RenderPort provides HTML rendering capabilities for WASM.
// It allows rendering PK templates to HTML strings without requiring
// file system access or Go code execution (static templates only).
type RenderPort interface {
	// Render produces HTML from in-memory sources.
	//
	// Takes ctx (context.Context) which is the request context.
	// Takes request (*wasm_dto.RenderFromSourcesRequest) which contains the source
	// files and configuration.
	//
	// Returns *wasm_dto.RenderFromSourcesResponse which contains the rendered HTML.
	// Returns error when rendering fails.
	Render(ctx context.Context, request *wasm_dto.RenderFromSourcesRequest) (*wasm_dto.RenderFromSourcesResponse, error)

	// RenderFromAST produces HTML from a pre-built TemplateAST.
	// This is used for dynamic rendering where the AST comes from
	// interpreter execution rather than annotation.
	//
	// Takes ctx (context.Context) which is the request context.
	// Takes request (*wasm_dto.RenderFromASTRequest) which contains the AST and
	// metadata.
	//
	// Returns *wasm_dto.RenderFromASTResponse which contains the rendered HTML.
	// Returns error when rendering fails.
	RenderFromAST(ctx context.Context, request *wasm_dto.RenderFromASTRequest) (*wasm_dto.RenderFromASTResponse, error)
}

// InterpreterPort provides Go code interpretation capabilities for WASM.
// It executes generated Go code to produce template ASTs with evaluated
// expressions.
type InterpreterPort interface {
	// Interpret executes generated Go code and returns the template AST.
	//
	// Takes ctx (context.Context) which is the request context.
	// Takes request (*wasm_dto.InterpretRequest) which contains the generated code
	// and configuration.
	//
	// Returns *wasm_dto.InterpretResponse which contains the template AST and
	// metadata.
	// Returns error when interpretation fails.
	Interpret(ctx context.Context, request *wasm_dto.InterpretRequest) (*wasm_dto.InterpretResponse, error)
}

// SymbolLoaderPort abstracts symbol loading for WASM interpreters,
// letting wasm_adapters accept symbol providers without depending on a
// concrete interpreter implementation.
type SymbolLoaderPort interface {
	// Use loads symbols into an interpreter.
	//
	// Takes interp (any) which is the interpreter to load symbols into.
	// The concrete type depends on the interpreter implementation.
	//
	// Returns error when symbol loading fails.
	Use(interp any) error
}

// InterpreterFactoryPort creates fresh interpreter instances.
// This abstracts interpreter creation so the adapter does not need
// to import interpreter implementations directly.
type InterpreterFactoryPort interface {
	// NewInterpreter creates a new interpreter instance.
	//
	// Returns any which is the interpreter instance. The concrete type
	// depends on the interpreter implementation.
	NewInterpreter() any
}

// HeadlessRendererPort provides AST-to-HTML rendering without HTTP context.
// This is implemented by the main render orchestrator and used by the WASM
// adapter for headless rendering scenarios.
type HeadlessRendererPort interface {
	// RenderASTToString renders an AST to an HTML string without requiring
	// HTTP context.
	//
	// Takes ctx (context.Context) which provides cancellation.
	// Takes opts (HeadlessRenderOptions) which configures the rendering.
	//
	// Returns string which contains the rendered HTML.
	// Returns error when rendering fails.
	RenderASTToString(ctx context.Context, opts HeadlessRenderOptions) (string, error)
}

// HeadlessRenderOptions contains options for headless AST rendering.
type HeadlessRenderOptions struct {
	// Template is the parsed AST to render.
	Template *ast_domain.TemplateAST

	// Metadata holds page details such as title, description, and language.
	Metadata *templater_dto.InternalMetadata

	// Styling specifies the CSS styles to include in the output.
	Styling string

	// IncludeDocumentWrapper determines whether to wrap output in full HTML
	// document structure.
	IncludeDocumentWrapper bool
}

// DefaultConfig returns the default configuration for WASM execution.
//
// Returns Config which contains sensible defaults for running WASM code.
func DefaultConfig() Config {
	return Config{
		StdlibPackages:    nil,
		DefaultModuleName: defaultPlaygroundModuleName,
		MaxSourceSize:     defaultMaxSourceSize,
		EnableMetrics:     false,
	}
}

// WithConfig sets the orchestrator configuration.
//
// Takes config (Config) which specifies the orchestrator settings.
//
// Returns Option which applies the configuration to an Orchestrator.
func WithConfig(config Config) Option {
	return func(o *Orchestrator) {
		o.config = config
	}
}

// WithStdlibLoader sets the stdlib loader adapter.
//
// Takes loader (StdlibLoaderPort) which provides the standard library loader.
//
// Returns Option which configures the Orchestrator with the given loader.
func WithStdlibLoader(loader StdlibLoaderPort) Option {
	return func(o *Orchestrator) {
		o.stdlibLoader = loader
	}
}

// WithJSInterop sets the JavaScript interop adapter.
//
// Takes interop (JSInteropPort) which provides the adapter for JavaScript
// interoperability.
//
// Returns Option which configures the orchestrator with the given adapter.
func WithJSInterop(interop JSInteropPort) Option {
	return func(o *Orchestrator) {
		o.jsInterop = interop
	}
}

// WithConsole sets the console output adapter.
//
// Takes console (ConsolePort) which provides the output interface for console
// messages.
//
// Returns Option which configures the Orchestrator with the given console.
func WithConsole(console ConsolePort) Option {
	return func(o *Orchestrator) {
		o.console = console
	}
}

// WithGenerator sets the code generator adapter.
//
// Takes generator (GeneratorPort) which provides code generation capabilities.
//
// Returns Option which configures the Orchestrator with the given generator.
func WithGenerator(generator GeneratorPort) Option {
	return func(o *Orchestrator) {
		o.generator = generator
	}
}

// WithRenderer sets the HTML renderer adapter.
//
// Takes renderer (RenderPort) which provides HTML rendering capabilities.
//
// Returns Option which configures the Orchestrator with the given renderer.
func WithRenderer(renderer RenderPort) Option {
	return func(o *Orchestrator) {
		o.renderer = renderer
	}
}

// WithInterpreter sets the Go code interpreter adapter.
//
// Takes interpreter (InterpreterPort) which provides code interpretation
// capabilities.
//
// Returns Option which configures the Orchestrator with the given interpreter.
func WithInterpreter(interpreter InterpreterPort) Option {
	return func(o *Orchestrator) {
		o.interpreter = interpreter
	}
}
