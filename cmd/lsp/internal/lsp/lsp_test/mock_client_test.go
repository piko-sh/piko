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

package lsp_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"go.lsp.dev/jsonrpc2"
	"piko.sh/piko/wdk/json"
	"go.lsp.dev/protocol"
)

type AnalysisCompleteParams struct {
	URI protocol.DocumentURI `json:"uri"`
}

type MockClient struct {
	conn                  jsonrpc2.Conn
	t                     testing.TB
	diagnostics           map[protocol.DocumentURI][]protocol.Diagnostic
	diagnosticsReceived   chan protocol.DocumentURI
	analysisComplete      map[protocol.DocumentURI]chan struct{}
	diagnosticsMutex      sync.RWMutex
	analysisCompleteMutex sync.Mutex
}

func NewMockClient(t testing.TB, stream jsonrpc2.Stream) *MockClient {
	client := &MockClient{
		t:                   t,
		diagnostics:         make(map[protocol.DocumentURI][]protocol.Diagnostic),
		diagnosticsReceived: make(chan protocol.DocumentURI, 100),
		analysisComplete:    make(map[protocol.DocumentURI]chan struct{}),
	}

	client.conn = jsonrpc2.NewConn(stream)
	client.conn.Go(context.Background(), client.handler())

	return client
}

func (c *MockClient) handler() jsonrpc2.Handler {
	return func(ctx context.Context, reply jsonrpc2.Replier, request jsonrpc2.Request) error {

		switch request.Method() {
		case protocol.MethodTextDocumentPublishDiagnostics:
			var params protocol.PublishDiagnosticsParams
			if err := json.Unmarshal(request.Params(), &params); err != nil {
				c.t.Logf("[MockClient] Failed to unmarshal diagnostics: %v", err)
				return reply(ctx, nil, err)
			}

			c.diagnosticsMutex.Lock()
			c.diagnostics[params.URI] = params.Diagnostics
			c.diagnosticsMutex.Unlock()

			c.t.Logf("[MockClient] Received %d diagnostics for %s", len(params.Diagnostics), params.URI)

			select {
			case c.diagnosticsReceived <- params.URI:
			default:
				c.t.Logf("[MockClient] Warning: diagnosticsReceived channel is full.")
			}

			return reply(ctx, nil, nil)

		case "piko/analysisComplete":
			var params AnalysisCompleteParams
			if err := json.Unmarshal(request.Params(), &params); err != nil {
				c.t.Logf("[MockClient] Failed to unmarshal piko/analysisComplete params: %v", err)
				return reply(ctx, nil, err)
			}

			c.t.Logf("[MockClient] Received analysis complete signal for %s", params.URI)

			c.analysisCompleteMutex.Lock()
			analysisChannel, exists := c.analysisComplete[params.URI]
			if exists {

				close(analysisChannel)
			}
			c.analysisCompleteMutex.Unlock()

			return reply(ctx, nil, nil)

		case protocol.MethodWindowShowMessage:
			var params protocol.ShowMessageParams
			if err := json.Unmarshal(request.Params(), &params); err != nil {
				c.t.Logf("[MockClient] Failed to unmarshal show message: %v", err)
				return reply(ctx, nil, err)
			}
			c.t.Logf("[MockClient] Server message: %s", params.Message)
			return reply(ctx, nil, nil)

		case protocol.MethodWindowLogMessage:
			var params protocol.LogMessageParams
			if err := json.Unmarshal(request.Params(), &params); err != nil {
				c.t.Logf("[MockClient] Failed to unmarshal log message: %v", err)
				return reply(ctx, nil, err)
			}
			c.t.Logf("[MockClient] Server log: %s", params.Message)
			return reply(ctx, nil, nil)

		default:

			if !strings.HasPrefix(request.Method(), "$/") {
				c.t.Logf("[MockClient] Received unhandled notification/request: %s", request.Method())
			}
			return reply(ctx, nil, nil)
		}
	}
}

func (c *MockClient) Initialize(ctx context.Context, rootURI protocol.DocumentURI) (*protocol.InitializeResult, error) {
	c.t.Logf("[MockClient] Sending Setup request with rootURI: %s", rootURI)

	params := &protocol.InitializeParams{
		RootURI: rootURI,
		Capabilities: protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				Synchronization: &protocol.TextDocumentSyncClientCapabilities{
					DidSave: true,
				},
			},
		},
	}

	var result protocol.InitializeResult
	_, err := c.conn.Call(ctx, protocol.MethodInitialize, params, &result)
	if err != nil {
		return nil, fmt.Errorf("initialize request failed: %w", err)
	}

	c.t.Logf("[MockClient] Setup succeeded, server: %s", result.ServerInfo.Name)
	return &result, nil
}

