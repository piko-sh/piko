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

// Expands partial component invocations by inlining their templates into the
// parent component's AST. Processes styles, handles slots, manages CSS scoping,
// and produces a flattened AST ready for type checking and linking.

import (
	"context"
	"fmt"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/sfcparser"
)

const (
	// attributeIs is the attribute name that marks partial component tags.
	attributeIs = "is"

	// attributeClass is the HTML class attribute name.
	attributeClass = "class"

	// attributeName is the attribute key used to retrieve the name of a slot.
	attributeName = "name"

	// tagPikoSlot is the tag name for server slot elements in templates.
	tagPikoSlot = "piko:slot"

	// tagPikoPartial is the tag name for partial template elements.
	tagPikoPartial = "piko:partial"

	// tagPikoElement is the tag name for the dynamic element tag.
	tagPikoElement = "piko:element"

	// attributePFragment is the HTML attribute name used to mark partial fragments.
	attributePFragment = "p-fragment"

	// attributePFragmentID is the HTML attribute name used to mark fragment elements.
	attributePFragmentID = "p-fragment-id"

	// prefixRequest is the prefix for attributes that apply to a single request.
	prefixRequest = "request."

	// prefixServer is the prefix that marks server-side properties.
	prefixServer = "server."

	// attributePartial is the attribute name that marks values to be grouped together.
	attributePartial = "partial"

	// attributePartialName is the attribute key for partial name matching.
	attributePartialName = "partial_name"

	// attributePartialSrc is the attribute name for partial source content.
	attributePartialSrc = "partial_src"

	// maxAnnotatorDepth caps the recursion depth of annotator stamping and
	// transform passes so a maliciously nested template tree cannot overflow
	// the Go stack. Real templates rarely nest beyond a few dozen levels, so
	// 256 is generous.
	maxAnnotatorDepth = 256
)

// PartialExpander is a pure, in-memory, recursive AST transformation engine.
// It transforms a graph of separate component ASTs into a single, flattened
// AST, stamping every node with its structural context and collecting any
// structural or validation diagnostics it finds.
type PartialExpander struct {
	// resolver handles path resolution for partial template imports.
	resolver resolver_domain.ResolverPort

	// cssProcessor handles CSS minification and scoping for style blocks.
	cssProcessor *CSSProcessor

	// fsReader provides file system access for CSS processing.
	fsReader FSReaderPort
}

// styleProcessingContext groups parameters for style block processing to reduce
// argument count.
type styleProcessingContext struct {
	// expCtx holds the context for expanding template variables.
	expCtx *expansionContext

	// mainComponent is the parsed component being processed.
	mainComponent *annotator_dto.ParsedComponent

	// scopedCSSBuilder collects scoped CSS content during style processing.
	scopedCSSBuilder *strings.Builder

	// hasScopedStyles points to a flag that shows whether scoped styles exist.
	hasScopedStyles *bool

	// entryPointHashedName is the hashed name of the entry point for CSS
	// processing.
	entryPointHashedName string

	// isPage indicates whether the current node is a page-level element.
	isPage bool

	// isEmail indicates whether the output is for email.
	isEmail bool
}

// NewPartialExpander creates a new, configured PartialExpander.
//
// Takes resolver (ResolverPort) which resolves partial references.
// Takes cssProcessor (*CSSProcessor) which handles CSS processing.
// Takes fsReader (FSReaderPort) which provides file system read access.
//
// Returns *PartialExpander which is ready for use.
func NewPartialExpander(resolver resolver_domain.ResolverPort, cssProcessor *CSSProcessor, fsReader FSReaderPort) *PartialExpander {
	return &PartialExpander{
		resolver:     resolver,
		cssProcessor: cssProcessor,
		fsReader:     fsReader,
	}
}

