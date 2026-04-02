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

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAttribute(t *testing.T) {
	testCases := []struct {
		name          string
		node          *TemplateNode
		attributeName string
		wantValue     string
		wantExists    bool
	}{
		{
			name:          "nil node returns empty",
			node:          nil,
			attributeName: "class",
			wantValue:     "",
			wantExists:    false,
		},
		{
			name:          "node with no attributes",
			node:          &TemplateNode{NodeType: NodeElement},
			attributeName: "class",
			wantValue:     "",
			wantExists:    false,
		},
		{
			name: "attribute exists exact case",
			node: &TemplateNode{
				NodeType: NodeElement,
				Attributes: []HTMLAttribute{
					{Name: "class", Value: "container"},
				},
			},
			attributeName: "class",
			wantValue:     "container",
			wantExists:    true,
		},
		{

			name: "attribute exists with uppercase input",
			node: &TemplateNode{
				NodeType: NodeElement,
				Attributes: []HTMLAttribute{
					{Name: "class", Value: "container"},
				},
			},
			attributeName: "CLASS",
			wantValue:     "container",
			wantExists:    true,
		},
		{
			name: "attribute does not exist",
			node: &TemplateNode{
				NodeType: NodeElement,
				Attributes: []HTMLAttribute{
					{Name: "id", Value: "main"},
				},
			},
			attributeName: "class",
			wantValue:     "",
			wantExists:    false,
		},
		{
			name: "multiple attributes finds correct one",
			node: &TemplateNode{
				NodeType: NodeElement,
				Attributes: []HTMLAttribute{
					{Name: "id", Value: "main"},
					{Name: "class", Value: "foo bar"},
					{Name: "data-value", Value: "123"},
				},
			},
			attributeName: "class",
			wantValue:     "foo bar",
			wantExists:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			value, exists := tc.node.GetAttribute(tc.attributeName)
			assert.Equal(t, tc.wantValue, value)
			assert.Equal(t, tc.wantExists, exists)
		})
	}
}

func TestHasAttribute(t *testing.T) {
	testCases := []struct {
		name          string
		node          *TemplateNode
		attributeName string
		want          bool
	}{
		{
			name:          "nil node returns false",
			node:          nil,
			attributeName: "class",
			want:          false,
		},
		{
			name:          "empty node returns false",
			node:          &TemplateNode{NodeType: NodeElement},
			attributeName: "class",
			want:          false,
		},
		{
			name: "attribute exists",
			node: &TemplateNode{
				NodeType: NodeElement,
				Attributes: []HTMLAttribute{
					{Name: "class", Value: "test"},
				},
			},
			attributeName: "class",
			want:          true,
		},
		{

			name: "attribute found with uppercase input",
			node: &TemplateNode{
				NodeType: NodeElement,
				Attributes: []HTMLAttribute{
					{Name: "class", Value: "test"},
				},
			},
			attributeName: "CLASS",
			want:          true,
		},
		{
			name: "attribute does not exist",
			node: &TemplateNode{
				NodeType: NodeElement,
				Attributes: []HTMLAttribute{
					{Name: "id", Value: "main"},
				},
			},
			attributeName: "class",
			want:          false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.node.HasAttribute(tc.attributeName)
			assert.Equal(t, tc.want, result)
		})
	}
}

func TestSetAttribute(t *testing.T) {
	t.Run("nil node does nothing", func(t *testing.T) {
		var node *TemplateNode
		node.SetAttribute("class", "test")

	})

	t.Run("non-element node does nothing", func(t *testing.T) {
		node := &TemplateNode{NodeType: NodeText}
		node.SetAttribute("class", "test")
		assert.Empty(t, node.Attributes)
	})

	t.Run("adds new attribute", func(t *testing.T) {
		node := &TemplateNode{NodeType: NodeElement}
		node.SetAttribute("class", "container")

		value, exists := node.GetAttribute("class")
		assert.True(t, exists)
		assert.Equal(t, "container", value)
	})

	t.Run("updates existing attribute", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Attributes: []HTMLAttribute{
				{Name: "class", Value: "old-class"},
			},
		}
		node.SetAttribute("class", "new-class")

		value, exists := node.GetAttribute("class")
		assert.True(t, exists)
		assert.Equal(t, "new-class", value)
		assert.Len(t, node.Attributes, 1)
	})

	t.Run("case insensitive update", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Attributes: []HTMLAttribute{
				{Name: "CLASS", Value: "old"},
			},
		}
		node.SetAttribute("class", "new")

		value, exists := node.GetAttribute("class")
		assert.True(t, exists)
		assert.Equal(t, "new", value)
	})
}

