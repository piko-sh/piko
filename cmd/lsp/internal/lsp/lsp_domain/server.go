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
	"path/filepath"
	"sync"
	"time"

	protocol "github.com/politepixels/golang-language-server"
	"go.lsp.dev/jsonrpc2"
	"golang.org/x/sync/errgroup"
	"piko.sh/piko/cmd/lsp/internal/lsp/gopls_bridge"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/formatter/formatter_domain"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/typegen/typegen_adapters"
	"piko.sh/piko/internal/typegen/typegen_domain"
	"piko.sh/piko/wdk/clock"
)

const (
	// shutdownGracePeriod is the longest time to wait for background goroutines to finish
	// during server shutdown.
	shutdownGracePeriod = 10 * time.Second

	// maxConcurrentAnalysis limits how many analysis tasks can run at the same time. This
	// applies to bulk operations like DidChangeWatchedFiles and ExecuteCommand.
	maxConcurrentAnalysis = 4
)

// Server holds the state for the language server. It implements the protocol.Server
// interface and manages the workspace which tracks all open documents.
type Server struct {
	// formatter provides document formatting using the FormatterService interface.
	formatter formatter_domain.FormatterService

	// conn is the JSON-RPC connection used to send messages to the client.
	conn jsonrpc2.Conn

	// serverCtx is the root context for all server operations. It is cancelled during
	// Shutdown to signal background goroutines to stop.
	serverCtx context.Context

	// fsReader reads files from the file system for LSP operations.
	fsReader annotator_domain.FSReaderPort

	// client sends messages to the LSP client.
	client protocol.Client

	// coordinator provides core service dependencies set up during bootstrap.
	coordinator coordinator_domain.CoordinatorService

	// resolver looks up module and import paths.
	resolver resolver_domain.ResolverPort

	// clock provides time operations for shutdown timeouts.
	clock clock.Clock

	// analysisSemaphore bounds how many single-document didOpen/didChange/didSave analyses
	// run concurrently, so a burst (e.g. a session restore opening many files) cannot launch
	// unbounded concurrent coordinator builds and overlay syncs.
	analysisSemaphore chan struct{}

	// watchedPending accumulates URIs from didChangeWatchedFiles notifications that still
	// need re-analysis. A checkout/regenerate storm coalesces into this set rather than
	// spawning a background fan-out per notification.
	watchedPending map[protocol.DocumentURI]struct{}

	// initialisationParameters stores the LSP setup parameters sent by the client.
	initialisationParameters *protocol.InitializeParams

	// docCache stores document contents for formatting and save operations.
	docCache *DocumentCache

	// serverCancel stops background goroutines when called during shutdown.
	serverCancel context.CancelCauseFunc

	// workspace manages document tracking and analysis.
	workspace *workspace

	// typeInspectorManager provides Go type lookup and analysis.
	typeInspectorManager *inspector_domain.TypeBuilder

	// goplsManager owns the shared gopls child processes used to answer Go-block requests.
	// It is shared across connections and may be disabled.
	goplsManager *gopls_bridge.Manager

	// goplsUnsubscribe removes this connection's gopls diagnostics subscriber during
	// shutdown. Nil when the bridge is inactive for this connection.
	goplsUnsubscribe func()

	// goplsSatellites caches, per module root, the synthetic satellite-overlay URI set and
	// the VirtualModule generation it was synced for, so the O(components) satellite walk
	// runs only when analysis advances, not on every keystroke. Guarded by mu.
	goplsSatellites map[string]*goplsSatelliteState

	// pathsConfig holds the workspace path settings.
	pathsConfig *config.PathsConfig

	// moduleRoot is this connection's Go module root, captured under mu at Initialize so
	// request goroutines never read the shared, mutable pathsConfig.BaseDir.
	moduleRoot string

	// backgroundWg tracks background goroutines started by the server. Shutdown waits for
	// all tracked goroutines to finish before it returns.
	backgroundWg sync.WaitGroup

	// goplsConnectionID is this connection's overlay-ownership token, vended by the shared
	// Manager so reference-counted overlays know which connection holds them.
	goplsConnectionID uint64

	// goplsShutdownOnce guards per-connection teardown so an abrupt disconnect and an
	// explicit LSP shutdown cannot double-run (and double-unsubscribe) it.
	goplsShutdownOnce sync.Once

	// watchedMu guards watchedPending and watchedRunning.
	watchedMu sync.Mutex

	// mu guards mutable server state during concurrent access.
	mu sync.Mutex

	// formattingEnabled indicates whether formatting features are active.
	formattingEnabled bool

	// goplsBridgeEnabled is the process-level default for the Go-block gopls bridge, used
	// unless a client overrides it via initialisation options.
	goplsBridgeEnabled bool

	// shuttingDown is set under mu once teardownConnection begins, before it waits on
	// backgroundWg.
	shuttingDown bool

	// watchedRunning is true while a single background goroutine is draining watchedPending,
	// so concurrent notifications add to the set instead of starting competing analyses.
	watchedRunning bool
}

