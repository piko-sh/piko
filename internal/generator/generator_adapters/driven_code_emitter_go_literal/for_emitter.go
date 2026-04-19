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

package driven_code_emitter_go_literal

import (
	"context"
	"fmt"
	goast "go/ast"
	"go/token"
	"slices"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

// ForEmitter provides an interface for emitting p-for loop directives.
// This enables mocking and testing of loop emission logic.
type ForEmitter interface {
	// emit generates Go AST statements for a template node.
	//
	// Takes node (*ast_domain.TemplateNode) which is the template node to process.
	// Takes parentSliceExpr (goast.Expr) which is the parent slice expression for
	// appending results.
	// Takes partialScopeID (string) which is the HashedName of the current partial
	// for CSS scoping.
	// Takes mainComponentScope (string) which is the HashedName of the main
	// component being generated.
	//
	// Returns []goast.Stmt which contains the generated statements.
	// Returns []*ast_domain.Diagnostic which contains any issues found during
	// emission.
	emit(
		ctx context.Context,
		node *ast_domain.TemplateNode,
		parentSliceExpr goast.Expr,
		partialScopeID string,
		mainComponentScope string,
	) ([]goast.Stmt, []*ast_domain.Diagnostic)

	// emitWithExtractedIterable emits a p-for loop using a pre-extracted iterable
	// variable. This avoids re-evaluating the collection expression and enables
	// accurate child slice capacity.
	//
	// Takes node (*ast_domain.TemplateNode) which is the loop node to emit.
	// Takes parentSliceExpr (goast.Expr) which is the parent slice expression.
	// Takes loopInfo (*LoopIterableInfo) which contains the extracted iterable.
	// Takes partialScopeID (string) which is the HashedName of the current partial
	// for CSS scoping.
	// Takes mainComponentScope (string) which is the HashedName of the main
	// component being generated.
	//
	// Returns []goast.Stmt which contains the generated loop statements.
	// Returns []*ast_domain.Diagnostic which contains any diagnostics produced.
	emitWithExtractedIterable(
		ctx context.Context,
		node *ast_domain.TemplateNode,
		parentSliceExpr goast.Expr,
		loopInfo *LoopIterableInfo,
		partialScopeID string,
		mainComponentScope string,
	) ([]goast.Stmt, []*ast_domain.Diagnostic)
}

// forEmitter generates Go code for the `p-for` directive. It handles
// iteration over slices, arrays, strings, and maps, applying optimisations
// where possible.
type forEmitter struct {
	// emitter is the parent emitter that provides access to annotation results.
	emitter *emitter

	// expressionEmitter converts expressions to Go code, stored
	// as interface for pooling.
	expressionEmitter ExpressionEmitter

	// astBuilder provides access to the main AST builder for creating code
	// within loop bodies.
	astBuilder AstBuilder

	// currentPartialScopeID is the hashed name of the current partial used for CSS
	// scoping. It is set during emit and used by generateLoopBody.
	currentPartialScopeID string

	// currentMainComponentScope is the hashed name of the main component being
	// built. It is set during emit and used in generateLoopBody.
	currentMainComponentScope string
}

var _ ForEmitter = (*forEmitter)(nil)

// emit is the primary entry point for this emitter. It is called by the
// astBuilder when it encounters a node with a `p-for` directive.
//
// Takes node (*ast_domain.TemplateNode) which contains the for directive to
// process.
// Takes parentSliceExpr (goast.Expr) which is the slice expression to append
// results to.
// Takes partialScopeID (string) which is the HashedName of the current partial
// for CSS scoping.
// Takes mainComponentScope (string) which is the HashedName of the main
// component being generated.
//
// Returns []goast.Stmt which contains the generated loop statements.
// Returns []*ast_domain.Diagnostic which contains any errors encountered
// during emission.
func (fe *forEmitter) emit(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	parentSliceExpr goast.Expr,
	partialScopeID string,
	mainComponentScope string,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	fe.currentPartialScopeID = partialScopeID
	fe.currentMainComponentScope = mainComponentScope
	forExpr, ok := node.DirFor.Expression.(*ast_domain.ForInExpression)
	if !ok {
		diagnostic := ast_domain.NewDiagnostic(
			ast_domain.Error,
			"Internal Emitter Error: p-for directive's expression was not a ForInExpr. The annotator may have failed.",
			node.DirFor.RawExpression,
			node.DirFor.Location,
			fe.emitter.computeRelativePath(*node.DirFor.GoAnnotations.OriginalSourcePath),
		)
		return nil, []*ast_domain.Diagnostic{diagnostic}
	}

	collGoExpr, prereqStmts, collDiags := fe.expressionEmitter.emit(forExpr.Collection)
	collAnn := getAnnotationFromExpression(forExpr.Collection)

	if collAnn != nil && collAnn.ResolvedType != nil {
		if _, isMap := collAnn.ResolvedType.TypeExpression.(*goast.MapType); isMap {
			mapCtx := &mapLoopContext{
				node:            node,
				forExpr:         forExpr,
				mapGoExpr:       collGoExpr,
				collAnn:         collAnn,
				parentSliceExpr: parentSliceExpr,
			}
			mapLoopStmts, mapDiags := fe.emitDeterministicMapLoopWithContext(ctx, mapCtx)
			prereqStmts = append(prereqStmts, fe.emitter.directiveMappingStmt(node, node.DirFor))
			ifStmt := &goast.IfStmt{
				Cond: &goast.BinaryExpr{X: collGoExpr, Op: token.NEQ, Y: cachedIdent("nil")},
				Body: &goast.BlockStmt{List: mapLoopStmts},
			}
			return append(prereqStmts, ifStmt), append(collDiags, mapDiags...)
		}
	}

	loopCtx := &rangeLoopContext{
		node:            node,
		forExpr:         forExpr,
		collGoExpr:      collGoExpr,
		collAnn:         collAnn,
		parentSliceExpr: parentSliceExpr,
		prereqStmts:     prereqStmts,
	}
	statements, diagnostics := fe.emitStandardRangeLoopWithContext(ctx, loopCtx)
	return statements, append(collDiags, diagnostics...)
}

// emitWithExtractedIterable emits a p-for loop using a pre-extracted iterable
// variable.
//
// This is called when the parent node has already extracted the collection to a
// variable for accurate child slice capacity calculation. Using the
// pre-extracted variable avoids double-evaluation of collection expressions
// (important for method calls with side effects) and redundant code emission.
//
// Takes node (*ast_domain.TemplateNode) which is the template node containing
// the p-for directive.
// Takes parentSliceExpr (goast.Expr) which is the expression for the parent
// slice capacity calculation.
// Takes loopInfo (*LoopIterableInfo) which contains the extracted iterable
// variable name and type information.
// Takes partialScopeID (string) which is the HashedName of the current partial
// for CSS scoping.
// Takes mainComponentScope (string) which is the HashedName of the main
// component being generated.
//
// Returns []goast.Stmt which contains the generated loop statements.
// Returns []*ast_domain.Diagnostic which contains any errors encountered during
// emission.
func (fe *forEmitter) emitWithExtractedIterable(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	parentSliceExpr goast.Expr,
	loopInfo *LoopIterableInfo,
	partialScopeID string,
	mainComponentScope string,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	fe.currentPartialScopeID = partialScopeID
	fe.currentMainComponentScope = mainComponentScope
	forExpr, ok := node.DirFor.Expression.(*ast_domain.ForInExpression)
	if !ok {
		diagnostic := ast_domain.NewDiagnostic(
			ast_domain.Error,
			"Internal Emitter Error: p-for directive's expression was not a ForInExpr.",
			node.DirFor.RawExpression,
			node.DirFor.Location,
			fe.emitter.computeRelativePath(*node.DirFor.GoAnnotations.OriginalSourcePath),
		)
		return nil, []*ast_domain.Diagnostic{diagnostic}
	}

	collGoExpr := cachedIdent(loopInfo.VarName)

	if loopInfo.CollectionAnn != nil && loopInfo.CollectionAnn.ResolvedType != nil {
		if _, isMap := loopInfo.CollectionAnn.ResolvedType.TypeExpression.(*goast.MapType); isMap {
			mapCtx := &mapLoopContext{
				node:            node,
				forExpr:         forExpr,
				mapGoExpr:       collGoExpr,
				collAnn:         loopInfo.CollectionAnn,
				parentSliceExpr: parentSliceExpr,
			}
			mapLoopStmts, mapDiags := fe.emitDeterministicMapLoopWithContext(ctx, mapCtx)
			dirStmt := fe.emitter.directiveMappingStmt(node, node.DirFor)
			if loopInfo.IsNillable {
				ifStmt := &goast.IfStmt{
					Cond: &goast.BinaryExpr{X: collGoExpr, Op: token.NEQ, Y: cachedIdent("nil")},
					Body: &goast.BlockStmt{List: mapLoopStmts},
				}
				return []goast.Stmt{dirStmt, ifStmt}, mapDiags
			}
			return append([]goast.Stmt{dirStmt}, mapLoopStmts...), mapDiags
		}
	}

	loopCtx := &rangeLoopContext{
		node:            node,
		forExpr:         forExpr,
		collGoExpr:      collGoExpr,
		collAnn:         loopInfo.CollectionAnn,
		parentSliceExpr: parentSliceExpr,
		prereqStmts:     nil,
	}
	return fe.emitStandardRangeLoopWithContext(ctx, loopCtx)
}

// rangeLoopContext holds the data needed to emit a standard range loop.
type rangeLoopContext struct {
	// node is the AST template node being processed for range loop emission.
	node *ast_domain.TemplateNode

	// forExpr is the for-in expression AST node being emitted.
	forExpr *ast_domain.ForInExpression

	// collGoExpr is the Go AST expression for the collection being iterated.
	collGoExpr goast.Expr

	// collAnn holds the annotation for the collection being ranged over.
	collAnn *ast_domain.GoGeneratorAnnotation

	// parentSliceExpr is the slice expression that holds the range target.
	parentSliceExpr goast.Expr

	// prereqStmts holds statements that must be added before the range loop.
	prereqStmts []goast.Stmt
}

// emitStandardRangeLoopWithContext builds the range loop statements.
//
// Takes ctx (context.Context) which carries cancellation and deadline signals
// for building the loop body.
// Takes loopCtx (*rangeLoopContext) which holds the loop settings and state.
//
// Returns []goast.Stmt which contains the generated loop statements.
// Returns []*ast_domain.Diagnostic which contains any issues found.
func (fe *forEmitter) emitStandardRangeLoopWithContext(ctx context.Context, loopCtx *rangeLoopContext) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	idxVarName, itemVarName := fe.determineLoopVarNames(loopCtx.node, loopCtx.forExpr)
	bodyStmts, bodyDiags := fe.buildLoopBody(ctx, loopCtx.node, loopCtx.forExpr, loopCtx.parentSliceExpr)

	bodyStmts, prereqStmts, combinedDiags := fe.wrapBodyWithConditional(
		loopCtx.node,
		bodyStmts,
		bodyDiags,
		loopCtx.prereqStmts,
	)

	prereqStmts = append(prereqStmts, fe.emitter.directiveMappingStmt(loopCtx.node, loopCtx.node.DirFor))

	rangeStmt := &goast.RangeStmt{
		Key:   cachedIdent(idxVarName),
		Value: cachedIdent(itemVarName),
		Tok:   token.DEFINE,
		X:     loopCtx.collGoExpr,
		Body:  &goast.BlockStmt{List: bodyStmts},
	}

	if fe.collectionNeedsNilCheck(loopCtx.collAnn) {
		ifStmt := &goast.IfStmt{
			Cond: &goast.BinaryExpr{X: loopCtx.collGoExpr, Op: token.NEQ, Y: cachedIdent("nil")},
			Body: &goast.BlockStmt{List: []goast.Stmt{rangeStmt}},
		}
		return append(prereqStmts, ifStmt), combinedDiags
	}

	return append(prereqStmts, rangeStmt), combinedDiags
}

