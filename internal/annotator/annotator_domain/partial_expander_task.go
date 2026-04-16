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

// Handles individual partial expansion tasks by processing a single partial
// invocation and its associated styles. Uses object pooling for efficiency and
// manages the expansion of one partial component at a time during the expansion
// phase.

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/sfcparser"
)

// partialExpansionTaskPool reuses partialExpansionTask instances
// to reduce allocation pressure.
var partialExpansionTaskPool = sync.Pool{
	New: func() any {
		return &partialExpansionTask{}
	},
}

// partialExpansionTask holds the state and logic for expanding a single
// partial.
type partialExpansionTask struct {
	// ec is the shared expansion context for this task.
	ec *expansionContext

	// invokerNode is the template node that triggered this partial expansion.
	invokerNode *ast_domain.TemplateNode

	// invokerComponent is the parsed component that started this expansion.
	invokerComponent *annotator_dto.ParsedComponent

	// aliasToRawPath maps import aliases to their raw file paths.
	aliasToRawPath map[string]string

	// loadedPartial holds the parsed partial template after it has been loaded.
	loadedPartial *annotator_dto.ParsedComponent

	// groupedSlotContent maps slot names to their content for filling the partial.
	groupedSlotContent map[string]invokerSlotContent

	// userAlias is the name used in the template to refer to a partial.
	userAlias string

	// targetHashedName is the hashed name of the partial to expand.
	targetHashedName string

	// expandedPartialBody holds the parsed template nodes after expanding a
	// partial.
	expandedPartialBody []*ast_domain.TemplateNode

	// hasError indicates whether an error occurred during expansion.
	hasError bool
}

// invokerSlotContent holds the nodes provided for a slot and its location for
// diagnostics.
type invokerSlotContent struct {
	// Nodes holds the template nodes that belong to this slot.
	Nodes []*ast_domain.TemplateNode

	// Location is the source position of the first node in the slot content.
	Location ast_domain.Location
}

// process runs all steps for a partialExpansionTask.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
//
// Returns []*ast_domain.TemplateNode which holds the expanded template nodes,
// or nil when an error happens during processing.
func (t *partialExpansionTask) process(ctx context.Context) []*ast_domain.TemplateNode {
	t.resolveAndLoadPartial(ctx)
	if t.hasError {
		return []*ast_domain.TemplateNode{t.invokerNode}
	}

	t.checkCircularDependencies()
	if t.hasError {
		return []*ast_domain.TemplateNode{t.invokerNode}
	}

	if t.loadedPartial.Template == nil {
		return nil
	}

	t.processCSS(ctx)

	if t.hasError {
		return nil
	}

	t.expandBodies(ctx)
	return t.assembleFinalNodes(ctx)
}

// resolveAndLoadPartial looks up the partial alias, resolves its import path,
// and loads the target component from the graph. Sets hasError to true if the
// alias is not defined, the path cannot be resolved, or the component is not
// found.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
func (t *partialExpansionTask) resolveAndLoadPartial(ctx context.Context) {
	if t.hasError {
		return
	}
	rawImportPath, ok := t.aliasToRawPath[t.userAlias]
	if !ok {
		diagnostic := ast_domain.NewDiagnosticWithCode(
			ast_domain.Error,
			fmt.Sprintf("Undefined partial alias '%s'", t.userAlias),
			t.userAlias,
			annotator_dto.CodeUndefinedPartialAlias,
			t.invokerNode.Location,
			t.invokerComponent.SourcePath,
		)
		t.ec.diagnostics = append(t.ec.diagnostics, diagnostic)
		t.hasError = true
		return
	}

	resolvedPath, err := t.ec.expander.resolver.ResolvePKPath(ctx, rawImportPath, t.invokerComponent.SourcePath)
	if err != nil {
		diagnostic := ast_domain.NewDiagnosticWithCode(
			ast_domain.Error,
			fmt.Sprintf("Could not resolve import path for partial '%s': %v", t.userAlias, err),
			rawImportPath,
			annotator_dto.CodeUnresolvedImport,
			t.invokerNode.Location,
			t.invokerComponent.SourcePath,
		)
		t.ec.diagnostics = append(t.ec.diagnostics, diagnostic)
		t.hasError = true
		return
	}

	t.targetHashedName, ok = t.ec.graph.PathToHashedName[resolvedPath]
	if !ok {
		_, taskL := logger_domain.From(ctx, log)
		taskL.Error("Internal consistency error: could not find hash for resolved import path in graph",
			logger_domain.String("path", resolvedPath))
		t.hasError = true
		return
	}

	t.loadedPartial, ok = t.ec.graph.Components[t.targetHashedName]
	if !ok || t.loadedPartial.Template == nil {
		diagnostic := ast_domain.NewDiagnosticWithCode(
			ast_domain.Error,
			fmt.Sprintf("Failed to load partial '%s'. The component file may be missing or does not contain a <template> block.", t.userAlias),
			rawImportPath,
			annotator_dto.CodePartialLoadError,
			t.invokerNode.Location,
			t.invokerComponent.SourcePath,
		)
		t.ec.diagnostics = append(t.ec.diagnostics, diagnostic)
		t.hasError = true
	}
}