func (c *MockClient) Initialized(ctx context.Context) error {
	c.t.Logf("[MockClient] Sending Initialized notification")
	params := &protocol.InitializedParams{}
	return c.conn.Notify(ctx, protocol.MethodInitialized, params)
}

func (c *MockClient) DidOpen(ctx context.Context, fileURI protocol.DocumentURI, content string) error {
	c.t.Logf("[MockClient] Sending DidOpen for %s", fileURI)

	c.analysisCompleteMutex.Lock()
	c.analysisComplete[fileURI] = make(chan struct{})
	c.analysisCompleteMutex.Unlock()

	params := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        fileURI,
			LanguageID: "pk",
			Version:    1,
			Text:       content,
		},
	}

	return c.conn.Notify(ctx, protocol.MethodTextDocumentDidOpen, params)
}

func (c *MockClient) DidChange(ctx context.Context, fileURI protocol.DocumentURI, version int32, content string) error {
	c.t.Logf("[MockClient] Sending DidChange for %s", fileURI)

	c.analysisCompleteMutex.Lock()
	c.analysisComplete[fileURI] = make(chan struct{})
	c.analysisCompleteMutex.Unlock()

	params := &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: fileURI},
			Version:                version,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			{Text: content},
		},
	}

	return c.conn.Notify(ctx, protocol.MethodTextDocumentDidChange, params)
}

func (c *MockClient) Hover(ctx context.Context, fileURI protocol.DocumentURI, position protocol.Position) (*protocol.Hover, error) {
	c.t.Logf("[MockClient] Sending Hover request for %s at %d:%d", fileURI, position.Line, position.Character)

	params := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: fileURI},
			Position:     position,
		},
	}

	var result protocol.Hover
	_, err := c.conn.Call(ctx, protocol.MethodTextDocumentHover, params, &result)
	if err != nil {
		return nil, fmt.Errorf("hover request failed: %w", err)
	}

	return &result, nil
}

func (c *MockClient) Completion(ctx context.Context, fileURI protocol.DocumentURI, position protocol.Position) (*protocol.CompletionList, error) {
	c.t.Logf("[MockClient] Sending Completion request for %s at %d:%d", fileURI, position.Line, position.Character)

	params := &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: fileURI},
			Position:     position,
		},
	}

	var result protocol.CompletionList
	_, err := c.conn.Call(ctx, protocol.MethodTextDocumentCompletion, params, &result)
	if err != nil {
		return nil, fmt.Errorf("completion request failed: %w", err)
	}

	return &result, nil
}

func (c *MockClient) Definition(ctx context.Context, fileURI protocol.DocumentURI, position protocol.Position) ([]protocol.Location, error) {
	c.t.Logf("[MockClient] Sending Definition request for %s at %d:%d", fileURI, position.Line, position.Character)

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: fileURI},
			Position:     position,
		},
	}

	var result []protocol.Location
	_, err := c.conn.Call(ctx, protocol.MethodTextDocumentDefinition, params, &result)
	if err != nil {
		return nil, fmt.Errorf("definition request failed: %w", err)
	}

	return result, nil
}

func (c *MockClient) Shutdown(ctx context.Context) error {
	c.t.Logf("[MockClient] Sending Shutdown request")
	_, err := c.conn.Call(ctx, protocol.MethodShutdown, nil, nil)
	return err
}

func (c *MockClient) Exit(ctx context.Context) error {
	c.t.Logf("[MockClient] Sending Exit notification")
	return c.conn.Notify(ctx, protocol.MethodExit, nil)
}

func (c *MockClient) GetDiagnostics(fileURI protocol.DocumentURI) []protocol.Diagnostic {
	c.diagnosticsMutex.RLock()
	defer c.diagnosticsMutex.RUnlock()
	return c.diagnostics[fileURI]
}

func (c *MockClient) WaitForDiagnostics(targetURI protocol.DocumentURI, timeout time.Duration) bool {
	timeoutChan := time.After(timeout)
	for {
		select {
		case receivedURI := <-c.diagnosticsReceived:
			if receivedURI == targetURI {
				return true
			}

		case <-timeoutChan:
			return false
		}
	}
}

