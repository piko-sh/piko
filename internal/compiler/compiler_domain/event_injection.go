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
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/logger/logger_domain"
)

// eventBinding holds a compiled event handler with its generated AST statement.
type eventBinding struct {
	// Expression is the JavaScript AST statement for this event binding.
	Expression js_ast.Stmt

	// JSPropValue is the expression to bind as a JavaScript property value.
	JSPropValue js_ast.Expr

	// EventName is the name of the event that triggers this binding.
	EventName string

	// Index is the position of this binding in the bindings list.
	Index int

	// IsFrameworkHandler indicates whether this binding is handled by the
	// framework.
	IsFrameworkHandler bool

	// IsHOF indicates whether this binding wraps a higher-order function.
	IsHOF bool
}

// eventBindingCollection holds event bindings for a component during
// compilation.
type eventBindingCollection struct {
	// registry provides symbol lookup and type resolution for AST expressions.
	registry *RegistryContext

	// bindings holds the event bindings to add to the constructor.
	bindings []eventBinding

	// nextBindingIndex is the counter used to generate unique binding suffixes.
	nextBindingIndex int
}

// getBindings returns the collected event bindings.
//
// Returns []eventBinding which contains all registered event bindings.
func (ec *eventBindingCollection) getBindings() []eventBinding {
	return ec.bindings
}

// getRegistry returns the registry context.
//
// Returns *RegistryContext which provides access to the shared registry.
func (ec *eventBindingCollection) getRegistry() *RegistryContext {
	return ec.registry
}

// eventHandlerNames groups the naming parameters used when constructing
// event handler bindings.
type eventHandlerNames struct {
	// safeEventName is the sanitised event name used in handler identifiers.
	safeEventName string

	// safeUserMethod is the sanitised method name used when building handlers.
	safeUserMethod string

	// suffix is added to the handler name to make it unique.
	suffix string

	// rawEventName is the original event name before any processing.
	rawEventName string
}

// astBindingOptions groups optional parameters for creating an AST-based
// event binding.
type astBindingOptions struct {
	// userArgs contains the arguments passed to the handler.
	userArgs []js_ast.Expr

	// loopVarNames lists the loop variable names in scope.
	loopVarNames []string

	// directFrameworkBody is the framework handler body, or nil for user
	// method calls.
	directFrameworkBody *js_ast.SBlock

	// eventModifiers contains the user-facing event modifiers.
	eventModifiers []string
}

// eventBindingParams holds the data needed to create an event binding.
type eventBindingParams struct {
	// rawEventName is the original event name before any processing.
	rawEventName string

	// rawUserMethod is the method string as given by the user before processing.
	rawUserMethod string

	// safeEventName is the sanitised event name used in handler identifiers.
	safeEventName string

	// safeUserMethod is the sanitised method name used when building handlers.
	safeUserMethod string

	// suffix is added to the handler name to make it unique.
	suffix string

	// directFrameworkBody contains the raw JavaScript handler body for framework
	// event bindings; empty string indicates a non-framework handler.
	directFrameworkBody string

	// isLoopContext indicates that the event is bound inside a loop.
	isLoopContext bool
}

// createAndStoreBinding creates and stores an event binding, returning a
// property access expression.
//
// Takes rawEventName (string) which is the original event name before
// sanitisation.
// Takes rawUserMethod (string) which is the method string as given by
// the user.
// Takes isLoopContext (bool) which indicates that the event is bound
// inside a loop.
// Takes directFrameworkBody (string) which contains the raw JavaScript
// handler body for framework event bindings, or empty for user method
// handlers.
//
// Returns js_ast.Expr which is a property access expression for the
// bound handler.
// Returns error when the handler code snippet cannot be parsed.
func (ec *eventBindingCollection) createAndStoreBinding(
	ctx context.Context,
	rawEventName string,
	rawUserMethod string,
	_ []js_ast.Expr,
	isLoopContext bool,
	_ []string,
	directFrameworkBody string,
) (js_ast.Expr, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, _ := l.Span(ctx, "eventBindingCollection.createAndStoreBinding")
	defer span.End()

	EventBindingCreationCount.Add(ctx, 1)

	ec.nextBindingIndex++
	params := eventBindingParams{
		rawEventName:        rawEventName,
		rawUserMethod:       rawUserMethod,
		safeEventName:       sanitiseForJSIdentifier(rawEventName),
		safeUserMethod:      sanitiseForJSIdentifier(rawUserMethod),
		suffix:              fmt.Sprintf("_evt_%d", ec.nextBindingIndex),
		isLoopContext:       isLoopContext,
		directFrameworkBody: directFrameworkBody,
	}

	handlerName, binding := ec.buildEventBinding(ctx, span, params)
	ec.bindings = append(ec.bindings, binding)

	return js_ast.Expr{Data: buildDotExpr(jsThis, handlerName, ec.registry)}, nil
}

