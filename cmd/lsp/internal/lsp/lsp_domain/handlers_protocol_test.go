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
	"errors"
	"fmt"
	"testing"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/formatter/formatter_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

type mockFormatterService struct {
	FormatFunc            func(ctx context.Context, source []byte) ([]byte, error)
	FormatWithOptionsFunc func(ctx context.Context, source []byte, opts *formatter_domain.FormatOptions) ([]byte, error)
	FormatRangeFunc       func(ctx context.Context, source []byte, formatRange formatter_domain.Range, opts *formatter_domain.FormatOptions) ([]byte, error)
}

func (m *mockFormatterService) Format(ctx context.Context, source []byte) ([]byte, error) {
	if m.FormatFunc != nil {
		return m.FormatFunc(ctx, source)
	}
	return source, nil
}

func (m *mockFormatterService) FormatWithOptions(ctx context.Context, source []byte, opts *formatter_domain.FormatOptions) ([]byte, error) {
	if m.FormatWithOptionsFunc != nil {
		return m.FormatWithOptionsFunc(ctx, source, opts)
	}
	return source, nil
}

func (m *mockFormatterService) FormatRange(ctx context.Context, source []byte, formatRange formatter_domain.Range, opts *formatter_domain.FormatOptions) ([]byte, error) {
	if m.FormatRangeFunc != nil {
		return m.FormatRangeFunc(ctx, source, formatRange, opts)
	}
	return source, nil
}

var _ formatter_domain.FormatterService = (*mockFormatterService)(nil)

type mockJSONRPCConn struct {
	NotifyFunc func(ctx context.Context, method string, params any) error
}

func (m *mockJSONRPCConn) Call(_ context.Context, _ string, _, _ any) (jsonrpc2.ID, error) {
	return jsonrpc2.NewNumberID(0), nil
}

func (m *mockJSONRPCConn) Notify(ctx context.Context, method string, params any) error {
	if m.NotifyFunc != nil {
		return m.NotifyFunc(ctx, method, params)
	}
	return nil
}

func (m *mockJSONRPCConn) Go(_ context.Context, _ jsonrpc2.Handler) {}
func (m *mockJSONRPCConn) Close() error                             { return nil }

func (m *mockJSONRPCConn) Done() <-chan struct{} {
	doneChannel := make(chan struct{})
	close(doneChannel)
	return doneChannel
}

func (m *mockJSONRPCConn) Err() error { return nil }

var _ jsonrpc2.Conn = (*mockJSONRPCConn)(nil)

func TestWillSaveWaitUntil_EnabledWithDocInCache_ReturnsEdits(t *testing.T) {
	docCache := NewDocumentCache()
	testURI := protocol.DocumentURI("file:///test.pk")
	original := []byte("<template>\n<div></div>\n</template>")
	formatted := []byte("<template>\n  <div></div>\n</template>")
	docCache.Set(testURI, original)

	server := &Server{
		formattingEnabled: true,
		docCache:          docCache,
		formatter: &mockFormatterService{
			FormatFunc: func(_ context.Context, _ []byte) ([]byte, error) {
				return formatted, nil
			},
		},
	}

	params := &protocol.WillSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
		Reason:       protocol.TextDocumentSaveReasonManual,
	}

	edits, err := server.WillSaveWaitUntil(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(edits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(edits))
	}
	if edits[0].NewText != string(formatted) {
		t.Errorf("edit text = %q, want %q", edits[0].NewText, formatted)
	}
}

func TestWillSaveWaitUntil_EnabledContentUnchanged_ReturnsEmpty(t *testing.T) {
	docCache := NewDocumentCache()
	testURI := protocol.DocumentURI("file:///test.pk")
	content := []byte("<template><div></div></template>")
	docCache.Set(testURI, content)

	server := &Server{
		formattingEnabled: true,
		docCache:          docCache,
		formatter: &mockFormatterService{
			FormatFunc: func(_ context.Context, src []byte) ([]byte, error) {
				return src, nil
			},
		},
	}

	params := &protocol.WillSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
		Reason:       protocol.TextDocumentSaveReasonManual,
	}

	edits, err := server.WillSaveWaitUntil(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(edits) != 0 {
		t.Errorf("expected empty edits when content unchanged, got %d", len(edits))
	}
}

