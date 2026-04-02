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

package annotator_domain

// Links prop data sources by resolving p-data attributes and connecting them to their corresponding component props.
// Validates data source expressions, enforces type compatibility, and annotates the AST with resolved data bindings.

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

// pdsSetterVisitor traverses the AST to set package-dot-symbol references.
type pdsSetterVisitor struct {
	// diagnostics collects problems found during traversal.
	diagnostics *[]*ast_domain.Diagnostic

	// virtualModule holds the virtual module being processed.
	virtualModule *annotator_dto.VirtualModule

	// typeResolver finds type information for documentation.
	typeResolver *TypeResolver

	// depth tracks how deep the current node is during AST traversal.
	depth int
}

// Enter implements the ast_domain.Visitor interface.
//
// Takes ctx (context.Context) which carries the session logger.
// Takes node (*ast_domain.TemplateNode) which is the current node in the tree.
//
// Returns ast_domain.Visitor which is this visitor to continue the traversal.
// Returns error when the node cannot be processed.
func (v *pdsSetterVisitor) Enter(_ context.Context, _ *ast_domain.TemplateNode) (ast_domain.Visitor, error) {
	v.depth++
	return v, nil
}

// Exit implements the ast_domain.Visitor interface. It processes partial
// invocation nodes when leaving the AST, setting up property data sources for
// passed props that have resolved annotations.
//
// Takes ctx (context.Context) which carries the session logger.
// Takes node (*ast_domain.TemplateNode) which is the AST node being exited.
//
// Returns error when processing fails.
func (v *pdsSetterVisitor) Exit(ctx context.Context, node *ast_domain.TemplateNode) error {
	defer func() { v.depth-- }()

	if !isPartialInvocationNode(node) {
		return nil
	}

	partialInfo := node.GoAnnotations.PartialInfo
	_, l := logger_domain.From(ctx, log)
	l.Trace("[PDS-SETTER Exit] Processing invocation.",
		logger_domain.Int("depth", v.depth),
		logger_domain.String("alias", partialInfo.PartialAlias),
		logger_domain.String("key", partialInfo.InvocationKey),
	)

	for htmlPropName, passedPropValue := range partialInfo.PassedProps {
		sourceExpressionession := passedPropValue.Expression
		sourceAnn := getAnnotationFromExpression(sourceExpressionession)
		if sourceAnn == nil {
			continue
		}

		if sourceAnn.PropDataSource == nil {
			l.Trace("[PDS-SETTER] Creating initial PropDataSource.",
				logger_domain.Int("depth", v.depth),
				logger_domain.String("prop", htmlPropName),
				logger_domain.String("expr", sourceExpressionession.String()),
			)
			sourceAnn.PropDataSource = &ast_domain.PropDataSource{
				ResolvedType:       sourceAnn.ResolvedType.Clone(),
				Symbol:             sourceAnn.Symbol.Clone(),
				BaseCodeGenVarName: sourceAnn.BaseCodeGenVarName,
			}
		}
	}
	return nil
}

// pdsLinkerVisitor links partial template calls to their property sources.
// It implements the ast_domain.Visitor interface.
type pdsLinkerVisitor struct {
	// diagnostics collects problems found during parsing.
	diagnostics *[]*ast_domain.Diagnostic

	// virtualModule holds the module being linked.
	virtualModule *annotator_dto.VirtualModule

	// typeResolver provides type information for linking documentation.
	typeResolver *TypeResolver

	// propMapsByHash stores cached property name maps keyed by component hash.
	propMapsByHash map[string]map[string]string

	// validPropInfoCache stores valid property info for each component, keyed by
	// hashed name. A nil entry means a previous lookup failed.
	validPropInfoCache map[string]map[string]validPropInfo

	// invocationByNode maps template nodes to their partial invocation info.
	invocationByNode map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo

	// invocationMapStack holds previous invocation maps when entering child nodes.
	invocationMapStack []map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo

	// depth tracks the current nesting level during AST tree traversal.
	depth int
}

