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

	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/logger/logger_domain"
)

// RewriteContext holds the state needed when rewriting AST nodes during
// compilation.
type RewriteContext struct {
	// builtInNames maps built-in type and function names to true for quick lookup.
	builtInNames map[string]bool

	// instanceProperties tracks property names already defined on an instance.
	instanceProperties map[string]bool

	// scopes is a stack of variable scopes for tracking declared names.
	scopes []map[string]bool

	// isInsideInstance indicates whether the current position is inside a class
	// instance method or function declaration during rewriting.
	isInsideInstance bool
}

// NewRewriteContext creates a new rewrite context with the given instance
// properties.
//
// Takes instanceProps ([]string) which specifies the property names that belong
// to the component instance.
//
// Returns *RewriteContext which is ready for use with built-in names and the
// provided instance properties.
func NewRewriteContext(instanceProps []string) *RewriteContext {
	propsMap := make(map[string]bool)
	for _, prop := range instanceProps {
		propsMap[prop] = true
	}

	return &RewriteContext{
		isInsideInstance: false,
		scopes:           []map[string]bool{},
		builtInNames: map[string]bool{
			"this": true, "super": true, "console": true, "window": true,
			"document": true, "Array": true, "Object": true, "Number": true,
			"String": true, "Boolean": true, "Math": true, "Date": true,
			"parseInt": true, "parseFloat": true, "RegExp": true,
			"PPElement": true, "piko": true, "dom": true,
			"makeReactive": true, "e": true,
		},
		instanceProperties: propsMap,
	}
}

// RewriteAST changes a syntax tree to add 'this' references for instance
// properties.
//
// When the syntax tree is nil or has no statements, returns without changes.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes syntaxTree (*js_ast.AST) which is the syntax tree to change.
// Takes instanceProps ([]string) which lists property names that need 'this'.
func RewriteAST(ctx context.Context, syntaxTree *js_ast.AST, instanceProps []string) {
	statements := getStmtsFromAST(syntaxTree)
	if syntaxTree == nil || len(statements) == 0 {
		return
	}
	rewriteCtx := NewRewriteContext(instanceProps)
	for _, statement := range statements {
		rewriteStatement(ctx, statement, rewriteCtx)
	}
}

// rewriteStatement rewrites a single JavaScript statement during AST
// transformation.
//
// Takes goCtx (context.Context) which carries logging context for
// trace/request ID propagation.
// Takes statement (js_ast.Stmt) which is the statement to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteStatement(goCtx context.Context, statement js_ast.Stmt, ctx *RewriteContext) {
	if statement.Data == nil {
		return
	}

	if tryRewriteDeclStmt(goCtx, statement, ctx) {
		return
	}

	if tryRewriteControlFlowStmt(goCtx, statement, ctx) {
		return
	}

	rewriteExpressionStmt(goCtx, statement, ctx)
}

// tryRewriteDeclStmt handles declaration statements (block, local, function,
// class).
//
// Takes goCtx (context.Context) which carries logging context for
// trace/request ID propagation.
// Takes statement (js_ast.Stmt) which is the statement to attempt to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
//
// Returns bool which is true if the statement was a declaration and was
// rewritten, false otherwise.
func tryRewriteDeclStmt(goCtx context.Context, statement js_ast.Stmt, ctx *RewriteContext) bool {
	switch node := statement.Data.(type) {
	case *js_ast.SBlock:
		rewriteBlockStmt(goCtx, node, ctx)
		return true
	case *js_ast.SLocal:
		rewriteVarDecl(goCtx, node, ctx)
		return true
	case *js_ast.SFunction:
		rewriteFuncDecl(goCtx, node, ctx)
		return true
	case *js_ast.SClass:
		rewriteClassDecl(goCtx, node, ctx)
		return true
	case *js_ast.SImport:
		_ = node
		return true
	default:
		return false
	}
}