// determineLoopVarNames determines the appropriate variable names for loop
// index and item.
//
// Takes node (*ast_domain.TemplateNode) which is the template subtree to check
// for variable usage.
// Takes forExpr (*ast_domain.ForInExpression) which contains the loop variable
// definitions.
//
// Returns idxVarName (string) which is the index variable name or blank
// identifier if unused.
// Returns itemVarName (string) which is the item variable name or blank
// identifier if unused.
func (*forEmitter) determineLoopVarNames(
	node *ast_domain.TemplateNode,
	forExpr *ast_domain.ForInExpression,
) (idxVarName string, itemVarName string) {
	isIndexUsed := forExpr.IndexVariable != nil && subtreeDependsOnLoopVars(node, &ast_domain.ForInExpression{
		IndexVariable:    forExpr.IndexVariable,
		ItemVariable:     nil,
		Collection:       nil,
		GoAnnotations:    nil,
		RelativeLocation: ast_domain.Location{},
		SourceLength:     0,
	})

	isItemUsed := forExpr.ItemVariable != nil && subtreeDependsOnLoopVars(node, &ast_domain.ForInExpression{
		IndexVariable:    nil,
		ItemVariable:     forExpr.ItemVariable,
		Collection:       nil,
		GoAnnotations:    nil,
		RelativeLocation: ast_domain.Location{},
		SourceLength:     0,
	})

	idxVarName = BlankIdentifier
	if isIndexUsed {
		idxVarName = forExpr.IndexVariable.Name
	}

	itemVarName = BlankIdentifier
	if isItemUsed {
		itemVarName = forExpr.ItemVariable.Name
	}

	return idxVarName, itemVarName
}

