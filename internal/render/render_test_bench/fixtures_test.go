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

//go:build bench

package render_test_bench

import (
	"fmt"

	"piko.sh/piko/internal/ast/ast_domain"
)

type FixtureSize int

const (
	SizeTiny FixtureSize = iota
	SizeSmall
	SizeMedium
	SizeLarge
	SizeXLarge
	SizeHuge
)

func (s FixtureSize) String() string {
	switch s {
	case SizeTiny:
		return "Tiny"
	case SizeSmall:
		return "Small"
	case SizeMedium:
		return "Medium"
	case SizeLarge:
		return "Large"
	case SizeXLarge:
		return "XLarge"
	case SizeHuge:
		return "Huge"
	default:
		return "Unknown"
	}
}

func (s FixtureSize) NodeCount() int {
	switch s {
	case SizeTiny:
		return 5
	case SizeSmall:
		return 20
	case SizeMedium:
		return 100
	case SizeLarge:
		return 500
	case SizeXLarge:
		return 1000
	case SizeHuge:
		return 5000
	default:
		return 10
	}
}

func BuildGenericAST(nodeCount int) *ast_domain.TemplateAST {
	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			buildNestedContainer(nodeCount, 0),
		},
	}
}

func BuildFlatAST(nodeCount int) *ast_domain.TemplateAST {
	children := make([]*ast_domain.TemplateNode, 0, nodeCount)
	for i := range nodeCount {
		children = append(children, &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: fmt.Sprintf("item item-%d", i)},
				{Name: "data-index", Value: fmt.Sprintf("%d", i)},
			},
			Children: []*ast_domain.TemplateNode{
				{
					NodeType:    ast_domain.NodeText,
					TextContent: fmt.Sprintf("Item %d content", i),
				},
			},
		})
	}

	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:   ast_domain.NodeElement,
				TagName:    "div",
				Attributes: []ast_domain.HTMLAttribute{{Name: "id", Value: "flat-container"}},
				Children:   children,
			},
		},
	}
}

func BuildDeepAST(depth int) *ast_domain.TemplateAST {
	var current *ast_domain.TemplateNode
	var root *ast_domain.TemplateNode

	for i := depth; i >= 0; i-- {
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: fmt.Sprintf("depth-%d", i)},
			},
		}

		if current != nil {
			node.Children = []*ast_domain.TemplateNode{current}
		} else {
			node.Children = []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: "Deepest level"},
			}
		}

		current = node
		if i == 0 {
			root = node
		}
	}

	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{root},
	}
}

func BuildSVGHeavyAST(svgCount int) *ast_domain.TemplateAST {
	children := make([]*ast_domain.TemplateNode, 0, svgCount)

	svgIcons := []string{"icon-home.svg", "icon-settings.svg", "icon-user.svg", "icon-search.svg", "icon-menu.svg"}

	for i := range svgCount {
		icon := svgIcons[i%len(svgIcons)]
		children = append(children, &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:svg",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: fmt.Sprintf("testmodule/lib/%s", icon)},
				{Name: "class", Value: fmt.Sprintf("icon icon-%d", i)},
				{Name: "aria-label", Value: fmt.Sprintf("Icon %d", i)},
			},
		})
	}

	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:   ast_domain.NodeElement,
				TagName:    "div",
				Attributes: []ast_domain.HTMLAttribute{{Name: "class", Value: "svg-container"}},
				Children:   children,
			},
		},
	}
}

func BuildAttributeHeavyAST(nodeCount, attrsPerNode int) *ast_domain.TemplateAST {
	children := make([]*ast_domain.TemplateNode, 0, nodeCount)

	for i := range nodeCount {
		attrs := make([]ast_domain.HTMLAttribute, 0, attrsPerNode)
		for j := range attrsPerNode {
			attrs = append(attrs, ast_domain.HTMLAttribute{
				Name:  fmt.Sprintf("data-attr-%d", j),
				Value: fmt.Sprintf("value-%d-%d", i, j),
			})
		}
		attrs = append(attrs, ast_domain.HTMLAttribute{
			Name:  "class",
			Value: "item heavy-attrs",
		})

		children = append(children, &ast_domain.TemplateNode{
			NodeType:   ast_domain.NodeElement,
			TagName:    "div",
			Attributes: attrs,
			Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: fmt.Sprintf("Node %d", i)},
			},
		})
	}

	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:   ast_domain.NodeElement,
				TagName:    "div",
				Attributes: []ast_domain.HTMLAttribute{{Name: "id", Value: "attrs-container"}},
				Children:   children,
			},
		},
	}
}

