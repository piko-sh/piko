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
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestCodeAction_AnalysisFails_ReturnsBaseActions(t *testing.T) {
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

	params := &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///nonexistent.pk",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	}

	result, err := server.CodeAction(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if len(result) < 2 {
		t.Errorf("expected at least 2 base actions, got %d", len(result))
	}

	hasFormat := false
	hasRefresh := false
	for _, action := range result {
		if action.Title == "Format Document" {
			hasFormat = true
		}
		if action.Title == "Refresh Diagnostics" {
			hasRefresh = true
		}
	}
	if !hasFormat {
		t.Error("expected Format Document action")
	}
	if !hasRefresh {
		t.Error("expected Refresh Diagnostics action")
	}
}

func TestCodeAction_WithDiagnostics_GeneratesQuickFixes(t *testing.T) {
	uri := protocol.DocumentURI("file:///test.pk")
	content := []byte(`<template><div>{{ undefined }}</div></template>`)

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

	params := &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 18},
			End:   protocol.Position{Line: 0, Character: 27},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 18},
						End:   protocol.Position{Line: 0, Character: 27},
					},
					Message:  "undefined: undefined",
					Severity: protocol.DiagnosticSeverityError,
				},
			},
		},
	}

	result, err := server.CodeAction(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if len(result) < 2 {
		t.Errorf("expected at least base actions, got %d", len(result))
	}
}

func TestDocumentSymbol_AnalysisFails_ReturnsEmptyList(t *testing.T) {
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

	params := &protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///nonexistent.pk",
		},
	}

	result, err := server.DocumentSymbol(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d symbols", len(result))
	}
}

func TestSymbols_EmptyQuery_ReturnsEmptyList(t *testing.T) {
	ws := &workspace{
		documents:    make(map[protocol.DocumentURI]*document),
		docCache:     NewDocumentCache(),
		cancelFuncs:  make(map[protocol.DocumentURI]context.CancelCauseFunc),
		analysisDone: make(map[protocol.DocumentURI]chan struct{}),
		client:       &mockClient{},
	}

	server := &Server{
		workspace: ws,
	}

	params := &protocol.WorkspaceSymbolParams{
		Query: "",
	}

	result, err := server.Symbols(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice for empty query, got %d symbols", len(result))
	}
}

func TestSymbols_ShortQuery_ReturnsEmptyList(t *testing.T) {
	ws := &workspace{
		documents:    make(map[protocol.DocumentURI]*document),
		docCache:     NewDocumentCache(),
		cancelFuncs:  make(map[protocol.DocumentURI]context.CancelCauseFunc),
		analysisDone: make(map[protocol.DocumentURI]chan struct{}),
		client:       &mockClient{},
	}

	server := &Server{
		workspace: ws,
	}

	params := &protocol.WorkspaceSymbolParams{
		Query: "a",
	}

	result, err := server.Symbols(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice for short query, got %d symbols", len(result))
	}
}

func TestSymbols_ValidQuery_SearchesBothSources(t *testing.T) {
	ws := &workspace{
		documents:    make(map[protocol.DocumentURI]*document),
		docCache:     NewDocumentCache(),
		cancelFuncs:  make(map[protocol.DocumentURI]context.CancelCauseFunc),
		analysisDone: make(map[protocol.DocumentURI]chan struct{}),
		client:       &mockClient{},
	}

	server := &Server{
		workspace: ws,
	}

	params := &protocol.WorkspaceSymbolParams{
		Query: "myWidget",
	}

	result, err := server.Symbols(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty result with no documents, got %d symbols", len(result))
	}
}

func TestCodeLens_ReturnsEmptyList(t *testing.T) {
	server := &Server{}

	params := &protocol.CodeLensParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.pk",
		},
	}

	result, err := server.CodeLens(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d code lenses", len(result))
	}
}

