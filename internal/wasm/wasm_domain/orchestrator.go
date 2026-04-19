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
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/internal/wasm/wasm_dto"
)

const (
	// pathSeparator is the forward slash used to build module paths.
	pathSeparator = "/"

	// severityError is the severity level for error diagnostics.
	severityError = "error"

	// defaultCompletionCapacity is the starting size for the completion items
	// slice.
	defaultCompletionCapacity = 50

	// logLevelDebug is the debug log level for detailed diagnostic messages.
	logLevelDebug = "debug"

	// logLevelInfo is the info severity level for log messages.
	logLevelInfo = "info"

	// logLevelWarn is the log level for warning messages.
	logLevelWarn = "warn"

	// logLevelError is the log level for error conditions.
	logLevelError = "error"

	// regexGroupLine is the index for the line number in the regex match.
	regexGroupLine = 2

	// regexGroupColumn is the index for the column number capture group.
	regexGroupColumn = 3

	// regexGroupMessage is the index for the error message capture group.
	regexGroupMessage = 4

	// regexMinMatchesForPosition is the minimum number of matches required for
	// position info.
	regexMinMatchesForPosition = 4

	// regexMinMatchesForMessage is the minimum number of matches required for
	// message extraction.
	regexMinMatchesForMessage = 5

	// previewEntryPoint is the virtual file path used for the preview template.
	previewEntryPoint = "pages/preview.pk"

	// minCatchAllSegmentLength is the minimum length of a catch-all route
	// segment like {x*} (opening brace + at least one char + star + closing brace).
	minCatchAllSegmentLength = 3
)

var (
	// version holds the current Piko version. This is set at build time.
	version = "dev"

	// errOrchestratorNotInitialised is returned when an operation is attempted
	// on the WASM orchestrator before it has been initialised.
	errOrchestratorNotInitialised = errors.New("orchestrator not initialised")

	// errStdlibLoaderNotConfigured is returned when a stdlib import is
	// requested but the stdlib loader has not been configured.
	errStdlibLoaderNotConfigured = errors.New("stdlib loader not configured")

	// errRendererNotConfigured is returned when a render operation is
	// requested but the renderer has not been configured.
	errRendererNotConfigured = errors.New("renderer not configured")

	// errorPosRegex matches Go error format: "file:line:col: message" or
	// "file:line: message".
	errorPosRegex = regexp.MustCompile(`^([^:]+):(\d+):(\d+)?:?\s*(.*)$`)
)

// Orchestrator manages WASM runtime tasks and implements WASMService.
// It uses the inspector's lite builder for type analysis and provides
// a simple API for JavaScript code.
type Orchestrator struct {
	// stdlibLoader loads standard library package data.
	stdlibLoader StdlibLoaderPort

	// jsInterop provides JavaScript interoperability.
	jsInterop JSInteropPort

	// console handles log output at different levels; nil turns off logging.
	console ConsolePort

	// generator provides code generation; nil if not configured.
	generator GeneratorPort

	// renderer handles HTML rendering; nil if not set up.
	renderer RenderPort

	// interpreter provides Go code interpretation; nil if not configured.
	interpreter InterpreterPort

	// stdlibData holds the cached standard library type data loaded at startup.
	stdlibData *inspector_dto.TypeData

	// config holds the orchestrator settings.
	config Config

	// mu guards the initialised flag and stdlibData fields.
	mu sync.RWMutex

	// initialised indicates whether the orchestrator has completed setup.
	initialised bool
}

// NewOrchestrator creates a new WASM orchestrator with the given options.
//
// Takes opts (...Option) which configures the orchestrator behaviour.
//
// Returns *Orchestrator which is ready to use with default settings.
func NewOrchestrator(opts ...Option) *Orchestrator {
	o := &Orchestrator{
		config: DefaultConfig(),
	}

	for _, opt := range opts {
		opt(o)
	}

	return o
}

// Initialise prepares the orchestrator by loading the stdlib data. Must be
// called before any other methods.
//
// Returns error when the stdlib loader is not configured or fails to load.
//
// Safe for concurrent use.
func (o *Orchestrator) Initialise(_ context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.initialised {
		return nil
	}

	if o.stdlibLoader == nil {
		return errStdlibLoaderNotConfigured
	}

	stdlibData, err := o.stdlibLoader.Load()
	if err != nil {
		return fmt.Errorf("failed to load stdlib data: %w", err)
	}
	o.stdlibData = stdlibData

	o.log("info", "WASM orchestrator initialised",
		"stdlib_packages", len(stdlibData.Packages))

	o.initialised = true
	return nil
}

// Analyse parses and analyses Go source code.
//
// Takes request (*wasm_dto.AnalyseRequest) which contains the source code and
// configuration for analysis.
//
// Returns *wasm_dto.AnalyseResponse which contains the analysis results.
// Returns error when stdlib data cannot be loaded.
func (o *Orchestrator) Analyse(ctx context.Context, request *wasm_dto.AnalyseRequest) (*wasm_dto.AnalyseResponse, error) {
	startTime := time.Now()
	analyseCount.Add(ctx, 1)

	defer func() {
		analyseDuration.Record(ctx, float64(time.Since(startTime).Milliseconds()))
	}()

	stdlibData, err := o.GetStdlibData()
	if err != nil {
		return nil, fmt.Errorf("loading stdlib data: %w", err)
	}

	if response := validateAnalyseRequest(ctx, request, o.config.MaxSourceSize); response != nil {
		return response, nil
	}

	moduleName := request.ModuleName
	if moduleName == "" {
		moduleName = o.config.DefaultModuleName
	}

	sourceBytes := prepareSourceBytes(request.Sources, moduleName)

	return runAnalysis(ctx, stdlibData, sourceBytes, moduleName)
}

// GetCompletions returns code completion suggestions.
//
// Takes request (*wasm_dto.CompletionRequest) which specifies the source code and
// cursor position for completion.
//
// Returns *wasm_dto.CompletionResponse which contains the completion items.
// Returns error when the orchestrator has not been initialised.
//
// Safe for concurrent use.
func (o *Orchestrator) GetCompletions(ctx context.Context, request *wasm_dto.CompletionRequest) (*wasm_dto.CompletionResponse, error) {
	completionCount.Add(ctx, 1)
	startTime := time.Now()
	defer func() {
		completionDuration.Record(ctx, float64(time.Since(startTime).Milliseconds()))
	}()

	o.mu.RLock()
	if !o.initialised {
		o.mu.RUnlock()
		return nil, errOrchestratorNotInitialised
	}
	stdlibData := o.stdlibData
	o.mu.RUnlock()

	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "main.go", request.Source, parser.ParseComments)

	items := findCompletionsAtPosition(file, request.Source, request.Line, request.Column, stdlibData)

	return &wasm_dto.CompletionResponse{
		Success: true,
		Items:   items,
		Error:   "",
	}, nil
}

