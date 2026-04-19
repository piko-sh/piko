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

// Implements the AST visitor pattern for the linking phase, walking the
// template tree with stateful context management. Handles context switches
// between components, manages scope transitions for partials and loops, and
// coordinates invocation processing.

import (
	"context"
	"fmt"
	"strings"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

// linkingSharedState tracks unique invocations and the order in which they
// were added during the linking phase.
type linkingSharedState struct {
	// uniqueInvocations maps canonical keys to their partial invocation data.
	uniqueInvocations map[string]*annotator_dto.PartialInvocation

	// invocationOrder holds canonical keys in the order they were added.
	invocationOrder []string
}

// linkingVisitor implements the ast.Visitor interface to perform stateful
// traversal. It manages the current AnalysisContext as it descends the AST,
// ensuring props are validated within the correct invoker's scope.
type linkingVisitor struct {
	// typeResolver resolves type information for expressions during linking.
	typeResolver *TypeResolver

	// virtualModule holds the virtual module being processed. It provides
	// lookup maps to find components by their Go package path or hash.
	virtualModule *annotator_dto.VirtualModule

	// diagnostics collects lint issues found while walking the AST.
	diagnostics *[]*ast_domain.Diagnostic

	// ctx holds the current analysis context for linking operations.
	ctx *AnalysisContext

	// state holds shared data that all child visitors can read and change.
	state *linkingSharedState

	// currentInvocationKey tracks the canonical key of the current partial; empty
	// at page level. Used to differentiate nested invocations with identical
	// expressions but different parent instances.
	currentInvocationKey string

	// depth tracks how deep the current node is in the AST tree.
	depth int
}

// Enter visits a node before its children are visited. It is the core of the
// stateful analysis, responsible for context switching and scope management.
//
// Takes node (*ast_domain.TemplateNode) which is the node to visit.
//
// Returns ast_domain.Visitor which is the visitor for the node's children.
// Returns error when partial invocation handling fails.
func (v *linkingVisitor) Enter(ctx context.Context, node *ast_domain.TemplateNode) (ast_domain.Visitor, error) {
	if node == nil {
		return nil, nil
	}

	_, l := logger_domain.From(ctx, log)
	traceEnabled := l.Enabled(logger_domain.LevelTrace)
	indent := ""
	if traceEnabled {
		indent = strings.Repeat("  ", v.depth)
		l.Trace("[Linker.Enter] Visiting node", logger_domain.Int(logKeyDepth, v.depth), logger_domain.String(logKeyTag, node.TagName), logger_domain.String("pkg", v.ctx.CurrentGoFullPackagePath))
		logAnalysisContext(v.ctx, indent+"  - Incoming Context")
	}

	ctxForThisNode := v.handleSlottedContentContextSwitch(ctx, node)
	ctxForChildren := ctxForThisNode
	childInvocationKey := v.currentInvocationKey

	if v.isPartialInvocation(node) {
		var err error
		ctxForChildren, childInvocationKey, err = v.handlePartialInvocation(ctx, node, ctxForThisNode, indent)
		if err != nil {
			return nil, fmt.Errorf("handling partial invocation for tag %q: %w", node.TagName, err)
		}
	}

	ctxForChildren = v.handleForDirective(ctx, node, ctxForThisNode, ctxForChildren, indent)

	l.Trace("[Linker.Exit] Descending to children", logger_domain.Int(logKeyDepth, v.depth), logger_domain.String("pkg", ctxForChildren.CurrentGoFullPackagePath))
	return v.newVisitorForChild(ctxForChildren, childInvocationKey), nil
}

// Exit is called after a node and all its children have been visited.
//
// Returns error when processing fails; always returns nil.
func (*linkingVisitor) Exit(_ context.Context, _ *ast_domain.TemplateNode) error {
	return nil
}

// handleSlottedContentContextSwitch checks if a node is slotted content and
// switches context if needed.
//
// For partial invocation nodes, the context is determined by
// PartialInfo.InvokerPackageAlias (the component that invoked the partial), not
// OriginalPackageAlias (the partial's template). This means prop bindings like
// :prop="state.Field" resolve against the invoker's state.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check
// for slotted content annotations.
//
// Returns *AnalysisContext which is either a new context for the appropriate
// package, or the current context if no switch is needed.
func (v *linkingVisitor) handleSlottedContentContextSwitch(ctx context.Context, node *ast_domain.TemplateNode) *AnalysisContext {
	if node.GoAnnotations == nil {
		return v.ctx
	}

	if node.GoAnnotations.PartialInfo != nil {
		return v.switchToInvokerContext(ctx, node)
	}

	if node.GoAnnotations.OriginalPackageAlias == nil {
		return v.ctx
	}

	return v.switchToOriginContext(ctx, node, *node.GoAnnotations.OriginalPackageAlias)
}

// switchToInvokerContext switches to the context of the component that called
// this partial, based on the invoker package alias in PartialInfo.
//
// Takes node (*ast_domain.TemplateNode) which provides the partial call details
// including the invoker package alias.
//
// Returns *AnalysisContext which is either a new context for the invoker
// component or the current context if no switch is needed.
func (v *linkingVisitor) switchToInvokerContext(ctx context.Context, node *ast_domain.TemplateNode) *AnalysisContext {
	invokerHash := node.GoAnnotations.PartialInfo.InvokerPackageAlias
	currentComp, ok := v.virtualModule.ComponentsByGoPath[v.ctx.CurrentGoFullPackagePath]
	if !ok {
		return v.ctx
	}

	if invokerHash == currentComp.HashedName {
		return v.ctx
	}

	invokerComp, compOk := v.virtualModule.ComponentsByHash[invokerHash]
	if !compOk {
		return v.ctx
	}

	_, l := logger_domain.From(ctx, log)
	l.Trace("[Linker] PARTIAL INVOCATION - switching to invoker context",
		logger_domain.String("from", currentComp.HashedName),
		logger_domain.String("toInvoker", invokerHash))

	invokerPackageName := ""
	if invokerComp.RewrittenScriptAST != nil {
		invokerPackageName = invokerComp.RewrittenScriptAST.Name.Name
	}

	invokerSourcePath := ""
	if invokerComp.Source != nil {
		invokerSourcePath = invokerComp.Source.SourcePath
	}

	ctxForThisNode := v.ctx.ForNewPackageContext(
		invokerComp.CanonicalGoPackagePath,
		invokerPackageName,
		invokerComp.VirtualGoFilePath,
		invokerSourcePath,
	)
	populateContextForLinking(ctxForThisNode, v.typeResolver, invokerComp)

	return ctxForThisNode
}

// switchToOriginContext switches to the context of the original source
// component for slotted content, based on OriginalPackageAlias.
//
// Takes node (*ast_domain.TemplateNode) which is the template node being
// processed.
// Takes nodeOriginHash (string) which identifies the original component.
//
// Returns *AnalysisContext which is the switched context, or the current
// context if no switch is needed.
func (v *linkingVisitor) switchToOriginContext(ctx context.Context, node *ast_domain.TemplateNode, nodeOriginHash string) *AnalysisContext {
	currentComp, ok := v.virtualModule.ComponentsByGoPath[v.ctx.CurrentGoFullPackagePath]
	if !ok {
		return v.ctx
	}

	if nodeOriginHash == currentComp.HashedName {
		return v.ctx
	}

	switchedComp, compOk := v.virtualModule.ComponentsByHash[nodeOriginHash]
	if !compOk {
		return v.ctx
	}

	_, l := logger_domain.From(ctx, log)
	l.Trace("[Linker] SLOTTED CONTENT DETECTED - switching context",
		logger_domain.String("from", currentComp.HashedName),
		logger_domain.String("to", nodeOriginHash))

	switchedPackageName := ""
	if switchedComp.RewrittenScriptAST != nil {
		switchedPackageName = switchedComp.RewrittenScriptAST.Name.Name
	}

	ctxForThisNode := v.ctx.ForNewPackageContext(
		switchedComp.CanonicalGoPackagePath,
		switchedPackageName,
		switchedComp.VirtualGoFilePath,
		*node.GoAnnotations.OriginalSourcePath,
	)
	populateContextForLinking(ctxForThisNode, v.typeResolver, switchedComp)

	return ctxForThisNode
}

// isPartialInvocation checks if a node represents a partial invocation.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true if the node has partial invocation annotations.
func (*linkingVisitor) isPartialInvocation(node *ast_domain.TemplateNode) bool {
	return node.GoAnnotations != nil && node.GoAnnotations.PartialInfo != nil
}

// handlePartialInvocation processes a partial invocation node and returns the
// context for its children. It also returns the canonical invocation key for
// this partial so it can be passed to nested partials.
//
// Takes node (*ast_domain.TemplateNode) which is the partial invocation node
// to process.
// Takes ctxForThisNode (*AnalysisContext) which provides the analysis context
// for the current node.
// Takes indent (string) which specifies the indentation for logging output.
//
// Returns *AnalysisContext which is the context for the partial's children.
// Returns string which is the canonical invocation key for this partial.
// Returns error when linking or context creation fails.
func (v *linkingVisitor) handlePartialInvocation(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	ctxForThisNode *AnalysisContext,
	indent string,
) (*AnalysisContext, string, error) {
	partialInfo := node.GoAnnotations.PartialInfo
	_, l := logger_domain.From(ctx, log)
	l.Trace("Node is a partial invocation", logger_domain.Int(logKeyDepth, v.depth), logger_domain.String("partialAlias", partialInfo.PartialAlias))

	partialInfo.InvokerInvocationKey = v.currentInvocationKey

	linkerCtx := v.prepareLinkerContext(ctx, node, ctxForThisNode, partialInfo)

	l.Trace("Linking props using context", logger_domain.Int(logKeyDepth, v.depth), logger_domain.String("pkg", linkerCtx.CurrentGoFullPackagePath))
	finalised, err := v.processPartialLinking(ctx, partialInfo, linkerCtx)
	if err != nil {
		return nil, "", err
	}

	v.storeUniqueInvocation(ctx, node, partialInfo, finalised)

	l.Trace("Context switch for children to partial's scope", logger_domain.Int(logKeyDepth, v.depth), logger_domain.String("partialPackage", partialInfo.PartialPackageName))
	ctxForChildren, err := v.createContextForNode(ctx, node, partialInfo)
	if err != nil {
		return nil, "", err
	}
	logAnalysisContext(ctxForChildren, indent+"  - New Context for Children (Partial's Context)")

	return ctxForChildren, finalised.canonicalKey, nil
}

// prepareLinkerContext creates the appropriate context for partial
// linking using ctxForThisNode, which has already been processed by
// handleSlottedContentContextSwitch.
//
// CRITICAL: For slotted content, ctxForThisNode is switched to the
// origin component's context. For non-slotted content, ctxForThisNode
// is the current traversal context (the invoker's context). This means
// prop bindings like :prop="state.Foo" are resolved against the
// invoker's context, not the current wrapper component's context (e.g.,
// a layout with NoResponse).
//
// Takes node (*ast_domain.TemplateNode) which is the template node being
// linked, checked for a p-for directive on the same element.
// Takes ctxForThisNode (*AnalysisContext) which is the already-switched
// context for the current node.
//
// Returns *AnalysisContext which is the context to use for linking,
// possibly augmented with loop variables if a p-for directive is
// present on the same node.
func (v *linkingVisitor) prepareLinkerContext(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	ctxForThisNode *AnalysisContext,
	_ *ast_domain.PartialInvocationInfo,
) *AnalysisContext {
	linkerCtx := ctxForThisNode
	if node.DirFor == nil {
		return linkerCtx
	}

	forExpr, ok := node.DirFor.Expression.(*ast_domain.ForInExpression)
	if !ok {
		return linkerCtx
	}

	_, l := logger_domain.From(ctx, log)
	l.Trace("SPECIAL CASE: p-for and partial on same node. Creating temp context for linking.", logger_domain.Int(logKeyDepth, v.depth))
	nestedCtx := linkerCtx.ForChildScope()
	collectionAnn := v.typeResolver.Resolve(ctx, linkerCtx, forExpr.Collection, node.DirFor.Location)

	if forExpr.ItemVariable != nil {
		itemTypeInfo := v.typeResolver.DetermineIterationItemType(ctx, linkerCtx, forExpr.Collection, collectionAnn.ResolvedType)
		nestedCtx.Symbols.Define(Symbol{Name: forExpr.ItemVariable.Name, CodeGenVarName: forExpr.ItemVariable.Name, TypeInfo: itemTypeInfo, SourceInvocationKey: ""})
	}
	if forExpr.IndexVariable != nil {
		indexTypeInfo := v.typeResolver.DetermineIterationIndexType(linkerCtx, collectionAnn.ResolvedType)
		nestedCtx.Symbols.Define(Symbol{Name: forExpr.IndexVariable.Name, CodeGenVarName: forExpr.IndexVariable.Name, TypeInfo: indexTypeInfo, SourceInvocationKey: ""})
	}

	return nestedCtx
}

// processPartialLinking converts partial invocation data to its complete form.
//
// Takes partialInfo (*ast_domain.PartialInvocationInfo) which holds the partial
// invocation details to link.
// Takes linkerCtx (*AnalysisContext) which provides the linking context.
//
// Returns *finalisedInvocationData which holds the completed linking result.
// Returns error when the linker cannot be obtained or processing fails.
func (v *linkingVisitor) processPartialLinking(
	ctx context.Context,
	partialInfo *ast_domain.PartialInvocationInfo,
	linkerCtx *AnalysisContext,
) (*finalisedInvocationData, error) {
	linker, err := getInvocationLinker(partialInfo, v.typeResolver, v.virtualModule, linkerCtx)
	if err != nil {
		return nil, fmt.Errorf("getting invocation linker for partial %q: %w", partialInfo.PartialAlias, err)
	}
	finalised, err := linker.process(ctx)
	putInvocationLinker(linker)
	if err != nil {
		return nil, fmt.Errorf("processing partial linking for %q: %w", partialInfo.PartialAlias, err)
	}
	_, l := logger_domain.From(ctx, log)
	l.Trace("Linking complete", logger_domain.Int(logKeyDepth, v.depth), logger_domain.String("canonicalKey", finalised.canonicalKey))
	return finalised, nil
}

// storeUniqueInvocation saves a new invocation if one with the same key does
// not already exist.
//
// Takes node (*ast_domain.TemplateNode) which provides the location data.
// Takes partialInfo (*ast_domain.PartialInvocationInfo) which receives the key and
// provides partial metadata.
// Takes finalised (*finalisedInvocationData) which contains the resolved
// invocation details to store.
func (v *linkingVisitor) storeUniqueInvocation(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	partialInfo *ast_domain.PartialInvocationInfo,
	finalised *finalisedInvocationData,
) {
	partialInfo.InvocationKey = finalised.canonicalKey

	partialInfo.PassedProps = finalised.canonicalProps

	if _, exists := v.state.uniqueInvocations[finalised.canonicalKey]; exists {
		return
	}

	_, l := logger_domain.From(ctx, log)
	l.Trace("New unique invocation discovered. Storing.", logger_domain.Int(logKeyDepth, v.depth))
	invocation := &annotator_dto.PartialInvocation{
		InvocationKey:        finalised.canonicalKey,
		PartialAlias:         partialInfo.PartialAlias,
		PartialHashedName:    partialInfo.PartialPackageName,
		PassedProps:          finalised.canonicalProps,
		RequestOverrides:     partialInfo.RequestOverrides,
		InvokerHashedName:    partialInfo.InvokerPackageAlias,
		InvokerInvocationKey: partialInfo.InvokerInvocationKey,
		DependsOn:            finalised.dependsOn,
		Location:             node.Location,
	}
	v.state.uniqueInvocations[finalised.canonicalKey] = invocation
	v.state.invocationOrder = append(v.state.invocationOrder, finalised.canonicalKey)
}

// handleForDirective processes a p-for directive and creates a nested scope
// for child nodes.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to process.
// Takes ctxForThisNode (*AnalysisContext) which provides the context for
// resolving the collection expression.
// Takes ctxForChildren (*AnalysisContext) which is the base context for child
// nodes.
// Takes indent (string) which sets the logging indent level.
//
// Returns *AnalysisContext which is a nested scope with loop variables defined,
// or the original child context if no p-for directive is present.
func (v *linkingVisitor) handleForDirective(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	ctxForThisNode *AnalysisContext,
	ctxForChildren *AnalysisContext,
	indent string,
) *AnalysisContext {
	if node.DirFor == nil {
		return ctxForChildren
	}

	forExpr, ok := node.DirFor.Expression.(*ast_domain.ForInExpression)
	if !ok {
		return ctxForChildren
	}

	_, l := logger_domain.From(ctx, log)
	l.Trace("Found p-for. Creating nested scope for children.", logger_domain.Int(logKeyDepth, v.depth))
	collectionAnn := v.typeResolver.Resolve(ctx, ctxForThisNode, forExpr.Collection, node.DirFor.Location)

	nestedCtx := ctxForChildren.ForChildScope()
	if forExpr.ItemVariable != nil {
		itemTypeInfo := v.typeResolver.DetermineIterationItemType(ctx, nestedCtx, forExpr.Collection, collectionAnn.ResolvedType)
		nestedCtx.Symbols.Define(Symbol{Name: forExpr.ItemVariable.Name, CodeGenVarName: forExpr.ItemVariable.Name, TypeInfo: itemTypeInfo, SourceInvocationKey: ""})
	}
	if forExpr.IndexVariable != nil {
		indexTypeInfo := v.typeResolver.DetermineIterationIndexType(nestedCtx, collectionAnn.ResolvedType)
		nestedCtx.Symbols.Define(Symbol{Name: forExpr.IndexVariable.Name, CodeGenVarName: forExpr.IndexVariable.Name, TypeInfo: indexTypeInfo, SourceInvocationKey: ""})
	}

	logAnalysisContext(nestedCtx, indent+"  - Final Context for Children (with loop vars)")
	return nestedCtx
}

// newVisitorForChild creates a new visitor for a child scope, inheriting depth
// for logging.
//
// Takes newCtx (*AnalysisContext) which provides the analysis context for the
// child scope.
// Takes invocationKey (string) which is the canonical key of the current
// partial invocation, or empty if not inside one.
//
// Returns ast_domain.Visitor which is the child visitor ready for traversal.
func (v *linkingVisitor) newVisitorForChild(newCtx *AnalysisContext, invocationKey string) ast_domain.Visitor {
	return &linkingVisitor{
		typeResolver:         v.typeResolver,
		virtualModule:        v.virtualModule,
		diagnostics:          v.diagnostics,
		ctx:                  newCtx,
		depth:                v.depth + 1,
		state:                v.state,
		currentInvocationKey: invocationKey,
	}
}

// createContextForNode builds the correct AnalysisContext for a given node
// based on its origin stamp from the expander.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to create
// context for.
// Takes partialInfo (*ast_domain.PartialInvocationInfo) which provides invocation
// specific variable names for SSR inlining, or nil for generic names used in
// the partial's own generated code.
//
// Returns *AnalysisContext which is the configured context for the node.
// Returns error when the virtual component for the node's hash cannot be found.
func (v *linkingVisitor) createContextForNode(ctx context.Context, node *ast_domain.TemplateNode, partialInfo *ast_domain.PartialInvocationInfo) (*AnalysisContext, error) {
	_, l := logger_domain.From(ctx, log)
	if node.GoAnnotations == nil || node.GoAnnotations.OriginalPackageAlias == nil || node.GoAnnotations.OriginalSourcePath == nil {
		l.Warn("Node is missing origin annotations; falling back to parent context.",
			logger_domain.String(logKeyTag, node.TagName), logger_domain.Int("line", node.Location.Line))
		return v.ctx, nil
	}

	hashedName := *node.GoAnnotations.OriginalPackageAlias
	sfcSourcePath := *node.GoAnnotations.OriginalSourcePath
	vc, ok := v.virtualModule.ComponentsByHash[hashedName]
	if !ok {
		return nil, fmt.Errorf("internal consistency error: could not find virtual component for hash '%s'", hashedName)
	}

	newCtx := NewRootAnalysisContext(
		v.diagnostics,
		vc.CanonicalGoPackagePath,
		vc.RewrittenScriptAST.Name.Name,
		vc.VirtualGoFilePath,
		sfcSourcePath,
	)
	newCtx.Logger = l

	if partialInfo != nil {
		populatePartialContext(newCtx, v.typeResolver, vc, partialInfo)
	} else {
		populateContextForLinking(newCtx, v.typeResolver, vc)
	}

	return newCtx, nil
}
