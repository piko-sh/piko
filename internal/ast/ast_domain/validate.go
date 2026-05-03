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

package ast_domain

// Validates AST structures for correctness and consistency, checking directive usage, attribute conflicts, and structural integrity.
// Detects issues like duplicate directives, invalid p-for/p-if combinations, and mismatched source paths in merged attributes.

import (
	"fmt"
	"strings"
)

// directivePrefix is the prefix added to directive names for display.
const directivePrefix = "p-"

// ValidateAST checks a template AST for structural and semantic problems.
// It adds any issues found as diagnostics to the tree.
//
// When tree is nil or has no root nodes, returns without doing anything.
//
// Takes tree (*TemplateAST) which is the parsed template to check.
func ValidateAST(tree *TemplateAST) {
	if tree == nil || len(tree.RootNodes) == 0 {
		return
	}
	validateNodeList(tree.RootNodes, tree)
}

// HoistDiagnostics gathers all diagnostics from child nodes and moves them to
// the tree-level diagnostics list.
//
// When the tree or its source path is nil, returns without making changes.
//
// Takes tree (*TemplateAST) which is the template tree to process.
func HoistDiagnostics(tree *TemplateAST) {
	if tree == nil || tree.SourcePath == nil {
		return
	}

	sourcePath := *tree.SourcePath

	tree.Walk(func(node *TemplateNode) bool {
		if len(node.Diagnostics) > 0 {
			for _, diagnostic := range node.Diagnostics {
				if diagnostic.SourcePath == "" {
					diagnostic.SourcePath = sourcePath
				}
			}
			tree.Diagnostics = append(tree.Diagnostics, node.Diagnostics...)
			node.Diagnostics = nil
		}
		return true
	})
}

// hasDifferentSourcePath checks whether two annotations come from different
// source files. This happens when attributes were merged from different files,
// such as during partial expansion.
//
// Takes pForAnn (*GoGeneratorAnnotation) which is the annotation from p-for.
// Takes otherAnn (*GoGeneratorAnnotation) which is the annotation to compare.
//
// Returns bool which is true if the source paths differ, or false if both come
// from the same file or if paths cannot be compared.
func hasDifferentSourcePath(pForAnn, otherAnn *GoGeneratorAnnotation) bool {
	if pForAnn == nil || otherAnn == nil {
		return false
	}
	if pForAnn.OriginalSourcePath == nil || otherAnn.OriginalSourcePath == nil {
		return false
	}
	return *pForAnn.OriginalSourcePath != *otherAnn.OriginalSourcePath
}

// validateNodeList checks each node in the list for template errors.
// It runs checks on element nodes and recurses into child nodes.
//
// Takes nodes ([]*TemplateNode) which is the list of nodes to check.
// Takes tree (*TemplateAST) which provides context for error reporting.
func validateNodeList(nodes []*TemplateNode, tree *TemplateAST) {
	for i, node := range nodes {
		previousElement := findPreviousElementSibling(nodes, i)
		validateAdjacency(node, previousElement, tree)

		if !isElementNode(node) {
			continue
		}

		validateAttributeConflicts(node, tree)
		validateContentDirectives(node, tree)
		validateRedundantConditionals(node, tree)
		validateDirectivePrecedence(node, tree)
		validateModelableElement(node, tree)
		validateKeyedForLoop(node, tree)

		if len(node.Children) > 0 {
			validateNodeList(node.Children, tree)
		}
	}
}

// validateAdjacency checks that else-type directives follow valid nodes.
//
// Else-type directives (such as p-else and p-else-if) must come straight after
// an element with p-if or p-else-if. Adds a diagnostic error to the tree when
// this rule is broken.
//
// Takes node (*TemplateNode) which is the node to check.
// Takes previousSibling (*TemplateNode) which is the node before it.
// Takes tree (*TemplateAST) which receives any diagnostic errors.
func validateAdjacency(node, previousSibling *TemplateNode, tree *TemplateAST) {
	directive, isElseDirective := getElseDirective(node)
	if !isElseDirective {
		return
	}

	directiveName, ok := DirectiveTypeToName[directive.Type]
	if !ok {
		directiveName = directivePrefix + strings.ToLower(directive.Type.String())
	}

	if !isValidConditionalPredecessor(previousSibling) {
		message := fmt.Sprintf("The '%s' directive must immediately follow an element with 'p-if' or 'p-else-if'. Whitespace and comments are ignored, but other nodes break the chain.", directiveName)
		sourcePath := getNodeSourcePath(node, tree)
		diagnostic := NewDiagnosticWithCode(Error, message, directiveName, CodeInvalidDirectivePlacement, directive.NameLocation, sourcePath)
		tree.Diagnostics = append(tree.Diagnostics, diagnostic)
	}
}