func TestWillSaveWaitUntil_EnabledNotInCache_ReturnsEmpty(t *testing.T) {
	docCache := NewDocumentCache()
	server := &Server{
		formattingEnabled: true,
		docCache:          docCache,
		formatter:         &mockFormatterService{},
	}

	params := &protocol.WillSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///missing.pk"},
		Reason:       protocol.TextDocumentSaveReasonManual,
	}

	edits, err := server.WillSaveWaitUntil(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(edits) != 0 {
		t.Errorf("expected empty edits, got %d", len(edits))
	}
}

func TestFormatting_EnabledWithDoc_ReturnsEdits(t *testing.T) {
	docCache := NewDocumentCache()
	testURI := protocol.DocumentURI("file:///test.pk")
	original := []byte("<template><div></div></template>")
	formatted := []byte("<template>\n  <div></div>\n</template>")
	docCache.Set(testURI, original)

	server := &Server{
		formattingEnabled: true,
		docCache:          docCache,
		formatter: &mockFormatterService{
			FormatFunc: func(_ context.Context, _ []byte) ([]byte, error) {
				return formatted, nil
			},
		},
	}

	params := &protocol.DocumentFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
	}

	edits, err := server.Formatting(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(edits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(edits))
	}
	if edits[0].NewText != string(formatted) {
		t.Errorf("edit text = %q, want %q", edits[0].NewText, formatted)
	}
}

func TestFormatting_EnabledNotInCache_ReturnsNil(t *testing.T) {
	docCache := NewDocumentCache()
	server := &Server{
		formattingEnabled: true,
		docCache:          docCache,
		formatter:         &mockFormatterService{},
	}

	params := &protocol.DocumentFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///missing.pk"},
	}

	edits, err := server.Formatting(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if edits != nil {
		t.Errorf("expected nil edits, got %v", edits)
	}
}

func TestRangeFormatting_EnabledWithDoc_ReturnsEdits(t *testing.T) {
	docCache := NewDocumentCache()
	testURI := protocol.DocumentURI("file:///test.pk")
	original := []byte("<template><div></div></template>")
	formatted := []byte("<template>\n  <div></div>\n</template>")
	docCache.Set(testURI, original)

	server := &Server{
		formattingEnabled: true,
		docCache:          docCache,
		formatter: &mockFormatterService{
			FormatRangeFunc: func(_ context.Context, _ []byte, _ formatter_domain.Range, _ *formatter_domain.FormatOptions) ([]byte, error) {
				return formatted, nil
			},
		},
	}

	params := &protocol.DocumentRangeFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 10, Character: 0},
		},
	}

	edits, err := server.RangeFormatting(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(edits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(edits))
	}
	if edits[0].NewText != string(formatted) {
		t.Errorf("edit text = %q, want %q", edits[0].NewText, formatted)
	}
}

func TestRangeFormatting_EnabledNotInCache_ReturnsNil(t *testing.T) {
	docCache := NewDocumentCache()
	server := &Server{
		formattingEnabled: true,
		docCache:          docCache,
		formatter:         &mockFormatterService{},
	}

	params := &protocol.DocumentRangeFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///missing.pk"},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 10, Character: 0},
		},
	}

	edits, err := server.RangeFormatting(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if edits != nil {
		t.Errorf("expected nil edits, got %v", edits)
	}
}