// tryRewriteControlFlowStmt rewrites a control flow statement if it matches a
// known type such as if, while, for, switch, or try.
//
// Takes goCtx (context.Context) which carries logging context for
// trace/request ID propagation.
// Takes statement (js_ast.Stmt) which is the statement to check and rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
//
// Returns bool which is true if the statement was a control flow statement.
//
//nolint:dupl // parallel dispatch for AST types
func tryRewriteControlFlowStmt(goCtx context.Context, statement js_ast.Stmt, ctx *RewriteContext) bool {
	switch node := statement.Data.(type) {
	case *js_ast.SIf:
		rewriteIfStmt(goCtx, node, ctx)
		return true
	case *js_ast.SDoWhile:
		rewriteDoWhileStmt(goCtx, node, ctx)
		return true
	case *js_ast.SWhile:
		rewriteWhileStmt(goCtx, node, ctx)
		return true
	case *js_ast.SFor:
		rewriteForStmt(goCtx, node, ctx)
		return true
	case *js_ast.SForIn:
		rewriteForInStmt(goCtx, node, ctx)
		return true
	case *js_ast.SForOf:
		rewriteForOfStmt(goCtx, node, ctx)
		return true
	case *js_ast.SSwitch:
		rewriteSwitchStmt(goCtx, node, ctx)
		return true
	case *js_ast.STry:
		rewriteTryStmt(goCtx, node, ctx)
		return true
	case *js_ast.SLabel:
		rewriteStatement(goCtx, node.Stmt, ctx)
		return true
	case *js_ast.SWith:
		rewriteExpression(goCtx, node.Value, false, ctx)
		rewriteStatement(goCtx, node.Body, ctx)
		return true
	case *js_ast.SContinue, *js_ast.SBreak, *js_ast.SEmpty, *js_ast.SDebugger:
		return true
	default:
		return false
	}
}

// rewriteExpressionStmt rewrites statements that wrap expressions, such as
// return, throw, and standalone expressions.
//
// Takes goCtx (context.Context) which carries logging context for
// trace/request ID propagation.
// Takes statement (js_ast.Stmt) which is the statement to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteExpressionStmt(goCtx context.Context, statement js_ast.Stmt, ctx *RewriteContext) {
	switch node := statement.Data.(type) {
	case *js_ast.SReturn:
		rewriteReturnStmt(goCtx, node, ctx)
	case *js_ast.SThrow:
		rewriteExpression(goCtx, node.Value, false, ctx)
	case *js_ast.SExpr:
		rewriteExpression(goCtx, node.Value, false, ctx)
	default:
		_, l := logger_domain.From(goCtx, log)
		l.Warn("Unhandled statement type in rewriter", logger_domain.String("type", fmt.Sprintf("%T", node)))
	}
}

// rewriteBlockStmt rewrites all statements within a block scope.
//
// Takes node (*js_ast.SBlock) which contains the statements to rewrite.
// Takes ctx (*RewriteContext) which holds the rewrite state.
func rewriteBlockStmt(goCtx context.Context, node *js_ast.SBlock, ctx *RewriteContext) {
	pushScope(ctx)
	for _, sub := range node.Stmts {
		rewriteStatement(goCtx, sub, ctx)
	}
	popScope(ctx)
}

// rewriteIfStmt rewrites an if statement by processing its test expression
// and its then and else branches.
//
// Takes node (*js_ast.SIf) which is the if statement to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteIfStmt(goCtx context.Context, node *js_ast.SIf, ctx *RewriteContext) {
	rewriteExpression(goCtx, node.Test, false, ctx)
	rewriteStatement(goCtx, node.Yes, ctx)
	if node.NoOrNil.Data != nil {
		rewriteStatement(goCtx, node.NoOrNil, ctx)
	}
}

// rewriteDoWhileStmt rewrites a do-while statement by processing its body and
// test expression.
//
// Takes node (*js_ast.SDoWhile) which is the do-while statement to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteDoWhileStmt(goCtx context.Context, node *js_ast.SDoWhile, ctx *RewriteContext) {
	rewriteStatement(goCtx, node.Body, ctx)
	rewriteExpression(goCtx, node.Test, false, ctx)
}

