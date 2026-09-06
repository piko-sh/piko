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
	"piko.sh/piko/internal/jsimport"
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
	// cssPreProcessor resolves CSS @import statements before CSS is embedded into compiled
	// output. When nil, raw CSS is used as-is.
	cssPreProcessor CSSPreProcessorPort

	// moduleName is the Go module name from go.mod, such as a GitHub-hosted module path.
	// Used to resolve @/ aliases in asset paths.
	moduleName string
}

const (
	// noMergedImportRecord marks a rebuilt statement that carries no import record of its
	// own, so there is nothing in the component's record list for it to resolve.
	noMergedImportRecord = -1
)

var (
	_ SFCCompiler = (*sfcCompiler)(nil)

	// errImportNotRecovered is returned when a hoisted import's text cannot be cut out of
	// the script exactly as written, so the component fails to build instead of shipping a
	// fused or missing import.
	errImportNotRecovered = errors.New("import statement could not be recovered from the script")

	// errReexportUnsupported is returned for an export-from statement, which has no place in
	// a component script.
	errReexportUnsupported = errors.New("re-exports are not supported in component scripts")

	// errNestedImportUnsupported is returned for an import that no top-level statement owns,
	// such as one inside a declare module block, which a component script cannot hoist.
	errNestedImportUnsupported = errors.New("imports inside a declare module block are not supported in component scripts; move the import to the top level of the script")

	// errStatementElided is returned when a module-level snippet parses cleanly but the
	// TypeScript parser removes every statement in it, as it does for an import whose
	// bindings are all type-only.
	errStatementElided = errors.New("statement holds only TypeScript type information and was removed")
)

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

	// moduleName is the Go module name from go.mod, such as a GitHub-hosted module path.
	// Used to resolve @/ aliases in asset paths.
	moduleName string

	// cssPreProcessor resolves CSS @import statements before CSS is embedded into compiled
	// output. When nil, raw CSS is used as-is.
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

	// sourceFilename is the filesystem path of the SFC source file, used for deriving the
	// component name when no explicit name is set.
	sourceFilename string

	// scaffoldHTML is the static HTML scaffold for server-side rendering. Defaults to
	// "<slot></slot>" when no template is provided or when scaffold building fails.
	scaffoldHTML string

	// astDump holds the text form of the template AST for debugging.
	astDump string

	// enabledBehaviours holds the list of behaviours enabled for this component. Parsed from
	// the script tag's enable attribute.
	enabledBehaviours []string

	// timelineJSON holds the parsed piko:timeline block as a JSON string, ready for
	// injection as a static property on the component class.
	timelineJSON string

	// jsDependencies holds JavaScript import paths that need registry registration.
	jsDependencies []compiler_dto.JSDependency

	// stylesLocation is where the first contributing style block's content begins in the
	// source file, so a style failure is reported at the line the author wrote.
	stylesLocation ast_domain.Location
}

// recordCompilationMetrics records timing and size metrics for SFC compilation.
//
// Takes span (trace.Span) which receives the compilation metrics as attributes.
// Takes startTime (time.Time) which marks when compilation started.
// Takes artefact (*compiler_dto.CompiledArtefact) which provides the compiled output for
// size measurement.
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

// extractScriptAndStyles fills the script and style fields from parsed SFC data.
func (cc *sfcCompilationContext) extractScriptAndStyles() {
	if jsScript, found := cc.sfcParseResult.JavaScriptScript(); found {
		cc.scriptCode = jsScript.Content
	}

	if enable, ok := cc.sfcParseResult.TemplateAttributes["enable"]; ok {
		cc.enabledBehaviours = strings.Fields(enable)
	}

	var stylesBuilder strings.Builder
	lineInBuffer := 0
	for _, style := range cc.sfcParseResult.Styles {
		if _, ok := style.Attributes["aesthetic"]; ok {
			continue
		}
		if stylesBuilder.Len() == 0 {
			cc.stylesLocation = ast_domain.Location{
				Line:   style.ContentLocation.Line,
				Column: style.ContentLocation.Column,
				Offset: 0,
			}
			lineInBuffer = style.ContentLocation.Line
		} else if padding := style.ContentLocation.Line - lineInBuffer; padding > 0 {
			stylesBuilder.WriteString(strings.Repeat("\n", padding))
			lineInBuffer += padding
		} else {
			stylesBuilder.WriteString("\n")
			lineInBuffer++
		}
		stylesBuilder.WriteString(style.Content)
		lineInBuffer += strings.Count(style.Content, "\n")
	}
	cc.stylesDefault = stylesBuilder.String()
}