// Expand runs the expansion stage starting from the given entry point.
//
// Takes graph (*annotator_dto.ComponentGraph) which holds all parsed
// components.
// Takes entryPointHashedName (string) which is the name of the root component
// to expand.
// Takes isPage (bool) which indicates whether the entry point is a page.
// Takes isEmail (bool) which indicates whether the entry point is an email.
//
// Returns *annotator_dto.ExpansionResult which holds the final flattened AST.
// Returns []*ast_domain.Diagnostic which holds any issues found during
// expansion.
// Returns error when a fatal system error occurs that cannot be recovered.
func (exp *PartialExpander) Expand(
	ctx context.Context,
	graph *annotator_dto.ComponentGraph,
	entryPointHashedName string,
	isPage bool,
	isEmail bool,
) (*annotator_dto.ExpansionResult, []*ast_domain.Diagnostic, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "PartialExpander.Expand", logger_domain.String("entryPoint", entryPointHashedName))
	defer span.End()

	l.Internal("--- [EXPANDER START] --- Starting Partial Expansion Stage ---")

	expCtx, mainComponent, astToExpand, err := newExpansionContext(exp, graph, entryPointHashedName)
	if err != nil {
		l.Error("Failed to set up expansion context", logger_domain.Error(err))
		return nil, nil, fmt.Errorf("setting up expansion context for %q: %w", entryPointHashedName, err)
	}
	if astToExpand == nil {
		return createEmptyExpansionResult(), nil, nil
	}

	if err := exp.processEntryPointStyles(ctx, expCtx, mainComponent, entryPointHashedName, isPage, isEmail); err != nil {
		return nil, expCtx.diagnostics, err
	}

	finalRootNodes := expCtx.expandPartialsRecursive(ctx, astToExpand.RootNodes, mainComponent)

	if err := checkForCriticalErrors(ctx, expCtx); err != nil {
		return nil, expCtx.diagnostics, err
	}
	if ast_domain.HasErrors(expCtx.diagnostics) {
		return nil, expCtx.diagnostics, nil
	}

	return exp.finaliseExpansion(ctx, expCtx, finalRootNodes, mainComponent)
}

// processEntryPointStyles processes style blocks for the entry point
// component.
//
// Takes ctx (context.Context) which carries the logger.
// Takes expCtx (*expansionContext) which holds the expansion state.
// Takes mainComponent (*annotator_dto.ParsedComponent) which is the entry
// point component to process styles for.
// Takes entryPointHashedName (string) which identifies the entry point.
// Takes isPage (bool) which indicates if this is a page entry point.
// Takes isEmail (bool) which indicates if this is an email entry point.
//
// Returns error when style block processing fails.
func (exp *PartialExpander) processEntryPointStyles(
	ctx context.Context,
	expCtx *expansionContext,
	mainComponent *annotator_dto.ParsedComponent,
	entryPointHashedName string,
	isPage bool,
	isEmail bool,
) error {
	if len(mainComponent.StyleBlocks) == 0 {
		return nil
	}

	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Processing styles for the entry-point component", logger_domain.String("path", mainComponent.SourcePath))

	var scopedCSSBuilder strings.Builder
	var hasScopedStyles bool

	styleCtx := &styleProcessingContext{
		expCtx:               expCtx,
		mainComponent:        mainComponent,
		entryPointHashedName: entryPointHashedName,
		isPage:               isPage,
		isEmail:              isEmail,
		scopedCSSBuilder:     &scopedCSSBuilder,
		hasScopedStyles:      &hasScopedStyles,
	}

	for _, styleBlock := range mainComponent.StyleBlocks {
		styleContent := styleBlock.Content
		if strings.TrimSpace(styleContent) == "" {
			continue
		}

		err := exp.processStyleBlock(ctx, styleCtx, styleBlock, &styleContent)
		if err != nil {
			return fmt.Errorf("processing style block for %q: %w", mainComponent.SourcePath, err)
		}
	}

	if hasScopedStyles {
		expCtx.scopedCSSBlocks[entryPointHashedName] = scopedCSSBuilder.String()
	}

	return nil
}

// processStyleBlock handles a single style block from a Piko SFC.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes styleCtx (*styleProcessingContext) which holds the current processing
// state.
// Takes styleBlock (sfcparser.Style) which is the style block to process.
// Takes styleContent (*string) which holds the style text content.
//
// Returns error when style processing fails.
func (exp *PartialExpander) processStyleBlock(
	ctx context.Context,
	styleCtx *styleProcessingContext,
	styleBlock sfcparser.Style,
	styleContent *string,
) error {
	_, isGlobal := styleBlock.Attributes["global"]
	styleStartLocation := ast_domain.Location{Line: styleBlock.ContentLocation.Line, Column: styleBlock.ContentLocation.Column, Offset: 0}

	if isGlobal {
		return exp.processGlobalStyleBlock(ctx, styleCtx.expCtx, styleCtx.mainComponent, *styleContent, styleStartLocation)
	}

	return exp.processScopedStyleBlock(ctx, styleCtx, styleContent, styleCtx.entryPointHashedName, styleStartLocation)
}