// createAndStoreBindingAST creates and stores an event binding using AST
// structures directly.
//
// Takes rawEventName (string) which is the original event name before
// sanitisation.
// Takes rawUserMethod (string) which is the method string as given by
// the user.
// Takes opts (astBindingOptions) which groups the handler arguments,
// loop variable names, framework body, and event modifiers.
//
// Returns js_ast.Expr which is the expression for the bound handler.
// Returns error when the handler code snippet cannot be parsed.
func (ec *eventBindingCollection) createAndStoreBindingAST(
	ctx context.Context,
	rawEventName string,
	rawUserMethod string,
	opts astBindingOptions,
) (js_ast.Expr, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, _ := l.Span(ctx, "eventBindingCollection.createAndStoreBindingAST")
	defer span.End()

	EventBindingCreationCount.Add(ctx, 1)

	ec.nextBindingIndex++
	names := eventHandlerNames{
		safeEventName:  sanitiseForJSIdentifier(rawEventName),
		safeUserMethod: sanitiseForJSIdentifier(rawUserMethod),
		suffix:         fmt.Sprintf("_evt_%d", ec.nextBindingIndex),
		rawEventName:   rawEventName,
	}

	needsHOF := len(opts.userArgs) > 0 && usesLoopVar(opts.userArgs, opts.loopVarNames, ec.registry)

	if needsHOF {
		return ec.createHOFBinding(ctx, names, opts.userArgs, opts.loopVarNames, opts.eventModifiers)
	}

	return ec.createDirectBinding(ctx, names, opts.userArgs, opts.directFrameworkBody, opts.eventModifiers)
}

// createHOFBinding creates a higher-order function binding for event handlers
// that reference loop variables.
//
// This generates code like:
//
//	this._hof_click_fn_evt_1 = (item) => (e) => {
//		this.$$ctx.fn.call(this, item.id, e);
//	};
//
// And returns a call expression: this._hof_click_fn_evt_1(item)
//
// Takes names (eventHandlerNames) which groups the sanitised and raw
// event and method names together with the unique suffix.
// Takes userArgs ([]js_ast.Expr) which contains the arguments passed to
// the handler.
// Takes loopVarNames ([]string) which lists the loop variable names in
// scope.
// Takes eventModifiers ([]string) which contains the user-facing event
// modifiers.
//
// Returns js_ast.Expr which is the call expression for the HOF binding.
// Returns error when the handler code snippet cannot be parsed.
func (ec *eventBindingCollection) createHOFBinding(
	ctx context.Context,
	names eventHandlerNames,
	userArgs []js_ast.Expr,
	loopVarNames []string,
	eventModifiers []string,
) (js_ast.Expr, error) {
	handlerName := fmt.Sprintf("_hof_%s_%s%s", names.safeEventName, names.safeUserMethod, names.suffix)

	usedLoopVars := findUsedLoopVars(userArgs, loopVarNames, ec.registry)
	if len(usedLoopVars) == 0 {
		usedLoopVars = loopVarNames
	}

	argumentsString := encodeEventHandlerArgs(userArgs, loopVarNames, ec.registry)

	guards := buildModifierGuards(eventModifiers, handlerName)
	loopVarsString := strings.Join(usedLoopVars, ", ")
	handlerSnippet := fmt.Sprintf("this.%s = (%s) => (e) => { %sthis.$$ctx.%s.call(this, %s); };",
		handlerName, loopVarsString, guards, names.safeUserMethod, argumentsString)

	assignmentStmt, err := parseSnippetAsStatement(handlerSnippet)
	if err != nil {
		EventBindingCreationErrorCount.Add(ctx, 1)
		return js_ast.Expr{}, fmt.Errorf("failed to create HOF event binding: %w", err)
	}

	callExpr := buildHOFCallExpr(handlerName, usedLoopVars, ec.registry)

	binding := eventBinding{
		Index:              ec.nextBindingIndex,
		EventName:          names.rawEventName,
		IsFrameworkHandler: false,
		IsHOF:              true,
		Expression:         assignmentStmt,
		JSPropValue:        callExpr,
	}

	ec.bindings = append(ec.bindings, binding)
	return callExpr, nil
}

