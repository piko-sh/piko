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
	"context"
	"fmt"
	goast "go/ast"
	"go/token"
	"html"
	"slices"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_domain"
)

const (
	// fieldNodeType is the field name that stores the node type in the AST struct.
	fieldNodeType = "NodeType"

	// fieldTagName is the field name for a node's tag name in the AST.
	fieldTagName = "TagName"

	// fieldTextContent is the AST field name for a node's text content.
	fieldTextContent = "TextContent"

	// fieldTextContentWriter is the struct field name for the
	// text content writer.
	fieldTextContentWriter = "TextContentWriter"

	// fieldInnerHTML is the field name for setting a node's
	// inner HTML content.
	fieldInnerHTML = "InnerHTML"

	// fieldAttributes is the field name for accessing node attributes.
	fieldAttributes = "Attributes"

	// fieldAttributeWriters is the field name for the attribute
	// writers slice.
	fieldAttributeWriters = "AttributeWriters"

	// fieldChildren is the field name for accessing child
	// elements of a node.
	fieldChildren = "Children"

	// attributeNameClass is the HTML attribute name for CSS classes.
	attributeNameClass = "class"

	// attributeNameStyle is the HTML attribute name for inline styles.
	attributeNameStyle = "style"

	// fieldDirModel is the field name for the model directive.
	fieldDirModel = "DirModel"

	// fieldDirScaffold is the field name for accessing the
	// scaffold directive.
	fieldDirScaffold = "DirScaffold"

	// runtimePackageName is the package name used when building AST
	// references to the generated runtime code.
	runtimePackageName = "pikoruntime"

	// arenaVarName is the variable name for the RenderArena used for pooled
	// allocations.
	arenaVarName = "arena"

	// facadePackageName is the package name for the Piko facade in generated code.
	facadePackageName = "piko"

	// pkeyAttributeName is the HTML attribute name for Piko component keys.
	pkeyAttributeName = "p-key"

	// prefAttributeName is the HTML attribute name for preload
	// reference hints.
	prefAttributeName = "p-ref"

	// partialAttrName is the HTML attribute name for partial
	// scope identification.
	partialAttrName = "partial"

	// maxCollectionInitStmts is the pre-allocation capacity for
	// collection initialiser statements (Attributes,
	// AttributeWriters, Children).
	maxCollectionInitStmts = 3
)

// NodeEmitter defines how to convert template nodes into code output.
// It allows for mocking and testing of the node output logic.
type NodeEmitter interface {
	// emit generates output for the given template node.
	//
	// Takes node (*ast_domain.TemplateNode) which is the
	// template node to process.
	// Takes partialScopeID (string) which is the current
	// partial's HashedName for CSS scoping.
	//
	// Returns string which is the generated output text.
	// Returns []goast.Stmt which contains any Go statements
	// produced.
	// Returns []*ast_domain.Diagnostic which reports any
	// issues found during emission.
	emit(ctx context.Context, node *ast_domain.TemplateNode, partialScopeID string) (string, []goast.Stmt, []*ast_domain.Diagnostic)
}

// nodeEmitter generates Go AST statements to build a single dynamic
// ast_domain.TemplateNode at runtime. It implements NodeEmitter, delegating
// to attributeEmitter for attributes and astBuilder for child nodes.
type nodeEmitter struct {
	// emitter holds shared state and helper methods for code generation.
	emitter *emitter

	// expressionEmitter converts template expressions into Go AST
	// expressions.
	expressionEmitter ExpressionEmitter

	// attributeEmitter creates attribute code for template
	// nodes.
	attributeEmitter AttributeEmitter

	// astBuilder provides access to the AST builder for
	// emitting child nodes. Uses an interface to break a
	// circular dependency between types.
	astBuilder AstBuilder
}

var _ NodeEmitter = (*nodeEmitter)(nil)

// getPartialScopeID returns the HashedName of the current component for CSS
// scoping. Works for both pages and partials.
//
// Returns string which is the hashed name, or empty if the main component
// cannot be found.
func (ne *nodeEmitter) getPartialScopeID() string {
	mainComp, err := generator_domain.GetMainComponent(ne.emitter.AnnotationResult)
	if err != nil {
		return ""
	}
	return mainComp.HashedName
}

// getChildScopeID determines the CSS scope ID for a node's children.
//
// If the node is a partial invocation (has PartialInfo), children get a
// combined scope ID (child + parent) to enable cross-partial CSS styling. The
// child scope comes first for CSS specificity. Otherwise, children inherit the
// parent's scope ID.
//
// Takes node (*ast_domain.TemplateNode) which is the current node being
// emitted.
// Takes parentScopeID (string) which is the scope ID from the
// parent context.
//
// Returns string which is the scope ID to use for this node's
// children.
func (ne *nodeEmitter) getChildScopeID(node *ast_domain.TemplateNode, parentScopeID string) string {
	if node.GoAnnotations == nil || node.GoAnnotations.PartialInfo == nil {
		return parentScopeID
	}

	pInfo := node.GoAnnotations.PartialInfo
	partialVC, ok := ne.emitter.AnnotationResult.VirtualModule.ComponentsByHash[pInfo.PartialPackageName]
	if !ok {
		return parentScopeID
	}

	childScope := partialVC.HashedName
	if parentScopeID != "" && parentScopeID != childScope && !containsScope(parentScopeID, childScope) {
		return childScope + " " + parentScopeID
	}
	return childScope
}

// emit generates Go code for a template node and is the main entry point for
// this emitter.
//
// Takes node (*ast_domain.TemplateNode) which specifies the template node to
// process.
// Takes partialScopeID (string) which is the current partial's HashedName for
// CSS scoping. If non-empty, a `partial` attribute will be emitted.
//
// Returns string which is the temporary variable name for the
// emitted node.
// Returns []goast.Stmt which contains the generated Go statements.
// Returns []*ast_domain.Diagnostic which contains any
// diagnostics found.
func (ne *nodeEmitter) emit(ctx context.Context, node *ast_domain.TemplateNode, partialScopeID string) (string, []goast.Stmt, []*ast_domain.Diagnostic) {
	statements := make([]goast.Stmt, 0, StatementSliceCapacity)
	var allDiags []*ast_domain.Diagnostic
	tempVarName := ne.emitter.nextTempName()
	tempVarIdent := cachedIdent(tempVarName)

	statements = append(statements,
		ne.emitter.sourceMappingStmt(node),
		defineAndAssign(tempVarName, &goast.CallExpr{
			Fun: &goast.SelectorExpr{X: cachedIdent(arenaVarName), Sel: cachedIdent("GetNode")},
		}))

	basicStmts, basicDiags := ne.emitBasicProperties(tempVarIdent, node)
	statements = append(statements, basicStmts...)
	allDiags = append(allDiags, basicDiags...)

	var hasContentDirective bool
	var loopInfoByIndex map[int]*LoopIterableInfo

	if node.NodeType == ast_domain.NodeElement || node.NodeType == ast_domain.NodeFragment {
		extractStmts, loopInfo, extractDiags := ne.extractLoopIterables(node.Children)
		statements = append(statements, extractStmts...)
		allDiags = append(allDiags, extractDiags...)
		loopInfoByIndex = loopInfo

		elemStmts, elemDiags, hasContent := ne.emitElementSpecificCode(tempVarIdent, node, loopInfoByIndex, partialScopeID)
		statements = append(statements, elemStmts...)
		allDiags = append(allDiags, elemDiags...)
		hasContentDirective = hasContent
	}

	if !hasContentDirective && node.NodeType != ast_domain.NodeText {
		childScopeID := ne.getChildScopeID(node, partialScopeID)
		childStmts, childDiags := ne.emitChildren(ctx, tempVarIdent, node, loopInfoByIndex, childScopeID)
		statements = append(statements, childStmts...)
		allDiags = append(allDiags, childDiags...)
	}

	return tempVarName, statements, allDiags
}

