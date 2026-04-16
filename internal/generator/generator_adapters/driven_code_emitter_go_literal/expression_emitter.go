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
	"strconv"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
)

const (
	// goKeywordNil is the Go nil keyword used when emitting nil values.
	goKeywordNil = "nil"

	// goTypeAny is the Go type name used when the actual type cannot be found.
	goTypeAny = "any"

	// helperEvaluateStrictEquality is the runtime helper function name for strict
	// equality checks.
	helperEvaluateStrictEquality = "EvaluateStrictEquality"

	// helperEvaluateLooseEquality is the runtime helper function name for loose
	// equality checks.
	helperEvaluateLooseEquality = "EvaluateLooseEquality"

	// helperEvaluateCoalesce is the helper function name for null coalescing
	// operations.
	helperEvaluateCoalesce = "EvaluateCoalesce"

	// helperEvaluateOr is the name of the helper function for logical OR.
	helperEvaluateOr = "EvaluateOr"

	// helperEvaluateBinary is the helper function name for binary operations.
	helperEvaluateBinary = "EvaluateBinary"
)

var (
	// builtInFunctionNames maps Go built-in function names that are emitted as direct calls.
	builtInFunctionNames = map[string]bool{
		"len":    true,
		"append": true,
		"cap":    true,
		"min":    true,
		"max":    true,
	}

	// runtimeHelperFunctionNames maps function names that are emitted as
	// pikoruntime.FuncName() calls. These are Piko built-in functions that live
	// in the wdk/runtime package.
	runtimeHelperFunctionNames = map[string]bool{
		"F": true,
	}

	// stringerBuilderCallNames lists built-in function names whose runtime return
	// type implements fmt.Stringer but which the annotator resolves as returning
	// string. The emitter wraps calls to these functions with .String() so that
	// the generated Go code compiles correctly.
	//
	// F/LF are NOT listed here because the annotator returns *FormatBuilder for
	// them, enabling the stringability pipeline to add .String() automatically.
	stringerBuilderCallNames = map[string]bool{
		"T": true, "LT": true,
	}

	// coercionFunctionNames maps coercion built-in function names.
	// These are special functions that convert values between types.
	coercionFunctionNames = map[string]bool{
		"string":  true,
		"int":     true,
		"int64":   true,
		"int32":   true,
		"int16":   true,
		"float":   true,
		"float64": true,
		"float32": true,
		"bool":    true,
		"decimal": true,
		"bigint":  true,
	}

	_ ExpressionEmitter = (*expressionEmitter)(nil)
)

// ExpressionEmitter emits Piko expressions as Go expressions.
// This enables mocking and testing of expression emission logic.
type ExpressionEmitter interface {
	// emit converts a domain expression into Go AST representation.
	//
	// Takes expression (ast_domain.Expression) which is the expression to convert.
	//
	// Returns goast.Expr which is the converted Go expression.
	// Returns []goast.Stmt which contains any statements needed before the
	// expression.
	// Returns []*ast_domain.Diagnostic which contains any issues found during
	// conversion.
	emit(expr ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic)

	// valueToString converts a Go expression to its string representation.
	//
	// Takes goExpr (goast.Expr) which is the expression to convert.
	// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides context for the
	// conversion.
	//
	// Returns goast.Expr which is the string representation of the input.
	valueToString(goExpr goast.Expr, ann *ast_domain.GoGeneratorAnnotation) goast.Expr

	// getTypeExprForVarDecl returns the type expression for a variable
	// declaration.
	//
	// Takes ann (*ast_domain.GoGeneratorAnnotation) which contains the annotation
	// to extract the type expression from.
	//
	// Returns goast.Expr which is the type expression for the variable.
	getTypeExprForVarDecl(ann *ast_domain.GoGeneratorAnnotation) goast.Expr

	// emitTemplateLiteralParts extracts each part of a template literal as a
	// separate Go expression, without concatenating them. This is used for
	// generating variadic calls like BuildClassBytesV(part1, part2, ...) that
	// avoid intermediate string allocation from the + operator.
	//
	// Takes n (*ast_domain.TemplateLiteral) which is the template literal to
	// process.
	//
	// Returns []goast.Expr which contains one Go expression per template part.
	// Returns []goast.Stmt which contains any prerequisite statements.
	// Returns []*ast_domain.Diagnostic which contains any issues found.
	emitTemplateLiteralParts(n *ast_domain.TemplateLiteral) ([]goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic)
}