// ServerDeps holds the dependencies for creating a new LSP server instance.
type ServerDeps struct {
	// Coordinator runs and controls server tasks.
	Coordinator coordinator_domain.CoordinatorService

	// Resolver provides symbol lookup for the server.
	Resolver resolver_domain.ResolverPort

	// TypeInspectorManager builds type data used for checking documentation.
	TypeInspectorManager *inspector_domain.TypeBuilder

	// DocCache stores parsed documentation to avoid repeated parsing.
	DocCache *DocumentCache

	// FSReader reads files from the file system.
	FSReader annotator_domain.FSReaderPort

	// PathsConfig supplies workspace path settings.
	PathsConfig *config.PathsConfig

	// Formatter provides output formatting for lint results.
	Formatter formatter_domain.FormatterService

	// Clock provides time operations. If nil, defaults to the real system clock.
	Clock clock.Clock

	// GoplsManager owns the shared gopls child processes for Go-block requests. When nil or
	// disabled the server behaves as if the bridge is off.
	GoplsManager *gopls_bridge.Manager

	// FormattingEnabled controls whether formatting features are shown to clients.
	FormattingEnabled bool

	// GoplsBridgeEnabled is the process-level default for the Go-block bridge.
	GoplsBridgeEnabled bool
}

// NewServer creates a new language server with the given dependencies.
//
// Takes deps (ServerDeps) which provides all required dependencies for the server.
//
// Returns *Server which is the configured language server ready for use.
func NewServer(deps ServerDeps) *Server {
	clk := deps.Clock
	if clk == nil {
		clk = clock.RealClock()
	}
	return &Server{
		coordinator:          deps.Coordinator,
		resolver:             deps.Resolver,
		typeInspectorManager: deps.TypeInspectorManager,
		docCache:             deps.DocCache,
		fsReader:             deps.FSReader,
		pathsConfig:          deps.PathsConfig,
		formatter:            deps.Formatter,
		formattingEnabled:    deps.FormattingEnabled,
		goplsManager:         deps.GoplsManager,
		goplsBridgeEnabled:   deps.GoplsBridgeEnabled,
		clock:                clk,
		analysisSemaphore:    make(chan struct{}, maxConcurrentAnalysis),
	}
}

// SetClient assigns the LSP client interface for server-to-client messages.
//
// Takes client (protocol.Client) which handles messages sent from server to client.
//
// Safe for concurrent use. Acquires both the server and workspace mutexes to ensure
// client updates are atomic.
func (s *Server) SetClient(client protocol.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.client = client
	if s.workspace != nil {
		s.workspace.setClient(client)
	}
}

// SetConn assigns the JSON-RPC connection for bidirectional communication.
//
// Takes conn (jsonrpc2.Conn) which is the connection to use for messaging.
//
// Safe for concurrent use. Acquires both the server and workspace mutexes to ensure
// connection updates are atomic.
func (s *Server) SetConn(conn jsonrpc2.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.conn = conn
	if s.workspace != nil {
		s.workspace.setConn(conn)
	}
}