// rewriteWhileStmt rewrites a while loop statement and its body.
//
// Takes node (*js_ast.SWhile) which is the while statement to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteWhileStmt(goCtx context.Context, node *js_ast.SWhile, ctx *RewriteContext) {
	rewriteExpression(goCtx, node.Test, false, ctx)
	rewriteStatement(goCtx, node.Body, ctx)
}

// rewriteForStmt rewrites a for loop statement and its parts.
//
// Takes node (*js_ast.SFor) which is the for statement to rewrite.
// Takes ctx (*RewriteContext) which provides the rewrite context.
func rewriteForStmt(goCtx context.Context, node *js_ast.SFor, ctx *RewriteContext) {
	if node.InitOrNil.Data != nil {
		rewriteForInitialiserStmt(goCtx, node.InitOrNil, ctx)
	}
	if node.TestOrNil.Data != nil {
		rewriteExpression(goCtx, node.TestOrNil, false, ctx)
	}
	if node.UpdateOrNil.Data != nil {
		rewriteExpression(goCtx, node.UpdateOrNil, false, ctx)
	}
	rewriteStatement(goCtx, node.Body, ctx)
}

// rewriteForInStmt rewrites a for-in statement by processing its initialiser,
// value expression, and body.
//
// Takes node (*js_ast.SForIn) which is the for-in statement to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteForInStmt(goCtx context.Context, node *js_ast.SForIn, ctx *RewriteContext) {
	rewriteForInitialiserStmt(goCtx, node.Init, ctx)
	rewriteExpression(goCtx, node.Value, false, ctx)
	rewriteStatement(goCtx, node.Body, ctx)
}

// rewriteForOfStmt rewrites a JavaScript for-of statement and its parts.
//
// Takes node (*js_ast.SForOf) which is the for-of statement to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteForOfStmt(goCtx context.Context, node *js_ast.SForOf, ctx *RewriteContext) {
	rewriteForInitialiserStmt(goCtx, node.Init, ctx)
	rewriteExpression(goCtx, node.Value, false, ctx)
	rewriteStatement(goCtx, node.Body, ctx)
}

// rewriteSwitchStmt handles a switch statement by rewriting its test
// expression and all case clauses within a new scope.
//
// Takes node (*js_ast.SSwitch) which is the switch statement to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteSwitchStmt(goCtx context.Context, node *js_ast.SSwitch, ctx *RewriteContext) {
	rewriteExpression(goCtx, node.Test, false, ctx)
	pushScope(ctx)
	for i := range node.Cases {
		caseClause := &node.Cases[i]
		if caseClause.ValueOrNil.Data != nil {
			rewriteExpression(goCtx, caseClause.ValueOrNil, false, ctx)
		}
		for _, caseStmt := range caseClause.Body {
			rewriteStatement(goCtx, caseStmt, ctx)
		}
	}
	popScope(ctx)
}

// rewriteReturnStmt processes the expression within a return statement.
//
// Takes node (*js_ast.SReturn) which is the return statement to process.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteReturnStmt(goCtx context.Context, node *js_ast.SReturn, ctx *RewriteContext) {
	if node.ValueOrNil.Data != nil {
		rewriteExpression(goCtx, node.ValueOrNil, false, ctx)
	}
}

// rewriteTryStmt rewrites a try-catch-finally statement, processing each block
// within its own scope.
//
// Takes node (*js_ast.STry) which is the try statement to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteTryStmt(goCtx context.Context, node *js_ast.STry, ctx *RewriteContext) {
	pushScope(ctx)
	for _, tryStmt := range node.Block.Stmts {
		rewriteStatement(goCtx, tryStmt, ctx)
	}
	popScope(ctx)

	if node.Catch != nil {
		rewriteCatchBlock(goCtx, node.Catch, ctx)
	}

	if node.Finally != nil {
		pushScope(ctx)
		for _, finalStmt := range node.Finally.Block.Stmts {
			rewriteStatement(goCtx, finalStmt, ctx)
		}
		popScope(ctx)
	}
}

