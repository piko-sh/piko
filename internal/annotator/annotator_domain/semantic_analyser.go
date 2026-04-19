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

// Performs semantic analysis on component templates by walking the AST
// and validating expressions, types, and structure. Coordinates type
// checking, attribute validation, and diagnostic collection to ensure
// templates are semantically correct before code generation.

import (
	"context"
	"fmt"
	goast "go/ast"
	"maps"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// initialPartialInvocationMapCapacity is the initial capacity for partial
	// invocation maps.
	initialPartialInvocationMapCapacity = 4

	// initialSemanticDiagnosticsCapacity is the starting size for the diagnostics
	// slice used during semantic analysis.
	initialSemanticDiagnosticsCapacity = 8

	// initialAnalysisMapCapacity is the initial size for the analysis map.
	initialAnalysisMapCapacity = 64
)

// SemanticAnalyserConfig holds settings for the SemanticAnalyser.
type SemanticAnalyserConfig struct {
	// PKValidator checks client-side event handlers.
	PKValidator *PKValidator

	// MainComponentHash identifies the main component for primary key
	// validator lookup.
	MainComponentHash string
}

// SemanticAnalyser is the final stage of the compilation pipeline.
// It implements ast.Visitor to walk a fully expanded and linked AST,
// passing work to helper services for context, attributes, and key handling.
type SemanticAnalyser struct {
	// parentEffectiveKey holds the effective key expression from the parent node.
	// It changes at each level of the AST during traversal.
	parentEffectiveKey ast_domain.Expression

	// typeResolver checks and resolves types; shared across the full tree walk.
	typeResolver *TypeResolver

	// contextManager handles scope and context during template analysis.
	contextManager *ContextManager

	// attributeManager checks node attributes during semantic analysis.
	attributeManager *AttributeAnalyser

	// keyAnalyser assigns effective keys to loop items.
	keyAnalyser *KeyAnalyser

	// internalsManager handles the analysis of internal expressions
	// within template nodes.
	internalsManager *InternalsAnalyser

	// ctx holds the current analysis context used when resolving nodes.
	ctx *AnalysisContext

	// currentPartialInfo holds details about the partial being called
	// during analysis.
	currentPartialInfo *ast_domain.PartialInvocationInfo

	// partialInvocationMap tracks all ancestor partial invocations,
	// mapping partial hash to invocation info. This enables correct
	// context resolution when expressions from ancestor partials need
	// to use invocation-specific variable names during SSR inlining.
	partialInvocationMap map[string]*ast_domain.PartialInvocationInfo

	// analysisMap links each TemplateNode to its AnalysisContext. It is shared
	// across all visitors so the LSP can provide code completion, hover info, and
	// go-to-definition for any position.
	analysisMap map[*ast_domain.TemplateNode]*AnalysisContext

	// depth tracks how deep the current node is in the template tree.
	depth int
}

// NewSemanticAnalyser creates a new semantic analyser for template annotation.
//
// Takes resolver (*TypeResolver) which provides type lookup services.
// Takes initialCtx (*AnalysisContext) which sets the starting analysis state.
// Takes pInfo (*ast_domain.PartialInvocationInfo) which holds partial call
// details.
// Takes actions (map[string]ActionInfoProvider) which maps action names to
// their providers.
// Takes virtualModule (*annotator_dto.VirtualModule) which allows context
// switching.
// Takes analysisMap (map[*ast_domain.TemplateNode]*AnalysisContext) which
// stores analysis results for each template node.
// Takes config (SemanticAnalyserConfig) which holds PK settings.
//
// Returns *SemanticAnalyser which is set up and ready to use.
func NewSemanticAnalyser(
	resolver *TypeResolver,
	initialCtx *AnalysisContext,
	pInfo *ast_domain.PartialInvocationInfo,
	actions map[string]ActionInfoProvider,
	virtualModule *annotator_dto.VirtualModule,
	analysisMap map[*ast_domain.TemplateNode]*AnalysisContext,
	config SemanticAnalyserConfig,
) *SemanticAnalyser {
	contextManager := newContextManager(resolver, virtualModule)
	keyAnalyser := newKeyAnalyser(resolver)
	attributeManager := newAttributeAnalyser(resolver, actions, contextManager, config.MainComponentHash, config.PKValidator)
	internalsManager := newInternalsAnalyser(resolver)

	return &SemanticAnalyser{
		parentEffectiveKey:   nil,
		typeResolver:         resolver,
		contextManager:       contextManager,
		attributeManager:     attributeManager,
		keyAnalyser:          keyAnalyser,
		internalsManager:     internalsManager,
		ctx:                  initialCtx,
		currentPartialInfo:   pInfo,
		partialInvocationMap: make(map[string]*ast_domain.PartialInvocationInfo, initialPartialInvocationMapCapacity),
		analysisMap:          analysisMap,
		depth:                0,
	}
}

