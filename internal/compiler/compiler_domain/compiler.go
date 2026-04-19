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
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	parsejs "github.com/tdewolff/parse/v2/js"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/compiler/compiler_dto"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/esbuild/logger"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/wdk/safeconv"
)

// SFCCompiler compiles single-file component bytes into build artefacts.
type SFCCompiler interface {
	// CompileSFC compiles a single-file component from its raw bytes.
	//
	// Takes sourceID (string) which identifies the source file being compiled.
	// Takes rawSFC ([]byte) which contains the raw SFC content to compile.
	//
	// Returns *compiler_dto.CompiledArtefact which contains the compiled output.
	// Returns error when compilation fails.
	CompileSFC(ctx context.Context, sourceID string, rawSFC []byte) (*compiler_dto.CompiledArtefact, error)
}

// sfcCompiler implements SFCCompiler with a separate registry for each build.
type sfcCompiler struct {
	// cssPreProcessor resolves CSS @import statements before CSS is embedded
	// into compiled output. When nil, raw CSS is used as-is.
	cssPreProcessor CSSPreProcessorPort

	// moduleName is the Go module name from go.mod, such as a GitHub-hosted
	// module path. Used to resolve @/ aliases in asset paths.
	moduleName string
}

var _ SFCCompiler = (*sfcCompiler)(nil)

// CompileSFC implements the SFCCompiler interface.
//
// Takes sourceID (string) which identifies the source file being compiled.
// Takes rawSFC ([]byte) which contains the raw single-file component to parse.
//
// Returns *compiler_dto.CompiledArtefact which contains the compiled output.
// Returns error when compilation fails.
func (c *sfcCompiler) CompileSFC(ctx context.Context, sourceID string, rawSFC []byte) (*compiler_dto.CompiledArtefact, error) {
	return compileSFC(ctx, sourceID, rawSFC, c.moduleName, c.cssPreProcessor)
}

// sfcCompilationContext holds the state needed during SFC compilation.
type sfcCompilationContext struct {
	// registry holds the component registry for tracking dependencies.
	registry *RegistryContext

	// moduleName is the Go module name from go.mod, such as a GitHub-hosted
	// module path. Used to resolve @/ aliases in asset paths.
	moduleName string

	// cssPreProcessor resolves CSS @import statements before CSS is embedded
	// into compiled output. When nil, raw CSS is used as-is.
	cssPreProcessor CSSPreProcessorPort

	// jsParseResult holds the parsed JavaScript AST and type assertions.
	jsParseResult *ParseJSResult

	// sfcParseResult holds the parsed SFC parts after the raw input is read.
	sfcParseResult *sfcparser.ParseResult

	// reactiveTransformResult holds the output from the reactive state transform.
	reactiveTransformResult *ReactiveTransformResult

	// metadata holds the extracted component properties and methods.
	metadata *ComponentMetadata

	// jsAST holds the parsed JavaScript abstract syntax tree.
	jsAST *js_ast.AST

	// scriptCode is the raw JavaScript or TypeScript source from the SFC.
	scriptCode string

	// className is the CSS class name derived from the tag name.
	className string

	// tagName is the HTML custom element tag name for the compiled component.
	tagName string

	// stylesDefault holds the combined CSS from default style blocks.
	stylesDefault string

	// sourceFilename is the filesystem path of the SFC source file, used
	// for deriving the component name when no explicit name is set.
	sourceFilename string

	// scaffoldHTML is the static HTML scaffold for server-side rendering.
	// Defaults to "<slot></slot>" when no template is provided or when
	// scaffold building fails.
	scaffoldHTML string

	// astDump holds the text form of the template AST for debugging.
	astDump string

	// enabledBehaviours holds the list of behaviours enabled for this component.
	// Parsed from the script tag's enable attribute.
	enabledBehaviours []string

	// timelineJSON holds the parsed piko:timeline block as a JSON string,
	// ready for injection as a static property on the component class.
	timelineJSON string

	// jsDependencies holds JavaScript import paths that need registry
	// registration.
	jsDependencies []compiler_dto.JSDependency

	// diagnostics collects non-fatal issues encountered during
	// compilation; these flow into CompiledArtefact.Diagnostics so
	// callers can surface them to the user.
	diagnostics []compiler_dto.CompilationDiagnostic
}

// recordCompilationMetrics records timing and size metrics for SFC compilation.
//
// Takes span (trace.Span) which receives the compilation metrics as attributes.
// Takes startTime (time.Time) which marks when compilation started.
// Takes artefact (*compiler_dto.CompiledArtefact) which provides the compiled
// output for size measurement.
func (cc *sfcCompilationContext) recordCompilationMetrics(ctx context.Context, span trace.Span, startTime time.Time, artefact *compiler_dto.CompiledArtefact) {
	ctx, l := logger_domain.From(ctx, log)
	compilationDuration := time.Since(startTime)
	SFCCompilationDuration.Record(ctx, float64(compilationDuration.Milliseconds()))

	l.Trace("SFC compilation completed",
		logger_domain.String(propTagName, cc.tagName),
		logger_domain.Int64("durationMs", compilationDuration.Milliseconds()),
		logger_domain.Int("jsSize", len(artefact.Files[artefact.BaseJSPath])),
		logger_domain.Int("htmlSize", len(cc.scaffoldHTML)))

	span.SetAttributes(
		attribute.Int64("compilationDuration", compilationDuration.Milliseconds()),
		attribute.Int("jsSize", len(artefact.Files[artefact.BaseJSPath])),
		attribute.Int("htmlSize", len(cc.scaffoldHTML)),
	)
	span.SetStatus(codes.Ok, "SFC compilation completed successfully")
}

// extractScriptAndStyles fills the script and style fields from parsed SFC
// data.
func (cc *sfcCompilationContext) extractScriptAndStyles() {
	if jsScript, found := cc.sfcParseResult.JavaScriptScript(); found {
		cc.scriptCode = jsScript.Content
	}

	if enable, ok := cc.sfcParseResult.TemplateAttributes["enable"]; ok {
		cc.enabledBehaviours = strings.Fields(enable)
	}

	var stylesBuilder strings.Builder
	for _, style := range cc.sfcParseResult.Styles {
		if _, ok := style.Attributes["aesthetic"]; ok {
			continue
		}
		if stylesBuilder.Len() > 0 {
			stylesBuilder.WriteString("\n")
		}
		stylesBuilder.WriteString(style.Content)
	}
	cc.stylesDefault = stylesBuilder.String()
}

