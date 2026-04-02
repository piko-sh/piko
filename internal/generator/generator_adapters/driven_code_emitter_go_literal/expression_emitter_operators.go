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
	"go/token"
	"slices"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
)

// tryEmitOperatorExpression handles operator-based expressions.
//
// Takes expression (ast_domain.Expression) which is the expression to emit.
//
// Returns goast.Expr which is the emitted Go expression, or nil if not handled.
// Returns []goast.Stmt which contains any statements needed for the expression.
// Returns []*ast_domain.Diagnostic which contains any diagnostics generated.
// Returns bool which indicates whether the expression was handled.
func (ee *expressionEmitter) tryEmitOperatorExpression(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic, bool) {
	switch n := expression.(type) {
	case *ast_domain.MemberExpression:
		goExpr, statements, diagnostics := ee.emitMemberExpr(n)
		return goExpr, statements, diagnostics, true
	case *ast_domain.IndexExpression:
		goExpr, statements, diagnostics := ee.emitIndexExpr(n)
		return goExpr, statements, diagnostics, true
	case *ast_domain.CallExpression:
		goExpr, statements, diagnostics := ee.emitCallExpr(n)
		return goExpr, statements, diagnostics, true
	case *ast_domain.BinaryExpression:
		goExpr, statements, diagnostics := ee.binaryEmitter.emit(n)
		return goExpr, statements, diagnostics, true
	case *ast_domain.UnaryExpression:
		goExpr, statements, diagnostics := ee.emitUnaryExpr(n)
		return goExpr, statements, diagnostics, true
	case *ast_domain.TernaryExpression:
		goExpr, statements, diagnostics := ee.emitTernaryExpr(n)
		return goExpr, statements, diagnostics, true
	}
	return nil, nil, nil, false
}

// emitMemberExpr outputs Go code for member access expressions (e.g. user.name).
//
// Takes n (*ast_domain.MemberExpression) which is the member
// expression to process.
//
// Returns goast.Expr which is the Go expression for the member access.
// Returns []goast.Stmt which holds any statements needed for safe access.
// Returns []*ast_domain.Diagnostic which holds any errors found.
func (ee *expressionEmitter) emitMemberExpr(n *ast_domain.MemberExpression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	ann := getAnnotationFromExpression(n)

	if !n.Optional && ann != nil && ann.NeedsRuntimeSafetyCheck {
		return ee.emitSafeMemberAccess(n, ann)
	}

	baseGoExpr, baseStmts, baseDiags := ee.emit(n.Base)

	propIdent, ok := n.Property.(*ast_domain.Identifier)
	if !ok {
		diagnostic := ast_domain.NewDiagnostic(
			ast_domain.Error,
			"Internal Emitter Error: Computed properties are not yet supported.",
			n.String(),
			n.RelativeLocation,
			"",
		)
		return cachedIdent(fmt.Sprintf("%s /* computed property */", goKeywordNil)), baseStmts, append(baseDiags, diagnostic)
	}

	goFieldName := propIdent.Name
	if ann := n.GoAnnotations; ann != nil && ann.Symbol != nil {
		goFieldName = ann.Symbol.Name
	}

	if ann != nil && ann.IsMapAccess {
		return ee.emitMapAccess(baseGoExpr, propIdent.Name), baseStmts, baseDiags
	}

	finalSelector := &goast.SelectorExpr{X: baseGoExpr, Sel: cachedIdent(goFieldName)}
	if !n.Optional {
		return finalSelector, baseStmts, baseDiags
	}

	return ee.emitOptionalChaining(n, baseGoExpr, finalSelector, baseStmts, baseDiags)
}

// emitMapAccess builds a map index access expression.
//
// This generates: base.(map[string]interface{})["property"].
//
// Takes baseExpr (goast.Expr) which is the expression to type-assert as a map.
// Takes propName (string) which is the property name to use as the map key.
//
// Returns goast.Expr which is the index expression for accessing the map.
func (*expressionEmitter) emitMapAccess(baseExpr goast.Expr, propName string) goast.Expr {
	mapAssert := &goast.TypeAssertExpr{
		X:    baseExpr,
		Type: &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("interface{}")},
	}
	return &goast.IndexExpr{X: mapAssert, Index: &goast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", propName)}}
}