// Enter implements the ast_domain.Visitor interface.
//
// Takes ctx (context.Context) which carries the session logger.
// Takes node (*ast_domain.TemplateNode) which is the AST node to visit.
//
// Returns ast_domain.Visitor which is the visitor for child nodes, or nil if
// the node is nil.
// Returns error when visiting fails.
func (v *pdsLinkerVisitor) Enter(ctx context.Context, node *ast_domain.TemplateNode) (ast_domain.Visitor, error) {
	if node == nil {
		return nil, nil
	}

	_, l := logger_domain.From(ctx, log)
	l.Trace("[PDS-LINKER Enter] Visiting node.", logger_domain.Int("depth", v.depth), logger_domain.String("tag", node.TagName))

	v.depth++
	v.invocationMapStack = append(v.invocationMapStack, v.invocationByNode)

	childInvocationMap := make(map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo, len(v.invocationByNode))
	maps.Copy(childInvocationMap, v.invocationByNode)

	if isPartialInvocationNode(node) {
		partialInfo := node.GoAnnotations.PartialInfo
		childInvocationMap[node] = partialInfo
		v.linkPropsForInvocation(ctx, node, partialInfo, childInvocationMap)
	}

	v.invocationByNode = childInvocationMap
	return v, nil
}

// Exit implements the ast_domain.Visitor interface. It restores the
// invocation map to the parent scope when leaving a template node.
//
// Takes ctx (context.Context) which carries the session logger.
//
// Returns error when the exit fails.
func (v *pdsLinkerVisitor) Exit(_ context.Context, _ *ast_domain.TemplateNode) error {
	v.depth--
	stackSize := len(v.invocationMapStack)
	v.invocationByNode = v.invocationMapStack[stackSize-1]
	v.invocationMapStack = v.invocationMapStack[:stackSize-1]
	return nil
}

// linkPropsForInvocation links property data sources for a partial
// invocation by walking the invocation node's children and matching
// expressions against the partial's prop map.
//
// Takes ctx (context.Context) which carries the session logger.
// Takes invocationNode (*ast_domain.TemplateNode) which is the template
// node representing the partial invocation.
// Takes partialInfo (*ast_domain.PartialInvocationInfo) which describes the
// partial being invoked and its passed props.
// Takes invocationMap
// (map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo)
// which maps template nodes to their invocation info.
func (v *pdsLinkerVisitor) linkPropsForInvocation(
	ctx context.Context,
	invocationNode *ast_domain.TemplateNode,
	partialInfo *ast_domain.PartialInvocationInfo,
	invocationMap map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo,
) {
	propMapForPartial := v.getPropMapForComponent(ctx, partialInfo.PartialPackageName)
	if propMapForPartial == nil {
		return
	}

	invocationNode.Walk(func(node *ast_domain.TemplateNode) bool {
		ann := node.GoAnnotations
		if ann != nil && ann.OriginalPackageAlias != nil && *ann.OriginalPackageAlias == partialInfo.PartialPackageName {
			visitNodeExpressions(node, func(expression ast_domain.Expression) {
				v.linkDataSourceForExpression(ctx, expression, partialInfo, propMapForPartial, invocationMap)
			})
		}
		return true
	})
}

// linkDataSourceForExpression resolves and links data sources for property
// usages within an expression tree.
//
// Takes ctx (context.Context) which carries the session logger.
// Takes rootExpr (ast_domain.Expression) which is the expression to traverse.
// Takes activePInfo (*ast_domain.PartialInvocationInfo) which provides the
// current partial invocation context.
// Takes propMapForExpr (map[string]string) which maps property names to their
// source identifiers.
// Takes invocationMap (map[*ast_domain.TemplateNode]
// *ast_domain.PartialInvocationInfo) which maps template nodes to their
// partial invocation info.
func (v *pdsLinkerVisitor) linkDataSourceForExpression(
	ctx context.Context,
	rootExpr ast_domain.Expression,
	activePInfo *ast_domain.PartialInvocationInfo,
	propMapForExpr map[string]string,
	invocationMap map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo,
) {
	visitExpression(rootExpr, func(currentExpr ast_domain.Expression) bool {
		if !isPropUsage(currentExpr) {
			return true
		}

		targetAnn := getAnnotationFromExpression(currentExpr)
		if targetAnn == nil {
			return true
		}

		ultimateDataSource := v.resolveTransitiveDataSource(ctx, currentExpr, activePInfo, propMapForExpr, invocationMap)

		if ultimateDataSource != nil {
			targetAnn.PropDataSource = ultimateDataSource.Clone()
			if ultimateDataSource.BaseCodeGenVarName != nil {
				_, l := logger_domain.From(ctx, log)
				l.Trace("[PDS-LINKER] Successfully linked prop to data source.",
					logger_domain.Int("depth", v.depth),
					logger_domain.String("expr", currentExpr.String()),
					logger_domain.String("source_var", *ultimateDataSource.BaseCodeGenVarName),
				)
			}
		}
		return true
	})
}