// GetHover returns hover information at a position.
//
// Takes request (*wasm_dto.HoverRequest) which specifies the source code and
// cursor position for hover lookup.
//
// Returns *wasm_dto.HoverResponse which contains the hover content and range.
// Returns error when the orchestrator is not initialised.
//
// Safe for concurrent use. Uses a read lock to access shared state.
func (o *Orchestrator) GetHover(ctx context.Context, request *wasm_dto.HoverRequest) (*wasm_dto.HoverResponse, error) {
	hoverCount.Add(ctx, 1)

	o.mu.RLock()
	if !o.initialised {
		o.mu.RUnlock()
		return nil, errOrchestratorNotInitialised
	}
	stdlibData := o.stdlibData
	o.mu.RUnlock()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", request.Source, parser.ParseComments)
	if err != nil {
		return &wasm_dto.HoverResponse{
			Success: false,
			Content: "",
			Range:   nil,
			Error:   fmt.Sprintf("parse error: %v", err),
		}, nil
	}

	tokFile := fset.File(file.Pos())
	offset := lineColumnToOffset(tokFile, request.Line, request.Column)
	position := tokFile.Pos(offset)
	target := findHoverTarget(file, position)
	if target == nil {
		return &wasm_dto.HoverResponse{
			Success: true,
			Content: "",
			Range:   nil,
			Error:   "",
		}, nil
	}

	var content string
	if target.selector != nil {
		if pkgIdent, ok := target.selector.X.(*ast.Ident); ok {
			content = getQualifiedHoverContent(pkgIdent.Name, target.identifier.Name, file, stdlibData)
		}
	} else {
		content = getHoverContent(target.identifier.Name, file, stdlibData)
	}

	return &wasm_dto.HoverResponse{
		Success: true,
		Content: content,
		Range:   nil,
		Error:   "",
	}, nil
}

// Validate runs a quick syntactic check on Go source code.
//
// Takes request (*wasm_dto.ValidateRequest) which contains the source code and
// file path to validate.
//
// Returns *wasm_dto.ValidateResponse which indicates whether the source is
// valid and includes any syntax error diagnostics.
// Returns error when validation cannot be performed.
func (*Orchestrator) Validate(_ context.Context, request *wasm_dto.ValidateRequest) (*wasm_dto.ValidateResponse, error) {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, request.FilePath, request.Source, parser.AllErrors)

	if err != nil {
		return &wasm_dto.ValidateResponse{
			Valid:       false,
			Diagnostics: parseErrorToDiagnostics(err, request.FilePath),
		}, nil
	}

	return &wasm_dto.ValidateResponse{
		Valid:       true,
		Diagnostics: nil,
	}, nil
}

// ParseTemplate parses a PK template.
//
// Takes request (*wasm_dto.ParseTemplateRequest) which contains the template to
// parse.
//
// Returns *wasm_dto.ParseTemplateResponse which contains the parsed AST or
// error details.
// Returns error when an unexpected failure occurs.
func (*Orchestrator) ParseTemplate(ctx context.Context, request *wasm_dto.ParseTemplateRequest) (*wasm_dto.ParseTemplateResponse, error) {
	parseTemplateCount.Add(ctx, 1)

	result, err := sfcparser.Parse([]byte(request.Template))
	if err != nil {
		return &wasm_dto.ParseTemplateResponse{
			Success:     false,
			AST:         nil,
			Diagnostics: nil,
			Error:       fmt.Sprintf("parse error: %v", err),
		}, nil
	}

	return &wasm_dto.ParseTemplateResponse{
		Success:     true,
		AST:         convertSFCResultToAST(result),
		Diagnostics: nil,
		Error:       "",
	}, nil
}

// RenderPreview renders a template preview by delegating to the static
// renderer. The template and optional script are assembled into a
// RenderFromSourcesRequest and passed through the existing render pipeline.
//
// Takes request (*wasm_dto.RenderPreviewRequest) which contains the template and
// optional script to render.
//
// Returns *wasm_dto.RenderPreviewResponse which contains the rendered HTML,
// CSS, and any diagnostics.
// Returns error when an unexpected failure occurs.
func (o *Orchestrator) RenderPreview(ctx context.Context, request *wasm_dto.RenderPreviewRequest) (*wasm_dto.RenderPreviewResponse, error) {
	renderPreviewCount.Add(ctx, 1)

	if o.renderer == nil {
		return &wasm_dto.RenderPreviewResponse{
			Success: false,
			Error:   "renderer not configured",
		}, nil
	}

	sources := map[string]string{
		previewEntryPoint: request.Template,
	}

	if request.Script != "" {
		sources[previewEntryPoint] = assemblePreviewTemplate(request.Template, request.Script)
	}

	renderReq := &wasm_dto.RenderFromSourcesRequest{
		Sources:    sources,
		ModuleName: request.ModuleName,
		EntryPoint: previewEntryPoint,
	}

	result, err := o.renderer.Render(ctx, renderReq)
	if err != nil {
		return &wasm_dto.RenderPreviewResponse{
			Success: false,
			Error:   fmt.Sprintf("preview rendering failed: %v", err),
		}, nil
	}

	diagnostics := make([]wasm_dto.Diagnostic, len(result.Diagnostics))
	copy(diagnostics, result.Diagnostics)

	return &wasm_dto.RenderPreviewResponse{
		Success:     result.Success,
		HTML:        result.HTML,
		CSS:         result.CSS,
		Diagnostics: diagnostics,
		Error:       result.Error,
	}, nil
}

// GetRuntimeInfo returns information about the WASM runtime.
//
// Returns *wasm_dto.RuntimeInfo which contains version, Go version, standard
// library packages, and available capabilities.
// Returns error which is always nil but included for interface compatibility.
//
// Safe for concurrent use; acquires a read lock on the orchestrator.
func (o *Orchestrator) GetRuntimeInfo(_ context.Context) (*wasm_dto.RuntimeInfo, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	var stdlibPackages []string
	if o.stdlibLoader != nil {
		stdlibPackages = o.stdlibLoader.GetPackageList()
	}

	capabilities := []string{
		"analyse",
		"completions",
		"hover",
		"validate",
	}

	if o.generator != nil {
		capabilities = append(capabilities, "generate")
	}

	return &wasm_dto.RuntimeInfo{
		Version:        version,
		GoVersion:      runtime.Version(),
		StdlibPackages: stdlibPackages,
		Capabilities:   capabilities,
	}, nil
}

// operationInstrumentation bundles the three OpenTelemetry instruments used by
// instrumentedOperation so callers pass a single value rather than three separate
// parameters.
type operationInstrumentation struct {
	// counter tracks the total number of completed operations.
	counter metric.Int64Counter

	// histogram records the duration distribution of operations.
	histogram metric.Float64Histogram

	// errorCounter tracks the total number of failed operations.
	errorCounter metric.Int64Counter
}