// preProcessStyles resolves CSS @import statements in the concatenated style content
// using the CSSPreProcessorPort stored on the compilation context. When no pre-processor
// is available the raw CSS is kept as-is.
//
// Returns error when an @import cannot be resolved or its target cannot be read.
func (cc *sfcCompilationContext) preProcessStyles(ctx context.Context) error {
	if cc.stylesDefault == "" {
		return nil
	}
	preProcessor := cc.cssPreProcessor
	if preProcessor == nil {
		return nil
	}
	processed, err := preProcessor.InlineImports(ctx, cc.stylesDefault, cc.sourceFilename, cc.stylesLocation)
	if err != nil {
		return fmt.Errorf("resolving component styles: %w", err)
	}
	cc.stylesDefault = processed
	return nil
}

// extractTimeline parses the piko:timeline blocks, if present, and stores the resulting
// JSON in cc.timelineJSON for later injection into the component class.
//
// When a single timeline block has no media attribute, the output is a flat JSON array of
// actions for backward compatibility. When multiple blocks exist or any block has a media
// attribute, the output is a JSON array of objects with "media" (string or null) and
// "actions" (array) fields.
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

// injectTimelineData adds the $$timeline static property to the component class when
// timeline data has been parsed.
//
// Returns error when there is timeline data but no component class to hold it.
func (cc *sfcCompilationContext) injectTimelineData(ctx context.Context) error {
	if cc.timelineJSON == "" {
		return nil
	}
	targetClass := findClassDeclarationByName(cc.jsAST, cc.className)
	if targetClass == nil {
		return fmt.Errorf("component class %q not found for the timeline data", cc.className)
	}
	injectStaticProperty(ctx, targetClass, `"$$timeline"`, cc.timelineJSON)
	return nil
}

// setupNaming resolves the component tag name and class name.
//
// Resolution order:
//  1. <template name="..."> attribute
//  2. Source filename without extension (e.g. my-counter.pkc -> my-counter)
//
// Returns error when the resolved name does not contain a hyphen, which is required by
// the web component specification.
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

