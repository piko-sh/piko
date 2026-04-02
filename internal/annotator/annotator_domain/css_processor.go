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

package annotator_domain

// Processes and scopes CSS for components by parsing, inlining @import
// statements, and applying component-specific attributes. Transforms CSS
// selectors to ensure styles are isolated to their component, preventing
// cross-component style leakage.

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	es_ast "piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/esbuild/css_ast"
	"piko.sh/piko/internal/esbuild/css_lexer"
	"piko.sh/piko/internal/esbuild/css_parser"
	"piko.sh/piko/internal/esbuild/css_printer"
	es_logger "piko.sh/piko/internal/esbuild/logger"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

// CSSProcessor handles CSS scoping and processing using esbuild.
type CSSProcessor struct {
	// resolver provides path resolution for @import statements in CSS files.
	resolver resolver_domain.ResolverPort

	// options holds options for CSS output formatting.
	options *config.Options

	// parseOpts holds the CSS parser settings.
	parseOpts css_parser.Options
}

// NewCSSProcessor creates a new CSS processor with the given settings.
//
// Takes loader (config.Loader) which loads configuration values.
// Takes options (*config.Options) which sets the processing options.
// Takes resolver (resolver_domain.ResolverPort) which resolves import paths.
//
// Returns *CSSProcessor which is ready to use.
func NewCSSProcessor(
	loader config.Loader,
	options *config.Options,
	resolver resolver_domain.ResolverPort,
) *CSSProcessor {
	if options == nil {
		options = &config.Options{}
	}
	return &CSSProcessor{
		parseOpts: css_parser.OptionsFromConfig(loader, options),
		options:   options,
		resolver:  resolver,
	}
}

// SetResolver updates the resolver used for @import resolution.
// This is used by the LSP to provide per-module resolvers.
//
// Takes resolver (resolver_domain.ResolverPort) which provides path resolution.
func (cp *CSSProcessor) SetResolver(resolver resolver_domain.ResolverPort) {
	cp.resolver = resolver
}

// WithResolver returns a shallow copy of the CSSProcessor that uses the given
// resolver. This is safe for concurrent use because the copy does not share
// mutable state with the original.
//
// Takes resolver (resolver_domain.ResolverPort) which provides path resolution.
//
// Returns *CSSProcessor which is a new processor with the given resolver.
func (cp *CSSProcessor) WithResolver(resolver resolver_domain.ResolverPort) *CSSProcessor {
	return &CSSProcessor{
		resolver:  resolver,
		options:   cp.options,
		parseOpts: cp.parseOpts,
	}
}

// Process takes a raw CSS string, resolves all @import rules, processes it
// (e.g., minifies), and returns the final bundled result. This is used for
// global, unscoped style blocks.
//
// Takes cssBlock (string) which contains the raw CSS to process.
// Takes sourcePath (string) which identifies the source file for diagnostics.
// Takes startLocation (ast_domain.Location) which marks the CSS block origin.
// Takes fsReader (FSReaderPort) which reads imported files from the filesystem.
//
// Returns string which is the processed and bundled CSS output.
// Returns []*ast_domain.Diagnostic which contains any warnings or errors found.
// Returns error when processing fails.
func (cp *CSSProcessor) Process(
	ctx context.Context,
	cssBlock string,
	sourcePath string,
	startLocation ast_domain.Location,
	fsReader FSReaderPort,
) (string, []*ast_domain.Diagnostic, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "CSSProcessor.Process")
	defer span.End()

	CSSProcessCount.Add(ctx, 1)
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		CSSProcessDuration.Record(ctx, float64(duration.Milliseconds()))
	}()

	trimmed := strings.TrimSpace(cssBlock)
	if trimmed == "" {
		return "", nil, nil
	}

	inliner := getCSSInliner(cp, fsReader)
	defer putCSSInliner(inliner)
	tree, inlinerDiags := inliner.InlineAndParse(ctx, trimmed, sourcePath, startLocation)

	if tree == nil {
		CSSProcessErrorCount.Add(ctx, 1)
		l.Error("Failed to process CSS with fatal error during import resolution")
		return "", inlinerDiags, nil
	}

	tree.Rules = cleanCSSTree(tree.Rules)

	symMap := es_ast.NewSymbolMap(1)
	symMap.SymbolsForSource[0] = tree.Symbols
	printOpts := css_printer.Options{
		MinifyWhitespace: cp.options.MinifyWhitespace,
		ASCIIOnly:        cp.options.ASCIIOnly,
	}
	printed := css_printer.Print(*tree, symMap, printOpts)
	result := string(printed.CSS)

	l.Trace("CSS processed and bundled successfully",
		logger_domain.Int("inputSize", len(trimmed)),
		logger_domain.Int("outputSize", len(result)),
	)
	return result, inlinerDiags, nil
}