func TestRemoveAttribute(t *testing.T) {
	t.Run("nil node does nothing", func(t *testing.T) {
		var node *TemplateNode
		node.RemoveAttribute("class")

	})

	t.Run("removes existing attribute", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Attributes: []HTMLAttribute{
				{Name: "id", Value: "main"},
				{Name: "class", Value: "container"},
				{Name: "data-value", Value: "123"},
			},
		}
		node.RemoveAttribute("class")

		assert.False(t, node.HasAttribute("class"))
		assert.True(t, node.HasAttribute("id"))
		assert.True(t, node.HasAttribute("data-value"))
		assert.Len(t, node.Attributes, 2)
	})

	t.Run("removal with uppercase input", func(t *testing.T) {

		node := &TemplateNode{
			NodeType: NodeElement,
			Attributes: []HTMLAttribute{
				{Name: "class", Value: "foo"},
			},
		}
		node.RemoveAttribute("CLASS")

		assert.False(t, node.HasAttribute("class"))
		assert.Empty(t, node.Attributes)
	})

	t.Run("removing non-existent attribute does nothing", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Attributes: []HTMLAttribute{
				{Name: "id", Value: "main"},
			},
		}
		node.RemoveAttribute("class")

		assert.Len(t, node.Attributes, 1)
		assert.True(t, node.HasAttribute("id"))
	})
}

func TestFirstElementChild(t *testing.T) {
	t.Run("nil node returns nil", func(t *testing.T) {
		var node *TemplateNode
		assert.Nil(t, node.FirstElementChild())
	})

	t.Run("node with no children returns nil", func(t *testing.T) {
		node := &TemplateNode{NodeType: NodeElement}
		assert.Nil(t, node.FirstElementChild())
	})

	t.Run("node with only text children returns nil", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Children: []*TemplateNode{
				{NodeType: NodeText, TextContent: "hello"},
				{NodeType: NodeText, TextContent: "world"},
			},
		}
		assert.Nil(t, node.FirstElementChild())
	})

	t.Run("finds first element child", func(t *testing.T) {
		firstElement := &TemplateNode{NodeType: NodeElement, TagName: "span"}
		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
			Children: []*TemplateNode{
				{NodeType: NodeText, TextContent: "text"},
				firstElement,
				{NodeType: NodeElement, TagName: "p"},
			},
		}
		result := node.FirstElementChild()
		assert.Equal(t, firstElement, result)
		assert.Equal(t, "span", result.TagName)
	})

	t.Run("first child is element", func(t *testing.T) {
		firstElement := &TemplateNode{NodeType: NodeElement, TagName: "div"}
		node := &TemplateNode{
			NodeType: NodeElement,
			Children: []*TemplateNode{
				firstElement,
				{NodeType: NodeText},
			},
		}
		assert.Equal(t, firstElement, node.FirstElementChild())
	})
}