// rewriteCatchBlock handles a catch clause within a try statement.
//
// Takes catch (*js_ast.Catch) which is the catch clause to handle.
// Takes ctx (*RewriteContext) which provides the rewrite context.
func rewriteCatchBlock(goCtx context.Context, catch *js_ast.Catch, ctx *RewriteContext) {
	pushScope(ctx)
	if catch.BindingOrNil.Data != nil {
		bindCatchParam(catch.BindingOrNil, ctx)
	}
	for _, catchStmt := range catch.Block.Stmts {
		rewriteStatement(goCtx, catchStmt, ctx)
	}
	popScope(ctx)
}

// rewriteVarDecl handles a variable declaration during the rewrite pass.
// It binds all declared variables and rewrites their initialiser expressions.
//
// Takes declaration (*js_ast.SLocal) which is the variable declaration to process.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteVarDecl(goCtx context.Context, declaration *js_ast.SLocal, ctx *RewriteContext) {
	for _, d := range declaration.Decls {
		bindDeclaration(goCtx, d.Binding, ctx)
	}
	for _, d := range declaration.Decls {
		if d.ValueOrNil.Data != nil {
			rewriteExpression(goCtx, d.ValueOrNil, false, ctx)
		}
	}
}

// rewriteFuncDecl rewrites a function declaration by processing its arguments
// and body within a new scope.
//
// Takes jsFunction (*js_ast.SFunction) which is the function
// declaration to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context and state.
func rewriteFuncDecl(goCtx context.Context, jsFunction *js_ast.SFunction, ctx *RewriteContext) {
	pushScope(ctx)
	wasInsideInstance := ctx.isInsideInstance
	ctx.isInsideInstance = true

	for _, argument := range jsFunction.Fn.Args {
		bindDeclaration(goCtx, argument.Binding, ctx)
		if argument.DefaultOrNil.Data != nil {
			rewriteExpression(goCtx, argument.DefaultOrNil, false, ctx)
		}
	}
	for _, functionStatement := range jsFunction.Fn.Body.Block.Stmts {
		rewriteStatement(goCtx, functionStatement, ctx)
	}
	ctx.isInsideInstance = wasInsideInstance
	popScope(ctx)
}

// rewriteClassDecl rewrites a class declaration by processing its extends
// clause and property values.
//
// Takes cd (*js_ast.SClass) which is the class declaration to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteClassDecl(goCtx context.Context, cd *js_ast.SClass, ctx *RewriteContext) {
	if cd.Class.ExtendsOrNil.Data != nil {
		rewriteExpression(goCtx, cd.Class.ExtendsOrNil, false, ctx)
	}
	for i := range cd.Class.Properties {
		prop := &cd.Class.Properties[i]
		if prop.ValueOrNil.Data == nil {
			continue
		}
		if jsFunction, ok := prop.ValueOrNil.Data.(*js_ast.EFunction); ok {
			rewriteMethod(goCtx, jsFunction, ctx)
		} else {
			rewriteExpression(goCtx, prop.ValueOrNil, false, ctx)
		}
	}
}

// rewriteMethod handles a class method during domain rewriting. It binds
// parameter declarations and rewrites all statements in the method body inside
// a new scope with instance context enabled.
//
// Takes method (*js_ast.EFunction) which is the method function to rewrite.
// Takes ctx (*RewriteContext) which holds the current rewriting state.
func rewriteMethod(goCtx context.Context, method *js_ast.EFunction, ctx *RewriteContext) {
	pushScope(ctx)
	wasInsideInstance := ctx.isInsideInstance
	ctx.isInsideInstance = true

	for _, argument := range method.Fn.Args {
		bindDeclaration(goCtx, argument.Binding, ctx)
		if argument.DefaultOrNil.Data != nil {
			rewriteExpression(goCtx, argument.DefaultOrNil, false, ctx)
		}
	}
	for _, bodyStmt := range method.Fn.Body.Block.Stmts {
		rewriteStatement(goCtx, bodyStmt, ctx)
	}
	ctx.isInsideInstance = wasInsideInstance
	popScope(ctx)
}

