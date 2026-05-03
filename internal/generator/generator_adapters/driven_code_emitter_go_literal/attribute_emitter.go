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

const (
	// actionModifierName is the modifier name for server-side action calls
	// using the action.namespace.Name() syntax. Must match the
	// actionCallModifier value in annotator_domain/transform.go.
	actionModifierName = "action"

	// helperModifierName is the modifier name for helper actions in event
	// handlers.
	helperModifierName = "helper"

	// directWriterMethodSetName is the method name used to set the attribute name
	// on a direct writer.
	directWriterMethodSetName = "SetName"
)

// decomposeContext indicates how dynamic expressions should be handled during
// decomposition.
type decomposeContext uint8

const (
	// decomposeContextAttribute marks a standard HTML attribute context. Dynamic
	// strings are escaped to prevent cross-site scripting attacks.
	decomposeContextAttribute decomposeContext = iota

	// decomposeContextPKey indicates a p-key attribute context where dynamic
	// strings, floats, and unknown types are FNV-32 hashed. This produces bounded
	// output length (8 chars), avoids HTML escaping issues, and handles float
	// precision problems.
	decomposeContextPKey
)

// AttributeEmitter is the interface for emitting HTML attributes.
// It enables mocking and testing of attribute emission logic.
type AttributeEmitter interface {
	// emit generates Go AST statements for the given template node.
	//
	// Takes nodeVar (*goast.Ident) which identifies the variable representing the
	// current node.
	// Takes node (*ast_domain.TemplateNode) which is the template node to emit.
	//
	// Returns []goast.Stmt which contains the generated statements.
	// Returns []*ast_domain.Diagnostic which contains any issues found during
	// emission.
	emit(nodeVar *goast.Ident, node *ast_domain.TemplateNode) ([]goast.Stmt, []*ast_domain.Diagnostic)

	// emitDynamicAttributesOnly emits only the dynamic attributes that are sent
	// to AttributeWriters. This is used when static attributes are hoisted; it
	// skips Attributes emission but still emits to DirectWriters.
	//
	// Takes nodeVar (*goast.Ident) which is the variable for the template node.
	// Takes node (*ast_domain.TemplateNode) which is the node to process.
	//
	// Returns []goast.Stmt which contains the generated statements.
	// Returns []*ast_domain.Diagnostic which contains any warnings or errors.
	emitDynamicAttributesOnly(nodeVar *goast.Ident, node *ast_domain.TemplateNode) ([]goast.Stmt, []*ast_domain.Diagnostic)
}

// attributeEmitter generates the Attributes slice for a TemplateNode.
type attributeEmitter struct {
	// emitter holds the shared code generation state and helper methods.
	emitter *emitter

	// expressionEmitter converts template expressions to Go AST nodes.
	expressionEmitter ExpressionEmitter
}

var _ AttributeEmitter = (*attributeEmitter)(nil)