// Enter performs semantic analysis on a node when the AST walker enters it.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to
// analyse.
//
// Returns ast_domain.Visitor which is the visitor for processing child nodes.
// Returns error when the for directive cannot be handled.
func (sa *SemanticAnalyser) Enter(goCtx context.Context, node *ast_domain.TemplateNode) (ast_domain.Visitor, error) {
	if node == nil {
		return nil, nil
	}

	if isPartialInvocationNode(node) && node.DirFor != nil {
		sa.resolveForDirectiveInContext(goCtx, node, sa.ctx)
	}

	ctxForThisNode, activePInfo := sa.contextManager.DetermineNodeContext(node, sa.ctx, sa.currentPartialInfo, sa.depth)

	if node.GoAnnotations != nil && node.GoAnnotations.PartialInfo != nil {
		sa.attributeManager.MarkPartialRendered(node.GoAnnotations.PartialInfo.PartialAlias)
	}

	ctxForAttributes, err := sa.handleForDirective(goCtx, node, ctxForThisNode)
	if err != nil {
		return nil, fmt.Errorf("handling for-directive on tag %q: %w", node.TagName, err)
	}

	partialMap := sa.buildPartialInvocationMap(activePInfo)
	sa.attributeManager.AnalyseNodeAttributes(goCtx, node, ctxForAttributes, activePInfo, partialMap)

	ctxForInternalsAndChildren := ctxForAttributes

	if node.DirIf != nil && node.DirIf.Expression != nil {
		guards := ExtractNilGuardsFromCondition(node.DirIf.Expression)
		if len(guards) > 0 {
			ctxForInternalsAndChildren = ctxForAttributes.ForChildScopeWithNilGuards(guards)
		}
	}

	if sa.analysisMap != nil {
		sa.analysisMap[node] = ctxForInternalsAndChildren
	}
	sa.internalsManager.AnalyseInternalExpressions(goCtx, node, ctxForInternalsAndChildren, activePInfo)

	effectiveKeyForChildren := determineEffectiveKeyForChildren(node)
	return sa.newVisitorForChild(ctxForInternalsAndChildren, activePInfo, effectiveKeyForChildren), nil
}

// Exit runs after a node and all its children have been visited.
//
// Returns error when the visit fails; always returns nil.
func (*SemanticAnalyser) Exit(_ context.Context, _ *ast_domain.TemplateNode) error {
	return nil
}