// rewriteForInitialiserStmt processes the initialiser statement of a for loop.
//
// When statement.Data is nil, returns immediately without action.
//
// Takes statement (js_ast.Stmt) which is the initialiser statement to process.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteForInitialiserStmt(goCtx context.Context, statement js_ast.Stmt, ctx *RewriteContext) {
	if statement.Data == nil {
		return
	}
	switch typed := statement.Data.(type) {
	case *js_ast.SLocal:
		rewriteVarDecl(goCtx, typed, ctx)
	case *js_ast.SExpr:
		rewriteExpression(goCtx, typed.Value, false, ctx)
	}
}

// rewriteExpression transforms a JavaScript AST expression for
// domain context.
//
// Takes goCtx (context.Context) which carries logging context for
// trace/request ID propagation.
// Takes expression (js_ast.Expr) which is the expression node to
// transform.
// Takes isLeft (bool) which indicates if the expression is on the
// left side of an assignment.
// Takes ctx (*RewriteContext) which provides the rewriting state
// and options.
func rewriteExpression(goCtx context.Context, expression js_ast.Expr, isLeft bool, ctx *RewriteContext) {
	if expression.Data == nil || ctx.isInsideInstance {
		return
	}

	if node, ok := expression.Data.(*js_ast.EIdentifier); ok {
		rewriteIdentifier(node, isLeft, ctx)
		return
	}

	if isLiteralExpr(expression) {
		return
	}

	if tryRewriteOperatorExpr(goCtx, expression, ctx) {
		return
	}

	if tryRewriteMemberCallExpr(goCtx, expression, ctx) {
		return
	}

	rewriteCollectionOrFunctionExpr(goCtx, expression, ctx)
}

// isLiteralExpr checks if the expression is a literal that does
// not need rewriting.
//
// Takes expression (js_ast.Expr) which is the expression to
// check.
//
// Returns bool which is true if the expression is a literal type.
func isLiteralExpr(expression js_ast.Expr) bool {
	switch expression.Data.(type) {
	case *js_ast.EString, *js_ast.ENumber, *js_ast.EBoolean, *js_ast.ENull:
		return true
	case *js_ast.EUndefined, *js_ast.EThis, *js_ast.ESuper:
		return true
	case *js_ast.ENewTarget, *js_ast.EImportMeta:
		return true
	default:
		return false
	}
}

// tryRewriteOperatorExpr handles operator expressions such as
// unary, binary, and conditional types.
//
// Takes goCtx (context.Context) which carries logging context for
// trace/request ID propagation.
// Takes expression (js_ast.Expr) which is the expression to check
// and rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting
// context.
//
// Returns bool which is true if the expression was an operator
// type and was rewritten.
func tryRewriteOperatorExpr(goCtx context.Context, expression js_ast.Expr, ctx *RewriteContext) bool {
	switch node := expression.Data.(type) {
	case *js_ast.EUnary:
		rewriteExpression(goCtx, node.Value, false, ctx)
		return true
	case *js_ast.EBinary:
		rewriteBinaryExpr(goCtx, node, ctx)
		return true
	case *js_ast.EIf:
		rewriteIfExpr(goCtx, node, ctx)
		return true
	default:
		return false
	}
}

