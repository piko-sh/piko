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
	"maps"
	"slices"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/helpers"
	"piko.sh/piko/internal/esbuild/js_ast"
)

// VDOMBuilder defines the interface for building VDOM render functions from
// template ASTs.
type VDOMBuilder interface {
	// BuildRenderVDOM builds a render function that produces a virtual DOM tree.
	//
	// Takes tmplAST (*ast_domain.TemplateAST) which is the parsed template to
	// render.
	// Takes events (*eventBindingCollection) which contains event handler
	// bindings.
	// Takes booleanProps ([]string) which lists properties to treat as boolean.
	//
	// Returns *js_ast.EFunction which is the generated render function.
	// Returns error when the template cannot be converted to a virtual DOM.
	BuildRenderVDOM(ctx context.Context, tmplAST *ast_domain.TemplateAST,
		events *eventBindingCollection, booleanProps []string) (*js_ast.EFunction, error)
}

// vdomBuilder builds a virtual DOM for rendering components.
// It implements the VDOMBuilder interface.
type vdomBuilder struct{}

// attributeIs is the HTML "is" attribute used for customised built-in elements.
const attributeIs = "is"

var _ VDOMBuilder = (*vdomBuilder)(nil)

// BuildRenderVDOM builds the renderVDOM method from a template AST.
//
// Takes tmplAST (*ast_domain.TemplateAST) which provides the parsed template
// structure.
// Takes events (*eventBindingCollection) which tracks event bindings for the
// template.
// Takes booleanProps ([]string) which lists properties to treat as booleans.
//
// Returns *js_ast.EFunction which is the generated renderVDOM method.
// Returns error when processing the root nodes fails.
func (*vdomBuilder) BuildRenderVDOM(
	ctx context.Context,
	tmplAST *ast_domain.TemplateAST,
	events *eventBindingCollection,
	booleanProps []string,
) (*js_ast.EFunction, error) {
	var body js_ast.SBlock

	if tmplAST == nil || len(tmplAST.RootNodes) == 0 {
		returnStmt := js_ast.Stmt{Data: &js_ast.SReturn{
			ValueOrNil: buildDOMCall("cmt", newStringLiteral("No template content"), newStringLiteral("root")),
		}}
		body = js_ast.SBlock{Stmts: []js_ast.Stmt{returnStmt}}
	} else {
		topLevelExprs, err := processChainAwareChildren(ctx, tmplAST.RootNodes, events, nil, booleanProps)
		if err != nil {
			return nil, fmt.Errorf("failed to build VDOM for root nodes: %w", err)
		}

		finalExpr := buildFinalExpr(topLevelExprs)
		returnStmt := js_ast.Stmt{Data: &js_ast.SReturn{ValueOrNil: finalExpr}}
		body = js_ast.SBlock{Stmts: []js_ast.Stmt{returnStmt}}
	}

	renderVDOMMethod := &js_ast.EFunction{
		Fn: js_ast.Fn{
			Body: js_ast.FnBody{Block: body},
		},
	}
	return renderVDOMMethod, nil
}

// NewVDOMBuilder creates a new VDOMBuilder.
//
// Returns VDOMBuilder which is ready to build virtual DOM structures.
func NewVDOMBuilder() VDOMBuilder {
	return &vdomBuilder{}
}

// buildFinalExpr builds the final expression from a slice of expressions.
//
// When the slice is empty, returns a null literal. When there is exactly one
// expression, returns it unchanged. Otherwise, wraps all expressions in a
// DOM fragment call.
//
// Takes topLevelExprs ([]js_ast.Expr) which contains the expressions to
// combine into the final result.
//
// Returns js_ast.Expr which is the resulting expression for the template.
func buildFinalExpr(topLevelExprs []js_ast.Expr) js_ast.Expr {
	if len(topLevelExprs) == 0 {
		return newNullLiteral()
	}
	if len(topLevelExprs) == 1 {
		return topLevelExprs[0]
	}
	return buildDOMCall("frag", newStringLiteral("root_fragment"), js_ast.Expr{Data: &js_ast.EArray{Items: topLevelExprs}})
}