// newVisitorForChild creates a new visitor for a child scope.
//
// The specialists and the analysisMap are shared; only the state changes.
// When entering a new partial, the partialInvocationMap is extended with
// the new partial's info.
//
// Takes newCtx (*AnalysisContext) which provides the context for the child.
// Takes pInfo (*ast_domain.PartialInvocationInfo) which holds partial info,
// or nil if not entering a partial.
// Takes parentKey (ast_domain.Expression) which identifies the parent node.
//
// Returns *SemanticAnalyser which is the new visitor for the child scope.
func (sa *SemanticAnalyser) newVisitorForChild(newCtx *AnalysisContext, pInfo *ast_domain.PartialInvocationInfo, parentKey ast_domain.Expression) *SemanticAnalyser {
	partialMap := sa.partialInvocationMap
	if pInfo != nil && pInfo.PartialPackageName != "" {
		partialMap = make(map[string]*ast_domain.PartialInvocationInfo, len(sa.partialInvocationMap)+1)
		maps.Copy(partialMap, sa.partialInvocationMap)
		partialMap[pInfo.PartialPackageName] = pInfo
	}

	return &SemanticAnalyser{
		parentEffectiveKey:   parentKey,
		typeResolver:         sa.typeResolver,
		contextManager:       sa.contextManager,
		attributeManager:     sa.attributeManager,
		keyAnalyser:          sa.keyAnalyser,
		internalsManager:     sa.internalsManager,
		ctx:                  newCtx,
		currentPartialInfo:   pInfo,
		partialInvocationMap: partialMap,
		analysisMap:          sa.analysisMap,
		depth:                sa.depth + 1,
	}
}

// resolveForDirectiveInContext resolves and checks the p-for directive
// expression using the given context. This is used for partial invocation
// nodes where the expression must be resolved in the parent context before
// any context switch happens.
//
// Takes node (*ast_domain.TemplateNode) which contains the directive to
// resolve.
// Takes ctx (*AnalysisContext) which provides the context for resolution.
func (sa *SemanticAnalyser) resolveForDirectiveInContext(goCtx context.Context, node *ast_domain.TemplateNode, ctx *AnalysisContext) {
	if node.DirFor == nil {
		return
	}

	resolveAndValidate(goCtx, node.DirFor, ctx, sa.typeResolver, validateForDirective)

	if forExpr, ok := node.DirFor.Expression.(*ast_domain.ForInExpression); ok {
		loopVarManager := getLoopVariableManager(ctx)
		if forExpr.ItemVariable != nil {
			loopVarManager.ValidateLoopVariable(forExpr.ItemVariable, node.DirFor)
		}
		if forExpr.IndexVariable != nil {
			loopVarManager.ValidateLoopVariable(forExpr.IndexVariable, node.DirFor)
		}
		putLoopVariableManager(loopVarManager)

		sa.keyAnalyser.AnalyseAndSetEffectiveKey(node, sa.parentEffectiveKey, ctx, sa.depth)
	}
}

// handleForDirective processes the p-for directive, validating loop variables
// and creating loop context. For partial invocation nodes, the expression has
// already been resolved by resolveForDirectiveInContext, and only the loop
// context is created here.
//
// Takes node (*ast_domain.TemplateNode) which contains the directive to
// process.
// Takes ctxForThisNode (*AnalysisContext) which provides the current analysis
// context.
//
// Returns *AnalysisContext which is the loop scope context for analysing
// attributes.
// Returns error when the loop context cannot be created.
func (sa *SemanticAnalyser) handleForDirective(goCtx context.Context, node *ast_domain.TemplateNode, ctxForThisNode *AnalysisContext) (*AnalysisContext, error) {
	if node.DirFor == nil {
		return ctxForThisNode, nil
	}

	if !isPartialInvocationNode(node) {
		resolveAndValidate(goCtx, node.DirFor, ctxForThisNode, sa.typeResolver, validateForDirective)

		if forExpr, ok := node.DirFor.Expression.(*ast_domain.ForInExpression); ok {
			loopVarManager := getLoopVariableManager(ctxForThisNode)
			if forExpr.ItemVariable != nil {
				loopVarManager.ValidateLoopVariable(forExpr.ItemVariable, node.DirFor)
			}
			if forExpr.IndexVariable != nil {
				loopVarManager.ValidateLoopVariable(forExpr.IndexVariable, node.DirFor)
			}
			putLoopVariableManager(loopVarManager)

			sa.keyAnalyser.AnalyseAndSetEffectiveKey(node, sa.parentEffectiveKey, ctxForThisNode, sa.depth)
		}
	}

	return sa.contextManager.CreateForLoopContext(goCtx, node, ctxForThisNode)
}

