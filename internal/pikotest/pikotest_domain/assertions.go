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

package pikotest_domain

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/ast/ast_domain"
)

// ASTQueryResult wraps a set of AST nodes returned by a CSS selector query
// and provides assertion methods for testing.
type ASTQueryResult struct {
	// tb is the test context for reporting failures and marking helpers.
	tb testing.TB

	// nodes holds the AST template nodes that matched the query.
	nodes []*ast_domain.TemplateNode
}

// Nodes returns the raw slice of matched nodes.
//
// Returns []*ast_domain.TemplateNode which contains the query results.
func (r *ASTQueryResult) Nodes() []*ast_domain.TemplateNode {
	return r.nodes
}

// Len returns the number of matched nodes.
//
// Returns int which is the count of nodes in the result set.
func (r *ASTQueryResult) Len() int {
	return len(r.nodes)
}

// First returns the first matched node, or nil if no nodes matched.
//
// Returns *ast_domain.TemplateNode which is the first matched node or nil.
func (r *ASTQueryResult) First() *ast_domain.TemplateNode {
	if len(r.nodes) == 0 {
		return nil
	}
	return r.nodes[0]
}

// FirstResult returns a new ASTQueryResult containing only the first node,
// enabling chained assertion methods on the first node.
//
// Returns *ASTQueryResult which contains only the first node from the results.
func (r *ASTQueryResult) FirstResult() *ASTQueryResult {
	return r.Index(0)
}

// Last returns the last matched node, or nil if no nodes matched.
//
// Returns *ast_domain.TemplateNode which is the final node in the result set.
func (r *ASTQueryResult) Last() *ast_domain.TemplateNode {
	if len(r.nodes) == 0 {
		return nil
	}
	return r.nodes[len(r.nodes)-1]
}

// At returns the node at the given index, or nil if index is out of bounds.
//
// Takes index (int) which specifies the position of the node to retrieve.
//
// Returns *ast_domain.TemplateNode which is the node at the given position,
// or nil if the index is out of bounds.
func (r *ASTQueryResult) At(index int) *ast_domain.TemplateNode {
	if index < 0 || index >= len(r.nodes) {
		return nil
	}
	return r.nodes[index]
}

// Index returns a new ASTQueryResult containing only the node at the given
// index, enabling chained assertion methods on a specific node.
//
// Takes index (int) which specifies the position of the node to select.
//
// Returns *ASTQueryResult which contains only the selected node, or an empty
// result if the index is out of bounds.
func (r *ASTQueryResult) Index(index int) *ASTQueryResult {
	r.tb.Helper()

	if index < 0 || index >= len(r.nodes) {
		r.tb.Errorf("Index %d out of bounds (have %d nodes)", index, len(r.nodes))
		return &ASTQueryResult{tb: r.tb, nodes: nil}
	}

	return &ASTQueryResult{
		tb:    r.tb,
		nodes: []*ast_domain.TemplateNode{r.nodes[index]},
	}
}

// Exists checks that at least one node was matched.
//
// Returns *ASTQueryResult which allows method chaining for further checks.
func (r *ASTQueryResult) Exists() *ASTQueryResult {
	r.tb.Helper()
	if len(r.nodes) == 0 {
		r.tb.Error("Expected at least one node to match, but found none")
	}
	return r
}

// NotExists asserts that no nodes were matched.
//
// Returns *ASTQueryResult which allows method chaining for further assertions.
func (r *ASTQueryResult) NotExists() *ASTQueryResult {
	r.tb.Helper()
	if len(r.nodes) > 0 {
		r.tb.Errorf("Expected no nodes to match, but found %d", len(r.nodes))
	}
	return r
}

// Count asserts that exactly the expected number of nodes were matched.
//
// Takes expected (int) which specifies the required node count.
//
// Returns *ASTQueryResult which allows method chaining for further assertions.
func (r *ASTQueryResult) Count(expected int) *ASTQueryResult {
	r.tb.Helper()
	assert.Equal(r.tb, expected, len(r.nodes), "Node count mismatch")
	return r
}

// MinCount asserts that at least the expected number of nodes were matched.
//
// Takes minCount (int) which specifies the minimum number of nodes required.
//
// Returns *ASTQueryResult which allows method chaining for further assertions.
func (r *ASTQueryResult) MinCount(minCount int) *ASTQueryResult {
	r.tb.Helper()
	if len(r.nodes) < minCount {
		r.tb.Errorf("Expected at least %d nodes, but found %d", minCount, len(r.nodes))
	}
	return r
}

// MaxCount asserts that no more than the expected number of nodes were
// matched.
//
// Takes maxCount (int) which specifies the maximum allowed node count.
//
// Returns *ASTQueryResult which allows method chaining for further assertions.
func (r *ASTQueryResult) MaxCount(maxCount int) *ASTQueryResult {
	r.tb.Helper()
	if len(r.nodes) > maxCount {
		r.tb.Errorf("Expected at most %d nodes, but found %d", maxCount, len(r.nodes))
	}
	return r
}

