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

package pml_domain

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestAutowrapping(t *testing.T) {
	tests := []struct {
		name             string
		parentTagName    string
		inputChildren    string
		expectedChildren string
	}{
		{
			name:             "Root: Single Content Component",
			parentTagName:    "",
			inputChildren:    `<pml-p>Hello</pml-p>`,
			expectedChildren: "pml-row(pml-col(pml-p(TEXT)))",
		},
		{
			name:             "Root: Single Raw Text Node",
			parentTagName:    "",
			inputChildren:    `Just some text`,
			expectedChildren: "pml-row(pml-col(pml-p(TEXT)))",
		},
		{
			name:             "Root: Two Consecutive Content Components",
			parentTagName:    "",
			inputChildren:    `<pml-p>Title</pml-p><pml-img src="..."></pml-img>`,
			expectedChildren: "pml-row(pml-col(pml-p(TEXT), pml-img))",
		},
		{
			name:             "Root: A Valid Row Is Not Wrapped",
			parentTagName:    "",
			inputChildren:    `<pml-row><pml-col></pml-col></pml-row>`,
			expectedChildren: "pml-row(pml-col)",
		},
		{
			name:             "Root: Valid Row Followed by Content",
			parentTagName:    "",
			inputChildren:    `<pml-row></pml-row><pml-p>More content</pml-p>`,
			expectedChildren: "pml-row, pml-row(pml-col(pml-p(TEXT)))",
		},
		{
			name:             "Root: Content Interrupted by a Valid Row",
			parentTagName:    "",
			inputChildren:    `<pml-p>First</pml-p><pml-row></pml-row><pml-p>Last</pml-p>`,
			expectedChildren: "pml-row(pml-col(pml-p(TEXT))), pml-row, pml-row(pml-col(pml-p(TEXT)))",
		},
		{
			name:             "Root: Plain HTML Div Is Not Wrapped",
			parentTagName:    "",
			inputChildren:    `<div>Hello</div>`,
			expectedChildren: "div(TEXT)",
		},
		{
			name:             "Root: Plain HTML Followed by Wrappable Content",
			parentTagName:    "",
			inputChildren:    `<div>Hello</div><pml-p>PML Here</pml-p>`,
			expectedChildren: "div(TEXT), pml-row(pml-col(pml-p(TEXT)))",
		},
		{
			name:             "Root: Only whitespace is ignored",
			parentTagName:    "",
			inputChildren:    `   `,
			expectedChildren: "",
		},
		{
			name:             "Root: Only a comment is passed through",
			parentTagName:    "",
			inputChildren:    `<!-- comment -->`,
			expectedChildren: "COMMENT",
		},
		{
			name:             "Row: Empty Row",
			parentTagName:    "pml-row",
			inputChildren:    ``,
			expectedChildren: "",
		},
		{
			name:             "Row: Single Explicit Column",
			parentTagName:    "pml-row",
			inputChildren:    `<pml-col></pml-col>`,
			expectedChildren: "pml-col",
		},
		{
			name:             "Row: Single pml-p",
			parentTagName:    "pml-row",
			inputChildren:    `<pml-p>Hello</pml-p>`,
			expectedChildren: "pml-col(pml-p(TEXT))",
		},
		{
			name:             "Row: Single Raw Text",
			parentTagName:    "pml-row",
			inputChildren:    `Hello`,
			expectedChildren: "pml-col(pml-p(TEXT))",
		},
		{
			name:             "Row: Two Consecutive pml-ps Grouped",
			parentTagName:    "pml-row",
			inputChildren:    `<pml-p>One</pml-p><pml-p>Two</pml-p>`,
			expectedChildren: "pml-col(pml-p(TEXT), pml-p(TEXT))",
		},
		{
			name:             "Row: Mixed Content Grouped",
			parentTagName:    "pml-row",
			inputChildren:    `<pml-p>One</pml-p><pml-img></pml-img>`,
			expectedChildren: "pml-col(pml-p(TEXT), pml-img)",
		},
		{
			name:             "Row: Explicit Column Breaks Group",
			parentTagName:    "pml-row",
			inputChildren:    `<pml-p>One</pml-p><pml-col></pml-col><pml-p>Two</pml-p>`,
			expectedChildren: "pml-col(pml-p(TEXT)), pml-col, pml-col(pml-p(TEXT))",
		},
		{
			name:             "Row: Plain HTML Breaks Group",
			parentTagName:    "pml-row",
			inputChildren:    `<pml-p>One</pml-p><div>HTML</div><pml-p>Two</pml-p>`,
			expectedChildren: "pml-col(pml-p(TEXT)), div(TEXT), pml-col(pml-p(TEXT))",
		},
		{
			name:             "Row: Comment Breaks Group",
			parentTagName:    "pml-row",
			inputChildren:    `<pml-p>One</pml-p><!-- comment --><pml-p>Two</pml-p>`,
			expectedChildren: "pml-col(pml-p(TEXT)), COMMENT, pml-col(pml-p(TEXT))",
		},
		{
			name:             "Row: Whitespace between content is ignored and content stays grouped",
			parentTagName:    "pml-row",
			inputChildren:    `<pml-p>One</pml-p>   <pml-p>Two</pml-p>`,
			expectedChildren: "pml-col(pml-p(TEXT), pml-p(TEXT))",
		},
		{
			name:             "Wrapper: Contains Valid Row",
			parentTagName:    "pml-container",
			inputChildren:    `<pml-row></pml-row>`,
			expectedChildren: "pml-row",
		},
		{
			name:             "Wrapper: Wraps a Single Column in a Row",
			parentTagName:    "pml-container",
			inputChildren:    `<pml-col></pml-col>`,
			expectedChildren: "pml-row(pml-col)",
		},
		{
			name:             "Wrapper: Groups Two Columns in One Row",
			parentTagName:    "pml-container",
			inputChildren:    `<pml-col></pml-col><pml-col></pml-col>`,
			expectedChildren: "pml-row(pml-col, pml-col)",
		},
		{
			name:             "Wrapper: Wraps Content with Double Layer (Row -> Column)",
			parentTagName:    "pml-container",
			inputChildren:    `<pml-p>Content</pml-p>`,
			expectedChildren: "pml-row(pml-col(pml-p(TEXT)))",
		},
		{
			name:             "Wrapper: Breaks Grouping with Explicit Row",
			parentTagName:    "pml-container",
			inputChildren:    `<pml-col></pml-col><pml-row></pml-row><pml-col></pml-col>`,
			expectedChildren: "pml-row(pml-col), pml-row, pml-row(pml-col)",
		},
		{
			name:             "OrderedList: Contains Valid OrderedList Item",
			parentTagName:    "pml-ol",
			inputChildren:    `<pml-li></pml-li>`,
			expectedChildren: "pml-li",
		},
		{
			name:             "OrderedList: Wraps Raw Text in OrderedList Item and then Text",
			parentTagName:    "pml-ol",
			inputChildren:    `Item 1`,
			expectedChildren: "pml-li(pml-p(TEXT))",
		},
		{
			name:             "OrderedList: Wraps Content Component in OrderedList Item",
			parentTagName:    "pml-ol",
			inputChildren:    `<pml-img></pml-img>`,
			expectedChildren: "pml-li(pml-img)",
		},
		{
			name:             "OrderedList: Handles Mixed Explicit and Implicit Items",
			parentTagName:    "pml-ol",
			inputChildren:    `<pml-li>One</pml-li>Two`,
			expectedChildren: "pml-li(TEXT), pml-li(pml-p(TEXT))",
		},
		{
			name:             "OrderedList: Ignores comments and whitespace",
			parentTagName:    "pml-ol",
			inputChildren:    `<!-- c1 --> Item 1 <!-- c2 -->`,
			expectedChildren: "COMMENT, pml-li(pml-p(TEXT)), COMMENT",
		},
		{
			name:             "Column: Children are NOT Wrapped",
			parentTagName:    "pml-col",
			inputChildren:    `<pml-p>Content</pml-p><div>Raw</div>And Text`,
			expectedChildren: "pml-p(TEXT), div(TEXT), TEXT",
		},
		{
			name:             "Hero: Children are NOT Wrapped",
			parentTagName:    "pml-hero",
			inputChildren:    `<pml-p>Content</pml-p><div>Raw</div>And Text`,
			expectedChildren: "pml-p(TEXT), div(TEXT), TEXT",
		},
		{
			name:             "Edge: Deeply nested HTML is not touched",
			parentTagName:    "",
			inputChildren:    `<div><table><tr><td><pml-p>Deep</pml-p></td></tr></table></div>`,
			expectedChildren: "div(table(tr(td(pml-p(TEXT)))))",
		},
		{
			name:             "Edge: Multiple implicit groups at root",
			parentTagName:    "",
			inputChildren:    `<pml-p>A</pml-p><div>Break</div><pml-p>B</pml-p>`,
			expectedChildren: "pml-row(pml-col(pml-p(TEXT))), div(TEXT), pml-row(pml-col(pml-p(TEXT)))",
		},
		{
			name:             "Edge: Multiple implicit groups in section",
			parentTagName:    "pml-row",
			inputChildren:    `<pml-p>A</pml-p><div>Break</div><pml-p>B</pml-p>`,
			expectedChildren: "pml-col(pml-p(TEXT)), div(TEXT), pml-col(pml-p(TEXT))",
		},
		{
			name:             "Edge: pml-no-stack does not get double-wrapped in section",
			parentTagName:    "pml-container",
			inputChildren:    `<pml-no-stack></pml-no-stack>`,
			expectedChildren: "pml-row(pml-no-stack)",
		},
		{
			name:             "Edge: Raw text adjacent to explicit column",
			parentTagName:    "pml-row",
			inputChildren:    `Raw Text<pml-col></pml-col>`,
			expectedChildren: "pml-col(pml-p(TEXT)), pml-col",
		},
		{
			name:             "Edge: Explicit column adjacent to raw text",
			parentTagName:    "pml-row",
			inputChildren:    `<pml-col></pml-col>Raw Text`,
			expectedChildren: "pml-col, pml-col(pml-p(TEXT))",
		},
		{
			name:             "Edge: Empty pml-p is still wrappable",
			parentTagName:    "pml-row",
			inputChildren:    `<pml-p></pml-p>`,
			expectedChildren: "pml-col(pml-p)",
		},
		{
			name:             "Edge: pml-ol containing only a comment",
			parentTagName:    "pml-ol",
			inputChildren:    `<!-- comment -->`,
			expectedChildren: "COMMENT",
		},
		{
			name:             "Edge: pml-row containing only a comment",
			parentTagName:    "pml-row",
			inputChildren:    `<!-- comment -->`,
			expectedChildren: "COMMENT",
		},
		{
			name:             "Edge: pml-container containing only a comment",
			parentTagName:    "pml-container",
			inputChildren:    `<!-- comment -->`,
			expectedChildren: "COMMENT",
		},
		{
			name:             "Edge: Root containing only a comment",
			parentTagName:    "",
			inputChildren:    `<!-- comment -->`,
			expectedChildren: "COMMENT",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inputNodes, err := parsePMLFragment(tc.inputChildren)
			require.NoError(t, err)

			var parentNode *ast_domain.TemplateNode
			if tc.parentTagName != "" {
				parentNode = &ast_domain.TemplateNode{
					NodeType: ast_domain.NodeElement,
					TagName:  tc.parentTagName,
				}
			}

			actualNodes := autowrapChildren(inputNodes, parentNode)
			actualString := astToDebugString(actualNodes)

			assert.Equal(t, tc.expectedChildren, actualString, "Autowrapping result did not match expectation.")
		})
	}
}