// wrapBodyWithConditional wraps loop body statements with a p-if condition
// if present.
//
// Takes node (*ast_domain.TemplateNode) which contains the template node with
// optional p-if directive.
// Takes bodyStmts ([]goast.Stmt) which contains the loop body statements to
// wrap.
// Takes bodyDiags ([]*ast_domain.Diagnostic) which contains diagnostics from
// body processing.
// Takes prereqStmts ([]goast.Stmt) which contains prerequisite statements to
// execute before the body.
//
// Returns wrappedBodyStmts ([]goast.Stmt) which contains the body statements,
// wrapped in an if statement when a p-if directive exists.
// Returns updatedPrereqStmts ([]goast.Stmt) which contains the prerequisite
// statements with any condition prerequisites appended.
// Returns allDiags ([]*ast_domain.Diagnostic) which contains all diagnostics
// including any from condition processing.
func (fe *forEmitter) wrapBodyWithConditional(
	node *ast_domain.TemplateNode,
	bodyStmts []goast.Stmt,
	bodyDiags []*ast_domain.Diagnostic,
	prereqStmts []goast.Stmt,
) (wrappedBodyStmts []goast.Stmt, updatedPrereqStmts []goast.Stmt, allDiags []*ast_domain.Diagnostic) {
	if node.DirIf == nil {
		return bodyStmts, prereqStmts, bodyDiags
	}

	ifCondGoExpr, ifPrereqs, ifDiags := fe.expressionEmitter.emit(node.DirIf.Expression)
	ifCondGoExpr = wrapInTruthinessCallIfNeeded(ifCondGoExpr, node.DirIf.Expression)
	prereqStmts = append(prereqStmts, ifPrereqs...)
	bodyDiags = append(bodyDiags, ifDiags...)

	ifStmt := &goast.IfStmt{
		Cond: ifCondGoExpr,
		Body: &goast.BlockStmt{List: bodyStmts},
	}

	dirStmt := fe.emitter.directiveMappingStmt(node, node.DirIf)
	return []goast.Stmt{dirStmt, ifStmt}, prereqStmts, bodyDiags
}

// collectionNeedsNilCheck determines if a collection requires a nil check
// before iteration.
//
// Takes collAnn (*ast_domain.GoGeneratorAnnotation) which specifies the
// collection annotation to check.
//
// Returns bool which is true when the collection type can be nil.
func (*forEmitter) collectionNeedsNilCheck(collAnn *ast_domain.GoGeneratorAnnotation) bool {
	return collAnn != nil && collAnn.ResolvedType != nil && isNillableType(collAnn.ResolvedType.TypeExpression)
}

// mapLoopContext holds the state needed to emit a map loop in Go code.
type mapLoopContext struct {
	// node is the template AST node for the current loop.
	node *ast_domain.TemplateNode

	// forExpr is the for-in expression AST node that holds the loop variables.
	forExpr *ast_domain.ForInExpression

	// mapGoExpr is the Go expression for the map being iterated over.
	mapGoExpr goast.Expr

	// collAnn holds the annotation for the collection being iterated.
	collAnn *ast_domain.GoGeneratorAnnotation

	// parentSliceExpr is the slice expression that holds the sorted map keys.
	parentSliceExpr goast.Expr
}

