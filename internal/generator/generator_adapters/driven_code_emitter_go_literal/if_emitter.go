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
	goast "go/ast"
	"go/token"
	"slices"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
)

// IfEmitter provides methods for emitting conditional directive chains
// (p-if, p-else-if, p-else). It allows mocking and testing of conditional
// logic emission.
type IfEmitter interface {
	// emitChain emits Go statements for a chain of template nodes.
	//
	// Takes startNode (*ast_domain.TemplateNode) which is the first node in the
	// chain.
	// Takes siblings ([]*ast_domain.TemplateNode) which contains all sibling nodes
	// at this level.
	// Takes currentNodeIndex (int) which is the position of startNode in siblings.
	// Takes parentSliceExpr (goast.Expr) which is the expression to append output
	// to.
	// Takes partialScopeID (string) which is the HashedName of the current partial
	// for CSS scoping.
	// Takes mainComponentScope (string) which is the HashedName of the main
	// component being generated.
	//
	// Returns []goast.Stmt which contains the generated Go statements.
	// Returns int which is the number of siblings consumed by the chain.
	// Returns []*ast_domain.Diagnostic which contains any errors or warnings.
	emitChain(
		ctx context.Context,
		startNode *ast_domain.TemplateNode,
		siblings []*ast_domain.TemplateNode,
		currentNodeIndex int,
		parentSliceExpr goast.Expr,
		partialScopeID string,
		mainComponentScope string,
	) ([]goast.Stmt, int, []*ast_domain.Diagnostic)
}

// ifEmitter generates Go code for conditional directive chains such as p-if,
// p-else-if, and p-else. It implements the IfEmitter interface.
type ifEmitter struct {
	// emitter provides helper methods for building Go AST nodes.
	emitter *emitter

	// expressionEmitter converts template expressions into Go AST nodes.
	expressionEmitter ExpressionEmitter

	// astBuilder is the main code builder used to create nodes inside
	// conditional block bodies.
	astBuilder AstBuilder

	// currentPartialScopeID is the HashedName of the current partial for CSS
	// scoping, set during emitChain to be used by buildConditionalBody.
	currentPartialScopeID string

	// currentMainComponentScope is the hashed name of the main component being
	// built. It is set during emitChain and used in buildConditionalBody.
	currentMainComponentScope string
}

var _ IfEmitter = (*ifEmitter)(nil)

// emitChain is the primary entry point for this emitter, called by the
// astBuilder when it encounters a node with a `p-if` directive. It generates
// a complete `if / else if / else` block by consuming the starting node and
// any subsequent, validly chained sibling nodes.
//
// Takes startNode (*ast_domain.TemplateNode) which is the node containing
// the p-if directive that starts the chain.
// Takes siblings ([]*ast_domain.TemplateNode) which contains all sibling
// nodes to scan for chained else-if and else directives.
// Takes currentNodeIndex (int) which is the index of startNode in siblings.
// Takes parentSliceExpr (goast.Expr) which is the parent slice expression
// for generating append statements.
// Takes partialScopeID (string) which is the HashedName of the current partial
// for CSS scoping.
// Takes mainComponentScope (string) which is the HashedName of the main
// component being generated.
//
// Returns []goast.Stmt which contains the prerequisite statements and the
// complete if statement chain.
// Returns int which is the number of sibling nodes consumed by this chain.
// Returns []*ast_domain.Diagnostic which contains any diagnostics generated
// during emission.
func (ie *ifEmitter) emitChain(
	ctx context.Context,
	startNode *ast_domain.TemplateNode,
	siblings []*ast_domain.TemplateNode,
	currentNodeIndex int,
	parentSliceExpr goast.Expr,
	partialScopeID string,
	mainComponentScope string,
) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {
	ie.currentPartialScopeID = partialScopeID
	ie.currentMainComponentScope = mainComponentScope
	ifStmt, prereqStmts, nodesInChain, ifDiags := ie.buildIfBlock(ctx, startNode, parentSliceExpr)
	if ifStmt == nil {
		return prereqStmts, 1, ifDiags
	}

	allDiags := ifDiags
	currentIfStmt := ifStmt
	nodesConsumed := nodesInChain

	for i := currentNodeIndex + nodesConsumed; i < len(siblings); i++ {
		sibling := siblings[i]

		if sibling.NodeType != ast_domain.NodeElement {
			if !isWhitespaceOrComment(sibling) {
				break
			}
			nodesConsumed++
			continue
		}

		if isChainedElseIf(sibling, startNode) {
			elseIfBlock, elseIfPrereqs, _, elseIfDiags := ie.buildElseIfBlock(ctx, sibling, parentSliceExpr)
			prereqStmts = append(prereqStmts, elseIfPrereqs...)
			allDiags = append(allDiags, elseIfDiags...)
			currentIfStmt.Else = elseIfBlock
			currentIfStmt = elseIfBlock
			nodesConsumed++
			continue
		}

		if isChainedElse(sibling, startNode) {
			elseBlock, elsePrereqs, _, elseDiags := ie.buildElseBlock(ctx, sibling, parentSliceExpr)
			prereqStmts = append(prereqStmts, elsePrereqs...)
			allDiags = append(allDiags, elseDiags...)
			currentIfStmt.Else = elseBlock
			nodesConsumed++
			break
		}
		break
	}

	prereqStmts = append(prereqStmts, ie.emitter.directiveMappingStmt(startNode, startNode.DirIf), ifStmt)
	return prereqStmts, nodesConsumed, allDiags
}