// processJavaScript handles JS parsing, metadata extraction, and reactive transformation.
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
// Returns error when the user script contains parse errors such as duplicate variable
// declarations. The error signals a permanent compilation failure that should not be
// retried.
func (cc *sfcCompilationContext) parseJavaScript(ctx context.Context) error {
	var parseErr error
	cc.jsParseResult, parseErr = ParseUserScript(ctx, cc.scriptCode, cc.tagName+".ts")
	if parseErr != nil {
		return fmt.Errorf("user script parse error in %s: %w", cc.tagName, parseErr)
	}

	if cc.jsParseResult == nil {
		cc.jsParseResult = &ParseJSResult{AST: &js_ast.AST{}, TypeAssertions: nil, Imports: nil}
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

// processTemplate parses the SFC template and transforms it into scaffold HTML and a VDOM
// render method.
//
// Returns error when the template contains syntax errors, or its render method or event
// bindings cannot be built into the component.
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
		ast_domain.SortAttributesByName(tAST)
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

// buildScaffoldHTML creates the static HTML scaffold for server-side rendering.
//
// Takes tAST (*ast_domain.TemplateAST) which is the parsed template structure.
//
// Reads scaffold settings from the context when available. If scaffold building fails,
// logs a warning and uses a simple slot element as a fallback.
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

// buildVDOMRenderMethod builds the virtual DOM render method from the template AST and
// adds it, with the template's event bindings, to the JavaScript AST.
//
// Takes tAST (*ast_domain.TemplateAST) which provides the parsed template structure.
// Takes moduleName (string) which is the Go module name for @/ alias resolution.
//
// Returns error when the render method cannot be built or added to the component class,
// or the event bindings cannot be added, so a component that would render nothing or
// ignore its events never ships.
func (cc *sfcCompilationContext) buildVDOMRenderMethod(ctx context.Context, tAST *ast_domain.TemplateAST, moduleName string) error {
	events := newEventBindingCollection(cc.registry)
	vdomBuilder := NewVDOMBuilder()
	buildContext := &nodeBuildContext{
		events:       events,
		loopVars:     nil,
		booleanProps: cc.reactiveTransformResult.BooleanProperties,
		moduleName:   moduleName,
	}
	renderMethod, err := vdomBuilder.BuildRenderVDOM(ctx, tAST, buildContext)
	if err != nil {
		return fmt.Errorf("building the render method of component %s: %w", cc.className, err)
	}

	if err := insertRenderMethod(ctx, cc.jsAST, cc.className, renderMethod, cc.registry); err != nil {
		return fmt.Errorf("adding the render method to component %s: %w", cc.className, err)
	}
	if err := injectEventBindings(ctx, cc.jsAST, cc.className, events); err != nil {
		return fmt.Errorf("adding the event bindings of component %s: %w", cc.className, err)
	}
	return nil
}

// insertStaticCSS adds the default scoped styles to the JavaScript AST.
//
// Returns error when the styles cannot be minified or the component class cannot be
// found.
func (cc *sfcCompilationContext) insertStaticCSS(ctx context.Context) error {
	if cc.stylesDefault == "" {
		return nil
	}
	if err := InsertStaticCSS(ctx, cc.jsAST, cc.stylesDefault, cc.className); err != nil {
		return fmt.Errorf("inserting styles for component %s: %w", cc.className, err)
	}
	return nil
}

// finaliseAST completes AST processing by rewriting it, adding custom element definitions
// when needed, and prepending the import preamble. It also gathers JavaScript
// dependencies from @/ imports for registry registration.
//
// Returns error when a user import cannot be hoisted exactly as written.
func (cc *sfcCompilationContext) finaliseAST(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Rewriting AST")
	RewriteAST(ctx, cc.jsAST, cc.reactiveTransformResult.InstanceProperties)

	if cc.tagName != "" {
		cc.addCustomElementsDefine(ctx)
	}

	jsimport.RewriteImportRecords(cc.jsAST.ImportRecords, cc.moduleName)

	l.Trace("Prepending preamble to AST")
	dependencies, err := prependPreambleToAST(ctx, cc.jsAST, cc.scriptCode, cc.enabledBehaviours, cc.moduleName, cc.registry)
	if err != nil {
		return fmt.Errorf("hoisting imports in %s: %w", cc.tagName, err)
	}
	cc.jsDependencies = dependencies

	if len(cc.jsDependencies) > 0 {
		l.Trace("Collected JS dependencies", logger_domain.Int("count", len(cc.jsDependencies)))
	}
	return nil
}

// addCustomElementsDefine appends a customElements.define statement to the JavaScript
// AST, linking the component's tag name to its class.
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
// Returns *compiler_dto.CompiledArtefact which holds the generated JavaScript code and
// metadata for the component.
// Returns error when the script cannot be printed as JavaScript, so a component whose
// script failed to compile never ships.
func (cc *sfcCompilationContext) buildArtefact(ctx context.Context) (*compiler_dto.CompiledArtefact, error) {
	body, err := printAST(ctx, cc.jsAST, cc.reactiveTransformResult.InstanceProperties, cc.registry)
	if err != nil {
		return nil, fmt.Errorf("compiling the script of component %s to JavaScript: %w", cc.className, err)
	}

	var builder strings.Builder
	if cc.astDump != "" {
		builder.WriteString(cc.astDump)
		builder.WriteString("\n\n")
	}
	builder.WriteString(body)

	mainJSFileName := fmt.Sprintf("%s.js", cc.tagName)

	return &compiler_dto.CompiledArtefact{
		TagName:          cc.tagName,
		ScaffoldHTML:     cc.scaffoldHTML,
		BaseJSPath:       mainJSFileName,
		SourceIdentifier: cc.sourceFilename,
		Files: map[string]string{
			mainJSFileName: builder.String(),
		},
		JSDependencies: cc.jsDependencies,
		Diagnostics:    nil,
	}, nil
}

// NewSFCCompiler creates a new compiler for single-file components.
//
// Takes moduleName (string) which is the Go module name for @/ alias resolution.
// Takes cssPreProcessor (CSSPreProcessorPort) which resolves CSS @import statements, or
// nil when not needed.
//
// Returns SFCCompiler which is ready to compile single-file components.
func NewSFCCompiler(moduleName string, cssPreProcessor CSSPreProcessorPort) SFCCompiler {
	return &sfcCompiler{moduleName: moduleName, cssPreProcessor: cssPreProcessor}
}

// getStmtsFromAST extracts all statements from an esbuild AST.
//
// Takes tree (*js_ast.AST) which is the parsed AST to extract statements from.
//
// Returns []js_ast.Stmt which contains all statements from all parts of the AST, or nil
// when tree is nil.
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

// setStmtsInAST sets the statements in an esbuild AST, placing them into a single Part.
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
// When tree is nil, returns at once without changes. When tree has no parts, creates an
// empty part before adding the statement.
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
// Takes moduleName (string) which is the Go module name for @/ alias resolution.
// Takes cssPreProcessor (CSSPreProcessorPort) which resolves CSS @import statements, or
// nil when not needed.
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

	cc := &sfcCompilationContext{}
	cc.registry = NewRegistryContext()
	cc.moduleName = moduleName
	cc.cssPreProcessor = cssPreProcessor
	cc.sourceFilename = sourceID

	ccCtx := logger_domain.WithLogger(ctx, l)

	var err error
	cc.sfcParseResult, err = sfcparser.Parse(rawSFC)
	if err != nil {
		l.ReportError(span, err, "SFC parsing failed")
		SFCCompilationErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("sfcparser failed: %w", err)
	}

	cc.extractScriptAndStyles()
	if err := cc.preProcessStyles(ccCtx); err != nil {
		l.ReportError(span, err, "style pre-processing failed")
		SFCCompilationErrorCount.Add(ctx, 1)
		return nil, err
	}
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

	if err := cc.injectTimelineData(ccCtx); err != nil {
		l.ReportError(span, err, "timeline injection failed")
		SFCCompilationErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("injecting timeline: %w", err)
	}

	if err := cc.processTemplate(ccCtx); err != nil {
		SFCCompilationErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("processing template: %w", err)
	}

	if err := cc.insertStaticCSS(ccCtx); err != nil {
		l.ReportError(span, err, "static CSS insertion failed")
		SFCCompilationErrorCount.Add(ctx, 1)
		return nil, err
	}

	if err := cc.finaliseAST(ccCtx); err != nil {
		l.ReportError(span, err, "finalising script failed")
		SFCCompilationErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("finalising script: %w", err)
	}

	artefact, err := cc.buildArtefact(ccCtx)
	if err != nil {
		l.ReportError(span, err, "printing compiled component failed")
		SFCCompilationErrorCount.Add(ctx, 1)
		return nil, err
	}

	cc.recordCompilationMetrics(ctx, span, startTime, artefact)
	return artefact, nil
}

// ensurePPElementClass creates a default PPElement subclass if one does not already exist
// in the AST.
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

// insertMethodIntoClass adds a method to an existing class declaration in the AST.
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

// prependPreambleToAST modifies the AST by adding imports at the start and wrapping
// existing statements in an IIFE.
//
// Extracts imports from the source code using AST-based parsing rather than regex.
// Handles all valid JavaScript import syntax including multi-line imports, aliased
// imports, and type imports. Converts @/ alias paths to served asset paths.
//
// When enabledBehaviours includes "animation", a side-effect import for the animation
// extension is prepended before the core import so that the extension's global is
// registered before the component class is defined.
//
// Takes tree (*js_ast.AST) which is the syntax tree to modify in place.
// Takes sourceCode (string) which is the original source for extracting import text.
// Takes enabledBehaviours ([]string) which lists behaviours enabled on the component.
// Takes moduleName (string) which resolves the @/ alias.
// Takes registry (*RegistryContext) which resolves registry-backed identifier names.
//
// Returns []compiler_dto.JSDependency which contains dependencies that need registry
// registration.
// Returns error when a user import cannot be recovered from the source.
func prependPreambleToAST(ctx context.Context, tree *js_ast.AST, sourceCode string, enabledBehaviours []string, moduleName string, registry *RegistryContext) ([]compiler_dto.JSDependency, error) {
	existingStmts := getStmtsFromAST(tree)

	_, nonImportStmts := separateImportsFromAST(existingStmts)

	userImportStmts, dependencies, err := buildImportStatementsFromSource(ctx, tree, sourceCode, moduleName)
	if err != nil {
		return nil, err
	}
	userImportStmts = elideTypeOnlyNamedImports(tree, nonImportStmts, userImportStmts, registry)

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
	return dependencies, nil
}

// buildImportStatementsFromSource builds SImport statements by extracting the original
// import text from source code.
//
// This keeps the exact import syntax (named vs namespace) from the source. This matters
// because esbuild may change named imports to namespace imports internally.
//
// The function uses ImportRecords for:
//   - Range data to find import paths in source
//   - Knowing how many imports exist
//
// Only ImportStmt records are hoisted. A dynamic import() belongs where the author wrote
// it, and require-style records carry a Range pointing at a string literal too, so
// searching backwards from one would find an unrelated import keyword. Hoisting either
// kind emits a second, eager fetch of a module the component meant to load lazily or not
// at all.
//
// But it rebuilds imports from source text to keep:
//   - Named imports: `import { foo, bar as baz } from '...'`
//   - Default imports: `import foo from '...'`
//   - Multi-line formatting
//
// Each statement's start and end come from a parser pass that keeps every import (see
// locateImportStatements). An import the TypeScript parser removes because all of its
// bindings are type-only, such as `import { type Foo } from './types'`, is dropped.
//
// When tree is nil or has no ImportRecords, returns nil for all values.
//
// Takes tree (*js_ast.AST) which holds ImportRecords with Range data.
// Takes sourceCode (string) which is the original source code.
// Takes moduleName (string) which resolves the @/ alias.
//
// Returns []js_ast.Stmt which holds the built import statements.
// Returns []compiler_dto.JSDependency which holds dependencies for the registry.
// Returns error when an import statement cannot be recovered exactly as written.
func buildImportStatementsFromSource(ctx context.Context, tree *js_ast.AST, sourceCode string, moduleName string) ([]js_ast.Stmt, []compiler_dto.JSDependency, error) {
	if tree == nil || len(tree.ImportRecords) == 0 {
		return nil, nil, nil
	}
	ctx, l := logger_domain.From(ctx, log)

	locator, err := locateImportStatements(sourceCode)
	if err != nil {
		return nil, nil, err
	}

	statements := make([]js_ast.Stmt, 0, len(tree.ImportRecords))
	dependencies := make([]compiler_dto.JSDependency, 0)

	originalRecords := slices.Clone(tree.ImportRecords)
	for _, record := range originalRecords {
		if record.Kind != ast.ImportStmt {
			continue
		}

		statement, recordIndex, err := rebuildImportStatement(ctx, tree, sourceCode, locator, record)
		if errors.Is(err, errStatementElided) {
			l.Trace("Dropped type-only import", logger_domain.String("path", record.Path.Text))
			continue
		}
		if err != nil {
			return nil, nil, err
		}

		if dependency := resolveStatementImportPath(ctx, tree, recordIndex, moduleName); dependency != nil {
			dependencies = append(dependencies, *dependency)
		}

		statements = append(statements, statement)
	}

	return statements, dependencies, nil
}

// rebuildImportStatement recovers one import statement from the original source text.
//
// Takes tree (*js_ast.AST) which receives the merged import record.
// Takes sourceCode (string) which is the original source.
// Takes locator (importStatementLocator) which gives each statement's span.
// Takes record (ast.ImportRecord) which locates the import in that source.
//
// Returns js_ast.Stmt which is the rebuilt statement.
// Returns int which is the index of the record merged into tree for this statement.
// Returns error when the statement cannot be found, does not parse, or parses to anything
// other than the single import that owns record, or one wrapping errStatementElided when
// the statement holds only type information.
func rebuildImportStatement(
	ctx context.Context,
	tree *js_ast.AST,
	sourceCode string,
	locator importStatementLocator,
	record ast.ImportRecord,
) (js_ast.Stmt, int, error) {
	importText, pathText, err := extractImportTextFromSource(sourceCode, locator, record)
	if err != nil {
		return js_ast.Stmt{}, noMergedImportRecord, err
	}

	statement, statementAST, err := parseModuleLevelStatement(ctx, importText)
	if err != nil {
		return js_ast.Stmt{}, noMergedImportRecord, fmt.Errorf("parsing import of %s: %w", pathText, err)
	}

	simport, ok := statement.Data.(*js_ast.SImport)
	if !ok || !recoversSingleImport(importText, pathText, statementAST) {
		return js_ast.Stmt{}, noMergedImportRecord, fmt.Errorf("recovering import of %s: got %q: %w", pathText, importText, errImportNotRecovered)
	}

	mergeImportRecords(tree, statementAST, &statement)
	mergedIndex := len(tree.ImportRecords) - 1
	simport.ImportRecordIndex = safeconv.IntToUint32(mergedIndex)

	return statement, mergedIndex, nil
}

// recoversSingleImport reports whether recovered text holds exactly the one import it was
// cut out for.
//
// A statement fused with its neighbour would carry two import records, and a mislocated
// one would name another path.
//
// Takes importText (string) which is the recovered statement text.
// Takes pathText (string) which is the quoted module path as written in the source.
// Takes statementAST (*js_ast.AST) which is the parse of importText.
//
// Returns bool which is true when importText imports pathText and nothing else.
func recoversSingleImport(importText string, pathText string, statementAST *js_ast.AST) bool {
	if statementAST == nil || len(statementAST.ImportRecords) != 1 {
		return false
	}
	pathRange := statementAST.ImportRecords[0].Range
	pathStart := int(pathRange.Loc.Start)
	pathEnd := pathStart + int(pathRange.Len)
	if pathStart < 0 || pathEnd > len(importText) {
		return false
	}
	return importText[pathStart:pathEnd] == pathText
}

// resolveStatementImportPath rewrites a hoisted import's specifier to its served URL.
//
// Takes tree (*js_ast.AST) which holds the merged record the statement points at.
// Takes recordIndex (int) which is the record rebuildImportStatement merged, or
// noMergedImportRecord when there is nothing to resolve.
// Takes moduleName (string) which resolves the @/ alias.
//
// Returns *compiler_dto.JSDependency describing the resolved module, or nil when the
// specifier needed no rewriting.
func resolveStatementImportPath(
	ctx context.Context,
	tree *js_ast.AST,
	recordIndex int,
	moduleName string,
) *compiler_dto.JSDependency {
	ctx, l := logger_domain.From(ctx, log)

	if recordIndex == noMergedImportRecord || recordIndex >= len(tree.ImportRecords) {
		return nil
	}

	recordSlot := &tree.ImportRecords[recordIndex]
	originalPath := recordSlot.Path.Text

	transformedPath, dependency := TransformJSImportPath(ctx, originalPath, moduleName)
	if dependency == nil {
		return nil
	}

	recordSlot.Path.Text = transformedPath
	l.Trace("Transformed import path",
		logger_domain.String("original", originalPath),
		logger_domain.String("transformed", transformedPath))

	return dependency
}

// usedIdentifierCollector is an AST visitor that records the name of every identifier
// referenced as a value (a tdewolff Var node). Property keys and member names are
// LiteralExpr or PropertyName nodes and string contents are literals, so none are
// recorded.
type usedIdentifierCollector struct {
	// names holds the set of identifier names referenced as a value.
	names map[string]struct{}
}

// Enter records a Var node's name and continues the walk.
//
// Takes node (parsejs.INode) which is the AST node currently being entered.
//
// Returns parsejs.IVisitor which is the collector itself, so the walk continues.
func (c *usedIdentifierCollector) Enter(node parsejs.INode) parsejs.IVisitor {
	if variable, ok := node.(*parsejs.Var); ok {
		c.names[string(variable.Name())] = struct{}{}
	}
	return c
}

// Exit is required by the visitor interface and does nothing.
func (*usedIdentifierCollector) Exit(_ parsejs.INode) {}

// extractImportTextFromSource gets the full import statement text from source code.
//
// The record's Range points at the module path, including quotes. The statement's span
// comes from locator.
//
// Takes sourceCode (string) which is the full source code.
// Takes locator (importStatementLocator) which gives each statement's span.
// Takes record (ast.ImportRecord) which contains the Range pointing to the path.
//
// Returns importText (string) which is the import statement text, without any trailing
// semicolon.
// Returns pathText (string) which is the quoted module path as written.
// Returns err (error) when the path or statement lies outside the source, or no top-level
// import statement owns the path, as for a re-export or an import inside a declare module
// block.
func extractImportTextFromSource(sourceCode string, locator importStatementLocator, record ast.ImportRecord) (importText string, pathText string, err error) {
	pathStart := int(record.Range.Loc.Start)
	pathEnd := pathStart + int(record.Range.Len)

	if pathStart < 0 || pathEnd > len(sourceCode) || pathStart >= pathEnd {
		return "", "", fmt.Errorf("import of %s: path range outside the script: %w", record.Path.Text, errImportNotRecovered)
	}
	pathText = sourceCode[pathStart:pathEnd]

	span, err := locator.statementSpan(record.Range.Loc.Start)
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", pathText, err)
	}
	if span.start < 0 || span.start > pathStart || span.end < pathEnd || span.end > len(sourceCode) {
		return "", "", fmt.Errorf("import of %s: statement range outside the script: %w", pathText, errImportNotRecovered)
	}

	return sourceCode[span.start:span.end], pathText, nil
}