func TestLastElementChild(t *testing.T) {
	t.Run("nil node returns nil", func(t *testing.T) {
		var node *TemplateNode
		assert.Nil(t, node.LastElementChild())
	})

	t.Run("node with no children returns nil", func(t *testing.T) {
		node := &TemplateNode{NodeType: NodeElement}
		assert.Nil(t, node.LastElementChild())
	})

	t.Run("node with only text children returns nil", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Children: []*TemplateNode{
				{NodeType: NodeText, TextContent: "hello"},
				{NodeType: NodeComment},
			},
		}
		assert.Nil(t, node.LastElementChild())
	})

	t.Run("finds last element child", func(t *testing.T) {
		lastElement := &TemplateNode{NodeType: NodeElement, TagName: "footer"}
		node := &TemplateNode{
			NodeType: NodeElement,
			Children: []*TemplateNode{
				{NodeType: NodeElement, TagName: "header"},
				{NodeType: NodeText},
				lastElement,
				{NodeType: NodeText},
			},
		}
		result := node.LastElementChild()
		assert.Equal(t, lastElement, result)
		assert.Equal(t, "footer", result.TagName)
	})

	t.Run("last child is element", func(t *testing.T) {
		lastElement := &TemplateNode{NodeType: NodeElement, TagName: "span"}
		node := &TemplateNode{
			NodeType: NodeElement,
			Children: []*TemplateNode{
				{NodeType: NodeText},
				lastElement,
			},
		}
		assert.Equal(t, lastElement, node.LastElementChild())
	})
}

func TestChildElementCount(t *testing.T) {
	t.Run("nil node returns 0", func(t *testing.T) {
		var node *TemplateNode
		assert.Equal(t, 0, node.ChildElementCount())
	})

	t.Run("empty node returns 0", func(t *testing.T) {
		node := &TemplateNode{NodeType: NodeElement}
		assert.Equal(t, 0, node.ChildElementCount())
	})

	t.Run("only text children returns 0", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Children: []*TemplateNode{
				{NodeType: NodeText},
				{NodeType: NodeComment},
				{NodeType: NodeText},
			},
		}
		assert.Equal(t, 0, node.ChildElementCount())
	})

	t.Run("counts only element children", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Children: []*TemplateNode{
				{NodeType: NodeElement},
				{NodeType: NodeText},
				{NodeType: NodeElement},
				{NodeType: NodeComment},
				{NodeType: NodeElement},
			},
		}
		assert.Equal(t, 3, node.ChildElementCount())
	})

	t.Run("all element children", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Children: []*TemplateNode{
				{NodeType: NodeElement},
				{NodeType: NodeElement},
			},
		}
		assert.Equal(t, 2, node.ChildElementCount())
	})
}