// tryRewriteMemberCallExpr handles member access and call
// expressions.
//
// Takes goCtx (context.Context) which carries logging context for
// trace/request ID propagation.
// Takes expression (js_ast.Expr) which is the expression to check
// and rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting
// context.
//
// Returns bool which is true if the expression was handled.
func tryRewriteMemberCallExpr(goCtx context.Context, expression js_ast.Expr, ctx *RewriteContext) bool {
	switch node := expression.Data.(type) {
	case *js_ast.EDot:
		rewriteExpression(goCtx, node.Target, false, ctx)
		return true
	case *js_ast.EIndex:
		rewriteIndexExpr(goCtx, node, ctx)
		return true
	case *js_ast.ECall:
		rewriteCallExpr(goCtx, node, ctx)
		return true
	case *js_ast.ENew:
		rewriteNewExpr(goCtx, node, ctx)
		return true
	default:
		return false
	}
}

// rewriteCollectionOrFunctionExpr rewrites collection literals and
// function expressions by passing them to the correct
// type-specific rewriter.
//
// Takes goCtx (context.Context) which carries logging context for
// trace/request ID propagation.
// Takes expression (js_ast.Expr) which is the expression to
// process.
// Takes ctx (*RewriteContext) which provides the rewriting
// context.
func rewriteCollectionOrFunctionExpr(goCtx context.Context, expression js_ast.Expr, ctx *RewriteContext) {
	switch node := expression.Data.(type) {
	case *js_ast.EObject:
		rewriteObjectExpr(goCtx, node, ctx)
	case *js_ast.EArray:
		rewriteArrayExpr(goCtx, node, ctx)
	case *js_ast.ETemplate:
		rewriteTemplateExpr(goCtx, node, ctx)
	case *js_ast.EArrow:
		rewriteArrowExpr(goCtx, node, ctx)
	case *js_ast.EYield:
		rewriteYieldExpr(goCtx, node, ctx)
	default:
		_, l := logger_domain.From(goCtx, log)
		l.Warn("Unhandled expression type in rewriter", logger_domain.String("type", fmt.Sprintf("%T", node)))
	}
}

// rewriteBinaryExpr rewrites both sides of a binary expression.
//
// Takes node (*js_ast.EBinary) which is the binary expression to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteBinaryExpr(goCtx context.Context, node *js_ast.EBinary, ctx *RewriteContext) {
	if isAssignmentOperator(node.Op) {
		rewriteExpression(goCtx, node.Left, true, ctx)
	} else {
		rewriteExpression(goCtx, node.Left, false, ctx)
	}
	rewriteExpression(goCtx, node.Right, false, ctx)
}

// rewriteIfExpr rewrites a ternary conditional expression.
//
// Takes node (*js_ast.EIf) which is the conditional expression to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteIfExpr(goCtx context.Context, node *js_ast.EIf, ctx *RewriteContext) {
	rewriteExpression(goCtx, node.Test, false, ctx)
	rewriteExpression(goCtx, node.Yes, false, ctx)
	rewriteExpression(goCtx, node.No, false, ctx)
}

// rewriteIndexExpr rewrites the target and index parts of an index expression.
//
// Takes node (*js_ast.EIndex) which is the index expression to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteIndexExpr(goCtx context.Context, node *js_ast.EIndex, ctx *RewriteContext) {
	rewriteExpression(goCtx, node.Target, false, ctx)
	rewriteExpression(goCtx, node.Index, false, ctx)
}

// rewriteCallExpr rewrites a function call expression and all its arguments.
//
// Takes node (*js_ast.ECall) which is the call expression to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteCallExpr(goCtx context.Context, node *js_ast.ECall, ctx *RewriteContext) {
	rewriteExpression(goCtx, node.Target, false, ctx)
	for i := range node.Args {
		rewriteExpression(goCtx, node.Args[i], false, ctx)
	}
}

// rewriteNewExpr rewrites a JavaScript new expression by processing its target
// and arguments.
//
// Takes node (*js_ast.ENew) which is the new expression to process.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteNewExpr(goCtx context.Context, node *js_ast.ENew, ctx *RewriteContext) {
	rewriteExpression(goCtx, node.Target, false, ctx)
	for i := range node.Args {
		rewriteExpression(goCtx, node.Args[i], false, ctx)
	}
}