// checkCircularDependencies checks whether the current partial expansion
// creates a cycle in the dependency chain.
func (t *partialExpansionTask) checkCircularDependencies() {
	if t.hasError {
		return
	}
	resolvedPath := t.loadedPartial.SourcePath
	if slices.Contains(t.ec.expansionPath, resolvedPath) {
		cyclePath := make([]string, len(t.ec.expansionPath)+1)
		copy(cyclePath, t.ec.expansionPath)
		cyclePath[len(t.ec.expansionPath)] = resolvedPath
		diagnostic := ast_domain.NewDiagnosticWithCode(
			ast_domain.Error,
			fmt.Sprintf("Circular dependency detected: %s", strings.Join(cyclePath, " -> ")),
			t.userAlias,
			annotator_dto.CodeCircularDependency,
			t.invokerNode.Location,
			t.invokerComponent.SourcePath,
		)
		t.ec.diagnostics = append(t.ec.diagnostics, diagnostic)
		t.hasError = true
	}
}

// processCSS gathers and scopes CSS from all style blocks in the partial.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
func (t *partialExpansionTask) processCSS(ctx context.Context) {
	if t.hasError {
		return
	}
	if _, seen := t.ec.scopedCSSBlocks[t.targetHashedName]; seen || len(t.loadedPartial.StyleBlocks) == 0 {
		return
	}

	var finalScopedCSS strings.Builder
	var hasScopedStyles bool

	for _, styleBlock := range t.loadedPartial.StyleBlocks {
		if t.hasError {
			return
		}
		scopedCSS := t.processSingleStyleBlock(ctx, styleBlock)
		if scopedCSS != "" {
			finalScopedCSS.WriteString(scopedCSS)
			finalScopedCSS.WriteString("\n")
			hasScopedStyles = true
		}
	}

	if hasScopedStyles {
		t.ec.scopedCSSBlocks[t.targetHashedName] = finalScopedCSS.String()
	}
}

// processSingleStyleBlock handles a single style block from the SFC.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes styleBlock (sfcparser.Style) which contains the style content and
// attributes to handle.
//
// Returns string which is the processed scoped style content, or an empty
// string for global styles or empty blocks.
func (t *partialExpansionTask) processSingleStyleBlock(ctx context.Context, styleBlock sfcparser.Style) string {
	styleContent := styleBlock.Content
	if strings.TrimSpace(styleContent) == "" {
		return ""
	}

	_, isGlobal := styleBlock.Attributes["global"]
	styleStartLocation := ast_domain.Location{Line: styleBlock.ContentLocation.Line, Column: styleBlock.ContentLocation.Column, Offset: 0}

	if isGlobal {
		t.processGlobalStyleBlock(ctx, styleContent, styleStartLocation)
		return ""
	}
	return t.processScopedStyleBlock(ctx, styleContent, styleStartLocation)
}

// processGlobalStyleBlock handles a global style block and stores the result.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes styleContent (string) which is the raw CSS content to process.
// Takes styleStartLocation (ast_domain.Location) which marks where the style
// block starts in the source file.
func (t *partialExpansionTask) processGlobalStyleBlock(ctx context.Context, styleContent string, styleStartLocation ast_domain.Location) {
	processedGlobalCSS, diagnostics, err := t.ec.expander.cssProcessor.Process(
		ctx,
		styleContent,
		t.loadedPartial.SourcePath,
		styleStartLocation,
		t.ec.expander.fsReader,
	)

	if t.handleCSSError(err, "Fatal error processing global CSS", "<style global>", styleStartLocation) {
		return
	}
	t.handleCSSDiagnostics(diagnostics)
	t.ec.globalCSSBlocks[t.loadedPartial.SourcePath] = processedGlobalCSS
}

// processScopedStyleBlock scopes CSS within a style block to the partial.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes styleContent (string) which is the raw CSS to process.
// Takes styleStartLocation (ast_domain.Location) which marks where the style
// block begins in the source.
//
// Returns string which is the scoped CSS, or empty string on error.
func (t *partialExpansionTask) processScopedStyleBlock(ctx context.Context, styleContent string, styleStartLocation ast_domain.Location) string {
	templateForScoping := t.loadedPartial.Template.DeepClone()

	diagnostics, err := t.ec.expander.cssProcessor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      templateForScoping,
		cssBlock:      &styleContent,
		scopeID:       t.targetHashedName,
		sourcePath:    t.loadedPartial.SourcePath,
		startLocation: styleStartLocation,
		fsReader:      t.ec.expander.fsReader,
	})

	if t.handleCSSError(err, "Fatal error scoping CSS", "<style> block", styleStartLocation) {
		return ""
	}
	t.handleCSSDiagnostics(diagnostics)
	return styleContent
}

// handleCSSError records a diagnostic if err is not nil.
//
// Takes err (error) which is the error to check.
// Takes msgPrefix (string) which is the prefix for the diagnostic message.
// Takes reference (string) which gives extra detail for the diagnostic.
// Takes location (ast_domain.Location) which shows where the error happened.
//
// Returns bool which is true if an error was recorded, false if not.
func (t *partialExpansionTask) handleCSSError(err error, msgPrefix, reference string, location ast_domain.Location) bool {
	if err == nil {
		return false
	}
	diagnostic := ast_domain.NewDiagnosticWithCode(ast_domain.Error,
		fmt.Sprintf("%s for '%s': %v", msgPrefix, t.loadedPartial.SourcePath, err),
		reference, annotator_dto.CodePartialCSSError, location, t.loadedPartial.SourcePath)
	t.ec.diagnostics = append(t.ec.diagnostics, diagnostic)
	t.hasError = true
	return true
}