// HasText asserts that the first matched node has the expected text content.
//
// Takes expected (string) which is the text content to compare against.
//
// Returns *ASTQueryResult which allows for method chaining.
func (r *ASTQueryResult) HasText(expected string) *ASTQueryResult {
	r.tb.Helper()

	if len(r.nodes) == 0 {
		r.tb.Error("Cannot check text: no nodes matched")
		return r
	}

	node := r.nodes[0]
	actual := r.getNodeText(node)

	assert.Equal(r.tb, expected, actual, "Text content mismatch")
	return r
}

// ContainsText asserts that the first matched node's text contains the
// expected substring.
//
// Takes substring (string) which is the text to search for within the node.
//
// Returns *ASTQueryResult which allows for method chaining.
func (r *ASTQueryResult) ContainsText(substring string) *ASTQueryResult {
	r.tb.Helper()

	if len(r.nodes) == 0 {
		r.tb.Error("Cannot check text: no nodes matched")
		return r
	}

	node := r.nodes[0]
	actual := r.getNodeText(node)

	if !strings.Contains(actual, substring) {
		r.tb.Errorf("Expected text to contain %q, but got %q", substring, actual)
	}
	return r
}

// HasAttribute asserts that the first matched node has an attribute with the
// given name and value.
//
// Takes name (string) which specifies the attribute name to check.
// Takes expectedValue (string) which specifies the expected attribute value.
//
// Returns *ASTQueryResult which allows method chaining for further assertions.
func (r *ASTQueryResult) HasAttribute(name, expectedValue string) *ASTQueryResult {
	r.tb.Helper()

	if len(r.nodes) == 0 {
		r.tb.Error("Cannot check attribute: no nodes matched")
		return r
	}

	node := r.nodes[0]
	actualValue, ok := node.GetAttribute(name)

	if !ok {
		r.tb.Errorf("Expected attribute %q not found on node <%s>", name, node.TagName)
		return r
	}

	assert.Equal(r.tb, expectedValue, actualValue, "Attribute value mismatch for %q", name)
	return r
}

// HasAttributeContaining asserts that the first matched node has an attribute
// with a value containing the substring.
//
// Takes name (string) which specifies the attribute to check.
// Takes substring (string) which specifies the value to search for.
//
// Returns *ASTQueryResult which allows method chaining for further assertions.
func (r *ASTQueryResult) HasAttributeContaining(name, substring string) *ASTQueryResult {
	r.tb.Helper()

	if len(r.nodes) == 0 {
		r.tb.Error("Cannot check attribute: no nodes matched")
		return r
	}

	node := r.nodes[0]
	actualValue, ok := node.GetAttribute(name)

	if !ok {
		r.tb.Errorf("Expected attribute %q not found on node <%s>", name, node.TagName)
		return r
	}

	if !strings.Contains(actualValue, substring) {
		r.tb.Errorf("Expected attribute %q to contain %q, but got %q", name, substring, actualValue)
	}
	return r
}

// HasAttributePresent asserts that the first matched node has the specified
// attribute, regardless of its value.
//
// Takes name (string) which specifies the attribute name to check for.
//
// Returns *ASTQueryResult which allows for method chaining.
func (r *ASTQueryResult) HasAttributePresent(name string) *ASTQueryResult {
	r.tb.Helper()

	if len(r.nodes) == 0 {
		r.tb.Error("Cannot check attribute: no nodes matched")
		return r
	}

	node := r.nodes[0]
	_, ok := node.GetAttribute(name)

	if !ok {
		r.tb.Errorf("Expected attribute %q not found on node <%s>", name, node.TagName)
	}
	return r
}

// HasClass asserts that the first matched node has the specified CSS class.
//
// Takes className (string) which specifies the CSS class name to check for.
//
// Returns *ASTQueryResult which allows for method chaining.
func (r *ASTQueryResult) HasClass(className string) *ASTQueryResult {
	r.tb.Helper()

	if len(r.nodes) == 0 {
		r.tb.Error("Cannot check class: no nodes matched")
		return r
	}

	node := r.nodes[0]
	if !node.HasClass(className) {
		r.tb.Errorf("Expected node <%s> to have class %q, but it does not", node.TagName, className)
	}
	return r
}

// HasTag asserts that the first matched node is an element with the specified
// tag name.
//
// Takes tagName (string) which specifies the expected HTML tag name.
//
// Returns *ASTQueryResult which allows method chaining for further assertions.
func (r *ASTQueryResult) HasTag(tagName string) *ASTQueryResult {
	r.tb.Helper()

	if len(r.nodes) == 0 {
		r.tb.Error("Cannot check tag: no nodes matched")
		return r
	}

	node := r.nodes[0]
	assert.Equal(r.tb, tagName, node.TagName, "Tag name mismatch")
	return r
}

