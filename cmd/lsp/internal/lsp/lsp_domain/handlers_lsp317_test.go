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

package lsp_domain

import (
	"context"
	"math"
	"testing"

	"go.lsp.dev/protocol"
)

func TestRemarshalParams(t *testing.T) {
	testCases := []struct {
		input   any
		target  func() any
		name    string
		wantErr bool
	}{
		{
			name: "valid struct to struct",
			input: InlayHintParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "file:///test.pk",
				},
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 10, Character: 0},
				},
			},
			target:  func() any { return &InlayHintParams{} },
			wantErr: false,
		},
		{
			name:    "unmarshalable input",
			input:   math.Inf(1),
			target:  func() any { return &InlayHintParams{} },
			wantErr: true,
		},
		{
			name: "map to struct",
			input: map[string]any{
				"textDocument": map[string]any{
					"uri": "file:///mapped.pk",
				},
				"range": map[string]any{
					"start": map[string]any{"line": 1, "character": 2},
					"end":   map[string]any{"line": 3, "character": 4},
				},
			},
			target:  func() any { return &InlayHintParams{} },
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			target := tc.target()
			err := remarshalParams(tc.input, target)

			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestHandleInlayHint(t *testing.T) {
	testCases := []struct {
		params any
		server *Server
		name   string
	}{
		{
			name:   "nil workspace returns empty slice",
			server: &Server{workspace: nil},
			params: InlayHintParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "file:///test.pk",
				},
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 10, Character: 0},
				},
			},
		},
		{
			name: "document not found returns empty slice",
			server: &Server{
				workspace: &workspace{
					documents:    make(map[protocol.DocumentURI]*document),
					docCache:     NewDocumentCache(),
					cancelFuncs:  make(map[protocol.DocumentURI]context.CancelCauseFunc),
					analysisDone: make(map[protocol.DocumentURI]chan struct{}),
					client:       &mockClient{},
				},
			},
			params: InlayHintParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "file:///nonexistent.pk",
				},
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 10, Character: 0},
				},
			},
		},
		{
			name:   "invalid params returns empty slice",
			server: &Server{workspace: nil},
			params: math.Inf(1),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.server.handleInlayHint(context.Background(), tc.params)

			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			hints, ok := result.([]InlayHint)
			if !ok {
				t.Fatalf("expected []InlayHint, got %T", result)
			}
			if len(hints) != 0 {
				t.Errorf("expected empty slice, got %d hints", len(hints))
			}
		})
	}
}

func TestHandlePrepareTypeHierarchy(t *testing.T) {
	testCases := []struct {
		params any
		server *Server
		name   string
	}{
		{
			name:   "nil workspace returns empty slice",
			server: &Server{workspace: nil},
			params: TypeHierarchyPrepareParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: "file:///test.pk",
					},
					Position: protocol.Position{Line: 0, Character: 0},
				},
			},
		},
		{
			name: "document not found returns empty slice",
			server: &Server{
				workspace: &workspace{
					documents:    make(map[protocol.DocumentURI]*document),
					docCache:     NewDocumentCache(),
					cancelFuncs:  make(map[protocol.DocumentURI]context.CancelCauseFunc),
					analysisDone: make(map[protocol.DocumentURI]chan struct{}),
					client:       &mockClient{},
				},
			},
			params: TypeHierarchyPrepareParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: "file:///nonexistent.pk",
					},
					Position: protocol.Position{Line: 0, Character: 0},
				},
			},
		},
		{
			name:   "invalid params returns empty slice",
			server: &Server{workspace: nil},
			params: math.Inf(1),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.server.handlePrepareTypeHierarchy(context.Background(), tc.params)

			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			items, ok := result.([]TypeHierarchyItem)
			if !ok {
				t.Fatalf("expected []TypeHierarchyItem, got %T", result)
			}
			if len(items) != 0 {
				t.Errorf("expected empty slice, got %d items", len(items))
			}
		})
	}
}

func TestHandleTypeHierarchySupertypes(t *testing.T) {
	testCases := []struct {
		params any
		server *Server
		name   string
	}{
		{
			name:   "nil workspace returns empty slice",
			server: &Server{workspace: nil},
			params: TypeHierarchySupertypesParams{
				Item: TypeHierarchyItem{
					Name: "TestType",
					Kind: protocol.SymbolKindClass,
					URI:  "file:///test.pk",
				},
			},
		},
		{
			name: "document not found returns empty slice",
			server: &Server{
				workspace: &workspace{
					documents:    make(map[protocol.DocumentURI]*document),
					docCache:     NewDocumentCache(),
					cancelFuncs:  make(map[protocol.DocumentURI]context.CancelCauseFunc),
					analysisDone: make(map[protocol.DocumentURI]chan struct{}),
					client:       &mockClient{},
				},
			},
			params: TypeHierarchySupertypesParams{
				Item: TypeHierarchyItem{
					Name: "TestType",
					Kind: protocol.SymbolKindClass,
					URI:  "file:///nonexistent.pk",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.server.handleTypeHierarchySupertypes(context.Background(), tc.params)

			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			items, ok := result.([]TypeHierarchyItem)
			if !ok {
				t.Fatalf("expected []TypeHierarchyItem, got %T", result)
			}
			if len(items) != 0 {
				t.Errorf("expected empty slice, got %d items", len(items))
			}
		})
	}
}

func TestHandleTypeHierarchySubtypes(t *testing.T) {
	testCases := []struct {
		params any
		server *Server
		name   string
	}{
		{
			name:   "nil workspace returns empty slice",
			server: &Server{workspace: nil},
			params: TypeHierarchySubtypesParams{
				Item: TypeHierarchyItem{
					Name: "TestType",
					Kind: protocol.SymbolKindClass,
					URI:  "file:///test.pk",
				},
			},
		},
		{
			name: "document not found returns empty slice",
			server: &Server{
				workspace: &workspace{
					documents:    make(map[protocol.DocumentURI]*document),
					docCache:     NewDocumentCache(),
					cancelFuncs:  make(map[protocol.DocumentURI]context.CancelCauseFunc),
					analysisDone: make(map[protocol.DocumentURI]chan struct{}),
					client:       &mockClient{},
				},
			},
			params: TypeHierarchySubtypesParams{
				Item: TypeHierarchyItem{
					Name: "TestType",
					Kind: protocol.SymbolKindClass,
					URI:  "file:///nonexistent.pk",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.server.handleTypeHierarchySubtypes(context.Background(), tc.params)

			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			items, ok := result.([]TypeHierarchyItem)
			if !ok {
				t.Fatalf("expected []TypeHierarchyItem, got %T", result)
			}
			if len(items) != 0 {
				t.Errorf("expected empty slice, got %d items", len(items))
			}
		})
	}
}