// instrumentedOperation handles the common instrumentation, nil-adapter check,
// call delegation, and error counting shared by Generate and Render.
//
// Takes ctx (context.Context) for metric recording.
// Takes instruments (operationInstrumentation) holding the counter, histogram and
// error counter.
// Takes adapter (any) checked for nil; a nil adapter is treated as not configured.
// Takes notConfiguredResponse (*Response) returned when adapter is nil.
// Takes delegate (func() (*Response, error)) performing the actual operation.
// Takes onFailure (func(error) *Response) constructing an error response on
// delegate failure.
// Takes isSuccess (func(*Response) bool) indicating whether the response succeeded.
//
// Returns (*Response, error); the error is always nil -- failures are encoded in
// the response.
func instrumentedOperation[Response any](
	ctx context.Context,
	instruments operationInstrumentation,
	adapter any,
	notConfiguredResponse *Response,
	delegate func() (*Response, error),
	onFailure func(error) *Response,
	isSuccess func(*Response) bool,
) (*Response, error) {
	instruments.counter.Add(ctx, 1)
	startTime := time.Now()
	defer func() {
		instruments.histogram.Record(ctx, float64(time.Since(startTime).Milliseconds()))
	}()

	if adapter == nil {
		instruments.errorCounter.Add(ctx, 1)
		return notConfiguredResponse, nil
	}

	response, err := delegate()
	if err != nil {
		instruments.errorCounter.Add(ctx, 1)
		return onFailure(err), nil
	}

	if !isSuccess(response) {
		instruments.errorCounter.Add(ctx, 1)
	}

	return response, nil
}

// Generate produces code artefacts from in-memory sources.
//
// Takes request (*wasm_dto.GenerateFromSourcesRequest) which contains the source
// files and configuration.
//
// Returns *wasm_dto.GenerateFromSourcesResponse which contains the generated
// artefacts and manifest.
// Returns error when the generator is not configured.
func (o *Orchestrator) Generate(ctx context.Context, request *wasm_dto.GenerateFromSourcesRequest) (*wasm_dto.GenerateFromSourcesResponse, error) {
	return instrumentedOperation(
		ctx,
		operationInstrumentation{generateCount, generateDuration, generateErrorCount},
		o.generator,
		&wasm_dto.GenerateFromSourcesResponse{Success: false, Error: "generator not configured"},
		func() (*wasm_dto.GenerateFromSourcesResponse, error) { return o.generator.Generate(ctx, request) },
		func(err error) *wasm_dto.GenerateFromSourcesResponse {
			return &wasm_dto.GenerateFromSourcesResponse{Success: false, Error: fmt.Sprintf("generation failed: %v", err)}
		},
		func(r *wasm_dto.GenerateFromSourcesResponse) bool { return r.Success },
	)
}

// Render produces HTML from in-memory sources.
// Only supports static templates (no Go code execution).
//
// Takes request (*wasm_dto.RenderFromSourcesRequest) which contains the source
// files and configuration.
//
// Returns *wasm_dto.RenderFromSourcesResponse which contains the rendered HTML.
// Returns error when the renderer is not configured.
func (o *Orchestrator) Render(ctx context.Context, request *wasm_dto.RenderFromSourcesRequest) (*wasm_dto.RenderFromSourcesResponse, error) {
	return instrumentedOperation(
		ctx,
		operationInstrumentation{renderCount, renderDuration, renderErrorCount},
		o.renderer,
		&wasm_dto.RenderFromSourcesResponse{Success: false, Error: "renderer not configured"},
		func() (*wasm_dto.RenderFromSourcesResponse, error) { return o.renderer.Render(ctx, request) },
		func(err error) *wasm_dto.RenderFromSourcesResponse {
			return &wasm_dto.RenderFromSourcesResponse{Success: false, Error: fmt.Sprintf("rendering failed: %v", err)}
		},
		func(r *wasm_dto.RenderFromSourcesResponse) bool { return r.Success },
	)
}

// DynamicRender produces HTML by generating Go code from PK templates,
// interpreting the code, and rendering the resulting AST.
//
// Takes request (*wasm_dto.DynamicRenderRequest) which contains the source files,
// module name, and request URL.
//
// Returns *wasm_dto.DynamicRenderResponse which contains the rendered HTML.
// Returns error when generation, interpretation, or rendering fails.
func (o *Orchestrator) DynamicRender(ctx context.Context, request *wasm_dto.DynamicRenderRequest) (*wasm_dto.DynamicRenderResponse, error) {
	dynamicRenderCount.Add(ctx, 1)
	startTime := time.Now()
	defer func() {
		dynamicRenderDuration.Record(ctx, float64(time.Since(startTime).Milliseconds()))
	}()

	if errResp := o.validateDynamicRenderAdapters(ctx); errResp != nil {
		return errResp, nil
	}

	genResp, errResp := o.dynamicRenderGenerate(ctx, request)
	if errResp != nil {
		return errResp, nil
	}

	interpResp, errResp := o.dynamicRenderInterpret(ctx, request, genResp)
	if errResp != nil {
		return errResp, nil
	}

	return o.dynamicRenderHTML(ctx, interpResp, genResp, request.RequestURL)
}

// GetStdlibData retrieves the stdlib data under lock.
//
// Returns *inspector_dto.TypeData which contains the standard library type
// information.
// Returns error when the orchestrator has not been initialised.
//
// Safe for concurrent use; protected by a read lock.
func (o *Orchestrator) GetStdlibData() (*inspector_dto.TypeData, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.initialised {
		return nil, errOrchestratorNotInitialised
	}
	return o.stdlibData, nil
}

// validateDynamicRenderAdapters checks that required adapters are configured.
//
// Returns *wasm_dto.DynamicRenderResponse which contains an error message when
// a required adapter is missing, or nil when all adapters are configured.
func (o *Orchestrator) validateDynamicRenderAdapters(ctx context.Context) *wasm_dto.DynamicRenderResponse {
	if o.generator == nil {
		dynamicRenderErrorCount.Add(ctx, 1)
		return &wasm_dto.DynamicRenderResponse{Success: false, Error: "generator not configured"}
	}
	if o.interpreter == nil {
		dynamicRenderErrorCount.Add(ctx, 1)
		return &wasm_dto.DynamicRenderResponse{Success: false, Error: "interpreter not configured"}
	}
	return nil
}

// dynamicRenderGenerate performs code generation and finds the page artefact.
//
// Takes request (*wasm_dto.DynamicRenderRequest) which contains the sources and
// module name for code generation.
//
// Returns *wasm_dto.GenerateFromSourcesResponse which contains the generation
// result when successful, or nil on failure.
// Returns *wasm_dto.DynamicRenderResponse which contains an error response on
// failure, or nil on success.
func (o *Orchestrator) dynamicRenderGenerate(ctx context.Context, request *wasm_dto.DynamicRenderRequest) (*wasm_dto.GenerateFromSourcesResponse, *wasm_dto.DynamicRenderResponse) {
	o.log(logLevelDebug, "DynamicRender: Phase 1 - Generating code...")
	genResp, err := o.generator.Generate(ctx, &wasm_dto.GenerateFromSourcesRequest{
		Sources:    request.Sources,
		ModuleName: request.ModuleName,
	})
	if err != nil {
		dynamicRenderErrorCount.Add(ctx, 1)
		return nil, &wasm_dto.DynamicRenderResponse{
			Success: false,
			Error:   fmt.Sprintf("code generation failed: %v", err),
		}
	}
	if !genResp.Success {
		dynamicRenderErrorCount.Add(ctx, 1)
		return nil, &wasm_dto.DynamicRenderResponse{Success: false, Error: genResp.Error}
	}
	return genResp, nil
}