// validateAttributeConflicts checks for conflicts between static attributes
// and dynamic bindings on a template node. It adds a warning to the tree when
// a static attribute would be replaced by a directive or binding.
//
// Takes node (*TemplateNode) which is the node to check for conflicts.
// Takes tree (*TemplateAST) which receives any warning messages.
func validateAttributeConflicts(node *TemplateNode, tree *TemplateAST) {
	staticAttrs := make(map[string]Location)
	for i := range node.Attributes {
		attr := &node.Attributes[i]
		staticAttrs[strings.ToLower(attr.Name)] = attr.Location
	}

	for attributeName, directive := range node.Binds {
		if _, exists := staticAttrs[strings.ToLower(attributeName)]; exists {
			message := fmt.Sprintf(
				"The '%s' attribute is defined statically but also targeted by a dynamic 'p-bind:%s' binding. The dynamic binding will overwrite the static one.",
				attributeName, attributeName,
			)
			sourcePath := getNodeSourcePath(node, tree)
			diagnostic := NewDiagnosticWithCode(Warning, message, "p-bind:"+attributeName, CodeAttributeConflict, directive.NameLocation, sourcePath)
			tree.Diagnostics = append(tree.Diagnostics, diagnostic)
		}
	}
	for i := range node.DynamicAttributes {
		dynAttr := &node.DynamicAttributes[i]
		if _, exists := staticAttrs[strings.ToLower(dynAttr.Name)]; exists {
			message := fmt.Sprintf(
				"The '%s' attribute is defined statically but also targeted by a dynamic ':%s' binding. The dynamic binding will overwrite the static one.",
				dynAttr.Name, dynAttr.Name,
			)
			sourcePath := getNodeSourcePath(node, tree)
			diagnostic := NewDiagnosticWithCode(Warning, message, ":"+dynAttr.Name, CodeAttributeConflict, dynAttr.NameLocation, sourcePath)
			tree.Diagnostics = append(tree.Diagnostics, diagnostic)
		}
	}
}

// validateContentDirectives checks a template node for content directives that
// conflict or are not needed.
//
// Takes node (*TemplateNode) which is the node to check.
// Takes tree (*TemplateAST) which receives any diagnostics found.
func validateContentDirectives(node *TemplateNode, tree *TemplateAST) {
	if node.DirText != nil && node.DirHTML != nil {
		message := "Cannot use both 'p-text' and 'p-html' on the same element. One will be ignored."
		sourcePath := getNodeSourcePath(node, tree)
		diagnostic := NewDiagnosticWithCode(Error, message, "p-html/p-text", CodeConflictingDirectives, node.DirHTML.NameLocation, sourcePath)
		tree.Diagnostics = append(tree.Diagnostics, diagnostic)
	}

	contentDirective := getContentDirective(node)
	if contentDirective == nil {
		return
	}

	directiveName := directivePrefix + strings.ToLower(contentDirective.Type.String())

	if hasMeaningfulContent(node) {
		message := fmt.Sprintf(
			"This element contains child nodes that will be overwritten by the '%s' directive at runtime. The existing content serves no purpose and can be misleading.",
			directiveName,
		)
		sourcePath := getNodeSourcePath(node, tree)
		diagnostic := NewDiagnosticWithCode(Warning, message, directiveName, CodeDirectiveOverwritesContent, contentDirective.NameLocation, sourcePath)
		tree.Diagnostics = append(tree.Diagnostics, diagnostic)
	}
}

// validateRedundantConditionals checks that an element does not have both
// p-else and p-else-if directives, and adds an error diagnostic if it does.
//
// Takes node (*TemplateNode) which is the node to check.
// Takes tree (*TemplateAST) which receives any error diagnostics.
func validateRedundantConditionals(node *TemplateNode, tree *TemplateAST) {
	if node.DirElse != nil && node.DirElseIf != nil {
		message := "An element cannot have both 'p-else' and 'p-else-if' directives. The 'p-else-if' will be ignored."
		sourcePath := getNodeSourcePath(node, tree)
		diagnostic := NewDiagnosticWithCode(Error, message, "p-else-if", CodeConflictingDirectives, node.DirElseIf.NameLocation, sourcePath)
		tree.Diagnostics = append(tree.Diagnostics, diagnostic)
	}
}