// emitElementSpecificCode handles all element and fragment-specific code
// generation.
//
// Takes tempVarIdent (*goast.Ident) which is the identifier for the temporary
// variable holding the element.
// Takes node (*ast_domain.TemplateNode) which is the template node to generate
// code for.
// Takes loopInfoByIndex (map[int]*LoopIterableInfo) which contains pre-extracted
// loop iterable info for p-for children, enabling accurate child slice
// capacity calculation.
// Takes partialScopeID (string) which is the current partial's HashedName for
// CSS scoping.
//
// Returns []goast.Stmt which contains the generated statements.
// Returns []*ast_domain.Diagnostic which contains any
// diagnostics encountered.
// Returns bool which indicates whether the node has a content
// directive.
func (ne *nodeEmitter) emitElementSpecificCode(
	tempVarIdent *goast.Ident,
	node *ast_domain.TemplateNode,
	loopInfoByIndex map[int]*LoopIterableInfo,
	partialScopeID string,
) ([]goast.Stmt, []*ast_domain.Diagnostic, bool) {
	statements := make([]goast.Stmt, 0, StatementSliceCapacity)
	var allDiags []*ast_domain.Diagnostic

	initStmts, usedStaticAttrs := ne.emitCollectionInitialisers(tempVarIdent, node, loopInfoByIndex, partialScopeID)
	statements = append(statements, initStmts...)

	if usedStaticAttrs {
		dynAttrStmts, dynAttrDiags := ne.attributeEmitter.emitDynamicAttributesOnly(tempVarIdent, node)
		statements = append(statements, dynAttrStmts...)
		allDiags = append(allDiags, dynAttrDiags...)
	} else {
		partialStmts := ne.emitPartialInfoAttributes(tempVarIdent, node)
		statements = append(statements, partialStmts...)

		effectiveScopeID := getEffectivePartialScopeID(node, partialScopeID, ne.getPartialScopeID())
		if len(partialStmts) == 0 && effectiveScopeID != "" && node.NodeType == ast_domain.NodeElement && !nodeHasPartialAttribute(node) {
			scopeStmt := ne.emitPartialScopeAttribute(tempVarIdent, effectiveScopeID)
			if scopeStmt != nil {
				statements = append(statements, scopeStmt)
			}
		}

		attributeStatements, attributeDiagnostics := ne.attributeEmitter.emit(tempVarIdent, node)
		statements = append(statements, attributeStatements...)
		allDiags = append(allDiags, attributeDiagnostics...)

		if node.TagName == "piko:img" && node.GoAnnotations != nil && len(node.GoAnnotations.Srcset) > 0 {
			srcsetStmts := ne.emitSrcsetAttributes(tempVarIdent, node)
			statements = append(statements, srcsetStmts...)
		}
	}

	partialPropsStmts := ne.emitPartialPropsAttribute(tempVarIdent, node)
	statements = append(statements, partialPropsStmts...)

	miscStmts, miscDiags := ne.emitMiscDirectives(tempVarIdent, node)
	statements = append(statements, miscStmts...)
	allDiags = append(allDiags, miscDiags...)

	hasContentDirective, contentStmts, contentDiags := ne.emitContentDirectives(tempVarIdent, node)
	statements = append(statements, contentStmts...)
	allDiags = append(allDiags, contentDiags...)

	return statements, allDiags, hasContentDirective
}

// emitPartialInfoAttributes creates metadata attributes for public partials.
// Skips if partial attributes already exist, such as those added by
// addPartialMetadataToNode for root nodes that gather both outer and inner
// partial IDs.
//
// Takes tempVarIdent (*goast.Ident) which is the temporary variable holding
// the node.
// Takes node (*ast_domain.TemplateNode) which is the template node to add
// attributes to.
//
// Returns []goast.Stmt which contains the attribute assignment statements, or
// nil if no attributes are needed.
func (ne *nodeEmitter) emitPartialInfoAttributes(tempVarIdent *goast.Ident, node *ast_domain.TemplateNode) []goast.Stmt {
	if node.GoAnnotations == nil || node.GoAnnotations.PartialInfo == nil {
		return nil
	}

	for i := range node.Attributes {
		if node.Attributes[i].Name == partialAttrName {
			return nil
		}
	}

	pInfo := node.GoAnnotations.PartialInfo
	partialVC, ok := ne.emitter.AnnotationResult.VirtualModule.ComponentsByHash[pInfo.PartialPackageName]
	if !ok {
		return nil
	}

	attributeSliceExpression := &goast.SelectorExpr{X: tempVarIdent, Sel: cachedIdent(fieldAttributes)}
	statements := make([]goast.Stmt, 0, PartialAttributeCapacity)

	partialAttr := &goast.CompositeLit{
		Type: cachedIdent(fmt.Sprintf(HTMLAttributeTypeFmt, runtimePackageName)),
		Elts: []goast.Expr{
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameName), Value: strLit(partialAttrName)},
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameValue), Value: strLit(partialVC.HashedName)},
		},
	}
	statements = append(statements, appendToSlice(attributeSliceExpression, partialAttr))

	partialNameAttr := &goast.CompositeLit{
		Type: cachedIdent(fmt.Sprintf(HTMLAttributeTypeFmt, runtimePackageName)),
		Elts: []goast.Expr{
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameName), Value: strLit("partial_name")},
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameValue), Value: strLit(partialVC.PartialName)},
		},
	}
	statements = append(statements, appendToSlice(attributeSliceExpression, partialNameAttr))

	if partialVC.IsPublic {
		partialSrcAttr := &goast.CompositeLit{
			Type: cachedIdent(fmt.Sprintf(HTMLAttributeTypeFmt, runtimePackageName)),
			Elts: []goast.Expr{
				&goast.KeyValueExpr{Key: cachedIdent(FieldNameName), Value: strLit("partial_src")},
				&goast.KeyValueExpr{Key: cachedIdent(FieldNameValue), Value: strLit(partialVC.PartialSrc)},
			},
		}
		statements = append(statements, appendToSlice(attributeSliceExpression, partialSrcAttr))
	}

	return statements
}

// emitPartialPropsAttribute creates a "partial_props" attribute for public
// partials that have query-bound primitive props. The attribute value is
// computed at render time by calling pikoruntime.BuildPartialPropsQuery with
// the resolved prop values converted to strings.
//
// This method runs after both the static and dynamic attribute paths so that
// the dynamic partial_props value is always emitted regardless of which path
// the node took.
//
// Takes tempVarIdent (*goast.Ident) which is the temporary variable holding
// the node.
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns []goast.Stmt which contains the attribute append statement, or nil
// when the partial has no query-bound primitive props.
func (ne *nodeEmitter) emitPartialPropsAttribute(tempVarIdent *goast.Ident, node *ast_domain.TemplateNode) []goast.Stmt {
	if node.GoAnnotations == nil || node.GoAnnotations.PartialInfo == nil {
		return nil
	}

	pInfo := node.GoAnnotations.PartialInfo
	partialVC, ok := ne.emitter.AnnotationResult.VirtualModule.ComponentsByHash[pInfo.PartialPackageName]
	if !ok || !partialVC.IsPublic {
		return nil
	}

	queryProps := extractPrimitiveQueryPropsFromComponent(partialVC)
	if len(queryProps) == 0 {
		return nil
	}

	propsVarName := "props_" + pInfo.InvocationKey
	pairArguments := make([]goast.Expr, 0, len(queryProps)*2)
	for _, queryProp := range queryProps {
		fieldAccess := &goast.SelectorExpr{
			X:   cachedIdent(propsVarName),
			Sel: cachedIdent(queryProp.GoFieldName),
		}
		valueExpression := buildPropToStringExpr(fieldAccess, getBaseTypeName(queryProp.TypeExpr))
		pairArguments = append(pairArguments, strLit(queryProp.QueryParamName), valueExpression)
	}

	buildQueryCall := &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("BuildPartialPropsQuery")},
		Args: pairArguments,
	}

	attributeSliceExpression := &goast.SelectorExpr{X: tempVarIdent, Sel: cachedIdent(fieldAttributes)}

	partialPropsAttr := &goast.CompositeLit{
		Type: cachedIdent(fmt.Sprintf(HTMLAttributeTypeFmt, runtimePackageName)),
		Elts: []goast.Expr{
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameName), Value: strLit("partial_props")},
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameValue), Value: buildQueryCall},
		},
	}

	return []goast.Stmt{appendToSlice(attributeSliceExpression, partialPropsAttr)}
}

