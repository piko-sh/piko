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
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
)

// emitClassAttribute builds code for the class attribute using DirectWriter
// with *Bytes helpers for zero-allocation rendering.
//
// Static-only classes still use Attributes slice for simplicity. Dynamic
// classes use AttributeWriters via DirectWriter.
//
// Takes nodeVar (*goast.Ident) which is the node variable for AttributeWriters.
// Takes attributeSlice (goast.Expr) which is the slice for static-only classes.
// Takes node (*ast_domain.TemplateNode) which holds the class attribute data.
//
// Returns []goast.Stmt which holds the built statements, or nil if no class
// attribute exists.
// Returns []*ast_domain.Diagnostic which holds any diagnostics from parsing
// dynamic expressions.
func (ae *attributeEmitter) emitClassAttribute(nodeVar *goast.Ident, attributeSlice goast.Expr, node *ast_domain.TemplateNode) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	staticClass := extractStaticClass(node)
	dynamicExpr := extractDynamicClassExpression(node)

	if staticClass == "" && dynamicExpr == nil {
		return nil, nil
	}

	if dynamicExpr == nil {
		return ae.emitStaticOnlyClass(attributeSlice, staticClass), nil
	}

	if staticClass == "" {
		return ae.emitDynamicOnlyClassWriter(nodeVar, *dynamicExpr)
	}

	return ae.emitMergedClassWriter(nodeVar, staticClass, *dynamicExpr)
}

// emitStaticOnlyClass builds a static class attribute.
//
// Takes attributeSlice (goast.Expr) which is the slice to add the attribute to.
// Takes staticClass (string) which is the class value.
//
// Returns []goast.Stmt which contains the append statement for the attribute.
func (*attributeEmitter) emitStaticOnlyClass(attributeSlice goast.Expr, staticClass string) []goast.Stmt {
	attributeLiteral := &goast.CompositeLit{
		Type: cachedIdent(fmt.Sprintf(HTMLAttributeTypeFmt, runtimePackageName)),
		Elts: []goast.Expr{
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameName), Value: strLit(attributeNameClass)},
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameValue), Value: strLit(staticClass)},
		},
	}
	return []goast.Stmt{appendToSlice(attributeSlice, attributeLiteral)}
}

// emitDynamicOnlyClassWriter creates a class attribute using DirectWriter
// for rendering without memory allocation.
//
// Takes nodeVar (*goast.Ident) which is the node to append AttributeWriters to.
// Takes dynamicExpr (ast_domain.Expression) which is the dynamic expression.
//
// Returns []goast.Stmt which contains the statements to emit.
// Returns []*ast_domain.Diagnostic which contains any diagnostics found.
func (ae *attributeEmitter) emitDynamicOnlyClassWriter(nodeVar *goast.Ident, dynamicExpr ast_domain.Expression) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	if tl, ok := dynamicExpr.(*ast_domain.TemplateLiteral); ok {
		parts, prereqs, diagnostics := ae.expressionEmitter.emitTemplateLiteralParts(tl)
		if len(parts) > 0 {
			helperName := selectBuildClassBytesHelper(len(parts))
			return ae.buildClassWriterStmts(nodeVar, prereqs, helperName, parts...), diagnostics
		}
		return nil, diagnostics
	}

	goExpr, prereqs, diagnostics := ae.expressionEmitter.emit(dynamicExpr)
	ann := getAnnotationFromExpression(dynamicExpr)

	helperName := getSpecialisedHelperName(ann, "MergeClassesBytesArena", map[string]string{
		"string":                 "ClassesFromStringBytesArena",
		"[]string":               "ClassesFromSliceBytesArena",
		"map[string]bool":        "MergeClassesBytesArena",
		"map[string]interface{}": "MergeClassesBytesArena",
	})

	return ae.buildClassWriterStmts(nodeVar, prereqs, helperName, goExpr), diagnostics
}