// validateDirectivePrecedence checks that p-for is the first directive on an
// element.
//
// The p-for directive always runs before other directives, so it should appear
// first in the source code. Checks all other directives, dynamic attributes,
// and event handlers. If any appear before p-for, reports a warning to help keep
// the code clear.
//
// Takes node (*TemplateNode) which is the element node to check.
// Takes tree (*TemplateAST) which provides the AST for reporting warnings.
func validateDirectivePrecedence(node *TemplateNode, tree *TemplateAST) {
	if node.DirFor == nil {
		return
	}

	pForLocation := node.DirFor.NameLocation
	pForAnn := node.DirFor.GoAnnotations
	checkStandardDirectives(node, pForLocation, pForAnn, tree)
	checkBindDirectives(node, pForLocation, pForAnn, tree)
	checkEventDirectives(node, pForLocation, pForAnn, tree)
	checkDynamicAttributes(node, pForLocation, pForAnn, tree)
}

// checkStandardDirectives checks that standard directives appear after p-for
// in source order. Skips the check if the directive comes from a different
// source file than p-for, for example due to partial expansion.
//
// Takes node (*TemplateNode) which contains the directives to check.
// Takes pForLocation (Location) which marks where the p-for directive is.
// Takes pForAnn (*GoGeneratorAnnotation) which is the annotation from p-for.
// Takes tree (*TemplateAST) which receives any precedence diagnostics.
func checkStandardDirectives(node *TemplateNode, pForLocation Location, pForAnn *GoGeneratorAnnotation, tree *TemplateAST) {
	directivesToCheck := []*Directive{
		node.DirIf, node.DirElseIf, node.DirShow, node.DirModel,
		node.DirRef, node.DirSlot, node.DirClass, node.DirStyle, node.DirText,
		node.DirHTML, node.DirKey, node.DirContext, node.DirScaffold,
	}

	for _, d := range directivesToCheck {
		if d == nil {
			continue
		}
		if hasDifferentSourcePath(pForAnn, d.GoAnnotations) {
			continue
		}
		if d.NameLocation.IsBefore(pForLocation) {
			directiveName := DirectiveTypeToName[d.Type]
			if directiveName == "" {
				directiveName = directivePrefix + strings.ToLower(d.Type.String())
			}
			addPrecedenceDiagnostic(tree, node, directiveName, d.NameLocation)
		}
	}
}

// checkBindDirectives checks that bind directives appear after p-for. It skips
// the check if the directive comes from a different source file than p-for.
//
// Takes node (*TemplateNode) which contains the bind directives to check.
// Takes pForLocation (Location) which marks where p-for appears.
// Takes pForAnn (*GoGeneratorAnnotation) which is the annotation from p-for.
// Takes tree (*TemplateAST) which receives any diagnostics.
func checkBindDirectives(node *TemplateNode, pForLocation Location, pForAnn *GoGeneratorAnnotation, tree *TemplateAST) {
	for _, bindDirective := range node.Binds {
		if hasDifferentSourcePath(pForAnn, bindDirective.GoAnnotations) {
			continue
		}
		if bindDirective.NameLocation.IsBefore(pForLocation) {
			directiveName := "p-bind:" + bindDirective.Arg
			addPrecedenceDiagnostic(tree, node, directiveName, bindDirective.NameLocation)
		}
	}
}

// checkEventDirectives checks that event directives do not appear before a
// p-for directive on the same node. Skips the check if the directive comes
// from a different source file than p-for.
//
// Takes node (*TemplateNode) which is the template node to check.
// Takes pForLocation (Location) which is the position of the p-for directive.
// Takes pForAnn (*GoGeneratorAnnotation) which is the annotation from p-for.
// Takes tree (*TemplateAST) which is the AST for adding diagnostics.
func checkEventDirectives(node *TemplateNode, pForLocation Location, pForAnn *GoGeneratorAnnotation, tree *TemplateAST) {
	checkEventDirectiveMap(node.OnEvents, "p-on:", node, pForLocation, pForAnn, tree)
	checkEventDirectiveMap(node.CustomEvents, "p-event:", node, pForLocation, pForAnn, tree)
}