// buildNodeAST builds a JavaScript AST expression from a template node.
//
// Takes n (*ast_domain.TemplateNode) which is the template node to convert.
// Takes events (*eventBindingCollection) which tracks event bindings.
// Takes loopVars (map[string]bool) which holds loop variable names in scope.
// Takes booleanProps ([]string) which lists properties to treat as boolean.
//
// Returns js_ast.Expr which is the JavaScript AST expression for the node.
// Returns error when the key cannot be resolved or child nodes fail to build.
func buildNodeAST(
	ctx context.Context,
	n *ast_domain.TemplateNode,
	events *eventBindingCollection,
	loopVars map[string]bool,
	booleanProps []string,
) (js_ast.Expr, error) {
	keyJSExpr, err := getKeyJSExpr(n, events.getRegistry())
	if err != nil {
		return js_ast.Expr{}, fmt.Errorf("could not resolve key for node <%s>: %w", n.TagName, err)
	}

	if n.DirFor != nil {
		return buildForLoopAST(ctx, n, events, loopVars, booleanProps)
	}

	var nodeJSExpr js_ast.Expr
	switch n.NodeType {
	case ast_domain.NodeText:
		if len(n.RichText) > 0 {
			nodeJSExpr, err = buildRichTextNodeAST(n, keyJSExpr, events.getRegistry())
		} else {
			nodeJSExpr, err = buildTextNodeAST(n, keyJSExpr)
		}
	case ast_domain.NodeComment:
		nodeJSExpr = buildDOMCall("cmt", newStringLiteral(n.TextContent), keyJSExpr)
	case ast_domain.NodeElement:
		nodeJSExpr, err = buildElementNodeAST(ctx, n, events, keyJSExpr, loopVars, booleanProps)
	case ast_domain.NodeFragment:
		nodeJSExpr, err = buildChildFragmentAST(ctx, n, events, loopVars, booleanProps)
	default:
		nodeJSExpr = newNullLiteral()
	}

	if err != nil {
		return js_ast.Expr{}, err
	}

	return nodeJSExpr, nil
}

// getKeyJSExpr gets the key expression from a template node.
//
// Takes n (*ast_domain.TemplateNode) which holds the key to transform.
// Takes registry (*RegistryContext) which provides context for AST changes.
//
// Returns js_ast.Expr which is the JavaScript AST expression for the key.
// Returns error when the AST transformation fails.
func getKeyJSExpr(n *ast_domain.TemplateNode, registry *RegistryContext) (js_ast.Expr, error) {
	return transformOurASTtoJSAST(n.Key, registry)
}

// buildForLoopAST builds a JavaScript AST for a p-for directive loop.
//
// Takes n (*ast_domain.TemplateNode) which is the template node containing the
// p-for directive.
// Takes events (*eventBindingCollection) which tracks event bindings.
// Takes outerVars (map[string]bool) which contains variable names from outer
// scopes.
// Takes booleanProps ([]string) which lists boolean property names.
//
// Returns js_ast.Expr which is the JavaScript AST for the map call.
// Returns error when the p-for expression is not valid or node building fails.
func buildForLoopAST(
	ctx context.Context,
	n *ast_domain.TemplateNode,
	events *eventBindingCollection,
	outerVars map[string]bool,
	booleanProps []string,
) (js_ast.Expr, error) {
	forIn, ok := n.DirFor.Expression.(*ast_domain.ForInExpression)
	if !ok {
		return js_ast.Expr{}, errors.New("internal compiler error: p-for expression was not a ForInExpr")
	}

	itemName := "item"
	if forIn.ItemVariable != nil && forIn.ItemVariable.Name != "" {
		itemName = forIn.ItemVariable.Name
	}

	idxName := ""
	if forIn.IndexVariable != nil && forIn.IndexVariable.Name != "" {
		idxName = forIn.IndexVariable.Name
	}

	registry := events.getRegistry()
	collectionExpr, err := transformOurASTtoJSAST(forIn.Collection, registry)
	if err != nil {
		return js_ast.Expr{}, fmt.Errorf("invalid collection expression in p-for: %w", err)
	}

	clone := cloneNode(n)
	clone.DirFor = nil

	loopScope := copyLoopVarsWith(outerVars, itemName, idxName)
	mapBodyExpr, err := buildNodeAST(ctx, clone, events, loopScope, booleanProps)
	if err != nil {
		return js_ast.Expr{}, fmt.Errorf("failed to build node inside p-for loop: %w", err)
	}

	arrowArgs := []js_ast.Arg{
		{Binding: registry.MakeBinding(itemName)},
	}
	if idxName != "" {
		arrowArgs = append(arrowArgs, js_ast.Arg{Binding: registry.MakeBinding(idxName)})
	}

	arrowFunc := js_ast.Expr{Data: &js_ast.EArrow{
		Args: arrowArgs,
		Body: js_ast.FnBody{
			Block: js_ast.SBlock{
				Stmts: []js_ast.Stmt{
					{Data: &js_ast.SReturn{ValueOrNil: mapBodyExpr}},
				},
			},
		},
	}}

	normalisedCollection := buildRuntimeNormaliseCollection(collectionExpr, registry)

	mapCall := buildMethodCallOnExpr(normalisedCollection, "map", arrowFunc)

	return mapCall, nil
}