// emitDeterministicMapLoopWithContext creates the loop statements for
// iterating over a map in sorted key order.
//
// Takes ctx (context.Context) which carries cancellation and deadline signals
// for building the loop body.
// Takes mapCtx (*mapLoopContext) which holds the map loop settings.
//
// Returns []goast.Stmt which contains the generated loop statements.
// Returns []*ast_domain.Diagnostic which contains any issues found during
// code generation.
//
// Panics if mapCtx contains a type that is not a map.
func (fe *forEmitter) emitDeterministicMapLoopWithContext(ctx context.Context, mapCtx *mapLoopContext) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	mapType, ok := mapCtx.collAnn.ResolvedType.TypeExpression.(*goast.MapType)
	if !ok {
		panic(fmt.Sprintf("emitDeterministicMapLoop called with non-map type: %T", mapCtx.collAnn.ResolvedType.TypeExpression))
	}

	var statements []goast.Stmt
	statements = append(statements, fe.buildMapKeyExtraction(mapCtx.mapGoExpr, mapType.Key)...)
	bodyStmts, bodyDiags := fe.buildMapLoopBody(ctx, mapCtx.node, mapCtx.forExpr, mapCtx.mapGoExpr, mapCtx.parentSliceExpr)

	var allPrereqs []goast.Stmt
	if mapCtx.node.DirIf != nil {
		ifCondGoExpr, ifPrereqs, ifDiags := fe.expressionEmitter.emit(mapCtx.node.DirIf.Expression)
		ifCondGoExpr = wrapInTruthinessCallIfNeeded(ifCondGoExpr, mapCtx.node.DirIf.Expression)
		allPrereqs = append(allPrereqs, ifPrereqs...)
		bodyDiags = append(bodyDiags, ifDiags...)

		ifStmt := &goast.IfStmt{
			Cond: ifCondGoExpr,
			Body: &goast.BlockStmt{List: bodyStmts},
		}
		bodyStmts = []goast.Stmt{ifStmt}
	}

	isIndexUsed := mapCtx.forExpr.IndexVariable != nil && subtreeDependsOnLoopVars(mapCtx.node, &ast_domain.ForInExpression{
		IndexVariable:    mapCtx.forExpr.IndexVariable,
		ItemVariable:     nil,
		Collection:       nil,
		GoAnnotations:    nil,
		RelativeLocation: ast_domain.Location{},
		SourceLength:     0,
	})
	loopKeyVarName := BlankIdentifier
	if isIndexUsed {
		loopKeyVarName = mapCtx.forExpr.IndexVariable.Name
	}

	finalLoop := &goast.RangeStmt{
		Key:   cachedIdent(BlankIdentifier),
		Value: cachedIdent(loopKeyVarName),
		Tok:   token.DEFINE,
		Body:  &goast.BlockStmt{List: bodyStmts},
		X:     cachedIdent("sortedKeys"),
	}
	statements = append(allPrereqs, statements...)
	statements = append(statements, finalLoop)

	return statements, bodyDiags
}

// buildLoopBody generates the Go statements for the body of a for-loop. It
// detects if any partials are invoked within the loop body whose props depend
// on the loop's variables and generates the Render() calls for those partials
// inside the loop, ensuring the loop variables are in scope.
//
// Takes originalNode (*ast_domain.TemplateNode) which is the template node
// being processed.
// Takes forExpr (*ast_domain.ForInExpression) which contains the
// for-loop expression details.
// Takes parentSliceExpr (goast.Expr) which is the slice expression being
// iterated over.
//
// Returns []goast.Stmt which contains the generated loop body statements.
// Returns []*ast_domain.Diagnostic which contains any diagnostics from code
// generation.
func (fe *forEmitter) buildLoopBody(
	ctx context.Context,
	originalNode *ast_domain.TemplateNode,
	forExpr *ast_domain.ForInExpression,
	parentSliceExpr goast.Expr,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	partialStmts, partialDiags := fe.generateLoopDependentPartialCalls(originalNode)
	bodyStmts, bodyDiags := fe.generateLoopItemCode(ctx, originalNode, forExpr, parentSliceExpr)

	partialStmts = append(partialStmts, bodyStmts...)
	partialDiags = append(partialDiags, bodyDiags...)

	return partialStmts, partialDiags
}

// generateLoopDependentPartialCalls finds and generates render calls for
// loop-dependent partials.
//
// Takes originalNode (*ast_domain.TemplateNode) which is the template node
// containing the loop to search for partial invocations.
//
// Returns []goast.Stmt which contains the generated render call statements.
// Returns []*ast_domain.Diagnostic which contains any diagnostics from sorting
// or generating the partial calls.
func (fe *forEmitter) generateLoopDependentPartialCalls(
	originalNode *ast_domain.TemplateNode,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	allInvocations := fe.collectPartialInvocationsInLoop(originalNode)
	if len(allInvocations) == 0 {
		return nil, nil
	}

	sortedInvocations, sortDiags := fe.astBuilder.topologicallySortInvocations(
		allInvocations,
		fe.emitter.AnnotationResult.VirtualModule,
	)

	invocationsByKey := make(map[string]*annotator_dto.PartialInvocation, len(sortedInvocations))
	for _, inv := range sortedInvocations {
		invocationsByKey[inv.InvocationKey] = inv
	}

	var statements []goast.Stmt
	var diagnostics []*ast_domain.Diagnostic
	diagnostics = append(diagnostics, sortDiags...)

	for _, invocation := range sortedInvocations {
		visited := make(map[string]bool)
		if !isInvocationLoopDependentRecursive(invocation, invocationsByKey, visited) {
			continue
		}

		renderStmts, renderDiags := fe.generatePartialRenderCall(invocation)
		statements = append(statements, renderStmts...)
		diagnostics = append(diagnostics, renderDiags...)
	}

	return statements, diagnostics
}

// collectPartialInvocationsInLoop walks the node tree and collects all partial
// invocations.
//
// Takes originalNode (*ast_domain.TemplateNode) which is the root node to walk.
//
// Returns []*annotator_dto.PartialInvocation which contains all partial
// invocations found in the tree. Nested for loops are not searched, but their
// siblings are still processed.
func (fe *forEmitter) collectPartialInvocationsInLoop(
	originalNode *ast_domain.TemplateNode,
) []*annotator_dto.PartialInvocation {
	var allInvocations []*annotator_dto.PartialInvocation
	fe.collectPartialInvocationsRecursive(originalNode, originalNode, &allInvocations)
	return allInvocations
}

