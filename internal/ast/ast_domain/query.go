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

// Implements CSS selector querying for AST nodes with support for combinators,
// pseudo-classes, and attribute matching. Provides QueryAll and related
// functions with cached selector parsing and precomputed tree relationships for
// efficient repeated queries.

import (
	"strconv"
	"strings"
	"sync"
)

const (
	// querySourcePath is the source file path used when reporting query errors.
	querySourcePath = "query.go"

	// maxQueryDepth is the limit for how deep queries can search through the
	// template tree. This prevents stack overflow errors in deeply nested
	// templates.
	maxQueryDepth = 10000

	// indexCapacityTags is the initial map capacity for tag-based node indexing.
	indexCapacityTags = 50

	// indexCapacityClasses is the initial map capacity for CSS class lookups.
	indexCapacityClasses = 50

	// indexCapacityIDs is the initial map capacity for ID-based element lookups.
	indexCapacityIDs = 20

	// indexCapacityElements is the initial capacity for the all elements slice.
	indexCapacityElements = 200
)

var (
	// selectorCache caches parsed selectors to avoid re-parsing identical
	// selector strings. This improves performance for repeated queries with the
	// same selectors.
	selectorCache sync.Map

	combinatorHandlers = map[string]combinatorHandler{
		" ": handleDescendantCombinator,
		">": handleChildCombinator,
		"+": handleAdjacentSiblingCombinator,
		"~": handleGeneralSiblingCombinator,
	}

	pseudoClassHandlers map[string]pseudoClassHandler
)

// cachedSelector holds a parsed selector set and its related diagnostics.
type cachedSelector struct {
	// set holds the parsed selector result.
	set SelectorSet

	// diagnostics holds any parse errors or warnings from parsing the selector.
	diagnostics []*Diagnostic
}

// effectiveTreeInfo tracks parent, sibling, and child links within a template
// tree. It includes indexes for fast query lookups.
type effectiveTreeInfo struct {
	// parentOf maps each node to its parent in the effective tree.
	parentOf map[*TemplateNode]*TemplateNode

	// siblingsOf maps each node to its list of sibling nodes, including itself.
	siblingsOf map[*TemplateNode][]*TemplateNode

	// childrenOf maps each node to its effective children, with fragments
	// expanded.
	childrenOf map[*TemplateNode][]*TemplateNode

	// attributeMapCache stores built attribute maps for nodes to avoid repeated work.
	attributeMapCache map[*TemplateNode]map[string]string

	// byTag maps tag names to matching elements for O(1) lookups
	// (e.g. "div" -> [node1, node2, ...]).
	byTag map[string][]*TemplateNode

	// byClass maps class names to all elements with that class ("button" ->
	// [node1, node2, ...]).
	byClass map[string][]*TemplateNode

	// byID maps ID values to elements with that ID ("header" -> [node1, ...]).
	// Supports duplicate IDs in invalid HTML.
	byID map[string][]*TemplateNode

	// allElements holds all element nodes for universal selector ("*") matching.
	allElements []*TemplateNode
}

// getOrBuildAttrMap returns a cached attribute map for the node, building it
// on first access.
//
// Takes node (*TemplateNode) which is the node to get attributes for.
//
// Returns map[string]string which maps attribute names in lowercase to their
// values.
func (info *effectiveTreeInfo) getOrBuildAttrMap(node *TemplateNode) map[string]string {
	if attrs, ok := info.attributeMapCache[node]; ok {
		return attrs
	}
	attrs := buildAttributeMap(node)
	if info.attributeMapCache == nil {
		info.attributeMapCache = make(map[*TemplateNode]map[string]string)
	}
	info.attributeMapCache[node] = attrs
	return attrs
}

// QueryContext provides efficient parent and sibling lookups for CSS selector
// matching.
type QueryContext struct {
	// virtualRoot is the starting node for tree walks; all nodes are its
	// descendants.
	virtualRoot *TemplateNode

	// treeInfo holds tree structure data used for child and sibling lookups.
	treeInfo effectiveTreeInfo
}

// getQueryContext returns the cached query context, building it if needed.
// Caching improves performance for repeated queries on the same AST.
//
// Returns *QueryContext which provides the query execution context for this
// AST.
func (ast *TemplateAST) getQueryContext() *QueryContext {
	if ast.queryContext == nil {
		ast.queryContext = newQueryContext(ast)
	}
	return ast.queryContext
}

// InvalidateQueryContext clears the cached query context. Call this after any
// changes to the AST structure, such as adding, removing, or moving nodes.
func (ast *TemplateAST) InvalidateQueryContext() {
	ast.queryContext = nil
}

// QueryAll finds all nodes that match a CSS selector, starting from this node
// as the root.
//
// Takes selector (string) which is the CSS selector to match against nodes.
// Takes sourcePath (string) which identifies the source for diagnostics.
//
// Returns []*TemplateNode which contains all matching nodes found.
// Returns []*Diagnostic which contains any issues found during the query.
//
// This method creates a new search context with this node as the root. This
// means selectors that depend on position in the full document (like sibling
// selectors + and ~, or :first-child) will work relative to this node, not
// the original document. For queries that need the full document context, use
// the top-level QueryAll function instead.
func (n *TemplateNode) QueryAll(selector, sourcePath string) ([]*TemplateNode, []*Diagnostic) {
	if n == nil {
		return nil, nil
	}
	tempAST := &TemplateAST{
		SourcePath:        nil,
		ExpiresAtUnixNano: nil,
		Metadata:          nil,
		queryContext:      nil,
		RootNodes:         []*TemplateNode{n},
		Diagnostics:       nil,
		SourceSize:        0,
		Tidied:            false,
		isPooled:          false,
	}
	return QueryAll(tempAST, selector, sourcePath)
}

// Query searches for the first node matching the CSS selector starting from
// this node as the root. See the documentation for TemplateNode.QueryAll for
// notes on context limitations.
//
// Takes selector (string) which specifies the CSS selector to match.
//
// Returns *TemplateNode which is the first matching node, or nil if none found.
// Returns []*Diagnostic which contains any warnings or errors from the query.
func (n *TemplateNode) Query(selector string) (*TemplateNode, []*Diagnostic) {
	results, diagnostics := n.QueryAll(selector, querySourcePath)
	if len(results) > 0 {
		return results[0], diagnostics
	}
	return nil, diagnostics
}

