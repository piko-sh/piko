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
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/config"
)

func TestNewServer_InitialisesFields(t *testing.T) {
	docCache := NewDocumentCache()
	pathsConfig := &config.PathsConfig{}

	server := NewServer(ServerDeps{
		DocCache:          docCache,
		PathsConfig:       pathsConfig,
		FormattingEnabled: true,
	})

	if server == nil {
		t.Fatal("expected non-nil server")
	}
	if server.docCache != docCache {
		t.Error("expected docCache to be set")
	}
	if server.pathsConfig != pathsConfig {
		t.Error("expected pathsConfig to be set")
	}
	if !server.formattingEnabled {
		t.Error("expected formattingEnabled to be true")
	}
}

func TestNewServer_FormattingDisabled(t *testing.T) {
	server := NewServer(ServerDeps{
		DocCache:          NewDocumentCache(),
		PathsConfig:       &config.PathsConfig{},
		FormattingEnabled: false,
	})

	if server.formattingEnabled {
		t.Error("expected formattingEnabled to be false")
	}
}

func TestBuildServerCapabilities_FormattingEnabled(t *testing.T) {
	caps := buildServerCapabilities(true)

	if enabled, ok := caps.DocumentFormattingProvider.(bool); !ok || !enabled {
		t.Error("expected DocumentFormattingProvider to be true")
	}
	if enabled, ok := caps.DocumentRangeFormattingProvider.(bool); !ok || !enabled {
		t.Error("expected DocumentRangeFormattingProvider to be true")
	}
	if caps.DocumentOnTypeFormattingProvider == nil {
		t.Error("expected DocumentOnTypeFormattingProvider to be non-nil")
	}
}

func TestBuildServerCapabilities_FormattingDisabled(t *testing.T) {
	caps := buildServerCapabilities(false)

	if enabled, ok := caps.DocumentFormattingProvider.(bool); ok && enabled {
		t.Error("expected DocumentFormattingProvider to be false")
	}
	if enabled, ok := caps.DocumentRangeFormattingProvider.(bool); ok && enabled {
		t.Error("expected DocumentRangeFormattingProvider to be false")
	}
	if caps.DocumentOnTypeFormattingProvider != nil {
		t.Error("expected DocumentOnTypeFormattingProvider to be nil")
	}
}

func TestBuildServerCapabilities_CoreCapabilities(t *testing.T) {
	caps := buildServerCapabilities(true)

	if enabled, ok := caps.HoverProvider.(bool); !ok || !enabled {
		t.Error("expected HoverProvider to be true")
	}
	if enabled, ok := caps.DefinitionProvider.(bool); !ok || !enabled {
		t.Error("expected DefinitionProvider to be true")
	}
	if enabled, ok := caps.DocumentSymbolProvider.(bool); !ok || !enabled {
		t.Error("expected DocumentSymbolProvider to be true")
	}
	if enabled, ok := caps.FoldingRangeProvider.(bool); !ok || !enabled {
		t.Error("expected FoldingRangeProvider to be true")
	}
	if enabled, ok := caps.CodeActionProvider.(bool); !ok || !enabled {
		t.Error("expected CodeActionProvider to be true")
	}
	if caps.CompletionProvider == nil {
		t.Error("expected CompletionProvider to be non-nil")
	}
	if caps.SignatureHelpProvider == nil {
		t.Error("expected SignatureHelpProvider to be non-nil")
	}
	if caps.RenameProvider == nil {
		t.Error("expected RenameProvider to be non-nil")
	}
}

func TestBuildOnTypeFormattingOptions_Enabled(t *testing.T) {
	opts := buildOnTypeFormattingOptions(true)

	if opts == nil {
		t.Fatal("expected non-nil options")
	}
	if opts.FirstTriggerCharacter != "\n" {
		t.Errorf("expected FirstTriggerCharacter to be newline, got %q", opts.FirstTriggerCharacter)
	}
	if len(opts.MoreTriggerCharacter) == 0 {
		t.Error("expected MoreTriggerCharacter to have elements")
	}
}

func TestBuildOnTypeFormattingOptions_Disabled(t *testing.T) {
	opts := buildOnTypeFormattingOptions(false)

	if opts != nil {
		t.Error("expected nil options when disabled")
	}
}

func TestServer_SetClient(t *testing.T) {
	server := &Server{}
	client := &mockClient{}

	server.SetClient(client)

	if server.client != client {
		t.Error("expected client to be set")
	}
}

func TestServer_SetClient_UpdatesWorkspace(t *testing.T) {
	ws := &workspace{
		documents:    make(map[protocol.DocumentURI]*document),
		docCache:     NewDocumentCache(),
		cancelFuncs:  make(map[protocol.DocumentURI]context.CancelCauseFunc),
		analysisDone: make(map[protocol.DocumentURI]chan struct{}),
	}
	server := &Server{workspace: ws}
	client := &mockClient{}

	server.SetClient(client)

	if ws.client != client {
		t.Error("expected workspace client to be updated")
	}
}