// resolveTransitiveDataSource finds the data source for an expression that
// references a property passed from a parent partial invocation.
//
// Takes ctx (context.Context) which carries the session logger.
// Takes currentExpr (ast_domain.Expression) which is the member expression
// being resolved.
// Takes activePInfo (*ast_domain.PartialInvocationInfo) which provides the
// current partial's invocation context.
// Takes propMapForExpr (map[string]string) which maps Go field names to
// HTML property names.
// Takes invocationMap (map[*ast_domain.TemplateNode]*
// ast_domain.PartialInvocationInfo) which maps template nodes to their
// invocation context.
//
// Returns *ast_domain.PropDataSource which is the resolved data source, or
// nil if the expression does not reference a passed property.
func (v *pdsLinkerVisitor) resolveTransitiveDataSource(
	ctx context.Context,
	currentExpr ast_domain.Expression,
	activePInfo *ast_domain.PartialInvocationInfo,
	propMapForExpr map[string]string,
	invocationMap map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo,
) *ast_domain.PropDataSource {
	goFieldName := getPropertyNameFromMemberExpr(currentExpr)
	htmlPropName, hasProp := propMapForExpr[goFieldName]
	if !hasProp {
		return nil
	}

	passedProp, exists := activePInfo.PassedProps[htmlPropName]
	if !exists {
		return nil
	}

	return v.followDataSourceChain(ctx, passedProp.Expression, activePInfo, invocationMap)
}

// followDataSourceChain follows a chain of prop data sources through
// partial invocations to find the ultimate data source. Stops when
// the expression is no longer a prop usage or when a transform is
// detected.
//
// Takes ctx (context.Context) which carries the session logger.
// Takes sourceExpression (ast_domain.Expression) which is the starting
// expression to trace.
// Takes sourcePartialInfo (*ast_domain.PartialInvocationInfo) which is the
// invocation info for the current partial.
// Takes invocationMap
// (map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo)
// which maps template nodes to their invocation info.
//
// Returns *ast_domain.PropDataSource which is the ultimate data
// source, or nil if the chain cannot be followed.
func (v *pdsLinkerVisitor) followDataSourceChain(
	ctx context.Context,
	sourceExpression ast_domain.Expression,
	sourcePartialInfo *ast_domain.PartialInvocationInfo,
	invocationMap map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo,
) *ast_domain.PropDataSource {
	currentSourceExpr := sourceExpression
	currentPInfo := sourcePartialInfo

	for isPropUsage(currentSourceExpr) {
		sourceAnn := getAnnotationFromExpression(currentSourceExpr)
		if sourceAnn == nil || sourceAnn.PropDataSource == nil {
			return nil
		}

		if v.didPropTransform(ctx, currentPInfo.PartialPackageName, currentSourceExpr) {
			return sourceAnn.PropDataSource
		}

		nextExpr, nextPInfo, ok := v.findNextLinkInChain(ctx, currentSourceExpr, currentPInfo, invocationMap)
		if !ok {
			return sourceAnn.PropDataSource
		}

		currentSourceExpr = nextExpr
		currentPInfo = nextPInfo
	}

	finalSourceAnn := getAnnotationFromExpression(currentSourceExpr)
	if finalSourceAnn == nil {
		return nil
	}
	return finalSourceAnn.PropDataSource
}

// didPropTransform checks if a prop was changed by coercion, defaults, or a
// factory at its component boundary. This ends the data source chain.
//
// Takes ctx (context.Context) which carries the session logger.
// Takes componentHash (string) which identifies the component to check.
// Takes propUsageExpr (ast_domain.Expression) which is the property usage to
// check.
//
// Returns bool which is true if the prop was changed at this boundary.
func (v *pdsLinkerVisitor) didPropTransform(ctx context.Context, componentHash string, propUsageExpr ast_domain.Expression) bool {
	validProps, err := v.getValidPropsForComponent(ctx, componentHash)
	if err != nil {
		_, errL := logger_domain.From(ctx, log)
		errL.Error("[PDS-LINKER] Could not get valid props to check for transformation",
			logger_domain.String("hash", componentHash),
			logger_domain.Error(err),
		)
		return false
	}

	propMap := v.getPropMapForComponent(ctx, componentHash)
	goFieldName := getPropertyNameFromMemberExpr(propUsageExpr)
	htmlPropName, ok := propMap[goFieldName]
	if !ok {
		return false
	}

	propInfo, ok := validProps[htmlPropName]
	if !ok {
		return false
	}

	return propInfo.ShouldCoerce || propInfo.DefaultValue != nil || propInfo.FactoryFuncName != ""
}