func TestGetDirective(t *testing.T) {
	t.Run("nil node returns nil", func(t *testing.T) {
		var node *TemplateNode
		assert.Nil(t, node.GetDirective(DirectiveIf))
	})

	t.Run("gets DirIf from dedicated field", func(t *testing.T) {
		directive := &Directive{Type: DirectiveIf, RawExpression: "condition"}
		node := &TemplateNode{
			NodeType: NodeElement,
			DirIf:    directive,
		}
		result := node.GetDirective(DirectiveIf)
		assert.Equal(t, directive, result)
	})

	t.Run("gets DirElseIf from dedicated field", func(t *testing.T) {
		directive := &Directive{Type: DirectiveElseIf, RawExpression: "other"}
		node := &TemplateNode{
			NodeType:  NodeElement,
			DirElseIf: directive,
		}
		result := node.GetDirective(DirectiveElseIf)
		assert.Equal(t, directive, result)
	})

	t.Run("gets DirElse from dedicated field", func(t *testing.T) {
		directive := &Directive{Type: DirectiveElse}
		node := &TemplateNode{
			NodeType: NodeElement,
			DirElse:  directive,
		}
		result := node.GetDirective(DirectiveElse)
		assert.Equal(t, directive, result)
	})

	t.Run("gets DirFor from dedicated field", func(t *testing.T) {
		directive := &Directive{Type: DirectiveFor, RawExpression: "item in items"}
		node := &TemplateNode{
			NodeType: NodeElement,
			DirFor:   directive,
		}
		result := node.GetDirective(DirectiveFor)
		assert.Equal(t, directive, result)
	})

	t.Run("gets DirShow from dedicated field", func(t *testing.T) {
		directive := &Directive{Type: DirectiveShow, RawExpression: "visible"}
		node := &TemplateNode{
			NodeType: NodeElement,
			DirShow:  directive,
		}
		result := node.GetDirective(DirectiveShow)
		assert.Equal(t, directive, result)
	})

	t.Run("gets DirModel from dedicated field", func(t *testing.T) {
		directive := &Directive{Type: DirectiveModel, RawExpression: "inputValue"}
		node := &TemplateNode{
			NodeType: NodeElement,
			DirModel: directive,
		}
		result := node.GetDirective(DirectiveModel)
		assert.Equal(t, directive, result)
	})

	t.Run("gets DirRef from dedicated field", func(t *testing.T) {
		directive := &Directive{Type: DirectiveRef, Arg: "myRef"}
		node := &TemplateNode{
			NodeType: NodeElement,
			DirRef:   directive,
		}
		result := node.GetDirective(DirectiveRef)
		assert.Equal(t, directive, result)
	})

	t.Run("gets DirClass from dedicated field", func(t *testing.T) {
		directive := &Directive{Type: DirectiveClass}
		node := &TemplateNode{
			NodeType: NodeElement,
			DirClass: directive,
		}
		result := node.GetDirective(DirectiveClass)
		assert.Equal(t, directive, result)
	})

	t.Run("gets DirStyle from dedicated field", func(t *testing.T) {
		directive := &Directive{Type: DirectiveStyle}
		node := &TemplateNode{
			NodeType: NodeElement,
			DirStyle: directive,
		}
		result := node.GetDirective(DirectiveStyle)
		assert.Equal(t, directive, result)
	})

	t.Run("gets DirText from dedicated field", func(t *testing.T) {
		directive := &Directive{Type: DirectiveText, RawExpression: "message"}
		node := &TemplateNode{
			NodeType: NodeElement,
			DirText:  directive,
		}
		result := node.GetDirective(DirectiveText)
		assert.Equal(t, directive, result)
	})

	t.Run("gets DirHTML from dedicated field", func(t *testing.T) {
		directive := &Directive{Type: DirectiveHTML, RawExpression: "rawContent"}
		node := &TemplateNode{
			NodeType: NodeElement,
			DirHTML:  directive,
		}
		result := node.GetDirective(DirectiveHTML)
		assert.Equal(t, directive, result)
	})

	t.Run("gets DirKey from dedicated field", func(t *testing.T) {
		directive := &Directive{Type: DirectiveKey, RawExpression: "item.id"}
		node := &TemplateNode{
			NodeType: NodeElement,
			DirKey:   directive,
		}
		result := node.GetDirective(DirectiveKey)
		assert.Equal(t, directive, result)
	})

	t.Run("gets DirContext from dedicated field", func(t *testing.T) {
		directive := &Directive{Type: DirectiveContext}
		node := &TemplateNode{
			NodeType:   NodeElement,
			DirContext: directive,
		}
		result := node.GetDirective(DirectiveContext)
		assert.Equal(t, directive, result)
	})

	t.Run("falls back to Directives slice for unknown types", func(t *testing.T) {
		directive := Directive{Type: DirectiveScaffold}
		node := &TemplateNode{
			NodeType:   NodeElement,
			Directives: []Directive{directive},
		}
		result := node.GetDirective(DirectiveScaffold)
		assert.NotNil(t, result)
		assert.Equal(t, DirectiveScaffold, result.Type)
	})

	t.Run("returns nil when directive not found", func(t *testing.T) {
		node := &TemplateNode{NodeType: NodeElement}
		result := node.GetDirective(DirectiveIf)
		assert.Nil(t, result)
	})
}