// processGlobalStyleBlock processes a global style block.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes expCtx (*expansionContext) which holds the expansion state.
// Takes mainComponent (*annotator_dto.ParsedComponent) which provides the
// source component being processed.
// Takes styleContent (string) which contains the raw CSS to process.
// Takes styleStartLocation (ast_domain.Location) which specifies where the
// style block begins in the source.
//
// Returns error when CSS processing fails.
func (exp *PartialExpander) processGlobalStyleBlock(
	ctx context.Context,
	expCtx *expansionContext,
	mainComponent *annotator_dto.ParsedComponent,
	styleContent string,
	styleStartLocation ast_domain.Location,
) error {
	processedGlobalCSS, diagnostics, err := exp.cssProcessor.Process(
		ctx, styleContent, mainComponent.SourcePath, styleStartLocation, exp.fsReader,
	)
	if err != nil {
		return fmt.Errorf("fatal error processing entry point global CSS: %w", err)
	}
	expCtx.diagnostics = append(expCtx.diagnostics, diagnostics...)
	expCtx.globalCSSBlocks[mainComponent.SourcePath] = processedGlobalCSS
	return nil
}

// processScopedStyleBlock processes a scoped style block.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes styleCtx (*styleProcessingContext) which holds the processing state.
// Takes styleContent (*string) which contains the CSS to process and scope.
// Takes scopeID (string) which identifies the scope for CSS isolation.
// Takes styleStartLocation (ast_domain.Location) which marks the source
// position.
//
// Returns error when CSS processing fails.
func (exp *PartialExpander) processScopedStyleBlock(
	ctx context.Context,
	styleCtx *styleProcessingContext,
	styleContent *string,
	scopeID string,
	styleStartLocation ast_domain.Location,
) error {
	diagnostics, err := exp.cssProcessor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      styleCtx.mainComponent.Template,
		cssBlock:      styleContent,
		scopeID:       scopeID,
		sourcePath:    styleCtx.mainComponent.SourcePath,
		startLocation: styleStartLocation,
		fsReader:      exp.fsReader,
	})
	if err != nil {
		return fmt.Errorf("fatal error processing entry point scoped CSS: %w", err)
	}
	styleCtx.expCtx.diagnostics = append(styleCtx.expCtx.diagnostics, diagnostics...)
	styleCtx.scopedCSSBuilder.WriteString(*styleContent)
	styleCtx.scopedCSSBuilder.WriteString("\n")
	*styleCtx.hasScopedStyles = true
	return nil
}

// finaliseExpansion builds the final expansion result.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes expCtx (*expansionContext) which holds the expansion state.
// Takes finalRootNodes ([]*ast_domain.TemplateNode) which are the processed
// template nodes.
// Takes mainComponent (*annotator_dto.ParsedComponent) which is the main
// component being expanded.
//
// Returns *annotator_dto.ExpansionResult which contains the flat AST, combined
// CSS, and possible invocations.
// Returns []*ast_domain.Diagnostic which lists all diagnostics found during
// expansion.
// Returns error when CSS assembly fails.
func (*PartialExpander) finaliseExpansion(
	ctx context.Context,
	expCtx *expansionContext,
	finalRootNodes []*ast_domain.TemplateNode,
	mainComponent *annotator_dto.ParsedComponent,
) (*annotator_dto.ExpansionResult, []*ast_domain.Diagnostic, error) {
	ctx, l := logger_domain.From(ctx, log)
	finalAST := finaliseAST(ctx, finalRootNodes, mainComponent)
	allDiagnostics := make([]*ast_domain.Diagnostic, len(expCtx.diagnostics)+len(finalAST.Diagnostics))
	copy(allDiagnostics, expCtx.diagnostics)
	copy(allDiagnostics[len(expCtx.diagnostics):], finalAST.Diagnostics)

	finalCSS, err := expCtx.assembleFinalCSS()
	if err != nil {
		return nil, allDiagnostics, err
	}

	result := &annotator_dto.ExpansionResult{
		FlattenedAST:         finalAST,
		CombinedCSS:          finalCSS,
		PotentialInvocations: expCtx.getSortedPartialInvocations(),
	}

	l.Internal("--- [EXPANDER END] Partial Expansion Stage Completed ---",
		logger_domain.Int("diagnostics_found", len(allDiagnostics)),
		logger_domain.Int("unique_invocations_found", len(result.PotentialInvocations)),
	)
	return result, allDiagnostics, nil
}