// buildRuntimeNormaliseCollection creates a runtime check that handles both
// arrays and objects in p-for loops.
//
// For arrays, the collection passes through unchanged. For objects, the
// collection is converted to Object.entries() which returns key-value pairs
// as [[key, value], ...].
//
// Takes collectionExpr (js_ast.Expr) which is the collection to iterate.
// Takes registry (*RegistryContext) which provides identifier creation.
//
// Returns js_ast.Expr which handles both arrays and objects safely.
func buildRuntimeNormaliseCollection(collectionExpr js_ast.Expr, registry *RegistryContext) js_ast.Expr {
	arrayIdent := registry.MakeIdentifierExpr("Array")
	isArrayCheck := buildMethodCallOnExpr(arrayIdent, "isArray", collectionExpr)

	typeofCheck := js_ast.Expr{Data: &js_ast.EBinary{
		Op: js_ast.BinOpStrictEq,
		Left: js_ast.Expr{Data: &js_ast.EUnary{
			Op:    js_ast.UnOpTypeof,
			Value: collectionExpr,
		}},
		Right: js_ast.Expr{Data: &js_ast.EString{Value: helpers.StringToUTF16("object")}},
	}}

	objectTruthyCheck := js_ast.Expr{Data: &js_ast.EBinary{
		Op:    js_ast.BinOpLogicalAnd,
		Left:  collectionExpr,
		Right: typeofCheck,
	}}

	objectIdent := registry.MakeIdentifierExpr("Object")
	objectEntries := buildMethodCallOnExpr(objectIdent, "entries", collectionExpr)

	objectToEntries := js_ast.Expr{Data: &js_ast.EIf{
		Test: objectTruthyCheck,
		Yes:  objectEntries,
		No:   js_ast.Expr{Data: &js_ast.EArray{Items: []js_ast.Expr{}}},
	}}

	finalTernary := js_ast.Expr{Data: &js_ast.EIf{
		Test: isArrayCheck,
		Yes:  collectionExpr,
		No:   objectToEntries,
	}}

	return finalTernary
}

// Loop variable helpers

// copyLoopVars creates a copy of a loop variable tracking map.
//
// Takes src (map[string]bool) which is the source map to copy.
//
// Returns map[string]bool which is a new map with the same entries, or an
// empty map if src is nil.
func copyLoopVars(src map[string]bool) map[string]bool {
	if src == nil {
		return map[string]bool{}
	}
	dst := make(map[string]bool, len(src))
	maps.Copy(dst, src)
	return dst
}

// copyLoopVarsWith copies loop variables and adds the given item and index
// names to the result.
//
// Takes src (map[string]bool) which holds the current loop variables.
// Takes itemName (string) which is the loop item variable name to add.
// Takes idxName (string) which is the loop index variable name to add.
//
// Returns map[string]bool which holds the copied variables with any non-empty
// item and index names added.
func copyLoopVarsWith(src map[string]bool, itemName, idxName string) map[string]bool {
	dst := copyLoopVars(src)
	if itemName != "" {
		dst[itemName] = true
	}
	if idxName != "" {
		dst[idxName] = true
	}
	return dst
}

// getLoopVarNames extracts the variable names from a loop variables map.
//
// Takes loopVars (map[string]bool) which contains the loop variable names as
// keys.
//
// Returns []string which contains the variable names, or nil if loopVars is
// nil.
func getLoopVarNames(loopVars map[string]bool) []string {
	if loopVars == nil {
		return nil
	}
	return slices.Collect(maps.Keys(loopVars))
}

