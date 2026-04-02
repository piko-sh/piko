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

package render_domain

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/valyala/quicktemplate"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/pml/pml_domain"
	"piko.sh/piko/internal/pml/pml_dto"
	"piko.sh/piko/internal/premailer"
	"piko.sh/piko/internal/render/render_templates"
	"piko.sh/piko/internal/templater/templater_dto"
)

// emailBaseStyles contains the base CSS reset styles for email rendering.
const emailBaseStyles = `#outlook a{padding:0}body{margin:0;padding:0;-webkit-text-size-adjust:100%;` +
	`-ms-text-size-adjust:100%;-webkit-font-smoothing:antialiased;-moz-osx-font-smoothing:grayscale}` +
	`table,td{border-collapse:collapse;mso-table-lspace:0;mso-table-rspace:0}` +
	`img{border:0;height:auto;line-height:100%;outline:none;text-decoration:none;-ms-interpolation-mode:bicubic;max-width:100%}` +
	`p,h1,h2,h3,h4,h5,h6{margin:0}*,*::before,*::after{box-sizing:border-box}`

// RenderEmail orchestrates the complete email rendering pipeline using the
// PREMAILER -> PML flow. It inlines CSS styles onto the raw PML AST via
// Premailer, transforms the styled PikoML tags into final standard HTML via
// pmlEngine, and renders the final HTML to the writer.
//
// Takes w (io.Writer) which receives the rendered email HTML output.
// Takes request (*http.Request) which provides request context for rendering.
// Takes opts (RenderEmailOptions) which specifies the template, styling, and
// metadata for the email.
//
// Returns error when streaming the AST to the writer fails.
func (ro *RenderOrchestrator) RenderEmail(
	ctx context.Context,
	w io.Writer,
	request *http.Request,
	opts RenderEmailOptions,
) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "RenderOrchestrator.RenderEmail", logger_domain.String("pageID", opts.PageID))
	defer span.End()

	if opts.Template != nil {
		defer ast_domain.PutTree(opts.Template)
	}

	preservedBlocks := ro.extractPreservedBlocks(opts.Template)
	l.Trace("Extracted preserved blocks", logger_domain.Int("preserved_blocks_count", len(preservedBlocks)))

	pipelineResult := ro.processEmailPipeline(ctx, opts.Template, opts.Styling, opts.PremailerOptions, opts.IsPreviewMode, span)

	renderCtx := ro.getRenderContext(ctx, opts.PageID, nil, request, nil)
	renderCtx.isEmailMode = true
	renderCtx.skipPrerenderedHTML = true
	defer ro.putRenderContext(renderCtx)

	qw := quicktemplate.AcquireWriter(w)
	defer quicktemplate.ReleaseWriter(qw)

	err := ro.renderEmailContent(ctx, qw, renderCtx, opts.Metadata, emailContentParams{
		HTMLAST:          pipelineResult.HTMLAST,
		PreservedBlocks:  preservedBlocks,
		BodyInlineStyles: pipelineResult.BodyInlineStyles,
		FinalCSS:         pipelineResult.FinalCSS,
	}, span)
	if err != nil {
		return fmt.Errorf("streaming AST for email %s: %w", opts.PageID, err)
	}

	logEmailDiagnostics(ctx, renderCtx)
	return nil
}

// emailPipelineResult holds the outputs from the email processing pipeline.
type emailPipelineResult struct {
	// HTMLAST holds the parsed HTML template as an abstract syntax tree.
	HTMLAST *ast_domain.TemplateAST

	// BodyInlineStyles contains CSS rules to be added directly to HTML body elements.
	BodyInlineStyles string

	// FinalCSS contains CSS rules to include in the email head after inlining.
	FinalCSS string
}