// buildIfBlock creates the code for the first if statement in a chain.
//
// Takes node (*ast_domain.TemplateNode) which holds the template node with the
// if directive to process.
// Takes parentSliceExpr (goast.Expr) which is the parent slice expression for
// appending created statements.
//
// Returns *goast.IfStmt which is the created if statement, or nil if no
// directive exists.
// Returns []goast.Stmt which holds extra statements created during processing.
// Returns int which shows the number of nodes used by this block.
// Returns []*ast_domain.Diagnostic which holds any diagnostics made during
// code creation.
func (ie *ifEmitter) buildIfBlock(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	parentSliceExpr goast.Expr,
) (*goast.IfStmt, []goast.Stmt, int, []*ast_domain.Diagnostic) {
	if node.DirIf == nil {
		return nil, nil, 0, nil
	}
	return ie.buildConditionalIfStatement(ctx, node, parentSliceExpr, node.DirIf.Expression, func(n *ast_domain.TemplateNode) { n.DirIf = nil })
}

// buildElseIfBlock builds an if statement for use in an else branch.
//
// Takes node (*ast_domain.TemplateNode) which is the template node with the
// else-if directive.
// Takes parentSliceExpr (goast.Expr) which is the parent slice expression for
// context.
//
// Returns *goast.IfStmt which is the if statement for the else branch.
// Returns []goast.Stmt which holds extra statements from processing.
// Returns int which is the number of nodes used.
// Returns []*ast_domain.Diagnostic which holds any errors found.
func (ie *ifEmitter) buildElseIfBlock(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	parentSliceExpr goast.Expr,
) (*goast.IfStmt, []goast.Stmt, int, []*ast_domain.Diagnostic) {
	if node.DirElseIf == nil {
		return nil, nil, 0, nil
	}
	return ie.buildConditionalIfStatement(ctx, node, parentSliceExpr, node.DirElseIf.Expression, func(n *ast_domain.TemplateNode) { n.DirElseIf = nil })
}

// buildConditionalIfStatement builds the if statement structure used by both
// p-if and p-else-if directives.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to process.
// Takes parentSliceExpr (goast.Expr) which is the parent slice expression.
// Takes condExpr (ast_domain.Expression) which is the condition expression.
// Takes clearDirective (func(...)) which clears the directive after processing.
//
// Returns *goast.IfStmt which is the built if statement.
// Returns []goast.Stmt which contains statements that must run before the if.
// Returns int which is the count of processed nodes.
// Returns []*ast_domain.Diagnostic which contains any errors or warnings.
func (ie *ifEmitter) buildConditionalIfStatement(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	parentSliceExpr goast.Expr,
	condExpr ast_domain.Expression,
	clearDirective func(*ast_domain.TemplateNode),
) (*goast.IfStmt, []goast.Stmt, int, []*ast_domain.Diagnostic) {
	condGoExpr, prereqStmts, condDiags := ie.expressionEmitter.emit(condExpr)
	condGoExpr = wrapInTruthinessCallIfNeeded(condGoExpr, condExpr)
	bodyStmts, bodyDiags := ie.buildConditionalBody(ctx, node, parentSliceExpr, clearDirective)
	condDiags = append(condDiags, bodyDiags...)

	ifStmt := &goast.IfStmt{
		Cond: condGoExpr,
		Body: &goast.BlockStmt{List: bodyStmts},
	}
	return ifStmt, prereqStmts, 1, condDiags
}