// separateImportsFromAST walks the AST statements and separates SImport statements from
// all other statements. Used to place imports at module top level while wrapping other
// code in an IIFE.
//
// esbuild stores import information in two places: 1. ImportRecords - contains path,
// range, and metadata for each import 2. Parts[].Stmts - contains SImport statements with
// ImportRecordIndex
//
// Extracts SImport statements from Parts[].Stmts so they can be placed at module top
// level, while other statements get wrapped in an IIFE.
//
// Takes statements ([]js_ast.Stmt) which contains all statements from the parsed AST.
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

// buildIIFEWrapper wraps the given statements in an immediately invoked function
// expression (IIFE).
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
// Returns js_ast.Stmt which is the import statement for the core framework module. This
// imports the piko namespace.
func buildCoreImport(tree *js_ast.AST) js_ast.Stmt {
	coreImportPath := "/_piko/dist/ppframework.core.es.js"
	coreImportRecord := ast.ImportRecord{
		Path: logger.Path{Text: coreImportPath},
		Kind: ast.ImportStmt,
	}
	coreImportRecordIndex := safeconv.IntToUint32(len(tree.ImportRecords))
	tree.ImportRecords = append(tree.ImportRecords, coreImportRecord)

	return js_ast.Stmt{Data: &js_ast.SImport{
		Items:             new([]js_ast.ClauseItem{{Alias: "piko", OriginalName: "piko"}}),
		ImportRecordIndex: coreImportRecordIndex,
		IsSingleLine:      true,
	}}
}