// Initialize initialises the server in response to the first request from the client,
// configuring capabilities and the workspace root.
//
// Takes params (*protocol.InitializeParams) which contains the client's initialisation
// settings and workspace information.
//
// Returns *protocol.InitializeResult which contains the server's capabilities and version
// information.
// Returns error when workspace path configuration fails.
//
// Safe for concurrent use; protected by mutex. Creates a server-scoped context that is
// cancelled during Shutdown to stop background goroutines.
func (s *Server) Initialize(ctx context.Context, params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	ctx, l := logger_domain.From(ctx, log)

	s.releasePriorGoplsSession(ctx)

	s.watchedMu.Lock()
	s.watchedRunning = false
	s.watchedPending = nil
	s.watchedMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.shuttingDown = false
	s.serverCtx, s.serverCancel = context.WithCancelCause(context.Background())

	rootURI := s.extractRootURI(params)
	l.Debug("Setup request received", logger_domain.String("rootURI", string(rootURI)))

	s.initialisationParameters = params
	s.applyGoplsInitOptions(params)

	if err := s.configureWorkspacePaths(ctx, rootURI); err != nil {
		l.Warn("Error configuring workspace paths", logger_domain.Error(err))
	}

	rootPath, _ := uriToPath(rootURI)
	s.moduleRoot = rootPath
	moduleManager := NewModuleContextManager(s.pathsConfig, rootPath, nil)

	s.workspace = newWorkspace(workspaceDeps{
		Coordinator:          s.coordinator,
		TypeInspectorManager: s.typeInspectorManager,
		ModuleManager:        moduleManager,
		Client:               s.client,
		Conn:                 s.conn,
		DocCache:             s.docCache,
	}, rootURI)
	l.Debug("Workspace created")

	if s.goplsManager != nil {
		s.goplsConnectionID = s.goplsManager.NewConnectionID()
		s.goplsUnsubscribe = s.goplsManager.Subscribe(gopls_bridge.Subscriber{
			Diagnostics: s.workspace.handleGoplsDiagnostics,
			ChildDied:   s.workspace.onGoplsChildDeath,
		})
	}

	s.maybeStartGopls(s.serverCtx, rootPath)

	return &protocol.InitializeResult{
		Capabilities: buildServerCapabilities(s.formattingEnabled),
		ServerInfo:   &protocol.ServerInfo{Name: "piko-lsp", Version: "0.0.1"},
	}, nil
}

// Initialized handles the notification that the client has finished its initialisation.
//
// Returns error when the notification cannot be processed.
func (*Server) Initialized(ctx context.Context, _ *protocol.InitializedParams) error {
	_, l := logger_domain.From(ctx, log)

	l.Debug("Client is initialised")
	return nil
}

// SetTrace sets the trace level for logging output.
//
// Takes params (*protocol.SetTraceParams) which specifies the new trace level.
//
// Returns error when the trace level cannot be set.
func (*Server) SetTrace(ctx context.Context, params *protocol.SetTraceParams) error {
	_, l := logger_domain.From(ctx, log)

	l.Debug("SetTrace", logger_domain.Field("value", params.Value))
	return nil
}

// LogTrace handles trace logging messages from the client.
//
// Takes params (*protocol.LogTraceParams) which contains the trace message and verbosity
// level.
//
// Returns error which is always nil as logging does not fail.
func (*Server) LogTrace(ctx context.Context, params *protocol.LogTraceParams) error {
	_, l := logger_domain.From(ctx, log)

	l.Debug("LogTrace", logger_domain.String("message", params.Message), logger_domain.Field("verbose", params.Verbose))
	return nil
}

// WorkDoneProgressCancel handles a request to cancel an in-flight progress operation.
//
// Takes params (*protocol.WorkDoneProgressCancelParams) which identifies the operation to
// cancel.
//
// Returns error when the cancellation fails.
func (*Server) WorkDoneProgressCancel(ctx context.Context, params *protocol.WorkDoneProgressCancelParams) error {
	_, l := logger_domain.From(ctx, log)

	l.Debug("WorkDoneProgressCancel", logger_domain.Field("token", params.Token))
	return nil
}