func TestGetDirectives(t *testing.T) {
	t.Run("nil node returns nil", func(t *testing.T) {
		var node *TemplateNode
		assert.Nil(t, node.GetDirectives(DirectiveIf))
	})

	t.Run("returns single directive as slice for dedicated field", func(t *testing.T) {
		directive := &Directive{Type: DirectiveIf, RawExpression: "cond"}
		node := &TemplateNode{
			NodeType: NodeElement,
			DirIf:    directive,
		}
		result := node.GetDirectives(DirectiveIf)
		assert.Len(t, result, 1)
		assert.Equal(t, "cond", result[0].RawExpression)
	})

	t.Run("returns OnEvents for DirectiveOn", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			OnEvents: map[string][]Directive{
				"click": {
					{Type: DirectiveOn, Arg: "click", RawExpression: "handleClick"},
				},
				"submit": {
					{Type: DirectiveOn, Arg: "submit", RawExpression: "handleSubmit"},
				},
			},
		}
		result := node.GetDirectives(DirectiveOn)
		assert.Len(t, result, 2)
	})

	t.Run("returns CustomEvents for DirectiveEvent", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			CustomEvents: map[string][]Directive{
				"custom-event": {
					{Type: DirectiveEvent, Arg: "custom-event"},
				},
			},
		}
		result := node.GetDirectives(DirectiveEvent)
		assert.Len(t, result, 1)
	})

	t.Run("returns Binds for DirectiveBind", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Binds: map[string]*Directive{
				"class": {Type: DirectiveBind, Arg: "class"},
				"style": {Type: DirectiveBind, Arg: "style"},
			},
		}
		result := node.GetDirectives(DirectiveBind)
		assert.Len(t, result, 2)
	})

	t.Run("returns empty for Binds with nil values", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Binds: map[string]*Directive{
				"class": nil,
			},
		}
		result := node.GetDirectives(DirectiveBind)
		assert.Empty(t, result)
	})

	t.Run("returns empty for empty events map", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			OnEvents: map[string][]Directive{},
		}
		result := node.GetDirectives(DirectiveOn)
		assert.Nil(t, result)
	})

	t.Run("finds directive in raw slice returns first only", func(t *testing.T) {

		node := &TemplateNode{
			NodeType: NodeElement,
			Directives: []Directive{
				{Type: DirectiveScaffold, RawExpression: "scaffold1"},
			},
		}
		result := node.GetDirectives(DirectiveScaffold)
		assert.Len(t, result, 1)
		assert.Equal(t, "scaffold1", result[0].RawExpression)
	})
}

func TestHasDirective(t *testing.T) {
	t.Run("returns false for nil node", func(t *testing.T) {

		var node *TemplateNode
		assert.False(t, node.HasDirective(DirectiveIf))
	})

	t.Run("returns true when OnEvents has entries", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			OnEvents: map[string][]Directive{
				"click": {{Type: DirectiveOn}},
			},
		}
		assert.True(t, node.HasDirective(DirectiveOn))
	})

	t.Run("returns false when OnEvents is empty", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			OnEvents: map[string][]Directive{},
		}
		assert.False(t, node.HasDirective(DirectiveOn))
	})

	t.Run("returns true when CustomEvents has entries", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			CustomEvents: map[string][]Directive{
				"custom": {{Type: DirectiveEvent}},
			},
		}
		assert.True(t, node.HasDirective(DirectiveEvent))
	})

	t.Run("returns false when CustomEvents is empty", func(t *testing.T) {
		node := &TemplateNode{NodeType: NodeElement}
		assert.False(t, node.HasDirective(DirectiveEvent))
	})

	t.Run("returns true when Binds has entries", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Binds: map[string]*Directive{
				"class": {Type: DirectiveBind},
			},
		}
		assert.True(t, node.HasDirective(DirectiveBind))
	})

	t.Run("returns false when Binds is empty", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Binds:    map[string]*Directive{},
		}
		assert.False(t, node.HasDirective(DirectiveBind))
	})

	t.Run("returns true when directive exists in dedicated field", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			DirIf:    &Directive{Type: DirectiveIf},
		}
		assert.True(t, node.HasDirective(DirectiveIf))
	})

	t.Run("returns false when directive does not exist", func(t *testing.T) {
		node := &TemplateNode{NodeType: NodeElement}
		assert.False(t, node.HasDirective(DirectiveIf))
		assert.False(t, node.HasDirective(DirectiveFor))
		assert.False(t, node.HasDirective(DirectiveShow))
	})
}