// dynamicRenderInterpret interprets the generated code.
//
// Takes request (*wasm_dto.DynamicRenderRequest) which contains the request URL
// and props for rendering.
// Takes genResp (*wasm_dto.GenerateFromSourcesResponse) which provides the
// generated artefacts and manifest to interpret.
//
// Returns *wasm_dto.InterpretResponse which contains the interpretation result
// on success, or nil on failure.
// Returns *wasm_dto.DynamicRenderResponse which contains error details when
// interpretation fails, or nil on success.
func (o *Orchestrator) dynamicRenderInterpret(
	ctx context.Context,
	request *wasm_dto.DynamicRenderRequest,
	genResp *wasm_dto.GenerateFromSourcesResponse,
) (*wasm_dto.InterpretResponse, *wasm_dto.DynamicRenderResponse) {
	pageArtefact, packagePath, deps := findPageArtefactForURL(genResp.Artefacts, genResp.Manifest, request.RequestURL, request.ModuleName)
	if pageArtefact == nil {
		dynamicRenderErrorCount.Add(ctx, 1)
		return nil, &wasm_dto.DynamicRenderResponse{
			Success: false,
			Error:   fmt.Sprintf("no page found for URL: %s", request.RequestURL),
		}
	}

	o.log(logLevelDebug, "DynamicRender: Phase 2 - Interpreting code...")
	interpResp, err := o.interpreter.Interpret(ctx, &wasm_dto.InterpretRequest{
		GeneratedCode: pageArtefact.Content,
		PackagePath:   packagePath,
		Dependencies:  deps,
		RequestURL:    request.RequestURL,
		Props:         request.Props,
	})
	if err != nil {
		dynamicRenderErrorCount.Add(ctx, 1)
		return nil, &wasm_dto.DynamicRenderResponse{
			Success:     false,
			Error:       fmt.Sprintf("interpretation failed: %v", err),
			Diagnostics: interpResp.Diagnostics,
		}
	}
	if !interpResp.Success {
		dynamicRenderErrorCount.Add(ctx, 1)
		return nil, &wasm_dto.DynamicRenderResponse{
			Success:     false,
			Error:       interpResp.Error,
			Diagnostics: interpResp.Diagnostics,
		}
	}
	return interpResp, nil
}

// dynamicRenderHTML renders the interpreted AST to HTML and assembles the
// full response, including any compiled client-side JavaScript artefacts
// captured during code generation and the matched page's CSS.
//
// Takes interpResp (*wasm_dto.InterpretResponse) which contains the AST and
// metadata from the interpretation phase.
// Takes genResp (*wasm_dto.GenerateFromSourcesResponse) which provides the
// generated artefacts; ArtefactTypeJS entries are surfaced verbatim under
// DynamicRenderResponse.Scripts and the matched page's StyleBlock becomes
// resp.CSS.
// Takes requestURL (string) which identifies the page whose CSS should be
// used; matched against the manifest's route patterns.
//
// Returns *wasm_dto.DynamicRenderResponse which contains the rendered HTML,
// CSS, scripts, and runtime-import URLs, or error details if rendering
// failed.
// Returns error when an unexpected failure occurs.
func (o *Orchestrator) dynamicRenderHTML(
	ctx context.Context,
	interpResp *wasm_dto.InterpretResponse,
	genResp *wasm_dto.GenerateFromSourcesResponse,
	requestURL string,
) (*wasm_dto.DynamicRenderResponse, error) {
	o.log(logLevelDebug, "DynamicRender: Phase 3 - Rendering HTML...")
	styleBlock := findPageStyleBlock(genResp, requestURL)
	html, renderErr := o.renderASTToHTML(ctx, interpResp.AST, interpResp.Metadata, styleBlock)
	if renderErr != nil {
		dynamicRenderErrorCount.Add(ctx, 1)
		return &wasm_dto.DynamicRenderResponse{
			Success:     false,
			Error:       fmt.Sprintf("rendering failed: %v", renderErr),
			Diagnostics: interpResp.Diagnostics,
		}, nil
	}

	return &wasm_dto.DynamicRenderResponse{
		Success:        true,
		HTML:           html,
		CSS:            styleBlock,
		Scripts:        collectScriptArtefacts(genResp),
		RuntimeImports: defaultRuntimeImports,
		Diagnostics:    interpResp.Diagnostics,
	}, nil
}

// renderASTToHTML converts a template AST to HTML using the WASM renderer.
// CSS is propagated through the renderer for any in-AST styling logic but
// the dynamic-render path treats styleBlock as the source of truth, so the
// renderer's CSS echo is intentionally not returned.
//
// Takes ctx (context.Context) which is the request context.
// Takes astNode (*ast_domain.TemplateAST) which is the template AST to render.
// Takes metadata (*templater_dto.InternalMetadata) which contains template
// metadata.
// Takes styleBlock (string) which is the CSS to apply during rendering.
//
// Returns html (string) which is the rendered HTML body markup (no document
// wrapper).
// Returns err (error) when rendering fails.
func (o *Orchestrator) renderASTToHTML(
	ctx context.Context,
	astNode *ast_domain.TemplateAST,
	metadata *templater_dto.InternalMetadata,
	styleBlock string,
) (string, error) {
	if o.renderer == nil {
		return "", errRendererNotConfigured
	}

	response, err := o.renderer.RenderFromAST(ctx, &wasm_dto.RenderFromASTRequest{
		AST:      astNode,
		Metadata: metadata,
		CSS:      styleBlock,
	})
	if err != nil {
		return "", fmt.Errorf("rendering AST to HTML: %w", err)
	}

	if !response.Success {
		return "", errors.New(response.Error)
	}

	return response.HTML, nil
}

// log writes a message to the console at the given level.
//
// Takes level (string) which sets the log level (debug, info, warn, or error).
// Takes message (string) which is the message to write.
// Takes arguments (...any) which provides optional values for formatting.
func (o *Orchestrator) log(level, message string, arguments ...any) {
	if o.console != nil {
		switch level {
		case logLevelDebug:
			o.console.Debug(message, arguments...)
		case logLevelInfo:
			o.console.Info(message, arguments...)
		case logLevelWarn:
			o.console.Warn(message, arguments...)
		case logLevelError:
			o.console.Error(message, arguments...)
		}
	}
}

// validateAnalyseRequest checks an analysis request and returns an error
// response if it is not valid.
//
// Takes request (*wasm_dto.AnalyseRequest) which contains the source files to
// check.
// Takes maxSourceSize (int) which sets the largest allowed total size of all
// source files in bytes.
//
// Returns *wasm_dto.AnalyseResponse which contains an error response if the
// check fails, or nil if the request is valid.
func validateAnalyseRequest(ctx context.Context, request *wasm_dto.AnalyseRequest, maxSourceSize int) *wasm_dto.AnalyseResponse {
	if len(request.Sources) == 0 {
		return newAnalyseErrorResponse("no source files provided", nil)
	}

	totalSize := 0
	for _, content := range request.Sources {
		totalSize += len(content)
	}
	sourceSizeBytes.Record(ctx, int64(totalSize))

	if totalSize > maxSourceSize {
		return newAnalyseErrorResponse(
			fmt.Sprintf("source size %d exceeds maximum %d", totalSize, maxSourceSize),
			nil,
		)
	}

	return nil
}