// emit generates all attribute statements for a template node. This is the
// main entry point for this emitter.
//
// Takes nodeVar (*goast.Ident) which is the variable that refers to the node.
// Takes node (*ast_domain.TemplateNode) which holds the template node data.
//
// Returns []goast.Stmt which contains the generated attribute statements.
// Returns []*ast_domain.Diagnostic which contains any issues found during
// generation.
func (ae *attributeEmitter) emit(nodeVar *goast.Ident, node *ast_domain.TemplateNode) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	estimatedCapacity := len(node.Attributes) + len(node.Binds) + AttributeCapacityBuffer
	statements := make([]goast.Stmt, 0, estimatedCapacity)
	allDiags := make([]*ast_domain.Diagnostic, 0, largeDiagnosticCapacity)
	attributeSliceExpression := &goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldAttributes)}

	staticStmts := ae.emitStaticAttributes(attributeSliceExpression, node)
	statements = append(statements, staticStmts...)

	dynStmts, dynDiags := ae.emitDynamicAttributes(nodeVar, attributeSliceExpression, node)
	statements = append(statements, dynStmts...)
	allDiags = append(allDiags, dynDiags...)

	classStmts, classDiags := ae.emitClassAttribute(nodeVar, attributeSliceExpression, node)
	statements = append(statements, classStmts...)
	allDiags = append(allDiags, classDiags...)

	styleStmts, styleDiags := ae.emitStyleAttribute(nodeVar, node)
	statements = append(statements, styleStmts...)
	allDiags = append(allDiags, styleDiags...)

	eventStmts, eventDiags := ae.emitEventHandlers(nodeVar, node)
	statements = append(statements, eventStmts...)
	allDiags = append(allDiags, eventDiags...)

	keyStmts, keyDiags := ae.emitKeyAttribute(nodeVar, node)
	statements = append(statements, keyStmts...)
	allDiags = append(allDiags, keyDiags...)

	refStmts, refDiags := ae.emitRefAttribute(nodeVar, node)
	statements = append(statements, refStmts...)
	allDiags = append(allDiags, refDiags...)

	return statements, allDiags
}

// emitStaticAttributes builds code for all fixed HTML attributes.
// It skips class and style attributes as these are handled separately.
//
// Takes attributeSliceExpression (goast.Expr) which is the slice to
// add attributes to.
// Takes node (*ast_domain.TemplateNode) which holds the attributes to write.
//
// Returns []goast.Stmt which holds the append statements for each attribute.
func (*attributeEmitter) emitStaticAttributes(attributeSliceExpression goast.Expr, node *ast_domain.TemplateNode) []goast.Stmt {
	statements := make([]goast.Stmt, 0, len(node.Attributes))

	for i := range node.Attributes {
		attr := &node.Attributes[i]
		if strings.EqualFold(attr.Name, attributeNameClass) || strings.EqualFold(attr.Name, attributeNameStyle) {
			continue
		}
		attributeLiteral := &goast.CompositeLit{
			Type: cachedIdent(fmt.Sprintf(HTMLAttributeTypeFmt, runtimePackageName)),
			Elts: []goast.Expr{
				&goast.KeyValueExpr{Key: cachedIdent(FieldNameName), Value: strLit(attr.Name)},
				&goast.KeyValueExpr{Key: cachedIdent(FieldNameValue), Value: strLit(html.EscapeString(attr.Value))},
			},
		}
		statements = append(statements, appendToSlice(attributeSliceExpression, attributeLiteral))
	}

	return statements
}

// emitDynamicAttributes creates code for all dynamic attributes (p-bind
// and :).
//
// Takes nodeVar (*goast.Ident) which is the variable for the node.
// Takes attributeSliceExpression (goast.Expr) which is the expression
// for the attribute slice.
// Takes node (*ast_domain.TemplateNode) which holds the template node data.
//
// Returns []goast.Stmt which holds the created statements.
// Returns []*ast_domain.Diagnostic which holds any problems found.
func (ae *attributeEmitter) emitDynamicAttributes(
	nodeVar *goast.Ident,
	attributeSliceExpression goast.Expr,
	node *ast_domain.TemplateNode,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	var statements []goast.Stmt
	var allDiags []*ast_domain.Diagnostic

	for name, directive := range node.Binds {
		dynAttr := &ast_domain.DynamicAttribute{
			Name:           name,
			Expression:     directive.Expression,
			GoAnnotations:  directive.GoAnnotations,
			RawExpression:  "",
			Location:       ast_domain.Location{},
			NameLocation:   ast_domain.Location{},
			AttributeRange: ast_domain.Range{},
		}
		attributeStatements, attributeDiagnostics := ae.emitSingleDynamicAttribute(nodeVar, attributeSliceExpression, dynAttr, node.TagName)
		statements = append(statements, attributeStatements...)
		allDiags = append(allDiags, attributeDiagnostics...)
	}

	for i := range node.DynamicAttributes {
		dynAttr := &node.DynamicAttributes[i]
		if strings.EqualFold(dynAttr.Name, attributeNameClass) || strings.EqualFold(dynAttr.Name, attributeNameStyle) {
			continue
		}
		if node.TagName == "piko:element" && strings.EqualFold(dynAttr.Name, "is") {
			continue
		}
		attributeStatements, attributeDiagnostics := ae.emitSingleDynamicAttribute(nodeVar, attributeSliceExpression, dynAttr, node.TagName)
		statements = append(statements, attributeStatements...)
		allDiags = append(allDiags, attributeDiagnostics...)
	}

	return statements, allDiags
}