// buildComponentsImport creates the components extension import statement for the AST.
//
// Takes tree (*js_ast.AST) which receives the new import record.
//
// Returns js_ast.Stmt which is the import statement for the components extension. This
// imports PPElement, dom, and makeReactive.
func buildComponentsImport(tree *js_ast.AST) js_ast.Stmt {
	componentsImportPath := "/_piko/dist/ppframework.components.es.js"
	componentsImportRecord := ast.ImportRecord{
		Path: logger.Path{Text: componentsImportPath},
		Kind: ast.ImportStmt,
	}
	componentsImportRecordIndex := safeconv.IntToUint32(len(tree.ImportRecords))
	tree.ImportRecords = append(tree.ImportRecords, componentsImportRecord)

	return js_ast.Stmt{Data: &js_ast.SImport{
		Items:             new([]js_ast.ClauseItem{{Alias: "PPElement", OriginalName: "PPElement"}, {Alias: "dom", OriginalName: "dom"}, {Alias: "makeReactive", OriginalName: "makeReactive"}}),
		ImportRecordIndex: componentsImportRecordIndex,
		IsSingleLine:      true,
	}}
}

// buildActionsImport creates the import statement for project actions.
//
// This imports the generated action namespace from the asset server. It allows pkc
// components to use typed action calls like action.media.search({}).call().
//
// Takes tree (*js_ast.AST) which receives the new import record.
//
// Returns js_ast.Stmt which is the import statement for the project's generated actions
// module.
func buildActionsImport(tree *js_ast.AST) js_ast.Stmt {
	actionsImportPath := "/_piko/assets/pk-js/pk/actions.gen.js"
	actionsImportRecord := ast.ImportRecord{
		Path: logger.Path{Text: actionsImportPath},
		Kind: ast.ImportStmt,
	}
	actionsImportRecordIndex := safeconv.IntToUint32(len(tree.ImportRecords))
	tree.ImportRecords = append(tree.ImportRecords, actionsImportRecord)

	return js_ast.Stmt{Data: &js_ast.SImport{
		Items:             new([]js_ast.ClauseItem{{Alias: "action", OriginalName: "action"}}),
		ImportRecordIndex: actionsImportRecordIndex,
		IsSingleLine:      true,
	}}
}