// emitOptionalChaining converts optional chaining (a?.b) into an if statement
// with a temporary variable.
//
// Takes n (*ast_domain.MemberExpression) which is the member
// expression to convert.
// Takes baseGoExpr (goast.Expr) which is the base expression to check for nil.
// Takes finalSelector (goast.Expr) which is the selector to use when not nil.
// Takes baseStmts ([]goast.Stmt) which are existing statements to add to.
// Takes baseDiags ([]*ast_domain.Diagnostic) which are existing diagnostics.
//
// Returns goast.Expr which is the temporary variable identifier.
// Returns []goast.Stmt which are the statements including the nil check.
// Returns []*ast_domain.Diagnostic which are the collected diagnostics.
func (ee *expressionEmitter) emitOptionalChaining(
	n *ast_domain.MemberExpression,
	baseGoExpr, finalSelector goast.Expr,
	baseStmts []goast.Stmt,
	baseDiags []*ast_domain.Diagnostic,
) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	tempVarName := ee.emitter.nextTempName()
	varType := ee.getTypeExprForVarDecl(n.GoAnnotations)

	varDecl := &goast.DeclStmt{Decl: &goast.GenDecl{Tok: token.VAR, Specs: []goast.Spec{
		&goast.ValueSpec{Names: []*goast.Ident{cachedIdent(tempVarName)}, Type: varType},
	}}}
	ifStmt := &goast.IfStmt{
		Cond: &goast.BinaryExpr{X: baseGoExpr, Op: token.NEQ, Y: cachedIdent(goKeywordNil)},
		Body: &goast.BlockStmt{List: []goast.Stmt{assignExpression(tempVarName, finalSelector)}},
	}
	baseStmts = append(baseStmts, varDecl, ifStmt)
	return cachedIdent(tempVarName), baseStmts, baseDiags
}

// emitSafeMemberAccess emits runtime-safe member access with nil checking.
//
// Takes n (*ast_domain.MemberExpression) which specifies the member expression to
// emit.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides generator
// annotations for code generation.
//
// Returns goast.Expr which is the IIFE expression for safe member access.
// Returns []goast.Stmt which contains statements from emitting the base
// expression.
// Returns []*ast_domain.Diagnostic which contains any diagnostics from base
// emission.
//
// Panics if the property is not an identifier.
func (ee *expressionEmitter) emitSafeMemberAccess(
	n *ast_domain.MemberExpression,
	ann *ast_domain.GoGeneratorAnnotation,
) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	baseGoExpr, baseStmts, baseDiags := ee.emit(n.Base)

	propIdent, ok := n.Property.(*ast_domain.Identifier)
	if !ok {
		panic(fmt.Sprintf("emitSafeMemberAccess called with non-identifier property: %T", n.Property))
	}

	iife := ee.buildSafeMemberAccessIIFE(n, propIdent.Name, baseGoExpr, ann)
	return iife, baseStmts, baseDiags
}

// buildSafeMemberAccessIIFE constructs the IIFE for safe member access.
//
// Takes n (*ast_domain.MemberExpression) which is the member expression to wrap.
// Takes goFieldName (string) which is the Go field name to access.
// Takes baseGoExpr (goast.Expr) which is the base expression to check for nil.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides type information.
//
// Returns goast.Expr which is the IIFE call expression with nil checking.
func (ee *expressionEmitter) buildSafeMemberAccessIIFE(
	n *ast_domain.MemberExpression,
	goFieldName string,
	baseGoExpr goast.Expr,
	ann *ast_domain.GoGeneratorAnnotation,
) goast.Expr {
	varType := ee.getTypeExprForVarDecl(ann)
	zeroValue := getZeroValueExpr(varType)

	diagnosticCall := ee.buildNilAccessDiagnostic(n, ann)
	iifeBody := buildNilCheckIIFEBody(baseGoExpr, goFieldName, diagnosticCall, zeroValue)

	return &goast.CallExpr{
		Fun: &goast.FuncLit{
			Type: &goast.FuncType{Results: &goast.FieldList{List: []*goast.Field{{Type: varType}}}},
			Body: iifeBody,
		},
	}
}

// buildNilAccessDiagnostic creates a diagnostic call for when code tries to
// access a member on a nil value.
//
// Takes n (*ast_domain.MemberExpression) which is the member expression being
// accessed.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the location and
// context for the diagnostic.
//
// Returns goast.Expr which is the diagnostic call expression.
//
// Panics if the property of the member expression is not an identifier.
func (ee *expressionEmitter) buildNilAccessDiagnostic(
	n *ast_domain.MemberExpression,
	ann *ast_domain.GoGeneratorAnnotation,
) goast.Expr {
	propIdent, ok := n.Property.(*ast_domain.Identifier)
	if !ok {
		panic(fmt.Sprintf("buildNilAccessDiagnostic called with non-identifier property: %T", n.Property))
	}

	message := fmt.Sprintf("Cannot access property '%s' because '%s' is nil", propIdent.Name, n.Base.String())

	return ee.buildDiagnosticCall(
		message,
		"R003",
		"Warning",
		ann,
		n.String(),
		n.GetRelativeLocation(),
	)
}

