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
	"fmt"
	goast "go/ast"
	"slices"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
)

// maxFixedArityArgs is the maximum number of arguments supported by fixed-arity
// encoders.
const maxFixedArityArgs = 4

// emitEventHandlers processes all event directives for a template node.
//
// It handles two cases:
//  1. Action/helper modifier: Creates base64 JSON payload for action and helper
//     event handlers. The generated JS in actions.gen.ts handles the actual
//     remote call dispatch via ActionBuilder.
//  2. No modifier with HasClientScript: Creates base64 JSON payload for PK
//     client handlers. DOMBinder looks up the function in PageContext, which
//     the PK module fills in.
//
// Standard Go handlers (no modifier, no client script) are skipped. The
// runtime handles these through AST node directives, not HTML attributes.
//
// Takes nodeVar (*goast.Ident) which identifies the node variable in the
// generated code.
// Takes node (*ast_domain.TemplateNode) which contains the event directives
// to process.
//
// Returns []goast.Stmt which contains the generated statements for event
// attribute assignments.
// Returns []*ast_domain.Diagnostic which contains any errors found during
// processing.
func (ae *attributeEmitter) emitEventHandlers(nodeVar *goast.Ident, node *ast_domain.TemplateNode) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	var allStmts []goast.Stmt
	var allDiags []*ast_domain.Diagnostic

	allEvents := make(map[string][]ast_domain.Directive)
	for eventName, directives := range node.OnEvents {
		allEvents[eventName] = append(allEvents[eventName], directives...)
	}
	for eventName, directives := range node.CustomEvents {
		allEvents[eventName] = append(allEvents[eventName], directives...)
	}

	if len(allEvents) == 0 {
		return nil, nil
	}

	sortedEventNames := make([]string, 0, len(allEvents))
	for k := range allEvents {
		sortedEventNames = append(sortedEventNames, k)
	}
	slices.Sort(sortedEventNames)

	for _, eventName := range sortedEventNames {
		directives := allEvents[eventName]
		for i := range directives {
			d := &directives[i]

			propKey, shouldEmit := ae.resolveEventHandlerEmission(d, eventName, node)
			if !shouldEmit {
				continue
			}

			payloadStmts, dwVar, bufferPointerVar, payloadDiags := ae.buildActionPayload(*d, node)
			allStmts = append(allStmts, payloadStmts...)
			allDiags = append(allDiags, payloadDiags...)

			allStmts = append(allStmts, ae.buildDirectWriterAttributeIfNotNil(nodeVar, propKey, dwVar, bufferPointerVar)...)
		}
	}
	return allStmts, allDiags
}

// resolveEventHandlerEmission checks if an event directive should produce an
// HTML attribute and returns the attribute name.
//
// Takes d (*ast_domain.Directive) which is the directive to check.
// Takes eventName (string) which is the event name for the attribute key.
// Takes node (*ast_domain.TemplateNode) which is the parent node that holds
// the directive; used for source path lookup when the directive has none.
//
// Returns string which is the p-on:* or p-event:* attribute name.
// Returns bool which is true if the directive should produce an HTML attribute.
//
// Emission rules:
//   - Action/helper modifiers: Always emit (e.g. p-on:click.prevent). Event
//     modifiers from the directive are appended to the attribute name so that
//     DOMBinder can apply them at runtime.
//   - No modifier with HasClientScript: Emit (e.g. p-on:click) for DOMBinder.
//   - No modifier without client script: Do not emit (Go runtime handles this
//     through AST directives).
func (ae *attributeEmitter) resolveEventHandlerEmission(d *ast_domain.Directive, eventName string, node *ast_domain.TemplateNode) (string, bool) {
	prefix := "p-on:"
	if d.Type != ast_domain.DirectiveOn {
		prefix = "p-event:"
	}

	attributeName := buildEventAttributeName(prefix, eventName, d.EventModifiers)

	switch d.Modifier {
	case actionModifierName, helperModifierName:
		return attributeName, true

	case "":
		if ae.directiveHasClientScript(d, node) {
			return attributeName, true
		}
		return "", false

	default:
		return "", false
	}
}