// processAndScopeParams holds the parameters for CSS scoping operations.
type processAndScopeParams struct {
	// fsReader provides file system access for reading CSS imports.
	fsReader FSReaderPort

	// template is the parsed template AST used for CSS processing.
	template *ast_domain.TemplateAST

	// cssBlock is the CSS content to process; nil means no custom styles.
	cssBlock *string

	// scopeID is the unique identifier added to CSS selectors for scoping.
	scopeID string

	// sourcePath is the file path of the CSS source being processed.
	sourcePath string

	// startLocation is the position where parsing starts in the source file.
	startLocation ast_domain.Location
}

// ProcessAndScope processes and bundles a CSS block, then applies scoped CSS
// rules. This is used for component-specific, non-global style blocks.
//
// Takes ctx (context.Context) which carries cancellation, tracing, and logging.
// Takes params (*processAndScopeParams) which contains the CSS block, scope ID,
// template, and other processing options.
//
// Returns []*ast_domain.Diagnostic which contains any warnings or issues found
// during CSS processing.
// Returns error when the input validation fails.
func (cp *CSSProcessor) ProcessAndScope(ctx context.Context, params *processAndScopeParams) ([]*ast_domain.Diagnostic, error) {
	var l logger_domain.Logger
	ctx, l = logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "CSSProcessor.ProcessAndScope", logger_domain.String("scopeID", params.scopeID))
	defer span.End()

	CSSScopeCount.Add(ctx, 1)
	startTime := time.Now()
	defer func() {
		CSSScopeDuration.Record(ctx, float64(time.Since(startTime).Milliseconds()))
	}()

	if err := cp.validateCSSProcessingInputs(ctx, params.template, params.cssBlock, params.scopeID); err != nil {
		return nil, fmt.Errorf("validating CSS processing inputs: %w", err)
	}

	trimmed := strings.TrimSpace(*params.cssBlock)
	if trimmed == "" {
		l.Trace("Empty CSS block, nothing to scope")
		*params.cssBlock = ""
		return nil, nil
	}

	preprocessed, markers := preprocessScopingPseudoClasses(trimmed)

	inliner := getCSSInliner(cp, params.fsReader)
	defer putCSSInliner(inliner)
	tree, inlinerDiags := inliner.InlineAndParse(ctx, preprocessed, params.sourcePath, params.startLocation)

	if tree == nil {
		CSSScopeErrorCount.Add(ctx, 1)
		l.Error("Failed to scope CSS with fatal error during import resolution")
		*params.cssBlock = ""
		return inlinerDiags, nil
	}

	*params.cssBlock = string(cp.transformAndPrintCSS(tree, params.scopeID, params.template, markers))

	l.Trace("CSS scoped and bundled successfully",
		logger_domain.Int("inputSize", len(trimmed)),
		logger_domain.Int("outputSize", len(*params.cssBlock)),
	)
	return inlinerDiags, nil
}