// buildElementNodeAST builds a JavaScript AST expression for an element node.
//
// Takes n (*ast_domain.TemplateNode) which is the template node to convert.
// Takes events (*eventBindingCollection) which collects event bindings.
// Takes keyJSExpr (js_ast.Expr) which provides the key expression for the
// element.
// Takes loopVars (map[string]bool) which tracks variables from enclosing loops.
// Takes booleanProps ([]string) which lists properties to treat as booleans.
//
// Returns js_ast.Expr which is the constructed element call expression.
// Returns error when property or child AST building fails.
func buildElementNodeAST(
	ctx context.Context,
	n *ast_domain.TemplateNode,
	events *eventBindingCollection,
	keyJSExpr js_ast.Expr,
	loopVars map[string]bool,
	booleanProps []string,
) (js_ast.Expr, error) {
	if strings.EqualFold(n.TagName, "piko:element") {
		return buildPikoElementNodeAST(ctx, n, events, keyJSExpr, loopVars, booleanProps)
	}

	if isAssetTag(n.TagName) {
		return buildAssetElementNodeAST(ctx, n, events, keyJSExpr, loopVars, booleanProps)
	}

	isLink := strings.EqualFold(n.TagName, "piko:a")

	propsExpr, err := buildPropsAST(ctx, n, events, isLink, loopVars, booleanProps)
	if err != nil {
		return js_ast.Expr{}, err
	}

	var childrenExpr js_ast.Expr
	if n.DirText != nil {
		childrenExpr, err = dirTextDynamicExpr(n.DirText.Expression, keyJSExpr, events.getRegistry())
	} else if n.DirHTML != nil {
		childrenExpr, err = dirHTMLDynamicExpr(n.DirHTML.Expression, keyJSExpr, events.getRegistry())
	} else {
		childrenExpr, err = buildChildFragmentAST(ctx, n, events, loopVars, booleanProps)
	}
	if err != nil {
		return js_ast.Expr{}, err
	}

	elementCall := buildDOMCall("el",
		newStringLiteral(pickTagName(n, isLink)),
		keyJSExpr,
		propsExpr,
		childrenExpr,
	)

	return elementCall, nil
}

// buildPikoElementNodeAST handles <piko:element :is="expr"> by compiling the
// dynamic :is expression and using dom.resolveTag(expr) as the tag argument.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes n (*ast_domain.TemplateNode) which is the piko:element node.
// Takes events (*eventBindingCollection) which collects event bindings.
// Takes keyJSExpr (js_ast.Expr) which is the key expression.
// Takes loopVars (map[string]bool) which tracks loop variables.
// Takes booleanProps ([]string) which lists boolean properties.
//
// Returns js_ast.Expr which is the dom.el() call with dynamic tag.
// Returns error when expression compilation or child building fails.
func buildPikoElementNodeAST(
	ctx context.Context,
	n *ast_domain.TemplateNode,
	events *eventBindingCollection,
	keyJSExpr js_ast.Expr,
	loopVars map[string]bool,
	booleanProps []string,
) (js_ast.Expr, error) {
	registry := events.getRegistry()

	var rawIsExpr js_ast.Expr
	var isDynamic bool
	for i := range n.DynamicAttributes {
		if strings.EqualFold(n.DynamicAttributes[i].Name, attributeIs) {
			jsExpr, err := transformOurASTtoJSAST(n.DynamicAttributes[i].Expression, registry)
			if err != nil {
				return js_ast.Expr{}, err
			}
			rawIsExpr = jsExpr
			isDynamic = true
			break
		}
	}

	var tagExpr js_ast.Expr
	if !isDynamic {
		if staticIs, ok := n.GetAttribute(attributeIs); ok && staticIs != "" {
			tagExpr = newStringLiteral(staticIs)
		} else {
			tagExpr = newStringLiteral("div")
		}
	}

	propsExpr, err := buildPikoElementPropsAST(ctx, n, events, loopVars, booleanProps)
	if err != nil {
		return js_ast.Expr{}, err
	}

	var childrenExpr js_ast.Expr
	if n.DirText != nil {
		childrenExpr, err = dirTextDynamicExpr(n.DirText.Expression, keyJSExpr, registry)
	} else if n.DirHTML != nil {
		childrenExpr, err = dirHTMLDynamicExpr(n.DirHTML.Expression, keyJSExpr, registry)
	} else {
		childrenExpr, err = buildChildFragmentAST(ctx, n, events, loopVars, booleanProps)
	}
	if err != nil {
		return js_ast.Expr{}, err
	}

	var elementCall js_ast.Expr
	if isDynamic {
		moduleNameExpr := newStringLiteral(GetModuleName(ctx))
		elementCall = buildDOMCall("pikoEl", rawIsExpr, keyJSExpr, propsExpr, childrenExpr, moduleNameExpr)
	} else {
		elementCall = buildDOMCall("el", tagExpr, keyJSExpr, propsExpr, childrenExpr)
	}
	return elementCall, nil
}

