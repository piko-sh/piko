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

// Manages the semantic analysis context by tracking state, diagnostics, and symbol tables during AST traversal.
// Provides helper methods for adding diagnostics, managing scopes, and accessing analysis context throughout the semantic analysis phase.

import (
	"context"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/logger/logger_domain"
)

// ContextManager creates and switches AnalysisContext instances during AST
// traversal. It handles scope creation for loops and context switching for
// partials and slotted content.
type ContextManager struct {
	// typeResolver finds Go types for loop items and indices.
	typeResolver *TypeResolver

	// virtualModule stores component data used for context switching during analysis.
	virtualModule *annotator_dto.VirtualModule
}

// CreateForLoopContext creates a new child context for a p-for loop, defining
// the loop variables (item and index) in the new scope based on the
// collection's type.
//
// Takes node (*ast_domain.TemplateNode) which contains the p-for directive to
// process.
// Takes parentCtx (*AnalysisContext) which provides the parent scope for
// variable resolution.
//
// Returns *AnalysisContext which is the new child context with loop variables
// defined, or the parent context if no p-for directive is present.
// Returns error when context creation fails.
func (cm *ContextManager) CreateForLoopContext(
	goCtx context.Context,
	node *ast_domain.TemplateNode,
	parentCtx *AnalysisContext,
) (*AnalysisContext, error) {
	if node.DirFor == nil {
		return parentCtx, nil
	}

	forExpr, ok := node.DirFor.Expression.(*ast_domain.ForInExpression)
	if !ok {
		return parentCtx, nil
	}

	collectionAnn := getAnnotationFromExpression(forExpr)
	if collectionAnn == nil || collectionAnn.ResolvedType == nil {
		return parentCtx, nil
	}

	loopCtx := parentCtx.ForChildScope()
	sfcSourcePath := parentCtx.SFCSourcePath

	cm.defineItemVariable(goCtx, forExpr, parentCtx, loopCtx, collectionAnn, sfcSourcePath)
	cm.defineIndexVariable(forExpr, parentCtx, loopCtx, collectionAnn, sfcSourcePath)

	return loopCtx, nil
}

// DeterminePartialSelfContext creates the analysis context for a partial's own
// template scope.
//
// If the node is a partial root and the partial component can be found, this
// creates a new context for the partial's scope. Otherwise, it returns the
// parent context unchanged.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check
// for partial root status.
// Takes parentCtx (*AnalysisContext) which is the parent context to use or
// return.
//
// Returns *AnalysisContext which is either a new context for the partial's
// scope or the parent context if not a partial root.
func (cm *ContextManager) DeterminePartialSelfContext(
	node *ast_domain.TemplateNode,
	parentCtx *AnalysisContext,
) *AnalysisContext {
	if node.GoAnnotations != nil && node.GoAnnotations.PartialInfo != nil {
		pInfo := node.GoAnnotations.PartialInfo
		vc, ok := cm.virtualModule.ComponentsByHash[pInfo.PartialPackageName]
		if ok {
			partialCtx := parentCtx.ForNewPackageContext(
				vc.CanonicalGoPackagePath,
				vc.RewrittenScriptAST.Name.Name,
				vc.VirtualGoFilePath,
				vc.Source.SourcePath,
			)
			populatePartialContext(partialCtx, cm.typeResolver, vc, pInfo)
			return partialCtx
		}
	}
	return parentCtx
}

