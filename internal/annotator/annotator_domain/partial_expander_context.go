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

// Manages the expansion context during partial component inlining, tracking state and recursively expanding nested partials.
// Handles slot content, maintains invocation keys, and maintains proper scoping during the recursive expansion process.

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

const (
	// initialScopedCSSCapacity is the starting size for the scoped CSS blocks map.
	initialScopedCSSCapacity = 16

	// initialInvocationsCapacity is the initial size for the unique invocations map.
	initialInvocationsCapacity = 8

	// initialGlobalCSSCapacity is the starting size for the global CSS blocks map.
	initialGlobalCSSCapacity = 8

	// initialOrderCapacity is the initial slice capacity for order collections.
	initialOrderCapacity = 8

	// initialExpansionDiagnosticsCapacity is the starting slice capacity for
	// collecting diagnostics during template expansion.
	initialExpansionDiagnosticsCapacity = 4

	// initialPropsCapacity is the initial size for property maps.
	initialPropsCapacity = 8

	// initialOverridesCapacity is the starting capacity for the overrides map.
	initialOverridesCapacity = 4
)

// expansionContext holds the state needed for a single partial expansion run.
type expansionContext struct {
	// expander is the PartialExpander that owns this context.
	expander *PartialExpander

	// graph holds the component dependency graph used to find partial references.
	graph *annotator_dto.ComponentGraph

	// scopedCSSBlocks maps entry point hashed names to their scoped CSS content.
	scopedCSSBlocks map[string]string

	// uniqueInvocations maps invocation keys to partial invocations for deduplication.
	uniqueInvocations map[string]*annotator_dto.PartialInvocation

	// globalCSSBlocks maps source file paths to their processed CSS content.
	globalCSSBlocks map[string]string

	// invocationOrder tracks the order in which partials were invoked.
	invocationOrder []string

	// expansionPath tracks visited nodes to detect circular dependencies.
	expansionPath []string

	// diagnostics collects errors and warnings found during partial expansion.
	diagnostics []*ast_domain.Diagnostic
}

// expandPartialsRecursive expands partial template nodes within the given list.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes nodes ([]*ast_domain.TemplateNode) which is the list of nodes to
// expand.
// Takes invoker (*annotator_dto.ParsedComponent) which provides the import
// context for resolving partial paths.
//
// Returns []*ast_domain.TemplateNode which is the expanded nodes, or nil if
// the context is cancelled or errors occur.
func (ec *expansionContext) expandPartialsRecursive(ctx context.Context, nodes []*ast_domain.TemplateNode, invoker *annotator_dto.ParsedComponent) []*ast_domain.TemplateNode {
	var aliasToRawPath map[string]string
	if invoker != nil && len(invoker.PikoImports) > 0 {
		aliasToRawPath = make(map[string]string, len(invoker.PikoImports))
		for _, imp := range invoker.PikoImports {
			aliasToRawPath[imp.Alias] = imp.Path
		}
	} else {
		aliasToRawPath = make(map[string]string)
	}

	finalNodes := make([]*ast_domain.TemplateNode, 0, len(nodes))
	for _, node := range nodes {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if ast_domain.HasErrors(ec.diagnostics) {
			return nil
		}
		expanded := ec.expandNodeRecursive(ctx, node, aliasToRawPath, invoker)
		finalNodes = append(finalNodes, expanded...)
	}
	return finalNodes
}

