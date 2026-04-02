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

package pml_components

import (
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_domain"
	"piko.sh/piko/internal/pml/pml_dto"
)

const (
	defaultTestContainerWidth = 600
)

type TestNodeBuilder struct {
	tagName     string
	textContent string
	attributes  []ast_domain.HTMLAttribute
	children    []*ast_domain.TemplateNode
	nodeType    ast_domain.NodeType
}

func NewTestNode() *TestNodeBuilder {
	return &TestNodeBuilder{
		attributes:  []ast_domain.HTMLAttribute{},
		children:    []*ast_domain.TemplateNode{},
		tagName:     "",
		textContent: "",
		nodeType:    ast_domain.NodeElement,
	}
}

func (b *TestNodeBuilder) WithTagName(name string) *TestNodeBuilder {
	b.tagName = name
	return b
}

func (b *TestNodeBuilder) WithAttribute(name, value string) *TestNodeBuilder {
	b.attributes = append(b.attributes, ast_domain.HTMLAttribute{
		Name:  name,
		Value: value,
		Location: ast_domain.Location{
			Line:   1,
			Column: 1,
			Offset: 0,
		},
		NameLocation: ast_domain.Location{
			Line:   1,
			Column: 1,
			Offset: 0,
		},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			End:   ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		},
	})
	return b
}

func (b *TestNodeBuilder) WithChildren(children ...*ast_domain.TemplateNode) *TestNodeBuilder {
	b.children = children
	return b
}

func (b *TestNodeBuilder) AsTextNode(content string) *TestNodeBuilder {
	b.nodeType = ast_domain.NodeText
	b.textContent = content
	return b
}

func (b *TestNodeBuilder) AsFragmentNode() *TestNodeBuilder {
	b.nodeType = ast_domain.NodeFragment
	return b
}

func (b *TestNodeBuilder) Build() *ast_domain.TemplateNode {
	loc := ast_domain.Location{Line: 1, Column: 1, Offset: 0}
	nodeRange := ast_domain.Range{Start: loc, End: loc}

	return &ast_domain.TemplateNode{
		NodeType:           b.nodeType,
		TagName:            b.tagName,
		Attributes:         b.attributes,
		Children:           b.children,
		TextContent:        b.textContent,
		Location:           loc,
		NodeRange:          nodeRange,
		OpeningTagRange:    nodeRange,
		ClosingTagRange:    nodeRange,
		Key:                nil,
		DirKey:             nil,
		DirHTML:            nil,
		GoAnnotations:      nil,
		RuntimeAnnotations: nil,
		CustomEvents:       nil,
		OnEvents:           nil,
		Binds:              nil,
		DirContext:         nil,
		DirElse:            nil,
		DirText:            nil,
		DirStyle:           nil,
		DirClass:           nil,
		DirIf:              nil,
		DirElseIf:          nil,
		DirFor:             nil,
		DirShow:            nil,
		DirRef:             nil,
		DirModel:           nil,
		DirScaffold:        nil,
		InnerHTML:          "",
		RichText:           nil,
		Diagnostics:        nil,
		DynamicAttributes:  nil,
		Directives:         nil,
		PreferredFormat:    0,
		IsPooled:           false,
		IsContentEditable:  false,
	}
}

type TestContextBuilder struct {
	component       pml_domain.Component
	registry        pml_domain.ComponentRegistry
	parentComponent pml_domain.Component
	styles          map[string]string
	containerWidth  float64
	siblingCount    int
	isInsideGroup   bool
}

func NewTestContext() *TestContextBuilder {
	return &TestContextBuilder{
		styles:         make(map[string]string),
		component:      nil,
		containerWidth: defaultTestContainerWidth,
		siblingCount:   1,
		isInsideGroup:  false,
		registry:       nil,
	}
}

func (b *TestContextBuilder) WithStyle(key, value string) *TestContextBuilder {
	b.styles[key] = value
	return b
}

func (b *TestContextBuilder) WithContainerWidth(width float64) *TestContextBuilder {
	b.containerWidth = width
	return b
}

func (b *TestContextBuilder) InsideGroup() *TestContextBuilder {
	b.isInsideGroup = true
	return b
}