func TestOnTypeFormatting_EnabledWithDoc_ReturnsEdits(t *testing.T) {
	docCache := NewDocumentCache()
	testURI := protocol.DocumentURI("file:///test.pk")
	original := []byte("<template>\n<div>\n</div>\n</template>")
	formatted := []byte("<template>\n  <div>\n  </div>\n</template>")
	docCache.Set(testURI, original)

	server := &Server{
		formattingEnabled: true,
		docCache:          docCache,
		formatter: &mockFormatterService{
			FormatRangeFunc: func(_ context.Context, _ []byte, _ formatter_domain.Range, _ *formatter_domain.FormatOptions) ([]byte, error) {
				return formatted, nil
			},
		},
	}

	params := &protocol.DocumentOnTypeFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
		Position:     protocol.Position{Line: 2, Character: 0},
		Ch:           "\n",
	}

	edits, err := server.OnTypeFormatting(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(edits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(edits))
	}
	if edits[0].NewText != string(formatted) {
		t.Errorf("edit text = %q, want %q", edits[0].NewText, formatted)
	}
}

func TestOnTypeFormatting_EnabledNotInCache_ReturnsNil(t *testing.T) {
	docCache := NewDocumentCache()
	server := &Server{
		formattingEnabled: true,
		docCache:          docCache,
		formatter:         &mockFormatterService{},
	}

	params := &protocol.DocumentOnTypeFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///missing.pk"},
		Position:     protocol.Position{Line: 2, Character: 0},
		Ch:           "\n",
	}

	edits, err := server.OnTypeFormatting(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if edits != nil {
		t.Errorf("expected nil edits, got %v", edits)
	}
}

func TestImplementation_WithExistingDocument(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")
	ws.documents[testURI] = &document{
		URI:     testURI,
		Content: []byte("<template><div></div></template>"),
	}

	server := &Server{workspace: ws}

	params := &protocol.ImplementationParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
			Position:     protocol.Position{Line: 0, Character: 5},
		},
	}

	locations, err := server.Implementation(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if locations == nil {
		t.Error("expected non-nil locations slice")
	}
}

func TestImplementation_NilWorkspace_ReturnsEmpty(t *testing.T) {
	server := &Server{workspace: nil}

	params := &protocol.ImplementationParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.pk"},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
	}

	locations, err := server.Implementation(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(locations) != 0 {
		t.Errorf("expected empty locations, got %d", len(locations))
	}
}

func TestMoniker_WithExistingDocument(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")
	ws.documents[testURI] = &document{
		URI:     testURI,
		Content: []byte("<template><div></div></template>"),
	}

	server := &Server{workspace: ws}

	params := &protocol.MonikerParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
			Position:     protocol.Position{Line: 0, Character: 5},
		},
	}

	monikers, err := server.Moniker(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if monikers == nil {
		t.Error("expected non-nil monikers slice")
	}
}

func TestMoniker_NilWorkspace_ReturnsEmpty(t *testing.T) {
	server := &Server{workspace: nil}

	params := &protocol.MonikerParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.pk"},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
	}

	monikers, err := server.Moniker(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(monikers) != 0 {
		t.Errorf("expected empty monikers, got %d", len(monikers))
	}
}

func TestRename_WithReferences_GeneratesEdits(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")

	ws.documents[testURI] = &document{
		URI:     testURI,
		Content: []byte("<template><div></div></template>"),
		dirty:   false,
	}

	server := &Server{workspace: ws}

	params := &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
			Position:     protocol.Position{Line: 0, Character: 5},
		},
		NewName: "newName",
	}

	edit, err := server.Rename(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if edit == nil {
		t.Fatal("expected non-nil workspace edit")
	}
}

func TestReferences_ReturnsEmptyForDocWithoutAnnotations(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")

	ws.documents[testURI] = &document{
		URI:     testURI,
		Content: []byte("<template><div></div></template>"),
		dirty:   false,
	}

	server := &Server{workspace: ws}

	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
			Position:     protocol.Position{Line: 0, Character: 5},
		},
	}

	locations, err := server.References(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if locations == nil {
		t.Error("expected non-nil locations slice")
	}
}

