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
	parsejs "github.com/tdewolff/parse/v2/js"
)

// TdewolffRewriteContext holds state for rewriting a tdewolff AST.
// It tracks built-in names, instance properties, and variable scopes.
type TdewolffRewriteContext struct {
	// builtInNames holds JavaScript built-in names to skip during rewriting.
	builtInNames map[string]bool

	// instanceProperties tracks which names are instance properties.
	instanceProperties map[string]bool

	// scopes holds a stack of variable name sets for tracking lexical scope.
	scopes []map[string]bool

	// inClassMethod indicates whether the rewriter is currently
	// inside a class method.
	inClassMethod bool
}

// NewTdewolffRewriteContext creates a context for tdewolff AST rewriting.
//
// Takes instanceProps ([]string) which specifies the property names that
// belong to the component instance.
//
// Returns *TdewolffRewriteContext which is the configured context ready for
// AST rewriting operations.
func NewTdewolffRewriteContext(instanceProps []string) *TdewolffRewriteContext {
	propsMap := make(map[string]bool)
	for _, prop := range instanceProps {
		propsMap[prop] = true
	}

	return &TdewolffRewriteContext{
		scopes:        []map[string]bool{},
		inClassMethod: false,
		builtInNames: map[string]bool{
			"this": true, "super": true, "console": true, "window": true,
			"document": true, "Array": true, "Object": true, "Number": true,
			"String": true, "Boolean": true, "Math": true, "Date": true,
			"parseInt": true, "parseFloat": true, "RegExp": true,
			"PPElement": true, "piko": true, "dom": true,
			"makeReactive": true, "e": true, "event": true,
			"JSON": true, "Promise": true, "Error": true, "Map": true,
			"Set": true, "WeakMap": true, "WeakSet": true, "Symbol": true,
			"Proxy": true, "Reflect": true, "Intl": true, "BigInt": true,
			"undefined": true, "null": true, "NaN": true, "Infinity": true,
			"isNaN": true, "isFinite": true, "encodeURI": true,
			"encodeURIComponent": true, "decodeURI": true, "decodeURIComponent": true,
			"eval": true, "fetch": true, "setTimeout": true, "setInterval": true,
			"clearTimeout": true, "clearInterval": true, "requestAnimationFrame": true,
			"cancelAnimationFrame": true, "atob": true, "btoa": true,
			"alert": true, "confirm": true, "prompt": true,
		},
		instanceProperties: propsMap,
	}
}

// RewriteTdewolffAST adds the this.$$ctx. prefix to identifiers in a tdewolff
// parsed JavaScript AST.
//
// Takes ast (*parsejs.AST) which is the parsed JavaScript AST to rewrite.
// Takes instanceProps ([]string) which lists the instance property names to
// prefix.
func RewriteTdewolffAST(ast *parsejs.AST, instanceProps []string) {
	if ast == nil || len(ast.List) == 0 {
		return
	}
	ctx := NewTdewolffRewriteContext(instanceProps)
	for _, statement := range ast.List {
		rewriteTdewolffStmt(statement, ctx)
	}
}

// rewriteTdewolffStmt rewrites a single statement in the tdewolff AST.
//
// When statement is nil, returns straight away without changes.
//
// Takes statement (parsejs.IStmt) which is the statement to rewrite.
// Takes ctx (*TdewolffRewriteContext) which holds the rewrite state.
func rewriteTdewolffStmt(statement parsejs.IStmt, ctx *TdewolffRewriteContext) {
	if statement == nil {
		return
	}

	if tryRewriteTdewolffDeclStmt(statement, ctx) {
		return
	}

	if tryRewriteTdewolffControlFlowStmt(statement, ctx) {
		return
	}

	rewriteTdewolffExpressionStmt(statement, ctx)
}