// createEmptyExpansionResult creates an expansion result for components that
// have no template.
//
// Returns *annotator_dto.ExpansionResult which contains empty or nil values
// for all fields.
func createEmptyExpansionResult() *annotator_dto.ExpansionResult {
	return &annotator_dto.ExpansionResult{
		FlattenedAST: &ast_domain.TemplateAST{
			SourcePath:        nil,
			ExpiresAtUnixNano: nil,
			Metadata:          nil,
			RootNodes:         nil,
			Diagnostics:       nil,
			SourceSize:        0,
			Tidied:            false,
		},
		CombinedCSS:          "",
		PotentialInvocations: nil,
	}
}

// checkForCriticalErrors looks for critical errors in the expansion context,
// such as circular dependencies.
//
// Takes ctx (context.Context) which carries the logger.
// Takes expCtx (*expansionContext) which contains the diagnostics to check.
//
// Returns error when a circular dependency is found in the expansion path.
func checkForCriticalErrors(ctx context.Context, expCtx *expansionContext) error {
	if !ast_domain.HasErrors(expCtx.diagnostics) {
		return nil
	}

	_, l := logger_domain.From(ctx, log)
	l.Error("Halting expansion due to critical errors found during recursive walk.", logger_domain.Int("error_count", countErrors(expCtx.diagnostics)))

	for _, diagnostic := range expCtx.diagnostics {
		if strings.Contains(diagnostic.Message, "Circular dependency detected") {
			return &CircularDependencyError{Path: expCtx.expansionPath}
		}
	}

	return nil
}

// fillSlotsInTree replaces slot placeholders in a template tree with the given
// content.
//
// Takes nodes ([]*ast_domain.TemplateNode) which is the tree to process.
// Takes content (map[string][]*ast_domain.TemplateNode) which maps slot names
// to their replacement nodes.
//
// Returns []*ast_domain.TemplateNode which is a new tree with slots filled.
// When a slot has no content provided, it uses its default children instead.
func fillSlotsInTree(nodes []*ast_domain.TemplateNode, content map[string][]*ast_domain.TemplateNode) []*ast_domain.TemplateNode {
	resultBuffer := make([]*ast_domain.TemplateNode, 0, len(nodes))
	for _, node := range nodes {
		if node.NodeType == ast_domain.NodeElement && node.TagName == tagPikoSlot {
			slotName, _ := node.GetAttribute(attributeName)
			if providedContent, ok := content[slotName]; ok {
				resultBuffer = append(resultBuffer, providedContent...)
			} else {
				resultBuffer = append(resultBuffer, fillSlotsInTree(node.Children, content)...)
			}
		} else {
			cloned := node.Clone()
			cloned.Children = fillSlotsInTree(node.Children, content)
			resultBuffer = append(resultBuffer, cloned)
		}
	}
	return resultBuffer
}

// finaliseAST builds and validates the final template AST from parsed nodes.
//
// Takes rootNodes ([]*ast_domain.TemplateNode) which contains the parsed root
// template nodes.
// Takes mainComponent (*annotator_dto.ParsedComponent) which provides source
// metadata and diagnostics.
//
// Returns *ast_domain.TemplateAST which is the validated and tidied AST.
func finaliseAST(ctx context.Context, rootNodes []*ast_domain.TemplateNode, mainComponent *annotator_dto.ParsedComponent) *ast_domain.TemplateAST {
	finalAST := &ast_domain.TemplateAST{
		SourcePath:        &mainComponent.SourcePath,
		ExpiresAtUnixNano: nil,
		Metadata:          nil,
		RootNodes:         rootNodes,
		Diagnostics:       mainComponent.Template.Diagnostics,
		SourceSize:        0,
		Tidied:            false,
	}
	ast_domain.ValidateAST(finalAST)
	ast_domain.TidyAST(ctx, finalAST)
	return finalAST
}