// buildPartialInvocationMap creates the partial invocation map for a node.
//
// Takes activePInfo (*ast_domain.PartialInvocationInfo) which specifies the
// active partial invocation to include in the map, or nil to use the existing
// map unchanged.
//
// Returns map[string]*ast_domain.PartialInvocationInfo which maps
// package names to their partial invocation info.
func (sa *SemanticAnalyser) buildPartialInvocationMap(activePInfo *ast_domain.PartialInvocationInfo) map[string]*ast_domain.PartialInvocationInfo {
	partialMap := sa.partialInvocationMap
	if activePInfo != nil && activePInfo.PartialPackageName != "" {
		partialMap = make(map[string]*ast_domain.PartialInvocationInfo, len(sa.partialInvocationMap)+1)
		maps.Copy(partialMap, sa.partialInvocationMap)
		partialMap[activePInfo.PartialPackageName] = activePInfo
	}
	return partialMap
}

// Annotate performs semantic analysis on a linked AST, resolving types,
// validating expressions, and enriching nodes with annotations.
//
// Takes linkingResult (*annotator_dto.LinkingResult) which contains the linked
// AST and virtual module to analyse.
// Takes resolver (*TypeResolver) which resolves type information during
// analysis.
// Takes entryPointPath (string) which identifies the main component file.
// Takes actions (map[string]ActionInfoProvider) which provides action metadata
// for validation.
//
// Returns *annotator_dto.AnnotationResult which contains the annotated AST and
// analysis context map.
// Returns []*ast_domain.Diagnostic which contains any semantic warnings or
// errors found.
// Returns error when the AST traversal fails or the main component cannot be
// found.
func Annotate(
	ctx context.Context,
	linkingResult *annotator_dto.LinkingResult,
	resolver *TypeResolver,
	entryPointPath string,
	actions map[string]ActionInfoProvider,
) (*annotator_dto.AnnotationResult, []*ast_domain.Diagnostic, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "Annotate")
	defer span.End()

	l.Internal("--- [ANNOTATOR START] --- Starting Semantic Analysis Stage ---")
	diagnostics := make([]*ast_domain.Diagnostic, 0, initialSemanticDiagnosticsCapacity)
	flattenedAST := linkingResult.LinkedAST
	virtualModule := linkingResult.VirtualModule

	if flattenedAST == nil || len(flattenedAST.RootNodes) == 0 {
		l.Internal("[ANNOTATOR] AST is empty, skipping analysis.")
		return createEmptyAnnotationResult(flattenedAST, diagnostics)
	}

	mainVirtualComponent, err := findMainComponent(ctx, virtualModule, entryPointPath)
	if err != nil {
		return nil, nil, fmt.Errorf("finding main component for %q: %w", entryPointPath, err)
	}

	rootCtx := setupAndPopulateRootContext(ctx, mainVirtualComponent, resolver, &diagnostics)
	analysisMap := make(map[*ast_domain.TemplateNode]*AnalysisContext, initialAnalysisMapCapacity)
	pkValidator := createPKValidator(ctx, mainVirtualComponent)

	initialVisitor := NewSemanticAnalyser(resolver, rootCtx, nil, actions, virtualModule, analysisMap, SemanticAnalyserConfig{
		MainComponentHash: mainVirtualComponent.HashedName,
		PKValidator:       pkValidator,
	})

	if err := runASTTraversal(ctx, flattenedAST, initialVisitor); err != nil {
		l.Error("AST traversal failed with an error.", logger_domain.Error(err))
		return nil, diagnostics, fmt.Errorf("semantic analysis walk failed: %w", err)
	}

	runPostTraversalValidation(pkValidator, rootCtx, mainVirtualComponent, virtualModule, &diagnostics)

	result := buildAnnotationResult(flattenedAST, virtualModule, linkingResult, analysisMap, mainVirtualComponent)
	l.Internal("--- [ANNOTATOR END] --- Semantic Analysis Stage Completed ---", logger_domain.Int("diagnostics_found", len(diagnostics)))
	return result, diagnostics, nil
}