func BuildCSRFHeavyAST(formCount int) *ast_domain.TemplateAST {
	children := make([]*ast_domain.TemplateNode, 0, formCount)

	for i := range formCount {
		children = append(children, &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "form",
			RuntimeAnnotations: &ast_domain.RuntimeAnnotation{
				NeedsCSRF: true,
			},
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "action", Value: fmt.Sprintf("/submit-%d", i)},
				{Name: "method", Value: "POST"},
			},
			Children: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "input",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "type", Value: "text"},
						{Name: "name", Value: fmt.Sprintf("field-%d", i)},
					},
				},
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "button",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "type", Value: "submit"},
					},
					Children: []*ast_domain.TemplateNode{
						{NodeType: ast_domain.NodeText, TextContent: "Submit"},
					},
				},
			},
		})
	}

	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:   ast_domain.NodeElement,
				TagName:    "div",
				Attributes: []ast_domain.HTMLAttribute{{Name: "id", Value: "forms-container"}},
				Children:   children,
			},
		},
	}
}

func BuildLinkHeavyAST(linkCount int) *ast_domain.TemplateAST {
	children := make([]*ast_domain.TemplateNode, 0, linkCount)

	for i := range linkCount {
		children = append(children, &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:a",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "href", Value: fmt.Sprintf("/page-%d", i)},
				{Name: "class", Value: "nav-link"},
			},
			Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: fmt.Sprintf("Link %d", i)},
			},
		})
	}

	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:   ast_domain.NodeElement,
				TagName:    "nav",
				Attributes: []ast_domain.HTMLAttribute{{Name: "class", Value: "navigation"}},
				Children:   children,
			},
		},
	}
}

func BuildMixedAST(scale int) *ast_domain.TemplateAST {
	header := buildHeader(scale)
	nav := buildNavigation(scale)
	main := buildMainContent(scale)
	footer := buildFooter()

	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:   ast_domain.NodeElement,
				TagName:    "div",
				Attributes: []ast_domain.HTMLAttribute{{Name: "id", Value: "app"}},
				Children:   []*ast_domain.TemplateNode{header, nav, main, footer},
			},
		},
	}
}

func BuildEventHeavyAST(nodeCount int) *ast_domain.TemplateAST {
	children := make([]*ast_domain.TemplateNode, 0, nodeCount)

	for i := range nodeCount {
		children = append(children, &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "button",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "interactive-btn"},
			},
			OnEvents: map[string][]ast_domain.Directive{
				"click": {
					{Type: ast_domain.DirectiveOn, Modifier: "", RawExpression: fmt.Sprintf("handleClick(%d)", i)},
				},
				"mouseover": {
					{Type: ast_domain.DirectiveOn, Modifier: "", RawExpression: fmt.Sprintf("handleHover(%d)", i)},
				},
			},
			CustomEvents: map[string][]ast_domain.Directive{
				"custom-event": {
					{Type: ast_domain.DirectiveEvent, Modifier: "", RawExpression: fmt.Sprintf("handleCustom(%d)", i)},
				},
			},
			Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: fmt.Sprintf("Button %d", i)},
			},
		})
	}

	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:   ast_domain.NodeElement,
				TagName:    "div",
				Attributes: []ast_domain.HTMLAttribute{{Name: "class", Value: "events-container"}},
				Children:   children,
			},
		},
	}
}

func BuildFragmentAST(fragmentCount, childrenPerFragment int) *ast_domain.TemplateAST {
	fragments := make([]*ast_domain.TemplateNode, 0, fragmentCount)

	for i := range fragmentCount {
		children := make([]*ast_domain.TemplateNode, 0, childrenPerFragment)
		for j := range childrenPerFragment {
			children = append(children, &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "span",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "class", Value: fmt.Sprintf("fragment-%d-child-%d", i, j)},
				},
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: fmt.Sprintf("F%d-C%d", i, j)},
				},
			})
		}

		fragments = append(fragments, &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeFragment,
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "fragment-class"},
				{Name: "data-fragment", Value: fmt.Sprintf("%d", i)},
			},
			Children: children,
		})
	}

	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:   ast_domain.NodeElement,
				TagName:    "div",
				Attributes: []ast_domain.HTMLAttribute{{Name: "id", Value: "fragments-container"}},
				Children:   fragments,
			},
		},
	}
}

func BuildCommentHeavyAST(commentCount int) *ast_domain.TemplateAST {
	children := make([]*ast_domain.TemplateNode, 0, commentCount*2)

	for i := range commentCount {
		children = append(children,
			&ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeComment,
				TextContent: fmt.Sprintf(" Comment %d: This is a test comment for benchmarking ", i),
			},
			&ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: fmt.Sprintf("Content %d", i)},
				},
			},
		)
	}

	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:   ast_domain.NodeElement,
				TagName:    "div",
				Attributes: []ast_domain.HTMLAttribute{{Name: "id", Value: "comments-container"}},
				Children:   children,
			},
		},
	}
}