// emitPartialScopeAttribute creates a partial scope attribute for CSS scoping
// on non-root elements. It adds only the scope ID to elements that did not
// receive partial attributes from emitPartialInfoAttributes.
//
// Takes tempVarIdent (*goast.Ident) which is the temporary variable that holds
// the node.
// Takes partialScopeID (string) which is the HashedName of the current partial.
//
// Returns goast.Stmt which is the statement to append the partial attribute,
// or nil if no scope ID is given.
func (*nodeEmitter) emitPartialScopeAttribute(tempVarIdent *goast.Ident, partialScopeID string) goast.Stmt {
	if partialScopeID == "" {
		return nil
	}

	attributeSliceExpression := &goast.SelectorExpr{X: tempVarIdent, Sel: cachedIdent(fieldAttributes)}

	partialAttr := &goast.CompositeLit{
		Type: cachedIdent(fmt.Sprintf(HTMLAttributeTypeFmt, runtimePackageName)),
		Elts: []goast.Expr{
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameName), Value: strLit(partialAttrName)},
			&goast.KeyValueExpr{Key: cachedIdent(FieldNameValue), Value: strLit(partialScopeID)},
		},
	}

	return appendToSlice(attributeSliceExpression, partialAttr)
}

// emitChildren creates Go statements to emit all child nodes.
//
// Takes tempVarIdent (*goast.Ident) which is the temporary variable for the
// parent node.
// Takes node (*ast_domain.TemplateNode) which is the parent node whose
// children will be emitted.
// Takes loopInfoByIndex (map[int]*LoopIterableInfo) which holds loop data for
// p-for children. This is used by the node emission context in for_emitter.
// Takes partialScopeID (string) which is the current partial's hashed name
// for CSS scoping.
//
// Returns []goast.Stmt which holds the generated statements for all children.
// Returns []*ast_domain.Diagnostic which holds any problems found during
// emission.
func (ne *nodeEmitter) emitChildren(
	ctx context.Context,
	tempVarIdent *goast.Ident,
	node *ast_domain.TemplateNode,
	loopInfoByIndex map[int]*LoopIterableInfo,
	partialScopeID string,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	statements := make([]goast.Stmt, 0, len(node.Children)*10)
	var allDiags []*ast_domain.Diagnostic
	childSliceExpr := &goast.SelectorExpr{X: tempVarIdent, Sel: cachedIdent(fieldChildren)}

	i := 0
	for i < len(node.Children) {
		child := node.Children[i]
		if ctx.Err() != nil {
			diagnostic := ast_domain.NewDiagnostic(ast_domain.Warning, "Code generation cancelled.", "", child.Location, "")
			allDiags = append(allDiags, diagnostic)
			return statements, allDiags
		}

		childScopeID := getEffectivePartialScopeID(child, partialScopeID, ne.getPartialScopeID())
		childCtx := newNodeEmissionContext(ctx, nodeEmissionParams{
			Node:                  child,
			ParentSliceExpression: childSliceExpr,
			Index:                 i,
			Siblings:              node.Children,
			IsRootNode:            false,
			PartialScopeID:        childScopeID,
			MainComponentScope:    ne.getPartialScopeID(),
		})

		if loopInfoByIndex != nil {
			if loopInfo, ok := loopInfoByIndex[i]; ok {
				childCtx.loopIterInfo = loopInfo
			}
		}

		childStmts, nodesConsumed, childDiags := ne.astBuilder.emitNode(childCtx)
		statements = append(statements, childStmts...)
		allDiags = append(allDiags, childDiags...)
		i += nodesConsumed
	}

	return statements, allDiags
}

// emitBasicProperties generates code to set simple fields like NodeType and
// TagName, and handles the logic for simple TextContent or complex RichText.
//
// Takes nodeVar (*goast.Ident) which identifies the node variable in the AST.
// Takes node (*ast_domain.TemplateNode) which provides the template node data.
//
// Returns []goast.Stmt which contains the generated assignment statements.
// Returns []*ast_domain.Diagnostic which contains any diagnostics from text
// content emission.
func (ne *nodeEmitter) emitBasicProperties(nodeVar *goast.Ident, node *ast_domain.TemplateNode) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	statements := []goast.Stmt{emitNodeType(nodeVar, node)}

	if node.TagName == "piko:element" {
		tagStmts, tagDiags := ne.emitPikoElementTagName(nodeVar, node)
		statements = append(statements, tagStmts...)
		textStmts, textDiags := ne.emitTextContent(nodeVar, node)
		statements = append(statements, textStmts...)
		return statements, append(tagDiags, textDiags...)
	}

	if node.TagName != "" {
		statements = append(statements, emitTagName(nodeVar, node.TagName))
	}

	textStmts, textDiags := ne.emitTextContent(nodeVar, node)
	statements = append(statements, textStmts...)

	return statements, textDiags
}

// emitPikoElementTagName compiles the dynamic :is expression on a
// <piko:element> node and emits a validated TagName assignment with runtime
// diagnostics for invalid targets.
//
// Takes nodeVar (*goast.Ident) which is the node variable to assign to.
// Takes node (*ast_domain.TemplateNode) which is the piko:element node.
//
// Returns []goast.Stmt which contains the generated statements.
// Returns []*ast_domain.Diagnostic which contains any compile-time issues.
func (ne *nodeEmitter) emitPikoElementTagName(nodeVar *goast.Ident, node *ast_domain.TemplateNode) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	var isAttr *ast_domain.DynamicAttribute
	for i := range node.DynamicAttributes {
		if strings.EqualFold(node.DynamicAttributes[i].Name, "is") {
			isAttr = &node.DynamicAttributes[i]
			break
		}
	}
	if isAttr == nil {
		return nil, nil
	}

	valExpr, prereqStmts, expressionDiagnostics := ne.expressionEmitter.emit(isAttr.Expression)

	ann := isAttr.GoAnnotations
	if ann == nil {
		ann = getAnnotationFromExpression(isAttr.Expression)
	}

	stringExpr := &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent("fmt"), Sel: cachedIdent("Sprint")},
		Args: []goast.Expr{valExpr},
	}

	sourcePath := ""
	if ann != nil && ann.OriginalSourcePath != nil {
		sourcePath = ne.emitter.computeRelativePath(*ann.OriginalSourcePath)
	}

	validateCall := &goast.CallExpr{
		Fun: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("ValidatePikoElementTagName")},
		Args: []goast.Expr{
			stringExpr,
			cachedIdent(DiagnosticsVarName),
			strLit(sourcePath),
			strLit(isAttr.RawExpression),
			intLit(isAttr.Location.Line),
			intLit(isAttr.Location.Column),
		},
	}

	assignStmt := &goast.AssignStmt{
		Lhs: []goast.Expr{
			&goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldTagName)},
			cachedIdent(DiagnosticsVarName),
		},
		Tok: token.ASSIGN,
		Rhs: []goast.Expr{validateCall},
	}

	prereqStmts = append(prereqStmts, assignStmt)
	return prereqStmts, expressionDiagnostics
}

// emitTextContent outputs either rich text or plain text content.
//
// Takes nodeVar (*goast.Ident) which is the node variable to assign to.
// Takes node (*ast_domain.TemplateNode) which holds the template node data.
//
// Returns []goast.Stmt which contains the generated statements, or nil if
// there is no text content.
// Returns []*ast_domain.Diagnostic which contains any issues found during
// rich text processing.
func (ne *nodeEmitter) emitTextContent(nodeVar *goast.Ident, node *ast_domain.TemplateNode) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	if len(node.RichText) > 0 {
		return ne.emitRichText(nodeVar, node)
	}

	if node.TextContent == "" {
		return nil, nil
	}

	return []goast.Stmt{emitStaticTextContent(nodeVar, node.TextContent)}, nil
}

// emitRichText generates code to render RichText with mixed expressions.
// It uses DirectWriter for efficient rendering with HTML escaping at render
// time.
//
// Takes nodeVar (*goast.Ident) which identifies the node variable to assign
// the writer to.
// Takes node (*ast_domain.TemplateNode) which contains the RichText parts to
// process.
//
// Returns []goast.Stmt which contains the generated statements for rendering.
// Returns []*ast_domain.Diagnostic which contains any errors found while
// processing the RichText parts.
func (ne *nodeEmitter) emitRichText(nodeVar *goast.Ident, node *ast_domain.TemplateNode) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	dwVar := ne.emitter.nextTempName()
	dwIdent := cachedIdent(dwVar)

	var allDiags []*ast_domain.Diagnostic

	statements := []goast.Stmt{defineAndAssign(dwVar, &goast.CallExpr{
		Fun: &goast.SelectorExpr{X: cachedIdent(arenaVarName), Sel: cachedIdent("GetDirectWriter")},
	})}

	for _, part := range node.RichText {
		partStmts, partDiags := ne.emitRichTextPartToWriter(dwIdent, part)
		statements = append(statements, partStmts...)
		allDiags = append(allDiags, partDiags...)
	}

	statements = append(statements, &goast.AssignStmt{
		Lhs: []goast.Expr{&goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldTextContentWriter)}},
		Tok: token.ASSIGN,
		Rhs: []goast.Expr{dwIdent},
	})

	return statements, allDiags
}