// DetermineNodeContext finds the correct analysis context for a node and its
// children. It handles context switches for partials and slotted content.
//
// Takes node (*ast_domain.TemplateNode) which is the node to analyse.
// Takes parentCtx (*AnalysisContext) which is the context from the parent node.
// Takes currentPartialInfo (*ast_domain.PartialInvocationInfo) which is the
// partial info from the parent scope.
// Takes depth (int) which is how deep this node is in the tree, used for
// logging.
//
// Returns *AnalysisContext which is the context to use for this node.
// Returns *ast_domain.PartialInvocationInfo which is the partial info for this
// node and its children.
func (cm *ContextManager) DetermineNodeContext(
	node *ast_domain.TemplateNode,
	parentCtx *AnalysisContext,
	currentPartialInfo *ast_domain.PartialInvocationInfo,
	depth int,
) (*AnalysisContext, *ast_domain.PartialInvocationInfo) {
	ctxForThisNode := parentCtx
	activePInfo := currentPartialInfo

	if node.GoAnnotations != nil && node.GoAnnotations.PartialInfo != nil {
		activePInfo = node.GoAnnotations.PartialInfo

		parentCtx.Logger.Trace("CONTEXT: Node is a partial root",
			logger_domain.Int(logKeyDepth, depth),
			logger_domain.String("invocationKey", activePInfo.InvocationKey))
	}

	if switchResult := cm.tryContextSwitch(node, parentCtx, activePInfo, depth); switchResult != nil {
		ctxForThisNode = switchResult.newCtx
		activePInfo = switchResult.activePInfo
	}

	return ctxForThisNode, activePInfo
}

// CreateForLoopVisitorContext wraps CreateForLoopContext for use in the old
// createForLoopVisitor pattern. This is used during the transition period.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to process.
// Takes parentCtx (*AnalysisContext) which provides the parent analysis scope.
// Takes depth (int) which shows the current nesting depth for logging.
//
// Returns *AnalysisContext which is the new loop context with defined symbols.
// Returns error when context creation fails.
func (cm *ContextManager) CreateForLoopVisitorContext(
	goCtx context.Context,
	node *ast_domain.TemplateNode,
	parentCtx *AnalysisContext,
	depth int,
) (*AnalysisContext, error) {
	if node.DirFor == nil {
		return parentCtx, nil
	}

	parentCtx.Logger.Trace("[createForLoopVisitor] For expression",
		logger_domain.Int(logKeyDepth, depth),
		logger_domain.String("expression", node.DirFor.RawExpression))

	forExpr, ok := node.DirFor.Expression.(*ast_domain.ForInExpression)
	if !ok {
		return parentCtx, nil
	}

	collectionAnn := getAnnotationFromExpression(forExpr)
	if collectionAnn == nil {
		parentCtx.Logger.Trace("[createForLoopVisitor] WARN: Collection expression has no annotation. Cannot create new scope.",
			logger_domain.Int(logKeyDepth, depth))
		return parentCtx, nil
	}

	collectionTypeString := goastutil.ASTToTypeString(collectionAnn.ResolvedType.TypeExpression, collectionAnn.ResolvedType.PackageAlias)
	parentCtx.Logger.Trace("[createForLoopVisitor] Resolved collection type",
		logger_domain.Int(logKeyDepth, depth),
		logger_domain.String("type", collectionTypeString))

	loopCtx := parentCtx.ForChildScope()

	if forExpr.ItemVariable != nil {
		itemTypeInfo := cm.typeResolver.DetermineIterationItemType(goCtx, parentCtx, forExpr.Collection, collectionAnn.ResolvedType)
		loopCtx.Symbols.Define(Symbol{
			Name:                forExpr.ItemVariable.Name,
			CodeGenVarName:      forExpr.ItemVariable.Name,
			TypeInfo:            itemTypeInfo,
			SourceInvocationKey: "",
		})
		itemTypeString := goastutil.ASTToTypeString(itemTypeInfo.TypeExpression, itemTypeInfo.PackageAlias)
		parentCtx.Logger.Trace("[createForLoopVisitor] Defining loop variable",
			logger_domain.Int(logKeyDepth, depth),
			logger_domain.String("name", forExpr.ItemVariable.Name),
			logger_domain.String("type", itemTypeString))
	}
	if forExpr.IndexVariable != nil {
		indexTypeInfo := cm.typeResolver.DetermineIterationIndexType(parentCtx, collectionAnn.ResolvedType)
		loopCtx.Symbols.Define(Symbol{
			Name:                forExpr.IndexVariable.Name,
			CodeGenVarName:      forExpr.IndexVariable.Name,
			TypeInfo:            indexTypeInfo,
			SourceInvocationKey: "",
		})
		indexTypeString := goastutil.ASTToTypeString(indexTypeInfo.TypeExpression, indexTypeInfo.PackageAlias)
		parentCtx.Logger.Trace("[createForLoopVisitor] Defining loop variable",
			logger_domain.Int(logKeyDepth, depth),
			logger_domain.String("name", forExpr.IndexVariable.Name),
			logger_domain.String("type", indexTypeString))
	}

	parentCtx.Logger.Trace("[createForLoopVisitor] New loop context symbols",
		logger_domain.Int(logKeyDepth, depth),
		logger_domain.Strings("symbols", loopCtx.Symbols.AllSymbolNames()))
	return loopCtx, nil
}

