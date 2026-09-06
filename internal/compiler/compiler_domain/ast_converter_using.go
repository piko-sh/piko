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
	"errors"
	"fmt"
	"strconv"

	"github.com/tdewolff/parse/v2"
	parsejs "github.com/tdewolff/parse/v2/js"
	"piko.sh/piko/internal/esbuild/js_ast"
)

const (
	// usingScopePrefix names the variable that holds a block's disposable resources.
	usingScopePrefix = "$$using"

	// usingErrorPrefix names the catch binding that records an error thrown by a block
	// holding using declarations.
	usingErrorPrefix = "$$usingError"

	// usingValuePrefix names the loop variable a for-of with a using declaration reads each
	// value into before it is registered for disposal.
	usingValuePrefix = "$$usingValue"

	// usingScopeHelper is the helper that creates an empty disposal scope.
	usingScopeHelper = "$$pikoUsingScope"

	// usingAddHelper is the helper that registers a declared value for disposal.
	usingAddHelper = "$$pikoUsingAdd"

	// usingFailHelper is the helper that records an error thrown inside a disposal scope.
	usingFailHelper = "$$pikoUsingFail"

	// usingDisposeHelper is the helper that disposes a scope's resources in reverse order
	// and rethrows any recorded error.
	usingDisposeHelper = "$$pikoUsingDispose"

	// usingHelpersSource defines the helpers that lowered using declarations call.
	//
	// Disposal follows the explicit resource management proposal. Resources are disposed in
	// reverse order when the block exits, a value that is null or undefined is skipped, an
	// await using value falls back to its Symbol.dispose method, and an error thrown while
	// disposing is combined with any earlier one as a SuppressedError. The well-known
	// symbols fall back to Symbol.for so resources still work where the runtime does not
	// define them.
	usingHelpersSource = `function $$pikoUsingScope() {
	return { stack: [], error: undefined, hasError: false };
}
function $$pikoUsingSymbol(name) {
	return Symbol[name] || Symbol.for("Symbol." + name);
}
function $$pikoUsingAdd(scope, value, isAsync) {
	if (value === null || value === undefined) {
		if (isAsync) {
			scope.stack.push({ value: value, dispose: undefined, isAsync: true });
		}
		return value;
	}
	if (typeof value !== "object" && typeof value !== "function") {
		throw new TypeError("a using declaration needs an object, a function, null or undefined");
	}
	let dispose;
	if (isAsync) {
		dispose = value[$$pikoUsingSymbol("asyncDispose")];
	}
	if (dispose === undefined) {
		const syncDispose = value[$$pikoUsingSymbol("dispose")];
		dispose = isAsync && typeof syncDispose === "function" ? function () { syncDispose.call(this); } : syncDispose;
	}
	if (typeof dispose !== "function") {
		throw new TypeError("the value of a using declaration is not disposable");
	}
	scope.stack.push({ value: value, dispose: dispose, isAsync: isAsync });
	return value;
}
function $$pikoUsingFail(scope, error) {
	scope.error = scope.hasError ? $$pikoUsingSuppressed(error, scope.error) : error;
	scope.hasError = true;
}
function $$pikoUsingSuppressed(error, suppressed) {
	if (typeof SuppressedError === "function") {
		return new SuppressedError(error, suppressed, "an error was suppressed during disposal");
	}
	const combined = new Error("an error was suppressed during disposal");
	combined.name = "SuppressedError";
	combined.error = error;
	combined.suppressed = suppressed;
	return combined;
}
function $$pikoUsingDispose(scope) {
	while (scope.stack.length > 0) {
		const resource = scope.stack.pop();
		if (resource.isAsync) {
			return $$pikoUsingDisposeAsync(scope, resource);
		}
		try {
			resource.dispose.call(resource.value);
		} catch (error) {
			$$pikoUsingFail(scope, error);
		}
	}
	if (scope.hasError) {
		throw scope.error;
	}
}
async function $$pikoUsingDisposeAsync(scope, resource) {
	for (; resource !== undefined; resource = scope.stack.pop()) {
		try {
			const result = resource.dispose === undefined ? undefined : resource.dispose.call(resource.value);
			if (resource.isAsync) {
				await result;
			}
		} catch (error) {
			$$pikoUsingFail(scope, error);
		}
	}
	if (scope.hasError) {
		throw scope.error;
	}
}`
)