// createDirectBinding creates a direct event handler binding that is not a
// higher-order function.
//
// Takes names (eventHandlerNames) which groups the sanitised and raw
// event and method names together with the unique suffix.
// Takes userArgs ([]js_ast.Expr) which contains arguments from the user.
// Takes directFrameworkBody (*js_ast.SBlock) which is the framework
// handler body, or nil for user method calls.
// Takes eventModifiers ([]string) which contains the user-facing event
// modifiers.
//
// Returns js_ast.Expr which is a property access expression for the
// handler.
// Returns error when the handler snippet cannot be parsed.
func (ec *eventBindingCollection) createDirectBinding(
	ctx context.Context,
	names eventHandlerNames,
	userArgs []js_ast.Expr,
	directFrameworkBody *js_ast.SBlock,
	eventModifiers []string,
) (js_ast.Expr, error) {
	handlerName := fmt.Sprintf("_dir_%s_%s%s", names.safeEventName, names.safeUserMethod, names.suffix)
	guards := buildModifierGuards(eventModifiers, handlerName)
	var handlerSnippet string

	if directFrameworkBody != nil {
		bodyString := encodeBlockStatements(directFrameworkBody, ec.registry)
		handlerSnippet = fmt.Sprintf("this.%s = (event) => { %s%s };", handlerName, guards, bodyString)
	} else if userArgs == nil {
		handlerSnippet = fmt.Sprintf("this.%s = (e) => { %sthis.$$ctx.%s.call(this, e); };", handlerName, guards, names.safeUserMethod)
	} else if len(userArgs) == 0 {
		handlerSnippet = fmt.Sprintf("this.%s = (e) => { %sthis.$$ctx.%s.call(this); };", handlerName, guards, names.safeUserMethod)
	} else {
		argumentsString := encodeEventHandlerArgs(userArgs, nil, ec.registry)
		handlerSnippet = fmt.Sprintf("this.%s = (e) => { %sthis.$$ctx.%s.call(this, %s); };", handlerName, guards, names.safeUserMethod, argumentsString)
	}

	assignmentStmt, err := parseSnippetAsStatement(handlerSnippet)
	if err != nil {
		EventBindingCreationErrorCount.Add(ctx, 1)
		return js_ast.Expr{}, fmt.Errorf("failed to create event binding: %w", err)
	}

	binding := eventBinding{
		Index:              ec.nextBindingIndex,
		EventName:          names.rawEventName,
		IsFrameworkHandler: directFrameworkBody != nil,
		IsHOF:              false,
		Expression:         assignmentStmt,
		JSPropValue:        js_ast.Expr{Data: buildDotExpr(jsThis, handlerName, ec.registry)},
	}

	ec.bindings = append(ec.bindings, binding)
	propertyAccess := js_ast.Expr{Data: buildDotExpr(jsThis, handlerName, ec.registry)}
	return propertyAccess, nil
}

// buildEventBinding creates an event binding and returns the handler name
// and binding.
//
// Takes span (trace.Span) which provides tracing context for the operation.
// Takes params (eventBindingParams) which contains the binding settings.
//
// Returns string which is the generated handler name.
// Returns eventBinding which is the built binding ready for use.
func (ec *eventBindingCollection) buildEventBinding(
	ctx context.Context,
	span trace.Span,
	params eventBindingParams,
) (string, eventBinding) {
	isFwHandler := params.directFrameworkBody != ""
	finalLogicExpr := ec.buildHandlerLogic(ctx, span, params, isFwHandler)

	_ = astContainsVar(finalLogicExpr, "event")

	handlerName := fmt.Sprintf("_dir_%s_%s%s", params.safeEventName, params.safeUserMethod, params.suffix)
	assignmentStmt := ec.createHandlerAssignment(ctx, handlerName, params.safeUserMethod)

	return handlerName, eventBinding{
		Index:              ec.nextBindingIndex,
		EventName:          params.rawEventName,
		IsFrameworkHandler: isFwHandler,
		IsHOF:              params.isLoopContext && !isFwHandler,
		Expression:         assignmentStmt,
		JSPropValue:        js_ast.Expr{Data: buildDotExpr(jsThis, handlerName, ec.registry)},
	}
}

// buildHandlerLogic creates the statement that handles an event.
//
// Takes span (trace.Span) which is used for error reporting.
// Takes params (eventBindingParams) which contains the event binding details.
// Takes isFwHandler (bool) which indicates if this is a framework handler.
//
// Returns js_ast.Stmt which is the handler statement.
func (*eventBindingCollection) buildHandlerLogic(
	ctx context.Context,
	span trace.Span,
	params eventBindingParams,
	isFwHandler bool,
) js_ast.Stmt {
	ctx, l := logger_domain.From(ctx, log)
	if isFwHandler {
		statement, err := parseSnippetAsStatement(params.directFrameworkBody)
		if err != nil {
			EventBindingCreationErrorCount.Add(ctx, 1)
			l.ReportError(span, err, "Invalid framework body for event binding")
			errorCall := fmt.Sprintf(`console.error('Error in framework event binding for %s');`, params.rawEventName)
			statement, _ = parseSnippetAsStatement(errorCall)
		}
		return statement
	}

	callSnippet := fmt.Sprintf("this.$$ctx.%s.call(this, e);", params.safeUserMethod)
	statement, err := parseSnippetAsStatement(callSnippet)
	if err != nil {
		l.Warn("Failed to create event handler call", logger_domain.String("error", err.Error()))
	}
	return statement
}