// buildPikoElementPropsAST builds props for a piko:element,
// excluding the :is attribute which is consumed for the dynamic
// tag name.
//
// Takes n (*ast_domain.TemplateNode) which is the piko:element
// node.
// Takes events (*eventBindingCollection) which collects event
// bindings.
// Takes loopVars (map[string]bool) which tracks loop variables.
// Takes booleanProps ([]string) which lists boolean properties.
//
// Returns js_ast.Expr which is the props object expression.
// Returns error when property building fails.
func buildPikoElementPropsAST(
	ctx context.Context,
	n *ast_domain.TemplateNode,
	events *eventBindingCollection,
	loopVars map[string]bool,
	booleanProps []string,
) (js_ast.Expr, error) {
	registry := events.getRegistry()
	properties := make(map[string]js_ast.Expr)
	multiValueProps := make(map[string][]js_ast.Expr)

	collectDirectiveProps(n, properties, registry)
	collectStaticAttrs(n, properties)
	delete(properties, attributeIs)

	for dynamicAttributeIndex := range n.DynamicAttributes {
		dynamicAttribute := &n.DynamicAttributes[dynamicAttributeIndex]
		if strings.EqualFold(dynamicAttribute.Name, attributeIs) {
			continue
		}
		jsExpr, _ := transformOurASTtoJSAST(dynamicAttribute.Expression, registry)
		if jsExpr.Data == nil {
			continue
		}
		propName := dynamicAttribute.Name
		if isBooleanBound(dynamicAttribute.Expression, booleanProps) {
			propName = "?" + propName
		}
		properties[propName] = js_ast.Expr{Data: &js_ast.EUnary{Op: js_ast.UnOpPos, Value: jsExpr}}
	}

	collectBindProps(n, properties, false, booleanProps, js_ast.Expr{}, registry)

	if n.DirModel != nil {
		handleModelDirective(ctx, n, properties, multiValueProps, events, loopVars)
	}

	userClicked := collectEventHandlers(ctx, n, events, loopVars, multiValueProps)
	_ = userClicked

	mergeMultiValueProps(properties, multiValueProps)

	return buildPropsObject(properties), nil
}

// pickTagName returns the tag name to use for a template node.
//
// When isLink is true, returns "a" for anchor elements. Otherwise, returns the
// tag name from the node.
//
// Takes n (*ast_domain.TemplateNode) which provides the default tag name.
// Takes isLink (bool) which shows whether the element is a link.
//
// Returns string which is the tag name to use.
func pickTagName(n *ast_domain.TemplateNode, isLink bool) string {
	if isLink {
		return "a"
	}
	return n.TagName
}

// buildChildFragmentAST handles rendering of a fragment's children.
//
// When there are no children, returns a null literal. When there is exactly
// one child, returns that child directly. Otherwise, wraps the children in a
// fragment with a derived key.
//
// Takes parentNode (*ast_domain.TemplateNode) which provides the parent
// element and its children to process.
// Takes events (*eventBindingCollection) which collects event bindings found
// during processing.
// Takes loopVars (map[string]bool) which tracks variables defined in loops.
// Takes booleanProps ([]string) which lists properties that should be treated
// as boolean attributes.
//
// Returns js_ast.Expr which is the JavaScript AST expression for the children.
// Returns error when child processing fails or the fragment key cannot be
// derived.
func buildChildFragmentAST(
	ctx context.Context,
	parentNode *ast_domain.TemplateNode,
	events *eventBindingCollection,
	loopVars map[string]bool,
	booleanProps []string,
) (js_ast.Expr, error) {
	childExprs, err := processChainAwareChildren(ctx, parentNode.Children, events, loopVars, booleanProps)
	if err != nil {
		return js_ast.Expr{}, err
	}

	if len(childExprs) == 0 {
		return newNullLiteral(), nil
	}
	if len(childExprs) == 1 {
		return childExprs[0], nil
	}

	parentKeyJS, err := transformOurASTtoJSAST(parentNode.Key, events.getRegistry())
	if err != nil {
		return js_ast.Expr{}, fmt.Errorf("could not get parent key to derive fragment key: %w", err)
	}

	fragmentKeyJS, err := deriveFragmentKeyJSExpr(parentKeyJS)
	if err != nil {
		return js_ast.Expr{}, fmt.Errorf("could not derive fragment key: %w", err)
	}

	return buildDOMCall("frag", fragmentKeyJS, js_ast.Expr{Data: &js_ast.EArray{Items: childExprs}}), nil
}

