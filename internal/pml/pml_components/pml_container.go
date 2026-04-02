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

// Container is a semantic alias for Row that stacks children vertically.
// It implements pml_domain.Component and inherits rendering logic from Row,
// activating the "stack-children" behaviour.
type Container struct {
	Row
}

var _ pml_domain.Component = (*Container)(nil)

// NewContainer creates a new Container component instance.
// A Container is a semantic alias for a Row that stacks its children
// vertically.
//
// Returns *Container which is the newly created container ready for use.
func NewContainer() *Container {
	return &Container{
		Row: Row{
			BaseComponent: BaseComponent{},
		},
	}
}

// TagName returns the tag name for this component.
//
// Returns string which is the HTML custom element tag name.
func (*Container) TagName() string {
	return "pml-container"
}

// AllowedParents returns the list of valid parent components for this component.
//
// Returns []string which contains the allowed parent element names.
func (*Container) AllowedParents() []string {
	return []string{"pml-body"}
}

// DefaultAttributes returns the default attributes for this container,
// overriding the padding inherited from pml-row.
//
// A wrapper's semantic purpose is to group sections, not add its own padding.
// Setting padding to "0" by default prevents the "double padding" issue and
// is more intuitive.
//
// Returns map[string]string which contains the default attribute values.
func (c *Container) DefaultAttributes() map[string]string {
	defaults := c.Row.DefaultAttributes()
	defaults[AttrPadding] = ValueZero
	return defaults
}

// Transform delegates to Row.Transform after injecting the stack-children flag.
// This means pml-container always uses vertical stacking for its children
// (child sections are rendered in separate <tr> elements) instead of horizontal
// column layout (child columns in a single <tr>).
//
// Takes node (*ast_domain.TemplateNode) which is the template node to transform.
// Takes ctx (*pml_domain.TransformationContext) which provides the transformation
// context including configuration and style management.
//
// Returns *ast_domain.TemplateNode which is the transformed node with stacking
// applied.
// Returns []*pml_domain.Error which contains any errors encountered during
// transformation.
func (c *Container) Transform(node *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) (*ast_domain.TemplateNode, []*pml_domain.Error) {
	hasStackChildren := false
	for i := range node.Attributes {
		if node.Attributes[i].Name == "stack-children" {
			hasStackChildren = true
			break
		}
	}

	if !hasStackChildren {
		node.Attributes = append(node.Attributes, ast_domain.HTMLAttribute{
			Name:           "stack-children",
			Value:          "true",
			Location:       NewLocation(),
			NameLocation:   NewLocation(),
			AttributeRange: NewRange(),
		})
	}

	ctx.StyleManager = pml_domain.NewStyleManager(node, c, ctx.Config)

	return c.Row.Transform(node, ctx)
}