var (
	// errUsingUnsupported is returned for a using declaration in a position the compiler
	// cannot lower, so it fails the build instead of losing its disposal.
	errUsingUnsupported = errors.New("using declaration cannot be compiled here")
)

// usingScope is one block whose using declarations share a disposal scope.
type usingScope struct {
	// name is the variable that holds the block's disposal scope.
	name string

	// errorName is the catch binding that records an error thrown by the block.
	errorName string

	// valueName is the loop variable of a for-of whose declaration is a using one.
	valueName string

	// isAsync is true when the block holds an await using declaration, so its disposal is
	// awaited.
	isAsync bool
}

// convertStatementList converts the statements of one block, function body or static
// block.
//
// Each using declaration, which tdewolff cannot print, becomes a const whose value is
// registered with a disposal scope, and the block's statements run inside a try whose
// finally disposes the registered values in reverse order.
//
// Takes statements ([]js_ast.Stmt) which are the statements of the block.
//
// Returns []parsejs.IStmt which holds the converted statements.
// Returns error when a statement cannot be converted.
func (c *ASTConverter) convertStatementList(statements []js_ast.Stmt) ([]parsejs.IStmt, error) {
	hasUsing, isAsync := scanUsingDeclarations(statements)
	if !hasUsing {
		return c.convertStatements(statements, nil)
	}

	scope := c.newUsingScope(isAsync)
	converted, err := c.convertStatements(statements, &scope)
	if err != nil {
		return nil, err
	}
	return scope.wrap(converted), nil
}

// convertStatements converts a list of statements in order, dropping any that convert to
// nothing.
//
// Takes statements ([]js_ast.Stmt) which are the statements to convert.
// Takes scope (*usingScope) which receives the list's using declarations; nil when the
// list holds none.
//
// Returns []parsejs.IStmt which holds the converted statements.
// Returns error when a statement cannot be converted.
func (c *ASTConverter) convertStatements(statements []js_ast.Stmt, scope *usingScope) ([]parsejs.IStmt, error) {
	converted := make([]parsejs.IStmt, 0, len(statements))
	for i, statement := range statements {
		var result parsejs.IStmt
		var err error
		if local, ok := statement.Data.(*js_ast.SLocal); ok && local.Kind.IsUsing() && scope != nil {
			result, err = c.convertUsingLocal(local, *scope)
		} else {
			result, err = c.convertStatement(statement)
		}
		if err != nil {
			return nil, fmt.Errorf("converting statement %d: %w", i, err)
		}
		if result != nil {
			converted = append(converted, result)
		}
	}
	return converted, nil
}

// convertUsingLocal lowers a using or await using declaration to a const whose value is
// registered with the block's disposal scope.
//
// Takes s (*js_ast.SLocal) which is the using declaration.
// Takes scope (usingScope) which is the disposal scope of the enclosing block.
//
// Returns parsejs.IStmt which is the lowered declaration.
// Returns error when a binding or value cannot be converted.
func (c *ASTConverter) convertUsingLocal(s *js_ast.SLocal, scope usingScope) (parsejs.IStmt, error) {
	isAsync := s.Kind == js_ast.LocalAwaitUsing
	bindings := make([]parsejs.BindingElement, 0, len(s.Decls))
	for _, declaration := range s.Decls {
		binding, err := c.convertBinding(declaration.Binding)
		if err != nil {
			return nil, fmt.Errorf("converting using declaration binding: %w", err)
		}
		value, err := c.convertExpression(declaration.ValueOrNil)
		if err != nil {
			return nil, fmt.Errorf("converting using declaration value: %w", err)
		}
		if binding == nil || value == nil {
			return nil, fmt.Errorf("%w: a using declaration needs a name and a value", errUsingUnsupported)
		}
		bindings = append(bindings, parsejs.BindingElement{
			Binding: binding,
			Default: usingAddCall(scope, value, isAsync),
		})
	}
	return &parsejs.VarDecl{TokenType: parsejs.ConstToken, List: bindings}, nil
}