// directiveHasClientScript checks whether a directive comes from a component
// with a client-side script. It first checks the directive's source path, then
// the parent node's source path, and finally falls back to the main component's
// HasClientScript setting.
//
// Event handlers from embedded partials are then emitted even when the parent
// page has no client script of its own.
//
// Takes d (*ast_domain.Directive) which is the directive to check.
// Takes node (*ast_domain.TemplateNode) which is the parent node containing
// the directive; used as fallback if the directive lacks its own annotations.
//
// Returns bool which is true if the directive's source has a client script.
func (ae *attributeEmitter) directiveHasClientScript(d *ast_domain.Directive, node *ast_domain.TemplateNode) bool {
	if d.GoAnnotations != nil && d.GoAnnotations.OriginalSourcePath != nil {
		sourcePath := *d.GoAnnotations.OriginalSourcePath
		if ae.emitter.config.SourcePathHasClientScript != nil {
			if hasScript, ok := ae.emitter.config.SourcePathHasClientScript[sourcePath]; ok {
				return hasScript
			}
		}
	}

	if node != nil && node.GoAnnotations != nil && node.GoAnnotations.OriginalSourcePath != nil {
		sourcePath := *node.GoAnnotations.OriginalSourcePath
		if ae.emitter.config.SourcePathHasClientScript != nil {
			if hasScript, ok := ae.emitter.config.SourcePathHasClientScript[sourcePath]; ok {
				return hasScript
			}
		}
	}

	return ae.emitter.config.HasClientScript
}

// buildActionPayload creates a DirectWriter with base64-encoded JSON
// payload for remote, helper, or client event handlers.
//
// Takes d (ast_domain.Directive) which specifies the directive to
// process.
//
// Returns statements ([]goast.Stmt) which contains the statements to
// build the payload.
// Returns dwVarExpr (goast.Expr) which is the DirectWriter variable
// for AttributeWriters.
// Returns bufferPointerVarExpr (goast.Expr) which is the buffer
// pointer variable for nil check.
// Returns diagnostics ([]*ast_domain.Diagnostic) which contains any
// validation errors.
func (ae *attributeEmitter) buildActionPayload(
	d ast_domain.Directive,
	_ *ast_domain.TemplateNode,
) (statements []goast.Stmt, dwVarExpr goast.Expr, bufferPointerVarExpr goast.Expr, diagnostics []*ast_domain.Diagnostic) {
	callExpr, functionName, diagnostic := ae.normaliseAndValidateDirectiveExpression(d)
	if diagnostic != nil {
		return nil, cachedIdent("nil"), cachedIdent("nil"), []*ast_domain.Diagnostic{diagnostic}
	}

	goArgConstructors, argStmts, argDiags := ae.buildActionArgumentuments(callExpr.Args)
	dwVar, bufferPointerVar, payloadStmts := ae.buildPayloadEncodingStatements(functionName.Name, goArgConstructors)
	return append(argStmts, payloadStmts...), dwVar, bufferPointerVar, argDiags
}