// findNextLinkInChain finds the next expression and invocation info in
// a prop data source chain by looking up the invoker node and mapping
// the Go field name to the HTML prop name in the next invocation.
//
// Takes ctx (context.Context) which carries the session logger.
// Takes currentPropExpr (ast_domain.Expression) which is the current
// prop expression being traced.
// Takes currentPInfo (*ast_domain.PartialInvocationInfo) which is the
// current invocation context.
// Takes invocationMap
// (map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo)
// which maps template nodes to their invocation info.
//
// Returns ast_domain.Expression which is the next expression in
// the chain.
// Returns *ast_domain.PartialInvocationInfo which is the next
// invocation context.
// Returns bool which is false if the chain cannot be extended.
func (v *pdsLinkerVisitor) findNextLinkInChain(
	ctx context.Context,
	currentPropExpr ast_domain.Expression,
	currentPInfo *ast_domain.PartialInvocationInfo,
	invocationMap map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo,
) (ast_domain.Expression, *ast_domain.PartialInvocationInfo, bool) {
	invokerNode := findInvokerNodeForComponentHash(currentPInfo.InvokerPackageAlias, invocationMap)
	if invokerNode == nil {
		return nil, nil, false
	}

	nextPInfo, hasNextPInfo := invocationMap[invokerNode]
	if !hasNextPInfo {
		return nil, nil, false
	}

	goFieldName := getPropertyNameFromMemberExpr(currentPropExpr)
	propMap := v.getPropMapForComponent(ctx, currentPInfo.InvokerPackageAlias)
	htmlPropName, hasHTMLProp := propMap[goFieldName]
	if !hasHTMLProp {
		return nil, nil, false
	}

	nextPassedProp, hasNextPassedProp := nextPInfo.PassedProps[htmlPropName]
	if !hasNextPassedProp {
		return nil, nil, false
	}

	return nextPassedProp.Expression, nextPInfo, true
}

// getValidPropsForComponent retrieves the valid properties for a component,
// caching results to avoid repeated lookups.
//
// Takes ctx (context.Context) which carries the session logger.
// Takes hashedName (string) which identifies the component by its hash.
//
// Returns map[string]validPropInfo which contains the valid properties for
// the component.
// Returns error when the component cannot be found in the virtual module.
func (v *pdsLinkerVisitor) getValidPropsForComponent(ctx context.Context, hashedName string) (map[string]validPropInfo, error) {
	if v.validPropInfoCache == nil {
		v.validPropInfoCache = make(map[string]map[string]validPropInfo)
	}
	if props, exists := v.validPropInfoCache[hashedName]; exists {
		return props, nil
	}

	partialComp, ok := v.virtualModule.ComponentsByHash[hashedName]
	if !ok {
		_, errL := logger_domain.From(ctx, log)
		errL.Error("[PDS-LINKER] Internal error: could not find component in virtual module during prop map creation",
			logger_domain.String("hashed_name", hashedName),
		)
		return nil, errors.New("could not find component in virtual module during prop map creation")
	}

	dummyCtx := NewRootAnalysisContext(
		new([]*ast_domain.Diagnostic),
		partialComp.CanonicalGoPackagePath,
		partialComp.RewrittenScriptAST.Name.Name,
		partialComp.VirtualGoFilePath,
		partialComp.Source.SourcePath,
	)

	validProps, err := getValidPropsForComponent(partialComp, v.typeResolver.inspector, dummyCtx)
	if err != nil {
		v.validPropInfoCache[hashedName] = nil
		return nil, fmt.Errorf("getting valid props for component %q: %w", hashedName, err)
	}

	v.validPropInfoCache[hashedName] = validProps
	return validProps, nil
}