// Request handles custom or non-standard LSP requests. This includes LSP 3.17 features
// like Inlay Hints that are not in the protocol library's Server interface.
//
// Takes method (string) which specifies the LSP method name to handle.
// Takes params (any) which contains the request parameters.
//
// Returns any which is the method-specific response data.
// Returns error when the method is not found.
func (s *Server) Request(ctx context.Context, method string, params any) (any, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("Non-standard request", logger_domain.String("method", method))

	switch method {
	case "textDocument/inlayHint":
		return s.handleInlayHint(ctx, params)
	case "textDocument/prepareTypeHierarchy":
		return s.handlePrepareTypeHierarchy(ctx, params)
	case "typeHierarchy/supertypes":
		return s.handleTypeHierarchySupertypes(ctx, params)
	case "typeHierarchy/subtypes":
		return s.handleTypeHierarchySubtypes(ctx, params)
	default:
		return nil, jsonrpc2.ErrMethodNotFound
	}
}

// Shutdown stops the server in a controlled way.
//
// Returns error when the shutdown fails.
//
// Cancels the server context to signal all background goroutines to stop, then waits up
// to the shutdown grace period for them to finish. Workspace goroutines are drained via
// workspace.Close so diagnostics-publishing goroutines spawned via context.WithoutCancel
// cannot leak past shutdown.
func (s *Server) Shutdown(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Shutdown request received")

	s.goplsShutdownOnce.Do(func() {
		s.teardownConnection(ctx)
	})
	return nil
}

// Exit handles the exit notification and stops the server.
//
// Returns error when the shutdown cannot be completed.
func (*Server) Exit(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)

	l.Debug("Exit notification received")
	return nil
}

// runDocumentAnalysisBounded analyses one document generation on the tracked background
// executor.
//
// Takes uri (protocol.DocumentURI) which identifies the document to analyse.
// Takes operation (string) which names the originating request for logging.
// Takes gen (uint64) which is the content generation this analysis targets.
// Takes after (func) which, when non-nil, runs with the freshly analysed document.
func (s *Server) runDocumentAnalysisBounded(uri protocol.DocumentURI, operation string, gen uint64, after func(ctx context.Context, document *document)) {
	s.goBackground(func(ctx context.Context) {
		ctx, l := logger_domain.From(ctx, log)

		if s.workspace.currentGeneration(uri) > gen {
			return
		}

		s.workspace.supersede(uri, gen)

		document, acquired, err := s.analyseWithinSlot(ctx, uri, gen)
		if !acquired {
			return
		}

		if err != nil {
			l.Trace("Analysis did not complete (superseded or cancelled, expected during rapid edits)",
				logger_domain.String(keyOperation, operation),
				logger_domain.String(keyURI, uri.Filename()),
				logger_domain.Error(err))
		} else if after != nil {
			after(ctx, document)
		}

		if pending := s.workspace.settleGeneration(uri, gen); pending != 0 {
			s.runDocumentAnalysisBounded(uri, operation+"(re-arm)", pending, after)
		}
	})
}

// analyseWithinSlot runs the piko analysis for a generation while holding one analysis
// slot.
//
// Takes uri (protocol.DocumentURI) which identifies the document to analyse.
// Takes gen (uint64) which is the content generation this analysis targets.
//
// Returns *document which is the freshly analysed document.
// Returns bool which is false when the context was cancelled before a slot was acquired.
// Returns error when the analysis itself failed.
func (s *Server) analyseWithinSlot(ctx context.Context, uri protocol.DocumentURI, gen uint64) (*document, bool, error) {
	select {
	case s.analysisSemaphore <- struct{}{}:
		defer func() { <-s.analysisSemaphore }()
	case <-ctx.Done():
		return nil, false, ctx.Err()
	}

	document, err := s.workspace.runAnalysisForGeneration(ctx, uri, gen)
	return document, true, err
}

// releasePriorGoplsSession tears down any prior session before a re-Initialize.
//
// Concurrency: takes s.mu only to snapshot and re-arm the shutdown guard, releasing it
// before performing gopls I/O.
func (s *Server) releasePriorGoplsSession(ctx context.Context) {
	s.mu.Lock()
	cancel := s.serverCancel
	unsubscribe := s.goplsUnsubscribe
	hadPriorSession := s.serverCancel != nil || s.workspace != nil
	s.goplsUnsubscribe = nil
	s.mu.Unlock()

	if !hadPriorSession {
		return
	}
	if cancel != nil {
		cancel(errors.New("LSP server re-initialised"))
	}
	if unsubscribe != nil {
		unsubscribe()
	}
	s.releaseGoplsOverlays(ctx)

	s.mu.Lock()
	s.goplsShutdownOnce = sync.Once{}
	s.mu.Unlock()
}