// validateCSSProcessingInputs checks that all required inputs are present.
//
// Takes template (*ast_domain.TemplateAST) which is the template to process.
// Takes cssBlock (*string) which points to the CSS content to scope.
// Takes scopeID (string) which is the scope identifier to apply.
//
// Returns error when template is nil, cssBlock is nil, or scopeID is empty.
func (*CSSProcessor) validateCSSProcessingInputs(ctx context.Context, template *ast_domain.TemplateAST, cssBlock *string, scopeID string) error {
	if template == nil || cssBlock == nil || scopeID == "" {
		err := errors.New("ProcessAndScope: templateAST, cssBlock and scopeID must be non-empty")
		CSSScopeErrorCount.Add(ctx, 1)
		_, l := logger_domain.From(ctx, log)
		l.Error("Invalid parameters", logger_domain.Error(err))
		return fmt.Errorf("validating CSS processing inputs: %w", err)
	}
	return nil
}

// transformAndPrintCSS applies scoping changes and prints the final CSS.
//
// Takes tree (*css_ast.AST) which is the parsed CSS syntax tree to modify.
// Takes scopeID (string) which is the unique identifier for scoping selectors.
// Takes template (*ast_domain.TemplateAST) which provides template context for
// scoping.
// Takes markers (*scopingMarkers) which tracks elements that need scope
// attributes.
//
// Returns []byte which is the printed CSS output.
func (cp *CSSProcessor) transformAndPrintCSS(tree *css_ast.AST, scopeID string, template *ast_domain.TemplateAST, markers *scopingMarkers) []byte {
	transformer := newCSSScopeTransformer(scopeID, template, tree.Symbols)
	transformer.markers = markers
	transformer.transform(tree.Rules)
	tree.Rules = cleanCSSTree(tree.Rules)

	symMap := es_ast.NewSymbolMap(1)
	symMap.SymbolsForSource[0] = tree.Symbols
	printOpts := css_printer.Options{
		MinifyWhitespace: cp.options.MinifyWhitespace,
		ASCIIOnly:        cp.options.ASCIIOnly,
	}

	return css_printer.Print(*tree, symMap, printOpts).CSS
}

// scopingMarkers holds information about :global() and :deep() pseudo-classes
// that were detected during pre-processing.
type scopingMarkers struct {
	// hasGlobal indicates whether any :global() markers were found.
	hasGlobal bool

	// hasDeep indicates whether a :deep pseudo-class was found.
	hasDeep bool
}

var (
	// globalPseudoRegex matches the :global() pseudo-class in CSS selectors.
	globalPseudoRegex = regexp.MustCompile(`:global\s*\(([^)]+)\)`)

	// deepPseudoRegex matches the :deep() pseudo-class in CSS selectors.
	deepPseudoRegex = regexp.MustCompile(`:deep\s*\(([^)]+)\)`)
)

// cleanCSSTree walks the CSS AST and removes any rules where Data is nil.
// This is needed because the esbuild minifier can leave empty rules in nested
// blocks (such as @media) that the printer cannot handle.
//
// Takes rules ([]css_ast.Rule) which is the slice of CSS rules to clean.
//
// Returns []css_ast.Rule which is the filtered slice with nil rules removed.
func cleanCSSTree(rules []css_ast.Rule) []css_ast.Rule {
	if rules == nil {
		return nil
	}

	n := 0
	for i := range len(rules) {
		rule := &rules[i]
		if rule.Data == nil {
			continue
		}

		switch r := rule.Data.(type) {
		case *css_ast.RAtKeyframes:
			for j := range r.Blocks {
				r.Blocks[j].Rules = cleanCSSTree(r.Blocks[j].Rules)
			}
		case *css_ast.RKnownAt:
			r.Rules = cleanCSSTree(r.Rules)
		case *css_ast.RAtMedia:
			r.Rules = cleanCSSTree(r.Rules)
		case *css_ast.RAtLayer:
			r.Rules = cleanCSSTree(r.Rules)
		case *css_ast.RSelector:
			r.Rules = cleanCSSTree(r.Rules)
		}

		rules[n] = *rule
		n++
	}
	return rules[:n]
}