// emitRichTextPartToWriter emits a single RichText part to a
// DirectWriter.
//
// Literal parts use AppendString (trusted developer content, no
// escaping needed). Expression parts use AppendEscapeString for
// safety unless known to be safe types.
//
// Takes dwIdent (*goast.Ident) which identifies the DirectWriter variable.
// Takes part (ast_domain.TextPart) which is the text part to emit.
//
// Returns []goast.Stmt which contains the append statements for the
// text part.
// Returns []*ast_domain.Diagnostic which contains any issues found
// during emission.
func (ne *nodeEmitter) emitRichTextPartToWriter(dwIdent *goast.Ident, part ast_domain.TextPart) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	if part.IsLiteral {
		return []goast.Stmt{&goast.ExprStmt{X: &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: dwIdent, Sel: cachedIdent("AppendString")},
			Args: []goast.Expr{strLit(part.Literal)},
		}}}, nil
	}

	statements := make([]goast.Stmt, 0, smallEmissionStatementCapacity)
	diagnostics := make([]*ast_domain.Diagnostic, 0, 2)

	partGoExpr, partPrereqs, partDiags := ne.expressionEmitter.emit(part.Expression)
	statements = append(statements, partPrereqs...)
	diagnostics = append(diagnostics, partDiags...)

	partGoExpr = ne.expressionEmitter.valueToString(partGoExpr, part.GoAnnotations)

	if shouldSkipEscaping(part.GoAnnotations) {
		statements = append(statements, &goast.ExprStmt{X: &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: dwIdent, Sel: cachedIdent("AppendString")},
			Args: []goast.Expr{partGoExpr},
		}})
	} else {
		statements = append(statements, &goast.ExprStmt{X: &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: dwIdent, Sel: cachedIdent("AppendEscapeString")},
			Args: []goast.Expr{partGoExpr},
		}})
	}

	return statements, diagnostics
}

// emitCollectionInitialisers generates pooled slice retrieval calls for slices
// on the node.
//
// Takes nodeVar (*goast.Ident) which identifies the node variable to assign to.
// Takes node (*ast_domain.TemplateNode) which provides the template structure.
// Takes loopInfoByIndex (map[int]*LoopIterableInfo) which maps loop indices to
// their iterable information for dynamic capacity calculation.
// Takes partialScopeID (string) which identifies the partial scope for static
// attribute registration.
//
// Returns []goast.Stmt which contains the slice initialisation statements.
// Returns bool which indicates whether static attributes were used.
//
// Performance optimisations:
//
// OnEvents and CustomEvents maps are NOT allocated - they are never written to
// in generated code (events are converted to attributes during code
// generation). The renderer handles nil maps gracefully via range over nil
// (which is a no-op in Go).
//
// Slices with capacity 0 are NOT allocated - pooled nodes already have properly
// initialised slices from Reset. Avoids allocating slice headers (24 bytes
// each) for empty slices.
//
// Accurate capacity calculation prevents append reallocations - includes
// partial info attrs, srcset attrs, etc. that are added after the initial
// make call.
//
// Slice pooling eliminates allocations after warmup - GetAttrSlice and
// GetChildSlice retrieve from size-class pools (buckets: 2, 4, 6, 8, 10, 12,
// 16, 24, 32). Only first render allocates.
//
// Dynamic child capacity for p-for loops - uses len(loopIter_N) to calculate
// actual size, preventing reallocation during loop iterations.
//
// Combined, these optimisations eliminate ~250+ allocations per page render
// after warmup.
func (ne *nodeEmitter) emitCollectionInitialisers(
	nodeVar *goast.Ident,
	node *ast_domain.TemplateNode,
	loopInfoByIndex map[int]*LoopIterableInfo,
	partialScopeID string,
) ([]goast.Stmt, bool) {
	numAttrs := ne.calculateAttributeCapacity(node, partialScopeID)
	numAttrWriters := ne.calculateAttributeWriterCapacity(node)

	statements := make([]goast.Stmt, 0, maxCollectionInitStmts)
	usedStaticAttrs := false

	canUseStaticAttrs := numAttrs > 0 && ne.hasAllStaticAttributes(node) && ne.emitter.staticEmitter != nil
	if canUseStaticAttrs {
		staticAttrVarName := ne.emitter.staticEmitter.registerStaticAttributes(node, partialScopeID)
		if staticAttrVarName != "" {
			statements = append(statements, createStaticSliceAssignment(nodeVar, fieldAttributes, staticAttrVarName))
			usedStaticAttrs = true
		} else {
			statements = append(statements, ne.createPooledSliceStmts(nodeVar, fieldAttributes, "GetAttrSlice", numAttrs)...)
		}
	} else if numAttrs > 0 {
		statements = append(statements, ne.createPooledSliceStmts(nodeVar, fieldAttributes, "GetAttrSlice", numAttrs)...)
	}

	if numAttrWriters > 0 {
		statements = append(statements, ne.createPooledSliceStmts(nodeVar, fieldAttributeWriters, "GetAttrWriterSlice", numAttrWriters)...)
	}

	childCapacityExpr := ne.calculateChildCapacityExpr(node, loopInfoByIndex)
	if childCapacityExpr != nil {
		statements = append(statements, ne.createPooledSliceStmtsDynamic(nodeVar, fieldChildren, "GetChildSlice", childCapacityExpr)...)
	}

	return statements, usedStaticAttrs
}

// calculateChildCapacityExpr builds the capacity expression for a children
// slice. Returns nil if there are no children, a static integer literal if all
// children are static, or a dynamic expression that sums the static count with
// len calls for each loop iterable.
//
// Takes node (*ast_domain.TemplateNode) which contains the children to count.
// Takes loopInfoByIndex (map[int]*LoopIterableInfo) which maps child indices to
// their loop iterable information.
//
// Returns goast.Expr which is the capacity expression, or nil if there are no
// children.
func (*nodeEmitter) calculateChildCapacityExpr(
	node *ast_domain.TemplateNode,
	loopInfoByIndex map[int]*LoopIterableInfo,
) goast.Expr {
	numChildren := len(node.Children)
	if numChildren == 0 {
		return nil
	}

	if len(loopInfoByIndex) == 0 {
		return intLit(numChildren)
	}

	staticCount := 0
	for i := range node.Children {
		if _, hasLoopInfo := loopInfoByIndex[i]; !hasLoopInfo {
			staticCount++
		}
	}

	var expression goast.Expr
	if staticCount > 0 {
		expression = intLit(staticCount)
	}

	sortedIdxs := make([]int, 0, len(loopInfoByIndex))
	for index := range loopInfoByIndex {
		sortedIdxs = append(sortedIdxs, index)
	}
	slices.Sort(sortedIdxs)

	for _, index := range sortedIdxs {
		loopInfo := loopInfoByIndex[index]
		lenCall := &goast.CallExpr{
			Fun:  cachedIdent("len"),
			Args: []goast.Expr{cachedIdent(loopInfo.VarName)},
		}

		if expression == nil {
			expression = lenCall
		} else {
			expression = &goast.BinaryExpr{
				X:  expression,
				Op: token.ADD,
				Y:  lenCall,
			}
		}
	}

	return expression
}