// collectPartialInvocationsRecursive walks the node tree to gather partial
// invocations. It skips nested loops but still visits their siblings.
//
// Takes node (*ast_domain.TemplateNode) which is the current node to process.
// Takes originalNode (*ast_domain.TemplateNode) which is the root loop node,
// used to tell it apart from nested loops.
// Takes allInvocations (*[]*annotator_dto.PartialInvocation) which gathers the
// found invocations.
func (fe *forEmitter) collectPartialInvocationsRecursive(
	node *ast_domain.TemplateNode,
	originalNode *ast_domain.TemplateNode,
	allInvocations *[]*annotator_dto.PartialInvocation,
) {
	if node == nil {
		return
	}

	if node != originalNode && node.DirFor != nil {
		return
	}

	if node.GoAnnotations != nil && node.GoAnnotations.PartialInfo != nil {
		if invocation := fe.findCanonicalInvocation(node.GoAnnotations.PartialInfo.InvocationKey); invocation != nil {
			*allInvocations = append(*allInvocations, invocation)
		}
	}

	for _, child := range node.Children {
		fe.collectPartialInvocationsRecursive(child, originalNode, allInvocations)
	}
}

// findCanonicalInvocation finds the matching invocation data from
// AnnotationResult.
//
// Takes invocationKey (string) which identifies the invocation to find.
//
// Returns *annotator_dto.PartialInvocation which is the matching invocation,
// or nil if not found.
func (fe *forEmitter) findCanonicalInvocation(invocationKey string) *annotator_dto.PartialInvocation {
	for _, inv := range fe.emitter.AnnotationResult.UniqueInvocations {
		if inv.InvocationKey == invocationKey {
			return inv
		}
	}
	return nil
}

// generatePartialRenderCall creates the render call for a partial invocation.
//
// Takes invocation (*annotator_dto.PartialInvocation) which provides the
// details of the partial to render.
//
// Returns []goast.Stmt which contains the generated statements for the call.
// Returns []*ast_domain.Diagnostic which contains any issues found during
// generation.
func (fe *forEmitter) generatePartialRenderCall(
	invocation *annotator_dto.PartialInvocation,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	pInfo := &ast_domain.PartialInvocationInfo{
		InvocationKey:        invocation.InvocationKey,
		PartialAlias:         invocation.PartialAlias,
		PartialPackageName:   invocation.PartialHashedName,
		RequestOverrides:     invocation.RequestOverrides,
		PassedProps:          invocation.PassedProps,
		InvokerPackageAlias:  invocation.InvokerHashedName,
		InvokerInvocationKey: invocation.InvokerInvocationKey,
		Location:             invocation.Location,
	}
	return fe.astBuilder.emitPartialRenderCall(pInfo, fe.emitter.AnnotationResult)
}

// generateLoopItemCode creates the code for a single loop item.
//
// Takes originalNode (*ast_domain.TemplateNode) which is the template node
// being processed.
// Takes forExpr (*ast_domain.ForInExpression) which contains the for-in expression
// to check.
// Takes parentSliceExpr (goast.Expr) which is the parent slice expression for
// adding results.
//
// Returns []goast.Stmt which contains the loop body statements.
// Returns []*ast_domain.Diagnostic which contains any errors found during code
// creation.
func (fe *forEmitter) generateLoopItemCode(
	ctx context.Context,
	originalNode *ast_domain.TemplateNode,
	forExpr *ast_domain.ForInExpression,
	parentSliceExpr goast.Expr,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	tempNode := fe.prepareLoopItemNode(originalNode)

	if fe.canHoistLoopBody(tempNode, forExpr) {
		return fe.generateHoistedLoopBody(ctx, originalNode, parentSliceExpr)
	}

	emitCtx := newNodeEmissionContext(ctx, nodeEmissionParams{
		Node:                  tempNode,
		ParentSliceExpression: parentSliceExpr,
		Index:                 0,
		Siblings:              []*ast_domain.TemplateNode{tempNode},
		IsRootNode:            false,
		PartialScopeID:        fe.currentPartialScopeID,
		MainComponentScope:    fe.currentMainComponentScope,
	})
	statements, _, diagnostics := fe.astBuilder.emitNode(emitCtx)
	return statements, diagnostics
}

// prepareLoopItemNode creates a clone of the node with loop directives removed.
//
// Takes originalNode (*ast_domain.TemplateNode) which is the node to clone.
//
// Returns *ast_domain.TemplateNode which is the cloned node without loop or
// conditional directives but with the original children preserved.
func (*forEmitter) prepareLoopItemNode(originalNode *ast_domain.TemplateNode) *ast_domain.TemplateNode {
	tempNode := originalNode.Clone()
	tempNode.DirFor = nil
	tempNode.DirIf = nil
	tempNode.Children = originalNode.Children
	return tempNode
}

// canHoistLoopBody determines if the loop body can be hoisted as a static node.
//
// Takes tempNode (*ast_domain.TemplateNode) which is the template node to check.
// Takes forExpr (*ast_domain.ForInExpression) which is the for loop
// expression context.
//
// Returns bool which is true if the body can be hoisted.
func (*forEmitter) canHoistLoopBody(
	tempNode *ast_domain.TemplateNode,
	forExpr *ast_domain.ForInExpression,
) bool {
	return tempNode.GoAnnotations != nil &&
		tempNode.GoAnnotations.IsStructurallyStatic &&
		!subtreeDependsOnLoopVars(tempNode, forExpr) &&
		!subtreeContainsForLoops(tempNode)
}