// processEmailPipeline runs the premailer and PML transformation pipeline.
//
// Takes tmplAST (*ast_domain.TemplateAST) which is the template to process.
// Takes styling (string) which contains CSS styles to inline.
// Takes premailerOptions (*premailer.Options) which configures the premailer.
// Takes span (trace.Span) which tracks the operation for tracing.
//
// Returns emailPipelineResult which contains the processed HTML AST and CSS.
func (ro *RenderOrchestrator) processEmailPipeline(
	ctx context.Context,
	tmplAST *ast_domain.TemplateAST,
	styling string,
	premailerOptions *premailer.Options,
	isPreviewMode bool,
	span trace.Span,
) emailPipelineResult {
	ctx, l := logger_domain.From(ctx, log)
	premailedAST, premailerLeftoverCSS := ro.performPremailerPass(ctx, tmplAST, styling, premailerOptions, span)
	if premailedAST != nil && premailedAST != tmplAST {
		defer ast_domain.PutTree(premailedAST)
	}

	htmlAST, pmlGeneratedCSS := ro.performPmlTransformation(ctx, premailedAST, isPreviewMode)
	if htmlAST == nil {
		l.Warn("PikoML transformation returned nil AST, using pre-inlined AST as fallback")
		htmlAST = premailedAST
	} else if htmlAST != premailedAST {
		defer ast_domain.PutTree(htmlAST)
	}

	bodyInlineStyles, finalCombinedCSS := combineEmailCSS(ctx, premailerLeftoverCSS, pmlGeneratedCSS)
	return emailPipelineResult{
		HTMLAST:          htmlAST,
		BodyInlineStyles: bodyInlineStyles,
		FinalCSS:         finalCombinedCSS,
	}
}

// emailContentParams holds the data needed by renderEmailContent.
type emailContentParams struct {
	// HTMLAST is the parsed template AST used to render HTML email content.
	HTMLAST *ast_domain.TemplateAST

	// BodyInlineStyles contains inline CSS styles to apply to the email body element.
	BodyInlineStyles string

	// FinalCSS is the processed CSS to apply to the email content.
	FinalCSS string

	// PreservedBlocks holds HTML blocks to keep in the email head section.
	PreservedBlocks []string
}

// renderEmailContent renders the email body with header, content, and footer.
//
// Takes qw (*quicktemplate.Writer) which receives the rendered output.
// Takes rctx (*renderContext) which holds the current rendering state.
// Takes metadata (*templater_dto.InternalMetadata) which provides email title
// and language settings.
// Takes params (emailContentParams) which contains the CSS, AST, and styling
// to render.
// Takes span (trace.Span) which tracks the rendering operation.
//
// Returns error when the AST cannot be rendered to the writer.
func (ro *RenderOrchestrator) renderEmailContent(
	ctx context.Context,
	qw *quicktemplate.Writer,
	rctx *renderContext,
	metadata *templater_dto.InternalMetadata,
	params emailContentParams,
	span trace.Span,
) error {
	ctx, l := logger_domain.From(ctx, log)
	data := &render_templates.EmailPageData{
		Title:               metadata.Title,
		Styling:             params.FinalCSS,
		BaseStyling:         emailBaseStyles,
		RenderedContent:     "",
		PreservedHeadBlocks: params.PreservedBlocks,
		BackgroundColor:     "",
		BodyInlineStyles:    params.BodyInlineStyles,
		Lang:                metadata.Language,
		Dir:                 "",
	}

	render_templates.StreamEmailPageHeader(qw, data)
	if err := ro.renderASTToWriter(params.HTMLAST, qw, rctx); err != nil {
		l.ReportError(span, err, "Failed to stream final AST for email")
		return fmt.Errorf("streaming email AST: %w", err)
	}

	if len(rctx.requiredSvgSymbols) > 0 {
		ro.embedEmailSVGSprite(ctx, qw, rctx, span)
	}

	render_templates.StreamEmailPageFooter(qw, data)
	return nil
}

// embedEmailSVGSprite builds and writes the SVG sprite sheet for email.
//
// Takes qw (*quicktemplate.Writer) which receives the rendered sprite output.
// Takes rctx (*renderContext) which provides the required SVG symbols.
// Takes span (trace.Span) which provides the tracing context for error reports.
func (ro *RenderOrchestrator) embedEmailSVGSprite(
	ctx context.Context,
	qw *quicktemplate.Writer,
	rctx *renderContext,
	span trace.Span,
) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Building SVG sprite sheet for email", logger_domain.Int("symbolCount", len(rctx.requiredSvgSymbols)))
	spriteSheet, err := ro.buildSvgSpriteSheet(rctx)
	if err != nil {
		l.ReportError(span, err, "Failed to build SVG sprite sheet for email")
		return
	}
	qw.N().S(spriteSheet)
}