// calculateAttributeCapacity calculates the required capacity for the
// Attributes slice.
//
// This counts items that go to node.Attributes (HTMLAttribute structs):
// static attributes, boolean dynamic attributes, class directive result,
// static key attribute, partial info attributes for public partials,
// srcset/sizes attributes for piko:img elements, and event handler attributes.
//
// Non-boolean dynamic attributes, dynamic keys, and style directive go to
// AttributeWriters instead (see calculateAttributeWriterCapacity).
//
// Accurate capacity prevents costly append() reallocations.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to
// analyse for attribute capacity.
// Takes partialScopeID (string) which is the current partial's HashedName for
// CSS scoping.
//
// Returns int which is the total capacity needed for the Attributes slice.
func (ne *nodeEmitter) calculateAttributeCapacity(node *ast_domain.TemplateNode, partialScopeID string) int {
	numAttrs := len(node.Attributes)

	numAttrs += countBooleanDynamicAttributes(node.DynamicAttributes)
	numAttrs += countBooleanBinds(node.Binds)

	if node.Key != nil {
		if _, isLiteral := node.Key.(*ast_domain.StringLiteral); isLiteral {
			numAttrs += SingleDirectiveAttrCount
		}
	}

	if node.DirRef != nil && node.DirRef.RawExpression != "" {
		numAttrs += SingleDirectiveAttrCount
	}

	if ne.willEmitPartialInfoAttributes(node) {
		numAttrs += PartialAttributeCapacity
	} else if partialScopeID != "" && node.NodeType == ast_domain.NodeElement && !nodeHasPartialAttribute(node) {
		numAttrs += SingleDirectiveAttrCount
	}
	if ne.willEmitPartialPropsAttribute(node) {
		numAttrs += PartialPropsAttributeCapacity
	}

	if node.TagName == "piko:img" && node.GoAnnotations != nil && len(node.GoAnnotations.Srcset) > 0 {
		numAttrs += PikoImgSrcsetAttrCount
		if hasWidthDescriptors(node.GoAnnotations.Srcset) && findAttributeValue(node.Attributes, "sizes") != "" {
			numAttrs += PikoImgSizesAttrCount
		}
	}

	numAttrs += ne.countEmittedEventAttributes(node.OnEvents, node)
	numAttrs += ne.countEmittedEventAttributes(node.CustomEvents, node)

	return numAttrs
}

// calculateAttributeWriterCapacity counts how many DirectWriter slots are
// needed for a template node.
//
// This counts items that use DirectWriter for zero-allocation rendering:
// non-boolean dynamic attributes, asset src attributes, dynamic key
// expressions, style directives with dynamic content, and event handlers.
// Getting the right capacity avoids costly slice reallocations from append().
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns int which is the number of DirectWriter slots needed.
func (ne *nodeEmitter) calculateAttributeWriterCapacity(node *ast_domain.TemplateNode) int {
	numWriters := 0

	numWriters += countNonBooleanDynamicAttributes(node.DynamicAttributes, node.TagName)
	numWriters += countNonBooleanBinds(node.Binds)

	if node.Key != nil {
		if _, isLiteral := node.Key.(*ast_domain.StringLiteral); !isLiteral {
			numWriters += SingleDirectiveAttrCount
		}
	}

	if hasDynamicClassContent(node) {
		numWriters += SingleDirectiveAttrCount
	}

	if hasDynamicStyleContent(node) {
		numWriters += SingleDirectiveAttrCount
	}

	numWriters += ne.countEmittedEventAttributes(node.OnEvents, node)
	numWriters += ne.countEmittedEventAttributes(node.CustomEvents, node)

	return numWriters
}

// hasAllStaticAttributes checks if all attributes on a node are static.
// When true, the attribute slice can be moved to package level, which
// removes pool allocations at runtime.
//
// All of these must be true for the node to qualify:
//   - No boolean dynamic attributes (`:disabled="expr"` bindings)
//   - No boolean p-bind directives
//   - Key is nil or a string literal
//   - No event handlers with action or helper modifiers
//   - Node is an element type
//   - Not a piko:img with srcset (processed at runtime)
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns bool which is true if all attributes are static.
//
// Note: Dynamic class and style directives (p-class, p-style) are not checked
// because they now go to AttributeWriters instead of Attributes, so static
// moving works even when the node has dynamic class or style bindings.
func (ne *nodeEmitter) hasAllStaticAttributes(node *ast_domain.TemplateNode) bool {
	if node.NodeType != ast_domain.NodeElement {
		return false
	}

	if countBooleanDynamicAttributes(node.DynamicAttributes) > 0 {
		return false
	}

	if countBooleanBinds(node.Binds) > 0 {
		return false
	}

	if node.Key != nil {
		if _, isLiteral := node.Key.(*ast_domain.StringLiteral); !isLiteral {
			return false
		}
	}

	if ne.countEmittedEventAttributes(node.OnEvents, node) > 0 {
		return false
	}
	if ne.countEmittedEventAttributes(node.CustomEvents, node) > 0 {
		return false
	}

	if node.TagName == "piko:img" && node.GoAnnotations != nil && len(node.GoAnnotations.Srcset) > 0 {
		return false
	}

	return true
}

// countEmittedEventAttributes counts event directives that will become HTML
// attributes.
//
// Takes events (map[string][]ast_domain.Directive) which maps event names to
// their directives.
// Takes node (*ast_domain.TemplateNode) which provides source path for
// HasClientScript lookup.
//
// Returns int which is the count of directives that will become attributes.
func (ne *nodeEmitter) countEmittedEventAttributes(events map[string][]ast_domain.Directive, node *ast_domain.TemplateNode) int {
	count := 0
	for _, directives := range events {
		for i := range directives {
			d := &directives[i]
			switch d.Modifier {
			case actionModifierName, helperModifierName:
				count++
			case "":
				if ne.directiveHasClientScript(d, node) {
					count++
				}
			}
		}
	}
	return count
}

// directiveHasClientScript checks whether a directive's source file has a
// linked client script.
//
// Takes d (*ast_domain.Directive) which is the directive to check.
// Takes node (*ast_domain.TemplateNode) which provides a fallback source path.
//
// Returns bool which is true if the source has a client script.
func (ne *nodeEmitter) directiveHasClientScript(d *ast_domain.Directive, node *ast_domain.TemplateNode) bool {
	if ne.emitter.config.SourcePathHasClientScript == nil {
		return false
	}

	if d.GoAnnotations != nil && d.GoAnnotations.OriginalSourcePath != nil {
		sourcePath := *d.GoAnnotations.OriginalSourcePath
		if hasScript, ok := ne.emitter.config.SourcePathHasClientScript[sourcePath]; ok {
			return hasScript
		}
	}

	if node != nil && node.GoAnnotations != nil && node.GoAnnotations.OriginalSourcePath != nil {
		sourcePath := *node.GoAnnotations.OriginalSourcePath
		if hasScript, ok := ne.emitter.config.SourcePathHasClientScript[sourcePath]; ok {
			return hasScript
		}
	}

	return false
}

// willEmitPartialInfoAttributes checks whether partial info attributes will be
// added for the given node.
//
// This mirrors the logic in emitPartialInfoAttributes to ensure accurate
// capacity calculation.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true when partial info attributes will be added.
func (ne *nodeEmitter) willEmitPartialInfoAttributes(node *ast_domain.TemplateNode) bool {
	if node.GoAnnotations == nil || node.GoAnnotations.PartialInfo == nil {
		return false
	}
	for i := range node.Attributes {
		if node.Attributes[i].Name == partialAttrName {
			return false
		}
	}
	if ne.emitter.AnnotationResult == nil || ne.emitter.AnnotationResult.VirtualModule == nil {
		return false
	}
	pInfo := node.GoAnnotations.PartialInfo
	_, ok := ne.emitter.AnnotationResult.VirtualModule.ComponentsByHash[pInfo.PartialPackageName]
	return ok
}

// willEmitPartialPropsAttribute checks whether a partial_props attribute will
// be emitted for the given node.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true when partial_props will be emitted.
func (ne *nodeEmitter) willEmitPartialPropsAttribute(node *ast_domain.TemplateNode) bool {
	if node.GoAnnotations == nil || node.GoAnnotations.PartialInfo == nil {
		return false
	}
	if ne.emitter.AnnotationResult == nil || ne.emitter.AnnotationResult.VirtualModule == nil {
		return false
	}
	pInfo := node.GoAnnotations.PartialInfo
	partialVC, ok := ne.emitter.AnnotationResult.VirtualModule.ComponentsByHash[pInfo.PartialPackageName]
	if !ok || !partialVC.IsPublic {
		return false
	}
	return len(extractPrimitiveQueryPropsFromComponent(partialVC)) > 0
}

