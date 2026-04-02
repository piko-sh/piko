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
	"html"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
)

// emitStyleAttribute creates statements for the style attribute.
//
// It handles static styles, dynamic style expressions, and p-show visibility.
// Returns nil values when no style-related attributes exist.
//
// Takes nodeVar (*goast.Ident) which identifies the node variable in the AST.
// Takes node (*ast_domain.TemplateNode) which provides the template node data.
//
// Returns []goast.Stmt which contains the generated statements for the style.
// Returns []*ast_domain.Diagnostic which contains any errors found.
func (ae *attributeEmitter) emitStyleAttribute(nodeVar *goast.Ident, node *ast_domain.TemplateNode) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	staticStyle := extractStaticStyle(node)
	dynamicExpr := extractDynamicStyleExpression(node)
	hasShow := node.DirShow != nil

	if staticStyle == "" && dynamicExpr == nil && !hasShow {
		return nil, nil
	}

	attributeSlice := &goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldAttributes)}

	if dynamicExpr == nil && !hasShow {
		return ae.emitStaticOnlyStyle(attributeSlice, staticStyle), nil
	}

	return ae.emitDynamicStyle(nodeVar, node, staticStyle, dynamicExpr)
}

// emitStaticOnlyStyle creates a static style attribute and appends it to a
// slice.
//
// Takes attributeSlice (goast.Expr) which is the slice to append the attribute to.
// Takes staticStyle (string) which is the CSS style value.
//
// Returns []goast.Stmt which contains the statement that appends the attribute.
func (*attributeEmitter) emitStaticOnlyStyle(attributeSlice goast.Expr, staticStyle string) []goast.Stmt {
	attributeLiteral := &goast.CompositeLit{
		Type: cachedIdent(fmt.Sprintf(HTMLAttributeTypeFmt, runtimePackageName)),
		Elts: []goast.Expr{
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameName), Value: strLit(attributeNameStyle)},
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameValue), Value: strLit(html.EscapeString(staticStyle))},
		},
	}
	return []goast.Stmt{appendToSlice(attributeSlice, attributeLiteral)}
}

// emitDynamicStyle handles dynamic styles with optional p-show directive.
//
// Uses DirectWriter for zero-allocation deferred HTML escaping. Checks
// node.DirShow directly instead of using a flag.
//
// Takes nodeVar (*goast.Ident) which identifies the target node variable.
// Takes node (*ast_domain.TemplateNode) which provides the template node.
// Takes staticStyle (string) which contains any static style content.
// Takes dynamicExpr (*ast_domain.Expression) which is the dynamic style.
//
// Returns []goast.Stmt which contains the generated Go statements.
// Returns []*ast_domain.Diagnostic which holds any diagnostics found.
func (ae *attributeEmitter) emitDynamicStyle(
	nodeVar *goast.Ident,
	node *ast_domain.TemplateNode,
	staticStyle string,
	dynamicExpr *ast_domain.Expression,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	styleVar, allStmts, allDiags := ae.buildStyleExpression(staticStyle, dynamicExpr)

	if node.DirShow != nil {
		styleVar, allStmts, allDiags = ae.applyShowDirective(node, styleVar, allStmts, allDiags)
	}

	allStmts = append(allStmts, ae.buildStyleWithDirectWriter(nodeVar, styleVar)...)

	return allStmts, allDiags
}

// buildStyleExpression builds a style expression from fixed and dynamic
// sources. Uses *BytesArena helpers for zero-allocation rendering via arena
// allocation.
//
// Takes staticStyle (string) which is the fixed CSS style string.
// Takes dynamicExpr (*ast_domain.Expression) which is a dynamic style value to
// process.
//
// Returns goast.Expr which is the final combined style expression as a *[]byte.
// Returns []goast.Stmt which contains any setup statements needed.
// Returns []*ast_domain.Diagnostic which contains any warnings or errors found.
func (ae *attributeEmitter) buildStyleExpression(
	staticStyle string,
	dynamicExpr *ast_domain.Expression,
) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	var allStmts []goast.Stmt
	var allDiags []*ast_domain.Diagnostic
	var sourceExprs []goast.Expr

	if staticStyle == "" && dynamicExpr != nil {
		if template, ok := (*dynamicExpr).(*ast_domain.TemplateLiteral); ok {
			if parts, prereqs, diagnostics := ae.expressionEmitter.emitTemplateLiteralParts(template); parts != nil {
				helperName := selectBuildStyleBytesHelper(len(parts))
				return callHelperArena(helperName, parts...), prereqs, diagnostics
			}
			return cachedIdent("nil"), allStmts, allDiags
		}
	}

	if staticStyle != "" {
		sourceExprs = append(sourceExprs, strLit(staticStyle))
	}

	if dynamicExpr != nil {
		goExpr, prereqs, diagnostics := ae.expressionEmitter.emit(*dynamicExpr)
		allStmts = append(allStmts, prereqs...)
		allDiags = append(allDiags, diagnostics...)
		sourceExprs = append(sourceExprs, goExpr)
	}

	var finalStyleVar goast.Expr
	if staticStyle == "" && dynamicExpr != nil {
		ann := getAnnotationFromExpression(*dynamicExpr)
		helperName := getSpecialisedHelperName(ann, "MergeStylesBytesArena", map[string]string{
			"string":            "StylesFromStringBytesArena",
			"map[string]string": "StylesFromStringMapBytesArena",
		})
		finalStyleVar = callHelperArena(helperName, sourceExprs[0])
	} else {
		finalStyleVar = callHelperArena("MergeStylesBytesArena", sourceExprs...)
	}

	return finalStyleVar, allStmts, allDiags
}