func (c *MockClient) WaitForAnalysisComplete(targetURI protocol.DocumentURI, timeout time.Duration) bool {
	c.analysisCompleteMutex.Lock()
	analysisChannel, exists := c.analysisComplete[targetURI]
	if !exists {

		c.analysisCompleteMutex.Unlock()
		c.t.Logf("[MockClient] WaitForAnalysisComplete called for a URI that has not been opened yet: %s", targetURI)
		return false
	}
	c.analysisCompleteMutex.Unlock()

	select {
	case <-analysisChannel:

		return true
	case <-time.After(timeout):

		return false
	}
}

func (c *MockClient) WorkspaceSymbol(ctx context.Context, query string) ([]protocol.SymbolInformation, error) {
	c.t.Logf("[MockClient] Sending workspace/symbol request with query: %s", query)

	params := &protocol.WorkspaceSymbolParams{
		Query: query,
	}

	var result []protocol.SymbolInformation
	_, err := c.conn.Call(ctx, protocol.MethodWorkspaceSymbol, params, &result)
	if err != nil {
		return nil, fmt.Errorf("workspace/symbol request failed: %w", err)
	}

	c.t.Logf("[MockClient] Received %d workspace symbols", len(result))
	return result, nil
}

func (c *MockClient) Rename(ctx context.Context, fileURI protocol.DocumentURI, position protocol.Position, newName string) (*protocol.WorkspaceEdit, error) {
	c.t.Logf("[MockClient] Sending textDocument/rename request at %s:%d:%d -> %s", fileURI, position.Line, position.Character, newName)

	params := &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: fileURI},
			Position:     position,
		},
		NewName: newName,
	}

	var result protocol.WorkspaceEdit
	_, err := c.conn.Call(ctx, protocol.MethodTextDocumentRename, params, &result)
	if err != nil {
		return nil, fmt.Errorf("textDocument/rename request failed: %w", err)
	}

	c.t.Logf("[MockClient] Received workspace edit with %d document changes", len(result.DocumentChanges))
	return &result, nil
}

func (c *MockClient) DocumentHighlight(ctx context.Context, fileURI protocol.DocumentURI, position protocol.Position) ([]protocol.DocumentHighlight, error) {
	c.t.Logf("[MockClient] Sending textDocument/documentHighlight request at %s:%d:%d", fileURI, position.Line, position.Character)

	params := &protocol.DocumentHighlightParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: fileURI},
			Position:     position,
		},
	}

	var result []protocol.DocumentHighlight
	_, err := c.conn.Call(ctx, protocol.MethodTextDocumentDocumentHighlight, params, &result)
	if err != nil {
		return nil, fmt.Errorf("textDocument/documentHighlight request failed: %w", err)
	}

	c.t.Logf("[MockClient] Received %d document highlights", len(result))
	return result, nil
}

func (c *MockClient) FoldingRange(ctx context.Context, fileURI protocol.DocumentURI) ([]protocol.FoldingRange, error) {
	c.t.Logf("[MockClient] Sending textDocument/foldingRange request for %s", fileURI)

	params := map[string]any{
		"textDocument": map[string]any{
			"uri": fileURI,
		},
	}

	var result []protocol.FoldingRange
	_, err := c.conn.Call(ctx, protocol.MethodTextDocumentFoldingRange, params, &result)
	if err != nil {
		return nil, fmt.Errorf("textDocument/foldingRange request failed: %w", err)
	}

	c.t.Logf("[MockClient] Received %d folding ranges", len(result))
	return result, nil
}

func (c *MockClient) CodeAction(ctx context.Context, fileURI protocol.DocumentURI, textRange protocol.Range, diagnostics []protocol.Diagnostic) ([]protocol.CodeAction, error) {
	c.t.Logf("[MockClient] Sending textDocument/codeAction request at %s", fileURI)

	params := &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: fileURI},
		Range:        textRange,
		Context: protocol.CodeActionContext{
			Diagnostics: diagnostics,
		},
	}

	var result []protocol.CodeAction
	_, err := c.conn.Call(ctx, protocol.MethodTextDocumentCodeAction, params, &result)
	if err != nil {
		return nil, fmt.Errorf("textDocument/codeAction request failed: %w", err)
	}

	c.t.Logf("[MockClient] Received %d code actions", len(result))
	return result, nil
}

func (c *MockClient) Close() error {
	c.t.Logf("[MockClient] Closing connection")
	return c.conn.Close()
}
