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

func TestTypeDefinition_NilDocument_ReturnsEmptyList(t *testing.T) {
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

	params := &protocol.TypeDefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///nonexistent.pk",
			},
			Position: protocol.Position{Line: 0, Character: 0},
		},
	}

	result, err := server.TypeDefinition(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d locations", len(result))
	}
}

func TestReferences_WorkspaceSearchFails_ReturnsEmptyList(t *testing.T) {
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

	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///nonexistent.pk",
			},
			Position: protocol.Position{Line: 0, Character: 0},
		},
	}

	result, err := server.References(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d locations", len(result))
	}
}

func TestRename_NoReferences_ReturnsEmptyEdit(t *testing.T) {
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

	params := &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///nonexistent.pk",
			},
			Position: protocol.Position{Line: 0, Character: 0},
		},
		NewName: "newName",
	}

	result, err := server.Rename(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Changes == nil {
		t.Error("expected non-nil Changes map")
	}
	if len(result.Changes) != 0 {
		t.Errorf("expected empty Changes, got %d entries", len(result.Changes))
	}
}

func TestFoldingRanges_AnalysisFails_ReturnsEmptyList(t *testing.T) {
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

	params := &protocol.FoldingRangeParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///nonexistent.pk",
			},
		},
	}

	result, err := server.FoldingRanges(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d ranges", len(result))
	}
}

func TestSignatureHelp_NilDocument_ReturnsEmptySignatures(t *testing.T) {
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

	params := &protocol.SignatureHelpParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///nonexistent.pk",
			},
			Position: protocol.Position{Line: 0, Character: 0},
		},
	}

	result, err := server.SignatureHelp(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Signatures) != 0 {
		t.Errorf("expected empty signatures, got %d", len(result.Signatures))
	}
}

func TestDocumentHighlight_NilDocument_ReturnsEmptyList(t *testing.T) {
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

	params := &protocol.DocumentHighlightParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///nonexistent.pk",
			},
			Position: protocol.Position{Line: 0, Character: 0},
		},
	}

	result, err := server.DocumentHighlight(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d highlights", len(result))
	}
}

func TestLinkedEditingRange_NilDocument_ReturnsEmptyRanges(t *testing.T) {
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

	params := &protocol.LinkedEditingRangeParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///nonexistent.pk",
			},
			Position: protocol.Position{Line: 0, Character: 0},
		},
	}

	result, err := server.LinkedEditingRange(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Ranges) != 0 {
		t.Errorf("expected empty ranges, got %d", len(result.Ranges))
	}
}

func TestCompletionResolve_AddsDefaultDetail(t *testing.T) {
	server := &Server{}

	params := &protocol.CompletionItem{
		Label:  "testItem",
		Detail: "",
	}

	result, err := server.CompletionResolve(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Detail != "Symbol from current scope" {
		t.Errorf("expected default detail, got %q", result.Detail)
	}
}

func TestCompletionResolve_PreservesExistingDetail(t *testing.T) {
	server := &Server{}

	params := &protocol.CompletionItem{
		Label:  "testItem",
		Detail: "Existing detail",
	}

	result, err := server.CompletionResolve(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Detail != "Existing detail" {
		t.Errorf("expected preserved detail, got %q", result.Detail)
	}
}

func TestDeclaration_DelegatesToDefinition(t *testing.T) {
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

	params := &protocol.DeclarationParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 12},
		},
	}

	result, err := server.Declaration(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	_ = result
}

func TestImplementation_ReturnsEmptyList(t *testing.T) {
	server := &Server{}

	params := &protocol.ImplementationParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.pk",
			},
			Position: protocol.Position{Line: 0, Character: 0},
		},
	}

	result, err := server.Implementation(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d locations", len(result))
	}
}

func TestPrepareCallHierarchy_ReturnsEmptyList(t *testing.T) {
	server := &Server{}

	params := &protocol.CallHierarchyPrepareParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.pk",
			},
			Position: protocol.Position{Line: 0, Character: 0},
		},
	}

	result, err := server.PrepareCallHierarchy(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestIncomingCalls_ReturnsEmptyList(t *testing.T) {
	server := &Server{}

	params := &protocol.CallHierarchyIncomingCallsParams{
		Item: protocol.CallHierarchyItem{
			Name: "testFunc",
		},
	}

	result, err := server.IncomingCalls(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d calls", len(result))
	}
}

func TestOutgoingCalls_ReturnsEmptyList(t *testing.T) {
	server := &Server{}

	params := &protocol.CallHierarchyOutgoingCallsParams{
		Item: protocol.CallHierarchyItem{
			Name: "testFunc",
		},
	}

	result, err := server.OutgoingCalls(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d calls", len(result))
	}
}

func TestMoniker_ReturnsEmptyList(t *testing.T) {
	server := &Server{}

	params := &protocol.MonikerParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.pk",
			},
			Position: protocol.Position{Line: 0, Character: 0},
		},
	}

	result, err := server.Moniker(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d monikers", len(result))
	}
}