func TestCodeLensResolve_ReturnsInput(t *testing.T) {
	server := &Server{}

	params := &protocol.CodeLens{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
	}

	result, err := server.CodeLensResolve(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != params {
		t.Error("expected CodeLensResolve to return input params")
	}
}

func TestDocumentLink_AnalysisFails_ReturnsEmptyList(t *testing.T) {
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

	params := &protocol.DocumentLinkParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///nonexistent.pk",
		},
	}

	result, err := server.DocumentLink(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d links", len(result))
	}
}

func TestDocumentLinkResolve_ReturnsInput(t *testing.T) {
	server := &Server{}

	params := &protocol.DocumentLink{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
	}

	result, err := server.DocumentLinkResolve(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != params {
		t.Error("expected DocumentLinkResolve to return input params")
	}
}

func TestDocumentColor_AnalysisFails_ReturnsEmptyList(t *testing.T) {
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

	params := &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///nonexistent.pk",
		},
	}

	result, err := server.DocumentColor(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d colors", len(result))
	}
}

func TestColorPresentation_AnalysisFails_ReturnsEmptyList(t *testing.T) {
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

	params := &protocol.ColorPresentationParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///nonexistent.pk",
		},
		Color: protocol.Color{Red: 1.0, Green: 0.0, Blue: 0.0, Alpha: 1.0},
	}

	result, err := server.ColorPresentation(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d presentations", len(result))
	}
}