// preProcessStyles resolves CSS @import statements in the concatenated style
// content using the CSSPreProcessorPort stored on the compilation context.
// When no pre-processor is available or processing fails, the raw CSS is kept
// as-is.
func (cc *sfcCompilationContext) preProcessStyles(ctx context.Context) {
	if cc.stylesDefault == "" {
		return
	}
	preProcessor := cc.cssPreProcessor
	if preProcessor == nil {
		return
	}
	processed, err := preProcessor.InlineImports(ctx, cc.stylesDefault, cc.sourceFilename)
	if err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("CSS import inlining failed, using raw CSS", logger_domain.Error(err))
		return
	}
	cc.stylesDefault = processed
}

// extractTimeline parses the piko:timeline blocks, if present, and stores the
// resulting JSON in cc.timelineJSON for later injection into the component
// class.
//
// When a single timeline block has no media attribute, the output is a flat
// JSON array of actions for backward compatibility. When multiple blocks exist
// or any block has a media attribute, the output is a JSON array of objects
// with "media" (string or null) and "actions" (array) fields.
func (cc *sfcCompilationContext) extractTimeline(ctx context.Context) {
	if len(cc.sfcParseResult.Timelines) == 0 {
		return
	}
	_, l := logger_domain.From(ctx, log)

	hasMedia := false
	for _, tb := range cc.sfcParseResult.Timelines {
		if tb.Attributes["media"] != "" {
			hasMedia = true
			break
		}
	}

	if len(cc.sfcParseResult.Timelines) == 1 && !hasMedia {
		jsonStr, err := ParseTimeline(cc.sfcParseResult.Timelines[0].Content)
		if err != nil {
			l.Warn("Failed to parse piko:timeline block",
				logger_domain.String(logKeyError, err.Error()))
			return
		}
		cc.timelineJSON = jsonStr
		l.Trace("Parsed piko:timeline block",
			logger_domain.String("timelineJSON", jsonStr))
		return
	}

	var parts []string
	for _, tb := range cc.sfcParseResult.Timelines {
		actionsJSON, err := ParseTimeline(tb.Content)
		if err != nil {
			l.Warn("Failed to parse piko:timeline block",
				logger_domain.String(logKeyError, err.Error()))
			return
		}
		media := tb.Attributes["media"]
		var mediaJSON string
		if media == "" {
			mediaJSON = "null"
		} else {
			mediaJSON = `"` + media + `"`
		}
		parts = append(parts, `{"media":`+mediaJSON+`,"actions":`+actionsJSON+`}`)
	}
	cc.timelineJSON = "[" + strings.Join(parts, ",") + "]"
	l.Trace("Parsed piko:timeline blocks",
		logger_domain.String("timelineJSON", cc.timelineJSON))
}

// injectTimelineData adds the $$timeline static property to the component
// class when timeline data has been parsed.
func (cc *sfcCompilationContext) injectTimelineData(ctx context.Context) {
	if cc.timelineJSON == "" {
		return
	}
	targetClass := findClassDeclarationByName(cc.jsAST, cc.className)
	if targetClass == nil {
		return
	}
	injectStaticProperty(ctx, targetClass, `"$$timeline"`, cc.timelineJSON)
}

// setupNaming resolves the component tag name and class name.
//
// Resolution order:
//  1. <template name="..."> attribute
//  2. Source filename without extension (e.g. my-counter.pkc -> my-counter)
//
// Returns error when the resolved name does not contain a hyphen, which is
// required by the web component specification.
func (cc *sfcCompilationContext) setupNaming() error {
	if name, ok := cc.sfcParseResult.TemplateAttributes["name"]; ok && name != "" {
		cc.tagName = name
	} else if cc.sourceFilename != "" {
		base := filepath.Base(cc.sourceFilename)
		cc.tagName = strings.TrimSuffix(base, filepath.Ext(base))
	}

	if !strings.Contains(cc.tagName, "-") {
		return errors.New("cannot build pkc file, pkc files require a '-' in their name, as per the webcomponent spec")
	}

	cc.className = buildClassName(cc.tagName)
	return nil
}

// setupNamingAndContext sets up naming and adds context to logging and tracing.
//
// Takes ctx (context.Context) which carries the current logger.
// Takes span (trace.Span) which receives the same attributes for tracing.
//
// Returns context.Context which is enriched with the component logger.
// Returns error when naming validation fails.
func (cc *sfcCompilationContext) setupNamingAndContext(ctx context.Context, span trace.Span) (context.Context, error) {
	_, l := logger_domain.From(ctx, log)
	if err := cc.setupNaming(); err != nil {
		return ctx, err
	}
	enrichedL := l.With(
		logger_domain.String(propTagName, cc.tagName),
		logger_domain.String(propClassName, cc.className),
	)
	span.SetAttributes(
		attribute.String(propTagName, cc.tagName),
		attribute.String(propClassName, cc.className),
	)
	return logger_domain.WithLogger(ctx, enrichedL), nil
}

// processJavaScript handles JS parsing, metadata extraction, and reactive
// transformation.
//
// Returns error when the user script contains parse errors such as duplicate
// declarations.
func (cc *sfcCompilationContext) processJavaScript(ctx context.Context) error {
	if err := cc.parseJavaScript(ctx); err != nil {
		return fmt.Errorf("parsing JavaScript for %q: %w", cc.tagName, err)
	}
	cc.extractMetadata(ctx)
	cc.transformReactiveState(ctx)
	return nil
}