// tryRewriteTdewolffDeclStmt handles declaration statements during rewriting.
//
// Takes statement (parsejs.IStmt) which is the statement to check and rewrite.
// Takes ctx (*TdewolffRewriteContext) which provides the rewrite context.
//
// Returns bool which is true if the statement was a declaration type and was
// rewritten, or false otherwise.
func tryRewriteTdewolffDeclStmt(statement parsejs.IStmt, ctx *TdewolffRewriteContext) bool {
	switch node := statement.(type) {
	case *parsejs.BlockStmt:
		rewriteTdewolffBlockStmt(node, ctx)
		return true
	case *parsejs.VarDecl:
		rewriteTdewolffVarDecl(node, ctx)
		return true
	case *parsejs.FuncDecl:
		rewriteTdewolffFuncDecl(node, ctx)
		return true
	case *parsejs.ClassDecl:
		rewriteTdewolffClassDecl(node, ctx)
		return true
	default:
		return false
	}
}

// tryRewriteTdewolffControlFlowStmt handles control flow statements by
// dispatching to the appropriate rewrite function based on statement type.
//
// Takes statement (parsejs.IStmt) which is the statement to process.
// Takes ctx (*TdewolffRewriteContext) which provides the rewrite context.
//
// Returns bool which is true if the statement was a control flow statement
// and was processed, or false if the statement type was not recognised.
//
//nolint:dupl // parallel dispatch for AST types
func tryRewriteTdewolffControlFlowStmt(statement parsejs.IStmt, ctx *TdewolffRewriteContext) bool {
	switch node := statement.(type) {
	case *parsejs.IfStmt:
		rewriteTdewolffIfStmt(node, ctx)
		return true
	case *parsejs.DoWhileStmt:
		rewriteTdewolffDoWhileStmt(node, ctx)
		return true
	case *parsejs.WhileStmt:
		rewriteTdewolffWhileStmt(node, ctx)
		return true
	case *parsejs.ForStmt:
		rewriteTdewolffForStmt(node, ctx)
		return true
	case *parsejs.ForInStmt:
		rewriteTdewolffForInStmt(node, ctx)
		return true
	case *parsejs.ForOfStmt:
		rewriteTdewolffForOfStmt(node, ctx)
		return true
	case *parsejs.SwitchStmt:
		rewriteTdewolffSwitchStmt(node, ctx)
		return true
	case *parsejs.TryStmt:
		rewriteTdewolffTryStmt(node, ctx)
		return true
	default:
		return false
	}
}

// rewriteTdewolffExpressionStmt handles statements that contain expressions.
//
// Takes statement (parsejs.IStmt) which is the statement to handle.
// Takes ctx (*TdewolffRewriteContext) which provides the rewrite context.
func rewriteTdewolffExpressionStmt(statement parsejs.IStmt, ctx *TdewolffRewriteContext) {
	switch node := statement.(type) {
	case *parsejs.ReturnStmt:
		rewriteTdewolffReturnStmt(node, ctx)
	case *parsejs.ThrowStmt:
		rewriteTdewolffExpr(node.Value, false, ctx)
	case *parsejs.ExprStmt:
		rewriteTdewolffExpr(node.Value, false, ctx)
	case *parsejs.LabelledStmt:
		rewriteTdewolffStmt(node.Value, ctx)
	}
}

// rewriteTdewolffBlockStmt processes all statements within a block statement.
// It creates a new scope, rewrites each statement, then removes the scope.
//
// Takes node (*parsejs.BlockStmt) which is the block statement to process.
// Takes ctx (*TdewolffRewriteContext) which tracks the rewrite state.
func rewriteTdewolffBlockStmt(node *parsejs.BlockStmt, ctx *TdewolffRewriteContext) {
	tdewolffPushScope(ctx)
	for _, sub := range node.List {
		rewriteTdewolffStmt(sub, ctx)
	}
	tdewolffPopScope(ctx)
}

// rewriteTdewolffIfStmt rewrites an if statement by processing its condition,
// body, and else clause if present.
//
// Takes node (*parsejs.IfStmt) which is the if statement to rewrite.
// Takes ctx (*TdewolffRewriteContext) which provides the rewrite context.
func rewriteTdewolffIfStmt(node *parsejs.IfStmt, ctx *TdewolffRewriteContext) {
	rewriteTdewolffExpr(node.Cond, false, ctx)
	rewriteTdewolffStmt(node.Body, ctx)
	if node.Else != nil {
		rewriteTdewolffStmt(node.Else, ctx)
	}
}