func TestExecuteCommand_UnknownCommand_ReturnsError(t *testing.T) {
	server := &Server{}

	params := &protocol.ExecuteCommandParams{
		Command:   "unknown.command",
		Arguments: []any{},
	}

	_, err := server.ExecuteCommand(context.Background(), params)

	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestDidChangeConfiguration_NoError(t *testing.T) {
	server := &Server{}

	params := &protocol.DidChangeConfigurationParams{
		Settings: nil,
	}

	err := server.DidChangeConfiguration(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDidChangeWorkspaceFolders_NoError(t *testing.T) {
	server := &Server{}

	params := &protocol.DidChangeWorkspaceFoldersParams{
		Event: protocol.WorkspaceFoldersChangeEvent{
			Added:   []protocol.WorkspaceFolder{{URI: "file:///new", Name: "new"}},
			Removed: []protocol.WorkspaceFolder{{URI: "file:///old", Name: "old"}},
		},
	}

	err := server.DidChangeWorkspaceFolders(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestWillCreateFiles_ReturnsEmptyEdit(t *testing.T) {
	server := &Server{}

	params := &protocol.CreateFilesParams{
		Files: []protocol.FileCreate{{URI: "file:///new.pk"}},
	}

	result, err := server.WillCreateFiles(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Changes == nil {
		t.Error("expected non-nil Changes map")
	}
}

func TestWillRenameFiles_ReturnsEmptyEdit(t *testing.T) {
	server := &Server{}

	params := &protocol.RenameFilesParams{
		Files: []protocol.FileRename{{OldURI: "file:///old.pk", NewURI: "file:///new.pk"}},
	}

	result, err := server.WillRenameFiles(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Changes == nil {
		t.Error("expected non-nil Changes map")
	}
}

func TestWillDeleteFiles_ReturnsEmptyEdit(t *testing.T) {
	server := &Server{}

	params := &protocol.DeleteFilesParams{
		Files: []protocol.FileDelete{{URI: "file:///delete.pk"}},
	}

	result, err := server.WillDeleteFiles(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Changes == nil {
		t.Error("expected non-nil Changes map")
	}
}

func TestShowDocument_ReturnsFalse(t *testing.T) {
	server := &Server{}

	params := &protocol.ShowDocumentParams{
		URI: "file:///test.pk",
	}

	result, err := server.ShowDocument(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Success {
		t.Error("expected Success=false")
	}
}

func TestSemanticTokensFull_ReturnsEmptyData(t *testing.T) {
	server := &Server{}

	params := &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.pk",
		},
	}

	result, err := server.SemanticTokensFull(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Data) != 0 {
		t.Errorf("expected empty data, got %d items", len(result.Data))
	}
}

func TestSemanticTokensRange_ReturnsEmptyData(t *testing.T) {
	server := &Server{}

	params := &protocol.SemanticTokensRangeParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.pk",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 10, Character: 0},
		},
	}

	result, err := server.SemanticTokensRange(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Data) != 0 {
		t.Errorf("expected empty data, got %d items", len(result.Data))
	}
}

func TestCodeLensRefresh_NoError(t *testing.T) {
	server := &Server{}

	err := server.CodeLensRefresh(context.Background())

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSemanticTokensRefresh_NoError(t *testing.T) {
	server := &Server{}

	err := server.SemanticTokensRefresh(context.Background())

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestMapGoSymbolKind(t *testing.T) {
	testCases := []struct {
		name     string
		kind     string
		expected protocol.SymbolKind
	}{
		{name: "type maps to Class", kind: "type", expected: protocol.SymbolKindClass},
		{name: "function maps to Function", kind: "function", expected: protocol.SymbolKindFunction},
		{name: "method maps to Method", kind: "method", expected: protocol.SymbolKindMethod},
		{name: "field maps to Field", kind: "field", expected: protocol.SymbolKindField},
		{name: "unknown maps to Variable", kind: "unknown", expected: protocol.SymbolKindVariable},
		{name: "empty string maps to Variable", kind: "", expected: protocol.SymbolKindVariable},
		{name: "arbitrary string maps to Variable", kind: "constant", expected: protocol.SymbolKindVariable},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mapGoSymbolKind(tc.kind)
			if result != tc.expected {
				t.Errorf("mapGoSymbolKind(%q) = %v, want %v", tc.kind, result, tc.expected)
			}
		})
	}
}

func TestBuildContainerName(t *testing.T) {
	testCases := []struct {
		name          string
		packageName   string
		containerName string
		expected      string
	}{
		{name: "with container name", packageName: "models", containerName: "User", expected: "models.User"},
		{name: "empty container name returns package only", packageName: "models", containerName: "", expected: "models"},
		{name: "both empty", packageName: "", containerName: "", expected: ""},
		{name: "empty package with container", packageName: "", containerName: "User", expected: ".User"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildContainerName(tc.packageName, tc.containerName)
			if result != tc.expected {
				t.Errorf("buildContainerName(%q, %q) = %q, want %q", tc.packageName, tc.containerName, result, tc.expected)
			}
		})
	}
}

func TestBuildSymbolWorkspaceLocation(t *testing.T) {
	testCases := []struct {
		name     string
		filePath string
		symName  string
		wantURI  protocol.DocumentURI
		line     int
		column   int
		wantLine uint32
		wantCol  uint32
		wantEnd  uint32
	}{
		{
			name:     "standard position converts to zero-based",
			filePath: "/home/user/project/main.go",
			line:     10,
			column:   5,
			symName:  "Foo",
			wantURI:  "file:///home/user/project/main.go",
			wantLine: 9,
			wantCol:  4,
			wantEnd:  7,
		},
		{
			name:     "line 1 column 1 maps to zero",
			filePath: "/test/file.go",
			line:     1,
			column:   1,
			symName:  "X",
			wantURI:  "file:///test/file.go",
			wantLine: 0,
			wantCol:  0,
			wantEnd:  1,
		},
		{
			name:     "long symbol name extends end position",
			filePath: "/test/file.go",
			line:     5,
			column:   3,
			symName:  "LongFunctionName",
			wantURI:  "file:///test/file.go",
			wantLine: 4,
			wantCol:  2,
			wantEnd:  18,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			loc := buildSymbolLocation(tc.filePath, tc.line, tc.column, tc.symName)

			if loc.URI != tc.wantURI {
				t.Errorf("URI = %q, want %q", loc.URI, tc.wantURI)
			}
			if loc.Range.Start.Line != tc.wantLine {
				t.Errorf("Start.Line = %d, want %d", loc.Range.Start.Line, tc.wantLine)
			}
			if loc.Range.Start.Character != tc.wantCol {
				t.Errorf("Start.Character = %d, want %d", loc.Range.Start.Character, tc.wantCol)
			}
			if loc.Range.End.Line != tc.wantLine {
				t.Errorf("End.Line = %d, want %d", loc.Range.End.Line, tc.wantLine)
			}
			if loc.Range.End.Character != tc.wantEnd {
				t.Errorf("End.Character = %d, want %d", loc.Range.End.Character, tc.wantEnd)
			}
		})
	}
}

func TestConvertGoSymbolToLSP(t *testing.T) {
	testCases := []struct {
		name       string
		queryLower string
		wantName   string
		sym        inspector_dto.WorkspaceSymbol
		wantKind   protocol.SymbolKind
		wantNil    bool
	}{
		{
			name: "matching symbol is converted",
			sym: inspector_dto.WorkspaceSymbol{
				Name:          "HandleRequest",
				Kind:          "function",
				PackageName:   "handlers",
				ContainerName: "",
				FilePath:      "/project/handlers.go",
				Line:          10,
				Column:        6,
			},
			queryLower: "handle",
			wantNil:    false,
			wantName:   "HandleRequest",
			wantKind:   protocol.SymbolKindFunction,
		},
		{
			name: "non-matching query returns nil",
			sym: inspector_dto.WorkspaceSymbol{
				Name:     "HandleRequest",
				Kind:     "function",
				FilePath: "/project/handlers.go",
				Line:     10,
				Column:   6,
			},
			queryLower: "zzzzz",
			wantNil:    true,
		},
		{
			name: "empty file path returns nil",
			sym: inspector_dto.WorkspaceSymbol{
				Name:     "HandleRequest",
				Kind:     "function",
				FilePath: "",
				Line:     10,
				Column:   6,
			},
			queryLower: "handle",
			wantNil:    true,
		},
		{
			name: "zero line returns nil",
			sym: inspector_dto.WorkspaceSymbol{
				Name:     "HandleRequest",
				Kind:     "function",
				FilePath: "/project/handlers.go",
				Line:     0,
				Column:   6,
			},
			queryLower: "handle",
			wantNil:    true,
		},
		{
			name: "case-insensitive matching works",
			sym: inspector_dto.WorkspaceSymbol{
				Name:          "MyStruct",
				Kind:          "type",
				PackageName:   "models",
				ContainerName: "",
				FilePath:      "/project/models.go",
				Line:          5,
				Column:        6,
			},
			queryLower: "mystruct",
			wantNil:    false,
			wantName:   "MyStruct",
			wantKind:   protocol.SymbolKindClass,
		},
		{
			name: "method with container name is converted",
			sym: inspector_dto.WorkspaceSymbol{
				Name:          "Save",
				Kind:          "method",
				PackageName:   "models",
				ContainerName: "User",
				FilePath:      "/project/user.go",
				Line:          20,
				Column:        1,
			},
			queryLower: "save",
			wantNil:    false,
			wantName:   "Save",
			wantKind:   protocol.SymbolKindMethod,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertGoSymbolToLSP(tc.sym, tc.queryLower)

			if tc.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if result.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tc.wantName)
			}
			if result.Kind != tc.wantKind {
				t.Errorf("Kind = %v, want %v", result.Kind, tc.wantKind)
			}
		})
	}
}