// teardownConnection cancels the server context, unsubscribes from gopls diagnostics,
// releases this connection's gopls overlays against the shared child, and drains
// background goroutines.
//
// Concurrency: takes s.mu to set the shutdown guard and snapshot state, then waits on
// backgroundWg via a helper goroutine bounded by the shutdown grace period.
func (s *Server) teardownConnection(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	s.mu.Lock()
	s.shuttingDown = true
	if s.serverCancel != nil {
		l.Trace("Cancelling server context")
		s.serverCancel(errors.New("LSP server shutting down"))
	}
	workspace := s.workspace
	unsubscribe := s.goplsUnsubscribe
	s.goplsUnsubscribe = nil
	s.mu.Unlock()

	if unsubscribe != nil {
		unsubscribe()
	}
	s.releaseGoplsOverlays(ctx)

	done := make(chan struct{})
	go func() {
		defer goroutine.RecoverPanic(context.WithoutCancel(ctx), "lsp.shutdownWait")
		s.backgroundWg.Wait()
		close(done)
	}()

	clk := s.clock
	if clk == nil {
		clk = clock.RealClock()
	}

	select {
	case <-done:
		l.Trace("All background goroutines stopped cleanly")
	case <-clk.NewTimer(shutdownGracePeriod).C():
		l.Warn("Shutdown timed out waiting for background goroutines",
			logger_domain.String("gracePeriod", shutdownGracePeriod.String()))
	}

	if workspace != nil {
		if err := workspace.Close(context.WithoutCancel(ctx)); err != nil {
			l.Warn("Workspace drain timed out during shutdown", logger_domain.Error(err))
		}
	}

	l.Trace("LSP server shutdown complete")
}

// maybeStartGopls warms a gopls child for the workspace root in the background when the
// Go-block bridge is enabled, so Go features are ready by the time the user navigates
// into a script block. It is a no-op when the bridge is off.
//
// Takes moduleRoot (string) which is the Go module root to warm a gopls child for.
//
// Must be called while holding s.mu (it reads serverCtx and tracks the spawn in
// backgroundWg without re-locking).
func (s *Server) maybeStartGopls(serverCtx context.Context, moduleRoot string) {
	if s.shuttingDown || moduleRoot == "" || !s.goplsBridgeEnabled || s.goplsManager == nil || !s.goplsManager.Enabled() {
		return
	}

	manager := s.goplsManager
	s.backgroundWg.Go(func() {
		defer goroutine.RecoverPanic(serverCtx, "lsp.maybeStartGopls")
		if _, err := manager.Acquire(serverCtx, moduleRoot); err != nil {
			return
		}
	})
}

// goBackground spawns a tracked background task that respects server shutdown, adding it
// to the WaitGroup so Shutdown waits for it to finish.
//
// Takes operation (func(context.Context)) which is the function to run. The function
// receives the server context, which will be cancelled during Shutdown.
//
// Concurrent use is safe. Spawns a goroutine that runs operation with the server context.
// The goroutine is tracked by backgroundWg so Shutdown waits for it to finish.
func (s *Server) goBackground(operation func(context.Context)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.shuttingDown {
		return
	}

	ctx := s.serverCtx
	if ctx == nil {
		ctx = context.Background()
	}

	s.backgroundWg.Go(func() {
		defer goroutine.RecoverPanic(ctx, "lsp.goBackground")
		operation(ctx)
	})
}