// stampNodesWithPackage adds package and source file details to template nodes.
// It stamps each node's directives, dynamic attributes, and rich text content,
// then handles child nodes using recursion.
//
// To reduce per-node allocations, it pre-allocates a batch of
// GoGeneratorAnnotation structs and hands out pointers from the batch.
//
// Takes nodes ([]*ast_domain.TemplateNode) which are the nodes to update.
// Takes packageAlias (string) which is the package alias to set on each node.
// Takes sourcePath (string) which is the path to the source file.
func stampNodesWithPackage(nodes []*ast_domain.TemplateNode, packageAlias string, sourcePath string) {
	needed := countNodesWithoutAnnotation(nodes)
	arena := make([]ast_domain.GoGeneratorAnnotation, needed)
	stampNodesWithPackageRecursive(nodes, packageAlias, sourcePath, arena, new(0))
}

// countNodesWithoutAnnotation counts nodes that lack a GoAnnotations field,
// so we can pre-allocate the right number. Recursion is capped at
// maxAnnotatorDepth so a pathological tree cannot overflow the stack.
//
// Takes nodes ([]*ast_domain.TemplateNode) which are the nodes to inspect
// recursively.
//
// Returns int which is the total count of nodes without annotations.
func countNodesWithoutAnnotation(nodes []*ast_domain.TemplateNode) int {
	return countNodesWithoutAnnotationAt(nodes, 0)
}

// countNodesWithoutAnnotationAt is the depth-tracked recursion behind
// countNodesWithoutAnnotation.
//
// Takes nodes ([]*ast_domain.TemplateNode) which are the nodes to inspect.
// Takes depth (int) which is the current recursion depth.
//
// Returns int which is the total count of nodes without annotations,
// truncated at maxAnnotatorDepth.
func countNodesWithoutAnnotationAt(nodes []*ast_domain.TemplateNode, depth int) int {
	if depth >= maxAnnotatorDepth {
		return 0
	}
	count := 0
	for _, node := range nodes {
		if node.GoAnnotations == nil {
			count++
		}
		count += countNodesWithoutAnnotationAt(node.Children, depth+1)
	}
	return count
}

// stampNodesWithPackageRecursive is the inner recursive implementation that
// stamps nodes using a pre-allocated arena.
//
// Takes nodes ([]*ast_domain.TemplateNode) which are the nodes to stamp.
// Takes packageAlias (string) which is the package alias to set.
// Takes sourcePath (string) which is the source file path to set.
// Takes arena ([]ast_domain.GoGeneratorAnnotation) which is the pre-allocated
// annotation batch.
// Takes idx (*int) which tracks the next free slot in the arena.
func stampNodesWithPackageRecursive(
	nodes []*ast_domain.TemplateNode,
	packageAlias string,
	sourcePath string,
	arena []ast_domain.GoGeneratorAnnotation,
	idx *int,
) {
	stampNodesWithPackageRecursiveAt(nodes, packageAlias, sourcePath, arena, idx, 0)
}

// stampNodesWithPackageRecursiveAt is the depth-tracked recursion behind
// stampNodesWithPackageRecursive.
//
// Takes nodes ([]*ast_domain.TemplateNode) which are the nodes to stamp.
// Takes packageAlias (string) which is the package alias to set.
// Takes sourcePath (string) which is the source file path to set.
// Takes arena ([]ast_domain.GoGeneratorAnnotation) which is the
// pre-allocated annotation batch.
// Takes idx (*int) which tracks the next free slot in the arena.
// Takes depth (int) which is the current recursion depth, capped at
// maxAnnotatorDepth.
func stampNodesWithPackageRecursiveAt(
	nodes []*ast_domain.TemplateNode,
	packageAlias string,
	sourcePath string,
	arena []ast_domain.GoGeneratorAnnotation,
	idx *int,
	depth int,
) {
	if depth >= maxAnnotatorDepth {
		return
	}
	for _, node := range nodes {
		if node.GoAnnotations == nil {
			node.GoAnnotations = &arena[*idx]
			*idx++
		}
		if node.GoAnnotations.OriginalPackageAlias == nil {
			node.GoAnnotations.OriginalPackageAlias = &packageAlias
		}
		if node.GoAnnotations.OriginalSourcePath == nil {
			node.GoAnnotations.OriginalSourcePath = &sourcePath
		}
		stampDynamicAttributesWithSourcePath(node, sourcePath)
		stampDirectivesWithSourcePath(node, sourcePath)
		stampRichTextWithSourcePath(node, sourcePath)
		if len(node.Children) > 0 {
			stampNodesWithPackageRecursiveAt(node.Children, packageAlias, sourcePath, arena, idx, depth+1)
		}
	}
}