// contextSwitchResult holds the outcome of a context switch attempt.
type contextSwitchResult struct {
	// newCtx is the analysis context to use after a context switch.
	newCtx *AnalysisContext

	// activePInfo holds the partial invocation info for this context.
	activePInfo *ast_domain.PartialInvocationInfo
}

// tryContextSwitch checks if a node's origin needs a context switch and
// performs it if needed.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
// Takes parentCtx (*AnalysisContext) which is the current analysis context.
// Takes activePInfo (*ast_domain.PartialInvocationInfo) which holds partial
// invocation state.
// Takes depth (int) which is the current traversal depth for logging.
//
// Returns *contextSwitchResult which holds the new context and partial info,
// or nil if no switch is needed.
func (cm *ContextManager) tryContextSwitch(
	node *ast_domain.TemplateNode,
	parentCtx *AnalysisContext,
	activePInfo *ast_domain.PartialInvocationInfo,
	depth int,
) *contextSwitchResult {
	if !cm.needsContextSwitch(node, parentCtx) {
		return nil
	}

	nodeOriginHashedName := *node.GoAnnotations.OriginalPackageAlias
	nodeOriginSFCPath := *node.GoAnnotations.OriginalSourcePath

	cm.logContextSwitchDetected(parentCtx, nodeOriginHashedName, depth)

	switchedVirtualComp, compOk := cm.virtualModule.ComponentsByHash[nodeOriginHashedName]
	if !compOk {
		return nil
	}

	newCtx := cm.createSwitchedContext(parentCtx, switchedVirtualComp, nodeOriginSFCPath)
	activePInfo = cm.populateSwitchedContext(newCtx, switchedVirtualComp, activePInfo, parentCtx, depth)

	parentCtx.Logger.Trace("Context switched",
		logger_domain.Int(logKeyDepth, depth),
		logger_domain.String("packagePath", newCtx.CurrentGoFullPackagePath),
		logger_domain.Strings("symbols", newCtx.Symbols.AllSymbolNames()))

	return &contextSwitchResult{newCtx: newCtx, activePInfo: activePInfo}
}

// needsContextSwitch checks if a node requires a context switch based on its
// origin.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
// Takes parentCtx (*AnalysisContext) which provides the current context.
//
// Returns bool which is true when the node's origin differs from the current
// context.
func (cm *ContextManager) needsContextSwitch(node *ast_domain.TemplateNode, parentCtx *AnalysisContext) bool {
	if node.GoAnnotations == nil {
		return false
	}
	if node.GoAnnotations.OriginalPackageAlias == nil || node.GoAnnotations.OriginalSourcePath == nil {
		return false
	}

	nodeOriginHashedName := *node.GoAnnotations.OriginalPackageAlias
	currentVirtualComponent, ok := cm.virtualModule.ComponentsByGoPath[parentCtx.CurrentGoFullPackagePath]
	if !ok {
		return false
	}

	return nodeOriginHashedName != currentVirtualComponent.HashedName
}

// logContextSwitchDetected logs debug information when a context switch is
// detected during traversal.
//
// Takes parentCtx (*AnalysisContext) which holds the current analysis state.
// Takes nodeOriginHashedName (string) which identifies where the node came
// from.
// Takes depth (int) which shows how deep the current traversal is.
func (cm *ContextManager) logContextSwitchDetected(parentCtx *AnalysisContext, nodeOriginHashedName string, depth int) {
	currentVirtualComponent, ok := cm.virtualModule.ComponentsByGoPath[parentCtx.CurrentGoFullPackagePath]
	if !ok {
		return
	}
	parentCtx.Logger.Trace("CONTEXT SWITCH DETECTED",
		logger_domain.Int(logKeyDepth, depth),
		logger_domain.String("current", currentVirtualComponent.HashedName),
		logger_domain.String("origin", nodeOriginHashedName))
	parentCtx.Logger.Trace("CONTEXT SWITCH",
		logger_domain.String("from_pkg", parentCtx.CurrentGoFullPackagePath),
		logger_domain.String("to_pkg", parentCtx.CurrentGoFullPackagePath),
		logger_domain.String("reason", "slotted content from different origin"))
}