// extractLoopIterables scans children for p-for directives and extracts their
// collection expressions to variables. This enables accurate child slice
// capacity calculation and guarantees collection expressions (which may be
// method calls) are only evaluated once.
//
// Takes children ([]*ast_domain.TemplateNode) which contains the child nodes
// to scan for p-for directives.
//
// Returns extractStmts ([]goast.Stmt) which contains assignment statements
// like `loopIter_1 := pageData.Items`.
// Returns loopInfoByIndex (map[int]*LoopIterableInfo) which maps child index to
// LoopIterableInfo for children with p-for directives.
// Returns diagnostics ([]*ast_domain.Diagnostic) which contains any
// diagnostics from expression emission.
func (ne *nodeEmitter) extractLoopIterables(
	children []*ast_domain.TemplateNode,
) (extractStmts []goast.Stmt, loopInfoByIndex map[int]*LoopIterableInfo, diagnostics []*ast_domain.Diagnostic) {
	loopInfoByIndex = make(map[int]*LoopIterableInfo)

	for i, child := range children {
		if child.DirFor == nil {
			continue
		}

		forExpr, ok := child.DirFor.Expression.(*ast_domain.ForInExpression)
		if !ok {
			continue
		}

		varName := ne.emitter.nextLoopIterName()

		collGoExpr, prereqStmts, collDiags := ne.expressionEmitter.emit(forExpr.Collection)
		extractStmts = append(extractStmts, prereqStmts...)
		diagnostics = append(diagnostics, collDiags...)

		extractStmts = append(extractStmts, defineAndAssign(varName, collGoExpr))

		collAnn := getAnnotationFromExpression(forExpr.Collection)
		isNillable := ne.isCollectionNillable(collAnn)

		loopInfoByIndex[i] = &LoopIterableInfo{
			VarName:              varName,
			CollectionExpression: collGoExpr,
			CollectionAnn:        collAnn,
			IsNillable:           isNillable,
		}
	}

	return extractStmts, loopInfoByIndex, diagnostics
}

// isCollectionNillable checks if a collection type can be nil and needs a nil
// check.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the type
// information to check.
//
// Returns bool which is true if the type can be nil.
func (*nodeEmitter) isCollectionNillable(ann *ast_domain.GoGeneratorAnnotation) bool {
	if ann == nil || ann.ResolvedType == nil {
		return true
	}

	switch ann.ResolvedType.TypeExpression.(type) {
	case *goast.ArrayType, *goast.MapType:
		return true
	case *goast.Ident:
		if identifier, ok := ann.ResolvedType.TypeExpression.(*goast.Ident); ok {
			return identifier.Name != "string"
		}
	}
	return true
}

// createPooledSliceStmts creates statements that retrieve a slice from the
// arena and assign it to a node field.
//
// Generates code like:
// _, sliceVar := arena.GetAttrSlice(capacity)
// nodeVar.Attributes = sliceVar
//
// Takes nodeVar (*goast.Ident) which is the variable to assign the slice to.
// Takes fieldName (string) which is the field name on the node variable.
// Takes poolFunc (string) which is the arena pool function name to call.
// Takes capacity (int) which is the initial capacity for the pooled slice.
//
// Returns []goast.Stmt which contains the built statements.
func (ne *nodeEmitter) createPooledSliceStmts(nodeVar *goast.Ident, fieldName string, poolFunc string, capacity int) []goast.Stmt {
	return ne.createPooledSliceStmtsDynamic(nodeVar, fieldName, poolFunc, intLit(capacity))
}

// createPooledSliceStmtsDynamic creates assignment statements that get a
// pooled slice with a dynamic capacity from the arena.
//
// Generates code like:
// _, sliceVar := arena.GetChildSlice(staticCount + len(loopIter_1))
// nodeVar.Children = sliceVar
// The arena handles all slice lifecycle management internally - no pool pointer
// tracking is needed on the node.
//
// Takes nodeVar (*goast.Ident) which is the variable to assign the slice to.
// Takes fieldName (string) which is the field name on the node variable.
// Takes poolFunc (string) which is the arena pool function to call.
// Takes capacityExpr (goast.Expr) which is the expression for the slice size.
//
// Returns []goast.Stmt which contains the generated statements.
func (ne *nodeEmitter) createPooledSliceStmtsDynamic(nodeVar *goast.Ident, fieldName string, poolFunc string, capacityExpr goast.Expr) []goast.Stmt {
	sliceVarName := ne.emitter.nextTempName() + "Slice"
	sliceVar := cachedIdent(sliceVarName)

	defineStmt := &goast.AssignStmt{
		Lhs: []goast.Expr{cachedIdent("_"), sliceVar},
		Tok: token.DEFINE,
		Rhs: []goast.Expr{&goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   cachedIdent(arenaVarName),
				Sel: cachedIdent(poolFunc),
			},
			Args: []goast.Expr{capacityExpr},
		}},
	}

	assignSliceStmt := &goast.AssignStmt{
		Lhs: []goast.Expr{&goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldName)}},
		Tok: token.ASSIGN,
		Rhs: []goast.Expr{sliceVar},
	}

	return []goast.Stmt{defineStmt, assignSliceStmt}
}

// emitContentDirectives handles p-text and p-html directives, which overwrite
// child content.
//
// p-text uses TextContentWriter for zero-allocation rendering with HTML
// escaping at render time. p-html uses InnerHTML directly (no escaping -
// intentionally raw HTML).
//
// Takes nodeVar (*goast.Ident) which is the variable referencing the DOM node.
// Takes node (*ast_domain.TemplateNode) which contains the directive to emit.
//
// Returns bool which indicates whether a directive was processed.
// Returns []goast.Stmt which contains the generated statements.
// Returns []*ast_domain.Diagnostic which contains any diagnostics raised.
func (ne *nodeEmitter) emitContentDirectives(nodeVar *goast.Ident, node *ast_domain.TemplateNode) (bool, []goast.Stmt, []*ast_domain.Diagnostic) {
	if node.DirText != nil {
		return ne.emitPTextDirective(nodeVar, node.DirText)
	}
	if node.DirHTML != nil {
		return ne.emitPHTMLDirective(nodeVar, node.DirHTML)
	}
	return false, nil, nil
}

// emitPTextDirective handles the p-text directive using TextContentWriter for
// rendering without memory allocation.
//
// Takes nodeVar (*goast.Ident) which identifies the target node variable.
// Takes directive (*ast_domain.Directive) which specifies the text directive
// to emit.
//
// Returns bool which indicates whether the directive was processed.
// Returns []goast.Stmt which contains the generated Go statements.
// Returns []*ast_domain.Diagnostic which holds any issues found during
// emission.
func (ne *nodeEmitter) emitPTextDirective(nodeVar *goast.Ident, directive *ast_domain.Directive) (bool, []goast.Stmt, []*ast_domain.Diagnostic) {
	valGoExpr, prereqs, expressionDiagnostics := ne.expressionEmitter.emit(directive.Expression)

	dwStmts := ne.buildTextContentWriterStmts(nodeVar, valGoExpr, directive.GoAnnotations)

	if ann := directive.GoAnnotations; ann != nil && ann.ResolvedType != nil && isNillableType(ann.ResolvedType.TypeExpression) {
		ifStmt := &goast.IfStmt{
			Cond: &goast.BinaryExpr{X: valGoExpr, Op: token.NEQ, Y: cachedIdent("nil")},
			Body: &goast.BlockStmt{List: dwStmts},
		}
		return true, append(prereqs, ifStmt), expressionDiagnostics
	}

	return true, append(prereqs, dwStmts...), expressionDiagnostics
}

// buildTextContentWriterStmts creates statements that set up a DirectWriter
// for plain text content.
//
// Takes nodeVar (*goast.Ident) which is the AST node to assign the writer to.
// Takes valGoExpr (goast.Expr) which is the value to write.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which controls escaping.
//
// Returns []goast.Stmt which gets a DirectWriter from the pool, adds the value,
// and assigns it to the node.
func (ne *nodeEmitter) buildTextContentWriterStmts(
	nodeVar *goast.Ident,
	valGoExpr goast.Expr,
	ann *ast_domain.GoGeneratorAnnotation,
) []goast.Stmt {
	dwVar := ne.emitter.nextTempName()
	dwIdent := cachedIdent(dwVar)

	stringValExpr := ne.expressionEmitter.valueToString(valGoExpr, ann)

	appendMethod := "AppendEscapeString"
	if shouldSkipEscaping(ann) {
		appendMethod = "AppendString"
	}

	return []goast.Stmt{
		defineAndAssign(dwVar, &goast.CallExpr{
			Fun: &goast.SelectorExpr{X: cachedIdent(arenaVarName), Sel: cachedIdent("GetDirectWriter")},
		}),
		&goast.ExprStmt{X: &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: dwIdent, Sel: cachedIdent(appendMethod)},
			Args: []goast.Expr{stringValExpr},
		}},
		&goast.AssignStmt{
			Lhs: []goast.Expr{&goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldTextContentWriter)}},
			Tok: token.ASSIGN,
			Rhs: []goast.Expr{dwIdent},
		},
	}
}