// generateHoistedLoopBody creates a loop body that uses a hoisted static node.
//
// Takes originalNode (*ast_domain.TemplateNode) which is the template node to
// hoist.
// Takes parentSliceExpr (goast.Expr) which is the slice to append results to.
//
// Returns []goast.Stmt which contains the loop body statements.
// Returns []*ast_domain.Diagnostic which contains any diagnostics from static
// node registration.
func (fe *forEmitter) generateHoistedLoopBody(
	ctx context.Context,
	originalNode *ast_domain.TemplateNode,
	parentSliceExpr goast.Expr,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	staticNodeForHoisting := originalNode.DeepClone()
	staticNodeForHoisting.DirFor = nil
	staticNodeForHoisting.DirIf = nil

	staticVarIdent, staticDiags := fe.emitter.staticEmitter.registerStaticNode(ctx, staticNodeForHoisting, fe.currentPartialScopeID)
	statements := []goast.Stmt{
		appendToSlice(parentSliceExpr, &goast.CallExpr{
			Fun: &goast.SelectorExpr{X: staticVarIdent, Sel: cachedIdent("DeepClone")},
		}),
	}

	return statements, staticDiags
}

// buildMapLoopBody creates the statements for the loop body when iterating
// over a map with sorted keys.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to process.
// Takes forExpr (*ast_domain.ForInExpression) which holds the loop
// variable details.
// Takes mapGoExpr (goast.Expr) which is the Go expression for the map.
// Takes parentSliceExpr (goast.Expr) which is the parent slice to append to.
//
// Returns []goast.Stmt which contains the loop body statements.
// Returns []*ast_domain.Diagnostic which contains any issues found.
func (fe *forEmitter) buildMapLoopBody(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	forExpr *ast_domain.ForInExpression,
	mapGoExpr goast.Expr,
	parentSliceExpr goast.Expr,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	var bodyStmts []goast.Stmt

	isItemUsed := forExpr.ItemVariable != nil && subtreeDependsOnLoopVars(node, &ast_domain.ForInExpression{
		IndexVariable:    nil,
		ItemVariable:     forExpr.ItemVariable,
		Collection:       nil,
		GoAnnotations:    nil,
		RelativeLocation: ast_domain.Location{},
		SourceLength:     0,
	})
	if isItemUsed {
		keyVarIdent := cachedIdent(BlankIdentifier)
		if forExpr.IndexVariable != nil {
			keyVarIdent = cachedIdent(forExpr.IndexVariable.Name)
		}
		bodyStmts = append(bodyStmts, defineAndAssign(forExpr.ItemVariable.Name, &goast.IndexExpr{
			X:     mapGoExpr,
			Index: keyVarIdent,
		}))
	}

	nodeCreationStmts, nodeDiags := fe.buildLoopBody(ctx, node, forExpr, parentSliceExpr)
	bodyStmts = append(bodyStmts, nodeCreationStmts...)

	return bodyStmts, nodeDiags
}

// buildMapKeyExtraction creates statements to extract and sort map keys.
//
// Takes mapGoExpr (goast.Expr) which is the map to get keys from.
// Takes keyType (goast.Expr) which is the type of the map keys.
//
// Returns []goast.Stmt which contains the key extraction and sorting
// statements. Sorting is only added when keys are strings.
func (*forEmitter) buildMapKeyExtraction(mapGoExpr goast.Expr, keyType goast.Expr) []goast.Stmt {
	sortedKeysVar := cachedIdent("sortedKeys")
	makeKeysStmt := defineAndAssign(sortedKeysVar.Name, &goast.CallExpr{
		Fun:  cachedIdent(MakeFuncName),
		Args: []goast.Expr{&goast.ArrayType{Elt: keyType}, intLit(0), &goast.CallExpr{Fun: cachedIdent("len"), Args: []goast.Expr{mapGoExpr}}},
	})
	extractKeysLoop := &goast.RangeStmt{
		Key:  cachedIdent("k"),
		Tok:  token.DEFINE,
		X:    mapGoExpr,
		Body: &goast.BlockStmt{List: []goast.Stmt{appendToSlice(sortedKeysVar, cachedIdent("k"))}},
	}
	statements := []goast.Stmt{makeKeysStmt, extractKeysLoop}
	if isStringType(keyType) {
		sortStmt := &goast.ExprStmt{
			X: &goast.CallExpr{
				Fun:  &goast.SelectorExpr{X: cachedIdent("sort"), Sel: cachedIdent("Strings")},
				Args: []goast.Expr{sortedKeysVar},
			},
		}
		statements = append(statements, sortStmt)
	}
	return statements
}

// newForEmitter creates a new emitter for p-for loops.
//
// Takes emitter (*emitter) which provides the base emitter functions.
// Takes expressionEmitter (ExpressionEmitter) which handles expression output.
// Takes astBuilder (AstBuilder) which builds AST nodes.
//
// Returns *forEmitter which is the configured loop emitter ready for use.
func newForEmitter(emitter *emitter, expressionEmitter ExpressionEmitter, astBuilder AstBuilder) *forEmitter {
	return &forEmitter{
		emitter:           emitter,
		expressionEmitter: expressionEmitter,
		astBuilder:        astBuilder,
	}
}

// isInvocationLoopDependentRecursive checks if an invocation depends on loop
// variables. This can be direct (through its props) or indirect (through its
// dependencies, such as a nested partial whose parent is inside a p-for loop).
//
// Takes invocation (*annotator_dto.PartialInvocation) which is the invocation
// to check.
// Takes invocationsByKey (map[string]*annotator_dto.PartialInvocation) which
// maps invocation keys to their invocations for looking up dependencies.
// Takes visited (map[string]bool) which tracks invocation keys that have
// already been checked to prevent infinite recursion in circular dependencies.
//
// Returns bool which is true if any passed prop depends on a loop variable, or
// if any dependency depends on a loop variable.
func isInvocationLoopDependentRecursive(
	invocation *annotator_dto.PartialInvocation,
	invocationsByKey map[string]*annotator_dto.PartialInvocation,
	visited map[string]bool,
) bool {
	if visited[invocation.InvocationKey] {
		return false
	}
	visited[invocation.InvocationKey] = true

	for _, propVal := range invocation.PassedProps {
		if propVal.IsLoopDependent {
			return true
		}
	}

	for _, depKey := range invocation.DependsOn {
		depInvocation, ok := invocationsByKey[depKey]
		if !ok {
			continue
		}
		if isInvocationLoopDependentRecursive(depInvocation, invocationsByKey, visited) {
			return true
		}
	}

	return false
}