// extractPreservedBlocks finds MSO conditional comments and extracts them for
// reinsertion after rendering. These blocks must be preserved exactly as-is
// and not processed by the rendering pipeline.
//
// Takes ast (*ast_domain.TemplateAST) which is the template to scan for
// preserved blocks.
//
// Returns []string which contains the extracted MSO conditional comments.
func (*RenderOrchestrator) extractPreservedBlocks(
	ast *ast_domain.TemplateAST,
) []string {
	if ast == nil {
		return nil
	}

	var preservedBlocks []string
	var nodesToRemove []*ast_domain.TemplateNode

	ast.Walk(func(node *ast_domain.TemplateNode) bool {
		if node.NodeType == ast_domain.NodeRawHTML && strings.HasPrefix(node.TextContent, "<!--[if mso") {
			preservedBlocks = append(preservedBlocks, node.TextContent)
			nodesToRemove = append(nodesToRemove, node)
		}
		return true
	})

	if len(nodesToRemove) > 0 {
		ast.RemoveNodes(nodesToRemove)
	}

	return preservedBlocks
}

// performPremailerPass runs the premailer transformation on the raw AST
// containing <pml-*> tags. It inlines styles as style attributes and extracts
// leftover CSS such as @media queries that cannot be inlined.
//
// Takes tmplAST (*ast_domain.TemplateAST) which is the raw template AST to
// transform.
// Takes styling (string) which contains external CSS to inline.
// Takes premailerOptions (*premailer.Options) which configures the premailer
// behaviour.
// Takes span (trace.Span) which provides tracing context for error reporting.
//
// Returns *ast_domain.TemplateAST which is the transformed AST with inlined
// styles, or the original AST if transformation fails.
// Returns string which contains any leftover CSS that could not be inlined.
func (ro *RenderOrchestrator) performPremailerPass(
	ctx context.Context,
	tmplAST *ast_domain.TemplateAST,
	styling string,
	premailerOptions *premailer.Options,
	span trace.Span,
) (*ast_domain.TemplateAST, string) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Starting premailer transformation on raw PML AST")

	var pmOpts []premailer.Option
	if premailerOptions != nil {
		pmOpts = premailerOptions.ToFunctionalOptions()
	}
	if styling != "" {
		pmOpts = append(pmOpts, premailer.WithExternalCSS(styling))
	}

	pm := premailer.New(tmplAST, pmOpts...)
	premailedAST, pmErr := pm.Transform()
	if pmErr != nil {
		l.ReportError(span, pmErr, "Premailer transformation failed, falling back to original AST")
		return tmplAST, ""
	}

	if premailedAST != nil && len(premailedAST.Diagnostics) > 0 {
		l.Warn("Premailer generated diagnostics during initial pass", logger_domain.Field("diagnostics", premailedAST.Diagnostics))
	}

	leftoverCSS := ro.extractPremailerLeftoverCSS(ctx, premailedAST)

	return premailedAST, leftoverCSS
}

// extractPremailerLeftoverCSS extracts CSS from the premailer-generated
// head tag and removes the head tag from the AST so it does not appear in the
// body content.
//
// Takes premailedAST (*ast_domain.TemplateAST) which is the AST to search.
//
// Returns string which contains the leftover CSS, or empty if none found.
func (*RenderOrchestrator) extractPremailerLeftoverCSS(
	ctx context.Context,
	premailedAST *ast_domain.TemplateAST,
) string {
	ctx, l := logger_domain.From(ctx, log)
	if premailedAST == nil {
		return ""
	}

	head := premailedAST.Find(func(node *ast_domain.TemplateNode) bool {
		return node.NodeType == ast_domain.NodeElement && node.TagName == "head"
	})
	if head == nil {
		return ""
	}

	for _, child := range head.Children {
		if child.NodeType == ast_domain.NodeElement && child.TagName == "style" && len(child.Attributes) == 0 {
			leftoverCSS := child.Text(ctx)
			premailedAST.RemoveNodes([]*ast_domain.TemplateNode{head})
			l.Trace("Removed premailer-generated <head> tag from AST before PML transformation")
			return leftoverCSS
		}
	}

	return ""
}