// createEmptyAnnotationResult creates a default AnnotationResult for empty
// ASTs.
//
// Takes flattenedAST (*ast_domain.TemplateAST) which provides the AST
// structure to wrap.
// Takes diagnostics ([]*ast_domain.Diagnostic) which contains any diagnostic
// messages to include.
//
// Returns *annotator_dto.AnnotationResult which contains the wrapped AST with
// nil values for all optional fields.
// Returns []*ast_domain.Diagnostic which is the unchanged diagnostics slice.
// Returns error which is always nil here.
func createEmptyAnnotationResult(flattenedAST *ast_domain.TemplateAST, diagnostics []*ast_domain.Diagnostic) (*annotator_dto.AnnotationResult, []*ast_domain.Diagnostic, error) {
	return &annotator_dto.AnnotationResult{
		AnnotatedAST:          flattenedAST,
		VirtualModule:         nil,
		StyleBlock:            "",
		AssetRefs:             nil,
		CustomTags:            nil,
		UniqueInvocations:     nil,
		AssetDependencies:     nil,
		AnalysisMap:           nil,
		EntryPointStyleBlocks: nil,
		ClientScript:          "",
	}, diagnostics, nil
}

// setupAndPopulateRootContext creates and fills the root analysis context
// with debug logging.
//
// Takes ctx (context.Context) which carries the session logger.
// Takes mainVirtualComponent (*annotator_dto.VirtualComponent) which provides
// the main component to analyse.
// Takes resolver (*TypeResolver) which resolves types during analysis.
// Takes diagnostics (*[]*ast_domain.Diagnostic) which collects any diagnostic
// messages.
//
// Returns *AnalysisContext which is the filled root context ready for use.
func setupAndPopulateRootContext(
	ctx context.Context,
	mainVirtualComponent *annotator_dto.VirtualComponent,
	resolver *TypeResolver,
	diagnostics *[]*ast_domain.Diagnostic,
) *AnalysisContext {
	ctx, l := logger_domain.From(ctx, log)
	if l.Enabled(logger_domain.LevelTrace) {
		l.Trace("--- [TYPE INSPECTOR STATE SNAPSHOT] ---")
		for _, line := range resolver.inspector.Debug(mainVirtualComponent.CanonicalGoPackagePath, mainVirtualComponent.VirtualGoFilePath) {
			l.Trace(line)
		}
		l.Trace("--- [END SNAPSHOT] ---")
	}

	l.Trace("[ANNOTATOR] Creating root analysis context.",
		logger_domain.String("packagePath", mainVirtualComponent.CanonicalGoPackagePath),
		logger_domain.String("packageName", mainVirtualComponent.RewrittenScriptAST.Name.Name),
		logger_domain.String("sourcePath", mainVirtualComponent.Source.SourcePath),
	)

	rootCtx := NewRootAnalysisContext(
		diagnostics,
		mainVirtualComponent.CanonicalGoPackagePath,
		mainVirtualComponent.RewrittenScriptAST.Name.Name,
		mainVirtualComponent.VirtualGoFilePath,
		mainVirtualComponent.Source.SourcePath,
	)
	rootCtx.Logger = l

	rootCtx.TranslationKeys = buildTranslationKeySet(mainVirtualComponent)

	PopulateRootContext(rootCtx, resolver, mainVirtualComponent)
	l.Trace("[SA-DEBUG] TOP-LEVEL: Populated Root Context")
	logAnalysisContext(rootCtx, "Initial Root Context")

	return rootCtx
}