// subtreeDependsOnLoopVars checks if a template node or any of its children
// use variables from a given for-in loop.
//
// Takes node (*ast_domain.TemplateNode) which is the root node to check.
// Takes forExpr (*ast_domain.ForInExpression) which defines the loop variables.
//
// Returns bool which is true if any part of the subtree uses the loop
// variables.
func subtreeDependsOnLoopVars(node *ast_domain.TemplateNode, forExpr *ast_domain.ForInExpression) bool {
	loopVars := make(map[string]bool)
	if forExpr.ItemVariable != nil {
		loopVars[forExpr.ItemVariable.Name] = true
	}
	if forExpr.IndexVariable != nil {
		loopVars[forExpr.IndexVariable.Name] = true
	}
	if len(loopVars) == 0 {
		return false
	}
	var checkNode func(*ast_domain.TemplateNode) bool
	checkNode = func(n *ast_domain.TemplateNode) bool {
		effectiveKey := getEffectiveKeyExpression(n)
		if effectiveKey != nil && expressionUsesVars(effectiveKey, loopVars) {
			return true
		}

		var depends bool
		visitNodeExpressions(n, func(expression ast_domain.Expression) {
			if depends {
				return
			}
			if expressionUsesVars(expression, loopVars) {
				depends = true
			}
		})
		if depends {
			return true
		}
		return slices.ContainsFunc(n.Children, checkNode)
	}
	result := checkNode(node)
	return result
}

// subtreeContainsForLoops checks if a node or any of its children contain a
// p-for directive.
//
// Used to stop hoisting of nodes that contain loops inside them, such as from
// expanded partials. Hoisting such nodes would cause problems with dynamic key
// expressions.
//
// Takes node (*ast_domain.TemplateNode) which is the root node to check.
//
// Returns bool which is true if a p-for directive exists in the subtree.
func subtreeContainsForLoops(node *ast_domain.TemplateNode) bool {
	if node == nil {
		return false
	}
	if node.DirFor != nil {
		return true
	}
	return slices.ContainsFunc(node.Children, subtreeContainsForLoops)
}

// visitNodeExpressions visits all expressions in a template node.
//
// It calls helpers to visit directives, dynamic content, events, and bindings.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to visit.
// Takes visit (func(...)) which is called for each expression found.
func visitNodeExpressions(node *ast_domain.TemplateNode, visit func(ast_domain.Expression)) {
	visitDirectiveExpressions(node, visit)
	visitDynamicContentExpressions(node, visit)
	visitEventExpressions(node, visit)
	visitBindExpressions(node, visit)
}

// visitDirectiveExpressions visits expressions from structural and display
// directives on a template node.
//
// Takes node (*ast_domain.TemplateNode) which contains the directive fields to
// check.
// Takes visit (func(...)) which is called for each non-nil directive
// expression found.
func visitDirectiveExpressions(node *ast_domain.TemplateNode, visit func(ast_domain.Expression)) {
	visitIfNotNil(node.DirIf, visit)
	visitIfNotNil(node.DirElseIf, visit)
	visitIfNotNil(node.DirFor, visit)
	visitIfNotNil(node.DirShow, visit)
	visitIfNotNil(node.DirModel, visit)
	visitIfNotNil(node.DirClass, visit)
	visitIfNotNil(node.DirStyle, visit)
	visitIfNotNil(node.DirText, visit)
	visitIfNotNil(node.DirHTML, visit)
	visitIfNotNil(node.DirKey, visit)

	effectiveKey := getEffectiveKeyExpression(node)
	if effectiveKey != nil {
		visit(effectiveKey)
	}
}

// visitDynamicContentExpressions visits each expression in the dynamic
// attributes and rich text parts of a template node.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to visit.
// Takes visit (func(...)) which is called for each expression found.
func visitDynamicContentExpressions(node *ast_domain.TemplateNode, visit func(ast_domain.Expression)) {
	for i := range node.DynamicAttributes {
		if node.DynamicAttributes[i].Expression != nil {
			visit(node.DynamicAttributes[i].Expression)
		}
	}

	for i := range node.RichText {
		part := &node.RichText[i]
		if !part.IsLiteral && part.Expression != nil {
			visit(part.Expression)
		}
	}
}

// visitEventExpressions visits expressions from event handlers, including
// both standard and custom events.
//
// Takes node (*ast_domain.TemplateNode) which contains the event handlers.
// Takes visit (func(...)) which is called for each expression found.
func visitEventExpressions(node *ast_domain.TemplateNode, visit func(ast_domain.Expression)) {
	visitEventsMap(node.OnEvents, visit)
	visitEventsMap(node.CustomEvents, visit)
}

// visitEventsMap iterates over a map of event handlers and calls the visit
// function for each expression found.
//
// Takes eventsMap (map[string][]ast_domain.Directive) which contains event
// handlers grouped by event name.
// Takes visit (func(...)) which is called for each expression that is not nil.
func visitEventsMap(eventsMap map[string][]ast_domain.Directive, visit func(ast_domain.Expression)) {
	for _, events := range eventsMap {
		for i := range events {
			if events[i].Expression != nil {
				visit(events[i].Expression)
			}
		}
	}
}

// visitBindExpressions visits expressions from bind directives.
//
// Takes node (*ast_domain.TemplateNode) which contains the bind directives to
// visit.
// Takes visit (func(...)) which is called for each bind expression that is not
// nil.
func visitBindExpressions(node *ast_domain.TemplateNode, visit func(ast_domain.Expression)) {
	for _, bind := range node.Binds {
		if bind != nil && bind.Expression != nil {
			visit(bind.Expression)
		}
	}
}