// prepareSourceBytes converts source strings to byte slices and makes paths
// absolute.
//
// Takes sources (map[string]string) which holds the source file contents keyed
// by their paths.
// Takes moduleName (string) which is the module name to add when building
// absolute paths.
//
// Returns map[string][]byte which holds the same content as byte slices, with
// paths made absolute by adding the module name as a prefix.
func prepareSourceBytes(sources map[string]string, moduleName string) map[string][]byte {
	sourceBytes := make(map[string][]byte, len(sources))
	for path, content := range sources {
		if !strings.HasPrefix(path, pathSeparator) {
			path = pathSeparator + moduleName + pathSeparator + path
		}
		sourceBytes[path] = []byte(content)
	}
	return sourceBytes
}

// runAnalysis performs the actual analysis with the lite builder.
//
// Takes stdlibData (*inspector_dto.TypeData) which provides standard library
// type information for the analysis.
// Takes sourceBytes (map[string][]byte) which contains the source files to
// analyse, keyed by file path.
// Takes moduleName (string) which specifies the Go module name for the code.
//
// Returns *wasm_dto.AnalyseResponse which contains the analysis results or
// error details.
// Returns error when an unexpected failure occurs.
func runAnalysis(ctx context.Context, stdlibData *inspector_dto.TypeData, sourceBytes map[string][]byte, moduleName string) (*wasm_dto.AnalyseResponse, error) {
	config := inspector_dto.Config{
		BaseDir:         pathSeparator + moduleName,
		ModuleName:      moduleName,
		MaxParseWorkers: nil,
		GOOS:            "",
		GOARCH:          "",
		GOCACHE:         "",
		GOMODCACHE:      "",
		BuildFlags:      nil,
	}

	builder, err := inspector_domain.NewLiteBuilder(stdlibData, config)
	if err != nil {
		analyseErrorCount.Add(ctx, 1)
		return newAnalyseErrorResponse(fmt.Sprintf("failed to create lite builder: %v", err), nil), nil
	}

	if err := builder.Build(ctx, sourceBytes); err != nil {
		analyseErrorCount.Add(ctx, 1)
		return newAnalyseErrorResponse(fmt.Sprintf("analysis failed: %v", err), errorToDiagnostics(err)), nil
	}

	typeData, err := builder.GetTypeData()
	if err != nil {
		analyseErrorCount.Add(ctx, 1)
		return newAnalyseErrorResponse(fmt.Sprintf("failed to get type data: %v", err), nil), nil
	}

	response := typeDataToResponse(typeData, moduleName)
	response.Success = true
	return response, nil
}

// newAnalyseErrorResponse creates a failed response for analysis requests.
//
// Takes errMessage (string) which is the error message to include.
// Takes diagnostics ([]wasm_dto.Diagnostic) which holds any diagnostic details
// to attach.
//
// Returns *wasm_dto.AnalyseResponse with Success set to false and the given
// error details filled in.
func newAnalyseErrorResponse(errMessage string, diagnostics []wasm_dto.Diagnostic) *wasm_dto.AnalyseResponse {
	return &wasm_dto.AnalyseResponse{
		Success:     false,
		Error:       errMessage,
		Types:       nil,
		Functions:   nil,
		Imports:     nil,
		Diagnostics: diagnostics,
	}
}

// convertSFCResultToAST converts a parsed SFC result into a template AST.
//
// Takes result (*sfcparser.ParseResult) which holds the parsed SFC content.
//
// Returns *wasm_dto.TemplateAST which holds the template nodes and script
// block details.
func convertSFCResultToAST(result *sfcparser.ParseResult) *wasm_dto.TemplateAST {
	templateAST := &wasm_dto.TemplateAST{
		Nodes:       make([]wasm_dto.TemplateNode, 0),
		ScriptBlock: nil,
	}

	if result.Template != "" {
		templateAST.Nodes = append(templateAST.Nodes, wasm_dto.TemplateNode{
			Type:       "template",
			Name:       "template",
			Content:    result.Template,
			Attributes: result.TemplateAttributes,
			Children:   nil,
			Location: wasm_dto.Location{
				FilePath: "",
				Line:     result.TemplateContentLocation.Line,
				Column:   result.TemplateContentLocation.Column,
			},
		})
	}

	for _, script := range result.Scripts {
		if script.IsGo() {
			templateAST.ScriptBlock = extractScriptBlockInfo(&script)
			break
		}
	}

	return templateAST
}

// extractScriptBlockInfo gathers details from a parsed Go script block.
//
// Takes script (*sfcparser.Script) which holds the parsed script content.
//
// Returns *wasm_dto.ScriptBlockInfo which contains the type declarations,
// props type, and whether an init function exists.
func extractScriptBlockInfo(script *sfcparser.Script) *wasm_dto.ScriptBlockInfo {
	info := &wasm_dto.ScriptBlockInfo{
		PropsType: "",
		Types:     make([]string, 0),
		HasInit:   false,
	}

	file := parseScriptContent(script.Content)
	if file == nil {
		return info
	}

	for _, declaration := range file.Decls {
		extractDeclInfo(declaration, info)
	}

	return info
}

// parseScriptContent parses Go source code into an AST.
//
// Adds a package declaration if the content does not already have one.
//
// Takes content (string) which is the Go source code to parse.
//
// Returns *ast.File which is the parsed AST, or nil if parsing fails.
func parseScriptContent(content string) *ast.File {
	if !strings.HasPrefix(strings.TrimSpace(content), "package ") {
		content = "package main\n" + content
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "script.go", content, parser.ParseComments)
	if err != nil {
		return nil
	}
	return file
}

// extractDeclInfo extracts type and function data from a declaration node.
//
// Takes declaration (ast.Decl) which is the declaration node to process.
// Takes info (*wasm_dto.ScriptBlockInfo) which receives the extracted data.
func extractDeclInfo(declaration ast.Decl, info *wasm_dto.ScriptBlockInfo) {
	switch d := declaration.(type) {
	case *ast.GenDecl:
		extractTypesFromGenDecl(d, info)
	case *ast.FuncDecl:
		extractInitFromFuncDecl(d, info)
	}
}

// extractTypesFromGenDecl gets type names from a general declaration.
//
// Takes d (*ast.GenDecl) which is the declaration to read types from.
// Takes info (*wasm_dto.ScriptBlockInfo) which stores the found type names.
func extractTypesFromGenDecl(d *ast.GenDecl, info *wasm_dto.ScriptBlockInfo) {
	if d.Tok != token.TYPE {
		return
	}
	for _, spec := range d.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		typeName := typeSpec.Name.Name
		info.Types = append(info.Types, typeName)

		if strings.HasSuffix(typeName, "Props") || typeName == "Props" {
			info.PropsType = typeName
		}
	}
}

