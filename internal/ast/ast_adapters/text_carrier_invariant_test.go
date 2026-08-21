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

package ast_adapters

import (
	"testing"

	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func writerHolding(text string) *ast_domain.DirectWriter {
	dw := &ast_domain.DirectWriter{}
	dw.AppendString(text)
	return dw
}

func richTextHolding(text string) []ast_domain.TextPart {
	return []ast_domain.TextPart{{
		Expression:    nil,
		GoAnnotations: nil,
		Literal:       text,
		RawExpression: "",
		Location:      ast_domain.Location{},
		IsLiteral:     true,
	}}
}

func TestCheckSingleTextCarrier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		node    *ast_domain.TemplateNode
		name    string
		wantErr bool
	}{
		{
			name:    "no text at all",
			node:    &ast_domain.TemplateNode{NodeType: ast_domain.NodeText},
			wantErr: false,
		},
		{
			name:    "static text only",
			node:    &ast_domain.TemplateNode{NodeType: ast_domain.NodeText, TextContent: "hello"},
			wantErr: false,
		},
		{
			name:    "rich text only",
			node:    &ast_domain.TemplateNode{NodeType: ast_domain.NodeText, RichText: richTextHolding("hello")},
			wantErr: false,
		},
		{
			name: "writer only",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeText,
				TextContentWriter: writerHolding("hello"),
			},
			wantErr: false,
		},
		{
			name: "rich text and writer together are rejected",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeText,
				RichText:          richTextHolding("authored"),
				TextContentWriter: writerHolding("evaluated"),
			},
			wantErr: true,
		},
		{
			name: "static text and writer together are rejected",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeText,
				TextContent:       "static",
				TextContentWriter: writerHolding("evaluated"),
			},
			wantErr: true,
		},
		{
			name: "static text and rich text together are rejected",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "static",
				RichText:    richTextHolding("authored"),
			},
			wantErr: true,
		},
		{
			name: "a writer holding no parts does not count as a carrier",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeText,
				TextContent:       "static",
				TextContentWriter: &ast_domain.DirectWriter{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := checkSingleTextCarrier(tt.node)

			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), "mutually exclusive")
		})
	}
}

func TestUnpackTemplateNode_RejectsConflictingTextCarriers(t *testing.T) {
	t.Parallel()

	tree := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{{
			NodeType:    ast_domain.NodeText,
			TextContent: "static",
			RichText:    richTextHolding("authored"),
		}},
	}

	encoded, err := EncodeAST(tree)
	require.NoError(t, err, "a malformed node must still encode; the decoder is the guard")

	_, err = DecodeAST(context.Background(), encoded)

	require.Error(t, err, "a node carrying two text fields must not decode")
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestUnpackTemplateNode_AcceptsASingleTextCarrier(t *testing.T) {
	t.Parallel()

	tree := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{{
			NodeType: ast_domain.NodeText,
			RichText: richTextHolding("authored"),
		}},
	}

	encoded, err := EncodeAST(tree)
	require.NoError(t, err)

	decoded, err := DecodeAST(context.Background(), encoded)

	require.NoError(t, err)
	require.Len(t, decoded.RootNodes, 1)
	assert.Equal(t, "authored", decoded.RootNodes[0].OwnText())
}