func TestSemanticTokensFullDelta_ReturnsEmptyEdits(t *testing.T) {
	server := &Server{}

	params := &protocol.SemanticTokensDeltaParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.pk",
		},
		PreviousResultID: "prev-id",
	}

	result, err := server.SemanticTokensFullDelta(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	delta, ok := result.(*protocol.SemanticTokensDelta)
	if !ok {
		t.Fatalf("expected *protocol.SemanticTokensDelta, got %T", result)
	}
	if len(delta.Edits) != 0 {
		t.Errorf("expected empty edits, got %d", len(delta.Edits))
	}
}

func TestRequest_InlayHint(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")
	ws.documents[testURI] = &document{
		URI:     testURI,
		Content: []byte("<template><div></div></template>"),
	}

	server := &Server{workspace: ws}

	params := map[string]any{
		"textDocument": map[string]any{
			"uri": string(testURI),
		},
		"range": map[string]any{
			"start": map[string]any{"line": 0, "character": 0},
			"end":   map[string]any{"line": 1, "character": 0},
		},
	}

	result, err := server.Request(context.Background(), "textDocument/inlayHint", params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestRequest_PrepareTypeHierarchy(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")
	ws.documents[testURI] = &document{
		URI:     testURI,
		Content: []byte("<template><div></div></template>"),
	}

	server := &Server{workspace: ws}

	params := map[string]any{
		"textDocument": map[string]any{
			"uri": string(testURI),
		},
		"position": map[string]any{"line": 0, "character": 5},
	}

	result, err := server.Request(context.Background(), "textDocument/prepareTypeHierarchy", params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestRequest_TypeHierarchySupertypes(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")
	ws.documents[testURI] = &document{
		URI:     testURI,
		Content: []byte("<template><div></div></template>"),
	}

	server := &Server{workspace: ws}

	params := map[string]any{
		"item": map[string]any{
			"name": "TestType",
			"kind": 5,
			"uri":  string(testURI),
			"range": map[string]any{
				"start": map[string]any{"line": 0, "character": 0},
				"end":   map[string]any{"line": 0, "character": 5},
			},
			"selectionRange": map[string]any{
				"start": map[string]any{"line": 0, "character": 0},
				"end":   map[string]any{"line": 0, "character": 5},
			},
		},
	}

	result, err := server.Request(context.Background(), "typeHierarchy/supertypes", params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestRequest_TypeHierarchySubtypes(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")
	ws.documents[testURI] = &document{
		URI:     testURI,
		Content: []byte("<template><div></div></template>"),
	}

	server := &Server{workspace: ws}

	params := map[string]any{
		"item": map[string]any{
			"name": "TestType",
			"kind": 5,
			"uri":  string(testURI),
			"range": map[string]any{
				"start": map[string]any{"line": 0, "character": 0},
				"end":   map[string]any{"line": 0, "character": 5},
			},
			"selectionRange": map[string]any{
				"start": map[string]any{"line": 0, "character": 0},
				"end":   map[string]any{"line": 0, "character": 5},
			},
		},
	}

	result, err := server.Request(context.Background(), "typeHierarchy/subtypes", params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestRequest_UnknownMethod_ReturnsMethodNotFound(t *testing.T) {
	server := &Server{}

	_, err := server.Request(context.Background(), "unknown/method", nil)
	if !errors.Is(err, jsonrpc2.ErrMethodNotFound) {
		t.Errorf("expected ErrMethodNotFound, got %v", err)
	}
}

func TestSetConn_UpdatesServerAndWorkspace(t *testing.T) {
	ws := createTestWorkspace()
	server := &Server{workspace: ws}

	conn := &mockJSONRPCConn{}
	server.SetConn(conn)

	if server.conn != conn {
		t.Error("expected server.conn to be updated")
	}

	wsConn := ws.getConn()
	if wsConn != conn {
		t.Error("expected workspace conn to be updated")
	}
}

func TestSetConn_NilWorkspace_DoesNotPanic(t *testing.T) {
	server := &Server{}

	conn := &mockJSONRPCConn{}
	server.SetConn(conn)

	if server.conn != conn {
		t.Error("expected server.conn to be updated")
	}
}

func TestSignalAnalysisComplete_WithConn_SendsNotification(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")

	var receivedMethod string
	conn := &mockJSONRPCConn{
		NotifyFunc: func(_ context.Context, method string, _ any) error {
			receivedMethod = method
			return nil
		},
	}
	ws.setConn(conn)

	ws.signalAnalysisComplete(context.Background(), testURI)

	if receivedMethod != "piko/analysisComplete" {
		t.Errorf("expected method %q, got %q", "piko/analysisComplete", receivedMethod)
	}
}

func TestSignalAnalysisComplete_NilConn_DoesNotPanic(t *testing.T) {
	ws := createTestWorkspace()
	ws.setConn(nil)

	ws.signalAnalysisComplete(context.Background(), "file:///test.pk")
}

func TestExecuteCommand_RefreshDiagnostics_ReturnsMessage(t *testing.T) {
	ws := createTestWorkspace()
	serverCtx, cancel := context.WithCancelCause(context.Background())
	defer cancel(fmt.Errorf("test: cleanup"))

	ws.docCache.Set("file:///a.pk", []byte("a"))
	ws.docCache.Set("file:///b.pk", []byte("b"))

	server := &Server{
		workspace: ws,
		docCache:  ws.docCache,
		serverCtx: serverCtx,
	}

	params := &protocol.ExecuteCommandParams{
		Command: "piko.refreshDiagnostics",
	}

	result, err := server.ExecuteCommand(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "Diagnostics refresh started" {
		t.Errorf("expected status message, got %v", result)
	}
}

func TestExecuteCommand_UnknownCommand_WithDocCache_ReturnsError(t *testing.T) {
	server := &Server{
		docCache: NewDocumentCache(),
	}

	params := &protocol.ExecuteCommandParams{
		Command: "unknown.command",
	}

	_, err := server.ExecuteCommand(context.Background(), params)
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestDidChangeWatchedFiles_WithChangedFiles_TriggersAnalysis(t *testing.T) {
	ws := createTestWorkspace()
	serverCtx, cancel := context.WithCancelCause(context.Background())
	defer cancel(fmt.Errorf("test: cleanup"))

	server := &Server{
		workspace: ws,
		serverCtx: serverCtx,
	}

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []*protocol.FileEvent{
			{URI: "file:///a.pk", Type: protocol.FileChangeTypeChanged},
			{URI: "file:///b.pk", Type: protocol.FileChangeTypeCreated},
			{URI: "file:///c.pk", Type: protocol.FileChangeTypeChanged},
		},
	}

	err := server.DidChangeWatchedFiles(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDidChangeWatchedFiles_NoChangedFiles_Returns(t *testing.T) {
	ws := createTestWorkspace()
	server := &Server{workspace: ws}

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []*protocol.FileEvent{
			{URI: "file:///a.pk", Type: protocol.FileChangeTypeCreated},
			{URI: "file:///b.pk", Type: protocol.FileChangeTypeDeleted},
		},
	}

	err := server.DidChangeWatchedFiles(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestNestedCompositeToTypeScript_AllBranches(t *testing.T) {
	gen := newTypeScriptGenerator()

	testCases := []struct {
		name     string
		part     *inspector_dto.CompositePart
		expected string
	}{
		{
			name: "nested slice",
			part: &inspector_dto.CompositePart{
				CompositeType: inspector_dto.CompositeTypeSlice,
				CompositeParts: []*inspector_dto.CompositePart{
					{CompositeType: inspector_dto.CompositeTypeNone, TypeString: "string"},
				},
			},
			expected: "string[]",
		},
		{
			name: "nested array",
			part: &inspector_dto.CompositePart{
				CompositeType: inspector_dto.CompositeTypeArray,
				CompositeParts: []*inspector_dto.CompositePart{
					{CompositeType: inspector_dto.CompositeTypeNone, TypeString: "int"},
				},
			},
			expected: "number[]",
		},
		{
			name: "nested map",
			part: &inspector_dto.CompositePart{
				CompositeType: inspector_dto.CompositeTypeMap,
				CompositeParts: []*inspector_dto.CompositePart{
					{CompositeType: inspector_dto.CompositeTypeNone, TypeString: "string", Role: "key"},
					{CompositeType: inspector_dto.CompositeTypeNone, TypeString: "int", Role: "value"},
				},
			},
			expected: "Record<string, number>",
		},
		{
			name: "nested pointer",
			part: &inspector_dto.CompositePart{
				CompositeType: inspector_dto.CompositeTypePointer,
				CompositeParts: []*inspector_dto.CompositePart{
					{CompositeType: inspector_dto.CompositeTypeNone, TypeString: "bool"},
				},
			},
			expected: "boolean | null",
		},
		{
			name: "default falls back to primitive",
			part: &inspector_dto.CompositePart{
				CompositeType: inspector_dto.CompositeTypeGeneric,
				TypeString:    "float64",
			},
			expected: "number",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := gen.nestedCompositeToTypeScript(tc.part)
			if result != tc.expected {
				t.Errorf("got %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestCompositePartToTypeScript_NestedComposite(t *testing.T) {
	gen := newTypeScriptGenerator()

	part := &inspector_dto.CompositePart{
		CompositeType: inspector_dto.CompositeTypeSlice,
		CompositeParts: []*inspector_dto.CompositePart{
			{CompositeType: inspector_dto.CompositeTypeNone, TypeString: "string"},
		},
	}

	result := gen.compositePartToTypeScript(part)
	if result != "string[]" {
		t.Errorf("got %q, want %q", result, "string[]")
	}
}

func TestCompositePartToTypeScript_NilPart(t *testing.T) {
	gen := newTypeScriptGenerator()

	result := gen.compositePartToTypeScript(nil)
	if result != "unknown" {
		t.Errorf("got %q, want %q", result, "unknown")
	}
}

func TestSliceOrArrayToTypeScript_EmptyParts(t *testing.T) {
	gen := newTypeScriptGenerator()

	result := gen.sliceOrArrayToTypeScript(nil)
	if result != "unknown[]" {
		t.Errorf("got %q, want %q", result, "unknown[]")
	}
}

func TestPointerToTypeScript_EmptyParts(t *testing.T) {
	gen := newTypeScriptGenerator()

	result := gen.pointerToTypeScript(nil)
	if result != "unknown | null" {
		t.Errorf("got %q, want %q", result, "unknown | null")
	}
}

func TestGoTypeToTypeScript_CompositeType(t *testing.T) {
	gen := newTypeScriptGenerator()

	field := &inspector_dto.Field{
		CompositeType: inspector_dto.CompositeTypeSlice,
		CompositeParts: []*inspector_dto.CompositePart{
			{CompositeType: inspector_dto.CompositeTypeNone, TypeString: "int"},
		},
	}

	result := gen.goTypeToTypeScript(field)
	if result != "number[]" {
		t.Errorf("got %q, want %q", result, "number[]")
	}
}

func TestPrepareRename_DocumentNotDirty_ReturnsResult(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")

	ws.documents[testURI] = &document{
		URI:     testURI,
		Content: []byte("<template><div></div></template>"),
		dirty:   false,
	}

	server := &Server{workspace: ws}

	params := &protocol.PrepareRenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
			Position:     protocol.Position{Line: 0, Character: 5},
		},
	}

	_, err := server.PrepareRename(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSignatureHelp_FallbackPath_ReturnsResult(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")

	ws.documents[testURI] = &document{
		URI:     testURI,
		Content: []byte("<template><div></div></template>"),
		dirty:   false,
	}

	server := &Server{workspace: ws}

	params := &protocol.SignatureHelpParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
			Position:     protocol.Position{Line: 0, Character: 5},
		},
	}

	result, err := server.SignatureHelp(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestGetTypeQuerier_ManagerWithNoQuerier_ReturnsNil(t *testing.T) {
	ws := createTestWorkspace()
	ws.typeInspectorManager = &inspector_domain.TypeBuilder{}

	result := ws.getTypeQuerier(context.Background())
	if result != nil {
		t.Error("expected nil when querier is not available")
	}
}