// parseJavaScript parses the user script and fills in the AST fields.
//
// Returns error when the user script contains parse errors such as duplicate
// variable declarations. The error signals a permanent compilation failure
// that should not be retried.
func (cc *sfcCompilationContext) parseJavaScript(ctx context.Context) error {
	var parseErr error
	cc.jsParseResult, parseErr = ParseUserScript(ctx, cc.scriptCode, cc.tagName+".ts")
	if parseErr != nil {
		return fmt.Errorf("user script parse error in %s: %w", cc.tagName, parseErr)
	}

	if cc.jsParseResult == nil {
		cc.jsParseResult = &ParseJSResult{AST: &js_ast.AST{}}
	}

	cc.jsAST = cc.jsParseResult.AST
	if cc.jsAST == nil {
		cc.jsAST = &js_ast.AST{}
	}
	ensurePPElementClass(ctx, cc.jsAST, cc.className)
	return nil
}

// extractMetadata parses the JavaScript AST to get component metadata.
//
// When extraction fails, logs a warning and uses empty metadata instead.
func (cc *sfcCompilationContext) extractMetadata(ctx context.Context) {
	_, l := logger_domain.From(ctx, log)
	extractor := NewTypeExtractor(cc.jsAST, cc.jsParseResult.TypeAssertions)
	var err error
	cc.metadata, err = extractor.ExtractMetadata()

	if err != nil {
		l.Warn("Metadata extraction issue", logger_domain.String(logKeyError, err.Error()))
		cc.metadata = NewComponentMetadata()
	} else {
		l.Trace("Metadata extracted successfully",
			logger_domain.Int("propertyCount", len(cc.metadata.StateProperties)),
			logger_domain.Int("methodCount", len(cc.metadata.Methods)),
		)
	}
}

// transformReactiveState applies reactive state changes to the AST.
func (cc *sfcCompilationContext) transformReactiveState(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	ASTTransformationCount.Add(ctx, 1)
	transformStartTime := time.Now()

	var err error
	cc.reactiveTransformResult, err = ReactiveStateTransform(ctx, cc.jsAST, cc.metadata, cc.className, cc.enabledBehaviours, cc.registry)

	ASTTransformationDuration.Record(ctx, float64(time.Since(transformStartTime).Milliseconds()))

	if err != nil {
		l.Warn("ReactiveStateTransform issue", logger_domain.String(logKeyError, err.Error()))
		ASTTransformationErrorCount.Add(ctx, 1)
	}

	if cc.reactiveTransformResult == nil {
		cc.reactiveTransformResult = &ReactiveTransformResult{
			InstanceProperties: []string{},
			BooleanProperties:  []string{},
		}
	}
}

// processTemplate parses the SFC template and transforms it into scaffold HTML
// and a VDOM render method.
//
// Returns error when the template contains syntax errors.
func (cc *sfcCompilationContext) processTemplate(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	if cc.sfcParseResult.Template == "" {
		return nil
	}

	tAST, tErr := ast_domain.ParseAndTransform(ctx, cc.sfcParseResult.Template, cc.tagName)
	if tErr != nil {
		l.Warn("AST parse error for template", logger_domain.String(logKeyError, tErr.Error()))
	}

	if tAST != nil {
		cc.astDump = ast_domain.DumpAST(ctx, tAST)
	}

	if tAST != nil && ast_domain.HasErrors(tAST.Diagnostics) {
		formattedErrors := ast_domain.FormatDiagnostics(cc.tagName, cc.sfcParseResult.Template, tAST.Diagnostics)
		l.Error("Found syntax errors in client component template:\n" + formattedErrors)
		return fmt.Errorf("template for '%s' contains syntax errors", cc.tagName)
	}

	if tAST == nil {
		cc.scaffoldHTML = "<slot></slot>"
		return nil
	}

	cc.buildScaffoldHTML(ctx, tAST)
	return cc.buildVDOMRenderMethod(ctx, tAST, cc.moduleName)
}

// buildScaffoldHTML creates the static HTML scaffold for server-side
// rendering.
//
// Takes tAST (*ast_domain.TemplateAST) which is the parsed template structure.
//
// Reads scaffold settings from the context when available. If scaffold
// building fails, logs a warning and uses a simple slot element as a fallback.
func (cc *sfcCompilationContext) buildScaffoldHTML(ctx context.Context, tAST *ast_domain.TemplateAST) {
	ctx, l := logger_domain.From(ctx, log)
	var scaffoldErr error
	scaffoldConfig := GetScaffoldConfig(ctx)
	scaffoldBuilder := NewScaffoldBuilder(scaffoldConfig)
	cc.scaffoldHTML, scaffoldErr = scaffoldBuilder.BuildStaticScaffold(ctx, tAST, cc.stylesDefault)
	if scaffoldErr != nil {
		l.Warn("Could not generate static scaffold for component, SSR may flicker.",
			logger_domain.String(propTagName, cc.tagName),
			logger_domain.String(logKeyError, scaffoldErr.Error()))
		cc.scaffoldHTML = "<slot></slot>"
	}
}

// buildVDOMRenderMethod builds the virtual DOM render method from the template
// AST and adds it to the JavaScript AST.
//
// Takes tAST (*ast_domain.TemplateAST) which provides the parsed template
// structure.
// Takes moduleName (string) which is the Go module name for @/ alias
// resolution.
//
// Returns error when building fails, though currently returns nil after
// logging a warning.
func (cc *sfcCompilationContext) buildVDOMRenderMethod(ctx context.Context, tAST *ast_domain.TemplateAST, moduleName string) error {
	ctx, l := logger_domain.From(ctx, log)
	events := newEventBindingCollection(cc.registry)
	vdomBuilder := NewVDOMBuilder()
	buildContext := &nodeBuildContext{
		events:       events,
		loopVars:     nil,
		booleanProps: cc.reactiveTransformResult.BooleanProperties,
		moduleName:   moduleName,
	}
	renderMethod, vdomErr := vdomBuilder.BuildRenderVDOM(ctx, tAST, buildContext)
	if vdomErr != nil {
		l.Warn("BuildRenderVDOM error", logger_domain.String(logKeyError, vdomErr.Error()))
		return nil
	}

	insertRenderMethod(ctx, cc.jsAST, cc.className, renderMethod, cc.registry)
	injectEventBindings(ctx, cc.jsAST, cc.className, events)
	return nil
}

