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
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_domain"
)

// UnorderedList renders bulleted lists using the <ul> HTML element.
// It implements the Component interface and reuses the OrderedList logic,
// but uses the <pml-ul> tag and a different default list style.
type UnorderedList struct {
	OrderedList
}

var _ pml_domain.Component = (*UnorderedList)(nil)

// NewUnorderedList creates a new UnorderedList component instance.
// An UnorderedList is an alias of OrderedList that renders unordered
// lists by default.
//
// Returns *UnorderedList which is a ready-to-use component instance.
func NewUnorderedList() *UnorderedList {
	return &UnorderedList{
		OrderedList: OrderedList{
			BaseComponent: BaseComponent{},
		},
	}
}

// TagName returns the tag name for this component.
//
// Returns string which is the HTML custom element tag name "pml-ul".
func (*UnorderedList) TagName() string {
	return "pml-ul"
}

// DefaultAttributes returns the default attribute values for this component.
//
// Returns map[string]string which is an empty map as unordered lists have no
// default attributes.
func (*UnorderedList) DefaultAttributes() map[string]string {
	return map[string]string{}
}

// Transform overrides OrderedList's Transform to inject the list-style
// attribute set to "unordered". This means pml-ul always renders as an
// unordered list by default.
//
// Takes node (*ast_domain.TemplateNode) which is the node to transform.
// Takes ctx (*pml_domain.TransformationContext) which provides the
// transformation context including the style manager.
//
// Returns *ast_domain.TemplateNode which is the transformed node.
// Returns []*pml_domain.Error which contains any errors from the parent
// OrderedList's Transform method.
func (c *UnorderedList) Transform(node *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) (*ast_domain.TemplateNode, []*pml_domain.Error) {
	if _, hasListStyle := ctx.StyleManager.Get("list-style"); !hasListStyle {
		node.Attributes = append(node.Attributes, ast_domain.HTMLAttribute{
			Name:           "list-style",
			Value:          "unordered",
			Location:       NewLocation(),
			NameLocation:   NewLocation(),
			AttributeRange: NewRange(),
		})

		ctx.StyleManager = pml_domain.NewStyleManager(node, c, ctx.Config)
	}

	return c.OrderedList.Transform(node, ctx)
}