// convertESBuildMessagesToDiagnostics converts esbuild log messages into
// diagnostic objects for the domain layer.
//
// When messages is empty, returns nil.
//
// Takes messages ([]es_logger.Msg) which contains the esbuild log messages to
// convert.
// Takes sourcePath (string) which identifies the source file for diagnostics.
// Takes startLocation (ast_domain.Location) which provides the offset for
// line and column values.
//
// Returns []*ast_domain.Diagnostic which contains the converted diagnostics.
func convertESBuildMessagesToDiagnostics(messages []es_logger.Msg, sourcePath string, startLocation ast_domain.Location) []*ast_domain.Diagnostic {
	if len(messages) == 0 {
		return nil
	}

	diagnostics := make([]*ast_domain.Diagnostic, 0, len(messages))
	for _, message := range messages {
		var severity ast_domain.Severity
		switch message.Kind {
		case es_logger.Error:
			severity = ast_domain.Error
		case es_logger.Warning:
			severity = ast_domain.Warning
		case es_logger.Info:
			severity = ast_domain.Info
		default:
			continue
		}

		var finalLocation ast_domain.Location
		var expression string
		if message.Data.Location != nil {
			messageLocation := message.Data.Location
			line := messageLocation.Line + 1
			column := messageLocation.Column + 1

			if line == 1 {
				finalLocation.Line = startLocation.Line
				finalLocation.Column = startLocation.Column + column - 1
			} else {
				finalLocation.Line = startLocation.Line + line - 1
				finalLocation.Column = column
			}
			expression = messageLocation.LineText
		}

		diagnostics = append(diagnostics, &ast_domain.Diagnostic{
			Data:         nil,
			Message:      message.Data.Text,
			Expression:   expression,
			SourcePath:   sourcePath,
			Code:         annotator_dto.CodeCSSProcessingError,
			RelatedInfo:  nil,
			Location:     finalLocation,
			SourceLength: 0,
			Severity:     severity,
		})
	}
	return diagnostics
}

// scopeDescendant adds a scope attribute selector to the start of a complex
// selector.
//
// Takes selector (css_ast.ComplexSelector) which is the selector to change.
// Takes scopeID (string) which is the scope attribute value to add.
// Takes location (es_logger.Loc) which is the source location for the
// new selector.
//
// Returns css_ast.ComplexSelector which is a new selector with the scope
// attribute at the start.
func scopeDescendant(selector css_ast.ComplexSelector, scopeID string, location es_logger.Loc) css_ast.ComplexSelector {
	scopeComp := makeScopeAttributeCompound(scopeID, location)
	newComps := append([]css_ast.CompoundSelector{scopeComp}, selector.Selectors...)
	return css_ast.ComplexSelector{Selectors: newComps}
}

// scopeDirect adds a scope attribute to each compound selector in a complex
// selector chain.
//
// This maintains proper CSS isolation: each element in a descendant chain must
// have the scope attribute to match. Elements like html, body, or those with
// :root are skipped since they exist outside partial boundaries.
//
// The scope attribute is placed before any pseudo-class selectors to keep the
// correct specificity order.
//
// Takes selector (css_ast.ComplexSelector) which is the selector to scope.
// Takes scopeID (string) which is the unique identifier for the scope.
//
// Returns css_ast.ComplexSelector which is a cloned selector with the scope
// attribute added to each suitable compound selector.
func scopeDirect(selector css_ast.ComplexSelector, scopeID string) css_ast.ComplexSelector {
	if len(selector.Selectors) == 0 {
		return selector
	}
	out := selector.Clone()

	for i := range out.Selectors {
		comp := &out.Selectors[i]

		if shouldSkipCompoundScoping(comp) {
			continue
		}

		addScopeToCompound(comp, scopeID)
	}

	return out
}

// shouldSkipCompoundScoping checks if a compound selector should not be scoped.
//
// Elements like html, body, or those with :root pseudo-class exist outside
// partial boundaries and should not have the scope attribute added.
//
// Takes comp (*css_ast.CompoundSelector) which is the selector to check.
//
// Returns bool which is true if the selector should not be scoped.
func shouldSkipCompoundScoping(comp *css_ast.CompoundSelector) bool {
	for _, sub := range comp.SubclassSelectors {
		if pseudo, ok := sub.Data.(*css_ast.SSPseudoClass); ok {
			if strings.EqualFold(pseudo.Name, "root") {
				return true
			}
		}
	}

	if comp.TypeSelector != nil {
		tag := strings.ToLower(comp.TypeSelector.Name.Text)
		if tag == "html" || tag == "body" {
			return true
		}
	}

	return false
}