func TestServer_Initialized_NoError(t *testing.T) {
	server := &Server{}

	err := server.Initialized(context.Background(), &protocol.InitializedParams{})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestServer_SetTrace_NoError(t *testing.T) {
	server := &Server{}

	params := &protocol.SetTraceParams{
		Value: "messages",
	}

	err := server.SetTrace(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestServer_LogTrace_NoError(t *testing.T) {
	server := &Server{}

	params := &protocol.LogTraceParams{
		Message: "test message",
		Verbose: "verbose info",
	}

	err := server.LogTrace(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestServer_WorkDoneProgressCancel_NoError(t *testing.T) {
	server := &Server{}

	params := &protocol.WorkDoneProgressCancelParams{
		Token: *protocol.NewProgressToken("token123"),
	}

	err := server.WorkDoneProgressCancel(context.Background(), params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestServer_Shutdown_NoError(t *testing.T) {
	server := &Server{}

	err := server.Shutdown(context.Background())

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestServer_Exit_NoError(t *testing.T) {
	server := &Server{}

	err := server.Exit(context.Background())

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestServer_Request_MethodNotFound(t *testing.T) {
	server := &Server{}

	result, err := server.Request(context.Background(), "custom/method", nil)

	if err == nil {
		t.Error("expected error for unknown method")
	}
	if result != nil {
		t.Error("expected nil result")
	}
}

func TestServer_ExtractRootURI_FromWorkspaceFolders(t *testing.T) {
	server := &Server{}

	params := &protocol.InitializeParams{
		WorkspaceFolders: []protocol.WorkspaceFolder{
			{URI: "file:///workspace/project", Name: "project"},
		},
		RootURI: "file:///old/root",
	}

	uri := server.extractRootURI(params)

	if uri != "file:///workspace/project" {
		t.Errorf("expected workspace folder URI, got %q", uri)
	}
}

func TestServer_ExtractRootURI_FallbackToRootURI(t *testing.T) {
	server := &Server{}

	params := &protocol.InitializeParams{
		WorkspaceFolders: []protocol.WorkspaceFolder{},
		RootURI:          "file:///root/path",
	}

	uri := server.extractRootURI(params)

	if uri != "file:///root/path" {
		t.Errorf("expected RootURI fallback, got %q", uri)
	}
}

func TestServer_CompletionProvider_HasTriggerCharacters(t *testing.T) {
	caps := buildServerCapabilities(true)

	if caps.CompletionProvider == nil {
		t.Fatal("expected non-nil CompletionProvider")
	}
	if len(caps.CompletionProvider.TriggerCharacters) == 0 {
		t.Error("expected TriggerCharacters to have elements")
	}

	triggers := make(map[string]bool)
	for _, triggerCharacter := range caps.CompletionProvider.TriggerCharacters {
		triggers[triggerCharacter] = true
	}

	if !triggers["."] {
		t.Error("expected dot (.) in TriggerCharacters for member access")
	}

	if !triggers["-"] {
		t.Error("expected hyphen (-) in TriggerCharacters for directive completions")
	}

	if !triggers["\""] {
		t.Error("expected quote (\") in TriggerCharacters for directive value completions")
	}
}

func TestServer_SignatureHelpProvider_HasTriggerCharacters(t *testing.T) {
	caps := buildServerCapabilities(true)

	if caps.SignatureHelpProvider == nil {
		t.Fatal("expected non-nil SignatureHelpProvider")
	}
	if len(caps.SignatureHelpProvider.TriggerCharacters) == 0 {
		t.Error("expected TriggerCharacters to have elements")
	}

	if !slices.Contains(caps.SignatureHelpProvider.TriggerCharacters, "(") {
		t.Error("expected open paren (() in TriggerCharacters")
	}
}

func TestServer_GoBackground_TracksGoroutines(t *testing.T) {
	server := &Server{}

	server.serverCtx, server.serverCancel = context.WithCancelCause(context.Background())

	var completed atomic.Bool

	server.goBackground(func(_ context.Context) {
		time.Sleep(10 * time.Millisecond)
		completed.Store(true)
	})

	time.Sleep(5 * time.Millisecond)

	err := server.Shutdown(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !completed.Load() {
		t.Error("expected background goroutine to complete before Shutdown returns")
	}
}

func TestServer_GoBackground_ContextCancelledOnShutdown(t *testing.T) {
	server := &Server{}

	server.serverCtx, server.serverCancel = context.WithCancelCause(context.Background())

	var contextWasCancelled atomic.Bool

	server.goBackground(func(ctx context.Context) {
		<-ctx.Done()
		contextWasCancelled.Store(true)
	})

	time.Sleep(5 * time.Millisecond)

	err := server.Shutdown(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !contextWasCancelled.Load() {
		t.Error("expected context to be cancelled during shutdown")
	}
}

func TestServer_Shutdown_WaitsForBackgroundGoroutines(t *testing.T) {
	server := &Server{}
	server.serverCtx, server.serverCancel = context.WithCancelCause(context.Background())

	var counter atomic.Int32

	for range 5 {
		server.goBackground(func(ctx context.Context) {
			time.Sleep(10 * time.Millisecond)
			counter.Add(1)
		})
	}

	err := server.Shutdown(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if counter.Load() != 5 {
		t.Errorf("expected all 5 goroutines to complete, got %d", counter.Load())
	}
}

func TestServer_GoBackground_NilContext_UsesBackground(t *testing.T) {

	server := &Server{}

	var gotContext atomic.Bool

	server.goBackground(func(ctx context.Context) {
		if ctx != nil {
			gotContext.Store(true)
		}
	})

	server.backgroundWg.Wait()

	if !gotContext.Load() {
		t.Error("expected non-nil context to be passed")
	}
}