// getPropMapForComponent retrieves or builds the Go field to prop name map.
//
// Takes ctx (context.Context) which carries the session logger.
// Takes hashedName (string) which identifies the component by its hash.
//
// Returns map[string]string which maps Go struct field names to prop names,
// or nil when the component's valid props cannot be determined.
func (v *pdsLinkerVisitor) getPropMapForComponent(ctx context.Context, hashedName string) map[string]string {
	if props, exists := v.propMapsByHash[hashedName]; exists {
		return props
	}

	validProps, err := v.getValidPropsForComponent(ctx, hashedName)
	if err != nil {
		return nil
	}

	goFieldToPropName := make(map[string]string, len(validProps))
	for propName, propInfo := range validProps {
		goFieldToPropName[propInfo.GoFieldName] = propName
	}
	v.propMapsByHash[hashedName] = goFieldToPropName
	return goFieldToPropName
}

// LinkAllPropDataSources links all PropDataSource annotations in an annotated
// AST. It runs two passes to trace data flow from each use back to its origin.
//
// Takes ctx (context.Context) which carries the session logger.
// Takes tree (*ast_domain.TemplateAST) which is the annotated AST to process.
// Takes virtualModule (*annotator_dto.VirtualModule) which provides module
// context for the linking process.
// Takes typeResolver (*TypeResolver) which resolves types during linking.
//
// Returns []*ast_domain.Diagnostic which contains any problems found during
// the linking passes, or nil if tree is nil.
func LinkAllPropDataSources(ctx context.Context, tree *ast_domain.TemplateAST, virtualModule *annotator_dto.VirtualModule, typeResolver *TypeResolver) []*ast_domain.Diagnostic {
	ctx, l := logger_domain.From(ctx, log)
	if tree == nil {
		return nil
	}
	l.Internal("--- [PDS-LINKER START] --- Starting Prop Data Source Linking Pass ---")
	var diagnostics []*ast_domain.Diagnostic

	l.Internal("[PDS-LINKER] Starting Pass 1: Setter (Post-Order)")
	setterVisitor := &pdsSetterVisitor{
		diagnostics:   &diagnostics,
		virtualModule: virtualModule,
		typeResolver:  typeResolver,
		depth:         0,
	}
	if err := tree.Accept(ctx, setterVisitor); err != nil {
		l.Error("[PDS-LINKER] AST traversal with Setter visitor failed unexpectedly.", logger_domain.Error(err))
	}

	l.Internal("[PDS-LINKER] Starting Pass 2: Linker (Pre-Order)")
	linkerVisitor := &pdsLinkerVisitor{
		diagnostics:        &diagnostics,
		virtualModule:      virtualModule,
		typeResolver:       typeResolver,
		propMapsByHash:     make(map[string]map[string]string),
		validPropInfoCache: nil,
		invocationByNode:   make(map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo),
		invocationMapStack: nil,
		depth:              0,
	}
	if err := tree.Accept(ctx, linkerVisitor); err != nil {
		l.Error("[PDS-LINKER] AST traversal with Linker visitor failed unexpectedly.", logger_domain.Error(err))
	}

	l.Internal("--- [PDS-LINKER END] --- Finished Prop Data Source Linking Pass ---", logger_domain.Int("diagnostics_found", len(diagnostics)))
	return diagnostics
}

// findInvokerNodeForComponentHash searches the invocation map for the
// template node whose partial package name matches the target hash.
//
// Takes targetHash (string) which is the hashed component name to find.
// Takes invocationMap
// (map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo)
// which maps template nodes to their invocation info.
//
// Returns *ast_domain.TemplateNode which is the matching node, or
// nil if no match is found.
func findInvokerNodeForComponentHash(targetHash string, invocationMap map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo) *ast_domain.TemplateNode {
	for node, pInfo := range invocationMap {
		if pInfo.PartialPackageName == targetHash {
			return node
		}
	}
	return nil
}

// visitNodeExpressions visits all expressions within a template node.
//
// Takes node (*ast_domain.TemplateNode) which is the node to visit.
// Takes visit (func(...)) which is called for each expression found.
func visitNodeExpressions(node *ast_domain.TemplateNode, visit func(ast_domain.Expression)) {
	visitStructuralDirectives(node, visit)
	visitContentDirectives(node, visit)
	visitBindingDirectives(node, visit)
	visitAttributeLikeExpressions(node, visit)
}

// visitStructuralDirectives calls the visit function for each structural
// directive expression in the given template node.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
// Takes visit (func(...)) which is called for each directive expression found.
func visitStructuralDirectives(node *ast_domain.TemplateNode, visit func(ast_domain.Expression)) {
	if node.DirIf != nil && node.DirIf.Expression != nil {
		visit(node.DirIf.Expression)
	}
	if node.DirElseIf != nil && node.DirElseIf.Expression != nil {
		visit(node.DirElseIf.Expression)
	}
	if node.DirFor != nil && node.DirFor.Expression != nil {
		visit(node.DirFor.Expression)
	}
	if node.DirShow != nil && node.DirShow.Expression != nil {
		visit(node.DirShow.Expression)
	}
}