// expressionEmitter translates annotated Piko AST expressions into Go AST
// expressions. It implements ExpressionEmitter and holds references to its
// specialist sub-emitters.
type expressionEmitter struct {
	// emitter provides shared output generation and import management.
	emitter *emitter

	// binaryEmitter handles binary operations; uses interface type for
	// flexibility.
	binaryEmitter BinaryOpEmitter

	// stringConv converts values to their string form.
	stringConv StringConverter
}

// emit is the main entry point for converting expressions.
//
// It converts a Piko AST expression to a Go AST expression. It tries several
// conversion methods in order: collection calls, operators, literals, and
// composite expressions.
//
// Takes expression (ast_domain.Expression) which is the Piko AST expression to
// convert.
//
// Returns goast.Expr which is the matching Go AST expression.
// Returns []goast.Stmt which contains any setup statements needed first.
// Returns []*ast_domain.Diagnostic which contains any issues found.
func (ee *expressionEmitter) emit(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	if expression == nil {
		return cachedIdent(goKeywordNil), nil, nil
	}

	if goExpr, statements, diagnostics, handled := ee.tryEmitCollectionCall(expression); handled {
		return goExpr, statements, diagnostics
	}

	if goExpr, statements, diagnostics, handled := ee.tryEmitOperatorExpression(expression); handled {
		return goExpr, statements, diagnostics
	}

	if goExpr, statements, diagnostics, handled := ee.tryEmitLiteralExpression(expression); handled {
		return goExpr, statements, diagnostics
	}

	if goExpr, statements, diagnostics, handled := ee.tryEmitCompositeExpression(expression); handled {
		return goExpr, statements, diagnostics
	}

	return ee.emitUnhandledExpression(expression)
}

// tryEmitLiteralExpression handles literal value expressions.
//
// Takes expression (ast_domain.Expression) which is the
// expression to check and emit.
//
// Returns goast.Expr which is the Go AST literal expression, or nil if the
// input is not a literal.
// Returns []goast.Stmt which is always nil for literal expressions.
// Returns []*ast_domain.Diagnostic which is always nil for literal expressions.
// Returns bool which is true if the expression was a literal type.
func (ee *expressionEmitter) tryEmitLiteralExpression(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic, bool) {
	switch n := expression.(type) {
	case *ast_domain.StringLiteral:
		return strLit(n.Value), nil, nil, true
	case *ast_domain.IntegerLiteral:
		return &goast.BasicLit{Kind: TokenKindInt, Value: strconv.FormatInt(n.Value, 10)}, nil, nil, true
	case *ast_domain.FloatLiteral:
		return &goast.BasicLit{Kind: TokenKindFloat, Value: strconv.FormatFloat(n.Value, 'f', -1, 64)}, nil, nil, true
	case *ast_domain.BooleanLiteral:
		return cachedIdent(strconv.FormatBool(n.Value)), nil, nil, true
	case *ast_domain.NilLiteral:
		return cachedIdent(goKeywordNil), nil, nil, true
	case *ast_domain.DecimalLiteral:
		ee.emitter.addImport(mathsPackagePath, pkgMaths)
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: cachedIdent(pkgMaths), Sel: cachedIdent(mathsNewDecimalFromString)},
			Args: []goast.Expr{strLit(n.Value)},
		}, nil, nil, true
	case *ast_domain.BigIntLiteral:
		ee.emitter.addImport(mathsPackagePath, pkgMaths)
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: cachedIdent(pkgMaths), Sel: cachedIdent(mathsNewBigIntFromString)},
			Args: []goast.Expr{strLit(n.Value)},
		}, nil, nil, true
	case *ast_domain.RuneLiteral:
		return &goast.BasicLit{Kind: TokenKindChar, Value: strconv.QuoteRune(n.Value)}, nil, nil, true
	case *ast_domain.DateTimeLiteral:
		return ee.emitTemporalParseLiteral(&goast.SelectorExpr{X: cachedIdent("time"), Sel: cachedIdent("RFC3339")}, n.Value), nil, nil, true
	case *ast_domain.DateLiteral:
		return ee.emitTemporalParseLiteral(strLit(timeDateFormat), n.Value), nil, nil, true
	case *ast_domain.TimeLiteral:
		return ee.emitTemporalParseLiteral(strLit(timeTimeFormat), n.Value), nil, nil, true
	case *ast_domain.DurationLiteral:
		return ee.emitDurationParseLiteral(n.Value), nil, nil, true
	}
	return nil, nil, nil, false
}