// rewriteTdewolffDoWhileStmt processes a do-while statement by rewriting its
// body and condition expression.
//
// Takes node (*parsejs.DoWhileStmt) which is the do-while
// statement to rewrite.
// Takes ctx (*TdewolffRewriteContext) which provides the rewrite context.
func rewriteTdewolffDoWhileStmt(node *parsejs.DoWhileStmt, ctx *TdewolffRewriteContext) {
	rewriteTdewolffStmt(node.Body, ctx)
	rewriteTdewolffExpr(node.Cond, false, ctx)
}

// rewriteTdewolffWhileStmt rewrites a while statement by processing its
// condition and body.
//
// Takes node (*parsejs.WhileStmt) which is the while statement to rewrite.
// Takes ctx (*TdewolffRewriteContext) which provides the rewrite context.
func rewriteTdewolffWhileStmt(node *parsejs.WhileStmt, ctx *TdewolffRewriteContext) {
	rewriteTdewolffExpr(node.Cond, false, ctx)
	rewriteTdewolffStmt(node.Body, ctx)
}

// rewriteTdewolffSwitchStmt rewrites a switch statement and all its cases.
//
// Takes node (*parsejs.SwitchStmt) which is the switch statement to rewrite.
// Takes ctx (*TdewolffRewriteContext) which provides the rewrite context.
func rewriteTdewolffSwitchStmt(node *parsejs.SwitchStmt, ctx *TdewolffRewriteContext) {
	rewriteTdewolffExpr(node.Init, false, ctx)
	for i := range node.List {
		rewriteTdewolffSwitchCase(&node.List[i], ctx)
	}
}

// rewriteTdewolffReturnStmt rewrites the return value expression if present.
//
// Takes node (*parsejs.ReturnStmt) which is the return statement to rewrite.
// Takes ctx (*TdewolffRewriteContext) which holds the rewrite context.
func rewriteTdewolffReturnStmt(node *parsejs.ReturnStmt, ctx *TdewolffRewriteContext) {
	if node.Value != nil {
		rewriteTdewolffExpr(node.Value, false, ctx)
	}
}

// rewriteTdewolffTryStmt rewrites a try-catch-finally statement and its blocks.
//
// Takes node (*parsejs.TryStmt) which is the try statement to rewrite.
// Takes ctx (*TdewolffRewriteContext) which tracks the rewrite state.
func rewriteTdewolffTryStmt(node *parsejs.TryStmt, ctx *TdewolffRewriteContext) {
	rewriteTdewolffStmt(node.Body, ctx)
	if node.Catch != nil {
		tdewolffPushScope(ctx)
		if node.Binding != nil {
			tdewolffBindDeclaration(node.Binding, ctx)
		}
		rewriteTdewolffStmt(node.Catch, ctx)
		tdewolffPopScope(ctx)
	}
	if node.Finally != nil {
		rewriteTdewolffStmt(node.Finally, ctx)
	}
}

// rewriteTdewolffExpr rewrites a JavaScript expression by
// passing it to the correct handler based on its type.
//
// Takes expression (parsejs.IExpr) which is the expression to
// rewrite.
// Takes isLeft (bool) which shows if this is on the left side
// of an assignment.
// Takes ctx (*TdewolffRewriteContext) which holds the rewriting
// state.
func rewriteTdewolffExpr(expression parsejs.IExpr, isLeft bool, ctx *TdewolffRewriteContext) {
	if expression == nil {
		return
	}

	if node, ok := expression.(*parsejs.Var); ok {
		rewriteTdewolffVar(node, isLeft, ctx)
		return
	}

	if tryRewriteTdewolffOperatorExpr(expression, ctx) {
		return
	}

	if tryRewriteTdewolffMemberCallExpr(expression, ctx) {
		return
	}

	rewriteTdewolffCollectionOrFunctionExpr(expression, isLeft, ctx)
}