// insertStaticCSS adds the default scoped styles to the JavaScript AST.
func (cc *sfcCompilationContext) insertStaticCSS(ctx context.Context) {
	if cc.stylesDefault == "" {
		return
	}
	if err := InsertStaticCSS(ctx, cc.jsAST, cc.stylesDefault, cc.className); err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("InsertStaticCSS failed; component CSS will be dropped",
			logger_domain.String("class", cc.className),
			logger_domain.String("source", cc.sourceFilename),
			logger_domain.Error(err),
		)
		cc.diagnostics = append(cc.diagnostics, compiler_dto.CompilationDiagnostic{
			Severity:         "warning",
			Message:          fmt.Sprintf("component %s: failed to insert static CSS, styling dropped: %v", cc.className, err),
			SourceIdentifier: cc.sourceFilename,
		})
	}
}

// finaliseAST completes AST processing by rewriting it, adding custom element
// definitions when needed, and prepending the import preamble. It also gathers
// JavaScript dependencies from @/ imports for registry registration.
func (cc *sfcCompilationContext) finaliseAST(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Rewriting AST")
	RewriteAST(ctx, cc.jsAST, cc.reactiveTransformResult.InstanceProperties)

	if cc.tagName != "" {
		cc.addCustomElementsDefine(ctx)
	}

	l.Trace("Prepending preamble to AST")
	cc.jsDependencies = prependPreambleToAST(ctx, cc.jsAST, cc.scriptCode, cc.enabledBehaviours, cc.moduleName)

	if len(cc.jsDependencies) > 0 {
		l.Trace("Collected JS dependencies", logger_domain.Int("count", len(cc.jsDependencies)))
	}
}

// addCustomElementsDefine appends a customElements.define statement to the
// JavaScript AST, linking the component's tag name to its class.
func (cc *sfcCompilationContext) addCustomElementsDefine(ctx context.Context) {
	_, l := logger_domain.From(ctx, log)
	l.Trace("Adding customElements.define statement",
		logger_domain.String(propTagName, cc.tagName),
		logger_domain.String(propClassName, cc.className))

	defineSnippet := fmt.Sprintf("customElements.define(%q, %s);", cc.tagName, cc.className)
	statementNode, _ := parseSnippetAsStatement(defineSnippet)
	if statementNode.Data != nil {
		appendStatementToAST(cc.jsAST, statementNode)
	}
}

// buildArtefact creates the final compiled output from the compilation state.
//
// Returns *compiler_dto.CompiledArtefact which holds the generated JavaScript
// code and metadata for the component.
func (cc *sfcCompilationContext) buildArtefact(ctx context.Context) *compiler_dto.CompiledArtefact {
	var builder strings.Builder
	if cc.astDump != "" {
		builder.WriteString(cc.astDump)
		builder.WriteString("\n\n")
	}

	mainJS := builder.String() + printAST(ctx, cc.jsAST, cc.reactiveTransformResult.InstanceProperties, cc.registry)
	mainJSFileName := fmt.Sprintf("%s.js", cc.tagName)

	return &compiler_dto.CompiledArtefact{
		TagName:          cc.tagName,
		ScaffoldHTML:     cc.scaffoldHTML,
		BaseJSPath:       mainJSFileName,
		SourceIdentifier: cc.sourceFilename,
		Files: map[string]string{
			mainJSFileName: mainJS,
		},
		JSDependencies: cc.jsDependencies,
		Diagnostics:    cc.diagnostics,
	}
}

// NewSFCCompiler creates a new compiler for single-file components.
//
// Takes moduleName (string) which is the Go module name for @/ alias
// resolution.
// Takes cssPreProcessor (CSSPreProcessorPort) which resolves CSS @import
// statements, or nil when not needed.
//
// Returns SFCCompiler which is ready to compile single-file components.
func NewSFCCompiler(moduleName string, cssPreProcessor CSSPreProcessorPort) SFCCompiler {
	return &sfcCompiler{moduleName: moduleName, cssPreProcessor: cssPreProcessor}
}

// getStmtsFromAST extracts all statements from an esbuild AST.
//
// Takes tree (*js_ast.AST) which is the parsed AST to extract statements from.
//
// Returns []js_ast.Stmt which contains all statements from all parts of the
// AST, or nil when tree is nil.
func getStmtsFromAST(tree *js_ast.AST) []js_ast.Stmt {
	if tree == nil {
		return nil
	}
	var statements []js_ast.Stmt
	for partIndex := range tree.Parts {
		statements = append(statements, tree.Parts[partIndex].Stmts...)
	}
	return statements
}

// setStmtsInAST sets the statements in an esbuild AST, placing them into a
// single Part.
//
// When tree is nil, returns without making changes.
//
// Takes tree (*js_ast.AST) which is the AST to update.
// Takes statements ([]js_ast.Stmt) which are the statements to set.
func setStmtsInAST(tree *js_ast.AST, statements []js_ast.Stmt) {
	if tree == nil {
		return
	}
	if len(tree.Parts) == 0 {
		tree.Parts = []js_ast.Part{{}}
	}
	tree.Parts[0].Stmts = statements
	if len(tree.Parts) > 1 {
		tree.Parts = tree.Parts[:1]
	}
}

// appendStatementToAST adds a statement to the start of an esbuild AST.
//
// When tree is nil, returns at once without changes. When tree has no parts,
// creates an empty part before adding the statement.
//
// Takes tree (*js_ast.AST) which is the AST to modify.
// Takes statement (js_ast.Stmt) which is the statement to add.
func appendStatementToAST(tree *js_ast.AST, statement js_ast.Stmt) {
	if tree == nil {
		return
	}
	if len(tree.Parts) == 0 {
		tree.Parts = []js_ast.Part{{}}
	}
	tree.Parts[0].Stmts = append(tree.Parts[0].Stmts, statement)
}