// MustQuery returns the first node matching the selector, ignoring
// diagnostics.
//
// Takes selector (string) which specifies the CSS-like query to match.
//
// Returns *TemplateNode which is the first matching node, or nil if none
// found.
//
// See the documentation for TemplateNode.QueryAll for notes on context
// limitations.
func (n *TemplateNode) MustQuery(selector string) *TemplateNode {
	results, _ := n.QueryAll(selector, querySourcePath)
	if len(results) > 0 {
		return results[0]
	}
	return nil
}

// combinatorHandler is a function type that handles a combinator selector and
// returns matching nodes.
type combinatorHandler func(startNode *TemplateNode, simple *SimpleSelector, qc *QueryContext) []*TemplateNode

// pseudoClassHandler is a function that checks if a template node matches a
// pseudo-class selector.
type pseudoClassHandler func(node *TemplateNode, pseudo PseudoClassSelector, qc *QueryContext) bool

// ClearSelectorCache resets the selector cache to an empty state.
// This is intended for test isolation between iterations.
func ClearSelectorCache() {
	selectorCache = sync.Map{}
}

// QueryAll searches the entire TemplateAST for all nodes matching the given
// CSS selector.
//
// This is the main query function that supports all combinators and
// pseudo-classes. The QueryContext is cached on the AST for efficient
// repeated queries.
//
// Takes selector (string) which specifies the CSS selector to match.
// Takes sourcePath (string) which identifies the source file for diagnostics.
//
// Returns []*TemplateNode which contains all matching nodes.
// Returns []*Diagnostic which contains any selector parsing errors.
func QueryAll(root *TemplateAST, selector, sourcePath string) ([]*TemplateNode, []*Diagnostic) {
	if root == nil || selector == "" {
		return nil, nil
	}

	selectorSet, diagnostics := parseSelector(selector, sourcePath)
	if diagnostics != nil {
		return nil, diagnostics
	}
	if len(selectorSet) == 0 {
		return nil, nil
	}

	queryContext := root.getQueryContext()

	var allMatches []*TemplateNode
	uniqueMatches := make(map[*TemplateNode]bool)

	for _, selectorGroup := range selectorSet {
		if len(selectorGroup) == 0 {
			continue
		}

		groupMatches := processSelectorGroup(queryContext.virtualRoot, selectorGroup, queryContext)
		allMatches = mergeUniqueMatches(allMatches, groupMatches, uniqueMatches)
	}

	return allMatches, nil
}

// Query finds the first node in a TemplateAST that matches a CSS selector.
//
// Takes selector (string) which specifies the CSS selector pattern to match.
//
// Returns *TemplateNode which is the first matching node, or nil if none is
// found.
// Returns []*Diagnostic which contains any warnings or errors from parsing.
func Query(root *TemplateAST, selector string) (*TemplateNode, []*Diagnostic) {
	results, diagnostics := QueryAll(root, selector, querySourcePath)
	if len(results) > 0 {
		return results[0], diagnostics
	}
	return nil, diagnostics
}

// MustQuery returns the first node that matches the selector, or nil if no
// match is found. It wraps QueryAll and ignores any diagnostics.
//
// Takes selector (string) which specifies the query pattern to match.
//
// Returns *TemplateNode which is the first matching node, or nil if none is
// found.
func MustQuery(root *TemplateAST, selector string) *TemplateNode {
	results, _ := QueryAll(root, selector, querySourcePath)
	if len(results) > 0 {
		return results[0]
	}
	return nil
}

// newQueryContext builds tree maps for fast parent, sibling, and child lookups.
//
// Takes root (*TemplateAST) which is the root node of the template AST.
//
// Returns *QueryContext which holds the precomputed tree information.
func newQueryContext(root *TemplateAST) *QueryContext {
	virtualRoot := newVirtualRootNode(root.RootNodes)
	return &QueryContext{
		treeInfo:    buildEffectiveTreeInfo(root, virtualRoot),
		virtualRoot: virtualRoot,
	}
}

// parseSelector parses a CSS selector string and returns the parsed result
// along with any errors found. Results are stored in a global cache.
//
// Takes selector (string) which is the CSS selector string to parse.
// Takes sourcePath (string) which identifies the source for error messages.
//
// Returns SelectorSet which contains the parsed selector parts.
// Returns []*Diagnostic which contains any parsing errors or warnings.
func parseSelector(selector, sourcePath string) (SelectorSet, []*Diagnostic) {
	cached, ok := selectorCache.Load(selector)
	if !ok {
		return parseSelectorUncached(selector, sourcePath)
	}

	cs, ok := cached.(cachedSelector)
	if !ok {
		selectorCache.Delete(selector)
		return parseSelectorUncached(selector, sourcePath)
	}
	return cs.set, cs.diagnostics
}

// parseSelectorUncached parses a selector string and stores the result in the
// cache.
//
// Takes selector (string) which is the selector text to parse.
// Takes sourcePath (string) which is the file path shown in error messages.
//
// Returns SelectorSet which holds the parsed selector rules.
// Returns []*Diagnostic which holds any parse errors found.
func parseSelectorUncached(selector, sourcePath string) (SelectorSet, []*Diagnostic) {
	queryLexer := NewQueryLexer(selector)
	parser := NewQueryParser(queryLexer, sourcePath)
	selectorSet := parser.Parse()

	var diagnostics []*Diagnostic
	if len(parser.Diagnostics()) > 0 {
		diagnostics = make([]*Diagnostic, len(parser.Diagnostics()))
		copy(diagnostics, parser.Diagnostics())
	}

	parser.Release()
	queryLexer.Release()

	selectorCache.Store(selector, cachedSelector{set: selectorSet, diagnostics: diagnostics})

	if diagnostics != nil {
		return nil, diagnostics
	}

	return selectorSet, nil
}

