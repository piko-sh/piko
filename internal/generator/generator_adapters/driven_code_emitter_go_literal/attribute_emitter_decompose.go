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
	goast "go/ast"
	"go/token"

	"piko.sh/piko/internal/ast/ast_domain"
)

// strconvHandler defines how to emit a typed append call for a strconv
// function.
type strconvHandler struct {
	// emitter builds an AST statement for the strconv optimisation.
	emitter func(ae *attributeEmitter, dwVar *goast.Ident, argument goast.Expr) goast.Stmt

	// convertToInt indicates whether to wrap the argument with int64() for Itoa.
	convertToInt bool

	// isFloat indicates a FormatFloat call that needs special handling
	// in p-key context.
	isFloat bool
}

// strconvHandlers maps strconv function names to their optimised handlers.
var strconvHandlers = map[string]strconvHandler{
	"FormatInt":   {emitter: (*attributeEmitter).emitAppendInt, convertToInt: false, isFloat: false},
	"Itoa":        {emitter: (*attributeEmitter).emitAppendInt, convertToInt: true, isFloat: false},
	"FormatUint":  {emitter: (*attributeEmitter).emitAppendUint, convertToInt: false, isFloat: false},
	"FormatFloat": {emitter: (*attributeEmitter).emitAppendFloat, convertToInt: false, isFloat: true},
	"FormatBool":  {emitter: (*attributeEmitter).emitAppendBool, convertToInt: false, isFloat: false},
}

// emitKeyWriterParts generates code to build a DirectWriter from expression
// parts via AttributeWriters. It decomposes string concatenations into typed
// Append calls for zero-allocation rendering.
//
// Strategy: First emit the Piko expression to get the Go AST, ensuring it is
// converted to string via valueToString. Then decompose the Go AST to
// intercept strconv patterns and emit the optimal Append method for each part
// (AppendInt, AppendUint, etc.).
//
// Takes nodeVar (*goast.Ident) which identifies the node to append writers to.
// Takes keyExpr (ast_domain.Expression) which is the key expression to emit.
//
// Returns []goast.Stmt which contains the generated statements.
// Returns []*ast_domain.Diagnostic which contains any diagnostics produced.
func (ae *attributeEmitter) emitKeyWriterParts(
	nodeVar *goast.Ident,
	keyExpr ast_domain.Expression,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	dwVar := ae.emitter.nextTempName()
	dwIdent := cachedIdent(dwVar)

	diagnostics := make([]*ast_domain.Diagnostic, 0, defaultDiagnosticCapacity)

	statements := make([]goast.Stmt, 0, directWriterStatementCapacity)
	statements = append(statements,
		defineAndAssign(dwVar, &goast.CallExpr{
			Fun: &goast.SelectorExpr{X: cachedIdent(arenaVarName), Sel: cachedIdent("GetDirectWriter")},
		}),
		&goast.ExprStmt{X: &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: dwIdent, Sel: cachedIdent(directWriterMethodSetName)},
			Args: []goast.Expr{strLit(pkeyAttributeName)},
		}},
	)

	goExpr, prereqs, expressionDiagnostics := ae.expressionEmitter.emit(keyExpr)
	statements = append(statements, prereqs...)
	diagnostics = append(diagnostics, expressionDiagnostics...)

	ann := getAnnotationFromExpression(keyExpr)
	goExpr = ae.expressionEmitter.valueToString(goExpr, ann)

	statements = append(statements, ae.decomposeGoExpr(dwIdent, goExpr, decomposeContextPKey)...)

	statements = append(statements, appendToSlice(&goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldAttributeWriters)}, dwIdent))

	return statements, diagnostics
}