// extractInitFromFuncDecl checks if the function declaration is an init
// function and sets the HasInit flag on the script block info.
//
// Takes d (*ast.FuncDecl) which is the function declaration to check.
// Takes info (*wasm_dto.ScriptBlockInfo) which is updated with the HasInit
// flag when the declaration is an init function.
func extractInitFromFuncDecl(d *ast.FuncDecl, info *wasm_dto.ScriptBlockInfo) {
	if d.Name.Name == "init" && d.Recv == nil {
		info.HasInit = true
	}
}

// assemblePreviewTemplate combines a template body and script block into a
// single PK template source.
//
// Takes template (string) which is the template body.
// Takes script (string) which is the Go script content.
//
// Returns string which is the combined template with embedded script block.
func assemblePreviewTemplate(template, script string) string {
	return "<script>\n" + script + "\n</script>\n\n" + template
}

// findPageArtefactForURL finds the page artefact that matches the given URL.
//
// It first checks the manifest for a matching page. If no match is found, it
// falls back to returning the first page artefact.
//
// Takes artefacts ([]wasm_dto.GeneratedArtefact) which contains all generated
// files.
// Takes manifest (*wasm_dto.GeneratedManifest) which contains page metadata
// and package paths.
// Takes requestURL (string) which is the URL to match against page routes.
//
// Returns *wasm_dto.GeneratedArtefact which is the matching page artefact.
// Returns string which is the package path for the page.
// Returns map[string]string which contains dependency paths mapped to content.
func findPageArtefactForURL(artefacts []wasm_dto.GeneratedArtefact, manifest *wasm_dto.GeneratedManifest, requestURL, moduleName string) (*wasm_dto.GeneratedArtefact, string, map[string]string) {
	deps := collectPartialDependencies(artefacts, moduleName)

	if manifest != nil && manifest.Pages != nil {
		if artefact, packagePath := findMatchingPageFromManifest(artefacts, manifest, requestURL); artefact != nil {
			return artefact, packagePath, deps
		}
	}

	pageArtefact, packagePath := findFirstPageArtefact(artefacts, manifest)
	return pageArtefact, packagePath, deps
}

// collectPartialDependencies builds a dependency map from partial artefacts so
// the interpreter can resolve partial imports during type checking.
//
// It derives import paths directly from artefact file paths rather than
// relying on the manifest, since the generator may not populate partial
// manifest entries.
//
// Takes artefacts ([]wasm_dto.GeneratedArtefact) which contains all generated
// files including partial artefacts.
// Takes moduleName (string) which is the Go module name prefix for import paths.
//
// Returns map[string]string which maps partial package paths to their generated
// source code.
func collectPartialDependencies(artefacts []wasm_dto.GeneratedArtefact, moduleName string) map[string]string {
	deps := make(map[string]string)

	for i := range artefacts {
		if artefacts[i].Type != wasm_dto.ArtefactTypePartial {
			continue
		}
		if !strings.HasSuffix(artefacts[i].Path, ".go") {
			continue
		}

		artefactDir := artefactDirectory(artefacts[i].Path)
		if artefactDir == "" || !strings.Contains(artefactDir, "/partials/") {
			continue
		}
		packagePath := moduleName + "/" + artefactDir
		deps[packagePath] = artefacts[i].Content
	}

	return deps
}

// artefactDirectory returns the directory portion of an artefact path.
//
// Takes path (string) which is the artefact file path
// (e.g., "dist/partials/partials_info_card_14ceb24d/generated.go").
//
// Returns string which is the directory
// (e.g., "dist/partials/partials_info_card_14ceb24d"), or empty if the
// path has no directory separator.
func artefactDirectory(path string) string {
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash <= 0 {
		return ""
	}
	return path[:lastSlash]
}

// findMatchingPageFromManifest searches the manifest for a page that matches
// the request URL.
//
// Takes artefacts ([]wasm_dto.GeneratedArtefact) which contains all generated
// files.
// Takes manifest (*wasm_dto.GeneratedManifest) which contains page metadata.
// Takes requestURL (string) which is the URL to match.
//
// Returns *wasm_dto.GeneratedArtefact which is the matching artefact, or nil.
// Returns string which is the package path for the page.
func findMatchingPageFromManifest(artefacts []wasm_dto.GeneratedArtefact, manifest *wasm_dto.GeneratedManifest, requestURL string) (*wasm_dto.GeneratedArtefact, string) {
	for _, pageEntry := range manifest.Pages {
		if !pageMatchesURL(pageEntry.RoutePatterns, requestURL) {
			continue
		}
		if artefact := findArtefactBySourcePath(artefacts, pageEntry.SourcePath); artefact != nil {
			return artefact, pageEntry.PackagePath
		}
	}
	return nil, ""
}

// pageMatchesURL checks whether any route pattern matches the request URL.
//
// Takes patterns (map[string]string) which maps locale codes to route patterns.
// Takes requestURL (string) which is the URL to match against.
//
// Returns bool which is true if any pattern matches the URL.
func pageMatchesURL(patterns map[string]string, requestURL string) bool {
	for _, pattern := range patterns {
		if matchesRoute(pattern, requestURL) {
			return true
		}
	}
	return false
}

// findArtefactBySourcePath finds a page artefact with the given source path.
//
// Takes artefacts ([]wasm_dto.GeneratedArtefact) which contains the list of
// artefacts to search through.
// Takes sourcePath (string) which is the path to match against.
//
// Returns *wasm_dto.GeneratedArtefact which is the matching artefact, or nil
// if no match is found.
func findArtefactBySourcePath(artefacts []wasm_dto.GeneratedArtefact, sourcePath string) *wasm_dto.GeneratedArtefact {
	for i := range artefacts {
		if artefacts[i].Type == wasm_dto.ArtefactTypePage && artefacts[i].SourcePath == sourcePath {
			return &artefacts[i]
		}
	}
	return nil
}

// findFirstPageArtefact finds the first page artefact to use as a fallback.
//
// Takes artefacts ([]wasm_dto.GeneratedArtefact) which contains the artefacts
// to search through.
// Takes manifest (*wasm_dto.GeneratedManifest) which provides package path
// lookup.
//
// Returns *wasm_dto.GeneratedArtefact which is the first page artefact found,
// or nil if none exists.
// Returns string which is the package path for the page, or empty if not found.
func findFirstPageArtefact(artefacts []wasm_dto.GeneratedArtefact, manifest *wasm_dto.GeneratedManifest) (*wasm_dto.GeneratedArtefact, string) {
	for i := range artefacts {
		if artefacts[i].Type != wasm_dto.ArtefactTypePage {
			continue
		}
		packagePath := lookupPackagePath(manifest, artefacts[i].SourcePath)
		return &artefacts[i], packagePath
	}
	return nil, ""
}

// lookupPackagePath finds the package path for an artefact in the manifest.
//
// Takes manifest (*wasm_dto.GeneratedManifest) which holds page metadata.
// Takes sourcePath (string) which is the artefact source path to look up.
//
// Returns string which is the package path, or empty if not found.
func lookupPackagePath(manifest *wasm_dto.GeneratedManifest, sourcePath string) string {
	if manifest == nil || manifest.Pages == nil {
		return ""
	}
	for _, pageEntry := range manifest.Pages {
		if pageEntry.SourcePath == sourcePath {
			return pageEntry.PackagePath
		}
	}
	return ""
}