// processSelectorGroup finds all nodes that match a selector group.
//
// Takes virtualRoot (*TemplateNode) which is the root node to search from.
// Takes selectorGroup (SelectorGroup) which defines the selectors to match.
// Takes qc (*QueryContext) which holds the query settings.
//
// Returns []*TemplateNode which holds all nodes that match the selectors.
func processSelectorGroup(virtualRoot *TemplateNode, selectorGroup SelectorGroup, qc *QueryContext) []*TemplateNode {
	var currentMatches []*TemplateNode
	collectMatches(virtualRoot, &selectorGroup[0].Simple, &currentMatches, qc)

	for i := 1; i < len(selectorGroup); i++ {
		if len(currentMatches) == 0 {
			break
		}
		currentMatches = applyCombinator(currentMatches, selectorGroup[i], qc)
	}

	return currentMatches
}

// applyCombinator applies a single combinator to the current set of matched
// nodes.
//
// Takes currentMatches ([]*TemplateNode) which is the set of nodes to process.
// Takes complexSel (ComplexSelector) which specifies the combinator and simple
// selector to apply.
// Takes qc (*QueryContext) which provides the query context.
//
// Returns []*TemplateNode which contains the nodes that match after applying
// the combinator. Duplicates are removed from the result.
func applyCombinator(currentMatches []*TemplateNode, complexSel ComplexSelector, qc *QueryContext) []*TemplateNode {
	handler, ok := combinatorHandlers[complexSel.Combinator]
	if !ok {
		return currentMatches
	}

	var nextMatches []*TemplateNode
	for _, match := range currentMatches {
		nextMatches = append(nextMatches, handler(match, &complexSel.Simple, qc)...)
	}

	return deduplicateNodes(nextMatches)
}

// deduplicateNodes removes duplicate nodes from a slice while keeping the
// original order.
//
// Takes nodes ([]*TemplateNode) which is the slice to filter.
//
// Returns []*TemplateNode which contains only unique nodes in their original
// order.
func deduplicateNodes(nodes []*TemplateNode) []*TemplateNode {
	if len(nodes) == 0 {
		return nodes
	}

	uniqueNodes := make([]*TemplateNode, 0, len(nodes))
	seen := make(map[*TemplateNode]bool, len(nodes))

	for _, node := range nodes {
		if !seen[node] {
			seen[node] = true
			uniqueNodes = append(uniqueNodes, node)
		}
	}

	return uniqueNodes
}

// mergeUniqueMatches adds new matches to the list of all matches, keeping
// only those not already seen.
//
// Takes allMatches ([]*TemplateNode) which is the current list of unique
// matches.
// Takes newMatches ([]*TemplateNode) which holds the items to add.
// Takes uniqueMap (map[*TemplateNode]bool) which tracks nodes already added.
//
// Returns []*TemplateNode which is the updated list with any new unique
// matches added.
func mergeUniqueMatches(allMatches, newMatches []*TemplateNode, uniqueMap map[*TemplateNode]bool) []*TemplateNode {
	for _, match := range newMatches {
		if !uniqueMap[match] {
			uniqueMap[match] = true
			allMatches = append(allMatches, match)
		}
	}
	return allMatches
}

// handleDescendantCombinator finds all nodes at any depth that match a
// selector.
//
// Takes startNode (*TemplateNode) which is the root node to search from.
// Takes simple (*SimpleSelector) which defines the matching criteria.
// Takes qc (*QueryContext) which holds the query state.
//
// Returns []*TemplateNode which contains all matching nodes found.
func handleDescendantCombinator(startNode *TemplateNode, simple *SimpleSelector, qc *QueryContext) []*TemplateNode {
	var nextMatches []*TemplateNode
	for _, child := range qc.treeInfo.childrenOf[startNode] {
		collectMatches(child, simple, &nextMatches, qc)
	}
	return nextMatches
}

// handleChildCombinator finds direct children of startNode that match the
// selector.
//
// Takes startNode (*TemplateNode) which is the parent node to search from.
// Takes simple (*SimpleSelector) which defines the matching criteria.
// Takes qc (*QueryContext) which provides query state and options.
//
// Returns []*TemplateNode which contains matching child nodes, or nil if none
// match.
func handleChildCombinator(startNode *TemplateNode, simple *SimpleSelector, qc *QueryContext) []*TemplateNode {
	var nextMatches []*TemplateNode
	for _, child := range qc.treeInfo.childrenOf[startNode] {
		if matches(child, simple, qc) {
			nextMatches = append(nextMatches, child)
		}
	}
	return nextMatches
}

// handleAdjacentSiblingCombinator finds the next sibling node that matches the
// selector.
//
// Takes startNode (*TemplateNode) which is the node to find the sibling of.
// Takes simple (*SimpleSelector) which sets the matching rules.
// Takes qc (*QueryContext) which provides tree structure data.
//
// Returns []*TemplateNode which holds the matching sibling, or nil if none is
// found.
func handleAdjacentSiblingCombinator(startNode *TemplateNode, simple *SimpleSelector, qc *QueryContext) []*TemplateNode {
	siblings := qc.treeInfo.siblingsOf[startNode]
	for j, sibling := range siblings {
		if sibling == startNode && j+1 < len(siblings) {
			nextSibling := siblings[j+1]
			if matches(nextSibling, simple, qc) {
				return []*TemplateNode{nextSibling}
			}
			break
		}
	}
	return nil
}

// handleGeneralSiblingCombinator finds all sibling nodes that come after
// startNode and match the given selector.
//
// Takes startNode (*TemplateNode) which is the reference node to find siblings
// after.
// Takes simple (*SimpleSelector) which defines the matching criteria.
// Takes qc (*QueryContext) which provides the tree structure.
//
// Returns []*TemplateNode which contains all matching siblings in document
// order.
func handleGeneralSiblingCombinator(startNode *TemplateNode, simple *SimpleSelector, qc *QueryContext) []*TemplateNode {
	var nextMatches []*TemplateNode
	siblings := qc.treeInfo.siblingsOf[startNode]
	foundStartNode := false
	for _, sibling := range siblings {
		if sibling == startNode {
			foundStartNode = true
			continue
		}
		if foundStartNode && matches(sibling, simple, qc) {
			nextMatches = append(nextMatches, sibling)
		}
	}
	return nextMatches
}