// createSwitchedContext creates a new context for the switched component.
//
// Takes parentCtx (*AnalysisContext) which provides the parent context to
// build from.
// Takes switchedVirtualComp (*annotator_dto.VirtualComponent) which specifies
// the virtual component to switch to.
// Takes nodeOriginSFCPath (string) which identifies the source SFC file path.
//
// Returns *AnalysisContext which is the new context for the switched component.
func (*ContextManager) createSwitchedContext(
	parentCtx *AnalysisContext,
	switchedVirtualComp *annotator_dto.VirtualComponent,
	nodeOriginSFCPath string,
) *AnalysisContext {
	var switchedPackageName string
	if switchedVirtualComp.RewrittenScriptAST != nil {
		switchedPackageName = switchedVirtualComp.RewrittenScriptAST.Name.Name
	}

	return parentCtx.ForNewPackageContext(
		switchedVirtualComp.CanonicalGoPackagePath,
		switchedPackageName,
		switchedVirtualComp.VirtualGoFilePath,
		nodeOriginSFCPath,
	)
}

// populateSwitchedContext fills in the switched context based on whether the
// switch is to the active partial or to a different component (slotted
// content).
//
// Takes newCtx (*AnalysisContext) which is the new context to fill in.
// Takes switchedVirtualComp (*annotator_dto.VirtualComponent) which is the
// component being switched to.
// Takes activePInfo (*ast_domain.PartialInvocationInfo) which is the current
// partial invocation info, or nil if not in a partial.
// Takes parentCtx (*AnalysisContext) which provides the parent context for
// logging.
// Takes depth (int) which is the current depth for logging.
//
// Returns *ast_domain.PartialInvocationInfo which is the active partial info
// when switching to the same component, or nil when switching to a different
// component.
func (cm *ContextManager) populateSwitchedContext(
	newCtx *AnalysisContext,
	switchedVirtualComp *annotator_dto.VirtualComponent,
	activePInfo *ast_domain.PartialInvocationInfo,
	parentCtx *AnalysisContext,
	depth int,
) *ast_domain.PartialInvocationInfo {
	isSwitchingToActivePartial := activePInfo != nil && switchedVirtualComp.HashedName == activePInfo.PartialPackageName

	if isSwitchingToActivePartial {
		parentCtx.Logger.Trace("Decision: Populate new context as PARTIAL",
			logger_domain.Int(logKeyDepth, depth),
			logger_domain.String("partialPackage", activePInfo.PartialPackageName))
		populatePartialContext(newCtx, cm.typeResolver, switchedVirtualComp, activePInfo)
		return activePInfo
	}

	parentCtx.Logger.Trace("Decision: Populate new context as ROOT", logger_domain.Int(logKeyDepth, depth))
	PopulateRootContext(newCtx, cm.typeResolver, switchedVirtualComp)

	parentCtx.Logger.Trace("State Change: Active PartialInfo reset to nil for this branch", logger_domain.Int(logKeyDepth, depth))
	return nil
}

