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

package fixtures

import (
	"piko.sh/piko/internal/ast/ast_domain"
)

// SvgComponentNode creates a template node for an SVG component.
//
// Returns *ast_domain.TemplateNode which represents a piko:svg element with source and
// class attributes configured for testing.
func SvgComponentNode() *ast_domain.TemplateNode {
	return ast_domain.NewElementNode("piko:svg", []ast_domain.HTMLAttribute{
		{
			Name:           "src",
			Value:          "testmodule/lib/icon.svg",
			Location:       ast_domain.Location{},
			NameLocation:   ast_domain.Location{},
			AttributeRange: ast_domain.Range{},
		},
		{
			Name:           "class",
			Value:          "icon",
			Location:       ast_domain.Location{},
			NameLocation:   ast_domain.Location{},
			AttributeRange: ast_domain.Range{},
		},
	}, nil)
}

// ComplexPageAST returns a test fixture representing a complex page template.
//
// Returns *ast_domain.TemplateAST which contains a page structure with nested components,
// a form with CSRF annotation, and text requiring HTML escaping.
func ComplexPageAST() *ast_domain.TemplateAST {
	t := ast_domain.TemplateAST{}
	t.RootNodes = []*ast_domain.TemplateNode{
		ast_domain.NewElementNode("main", []ast_domain.HTMLAttribute{
			{
				Name:           "id",
				Value:          "content",
				Location:       ast_domain.Location{},
				NameLocation:   ast_domain.Location{},
				AttributeRange: ast_domain.Range{},
			},
		}, []*ast_domain.TemplateNode{
			ast_domain.NewElementNode("my-card", []ast_domain.HTMLAttribute{
				{
					Name:           "title",
					Value:          "My Awesome Card",
					Location:       ast_domain.Location{},
					NameLocation:   ast_domain.Location{},
					AttributeRange: ast_domain.Range{},
				},
			}, []*ast_domain.TemplateNode{
				ast_domain.NewTextNode("Card content here."),
			}),
			SvgComponentNode(),
			ast_domain.NewElementNode("piko:a", []ast_domain.HTMLAttribute{
				{
					Name:           "href",
					Value:          "/about-us",
					Location:       ast_domain.Location{},
					NameLocation:   ast_domain.Location{},
					AttributeRange: ast_domain.Range{},
				},
			}, []*ast_domain.TemplateNode{
				ast_domain.NewTextNode("Learn More"),
			}),
			ast_domain.NewElementNode("p", nil, []*ast_domain.TemplateNode{
				ast_domain.NewTextNode("This text contains characters that need escaping: < > & \" '"),
			}),
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "form",
				RuntimeAnnotations: &ast_domain.RuntimeAnnotation{
					NeedsCSRF: true,
				},
				Attributes: []ast_domain.HTMLAttribute{
					{
						Name:           "action",
						Value:          "/submit",
						Location:       ast_domain.Location{},
						NameLocation:   ast_domain.Location{},
						AttributeRange: ast_domain.Range{},
					},
					{
						Name:           "method",
						Value:          "POST",
						Location:       ast_domain.Location{},
						NameLocation:   ast_domain.Location{},
						AttributeRange: ast_domain.Range{},
					},
				},
				Children: []*ast_domain.TemplateNode{
					ast_domain.NewElementNode("input", []ast_domain.HTMLAttribute{
						{
							Name:           "type",
							Value:          "text",
							Location:       ast_domain.Location{},
							NameLocation:   ast_domain.Location{},
							AttributeRange: ast_domain.Range{},
						},
						{
							Name:           "name",
							Value:          "username",
							Location:       ast_domain.Location{},
							NameLocation:   ast_domain.Location{},
							AttributeRange: ast_domain.Range{},
						},
					}, nil),
					ast_domain.NewElementNode("button", []ast_domain.HTMLAttribute{
						{
							Name:           "type",
							Value:          "submit",
							Location:       ast_domain.Location{},
							NameLocation:   ast_domain.Location{},
							AttributeRange: ast_domain.Range{},
						},
						{
							Name:           "p-on:click.prevent",
							Value:          "submitAction",
							Location:       ast_domain.Location{},
							NameLocation:   ast_domain.Location{},
							AttributeRange: ast_domain.Range{},
						},
					}, []*ast_domain.TemplateNode{
						ast_domain.NewTextNode("Submit"),
					}),
				},
				Key:                nil,
				DirScaffold:        nil,
				DirHTML:            nil,
				GoAnnotations:      nil,
				TextContentWriter:  nil,
				CustomEvents:       nil,
				OnEvents:           nil,
				Binds:              nil,
				TimelineDirectives: nil,
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
				DirMemo:            nil,
				DirSlot:            nil,
				DirModel:           nil,
				DirKey:             nil,
				TextContent:        "",
				InnerHTML:          "",
				PrerenderedHTML:    nil,
				RichText:           nil,
				Diagnostics:        nil,
				DynamicAttributes:  nil,
				Directives:         nil,
				AttributeWriters:   nil,
				ClosingTagRange:    ast_domain.Range{},
				OpeningTagRange:    ast_domain.Range{},
				NodeRange:          ast_domain.Range{},
				Location:           ast_domain.Location{},
				PreferredFormat:    0,
				IsPooled:           false,
				IsContentEditable:  false,
				PreserveWhitespace: false,
			},
			ast_domain.NewElementNode("another-component", nil, []*ast_domain.TemplateNode{
				ast_domain.NewTextNode("This one is lazy."),
			}),
		}),
	}
	return &t
}