// compileSFC compiles a raw single-file component into a compiled artefact.
//
// Takes sourceID (string) which identifies the source file being compiled.
// Takes rawSFC ([]byte) which contains the raw SFC content to compile.
// Takes moduleName (string) which is the Go module name for @/ alias
// resolution.
// Takes cssPreProcessor (CSSPreProcessorPort) which resolves CSS @import
// statements, or nil when not needed.
//
// Returns *compiler_dto.CompiledArtefact which contains the compiled output.
// Returns error when SFC parsing or template processing fails.
func compileSFC(ctx context.Context, sourceID string, rawSFC []byte, moduleName string, cssPreProcessor CSSPreProcessorPort) (*compiler_dto.CompiledArtefact, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "compileSFC",
		logger_domain.Int("rawSFCSize", len(rawSFC)),
	)
	defer span.End()

	SFCCompilationCount.Add(ctx, 1)
	startTime := time.Now()
	l.Trace("Starting SFC compilation")

	cc := &sfcCompilationContext{
		registry:        NewRegistryContext(),
		moduleName:      moduleName,
		cssPreProcessor: cssPreProcessor,
		sourceFilename:  sourceID,
	}

	ccCtx := logger_domain.WithLogger(ctx, l)

	var err error
	cc.sfcParseResult, err = sfcparser.Parse(rawSFC)
	if err != nil {
		l.ReportError(span, err, "SFC parsing failed")
		SFCCompilationErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("sfcparser failed: %w", err)
	}

	cc.extractScriptAndStyles()
	cc.preProcessStyles(ccCtx)
	cc.extractTimeline(ccCtx)

	ccCtx, err = cc.setupNamingAndContext(ccCtx, span)
	if err != nil {
		l.ReportError(span, err, "naming validation failed")
		SFCCompilationErrorCount.Add(ctx, 1)
		return nil, err
	}

	if err := cc.processJavaScript(ccCtx); err != nil {
		l.Trace("JavaScript processing failed", logger_domain.Error(err))
		SFCCompilationErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("processing javascript: %w", err)
	}

	cc.injectTimelineData(ccCtx)

	if err := cc.processTemplate(ccCtx); err != nil {
		return nil, fmt.Errorf("processing template: %w", err)
	}

	cc.insertStaticCSS(ccCtx)

	cc.finaliseAST(ccCtx)

	artefact := cc.buildArtefact(ccCtx)

	cc.recordCompilationMetrics(ctx, span, startTime, artefact)
	return artefact, nil
}

// ensurePPElementClass creates a default PPElement subclass if one does not
// already exist in the AST.
//
// When a class with the given name already exists, returns without changes.
//
// Takes tree (*js_ast.AST) which is the syntax tree to search and modify.
// Takes className (string) which is the name of the class to create if missing.
func ensurePPElementClass(ctx context.Context, tree *js_ast.AST, className string) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "ensurePPElementClass")
	defer span.End()

	if findClassDeclarationByName(tree, className) != nil {
		return
	}
	l.Trace("No existing class with name, creating default",
		logger_domain.String(propClassName, className))

	snippet := fmt.Sprintf(`class %s extends PPElement {}`, className)
	classStmt, parseErr := parseSnippetAsStatement(snippet)
	if parseErr != nil {
		l.Error("Failed to parse fallback class snippet",
			logger_domain.String(logKeyError, parseErr.Error()),
			logger_domain.String("snippet", snippet))
		return
	}
	if classStmt.Data != nil {
		appendStatementToAST(tree, classStmt)
	}
}

// insertMethodIntoClass adds a method to an existing class declaration in the
// AST.
//
// Takes fullAst (*js_ast.AST) which is the parsed JavaScript AST to modify.
// Takes className (string) which is the name of the target class.
// Takes method (*js_ast.EFunction) which is the method to add.
// Takes registry (*RegistryContext) which is used to create identifiers.
//
// Returns error when the method is nil or the target class cannot be found.
func insertMethodIntoClass(
	ctx context.Context,
	fullAst *js_ast.AST,
	className string,
	method *js_ast.EFunction,
	registry *RegistryContext,
) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "insertMethodIntoClass",
		logger_domain.String(propClassName, className),
	)
	defer span.End()

	if method == nil {
		return errors.New("method to insert is nil")
	}

	targetClass := findClassDeclarationByName(fullAst, className)
	if targetClass == nil {
		err := fmt.Errorf("target class %q not found for method insertion", className)
		l.ReportError(span, err, "Target class not found")
		return fmt.Errorf("inserting method into class: %w", err)
	}

	methodProp := js_ast.Property{
		Key:        registry.MakeIdentifierExpr("renderVDOM"),
		ValueOrNil: js_ast.Expr{Data: method},
		Kind:       js_ast.PropertyMethod,
	}
	targetClass.Properties = append(targetClass.Properties, methodProp)
	l.Trace("Successfully inserted method into class", logger_domain.String(propClassName, className))
	return nil
}

// buildClassName converts a hyphen-separated tag name to PascalCase.
//
// Takes rawTag (string) which is the tag name with hyphens between words.
//
// Returns string which is the PascalCase name with "Element" added at the end.
func buildClassName(rawTag string) string {
	parts := strings.Split(rawTag, "-")
	var result strings.Builder
	titleCaser := cases.Title(language.English)
	for _, p := range parts {
		if p == "" {
			continue
		}
		result.WriteString(titleCaser.String(p))
	}
	result.WriteString("Element")
	return result.String()
}