// normaliseAndValidateDirectiveExpression converts a directive expression to a
// call expression and checks that it is valid.
//
// Takes d (ast_domain.Directive) which is the directive to process.
//
// Returns *ast_domain.CallExpression which is the call expression form of the
// directive.
// Returns *ast_domain.Identifier which is the function name from the callee.
// Returns *ast_domain.Diagnostic which describes any error found.
func (ae *attributeEmitter) normaliseAndValidateDirectiveExpression(d ast_domain.Directive) (*ast_domain.CallExpression, *ast_domain.Identifier, *ast_domain.Diagnostic) {
	path := ae.getSourcePath(d.GoAnnotations)

	callExpr := normaliseToCallExpr(d.Expression)
	if callExpr == nil {
		return nil, nil, ast_domain.NewDiagnostic(
			ast_domain.Error,
			fmt.Sprintf("Expression for .%s modifier must be a function call or identifier", d.Modifier),
			d.RawExpression,
			d.Location,
			path,
		)
	}

	functionName, ok := callExpr.Callee.(*ast_domain.Identifier)
	if !ok {
		return nil, nil, ast_domain.NewDiagnostic(
			ast_domain.Error,
			fmt.Sprintf("Callee for .%s modifier must be a simple identifier", d.Modifier),
			callExpr.Callee.String(),
			d.Location,
			path,
		)
	}

	return callExpr, functionName, nil
}

// buildActionArgumentuments builds the ActionArgument constructors for all call
// arguments.
//
// Takes arguments ([]ast_domain.Expression) which contains the expressions to
// convert into action arguments.
//
// Returns []goast.Expr which contains the generated argument constructors.
// Returns []goast.Stmt which contains any statements needed for the arguments.
// Returns []*ast_domain.Diagnostic which contains any issues found.
func (ae *attributeEmitter) buildActionArgumentuments(arguments []ast_domain.Expression) ([]goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	goArgConstructors := make([]goast.Expr, 0, len(arguments))
	var allStmts []goast.Stmt
	var allDiags []*ast_domain.Diagnostic

	for _, argument := range arguments {
		argConstructor, argStmts, argDiags := ae.buildSingleActionArgument(argument)
		goArgConstructors = append(goArgConstructors, argConstructor)
		allStmts = append(allStmts, argStmts...)
		allDiags = append(allDiags, argDiags...)
	}

	return goArgConstructors, allStmts, allDiags
}

// buildSingleActionArgument builds a single ActionArgument
// constructor, handling both fixed and changing arguments.
//
// Takes argument (ast_domain.Expression) which is the expression
// to convert into an ActionArgument.
//
// Returns goast.Expr which is the constructed ActionArgument composite literal.
// Returns []goast.Stmt which contains statements that must run before the
// expression.
// Returns []*ast_domain.Diagnostic which contains any problems found during
// conversion.
func (ae *attributeEmitter) buildSingleActionArgument(argument ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	argType, argValue, prereqs, diagnostics := ae.emitActionArgumentValue(argument)

	if argType == "e" {
		constructor := &goast.CompositeLit{
			Type: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(actionArgTypeName)},
			Elts: []goast.Expr{
				&goast.KeyValueExpr{Key: cachedIdent("Type"), Value: strLit("e")},
			},
		}
		return constructor, prereqs, diagnostics
	}

	if argType == "f" {
		constructor := &goast.CompositeLit{
			Type: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(actionArgTypeName)},
			Elts: []goast.Expr{
				&goast.KeyValueExpr{Key: cachedIdent("Type"), Value: strLit("f")},
			},
		}
		return constructor, prereqs, diagnostics
	}

	constructor := &goast.CompositeLit{
		Type: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(actionArgTypeName)},
		Elts: []goast.Expr{
			&goast.KeyValueExpr{Key: cachedIdent("Type"), Value: strLit(argType)},
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameValue), Value: argValue},
		},
	}

	return constructor, prereqs, diagnostics
}