func TestFindIDSymbolInNode(t *testing.T) {
	testCases := []struct {
		node       *ast_domain.TemplateNode
		name       string
		uri        protocol.DocumentURI
		queryLower string
		wantName   string
		wantNil    bool
	}{
		{
			name: "matching ID attribute returns symbol",
			uri:  "file:///test.pk",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 5, 1)
				addAttribute(n, "id", "main-header")
				return n
			}(),
			queryLower: "main",
			wantNil:    false,
			wantName:   "main-header",
		},
		{
			name: "non-matching ID returns nil",
			uri:  "file:///test.pk",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 5, 1)
				addAttribute(n, "id", "footer")
				return n
			}(),
			queryLower: "header",
			wantNil:    true,
		},
		{
			name: "non-ID attribute is ignored",
			uri:  "file:///test.pk",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 5, 1)
				addAttribute(n, "class", "main-header")
				return n
			}(),
			queryLower: "main",
			wantNil:    true,
		},
		{
			name:       "node with no attributes returns nil",
			uri:        "file:///test.pk",
			node:       newTestNode("div", 5, 1),
			queryLower: "anything",
			wantNil:    true,
		},
		{
			name: "case-insensitive ID matching",
			uri:  "file:///test.pk",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("section", 3, 1)
				addAttribute(n, "id", "MyWidget")
				return n
			}(),
			queryLower: "mywidget",
			wantNil:    false,
			wantName:   "MyWidget",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findIDSymbolInNode(tc.uri, tc.node, tc.queryLower)

			if tc.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if result.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tc.wantName)
			}
			if result.Kind != protocol.SymbolKindClass {
				t.Errorf("Kind = %v, want %v (SymbolKindClass)", result.Kind, protocol.SymbolKindClass)
			}
			if result.Location.URI != tc.uri {
				t.Errorf("Location.URI = %q, want %q", result.Location.URI, tc.uri)
			}
		})
	}
}