// prependPreambleToAST modifies the AST by adding imports at the start and
// wrapping existing statements in an IIFE.
//
// Extracts imports from the source code using AST-based parsing rather than
// regex. Handles all valid JavaScript import syntax including multi-line
// imports, aliased imports, and type imports. Converts @/ alias paths to served
// asset paths.
//
// When enabledBehaviours includes "animation", a side-effect import for the
// animation extension is prepended before the core import so that the
// extension's global is registered before the component class is defined.
//
// Takes tree (*js_ast.AST) which is the syntax tree to modify in place.
// Takes sourceCode (string) which is the original source for extracting
// import text.
// Takes enabledBehaviours ([]string) which lists behaviours enabled on the
// component.
//
// Returns []compiler_dto.JSDependency which contains dependencies that need
// registry registration.
func prependPreambleToAST(ctx context.Context, tree *js_ast.AST, sourceCode string, enabledBehaviours []string, moduleName string) []compiler_dto.JSDependency {
	existingStmts := getStmtsFromAST(tree)

	_, nonImportStmts := separateImportsFromAST(existingStmts)

	userImportStmts, dependencies := buildImportStatementsFromSource(ctx, tree, sourceCode, moduleName)

	iifeStatement := buildIIFEWrapper(nonImportStmts)
	coreImport := buildCoreImport(tree)
	componentsImport := buildComponentsImport(tree)
	actionsImport := buildActionsImport(tree)

	newStmtList := make([]js_ast.Stmt, 0, 6+len(userImportStmts))

	if slices.Contains(enabledBehaviours, "animation") {
		animationImport := buildAnimationExtensionImport(tree)
		newStmtList = append(newStmtList, animationImport)
	}

	newStmtList = append(newStmtList, coreImport, componentsImport, actionsImport)
	newStmtList = append(newStmtList, userImportStmts...)
	newStmtList = append(newStmtList,
		js_ast.Stmt{Data: &js_ast.SEmpty{}},
		iifeStatement,
	)

	setStmtsInAST(tree, newStmtList)
	return dependencies
}

// buildImportStatementsFromSource builds SImport statements by extracting the
// original import text from source code.
//
// This keeps the exact import syntax (named vs namespace) from the source. This
// matters because esbuild may change named imports to namespace imports
// internally.
//
// The function uses ImportRecords for:
//   - Range data to find import paths in source
//   - Knowing how many imports exist
//
// But it rebuilds imports from source text to keep:
//   - Named imports: `import { foo, bar as baz } from '...'`
//   - Default imports: `import foo from '...'`
//   - Multi-line formatting
//
// When tree is nil or has no ImportRecords, returns nil for both values.
//
// Takes tree (*js_ast.AST) which holds ImportRecords with Range data.
// Takes sourceCode (string) which is the original source code.
//
// Returns []js_ast.Stmt which holds the built import statements.
// Returns []compiler_dto.JSDependency which holds dependencies for the
// registry.
func buildImportStatementsFromSource(ctx context.Context, tree *js_ast.AST, sourceCode string, moduleName string) ([]js_ast.Stmt, []compiler_dto.JSDependency) {
	ctx, l := logger_domain.From(ctx, log)
	if tree == nil || len(tree.ImportRecords) == 0 {
		return nil, nil
	}

	statements := make([]js_ast.Stmt, 0, len(tree.ImportRecords))
	dependencies := make([]compiler_dto.JSDependency, 0)

	for i, record := range tree.ImportRecords {
		importText := extractImportTextFromSource(sourceCode, record)
		if importText == "" {
			l.Trace("Could not extract import text",
				logger_domain.Int("recordIndex", i),
				logger_domain.String("path", record.Path.Text))
			continue
		}

		statement, statementAST, err := parseModuleLevelStatement(ctx, importText)
		if err != nil || statement.Data == nil {
			l.Trace("Failed to parse import statement",
				logger_domain.String("import", importText),
				logger_domain.Int("recordIndex", i))
			continue
		}

		if simport, ok := statement.Data.(*js_ast.SImport); ok && len(statementAST.ImportRecords) > 0 {
			originalPath := statementAST.ImportRecords[0].Path.Text
			transformedPath, dependency := TransformJSImportPath(ctx, originalPath, moduleName)
			if dependency != nil {
				statementAST.ImportRecords[0].Path.Text = transformedPath
				dependencies = append(dependencies, *dependency)
				l.Trace("Transformed import path",
					logger_domain.String("original", originalPath),
					logger_domain.String("transformed", transformedPath))
			}

			mergeImportRecords(tree, statementAST, &statement)

			simport.ImportRecordIndex = safeconv.IntToUint32(len(tree.ImportRecords) - 1)
		}

		statements = append(statements, statement)
	}

	return statements, dependencies
}

// extractImportTextFromSource gets the full import statement text from source
// code using the ImportRecord's Range data.
//
// The Range in ImportRecord points to the path string, including quotes. This
// function searches backwards for the "import" keyword and forwards for the
// statement end to get the complete import text.
//
// Takes sourceCode (string) which is the full source code.
// Takes record (ast.ImportRecord) which contains the Range pointing to the
// path.
//
// Returns string which is the full import statement text, or empty if not
// found.
func extractImportTextFromSource(sourceCode string, record ast.ImportRecord) string {
	if sourceCode == "" {
		return ""
	}

	pathStart := int(record.Range.Loc.Start)
	pathEnd := pathStart + int(record.Range.Len)

	if pathStart < 0 || pathEnd > len(sourceCode) {
		return ""
	}

	importStart := findImportKeyword(sourceCode, pathStart)
	if importStart == -1 {
		return ""
	}

	statementEnd := findStatementEnd(sourceCode, pathEnd)

	return strings.TrimSpace(sourceCode[importStart:statementEnd])
}

// findImportKeyword searches backwards from pathStart to find the "import"
// keyword. It checks that "import" is at a word boundary and not part of a
// longer name.
//
// Takes sourceCode (string) which contains the source text to search.
// Takes pathStart (int) which is the position to start searching backwards
// from.
//
// Returns int which is the start index of "import", or -1 if not found.
func findImportKeyword(sourceCode string, pathStart int) int {
	searchStart := pathStart
	for searchStart >= 0 {
		chunkStart := max(0, searchStart-200)
		chunk := sourceCode[chunkStart:searchStart]
		index := strings.LastIndex(chunk, "import")
		if index == -1 {
			if chunkStart == 0 {
				break
			}
			searchStart = chunkStart
			continue
		}

		position := chunkStart + index
		if position > 0 && isIdentifierChar(sourceCode[position-1]) {
			searchStart = position
			continue
		}
		return position
	}
	return -1
}

// findStatementEnd finds where an import statement ends in the source code.
// It starts from the given position and looks for a semicolon or newline.
//
// Takes sourceCode (string) which is the source text to search.
// Takes pathEnd (int) which is the position after the closing quote.
//
// Returns int which is the position after the statement ends.
func findStatementEnd(sourceCode string, pathEnd int) int {
	statementEnd := pathEnd
	for statementEnd < len(sourceCode) {
		character := sourceCode[statementEnd]
		if character == ';' {
			statementEnd++
			break
		}
		if character == '\n' && statementEnd > pathEnd {
			break
		}
		statementEnd++
	}
	return statementEnd
}