// stampDynamicAttributesWithSourcePath sets the OriginalSourcePath on dynamic
// attributes that do not already have one. This means attributes from
// partials are tracked to their source file.
//
// Takes node (*ast_domain.TemplateNode) which is the node whose attributes will
// be updated.
// Takes sourcePath (string) which is the path to set on the attributes.
func stampDynamicAttributesWithSourcePath(node *ast_domain.TemplateNode, sourcePath string) {
	for i := range node.DynamicAttributes {
		attr := &node.DynamicAttributes[i]
		if attr.GoAnnotations == nil {
			attr.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}
		}
		if attr.GoAnnotations.OriginalSourcePath == nil {
			attr.GoAnnotations.OriginalSourcePath = &sourcePath
		}
	}
}

// stampDirectivesWithSourcePath sets the OriginalSourcePath on directives that
// do not already have one. This makes sure that directives from partials are
// tracked as coming from the partial's source file.
//
// Takes node (*ast_domain.TemplateNode) which is the node whose directives to
// stamp.
// Takes sourcePath (string) which is the source path to set.
func stampDirectivesWithSourcePath(node *ast_domain.TemplateNode, sourcePath string) {
	stampDirective := func(d *ast_domain.Directive) {
		if d == nil {
			return
		}
		if d.GoAnnotations == nil {
			d.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}
		}
		if d.GoAnnotations.OriginalSourcePath == nil {
			d.GoAnnotations.OriginalSourcePath = &sourcePath
		}
	}

	stampDirective(node.DirIf)
	stampDirective(node.DirElseIf)
	stampDirective(node.DirElse)
	stampDirective(node.DirFor)
	stampDirective(node.DirShow)
	stampDirective(node.DirModel)
	stampDirective(node.DirRef)
	stampDirective(node.DirClass)
	stampDirective(node.DirStyle)
	stampDirective(node.DirText)
	stampDirective(node.DirHTML)
	stampDirective(node.DirKey)
	stampDirective(node.DirContext)
	stampDirective(node.DirScaffold)
	for _, bindDirective := range node.Binds {
		stampDirective(bindDirective)
	}

	for _, eventDirectives := range node.OnEvents {
		for i := range eventDirectives {
			stampDirective(&eventDirectives[i])
		}
	}

	for _, eventDirectives := range node.CustomEvents {
		for i := range eventDirectives {
			stampDirective(&eventDirectives[i])
		}
	}
}

// stampRichTextWithSourcePath sets the source path on RichText parts that do
// not already have one. This means text from partials can be tracked
// back to the partial's source file.
//
// Takes node (*ast_domain.TemplateNode) which is the node whose RichText parts
// to stamp.
// Takes sourcePath (string) which is the source path to set.
func stampRichTextWithSourcePath(node *ast_domain.TemplateNode, sourcePath string) {
	for i := range node.RichText {
		part := &node.RichText[i]
		if part.GoAnnotations == nil {
			part.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}
		}
		if part.GoAnnotations.OriginalSourcePath == nil {
			part.GoAnnotations.OriginalSourcePath = &sourcePath
		}
	}
}

// countErrors returns the number of diagnostics that have error severity.
//
// Takes diagnostics ([]*ast_domain.Diagnostic) which contains the diagnostics to
// count.
//
// Returns int which is the count of diagnostics with error severity.
func countErrors(diagnostics []*ast_domain.Diagnostic) int {
	count := 0
	for _, d := range diagnostics {
		if d.Severity == ast_domain.Error {
			count++
		}
	}
	return count
}