func buildNestedContainer(remainingNodes int, depth int) *ast_domain.TemplateNode {
	if remainingNodes <= 0 {
		return nil
	}

	childCount := min(5, remainingNodes-1)
	children := make([]*ast_domain.TemplateNode, 0, childCount+1)

	children = append(children, &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeText,
		TextContent: fmt.Sprintf("Level %d content", depth),
	})

	nodesPerChild := (remainingNodes - 1) / max(childCount, 1)
	for range childCount {
		if child := buildNestedContainer(nodesPerChild, depth+1); child != nil {
			children = append(children, child)
		}
	}

	return &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "class", Value: fmt.Sprintf("container depth-%d", depth)},
		},
		Children: children,
	}
}

func buildHeader(scale int) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType:   ast_domain.NodeElement,
		TagName:    "header",
		Attributes: []ast_domain.HTMLAttribute{{Name: "class", Value: "site-header"}},
		Children: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "piko:svg",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "src", Value: "testmodule/lib/logo.svg"},
					{Name: "class", Value: "logo"},
					{Name: "aria-label", Value: "Company Logo"},
				},
			},
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "h1",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: "Page Title"},
				},
			},
		},
	}
}

func buildNavigation(scale int) *ast_domain.TemplateNode {
	linkCount := min(scale*3, 20)
	links := make([]*ast_domain.TemplateNode, 0, linkCount)

	for i := range linkCount {
		links = append(links, &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:a",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "href", Value: fmt.Sprintf("/section-%d", i)},
				{Name: "class", Value: "nav-link"},
			},
			Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: fmt.Sprintf("Section %d", i)},
			},
		})
	}

	return &ast_domain.TemplateNode{
		NodeType:   ast_domain.NodeElement,
		TagName:    "nav",
		Attributes: []ast_domain.HTMLAttribute{{Name: "class", Value: "main-nav"}},
		Children:   links,
	}
}

func buildMainContent(scale int) *ast_domain.TemplateNode {
	sections := make([]*ast_domain.TemplateNode, 0, scale)

	for i := range scale {
		sections = append(sections, buildContentSection(i))
	}

	return &ast_domain.TemplateNode{
		NodeType:   ast_domain.NodeElement,
		TagName:    "main",
		Attributes: []ast_domain.HTMLAttribute{{Name: "class", Value: "main-content"}},
		Children:   sections,
	}
}

func buildContentSection(index int) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "section",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "class", Value: "content-section"},
			{Name: "id", Value: fmt.Sprintf("section-%d", index)},
		},
		Children: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "h2",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: fmt.Sprintf("Section %d", index)},
				},
			},
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "my-card",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "title", Value: fmt.Sprintf("Card %d", index)},
				},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "p",
						Children: []*ast_domain.TemplateNode{
							{NodeType: ast_domain.NodeText, TextContent: "Card content here"},
						},
					},
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "piko:svg",
						Attributes: []ast_domain.HTMLAttribute{
							{Name: "src", Value: "testmodule/lib/icon.svg"},
							{Name: "class", Value: "card-icon"},
						},
					},
				},
			},
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "form",
				RuntimeAnnotations: &ast_domain.RuntimeAnnotation{
					NeedsCSRF: true,
				},
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "action", Value: fmt.Sprintf("/submit-%d", index)},
					{Name: "method", Value: "POST"},
				},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "input",
						Attributes: []ast_domain.HTMLAttribute{
							{Name: "type", Value: "email"},
							{Name: "name", Value: "email"},
							{Name: "placeholder", Value: "Enter email"},
						},
					},
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "button",
						Attributes: []ast_domain.HTMLAttribute{
							{Name: "type", Value: "submit"},
						},
						OnEvents: map[string][]ast_domain.Directive{
							"click": {
								{Type: ast_domain.DirectiveOn, Modifier: "action", RawExpression: "submitForm"},
							},
						},
						Children: []*ast_domain.TemplateNode{
							{NodeType: ast_domain.NodeText, TextContent: "Subscribe"},
						},
					},
				},
			},
		},
	}
}

func buildFooter() *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType:   ast_domain.NodeElement,
		TagName:    "footer",
		Attributes: []ast_domain.HTMLAttribute{{Name: "class", Value: "site-footer"}},
		Children: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeComment, TextContent: " Footer content "},
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "p",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: "Copyright 2024"},
				},
			},
		},
	}
}

func CountNodes(ast *ast_domain.TemplateAST) int {
	if ast == nil {
		return 0
	}
	count := 0
	for _, node := range ast.RootNodes {
		count += countNodeRecursive(node)
	}
	return count
}

func countNodeRecursive(node *ast_domain.TemplateNode) int {
	if node == nil {
		return 0
	}
	count := 1
	for _, child := range node.Children {
		count += countNodeRecursive(child)
	}
	return count
}