// isIdentifierChar reports whether c can be part of a JavaScript identifier.
//
// Takes c (byte) which is the character to check.
//
// Returns bool which is true if c is a letter, digit, underscore, or dollar
// sign.
func isIdentifierChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c == '$'
}

// separateImportsFromAST walks the AST statements and separates SImport
// statements from all other statements. Used to place imports at module top
// level while wrapping other code in an IIFE.
//
// esbuild stores import information in two places:
// 1. ImportRecords - contains path, range, and metadata for each import
// 2. Parts[].Stmts - contains SImport statements with ImportRecordIndex
//
// Extracts SImport statements from Parts[].Stmts so they can be placed at
// module top level, while other statements get wrapped in an IIFE.
//
// Takes statements ([]js_ast.Stmt) which contains all statements from the parsed
// AST.
//
// Returns imports ([]js_ast.Stmt) which contains only the import statements.
// Returns nonImports ([]js_ast.Stmt) which contains all non-import statements.
func separateImportsFromAST(statements []js_ast.Stmt) (imports []js_ast.Stmt, nonImports []js_ast.Stmt) {
	imports = make([]js_ast.Stmt, 0)
	nonImports = make([]js_ast.Stmt, 0, len(statements))

	for _, statement := range statements {
		if _, isImport := statement.Data.(*js_ast.SImport); isImport {
			imports = append(imports, statement)
		} else {
			nonImports = append(nonImports, statement)
		}
	}
	return imports, nonImports
}

// buildIIFEWrapper wraps the given statements in an immediately invoked
// function expression (IIFE).
//
// Takes statements ([]js_ast.Stmt) which contains the statements to wrap.
//
// Returns js_ast.Stmt which is the IIFE call statement.
func buildIIFEWrapper(statements []js_ast.Stmt) js_ast.Stmt {
	arrowBody := js_ast.FnBody{
		Block: js_ast.SBlock{Stmts: statements},
	}
	arrowFunc := &js_ast.EArrow{
		Body:       arrowBody,
		PreferExpr: false,
		IsAsync:    false,
		HasRestArg: false,
	}
	iifeCall := js_ast.Expr{Data: &js_ast.ECall{
		Target: js_ast.Expr{Data: arrowFunc},
		Args:   []js_ast.Expr{},
	}}
	return js_ast.Stmt{Data: &js_ast.SExpr{Value: iifeCall}}
}

// buildCoreImport creates the core framework import statement for the AST.
//
// Takes tree (*js_ast.AST) which receives the new import record.
//
// Returns js_ast.Stmt which is the import statement for the core framework
// module. This imports the piko namespace.
func buildCoreImport(tree *js_ast.AST) js_ast.Stmt {
	coreImportPath := "/_piko/dist/ppframework.core.es.js"
	coreImportRecord := ast.ImportRecord{
		Path: logger.Path{Text: coreImportPath},
		Kind: ast.ImportStmt,
	}
	coreImportRecordIndex := safeconv.IntToUint32(len(tree.ImportRecords))
	tree.ImportRecords = append(tree.ImportRecords, coreImportRecord)

	coreImportItems := []js_ast.ClauseItem{
		{Alias: "piko", OriginalName: "piko"},
	}

	return js_ast.Stmt{Data: &js_ast.SImport{
		Items:             &coreImportItems,
		ImportRecordIndex: coreImportRecordIndex,
		IsSingleLine:      true,
	}}
}

// buildComponentsImport creates the components extension import statement for
// the AST.
//
// Takes tree (*js_ast.AST) which receives the new import record.
//
// Returns js_ast.Stmt which is the import statement for the components
// extension. This imports PPElement, dom, and makeReactive.
func buildComponentsImport(tree *js_ast.AST) js_ast.Stmt {
	componentsImportPath := "/_piko/dist/ppframework.components.es.js"
	componentsImportRecord := ast.ImportRecord{
		Path: logger.Path{Text: componentsImportPath},
		Kind: ast.ImportStmt,
	}
	componentsImportRecordIndex := safeconv.IntToUint32(len(tree.ImportRecords))
	tree.ImportRecords = append(tree.ImportRecords, componentsImportRecord)

	componentsImportItems := []js_ast.ClauseItem{
		{Alias: "PPElement", OriginalName: "PPElement"},
		{Alias: "dom", OriginalName: "dom"},
		{Alias: "makeReactive", OriginalName: "makeReactive"},
	}

	return js_ast.Stmt{Data: &js_ast.SImport{
		Items:             &componentsImportItems,
		ImportRecordIndex: componentsImportRecordIndex,
		IsSingleLine:      true,
	}}
}

// buildActionsImport creates the import statement for project actions.
//
// This imports the generated action namespace from the asset server. It allows
// pkc components to use typed action calls like action.media.search({}).call().
//
// Takes tree (*js_ast.AST) which receives the new import record.
//
// Returns js_ast.Stmt which is the import statement for the project's generated
// actions module.
func buildActionsImport(tree *js_ast.AST) js_ast.Stmt {
	actionsImportPath := "/_piko/assets/pk-js/pk/actions.gen.js"
	actionsImportRecord := ast.ImportRecord{
		Path: logger.Path{Text: actionsImportPath},
		Kind: ast.ImportStmt,
	}
	actionsImportRecordIndex := safeconv.IntToUint32(len(tree.ImportRecords))
	tree.ImportRecords = append(tree.ImportRecords, actionsImportRecord)

	actionsImportItems := []js_ast.ClauseItem{
		{Alias: "action", OriginalName: "action"},
	}

	return js_ast.Stmt{Data: &js_ast.SImport{
		Items:             &actionsImportItems,
		ImportRecordIndex: actionsImportRecordIndex,
		IsSingleLine:      true,
	}}
}