// checkEventDirectiveMap checks a map of event directives for ordering issues
// with p-for.
//
// Takes eventMap (map[string][]Directive) which maps event names to their
// directives.
// Takes prefix (string) which is the directive prefix for error messages.
// Takes node (*TemplateNode) which is the template node being checked.
// Takes pForLocation (Location) which is the position of the p-for directive.
// Takes pForAnn (*GoGeneratorAnnotation) which is the annotation from p-for.
// Takes tree (*TemplateAST) which is the AST for adding error messages.
func checkEventDirectiveMap(eventMap map[string][]Directive, prefix string, node *TemplateNode, pForLocation Location, pForAnn *GoGeneratorAnnotation, tree *TemplateAST) {
	for _, eventDirectives := range eventMap {
		for i := range eventDirectives {
			eventDirective := &eventDirectives[i]
			if hasDifferentSourcePath(pForAnn, eventDirective.GoAnnotations) {
				continue
			}
			if eventDirective.NameLocation.IsBefore(pForLocation) {
				directiveName := prefix + eventDirective.Arg
				addPrecedenceDiagnostic(tree, node, directiveName, eventDirective.NameLocation)
			}
		}
	}
}

// checkDynamicAttributes checks that dynamic attributes come after the p-for
// directive to ensure correct order. Skips the check if the attribute comes
// from a different source file than p-for.
//
// Takes node (*TemplateNode) which is the template node to check.
// Takes pForLocation (Location) which is the position of the p-for directive.
// Takes pForAnn (*GoGeneratorAnnotation) which is the annotation from p-for.
// Takes tree (*TemplateAST) which is the AST for adding diagnostics.
func checkDynamicAttributes(node *TemplateNode, pForLocation Location, pForAnn *GoGeneratorAnnotation, tree *TemplateAST) {
	for i := range node.DynamicAttributes {
		attr := &node.DynamicAttributes[i]
		if attr.NameLocation.IsSynthetic() {
			continue
		}
		if hasDifferentSourcePath(pForAnn, attr.GoAnnotations) {
			continue
		}
		if attr.NameLocation.IsBefore(pForLocation) {
			attributeName := ":" + attr.Name
			addPrecedenceDiagnostic(tree, node, attributeName, attr.NameLocation)
		}
	}
}

// addPrecedenceDiagnostic creates and adds a warning about directive order to
// the AST diagnostics.
//
// Takes tree (*TemplateAST) which receives the new diagnostic.
// Takes node (*TemplateNode) which is the element with the ordering issue.
// Takes violatorName (string) which is the name of the directive that appears
// before p-for.
// Takes violatorLocation (Location) which is the source position of the
// directive.
func addPrecedenceDiagnostic(tree *TemplateAST, node *TemplateNode, violatorName string, violatorLocation Location) {
	message := fmt.Sprintf(
		"Directive precedence on <%s>: `%s` is written before `p-for`, but `p-for` always has the highest "+
			"precedence and is evaluated first. To make this behaviour explicit and avoid potential confusion, "+
			"the `p-for` attribute should be placed before all other directives and dynamic attributes on an element.",
		node.TagName,
		violatorName,
	)
	sourcePath := getNodeSourcePath(node, tree)

	diagnostic := NewDiagnosticWithCode(Warning, message, violatorName, CodeDirectivePrecedence, violatorLocation, sourcePath)
	tree.Diagnostics = append(tree.Diagnostics, diagnostic)
}

// validateModelableElement checks that p-model is used on a valid element.
//
// Takes node (*TemplateNode) which is the element to check.
// Takes tree (*TemplateAST) which receives any error diagnostics.
func validateModelableElement(node *TemplateNode, tree *TemplateAST) {
	if node.DirModel == nil {
		return
	}
	if !isModelableElement(node.TagName) {
		message := fmt.Sprintf("'p-model' can only be used on <input>, <textarea>, and <select> elements, not on <%s>.", node.TagName)
		sourcePath := getNodeSourcePath(node, tree)
		diagnostic := NewDiagnosticWithCode(Error, message, "p-model", CodeInvalidDirectiveTarget, node.DirModel.NameLocation, sourcePath)
		tree.Diagnostics = append(tree.Diagnostics, diagnostic)
	}
}

// validateKeyedForLoop checks that elements with p-for directives also have a
// p-key binding for efficient updates.
//
// Takes node (*TemplateNode) which is the template node to check.
// Takes tree (*TemplateAST) which receives any warnings found.
func validateKeyedForLoop(node *TemplateNode, tree *TemplateAST) {
	if node.DirFor != nil && node.DirKey == nil {
		message := "Elements with 'p-for' directives should have a unique 'p-key' binding for efficient updates. " +
			"The framework will fall back to using the loop index, which may cause performance issues or " +
			"unexpected behaviour with stateful components."
		sourcePath := getNodeSourcePath(node, tree)
		diagnostic := NewDiagnosticWithCode(Warning, message, "p-for", CodeMissingLoopKey, node.DirFor.NameLocation, sourcePath)
		tree.Diagnostics = append(tree.Diagnostics, diagnostic)
	}
}