// matches checks whether a node satisfies a simple selector.
//
// Takes node (*TemplateNode) which is the node to test.
// Takes simple (*SimpleSelector) which defines the matching criteria.
// Takes qc (*QueryContext) which provides context for pseudo-class matching.
//
// Returns bool which is true if the node matches all selector criteria.
func matches(node *TemplateNode, simple *SimpleSelector, qc *QueryContext) bool {
	if !matchesCore(node, simple) {
		return false
	}
	if !matchesAttributes(node, simple.Attributes, qc) {
		return false
	}
	if !matchesAllPseudoClasses(node, simple.PseudoClasses, qc) {
		return false
	}
	return true
}

// matchesCore checks whether a template node matches a simple CSS selector.
//
// Takes node (*TemplateNode) which is the element to test.
// Takes simple (*SimpleSelector) which holds the tag name, ID, and class
// names to match.
//
// Returns bool which is true if the node matches all parts of the selector.
func matchesCore(node *TemplateNode, simple *SimpleSelector) bool {
	if node.NodeType != NodeElement {
		return false
	}

	if simple.Tag != "" && simple.Tag != "*" && node.TagName != simple.Tag {
		return false
	}

	if simple.ID != "" {
		idVal, ok := node.GetAttribute("id")
		if !ok || idVal != simple.ID {
			return false
		}
	}

	if len(simple.Classes) > 0 {
		for _, class := range simple.Classes {
			if !node.HasClass(class) {
				return false
			}
		}
	}

	return true
}

// matchesAttributes checks whether a template node matches all the given
// attribute selectors.
//
// Takes node (*TemplateNode) which is the node to check.
// Takes attributeSelectors ([]AttributeSelector) which lists the attribute
// conditions to match.
// Takes qc (*QueryContext) which provides the attribute map cache.
//
// Returns bool which is true if all selectors match or if none are given.
func matchesAttributes(node *TemplateNode, attributeSelectors []AttributeSelector, qc *QueryContext) bool {
	if len(attributeSelectors) == 0 {
		return true
	}

	var nodeAttrs map[string]string

	for _, attributeSelector := range attributeSelectors {
		if !matchesSingleAttribute(node, attributeSelector, &nodeAttrs, qc) {
			return false
		}
	}

	return true
}

// buildAttributeMap creates a map from attribute names to their values.
//
// Includes both static Attributes and dynamic AttributeWriters. Static
// attributes take precedence. If an attribute exists in both, the static
// value is used.
//
// Takes node (*TemplateNode) which contains the attributes to map.
//
// Returns map[string]string which maps lowercase attribute names to their
// values.
func buildAttributeMap(node *TemplateNode) map[string]string {
	attrs := make(map[string]string, len(node.Attributes)+len(node.AttributeWriters))

	for i := range node.Attributes {
		attr := &node.Attributes[i]
		attrs[strings.ToLower(attr.Name)] = attr.Value
	}

	for _, dw := range node.AttributeWriters {
		if dw == nil {
			continue
		}
		name := strings.ToLower(dw.Name)
		if _, exists := attrs[name]; exists {
			continue
		}
		if s, ok := dw.SingleStringValue(); ok {
			attrs[name] = s
		} else {
			attrs[name] = dw.StringRaw()
		}
	}
	return attrs
}

// matchesSingleAttribute checks whether a node matches a single attribute
// selector.
//
// Takes node (*TemplateNode) which is the node to check.
// Takes attributeSelector (AttributeSelector) which specifies the
// attribute condition.
// Takes nodeAttrs (*map[string]string) which caches the node's attributes
// within the current matchesAttributes call.
// Takes qc (*QueryContext) which provides the shared attribute map cache.
//
// Returns bool which is true if the node matches the attribute selector.
func matchesSingleAttribute(node *TemplateNode, attributeSelector AttributeSelector, nodeAttrs *map[string]string, qc *QueryContext) bool {
	if dirType, isDirective := DirectiveNameToType[attributeSelector.Name]; isDirective {
		return node.HasDirective(dirType)
	}

	if *nodeAttrs == nil {
		*nodeAttrs = qc.treeInfo.getOrBuildAttrMap(node)
	}

	nodeVal, ok := (*nodeAttrs)[strings.ToLower(attributeSelector.Name)]

	if attributeSelector.Operator == "" {
		return ok
	}

	if !ok {
		return false
	}

	return matchesAttributeOperator(nodeVal, attributeSelector)
}

// matchesAttributeOperator checks if a node value matches an attribute
// selector based on the operator.
//
// Takes nodeVal (string) which is the value from the node to compare.
// Takes attributeSelector (AttributeSelector) which holds the operator and value.
//
// Returns bool which is true if the node value matches the selector.
func matchesAttributeOperator(nodeVal string, attributeSelector AttributeSelector) bool {
	compareNodeVal, compareAttrVal := nodeVal, attributeSelector.Value
	if attributeSelector.CaseInsensitive {
		compareNodeVal = strings.ToLower(compareNodeVal)
		compareAttrVal = strings.ToLower(compareAttrVal)
	}

	switch attributeSelector.Operator {
	case "=":
		return compareNodeVal == compareAttrVal
	case "~=":
		return matchesWordInList(compareNodeVal, compareAttrVal)
	case "|=":
		return matchesDashPrefix(compareNodeVal, compareAttrVal)
	case "^=":
		return strings.HasPrefix(compareNodeVal, compareAttrVal)
	case "$=":
		return strings.HasSuffix(compareNodeVal, compareAttrVal)
	case "*=":
		return strings.Contains(compareNodeVal, compareAttrVal)
	default:
		return false
	}
}

// matchesWordInList checks whether a word appears in a space-separated list.
//
// Takes nodeVal (string) which is the list of words to search, separated by
// spaces.
// Takes targetWord (string) which is the word to find.
//
// Returns bool which is true if the word is found, false otherwise.
func matchesWordInList(nodeVal string, targetWord string) bool {
	for word := range strings.FieldsSeq(nodeVal) {
		if word == targetWord {
			return true
		}
	}
	return false
}

