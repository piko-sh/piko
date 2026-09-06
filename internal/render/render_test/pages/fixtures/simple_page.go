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

// SimplePageAST returns a sample template AST for testing purposes.
//
// Returns *ast_domain.TemplateAST which contains a basic page structure with a container
// div, heading, paragraph, and comment node.
func SimplePageAST() *ast_domain.TemplateAST {
	t := ast_domain.TemplateAST{}
	t.RootNodes = []*ast_domain.TemplateNode{
		ast_domain.NewElementNode("div", []ast_domain.HTMLAttribute{
			{
				Name:           "id",
				Value:          "main-container",
				Location:       ast_domain.Location{},
				NameLocation:   ast_domain.Location{},
				AttributeRange: ast_domain.Range{},
			},
			{
				Name:           "class",
				Value:          "container",
				Location:       ast_domain.Location{},
				NameLocation:   ast_domain.Location{},
				AttributeRange: ast_domain.Range{},
			},
		}, []*ast_domain.TemplateNode{
			ast_domain.NewElementNode("h1", nil, []*ast_domain.TemplateNode{
				ast_domain.NewTextNode("Welcome"),
			}),
			ast_domain.NewElementNode("p", nil, []*ast_domain.TemplateNode{
				ast_domain.NewTextNode("This is a simple page."),
			}),
			{
				NodeType:           ast_domain.NodeComment,
				TextContent:        " This is a comment ",
				Key:                nil,
				DirScaffold:        nil,
				DirHTML:            nil,
				GoAnnotations:      nil,
				RuntimeAnnotations: nil,
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
				TagName:            "",
				InnerHTML:          "",
				PrerenderedHTML:    nil,
				Children:           nil,
				RichText:           nil,
				Attributes:         nil,
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
		}),
	}
	return &t
}