// handleCSSDiagnostics adds CSS diagnostics to the expansion context.
//
// Takes diagnostics ([]*ast_domain.Diagnostic) which holds the diagnostics to add.
func (t *partialExpansionTask) handleCSSDiagnostics(diagnostics []*ast_domain.Diagnostic) {
	if len(diagnostics) == 0 {
		return
	}
	t.ec.diagnostics = append(t.ec.diagnostics, diagnostics...)
	if ast_domain.HasErrors(diagnostics) {
		t.hasError = true
	}
}

// expandBodies expands slot content and fills the partial template body.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
func (t *partialExpansionTask) expandBodies(ctx context.Context) {
	if t.hasError {
		return
	}

	var expandedSlotContent []*ast_domain.TemplateNode
	for _, child := range t.invokerNode.Children {
		expandedChildren := t.ec.expandNodeRecursive(ctx, child, t.aliasToRawPath, t.invokerComponent)
		expandedSlotContent = append(expandedSlotContent, expandedChildren...)
	}
	t.groupedSlotContent = groupContentBySlot(expandedSlotContent)

	partialBodyToExpand := t.loadedPartial.Template.DeepClone()
	stampNodesWithPackage(partialBodyToExpand.RootNodes, t.targetHashedName, t.loadedPartial.SourcePath)
	contentForFilling := make(map[string][]*ast_domain.TemplateNode)
	for name, content := range t.groupedSlotContent {
		contentForFilling[name] = content.Nodes
	}
	filledBody := fillSlotsInTree(partialBodyToExpand.RootNodes, contentForFilling)

	t.ec.expansionPath = append(t.ec.expansionPath, t.loadedPartial.SourcePath)
	defer func() {
		t.ec.expansionPath = t.ec.expansionPath[:len(t.ec.expansionPath)-1]
	}()
	t.expandedPartialBody = t.ec.expandPartialsRecursive(ctx, filledBody, t.loadedPartial)
}

// assembleFinalNodes checks slots, creates metadata, and converts the expanded
// partial body into final template nodes.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
//
// Returns []*ast_domain.TemplateNode which contains the processed nodes ready
// for insertion.
func (t *partialExpansionTask) assembleFinalNodes(ctx context.Context) []*ast_domain.TemplateNode {
	t.validateSlots()

	invokerHashedName := t.ec.graph.PathToHashedName[t.invokerComponent.SourcePath]
	pInfo := t.ec.createPartialInfoAnnotation(ctx, t.invokerNode, t.userAlias, t.targetHashedName, invokerHashedName)

	return processExpandedNodes(t.expandedPartialBody, t.invokerNode, pInfo)
}

// validateSlots checks that slot content given to a component matches the
// slots defined in the partial template.
func (t *partialExpansionTask) validateSlots() {
	definedSlots := collectDefinedSlots(t.loadedPartial.Template.RootNodes)
	definedSlotNames := make([]string, 0, len(definedSlots))
	for name := range definedSlots {
		if name != "" {
			definedSlotNames = append(definedSlotNames, name)
		}
	}

	for providedName, providedContent := range t.groupedSlotContent {
		if _, isDefined := definedSlots[providedName]; !isDefined {
			var message string
			if providedName == "" {
				message = fmt.Sprintf("Component <%s> does not have a default slot, but content was provided.", t.userAlias)
			} else {
				message = fmt.Sprintf("Component <%s> does not have a slot named '%s'", t.userAlias, providedName)
				if suggestion := findClosestMatch(providedName, definedSlotNames); suggestion != "" {
					message += fmt.Sprintf(". Did you mean '%s'?", suggestion)
				}
			}
			diagnostic := ast_domain.NewDiagnosticWithCode(
				ast_domain.Warning, message,
				fmt.Sprintf("<piko:slot name=%q>", providedName),
				annotator_dto.CodeSlotMismatch,
				providedContent.Location,
				t.invokerComponent.SourcePath,
			)
			t.ec.diagnostics = append(t.ec.diagnostics, diagnostic)
		}
	}
}

// getPartialExpansionTask gets a task from the pool and sets it up with the
// given values.
//
// Takes ec (*expansionContext) which holds the expansion state.
// Takes invokerNode (*ast_domain.TemplateNode) which is the template node that
// called the partial.
// Takes invokerComponent (*annotator_dto.ParsedComponent) which is the parsed
// component that contains the invoker.
// Takes aliasToRawPath (map[string]string) which maps user aliases to raw file
// paths.
// Takes userAlias (string) which is the alias used to refer to the partial.
//
// Returns *partialExpansionTask which is the task ready for use.
func getPartialExpansionTask(
	ec *expansionContext,
	invokerNode *ast_domain.TemplateNode,
	invokerComponent *annotator_dto.ParsedComponent,
	aliasToRawPath map[string]string,
	userAlias string,
) *partialExpansionTask {
	t, ok := partialExpansionTaskPool.Get().(*partialExpansionTask)
	if !ok {
		t = &partialExpansionTask{}
	}
	t.ec = ec
	t.invokerNode = invokerNode
	t.invokerComponent = invokerComponent
	t.aliasToRawPath = aliasToRawPath
	t.userAlias = userAlias
	t.loadedPartial = nil
	t.groupedSlotContent = nil
	t.targetHashedName = ""
	t.expandedPartialBody = nil
	t.hasError = false
	return t
}