// matchesRoute checks if a URL matches a route pattern.
//
// Supports Chi-style dynamic segments:
//   - {param} matches a single non-empty path segment
//   - {param*} matches all remaining path segments (catch-all, must be last)
//
// The URL's query string and fragment are stripped before comparison so
// that `/?sort=name` and `/#section` both match the pattern `/`. Without
// this, a page-level link click that sends e.g. `/?sort=department` would
// fail to match its own page route, and dynamicRender would fall back to
// findFirstPageArtefact (no styles, wrong page).
//
// Takes pattern (string) which is the route pattern to match against.
// Takes url (string) which is the URL to check.
//
// Returns bool which is true if the URL matches the pattern.
func matchesRoute(pattern, url string) bool {
	if i := strings.IndexAny(url, "?#"); i >= 0 {
		url = url[:i]
	}
	pattern = strings.TrimSuffix(pattern, "/")
	url = strings.TrimSuffix(url, "/")

	if pattern == url {
		return true
	}

	if pattern == "" && (url == "" || url == "/") {
		return true
	}

	patternSegs := strings.Split(strings.TrimPrefix(pattern, "/"), "/")
	urlSegs := strings.Split(strings.TrimPrefix(url, "/"), "/")

	return matchSegments(patternSegs, urlSegs)
}

// matchSegments compares pattern segments against URL segments.
//
// Takes patternSegs ([]string) which contains the pattern path segments.
// Takes urlSegs ([]string) which contains the URL path segments.
//
// Returns bool which is true if all segments match.
func matchSegments(patternSegs, urlSegs []string) bool {
	for i, pSeg := range patternSegs {
		if isCatchAllSegment(pSeg) {
			return true
		}

		if i >= len(urlSegs) {
			return false
		}

		if isDynamicSegment(pSeg) {
			if urlSegs[i] == "" {
				return false
			}

			continue
		}

		if pSeg != urlSegs[i] {
			return false
		}
	}

	return len(patternSegs) == len(urlSegs)
}

// isDynamicSegment reports whether a path segment is a Chi-style dynamic
// parameter like {slug}.
//
// Takes seg (string) which is the path segment to check.
//
// Returns bool which is true if the segment is a dynamic parameter.
func isDynamicSegment(seg string) bool {
	return len(seg) > 2 && seg[0] == '{' && seg[len(seg)-1] == '}' && !strings.HasSuffix(seg, "*}")
}

// isCatchAllSegment reports whether a path segment is a Chi-style catch-all
// parameter like {path*}.
//
// Takes seg (string) which is the path segment to check.
//
// Returns bool which is true if the segment is a catch-all parameter.
func isCatchAllSegment(seg string) bool {
	return len(seg) > minCatchAllSegmentLength && seg[0] == '{' && strings.HasSuffix(seg, "*}")
}

// typeDataToResponse converts type inspection data into an analysis response.
//
// Takes data (*inspector_dto.TypeData) which holds the inspected type and
// package details.
// Takes moduleName (string) which is the user's module name, used to filter
// out standard library packages.
//
// Returns *wasm_dto.AnalyseResponse which contains the extracted types,
// functions, and imports from packages that match the module name.
func typeDataToResponse(data *inspector_dto.TypeData, moduleName string) *wasm_dto.AnalyseResponse {
	response := &wasm_dto.AnalyseResponse{
		Success:     false,
		Types:       make([]wasm_dto.TypeInfo, 0),
		Functions:   make([]wasm_dto.FunctionInfo, 0),
		Imports:     make([]wasm_dto.ImportInfo, 0),
		Diagnostics: nil,
		Error:       "",
	}

	for packagePath, inspectedPackage := range data.Packages {
		if !strings.HasPrefix(packagePath, moduleName) {
			continue
		}

		extractTypesFromPackage(inspectedPackage, response)
		extractFunctionsFromPackage(inspectedPackage, response)
		extractImportsFromPackage(inspectedPackage, response)
	}

	return response
}

// extractTypesFromPackage copies type data from a package into a response.
//
// Takes inspectedPackage (*inspector_dto.Package) which
// provides the parsed package data.
// Takes response (*wasm_dto.AnalyseResponse) which receives
// the extracted types.
func extractTypesFromPackage(inspectedPackage *inspector_dto.Package, response *wasm_dto.AnalyseResponse) {
	for _, typ := range inspectedPackage.NamedTypes {
		response.Types = append(response.Types, convertTypeToInfo(typ))
	}
}

// convertTypeToInfo converts an inspector type to a TypeInfo structure.
//
// Takes typ (*inspector_dto.Type) which is the source type to convert.
//
// Returns wasm_dto.TypeInfo which holds the converted type data including
// fields, methods, and location.
func convertTypeToInfo(typ *inspector_dto.Type) wasm_dto.TypeInfo {
	typeInfo := wasm_dto.TypeInfo{
		Name:          typ.Name,
		Kind:          typeKind(typ),
		Fields:        make([]wasm_dto.FieldInfo, 0, len(typ.Fields)),
		Methods:       make([]wasm_dto.MethodInfo, 0, len(typ.Methods)),
		IsExported:    isExported(typ.Name),
		Documentation: "",
		Location: wasm_dto.Location{
			FilePath: typ.DefinedInFilePath,
			Line:     typ.DefinitionLine,
			Column:   typ.DefinitionColumn,
		},
	}

	for _, field := range typ.Fields {
		typeInfo.Fields = append(typeInfo.Fields, wasm_dto.FieldInfo{
			Name:          field.Name,
			TypeString:    field.TypeString,
			Tag:           field.RawTag,
			IsEmbedded:    field.IsEmbedded,
			Documentation: "",
		})
	}

	for _, method := range typ.Methods {
		typeInfo.Methods = append(typeInfo.Methods, wasm_dto.MethodInfo{
			Name:              method.Name,
			Signature:         method.TypeString,
			IsPointerReceiver: method.IsPointerReceiver,
			Documentation:     "",
		})
	}

	return typeInfo
}

// extractFunctionsFromPackage copies function data from a package into the
// response.
//
// Takes inspectedPackage (*inspector_dto.Package) which
// contains the parsed package data.
// Takes response (*wasm_dto.AnalyseResponse) which receives
// the function details.
func extractFunctionsFromPackage(inspectedPackage *inspector_dto.Package, response *wasm_dto.AnalyseResponse) {
	for _, inspectedFunction := range inspectedPackage.Funcs {
		response.Functions = append(response.Functions, wasm_dto.FunctionInfo{
			Name:          inspectedFunction.Name,
			Signature:     inspectedFunction.TypeString,
			IsExported:    isExported(inspectedFunction.Name),
			Documentation: "",
			Location: wasm_dto.Location{
				FilePath: inspectedFunction.DefinitionFilePath,
				Line:     inspectedFunction.DefinitionLine,
				Column:   inspectedFunction.DefinitionColumn,
			},
		})
	}
}