// visitContentDirectives walks through content directives in a template node
// and calls the visit function for each expression found.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
// Takes visit (func(...)) which is called for each content expression found.
func visitContentDirectives(node *ast_domain.TemplateNode, visit func(ast_domain.Expression)) {
	if node.DirText != nil && node.DirText.Expression != nil {
		visit(node.DirText.Expression)
	}
	if node.DirHTML != nil && node.DirHTML.Expression != nil {
		visit(node.DirHTML.Expression)
	}
}

// visitBindingDirectives processes all binding directives on a template node.
//
// Takes node (*TemplateNode) which is the template node to process.
// Takes visit (func(...)) which is called for each binding expression found.
func visitBindingDirectives(node *ast_domain.TemplateNode, visit func(ast_domain.Expression)) {
	visitModelDirective(node, visit)
	visitBindDirectives(node, visit)
	visitEventDirectives(node, visit)
	visitCustomEventDirectives(node, visit)
}

// visitModelDirective checks a template node for a p-model directive and calls
// the visit function with its expression if one is found.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
// Takes visit (func(...)) which is called with the model expression if found.
func visitModelDirective(node *ast_domain.TemplateNode, visit func(ast_domain.Expression)) {
	if node.DirModel != nil && node.DirModel.Expression != nil {
		visit(node.DirModel.Expression)
	}
}

// visitBindDirectives processes all p-bind directives on a template node.
//
// Takes node (*ast_domain.TemplateNode) which contains the bind directives.
// Takes visit (func(...)) which is called for each bind expression that is
// not nil.
func visitBindDirectives(node *ast_domain.TemplateNode, visit func(ast_domain.Expression)) {
	for _, bind := range node.Binds {
		if bind != nil && bind.Expression != nil {
			visit(bind.Expression)
		}
	}
}

// visitEventDirectives processes all standard event directives (p-on:*).
//
// Takes node (*ast_domain.TemplateNode) which holds the event bindings to
// process.
// Takes visit (func(...)) which is called for each expression found.
func visitEventDirectives(node *ast_domain.TemplateNode, visit func(ast_domain.Expression)) {
	for _, events := range node.OnEvents {
		for i := range events {
			if events[i].Expression != nil {
				visit(events[i].Expression)
			}
		}
	}
}

// visitCustomEventDirectives calls the visit function for each expression in
// the custom event directives on a node.
//
// Takes node (*ast_domain.TemplateNode) which holds the custom events to check.
// Takes visit (func(...)) which is called for each expression that is not nil.
func visitCustomEventDirectives(node *ast_domain.TemplateNode, visit func(ast_domain.Expression)) {
	for _, events := range node.CustomEvents {
		for i := range events {
			if events[i].Expression != nil {
				visit(events[i].Expression)
			}
		}
	}
}

// visitAttributeLikeExpressions walks through a template node and calls the
// visit function for each attribute-like expression it finds. These include
// class directives, style directives, dynamic attributes, and rich text.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to walk.
// Takes visit (func(...)) which is called for each expression found.
func visitAttributeLikeExpressions(node *ast_domain.TemplateNode, visit func(ast_domain.Expression)) {
	if node.DirClass != nil && node.DirClass.Expression != nil {
		visit(node.DirClass.Expression)
	}
	if node.DirStyle != nil && node.DirStyle.Expression != nil {
		visit(node.DirStyle.Expression)
	}
	for i := range node.DynamicAttributes {
		if node.DynamicAttributes[i].Expression != nil {
			visit(node.DynamicAttributes[i].Expression)
		}
	}
	for i := range node.RichText {
		if !node.RichText[i].IsLiteral && node.RichText[i].Expression != nil {
			visit(node.RichText[i].Expression)
		}
	}
}

