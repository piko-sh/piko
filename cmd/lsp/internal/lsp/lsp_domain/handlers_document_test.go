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
	"fmt"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"piko.sh/piko/internal/formatter/formatter_domain"
)

func createTestServerWithFormatting(formattingEnabled bool) *Server {
	docCache := NewDocumentCache()
	return &Server{
		formattingEnabled: formattingEnabled,
		docCache:          docCache,
	}
}

func TestOnTypeFormatting_DisabledReturnsEmpty(t *testing.T) {
	server := createTestServerWithFormatting(false)

	params := &protocol.DocumentOnTypeFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: uri.URI("file:///test.pk"),
		},
		Position: protocol.Position{Line: 5, Character: 0},
		Ch:       "\n",
	}

	edits, err := server.OnTypeFormatting(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(edits) != 0 {
		t.Errorf("expected empty edits when formatting is disabled, got %d edits", len(edits))
	}
}

func TestFormatting_DisabledReturnsEmpty(t *testing.T) {
	server := createTestServerWithFormatting(false)

	params := &protocol.DocumentFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: uri.URI("file:///test.pk"),
		},
	}

	edits, err := server.Formatting(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(edits) != 0 {
		t.Errorf("expected empty edits when formatting is disabled, got %d edits", len(edits))
	}
}

func TestRangeFormatting_DisabledReturnsEmpty(t *testing.T) {
	server := createTestServerWithFormatting(false)

	params := &protocol.DocumentRangeFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: uri.URI("file:///test.pk"),
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 10, Character: 0},
		},
	}

	edits, err := server.RangeFormatting(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(edits) != 0 {
		t.Errorf("expected empty edits when formatting is disabled, got %d edits", len(edits))
	}
}

func TestWillSaveWaitUntil_DisabledReturnsEmpty(t *testing.T) {
	server := createTestServerWithFormatting(false)

	params := &protocol.WillSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: uri.URI("file:///test.pk"),
		},
		Reason: protocol.TextDocumentSaveReasonManual,
	}

	edits, err := server.WillSaveWaitUntil(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(edits) != 0 {
		t.Errorf("expected empty edits when formatting is disabled, got %d edits", len(edits))
	}
}

func TestCalculateFormatRange(t *testing.T) {
	testCases := []struct {
		name        string
		triggerChar string
		currentLine uint32
		expected    formatter_domain.Range
	}{
		{
			name:        "newline trigger includes previous line",
			triggerChar: "\n",
			currentLine: 5,
			expected: formatter_domain.Range{
				StartLine:      4,
				StartCharacter: 0,
				EndLine:        6,
				EndCharacter:   0,
			},
		},
		{
			name:        "newline trigger at line zero does not underflow",
			triggerChar: "\n",
			currentLine: 0,
			expected: formatter_domain.Range{
				StartLine:      0,
				StartCharacter: 0,
				EndLine:        1,
				EndCharacter:   0,
			},
		},
		{
			name:        "closing angle bracket includes previous line",
			triggerChar: ">",
			currentLine: 10,
			expected: formatter_domain.Range{
				StartLine:      9,
				StartCharacter: 0,
				EndLine:        11,
				EndCharacter:   0,
			},
		},
		{
			name:        "closing angle bracket at line zero does not underflow",
			triggerChar: ">",
			currentLine: 0,
			expected: formatter_domain.Range{
				StartLine:      0,
				StartCharacter: 0,
				EndLine:        1,
				EndCharacter:   0,
			},
		},
		{
			name:        "other character formats single line",
			triggerChar: "a",
			currentLine: 7,
			expected: formatter_domain.Range{
				StartLine:      7,
				StartCharacter: 0,
				EndLine:        8,
				EndCharacter:   0,
			},
		},
		{
			name:        "closing brace formats single line",
			triggerChar: "}",
			currentLine: 3,
			expected: formatter_domain.Range{
				StartLine:      3,
				StartCharacter: 0,
				EndLine:        4,
				EndCharacter:   0,
			},
		},
		{
			name:        "empty trigger char formats single line",
			triggerChar: "",
			currentLine: 2,
			expected: formatter_domain.Range{
				StartLine:      2,
				StartCharacter: 0,
				EndLine:        3,
				EndCharacter:   0,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateFormatRange(tc.triggerChar, tc.currentLine)

			if result.StartLine != tc.expected.StartLine {
				t.Errorf("StartLine = %d, want %d", result.StartLine, tc.expected.StartLine)
			}
			if result.StartCharacter != tc.expected.StartCharacter {
				t.Errorf("StartCharacter = %d, want %d", result.StartCharacter, tc.expected.StartCharacter)
			}
			if result.EndLine != tc.expected.EndLine {
				t.Errorf("EndLine = %d, want %d", result.EndLine, tc.expected.EndLine)
			}
			if result.EndCharacter != tc.expected.EndCharacter {
				t.Errorf("EndCharacter = %d, want %d", result.EndCharacter, tc.expected.EndCharacter)
			}
		})
	}
}

func TestDidOpen_DelegatesToDidOpenTextDocument(t *testing.T) {
	ws := createTestWorkspace()
	serverCtx, cancel := context.WithCancelCause(context.Background())
	defer cancel(fmt.Errorf("test: cleanup"))

	server := &Server{
		workspace: ws,
		docCache:  ws.docCache,
		serverCtx: serverCtx,
	}

	testURI := protocol.DocumentURI("file:///test.pk")
	params := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:  testURI,
			Text: "<template>hello</template>",
		},
	}

	err := server.DidOpen(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	cached, found := ws.docCache.Get(testURI)
	if !found {
		t.Fatal("expected document to be in cache after DidOpen")
	}
	if string(cached) != "<template>hello</template>" {
		t.Errorf("cached content = %q, want %q", cached, "<template>hello</template>")
	}
}