// buildDiagnosticCall generates the AST for a call to
// generator_helpers.AppendDiagnostic.
//
// Takes message (string) which is the diagnostic message text.
// Takes code (string) which is the diagnostic code identifier (e.g. "R003").
// Takes severity (string) which is the severity level for the diagnostic.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides source context.
// Takes expressionString (string) which is the expression that caused the issue.
// Takes location (ast_domain.Location) which specifies the source location.
//
// Returns goast.Expr which is the call expression AST node.
func (ee *expressionEmitter) buildDiagnosticCall(
	message string,
	code string,
	severity string,
	ann *ast_domain.GoGeneratorAnnotation,
	expressionString string,
	location ast_domain.Location,
) goast.Expr {
	sourcePath := ""
	if ann != nil && ann.OriginalSourcePath != nil {
		sourcePath = ee.emitter.computeRelativePath(*ann.OriginalSourcePath)
	}

	return &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   cachedIdent(runtimePackageName),
			Sel: cachedIdent("AppendDiagnostic"),
		},
		Args: []goast.Expr{
			cachedIdent(DiagnosticsVarName),
			&goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(severity)},
			strLit(message),
			strLit(code),
			strLit(sourcePath),
			strLit(expressionString),
			intLit(location.Line),
			intLit(location.Column),
		},
	}
}

// emitIndexExpr emits code for index access expressions (e.g., arr[0],
// map[key]).
//
// Takes n (*ast_domain.IndexExpression) which specifies the index
// expression to emit.
//
// Returns goast.Expr which is the generated Go expression for the index access.
// Returns []goast.Stmt which contains any statements needed for safe access.
// Returns []*ast_domain.Diagnostic which contains any diagnostics found.
func (ee *expressionEmitter) emitIndexExpr(n *ast_domain.IndexExpression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	ann := getAnnotationFromExpression(n)

	if !n.Optional && ann != nil && ann.NeedsRuntimeSafetyCheck {
		return ee.emitSafeIndexAccess(n, ann)
	}

	baseGoExpr, baseStmts, baseDiags := ee.emit(n.Base)
	indexGoExpr, indexStmts, indexDiags := ee.emit(n.Index)
	baseStmts = append(baseStmts, indexStmts...)
	baseDiags = append(baseDiags, indexDiags...)
	allStmts := baseStmts
	allDiags := baseDiags

	finalIndexExpr := &goast.IndexExpr{X: baseGoExpr, Index: indexGoExpr}
	if !n.Optional {
		return finalIndexExpr, allStmts, allDiags
	}

	tempVarName := ee.emitter.nextTempName()
	varType := ee.getTypeExprForVarDecl(n.GoAnnotations)

	varDecl := &goast.DeclStmt{Decl: &goast.GenDecl{Tok: token.VAR, Specs: []goast.Spec{
		&goast.ValueSpec{Names: []*goast.Ident{cachedIdent(tempVarName)}, Type: varType},
	}}}

	nilCheck := &goast.BinaryExpr{X: baseGoExpr, Op: token.NEQ, Y: cachedIdent(goKeywordNil)}
	lenCall := &goast.CallExpr{Fun: cachedIdent("len"), Args: []goast.Expr{baseGoExpr}}
	lenCheck := &goast.BinaryExpr{X: indexGoExpr, Op: token.LSS, Y: lenCall}

	combinedCond := &goast.BinaryExpr{
		X:  nilCheck,
		Op: token.LAND,
		Y:  lenCheck,
	}

	ifStmt := &goast.IfStmt{
		Cond: combinedCond,
		Body: &goast.BlockStmt{List: []goast.Stmt{assignExpression(tempVarName, finalIndexExpr)}},
	}

	allStmts = append(allStmts, varDecl, ifStmt)
	return cachedIdent(tempVarName), allStmts, allDiags
}