// decomposeGoExpr recursively decomposes a Go AST expression into
// DirectWriter Append calls. This handles Go binary expressions (string
// concatenation), string literals, and strconv.Format* calls to extract the
// optimal Append method for each part.
//
// Takes dwVar (*goast.Ident) which is the DirectWriter variable to append to.
// Takes expression (goast.Expr) which is the expression to decompose.
// Takes ctx (decomposeContext) which controls how dynamic expressions are
// handled: decomposeContextAttribute uses AppendEscapeString for untrusted
// strings (XSS protection), decomposeContextPKey uses AppendFNV* for untrusted
// strings/floats/any (bounded safe output).
//
// Returns []goast.Stmt which contains the generated Append call statements.
//
// Pattern detection for zero-allocation rendering:
//   - strconv.FormatInt -> AppendInt (int64) - safe in all contexts
//   - strconv.FormatUint -> AppendUint (uint64) - safe in all contexts
//   - strconv.FormatFloat -> AppendFloat or AppendFNVFloat (context-dependent)
//   - strconv.FormatBool -> AppendBool (bool) - safe in all contexts
//   - strconv.Itoa -> AppendInt (converted to int64) - safe in all contexts
//   - string literals -> AppendString (always trusted - developer content)
//   - other expressions -> context-dependent handling
func (ae *attributeEmitter) decomposeGoExpr(dwVar *goast.Ident, expression goast.Expr, ctx decomposeContext) []goast.Stmt {
	switch e := expression.(type) {
	case *goast.BinaryExpr:
		return ae.decomposeBinaryExpr(dwVar, e, ctx)

	case *goast.BasicLit:
		return ae.decomposeBasicLit(dwVar, e, ctx)

	case *goast.CallExpr:
		return ae.decomposeCallExpr(dwVar, e, ctx)

	case *goast.ParenExpr:
		return ae.decomposeGoExpr(dwVar, e.X, ctx)

	default:
		return []goast.Stmt{ae.emitDynamicExpr(dwVar, e, ctx)}
	}
}

// decomposeBinaryExpr handles binary expressions such as string joining and
// other operators.
//
// Takes dwVar (*goast.Ident) which is the variable to append results to.
// Takes e (*goast.BinaryExpr) which is the binary expression to process.
// Takes ctx (decomposeContext) which provides the context for processing.
//
// Returns []goast.Stmt which contains the statements that emit the expression.
func (ae *attributeEmitter) decomposeBinaryExpr(dwVar *goast.Ident, e *goast.BinaryExpr, ctx decomposeContext) []goast.Stmt {
	if e.Op != token.ADD {
		return []goast.Stmt{ae.emitAppendString(dwVar, e)}
	}
	statements := ae.decomposeGoExpr(dwVar, e.X, ctx)
	return append(statements, ae.decomposeGoExpr(dwVar, e.Y, ctx)...)
}

// decomposeBasicLit handles literal values (strings, ints, floats),
// where all literals are safe developer-controlled content and float
// literals are deterministic compile-time constants.
//
// Takes dwVar (*goast.Ident) which is the DirectWriter variable to
// append to.
// Takes e (*goast.BasicLit) which is the literal expression to emit.
//
// Returns []goast.Stmt which contains the append statement for the
// literal value.
func (ae *attributeEmitter) decomposeBasicLit(dwVar *goast.Ident, e *goast.BasicLit, _ decomposeContext) []goast.Stmt {
	switch e.Kind {
	case token.STRING:
		return []goast.Stmt{&goast.ExprStmt{X: &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: dwVar, Sel: cachedIdent("AppendString")},
			Args: []goast.Expr{e},
		}}}
	case token.INT:
		return []goast.Stmt{ae.emitAppendInt(dwVar, e)}
	case token.FLOAT:
		return []goast.Stmt{ae.emitAppendFloat(dwVar, e)}
	default:
		return []goast.Stmt{ae.emitAppendString(dwVar, e)}
	}
}

// decomposeCallExpr handles function calls, using faster strconv.Format*
// patterns when possible.
//
// Takes dwVar (*goast.Ident) which is the variable to store the result.
// Takes e (*goast.CallExpr) which is the call expression to process.
// Takes ctx (decomposeContext) which provides the processing context.
//
// Returns []goast.Stmt which contains the statements for the call expression.
func (ae *attributeEmitter) decomposeCallExpr(dwVar *goast.Ident, e *goast.CallExpr, ctx decomposeContext) []goast.Stmt {
	if statement := ae.tryEmitStrconvOptimisation(dwVar, e, ctx); statement != nil {
		return []goast.Stmt{statement}
	}
	return []goast.Stmt{ae.emitDynamicExpr(dwVar, e, ctx)}
}

// tryEmitStrconvOptimisation checks if a call is a strconv function and
// emits an optimised append statement.
//
// Takes dwVar (*goast.Ident) which is the variable to append to.
// Takes call (*goast.CallExpr) which is the strconv call to optimise.
// Takes ctx (decomposeContext) which is the decomposition context.
//
// Returns goast.Stmt which is the optimised append statement, or nil if not a
// strconv call or if the call has too few arguments.
//
// In p-key context, floats are FNV-hashed to avoid precision issues in the
// string representation.
func (ae *attributeEmitter) tryEmitStrconvOptimisation(dwVar *goast.Ident, call *goast.CallExpr, ctx decomposeContext) goast.Stmt {
	functionName := ae.getStrconvFuncName(call)
	handler, ok := strconvHandlers[functionName]
	if !ok || len(call.Args) < 1 {
		return nil
	}

	argument := call.Args[0]
	if handler.convertToInt {
		argument = &goast.CallExpr{Fun: cachedIdent("int64"), Args: []goast.Expr{argument}}
	}

	if handler.isFloat && ctx == decomposeContextPKey {
		return ae.emitAppendFNVFloat(dwVar, argument)
	}

	return handler.emitter(ae, dwVar, argument)
}