// emitDynamicAttributesOnly emits only the dynamic attributes that go to
// AttributeWriters. This is used when static attributes are hoisted - we skip
// Attributes emission but still need DirectWriters for non-boolean dynamic
// attributes, class, and style.
//
// Takes nodeVar (*goast.Ident) which is the node variable.
// Takes node (*ast_domain.TemplateNode) which holds the template node data.
//
// Returns []goast.Stmt which holds the created statements.
// Returns []*ast_domain.Diagnostic which holds any issues found.
func (ae *attributeEmitter) emitDynamicAttributesOnly(
	nodeVar *goast.Ident,
	node *ast_domain.TemplateNode,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	statements := make([]goast.Stmt, 0, defaultEmissionStatementCapacity)
	allDiags := make([]*ast_domain.Diagnostic, 0, defaultDiagnosticCapacity)

	bindStmts, bindDiags := ae.emitBindAttributes(nodeVar, node)
	statements = append(statements, bindStmts...)
	allDiags = append(allDiags, bindDiags...)

	dynStmts, dynDiags := ae.emitNonClassStyleDynamicAttrs(nodeVar, node)
	statements = append(statements, dynStmts...)
	allDiags = append(allDiags, dynDiags...)

	classStyleStmts, classStyleDiags := ae.emitDynamicClassAndStyle(nodeVar, node)
	statements = append(statements, classStyleStmts...)
	allDiags = append(allDiags, classStyleDiags...)

	return statements, allDiags
}

// emitBindAttributes processes bind directives for a template node.
// It skips boolean types and delegates non-boolean attributes to DirectWriters.
//
// Takes nodeVar (*goast.Ident) which identifies the node variable in the AST.
// Takes node (*ast_domain.TemplateNode) which contains the bind directives.
//
// Returns []goast.Stmt which contains the generated statements.
// Returns []*ast_domain.Diagnostic which contains any errors found.
func (ae *attributeEmitter) emitBindAttributes(
	nodeVar *goast.Ident,
	node *ast_domain.TemplateNode,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	var statements []goast.Stmt
	var diagnostics []*ast_domain.Diagnostic
	for name, directive := range node.Binds {
		if isBindBooleanType(directive) {
			continue
		}
		dynAttr := &ast_domain.DynamicAttribute{
			Name:          name,
			Expression:    directive.Expression,
			GoAnnotations: directive.GoAnnotations,
		}
		attributeStatements, attributeDiagnostics := ae.emitNonBooleanDynamicAttribute(nodeVar, dynAttr, node.TagName)
		statements = append(statements, attributeStatements...)
		diagnostics = append(diagnostics, attributeDiagnostics...)
	}
	return statements, diagnostics
}

