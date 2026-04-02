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
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

// getSourcePath extracts the source path from an annotation for use in
// diagnostic messages. The path is converted to a relative path using the
// emitter's base directory.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the annotation
// containing the original source path.
//
// Returns string which is the relative source path, or an empty string if the
// annotation or its path is nil.
func (ae *attributeEmitter) getSourcePath(ann *ast_domain.GoGeneratorAnnotation) string {
	if ann != nil && ann.OriginalSourcePath != nil {
		return ae.emitter.computeRelativePath(*ann.OriginalSourcePath)
	}
	return ""
}

// buildAttributeIfNotEmpty builds an if statement that appends an attribute
// to a slice only when the value is not empty.
//
// Takes attributeSlice (goast.Expr) which is the slice to append to.
// Takes attributeName (string) which is the name of the HTML attribute.
// Takes valueVar (goast.Expr) which is the expression for the attribute value.
//
// Returns []goast.Stmt which contains the if statement that adds the attribute
// when the value is not empty.
func (*attributeEmitter) buildAttributeIfNotEmpty(attributeSlice goast.Expr, attributeName string, valueVar goast.Expr) []goast.Stmt {
	attributeLiteral := &goast.CompositeLit{
		Type: cachedIdent(fmt.Sprintf(HTMLAttributeTypeFmt, runtimePackageName)),
		Elts: []goast.Expr{
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameName), Value: strLit(attributeName)},
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameValue), Value: valueVar},
		},
	}
	ifStmt := &goast.IfStmt{
		Cond: &goast.BinaryExpr{X: valueVar, Op: token.NEQ, Y: strLit("")},
		Body: &goast.BlockStmt{List: []goast.Stmt{appendToSlice(attributeSlice, attributeLiteral)}},
	}
	return []goast.Stmt{ifStmt}
}

// wrapWithModulePathResolver wraps an expression with a runtime call to
// ResolveModulePath, enabling @/ alias resolution for dynamic paths at runtime.
//
// Takes expression (goast.Expr) which is the expression to wrap
// with path resolution.
//
// Returns goast.Expr which is a call expression that invokes
// pikoruntime.ResolveModulePath with the given expression and module name.
func (ae *attributeEmitter) wrapWithModulePathResolver(expression goast.Expr) goast.Expr {
	return &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   cachedIdent(runtimePackageName),
			Sel: cachedIdent("ResolveModulePath"),
		},
		Args: []goast.Expr{
			expression,
			strLit(ae.emitter.config.ModuleName),
		},
	}
}

// buildDirectWriterAttributeIfNotNil generates an if block that sets the name
// on a DirectWriter and appends it to AttributeWriters only when the buffer
// pointer is not nil.
//
// Takes nodeVar (*goast.Ident) which identifies the node variable.
// Takes attributeName (string) which is the name of the HTML attribute.
// Takes dwVar (goast.Expr) which is the DirectWriter variable.
// Takes bufferPointerVar (goast.Expr) which is the buffer pointer
// to check for nil.
//
// Returns []goast.Stmt which contains the if statement that conditionally
// sets up and appends the DirectWriter.
//
// The generated code checks if bufferPointerVar is not nil, then calls SetName on
// the DirectWriter and appends it to the node's AttributeWriters slice.
func (*attributeEmitter) buildDirectWriterAttributeIfNotNil(nodeVar *goast.Ident, attributeName string, dwVar, bufferPointerVar goast.Expr) []goast.Stmt {
	ifBody := []goast.Stmt{
		&goast.ExprStmt{X: &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: dwVar, Sel: cachedIdent(directWriterMethodSetName)},
			Args: []goast.Expr{strLit(attributeName)},
		}},
		appendToSlice(&goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldAttributeWriters)}, dwVar),
	}

	ifStmt := &goast.IfStmt{
		Cond: &goast.BinaryExpr{X: bufferPointerVar, Op: token.NEQ, Y: cachedIdent("nil")},
		Body: &goast.BlockStmt{List: ifBody},
	}
	return []goast.Stmt{ifStmt}
}

// isExpressionIntrinsicallySafe recursively analyses an AST expression to
// determine if it can only produce XSS-safe strings (no HTML special
// characters). This is used to optimise HTML escaping for p-key attributes.
//
// Safe expressions include:
//   - String literals (static text)
//   - Numeric formatting functions (strconv.FormatInt, strconv.FormatFloat)
//   - Primitive non-string types with Stringability
//   - Binary + operations where both sides are safe
//
// Unsafe expressions include:
//   - String variables from unknown sources (could be user input)
//   - Map keys that could be user-controlled
//   - Member accesses on string fields
//
// Delegates to specialised type checkers following the Extract Method pattern.
//
// Takes expression (ast_domain.Expression) which is the expression to analyse.
//
// Returns bool which is true if the expression is intrinsically safe.
func isExpressionIntrinsicallySafe(expression ast_domain.Expression) bool {
	if expression == nil {
		return false
	}

	switch e := expression.(type) {
	case *ast_domain.StringLiteral, *ast_domain.IntegerLiteral, *ast_domain.FloatLiteral, *ast_domain.BooleanLiteral:
		return true

	case *ast_domain.BinaryExpression:
		return isSafeBinaryExpr(e)

	case *ast_domain.CallExpression:
		return isSafeCallExpr(e)

	case *ast_domain.Identifier, *ast_domain.MemberExpression, *ast_domain.IndexExpression:
		return isSafePrimitiveIdentifier(expression)

	case *ast_domain.UnaryExpression:
		return isExpressionIntrinsicallySafe(e.Right)

	case *ast_domain.TernaryExpression:
		return isExpressionIntrinsicallySafe(e.Consequent) && isExpressionIntrinsicallySafe(e.Alternate)

	case *ast_domain.TemplateLiteral:
		return isSafeTemplateLiteral(e)

	default:
		return false
	}
}