// visitExpression walks an expression tree and calls the visitor function on
// each node.
//
// When expression is nil or the visitor returns false, the walk stops. The visitor
// function controls whether to continue into child nodes.
//
// Takes expression (ast_domain.Expression) which is the root expression to visit.
// Takes visitor (func(...)) which is called for each node and returns whether
// to continue.
func visitExpression(expression ast_domain.Expression, visitor func(ast_domain.Expression) bool) {
	if expression == nil || !visitor(expression) {
		return
	}
	switch n := expression.(type) {
	case *ast_domain.MemberExpression:
		visitExpression(n.Base, visitor)
		visitExpression(n.Property, visitor)
	case *ast_domain.IndexExpression:
		visitExpression(n.Base, visitor)
		visitExpression(n.Index, visitor)
	case *ast_domain.UnaryExpression:
		visitExpression(n.Right, visitor)
	case *ast_domain.BinaryExpression:
		visitExpression(n.Left, visitor)
		visitExpression(n.Right, visitor)
	case *ast_domain.CallExpression:
		visitExpression(n.Callee, visitor)
		for _, argument := range n.Args {
			visitExpression(argument, visitor)
		}
	default:
		visitCompositeLiteralExpression(n, visitor)
		visitControlFlowExpression(n, visitor)
	}
}

// visitCompositeLiteralExpression walks through a composite literal expression
// and calls the visitor function on each nested expression it finds.
//
// Takes expression (ast_domain.Expression) which is the composite
// literal to visit.
// Takes visitor (func(...)) which is called for each nested expression.
func visitCompositeLiteralExpression(expression ast_domain.Expression, visitor func(ast_domain.Expression) bool) {
	switch n := expression.(type) {
	case *ast_domain.TemplateLiteral:
		for _, part := range n.Parts {
			if !part.IsLiteral {
				visitExpression(part.Expression, visitor)
			}
		}
	case *ast_domain.ObjectLiteral:
		for _, value := range n.Pairs {
			visitExpression(value, visitor)
		}
	case *ast_domain.ArrayLiteral:
		for _, element := range n.Elements {
			visitExpression(element, visitor)
		}
	}
}

// visitControlFlowExpression walks through a control flow expression and calls
// the visitor function on each child expression.
//
// Takes expression (ast_domain.Expression) which is the expression to walk.
// Takes visitor (func(...)) which is called for each child expression.
func visitControlFlowExpression(expression ast_domain.Expression, visitor func(ast_domain.Expression) bool) {
	if n, ok := expression.(*ast_domain.TernaryExpression); ok {
		visitExpression(n.Condition, visitor)
		visitExpression(n.Consequent, visitor)
		visitExpression(n.Alternate, visitor)
	}
}

// getRootIdentifier finds the base identifier of an expression by walking
// through member access, index, and call expressions.
//
// Takes expression (ast_domain.Expression) which is the expression to examine.
//
// Returns *ast_domain.Identifier which is the root identifier found.
// Returns bool which is true when a root identifier was found.
func getRootIdentifier(expression ast_domain.Expression) (*ast_domain.Identifier, bool) {
	current := expression
	for {
		switch n := current.(type) {
		case *ast_domain.Identifier:
			return n, true
		case *ast_domain.MemberExpression:
			current = n.Base
		case *ast_domain.IndexExpression:
			current = n.Base
		case *ast_domain.CallExpression:
			current = n.Callee
		default:
			return nil, false
		}
	}
}

// getPropertyNameFromMemberExpr returns the property name from a member
// expression when the property is an identifier.
//
// Takes expression (ast_domain.Expression) which is the expression to check.
//
// Returns string which is the property name, or an empty string if the
// expression is not a member expression or the property is not an identifier.
func getPropertyNameFromMemberExpr(expression ast_domain.Expression) string {
	if mem, ok := expression.(*ast_domain.MemberExpression); ok {
		if prop, isIdent := mem.Property.(*ast_domain.Identifier); isIdent {
			return prop.Name
		}
	}
	return ""
}

// isPropUsage reports whether the expression refers to props.
//
// Takes expression (ast_domain.Expression) which is the expression to check.
//
// Returns bool which is true if the root identifier is named "props".
func isPropUsage(expression ast_domain.Expression) bool {
	identifier, ok := getRootIdentifier(expression)
	return ok && identifier.Name == "props"
}

// isPartialInvocationNode checks whether the given node represents a partial
// template invocation.
//
// Takes node (*TemplateNode) which is the template node to check.
//
// Returns bool which is true if the node has partial invocation details.
func isPartialInvocationNode(node *ast_domain.TemplateNode) bool {
	return node != nil && node.GoAnnotations != nil && node.GoAnnotations.PartialInfo != nil
}