// expandNodeRecursive expands a template node and all its children.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes node (*ast_domain.TemplateNode) which is the node to expand.
// Takes aliasToRawPath (map[string]string) which maps import aliases to their
// file paths.
// Takes invoker (*annotator_dto.ParsedComponent) which is the component that
// started this expansion.
//
// Returns []*ast_domain.TemplateNode which contains the expanded nodes, or nil
// if the context is cancelled or errors occur.
func (ec *expansionContext) expandNodeRecursive(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	aliasToRawPath map[string]string,
	invoker *annotator_dto.ParsedComponent,
) []*ast_domain.TemplateNode {
	if diagnostic := validatePikoElementNode(node, invoker.SourcePath); diagnostic != nil {
		ec.diagnostics = append(ec.diagnostics, diagnostic)
	}
	resolvePikoElementStaticIs(node)

	if diagnostic := validatePikoPartialElement(node, invoker.SourcePath); diagnostic != nil {
		ec.diagnostics = append(ec.diagnostics, diagnostic)
	}

	if alias, isPartial := getPartialImportAliasFromIsAttr(node); isPartial {
		return ec.expandPartial(ctx, node, alias, aliasToRawPath, invoker)
	}

	newChildren := make([]*ast_domain.TemplateNode, 0, len(node.Children))
	for _, child := range node.Children {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if ast_domain.HasErrors(ec.diagnostics) {
			return nil
		}
		expandedChildren := ec.expandNodeRecursive(ctx, child, aliasToRawPath, invoker)
		newChildren = append(newChildren, expandedChildren...)
	}
	node.Children = newChildren
	return []*ast_domain.TemplateNode{node}
}

// expandPartial creates and runs a task to expand a single partial template.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes invokerNode (*ast_domain.TemplateNode) which is the template node that
// calls the partial.
// Takes userAlias (string) which is the alias used to refer to the partial.
// Takes aliasToRawPath (map[string]string) which maps aliases to their raw
// file paths.
// Takes invokerComponent (*annotator_dto.ParsedComponent) which is the
// component that contains the calling node.
//
// Returns []*ast_domain.TemplateNode which contains the expanded template
// nodes.
func (ec *expansionContext) expandPartial(
	ctx context.Context,
	invokerNode *ast_domain.TemplateNode,
	userAlias string,
	aliasToRawPath map[string]string,
	invokerComponent *annotator_dto.ParsedComponent,
) []*ast_domain.TemplateNode {
	task := getPartialExpansionTask(ec, invokerNode, invokerComponent, aliasToRawPath, userAlias)
	result := task.process(ctx)
	putPartialExpansionTask(task)
	return result
}

// assembleFinalCSS joins global and scoped CSS blocks into a single string.
// Blocks are sorted by path or scope ID to produce the same output each time.
//
// Returns string which contains the joined CSS content.
// Returns error which is always nil (kept for future use).
func (ec *expansionContext) assembleFinalCSS() (string, error) {
	var finalCSS strings.Builder

	globalPaths := slices.Sorted(maps.Keys(ec.globalCSSBlocks))

	for _, path := range globalPaths {
		css := ec.globalCSSBlocks[path]
		if css != "" {
			finalCSS.WriteString(css)
			finalCSS.WriteString("\n")
		}
	}

	sortedScopeIDs := slices.Sorted(maps.Keys(ec.scopedCSSBlocks))

	for _, scopeID := range sortedScopeIDs {
		css := ec.scopedCSSBlocks[scopeID]
		if css != "" {
			finalCSS.WriteString(css)
		}
	}

	return strings.TrimSpace(finalCSS.String()), nil
}

// getSortedPartialInvocations returns the unique invocations sorted by key.
//
// Returns []*annotator_dto.PartialInvocation which contains the invocations
// in alphabetical key order.
func (ec *expansionContext) getSortedPartialInvocations() []*annotator_dto.PartialInvocation {
	sortedKeys := slices.Sorted(maps.Keys(ec.uniqueInvocations))

	sorted := make([]*annotator_dto.PartialInvocation, len(sortedKeys))
	for i, key := range sortedKeys {
		sorted[i] = ec.uniqueInvocations[key]
	}
	return sorted
}