// putPartialExpansionTask clears the task fields and returns it to the pool.
//
// Takes t (*partialExpansionTask) which is the task to reset and recycle.
func putPartialExpansionTask(t *partialExpansionTask) {
	t.ec = nil
	t.invokerNode = nil
	t.invokerComponent = nil
	t.aliasToRawPath = nil
	t.loadedPartial = nil
	t.groupedSlotContent = nil
	t.userAlias = ""
	t.targetHashedName = ""
	t.expandedPartialBody = nil
	t.hasError = false
	partialExpansionTaskPool.Put(t)
}

// collectDefinedSlots walks a template AST and returns a set of all slot
// names defined within it.
//
// Takes nodes ([]*ast_domain.TemplateNode) which is the template AST to walk.
//
// Returns map[string]bool which contains slot names as keys, each set to true.
func collectDefinedSlots(nodes []*ast_domain.TemplateNode) map[string]bool {
	definedSlots := make(map[string]bool)
	var walk func([]*ast_domain.TemplateNode)
	walk = func(nodeList []*ast_domain.TemplateNode) {
		for _, node := range nodeList {
			if node.NodeType == ast_domain.NodeElement && node.TagName == tagPikoSlot {
				slotName, _ := node.GetAttribute(attributeName)
				definedSlots[slotName] = true
			}
			if node.TagName != tagPikoSlot {
				walk(node.Children)
			}
		}
	}

	walk(nodes)
	return definedSlots
}

// groupContentBySlot sorts template nodes into named slots.
//
// Supports two syntaxes:
//   - <piko:slot name="header">content</piko:slot> places children in the
//     named slot
//   - <article p-slot="header">content</article> places the element itself
//     in the named slot
//
// Takes nodes ([]*ast_domain.TemplateNode) which are the child nodes to group.
//
// Returns map[string]invokerSlotContent which maps slot names to their
// content. An empty string key holds the default slot content.
func groupContentBySlot(nodes []*ast_domain.TemplateNode) map[string]invokerSlotContent {
	grouped := make(map[string]invokerSlotContent)
	defaultContent := make([]*ast_domain.TemplateNode, 0, len(nodes))

	defaultLocation := findDefaultSlotLocation(nodes)

	for _, child := range nodes {
		if child.NodeType == ast_domain.NodeElement && child.TagName == tagPikoSlot {
			slotName, _ := child.GetAttribute(attributeName)
			grouped[slotName] = invokerSlotContent{
				Nodes:    child.Children,
				Location: child.Location,
			}
			continue
		}

		if child.NodeType == ast_domain.NodeElement && child.DirSlot != nil {
			slotName := child.DirSlot.RawExpression
			child.DirSlot = nil
			existing := grouped[slotName]
			existing.Nodes = append(existing.Nodes, child)
			if existing.Location.IsSynthetic() {
				existing.Location = child.Location
			}
			grouped[slotName] = existing
			continue
		}

		if !isWhitespaceOrCommentNode(child) {
			defaultContent = append(defaultContent, child)
		}
	}

	if len(defaultContent) > 0 {
		existing := grouped[""]
		existing.Nodes = append(existing.Nodes, defaultContent...)
		if existing.Location.IsSynthetic() {
			existing.Location = defaultLocation
		}
		grouped[""] = existing
	}

	return grouped
}

// findDefaultSlotLocation finds the location of the first useful node to use
// as the default slot location for error messages.
//
// Takes nodes ([]*ast_domain.TemplateNode) which are the template nodes to
// search.
//
// Returns ast_domain.Location which is the location of the first node that is
// not whitespace or a comment. Returns a zero location if no nodes exist.
func findDefaultSlotLocation(nodes []*ast_domain.TemplateNode) ast_domain.Location {
	for _, n := range nodes {
		if !isWhitespaceOrCommentNode(n) {
			return n.Location
		}
	}
	if len(nodes) > 0 {
		return nodes[0].Location
	}
	return ast_domain.Location{Line: 0, Column: 0, Offset: 0}
}