// deriveFragmentKeyJSExpr creates a child key from a parent key by adding a
// suffix to make it unique.
//
// Takes parentKey (js_ast.Expr) which is the parent key expression to build
// from.
//
// Returns js_ast.Expr which is the new child key expression.
// Returns error when the parent key cannot be processed.
func deriveFragmentKeyJSExpr(parentKey js_ast.Expr) (js_ast.Expr, error) {
	suffix := "_f"

	switch k := parentKey.Data.(type) {
	case *js_ast.EString:
		str := helpers.UTF16ToString(k.Value)
		return newStringLiteral(str + suffix), nil
	case *js_ast.ETemplate:
		if len(k.Parts) > 0 {
			lastIndex := len(k.Parts) - 1
			k.Parts[lastIndex].TailCooked = append(k.Parts[lastIndex].TailCooked, helpers.StringToUTF16(suffix)...)
		} else {
			k.HeadCooked = append(k.HeadCooked, helpers.StringToUTF16(suffix)...)
		}
		return parentKey, nil
	}

	return js_ast.Expr{Data: &js_ast.ETemplate{
		HeadCooked: nil,
		Parts: []js_ast.TemplatePart{
			{
				Value:      parentKey,
				TailCooked: helpers.StringToUTF16(suffix),
			},
		},
	}}, nil
}

// buildRichTextNodeAST builds a text node that contains mixed literal text and
// expressions.
//
// Takes n (*ast_domain.TemplateNode) which contains the text parts to process.
// Takes keyExpr (js_ast.Expr) which provides the key for the text node.
// Takes registry (*RegistryContext) which holds the compilation context.
//
// Returns js_ast.Expr which is the JavaScript AST for the text node.
// Returns error when expression transformation fails.
func buildRichTextNodeAST(n *ast_domain.TemplateNode, keyExpr js_ast.Expr, registry *RegistryContext) (js_ast.Expr, error) {
	if len(n.RichText) == 1 && n.RichText[0].IsLiteral {
		return buildTextNodeAST(n, keyExpr)
	}

	var parts []js_ast.Expr
	for _, part := range n.RichText {
		if part.IsLiteral {
			if part.Literal != "" {
				parts = append(parts, newStringLiteral(part.Literal))
			}
		} else if part.Expression != nil {
			jsExpr, err := transformOurASTtoJSAST(part.Expression, registry)
			if err != nil {
				jsExpr = newStringLiteral("/* Template Expression Error */")
			}

			safeExpr := js_ast.Expr{Data: &js_ast.ECall{
				Target: newIdentifier(jsString),
				Args: []js_ast.Expr{{Data: &js_ast.EBinary{
					Op:    js_ast.BinOpNullishCoalescing,
					Left:  jsExpr,
					Right: newStringLiteral(""),
				}}},
			}}
			parts = append(parts, safeExpr)
		}
	}

	if len(parts) == 0 {
		return newNullLiteral(), nil
	}

	concatenatedExpr := parts[0]
	for i := 1; i < len(parts); i++ {
		concatenatedExpr = js_ast.Expr{Data: &js_ast.EBinary{
			Op:    js_ast.BinOpAdd,
			Left:  concatenatedExpr,
			Right: parts[i],
		}}
	}

	return buildDOMCall("txt", concatenatedExpr, keyExpr), nil
}

// buildTextNodeAST builds a text node for the virtual DOM.
//
// Takes n (*ast_domain.TemplateNode) which holds the text content to render.
// Takes keyExpr (js_ast.Expr) which provides the node key expression.
//
// Returns js_ast.Expr which is the compiled JavaScript AST for the text node.
// Returns error when the node cannot be built.
func buildTextNodeAST(n *ast_domain.TemplateNode, keyExpr js_ast.Expr) (js_ast.Expr, error) {
	text := n.TextContent
	if n.PreserveWhitespace {
		if text == "" {
			return buildDOMCall("ws", keyExpr), nil
		}
	} else {
		text = squashWhitespace(text)
		if strings.TrimSpace(text) == "" {
			return buildDOMCall("ws", keyExpr), nil
		}
	}
	return buildDOMCall("txt", newStringLiteral(text), keyExpr), nil
}