// createPartialInfoAnnotation builds a partial invocation info record for a
// template node.
//
// Takes ctx (context.Context) which controls cancellation and deadlines for
// expression parsing.
// Takes invokerNode (*ast_domain.TemplateNode) which is the node that calls
// the partial.
// Takes userAlias (string) which is the alias used in the template.
// Takes targetHash (string) which identifies the target partial.
// Takes invokerHash (string) which identifies the calling component.
//
// Returns *ast_domain.PartialInvocationInfo which holds the invocation data.
func (ec *expansionContext) createPartialInfoAnnotation(ctx context.Context, invokerNode *ast_domain.TemplateNode, userAlias, targetHash, invokerHash string) *ast_domain.PartialInvocationInfo {
	sourcePath := ""
	if invokerComp, ok := ec.graph.Components[invokerHash]; ok && invokerComp != nil {
		sourcePath = invokerComp.SourcePath
	}

	requestOverrides, passedProps := extractPropsForLinking(ctx, invokerNode, sourcePath)
	invocationKey := calculatePotentialInvocationKey(userAlias, requestOverrides, passedProps)

	partialInfo := &ast_domain.PartialInvocationInfo{
		InvocationKey:        invocationKey,
		PartialAlias:         userAlias,
		PartialPackageName:   targetHash,
		RequestOverrides:     requestOverrides,
		PassedProps:          passedProps,
		InvokerPackageAlias:  invokerHash,
		Location:             invokerNode.Location,
		InvokerInvocationKey: "",
	}

	if _, exists := ec.uniqueInvocations[invocationKey]; !exists {
		ec.uniqueInvocations[invocationKey] = &annotator_dto.PartialInvocation{
			InvocationKey:        invocationKey,
			PartialAlias:         userAlias,
			PartialHashedName:    targetHash,
			PassedProps:          passedProps,
			RequestOverrides:     requestOverrides,
			InvokerHashedName:    invokerHash,
			InvokerInvocationKey: "",
			DependsOn:            nil,
			Location:             invokerNode.Location,
		}
		ec.invocationOrder = append(ec.invocationOrder, invocationKey)
	}
	return partialInfo
}

// rejectedPikoElementTargets lists tag names that cannot be used as the target
// of a <piko:element is="..."> resolution.
var rejectedPikoElementTargets = map[string]bool{
	tagPikoPartial: true,
	tagPikoSlot:    true,
	tagPikoElement: true,
}

// newExpansionContext creates a new context for template expansion.
//
// Takes exp (*PartialExpander) which expands partial templates.
// Takes graph (*annotator_dto.ComponentGraph) which holds the component
// dependency graph.
// Takes entryPointHashedName (string) which is the hashed name of the main
// component to expand.
//
// Returns *expansionContext which is the prepared context for expansion.
// Returns *annotator_dto.ParsedComponent which is the main component.
// Returns *ast_domain.TemplateAST which is a cloned AST ready for expansion.
// Returns error when the entry point component is not found in the graph.
func newExpansionContext(
	exp *PartialExpander,
	graph *annotator_dto.ComponentGraph,
	entryPointHashedName string,
) (*expansionContext, *annotator_dto.ParsedComponent, *ast_domain.TemplateAST, error) {
	mainComponent, ok := graph.Components[entryPointHashedName]
	if !ok {
		return nil, nil, nil, fmt.Errorf("entry point component with hash '%s' not found in component graph", entryPointHashedName)
	}
	if mainComponent.Template == nil {
		return nil, mainComponent, nil, nil
	}

	expCtx := &expansionContext{
		expander:          exp,
		graph:             graph,
		scopedCSSBlocks:   make(map[string]string, initialScopedCSSCapacity),
		uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation, initialInvocationsCapacity),
		globalCSSBlocks:   make(map[string]string, initialGlobalCSSCapacity),
		invocationOrder:   make([]string, 0, initialOrderCapacity),
		expansionPath:     []string{mainComponent.SourcePath},
		diagnostics:       make([]*ast_domain.Diagnostic, 0, initialExpansionDiagnosticsCapacity),
	}

	astToExpand := mainComponent.Template.DeepClone()
	stampNodesWithPackage(astToExpand.RootNodes, entryPointHashedName, mainComponent.SourcePath)

	return expCtx, mainComponent, astToExpand, nil
}

// getPartialImportAliasFromIsAttr gets the partial import alias from the "is"
// attribute of a piko:partial element node. Only piko:partial elements can
// trigger partial invocation - the "is" attribute on other elements is treated
// as a normal HTML attribute.
//
// Takes node (*TemplateNode) which is the template node to check.
//
// Returns alias (string) which holds the value of the "is" attribute.
// Returns isPartial (bool) which is true when a valid alias was found on a
// piko:partial element.
func getPartialImportAliasFromIsAttr(node *ast_domain.TemplateNode) (alias string, isPartial bool) {
	if node == nil || node.NodeType != ast_domain.NodeElement {
		return "", false
	}
	if node.TagName != tagPikoPartial {
		return "", false
	}
	value, ok := node.GetAttribute(attributeIs)
	return value, ok && value != ""
}