// emitSafeIndexAccess emits runtime-safe index access with bounds checking.
//
// Takes n (*ast_domain.IndexExpression) which is the index expression to emit.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides generation options.
//
// Returns goast.Expr which is the IIFE expression for safe index access.
// Returns []goast.Stmt which contains statements from base and index emission.
// Returns []*ast_domain.Diagnostic which contains diagnostics from emission.
func (ee *expressionEmitter) emitSafeIndexAccess(
	n *ast_domain.IndexExpression,
	ann *ast_domain.GoGeneratorAnnotation,
) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	baseGoExpr, baseStmts, baseDiags := ee.emit(n.Base)
	indexGoExpr, indexStmts, indexDiags := ee.emit(n.Index)

	allStmts := baseStmts
	allStmts = append(allStmts, indexStmts...)
	allDiags := baseDiags
	allDiags = append(allDiags, indexDiags...)

	iife := ee.buildSafeIndexAccessIIFE(n, baseGoExpr, indexGoExpr, ann)
	return iife, allStmts, allDiags
}

// buildSafeIndexAccessIIFE constructs the IIFE for safe index access.
//
// Takes n (*ast_domain.IndexExpression) which is the index expression node.
// Takes baseGoExpr (goast.Expr) which is the base expression to index into.
// Takes indexGoExpr (goast.Expr) which is the index value expression.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides type info.
//
// Returns goast.Expr which is an IIFE that safely accesses the index.
func (ee *expressionEmitter) buildSafeIndexAccessIIFE(
	n *ast_domain.IndexExpression,
	baseGoExpr goast.Expr,
	indexGoExpr goast.Expr,
	ann *ast_domain.GoGeneratorAnnotation,
) goast.Expr {
	varType := ee.getTypeExprForVarDecl(ann)
	zeroValue := getZeroValueExpr(varType)

	baseAnn := getAnnotationFromExpression(n.Base)
	iifeBodyList := ee.buildIndexAccessChecks(n, baseGoExpr, indexGoExpr, baseAnn, zeroValue, ann)

	return &goast.CallExpr{
		Fun: &goast.FuncLit{
			Type: &goast.FuncType{Results: &goast.FieldList{List: []*goast.Field{{Type: varType}}}},
			Body: &goast.BlockStmt{List: iifeBodyList},
		},
	}
}

// indexCheckContext holds the data needed to build index safety checks.
type indexCheckContext struct {
	// expression is the index expression to check for nil or out-of-bounds access.
	expression *ast_domain.IndexExpression

	// baseExpr is the expression being indexed, used for nil and bounds checks.
	baseExpr goast.Expr

	// indexExpr is the expression used as the index into the array or slice.
	indexExpr goast.Expr

	// zeroValue is the expression to return when bounds or nil checks fail.
	zeroValue goast.Expr

	// emitter builds diagnostic calls for nil and bounds check errors.
	emitter *expressionEmitter

	// ann is the generator annotation used when building check statements.
	ann *ast_domain.GoGeneratorAnnotation

	// baseAnn holds the generator annotation for the current code block.
	baseAnn *ast_domain.GoGeneratorAnnotation
}

// buildIndexAccessChecks creates the safety check statements for index access.
//
// Takes n (*ast_domain.IndexExpression) which is the index expression to check.
// Takes baseGoExpr (goast.Expr) which is the base expression being indexed.
// Takes indexGoExpr (goast.Expr) which is the index value expression.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which provides type
// information for the base expression.
// Takes zeroValue (goast.Expr) which is the zero value to return on failure.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides annotation
// context for the operation.
//
// Returns []goast.Stmt which contains the nil check, optional bounds check,
// and final return statement.
func (ee *expressionEmitter) buildIndexAccessChecks(
	n *ast_domain.IndexExpression,
	baseGoExpr goast.Expr,
	indexGoExpr goast.Expr,
	baseAnn *ast_domain.GoGeneratorAnnotation,
	zeroValue goast.Expr,
	ann *ast_domain.GoGeneratorAnnotation,
) []goast.Stmt {
	ctx := &indexCheckContext{
		expression: n,
		baseExpr:   baseGoExpr,
		indexExpr:  indexGoExpr,
		zeroValue:  zeroValue,
		emitter:    ee,
		ann:        ann,
		baseAnn:    baseAnn,
	}

	statements := []goast.Stmt{buildNilCheckWithContext(ctx)}

	if baseAnn != nil && baseAnn.ResolvedType != nil {
		if _, isSlice := baseAnn.ResolvedType.TypeExpression.(*goast.ArrayType); isSlice {
			boundsCheck := buildBoundsCheckWithContext(ctx)
			statements = append(statements, boundsCheck)
		}
	}

	statements = append(statements, &goast.ReturnStmt{Results: []goast.Expr{
		&goast.IndexExpr{X: baseGoExpr, Index: indexGoExpr},
	}})

	return statements
}