// createHandlerAssignment creates the handler assignment statement.
//
// Takes handlerName (string) which is the property name for the handler.
// Takes safeUserMethod (string) which is the cleaned method name to call.
//
// Returns js_ast.Stmt which is the parsed assignment statement.
func (*eventBindingCollection) createHandlerAssignment(ctx context.Context, handlerName, safeUserMethod string) js_ast.Stmt {
	ctx, l := logger_domain.From(ctx, log)
	handlerSnippet := fmt.Sprintf("this.%s = (e) => { this.$$ctx.%s.call(this, e); };", handlerName, safeUserMethod)
	statement, err := parseSnippetAsStatement(handlerSnippet)
	if err != nil {
		l.Warn("Failed to create event handler", logger_domain.String("error", err.Error()))
	}
	return statement
}

// newEventBindingCollection creates a new event binding collection with the
// given registry.
//
// Takes registry (*RegistryContext) which provides the context for event
// registration.
//
// Returns *eventBindingCollection which is ready to accept bindings.
func newEventBindingCollection(registry *RegistryContext) *eventBindingCollection {
	return &eventBindingCollection{
		registry:         registry,
		bindings:         nil,
		nextBindingIndex: 0,
	}
}

// astContainsVar checks whether a JavaScript AST node contains any
// identifier reference. Accepts both statement and expression nodes.
//
// Takes node (any) which is the AST node to search, expected to be
// a js_ast.Stmt or js_ast.Expr.
//
// Returns bool which is true if the node contains an identifier.
func astContainsVar(node any, _ string) bool {
	switch n := node.(type) {
	case js_ast.Stmt:
		return statementContainsIdentifier(n)
	case js_ast.Expr:
		return expressionContainsIdentifier(n)
	}
	return false
}

// statementContainsIdentifier checks whether the statement contains an identifier.
//
// Takes statement (js_ast.Stmt) which is the statement to search.
//
// Returns bool which is true if any expression in the statement tree contains
// an identifier.
func statementContainsIdentifier(statement js_ast.Stmt) bool {
	found := false
	walkStmt(statement, func(s js_ast.Stmt) bool {
		if expression, ok := s.Data.(*js_ast.SExpr); ok && expressionContainsIdentifier(expression.Value) {
			found = true
			return false
		}
		return !found
	})
	return found
}

// expressionContainsIdentifier checks whether the given expression
// contains an identifier.
//
// Takes expression (js_ast.Expr) which is the expression to search.
//
// Returns bool which indicates whether an identifier was found.
func expressionContainsIdentifier(expression js_ast.Expr) bool {
	found := false
	walkExpr(expression, func(e js_ast.Expr) bool {
		if _, ok := e.Data.(*js_ast.EIdentifier); ok {
			found = true
			return false
		}
		return !found
	})
	return found
}

// injectEventBindingsIntoConstructor adds event binding assignments into a
// class constructor.
//
// Takes classDecl (*js_ast.Class) which is the class declaration to modify.
// Takes ec (*eventBindingCollection) which holds the event bindings to inject.
//
// Returns error when the constructor cannot be found or is invalid.
func injectEventBindingsIntoConstructor(
	ctx context.Context,
	classDecl *js_ast.Class,
	ec *eventBindingCollection,
) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "injectEventBindingsIntoConstructor")
	defer span.End()

	EventBindingInjectionCount.Add(ctx, 1)
	startTime := time.Now()

	if ec == nil || len(ec.bindings) == 0 {
		l.Trace("No event bindings to inject.")
		return nil
	}

	constructor, err := findAndValidateConstructor(ctx, classDecl, ec.getRegistry())
	if err != nil {
		l.ReportError(span, err, "Failed to find constructor method")
		EventBindingInjectionErrorCount.Add(ctx, 1)
		return fmt.Errorf("finding constructor for event binding injection: %w", err)
	}

	newStmts := collectBindingStatements(ec)
	insertBindingsIntoConstructor(ctx, constructor, newStmts)
	recordInjectionMetrics(ctx, span, startTime, len(newStmts))

	return nil
}