// rewriteObjectExpr rewrites the property values of an object literal.
//
// Takes node (*js_ast.EObject) which is the object expression to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteObjectExpr(goCtx context.Context, node *js_ast.EObject, ctx *RewriteContext) {
	for i := range node.Properties {
		property := &node.Properties[i]
		if property.Kind == js_ast.PropertySpread || property.Kind == js_ast.PropertyField {
			if property.ValueOrNil.Data != nil {
				rewriteExpression(goCtx, property.ValueOrNil, false, ctx)
			}
		}
	}
}

// rewriteArrayExpr rewrites each item in an array expression.
//
// Takes node (*js_ast.EArray) which is the array expression to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteArrayExpr(goCtx context.Context, node *js_ast.EArray, ctx *RewriteContext) {
	for i := range node.Items {
		if node.Items[i].Data != nil {
			rewriteExpression(goCtx, node.Items[i], false, ctx)
		}
	}
}

// rewriteTemplateExpr rewrites a template literal expression and its parts.
//
// Takes node (*js_ast.ETemplate) which is the template expression to rewrite.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteTemplateExpr(goCtx context.Context, node *js_ast.ETemplate, ctx *RewriteContext) {
	if node.TagOrNil.Data != nil {
		rewriteExpression(goCtx, node.TagOrNil, false, ctx)
	}
	for i := range node.Parts {
		if node.Parts[i].Value.Data != nil {
			rewriteExpression(goCtx, node.Parts[i].Value, false, ctx)
		}
	}
}

// rewriteArrowExpr rewrites an arrow function expression by binding its
// parameters and rewriting its body statements within a new scope.
//
// Takes node (*js_ast.EArrow) which is the arrow function to rewrite.
// Takes ctx (*RewriteContext) which holds the rewriting state.
func rewriteArrowExpr(goCtx context.Context, node *js_ast.EArrow, ctx *RewriteContext) {
	pushScope(ctx)
	for _, argument := range node.Args {
		bindDeclaration(goCtx, argument.Binding, ctx)
		if argument.DefaultOrNil.Data != nil {
			rewriteExpression(goCtx, argument.DefaultOrNil, false, ctx)
		}
	}
	for _, arrowStmt := range node.Body.Block.Stmts {
		rewriteStatement(goCtx, arrowStmt, ctx)
	}
	popScope(ctx)
}

// rewriteYieldExpr rewrites the value expression within a yield statement.
//
// Takes node (*js_ast.EYield) which is the yield expression to process.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func rewriteYieldExpr(goCtx context.Context, node *js_ast.EYield, ctx *RewriteContext) {
	if node.ValueOrNil.Data != nil {
		rewriteExpression(goCtx, node.ValueOrNil, false, ctx)
	}
}

// rewriteIdentifier does nothing and is a placeholder for identifier
// expression rewriting.
func rewriteIdentifier(_ *js_ast.EIdentifier, _ bool, _ *RewriteContext) {
}

// bindDeclaration adds names from a binding pattern to the current scope.
//
// Takes binding (js_ast.Binding) which is the binding pattern to process.
// Takes ctx (*RewriteContext) which provides the current rewrite context.
func bindDeclaration(goCtx context.Context, binding js_ast.Binding, ctx *RewriteContext) {
	if binding.Data == nil {
		return
	}
	switch typedBinding := binding.Data.(type) {
	case *js_ast.BIdentifier:
		addNameToScope("binding", ctx)
	case *js_ast.BArray:
		bindArrayBinding(goCtx, typedBinding, ctx)
	case *js_ast.BObject:
		bindObjectBinding(goCtx, typedBinding, ctx)
	}
}

// bindArrayBinding handles array destructuring bindings by walking through
// each element in the pattern recursively.
//
// Takes arr (*js_ast.BArray) which is the array binding pattern to process.
// Takes ctx (*RewriteContext) which provides the rewriting context.
func bindArrayBinding(goCtx context.Context, arr *js_ast.BArray, ctx *RewriteContext) {
	for i := range arr.Items {
		if arr.Items[i].Binding.Data != nil {
			bindDeclaration(goCtx, arr.Items[i].Binding, ctx)
		}
		if arr.Items[i].DefaultValueOrNil.Data != nil {
			rewriteExpression(goCtx, arr.Items[i].DefaultValueOrNil, false, ctx)
		}
	}
}