// emitCallExpr builds Go code for a function or method call.
//
// Takes n (*ast_domain.CallExpression) which is the call expression to process.
//
// Returns goast.Expr which is the Go call expression.
// Returns []goast.Stmt which holds any statements needed before the call.
// Returns []*ast_domain.Diagnostic which holds any issues found.
func (ee *expressionEmitter) emitCallExpr(n *ast_domain.CallExpression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	var allStmts []goast.Stmt
	var allDiags []*ast_domain.Diagnostic

	if identifier, isIdent := n.Callee.(*ast_domain.Identifier); isIdent && coercionFunctionNames[identifier.Name] {
		return ee.emitCoercionCallExpr(n, identifier.Name)
	}

	if identifier, isIdent := n.Callee.(*ast_domain.Identifier); isIdent && runtimeHelperFunctionNames[identifier.Name] {
		return ee.emitRuntimeHelperCallExpr(n, identifier.Name)
	}

	var calleeGoExpr goast.Expr
	if identifier, isIdent := n.Callee.(*ast_domain.Identifier); isIdent && builtInFunctionNames[identifier.Name] {
		calleeGoExpr = cachedIdent(identifier.Name)
	} else {
		var calleeStmts []goast.Stmt
		var calleeDiags []*ast_domain.Diagnostic
		calleeGoExpr, calleeStmts, calleeDiags = ee.emit(n.Callee)
		allStmts = append(allStmts, calleeStmts...)
		allDiags = append(allDiags, calleeDiags...)
	}

	goArgs := make([]goast.Expr, len(n.Args))
	for i, argument := range n.Args {
		argGoExpr, argStmts, argDiags := ee.emit(argument)
		allStmts = append(allStmts, argStmts...)
		allDiags = append(allDiags, argDiags...)
		goArgs[i] = argGoExpr
	}

	result := &goast.CallExpr{Fun: calleeGoExpr, Args: goArgs}

	if identifier, isIdent := n.Callee.(*ast_domain.Identifier); isIdent && stringerBuilderCallNames[identifier.Name] {
		return wrapWithStringerCall(result), allStmts, allDiags
	}

	return result, allStmts, allDiags
}

// emitCoercionCallExpr handles type coercion function calls such as string,
// int, and float.
//
// Takes n (*ast_domain.CallExpression) which is the coercion call expression.
// Takes functionName (string) which is the name of the coercion function.
//
// Returns goast.Expr which is the generated coercion expression.
// Returns []goast.Stmt which holds any statements needed before the expression.
// Returns []*ast_domain.Diagnostic which holds any issues found.
func (ee *expressionEmitter) emitCoercionCallExpr(n *ast_domain.CallExpression, functionName string) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	if len(n.Args) != 1 {
		diagnostic := ast_domain.NewDiagnostic(
			ast_domain.Error,
			fmt.Sprintf("Coercion function '%s' expects exactly one argument, got %d", functionName, len(n.Args)),
			n.String(),
			n.RelativeLocation,
			"",
		)
		return cachedIdent(goKeywordNil), nil, []*ast_domain.Diagnostic{diagnostic}
	}

	argExpr, argStmts, argDiags := ee.emit(n.Args[0])
	argAnn := getAnnotationFromExpression(n.Args[0])

	coercionEmitter := newCoercionEmitter(ee)
	result := coercionEmitter.emitCoercionCall(functionName, n, argExpr, argAnn)

	return result, argStmts, argDiags
}

// emitRuntimeHelperCallExpr handles calls to Piko runtime helper
// functions (e.g. F), emitting them as pikoruntime.FuncName(arguments...)
// and adding the pikoruntime import automatically.
//
// The annotator resolves these as returning *FormatBuilder, so the
// stringability pipeline adds .String() in contexts that require a
// string value.
//
// Takes n (*ast_domain.CallExpression) which is the call expression to emit.
// Takes functionName (string) which is the runtime helper function name.
//
// Returns goast.Expr which is the generated
// pikoruntime.FuncName(arguments...) call.
// Returns []goast.Stmt which holds any statements needed before the call.
// Returns []*ast_domain.Diagnostic which holds any issues found.
func (ee *expressionEmitter) emitRuntimeHelperCallExpr(n *ast_domain.CallExpression, functionName string) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	var allStmts []goast.Stmt
	var allDiags []*ast_domain.Diagnostic

	goArgs := make([]goast.Expr, len(n.Args))
	for i, argument := range n.Args {
		argGoExpr, argStmts, argDiags := ee.emit(argument)
		allStmts = append(allStmts, argStmts...)
		allDiags = append(allDiags, argDiags...)
		goArgs[i] = argGoExpr
	}

	ee.emitter.addImport(coercionRuntimePackagePath, runtimePackageName)

	callExpr := &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   cachedIdent(runtimePackageName),
			Sel: cachedIdent(functionName),
		},
		Args: goArgs,
	}

	return callExpr, allStmts, allDiags
}