// buildTranslationKeySet collects all translation keys from a component.
//
// Takes vc (*annotator_dto.VirtualComponent) which provides the component
// that holds translation data.
//
// Returns *TranslationKeySet which contains the local translation keys, or nil
// if the component is nil or has no translations.
func buildTranslationKeySet(vc *annotator_dto.VirtualComponent) *TranslationKeySet {
	if vc == nil || vc.Source == nil {
		return nil
	}

	localTrans := vc.Source.LocalTranslations
	if len(localTrans) == 0 {
		return nil
	}

	var estimatedSize int
	for _, localeKeys := range localTrans {
		estimatedSize = len(localeKeys)
		break
	}
	localKeys := make(map[string]struct{}, estimatedSize)
	for _, localeKeys := range localTrans {
		for key := range localeKeys {
			localKeys[key] = struct{}{}
		}
	}

	if len(localKeys) == 0 {
		return nil
	}

	return &TranslationKeySet{
		LocalKeys:  localKeys,
		GlobalKeys: nil,
	}
}

// findMainComponent finds and checks the main component in the virtual module.
//
// Takes ctx (context.Context) which carries the session logger.
// Takes virtualModule (*annotator_dto.VirtualModule) which contains the
// component graph and hash mappings.
// Takes entryPointPath (string) which specifies the path to look up in the
// component graph.
//
// Returns *annotator_dto.VirtualComponent which is the main component for the
// given entry point.
// Returns error when the entry point path is not found in the graph or the
// component hash cannot be resolved.
func findMainComponent(ctx context.Context, virtualModule *annotator_dto.VirtualModule, entryPointPath string) (*annotator_dto.VirtualComponent, error) {
	ctx, l := logger_domain.From(ctx, log)
	mainComponentHashedName, ok := virtualModule.Graph.PathToHashedName[entryPointPath]
	if !ok {
		return nil, fmt.Errorf("internal consistency error: entry point path '%s' not found in component graph", entryPointPath)
	}
	l.Trace("[ANNOTATOR] Entry point component identified.", logger_domain.String("HashedName", mainComponentHashedName))

	mainVirtualComponent, ok := virtualModule.ComponentsByHash[mainComponentHashedName]
	if !ok {
		l.Error("FATAL: Inconsistency detected. Cannot find main component in VirtualModule.ComponentsByHash.",
			logger_domain.String("lookupKey", mainComponentHashedName),
		)
		return nil, fmt.Errorf("internal consistency error: could not find main virtual component for hash '%s'", mainComponentHashedName)
	}

	return mainVirtualComponent, nil
}

// createPKValidator creates a PK validator for client-side event handlers.
//
// When the component has no client script, returns nil without creating a
// validator.
//
// Takes ctx (context.Context) which carries the session logger.
// Takes comp (*annotator_dto.VirtualComponent) which provides the component
// source containing the client script and imports.
//
// Returns *PKValidator which checks client-side event handler functions, or
// nil if no client script exists.
func createPKValidator(ctx context.Context, comp *annotator_dto.VirtualComponent) *PKValidator {
	ctx, l := logger_domain.From(ctx, log)
	if comp.Source.ClientScript == "" {
		return nil
	}

	validator := NewPKValidator(comp.Source.ClientScript, comp.Source.SourcePath)

	partialAliases := make([]string, 0, len(comp.Source.PikoImports))
	for _, imp := range comp.Source.PikoImports {
		partialAliases = append(partialAliases, imp.Alias)
	}
	validator.RegisterImportedPartials(partialAliases)

	l.Internal("[ANNOTATOR] Created PK validator for client script exports.")
	return validator
}