// findAndValidateConstructor finds the constructor method in the given class
// and checks that it exists.
//
// Takes classDecl (*js_ast.Class) which is the class to search for a
// constructor.
// Takes registry (*RegistryContext) which provides the registry context for
// lookup.
//
// Returns *js_ast.EFunction which is the constructor method if found.
// Returns error when the class has no constructor after standardisation.
func findAndValidateConstructor(_ context.Context, classDecl *js_ast.Class, registry *RegistryContext) (*js_ast.EFunction, error) {
	constructor := findConstructorMethod(classDecl, registry)
	if constructor == nil {
		return nil, errors.New("no constructor found in class after standardisation")
	}
	return constructor, nil
}

// collectBindingStatements gathers statements from event bindings.
//
// Takes ec (*eventBindingCollection) which holds the bindings to process.
//
// Returns []js_ast.Stmt which contains the expression statements from each
// binding.
func collectBindingStatements(ec *eventBindingCollection) []js_ast.Stmt {
	statements := make([]js_ast.Stmt, 0, len(ec.bindings))
	for _, b := range ec.bindings {
		statements = append(statements, b.Expression)
	}
	return statements
}

// insertBindingsIntoConstructor adds binding statements to a constructor
// function body, placing them after super() and init() calls.
//
// Takes constructor (*js_ast.EFunction) which is the constructor to modify.
// Takes newStmts ([]js_ast.Stmt) which contains the binding statements to add.
func insertBindingsIntoConstructor(ctx context.Context, constructor *js_ast.EFunction, newStmts []js_ast.Stmt) {
	ctx, l := logger_domain.From(ctx, log)
	body := constructor.Fn.Body.Block.Stmts
	if len(body) < 2 {
		l.Trace("Constructor body < 2 statements, appending event bindings at end",
			logger_domain.Int("bodyLen", len(body)),
		)
		constructor.Fn.Body.Block.Stmts = append(constructor.Fn.Body.Block.Stmts, newStmts...)
		return
	}

	const insertPosition = 2
	updated := make([]js_ast.Stmt, 0, len(body)+len(newStmts))
	updated = append(updated, body[:insertPosition]...)
	updated = append(updated, newStmts...)
	if len(body) > insertPosition {
		updated = append(updated, body[insertPosition:]...)
	}
	constructor.Fn.Body.Block.Stmts = updated
}

// recordInjectionMetrics records metrics for the event binding injection.
//
// Takes span (trace.Span) which receives duration and binding count values.
// Takes startTime (time.Time) which marks when injection started.
// Takes bindingCount (int) which is the number of bindings injected.
func recordInjectionMetrics(ctx context.Context, span trace.Span, startTime time.Time, bindingCount int) {
	ctx, l := logger_domain.From(ctx, log)
	duration := time.Since(startTime)
	EventBindingInjectionDuration.Record(ctx, float64(duration.Milliseconds()))
	span.SetAttributes(
		attribute.Int64("durationMs", duration.Milliseconds()),
		attribute.Int("bindingCount", bindingCount),
	)
	l.Trace("Injected event bindings", logger_domain.Int("count", bindingCount))
	span.SetStatus(codes.Ok, "Event bindings injected successfully")
}

// sanitiseForJSIdentifier replaces characters that are not valid in JavaScript
// identifiers with underscores.
//
// Takes name (string) which is the raw name to make safe.
//
// Returns string which is the name safe for use as a JavaScript identifier.
func sanitiseForJSIdentifier(name string) string {
	r := strings.NewReplacer("-", identifierUnderscore, ":", identifierUnderscore, ".", identifierUnderscore, strSpace, identifierUnderscore)
	return r.Replace(name)
}

// buildDotExpr creates a dot expression to access a property on a base name.
//
// Takes base (string) which is the name of the base identifier.
// Takes prop (string) which is the property name to access.
// Takes registry (*RegistryContext) which creates the base identifier
// expression.
//
// Returns *js_ast.EDot which is the property access expression.
func buildDotExpr(base string, prop string, registry *RegistryContext) *js_ast.EDot {
	return &js_ast.EDot{
		Target: registry.MakeIdentifierExpr(base),
		Name:   prop,
	}
}

// walkExpr walks a JavaScript expression tree, calling the callback
// for each node.
//
// Takes expression (js_ast.Expr) which is the root expression to
// walk.
// Takes callback (func(js_ast.Expr) bool) which is called for
// each node. When callback returns false, walking stops for that
// branch.
func walkExpr(expression js_ast.Expr, callback func(js_ast.Expr) bool) {
	if expression.Data == nil || !callback(expression) {
		return
	}
	walkExprChildren(expression, callback)
}