// emitUnaryExpr emits code for unary expressions such as !value, -num, or
// &ptr.
//
// Takes n (*ast_domain.UnaryExpression) which is the unary expression to emit.
//
// Returns goast.Expr which is the emitted Go expression.
// Returns []goast.Stmt which contains any statements made during emission.
// Returns []*ast_domain.Diagnostic which contains any diagnostics found.
func (ee *expressionEmitter) emitUnaryExpr(n *ast_domain.UnaryExpression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	rightGoExpr, rightStmts, rightDiags := ee.emit(n.Right)

	if n.Operator == ast_domain.OpAddrOf {
		return &goast.UnaryExpr{Op: token.AND, X: rightGoExpr}, rightStmts, rightDiags
	}

	if n.Operator == ast_domain.OpTruthy {
		ann := getAnnotationFromExpression(n.Right)
		return emitTruthinessCheck(rightGoExpr, ann), rightStmts, rightDiags
	}

	op, _ := unaryOpToToken(n.Operator)
	if n.Operator == ast_domain.OpNot {
		rightGoExpr = wrapInTruthinessCallIfNeeded(rightGoExpr, n.Right)
	}
	return &goast.UnaryExpr{Op: op, X: rightGoExpr}, rightStmts, rightDiags
}

// emitTernaryExpr outputs Go code for a ternary conditional expression
// (e.g., `condition ? a : b`).
//
// Takes n (*ast_domain.TernaryExpression) which is the ternary expression to emit.
//
// Returns goast.Expr which is the generated Go expression as an IIFE.
// Returns []goast.Stmt which holds statements from all sub-expressions.
// Returns []*ast_domain.Diagnostic which holds any diagnostics from the output.
func (ee *expressionEmitter) emitTernaryExpr(n *ast_domain.TernaryExpression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	condGoExpr, condStmts, condDiags := ee.emit(n.Condition)
	consGoExpr, consStmts, consDiags := ee.emit(n.Consequent)
	altGoExpr, altStmts, altDiags := ee.emit(n.Alternate)
	allStmts := slices.Concat(condStmts, consStmts, altStmts)
	allDiags := slices.Concat(condDiags, consDiags, altDiags)

	condGoExpr = wrapInTruthinessCallIfNeeded(condGoExpr, n.Condition)

	var resultType goast.Expr = cachedIdent(goTypeAny)
	if ann := n.GoAnnotations; ann != nil && ann.ResolvedType != nil && ann.ResolvedType.TypeExpression != nil {
		resultType = ann.ResolvedType.TypeExpression
	}

	iife := &goast.CallExpr{
		Fun: &goast.FuncLit{
			Type: &goast.FuncType{Results: &goast.FieldList{List: []*goast.Field{{Type: resultType}}}},
			Body: &goast.BlockStmt{List: []goast.Stmt{
				&goast.IfStmt{
					Cond: condGoExpr,
					Body: &goast.BlockStmt{List: []goast.Stmt{&goast.ReturnStmt{Results: []goast.Expr{consGoExpr}}}},
					Else: &goast.BlockStmt{List: []goast.Stmt{&goast.ReturnStmt{Results: []goast.Expr{altGoExpr}}}},
				},
			}},
		},
	}
	return iife, allStmts, allDiags
}