// emitActionArgumentValue outputs the value for an action argument, using a
// simpler path for fixed literals.
//
// Takes argument (ast_domain.Expression) which is the expression to output.
//
// Returns argType (string) which shows the argument type: "s" for static,
// "v" for variable, "e" for event placeholder, or "f" for form placeholder.
// Returns value (goast.Expr) which is the resulting Go expression. This is
// nil for "e" and "f" types.
// Returns prereqs ([]goast.Stmt) which holds any statements that must run
// before the value can be used.
// Returns diagnostics ([]*ast_domain.Diagnostic) which holds any errors or
// warnings found.
func (ae *attributeEmitter) emitActionArgumentValue(argument ast_domain.Expression) (argType string, value goast.Expr, prereqs []goast.Stmt, diagnostics []*ast_domain.Diagnostic) {
	if identifier, ok := argument.(*ast_domain.Identifier); ok {
		if identifier.Name == "$event" {
			return "e", nil, nil, nil
		}
		if identifier.Name == "$form" {
			return "f", nil, nil, nil
		}
	}

	switch argNode := argument.(type) {
	case *ast_domain.StringLiteral:
		return "s", strLit(argNode.Value), nil, nil
	case *ast_domain.IntegerLiteral, *ast_domain.FloatLiteral, *ast_domain.BooleanLiteral:
		return "s", strLit(argNode.String()), nil, nil
	default:
		value, prereqs, diagnostics = ae.expressionEmitter.emit(argument)
		return "v", value, prereqs, diagnostics
	}
}

// buildPayloadEncodingStatements builds statements that encode the
// payload to base64 JSON using a pooled DirectWriter for
// zero-allocation rendering.
//
// For argCount <= 4, uses fixed-arity encoder functions to avoid
// slice allocation. For argCount > 4, falls back to the slice-based
// variadic form.
//
// Takes functionName (string) which specifies the function name for
// the payload.
// Takes argConstructors ([]goast.Expr) which provides the argument
// builder expressions.
//
// Returns dwVarExpr (goast.Expr) which is the DirectWriter variable
// for AttributeWriters.
// Returns bufferPointerVarExpr (goast.Expr) which is the buffer
// pointer variable for nil check.
// Returns statements ([]goast.Stmt) which contains the statements
// that do the encoding.
//
// Generated code pattern (fixed-arity, 0-4 args):
//
//	bpVar := pikoruntime.EncodeActionPayloadBytes2(
//	    functionName, arg0, arg1)
//	dwVar := pikoruntime.GetDirectWriter()
//	dwVar.AppendPooledBytes(bpVar)
//
// Generated code pattern (variadic, >4 args):
//
//	bpVar := pikoruntime.EncodeActionPayloadBytes(
//	    ActionPayload{...})
//	dwVar := pikoruntime.GetDirectWriter()
//	dwVar.AppendPooledBytes(bpVar)
func (ae *attributeEmitter) buildPayloadEncodingStatements(functionName string, argConstructors []goast.Expr) (dwVarExpr goast.Expr, bufferPointerVarExpr goast.Expr, statements []goast.Stmt) {
	bufferPointerVar := cachedIdent(ae.emitter.nextTempName())
	dwVar := cachedIdent(ae.emitter.nextTempName())

	encodeCallExpr := ae.buildEncodeActionPayloadCall(functionName, argConstructors)

	statements = []goast.Stmt{
		defineAndAssign(bufferPointerVar.Name, encodeCallExpr),
		defineAndAssign(dwVar.Name, &goast.CallExpr{
			Fun: &goast.SelectorExpr{X: cachedIdent(arenaVarName), Sel: cachedIdent("GetDirectWriter")},
		}),
		&goast.ExprStmt{X: &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: dwVar, Sel: cachedIdent("AppendPooledBytes")},
			Args: []goast.Expr{bufferPointerVar},
		}},
	}

	dwVarExpr = dwVar
	bufferPointerVarExpr = bufferPointerVar
	return dwVarExpr, bufferPointerVarExpr, statements
}