// buildElseBlock generates the code for a final else block.
//
// Takes node (*ast_domain.TemplateNode) which contains the else directive.
// Takes parentSliceExpr (goast.Expr) which is the parent slice expression.
//
// Returns *goast.BlockStmt which contains the generated else block statements.
// Returns []goast.Stmt which is always nil for else blocks.
// Returns int which shows whether an else block was generated (1) or not (0).
// Returns []*ast_domain.Diagnostic which contains any errors from building the
// body.
func (ie *ifEmitter) buildElseBlock(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	parentSliceExpr goast.Expr,
) (*goast.BlockStmt, []goast.Stmt, int, []*ast_domain.Diagnostic) {
	if node.DirElse == nil {
		return nil, nil, 0, nil
	}

	bodyStmts, bodyDiags := ie.buildConditionalBody(ctx, node, parentSliceExpr, func(n *ast_domain.TemplateNode) { n.DirElse = nil })
	return &goast.BlockStmt{List: bodyStmts}, nil, 1, bodyDiags
}

// buildConditionalBody creates the body statements for a conditional block.
// It uses the clone and modify pattern to avoid side effects on the original
// AST.
//
// Takes originalNode (*ast_domain.TemplateNode) which is the node to process.
// Takes parentSliceExpr (goast.Expr) which is the slice to append results to.
// Takes clearDirective (func(...)) which clears the directive from the node.
//
// Returns []goast.Stmt which contains the generated statements.
// Returns []*ast_domain.Diagnostic which contains any problems found.
func (ie *ifEmitter) buildConditionalBody(
	ctx context.Context,
	originalNode *ast_domain.TemplateNode,
	parentSliceExpr goast.Expr,
	clearDirective func(*ast_domain.TemplateNode),
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	var bodyStmts []goast.Stmt
	var bodyDiags []*ast_domain.Diagnostic

	canBeStatic := originalNode.GoAnnotations != nil &&
		originalNode.GoAnnotations.IsStructurallyStatic &&
		!ie.nodeContainsForLoops(originalNode) &&
		!ie.nodeContainsDynamicContent(originalNode)

	if canBeStatic {
		tempNode := originalNode.DeepClone()
		clearDirective(tempNode)
		staticVarIdent, staticDiags := ie.emitter.staticEmitter.registerStaticNode(ctx, tempNode, ie.currentPartialScopeID)
		bodyDiags = staticDiags
		bodyStmts = []goast.Stmt{
			appendToSlice(parentSliceExpr, staticVarIdent),
		}
	} else {
		tempNode := originalNode.Clone()
		clearDirective(tempNode)

		tempNode.Children = originalNode.Children

		emitCtx := newNodeEmissionContext(ctx, nodeEmissionParams{
			Node:                  tempNode,
			ParentSliceExpression: parentSliceExpr,
			Index:                 0,
			Siblings:              []*ast_domain.TemplateNode{tempNode},
			IsRootNode:            false,
			PartialScopeID:        ie.currentPartialScopeID,
			MainComponentScope:    ie.currentMainComponentScope,
		})
		bodyStmts, _, bodyDiags = ie.astBuilder.emitNode(emitCtx)
	}

	return bodyStmts, bodyDiags
}