// emitDynamicExpr creates the correct append call based on the context.
//
// Takes dwVar (*goast.Ident) which is the variable to append to.
// Takes expression (goast.Expr) which is the expression to emit.
// Takes ctx (decomposeContext) which sets the escaping method.
//
// Returns goast.Stmt which is the created append call statement.
//
// For attribute context, uses AppendEscapeString to stop XSS attacks.
// For p-key context, uses AppendFNVString to produce bounded, safe output.
func (ae *attributeEmitter) emitDynamicExpr(dwVar *goast.Ident, expression goast.Expr, ctx decomposeContext) goast.Stmt {
	if ctx == decomposeContextPKey {
		return ae.emitAppendFNVString(dwVar, expression)
	}
	return ae.emitAppendEscapeString(dwVar, expression)
}

// getStrconvFuncName checks if a call expression is a strconv function and
// returns its name.
//
// Takes call (*goast.CallExpr) which is the call expression to check.
//
// Returns string which is the strconv function name, or an empty string if
// the call is not a strconv function.
func (*attributeEmitter) getStrconvFuncName(call *goast.CallExpr) string {
	selectorExpression, ok := call.Fun.(*goast.SelectorExpr)
	if !ok {
		return ""
	}
	id, ok := selectorExpression.X.(*goast.Ident)
	if !ok || id.Name != "strconv" {
		return ""
	}
	return selectorExpression.Sel.Name
}

// emitAppendInt generates a statement that appends an integer to a writer.
//
// Takes dwVar (*goast.Ident) which is the data writer variable identifier.
// Takes intExpr (goast.Expr) which is the integer expression to append.
//
// Returns goast.Stmt which is the generated dw.AppendInt(expr) statement.
func (*attributeEmitter) emitAppendInt(dwVar *goast.Ident, intExpr goast.Expr) goast.Stmt {
	return &goast.ExprStmt{X: &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: dwVar, Sel: cachedIdent("AppendInt")},
		Args: []goast.Expr{intExpr},
	}}
}

// emitAppendUint generates a statement that appends an unsigned integer.
//
// Takes dwVar (*goast.Ident) which identifies the data writer variable.
// Takes uintExpr (goast.Expr) which provides the unsigned integer expression.
//
// Returns goast.Stmt which is the generated dw.AppendUint(expr) statement.
func (*attributeEmitter) emitAppendUint(dwVar *goast.Ident, uintExpr goast.Expr) goast.Stmt {
	return &goast.ExprStmt{X: &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: dwVar, Sel: cachedIdent("AppendUint")},
		Args: []goast.Expr{uintExpr},
	}}
}

// emitAppendFloat generates a statement that appends a float value.
//
// Takes dwVar (*goast.Ident) which identifies the data writer variable.
// Takes floatExpr (goast.Expr) which provides the float expression to append.
//
// Returns goast.Stmt which calls dw.AppendFloat(expr) on the data writer.
func (*attributeEmitter) emitAppendFloat(dwVar *goast.Ident, floatExpr goast.Expr) goast.Stmt {
	return &goast.ExprStmt{X: &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: dwVar, Sel: cachedIdent("AppendFloat")},
		Args: []goast.Expr{floatExpr},
	}}
}

// emitAppendBool generates a statement that appends a boolean to the writer.
//
// Takes dwVar (*goast.Ident) which identifies the DocWriter variable.
// Takes boolExpr (goast.Expr) which is the boolean expression to append.
//
// Returns goast.Stmt which is the generated dw.AppendBool(expr) call.
func (*attributeEmitter) emitAppendBool(dwVar *goast.Ident, boolExpr goast.Expr) goast.Stmt {
	return &goast.ExprStmt{X: &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: dwVar, Sel: cachedIdent("AppendBool")},
		Args: []goast.Expr{boolExpr},
	}}
}