// visitIfNotNil calls the visit function with the directive's expression if
// both the directive and its expression are not nil.
//
// Takes directive (*ast_domain.Directive) which is the directive to check.
// Takes visit (func(...)) which is called with the expression if present.
func visitIfNotNil(directive *ast_domain.Directive, visit func(ast_domain.Expression)) {
	if directive != nil && directive.Expression != nil {
		visit(directive.Expression)
	}
}

// expressionUsesVars checks whether an expression uses any loop variables.
//
// Takes expression (ast_domain.Expression) which is the expression to check.
// Takes loopVars (map[string]bool) which holds the loop variable names to look
// for.
//
// Returns bool which is true if the expression contains any of the loop
// variables.
func expressionUsesVars(expression ast_domain.Expression, loopVars map[string]bool) bool {
	var found bool
	visitExpression(expression, func(e ast_domain.Expression) bool {
		if identifier, ok := e.(*ast_domain.Identifier); ok {
			if loopVars[identifier.Name] {
				found = true
				return false
			}
		}
		return !found
	})
	return found
}

// visitExpression walks an AST expression tree and calls the visitor function
// on each node.
//
// The visitor is called before visiting child nodes. When expr is nil or
// visitor returns false, the walk stops.
//
// Takes expression (ast_domain.Expression) which is the root expression to walk.
// Takes visitor (func(...)) which returns true to continue walking.
func visitExpression(expression ast_domain.Expression, visitor func(ast_domain.Expression) bool) {
	if expression == nil || !visitor(expression) {
		return
	}

	switch n := expression.(type) {
	case *ast_domain.MemberExpression:
		visitMemberExpression(n, visitor)
	case *ast_domain.IndexExpression:
		visitIndexExpression(n, visitor)
	case *ast_domain.UnaryExpression:
		visitExpression(n.Right, visitor)
	case *ast_domain.BinaryExpression:
		visitBinaryExpression(n, visitor)
	case *ast_domain.CallExpression:
		visitCallExpression(n, visitor)
	case *ast_domain.TemplateLiteral:
		visitTemplateLiteral(n, visitor)
	case *ast_domain.ObjectLiteral:
		visitObjectLiteral(n, visitor)
	case *ast_domain.ArrayLiteral:
		visitArrayLiteral(n, visitor)
	case *ast_domain.TernaryExpression:
		visitTernaryExpression(n, visitor)
	}
}

// visitMemberExpression visits the base and property of a member expression.
//
// Takes n (*ast_domain.MemberExpression) which is the member expression to visit.
// Takes visitor (func(...)) which is called for each child expression.
func visitMemberExpression(n *ast_domain.MemberExpression, visitor func(ast_domain.Expression) bool) {
	visitExpression(n.Base, visitor)
	visitExpression(n.Property, visitor)
}

// visitIndexExpression processes the base and index parts of an index
// expression.
//
// Takes n (*ast_domain.IndexExpression) which is the index expression to process.
// Takes visitor (func(...)) which is called for each sub-expression.
func visitIndexExpression(n *ast_domain.IndexExpression, visitor func(ast_domain.Expression) bool) {
	visitExpression(n.Base, visitor)
	visitExpression(n.Index, visitor)
}

// visitBinaryExpression visits the left and right sides of a binary expression.
//
// Takes n (*ast_domain.BinaryExpression) which is the binary expression to visit.
// Takes visitor (func(...)) which is called for each sub-expression.
func visitBinaryExpression(n *ast_domain.BinaryExpression, visitor func(ast_domain.Expression) bool) {
	visitExpression(n.Left, visitor)
	visitExpression(n.Right, visitor)
}

// visitCallExpression visits the callee and all arguments of a call
// expression.
//
// Takes n (*ast_domain.CallExpression) which is the call expression to visit.
// Takes visitor (func(...)) which is called for each child expression.
func visitCallExpression(n *ast_domain.CallExpression, visitor func(ast_domain.Expression) bool) {
	visitExpression(n.Callee, visitor)
	for _, argument := range n.Args {
		visitExpression(argument, visitor)
	}
}

// visitTemplateLiteral calls the visitor function for each non-literal part of
// a template literal.
//
// Takes n (*ast_domain.TemplateLiteral) which is the template literal to visit.
// Takes visitor (func(...)) which is called for each non-literal expression.
func visitTemplateLiteral(n *ast_domain.TemplateLiteral, visitor func(ast_domain.Expression) bool) {
	for _, part := range n.Parts {
		if !part.IsLiteral {
			visitExpression(part.Expression, visitor)
		}
	}
}

// visitObjectLiteral visits each value in an object literal.
//
// Takes n (*ast_domain.ObjectLiteral) which is the object literal to traverse.
// Takes visitor (func(...)) which is called for each expression in the object.
func visitObjectLiteral(n *ast_domain.ObjectLiteral, visitor func(ast_domain.Expression) bool) {
	for _, value := range n.Pairs {
		visitExpression(value, visitor)
	}
}

// visitArrayLiteral visits each element in an array literal.
//
// Takes n (*ast_domain.ArrayLiteral) which is the array literal to visit.
// Takes visitor (func(...)) which is called for each element expression.
func visitArrayLiteral(n *ast_domain.ArrayLiteral, visitor func(ast_domain.Expression) bool) {
	for _, el := range n.Elements {
		visitExpression(el, visitor)
	}
}

// visitTernaryExpression visits each part of a ternary expression.
//
// Takes n (*ast_domain.TernaryExpression) which is the ternary
// expression to visit.
// Takes visitor (func(...)) which is called for each sub-expression.
func visitTernaryExpression(n *ast_domain.TernaryExpression, visitor func(ast_domain.Expression) bool) {
	visitExpression(n.Condition, visitor)
	visitExpression(n.Consequent, visitor)
	visitExpression(n.Alternate, visitor)
}