// addScopeToCompound adds a scope attribute to a compound selector.
//
// The scope attribute is placed before any pseudo-class selectors to keep the
// correct specificity order.
//
// Takes comp (*css_ast.CompoundSelector) which is the selector to modify.
// Takes scopeID (string) which is the scope identifier to add.
func addScopeToCompound(comp *css_ast.CompoundSelector, scopeID string) {
	insertPosition := findPseudoClassInsertPosition(comp.SubclassSelectors)

	scopeAttr := makeScopeAttributeSelector(scopeID, comp.Range().Loc)
	newSubclass := make([]css_ast.SubclassSelector, 0, len(comp.SubclassSelectors)+1)
	newSubclass = append(newSubclass, comp.SubclassSelectors[:insertPosition]...)
	newSubclass = append(newSubclass, scopeAttr)
	newSubclass = append(newSubclass, comp.SubclassSelectors[insertPosition:]...)
	comp.SubclassSelectors = newSubclass
}

// findPseudoClassInsertPosition finds the index where a scope attribute should
// be inserted in a list of subclass selectors.
//
// The scope attribute must be placed before any pseudo-class selectors to keep
// the correct CSS specificity order.
//
// Takes selectors ([]css_ast.SubclassSelector) which is the list to search.
//
// Returns int which is the index where the scope attribute should be inserted.
func findPseudoClassInsertPosition(selectors []css_ast.SubclassSelector) int {
	for i, sub := range selectors {
		switch sub.Data.(type) {
		case *css_ast.SSPseudoClass, *css_ast.SSPseudoClassWithSelectorList:
			return i
		}
	}
	return len(selectors)
}

// makeScopeAttributeCompound creates a compound selector that holds a single
// scope attribute selector.
//
// Takes scopeID (string) which sets the scope for the attribute.
// Takes location (es_logger.Loc) which sets the source location.
//
// Returns css_ast.CompoundSelector which wraps the scope attribute selector.
func makeScopeAttributeCompound(scopeID string, location es_logger.Loc) css_ast.CompoundSelector {
	return css_ast.CompoundSelector{
		SubclassSelectors: []css_ast.SubclassSelector{makeScopeAttributeSelector(scopeID, location)},
	}
}

// makeScopeAttributeSelector creates a CSS attribute selector for partial
// scope matching.
//
// Takes scopeID (string) which is the scope value to match.
// Takes location (es_logger.Loc) which specifies the source location for the
// selector.
//
// Returns css_ast.SubclassSelector which matches elements where the partial
// attribute contains the scope ID.
func makeScopeAttributeSelector(scopeID string, location es_logger.Loc) css_ast.SubclassSelector {
	r := es_logger.Range{Loc: location, Len: 0}
	return css_ast.SubclassSelector{
		Range: r,
		Data: &css_ast.SSAttribute{
			NamespacedName: css_ast.NamespacedName{
				Name: css_ast.NameToken{Text: "partial", Kind: css_lexer.TIdent, Range: r},
			},
			MatcherOp:    "~=",
			MatcherValue: scopeID,
		},
	}
}

// preprocessScopingPseudoClasses pre-processes CSS to replace :global() and
// :deep() with marker classes that will survive esbuild's parsing.
//
// Takes css (string) which is the raw CSS input to pre-process.
//
// Returns string which is the pre-processed CSS output.
// Returns *scopingMarkers which tracks which
// pseudo-classes were found.
func preprocessScopingPseudoClasses(css string) (string, *scopingMarkers) {
	markers := &scopingMarkers{hasGlobal: false, hasDeep: false}
	result := css

	if globalPseudoRegex.MatchString(result) {
		markers.hasGlobal = true
		result = globalPseudoRegex.ReplaceAllString(result, `$1.__piko_global__`)
	}

	if deepPseudoRegex.MatchString(result) {
		markers.hasDeep = true
		result = deepPseudoRegex.ReplaceAllString(result, `.__piko_deep__ $1`)
	}

	return result, markers
}