// tryRewriteTdewolffOperatorExpr checks if an expression is an
// operator type and rewrites it.
//
// Takes expression (parsejs.IExpr) which is the expression to
// check and rewrite.
// Takes ctx (*TdewolffRewriteContext) which provides the
// rewrite context.
//
// Returns bool which is true if the expression was an operator
// type and was rewritten.
func tryRewriteTdewolffOperatorExpr(expression parsejs.IExpr, ctx *TdewolffRewriteContext) bool {
	switch node := expression.(type) {
	case *parsejs.UnaryExpr:
		rewriteTdewolffExpr(node.X, false, ctx)
		return true
	case *parsejs.BinaryExpr:
		rewriteTdewolffBinaryExpr(node, ctx)
		return true
	case *parsejs.CondExpr:
		rewriteTdewolffCondExpr(node, ctx)
		return true
	default:
		return false
	}
}

// tryRewriteTdewolffMemberCallExpr handles member access and
// call expressions.
//
// Takes expression (parsejs.IExpr) which is the expression to
// check and rewrite.
// Takes ctx (*TdewolffRewriteContext) which provides the
// rewrite state.
//
// Returns bool which is true if the expression was handled.
func tryRewriteTdewolffMemberCallExpr(expression parsejs.IExpr, ctx *TdewolffRewriteContext) bool {
	switch node := expression.(type) {
	case *parsejs.DotExpr:
		rewriteTdewolffExpr(node.X, false, ctx)
		return true
	case *parsejs.IndexExpr:
		rewriteTdewolffIndexExpr(node, ctx)
		return true
	case *parsejs.CallExpr:
		rewriteTdewolffCallExpr(node, ctx)
		return true
	case *parsejs.NewExpr:
		rewriteTdewolffNewExpr(node, ctx)
		return true
	default:
		return false
	}
}

// rewriteTdewolffCollectionOrFunctionExpr handles collection
// literals and function expressions by passing them to the
// correct rewriter.
//
// Takes expression (parsejs.IExpr) which is the expression to
// rewrite.
// Takes isLeft (bool) which shows if the expression is on the
// left side.
// Takes ctx (*TdewolffRewriteContext) which provides the
// rewrite context.
func rewriteTdewolffCollectionOrFunctionExpr(expression parsejs.IExpr, isLeft bool, ctx *TdewolffRewriteContext) {
	switch node := expression.(type) {
	case *parsejs.ArrayExpr:
		rewriteTdewolffArrayExpr(node, ctx)
	case *parsejs.ObjectExpr:
		rewriteTdewolffObjectExpr(node, ctx)
	case *parsejs.GroupExpr:
		rewriteTdewolffExpr(node.X, isLeft, ctx)
	case *parsejs.ArrowFunc:
		rewriteTdewolffArrowFunc(node, ctx)
	case *parsejs.FuncDecl:
		rewriteTdewolffFuncDecl(node, ctx)
	case *parsejs.TemplateExpr:
		rewriteTdewolffTemplateExpr(node, ctx)
	}
}

// rewriteTdewolffBinaryExpr rewrites a binary expression by processing both
// operands. It marks the left operand as assigned if the operator is an
// assignment operator.
//
// Takes node (*parsejs.BinaryExpr) which is the binary
// expression to rewrite.
// Takes ctx (*TdewolffRewriteContext) which tracks the rewrite state.
func rewriteTdewolffBinaryExpr(node *parsejs.BinaryExpr, ctx *TdewolffRewriteContext) {
	isAssign := isAssignOp(node.Op)
	rewriteTdewolffExpr(node.X, isAssign, ctx)
	rewriteTdewolffExpr(node.Y, false, ctx)
}

// rewriteTdewolffCondExpr processes a ternary expression by rewriting its
// condition and both branches.
//
// Takes node (*parsejs.CondExpr) which is the ternary expression
// to rewrite.
// Takes ctx (*TdewolffRewriteContext) which provides the
// rewriting context.
func rewriteTdewolffCondExpr(node *parsejs.CondExpr, ctx *TdewolffRewriteContext) {
	rewriteTdewolffExpr(node.Cond, false, ctx)
	rewriteTdewolffExpr(node.X, false, ctx)
	rewriteTdewolffExpr(node.Y, false, ctx)
}