// buildAnimationExtensionImport creates a side-effect import for the animation extension.
// This is a bare import with no bindings; it runs the extension's module-level code which
// registers the timeline setup function on a global.
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

// mergeImportRecords combines import records from a statement AST into the main tree AST.
// It updates symbol and import record indices to avoid conflicts.
//
// When statementAST is nil or has no import records, returns at once.
//
// Takes tree (*js_ast.AST) which is the target AST to merge records into.
// Takes statementAST (*js_ast.AST) which holds the import records to merge.
// Takes statement (*js_ast.Stmt) which is the import statement to update indices for.
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

// printAST converts an esbuild AST to JavaScript source code and rewrites identifiers to
// add this.$$ctx. prefix for instance properties.
//
// Takes tree (*js_ast.AST) which is the esbuild AST to convert.
// Takes instanceProps ([]string) which lists the instance property names to prefix.
// Takes registry (*RegistryContext) which provides the context for conversion.
//
// Returns string which is the generated JavaScript source code, or empty when tree is
// nil.
// Returns error when the AST cannot be converted for printing.
func printAST(ctx context.Context, tree *js_ast.AST, instanceProps []string, registry *RegistryContext) (string, error) {
	_, l := logger_domain.From(ctx, log)
	if tree == nil {
		return "", nil
	}

	statements := getStmtsFromAST(tree)
	l.Trace("printAST called",
		logger_domain.Int("parts", len(tree.Parts)),
		logger_domain.Int("statements", len(statements)),
		logger_domain.Int("symbols", len(tree.Symbols)))

	tdewolffAST, err := ConvertEsbuildToTdewolff(tree, registry)
	if err != nil {
		return "", fmt.Errorf("converting AST for printing: %w", err)
	}

	RewriteTdewolffAST(tdewolffAST, instanceProps)

	printed, err := printTdewolffAST(tdewolffAST)
	if err != nil {
		return "", fmt.Errorf("printing converted AST: %w", err)
	}
	return printed, nil
}