// emitTemporalParseLiteral generates an IIFE that parses a time string literal.
// The generated code is: func() time.Time { t, _ := time.Parse(layout, value);
// return t }() The parser has already validated the format, so the error is
// unreachable.
//
// Takes layoutExpr (goast.Expr) which is the layout argument (string literal or
// time.RFC3339 selector).
// Takes value (string) which is the time string to parse.
//
// Returns goast.Expr which is the IIFE call expression producing a time.Time.
func (ee *expressionEmitter) emitTemporalParseLiteral(layoutExpr goast.Expr, value string) goast.Expr {
	ee.emitter.addImport(timePackagePath, "")

	parseCall := &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent("time"), Sel: cachedIdent(timeParseFunc)},
		Args: []goast.Expr{layoutExpr, strLit(value)},
	}

	return buildTemporalIIFE("Time", "t", parseCall)
}

// emitDurationParseLiteral generates an IIFE that parses a duration string
// literal. The generated code is: func() time.Duration { d, _ :=
// time.ParseDuration(value); return d }() The parser has already validated the
// format, so the error is unreachable.
//
// Takes value (string) which is the duration string to parse.
//
// Returns goast.Expr which is the IIFE call expression producing a
// time.Duration.
func (ee *expressionEmitter) emitDurationParseLiteral(value string) goast.Expr {
	ee.emitter.addImport(timePackagePath, "")

	parseCall := &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent("time"), Sel: cachedIdent(timeParseDuration)},
		Args: []goast.Expr{strLit(value)},
	}

	return buildTemporalIIFE("Duration", "d", parseCall)
}

// emitUnhandledExpression handles expression types that are not known.
//
// Takes expression (ast_domain.Expression) which is the
// expression that could not be matched to a known type.
//
// Returns goast.Expr which is a nil placeholder with a comment noting the
// unhandled type.
// Returns []goast.Stmt which is always nil.
// Returns []*ast_domain.Diagnostic which contains an error describing the
// unhandled expression type.
func (*expressionEmitter) emitUnhandledExpression(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	diagnostic := ast_domain.NewDiagnostic(
		ast_domain.Error,
		fmt.Sprintf("Internal Emitter Error: unhandled expression type '%T' in generator.", expression),
		expression.String(),
		expression.GetRelativeLocation(),
		"",
	)
	placeholder := cachedIdent(fmt.Sprintf("%s /* unhandled expr type: %T */", goKeywordNil, expression))
	return placeholder, nil, []*ast_domain.Diagnostic{diagnostic}
}

// valueToString passes the work to the stringConverter helper.
//
// Takes goExpr (goast.Expr) which is the expression to convert.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides generator
// context.
//
// Returns goast.Expr which is the string conversion expression.
func (ee *expressionEmitter) valueToString(goExpr goast.Expr, ann *ast_domain.GoGeneratorAnnotation) goast.Expr {
	return ee.stringConv.valueToString(goExpr, ann)
}

// getTypeExprForVarDecl determines the correct type expression to use for a
// variable declaration.
//
// It strips the package qualifier if the type belongs to the package currently
// being generated. It also handles import alias conflicts by updating the type
// expression to use the resolved alias.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the resolved
// type information for the variable.
//
// Returns goast.Expr which is the type expression to use, or an 'any' type
// identifier when the annotation is nil or has no resolved type.
func (ee *expressionEmitter) getTypeExprForVarDecl(ann *ast_domain.GoGeneratorAnnotation) goast.Expr {
	if ann == nil || ann.ResolvedType == nil || ann.ResolvedType.TypeExpression == nil {
		return cachedIdent(goTypeAny)
	}

	resolvedType := ann.ResolvedType

	if resolvedType.IsSynthetic {
		return cachedIdent(goTypeAny)
	}

	if identifier, ok := resolvedType.TypeExpression.(*goast.Ident); ok && identifier.Name == "function" {
		return cachedIdent(goTypeAny)
	}
	currentCanonicalPath := ee.emitter.config.CanonicalGoPackagePath

	if resolvedType.CanonicalPackagePath == currentCanonicalPath {
		return goastutil.UnqualifyTypeExpr(resolvedType.TypeExpression)
	}

	if resolvedType.CanonicalPackagePath != "" && !isBasicGoType(resolvedType.TypeExpression) {
		ee.emitter.addImport(resolvedType.CanonicalPackagePath, resolvedType.PackageAlias)

		actualAlias := ee.emitter.getImportAlias(resolvedType.CanonicalPackagePath)
		if actualAlias != "" && actualAlias != resolvedType.PackageAlias {
			return updateTypeExprAlias(resolvedType.TypeExpression, actualAlias)
		}
	}

	return resolvedType.TypeExpression
}