// matchesDashPrefix reports whether nodeVal equals prefix or starts with
// prefix followed by a dash.
//
// Takes nodeVal (string) which is the value to check.
// Takes prefix (string) which is the prefix to match against.
//
// Returns bool which is true if nodeVal matches the prefix pattern.
func matchesDashPrefix(nodeVal string, prefix string) bool {
	return nodeVal == prefix || strings.HasPrefix(nodeVal, prefix+"-")
}

// matchesAllPseudoClasses checks if a node matches all pseudo-class selectors.
//
// Takes node (*TemplateNode) which is the node to check.
// Takes pseudos ([]PseudoClassSelector) which lists the selectors to match.
// Takes qc (*QueryContext) which provides the query context.
//
// Returns bool which is true only if all pseudo-classes match.
func matchesAllPseudoClasses(node *TemplateNode, pseudos []PseudoClassSelector, qc *QueryContext) bool {
	for _, pseudo := range pseudos {
		handler, ok := pseudoClassHandlers[pseudo.Type]
		if !ok || !handler(node, pseudo, qc) {
			return false
		}
	}
	return true
}

// handlePseudoNot checks if a node does not match the :not() pseudo-class.
//
// Takes node (*TemplateNode) which is the node to check.
// Takes pseudo (PseudoClassSelector) which holds the selector to negate.
// Takes qc (*QueryContext) which holds the query state.
//
// Returns bool which is true if the node does not match the inner selector.
func handlePseudoNot(node *TemplateNode, pseudo PseudoClassSelector, qc *QueryContext) bool {
	return pseudo.SubSelector != nil && !matches(node, pseudo.SubSelector, qc)
}

// handlePseudoFirstChild matches the :first-child CSS pseudo-class selector.
//
// Takes node (*TemplateNode) which is the node to test.
// Takes qc (*QueryContext) which provides sibling data for the check.
//
// Returns bool which is true if the node is the first child among its
// siblings.
func handlePseudoFirstChild(node *TemplateNode, _ PseudoClassSelector, qc *QueryContext) bool {
	siblings := qc.treeInfo.siblingsOf[node]
	return len(siblings) > 0 && siblings[0] == node
}

// handlePseudoLastChild matches the :last-child CSS pseudo-class selector.
//
// Takes node (*TemplateNode) which is the node to test.
// Takes qc (*QueryContext) which provides sibling data for the check.
//
// Returns bool which is true if the node is the last child among its
// siblings.
func handlePseudoLastChild(node *TemplateNode, _ PseudoClassSelector, qc *QueryContext) bool {
	siblings := qc.treeInfo.siblingsOf[node]
	return len(siblings) > 0 && siblings[len(siblings)-1] == node
}

// handlePseudoOnlyChild matches the :only-child CSS pseudo-class selector.
//
// Takes node (*TemplateNode) which is the node to test.
// Takes qc (*QueryContext) which provides sibling data for the check.
//
// Returns bool which is true if the node is the only child among its
// siblings.
func handlePseudoOnlyChild(node *TemplateNode, _ PseudoClassSelector, qc *QueryContext) bool {
	siblings := qc.treeInfo.siblingsOf[node]
	return len(siblings) == 1 && siblings[0] == node
}

// handlePseudoNthChild checks if a node matches an :nth-child selector.
//
// Takes node (*TemplateNode) which is the element to test.
// Takes pseudo (PseudoClassSelector) which holds the nth-child pattern.
// Takes qc (*QueryContext) which provides sibling data.
//
// Returns bool which is true if the node position matches the pattern.
func handlePseudoNthChild(node *TemplateNode, pseudo PseudoClassSelector, qc *QueryContext) bool {
	siblings := qc.treeInfo.siblingsOf[node]
	for i, s := range siblings {
		if s == node {
			return matchesNth(i+1, pseudo.Value)
		}
	}
	return false
}

// handlePseudoFirstOfType matches the :first-of-type CSS pseudo-class
// selector.
//
// Takes node (*TemplateNode) which is the node to test.
// Takes qc (*QueryContext) which provides sibling data for the check.
//
// Returns bool which is true if the node is the first sibling with its tag
// name.
func handlePseudoFirstOfType(node *TemplateNode, _ PseudoClassSelector, qc *QueryContext) bool {
	siblings := qc.treeInfo.siblingsOf[node]
	for _, s := range siblings {
		if s.TagName == node.TagName {
			return s == node
		}
	}
	return false
}

// handlePseudoLastOfType matches the :last-of-type CSS pseudo-class selector.
//
// Takes node (*TemplateNode) which is the node to test.
// Takes qc (*QueryContext) which provides sibling data for the check.
//
// Returns bool which is true if the node is the last sibling with its tag
// name.
func handlePseudoLastOfType(node *TemplateNode, _ PseudoClassSelector, qc *QueryContext) bool {
	siblings := qc.treeInfo.siblingsOf[node]
	for i := len(siblings) - 1; i >= 0; i-- {
		if siblings[i].TagName == node.TagName {
			return siblings[i] == node
		}
	}
	return false
}

// handlePseudoOnlyOfType matches the :only-of-type CSS pseudo-class selector.
//
// Takes node (*TemplateNode) which is the node to test.
// Takes qc (*QueryContext) which provides sibling data for the check.
//
// Returns bool which is true if the node is the only sibling with its tag
// name.
func handlePseudoOnlyOfType(node *TemplateNode, _ PseudoClassSelector, qc *QueryContext) bool {
	siblings := qc.treeInfo.siblingsOf[node]
	count := 0
	for _, sibling := range siblings {
		if sibling.TagName == node.TagName {
			count++
			if count > 1 {
				return false
			}
		}
	}
	return count == 1
}

