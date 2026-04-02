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

//go:build integration

package lsp_stress_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.lsp.dev/jsonrpc2"
	"piko.sh/piko/wdk/json"
	"go.lsp.dev/protocol"
)

type analysisCompleteParams struct {
	URI protocol.DocumentURI `json:"uri"`
}

type stressClient struct {
	conn                jsonrpc2.Conn
	t                   *testing.T
	diagnostics         map[protocol.DocumentURI][]protocol.Diagnostic
	diagnosticsReceived chan protocol.DocumentURI
	analysisComplete    map[protocol.DocumentURI]chan struct{}
	diagnosticsMu       sync.RWMutex
	analysisCompleteMu  sync.Mutex
	errors              []error
	errMu               sync.Mutex
	notificationCount   atomic.Int64
	requestCount        atomic.Int64
}

func newStressClient(t *testing.T, stream jsonrpc2.Stream) *stressClient {
	c := &stressClient{
		t:                   t,
		diagnostics:         make(map[protocol.DocumentURI][]protocol.Diagnostic),
		diagnosticsReceived: make(chan protocol.DocumentURI, 10000),
		analysisComplete:    make(map[protocol.DocumentURI]chan struct{}),
	}

	c.conn = jsonrpc2.NewConn(stream)
	c.conn.Go(context.Background(), c.handler())

	return c
}

func (c *stressClient) handler() jsonrpc2.Handler {
	return func(ctx context.Context, reply jsonrpc2.Replier, request jsonrpc2.Request) error {
		c.notificationCount.Add(1)

		switch request.Method() {
		case protocol.MethodTextDocumentPublishDiagnostics:
			var params protocol.PublishDiagnosticsParams
			if err := json.Unmarshal(request.Params(), &params); err != nil {
				c.recordError(fmt.Errorf("unmarshal diagnostics: %w", err))
				return reply(ctx, nil, err)
			}

			c.diagnosticsMu.Lock()
			c.diagnostics[params.URI] = params.Diagnostics
			c.diagnosticsMu.Unlock()

			select {
			case c.diagnosticsReceived <- params.URI:
			default:
			}

			return reply(ctx, nil, nil)

		case "piko/analysisComplete":
			var params analysisCompleteParams
			if err := json.Unmarshal(request.Params(), &params); err != nil {
				c.recordError(fmt.Errorf("unmarshal analysisComplete: %w", err))
				return reply(ctx, nil, err)
			}

			c.analysisCompleteMu.Lock()
			ch, exists := c.analysisComplete[params.URI]
			if exists {

				c.analysisComplete[params.URI] = make(chan struct{})
				close(ch)
			}
			c.analysisCompleteMu.Unlock()

			return reply(ctx, nil, nil)

		case protocol.MethodWindowShowMessage,
			protocol.MethodWindowLogMessage:
			return reply(ctx, nil, nil)

		default:
			if !strings.HasPrefix(request.Method(), "$/") {
				c.t.Logf("[StressClient] Unhandled method: %s", request.Method())
			}
			return reply(ctx, nil, nil)
		}
	}
}

func (c *stressClient) recordError(err error) {
	c.errMu.Lock()
	c.errors = append(c.errors, err)
	c.errMu.Unlock()
}

func (c *stressClient) GetErrors() []error {
	c.errMu.Lock()
	defer c.errMu.Unlock()
	result := make([]error, len(c.errors))
	copy(result, c.errors)
	return result
}

func (c *stressClient) Initialize(ctx context.Context, rootURI protocol.DocumentURI) (*protocol.InitializeResult, error) {
	c.requestCount.Add(1)
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
		return nil, fmt.Errorf("initialise request failed: %w", err)
	}
	return &result, nil
}

func (c *stressClient) Initialized(ctx context.Context) error {
	return c.conn.Notify(ctx, protocol.MethodInitialized, &protocol.InitializedParams{})
}

func (c *stressClient) DidOpen(ctx context.Context, fileURI protocol.DocumentURI, content string) error {
	c.analysisCompleteMu.Lock()
	c.analysisComplete[fileURI] = make(chan struct{})
	c.analysisCompleteMu.Unlock()

	return c.conn.Notify(ctx, protocol.MethodTextDocumentDidOpen, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        fileURI,
			LanguageID: "pk",
			Version:    1,
			Text:       content,
		},
	})
}

func (c *stressClient) DidChange(ctx context.Context, fileURI protocol.DocumentURI, version int32, content string) error {
	c.analysisCompleteMu.Lock()
	c.analysisComplete[fileURI] = make(chan struct{})
	c.analysisCompleteMu.Unlock()

	return c.conn.Notify(ctx, protocol.MethodTextDocumentDidChange, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: fileURI},
			Version:                version,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			{Text: content},
		},
	})
}

func (c *stressClient) DidSave(ctx context.Context, fileURI protocol.DocumentURI, content string) error {
	return c.conn.Notify(ctx, protocol.MethodTextDocumentDidSave, &protocol.DidSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: fileURI},
		Text:         content,
	})
}

func (c *stressClient) Completion(ctx context.Context, fileURI protocol.DocumentURI, position protocol.Position) (*protocol.CompletionList, error) {
	c.requestCount.Add(1)

	var result protocol.CompletionList
	_, err := c.conn.Call(ctx, protocol.MethodTextDocumentCompletion, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: fileURI},
			Position:     position,
		},
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("completion request failed: %w", err)
	}
	return &result, nil
}

func (c *stressClient) Hover(ctx context.Context, fileURI protocol.DocumentURI, position protocol.Position) (*protocol.Hover, error) {
	c.requestCount.Add(1)

	var result protocol.Hover
	_, err := c.conn.Call(ctx, protocol.MethodTextDocumentHover, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: fileURI},
			Position:     position,
		},
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("hover request failed: %w", err)
	}
	return &result, nil
}

func (c *stressClient) Definition(ctx context.Context, fileURI protocol.DocumentURI, position protocol.Position) ([]protocol.Location, error) {
	c.requestCount.Add(1)

	var result []protocol.Location
	_, err := c.conn.Call(ctx, protocol.MethodTextDocumentDefinition, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: fileURI},
			Position:     position,
		},
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("definition request failed: %w", err)
	}
	return result, nil
}

func (c *stressClient) WaitForAnalysisComplete(uri protocol.DocumentURI, timeout time.Duration) bool {
	c.analysisCompleteMu.Lock()
	ch, exists := c.analysisComplete[uri]
	if !exists {
		c.analysisCompleteMu.Unlock()
		return false
	}
	c.analysisCompleteMu.Unlock()

	select {
	case <-ch:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (c *stressClient) Shutdown(ctx context.Context) error {
	_, err := c.conn.Call(ctx, protocol.MethodShutdown, nil, nil)
	return err
}

func (c *stressClient) Exit(ctx context.Context) error {
	return c.conn.Notify(ctx, protocol.MethodExit, nil)
}

func (c *stressClient) Close() error {
	return c.conn.Close()
}