// buildEncodeActionPayloadCall builds a call expression to encode an action
// payload.
//
// Uses fixed-arity arena functions when argCount <= 4 to avoid slice
// allocation. All variants use arena-aware functions to eliminate sync.Pool
// allocations.
//
// Takes functionName (string) which is the action function name.
// Takes argConstructors ([]goast.Expr) which are the ActionArgument composite
// literals.
//
// Returns goast.Expr which is the call expression for the encoder.
func (*attributeEmitter) buildEncodeActionPayloadCall(functionName string, argConstructors []goast.Expr) goast.Expr {
	argCount := len(argConstructors)

	if argCount <= maxFixedArityArgs {
		encoderName := selectActionEncoderFunc(argCount)
		arguments := make([]goast.Expr, 0, argCount+2)
		arguments = append(arguments, cachedIdent(arenaVarName), strLit(functionName))
		arguments = append(arguments, argConstructors...)
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(encoderName)},
			Args: arguments,
		}
	}

	payloadLit := &goast.CompositeLit{
		Type: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("ActionPayload")},
		Elts: []goast.Expr{
			&goast.KeyValueExpr{Key: cachedIdent("Function"), Value: strLit(functionName)},
			&goast.KeyValueExpr{Key: cachedIdent("Args"), Value: &goast.CompositeLit{
				Type: &goast.ArrayType{Elt: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(actionArgTypeName)}},
				Elts: argConstructors,
			}},
		},
	}
	return &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("EncodeActionPayloadBytesArena")},
		Args: []goast.Expr{cachedIdent(arenaVarName), payloadLit},
	}
}

// buildEventAttributeName constructs the full HTML attribute name for an event
// directive, appending any user-facing modifiers after the event name.
//
// Takes prefix (string) which is "p-on:" or "p-event:".
// Takes eventName (string) which is the DOM event name (e.g., "click").
// Takes modifiers ([]string) which holds user-facing modifiers (e.g.,
// ["prevent", "stop"]).
//
// Returns string which is the complete attribute name (e.g.,
// "p-on:click.prevent.stop").
func buildEventAttributeName(prefix, eventName string, modifiers []string) string {
	if len(modifiers) == 0 {
		return fmt.Sprintf("%s%s", prefix, eventName)
	}
	return fmt.Sprintf("%s%s.%s", prefix, eventName, strings.Join(modifiers, "."))
}

// normaliseToCallExpr converts an expression to a CallExpr.
//
// This handles both CallExpr and Identifier cases. When the expression is a
// CallExpr, it returns it directly. When the expression is an Identifier
// (bare handler name like "handler"), it wraps it in a new CallExpr with an
// implicit $event argument, making bare handler equivalent to handler($event).
//
// Takes expression (ast_domain.Expression) which is the expression to convert.
//
// Returns *ast_domain.CallExpression which is the converted call
// expression, or nil if the expression is neither a CallExpr nor
// an Identifier.
func normaliseToCallExpr(expression ast_domain.Expression) *ast_domain.CallExpression {
	if ce, isCall := expression.(*ast_domain.CallExpression); isCall {
		return ce
	}

	if identifier, isIdent := expression.(*ast_domain.Identifier); isIdent {
		return &ast_domain.CallExpression{
			Callee: identifier,
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "$event"},
			},
			GoAnnotations:    nil,
			RelativeLocation: ast_domain.Location{},
			LparenLocation:   ast_domain.Location{},
			RparenLocation:   ast_domain.Location{},
			SourceLength:     0,
		}
	}

	return nil
}

// selectActionEncoderFunc returns the fixed-arity arena encoder function name
// for the given argument count. Uses arena-aware functions to eliminate
// sync.Pool allocations.
//
// Takes argCount (int) which specifies the number of arguments to encode.
//
// Returns string which is the name of the appropriate encoder function.
func selectActionEncoderFunc(argCount int) string {
	switch argCount {
	case 0:
		return "EncodeActionPayloadBytes0Arena"
	case 1:
		return "EncodeActionPayloadBytes1Arena"
	case 2:
		return "EncodeActionPayloadBytes2Arena"
	case actionArgsCount3:
		return "EncodeActionPayloadBytes3Arena"
	case actionArgsCount4:
		return "EncodeActionPayloadBytes4Arena"
	default:
		return "EncodeActionPayloadBytesArena"
	}
}