func TestCollectEventDirectives(t *testing.T) {
	t.Run("nil map returns nil", func(t *testing.T) {
		result := collectEventDirectives(nil)
		assert.Nil(t, result)
	})

	t.Run("empty map returns nil", func(t *testing.T) {
		result := collectEventDirectives(map[string][]Directive{})
		assert.Nil(t, result)
	})

	t.Run("flattens single event", func(t *testing.T) {
		events := map[string][]Directive{
			"click": {
				{Type: DirectiveOn, Arg: "click"},
			},
		}
		result := collectEventDirectives(events)
		assert.Len(t, result, 1)
	})

	t.Run("flattens multiple events", func(t *testing.T) {
		events := map[string][]Directive{
			"click": {
				{Type: DirectiveOn, Arg: "click"},
				{Type: DirectiveOn, Arg: "click", Modifier: "prevent"},
			},
			"submit": {
				{Type: DirectiveOn, Arg: "submit"},
			},
		}
		result := collectEventDirectives(events)
		assert.Len(t, result, 3)
	})
}

func TestCollectBindDirectives(t *testing.T) {
	t.Run("nil map returns nil", func(t *testing.T) {
		result := collectBindDirectives(nil)
		assert.Nil(t, result)
	})

	t.Run("empty map returns nil", func(t *testing.T) {
		result := collectBindDirectives(map[string]*Directive{})
		assert.Nil(t, result)
	})

	t.Run("collects non-nil binds", func(t *testing.T) {
		binds := map[string]*Directive{
			"class": {Type: DirectiveBind, Arg: "class"},
			"style": {Type: DirectiveBind, Arg: "style"},
		}
		result := collectBindDirectives(binds)
		assert.Len(t, result, 2)
	})

	t.Run("skips nil values", func(t *testing.T) {
		binds := map[string]*Directive{
			"class": {Type: DirectiveBind, Arg: "class"},
			"style": nil,
		}
		result := collectBindDirectives(binds)
		assert.Len(t, result, 1)
	})
}

func TestClasses(t *testing.T) {
	t.Run("returns nil for no class attribute", func(t *testing.T) {
		node := &TemplateNode{NodeType: NodeElement}
		assert.Nil(t, node.Classes())
	})

	t.Run("returns nil for empty class attribute", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Attributes: []HTMLAttribute{
				{Name: "class", Value: ""},
			},
		}
		assert.Nil(t, node.Classes())
	})

	t.Run("returns single class", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Attributes: []HTMLAttribute{
				{Name: "class", Value: "container"},
			},
		}
		classes := node.Classes()
		assert.Equal(t, []string{"container"}, classes)
	})

	t.Run("returns multiple classes", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Attributes: []HTMLAttribute{
				{Name: "class", Value: "foo bar baz"},
			},
		}
		classes := node.Classes()
		assert.Equal(t, []string{"foo", "bar", "baz"}, classes)
	})

	t.Run("handles extra whitespace", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Attributes: []HTMLAttribute{
				{Name: "class", Value: "  foo   bar  "},
			},
		}
		classes := node.Classes()
		assert.Equal(t, []string{"foo", "bar"}, classes)
	})
}

func TestHasClass(t *testing.T) {
	t.Run("returns false for no classes", func(t *testing.T) {
		node := &TemplateNode{NodeType: NodeElement}
		assert.False(t, node.HasClass("test"))
	})

	t.Run("returns true for existing class", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Attributes: []HTMLAttribute{
				{Name: "class", Value: "foo bar baz"},
			},
		}
		assert.True(t, node.HasClass("bar"))
	})

	t.Run("returns false for non-existing class", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Attributes: []HTMLAttribute{
				{Name: "class", Value: "foo bar"},
			},
		}
		assert.False(t, node.HasClass("baz"))
	})
}

