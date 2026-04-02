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
	"fmt"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/js_ast"
)

// processChainAwareChildren processes sibling nodes and groups if/else-if/else
// chains into single ternary expressions.
//
// Takes children ([]*ast_domain.TemplateNode) which contains the sibling nodes
// to process.
// Takes events (*eventBindingCollection) which collects event bindings found
// during processing.
// Takes loopVars (map[string]bool) which tracks variables defined in enclosing
// loops.
// Takes booleanProps ([]string) which lists property names to treat as boolean.
//
// Returns []js_ast.Expr which contains the JavaScript AST expressions for each
// processed child.
// Returns error when a child node cannot be processed.
func processChainAwareChildren(
	ctx context.Context,
	children []*ast_domain.TemplateNode,
	events *eventBindingCollection,
	loopVars map[string]bool,
	booleanProps []string,
) ([]js_ast.Expr, error) {
	if len(children) == 0 {
		return nil, nil
	}

	var childExprs []js_ast.Expr

	for i := 0; i < len(children); i++ {
		child := children[i]

		expression, skip, err := processChildNode(ctx, child, children, &i, events, loopVars, booleanProps)
		if err != nil {
			return nil, fmt.Errorf("processing child node: %w", err)
		}
		if skip {
			continue
		}
		if expression.Data != nil {
			childExprs = append(childExprs, expression)
		}
	}

	return childExprs, nil
}

// processChildNode handles a single child node, dealing with conditionals and
// normal nodes.
//
// Takes child (*ast_domain.TemplateNode) which is the node to handle.
// Takes children ([]*ast_domain.TemplateNode) which is the full list of sibling
// nodes, needed to handle conditional chains.
// Takes index (*int) which tracks the current position in the children list.
// Takes events (*eventBindingCollection) which gathers event bindings.
// Takes loopVars (map[string]bool) which tracks active loop variables.
// Takes booleanProps ([]string) which lists properties to treat as boolean.
//
// Returns js_ast.Expr which is the built AST expression for the node.
// Returns bool which shows whether the node should be skipped.
// Returns error when building the node AST fails.
func processChildNode(
	ctx context.Context,
	child *ast_domain.TemplateNode,
	children []*ast_domain.TemplateNode,
	index *int,
	events *eventBindingCollection,
	loopVars map[string]bool,
	booleanProps []string,
) (js_ast.Expr, bool, error) {
	if child.DirIf != nil {
		return processConditionalChain(ctx, child, children, index, events, loopVars, booleanProps)
	}

	if child.DirElseIf != nil || child.DirElse != nil {
		return js_ast.Expr{}, true, nil
	}

	nodeExpr, err := buildNodeAST(ctx, child, events, loopVars, booleanProps)
	if err != nil {
		return js_ast.Expr{}, false, err
	}
	if isNull(nodeExpr) {
		return js_ast.Expr{}, true, nil
	}
	return nodeExpr, false, nil
}

// processConditionalChain handles a p-if node and its following else-if/else
// chain.
//
// Takes ifNode (*ast_domain.TemplateNode) which is the initial p-if node.
// Takes children ([]*ast_domain.TemplateNode) which contains sibling nodes to
// scan for else-if/else.
// Takes index (*int) which points to the current index, updated after processing.
// Takes events (*eventBindingCollection) which collects event bindings found.
// Takes loopVars (map[string]bool) which tracks variables from enclosing loops.
// Takes booleanProps ([]string) which lists properties to treat as boolean.
//
// Returns js_ast.Expr which is the conditional chain as a ternary expression.
// Returns bool which is always false for conditional chains.
// Returns error when building the conditional chain AST fails.
func processConditionalChain(
	ctx context.Context,
	ifNode *ast_domain.TemplateNode,
	children []*ast_domain.TemplateNode,
	index *int,
	events *eventBindingCollection,
	loopVars map[string]bool,
	booleanProps []string,
) (js_ast.Expr, bool, error) {
	chain, nextIndex := collectConditionalChain(ifNode, children, *index)

	chainExpr, err := buildConditionalChainAST(ctx, chain, events, loopVars, booleanProps)
	if err != nil {
		return js_ast.Expr{}, false, err
	}

	*index = nextIndex - 1
	return chainExpr, false, nil
}

// collectConditionalChain gathers all nodes in a p-if/p-else-if/p-else chain.
//
// Takes ifNode (*ast_domain.TemplateNode) which is the starting p-if node.
// Takes children ([]*ast_domain.TemplateNode) which contains all sibling nodes.
// Takes startIndex (int) which is the position of ifNode within children.
//
// Returns []*ast_domain.TemplateNode which contains all nodes in the chain.
// Returns int which is the index of the first node after the chain.
func collectConditionalChain(ifNode *ast_domain.TemplateNode, children []*ast_domain.TemplateNode, startIndex int) ([]*ast_domain.TemplateNode, int) {
	chain := []*ast_domain.TemplateNode{ifNode}
	nextIndex := startIndex + 1

	for nextIndex < len(children) {
		nextSibling := children[nextIndex]

		if isInsignificantNode(nextSibling) {
			nextIndex++
			continue
		}

		if !isChainContinuation(nextSibling) {
			break
		}
		chain = append(chain, nextSibling)
		nextIndex++
	}

	return chain, nextIndex
}

// isInsignificantNode checks whether a node can be ignored during rendering.
// Insignificant nodes are whitespace-only text nodes or comment nodes.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true if the node is insignificant.
func isInsignificantNode(node *ast_domain.TemplateNode) bool {
	if node.NodeType == ast_domain.NodeComment {
		return true
	}
	if node.NodeType == ast_domain.NodeText && strings.TrimSpace(node.TextContent) == "" {
		return true
	}
	return false
}

