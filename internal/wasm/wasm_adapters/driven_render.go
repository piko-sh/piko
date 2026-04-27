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

package wasm_adapters

import (
	"context"
	"fmt"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/internal/wasm/wasm_domain"
	"piko.sh/piko/internal/wasm/wasm_dto"
)

// RenderAdapter implements RenderPort by using the annotator to parse templates
// and a HeadlessRendererPort to convert the AST to HTML.
//
// This adapter only supports static templates (no Go code execution).
// Templates with Go handlers or expressions will render with placeholder values.
type RenderAdapter struct {
	// headlessRenderer converts TemplateAST to HTML strings.
	headlessRenderer wasm_domain.HeadlessRendererPort

	// stdlibDataGetter retrieves the pre-bundled standard library type
	// information. This is a function because the data may not be available
	// when the adapter is created.
	stdlibDataGetter func() (*inspector_dto.TypeData, error)

	// moduleName is the Go module name used for generated code.
	moduleName string
}

var _ wasm_domain.RenderPort = (*RenderAdapter)(nil)

// RenderAdapterOption configures a RenderAdapter.
type RenderAdapterOption func(*RenderAdapter)

// NewRenderAdapter creates a new render adapter with the given options.
//
// Takes opts (...RenderAdapterOption) which configure the adapter.
//
// Returns *RenderAdapter which is ready to render templates.
func NewRenderAdapter(opts ...RenderAdapterOption) *RenderAdapter {
	a := &RenderAdapter{
		moduleName: "playground",
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Render produces HTML from in-memory sources.
//
// Takes request (*wasm_dto.RenderFromSourcesRequest) which contains the source
// files and rendering options.
//
// Returns *wasm_dto.RenderFromSourcesResponse which contains the rendered HTML
// and CSS, or error details if rendering fails.
// Returns error when an unexpected failure occurs.
func (a *RenderAdapter) Render(
	ctx context.Context,
	request *wasm_dto.RenderFromSourcesRequest,
) (*wasm_dto.RenderFromSourcesResponse, error) {
	stdlibData, errResp := a.validateAndGetStdlib()
	if errResp != nil {
		return errResp, nil
	}

	moduleName := request.ModuleName
	if moduleName == "" {
		moduleName = a.moduleName
	}

	annotator, errResp := a.createAnnotator(request.Sources, moduleName, stdlibData)
	if errResp != nil {
		return errResp, nil
	}

	entryPoints := a.findEntryPoints(request.Sources, moduleName, request.EntryPoint)
	if len(entryPoints) == 0 {
		return a.errorResponse("no .pk files found in sources"), nil
	}

	result, _, err := annotator.AnnotateProject(ctx, entryPoints, nil)
	if err != nil {
		return a.errorResponse(fmt.Sprintf("annotation failed: %v", err)), nil
	}

	component, styling := a.selectComponentToRender(result, request.EntryPoint)
	if component == nil {
		return a.errorResponse("no component found to render"), nil
	}

	if component.AnnotatedAST == nil {
		return a.errorResponse("component has no annotated AST"), nil
	}

	html, errResp := a.renderToHTML(ctx, component, styling)
	if errResp != nil {
		return errResp, nil
	}

	return &wasm_dto.RenderFromSourcesResponse{
		Success:      true,
		HTML:         html,
		CSS:          styling,
		Diagnostics:  a.convertDiagnostics(result.AllDiagnostics),
		IsStaticOnly: true,
	}, nil
}

// RenderFromAST produces HTML from a pre-built TemplateAST.
// This is used for dynamic rendering where the AST comes from interpreter
// execution rather than annotation.
//
// Takes request (*wasm_dto.RenderFromASTRequest) which contains the AST and
// metadata.
//
// Returns *wasm_dto.RenderFromASTResponse which contains the rendered HTML.
// Returns error when rendering fails.
func (a *RenderAdapter) RenderFromAST(
	ctx context.Context,
	request *wasm_dto.RenderFromASTRequest,
) (*wasm_dto.RenderFromASTResponse, error) {
	if request.AST == nil {
		return &wasm_dto.RenderFromASTResponse{
			Success: true,
			HTML:    "",
			CSS:     request.CSS,
		}, nil
	}

	if a.headlessRenderer == nil {
		return &wasm_dto.RenderFromASTResponse{
			Success: false,
			Error:   "headless renderer not configured",
		}, nil
	}

	html, err := a.headlessRenderer.RenderASTToString(ctx, wasm_domain.HeadlessRenderOptions{
		Template:               request.AST,
		Metadata:               request.Metadata,
		Styling:                request.CSS,
		IncludeDocumentWrapper: false,
	})
	if err != nil {
		return &wasm_dto.RenderFromASTResponse{
			Success: false,
			Error:   fmt.Sprintf("rendering failed: %v", err),
		}, nil
	}

	return &wasm_dto.RenderFromASTResponse{
		Success: true,
		HTML:    html,
		CSS:     request.CSS,
	}, nil
}

// validateAndGetStdlib validates the adapter and returns stdlib data.
//
// Returns *inspector_dto.TypeData which contains the standard library type
// information.
// Returns *wasm_dto.RenderFromSourcesResponse which is non-nil when validation
// fails, containing the error details.
func (a *RenderAdapter) validateAndGetStdlib() (*inspector_dto.TypeData, *wasm_dto.RenderFromSourcesResponse) {
	if a.stdlibDataGetter == nil {
		return nil, a.errorResponse("render adapter not configured: stdlib data getter is nil")
	}
	stdlibData, err := a.stdlibDataGetter()
	if err != nil {
		return nil, a.errorResponse(fmt.Sprintf("failed to get stdlib data: %v", err))
	}
	return stdlibData, nil
}

// createAnnotator creates the in-memory annotator service.
//
// Takes sources (map[string]string) which provides the source files to parse.
// Takes moduleName (string) which specifies the module path for the sources.
// Takes stdlibData (*inspector_dto.TypeData) which provides standard library
// type information.
//
// Returns annotator_domain.AnnotatorPort which is the configured annotator
// service ready for use.
// Returns *wasm_dto.RenderFromSourcesResponse which contains an error response
// when the annotator service cannot be created, or nil on success.
func (a *RenderAdapter) createAnnotator(sources map[string]string, moduleName string, stdlibData *inspector_dto.TypeData) (annotator_domain.AnnotatorPort, *wasm_dto.RenderFromSourcesResponse) {
	annotator, err := NewInMemoryAnnotatorService(sources, moduleName, stdlibData)
	if err != nil {
		return nil, a.errorResponse(fmt.Sprintf("failed to create annotator service: %v", err))
	}
	return annotator, nil
}

// renderToHTML renders the component to HTML using the headless renderer.
//
// Takes component (*annotator_dto.AnnotationResult) which provides the
// annotated AST to render.
// Takes styling (string) which specifies the styling to apply.
//
// Returns string which contains the rendered HTML output.
// Returns *wasm_dto.RenderFromSourcesResponse which contains error details
// when rendering fails, or nil on success.
func (a *RenderAdapter) renderToHTML(ctx context.Context, component *annotator_dto.AnnotationResult, styling string) (string, *wasm_dto.RenderFromSourcesResponse) {
	if a.headlessRenderer == nil {
		return "", a.errorResponse("headless renderer not configured")
	}

	html, err := a.headlessRenderer.RenderASTToString(ctx, wasm_domain.HeadlessRenderOptions{
		Template:               component.AnnotatedAST,
		Metadata:               a.buildMetadata(component),
		Styling:                styling,
		IncludeDocumentWrapper: true,
	})
	if err != nil {
		return "", a.errorResponse(fmt.Sprintf("rendering failed: %v", err))
	}
	return html, nil
}

// errorResponse creates a failed response with the given error message.
//
// Takes message (string) which contains the error message to include.
//
// Returns *wasm_dto.RenderFromSourcesResponse which is the failed response.
func (*RenderAdapter) errorResponse(message string) *wasm_dto.RenderFromSourcesResponse {
	return &wasm_dto.RenderFromSourcesResponse{Success: false, Error: message}
}

// findEntryPoints discovers .pk files from the sources map.
//
// Takes sources (map[string]string) which contains the source files to search.
// Takes moduleName (string) which specifies the module prefix for paths.
// Takes preferredEntry (string) which filters to a specific entry point if set.
//
// Returns []annotator_dto.EntryPoint which contains the discovered entry points.
func (*RenderAdapter) findEntryPoints(
	sources map[string]string,
	moduleName string,
	preferredEntry string,
) []annotator_dto.EntryPoint {
	entryPoints := make([]annotator_dto.EntryPoint, 0, len(sources))

	for path := range sources {
		if !strings.HasSuffix(path, ".pk") {
			continue
		}

		if preferredEntry != "" && !strings.HasSuffix(path, preferredEntry) && path != preferredEntry {
			continue
		}

		isPage := strings.Contains(path, "pages/") || strings.HasPrefix(path, "pages/")

		fullPath := moduleName + "/" + path

		entryPoints = append(entryPoints, annotator_dto.EntryPoint{
			Path:   fullPath,
			IsPage: isPage,
		})
	}

	return entryPoints
}

// selectComponentToRender chooses which component to render from the results.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the
// annotated components to select from.
// Takes preferredEntry (string) which specifies a preferred component name to
// match.
//
// Returns *annotator_dto.AnnotationResult which is the selected component, or
// nil if none found.
// Returns string which is the style block for the selected component.
func (*RenderAdapter) selectComponentToRender(
	result *annotator_dto.ProjectAnnotationResult,
	preferredEntry string,
) (*annotator_dto.AnnotationResult, string) {
	if result == nil {
		return nil, ""
	}

	for key, comp := range result.ComponentResults {
		if comp == nil {
			continue
		}

		if preferredEntry != "" {
			if strings.Contains(key, preferredEntry) || strings.HasSuffix(key, preferredEntry) {
				return comp, comp.StyleBlock
			}
			continue
		}

		return comp, comp.StyleBlock
	}

	return nil, ""
}

// buildMetadata creates InternalMetadata from a component's annotation result.
//
// Takes component (*annotator_dto.AnnotationResult) which provides the
// annotation data to convert.
//
// Returns *templater_dto.InternalMetadata which contains the converted
// metadata, or an empty instance if component is nil.
func (*RenderAdapter) buildMetadata(component *annotator_dto.AnnotationResult) *templater_dto.InternalMetadata {
	if component == nil {
		return &templater_dto.InternalMetadata{}
	}

	return &templater_dto.InternalMetadata{
		AssetRefs:  component.AssetRefs,
		CustomTags: component.CustomTags,
	}
}

// convertDiagnostics converts annotator diagnostics to WASM DTOs.
//
// Takes diagnostics ([]*ast_domain.Diagnostic) which contains the diagnostics to
// convert.
//
// Returns []wasm_dto.Diagnostic which contains the converted diagnostics ready
// for WASM transport.
func (*RenderAdapter) convertDiagnostics(diagnostics []*ast_domain.Diagnostic) []wasm_dto.Diagnostic {
	if len(diagnostics) == 0 {
		return nil
	}

	result := make([]wasm_dto.Diagnostic, 0, len(diagnostics))
	for _, d := range diagnostics {
		if d == nil {
			continue
		}
		result = append(result, wasm_dto.Diagnostic{
			Severity: d.Severity.String(),
			Message:  d.Message,
			Code:     d.Code,
			Location: wasm_dto.Location{
				FilePath: d.SourcePath,
				Line:     d.Location.Line,
				Column:   d.Location.Column,
			},
		})
	}

	return result
}

// WithRendererStdlibDataGetter sets a function to retrieve the pre-bundled
// standard library type information.
//
// Takes getter (func() (*inspector_dto.TypeData, error)) which retrieves
// the stdlib types when called.
//
// Returns RenderAdapterOption which configures the adapter.
func WithRendererStdlibDataGetter(getter func() (*inspector_dto.TypeData, error)) RenderAdapterOption {
	return func(a *RenderAdapter) {
		a.stdlibDataGetter = getter
	}
}

// WithRendererModuleName sets the Go module name for generated code.
//
// Takes moduleName (string) which specifies the module name to use.
//
// Returns RenderAdapterOption which configures the adapter.
func WithRendererModuleName(moduleName string) RenderAdapterOption {
	return func(a *RenderAdapter) {
		a.moduleName = moduleName
	}
}

// WithHeadlessRenderer sets the headless renderer for AST-to-HTML conversion.
//
// Takes renderer (wasm_domain.HeadlessRendererPort) which provides headless
// rendering features.
//
// Returns RenderAdapterOption which configures the adapter.
func WithHeadlessRenderer(renderer wasm_domain.HeadlessRendererPort) RenderAdapterOption {
	return func(a *RenderAdapter) {
		a.headlessRenderer = renderer
	}
}