// emitPHTMLDirective handles the p-html directive by setting InnerHTML with no
// escaping, for use when raw HTML output is intended.
//
// Takes nodeVar (*goast.Ident) which identifies the DOM node variable.
// Takes directive (*ast_domain.Directive) which contains the p-html directive.
//
// Returns bool which indicates whether the directive was handled.
// Returns []goast.Stmt which contains the generated assignment statements.
// Returns []*ast_domain.Diagnostic which contains any diagnostics from
// expression processing.
func (ne *nodeEmitter) emitPHTMLDirective(nodeVar *goast.Ident, directive *ast_domain.Directive) (bool, []goast.Stmt, []*ast_domain.Diagnostic) {
	valGoExpr, prereqStmts, expressionDiagnostics := ne.expressionEmitter.emit(directive.Expression)
	stringValExpr := ne.expressionEmitter.valueToString(valGoExpr, directive.GoAnnotations)

	assignment := &goast.AssignStmt{
		Lhs: []goast.Expr{&goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldInnerHTML)}},
		Tok: token.ASSIGN,
		Rhs: []goast.Expr{stringValExpr},
	}

	if ann := directive.GoAnnotations; ann != nil && ann.ResolvedType != nil && isNillableType(ann.ResolvedType.TypeExpression) {
		ifStmt := &goast.IfStmt{
			Cond: &goast.BinaryExpr{X: valGoExpr, Op: token.NEQ, Y: cachedIdent("nil")},
			Body: &goast.BlockStmt{List: []goast.Stmt{assignment}},
		}
		return true, append(prereqStmts, ifStmt), expressionDiagnostics
	}

	return true, append(prereqStmts, assignment), expressionDiagnostics
}

// emitMiscDirectives handles simple directives and flags attached to the node.
//
// DirRef is not included here as it is handled separately in
// attribute_emitter.emitRefAttribute because p-ref is always a static string
// literal (per validation) and emits as an HTML attribute.
//
// Takes nodeVar (*goast.Ident) which identifies the node variable to assign to.
// Takes node (*ast_domain.TemplateNode) which provides the template node with
// directives.
//
// Returns []goast.Stmt which contains the generated assignment statements.
// Returns []*ast_domain.Diagnostic which contains any diagnostics from
// expression emission.
func (ne *nodeEmitter) emitMiscDirectives(nodeVar *goast.Ident, node *ast_domain.TemplateNode) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	statements := make([]goast.Stmt, 0, MiscDirectiveCapacity)
	var allDiags []*ast_domain.Diagnostic
	directives := map[string]*ast_domain.Directive{
		fieldDirModel:    node.DirModel,
		fieldDirScaffold: node.DirScaffold,
	}

	for fieldName, directive := range directives {
		if directive != nil && directive.Expression != nil {
			value, prereqs, diagnostics := ne.expressionEmitter.emit(directive.Expression)
			statements = append(statements, prereqs...)
			allDiags = append(allDiags, diagnostics...)
			assignment := &goast.AssignStmt{
				Lhs: []goast.Expr{&goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldName)}},
				Tok: token.ASSIGN,
				Rhs: []goast.Expr{value},
			}
			statements = append(statements, assignment)
		}
	}

	if node.GoAnnotations != nil && node.GoAnnotations.NeedsCSRF {
		statements = append(statements,
			&goast.AssignStmt{
				Lhs: []goast.Expr{&goast.SelectorExpr{X: nodeVar, Sel: cachedIdent("RuntimeAnnotations")}},
				Tok: token.ASSIGN,
				Rhs: []goast.Expr{&goast.CallExpr{
					Fun: &goast.SelectorExpr{X: cachedIdent(arenaVarName), Sel: cachedIdent("GetRuntimeAnnotation")},
				}},
			},
			&goast.AssignStmt{
				Lhs: []goast.Expr{&goast.SelectorExpr{
					X:   &goast.SelectorExpr{X: nodeVar, Sel: cachedIdent("RuntimeAnnotations")},
					Sel: cachedIdent("NeedsCSRF"),
				}},
				Tok: token.ASSIGN,
				Rhs: []goast.Expr{cachedIdent("true")},
			},
		)
	}
	return statements, allDiags
}

// emitSrcsetAttributes builds srcset and sizes attributes for responsive
// piko:img tags.
//
// Takes nodeVar (*goast.Ident) which identifies the node variable in the AST.
// Takes node (*ast_domain.TemplateNode) which holds the template node data.
//
// Returns []goast.Stmt which contains the statements that add the attributes.
func (*nodeEmitter) emitSrcsetAttributes(nodeVar *goast.Ident, node *ast_domain.TemplateNode) []goast.Stmt {
	attributeSliceExpression := &goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldAttributes)}

	useWidthDescriptors := shouldUseWidthDescriptors(node.GoAnnotations.Srcset)
	srcsetValue := buildSrcsetValue(node.GoAnnotations.Srcset, useWidthDescriptors)

	statements := []goast.Stmt{
		appendToSlice(attributeSliceExpression, createHTMLAttributeLiteral("srcset", srcsetValue)),
	}

	if sizesStmt := buildSizesAttributeIfNeeded(node, attributeSliceExpression, useWidthDescriptors); sizesStmt != nil {
		statements = append(statements, sizesStmt)
	}

	return statements
}

// newNodeEmitter creates a node emitter with the given parts.
//
// Takes emitter (*emitter) which handles writing output.
// Takes expressionEmitter (ExpressionEmitter) which writes
// expressions.
// Takes attributeEmitter (AttributeEmitter) which writes
// attributes.
// Takes astBuilder (AstBuilder) which builds AST nodes.
//
// Returns *nodeEmitter which is ready to emit nodes.
func newNodeEmitter(emitter *emitter, expressionEmitter ExpressionEmitter, attributeEmitter AttributeEmitter, astBuilder AstBuilder) *nodeEmitter {
	return &nodeEmitter{
		emitter:           emitter,
		expressionEmitter: expressionEmitter,
		attributeEmitter:  attributeEmitter,
		astBuilder:        astBuilder,
	}
}

// nodeHasPartialAttribute checks if a node already has a partial attribute.
// This prevents adding partial metadata twice when prepareNodeForEmission
// has already added it to root nodes.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true if the node has a partial attribute.
func nodeHasPartialAttribute(node *ast_domain.TemplateNode) bool {
	for i := range node.Attributes {
		if node.Attributes[i].Name == partialAttrName {
			return true
		}
	}
	return false
}

// emitNodeType creates an assignment statement that sets the type field of a
// node.
//
// Takes nodeVar (*goast.Ident) which is the node variable to assign to.
// Takes node (*ast_domain.TemplateNode) which provides the node type value.
//
// Returns goast.Stmt which is the assignment statement for the node type.
func emitNodeType(nodeVar *goast.Ident, node *ast_domain.TemplateNode) goast.Stmt {
	return &goast.AssignStmt{
		Lhs: []goast.Expr{&goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldNodeType)}},
		Tok: token.ASSIGN,
		Rhs: []goast.Expr{&goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(node.NodeType.String())}},
	}
}

// emitTagName creates an assignment statement that sets the TagName field.
//
// Takes nodeVar (*goast.Ident) which is the variable to assign to.
// Takes tagName (string) which is the tag name value to assign.
//
// Returns goast.Stmt which is the assignment statement for the TagName field.
func emitTagName(nodeVar *goast.Ident, tagName string) goast.Stmt {
	return &goast.AssignStmt{
		Lhs: []goast.Expr{&goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldTagName)}},
		Tok: token.ASSIGN,
		Rhs: []goast.Expr{strLit(tagName)},
	}
}

// emitStaticTextContent creates an assignment statement for static text.
//
// Takes nodeVar (*goast.Ident) which is the variable for the node.
// Takes textContent (string) which is the text to assign. The text is escaped
// for HTML before being assigned.
//
// Returns goast.Stmt which is the assignment statement for the text content.
func emitStaticTextContent(nodeVar *goast.Ident, textContent string) goast.Stmt {
	return &goast.AssignStmt{
		Lhs: []goast.Expr{&goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldTextContent)}},
		Tok: token.ASSIGN,
		Rhs: []goast.Expr{strLit(html.EscapeString(textContent))},
	}
}

// hasDynamicClassContent checks if a node has dynamic class content that gets
// passed to Attributes.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns bool which is true if the node has a DirClass directive or a dynamic
// class attribute such as `:class="..."`.
func hasDynamicClassContent(node *ast_domain.TemplateNode) bool {
	if node.DirClass != nil {
		return true
	}
	for i := range node.DynamicAttributes {
		if strings.EqualFold(node.DynamicAttributes[i].Name, "class") {
			return true
		}
	}
	return false
}