// rewriteTdewolffIndexExpr rewrites the base and index parts of an index
// expression node.
//
// Takes node (*parsejs.IndexExpr) which is the index expression
// to rewrite.
// Takes ctx (*TdewolffRewriteContext) which provides the rewrite context.
func rewriteTdewolffIndexExpr(node *parsejs.IndexExpr, ctx *TdewolffRewriteContext) {
	rewriteTdewolffExpr(node.X, false, ctx)
	rewriteTdewolffExpr(node.Y, false, ctx)
}

// rewriteTdewolffCallExpr rewrites a call expression and all its arguments.
//
// Takes node (*parsejs.CallExpr) which is the call expression to rewrite.
// Takes ctx (*TdewolffRewriteContext) which holds the rewrite state.
func rewriteTdewolffCallExpr(node *parsejs.CallExpr, ctx *TdewolffRewriteContext) {
	rewriteTdewolffExpr(node.X, false, ctx)
	for i := range node.Args.List {
		rewriteTdewolffExpr(node.Args.List[i].Value, false, ctx)
	}
}

// rewriteTdewolffNewExpr rewrites a JavaScript new expression
// and its arguments.
//
// Takes node (*parsejs.NewExpr) which is the new expression to rewrite.
// Takes ctx (*TdewolffRewriteContext) which provides the rewrite context.
func rewriteTdewolffNewExpr(node *parsejs.NewExpr, ctx *TdewolffRewriteContext) {
	rewriteTdewolffExpr(node.X, false, ctx)
	if node.Args != nil {
		for i := range node.Args.List {
			rewriteTdewolffExpr(node.Args.List[i].Value, false, ctx)
		}
	}
}

// rewriteTdewolffArrayExpr rewrites each element within an array expression.
//
// Takes node (*parsejs.ArrayExpr) which is the array expression to process.
// Takes ctx (*TdewolffRewriteContext) which provides the rewrite context.
func rewriteTdewolffArrayExpr(node *parsejs.ArrayExpr, ctx *TdewolffRewriteContext) {
	for i := range node.List {
		rewriteTdewolffExpr(node.List[i].Value, false, ctx)
	}
}

// rewriteTdewolffObjectExpr applies rewrite rules to each property value in an
// object literal.
//
// Takes node (*parsejs.ObjectExpr) which is the object expression to process.
// Takes ctx (*TdewolffRewriteContext) which holds the rewrite state.
func rewriteTdewolffObjectExpr(node *parsejs.ObjectExpr, ctx *TdewolffRewriteContext) {
	for i := range node.List {
		if node.List[i].Value != nil {
			rewriteTdewolffExpr(node.List[i].Value, false, ctx)
		}
	}
}

// rewriteTdewolffTemplateExpr processes a template expression node by
// rewriting each embedded expression within the template.
//
// Takes node (*parsejs.TemplateExpr) which is the template expression to
// process.
// Takes ctx (*TdewolffRewriteContext) which holds the rewrite state.
func rewriteTdewolffTemplateExpr(node *parsejs.TemplateExpr, ctx *TdewolffRewriteContext) {
	for i := range node.List {
		rewriteTdewolffExpr(node.List[i].Expr, false, ctx)
	}
}

// rewriteTdewolffVar changes variable references to use the
// component context.
//
// When inside a class method, this rewrites instance property
// references that are not in scope from their bare name to
// "this.$$ctx.<name>" format.
//
// Takes v (*parsejs.Var) which is the variable AST node to
// rewrite.
// Takes isLeft (bool) which shows if this is a left-hand side
// assignment.
// Takes ctx (*TdewolffRewriteContext) which provides the
// rewriting context.
func rewriteTdewolffVar(v *parsejs.Var, isLeft bool, ctx *TdewolffRewriteContext) {
	if !ctx.inClassMethod {
		return
	}
	if isLeft {
		return
	}
	name := string(v.Name())

	if ctx.builtInNames[name] {
		return
	}
	if tdewolffIsNameInScope(name, ctx) {
		return
	}
	if !ctx.instanceProperties[name] {
		return
	}

	v.Data = []byte("this.$$ctx." + name)
}