// validatePikoPartialElement checks that a piko:partial element has a valid
// 'is' attribute.
//
// It validates that the 'is' attribute is present and not dynamic. The 'is'
// attribute on other elements is treated as a normal HTML attribute and does
// not produce errors.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to validate.
// Takes sourcePath (string) which is the file path for diagnostics.
//
// Returns *ast_domain.Diagnostic which is an error if validation fails, or nil
// if the node is valid or is not a piko:partial element.
func validatePikoPartialElement(node *ast_domain.TemplateNode, sourcePath string) *ast_domain.Diagnostic {
	if node == nil || node.NodeType != ast_domain.NodeElement {
		return nil
	}

	if node.TagName != tagPikoPartial {
		return nil
	}

	for i := range node.DynamicAttributes {
		attr := &node.DynamicAttributes[i]
		if strings.EqualFold(attr.Name, attributeIs) {
			return ast_domain.NewDiagnosticWithCode(
				ast_domain.Error,
				"The 'is' attribute on <piko:partial> cannot be dynamic. Use a static string value.",
				":is",
				annotator_dto.CodeInvalidPartialAttribute,
				attr.Location,
				sourcePath,
			)
		}
	}

	value, ok := node.GetAttribute(attributeIs)
	if !ok || value == "" {
		return ast_domain.NewDiagnosticWithCode(
			ast_domain.Error,
			"The <piko:partial> element requires an 'is' attribute specifying the partial alias",
			"piko:partial",
			annotator_dto.CodeMissingPartialAttribute,
			node.Location,
			sourcePath,
		)
	}

	return nil
}

// validatePikoElementNode checks that a <piko:element> has a valid 'is'
// attribute.
//
// Behaviour:
//   - Returns a diagnostic if the attribute is missing (and no dynamic :is is
//     present), empty, or targets a rejected tag name.
//   - Dynamic :is is allowed and produces no diagnostic here; runtime
//     validation handles it.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to validate.
// Takes sourcePath (string) which is the file path for diagnostics.
//
// Returns *ast_domain.Diagnostic which is an error if validation fails, or nil
// if the node is valid or is not a piko:element.
func validatePikoElementNode(node *ast_domain.TemplateNode, sourcePath string) *ast_domain.Diagnostic {
	if node == nil || node.NodeType != ast_domain.NodeElement {
		return nil
	}
	if node.TagName != tagPikoElement {
		return nil
	}

	for i := range node.DynamicAttributes {
		attr := &node.DynamicAttributes[i]
		if strings.EqualFold(attr.Name, attributeIs) {
			return nil
		}
	}

	value, ok := node.GetAttribute(attributeIs)
	if !ok || value == "" {
		return ast_domain.NewDiagnosticWithCode(
			ast_domain.Error,
			"The <piko:element> element requires an 'is' attribute specifying the target tag name",
			"piko:element",
			annotator_dto.CodeMissingPartialAttribute,
			node.Location,
			sourcePath,
		)
	}

	if rejectedPikoElementTargets[value] {
		return ast_domain.NewDiagnosticWithCode(
			ast_domain.Error,
			fmt.Sprintf("The <piko:element> element cannot target '%s'", value),
			value,
			annotator_dto.CodeInvalidPartialAttribute,
			node.Location,
			sourcePath,
		)
	}

	return nil
}

// resolvePikoElementStaticIs rewrites a <piko:element is="tag"> node by
// setting its TagName to the is value and removing the is attribute. Dynamic
// :is nodes are left as piko:element for runtime resolution.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check and
// potentially rewrite.
//
// Returns bool which is true if the node was resolved (tag name changed).
func resolvePikoElementStaticIs(node *ast_domain.TemplateNode) bool {
	if node == nil || node.NodeType != ast_domain.NodeElement {
		return false
	}
	if node.TagName != tagPikoElement {
		return false
	}

	for i := range node.DynamicAttributes {
		if strings.EqualFold(node.DynamicAttributes[i].Name, attributeIs) {
			return false
		}
	}

	value, ok := node.GetAttribute(attributeIs)
	if !ok || value == "" || rejectedPikoElementTargets[value] {
		return false
	}

	node.TagName = value
	node.RemoveAttribute(attributeIs)
	return true
}