func TestDidChange_UpdatesDocument(t *testing.T) {
	ws := createTestWorkspace()
	serverCtx, cancel := context.WithCancelCause(context.Background())
	defer cancel(fmt.Errorf("test: cleanup"))

	server := &Server{
		workspace: ws,
		docCache:  ws.docCache,
		serverCtx: serverCtx,
	}

	testURI := protocol.DocumentURI("file:///test.pk")

	params := &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: testURI},
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			{Text: "<template>changed</template>"},
		},
	}

	err := server.DidChange(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	cached, found := ws.docCache.Get(testURI)
	if !found {
		t.Fatal("expected document to be in cache after DidChange")
	}
	if string(cached) != "<template>changed</template>" {
		t.Errorf("cached content = %q, want %q", cached, "<template>changed</template>")
	}
}

func TestDidChangeTextDocument_EmptyContentChanges_NoUpdate(t *testing.T) {
	ws := createTestWorkspace()
	serverCtx, cancel := context.WithCancelCause(context.Background())
	defer cancel(fmt.Errorf("test: cleanup"))

	server := &Server{
		workspace: ws,
		docCache:  ws.docCache,
		serverCtx: serverCtx,
	}

	testURI := protocol.DocumentURI("file:///test.pk")

	params := &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: testURI},
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{},
	}

	err := server.DidChangeTextDocument(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if _, found := ws.docCache.Get(testURI); found {
		t.Error("expected document not to be in cache with empty content changes")
	}
}

func TestDidSave_WithText_UpdatesDocument(t *testing.T) {
	ws := createTestWorkspace()
	serverCtx, cancel := context.WithCancelCause(context.Background())
	defer cancel(fmt.Errorf("test: cleanup"))

	server := &Server{
		workspace: ws,
		docCache:  ws.docCache,
		serverCtx: serverCtx,
	}

	testURI := protocol.DocumentURI("file:///test.pk")

	params := &protocol.DidSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
		Text:         "<template>saved</template>",
	}

	err := server.DidSave(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	cached, found := ws.docCache.Get(testURI)
	if !found {
		t.Fatal("expected document to be in cache after DidSave with text")
	}
	if string(cached) != "<template>saved</template>" {
		t.Errorf("cached content = %q, want %q", cached, "<template>saved</template>")
	}
}

func TestDidSaveTextDocument_EmptyText_DoesNotUpdateContent(t *testing.T) {
	ws := createTestWorkspace()
	serverCtx, cancel := context.WithCancelCause(context.Background())
	defer cancel(fmt.Errorf("test: cleanup"))

	testURI := protocol.DocumentURI("file:///test.pk")
	ws.docCache.Set(testURI, []byte("<template>original</template>"))

	server := &Server{
		workspace: ws,
		docCache:  ws.docCache,
		serverCtx: serverCtx,
	}

	params := &protocol.DidSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
		Text:         "",
	}

	err := server.DidSaveTextDocument(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	cached, found := ws.docCache.Get(testURI)
	if !found {
		t.Fatal("expected document to still be in cache")
	}
	if string(cached) != "<template>original</template>" {
		t.Errorf("cached content = %q, want %q", cached, "<template>original</template>")
	}
}

func TestDidClose_RemovesDocument(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///test.pk")
	ws.documents[testURI] = &document{URI: testURI, Content: []byte("test")}
	ws.docCache.Set(testURI, []byte("test"))

	server := &Server{
		workspace: ws,
		docCache:  ws.docCache,
	}

	params := &protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
	}

	err := server.DidClose(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if _, exists := ws.GetDocument(testURI); exists {
		t.Error("expected document to be removed from workspace after DidClose")
	}
}

func TestWillSave_ReturnsNil(t *testing.T) {
	server := &Server{}

	params := &protocol.WillSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: uri.URI("file:///test.pk"),
		},
		Reason: protocol.TextDocumentSaveReasonManual,
	}

	err := server.WillSave(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestWillSave_AfterSave_ReturnsNil(t *testing.T) {
	server := &Server{}

	params := &protocol.WillSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: uri.URI("file:///test.pk"),
		},
		Reason: protocol.TextDocumentSaveReasonAfterDelay,
	}

	err := server.WillSave(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDidOpenTextDocument_CreatesDocumentEntry(t *testing.T) {
	ws := createTestWorkspace()
	serverCtx, cancel := context.WithCancelCause(context.Background())
	defer cancel(fmt.Errorf("test: cleanup"))

	server := &Server{
		workspace: ws,
		docCache:  ws.docCache,
		serverCtx: serverCtx,
	}

	testURI := protocol.DocumentURI("file:///new-document.pk")
	params := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:  testURI,
			Text: "<template>new</template>",
		},
	}

	err := server.DidOpenTextDocument(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	document, exists := ws.GetDocument(testURI)
	if !exists {
		t.Fatal("expected document to be in workspace after DidOpenTextDocument")
	}
	if !document.dirty {
		t.Error("expected document to be marked dirty after opening")
	}
}

func TestDidCloseTextDocument_RemovesDocumentAndClearsCache(t *testing.T) {
	ws := createTestWorkspace()
	testURI := protocol.DocumentURI("file:///closing.pk")
	ws.documents[testURI] = &document{URI: testURI, Content: []byte("content")}
	ws.docCache.Set(testURI, []byte("content"))

	server := &Server{
		workspace: ws,
		docCache:  ws.docCache,
	}

	params := &protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
	}

	err := server.DidCloseTextDocument(context.Background(), params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if _, exists := ws.GetDocument(testURI); exists {
		t.Error("expected document to be removed from workspace")
	}
	if _, found := ws.docCache.Get(testURI); found {
		t.Error("expected document to be removed from docCache")
	}
}