// rewriteTdewolffVarDecl handles a variable declaration node by
// binding each declared variable and rewriting any default value
// expressions.
//
// Takes node (*parsejs.VarDecl) which is the variable
// declaration to process.
// Takes ctx (*TdewolffRewriteContext) which holds the rewrite state.
func rewriteTdewolffVarDecl(node *parsejs.VarDecl, ctx *TdewolffRewriteContext) {
	for i := range node.List {
		tdewolffBindDeclaration(node.List[i].Binding, ctx)
		if node.List[i].Default != nil {
			rewriteTdewolffExpr(node.List[i].Default, false, ctx)
		}
	}
}

// rewriteTdewolffFuncDecl processes a function declaration
// node. It adds the function name to the current scope, creates
// a new scope for the function body, binds each parameter,
// rewrites all body statements, then removes the scope.
//
// Takes node (*parsejs.FuncDecl) which is the function
// declaration to process.
// Takes ctx (*TdewolffRewriteContext) which holds the rewrite
// state and scopes.
func rewriteTdewolffFuncDecl(node *parsejs.FuncDecl, ctx *TdewolffRewriteContext) {
	if node.Name != nil {
		tdewolffAddNameToScope(string(node.Name.Name()), ctx)
	}
	tdewolffPushScope(ctx)
	for i := range node.Params.List {
		tdewolffBindDeclaration(node.Params.List[i].Binding, ctx)
	}
	for _, statement := range node.Body.List {
		rewriteTdewolffStmt(statement, ctx)
	}
	tdewolffPopScope(ctx)
}

// rewriteTdewolffArrowFunc handles an arrow function node by
// binding its parameters and rewriting its body statements
// within a new scope.
//
// Takes node (*parsejs.ArrowFunc) which is the arrow function
// to process.
// Takes ctx (*TdewolffRewriteContext) which tracks the rewrite
// state.
func rewriteTdewolffArrowFunc(node *parsejs.ArrowFunc, ctx *TdewolffRewriteContext) {
	tdewolffPushScope(ctx)
	for i := range node.Params.List {
		tdewolffBindDeclaration(node.Params.List[i].Binding, ctx)
	}
	for _, statement := range node.Body.List {
		rewriteTdewolffStmt(statement, ctx)
	}
	tdewolffPopScope(ctx)
}

// rewriteTdewolffClassDecl processes a class declaration node
// by adding its name to the scope and rewriting each class
// element.
//
// Takes node (*parsejs.ClassDecl) which is the class
// declaration to process.
// Takes ctx (*TdewolffRewriteContext) which provides the
// rewrite context.
func rewriteTdewolffClassDecl(node *parsejs.ClassDecl, ctx *TdewolffRewriteContext) {
	if node.Name != nil {
		tdewolffAddNameToScope(string(node.Name.Name()), ctx)
	}
	for i := range node.List {
		rewriteTdewolffClassElement(&node.List[i], ctx)
	}
}

// rewriteTdewolffClassElement processes a single class element
// by rewriting its method declaration or field initialiser
// expression.
//
// Takes element (*parsejs.ClassElement) which is the class element
// to process.
// Takes ctx (*TdewolffRewriteContext) which provides the
// rewrite context.
func rewriteTdewolffClassElement(element *parsejs.ClassElement, ctx *TdewolffRewriteContext) {
	if element.Method != nil {
		rewriteTdewolffMethodDecl(element.Method, ctx)
	}
	if element.Init != nil {
		rewriteTdewolffExpr(element.Init, false, ctx)
	}
}