// buildAnimationExtensionImport creates a side-effect import for the animation
// extension. This is a bare import with no bindings; it runs the extension's
// module-level code which registers the timeline setup function on a global.
//
// Takes tree (*js_ast.AST) which receives the new import record.
//
// Returns js_ast.Stmt which is the side-effect import statement.
func buildAnimationExtensionImport(tree *js_ast.AST) js_ast.Stmt {
	animImportPath := "/_piko/dist/ppframework.animation.es.js"
	animImportRecord := ast.ImportRecord{
		Path: logger.Path{Text: animImportPath},
		Kind: ast.ImportStmt,
	}
	animImportRecordIndex := safeconv.IntToUint32(len(tree.ImportRecords))
	tree.ImportRecords = append(tree.ImportRecords, animImportRecord)

	return js_ast.Stmt{Data: &js_ast.SImport{
		ImportRecordIndex: animImportRecordIndex,
		IsSingleLine:      true,
	}}
}

// mergeImportRecords combines import records from a statement AST into the main
// tree AST. It updates symbol and import record indices to avoid conflicts.
//
// When statementAST is nil or has no import records, returns at once.
//
// Takes tree (*js_ast.AST) which is the target AST to merge records into.
// Takes statementAST (*js_ast.AST) which holds the import records to merge.
// Takes statement (*js_ast.Stmt) which is the import statement to update indices
// for.
func mergeImportRecords(tree *js_ast.AST, statementAST *js_ast.AST, statement *js_ast.Stmt) {
	if statementAST == nil || len(statementAST.ImportRecords) == 0 {
		return
	}

	symbolBaseIndex := safeconv.IntToUint32(len(tree.Symbols))

	if simport, ok := statement.Data.(*js_ast.SImport); ok {
		importRecordBaseIndex := safeconv.IntToUint32(len(tree.ImportRecords))
		simport.ImportRecordIndex += importRecordBaseIndex

		if simport.DefaultName != nil {
			simport.DefaultName.Ref.InnerIndex += symbolBaseIndex
		}
		if simport.Items != nil {
			for i := range *simport.Items {
				(*simport.Items)[i].Name.Ref.InnerIndex += symbolBaseIndex
			}
		}
		if simport.StarNameLoc != nil || simport.NamespaceRef.InnerIndex != 0 {
			simport.NamespaceRef.InnerIndex += symbolBaseIndex
		}
	}
	tree.ImportRecords = append(tree.ImportRecords, statementAST.ImportRecords...)
	if len(statementAST.Symbols) > 0 {
		tree.Symbols = append(tree.Symbols, statementAST.Symbols...)
	}
}

// printAST converts an esbuild AST to JavaScript source code and rewrites
// identifiers to add this.$$ctx. prefix for instance properties.
//
// Takes tree (*js_ast.AST) which is the esbuild AST to convert.
// Takes instanceProps ([]string) which lists the instance property names to
// prefix.
// Takes registry (*RegistryContext) which provides the context for conversion.
//
// Returns string which is the generated JavaScript source code, or an empty
// string if tree is nil.
func printAST(ctx context.Context, tree *js_ast.AST, instanceProps []string, registry *RegistryContext) string {
	_, l := logger_domain.From(ctx, log)
	if tree == nil {
		return ""
	}

	statements := getStmtsFromAST(tree)
	l.Trace("printAST called",
		logger_domain.Int("parts", len(tree.Parts)),
		logger_domain.Int("statements", len(statements)),
		logger_domain.Int("symbols", len(tree.Symbols)))

	tdewolffAST, err := ConvertEsbuildToTdewolff(tree, registry)
	if err != nil {
		l.Warn("Failed to convert AST for printing",
			logger_domain.String(logKeyError, err.Error()))
		return "/* AST conversion error */"
	}

	RewriteTdewolffAST(tdewolffAST, instanceProps)

	return printTdewolffAST(tdewolffAST)
}

// printTdewolffAST converts a tdewolff AST back to JavaScript source code.
//
// Takes tree (*parsejs.AST) which is the parsed JavaScript syntax tree.
//
// Returns string which contains the JavaScript source code, or an empty string
// if tree is nil.
func printTdewolffAST(tree *parsejs.AST) string {
	if tree == nil {
		return ""
	}

	var builder strings.Builder
	for i, statement := range tree.List {
		if i > 0 {
			builder.WriteString("\n")
		}
		statement.JS(&builder)
	}

	return builder.String()
}

// insertRenderMethod inserts the renderVDOM method into the target class.
//
// Takes jsAST (*js_ast.AST) which is the JavaScript AST to modify.
// Takes className (string) which names the target class.
// Takes renderMethod (*js_ast.EFunction) which is the method to insert.
// Takes registry (*RegistryContext) which provides the registry context.
func insertRenderMethod(ctx context.Context, jsAST *js_ast.AST, className string, renderMethod *js_ast.EFunction, registry *RegistryContext) {
	ctx, l := logger_domain.From(ctx, log)
	MethodInsertionCount.Add(ctx, 1)
	insertStartTime := time.Now()

	insertErr := insertMethodIntoClass(ctx, jsAST, className, renderMethod, registry)

	MethodInsertionDuration.Record(ctx, float64(time.Since(insertStartTime).Milliseconds()))

	if insertErr != nil {
		l.Warn("Could not insert renderVDOM method", logger_domain.String(logKeyError, insertErr.Error()))
		MethodInsertionErrorCount.Add(ctx, 1)
	}
}

// injectEventBindings adds event bindings to a class constructor.
//
// Takes jsAST (*js_ast.AST) which provides the JavaScript AST to change.
// Takes className (string) which names the target class.
// Takes events (*eventBindingCollection) which holds the bindings to add.
func injectEventBindings(ctx context.Context, jsAST *js_ast.AST, className string, events *eventBindingCollection) {
	ctx, l := logger_domain.From(ctx, log)
	if len(events.getBindings()) == 0 {
		return
	}

	targetClass := findClassDeclarationByName(jsAST, className)
	if targetClass == nil {
		return
	}

	if injErr := injectEventBindingsIntoConstructor(ctx, targetClass, events); injErr != nil {
		l.Warn("Failed to inject event bindings into constructor",
			logger_domain.String(logKeyError, injErr.Error()))
	}
}