// Each iterates over all matched nodes and calls the callback for each.
//
// Takes callback (func(int, *TemplateNode)) which is called for each matched node
// with its index and the node itself.
//
// Returns *ASTQueryResult which allows method chaining.
func (r *ASTQueryResult) Each(callback func(index int, node *ast_domain.TemplateNode)) *ASTQueryResult {
	r.tb.Helper()

	for i, node := range r.nodes {
		callback(i, node)
	}
	return r
}

// Filter returns a new ASTQueryResult containing only nodes that match the
// predicate.
//
// Takes predicate (func) which tests each node and returns true to include it.
//
// Returns *ASTQueryResult which contains only the nodes that passed the test.
func (r *ASTQueryResult) Filter(predicate func(node *ast_domain.TemplateNode) bool) *ASTQueryResult {
	filtered := make([]*ast_domain.TemplateNode, 0)

	for _, node := range r.nodes {
		if predicate(node) {
			filtered = append(filtered, node)
		}
	}

	return &ASTQueryResult{
		tb:    r.tb,
		nodes: filtered,
	}
}

// Map transforms each node using the provided function and returns a slice of
// results.
//
// Takes callback (func(node *ast_domain.TemplateNode) any) which transforms each
// matched node into a result value.
//
// Returns []any which contains the transformed results in the same order as
// the matched nodes.
func (r *ASTQueryResult) Map(callback func(node *ast_domain.TemplateNode) any) []any {
	results := make([]any, len(r.nodes))

	for i, node := range r.nodes {
		results[i] = callback(node)
	}

	return results
}

// Dump prints debug information about the matched nodes. Use it to debug
// tests when assertions fail.
//
// Returns *ASTQueryResult which allows method chaining.
func (r *ASTQueryResult) Dump() *ASTQueryResult {
	r.tb.Helper()

	if len(r.nodes) == 0 {
		r.tb.Log("No nodes matched")
		return r
	}

	r.tb.Logf("Matched %d node(s):", len(r.nodes))
	for i, node := range r.nodes {
		r.tb.Logf("  [%d] %s", i, r.dumpNode(node))
	}

	return r
}

// getNodeText extracts text content from a node, returning the text directly
// for text nodes or collecting all descendant text for element nodes.
//
// Takes node (*ast_domain.TemplateNode) which specifies the node to extract
// text from.
//
// Returns string which contains the extracted text, or empty if the node is
// nil or has an unsupported type.
func (r *ASTQueryResult) getNodeText(node *ast_domain.TemplateNode) string {
	if node == nil {
		return ""
	}

	if node.NodeType == ast_domain.NodeText {
		return r.resolveTextContent(node)
	}

	if node.NodeType == ast_domain.NodeElement {
		return r.collectTextRecursive(node)
	}

	return ""
}

// resolveTextContent returns the text content from a node, preferring the
// dynamic TextContentWriter (used for template expressions) over the static
// TextContent field.
//
// Takes node (*ast_domain.TemplateNode) which is the node to extract text
// from.
//
// Returns string which is the resolved text content.
func (*ASTQueryResult) resolveTextContent(node *ast_domain.TemplateNode) string {
	if node.TextContentWriter != nil && node.TextContentWriter.Len() > 0 {
		return node.TextContentWriter.String()
	}
	return node.TextContent
}

// collectTextRecursive walks the node tree and gathers all text content.
//
// Takes node (*ast_domain.TemplateNode) which is the root node to traverse.
//
// Returns string which contains the concatenated text from all text nodes.
func (r *ASTQueryResult) collectTextRecursive(node *ast_domain.TemplateNode) string {
	if node == nil {
		return ""
	}

	if node.NodeType == ast_domain.NodeText {
		return r.resolveTextContent(node)
	}

	var parts []string
	for _, child := range node.Children {
		text := r.collectTextRecursive(child)
		if text != "" {
			parts = append(parts, text)
		}
	}

	return strings.Join(parts, "")
}

// dumpNode creates a string form of a node for debugging.
//
// Takes node (*ast_domain.TemplateNode) which is the node to format.
//
// Returns string which is the formatted representation of the node.
func (r *ASTQueryResult) dumpNode(node *ast_domain.TemplateNode) string {
	if node.NodeType == ast_domain.NodeText {
		return fmt.Sprintf("Text: %q", r.resolveTextContent(node))
	}

	if node.NodeType == ast_domain.NodeElement {
		attrs := make([]string, 0, len(node.Attributes))
		for i := range node.Attributes {
			attrs = append(attrs, fmt.Sprintf("%s=%q", node.Attributes[i].Name, node.Attributes[i].Value))
		}

		if len(attrs) > 0 {
			return fmt.Sprintf("<%s %s>", node.TagName, strings.Join(attrs, " "))
		}
		return fmt.Sprintf("<%s>", node.TagName)
	}

	return node.NodeType.String()
}