// rewriteTdewolffMethodDecl processes a method declaration
// within a class by binding its parameters and rewriting its
// body statements.
//
// Takes node (*parsejs.MethodDecl) which is the method
// declaration to process.
// Takes ctx (*TdewolffRewriteContext) which holds the rewrite state.
func rewriteTdewolffMethodDecl(node *parsejs.MethodDecl, ctx *TdewolffRewriteContext) {
	wasInClassMethod := ctx.inClassMethod
	ctx.inClassMethod = true

	tdewolffPushScope(ctx)
	for i := range node.Params.List {
		tdewolffBindDeclaration(node.Params.List[i].Binding, ctx)
	}
	for _, statement := range node.Body.List {
		rewriteTdewolffStmt(statement, ctx)
	}
	tdewolffPopScope(ctx)

	ctx.inClassMethod = wasInClassMethod
}

// rewriteTdewolffForStmt rewrites a JavaScript for statement AST node.
//
// Takes node (*parsejs.ForStmt) which is the for statement to rewrite.
// Takes ctx (*TdewolffRewriteContext) which holds the rewrite state.
func rewriteTdewolffForStmt(node *parsejs.ForStmt, ctx *TdewolffRewriteContext) {
	if node.Init != nil {
		if varDecl, ok := node.Init.(*parsejs.VarDecl); ok {
			rewriteTdewolffVarDecl(varDecl, ctx)
		} else {
			rewriteTdewolffExpr(node.Init, false, ctx)
		}
	}
	if node.Cond != nil {
		rewriteTdewolffExpr(node.Cond, false, ctx)
	}
	if node.Post != nil {
		rewriteTdewolffExpr(node.Post, false, ctx)
	}
	rewriteTdewolffStmt(node.Body, ctx)
}

// rewriteTdewolffForInStmt rewrites a for-in statement during
// AST processing.
//
// Takes node (*parsejs.ForInStmt) which is the for-in
// statement to rewrite.
// Takes ctx (*TdewolffRewriteContext) which provides the rewrite context.
func rewriteTdewolffForInStmt(node *parsejs.ForInStmt, ctx *TdewolffRewriteContext) {
	if varDecl, ok := node.Init.(*parsejs.VarDecl); ok {
		rewriteTdewolffVarDecl(varDecl, ctx)
	} else {
		rewriteTdewolffExpr(node.Init, true, ctx)
	}
	rewriteTdewolffExpr(node.Value, false, ctx)
	rewriteTdewolffStmt(node.Body, ctx)
}

// rewriteTdewolffForOfStmt rewrites a JavaScript for-of statement by
// processing its initialiser, value expression, and body.
//
// Takes node (*parsejs.ForOfStmt) which is the for-of
// statement to rewrite.
// Takes ctx (*TdewolffRewriteContext) which provides the rewrite context.
func rewriteTdewolffForOfStmt(node *parsejs.ForOfStmt, ctx *TdewolffRewriteContext) {
	if varDecl, ok := node.Init.(*parsejs.VarDecl); ok {
		rewriteTdewolffVarDecl(varDecl, ctx)
	} else {
		rewriteTdewolffExpr(node.Init, true, ctx)
	}
	rewriteTdewolffExpr(node.Value, false, ctx)
	rewriteTdewolffStmt(node.Body, ctx)
}

// rewriteTdewolffSwitchCase rewrites a switch case clause by processing its
// condition expression and body statements.
//
// Takes sc (*parsejs.CaseClause) which is the case clause to rewrite.
// Takes ctx (*TdewolffRewriteContext) which provides the rewrite context.
func rewriteTdewolffSwitchCase(sc *parsejs.CaseClause, ctx *TdewolffRewriteContext) {
	if sc.Cond != nil {
		rewriteTdewolffExpr(sc.Cond, false, ctx)
	}
	for _, statement := range sc.List {
		rewriteTdewolffStmt(statement, ctx)
	}
}

// tdewolffBindDeclaration registers a variable binding in the
// current scope. It handles simple variables, array
// destructuring, and object destructuring patterns from the
// tdewolff parser.
//
// When binding is nil, returns straight away without action.
//
// Takes binding (parsejs.IBinding) which is the variable
// binding to register.
// Takes ctx (*TdewolffRewriteContext) which provides the
// current scope context.
func tdewolffBindDeclaration(binding parsejs.IBinding, ctx *TdewolffRewriteContext) {
	if binding == nil {
		return
	}
	switch typed := binding.(type) {
	case *parsejs.Var:
		tdewolffAddNameToScope(string(typed.Name()), ctx)
	case *parsejs.BindingArray:
		tdewolffBindArrayElements(typed, ctx)
	case *parsejs.BindingObject:
		tdewolffBindObjectProperties(typed, ctx)
	}
}