func TestAddClass(t *testing.T) {
	t.Run("nil node does nothing", func(t *testing.T) {
		var node *TemplateNode
		node.AddClass("test")

	})

	t.Run("empty names does nothing", func(t *testing.T) {
		node := &TemplateNode{NodeType: NodeElement}
		node.AddClass()
		assert.False(t, node.HasAttribute("class"))
	})

	t.Run("adds first class", func(t *testing.T) {
		node := &TemplateNode{NodeType: NodeElement}
		node.AddClass("container")
		value, _ := node.GetAttribute("class")
		assert.Equal(t, "container", value)
	})

	t.Run("adds to existing classes", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Attributes: []HTMLAttribute{
				{Name: "class", Value: "foo"},
			},
		}
		node.AddClass("bar")
		classes := node.Classes()
		assert.Contains(t, classes, "foo")
		assert.Contains(t, classes, "bar")
	})

	t.Run("avoids duplicates", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			Attributes: []HTMLAttribute{
				{Name: "class", Value: "foo bar"},
			},
		}
		node.AddClass("foo")
		classes := node.Classes()
		assert.Len(t, classes, 2)
	})

	t.Run("trims whitespace from names", func(t *testing.T) {
		node := &TemplateNode{NodeType: NodeElement}
		node.AddClass("  foo  ", "  bar  ")
		classes := node.Classes()
		assert.Contains(t, classes, "foo")
		assert.Contains(t, classes, "bar")
	})

	t.Run("ignores empty names", func(t *testing.T) {
		node := &TemplateNode{NodeType: NodeElement}
		node.AddClass("foo", "", "  ", "bar")
		classes := node.Classes()
		assert.Len(t, classes, 2)
	})

	t.Run("classes are sorted", func(t *testing.T) {
		node := &TemplateNode{NodeType: NodeElement}
		node.AddClass("zebra", "apple", "mango")
		classes := node.Classes()
		assert.Equal(t, []string{"apple", "mango", "zebra"}, classes)
	})
}

func TestShouldFormatInline(t *testing.T) {
	t.Run("nil node returns false", func(t *testing.T) {
		var node *TemplateNode
		assert.False(t, node.ShouldFormatInline())
	})

	t.Run("FormatAuto returns false", func(t *testing.T) {
		node := &TemplateNode{
			NodeType:        NodeElement,
			PreferredFormat: FormatAuto,
		}
		assert.False(t, node.ShouldFormatInline())
	})

	t.Run("FormatInline returns true", func(t *testing.T) {
		node := &TemplateNode{
			NodeType:        NodeElement,
			PreferredFormat: FormatInline,
		}
		assert.True(t, node.ShouldFormatInline())
	})

	t.Run("FormatBlock returns false", func(t *testing.T) {
		node := &TemplateNode{
			NodeType:        NodeElement,
			PreferredFormat: FormatBlock,
		}
		assert.False(t, node.ShouldFormatInline())
	})
}

func TestShouldFormatBlock(t *testing.T) {
	t.Run("nil node returns false", func(t *testing.T) {
		var node *TemplateNode
		assert.False(t, node.ShouldFormatBlock())
	})

	t.Run("FormatAuto returns false", func(t *testing.T) {
		node := &TemplateNode{
			NodeType:        NodeElement,
			PreferredFormat: FormatAuto,
		}
		assert.False(t, node.ShouldFormatBlock())
	})

	t.Run("FormatInline returns false", func(t *testing.T) {
		node := &TemplateNode{
			NodeType:        NodeElement,
			PreferredFormat: FormatInline,
		}
		assert.False(t, node.ShouldFormatBlock())
	})

	t.Run("FormatBlock returns true", func(t *testing.T) {
		node := &TemplateNode{
			NodeType:        NodeElement,
			PreferredFormat: FormatBlock,
		}
		assert.True(t, node.ShouldFormatBlock())
	})
}

func TestNodeType_String(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		want     string
		nodeType NodeType
	}{
		{
			name:     "element node returns NodeElement",
			nodeType: NodeElement,
			want:     "NodeElement",
		},
		{
			name:     "text node returns NodeText",
			nodeType: NodeText,
			want:     "NodeText",
		},
		{
			name:     "comment node returns NodeComment",
			nodeType: NodeComment,
			want:     "NodeComment",
		},
		{
			name:     "fragment node returns NodeFragment",
			nodeType: NodeFragment,
			want:     "NodeFragment",
		},
		{
			name:     "raw HTML node returns NodeRawHTML",
			nodeType: NodeRawHTML,
			want:     "NodeRawHTML",
		},
		{
			name:     "unknown node type returns UnknownNode",
			nodeType: NodeType(99),
			want:     "UnknownNode",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.nodeType.String())
		})
	}
}

