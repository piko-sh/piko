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

package premailer

import (
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
)

// removeAttributesIfConfigured removes class and ID attributes from elements
// based on the current options.
func (p *Premailer) removeAttributesIfConfigured() {
	if p.options.RemoveClasses {
		p.removeClasses()
	}
	if p.options.RemoveIDs {
		p.removeIDs()
	}
}

// removeNodes removes the given nodes from the tree.
//
// Takes nodesToRemove ([]*ast_domain.TemplateNode) which specifies the nodes
// to remove.
func (p *Premailer) removeNodes(nodesToRemove []*ast_domain.TemplateNode) {
	if len(nodesToRemove) == 0 {
		return
	}

	removalMap := make(map[*ast_domain.TemplateNode]bool, len(nodesToRemove))
	for _, node := range nodesToRemove {
		removalMap[node] = true
	}

	newRootNodes := make([]*ast_domain.TemplateNode, 0, len(p.tree.RootNodes))
	for _, child := range p.tree.RootNodes {
		if !removalMap[child] {
			newRootNodes = append(newRootNodes, child)
		}
	}
	p.tree.RootNodes = newRootNodes

	p.tree.Walk(filterChildrenFromRemovalMap(removalMap))
}

// removeClasses walks the tree and removes all class attributes from elements.
// This makes the HTML smaller and stops class-based styles from clashing.
func (p *Premailer) removeClasses() {
	p.tree.Walk(removeClassAttribute)
}

// collectAnchorTargets scans the tree for all anchor links with href="#target".
// This enables the "Gmail anchor link hack" where IDs are converted to name
// attributes.
//
// Returns map[string]bool which contains target IDs that should be preserved
// for anchor navigation.
func (p *Premailer) collectAnchorTargets() map[string]bool {
	targets := make(map[string]bool)
	p.tree.Walk(collectAnchorTarget(targets))
	return targets
}

// removeIDs removes id attributes from elements to improve email client
// support.
//
// Keeps anchor link navigation working while removing unused IDs. First collects
// all anchor link targets (such as href="#section") into a map. Then for each
// element with an id attribute: if the id is used by an anchor link, wraps the
// element with an <a name="id"></a> tag; if the id is not used, removes the
// attribute.
//
// Gmail strips id attributes but keeps name attributes on <a> tags. Other
// email clients (Apple Mail, Outlook.com, Yahoo) support both id and name.
// The <a name="target"></a> pattern is a common fix for Gmail anchor links.
func (p *Premailer) removeIDs() {
	targetsToKeep := p.collectAnchorTargets()

	p.tree.Walk(processIDAttribute(targetsToKeep))
}

var (
	// skipLinkPrefixes defines protocols and patterns that should be excluded from
	// query parameter appending.
	skipLinkPrefixes = []string{
		"javascript:",
		"mailto:",
		"tel:",
		"sms:",
		"data:",
	}

	// allowedLinkProtocols defines the only protocols that should have query
	// parameters appended.
	allowedLinkProtocols = []string{
		"http://",
		"https://",
	}
)

// filterChildrenFromRemovalMap returns a walker function that removes marked
// child nodes from each parent it visits.
//
// Takes removalMap (map[*ast_domain.TemplateNode]bool) which shows which nodes
// should be removed from their parent's children list.
//
// Returns func(*ast_domain.TemplateNode) bool which is a walker that removes
// marked children from each parent node it visits.
func filterChildrenFromRemovalMap(removalMap map[*ast_domain.TemplateNode]bool) func(*ast_domain.TemplateNode) bool {
	return func(parent *ast_domain.TemplateNode) bool {
		if len(parent.Children) == 0 {
			return true
		}
		newChildren := make([]*ast_domain.TemplateNode, 0, len(parent.Children))
		for _, child := range parent.Children {
			if !removalMap[child] {
				newChildren = append(newChildren, child)
			}
		}
		parent.Children = newChildren
		return true
	}
}

// removeClassAttribute removes the class attribute from element nodes.
// It is a walker function for tree traversal.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check and modify.
//
// Returns bool which is always true to continue walking the tree.
func removeClassAttribute(node *ast_domain.TemplateNode) bool {
	if node.NodeType == ast_domain.NodeElement {
		node.RemoveAttribute("class")
	}
	return true
}

// collectAnchorTarget returns a walker function that collects anchor target
// IDs.
//
// Takes targets (map[string]bool) which stores the collected anchor target IDs.
//
// Returns func(*ast_domain.TemplateNode) bool which is a walker that records
// anchor href values starting with # as target IDs in the targets map.
func collectAnchorTarget(targets map[string]bool) func(*ast_domain.TemplateNode) bool {
	return func(node *ast_domain.TemplateNode) bool {
		if node.NodeType != ast_domain.NodeElement || node.TagName != "a" {
			return true
		}

		href, exists := node.GetAttribute("href")
		if !exists || href == "" {
			return true
		}

		if strings.HasPrefix(href, "#") && len(href) > 1 {
			targetID := href[1:]
			targets[targetID] = true
		}

		return true
	}
}

// processIDAttribute returns a walker function that removes or converts ID
// attributes.
//
// Takes targetsToKeep (map[string]bool) which specifies IDs that should be
// changed to anchor name patterns instead of being removed.
//
// Returns func(*ast_domain.TemplateNode) bool which walks template nodes and
// processes their ID attributes.
func processIDAttribute(targetsToKeep map[string]bool) func(*ast_domain.TemplateNode) bool {
	return func(node *ast_domain.TemplateNode) bool {
		if node.NodeType != ast_domain.NodeElement {
			return true
		}

		idValue, hasID := node.GetAttribute("id")
		if !hasID || idValue == "" {
			return true
		}

		node.RemoveAttribute("id")

		if targetsToKeep[idValue] {
			convertIDToAnchorName(node, idValue)
		}

		return true
	}
}

// convertIDToAnchorName creates an anchor element and adds it as the first
// child of the given node.
//
// Takes node (*ast_domain.TemplateNode) which receives the anchor as its first
// child.
// Takes idValue (string) which sets the name attribute for the anchor.
func convertIDToAnchorName(node *ast_domain.TemplateNode, idValue string) {
	anchorNode := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "a",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "name", Value: idValue},
		},
		Children: []*ast_domain.TemplateNode{},
	}

	node.Children = append([]*ast_domain.TemplateNode{anchorNode}, node.Children...)
}

// shouldSkipLink reports whether a link should be skipped when adding query
// parameters.
//
// Takes href (string) which is the link URL to check.
//
// Returns bool which is true for empty strings, anchor links, and non-HTTP
// schemes such as mailto:, tel:, or JavaScript URLs.
func shouldSkipLink(href string) bool {
	if href == "" || strings.HasPrefix(href, "#") {
		return true
	}

	lower := strings.ToLower(href)

	for _, prefix := range skipLinkPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}

	if strings.Contains(href, ":") {
		for _, allowed := range allowedLinkProtocols {
			if strings.HasPrefix(lower, allowed) {
				return false
			}
		}
		return true
	}

	return false
}