// extractImportsFromPackage collects import details from a package.
//
// Takes inspectedPackage (*inspector_dto.Package) which
// contains the parsed package data.
// Takes response (*wasm_dto.AnalyseResponse) which receives
// the collected imports.
func extractImportsFromPackage(inspectedPackage *inspector_dto.Package, response *wasm_dto.AnalyseResponse) {
	for _, imports := range inspectedPackage.FileImports {
		for importPath, alias := range imports {
			response.Imports = append(response.Imports, wasm_dto.ImportInfo{
				Path:   importPath,
				Alias:  alias,
				IsUsed: false,
			})
		}
	}
}

// typeKind returns a string that describes the kind of a type definition.
//
// Takes typ (*inspector_dto.Type) which is the type to classify.
//
// Returns string which is one of "alias", "struct", "interface", or "type".
func typeKind(typ *inspector_dto.Type) string {
	if typ.IsAlias {
		return "alias"
	}
	if typ.UnderlyingTypeString == "struct{...}" {
		return "struct"
	}
	if strings.HasPrefix(typ.UnderlyingTypeString, "interface{") {
		return "interface"
	}
	return "type"
}

// errorToDiagnostics converts an error into a slice of diagnostics.
//
// Takes err (error) which is the error to convert.
//
// Returns []wasm_dto.Diagnostic which holds a single diagnostic with error
// severity and no location set.
func errorToDiagnostics(err error) []wasm_dto.Diagnostic {
	return []wasm_dto.Diagnostic{
		{
			Severity: severityError,
			Message:  err.Error(),
			Location: wasm_dto.Location{
				FilePath: "",
				Line:     0,
				Column:   0,
			},
			Code: "",
		},
	}
}

// parseErrorToDiagnostics converts a parse error to diagnostics with proper
// positions. It handles scanner.ErrorList for multiple errors and falls back
// to regex parsing.
//
// Takes err (error) which is the parse error to convert.
// Takes filePath (string) which identifies the source file for the diagnostic.
//
// Returns []wasm_dto.Diagnostic which contains the converted diagnostics.
func parseErrorToDiagnostics(err error, filePath string) []wasm_dto.Diagnostic {
	if errList, ok := errors.AsType[scanner.ErrorList](err); ok {
		diagnostics := make([]wasm_dto.Diagnostic, 0, len(errList))
		for _, e := range errList {
			diagnostics = append(diagnostics, wasm_dto.Diagnostic{
				Severity: severityError,
				Message:  e.Msg,
				Location: wasm_dto.Location{
					FilePath: filePath,
					Line:     e.Pos.Line,
					Column:   e.Pos.Column,
				},
				Code: "",
			})
		}
		return diagnostics
	}

	return parseErrorStringToDiagnostics(err.Error(), filePath)
}

// parseErrorStringToDiagnostics parses an error string to extract line and
// column positions.
//
// Takes errMessage (string) which contains the error message to parse.
// Takes filePath (string) which specifies the file path for the diagnostic.
//
// Returns []wasm_dto.Diagnostic which contains a single diagnostic with the
// extracted line and column positions. If parsing fails, the positions are
// set to zero.
func parseErrorStringToDiagnostics(errMessage, filePath string) []wasm_dto.Diagnostic {
	matches := errorPosRegex.FindStringSubmatch(errMessage)
	if len(matches) >= regexMinMatchesForPosition {
		line, lineErr := strconv.Atoi(matches[regexGroupLine])
		column := 0
		if matches[regexGroupColumn] != "" {
			column, _ = strconv.Atoi(matches[regexGroupColumn])
		}

		message := errMessage
		if len(matches) >= regexMinMatchesForMessage && matches[regexGroupMessage] != "" {
			message = matches[regexGroupMessage]
		}

		if lineErr == nil && line > 0 {
			return []wasm_dto.Diagnostic{
				{
					Severity: severityError,
					Message:  message,
					Location: wasm_dto.Location{
						FilePath: filePath,
						Line:     line,
						Column:   column,
					},
					Code: "",
				},
			}
		}
	}

	return []wasm_dto.Diagnostic{
		{
			Severity: severityError,
			Message:  errMessage,
			Location: wasm_dto.Location{
				FilePath: filePath,
				Line:     0,
				Column:   0,
			},
			Code: "",
		},
	}
}

// collectScriptArtefacts pulls every JavaScript artefact out of a generator
// response so the dynamic-render path can surface them to the consumer.
//
// The generator returns a mixed bag (Go code, manifests, register files,
// CSS) so we filter to ArtefactTypeJS. The returned slice preserves
// generator order, which keeps responses deterministic for golden tests.
//
// Takes genResp (*wasm_dto.GenerateFromSourcesResponse) which carries every
// artefact captured during generation.
//
// Returns []wasm_dto.ScriptArtefact where each entry is one compiled ES
// module. Returns nil when genResp is nil or no JS artefacts were emitted;
// JSON-omitempty hides the field from the response in that case.
func collectScriptArtefacts(genResp *wasm_dto.GenerateFromSourcesResponse) []wasm_dto.ScriptArtefact {
	if genResp == nil || len(genResp.Artefacts) == 0 {
		return nil
	}

	scripts := make([]wasm_dto.ScriptArtefact, 0, len(genResp.Artefacts))
	for _, artefact := range genResp.Artefacts {
		if artefact.Type != wasm_dto.ArtefactTypeJS {
			continue
		}
		scripts = append(scripts, wasm_dto.ScriptArtefact{
			Path:    artefact.Path,
			Content: artefact.Content,
		})
	}
	if len(scripts) == 0 {
		return nil
	}
	return scripts
}

// defaultRuntimeImports lists the framework-runtime URLs.
//
// The dynamic-render path echoes this list in every response so
// consumers can pre-resolve the imports (typically by fetching the
// framework bundles from the parent daemon and adding entries to an
// importmap) without having to scan emitted JS for `import` statements.
// Sourced from the exported constants in generator_domain so any future
// path change ripples through automatically.
var defaultRuntimeImports = []string{
	generator_domain.PKFrameworkURL,
	generator_domain.PKComponentsURL,
	generator_domain.PKActionsGenURL,
}

// findPageStyleBlock returns the aggregated CSS for the page that matches
// requestURL. The block already includes CSS from every transitively
// referenced partial (the manifest builder collapses them at generation
// time), so it can be used verbatim as the page's <style> contents.
//
// Takes genResp (*wasm_dto.GenerateFromSourcesResponse) which carries the
// manifest produced this run.
// Takes requestURL (string) which is matched against each page entry's
// route patterns.
//
// Returns string which is the matched page's StyleBlock, or empty when no
// page matches or the manifest is absent.
func findPageStyleBlock(genResp *wasm_dto.GenerateFromSourcesResponse, requestURL string) string {
	if genResp == nil || genResp.Manifest == nil {
		return ""
	}
	for _, pageEntry := range genResp.Manifest.Pages {
		if pageMatchesURL(pageEntry.RoutePatterns, requestURL) {
			return pageEntry.StyleBlock
		}
	}
	return ""
}