// performPmlTransformation transforms the pre-inlined AST using the PML engine
// for email output.
//
// Takes premailedAST (*ast_domain.TemplateAST) which is the pre-inlined AST to
// transform.
//
// Returns *ast_domain.TemplateAST which is the HTML AST after transformation.
// Returns string which is the CSS generated by the PML engine.
func (ro *RenderOrchestrator) performPmlTransformation(
	ctx context.Context,
	premailedAST *ast_domain.TemplateAST,
	isPreviewMode bool,
) (*ast_domain.TemplateAST, string) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Starting PikoML email transformation on pre-inlined AST")
	pmlConfig := pml_dto.DefaultConfig()
	pmlConfig.PreviewMode = isPreviewMode
	if isPreviewMode {
		pmlConfig.AssetServePath = "/_piko/assets"
	}

	htmlAST, pmlGeneratedCSS, assetRequests, pmlErrors := ro.pmlEngine.TransformForEmail(ctx, premailedAST, pmlConfig)

	ro.lastEmailAssetRequests = assetRequests
	l.Trace("Collected email asset requests", logger_domain.Int("requestCount", len(assetRequests)))

	logPmlErrors(ctx, pmlErrors)

	return htmlAST, pmlGeneratedCSS
}

// logEmailDiagnostics logs warnings and errors found during email rendering.
//
// Takes rctx (*renderContext) which contains the diagnostics to log.
func logEmailDiagnostics(ctx context.Context, rctx *renderContext) {
	ctx, l := logger_domain.From(ctx, log)
	if len(rctx.diagnostics.Warnings) > 0 {
		l.Trace("Total warnings during email render", logger_domain.Int("count", len(rctx.diagnostics.Warnings)))
	}
	if len(rctx.diagnostics.Errors) > 0 {
		l.Trace("Total errors during email render", logger_domain.Int("count", len(rctx.diagnostics.Errors)))
	}
	l.Trace("Email rendering completed successfully")
}

// logPmlErrors logs errors from the PikoML transformation.
//
// Takes pmlErrors ([]*pml_domain.Error) which contains the errors to log.
func logPmlErrors(ctx context.Context, pmlErrors []*pml_domain.Error) {
	ctx, l := logger_domain.From(ctx, log)
	for _, pmlErr := range pmlErrors {
		locationString := fmt.Sprintf("L%d:C%d", pmlErr.Location.Line, pmlErr.Location.Column)
		level := l.Warn
		if pmlErr.Severity == pml_domain.SeverityError {
			level = l.Error
		}
		level("PikoML transformation diagnostic",
			logger_domain.String("message", pmlErr.Message),
			logger_domain.String("location", locationString),
			logger_domain.String("tagName", pmlErr.TagName))
	}
}

// combineEmailCSS extracts body styles and combines leftover CSS with
// PML-generated CSS.
//
// Takes premailerLeftoverCSS (string) which contains CSS not inlined by
// premailer.
// Takes pmlGeneratedCSS (string) which contains CSS generated by PML.
//
// Returns bodyInlineStyles (string) which contains styles extracted for the
// body element.
// Returns finalCombinedCSS (string) which contains the merged leftover and
// PML-generated CSS.
func combineEmailCSS(ctx context.Context, premailerLeftoverCSS, pmlGeneratedCSS string) (bodyInlineStyles, finalCombinedCSS string) {
	ctx, l := logger_domain.From(ctx, log)
	leftoverCSS := premailerLeftoverCSS

	if leftoverCSS != "" {
		var cleanedLeftoverCSS string
		bodyInlineStyles, cleanedLeftoverCSS = premailer.ExtractBodyStyles(leftoverCSS)
		leftoverCSS = cleanedLeftoverCSS
		if bodyInlineStyles != "" {
			l.Trace("Extracted body inline styles", logger_domain.String("styles", bodyInlineStyles))
		}
	}

	var finalCSSBuilder strings.Builder
	if leftoverCSS != "" {
		finalCSSBuilder.WriteString(strings.TrimSpace(leftoverCSS))
	}
	if pmlGeneratedCSS != "" {
		if finalCSSBuilder.Len() > 0 {
			finalCSSBuilder.WriteString("\n\n")
		}
		finalCSSBuilder.WriteString(strings.TrimSpace(pmlGeneratedCSS))
	}

	finalCombinedCSS = finalCSSBuilder.String()
	return bodyInlineStyles, finalCombinedCSS
}
