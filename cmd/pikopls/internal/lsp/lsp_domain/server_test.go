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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestServer_RunBoundedAnalysis_RecoversWorkerPanic(t *testing.T) {

	ws := &workspace{
		documents:           make(map[protocol.DocumentURI]*document),
		docCache:            NewDocumentCache(),
		analysisDone:        make(map[protocol.DocumentURI]chan struct{}),
		requestedGeneration: make(map[protocol.DocumentURI]uint64),
		client:              &mockClient{},
	}
	server := &Server{
		workspace:         ws,
		analysisSemaphore: make(chan struct{}, maxConcurrentAnalysis),
	}

	server.runBoundedAnalysis(context.Background(), []protocol.DocumentURI{"file:///panic-a.pk", "file:///panic-b.pk"}, "panic-recovery-1")
	server.runBoundedAnalysis(context.Background(), []protocol.DocumentURI{"file:///panic-c.pk"}, "panic-recovery-2")

	if len(ws.inFlight) != 0 {
		t.Errorf("expected no leaked in-flight analyses after recovered panics, got %d", len(ws.inFlight))
	}
}

func TestGoBackground_SkipsAfterShutdownBegins(t *testing.T) {
	t.Parallel()

	server := &Server{}
	server.mu.Lock()
	server.shuttingDown = true
	server.mu.Unlock()

	var ran atomic.Bool
	server.goBackground(func(context.Context) { ran.Store(true) })

	if ran.Load() {
		t.Fatal("goBackground ran an operation after shutdown began")
	}
}

func TestGoBackground_NoAddRaceWithTeardownWait(t *testing.T) {
	t.Parallel()

	for range 100 {
		server := &Server{}
		server.mu.Lock()
		server.serverCtx, server.serverCancel = context.WithCancelCause(context.Background())
		server.mu.Unlock()

		var spawners sync.WaitGroup
		spawners.Go(func() {
			for range 50 {
				server.goBackground(func(context.Context) {})
			}
		})

		server.teardownConnection(context.Background())
		spawners.Wait()
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
		RootURI: "file:///old/root", //nolint:staticcheck // tests the RootURI fallback
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
		RootURI:          "file:///root/path", //nolint:staticcheck // tests the RootURI fallback
	}

	uri := server.extractRootURI(params)

	if uri != "file:///root/path" {
		t.Errorf("expected RootURI fallback, got %q", uri)
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

func TestServer_Initialized_RegistersWatchedFilesWhenSupported(t *testing.T) {
	client := &registrationRecordingClient{}
	server := NewServer(ServerDeps{DocCache: NewDocumentCache()})
	server.client = client
	server.initialisationParameters = watchedFilesParameters(true)

	require.NoError(t, server.Initialized(t.Context(), &protocol.InitializedParams{}))
	server.backgroundWg.Wait()

	registrations := client.recorded()
	require.Len(t, registrations, 1)
	require.Len(t, registrations[0].Registrations, 1)

	registration := registrations[0].Registrations[0]
	assert.Equal(t, methodDidChangeWatchedFiles, registration.Method)
	assert.Equal(t, watchedFilesRegistrationID, registration.ID)

	options, ok := registration.RegisterOptions.(protocol.DidChangeWatchedFilesRegistrationOptions)
	require.True(t, ok)
	globs := make([]string, 0, len(options.Watchers))
	for _, watcher := range options.Watchers {
		globs = append(globs, watcher.GlobPattern)
	}
	assert.Equal(t, []string{"**/*.{pk,pkc,go}", "**/go.mod", "**/go.work"}, globs)
}

func TestServer_Initialized_SkipsRegistrationWithoutDynamicSupport(t *testing.T) {
	testCases := []struct {
		parameters *protocol.InitializeParams
		name       string
	}{
		{name: "no initialisation parameters", parameters: nil},
		{name: "no workspace capabilities", parameters: &protocol.InitializeParams{}},
		{name: "dynamic registration disabled", parameters: watchedFilesParameters(false)},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client := &registrationRecordingClient{}
			server := NewServer(ServerDeps{DocCache: NewDocumentCache()})
			server.client = client
			server.initialisationParameters = testCase.parameters

			require.NoError(t, server.Initialized(t.Context(), &protocol.InitializedParams{}))
			server.backgroundWg.Wait()

			assert.Empty(t, client.recorded())
		})
	}
}

func TestServer_Initialized_SkipsRegistrationWithoutClient(t *testing.T) {
	server := NewServer(ServerDeps{DocCache: NewDocumentCache()})
	server.initialisationParameters = watchedFilesParameters(true)

	require.NoError(t, server.Initialized(t.Context(), &protocol.InitializedParams{}))
	server.backgroundWg.Wait()
}

func TestNewServer_ServerVersion(t *testing.T) {
	assert.Equal(t, defaultServerVersion, NewServer(ServerDeps{}).serverVersion)
	assert.Equal(t, "1.2.3", NewServer(ServerDeps{ServerVersion: "1.2.3"}).serverVersion)
}