// emitNonClassStyleDynamicAttrs processes dynamic attributes, skipping class,
// style, and boolean attributes.
//
// Takes nodeVar (*goast.Ident) which identifies the node variable in the AST.
// Takes node (*ast_domain.TemplateNode) which provides the template node to
// process.
//
// Returns []goast.Stmt which contains the generated statements for the
// attributes.
// Returns []*ast_domain.Diagnostic which contains any issues found during
// processing.
func (ae *attributeEmitter) emitNonClassStyleDynamicAttrs(
	nodeVar *goast.Ident,
	node *ast_domain.TemplateNode,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	var statements []goast.Stmt
	var diagnostics []*ast_domain.Diagnostic
	for i := range node.DynamicAttributes {
		dynAttr := &node.DynamicAttributes[i]
		if shouldSkipDynAttrForWriter(dynAttr) {
			continue
		}
		if node.TagName == "piko:element" && strings.EqualFold(dynAttr.Name, "is") {
			continue
		}
		attributeStatements, attributeDiagnostics := ae.emitNonBooleanDynamicAttribute(nodeVar, dynAttr, node.TagName)
		statements = append(statements, attributeStatements...)
		diagnostics = append(diagnostics, attributeDiagnostics...)
	}
	return statements, diagnostics
}

// emitDynamicClassAndStyle emits class and style attributes if they have
// dynamic content.
//
// Takes nodeVar (*goast.Ident) which identifies the node variable in the AST.
// Takes node (*ast_domain.TemplateNode) which provides the template node to
// process.
//
// Returns []goast.Stmt which contains the generated statements for dynamic
// attributes.
// Returns []*ast_domain.Diagnostic which contains any diagnostics from
// processing.
func (ae *attributeEmitter) emitDynamicClassAndStyle(
	nodeVar *goast.Ident,
	node *ast_domain.TemplateNode,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	var statements []goast.Stmt
	var diagnostics []*ast_domain.Diagnostic

	if extractDynamicClassExpression(node) != nil {
		attributeSliceExpression := &goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldAttributes)}
		classStmts, classDiags := ae.emitClassAttribute(nodeVar, attributeSliceExpression, node)
		statements = append(statements, classStmts...)
		diagnostics = append(diagnostics, classDiags...)
	}

	if extractDynamicStyleExpression(node) != nil || node.DirShow != nil {
		styleStmts, styleDiags := ae.emitStyleAttribute(nodeVar, node)
		statements = append(statements, styleStmts...)
		diagnostics = append(diagnostics, styleDiags...)
	}

	return statements, diagnostics
}

// emitNonBooleanDynamicAttribute writes a single non-boolean dynamic attribute
// using DirectWriter. It handles special cases for asset source path resolution
// and standard dynamic attribute writers.
//
// Takes nodeVar (*goast.Ident) which is the node variable for the output.
// Takes dynAttr (*ast_domain.DynamicAttribute) which is the attribute to write.
// Takes tagName (string) which decides if @ path resolution is needed.
//
// Returns []goast.Stmt which are the statements to write the attribute.
// Returns []*ast_domain.Diagnostic which are any problems found.
func (ae *attributeEmitter) emitNonBooleanDynamicAttribute(
	nodeVar *goast.Ident,
	dynAttr *ast_domain.DynamicAttribute,
	tagName string,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	ann := dynAttr.GoAnnotations
	if ann == nil {
		ann = getAnnotationFromExpression(dynAttr.Expression)
	}

	if shouldResolveModulePath(tagName, dynAttr.Name) {
		return ae.emitAssetSrcAttribute(nodeVar, dynAttr)
	}

	return ae.emitDynamicAttributeWriter(nodeVar, dynAttr.Name, dynAttr.Expression, ann)
}

// emitKeyAttribute generates code for the p-key attribute if present.
// Uses HTMLAttribute for static keys to avoid pool overhead, and DirectWriter
// for dynamic keys to allow rendering without copying.
//
// Takes nodeVar (*goast.Ident) which identifies the variable for the node.
// Takes node (*ast_domain.TemplateNode) which provides the template node to
// check.
//
// Returns []goast.Stmt which contains the generated statements, or nil if no
// key attribute is present.
// Returns []*ast_domain.Diagnostic which contains any issues found.
func (ae *attributeEmitter) emitKeyAttribute(
	nodeVar *goast.Ident,
	node *ast_domain.TemplateNode,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	effectiveKey := getEffectiveKeyExpression(node)
	if effectiveKey == nil {
		return nil, nil
	}

	if sl, ok := effectiveKey.(*ast_domain.StringLiteral); ok {
		return ae.emitStaticKeyAttribute(nodeVar, sl.Value), nil
	}

	return ae.emitKeyWriterParts(nodeVar, effectiveKey)
}