// walkExprChildren visits the child nodes of the given expression.
//
// Takes expression (js_ast.Expr) which is the parent expression to
// visit.
// Takes callback (func(...)) which is called for each child
// expression found.
func walkExprChildren(expression js_ast.Expr, callback func(js_ast.Expr) bool) {
	switch e := expression.Data.(type) {
	case *js_ast.EDot:
		walkExpr(e.Target, callback)
	case *js_ast.EIndex:
		walkIndexExpr(e, callback)
	case *js_ast.ECall:
		walkCallExpr(e, callback)
	case *js_ast.EUnary:
		walkExpr(e.Value, callback)
	case *js_ast.EBinary:
		walkBinaryExpr(e, callback)
	case *js_ast.EIf:
		walkConditionalExpr(e, callback)
	case *js_ast.EArray:
		walkArrayExpr(e, callback)
	case *js_ast.EObject:
		walkObjectExpr(e, callback)
	case *js_ast.EArrow:
		walkArrowExpr(e, callback)
	}
}

// walkIndexExpr walks an index expression by visiting both its target and
// index parts.
//
// Takes e (*js_ast.EIndex) which is the index expression to walk.
// Takes callback (func(...)) which is called for each expression node visited.
func walkIndexExpr(e *js_ast.EIndex, callback func(js_ast.Expr) bool) {
	walkExpr(e.Target, callback)
	walkExpr(e.Index, callback)
}

// walkCallExpr walks a function call expression and its arguments.
//
// Takes e (*js_ast.ECall) which is the call expression to walk.
// Takes callback (func(...)) which is called for each expression visited.
func walkCallExpr(e *js_ast.ECall, callback func(js_ast.Expr) bool) {
	walkExpr(e.Target, callback)
	for _, argument := range e.Args {
		walkExpr(argument, callback)
	}
}

// walkBinaryExpr walks both sides of a binary expression.
//
// Takes e (*js_ast.EBinary) which is the binary expression to walk.
// Takes callback (func(...)) which is called for each child expression.
func walkBinaryExpr(e *js_ast.EBinary, callback func(js_ast.Expr) bool) {
	walkExpr(e.Left, callback)
	walkExpr(e.Right, callback)
}

// walkConditionalExpr walks through a conditional expression and its branches.
//
// Takes e (*js_ast.EIf) which is the conditional expression to walk.
// Takes callback (func(...)) which is called for each expression found.
func walkConditionalExpr(e *js_ast.EIf, callback func(js_ast.Expr) bool) {
	walkExpr(e.Test, callback)
	walkExpr(e.Yes, callback)
	walkExpr(e.No, callback)
}

// walkArrayExpr walks each element in a JavaScript array expression.
//
// Takes e (*js_ast.EArray) which is the array expression to walk.
// Takes callback (func(...)) which is called for each nested expression.
func walkArrayExpr(e *js_ast.EArray, callback func(js_ast.Expr) bool) {
	for _, item := range e.Items {
		walkExpr(item, callback)
	}
}

// walkObjectExpr visits each property in a JavaScript object expression and
// calls the callback for each property value.
//
// Takes e (*js_ast.EObject) which is the object expression to walk.
// Takes callback (func(...)) which is called for each expression found.
func walkObjectExpr(e *js_ast.EObject, callback func(js_ast.Expr) bool) {
	for _, prop := range e.Properties {
		if prop.ValueOrNil.Data != nil {
			walkExpr(prop.ValueOrNil, callback)
		}
	}
}

// walkArrowExpr walks through an arrow function expression and applies the
// callback to each expression within its body statements.
//
// Takes e (*js_ast.EArrow) which is the arrow expression to walk.
// Takes callback (func(...)) which is called for each expression found.
func walkArrowExpr(e *js_ast.EArrow, callback func(js_ast.Expr) bool) {
	for _, statement := range e.Body.Block.Stmts {
		walkStmt(statement, func(s js_ast.Stmt) bool {
			if expression, ok := s.Data.(*js_ast.SExpr); ok {
				walkExpr(expression.Value, callback)
			}
			return true
		})
	}
}

// walkStmt walks a statement tree and calls the callback for each statement.
//
// Takes statement (js_ast.Stmt) which is the root statement to walk.
// Takes callback (func(...)) which is called for each statement. Return false to
// stop walking children.
func walkStmt(statement js_ast.Stmt, callback func(js_ast.Stmt) bool) {
	if statement.Data == nil || !callback(statement) {
		return
	}
	walkStmtChildren(statement, callback)
}

// walkStmtChildren visits the child statements of a JavaScript AST statement.
//
// Takes statement (js_ast.Stmt) which is the statement whose children to visit.
// Takes callback (func(...)) which is called for each child statement found.
func walkStmtChildren(statement js_ast.Stmt, callback func(js_ast.Stmt) bool) {
	switch s := statement.Data.(type) {
	case *js_ast.SBlock:
		walkBlockStmt(s, callback)
	case *js_ast.SIf:
		walkIfStmt(s, callback)
	case *js_ast.SFor:
		walkForStmt(s, callback)
	case *js_ast.SForIn:
		walkStmt(s.Body, callback)
	case *js_ast.SForOf:
		walkStmt(s.Body, callback)
	case *js_ast.SWhile:
		walkStmt(s.Body, callback)
	case *js_ast.SDoWhile:
		walkStmt(s.Body, callback)
	case *js_ast.STry:
		walkTryStmt(s, callback)
	}
}