func TestRecordDuplicateDirectiveDiagnostic(t *testing.T) {
	t.Parallel()

	node := &TemplateNode{
		NodeType: NodeElement,
		TagName:  "div",
	}
	directive := &Directive{
		Type:         DirectiveIf,
		NameLocation: Location{Line: 5, Column: 10},
	}
	firstLocation := Location{Line: 2, Column: 3}

	recordDuplicateDirectiveDiagnostic(node, directive, firstLocation)

	require.Len(t, node.Diagnostics, 1, "Should have exactly one diagnostic")
	diagnostic := node.Diagnostics[0]
	assert.Equal(t, Warning, diagnostic.Severity, "Diagnostic should be a warning")
	assert.Contains(t, diagnostic.Message, "Duplicate", "Message should mention 'Duplicate'")
	assert.Contains(t, diagnostic.Message, "p-if", "Message should mention the directive name")
	assert.Contains(t, diagnostic.Message, "line 2", "Message should reference the first location line")
	assert.Contains(t, diagnostic.Message, "column 3", "Message should reference the first location column")
}

func TestFindDirectivesInSlice(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node    *TemplateNode
		name    string
		dirType DirectiveType
		wantLen int
		wantNil bool
	}{
		{
			name: "matching directive type found",
			node: &TemplateNode{
				NodeType: NodeElement,
				Directives: []Directive{
					{Type: DirectiveScaffold, RawExpression: "scaffold1"},
					{Type: DirectiveIf, RawExpression: "cond"},
					{Type: DirectiveScaffold, RawExpression: "scaffold2"},
				},
			},
			dirType: DirectiveScaffold,
			wantLen: 2,
			wantNil: false,
		},
		{
			name: "no matching directive type returns nil",
			node: &TemplateNode{
				NodeType: NodeElement,
				Directives: []Directive{
					{Type: DirectiveIf, RawExpression: "cond"},
				},
			},
			dirType: DirectiveScaffold,
			wantLen: 0,
			wantNil: true,
		},
		{
			name: "empty directives slice returns nil",
			node: &TemplateNode{
				NodeType:   NodeElement,
				Directives: []Directive{},
			},
			dirType: DirectiveScaffold,
			wantLen: 0,
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := tc.node.findDirectivesInSlice(tc.dirType)
			if tc.wantNil {
				assert.Nil(t, result)
			} else {
				assert.Len(t, result, tc.wantLen)
			}
		})
	}
}

func TestGetAttributeWriter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node          *TemplateNode
		name          string
		attributeName string
		wantName      string
		wantFound     bool
	}{
		{
			name:          "nil node returns nil and false",
			node:          nil,
			attributeName: "title",
			wantFound:     false,
			wantName:      "",
		},
		{
			name: "matching writer found",
			node: &TemplateNode{
				NodeType: NodeElement,
				AttributeWriters: []*DirectWriter{
					{Name: "title"},
					{Name: "href"},
				},
			},
			attributeName: "title",
			wantFound:     true,
			wantName:      "title",
		},
		{
			name: "no matching writer returns nil and false",
			node: &TemplateNode{
				NodeType: NodeElement,
				AttributeWriters: []*DirectWriter{
					{Name: "href"},
				},
			},
			attributeName: "title",
			wantFound:     false,
			wantName:      "",
		},
		{
			name: "case insensitive matching",
			node: &TemplateNode{
				NodeType: NodeElement,
				AttributeWriters: []*DirectWriter{
					{Name: "title"},
				},
			},
			attributeName: "TITLE",
			wantFound:     true,
			wantName:      "title",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			writer, found := tc.node.GetAttributeWriter(tc.attributeName)
			assert.Equal(t, tc.wantFound, found)
			if tc.wantFound {
				require.NotNil(t, writer)
				assert.Equal(t, tc.wantName, writer.Name)
			} else {
				assert.Nil(t, writer)
			}
		})
	}
}