func TestSearchDocumentForIDs(t *testing.T) {
	testCases := []struct {
		name       string
		uri        protocol.DocumentURI
		document   *document
		queryLower string
		wantCount  int
	}{
		{
			name: "nil annotation result returns nil",
			uri:  "file:///test.pk",
			document: &document{
				AnnotationResult: nil,
			},
			queryLower: "test",
			wantCount:  0,
		},
		{
			name: "nil annotated AST returns nil",
			uri:  "file:///test.pk",
			document: &document{
				AnnotationResult: &annotator_dto.AnnotationResult{
					AnnotatedAST: nil,
				},
			},
			queryLower: "test",
			wantCount:  0,
		},
		{
			name: "finds matching IDs in AST",
			uri:  "file:///test.pk",
			document: func() *document {
				node1 := newTestNode("div", 1, 1)
				addAttribute(node1, "id", "header")

				node2 := newTestNode("span", 2, 1)
				addAttribute(node2, "id", "header-title")

				node3 := newTestNode("p", 3, 1)
				addAttribute(node3, "id", "footer")

				return &document{
					AnnotationResult: &annotator_dto.AnnotationResult{
						AnnotatedAST: newTestAnnotatedAST(node1, node2, node3),
					},
				}
			}(),
			queryLower: "header",
			wantCount:  2,
		},
		{
			name: "no matching IDs returns empty",
			uri:  "file:///test.pk",
			document: func() *document {
				node := newTestNode("div", 1, 1)
				addAttribute(node, "id", "sidebar")
				return &document{
					AnnotationResult: &annotator_dto.AnnotationResult{
						AnnotatedAST: newTestAnnotatedAST(node),
					},
				}
			}(),
			queryLower: "zzzzz",
			wantCount:  0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := searchDocumentForIDs(tc.uri, tc.document, tc.queryLower)

			if len(result) != tc.wantCount {
				t.Errorf("got %d symbols, want %d", len(result), tc.wantCount)
			}
		})
	}
}