// walkBlockStmt visits each statement in a block statement.
//
// Takes s (*js_ast.SBlock) which is the block statement to visit.
// Takes callback (func(...)) which is called for each statement in the block.
func walkBlockStmt(s *js_ast.SBlock, callback func(js_ast.Stmt) bool) {
	for _, st := range s.Stmts {
		walkStmt(st, callback)
	}
}

// walkIfStmt walks through the branches of an if statement.
//
// Takes s (*js_ast.SIf) which is the if statement to walk.
// Takes callback (func(...)) which is called for each statement in the branches.
func walkIfStmt(s *js_ast.SIf, callback func(js_ast.Stmt) bool) {
	walkStmt(s.Yes, callback)
	if s.NoOrNil.Data != nil {
		walkStmt(s.NoOrNil, callback)
	}
}

// walkForStmt traverses a JavaScript for statement and its child statements.
//
// Takes s (*js_ast.SFor) which is the for statement to traverse.
// Takes callback (func(...)) which is called for each statement visited.
func walkForStmt(s *js_ast.SFor, callback func(js_ast.Stmt) bool) {
	if s.InitOrNil.Data != nil {
		walkStmt(s.InitOrNil, callback)
	}
	walkStmt(s.Body, callback)
}

// walkTryStmt visits all statements in a try-catch-finally block.
//
// Takes s (*js_ast.STry) which is the try statement to walk.
// Takes callback (func(...)) which is called for each nested statement.
func walkTryStmt(s *js_ast.STry, callback func(js_ast.Stmt) bool) {
	for _, st := range s.Block.Stmts {
		walkStmt(st, callback)
	}
	if s.Catch != nil {
		for _, st := range s.Catch.Block.Stmts {
			walkStmt(st, callback)
		}
	}
	if s.Finally != nil {
		for _, st := range s.Finally.Block.Stmts {
			walkStmt(st, callback)
		}
	}
}

// buildModifierGuards generates JavaScript guard statements for
// event modifiers.
//
// The generated code is prepended to the event handler body. Only
// handler-body modifiers (prevent, stop, once, self) are emitted;
// listener-option modifiers (passive, capture) are handled
// separately via the prop key encoding. The execution order matches
// the DOMBinder implementation: self guard -> preventDefault ->
// stopPropagation -> once guard.
//
// Takes modifiers ([]string) which contains the user-facing event modifiers.
// Takes handlerName (string) which is used to derive the once-flag property name.
//
// Returns string which is the concatenated guard statements with a trailing
// space if non-empty, or an empty string if no handler-body modifiers are present.
func buildModifierGuards(modifiers []string, handlerName string) string {
	if len(modifiers) == 0 {
		return ""
	}

	modSet := make(map[string]bool, len(modifiers))
	for _, m := range modifiers {
		modSet[m] = true
	}

	var guards []string

	if modSet["self"] {
		guards = append(guards, "if(e.target!==e.currentTarget)return;")
	}
	if modSet["prevent"] {
		guards = append(guards, "e.preventDefault();")
	}
	if modSet["stop"] {
		guards = append(guards, "e.stopPropagation();")
	}
	if modSet["once"] {
		onceName := strings.Replace(handlerName, "_dir_", "_once_", 1)
		onceName = strings.Replace(onceName, "_hof_", "_once_", 1)
		guards = append(guards, fmt.Sprintf("if(this.%s)return;this.%s=true;", onceName, onceName))
	}

	if len(guards) == 0 {
		return ""
	}

	return strings.Join(guards, " ") + " "
}

// filterHandlerModifiers returns only the handler-body modifiers
// from the full modifier list.
//
// Handler-body modifiers are prevent, stop, once, and self.
// Listener-option modifiers (passive, capture) are excluded.
//
// Takes modifiers ([]string) which contains the full list of
// user-facing event modifiers.
//
// Returns []string which contains only the handler-body modifiers.
func filterHandlerModifiers(modifiers []string) []string {
	if len(modifiers) == 0 {
		return nil
	}
	var result []string
	for _, m := range modifiers {
		switch m {
		case "prevent", "stop", "once", "self":
			result = append(result, m)
		}
	}
	return result
}