// runBoundedAnalysis runs analysis on multiple URIs with limited concurrency. This stops
// resource exhaustion when many files change at once.
//
// Takes uris ([]protocol.DocumentURI) which specifies the files to analyse.
// Takes operationName (string) which describes the operation for logging.
func (s *Server) runBoundedAnalysis(ctx context.Context, uris []protocol.DocumentURI, operationName string) {
	ctx, l := logger_domain.From(ctx, log)

	if len(uris) == 0 {
		return
	}

	l.Debug("Starting bounded analysis",
		logger_domain.String(keyOperation, operationName),
		logger_domain.Int("uris", len(uris)))

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrentAnalysis)

	for _, uri := range uris {
		g.Go(func() error {
			defer goroutine.RecoverPanic(gCtx, "lsp.runBoundedAnalysis.worker")
			if gCtx.Err() != nil {
				return gCtx.Err()
			}
			if _, err := s.workspace.RunAnalysisForURI(gCtx, uri); err != nil {
				l.Error("Failed to run analysis",
					logger_domain.Error(err),
					logger_domain.String(keyOperation, operationName),
					logger_domain.String(keyURI, uri.Filename()))
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		l.Debug("Bounded analysis stopped",
			logger_domain.Error(err),
			logger_domain.String(keyOperation, operationName))
	} else {
		l.Debug("Bounded analysis completed",
			logger_domain.String(keyOperation, operationName),
			logger_domain.Int("uris", len(uris)))
	}
}

// scheduleWatchedAnalysis coalesces watched-file re-analysis into a single drainer.
//
// It adds uris to a pending set and starts a single background drainer if one is not
// already running. A checkout or regenerate storm then produces one bounded analysis loop
// (re-running while fresh changes accumulate) instead of a background fan-out per
// notification, each with its own pool.
//
// Takes uris ([]protocol.DocumentURI) which are the watched files needing re-analysis.
//
// Safe for concurrent use; guarded by watchedMu.
func (s *Server) scheduleWatchedAnalysis(uris []protocol.DocumentURI) {
	if len(uris) == 0 {
		return
	}

	s.watchedMu.Lock()
	if s.watchedPending == nil {
		s.watchedPending = make(map[protocol.DocumentURI]struct{}, len(uris))
	}
	for _, uri := range uris {
		s.watchedPending[uri] = struct{}{}
	}
	if s.watchedRunning {
		s.watchedMu.Unlock()
		return
	}
	s.watchedRunning = true
	s.watchedMu.Unlock()

	s.goBackground(func(ctx context.Context) {
		for {
			s.watchedMu.Lock()
			if len(s.watchedPending) == 0 {
				s.watchedRunning = false
				s.watchedMu.Unlock()
				return
			}
			batch := make([]protocol.DocumentURI, 0, len(s.watchedPending))
			for uri := range s.watchedPending {
				batch = append(batch, uri)
			}
			clear(s.watchedPending)
			s.watchedMu.Unlock()

			s.runBoundedAnalysis(ctx, batch, "watched file change")
		}
	})
}

// openURIs filters uris to those the editor currently has open, so watched-file
// re-analysis never commits documents (and publishes diagnostics) for files the editor
// never opened and no DidClose will ever remove.
//
// Takes uris ([]protocol.DocumentURI) which are the candidate files to filter.
//
// Returns []protocol.DocumentURI which contains only the URIs the editor currently has
// open.
func (s *Server) openURIs(uris []protocol.DocumentURI) []protocol.DocumentURI {
	open := make([]protocol.DocumentURI, 0, len(uris))
	for _, uri := range uris {
		if _, isOpen := s.docCache.Get(uri); isOpen {
			open = append(open, uri)
		}
	}
	return open
}

// extractRootURI gets the root URI from the given parameters. It prefers WorkspaceFolders
// over the older RootURI field.
//
// Takes params (*protocol.InitializeParams) which holds the workspace folders and root
// URI from the client.
//
// Returns protocol.DocumentURI which is the root URI of the workspace.
func (*Server) extractRootURI(params *protocol.InitializeParams) protocol.DocumentURI {
	if len(params.WorkspaceFolders) > 0 {
		return protocol.DocumentURI(params.WorkspaceFolders[0].URI)
	}
	return params.RootURI //nolint:staticcheck // fallback for older clients
}

// configureWorkspacePaths sets up the configuration with the workspace root path.
//
// Takes rootURI (protocol.DocumentURI) which specifies the workspace root as a URI to be
// converted to a file system path.
//
// Returns error when the URI cannot be converted to a valid path.
func (s *Server) configureWorkspacePaths(ctx context.Context, rootURI protocol.DocumentURI) error {
	ctx, l := logger_domain.From(ctx, log)

	rootPath, err := uriToPath(rootURI)
	if err != nil {
		return fmt.Errorf("converting root URI to path: %w", err)
	}

	l.Debug("Workspace root path", logger_domain.String("path", rootPath))

	s.pathsConfig.BaseDir = &rootPath
	s.pathsConfig.ComponentsSourceDir = new("components")
	s.pathsConfig.PagesSourceDir = new("pages")
	s.pathsConfig.PartialsSourceDir = new("partials")
	s.pathsConfig.AssetsSourceDir = new("lib")

	s.ensureTypeDefinitionsIfNotExists(ctx, rootPath)

	l.Debug("Configuration updated with workspace root")
	return nil
}

// ensureTypeDefinitionsIfNotExists writes TypeScript type definitions to dist/ts/ only if
// they do not already exist. This provides a way to set up IDE support before the dev
// server has been run.
//
// The dev server writes type definitions when it starts, so this only fills in the gap
// when the LSP is opened before the dev server.
//
// Takes rootPath (string) which is the project root directory containing the dist folder.
//
// Errors are logged but do not stop LSP startup.
func (*Server) ensureTypeDefinitionsIfNotExists(ctx context.Context, rootPath string) {
	_, l := logger_domain.From(ctx, log)

	typegenService := typegen_adapters.NewTypeDefinitionService()
	distTsDir := filepath.Join(rootPath, "dist", "ts")

	opts := typegen_domain.EnsureOptions{
		OnlyIfNotExists: true,
	}

	if err := typegenService.EnsureTypeDefinitionsWithOptions(ctx, distTsDir, opts); err != nil {
		l.Warn("Failed to write TypeScript type definitions",
			logger_domain.Error(err),
			logger_domain.String("dist_ts_dir", distTsDir),
		)
		return
	}

	l.Internal("TypeScript type definitions ensured for IDE (only if not present)")
}

// buildServerCapabilities creates the server capabilities for the LSP response.
//
// Takes formattingEnabled (bool) which controls whether formatting features are shown to
// the client.
//
// Returns protocol.ServerCapabilities which lists the features this server supports,
// including text sync, hover, completion, and optionally formatting.
func buildServerCapabilities(formattingEnabled bool) protocol.ServerCapabilities {
	return protocol.ServerCapabilities{
		TextDocumentSync: protocol.TextDocumentSyncOptions{
			OpenClose:         true,
			Change:            protocol.TextDocumentSyncKindFull,
			Save:              &protocol.SaveOptions{IncludeText: true},
			WillSave:          true,
			WillSaveWaitUntil: true,
		},
		HoverProvider:                    true,
		DefinitionProvider:               true,
		DocumentSymbolProvider:           true,
		DocumentFormattingProvider:       formattingEnabled,
		DocumentRangeFormattingProvider:  formattingEnabled,
		DocumentOnTypeFormattingProvider: buildOnTypeFormattingOptions(formattingEnabled),
		FoldingRangeProvider:             true,
		LinkedEditingRangeProvider:       true,
		CompletionProvider:               &protocol.CompletionOptions{TriggerCharacters: []string{".", "-", "\""}, ResolveProvider: false},
		SignatureHelpProvider:            &protocol.SignatureHelpOptions{TriggerCharacters: []string{"(", ","}, RetriggerCharacters: []string{","}},
		CodeActionProvider:               true,
		RenameProvider:                   &protocol.RenameOptions{PrepareProvider: true},
		MonikerProvider:                  true,
		ImplementationProvider:           true,
	}
}

// buildOnTypeFormattingOptions creates the on-type formatting options.
//
// Takes enabled (bool) which controls whether on-type formatting is available.
//
// Returns *protocol.DocumentOnTypeFormattingOptions which lists the characters that start
// formatting as the user types, or nil if disabled.
func buildOnTypeFormattingOptions(enabled bool) *protocol.DocumentOnTypeFormattingOptions {
	if !enabled {
		return nil
	}
	return &protocol.DocumentOnTypeFormattingOptions{
		FirstTriggerCharacter: "\n",
		MoreTriggerCharacter:  []string{">", "}"},
	}
}