// applyShowDirective applies the p-show directive logic to hide elements when
// the condition is false. Uses AppendHiddenToStyleBytes for zero-allocation.
//
// Takes node (*ast_domain.TemplateNode) which contains the p-show directive.
// Takes styleVar (goast.Expr) which is the current style expression (*[]byte).
// Takes allStmts ([]goast.Stmt) which accumulates generated statements.
// Takes allDiags ([]*ast_domain.Diagnostic) which accumulates diagnostics.
//
// Returns goast.Expr which is the updated style variable (*[]byte).
// Returns []goast.Stmt which is the updated list of statements.
// Returns []*ast_domain.Diagnostic which is the updated list of diagnostics.
func (ae *attributeEmitter) applyShowDirective(
	node *ast_domain.TemplateNode,
	styleVar goast.Expr,
	allStmts []goast.Stmt,
	allDiags []*ast_domain.Diagnostic,
) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	tempStyleVar := cachedIdent(ae.emitter.nextTempName())
	allStmts = append(allStmts, defineAndAssign(tempStyleVar.Name, styleVar))

	condGoExpr, condPrereqs, condDiags := ae.expressionEmitter.emit(node.DirShow.Expression)
	allStmts = append(allStmts, condPrereqs...)
	allDiags = append(allDiags, condDiags...)
	condGoExpr = wrapInTruthinessCallIfNeeded(condGoExpr, node.DirShow.Expression)

	ifStmt := &goast.IfStmt{
		Cond: &goast.UnaryExpr{Op: token.NOT, X: condGoExpr},
		Body: &goast.BlockStmt{List: []goast.Stmt{
			assignExpression(tempStyleVar.Name, callHelper("AppendHiddenToStyleBytes", tempStyleVar)),
		}},
	}
	allStmts = append(allStmts, ifStmt)

	return tempStyleVar, allStmts, allDiags
}

// buildStyleWithDirectWriter builds statements that output a style attribute
// using a direct writer.
//
// Generated code pattern:
// bufferPointerVar := styleExpr  // *[]byte from *Bytes helper
//
//	if bufferPointerVar != nil {
//	    dwVar := pikoruntime.GetDirectWriter()
//	    dwVar.SetName("style")
//	    dwVar.AppendEscapePooledBytes(bufferPointerVar)
//	    nodeVar.AttributeWriters = append(nodeVar.AttributeWriters, dwVar)
//	}
//
// Takes nodeVar (*goast.Ident) which is the node to attach the style to.
// Takes styleVar (goast.Expr) which is the style expression (*[]byte) to
// output.
//
// Returns []goast.Stmt which contains the built AST statements.
func (ae *attributeEmitter) buildStyleWithDirectWriter(nodeVar *goast.Ident, styleVar goast.Expr) []goast.Stmt {
	bufferPointerVar := ae.emitter.nextTempName()
	bufferPointerIdent := cachedIdent(bufferPointerVar)

	return []goast.Stmt{
		defineAndAssign(bufferPointerVar, styleVar),
		ae.buildDirectWriterBlock(nodeVar, attributeNameStyle, bufferPointerIdent, "AppendEscapePooledBytes"),
	}
}

// extractStaticStyle retrieves the static style attribute value from a node.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to extract
// from.
//
// Returns string which is the style attribute value, or empty if not found.
func extractStaticStyle(node *ast_domain.TemplateNode) string {
	staticStyle, _ := node.GetAttribute(attributeNameStyle)
	return staticStyle
}

// extractDynamicStyleExpression returns the dynamic style expression from a
// template node.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns *ast_domain.Expression which is the style expression, or nil if the
// node has no dynamic style.
func extractDynamicStyleExpression(node *ast_domain.TemplateNode) *ast_domain.Expression {
	if node.DirStyle != nil && node.DirStyle.Expression != nil {
		return &node.DirStyle.Expression
	}

	for i := range node.DynamicAttributes {
		if strings.EqualFold(node.DynamicAttributes[i].Name, attributeNameStyle) {
			return &node.DynamicAttributes[i].Expression
		}
	}

	return nil
}

// selectBuildStyleBytesHelper returns the appropriate fixed-arity
// BuildStyleStringBytesArena function name based on the number of parts. Uses
// fixed-arity arena functions to avoid variadic slice heap escape and eliminate
// sync.Pool allocations.
//
// Takes partCount (int) which is the number of string parts.
//
// Returns string which is the helper function name.
func selectBuildStyleBytesHelper(partCount int) string {
	switch partCount {
	case 2:
		return "BuildStyleStringBytes2Arena"
	case stylePartsCount3:
		return "BuildStyleStringBytes3Arena"
	case stylePartsCount4:
		return "BuildStyleStringBytes4Arena"
	default:
		return "BuildStyleStringBytesVArena"
	}
}