// convertForOfUsing lowers a for-of loop whose declaration is a using one, so each value
// is disposed when its iteration ends.
//
// Takes s (*js_ast.SForOf) which is the loop.
// Takes local (*js_ast.SLocal) which is the loop's using declaration.
//
// Returns parsejs.IStmt which is the lowered loop.
// Returns error when the declaration, the iterated value or the body cannot be converted.
func (c *ASTConverter) convertForOfUsing(s *js_ast.SForOf, local *js_ast.SLocal) (parsejs.IStmt, error) {
	if len(local.Decls) != 1 {
		return nil, fmt.Errorf("%w: a for-of using declaration declares exactly one name", errUsingUnsupported)
	}
	binding, err := c.convertBinding(local.Decls[0].Binding)
	if err != nil {
		return nil, fmt.Errorf("converting for-of using binding: %w", err)
	}
	value, err := c.convertExpression(s.Value)
	if err != nil {
		return nil, fmt.Errorf("converting for-of value: %w", err)
	}
	body, err := c.convertStatement(s.Body)
	if err != nil {
		return nil, fmt.Errorf("converting for-of body: %w", err)
	}

	scope := c.newUsingScope(local.Kind == js_ast.LocalAwaitUsing)
	iteration := []parsejs.IStmt{&parsejs.VarDecl{
		TokenType: parsejs.ConstToken,
		List: []parsejs.BindingElement{{
			Binding: binding,
			Default: usingAddCall(scope, &parsejs.Var{Data: []byte(scope.valueName)}, scope.isAsync),
		}},
	}}
	if body != nil {
		iteration = append(iteration, body)
	}

	return &parsejs.ForOfStmt{
		Await: s.Await.Len > 0,
		Init: &parsejs.VarDecl{
			TokenType: parsejs.ConstToken,
			List:      []parsejs.BindingElement{{Binding: &parsejs.Var{Data: []byte(scope.valueName)}}},
		},
		Value: value,
		Body:  &parsejs.BlockStmt{List: scope.wrap(iteration)},
	}, nil
}

// convertForUsing lowers a for loop whose initialiser is a using declaration, so the
// declared values are disposed once the loop ends.
//
// The loop runs inside the disposal scope with its initialiser moved in front of it, and
// keeps its label so a labelled continue still names an iteration statement.
//
// Takes s (*js_ast.SFor) which is the loop.
// Takes local (*js_ast.SLocal) which is the loop's using declaration.
// Takes label ([]byte) which is the loop's label; nil when it has none.
//
// Returns parsejs.IStmt which is a block holding the scope and the loop.
// Returns error when the declaration or the loop cannot be converted.
func (c *ASTConverter) convertForUsing(s *js_ast.SFor, local *js_ast.SLocal, label []byte) (parsejs.IStmt, error) {
	scope := c.newUsingScope(local.Kind == js_ast.LocalAwaitUsing)
	declaration, err := c.convertUsingLocal(local, scope)
	if err != nil {
		return nil, fmt.Errorf("converting for-loop using declaration: %w", err)
	}

	withoutInit := *s
	withoutInit.InitOrNil = js_ast.Stmt{}
	loop, err := c.convertSFor(&withoutInit)
	if err != nil {
		return nil, err
	}
	if label != nil {
		loop = &parsejs.LabelledStmt{Label: label, Value: loop}
	}

	return &parsejs.BlockStmt{List: scope.wrap([]parsejs.IStmt{declaration, loop})}, nil
}

// newUsingScope starts a disposal scope with names unique within this conversion and
// notes that the disposal helpers must be emitted.
//
// Takes isAsync (bool) which is true when the scope's disposal is awaited.
//
// Returns usingScope which names the scope's variables.
func (c *ASTConverter) newUsingScope(isAsync bool) usingScope {
	suffix := strconv.Itoa(c.usingScopeCount)
	c.usingScopeCount++
	c.usesDisposal = true
	return usingScope{
		name:      usingScopePrefix + suffix,
		errorName: usingErrorPrefix + suffix,
		valueName: usingValuePrefix + suffix,
		isAsync:   isAsync,
	}
}

// disposalHelpers returns the helper declarations that lowered using declarations call,
// or nothing when the conversion lowered none.
//
// Returns []parsejs.IStmt which holds the helper function declarations.
// Returns error when the helper source cannot be parsed.
func (c *ASTConverter) disposalHelpers() ([]parsejs.IStmt, error) {
	if !c.usesDisposal {
		return nil, nil
	}
	helpers, err := parsejs.Parse(parse.NewInputString(usingHelpersSource), parsejs.Options{})
	if err != nil {
		return nil, fmt.Errorf("parsing the using declaration helpers: %w", err)
	}
	return helpers.List, nil
}