// emitIdentifier creates Go code for an identifier expression.
//
// Takes n (*ast_domain.Identifier) which is the identifier node to convert.
//
// Returns goast.Expr which is the Go expression for this identifier.
// Returns []goast.Stmt which is nil because identifiers do not produce
// statements.
// Returns []*ast_domain.Diagnostic which contains an error when the identifier
// has no resolved variable name or has a synthetic type.
func (ee *expressionEmitter) emitIdentifier(n *ast_domain.Identifier) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	if isSyntheticAnnotation(n.GoAnnotations) {
		sourcePath := ""
		if n.GoAnnotations != nil && n.GoAnnotations.OriginalSourcePath != nil {
			sourcePath = ee.emitter.computeRelativePath(*n.GoAnnotations.OriginalSourcePath)
		}
		typeName := getSyntheticTypeName(n.GoAnnotations.ResolvedType)
		diagnostic := ast_domain.NewDiagnostic(
			ast_domain.Error,
			fmt.Sprintf("Cannot use '%s' (type %s) in server-side code. This is a JavaScript-only placeholder that is only valid in client-side event handlers (p-on:*).", n.Name, typeName),
			n.String(),
			n.RelativeLocation,
			sourcePath,
		)
		return cachedIdent(n.Name + "/*_SYNTHETIC_*/"), nil, []*ast_domain.Diagnostic{diagnostic}
	}

	if n.GoAnnotations != nil && n.GoAnnotations.BaseCodeGenVarName != nil {
		codeGenVar := *n.GoAnnotations.BaseCodeGenVarName

		if strings.Contains(codeGenVar, ".") {
			parts := strings.SplitN(codeGenVar, ".", 2)
			return &goast.SelectorExpr{
				X:   cachedIdent(parts[0]),
				Sel: cachedIdent(parts[1]),
			}, nil, nil
		}

		if needsCrossPackageQualification(n.GoAnnotations, ee.emitter.config.CanonicalGoPackagePath) {
			return ee.emitCrossPackageIdentifier(n.GoAnnotations, codeGenVar), nil, nil
		}

		return cachedIdent(codeGenVar), nil, nil
	}

	sourcePath := ""
	if n.GoAnnotations != nil && n.GoAnnotations.OriginalSourcePath != nil {
		sourcePath = ee.emitter.computeRelativePath(*n.GoAnnotations.OriginalSourcePath)
	}

	diagnostic := ast_domain.NewDiagnostic(
		ast_domain.Error,
		fmt.Sprintf("Internal Emitter Error: Identifier '%s' has no resolved CodeGenVarName.", n.Name),
		n.String(),
		n.RelativeLocation,
		sourcePath,
	)
	return cachedIdent(n.Name + "/*_UNRESOLVED_*/"), nil, []*ast_domain.Diagnostic{diagnostic}
}

// emitCrossPackageIdentifier creates a qualified identifier expression
// (pkg.Name) for identifiers from other packages, adding the required import.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the package path
// and alias for the external identifier.
// Takes name (string) which is the identifier name to qualify.
//
// Returns goast.Expr which is a SelectorExpr in the form pkgAlias.name.
func (ee *expressionEmitter) emitCrossPackageIdentifier(ann *ast_domain.GoGeneratorAnnotation, name string) goast.Expr {
	resolvedType := ann.ResolvedType

	ee.emitter.addImport(resolvedType.CanonicalPackagePath, resolvedType.PackageAlias)

	actualAlias := ee.emitter.getImportAlias(resolvedType.CanonicalPackagePath)
	if actualAlias == "" {
		actualAlias = resolvedType.PackageAlias
	}

	return &goast.SelectorExpr{
		X:   cachedIdent(actualAlias),
		Sel: cachedIdent(name),
	}
}

// buildNilCheckIIFEBody creates the body of a function that is called at once
// to check for nil before accessing a field.
//
// Takes baseGoExpr (goast.Expr) which is the expression to check for nil.
// Takes goFieldName (string) which is the name of the field to access.
// Takes diagnosticCall (goast.Expr) which is the call to make if nil.
// Takes zeroValue (goast.Expr) which is the value to return if nil.
//
// Returns *goast.BlockStmt which contains the nil check and field access.
func buildNilCheckIIFEBody(baseGoExpr goast.Expr, goFieldName string, diagnosticCall goast.Expr, zeroValue goast.Expr) *goast.BlockStmt {
	return &goast.BlockStmt{
		List: []goast.Stmt{
			&goast.IfStmt{
				Cond: &goast.BinaryExpr{X: baseGoExpr, Op: token.EQL, Y: cachedIdent("nil")},
				Body: &goast.BlockStmt{List: []goast.Stmt{
					&goast.AssignStmt{
						Lhs: []goast.Expr{cachedIdent(DiagnosticsVarName)},
						Tok: token.ASSIGN,
						Rhs: []goast.Expr{diagnosticCall},
					},
					&goast.ReturnStmt{Results: []goast.Expr{zeroValue}},
				}},
			},
			&goast.ReturnStmt{Results: []goast.Expr{
				&goast.SelectorExpr{X: baseGoExpr, Sel: cachedIdent(goFieldName)},
			}},
		},
	}
}