// hasDynamicStyleContent checks whether a template node has dynamic style
// content that needs a DirectWriter.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns bool which is true if the node has a DirStyle directive or a dynamic
// style attribute.
func hasDynamicStyleContent(node *ast_domain.TemplateNode) bool {
	if node.DirStyle != nil {
		return true
	}
	for i := range node.DynamicAttributes {
		if strings.EqualFold(node.DynamicAttributes[i].Name, "style") {
			return true
		}
	}
	return false
}

// countBooleanDynamicAttributes counts how many dynamic attributes have a
// boolean type. Boolean attributes are added to the Attributes collection and
// are only shown when their value is true.
//
// Takes dynAttrs ([]ast_domain.DynamicAttribute) which contains the dynamic
// attributes to check.
//
// Returns int which is the number of boolean-typed dynamic attributes.
func countBooleanDynamicAttributes(dynAttrs []ast_domain.DynamicAttribute) int {
	count := 0
	for i := range dynAttrs {
		if isDynamicAttributeBoolean(&dynAttrs[i]) {
			count++
		}
	}
	return count
}

// countNonBooleanDynamicAttributes counts dynamic attributes with
// non-boolean type that go to AttributeWriters, excluding class and
// style attributes which are handled separately.
//
// Takes dynAttrs ([]ast_domain.DynamicAttribute) which contains the
// attributes to inspect.
//
// Returns int which is the number of non-boolean dynamic attributes
// found.
func countNonBooleanDynamicAttributes(dynAttrs []ast_domain.DynamicAttribute, tagName string) int {
	count := 0
	for i := range dynAttrs {
		da := &dynAttrs[i]
		if strings.EqualFold(da.Name, "class") || strings.EqualFold(da.Name, "style") {
			continue
		}
		if tagName == "piko:element" && strings.EqualFold(da.Name, "is") {
			continue
		}
		if !isDynamicAttributeBoolean(da) {
			count++
		}
	}
	return count
}

// countBooleanBinds counts the bind directives that have a boolean type.
//
// Takes binds (map[string]*ast_domain.Directive) which contains the bind
// directives to check.
//
// Returns int which is the count of boolean bind directives found.
func countBooleanBinds(binds map[string]*ast_domain.Directive) int {
	count := 0
	for _, directive := range binds {
		if isBindDirectiveBoolean(directive) {
			count++
		}
	}
	return count
}

// countNonBooleanBinds counts bind directives that are not boolean.
//
// Takes binds (map[string]*ast_domain.Directive) which maps directive names to
// their definitions.
//
// Returns int which is the number of non-boolean bind directives.
func countNonBooleanBinds(binds map[string]*ast_domain.Directive) int {
	count := 0
	for _, directive := range binds {
		if !isBindDirectiveBoolean(directive) {
			count++
		}
	}
	return count
}

// isDynamicAttributeBoolean reports whether a dynamic attribute has a boolean
// type.
//
// Takes da (*ast_domain.DynamicAttribute) which is the attribute to check.
//
// Returns bool which is true when the attribute's resolved type is boolean.
func isDynamicAttributeBoolean(da *ast_domain.DynamicAttribute) bool {
	ann := da.GoAnnotations
	if ann == nil {
		ann = getAnnotationFromExpression(da.Expression)
	}
	return ann != nil && ann.ResolvedType != nil && isBoolType(ann.ResolvedType.TypeExpression)
}

// isBindDirectiveBoolean checks whether a bind directive has a boolean type.
//
// Takes directive (*ast_domain.Directive) which is the directive to check.
//
// Returns bool which is true if the directive has a boolean type.
func isBindDirectiveBoolean(directive *ast_domain.Directive) bool {
	if directive == nil {
		return false
	}
	ann := directive.GoAnnotations
	if ann == nil {
		ann = getAnnotationFromExpression(directive.Expression)
	}
	return ann != nil && ann.ResolvedType != nil && isBoolType(ann.ResolvedType.TypeExpression)
}

// hasWidthDescriptors checks whether all srcset variants have width values
// set for use as width descriptors.
//
// Takes srcset ([]ast_domain.ResponsiveVariantMetadata) which contains the
// image variants to check.
//
// Returns bool which is true if all variants have a width value greater than
// zero, or false if any variant has a width of zero.
func hasWidthDescriptors(srcset []ast_domain.ResponsiveVariantMetadata) bool {
	for _, variant := range srcset {
		if variant.Width == 0 {
			return false
		}
	}
	return true
}

// createStaticSliceAssignment creates an assignment statement that sets a
// field to a package-level static slice variable.
// Produces code like: nodeVar.fieldName = staticVarName.
//
// Takes nodeVar (*goast.Ident) which is the variable to assign to.
// Takes fieldName (string) which is the field name on the node variable.
// Takes staticVarName (string) which is the name of the static variable.
//
// Returns *goast.AssignStmt which is the built assignment statement.
func createStaticSliceAssignment(nodeVar *goast.Ident, fieldName string, staticVarName string) *goast.AssignStmt {
	return &goast.AssignStmt{
		Lhs: []goast.Expr{&goast.SelectorExpr{X: nodeVar, Sel: cachedIdent(fieldName)}},
		Tok: token.ASSIGN,
		Rhs: []goast.Expr{cachedIdent(staticVarName)},
	}
}

// shouldUseWidthDescriptors checks whether to use width (w) or density (x)
// descriptors in a srcset attribute.
//
// Takes srcset ([]ast_domain.ResponsiveVariantMetadata) which contains the
// image variants to check.
//
// Returns bool which is true if all variants have a width set, or false if any
// variant has no width set.
func shouldUseWidthDescriptors(srcset []ast_domain.ResponsiveVariantMetadata) bool {
	for _, variant := range srcset {
		if variant.Width == 0 {
			return false
		}
	}
	return true
}

// buildSrcsetValue builds the srcset attribute value string from
// variants, where useWidthDescriptors is a computed property of
// the srcset data (whether all variants have width values).
//
// Takes srcset ([]ast_domain.ResponsiveVariantMetadata) which contains
// the image variants to format.
// Takes useWidthDescriptors (bool) which selects width descriptors
// when true, density descriptors when false.
//
// Returns string which is the formatted srcset attribute value.
//
// represents data property, not control flag.
func buildSrcsetValue(srcset []ast_domain.ResponsiveVariantMetadata, useWidthDescriptors bool) string {
	srcsetParts := make([]string, 0, len(srcset))
	for _, variant := range srcset {
		descriptor := variant.Density
		if useWidthDescriptors {
			descriptor = fmt.Sprintf("%dw", variant.Width)
		}
		srcsetParts = append(srcsetParts, fmt.Sprintf("%s %s", variant.URL, descriptor))
	}
	return strings.Join(srcsetParts, ", ")
}

// buildSizesAttributeIfNeeded adds a sizes attribute when present
// and width descriptors are in use, where useWidthDescriptors is
// a computed property determining descriptor type.
//
// Takes node (*ast_domain.TemplateNode) which provides the attribute
// list to search for a sizes value.
// Takes attributeSliceExpression (goast.Expr) which is the slice to append the
// sizes attribute to.
// Takes useWidthDescriptors (bool) which gates whether the sizes
// attribute is needed.
//
// Returns goast.Stmt which appends the sizes attribute, or nil when
// width descriptors are not used or no sizes value exists.
//
// represents data property, not control flag.
func buildSizesAttributeIfNeeded(node *ast_domain.TemplateNode, attributeSliceExpression goast.Expr, useWidthDescriptors bool) goast.Stmt {
	if !useWidthDescriptors {
		return nil
	}

	sizesValue := findAttributeValue(node.Attributes, "sizes")
	if sizesValue == "" {
		return nil
	}

	return appendToSlice(attributeSliceExpression, createHTMLAttributeLiteral("sizes", html.EscapeString(sizesValue)))
}

// findAttributeValue returns the value of an attribute with the given name.
//
// Takes attributes ([]ast_domain.HTMLAttribute) which is the list to search.
// Takes name (string) which is the attribute name to find.
//
// Returns string which is the attribute value, or empty if not found.
func findAttributeValue(attributes []ast_domain.HTMLAttribute, name string) string {
	for i := range attributes {
		if attributes[i].Name == name {
			return attributes[i].Value
		}
	}
	return ""
}
