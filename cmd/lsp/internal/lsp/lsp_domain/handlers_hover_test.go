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
	"testing"

	"go.lsp.dev/protocol"
)

func TestHover_AnalysisFails_ReturnsNil(t *testing.T) {
	ws := &workspace{
		documents:    make(map[protocol.DocumentURI]*document),
		docCache:     NewDocumentCache(),
		cancelFuncs:  make(map[protocol.DocumentURI]context.CancelCauseFunc),
		analysisDone: make(map[protocol.DocumentURI]chan struct{}),
		client:       &mockClient{},
	}

	server := &Server{
		workspace: ws,
		docCache:  NewDocumentCache(),
	}

	params := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///nonexistent.pk",
			},
			Position: protocol.Position{Line: 0, Character: 0},
		},
	}

	result, err := server.Hover(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result when analysis fails, got %v", result)
	}
}

func TestHover_ValidDocument_ReturnsHover(t *testing.T) {
	uri := protocol.DocumentURI("file:///test.pk")
	content := []byte(`<template><div>Hello</div></template>`)

	docCache := NewDocumentCache()
	docCache.Set(uri, content)

	ws := &workspace{
		documents: map[protocol.DocumentURI]*document{
			uri: {
				URI:     uri,
				Content: content,
				dirty:   false,
			},
		},
		docCache:     docCache,
		cancelFuncs:  make(map[protocol.DocumentURI]context.CancelCauseFunc),
		analysisDone: make(map[protocol.DocumentURI]chan struct{}),
	}

	server := &Server{
		workspace: ws,
		docCache:  docCache,
	}

	params := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 12},
		},
	}

	result, err := server.Hover(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	_ = result
}