// processExpandedNodes applies partial invocation info and invoker attributes
// to expanded template nodes.
//
// When there is a single effective root element, it attaches the partial info
// directly to that element. Otherwise, it wraps all nodes in a fragment node
// and marks each effective root with fragment tracking attributes.
//
// Takes expandedNodes ([]*ast_domain.TemplateNode) which are the nodes
// resulting from partial expansion.
// Takes invokerNode (*ast_domain.TemplateNode) which is the original node that
// invoked the partial.
// Takes pInfo (*ast_domain.PartialInvocationInfo) which contains metadata about
// the partial invocation.
//
// Returns []*ast_domain.TemplateNode which contains either the original nodes
// with annotations applied, or a single fragment node wrapping them.
func processExpandedNodes(expandedNodes []*ast_domain.TemplateNode, invokerNode *ast_domain.TemplateNode, pInfo *ast_domain.PartialInvocationInfo) []*ast_domain.TemplateNode {
	effectiveRootElements := findEffectiveRootElements(expandedNodes)

	if len(effectiveRootElements) == 1 {
		singleRoot := effectiveRootElements[0]
		if singleRoot.GoAnnotations == nil {
			singleRoot.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}
		}
		singleRoot.GoAnnotations.PartialInfo = pInfo
		applyInvokerAttributesToExpandedRoot(singleRoot, invokerNode)
		return expandedNodes
	}

	fragmentNode := newFragmentNode(expandedNodes)
	fragmentNode.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}
	fragmentNode.GoAnnotations.PartialInfo = pInfo
	if invokerNode.GoAnnotations != nil {
		fragmentNode.GoAnnotations.OriginalPackageAlias = invokerNode.GoAnnotations.OriginalPackageAlias
		fragmentNode.GoAnnotations.OriginalSourcePath = invokerNode.GoAnnotations.OriginalSourcePath
	}
	applyInvokerAttributesToExpandedRoot(fragmentNode, invokerNode)

	zeroLocation := ast_domain.Location{Line: 0, Column: 0, Offset: 0}
	zeroRange := ast_domain.Range{Start: zeroLocation, End: zeroLocation}
	for i, elementNode := range effectiveRootElements {
		elementNode.Attributes = append(elementNode.Attributes,
			ast_domain.HTMLAttribute{
				Name:           attributePFragment,
				Value:          pInfo.InvocationKey,
				Location:       zeroLocation,
				NameLocation:   zeroLocation,
				AttributeRange: zeroRange,
			},
			ast_domain.HTMLAttribute{
				Name:           attributePFragmentID,
				Value:          strconv.Itoa(i),
				Location:       zeroLocation,
				NameLocation:   zeroLocation,
				AttributeRange: zeroRange,
			},
		)
	}

	return []*ast_domain.TemplateNode{fragmentNode}
}

// applyInvokerAttributesToExpandedRoot copies attributes and directives from
// the invoker node onto the expanded template root node.
//
// When either node is nil, returns without making changes.
//
// Takes targetNode (*ast_domain.TemplateNode) which is the expanded root that
// will receive the attributes.
// Takes invokerNode (*ast_domain.TemplateNode) which provides the attributes
// to copy.
func applyInvokerAttributesToExpandedRoot(targetNode, invokerNode *ast_domain.TemplateNode) {
	if targetNode == nil || invokerNode == nil {
		return
	}
	invokerOriginPackageAlias := ""
	if invokerNode.GoAnnotations != nil && invokerNode.GoAnnotations.OriginalPackageAlias != nil {
		invokerOriginPackageAlias = *invokerNode.GoAnnotations.OriginalPackageAlias
	}

	mergeStaticAttributes(targetNode, invokerNode)
	mergeDynamicAttributes(targetNode, invokerNode, invokerOriginPackageAlias)
	applyInvokerDirectives(targetNode, invokerNode, invokerOriginPackageAlias)
}

// mergeStaticAttributes copies static attributes from the invoker node into the
// target node. It gathers the target's current attributes, adds the invoker's
// attributes, then builds a sorted attribute list on the target.
//
// Takes targetNode (*ast_domain.TemplateNode) which receives the merged
// attributes.
// Takes invokerNode (*ast_domain.TemplateNode) which provides the attributes
// to add.
func mergeStaticAttributes(targetNode, invokerNode *ast_domain.TemplateNode) {
	finalAttrs := collectTargetStaticAttrs(targetNode)
	mergeInvokerStaticAttrs(invokerNode, finalAttrs)
	targetNode.Attributes = rebuildSortedStaticAttrs(finalAttrs)
}

// collectTargetStaticAttrs builds a map of attributes from a target node.
//
// Takes targetNode (*ast_domain.TemplateNode) which provides the template node
// to collect attributes from.
//
// Returns map[string]ast_domain.HTMLAttribute which maps attribute names in
// lowercase to their values.
func collectTargetStaticAttrs(targetNode *ast_domain.TemplateNode) map[string]ast_domain.HTMLAttribute {
	finalAttrs := make(map[string]ast_domain.HTMLAttribute)
	for i := range targetNode.Attributes {
		attr := &targetNode.Attributes[i]
		finalAttrs[strings.ToLower(attr.Name)] = *attr
	}
	return finalAttrs
}

// mergeInvokerStaticAttrs copies attributes from the invoker node into the
// final attribute map.
//
// Takes invokerNode (*ast_domain.TemplateNode) which provides the source
// attributes to copy.
// Takes finalAttrs (map[string]ast_domain.HTMLAttribute) which receives the
// copied attributes.
func mergeInvokerStaticAttrs(invokerNode *ast_domain.TemplateNode, finalAttrs map[string]ast_domain.HTMLAttribute) {
	for i := range invokerNode.Attributes {
		invokerAttr := &invokerNode.Attributes[i]
		attributeNameLower := strings.ToLower(invokerAttr.Name)

		if shouldSkipStaticAttr(attributeNameLower) {
			continue
		}

		mergeStaticAttrByType(attributeNameLower, invokerAttr, finalAttrs)
	}
}