// buildNilCheckWithContext creates a nil check statement for index access.
//
// Takes ctx (*indexCheckContext) which provides the expression and emitter
// context for building the nil check.
//
// Returns goast.Stmt which is an if statement that emits a warning and returns
// a zero value when the indexed expression is nil.
func buildNilCheckWithContext(ctx *indexCheckContext) goast.Stmt {
	nilCheckMessage := fmt.Sprintf("Cannot index a nil slice/map in expression '%s'", ctx.expression.String())
	nilDiagnosticCall := ctx.emitter.buildDiagnosticCall(
		nilCheckMessage,
		"R004",
		"Warning",
		ctx.ann,
		ctx.expression.String(),
		ctx.expression.GetRelativeLocation(),
	)

	return &goast.IfStmt{
		Cond: &goast.BinaryExpr{X: ctx.baseExpr, Op: token.EQL, Y: cachedIdent("nil")},
		Body: &goast.BlockStmt{List: []goast.Stmt{
			&goast.AssignStmt{
				Lhs: []goast.Expr{cachedIdent(DiagnosticsVarName)},
				Tok: token.ASSIGN,
				Rhs: []goast.Expr{nilDiagnosticCall},
			},
			&goast.ReturnStmt{Results: []goast.Expr{ctx.zeroValue}},
		}},
	}
}

// buildBoundsCheckWithContext creates a bounds check statement for slice
// access.
//
// Takes ctx (*indexCheckContext) which provides the expression and index
// details for the bounds check.
//
// Returns goast.Stmt which is an if statement that reports a warning and
// returns a zero value when the index is outside the slice bounds.
func buildBoundsCheckWithContext(ctx *indexCheckContext) goast.Stmt {
	boundsCheckMessage := fmt.Sprintf("Index out of bounds while accessing '%s'", ctx.expression.String())
	boundsDiagnosticCall := ctx.emitter.buildDiagnosticCall(
		boundsCheckMessage,
		"R005",
		"Warning",
		ctx.ann,
		ctx.expression.String(),
		ctx.expression.GetRelativeLocation(),
	)

	return &goast.IfStmt{
		Cond: &goast.BinaryExpr{
			X:  &goast.CallExpr{Fun: cachedIdent("int"), Args: []goast.Expr{ctx.indexExpr}},
			Op: token.GEQ,
			Y:  &goast.CallExpr{Fun: cachedIdent("len"), Args: []goast.Expr{ctx.baseExpr}},
		},
		Body: &goast.BlockStmt{List: []goast.Stmt{
			&goast.AssignStmt{
				Lhs: []goast.Expr{cachedIdent(DiagnosticsVarName)},
				Tok: token.ASSIGN,
				Rhs: []goast.Expr{boundsDiagnosticCall},
			},
			&goast.ReturnStmt{Results: []goast.Expr{ctx.zeroValue}},
		}},
	}
}

// unaryOpToToken converts a Piko unary operator to its Go token equivalent.
//
// Takes operator (ast_domain.UnaryOp) which specifies the unary operator to
// convert.
//
// Returns token.Token which is the matching Go token.
// Returns bool which is true if the conversion succeeded.
func unaryOpToToken(operator ast_domain.UnaryOp) (token.Token, bool) {
	switch operator {
	case ast_domain.OpNot:
		return token.NOT, true
	case ast_domain.OpNeg:
		return token.SUB, true
	default:
		return token.ILLEGAL, false
	}
}

// needsCrossPackageQualification checks whether an identifier needs a package
// alias because it comes from a different package.
//
// This applies only to exported package-level symbols (functions, constants,
// variables) defined in the component's script block. It does not apply to
// locally-generated binding variables like props_xxx, which have the partial's
// CanonicalPackagePath but are defined in the parent's generated code.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the resolved
// type information for the identifier.
// Takes currentCanonicalPath (string) which is the canonical Go import path of
// the package being generated.
//
// Returns bool which is true when the identifier is an exported package-level
// symbol from a different package that needs a package alias.
func needsCrossPackageQualification(ann *ast_domain.GoGeneratorAnnotation, currentCanonicalPath string) bool {
	if ann == nil || ann.ResolvedType == nil {
		return false
	}

	if !ann.ResolvedType.IsExportedPackageSymbol {
		return false
	}

	canonicalPath := ann.ResolvedType.CanonicalPackagePath
	return canonicalPath != "" && canonicalPath != currentCanonicalPath
}