// handlePseudoNthOfType checks if a node matches the :nth-of-type selector.
//
// Takes node (*TemplateNode) which is the node to test.
// Takes pseudo (PseudoClassSelector) which holds the nth formula.
// Takes qc (*QueryContext) which provides access to sibling data.
//
// Returns bool which is true if the node is at the correct position among
// siblings with the same tag name.
func handlePseudoNthOfType(node *TemplateNode, pseudo PseudoClassSelector, qc *QueryContext) bool {
	siblings := qc.treeInfo.siblingsOf[node]
	count := 0
	for _, s := range siblings {
		if s.TagName == node.TagName {
			count++
			if s == node {
				return matchesNth(count, pseudo.Value)
			}
		}
	}
	return false
}

// handlePseudoNthLastChild checks if a node matches an :nth-last-child
// selector.
//
// Takes node (*TemplateNode) which is the node to check.
// Takes pseudo (PseudoClassSelector) which holds the nth expression.
// Takes qc (*QueryContext) which gives access to sibling data.
//
// Returns bool which is true if the node matches the nth-last-child pattern.
func handlePseudoNthLastChild(node *TemplateNode, pseudo PseudoClassSelector, qc *QueryContext) bool {
	siblings := qc.treeInfo.siblingsOf[node]
	for i, s := range siblings {
		if s == node {
			lastIndex := len(siblings) - i
			return matchesNth(lastIndex, pseudo.Value)
		}
	}
	return false
}

// handlePseudoNthLastOfType checks if a node matches the :nth-last-of-type
// pseudo-class selector.
//
// Uses a single-pass method to find matching siblings and node position,
// avoiding extra loops for better speed.
//
// Takes node (*TemplateNode) which is the node to check.
// Takes pseudo (PseudoClassSelector) which holds the nth expression.
// Takes qc (*QueryContext) which provides sibling data.
//
// Returns bool which is true if the node matches the selector.
func handlePseudoNthLastOfType(node *TemplateNode, pseudo PseudoClassSelector, qc *QueryContext) bool {
	siblings := qc.treeInfo.siblingsOf[node]

	var matchingSiblings []*TemplateNode
	nodeIndex := -1

	for _, s := range siblings {
		if s.TagName == node.TagName {
			if s == node {
				nodeIndex = len(matchingSiblings)
			}
			matchingSiblings = append(matchingSiblings, s)
		}
	}

	if nodeIndex == -1 {
		return false
	}

	lastIndex := len(matchingSiblings) - nodeIndex
	return matchesNth(lastIndex, pseudo.Value)
}

// canUseIndex determines if a simple selector can use query indexes
// for O(1) lookups. It returns true for simple tag, class, or ID
// selectors without complex filters.
//
// Takes simple (*SimpleSelector) which defines the matching criteria.
//
// Returns bool which is true if indexes can be used.
func canUseIndex(simple *SimpleSelector) bool {
	hasIndexableSelector := simple.ID != "" || simple.Tag != "" || len(simple.Classes) > 0
	hasComplexFilters := len(simple.Attributes) > 0 || len(simple.PseudoClasses) > 0
	return hasIndexableSelector && !hasComplexFilters
}

// getCandidatesFromIndex retrieves candidate nodes from query indexes using
// O(1) lookups.
// Returns nil if no candidates are found.
//
// Takes simple (*SimpleSelector) which specifies what to look up.
// Takes info (effectiveTreeInfo) which provides the query indexes.
//
// Returns []*TemplateNode which contains candidate nodes for further filtering.
func getCandidatesFromIndex(simple *SimpleSelector, info effectiveTreeInfo) []*TemplateNode {
	if simple.ID != "" {
		return info.byID[simple.ID]
	}

	if simple.Tag != "" && simple.Tag != "*" {
		return info.byTag[simple.Tag]
	}

	if len(simple.Classes) > 0 {
		return info.byClass[simple.Classes[0]]
	}

	return info.allElements
}

// isDescendantOf checks if a candidate node is a descendant of the start node
// by traversing the parent map in O(log n) time, handling the virtual root
// case where root-level nodes have nil as their effective parent.
//
// Takes candidate (*TemplateNode) which is the node to check.
// Takes startNode (*TemplateNode) which is the ancestor to search for.
// Takes qc (*QueryContext) which provides the parent map and virtual root.
//
// Returns bool which is true if candidate is a descendant of startNode.
func isDescendantOf(candidate *TemplateNode, startNode *TemplateNode, qc *QueryContext) bool {
	if candidate == startNode {
		return true
	}

	if startNode == qc.virtualRoot {
		return true
	}

	current := qc.treeInfo.parentOf[candidate]
	for current != nil {
		if current == startNode {
			return true
		}
		current = qc.treeInfo.parentOf[current]
	}
	return false
}

// collectMatches finds all nodes that match a simple selector within a subtree.
// Uses query indexes for O(1) lookups when possible, falling back to tree
// traversal for complex selectors.
//
// Takes startNode (*TemplateNode) which is the root of the subtree to search.
// Takes simple (*SimpleSelector) which defines the matching criteria.
// Takes results (*[]*TemplateNode) which collects the matched nodes.
// Takes qc (*QueryContext) which provides query state and options.
func collectMatches(startNode *TemplateNode, simple *SimpleSelector, results *[]*TemplateNode, qc *QueryContext) {
	if canUseIndex(simple) {
		candidates := getCandidatesFromIndex(simple, qc.treeInfo)

		for _, candidate := range candidates {
			if isDescendantOf(candidate, startNode, qc) && matches(candidate, simple, qc) {
				*results = append(*results, candidate)
			}
		}
		return
	}

	collectMatchesWithDepth(startNode, simple, results, qc, 0)
}

// collectMatchesWithDepth finds nodes that match a selector by walking the
// tree up to a set depth limit.
//
// Takes startNode (*TemplateNode) which is the root node to start from.
// Takes simple (*SimpleSelector) which defines the matching rules.
// Takes results (*[]*TemplateNode) which collects the matched nodes.
// Takes qc (*QueryContext) which provides context for query checks.
// Takes depth (int) which tracks how deep the search has gone.
func collectMatchesWithDepth(startNode *TemplateNode, simple *SimpleSelector, results *[]*TemplateNode, qc *QueryContext, depth int) {
	if startNode == nil || depth > maxQueryDepth {
		return
	}
	if startNode.NodeType == NodeElement && matches(startNode, simple, qc) {
		*results = append(*results, startNode)
	}
	for _, child := range qc.treeInfo.childrenOf[startNode] {
		collectMatchesWithDepth(child, simple, results, qc, depth+1)
	}
}