// emitRefAttribute builds code for the p-ref attribute if present.
//
// Since p-ref is a raw string identifier (not an expression), the emission uses
// RawExpression directly. The value is validated by the annotator to be a valid
// JavaScript identifier.
//
// Takes nodeVar (*goast.Ident) which is the AST variable for the node.
// Takes node (*ast_domain.TemplateNode) which holds the p-ref directive.
//
// Returns []goast.Stmt which holds the generated attribute statements.
// Returns []*ast_domain.Diagnostic which is always nil as validation occurs
// in the annotator.
func (ae *attributeEmitter) emitRefAttribute(
	nodeVar *goast.Ident,
	node *ast_domain.TemplateNode,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	if node.DirRef == nil || node.DirRef.RawExpression == "" {
		return nil, nil
	}

	attributeSliceExpression := &goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldAttributes)}

	refAttrLiteral := &goast.CompositeLit{
		Type: cachedIdent(fmt.Sprintf(HTMLAttributeTypeFmt, runtimePackageName)),
		Elts: []goast.Expr{
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameName), Value: strLit(prefAttributeName)},
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameValue), Value: strLit(html.EscapeString(node.DirRef.RawExpression))},
		},
	}

	statements := []goast.Stmt{appendToSlice(attributeSliceExpression, refAttrLiteral)}

	componentHash := ae.emitter.config.HashedName
	if componentHash != "" {
		scopeAttrLiteral := &goast.CompositeLit{
			Type: cachedIdent(fmt.Sprintf(HTMLAttributeTypeFmt, runtimePackageName)),
			Elts: []goast.Expr{
				&goast.KeyValueExpr{Key: cachedIdent(FieldNameName), Value: strLit("data-pk-partial")},
				&goast.KeyValueExpr{Key: cachedIdent(FieldNameValue), Value: strLit(componentHash)},
			},
		}
		statements = append(statements, appendToSlice(attributeSliceExpression, scopeAttrLiteral))
	}

	return statements, nil
}

// emitSingleDynamicAttribute generates code for one dynamic attribute binding
// (e.g., `:title="expr"`).
//
// Uses DirectWriter to render without memory use and with smart HTML escaping.
// Boolean attributes get special handling and are only output when truthy.
// Asset element src attributes use module path resolution.
//
// Takes nodeVar (*goast.Ident) which is the variable pointing to the DOM node.
// Takes attributeSlice (goast.Expr) which is the slice to append attributes to.
// Takes dynAttr (*ast_domain.DynamicAttribute) which is the attribute to emit.
// Takes tagName (string) which determines if @ path resolution is needed.
//
// Returns []goast.Stmt which are the statements to emit the attribute.
// Returns []*ast_domain.Diagnostic which are any diagnostics found.
func (ae *attributeEmitter) emitSingleDynamicAttribute(
	nodeVar *goast.Ident,
	attributeSlice goast.Expr,
	dynAttr *ast_domain.DynamicAttribute,
	tagName string,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	ann := dynAttr.GoAnnotations
	if ann == nil {
		ann = getAnnotationFromExpression(dynAttr.Expression)
	}

	if ann != nil && ann.ResolvedType != nil && isBoolType(ann.ResolvedType.TypeExpression) {
		return ae.emitBooleanAttribute(attributeSlice, dynAttr)
	}

	if shouldResolveModulePath(tagName, dynAttr.Name) {
		return ae.emitAssetSrcAttribute(nodeVar, dynAttr)
	}

	return ae.emitDynamicAttributeWriter(nodeVar, dynAttr.Name, dynAttr.Expression, ann)
}