func (b *TestContextBuilder) WithSiblingCount(count int) *TestContextBuilder {
	b.siblingCount = count
	return b
}

func (b *TestContextBuilder) WithRegistry(registry pml_domain.ComponentRegistry) *TestContextBuilder {
	b.registry = registry
	return b
}

func (b *TestContextBuilder) WithParentComponent(parent pml_domain.Component) *TestContextBuilder {
	b.parentComponent = parent
	return b
}

func (b *TestContextBuilder) Build(node *ast_domain.TemplateNode, component pml_domain.Component) *pml_domain.TransformationContext {
	if len(b.styles) > 0 {
		loc := ast_domain.Location{Line: 1, Column: 1, Offset: 0}
		for key, value := range b.styles {
			node.Attributes = append(node.Attributes, ast_domain.HTMLAttribute{
				Name:           key,
				Value:          value,
				Location:       loc,
				NameLocation:   loc,
				AttributeRange: ast_domain.Range{Start: loc, End: loc},
			})
		}
	}

	testConfig := &pml_dto.Config{
		ValidationLevel:        pml_dto.ValidationSoft,
		Breakpoint:             "480px",
		ClearDefaultAttributes: false,
		OverrideAttributes:     make(map[string]map[string]string),
		CustomComponents:       make(map[string]string),
		Beautify:               false,
		Minify:                 false,
	}

	sm := pml_domain.NewStyleManager(node, component, testConfig)

	return &pml_domain.TransformationContext{
		Config:                  testConfig,
		StyleManager:            sm,
		ParentNode:              nil,
		ParentComponent:         b.parentComponent,
		MediaQueryCollector:     nil,
		MSOConditionalCollector: nil,
		Registry:                b.registry,
		EmailAssetRegistry:      nil,
		ComponentPath:           []string{},
		ContainerWidth:          b.containerWidth,
		SiblingCount:            b.siblingCount,
		IsInsideGroup:           b.isInsideGroup,
		IsEmailContext:          false,
	}
}

func FindAttribute(node *ast_domain.TemplateNode, name string) (ast_domain.HTMLAttribute, bool) {
	if node == nil {
		return ast_domain.HTMLAttribute{}, false
	}

	for i := range node.Attributes {
		if node.Attributes[i].Name == name {
			return node.Attributes[i], true
		}
	}

	return ast_domain.HTMLAttribute{}, false
}

func ExtractTagNames(node *ast_domain.TemplateNode) []string {
	if node == nil {
		return []string{}
	}

	tags := []string{}

	if node.TagName != "" {
		tags = append(tags, node.TagName)
	}

	for _, child := range node.Children {
		tags = append(tags, ExtractTagNames(child)...)
	}

	return tags
}

func FindNodesByType(node *ast_domain.TemplateNode, nodeType ast_domain.NodeType) []*ast_domain.TemplateNode {
	if node == nil {
		return []*ast_domain.TemplateNode{}
	}

	nodes := []*ast_domain.TemplateNode{}

	if node.NodeType == nodeType {
		nodes = append(nodes, node)
	}

	for _, child := range node.Children {
		nodes = append(nodes, FindNodesByType(child, nodeType)...)
	}

	return nodes
}

func FindNodeByTagName(node *ast_domain.TemplateNode, tagName string) *ast_domain.TemplateNode {
	if node == nil {
		return nil
	}

	if node.TagName == tagName {
		return node
	}

	for _, child := range node.Children {
		if found := FindNodeByTagName(child, tagName); found != nil {
			return found
		}
	}

	return nil
}

func ContainsVML(node *ast_domain.TemplateNode) bool {
	vmlNodes := FindNodesByType(node, ast_domain.NodeRawHTML)

	for _, vmlNode := range vmlNodes {
		if strings.Contains(vmlNode.TextContent, "<!--[if mso") {
			return true
		}
	}

	return false
}

func GetStyleAttribute(node *ast_domain.TemplateNode) string {
	attr, found := FindAttribute(node, "style")
	if !found {
		return ""
	}
	return attr.Value
}

func ContainsStyleProperty(node *ast_domain.TemplateNode, property string) bool {
	style := GetStyleAttribute(node)
	return strings.Contains(style, property+":")
}