// buildListenerOptionSuffix returns a $-delimited suffix encoding listener-option
// modifiers (capture, passive) for the VDOM prop key. Handler-body modifiers
// (prevent, stop, once, self) are not included as they are injected into the
// handler body at compile time.
//
// The suffixes are emitted in alphabetical order for deterministic output.
//
// Takes modifiers ([]string) which is the full list of user-facing
// event modifiers.
//
// Returns string which is the suffix (e.g. "$capture", "$capture$passive"),
// or an empty string if no listener-option modifiers are present.
func buildListenerOptionSuffix(modifiers []string) string {
	if len(modifiers) == 0 {
		return ""
	}

	modSet := make(map[string]bool, len(modifiers))
	for _, m := range modifiers {
		modSet[m] = true
	}

	var parts []string

	if modSet["capture"] {
		parts = append(parts, "$capture")
	}
	if modSet["passive"] {
		parts = append(parts, "$passive")
	}

	return strings.Join(parts, "")
}

// findUsedLoopVars returns the loop variable names that appear in the given
// expressions.
//
// Takes arguments ([]js_ast.Expr) which contains the expressions to search.
// Takes loopVarNames ([]string) which lists the variable names to find.
// Takes registry (*RegistryContext) which provides context for printing
// expressions.
//
// Returns []string which contains the matching variable names.
func findUsedLoopVars(arguments []js_ast.Expr, loopVarNames []string, registry *RegistryContext) []string {
	var used []string
	for _, varName := range loopVarNames {
		for _, argument := range arguments {
			argString := PrintExpr(argument, registry)
			if strings.Contains(argString, varName) {
				used = append(used, varName)
				break
			}
		}
	}
	return used
}

// buildHOFCallExpr builds a higher-order function call expression in the form
// this._hof_handler(loopVar1, loopVar2, ...).
//
// Takes handlerName (string) which specifies the method name to call on this.
// Takes loopVarNames ([]string) which lists the loop variable names to pass as
// arguments.
// Takes registry (*RegistryContext) which provides identifier creation.
//
// Returns js_ast.Expr which is the built call expression.
func buildHOFCallExpr(handlerName string, loopVarNames []string, registry *RegistryContext) js_ast.Expr {
	arguments := make([]js_ast.Expr, len(loopVarNames))
	for i, varName := range loopVarNames {
		arguments[i] = registry.MakeIdentifierExpr(varName)
	}

	return js_ast.Expr{
		Data: &js_ast.ECall{
			Target: js_ast.Expr{Data: buildDotExpr(jsThis, handlerName, registry)},
			Args:   arguments,
		},
	}
}

// encodeEventHandlerArgs converts user arguments to a comma-separated JS
// string, replacing "$event" with "e" (the closure's event parameter name)
// and "$form" with a FormData expression for the closest form element.
//
// Takes arguments ([]js_ast.Expr) which contains the argument expressions to
// encode.
// Takes registry (*RegistryContext) which provides context for printing
// expressions.
//
// Returns string which is the comma-separated JavaScript argument list.
func encodeEventHandlerArgs(arguments []js_ast.Expr, _ []string, registry *RegistryContext) string {
	parts := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		argString := PrintExpr(argument, registry)
		switch argString {
		case "$event":
			argString = "e"
		case "$form":
			argString = "new FormData(e.target.closest('form'))"
		}
		parts = append(parts, argString)
	}
	return strings.Join(parts, ", ")
}

// usesLoopVar checks if any of the arguments refer to a loop variable.
//
// Takes arguments ([]js_ast.Expr) which contains the expressions to check.
// Takes loopVarNames ([]string) which lists the loop variable names to look
// for.
// Takes registry (*RegistryContext) which provides context for printing
// expressions.
//
// Returns bool which is true if any argument contains a loop variable name.
func usesLoopVar(arguments []js_ast.Expr, loopVarNames []string, registry *RegistryContext) bool {
	if len(loopVarNames) == 0 {
		return false
	}
	for _, argument := range arguments {
		argString := PrintExpr(argument, registry)
		for _, varName := range loopVarNames {
			if strings.Contains(argString, varName) {
				return true
			}
		}
	}
	return false
}

// encodeBlockStatements converts the statements in a block to a JavaScript
// string.
//
// Takes block (*js_ast.SBlock) which contains the statements to convert.
// Takes registry (*RegistryContext) which provides the compilation context.
//
// Returns string which contains the converted statements joined by spaces, or
// an empty string if the block is nil or has no statements.
func encodeBlockStatements(block *js_ast.SBlock, registry *RegistryContext) string {
	if block == nil || len(block.Stmts) == 0 {
		return ""
	}
	parts := make([]string, 0, len(block.Stmts))
	for _, statement := range block.Stmts {
		parts = append(parts, PrintStatement(statement, registry))
	}
	return strings.Join(parts, " ")
}