// shouldSkipStaticAttr reports whether an attribute should be skipped during
// merging.
//
// Takes attributeNameLower (string) which is the lowercase attribute
// name to check.
//
// Returns bool which is true when the attribute is the "is" attribute or has a
// server or request prefix.
func shouldSkipStaticAttr(attributeNameLower string) bool {
	if attributeNameLower == attributeIs {
		return true
	}
	return strings.HasPrefix(attributeNameLower, prefixServer) || strings.HasPrefix(attributeNameLower, prefixRequest)
}

// mergeStaticAttrByType merges a static attribute using the correct method for
// its type.
//
// Takes attributeNameLower (string) which is the lowercase attribute name.
// Takes invokerAttr (*ast_domain.HTMLAttribute) which is the attribute to
// merge.
// Takes finalAttrs (map[string]ast_domain.HTMLAttribute) which holds the merged
// attributes.
func mergeStaticAttrByType(attributeNameLower string, invokerAttr *ast_domain.HTMLAttribute, finalAttrs map[string]ast_domain.HTMLAttribute) {
	switch {
	case attributeNameLower == attributeClass:
		mergeClassAttr(invokerAttr, finalAttrs)
	case isPartialMetadataAttr(attributeNameLower):
		mergePartialMetadataAttr(attributeNameLower, invokerAttr, finalAttrs)
	default:
		finalAttrs[attributeNameLower] = *invokerAttr
	}
}

// mergeClassAttr merges a class attribute into the final attributes map.
//
// When a class attribute already exists, the values are joined with a space.
// When no class attribute exists, the invoker attribute is added directly.
//
// Takes invokerAttr (*ast_domain.HTMLAttribute) which provides the class
// attribute to merge.
// Takes finalAttrs (map[string]ast_domain.HTMLAttribute) which holds the
// collected attributes where the class will be merged.
func mergeClassAttr(invokerAttr *ast_domain.HTMLAttribute, finalAttrs map[string]ast_domain.HTMLAttribute) {
	if existing, ok := finalAttrs[attributeClass]; ok {
		existing.Value = strings.TrimSpace(existing.Value + " " + invokerAttr.Value)
		finalAttrs[attributeClass] = existing
	} else {
		finalAttrs[attributeClass] = *invokerAttr
	}
}

// isPartialMetadataAttr reports whether the given attribute name is a partial
// metadata attribute.
//
// Takes attributeNameLower (string) which is the lowercase attribute
// name to check.
//
// Returns bool which is true if the attribute is partial, partialName, or
// partialSrc.
func isPartialMetadataAttr(attributeNameLower string) bool {
	return attributeNameLower == attributePartial || attributeNameLower == attributePartialName || attributeNameLower == attributePartialSrc
}

// mergePartialMetadataAttr merges a partial metadata attribute by placing the
// invoker value before the existing value.
//
// Takes attributeNameLower (string) which is the lowercase name of the attribute.
// Takes invokerAttr (*ast_domain.HTMLAttribute) which is the attribute from
// the invoker element.
// Takes finalAttrs (map[string]ast_domain.HTMLAttribute) which is the target
// map to merge into.
func mergePartialMetadataAttr(attributeNameLower string, invokerAttr *ast_domain.HTMLAttribute, finalAttrs map[string]ast_domain.HTMLAttribute) {
	if existing, ok := finalAttrs[attributeNameLower]; ok {
		existing.Value = strings.TrimSpace(invokerAttr.Value + " " + existing.Value)
		finalAttrs[attributeNameLower] = existing
	} else {
		finalAttrs[attributeNameLower] = *invokerAttr
	}
}

// rebuildSortedStaticAttrs builds a slice of attributes sorted by key name.
//
// Takes finalAttrs (map[string]ast_domain.HTMLAttribute) which holds the
// attributes to sort.
//
// Returns []ast_domain.HTMLAttribute which contains the attributes in
// alphabetical order by key.
func rebuildSortedStaticAttrs(finalAttrs map[string]ast_domain.HTMLAttribute) []ast_domain.HTMLAttribute {
	sortedKeys := make([]string, 0, len(finalAttrs))
	for k := range finalAttrs {
		sortedKeys = append(sortedKeys, k)
	}
	slices.Sort(sortedKeys)

	rebuiltAttrs := make([]ast_domain.HTMLAttribute, 0, len(finalAttrs))
	for _, key := range sortedKeys {
		rebuiltAttrs = append(rebuiltAttrs, finalAttrs[key])
	}
	return rebuiltAttrs
}

// mergeDynamicAttributes combines dynamic attributes from a partial and its
// invoker into a single sorted set on the target node.
//
// Takes targetNode (*ast_domain.TemplateNode) which receives the merged
// attributes.
// Takes invokerNode (*ast_domain.TemplateNode) which provides the dynamic
// attributes from the caller.
// Takes invokerOrigin (string) which identifies where the invoker comes from.
func mergeDynamicAttributes(targetNode, invokerNode *ast_domain.TemplateNode, invokerOrigin string) {
	ensureDynamicAttributeAnnotations(targetNode)
	partialOrigin := getPartialOrigin(targetNode)

	finalDynAttrs := make(map[string]ast_domain.DynamicAttribute)
	collectPartialDynamicAttrs(targetNode, finalDynAttrs, partialOrigin)
	collectInvokerDynamicAttrs(invokerNode, targetNode, finalDynAttrs, invokerOrigin)

	targetNode.DynamicAttributes = rebuildSortedDynamicAttrs(finalDynAttrs, targetNode, invokerNode, invokerOrigin)
}