// emitAppendString generates a statement that calls dw.AppendString(expr).
//
// Takes dwVar (*goast.Ident) which is the identifier for the data writer.
// Takes strExpr (goast.Expr) which is the string expression to append.
//
// Returns goast.Stmt which is the generated method call statement.
func (*attributeEmitter) emitAppendString(dwVar *goast.Ident, strExpr goast.Expr) goast.Stmt {
	return &goast.ExprStmt{X: &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: dwVar, Sel: cachedIdent("AppendString")},
		Args: []goast.Expr{strExpr},
	}}
}

// emitAppendEscapeString generates a dw.AppendEscapeString(expr) statement.
// Used for dynamic string expressions that may contain user input requiring
// HTML escaping.
//
// Takes dwVar (*goast.Ident) which is the document writer variable.
// Takes strExpr (goast.Expr) which is the string expression to escape.
//
// Returns goast.Stmt which is the generated method call statement.
func (*attributeEmitter) emitAppendEscapeString(dwVar *goast.Ident, strExpr goast.Expr) goast.Stmt {
	return &goast.ExprStmt{X: &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: dwVar, Sel: cachedIdent("AppendEscapeString")},
		Args: []goast.Expr{strExpr},
	}}
}

// emitAppendFNVString generates a statement that calls dw.AppendFNVString
// with the given expression for FNV-32 hashing of dynamic strings.
//
// Takes dwVar (*goast.Ident) which is the data writer variable.
// Takes strExpr (goast.Expr) which is the string expression to hash.
//
// Returns goast.Stmt which is the generated method call statement.
func (*attributeEmitter) emitAppendFNVString(dwVar *goast.Ident, strExpr goast.Expr) goast.Stmt {
	return &goast.ExprStmt{X: &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: dwVar, Sel: cachedIdent("AppendFNVString")},
		Args: []goast.Expr{strExpr},
	}}
}

// emitAppendFNVFloat generates a statement that appends an FNV-hashed float
// to the data writer.
//
// Takes dwVar (*goast.Ident) which identifies the data writer variable.
// Takes floatExpr (goast.Expr) which is the float expression to hash.
//
// Returns goast.Stmt which is the generated dw.AppendFNVFloat(expr) call.
//
// Used for p-key values where floats need FNV-32 hashing to avoid confusion
// with key path delimiters (floats contain '.') and to produce consistent
// 8-character output regardless of precision.
func (*attributeEmitter) emitAppendFNVFloat(dwVar *goast.Ident, floatExpr goast.Expr) goast.Stmt {
	return &goast.ExprStmt{X: &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: dwVar, Sel: cachedIdent("AppendFNVFloat")},
		Args: []goast.Expr{floatExpr},
	}}
}

// emitDynamicAttributeWriter creates DirectWriter-based code for a dynamic
// attribute, rendering without memory allocation and with proper HTML escaping
// at render time.
//
// Takes nodeVar (*goast.Ident) which identifies the node to attach the
// attribute writer to.
// Takes attributeName (string) which specifies the HTML attribute name.
// Takes pikoExpr (ast_domain.Expression) which provides the dynamic value
// expression.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which supplies generation
// metadata.
//
// Returns []goast.Stmt which contains the generated Go statements.
// Returns []*ast_domain.Diagnostic which contains any diagnostics found.
func (ae *attributeEmitter) emitDynamicAttributeWriter(
	nodeVar *goast.Ident,
	attributeName string,
	pikoExpr ast_domain.Expression,
	ann *ast_domain.GoGeneratorAnnotation,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	dwVar := ae.emitter.nextTempName()
	dwIdent := cachedIdent(dwVar)

	diagnostics := make([]*ast_domain.Diagnostic, 0, defaultDiagnosticCapacity)

	statements := make([]goast.Stmt, 0, directWriterStatementCapacity)
	statements = append(statements,
		defineAndAssign(dwVar, &goast.CallExpr{
			Fun: &goast.SelectorExpr{X: cachedIdent(arenaVarName), Sel: cachedIdent("GetDirectWriter")},
		}),
		&goast.ExprStmt{X: &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: dwIdent, Sel: cachedIdent(directWriterMethodSetName)},
			Args: []goast.Expr{strLit(attributeName)},
		}},
	)

	goExpr, prereqs, expressionDiagnostics := ae.expressionEmitter.emit(pikoExpr)
	statements = append(statements, prereqs...)
	diagnostics = append(diagnostics, expressionDiagnostics...)

	goExpr = ae.expressionEmitter.valueToString(goExpr, ann)

	statements = append(statements, ae.decomposeGoExpr(dwIdent, goExpr, decomposeContextAttribute)...)

	statements = append(statements, appendToSlice(&goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldAttributeWriters)}, dwIdent))

	return statements, diagnostics
}