// printTdewolffAST converts a tdewolff AST back to JavaScript source code.
//
// Takes tree (*parsejs.AST) which is the parsed JavaScript syntax tree.
//
// Returns string which contains the JavaScript source code, or an empty string when tree
// is nil.
// Returns error when the tree cannot be normalised for printing.
func printTdewolffAST(tree *parsejs.AST) (string, error) {
	if tree == nil {
		return "", nil
	}

	if err := normaliseAST(tree); err != nil {
		return "", fmt.Errorf("normalising AST for printing: %w", err)
	}

	var builder strings.Builder
	for i, statement := range tree.List {
		if i > 0 {
			builder.WriteString("\n")
		}
		statement.JS(&builder)
	}

	return builder.String(), nil
}

// insertRenderMethod inserts the renderVDOM method into the target class.
//
// Takes jsAST (*js_ast.AST) which is the JavaScript AST to modify.
// Takes className (string) which names the target class.
// Takes renderMethod (*js_ast.EFunction) which is the method to insert.
// Takes registry (*RegistryContext) which provides the registry context.
//
// Returns error when the method is nil or the target class cannot be found.
func insertRenderMethod(ctx context.Context, jsAST *js_ast.AST, className string, renderMethod *js_ast.EFunction, registry *RegistryContext) error {
	MethodInsertionCount.Add(ctx, 1)
	insertStartTime := time.Now()

	insertErr := insertMethodIntoClass(ctx, jsAST, className, renderMethod, registry)

	MethodInsertionDuration.Record(ctx, float64(time.Since(insertStartTime).Milliseconds()))

	if insertErr != nil {
		MethodInsertionErrorCount.Add(ctx, 1)
		return insertErr
	}
	return nil
}