// defineItemVariable defines and enriches the item variable in a for loop.
// It enables hover type previews and go-to-definition for loop variables.
//
// Takes forExpr (*ast_domain.ForInExpression) which contains the for
// loop expression
// with the item variable to define.
// Takes parentCtx (*AnalysisContext) which provides the parent scope for type
// resolution.
// Takes loopCtx (*AnalysisContext) which is the loop scope where the variable
// will be defined.
// Takes collectionAnn (*ast_domain.GoGeneratorAnnotation) which holds the
// resolved type of the collection being iterated.
// Takes sfcSourcePath (string) which specifies the source file path for
// go-to-definition support.
func (cm *ContextManager) defineItemVariable(
	goCtx context.Context,
	forExpr *ast_domain.ForInExpression,
	parentCtx *AnalysisContext,
	loopCtx *AnalysisContext,
	collectionAnn *ast_domain.GoGeneratorAnnotation,
	sfcSourcePath string,
) {
	if forExpr.ItemVariable == nil {
		return
	}

	itemTypeInfo := cm.typeResolver.DetermineIterationItemType(goCtx, parentCtx, forExpr.Collection, collectionAnn.ResolvedType)
	loopCtx.Symbols.Define(Symbol{
		Name:                forExpr.ItemVariable.Name,
		CodeGenVarName:      forExpr.ItemVariable.Name,
		TypeInfo:            itemTypeInfo,
		SourceInvocationKey: "",
	})

	forExpr.ItemVariable.GoAnnotations = cm.createLoopVariableAnnotation(forExpr.ItemVariable, itemTypeInfo, sfcSourcePath)
}

// defineIndexVariable defines and enriches the index variable in a for loop.
// It enables hover type previews and go-to-definition for loop variables.
//
// Takes forExpr (*ast_domain.ForInExpression) which is the for-in expression
// containing the index variable to define.
// Takes parentCtx (*AnalysisContext) which provides the parent scope for type
// resolution.
// Takes loopCtx (*AnalysisContext) which is the loop scope where the index
// variable will be defined.
// Takes collectionAnn (*ast_domain.GoGeneratorAnnotation) which contains the
// resolved type of the collection being iterated.
// Takes sfcSourcePath (string) which is the source file path for location
// tracking.
func (cm *ContextManager) defineIndexVariable(
	forExpr *ast_domain.ForInExpression,
	parentCtx *AnalysisContext,
	loopCtx *AnalysisContext,
	collectionAnn *ast_domain.GoGeneratorAnnotation,
	sfcSourcePath string,
) {
	if forExpr.IndexVariable == nil {
		return
	}

	indexTypeInfo := cm.typeResolver.DetermineIterationIndexType(parentCtx, collectionAnn.ResolvedType)
	loopCtx.Symbols.Define(Symbol{
		Name:                forExpr.IndexVariable.Name,
		CodeGenVarName:      forExpr.IndexVariable.Name,
		TypeInfo:            indexTypeInfo,
		SourceInvocationKey: "",
	})

	forExpr.IndexVariable.GoAnnotations = cm.createLoopVariableAnnotation(forExpr.IndexVariable, indexTypeInfo, sfcSourcePath)
}

// createLoopVariableAnnotation creates an annotation for a loop variable.
//
// Takes variable (*ast_domain.Identifier) which is the loop variable to
// annotate.
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the resolved
// type of the variable.
// Takes sfcSourcePath (string) which is the path to the original SFC source
// file.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the annotation
// metadata for code generation.
func (*ContextManager) createLoopVariableAnnotation(
	variable *ast_domain.Identifier,
	typeInfo *ast_domain.ResolvedTypeInfo,
	sfcSourcePath string,
) *ast_domain.GoGeneratorAnnotation {
	return &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      &variable.Name,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType:            typeInfo,
		Symbol: &ast_domain.ResolvedSymbol{
			Name:                variable.Name,
			ReferenceLocation:   variable.RelativeLocation,
			DeclarationLocation: ast_domain.Location{},
		},
		PartialInfo:             nil,
		PropDataSource:          nil,
		OriginalSourcePath:      &sfcSourcePath,
		OriginalPackageAlias:    nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		StaticCollectionData:    nil,
		Srcset:                  nil,
		Stringability:           0,
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   false,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
}

// newContextManager creates a new ContextManager with the given type
// resolver and virtual module.
//
// Takes tr (*TypeResolver) which resolves types during context
// operations.
// Takes vm (*annotator_dto.VirtualModule) which provides the module
// for analysis.
//
// Returns *ContextManager which is ready for use.
func newContextManager(tr *TypeResolver, vm *annotator_dto.VirtualModule) *ContextManager {
	return &ContextManager{
		typeResolver:  tr,
		virtualModule: vm,
	}
}