// emitBooleanAttribute generates code for boolean attributes that are added
// only when a condition is true. Boolean attributes appear in HTML when the
// value is true and are absent when false (e.g., <input disabled>).
//
// Takes attributeSlice (goast.Expr) which is the slice to append the attribute to.
// Takes dynAttr (*ast_domain.DynamicAttribute) which provides the attribute
// name and boolean expression.
//
// Returns []goast.Stmt which contains the setup statements and the conditional
// append.
// Returns []*ast_domain.Diagnostic which contains any issues found when
// processing the expression.
func (ae *attributeEmitter) emitBooleanAttribute(
	attributeSlice goast.Expr,
	dynAttr *ast_domain.DynamicAttribute,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	valGoExpr, prereqStmts, expressionDiagnostics := ae.expressionEmitter.emit(dynAttr.Expression)

	attributeLiteral := &goast.CompositeLit{
		Type: cachedIdent(fmt.Sprintf(HTMLAttributeTypeFmt, runtimePackageName)),
		Elts: []goast.Expr{
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameName), Value: strLit(dynAttr.Name)},
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameValue), Value: strLit("")},
		},
	}
	ifStmt := &goast.IfStmt{
		Cond: valGoExpr,
		Body: &goast.BlockStmt{List: []goast.Stmt{appendToSlice(attributeSlice, attributeLiteral)}},
	}
	return append(prereqStmts, ifStmt), expressionDiagnostics
}

// emitAssetSrcAttribute generates code for src attributes on asset elements
// (piko:svg, piko:img, pml-img).
//
// The generated code handles module path resolution for @/ alias support and
// uses DirectWriter via AttributeWriters for zero-allocation rendering with
// HTML escaping.
//
// Takes nodeVar (*goast.Ident) which represents the node variable.
// Takes dynAttr (*ast_domain.DynamicAttribute) which defines the attribute.
//
// Returns []goast.Stmt which contains the generated statements.
// Returns []*ast_domain.Diagnostic which contains any emission diagnostics.
func (ae *attributeEmitter) emitAssetSrcAttribute(
	nodeVar *goast.Ident,
	dynAttr *ast_domain.DynamicAttribute,
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
			Args: []goast.Expr{strLit(dynAttr.Name)},
		}},
	)

	valGoExpr, prereqs, expressionDiagnostics := ae.expressionEmitter.emit(dynAttr.Expression)
	statements = append(statements, prereqs...)
	diagnostics = append(diagnostics, expressionDiagnostics...)

	stringValExpr := ae.expressionEmitter.valueToString(valGoExpr, dynAttr.GoAnnotations)
	stringValExpr = ae.wrapWithModulePathResolver(stringValExpr)

	statements = append(statements,
		&goast.ExprStmt{X: &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: dwIdent, Sel: cachedIdent("AppendString")},
			Args: []goast.Expr{stringValExpr},
		}},
		appendToSlice(&goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldAttributeWriters)}, dwIdent),
	)

	return statements, diagnostics
}

// emitStaticKeyAttribute generates code for a static key attribute using a
// simple HTMLAttribute struct. This avoids pool overhead for keys that are
// compile-time constants by generating a direct append to node.Attributes.
//
// Takes nodeVar (*goast.Ident) which is the AST identifier for the node
// variable to append the attribute to.
// Takes keyValue (string) which is the static key value to embed in the
// generated code.
//
// Returns []goast.Stmt which contains the append statement for the attribute.
func (*attributeEmitter) emitStaticKeyAttribute(nodeVar *goast.Ident, keyValue string) []goast.Stmt {
	attributeSliceExpression := &goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldAttributes)}
	attributeLiteral := &goast.CompositeLit{
		Type: cachedIdent(fmt.Sprintf(HTMLAttributeTypeFmt, runtimePackageName)),
		Elts: []goast.Expr{
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameName), Value: strLit(pkeyAttributeName)},
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameValue), Value: strLit(keyValue)},
		},
	}
	return []goast.Stmt{appendToSlice(attributeSliceExpression, attributeLiteral)}
}