// wrap runs a block's statements inside the scope.
//
// The scope is created first, the statements run in a try that records any error, and the
// finally disposes the scope's resources; an async scope's disposal is awaited.
//
// Takes statements ([]parsejs.IStmt) which are the block's converted statements.
//
// Returns []parsejs.IStmt which holds the scope declaration and the try statement.
func (s usingScope) wrap(statements []parsejs.IStmt) []parsejs.IStmt {
	var dispose parsejs.IExpr = helperCall(usingDisposeHelper, &parsejs.Var{Data: []byte(s.name)})
	if s.isAsync {
		dispose = &parsejs.UnaryExpr{Op: parsejs.AwaitToken, X: dispose}
	}

	return []parsejs.IStmt{
		&parsejs.VarDecl{
			TokenType: parsejs.ConstToken,
			List: []parsejs.BindingElement{{
				Binding: &parsejs.Var{Data: []byte(s.name)},
				Default: helperCall(usingScopeHelper),
			}},
		},
		&parsejs.TryStmt{
			Body:    &parsejs.BlockStmt{List: statements},
			Binding: &parsejs.Var{Data: []byte(s.errorName)},
			Catch: &parsejs.BlockStmt{List: []parsejs.IStmt{&parsejs.ExprStmt{Value: helperCall(
				usingFailHelper,
				&parsejs.Var{Data: []byte(s.name)},
				&parsejs.Var{Data: []byte(s.errorName)},
			)}}},
			Finally: &parsejs.BlockStmt{List: []parsejs.IStmt{&parsejs.ExprStmt{Value: dispose}}},
		},
	}
}

// scanUsingDeclarations reports whether a block directly holds using declarations.
//
// Takes statements ([]js_ast.Stmt) which are the block's statements.
//
// Returns hasUsing (bool) which is true when any statement is a using declaration.
// Returns isAsync (bool) which is true when any of them is an await using declaration.
func scanUsingDeclarations(statements []js_ast.Stmt) (hasUsing bool, isAsync bool) {
	for _, statement := range statements {
		local, ok := statement.Data.(*js_ast.SLocal)
		if !ok || !local.Kind.IsUsing() {
			continue
		}
		hasUsing = true
		if local.Kind == js_ast.LocalAwaitUsing {
			isAsync = true
		}
	}
	return hasUsing, isAsync
}

// forWithUsingInit reports whether a statement is a for loop whose initialiser is a using
// declaration.
//
// Takes statement (js_ast.Stmt) which is the statement to inspect.
//
// Returns *js_ast.SFor which is the loop.
// Returns *js_ast.SLocal which is its using declaration.
// Returns bool which is true when the statement is such a loop.
func forWithUsingInit(statement js_ast.Stmt) (*js_ast.SFor, *js_ast.SLocal, bool) {
	loop, ok := statement.Data.(*js_ast.SFor)
	if !ok {
		return nil, nil, false
	}
	local, ok := loop.InitOrNil.Data.(*js_ast.SLocal)
	if !ok || !local.Kind.IsUsing() {
		return nil, nil, false
	}
	return loop, local, true
}

// usingAddCall builds the call that registers a declared value with a disposal scope.
//
// Takes scope (usingScope) which is the scope that disposes the value.
// Takes value (parsejs.IExpr) which is the declared value.
// Takes isAsync (bool) which is true for an await using declaration.
//
// Returns parsejs.IExpr which is the call, evaluating to value.
func usingAddCall(scope usingScope, value parsejs.IExpr, isAsync bool) parsejs.IExpr {
	flag := parsejs.FalseToken
	flagText := "false"
	if isAsync {
		flag = parsejs.TrueToken
		flagText = "true"
	}
	return helperCall(usingAddHelper,
		&parsejs.Var{Data: []byte(scope.name)},
		value,
		&parsejs.LiteralExpr{TokenType: flag, Data: []byte(flagText)},
	)
}

// helperCall builds a call to a named helper function.
//
// Takes name (string) which is the helper's name.
// Takes arguments (...parsejs.IExpr) which are the call's arguments.
//
// Returns *parsejs.CallExpr which is the call.
func helperCall(name string, arguments ...parsejs.IExpr) *parsejs.CallExpr {
	list := make([]parsejs.Arg, 0, len(arguments))
	for _, argument := range arguments {
		list = append(list, parsejs.Arg{Value: argument})
	}
	return &parsejs.CallExpr{X: &parsejs.Var{Data: []byte(name)}, Args: parsejs.Args{List: list}}
}