// buildEffectiveTreeInfo creates parent, sibling, and children maps for the
// effective tree, treating fragments as invisible wrappers.
//
// Takes root (*TemplateAST) which provides the template tree to analyse.
// Takes virtualRoot (*TemplateNode) which is the virtual root node for tree
// traversal.
//
// Returns effectiveTreeInfo which contains the computed parent, sibling, and
// children relationships.
func buildEffectiveTreeInfo(root *TemplateAST, virtualRoot *TemplateNode) effectiveTreeInfo {
	_ = root

	info := createEmptyTreeInfo()
	walkAndBuildIndexes(virtualRoot, nil, 0, &info)
	return info
}

// createEmptyTreeInfo initialises an empty tree info structure with proper
// capacities for parent, sibling, child, and index maps.
//
// Returns effectiveTreeInfo which is ready to be populated with tree data.
func createEmptyTreeInfo() effectiveTreeInfo {
	return effectiveTreeInfo{
		parentOf:    make(map[*TemplateNode]*TemplateNode),
		siblingsOf:  make(map[*TemplateNode][]*TemplateNode),
		childrenOf:  make(map[*TemplateNode][]*TemplateNode),
		byTag:       make(map[string][]*TemplateNode, indexCapacityTags),
		byClass:     make(map[string][]*TemplateNode, indexCapacityClasses),
		byID:        make(map[string][]*TemplateNode, indexCapacityIDs),
		allElements: make([]*TemplateNode, 0, indexCapacityElements),
	}
}

// walkAndBuildIndexes recursively walks the tree and builds query indexes.
//
// Takes node (*TemplateNode) which is the current node to process.
// Takes effectiveParent (*TemplateNode) which is the logical parent for
// indexing purposes.
// Takes depth (int) which tracks recursion depth to prevent infinite loops.
// Takes info (*effectiveTreeInfo) which accumulates the index data.
func walkAndBuildIndexes(node *TemplateNode, effectiveParent *TemplateNode, depth int, info *effectiveTreeInfo) {
	if node == nil || depth > maxQueryDepth {
		return
	}

	effectiveChildren := getEffectiveChildrenWithDepth(node, 0)
	info.childrenOf[node] = effectiveChildren

	currentEffectiveParent := determineEffectiveParent(node, effectiveParent)
	indexElementNode(node, info)
	processChildren(effectiveChildren, currentEffectiveParent, depth, info)
}

// determineEffectiveParent returns the effective parent for child nodes.
//
// Takes node (*TemplateNode) which is the current node to evaluate.
// Takes effectiveParent (*TemplateNode) which is the fallback parent for
// fragment nodes.
//
// Returns *TemplateNode which is the node itself unless it is a fragment, in
// which case the effective parent is returned.
func determineEffectiveParent(node *TemplateNode, effectiveParent *TemplateNode) *TemplateNode {
	if node.NodeType != NodeFragment {
		return node
	}
	return effectiveParent
}

// indexElementNode builds query indexes for element nodes.
//
// Takes node (*TemplateNode) which is the node to index.
// Takes info (*effectiveTreeInfo) which holds the index structures to update.
func indexElementNode(node *TemplateNode, info *effectiveTreeInfo) {
	if node.NodeType != NodeElement {
		return
	}

	info.allElements = append(info.allElements, node)
	info.byTag[node.TagName] = append(info.byTag[node.TagName], node)

	indexByID(node, info)
	indexByClass(node, info)
}

// indexByID adds the node to the ID index if it has an ID attribute.
//
// Takes node (*TemplateNode) which is the node to index.
// Takes info (*effectiveTreeInfo) which holds the index map to update.
func indexByID(node *TemplateNode, info *effectiveTreeInfo) {
	for i := range node.Attributes {
		attr := &node.Attributes[i]
		if attr.Name == "id" && attr.Value != "" {
			info.byID[attr.Value] = append(info.byID[attr.Value], node)
			return
		}
	}
}

// indexByClass adds the node to class indexes for each class it has.
//
// Takes node (*TemplateNode) which is the node to index by its classes.
// Takes info (*effectiveTreeInfo) which holds the class index to update.
func indexByClass(node *TemplateNode, info *effectiveTreeInfo) {
	for i := range node.Attributes {
		attr := &node.Attributes[i]
		if attr.Name == "class" && attr.Value != "" {
			for cls := range strings.FieldsSeq(attr.Value) {
				info.byClass[cls] = append(info.byClass[cls], node)
			}
			return
		}
	}
}

// processChildren sets up parent and sibling relationships and recurses.
//
// Takes children ([]*TemplateNode) which is the list of child nodes to process.
// Takes parent (*TemplateNode) which is the parent node for these children.
// Takes depth (int) which is the current depth in the tree.
// Takes info (*effectiveTreeInfo) which stores the relationship mappings.
func processChildren(children []*TemplateNode, parent *TemplateNode, depth int, info *effectiveTreeInfo) {
	for _, child := range children {
		info.parentOf[child] = parent
		info.siblingsOf[child] = children
		walkAndBuildIndexes(child, parent, depth+1, info)
	}
}