// findPreviousElementSibling searches backwards through nodes to find the
// nearest element node before the current position.
//
// Takes nodes ([]*TemplateNode) which is the list of sibling nodes to search.
// Takes currentIndex (int) which is the position to start searching from.
//
// Returns *TemplateNode which is the previous element sibling, or nil if none
// exists or if non-whitespace text appears first.
func findPreviousElementSibling(nodes []*TemplateNode, currentIndex int) *TemplateNode {
	for i := currentIndex - 1; i >= 0; i-- {
		previousNode := nodes[i]
		if previousNode.NodeType == NodeElement {
			return previousNode
		}
		if previousNode.NodeType == NodeText && !isWhitespaceOnlyText(previousNode) {
			return nil
		}
	}
	return nil
}

// getElseDirective returns the else-if or else directive from a template node.
//
// Takes node (*TemplateNode) which is the node to check.
//
// Returns *Directive which is the else-if or else directive, or nil if not
// found.
// Returns bool which is true when a directive was found.
func getElseDirective(node *TemplateNode) (*Directive, bool) {
	if node.DirElseIf != nil {
		return node.DirElseIf, true
	}
	if node.DirElse != nil {
		return node.DirElse, true
	}
	return nil, false
}

// getContentDirective returns the content directive from a template node.
//
// Takes node (*TemplateNode) which is the node to get the directive from.
//
// Returns *Directive which is the text or HTML directive, or nil if neither
// exists.
func getContentDirective(node *TemplateNode) *Directive {
	if node.DirText != nil {
		return node.DirText
	}
	if node.DirHTML != nil {
		return node.DirHTML
	}
	return nil
}

// isElementNode reports whether the given node is an element node.
//
// Takes node (*TemplateNode) which is the node to check.
//
// Returns bool which is true if the node type is NodeElement.
func isElementNode(node *TemplateNode) bool {
	return node.NodeType == NodeElement
}

// isValidConditionalPredecessor checks whether a node can come before an else
// or else-if directive.
//
// Takes node (*TemplateNode) which is the template node to check.
//
// Returns bool which is true if the node is not nil and contains an if or
// else-if directive.
func isValidConditionalPredecessor(node *TemplateNode) bool {
	return node != nil && (node.DirIf != nil || node.DirElseIf != nil)
}

// isModelableElement reports whether the given HTML tag name is a form element
// that can be bound to a model.
//
// Takes tagName (string) which is the HTML element tag name to check.
//
// Returns bool which is true for input, textarea, and select elements.
func isModelableElement(tagName string) bool {
	switch tagName {
	case "input", "textarea", "select":
		return true
	default:
		return false
	}
}

// isWhitespaceOnlyText reports whether the node is a text node that contains
// only whitespace characters.
//
// Takes node (*TemplateNode) which is the node to check.
//
// Returns bool which is true if the node is a text node with only whitespace.
func isWhitespaceOnlyText(node *TemplateNode) bool {
	if node.NodeType != NodeText {
		return false
	}
	if strings.TrimSpace(node.TextContent) != "" {
		return false
	}
	if len(node.RichText) > 0 {
		for _, part := range node.RichText {
			if !part.IsLiteral || strings.TrimSpace(part.Literal) != "" {
				return false
			}
		}
	}
	return true
}

// hasMeaningfulContent reports whether the node contains any elements or
// text that is not just whitespace.
//
// Takes node (*TemplateNode) which is the node to check.
//
// Returns bool which is true if the node has element children or text
// children with non-whitespace content.
func hasMeaningfulContent(node *TemplateNode) bool {
	if len(node.Children) == 0 {
		return false
	}
	for _, child := range node.Children {
		if child.NodeType == NodeElement {
			return true
		}
		if child.NodeType == NodeText && !isWhitespaceOnlyText(child) {
			return true
		}
	}
	return false
}

// getNodeSourcePath returns the source file path for a template node.
//
// Takes node (*TemplateNode) which is the node to get the path from.
// Takes tree (*TemplateAST) which provides a fallback path if the node has none.
//
// Returns string which is the source path, or empty if none is found.
func getNodeSourcePath(node *TemplateNode, tree *TemplateAST) string {
	if node != nil && node.GoAnnotations != nil && node.GoAnnotations.OriginalSourcePath != nil {
		return *node.GoAnnotations.OriginalSourcePath
	}
	if tree != nil && tree.SourcePath != nil {
		return *tree.SourcePath
	}
	return ""
}