// calculatePotentialInvocationKey builds a unique cache key from a partial
// alias and its property maps.
//
// Takes partialAlias (string) which is the base part of the key.
// Takes reqOverrides (map[string]ast_domain.PropValue) which holds request
// override properties to include in the key.
// Takes passedProps (map[string]ast_domain.PropValue) which holds passed
// properties to include in the key.
//
// Returns string which is a stable key built from all inputs.
func calculatePotentialInvocationKey(partialAlias string, reqOverrides, passedProps map[string]ast_domain.PropValue) string {
	var builder strings.Builder
	builder.WriteString(partialAlias)
	appendMap := func(m map[string]ast_domain.PropValue, prefix string) {
		keys := slices.Sorted(maps.Keys(m))
		for _, k := range keys {
			prop := m[k]
			expressionString := ""
			if prop.Expression != nil {
				expressionString = prop.Expression.String()
			}
			builder.WriteString(prefix + k + "=" + expressionString)
		}
	}
	appendMap(reqOverrides, ":req.")
	appendMap(passedProps, ":")
	return buildAliasFromPath(builder.String())
}

// extractPropsForLinking extracts properties from a template node for use in
// component linking.
//
// Takes ctx (context.Context) which controls cancellation and deadlines for
// expression parsing.
// Takes node (*ast_domain.TemplateNode) which holds the template attributes to
// process.
// Takes sourcePath (string) which gives the source file path for expression
// parsing.
//
// Returns requestOverrides (map[string]ast_domain.PropValue) which holds
// attributes that have the request prefix.
// Returns passedProps (map[string]ast_domain.PropValue) which holds all other
// non-structural attributes as properties with lowercase keys.
func extractPropsForLinking(ctx context.Context, node *ast_domain.TemplateNode, sourcePath string) (requestOverrides, passedProps map[string]ast_domain.PropValue) {
	requestOverrides = make(map[string]ast_domain.PropValue, initialOverridesCapacity)
	passedProps = make(map[string]ast_domain.PropValue, initialPropsCapacity)

	for i := range node.DynamicAttributes {
		attr := &node.DynamicAttributes[i]
		if fieldName, found := strings.CutPrefix(attr.Name, prefixRequest); found {
			requestOverrides[fieldName] = ast_domain.PropValue{
				Expression:        attr.Expression,
				InvokerAnnotation: nil,
				GoFieldName:       "",
				Location:          attr.Location,
				NameLocation:      attr.NameLocation,
				IsLoopDependent:   false,
			}
		} else {
			passedProps[strings.ToLower(attr.Name)] = ast_domain.PropValue{
				Expression:        attr.Expression,
				InvokerAnnotation: nil,
				GoFieldName:       "",
				Location:          attr.Location,
				NameLocation:      attr.NameLocation,
				IsLoopDependent:   false,
			}
		}
	}

	for i := range node.Attributes {
		attr := &node.Attributes[i]
		if isStructuralAttribute(attr.Name) {
			continue
		}
		propName := strings.ToLower(attr.Name)
		parsedExpr, _ := ast_domain.NewExpressionParser(ctx, strconv.Quote(attr.Value), sourcePath).ParseExpression(ctx)
		if parsedExpr != nil {
			passedProps[propName] = ast_domain.PropValue{
				Expression:        parsedExpr,
				InvokerAnnotation: nil,
				GoFieldName:       "",
				Location:          attr.Location,
				NameLocation:      attr.NameLocation,
				IsLoopDependent:   false,
			}
		}
	}

	return requestOverrides, passedProps
}

// isStructuralAttribute checks whether the given attribute name is a structural
// microformat attribute.
//
// Takes name (string) which is the attribute name to check.
//
// Returns bool which is true if name is "is", "class", or starts with "p-".
func isStructuralAttribute(name string) bool {
	lower := strings.ToLower(name)
	return lower == attributeIs || lower == attributeClass || strings.HasPrefix(lower, "p-")
}