// runASTTraversal walks through the AST using the given visitor.
//
// Takes ctx (context.Context) which carries the session logger.
// Takes ast (*ast_domain.TemplateAST) which is the tree to traverse.
// Takes visitor (*SemanticAnalyser) which processes each node.
//
// Returns error when the traversal fails.
func runASTTraversal(ctx context.Context, ast *ast_domain.TemplateAST, visitor *SemanticAnalyser) error {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("[ANNOTATOR] Starting AST traversal with visitor...")
	err := ast.Accept(ctx, visitor)
	l.Internal("[ANNOTATOR] AST traversal complete.")
	if err != nil {
		return fmt.Errorf("traversing AST: %w", err)
	}
	return nil
}

// runPostTraversalValidation runs checks after the full AST walk is complete.
//
// Takes pkValidator (*PKValidator) which checks partial file templates.
// Takes rootCtx (*AnalysisContext) which provides the analysis context.
// Takes mainComp (*annotator_dto.VirtualComponent) which is the main
// component.
// Takes virtualModule (*annotator_dto.VirtualModule) which is the module to
// check.
// Takes diagnostics (*[]*ast_domain.Diagnostic) which collects any problems
// found.
func runPostTraversalValidation(
	pkValidator *PKValidator,
	rootCtx *AnalysisContext,
	mainComp *annotator_dto.VirtualComponent,
	virtualModule *annotator_dto.VirtualModule,
	diagnostics *[]*ast_domain.Diagnostic,
) {
	if pkValidator != nil && pkValidator.HasClientScript() {
		scriptLocation := getClientScriptLocation(mainComp)
		pkValidator.ReportUnusedExports(rootCtx, scriptLocation)
		pkValidator.ReportOrphanedPartials(rootCtx, scriptLocation)
	}

	partialAnalyser := NewPartialDependencyAnalyser()
	cycleDiagnostics := partialAnalyser.AnalyseVirtualModule(virtualModule, mainComp)
	*diagnostics = append(*diagnostics, cycleDiagnostics...)
}

// buildAnnotationResult builds the final annotation result by combining all
// processed data into a single structure.
//
// Takes flattenedAST (*ast_domain.TemplateAST) which is the processed template
// tree.
// Takes virtualModule (*annotator_dto.VirtualModule) which holds the module
// data.
// Takes linkingResult (*annotator_dto.LinkingResult) which contains the CSS
// and invocation data.
// Takes analysisMap (map[*ast_domain.TemplateNode]*AnalysisContext) which maps
// nodes to their analysis results.
// Takes mainComp (*annotator_dto.VirtualComponent) which provides the entry
// point style blocks and client script.
//
// Returns *annotator_dto.AnnotationResult which holds all annotation data in
// one structure.
func buildAnnotationResult(
	flattenedAST *ast_domain.TemplateAST,
	virtualModule *annotator_dto.VirtualModule,
	linkingResult *annotator_dto.LinkingResult,
	analysisMap map[*ast_domain.TemplateNode]*AnalysisContext,
	mainComp *annotator_dto.VirtualComponent,
) *annotator_dto.AnnotationResult {
	return &annotator_dto.AnnotationResult{
		AnnotatedAST:          flattenedAST,
		VirtualModule:         virtualModule,
		StyleBlock:            linkingResult.CombinedCSS,
		AssetRefs:             nil,
		CustomTags:            nil,
		UniqueInvocations:     linkingResult.UniqueInvocations,
		AssetDependencies:     nil,
		AnalysisMap:           analysisMap,
		EntryPointStyleBlocks: mainComp.Source.StyleBlocks,
		ClientScript:          mainComp.Source.ClientScript,
	}
}

// determineEffectiveKeyForChildren finds the key expression to pass to child
// nodes.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns ast_domain.Expression which is the key expression, or nil if none
// is found.
func determineEffectiveKeyForChildren(node *ast_domain.TemplateNode) ast_domain.Expression {
	if node.GoAnnotations != nil && node.GoAnnotations.EffectiveKeyExpression != nil {
		return node.GoAnnotations.EffectiveKeyExpression
	}
	if node.Key != nil {
		return node.Key
	}
	return nil
}