// nodeContainsForLoops checks if a node or any of its descendants contain a
// p-for directive.
//
// This is used to prevent treating nodes with internal loops as static, which
// would cause issues with dynamic key expressions.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true if the node or any descendant has a p-for
// directive.
func (ie *ifEmitter) nodeContainsForLoops(node *ast_domain.TemplateNode) bool {
	if node == nil {
		return false
	}
	if node.DirFor != nil {
		return true
	}
	return slices.ContainsFunc(node.Children, ie.nodeContainsForLoops)
}

// nodeContainsDynamicContent checks if a node or any of its children contain
// dynamic content that requires runtime evaluation. This includes RichText,
// p-text directives, p-html directives, and dynamic attribute bindings.
//
// Takes node (*ast_domain.TemplateNode) which is the root node to check.
//
// Returns bool which is true when the node or any child has dynamic content.
func (ie *ifEmitter) nodeContainsDynamicContent(node *ast_domain.TemplateNode) bool {
	if node == nil {
		return false
	}
	if len(node.RichText) > 0 {
		return true
	}
	if node.DirText != nil {
		return true
	}
	if node.DirHTML != nil {
		return true
	}
	if len(node.DynamicAttributes) > 0 {
		return true
	}
	return slices.ContainsFunc(node.Children, ie.nodeContainsDynamicContent)
}

// newIfEmitter creates a new emitter for if statements.
//
// Takes emitter (*emitter) which provides the base output methods.
// Takes expressionEmitter (ExpressionEmitter) which builds expressions.
// Takes astBuilder (AstBuilder) which builds AST nodes.
//
// Returns *ifEmitter which is ready to output if statement code.
func newIfEmitter(emitter *emitter, expressionEmitter ExpressionEmitter, astBuilder AstBuilder) *ifEmitter {
	return &ifEmitter{
		emitter:           emitter,
		expressionEmitter: expressionEmitter,
		astBuilder:        astBuilder,
	}
}

// isChainedElseIf checks if a node is a chained else-if that belongs to the
// same conditional block as the start node.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
// Takes startNode (*ast_domain.TemplateNode) which is the first node in the
// conditional chain.
//
// Returns bool which is true if the node is a chained else-if with a matching
// chain key.
func isChainedElseIf(node, startNode *ast_domain.TemplateNode) bool {
	return node.DirElseIf != nil &&
		node.DirElseIf.ChainKey != nil &&
		node.DirElseIf.ChainKey.String() == startNode.Key.String()
}

// isChainedElse checks whether a node is a chained else linked to startNode.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
// Takes startNode (*ast_domain.TemplateNode) which is the node to match against.
//
// Returns bool which is true when the node has an else directive with a chain
// key that matches the start node's key.
func isChainedElse(node, startNode *ast_domain.TemplateNode) bool {
	return node.DirElse != nil &&
		node.DirElse.ChainKey != nil &&
		node.DirElse.ChainKey.String() == startNode.Key.String()
}

// isWhitespaceOrComment checks whether a template node contains only
// whitespace or is a comment node.
//
// Takes node (*TemplateNode) which is the template node to check.
//
// Returns bool which is true if the node is a comment or contains only
// whitespace text.
func isWhitespaceOrComment(node *ast_domain.TemplateNode) bool {
	if node.NodeType == ast_domain.NodeComment {
		return true
	}
	if node.NodeType == ast_domain.NodeText {
		if len(node.RichText) > 0 {
			for _, part := range node.RichText {
				if !part.IsLiteral || strings.TrimSpace(part.Literal) != "" {
					return false
				}
			}
		}
		return strings.TrimSpace(node.TextContent) == ""
	}
	return false
}

// wrapInTruthinessCallIfNeeded wraps a Go expression in a boolean check when
// needed. It uses type information to choose the right method.
//
// Takes condGoExpr (goast.Expr) which is the Go expression to wrap.
// Takes directiveExpr (ast_domain.Expression) which provides type details.
//
// Returns goast.Expr which is a boolean expression for use in if conditions.
func wrapInTruthinessCallIfNeeded(condGoExpr goast.Expr, directiveExpr ast_domain.Expression) goast.Expr {
	ann := getAnnotationFromExpression(directiveExpr)
	return emitTruthinessCheck(condGoExpr, ann)
}