// bindObjectBinding processes an object destructuring pattern to bind its
// properties.
//
// Takes objectBinding (*js_ast.BObject) which is the object binding
// pattern to process.
// Takes ctx (*RewriteContext) which provides the rewrite context.
func bindObjectBinding(goCtx context.Context, objectBinding *js_ast.BObject, ctx *RewriteContext) {
	for i := range objectBinding.Properties {
		prop := &objectBinding.Properties[i]
		if prop.Value.Data != nil {
			bindDeclaration(goCtx, prop.Value, ctx)
		}
		if prop.DefaultValueOrNil.Data != nil {
			rewriteExpression(goCtx, prop.DefaultValueOrNil, false, ctx)
		}
	}
}

// bindCatchParam adds catch block parameter bindings to the scope.
//
// Takes binding (js_ast.Binding) which is the catch parameter to bind.
// Takes ctx (*RewriteContext) which provides the current rewrite state.
func bindCatchParam(binding js_ast.Binding, ctx *RewriteContext) {
	if binding.Data == nil {
		return
	}
	switch typed := binding.Data.(type) {
	case *js_ast.BIdentifier:
		addNameToScope("catchParam", ctx)
	case *js_ast.BArray:
		for i := range typed.Items {
			if typed.Items[i].Binding.Data != nil {
				bindCatchParam(typed.Items[i].Binding, ctx)
			}
		}
	case *js_ast.BObject:
		for i := range typed.Properties {
			prop := &typed.Properties[i]
			if prop.Value.Data != nil {
				bindCatchParam(prop.Value, ctx)
			}
		}
	}
}

// pushScope adds a new empty scope to the scope stack.
//
// Takes ctx (*RewriteContext) which holds the scope stack to modify.
func pushScope(ctx *RewriteContext) {
	ctx.scopes = append(ctx.scopes, map[string]bool{})
}

// popScope removes the top scope from the rewrite context's scope stack.
//
// Takes ctx (*RewriteContext) which holds the current scope stack.
func popScope(ctx *RewriteContext) {
	if len(ctx.scopes) > 0 {
		ctx.scopes = ctx.scopes[:len(ctx.scopes)-1]
	}
}

// addNameToScope adds the given identifier to the current scope.
//
// When no scope exists, creates a new scope before adding the identifier.
//
// Takes identifier (string) which is the name to add to the scope.
// Takes ctx (*RewriteContext) which holds the scope stack.
func addNameToScope(identifier string, ctx *RewriteContext) {
	if len(ctx.scopes) == 0 {
		pushScope(ctx)
	}
	ctx.scopes[len(ctx.scopes)-1][identifier] = true
}

// isAssignmentOperator checks whether the given operator is an assignment
// operator.
//
// Takes op (js_ast.OpCode) which is the binary operator to check.
//
// Returns bool which is true if op is any form of assignment operator.
func isAssignmentOperator(op js_ast.OpCode) bool {
	switch op {
	case js_ast.BinOpAssign,
		js_ast.BinOpAddAssign, js_ast.BinOpSubAssign,
		js_ast.BinOpMulAssign, js_ast.BinOpDivAssign, js_ast.BinOpRemAssign,
		js_ast.BinOpPowAssign,
		js_ast.BinOpShlAssign, js_ast.BinOpShrAssign, js_ast.BinOpUShrAssign,
		js_ast.BinOpBitwiseAndAssign, js_ast.BinOpBitwiseOrAssign, js_ast.BinOpBitwiseXorAssign,
		js_ast.BinOpLogicalAndAssign, js_ast.BinOpLogicalOrAssign,
		js_ast.BinOpNullishCoalescingAssign:
		return true
	default:
		return false
	}
}