// ensureDynamicAttributeAnnotations sets up the GoAnnotations and
// DynamicAttributeOrigins fields on a template node if they are nil.
//
// Takes node (*ast_domain.TemplateNode) which is the node to set up.
func ensureDynamicAttributeAnnotations(node *ast_domain.TemplateNode) {
	if node.GoAnnotations == nil {
		node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}
	}
	if node.GoAnnotations.DynamicAttributeOrigins == nil {
		node.GoAnnotations.DynamicAttributeOrigins = make(map[string]string)
	}
}

// getPartialOrigin returns the original package alias from a template node.
//
// Takes node (*ast_domain.TemplateNode) which holds the template metadata.
//
// Returns string which is the package alias, or empty if none is set.
func getPartialOrigin(node *ast_domain.TemplateNode) string {
	if node.GoAnnotations != nil && node.GoAnnotations.OriginalPackageAlias != nil {
		return *node.GoAnnotations.OriginalPackageAlias
	}
	return ""
}

// collectPartialDynamicAttrs gathers dynamic attributes from a template node
// and adds them to the final attributes map.
//
// Takes targetNode (*ast_domain.TemplateNode) which is the node to collect
// attributes from.
// Takes finalDynAttrs (map[string]ast_domain.DynamicAttribute) which stores
// the collected attributes using lowercase names as keys.
// Takes partialOrigin (string) which names the source partial. When set, it
// records where each attribute came from.
func collectPartialDynamicAttrs(targetNode *ast_domain.TemplateNode, finalDynAttrs map[string]ast_domain.DynamicAttribute, partialOrigin string) {
	for i := range targetNode.DynamicAttributes {
		dynAttr := &targetNode.DynamicAttributes[i]
		attributeNameLower := strings.ToLower(dynAttr.Name)
		finalDynAttrs[attributeNameLower] = *dynAttr
		if partialOrigin != "" {
			targetNode.GoAnnotations.DynamicAttributeOrigins[attributeNameLower] = partialOrigin
		}
	}
}

// collectInvokerDynamicAttrs gathers dynamic attributes from an invoker node
// and adds them to the final attributes map.
//
// Attributes with server or request prefixes are skipped. Each collected
// attribute is marked with its origin and stored in the target node's
// annotations.
//
// Takes invokerNode (*ast_domain.TemplateNode) which provides the source
// attributes to collect.
// Takes targetNode (*ast_domain.TemplateNode) which receives the origin
// annotations.
// Takes finalDynAttrs (map[string]ast_domain.DynamicAttribute) which stores
// the collected attributes by lowercase name.
// Takes invokerOrigin (string) which identifies where the attributes came
// from.
func collectInvokerDynamicAttrs(invokerNode, targetNode *ast_domain.TemplateNode, finalDynAttrs map[string]ast_domain.DynamicAttribute, invokerOrigin string) {
	for i := range invokerNode.DynamicAttributes {
		invokerDynAttr := invokerNode.DynamicAttributes[i]
		attributeNameLower := strings.ToLower(invokerDynAttr.Name)

		if strings.HasPrefix(attributeNameLower, prefixServer) || strings.HasPrefix(attributeNameLower, prefixRequest) {
			continue
		}

		stampInvokerOriginOnAttr(&invokerDynAttr, invokerOrigin)
		finalDynAttrs[attributeNameLower] = invokerDynAttr
		targetNode.GoAnnotations.DynamicAttributeOrigins[attributeNameLower] = invokerOrigin
	}
}

// stampInvokerOriginOnAttr sets the package alias origin on an attribute.
//
// When origin is empty, returns without making changes.
//
// Takes attr (*ast_domain.DynamicAttribute) which receives the origin value.
// Takes origin (string) which specifies the package alias to set.
func stampInvokerOriginOnAttr(attr *ast_domain.DynamicAttribute, origin string) {
	if origin == "" {
		return
	}
	if attr.GoAnnotations == nil {
		attr.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}
	}
	attr.GoAnnotations.OriginalPackageAlias = &origin
}

// rebuildSortedDynamicAttrs creates a sorted slice of dynamic attributes with
// completed origin information.
//
// Takes finalDynAttrs (map[string]ast_domain.DynamicAttribute) which contains
// the dynamic attributes to rebuild.
// Takes targetNode (*ast_domain.TemplateNode) which is the node that receives
// the attributes.
// Takes invokerNode (*ast_domain.TemplateNode) which is the node that started
// the attribute resolution.
// Takes invokerOrigin (string) which names the source of the invocation.
//
// Returns []ast_domain.DynamicAttribute which contains the attributes sorted
// by key with origin metadata set.
func rebuildSortedDynamicAttrs(finalDynAttrs map[string]ast_domain.DynamicAttribute, targetNode, invokerNode *ast_domain.TemplateNode, invokerOrigin string) []ast_domain.DynamicAttribute {
	sortedKeys := getSortedAttrKeys(finalDynAttrs)
	rebuiltDynAttrs := make([]ast_domain.DynamicAttribute, 0, len(finalDynAttrs))

	for _, key := range sortedKeys {
		attr := finalDynAttrs[key]
		finaliseAttrOrigin(&attr, targetNode, invokerNode, invokerOrigin, key)
		rebuiltDynAttrs = append(rebuiltDynAttrs, attr)
	}
	return rebuiltDynAttrs
}