// buildDirectWriterBlock creates an if-statement that initialises a
// DirectWriter, sets its attribute name, appends a value from a buffer,
// and adds it to the node's AttributeWriters slice.
//
// Takes nodeVar (*goast.Ident) which identifies the node to receive the
// attribute writer.
// Takes attributeName (string) which specifies the name to assign to the
// DirectWriter.
// Takes bufferPointerIdent (*goast.Ident) which points to the buffer containing
// the value to append.
// Takes appendMethodName (string) which specifies the method to use for
// appending the buffer value.
//
// Returns *goast.IfStmt which wraps the DirectWriter setup in a nil check.
func (ae *attributeEmitter) buildDirectWriterBlock(
	nodeVar *goast.Ident,
	attributeName string,
	bufferPointerIdent *goast.Ident,
	appendMethodName string,
) *goast.IfStmt {
	dwVar := ae.emitter.nextTempName()
	dwIdent := cachedIdent(dwVar)

	body := []goast.Stmt{
		defineAndAssign(dwVar, &goast.CallExpr{
			Fun: &goast.SelectorExpr{X: cachedIdent(arenaVarName), Sel: cachedIdent("GetDirectWriter")},
		}),
		&goast.ExprStmt{X: &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: dwIdent, Sel: cachedIdent(directWriterMethodSetName)},
			Args: []goast.Expr{strLit(attributeName)},
		}},
		&goast.ExprStmt{X: &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: dwIdent, Sel: cachedIdent(appendMethodName)},
			Args: []goast.Expr{bufferPointerIdent},
		}},
		appendToSlice(&goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldAttributeWriters)}, dwIdent),
	}

	return &goast.IfStmt{
		Cond: &goast.BinaryExpr{
			X:  bufferPointerIdent,
			Op: token.NEQ,
			Y:  cachedIdent("nil"),
		},
		Body: &goast.BlockStmt{List: body},
	}
}

// newAttributeEmitter creates an emitter for attribute nodes.
//
// Takes emitter (*emitter) which provides the base emission functions.
// Takes expressionEmitter (ExpressionEmitter) which handles expression output.
//
// Returns *attributeEmitter which is ready to emit attribute nodes.
func newAttributeEmitter(emitter *emitter, expressionEmitter ExpressionEmitter) *attributeEmitter {
	return &attributeEmitter{
		emitter:           emitter,
		expressionEmitter: expressionEmitter,
	}
}

// isBindBooleanType checks if a bind directive resolves to a boolean type.
//
// Takes directive (*ast_domain.Directive) which is the directive to check.
//
// Returns bool which is true if the directive resolves to a boolean type.
func isBindBooleanType(directive *ast_domain.Directive) bool {
	ann := directive.GoAnnotations
	if ann == nil {
		ann = getAnnotationFromExpression(directive.Expression)
	}
	return ann != nil && ann.ResolvedType != nil && isBoolType(ann.ResolvedType.TypeExpression)
}

// shouldSkipDynAttrForWriter returns true if the dynamic attribute should not
// be emitted as a writer.
//
// Takes dynAttr (*ast_domain.DynamicAttribute) which is the attribute to check.
//
// Returns bool which is true when the attribute is a class or style attribute,
// or when it has a resolved boolean type annotation.
func shouldSkipDynAttrForWriter(dynAttr *ast_domain.DynamicAttribute) bool {
	if strings.EqualFold(dynAttr.Name, attributeNameClass) || strings.EqualFold(dynAttr.Name, attributeNameStyle) {
		return true
	}
	ann := dynAttr.GoAnnotations
	if ann == nil {
		ann = getAnnotationFromExpression(dynAttr.Expression)
	}
	return ann != nil && ann.ResolvedType != nil && isBoolType(ann.ResolvedType.TypeExpression)
}