// tdewolffBindArrayElements binds each element in an array
// binding pattern.
//
// Takes arr (*parsejs.BindingArray) which is the array pattern
// to process.
// Takes ctx (*TdewolffRewriteContext) which tracks binding
// declarations.
func tdewolffBindArrayElements(arr *parsejs.BindingArray, ctx *TdewolffRewriteContext) {
	for _, item := range arr.List {
		if item.Binding != nil {
			tdewolffBindDeclaration(item.Binding, ctx)
		}
	}
	if arr.Rest != nil {
		tdewolffBindDeclaration(arr.Rest, ctx)
	}
}

// tdewolffBindObjectProperties binds all property declarations
// within a destructuring object pattern.
//
// Takes objectBinding (*parsejs.BindingObject) which contains the object
// binding pattern to process.
// Takes ctx (*TdewolffRewriteContext) which provides the rewrite context.
func tdewolffBindObjectProperties(objectBinding *parsejs.BindingObject, ctx *TdewolffRewriteContext) {
	for _, item := range objectBinding.List {
		if item.Value.Binding != nil {
			tdewolffBindDeclaration(item.Value.Binding, ctx)
		}
	}
	if objectBinding.Rest != nil {
		tdewolffBindDeclaration(objectBinding.Rest, ctx)
	}
}

// tdewolffPushScope pushes a new empty scope onto the scope stack.
//
// Takes ctx (*TdewolffRewriteContext) which holds the scope
// stack to modify.
func tdewolffPushScope(ctx *TdewolffRewriteContext) {
	ctx.scopes = append(ctx.scopes, make(map[string]bool))
}

// tdewolffPopScope removes the top scope from the rewrite context.
//
// Takes ctx (*TdewolffRewriteContext) which holds the current
// scope stack.
func tdewolffPopScope(ctx *TdewolffRewriteContext) {
	if len(ctx.scopes) > 0 {
		ctx.scopes = ctx.scopes[:len(ctx.scopes)-1]
	}
}

// tdewolffAddNameToScope adds a name to the current scope in the rewrite
// context.
//
// Takes name (string) which is the identifier to add.
// Takes ctx (*TdewolffRewriteContext) which holds the scope stack.
func tdewolffAddNameToScope(name string, ctx *TdewolffRewriteContext) {
	if len(ctx.scopes) > 0 {
		ctx.scopes[len(ctx.scopes)-1][name] = true
	}
}

// tdewolffIsNameInScope checks whether a variable name exists in any active
// scope.
//
// Takes name (string) which is the variable name to search
// for.
// Takes ctx (*TdewolffRewriteContext) which holds the stack of
// active scopes.
//
// Returns bool which is true if the name is found in any scope, false
// otherwise.
func tdewolffIsNameInScope(name string, ctx *TdewolffRewriteContext) bool {
	for i := len(ctx.scopes) - 1; i >= 0; i-- {
		if ctx.scopes[i][name] {
			return true
		}
	}
	return false
}

// isAssignOp checks if a token is a JavaScript assignment operator.
//
// Takes op (parsejs.TokenType) which is the token type to check.
//
// Returns bool which is true if the token is an assignment operator.
func isAssignOp(op parsejs.TokenType) bool {
	switch op {
	case parsejs.EqToken, parsejs.AddEqToken, parsejs.SubEqToken,
		parsejs.MulEqToken, parsejs.DivEqToken, parsejs.ModEqToken,
		parsejs.ExpEqToken, parsejs.LtLtEqToken, parsejs.GtGtEqToken,
		parsejs.GtGtGtEqToken, parsejs.BitAndEqToken, parsejs.BitOrEqToken,
		parsejs.BitXorEqToken, parsejs.NullishEqToken, parsejs.AndEqToken,
		parsejs.OrEqToken:
		return true
	default:
		return false
	}
}