// isChainContinuation checks if a node is part of a conditional chain
// (else-if or else).
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns bool which is true if the node has an else-if or else directive.
func isChainContinuation(node *ast_domain.TemplateNode) bool {
	return node.DirElseIf != nil || node.DirElse != nil
}

// buildConditionalChainAST builds a nested JavaScript ternary expression from
// a slice of nodes that represent an if/else-if/else chain.
//
// Takes chain ([]*ast_domain.TemplateNode) which contains the conditional
// nodes to process.
// Takes events (*eventBindingCollection) which collects event bindings found
// during processing.
// Takes loopVars (map[string]bool) which tracks variables defined in parent
// loops.
// Takes booleanProps ([]string) which lists properties to treat as booleans.
//
// Returns js_ast.Expr which is the nested ternary expression.
// Returns error when building any conditional in the chain fails.
func buildConditionalChainAST(
	ctx context.Context,
	chain []*ast_domain.TemplateNode,
	events *eventBindingCollection,
	loopVars map[string]bool,
	booleanProps []string,
) (js_ast.Expr, error) {
	if len(chain) == 0 {
		return newNullLiteral(), nil
	}

	if len(chain) == 1 {
		return buildSingleConditional(ctx, chain[0], events, loopVars, booleanProps)
	}

	return buildChainedConditional(ctx, chain, events, loopVars, booleanProps)
}

// buildSingleConditional handles a standalone p-if directive.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to process.
// Takes events (*eventBindingCollection) which tracks event bindings.
// Takes loopVars (map[string]bool) which contains loop variable names in scope.
// Takes booleanProps ([]string) which lists properties to treat as boolean.
//
// Returns js_ast.Expr which is the conditional expression for the virtual DOM.
// Returns error when expression transformation or node building fails.
func buildSingleConditional(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	events *eventBindingCollection,
	loopVars map[string]bool,
	booleanProps []string,
) (js_ast.Expr, error) {
	if node.DirIf == nil {
		return newNullLiteral(), nil
	}

	condJS, err := transformOurASTtoJSAST(node.DirIf.Expression, events.getRegistry())
	if err != nil {
		return js_ast.Expr{}, err
	}

	clone := cloneNode(node)
	clone.DirIf = nil
	consequentExpr, err := buildNodeAST(ctx, clone, events, loopVars, booleanProps)
	if err != nil {
		return js_ast.Expr{}, err
	}

	return js_ast.Expr{Data: &js_ast.EIf{
		Test: condJS,
		Yes:  consequentExpr,
		No:   newNullLiteral(),
	}}, nil
}

// buildChainedConditional builds a chained ternary for if/else-if/else.
//
// Takes chain ([]*ast_domain.TemplateNode) which contains the conditional nodes
// to chain together.
// Takes events (*eventBindingCollection) which collects event bindings found
// during processing.
// Takes loopVars (map[string]bool) which tracks variables defined in enclosing
// loops.
// Takes booleanProps ([]string) which lists properties that should be treated
// as booleans.
//
// Returns js_ast.Expr which is the chained ternary expression.
// Returns error when building any node in the chain fails.
func buildChainedConditional(
	ctx context.Context,
	chain []*ast_domain.TemplateNode,
	events *eventBindingCollection,
	loopVars map[string]bool,
	booleanProps []string,
) (js_ast.Expr, error) {
	lastNode := chain[len(chain)-1]
	var ternaryExpr js_ast.Expr

	if lastNode.DirElse != nil {
		clone := cloneNode(lastNode)
		clone.DirElse = nil
		elseExpr, err := buildNodeAST(ctx, clone, events, loopVars, booleanProps)
		if err != nil {
			return js_ast.Expr{}, err
		}
		ternaryExpr = elseExpr
	} else {
		ternaryExpr = newNullLiteral()
	}

	for i := len(chain) - 2; i >= 0; i-- {
		node := chain[i]
		var err error
		ternaryExpr, err = wrapNodeInTernary(ctx, node, ternaryExpr, events, loopVars, booleanProps)
		if err != nil {
			return js_ast.Expr{}, err
		}
	}

	return ternaryExpr, nil
}

// wrapNodeInTernary wraps a node in a ternary expression.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to wrap.
// Takes alternate (js_ast.Expr) which is the fallback expression for the else
// branch.
// Takes events (*eventBindingCollection) which tracks event bindings during
// transformation.
// Takes loopVars (map[string]bool) which contains loop variable names in scope.
// Takes booleanProps ([]string) which lists property names to treat as boolean.
//
// Returns js_ast.Expr which is the resulting ternary expression.
// Returns error when the conditional expression cannot be transformed.
func wrapNodeInTernary(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	alternate js_ast.Expr,
	events *eventBindingCollection,
	loopVars map[string]bool,
	booleanProps []string,
) (js_ast.Expr, error) {
	var condDirective *ast_domain.Directive

	if node.DirIf != nil {
		condDirective = node.DirIf
	} else if node.DirElseIf != nil {
		condDirective = node.DirElseIf
	} else {
		return alternate, nil
	}

	condJS, err := transformOurASTtoJSAST(condDirective.Expression, events.getRegistry())
	if err != nil {
		return js_ast.Expr{}, fmt.Errorf("failed to transform conditional expression '%s': %w", condDirective.RawExpression, err)
	}

	clone := cloneNode(node)
	clone.DirIf = nil
	clone.DirElseIf = nil
	consequentExpr, err := buildNodeAST(ctx, clone, events, loopVars, booleanProps)
	if err != nil {
		return js_ast.Expr{}, err
	}

	return js_ast.Expr{Data: &js_ast.EIf{
		Test: condJS,
		Yes:  consequentExpr,
		No:   alternate,
	}}, nil
}