// emitTruthinessCheck converts a Go expression to a boolean condition.
//
// Uses type-specific checks when the type is known:
//   - bool: returns as-is
//   - numeric types: expr != 0
//   - string: expr != ""
//   - pointer types: expr != nil
//   - slices/maps: len(expr) > 0
//   - unknown types: calls pikoruntime.EvaluateTruthiness
//
// Takes goExpr (goast.Expr) which is the expression to convert.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides type details.
//
// Returns goast.Expr which is a boolean expression for use in if conditions.
func emitTruthinessCheck(goExpr goast.Expr, ann *ast_domain.GoGeneratorAnnotation) goast.Expr {
	if ann == nil || ann.ResolvedType == nil || ann.ResolvedType.TypeExpression == nil {
		return wrapInTruthinessCall(goExpr)
	}

	typeExpr := ann.ResolvedType.TypeExpression

	if isBoolType(typeExpr) {
		return goExpr
	}
	if isNumeric(ann.ResolvedType) {
		return emitNotEqualZero(goExpr)
	}
	if isExpressionStringType(ann.ResolvedType) {
		return emitNotEqualEmptyString(goExpr)
	}
	if result := emitTruthinessForCompositeType(goExpr, typeExpr); result != nil {
		return result
	}

	return wrapInTruthinessCall(goExpr)
}

// emitNotEqualZero creates an expression that compares a value against zero.
//
// Takes goExpr (goast.Expr) which is the expression to compare against zero.
//
// Returns *goast.BinaryExpr which is the != 0 comparison for numeric checks.
func emitNotEqualZero(goExpr goast.Expr) *goast.BinaryExpr {
	return &goast.BinaryExpr{X: goExpr, Op: token.NEQ, Y: intLit(0)}
}

// emitNotEqualEmptyString creates an expression that checks if a string is not
// empty (expr != "").
//
// Takes goExpr (goast.Expr) which is the expression to compare against an empty
// string.
//
// Returns *goast.BinaryExpr which is the inequality comparison.
func emitNotEqualEmptyString(goExpr goast.Expr) *goast.BinaryExpr {
	return &goast.BinaryExpr{X: goExpr, Op: token.NEQ, Y: strLit("")}
}

// emitNotEqualNil builds an expr != nil comparison for pointer or interface
// truthiness checks.
//
// Takes goExpr (goast.Expr) which is the expression to compare against nil.
//
// Returns *goast.BinaryExpr which is the inequality comparison.
func emitNotEqualNil(goExpr goast.Expr) *goast.BinaryExpr {
	return &goast.BinaryExpr{X: goExpr, Op: token.NEQ, Y: cachedIdent(nilLiteral)}
}

// emitLenGreaterThanZero creates a len(expr) > 0 comparison for checking if a
// slice or map is not empty.
//
// Takes goExpr (goast.Expr) which is the expression to check.
//
// Returns *goast.BinaryExpr which is the len(goExpr) > 0 comparison.
func emitLenGreaterThanZero(goExpr goast.Expr) *goast.BinaryExpr {
	return &goast.BinaryExpr{
		X:  &goast.CallExpr{Fun: cachedIdent("len"), Args: []goast.Expr{goExpr}},
		Op: token.GTR,
		Y:  intLit(0),
	}
}

// emitTruthinessForCompositeType builds a truthiness check for composite types
// such as arrays, slices, maps, pointers, interfaces, functions, and channels.
//
// Takes goExpr (goast.Expr) which is the expression to check.
// Takes typeExpr (goast.Expr) which is the type used to decide how to check.
//
// Returns goast.Expr which is the truthiness check expression, or nil if the
// type is not a composite type that can be checked.
func emitTruthinessForCompositeType(goExpr goast.Expr, typeExpr goast.Expr) goast.Expr {
	switch t := typeExpr.(type) {
	case *goast.ArrayType:
		if t.Len == nil {
			return emitLenGreaterThanZero(goExpr)
		}
		return cachedIdent("true")
	case *goast.MapType:
		return emitLenGreaterThanZero(goExpr)
	case *goast.StarExpr, *goast.InterfaceType, *goast.FuncType, *goast.ChanType:
		return emitNotEqualNil(goExpr)
	}
	return nil
}