func TestDidChangeWatchedFiles_NoChangedFiles_ReturnsNil(t *testing.T) {
	server := &Server{
		workspace: createTestWorkspace(),
		docCache:  NewDocumentCache(),
	}

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []*protocol.FileEvent{
			{URI: "file:///test.pk", Type: protocol.FileChangeTypeCreated},
			{URI: "file:///test2.pk", Type: protocol.FileChangeTypeDeleted},
		},
	}

	err := server.DidChangeWatchedFiles(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDidChangeWatchedFiles_EmptyChanges_ReturnsNil(t *testing.T) {
	server := &Server{
		workspace: createTestWorkspace(),
		docCache:  NewDocumentCache(),
	}

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []*protocol.FileEvent{},
	}

	err := server.DidChangeWatchedFiles(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDidCreateFiles_ReturnsNil(t *testing.T) {
	server := &Server{}

	params := &protocol.CreateFilesParams{
		Files: []protocol.FileCreate{
			{URI: "file:///new1.pk"},
			{URI: "file:///new2.pk"},
		},
	}

	err := server.DidCreateFiles(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDidCreateFiles_EmptyFiles_ReturnsNil(t *testing.T) {
	server := &Server{}

	params := &protocol.CreateFilesParams{
		Files: []protocol.FileCreate{},
	}

	err := server.DidCreateFiles(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDidRenameFiles_UpdatesCache(t *testing.T) {
	docCache := NewDocumentCache()
	oldURI := protocol.DocumentURI("file:///old.pk")
	newURI := protocol.DocumentURI("file:///new.pk")
	content := []byte("<template>test</template>")
	docCache.Set(oldURI, content)

	server := &Server{
		docCache: docCache,
	}

	params := &protocol.RenameFilesParams{
		Files: []protocol.FileRename{
			{OldURI: string(oldURI), NewURI: string(newURI)},
		},
	}

	err := server.DidRenameFiles(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if _, found := docCache.Get(oldURI); found {
		t.Error("expected old URI to be removed from cache")
	}

	cached, found := docCache.Get(newURI)
	if !found {
		t.Fatal("expected new URI to be in cache")
	}
	if string(cached) != string(content) {
		t.Errorf("cached content = %q, want %q", cached, content)
	}
}

func TestDidRenameFiles_OldURINotInCache_NoError(t *testing.T) {
	docCache := NewDocumentCache()
	server := &Server{
		docCache: docCache,
	}

	params := &protocol.RenameFilesParams{
		Files: []protocol.FileRename{
			{OldURI: "file:///nonexistent.pk", NewURI: "file:///new.pk"},
		},
	}

	err := server.DidRenameFiles(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDidDeleteFiles_RemovesFromCache(t *testing.T) {
	docCache := NewDocumentCache()
	uri := protocol.DocumentURI("file:///to-delete.pk")
	docCache.Set(uri, []byte("<template>bye</template>"))

	server := &Server{
		docCache: docCache,
	}

	params := &protocol.DeleteFilesParams{
		Files: []protocol.FileDelete{
			{URI: string(uri)},
		},
	}

	err := server.DidDeleteFiles(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if _, found := docCache.Get(uri); found {
		t.Error("expected URI to be removed from cache after delete")
	}
}

func TestDidDeleteFiles_NonexistentURI_NoError(t *testing.T) {
	docCache := NewDocumentCache()
	server := &Server{
		docCache: docCache,
	}

	params := &protocol.DeleteFilesParams{
		Files: []protocol.FileDelete{
			{URI: "file:///nonexistent.pk"},
		},
	}

	err := server.DidDeleteFiles(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSearchGoSymbols_NilTypeInspectorManager_ReturnsNil(t *testing.T) {
	ws := createTestWorkspace()
	ws.typeInspectorManager = nil

	server := &Server{
		workspace: ws,
	}

	result := server.searchGoSymbols("test")
	if result != nil {
		t.Errorf("expected nil when typeInspectorManager is nil, got %v", result)
	}
}

func TestSearchPKDocuments_FindsMatchingIDs(t *testing.T) {
	ws := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")

	node := newTestNode("div", 1, 1)
	addAttribute(node, "id", "my-widget")

	ws.documents[uri] = &document{
		URI: uri,
		AnnotationResult: &annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		},
	}

	server := &Server{
		workspace: ws,
	}

	result := server.searchPKDocuments("widget")

	if len(result) != 1 {
		t.Errorf("expected 1 symbol, got %d", len(result))
	}
}

func TestSearchPKDocuments_EmptyDocuments_ReturnsEmpty(t *testing.T) {
	ws := createTestWorkspace()

	server := &Server{
		workspace: ws,
	}

	result := server.searchPKDocuments("test")
	if len(result) != 0 {
		t.Errorf("expected 0 symbols, got %d", len(result))
	}
}