func parsePMLFragment(pml string) ([]*ast_domain.TemplateNode, error) {
	if strings.TrimSpace(pml) == "" {
		return nil, nil
	}
	ast, err := ast_domain.Parse(context.Background(), "<root>"+pml+"</root>", "test_fragment.pml", nil)
	if err != nil {
		return nil, err
	}
	if len(ast.RootNodes) == 0 {
		return nil, nil
	}
	return ast.RootNodes[0].Children, nil
}

func astToDebugString(nodes []*ast_domain.TemplateNode) string {
	var parts []string
	for _, node := range nodes {
		if part := nodeToString(node); part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, ", ")
}

func nodeToString(node *ast_domain.TemplateNode) string {
	if node == nil {
		return ""
	}

	switch node.NodeType {
	case ast_domain.NodeElement:
		var childrenString string
		if len(node.Children) > 0 {
			childrenString = astToDebugString(node.Children)
		}
		if childrenString != "" {
			return node.TagName + "(" + childrenString + ")"
		}
		return node.TagName
	case ast_domain.NodeText:
		if isMeaningfulTextNode(node) {
			return "TEXT"
		}
		return ""
	case ast_domain.NodeComment:
		return "COMMENT"
	case ast_domain.NodeFragment:
		return astToDebugString(node.Children)
	default:
		return ""
	}
}