// isSafeBinaryExpr checks whether a binary expression produces safe output.
//
// Takes e (*ast_domain.BinaryExpression) which is the binary expression to check.
//
// Returns bool which is true if both operands are safe.
func isSafeBinaryExpr(e *ast_domain.BinaryExpression) bool {
	return isExpressionIntrinsicallySafe(e.Left) && isExpressionIntrinsicallySafe(e.Right)
}

// isSafeCallExpr checks if a function call returns output that is safe to use
// without escaping.
//
// Takes e (*ast_domain.CallExpression) which is the call expression to check.
//
// Returns bool which is true if the call is to a known safe function, such as
// strconv conversion functions.
func isSafeCallExpr(e *ast_domain.CallExpression) bool {
	memberExpr, ok := e.Callee.(*ast_domain.MemberExpression)
	if !ok {
		return false
	}

	pkgIdent, ok := memberExpr.Base.(*ast_domain.Identifier)
	if !ok {
		return false
	}

	if pkgIdent.Name == "strconv" {
		return isSafeStrconvFunction(memberExpr.Property)
	}

	return false
}

// isSafeStrconvFunction checks whether a strconv function produces safe output.
//
// Takes property (ast_domain.Expression) which is the expression to check.
//
// Returns bool which is true if the function produces only safe characters.
func isSafeStrconvFunction(property ast_domain.Expression) bool {
	propIdent, ok := property.(*ast_domain.Identifier)
	if !ok {
		return false
	}

	safeStrconvFuncs := map[string]bool{
		"FormatInt":     true,
		"FormatUint":    true,
		"FormatFloat":   true,
		"FormatBool":    true,
		"Itoa":          true,
		"FormatComplex": true,
	}

	return safeStrconvFuncs[propIdent.Name]
}

// isSafePrimitiveIdentifier checks if an identifier, member, or index
// expression refers to a safe primitive type. Strings and runes are not
// considered safe because they may contain user input.
//
// Takes expression (ast_domain.Expression) which is the expression to check.
//
// Returns bool which is true if the expression is a safe primitive type.
func isSafePrimitiveIdentifier(expression ast_domain.Expression) bool {
	ann := getAnnotationFromExpression(expression)
	if ann == nil || ann.ResolvedType == nil {
		return false
	}

	if inspector_dto.StringabilityMethod(ann.Stringability) != inspector_dto.StringablePrimitive {
		return false
	}

	typeIdent, ok := ann.ResolvedType.TypeExpression.(*goast.Ident)
	if !ok {
		return false
	}

	return typeIdent.Name != "string" && typeIdent.Name != "rune"
}

// isSafeTemplateLiteral checks whether a template literal contains only safe
// embedded expressions.
//
// Takes e (*ast_domain.TemplateLiteral) which is the template literal to check.
//
// Returns bool which is true when all non-literal parts are safe.
func isSafeTemplateLiteral(e *ast_domain.TemplateLiteral) bool {
	for _, part := range e.Parts {
		if !part.IsLiteral && !isExpressionIntrinsicallySafe(part.Expression) {
			return false
		}
	}
	return true
}

// getSpecialisedHelperName returns a helper name for a specific type if one
// exists.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the resolved
// type details.
// Takes genericName (string) which is the default helper name to use.
// Takes specificHelpers (map[string]string) which maps type strings to helper
// names.
//
// Returns string which is the matching helper name if found, or the default
// name if no match exists.
func getSpecialisedHelperName(ann *ast_domain.GoGeneratorAnnotation, genericName string, specificHelpers map[string]string) string {
	if ann != nil && ann.ResolvedType != nil {
		typeString := goastutil.ASTToTypeString(ann.ResolvedType.TypeExpression, ann.ResolvedType.PackageAlias)
		if specificName, ok := specificHelpers[typeString]; ok {
			return specificName
		}
	}
	return genericName
}

// shouldResolveModulePath checks whether a dynamic attribute should be wrapped
// with module path resolution.
//
// Takes tagName (string) which is the HTML element tag name.
// Takes attributeName (string) which is the attribute name to check.
//
// Returns bool which is true for src attributes on piko:svg, piko:img, pml-img,
// and
// piko:video elements.
func shouldResolveModulePath(tagName, attributeName string) bool {
	if !strings.EqualFold(attributeName, "src") {
		return false
	}

	switch tagName {
	case "piko:svg", "piko:img", "pml-img", "piko:video":
		return true
	default:
		return false
	}
}