// wrapWithStringerCall wraps a Go expression with a .String() method call.
// Used for builder-pattern functions (T, LT, F, LF) whose runtime return type
// implements fmt.Stringer.
//
// Takes expression (goast.Expr) which is the expression to wrap.
//
// Returns *goast.CallExpr which calls .String() on the expression.
func wrapWithStringerCall(expression goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{
		Fun: &goast.SelectorExpr{X: expression, Sel: cachedIdent("String")},
	}
}

// newExpressionEmitter creates a new emitter for expressions.
//
// Takes emitter (*emitter) which provides the base emitting functions.
// Takes binaryEmitter (BinaryOpEmitter) which handles binary operations.
// Takes stringConv (StringConverter) which converts values to strings.
//
// Returns *expressionEmitter which is ready to emit expressions.
func newExpressionEmitter(emitter *emitter, binaryEmitter BinaryOpEmitter, stringConv StringConverter) *expressionEmitter {
	return &expressionEmitter{
		emitter:       emitter,
		binaryEmitter: binaryEmitter,
		stringConv:    stringConv,
	}
}

// buildTemporalIIFE creates an IIFE AST node that calls parseCall,
// discards the error, and returns the result using the given typeName
// (e.g. "Time" or "Duration") and varName for the local assignment.
//
// Takes typeName (string) which is the time package type name to return.
// Takes varName (string) which is the local variable name in the IIFE.
// Takes parseCall (goast.Expr) which is the parse function call
// expression.
//
// Returns *goast.CallExpr which is the IIFE that parses and returns the
// temporal value.
func buildTemporalIIFE(typeName, varName string, parseCall goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{
		Fun: &goast.FuncLit{
			Type: &goast.FuncType{
				Params: &goast.FieldList{},
				Results: &goast.FieldList{List: []*goast.Field{{
					Type: &goast.SelectorExpr{X: cachedIdent("time"), Sel: cachedIdent(typeName)},
				}}},
			},
			Body: &goast.BlockStmt{List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs: []goast.Expr{cachedIdent(varName), cachedIdent("_")},
					Tok: token.DEFINE,
					Rhs: []goast.Expr{parseCall},
				},
				&goast.ReturnStmt{Results: []goast.Expr{cachedIdent(varName)}},
			}},
		},
	}
}

// updateTypeExprAlias creates a copy of a type expression with the package
// alias changed to a new value. This is used when fixing import alias clashes
// by creating unique aliases.
//
// Takes typeExpr (goast.Expr) which is the type expression to update.
// Takes newAlias (string) which is the new package alias to use.
//
// Returns goast.Expr which is a new expression with the alias replaced.
func updateTypeExprAlias(typeExpr goast.Expr, newAlias string) goast.Expr {
	switch t := typeExpr.(type) {
	case *goast.SelectorExpr:
		return &goast.SelectorExpr{
			X:   cachedIdent(newAlias),
			Sel: t.Sel,
		}
	case *goast.StarExpr:
		return &goast.StarExpr{
			X: updateTypeExprAlias(t.X, newAlias),
		}
	case *goast.ArrayType:
		return &goast.ArrayType{
			Len: t.Len,
			Elt: updateTypeExprAlias(t.Elt, newAlias),
		}
	case *goast.MapType:
		return &goast.MapType{
			Key:   updateTypeExprAlias(t.Key, newAlias),
			Value: updateTypeExprAlias(t.Value, newAlias),
		}
	case *goast.IndexExpr:
		return &goast.IndexExpr{
			X:     updateTypeExprAlias(t.X, newAlias),
			Index: updateTypeExprAlias(t.Index, newAlias),
		}
	default:
		return typeExpr
	}
}