// getSortedAttrKeys returns the keys from a dynamic attributes map in sorted
// order.
//
// Takes attrs (map[string]ast_domain.DynamicAttribute) which contains the
// attributes whose keys should be extracted.
//
// Returns []string which contains the attribute keys in alphabetical order.
func getSortedAttrKeys(attrs map[string]ast_domain.DynamicAttribute) []string {
	sortedKeys := make([]string, 0, len(attrs))
	for k := range attrs {
		sortedKeys = append(sortedKeys, k)
	}
	slices.Sort(sortedKeys)
	return sortedKeys
}

// finaliseAttrOrigin sets the package alias and source path on a dynamic
// attribute when it comes from the invoker node.
//
// Takes attr (*ast_domain.DynamicAttribute) which is the attribute to update.
// Takes targetNode (*ast_domain.TemplateNode) which holds the origin map.
// Takes invokerNode (*ast_domain.TemplateNode) which provides source details.
// Takes invokerOrigin (string) which is the expected origin to match.
// Takes key (string) which is the key to find the attribute in the origin map.
func finaliseAttrOrigin(attr *ast_domain.DynamicAttribute, targetNode, invokerNode *ast_domain.TemplateNode, invokerOrigin, key string) {
	origin, isFromInvoker := targetNode.GoAnnotations.DynamicAttributeOrigins[key]
	if !isFromInvoker || origin != invokerOrigin {
		return
	}

	if attr.GoAnnotations == nil {
		attr.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}
	}
	attr.GoAnnotations.OriginalPackageAlias = &invokerOrigin

	if invokerNode.GoAnnotations != nil && invokerNode.GoAnnotations.OriginalSourcePath != nil {
		attr.GoAnnotations.OriginalSourcePath = invokerNode.GoAnnotations.OriginalSourcePath
	}
}

// applyInvokerDirectives copies directives from the invoker node to the target
// node. Each copied directive is marked with origin data to show where it came
// from.
//
// Takes targetNode (*ast_domain.TemplateNode) which receives the directives.
// Takes invokerNode (*ast_domain.TemplateNode) which provides the directives.
// Takes invokerOrigin (string) which is the source package alias.
func applyInvokerDirectives(targetNode, invokerNode *ast_domain.TemplateNode, invokerOrigin string) {
	stamp := func(d *ast_domain.Directive) *ast_domain.Directive {
		if d == nil {
			return nil
		}
		cloned := d.Clone()
		if cloned.GoAnnotations == nil {
			cloned.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}
		}
		cloned.GoAnnotations.OriginalPackageAlias = &invokerOrigin
		if invokerNode.GoAnnotations != nil && invokerNode.GoAnnotations.OriginalSourcePath != nil {
			cloned.GoAnnotations.OriginalSourcePath = invokerNode.GoAnnotations.OriginalSourcePath
		}
		return &cloned
	}
	if invokerNode.DirIf != nil {
		targetNode.DirIf = stamp(invokerNode.DirIf)
	}
	if invokerNode.DirElseIf != nil {
		targetNode.DirElseIf = stamp(invokerNode.DirElseIf)
	}
	if invokerNode.DirElse != nil {
		targetNode.DirElse = stamp(invokerNode.DirElse)
	}
	if invokerNode.DirFor != nil {
		targetNode.DirFor = stamp(invokerNode.DirFor)
	}
	if invokerNode.DirShow != nil {
		targetNode.DirShow = stamp(invokerNode.DirShow)
	}
	if invokerNode.DirKey != nil {
		targetNode.DirKey = stamp(invokerNode.DirKey)
	}
	if invokerNode.DirClass != nil {
		targetNode.DirClass = stamp(invokerNode.DirClass)
	}
	if invokerNode.DirStyle != nil {
		targetNode.DirStyle = stamp(invokerNode.DirStyle)
	}
}

// findEffectiveRootElements filters the given nodes to return only element
// nodes.
//
// Takes nodes ([]*ast_domain.TemplateNode) which is the list of nodes to
// filter.
//
// Returns []*ast_domain.TemplateNode which contains only nodes with type
// NodeElement.
func findEffectiveRootElements(nodes []*ast_domain.TemplateNode) []*ast_domain.TemplateNode {
	elements := make([]*ast_domain.TemplateNode, 0, len(nodes))
	for _, node := range nodes {
		if node.NodeType == ast_domain.NodeElement {
			elements = append(elements, node)
		}
	}
	return elements
}

// isWhitespaceOrCommentNode checks whether a node should be skipped when
// gathering default slot content.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true if the node is a comment or contains only
// whitespace.
func isWhitespaceOrCommentNode(node *ast_domain.TemplateNode) bool {
	if node.NodeType == ast_domain.NodeComment {
		return true
	}
	return node.NodeType == ast_domain.NodeText &&
		len(node.RichText) == 0 &&
		strings.TrimSpace(node.TextContent) == ""
}