// resolveAndValidate resolves type annotations for a directive and runs
// validation.
//
// When d is nil or has no expression, returns at once without action.
//
// Takes d (*ast_domain.Directive) which is the directive to process.
// Takes ctx (*AnalysisContext) which provides the analysis context.
// Takes resolver (*TypeResolver) which resolves type annotations.
// Takes validateFunction (func(...)) which validates the directive
// after resolution.
func resolveAndValidate(goCtx context.Context, d *ast_domain.Directive, ctx *AnalysisContext, resolver *TypeResolver, validateFunction func(*ast_domain.Directive, *AnalysisContext)) {
	if d == nil || d.Expression == nil {
		return
	}
	d.GoAnnotations = resolver.Resolve(goCtx, ctx, d.Expression, d.Location)
	if validateFunction != nil {
		validateFunction(d, ctx)
	}
}

// validateForDirective checks that a p-for directive has valid loop syntax.
//
// Takes d (*ast_domain.Directive) which is the directive to validate.
// Takes ctx (*AnalysisContext) which holds the analysis state for reporting
// errors.
func validateForDirective(d *ast_domain.Directive, ctx *AnalysisContext) {
	if rejectEventPlaceholderInDirective(d, ctx) || rejectFormPlaceholderInDirective(d, ctx) {
		return
	}
	forExpr, ok := d.Expression.(*ast_domain.ForInExpression)
	if !ok {
		ctx.addDiagnostic(ast_domain.Error, "p-for expression is not a valid 'in' loop (e.g., 'item in items')", d.RawExpression, d.Location, d.GoAnnotations, annotator_dto.CodeInvalidLoopExpression)
		return
	}
	ann := getAnnotationFromExpression(forExpr.Collection)
	if ann != nil && ann.ResolvedType != nil && !isIterable(ann.ResolvedType) {
		typeName := goastutil.ASTToTypeString(ann.ResolvedType.TypeExpression, ann.ResolvedType.PackageAlias)
		ctx.addDiagnosticForExpression(ast_domain.Error, fmt.Sprintf("Cannot loop over type '%s'", typeName), d.Expression, d.Location, d.GoAnnotations, annotator_dto.CodeInvalidLoopExpression)
	}
}

// isIterable reports whether the given type can be iterated over.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which describes the type to
// check.
//
// Returns bool which is true for arrays, maps, and strings.
func isIterable(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return false
	}
	switch typeInfo.TypeExpression.(type) {
	case *goast.ArrayType, *goast.MapType:
		return true
	case *goast.Ident:
		return typeInfo.TypeExpression.(*goast.Ident).Name == "string"
	}
	return false
}

// logAnalysisContext writes the current analysis context state to the log.
//
// Takes ctx (*AnalysisContext) which provides the context to log.
// Takes title (string) which labels the log entry.
func logAnalysisContext(ctx *AnalysisContext, title string) {
	if ctx == nil || !ctx.Logger.Enabled(logger_domain.LevelTrace) {
		return
	}
	ctx.Logger.Trace(title,
		logger_domain.String("CurrentGoFullPackagePath", ctx.CurrentGoFullPackagePath),
		logger_domain.String("CurrentGoPackageName", ctx.CurrentGoPackageName),
		logger_domain.String("SFCSourcePath", ctx.SFCSourcePath),
		logger_domain.String("SymbolsInScope", strings.Join(ctx.Symbols.AllSymbolNames(), ", ")),
	)
}

// getClientScriptLocation returns the source location of the client script
// block in a virtual component.
//
// Takes vc (*annotator_dto.VirtualComponent) which provides the component
// containing the client script.
//
// Returns ast_domain.Location which gives the position of the script block,
// or line 1 if no location is found.
func getClientScriptLocation(vc *annotator_dto.VirtualComponent) ast_domain.Location {
	if vc == nil || vc.Source == nil {
		return ast_domain.Location{Line: 1, Column: 1, Offset: 0}
	}
	return ast_domain.Location{Line: 1, Column: 1, Offset: 0}
}