// emitMergedClassWriter creates a class attribute that combines static and
// dynamic values using DirectWriter for zero-allocation rendering.
//
// Takes nodeVar (*goast.Ident) which is the node to append AttributeWriters to.
// Takes staticClass (string) which is the static class value to merge.
// Takes dynamicExpr (ast_domain.Expression) which is the dynamic expression.
//
// Returns []goast.Stmt which contains the statements to emit.
// Returns []*ast_domain.Diagnostic which contains any diagnostics produced.
func (ae *attributeEmitter) emitMergedClassWriter(nodeVar *goast.Ident, staticClass string, dynamicExpr ast_domain.Expression) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	if tl, ok := dynamicExpr.(*ast_domain.TemplateLiteral); ok {
		parts, prereqs, diagnostics := ae.expressionEmitter.emitTemplateLiteralParts(tl)
		allParts := make([]goast.Expr, 0, len(parts)+2)
		allParts = append(allParts, strLit(staticClass), strLit(" "))
		allParts = append(allParts, parts...)
		helperName := selectBuildClassBytesHelper(len(allParts))
		return ae.buildClassWriterStmts(nodeVar, prereqs, helperName, allParts...), diagnostics
	}

	goExpr, prereqs, diagnostics := ae.expressionEmitter.emit(dynamicExpr)

	return ae.buildClassWriterStmts(nodeVar, prereqs, "MergeClassesBytesArena", strLit(staticClass), goExpr), diagnostics
}

// buildClassWriterStmts builds DirectWriter code for a class attribute using
// AppendPooledBytes to avoid memory allocation when using *BytesArena helpers.
// The arena is passed as the first argument to eliminate sync.Pool churn.
//
// Generates:
// bufferPointerVar := pikoruntime.HelperNameArena(arena, arguments...)
//
//	if bufferPointerVar != nil {
//	    dwVar := arena.GetDirectWriter()
//	    dwVar.SetName("class")
//	    dwVar.AppendPooledBytes(bufferPointerVar)
//	    nodeVar.AttributeWriters = append(nodeVar.AttributeWriters, dwVar)
//	}
//
// Takes nodeVar (*goast.Ident) which is the node variable to attach writers to.
// Takes prereqs ([]goast.Stmt) which are statements to run before the helper.
// Takes helperName (string) which is the name of the *BytesArena helper function.
// Takes arguments (...goast.Expr) which are the arguments for the helper function.
//
// Returns []goast.Stmt which contains the generated statements.
func (ae *attributeEmitter) buildClassWriterStmts(nodeVar *goast.Ident, prereqs []goast.Stmt, helperName string, arguments ...goast.Expr) []goast.Stmt {
	bufferPointerVar := ae.emitter.nextTempName()
	bufferPointerIdent := cachedIdent(bufferPointerVar)

	statements := make([]goast.Stmt, 0, len(prereqs)+2)
	statements = append(statements, prereqs...)

	arenaArgs := make([]goast.Expr, 0, len(arguments)+1)
	arenaArgs = append(arenaArgs, cachedIdent(arenaVarName))
	arenaArgs = append(arenaArgs, arguments...) //nolint:gocritic // non-variadic + variadic append

	statements = append(statements,
		defineAndAssign(bufferPointerVar, &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(helperName)},
			Args: arenaArgs,
		}),
		ae.buildDirectWriterBlock(nodeVar, attributeNameClass, bufferPointerIdent, "AppendPooledBytes"),
	)

	return statements
}

// extractStaticClass retrieves the static class attribute value from a node.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to extract
// the class from.
//
// Returns string which is the class attribute value, or empty if not found.
func extractStaticClass(node *ast_domain.TemplateNode) string {
	staticClass, _ := node.GetAttribute(attributeNameClass)
	return staticClass
}

// extractDynamicClassExpression returns the dynamic class expression from a
// template node.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns *ast_domain.Expression which is the dynamic class expression, or nil
// if none is found.
func extractDynamicClassExpression(node *ast_domain.TemplateNode) *ast_domain.Expression {
	if node.DirClass != nil && node.DirClass.Expression != nil {
		return &node.DirClass.Expression
	}

	for i := range node.DynamicAttributes {
		if strings.EqualFold(node.DynamicAttributes[i].Name, attributeNameClass) {
			return &node.DynamicAttributes[i].Expression
		}
	}

	return nil
}

// selectBuildClassBytesHelper returns the fixed-arity BuildClassBytesArena
// function name for the given part count. Uses fixed-arity arena functions
// to avoid variadic slice heap escape and remove sync.Pool churn.
//
// Takes partCount (int) which is the number of string parts.
//
// Returns string which is the helper function name.
func selectBuildClassBytesHelper(partCount int) string {
	switch partCount {
	case 2:
		return "BuildClassBytes2Arena"
	case classPartsCount4:
		return "BuildClassBytes4Arena"
	case classPartsCount6:
		return "BuildClassBytes6Arena"
	case classPartsCount8:
		return "BuildClassBytes8Arena"
	default:
		return "BuildClassBytesVArena"
	}
}