// newVirtualRootNode creates a fragment node that acts as a virtual root for
// tree traversal.
//
// Takes children ([]*TemplateNode) which provides the child nodes to attach to
// the virtual root.
//
// Returns *TemplateNode which is a fragment node with all fields set to zero
// values except Children.
func newVirtualRootNode(children []*TemplateNode) *TemplateNode {
	return &TemplateNode{
		Key: nil, DirKey: nil, DirHTML: nil, GoAnnotations: nil, RuntimeAnnotations: nil,
		AttributeWriters: nil, TextContentWriter: nil,
		CustomEvents: nil, OnEvents: nil, Binds: nil, DirContext: nil,
		DirElse: nil, DirText: nil, DirStyle: nil, DirClass: nil, DirIf: nil, DirElseIf: nil,
		DirFor: nil, DirShow: nil, DirRef: nil, DirSlot: nil, DirModel: nil, DirScaffold: nil,
		TagName: "", TextContent: "", InnerHTML: "", Children: children,
		RichText: nil, Attributes: nil, Diagnostics: nil, DynamicAttributes: nil, Directives: nil,
		Location: Location{}, NodeType: NodeFragment, IsPooled: false, IsContentEditable: false,
		NodeRange: Range{}, OpeningTagRange: Range{}, ClosingTagRange: Range{}, PreferredFormat: FormatAuto,
	}
}

// getEffectiveChildrenWithDepth returns the child nodes of a parent node,
// skipping fragment nodes and looking inside them up to the given depth.
//
// Takes parent (*TemplateNode) which is the node to get children from.
// Takes depth (int) which tracks how deep the search has gone to prevent
// infinite loops.
//
// Returns []*TemplateNode which contains element children, with fragment
// contents included directly.
func getEffectiveChildrenWithDepth(parent *TemplateNode, depth int) []*TemplateNode {
	if parent == nil || depth > maxQueryDepth {
		return nil
	}
	var effectiveChildren []*TemplateNode
	for _, child := range parent.Children {
		switch child.NodeType {
		case NodeFragment:
			effectiveChildren = append(effectiveChildren, getEffectiveChildrenWithDepth(child, depth+1)...)
		case NodeElement:
			effectiveChildren = append(effectiveChildren, child)
		default:
		}
	}
	return effectiveChildren
}

// matchesNth checks if a position matches a CSS nth-child style formula.
//
// The formula follows the "an+b" pattern used in CSS selectors. It also
// accepts keywords like "odd" and "even", or simple numbers.
//
// Takes index (int) which is the position to test against the formula.
// Takes formula (string) which is the nth-child formula to match.
//
// Returns bool which is true if the index matches the formula.
func matchesNth(index int, formula string) bool {
	formula = strings.TrimSpace(strings.ToLower(formula))

	if matched, result := matchNthKeyword(index, formula); matched {
		return result
	}

	if !strings.Contains(formula, bigIntSuffix) {
		return matchNthSimpleNumber(index, formula)
	}

	return matchNthFormula(index, formula)
}

// matchNthKeyword checks if an index matches the "odd" or "even" keyword.
//
// Takes index (int) which is the position to check (must be greater than 0).
// Takes formula (string) which is the keyword to match ("odd" or "even").
//
// Returns matched (bool) which is true if the formula was a known keyword.
// Returns result (bool) which is true if the index matches the keyword.
func matchNthKeyword(index int, formula string) (matched bool, result bool) {
	switch formula {
	case "odd":
		return true, index > 0 && index%2 != 0
	case "even":
		return true, index > 0 && index%2 == 0
	default:
		return false, false
	}
}

// matchNthSimpleNumber checks whether the index matches a simple number
// formula.
//
// Takes index (int) which is the position to check.
// Takes formula (string) which is the number to match as text.
//
// Returns bool which is true when the index equals the parsed formula number,
// or false when the formula is not a valid number.
func matchNthSimpleNumber(index int, formula string) bool {
	b, err := strconv.Atoi(formula)
	if err != nil {
		return false
	}
	return index == b
}

// matchNthFormula checks if an index matches an An+B formula pattern.
//
// Formulas follow CSS nth-child syntax such as "2n+1", "n", or "-n+3".
//
// Takes index (int) which is the position to test (must be greater than zero).
// Takes formula (string) which is the An+B expression to match against.
//
// Returns bool which is true when the index matches the formula pattern.
func matchNthFormula(index int, formula string) bool {
	a, b, ok := parseNthFormulaCoefficients(formula)
	if !ok {
		return false
	}

	if index <= 0 {
		return false
	}

	return nthFormulaMatches(index, a, b)
}

// parseNthFormulaCoefficients extracts the coefficients from an An+B formula.
//
// Takes formula (string) which contains the An+B expression to parse.
//
// Returns a (int) which is the coefficient of n.
// Returns b (int) which is the constant offset.
// Returns ok (bool) which is true when parsing was successful.
func parseNthFormulaCoefficients(formula string) (a int, b int, ok bool) {
	parts := strings.Split(formula, bigIntSuffix)
	aString := strings.TrimSpace(parts[0])

	switch aString {
	case "", "+":
		a = 1
	case "-":
		a = -1
	default:
		var err error
		a, err = strconv.Atoi(aString)
		if err != nil {
			return 0, 0, false
		}
	}

	if len(parts) > 1 && parts[1] != "" {
		bString := strings.ReplaceAll(parts[1], " ", "")
		var err error
		b, err = strconv.Atoi(bString)
		if err != nil {
			return 0, 0, false
		}
	}

	return a, b, true
}

// nthFormulaMatches checks if an index matches the An+B formula.
//
// Takes index (int) which is the position to test.
// Takes a (int) which is the step size.
// Takes b (int) which is the starting offset.
//
// Returns bool which is true when the index fits the formula An+B.
func nthFormulaMatches(index int, a int, b int) bool {
	if a == 0 {
		return index == b
	}
	if a > 0 {
		return index >= b && (index-b)%a == 0
	}
	return index <= b && (index-b)%a == 0
}

func init() {
	pseudoClassHandlers = map[string]pseudoClassHandler{
		"not":              handlePseudoNot,
		"first-child":      handlePseudoFirstChild,
		"last-child":       handlePseudoLastChild,
		"only-child":       handlePseudoOnlyChild,
		"nth-child":        handlePseudoNthChild,
		"first-of-type":    handlePseudoFirstOfType,
		"last-of-type":     handlePseudoLastOfType,
		"only-of-type":     handlePseudoOnlyOfType,
		"nth-of-type":      handlePseudoNthOfType,
		"nth-last-child":   handlePseudoNthLastChild,
		"nth-last-of-type": handlePseudoNthLastOfType,
	}
}