// injectEventBindings adds event bindings to a class constructor.
//
// Takes jsAST (*js_ast.AST) which provides the JavaScript AST to change.
// Takes className (string) which names the target class.
// Takes events (*eventBindingCollection) which holds the bindings to add.
//
// Returns error when there are bindings to add but the target class or its constructor
// cannot be found.
func injectEventBindings(ctx context.Context, jsAST *js_ast.AST, className string, events *eventBindingCollection) error {
	if len(events.getBindings()) == 0 {
		return nil
	}

	targetClass := findClassDeclarationByName(jsAST, className)
	if targetClass == nil {
		return fmt.Errorf("target class %q not found for event bindings", className)
	}

	if err := injectEventBindingsIntoConstructor(ctx, targetClass, events); err != nil {
		return fmt.Errorf("injecting event bindings into %s: %w", className, err)
	}
	return nil
}

// elideTypeOnlyNamedImports drops named import bindings never referenced as a value in
// the compiled component body.
//
// Takes tree (*js_ast.AST) which supplies the symbol table for name resolution.
// Takes bodyStmts ([]js_ast.Stmt) which are the non-import statements (script +
// template).
// Takes imports ([]js_ast.Stmt) which are the reconstructed import statements to filter.
// Takes registry (*RegistryContext) which resolves registry-backed identifier names.
//
// Returns []js_ast.Stmt which holds the imports with unused type-only bindings removed.
func elideTypeOnlyNamedImports(tree *js_ast.AST, bodyStmts, imports []js_ast.Stmt, registry *RegistryContext) []js_ast.Stmt {
	used, ok := collectUsedIdentifiers(tree, bodyStmts, registry)
	if !ok {
		return imports
	}

	filtered := make([]js_ast.Stmt, 0, len(imports))
	for _, stmt := range imports {
		if keepImportStatement(stmt, used) {
			filtered = append(filtered, stmt)
		}
	}
	return filtered
}

// keepImportStatement removes named bindings not present in the used set and reports
// whether the statement still binds something and should be kept.
//
// Side-effect, default-only and namespace imports are always kept.
//
// Takes stmt (js_ast.Stmt) which is the import statement to filter in place.
// Takes used (map[string]struct{}) which is the set of value-referenced identifier names.
//
// Returns bool which is true when the statement still binds a name and should be kept.
func keepImportStatement(stmt js_ast.Stmt, used map[string]struct{}) bool {
	simport, ok := stmt.Data.(*js_ast.SImport)
	if !ok || simport.Items == nil {
		return true
	}

	kept := make([]js_ast.ClauseItem, 0, len(*simport.Items))
	for _, item := range *simport.Items {
		if clauseItemIsUsed(item, used) {
			kept = append(kept, item)
		}
	}

	if len(kept) > 0 {
		*simport.Items = kept
		return true
	}
	simport.Items = nil
	return simport.DefaultName != nil || simport.StarNameLoc != nil
}

// clauseItemIsUsed reports whether a named import binding's local name is referenced as a
// value, conservatively keeping a binding whose local name cannot be determined.
//
// Takes item (js_ast.ClauseItem) which is the named import binding to test.
// Takes used (map[string]struct{}) which is the set of value-referenced identifier names.
//
// Returns bool which is true when the binding's local name is referenced as a value.
func clauseItemIsUsed(item js_ast.ClauseItem, used map[string]struct{}) bool {
	name := item.OriginalName
	if name == "" {
		name = item.Alias
	}
	if name == "" {
		return true
	}
	_, ok := used[name]
	return ok
}

// collectUsedIdentifiers records the name of every identifier referenced as a value in
// the converted body.
//
// Walking the AST rather than scanning printed text means a name that appears only as a
// property key or in a string literal is not mistaken for a value use. The final-print
// registry is reused and conversion only reads it, so this throwaway pass cannot change
// the final output.
//
// Takes tree (*js_ast.AST) which supplies the symbol table for name resolution.
// Takes bodyStmts ([]js_ast.Stmt) which are the non-import statements to scan.
// Takes registry (*RegistryContext) which resolves registry-backed identifier names.
//
// Returns map[string]struct{} which is the set of value-referenced identifier names.
// Returns bool which is false when the body cannot be analysed, so the caller keeps all.
func collectUsedIdentifiers(tree *js_ast.AST, bodyStmts []js_ast.Stmt, registry *RegistryContext) (map[string]struct{}, bool) {
	if tree == nil || len(bodyStmts) == 0 {
		return nil, false
	}
	scanTree := &js_ast.AST{
		Symbols:       tree.Symbols,
		ImportRecords: tree.ImportRecords,
		Parts:         []js_ast.Part{{Stmts: bodyStmts}},
	}
	tdewolffAST, err := ConvertEsbuildToTdewolff(scanTree, registry)
	if err != nil {
		return nil, false
	}
	collector := &usedIdentifierCollector{names: make(map[string]struct{})}
	parsejs.Walk(collector, tdewolffAST)
	return collector.names, true
}
