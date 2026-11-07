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

package lsp_adapters

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	protocol "github.com/politepixels/golang-language-server"
	"go.lsp.dev/jsonrpc2"
	"piko.sh/piko/cmd/pikopls/internal/lsp/gopls_bridge"
	"piko.sh/piko/cmd/pikopls/internal/lsp/lsp_domain"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/formatter/formatter_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/safedisk"
)

const (
	// lspLogFilePermissions is the file permission for LSP log files.
	lspLogFilePermissions = 0660
)

// stdioAdapter is a driving adapter for the LSP hexagon.
//
// It implements lsp_domain.LSPServerPort to connect the core LSP domain logic to an
// external communication channel provided as an io.ReadWriteCloser (typically
// stdin/stdout).
//
// It is also the composition root. It receives pre-built dependencies and uses them to
// instantiate the lsp_domain.Server. It then drives the domain by connecting it to the
// JSON-RPC stream.
type stdioAdapter struct {
	// coordinatorService handles document analysis across language services.
	coordinatorService coordinator_domain.CoordinatorService

	// resolver provides name lookup for cross-reference searches.
	resolver resolver_domain.ResolverPort

	// typeInspectorManager builds type information for LSP operations.
	typeInspectorManager *inspector_domain.TypeBuilder

	// docCache stores parsed document data for LSP operations.
	docCache *lsp_domain.DocumentCache

	// lspReader reads files from the file system for the LSP server.
	lspReader annotator_domain.FSReaderPort

	// pathsConfig supplies workspace path settings for the LSP server.
	pathsConfig *config.PathsConfig

	// sandboxFactory creates sandboxes for filesystem access. When nil, a sandbox scoped to
	// the protocol log directory is used as a fallback.
	sandboxFactory safedisk.Factory

	// protocolLogPath is the file that receives the JSON-RPC protocol log; empty disables
	// protocol logging.
	protocolLogPath string

	// goplsManager owns the shared gopls child processes for Go-block requests.
	goplsManager *gopls_bridge.Manager

	// serverVersion is the version reported to clients in serverInfo.
	serverVersion string

	// formattingEnabled controls whether formatting features are shown to clients.
	formattingEnabled bool

	// goplsBridgeEnabled is the process-level default for the Go-block bridge.
	goplsBridgeEnabled bool
}

var (
	_ lsp_domain.LSPServerPort = (*stdioAdapter)(nil)
)

// lspLogResources holds the log file and sandbox used by the LSP server.
type lspLogResources struct {
	// logFile is the file handle for writing LSP protocol debug logs.
	logFile safedisk.FileHandle

	// sandbox is the isolated file system for LSP logging; nil means no sandbox.
	sandbox safedisk.Sandbox
}

// close releases all log resources held by the LSP logger.
func (r *lspLogResources) close() {
	if r == nil {
		return
	}
	if r.logFile != nil {
		_ = r.logFile.Close()
	}
	if r.sandbox != nil {
		_ = r.sandbox.Close()
	}
}

// Run implements the LSPServerPort interface. It sets up the JSON-RPC communication over
// the provided stream and starts the language server, blocking until the session is
// complete.
//
// Takes stream (io.ReadWriteCloser) which provides the bidirectional communication
// channel for JSON-RPC messages.
//
// Returns error when the connection closes with an error condition.
func (a *stdioAdapter) Run(ctx context.Context, stream io.ReadWriteCloser) error {
	ctx, l := logger_domain.From(ctx, log)

	logRes := setupLogFile(ctx, a.protocolLogPath, nil, a.sandboxFactory)
	defer logRes.close()

	if a.goplsManager != nil {
		defer func() { _ = a.goplsManager.Close(context.WithoutCancel(ctx)) }()
	}

	rpcStream := jsonrpc2.NewStream(stream)
	loggingStream := rpcStream
	if logRes != nil && logRes.logFile != nil {
		loggingStream = protocol.LoggingStream(rpcStream, logRes.logFile)
	}

	formatter := formatter_domain.NewFormatterService()
	pikoServer := lsp_domain.NewServer(lsp_domain.ServerDeps{
		Coordinator:          a.coordinatorService,
		Resolver:             a.resolver,
		TypeInspectorManager: a.typeInspectorManager,
		DocCache:             a.docCache,
		FSReader:             a.lspReader,
		PathsConfig:          a.pathsConfig,
		Formatter:            formatter,
		GoplsManager:         a.goplsManager,
		FormattingEnabled:    a.formattingEnabled,
		GoplsBridgeEnabled:   a.goplsBridgeEnabled,
		ServerVersion:        a.serverVersion,
		Clock:                nil,
	})

	conn, client := serveProtocol(ctx, pikoServer, loggingStream, slog.Default())
	pikoServer.SetClient(client)
	pikoServer.SetConn(conn)

	l.Debug("Piko LSP server is running over stdio. Waiting for connection to close.")
	<-conn.Done()

	return conn.Err()
}

// StdioAdapterDeps holds the dependencies for creating a stdio driving adapter.
type StdioAdapterDeps struct {
	// CoordinatorService coordinates build and annotation operations.
	CoordinatorService coordinator_domain.CoordinatorService

	// Resolver resolves module and import paths.
	Resolver resolver_domain.ResolverPort

	// TypeInspectorManager builds Go type information for documentation analysis.
	TypeInspectorManager *inspector_domain.TypeBuilder

	// DocCache caches parsed documents.
	DocCache *lsp_domain.DocumentCache

	// LSPReader reads files from the filesystem.
	LSPReader annotator_domain.FSReaderPort

	// PathsConfig supplies workspace path settings.
	PathsConfig *config.PathsConfig

	// GoplsManager owns the shared gopls child processes for Go-block requests.
	GoplsManager *gopls_bridge.Manager

	// ServerVersion is the version reported to clients in serverInfo. Empty falls back to
	// the domain default.
	ServerVersion string

	// ProtocolLogPath is the file that receives the JSON-RPC protocol log. Empty disables
	// protocol logging, which is the default so nothing is written unless file logging is
	// enabled.
	ProtocolLogPath string

	// FormattingEnabled controls whether formatting is applied.
	FormattingEnabled bool

	// GoplsBridgeEnabled is the process-level default for the Go-block bridge.
	GoplsBridgeEnabled bool
}

// NewStdioAdapter is the factory function for creating the stdio driving adapter.
//
// Takes deps (StdioAdapterDeps) which provides all dependencies the LSP server needs.
//
// Returns lsp_domain.LSPServerPort which is the configured LSP server adapter ready to
// handle stdio communication.
// Returns error when any required dependency is nil.
func NewStdioAdapter(deps StdioAdapterDeps) (lsp_domain.LSPServerPort, error) {
	switch {
	case deps.CoordinatorService == nil:
		return nil, errors.New("NewStdioAdapter: coordinatorService cannot be nil")
	case deps.Resolver == nil:
		return nil, errors.New("NewStdioAdapter: resolver cannot be nil")
	case deps.TypeInspectorManager == nil:
		return nil, errors.New("NewStdioAdapter: typeInspectorManager cannot be nil")
	case deps.DocCache == nil:
		return nil, errors.New("NewStdioAdapter: docCache cannot be nil")
	case deps.LSPReader == nil:
		return nil, errors.New("NewStdioAdapter: lspReader cannot be nil")
	case deps.PathsConfig == nil:
		return nil, errors.New("NewStdioAdapter: pathsConfig cannot be nil")
	}
	return &stdioAdapter{
		coordinatorService:   deps.CoordinatorService,
		resolver:             deps.Resolver,
		typeInspectorManager: deps.TypeInspectorManager,
		docCache:             deps.DocCache,
		lspReader:            deps.LSPReader,
		pathsConfig:          deps.PathsConfig,
		goplsManager:         deps.GoplsManager,
		formattingEnabled:    deps.FormattingEnabled,
		goplsBridgeEnabled:   deps.GoplsBridgeEnabled,
		serverVersion:        deps.ServerVersion,
		sandboxFactory:       nil,
		protocolLogPath:      deps.ProtocolLogPath,
	}, nil
}

// setupLogFile creates the log file and sandbox for LSP protocol logging.
//
// Takes logPath (string) which is the protocol log file path. When empty, protocol logging
// is disabled and nil is returned without touching the filesystem.
// Takes injectedSandbox (safedisk.Sandbox) which is an optional sandbox for testing. When
// nil, a sandbox is created for the directory containing logPath.
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem access. When
// nil, a real sandbox scoped to the log directory is created directly, so the protocol
// log writer is still confined and never falls back to unsandboxed access.
//
// Returns *lspLogResources which holds the log file and sandbox handles, or nil if protocol
// logging is disabled or the log file could not be created. The LSP server will continue
// without protocol logging in that case.
func setupLogFile(ctx context.Context, logPath string, injectedSandbox safedisk.Sandbox, factory safedisk.Factory) *lspLogResources {
	if logPath == "" {
		return nil
	}

	_, l := logger_domain.From(ctx, log)

	logDirectory := filepath.Dir(logPath)

	sandbox := injectedSandbox
	sandboxOwned := false
	if sandbox == nil {
		created, err := openLogSandbox(logDirectory, factory)
		if err != nil {
			l.Warn("LSP protocol logging disabled: could not create sandbox for log directory",
				logger_domain.String("directory", logDirectory),
				logger_domain.Error(err))
			return nil
		}
		sandbox = created
		sandboxOwned = true
	}

	logFile, err := sandbox.OpenFile(filepath.Base(logPath), os.O_CREATE|os.O_RDWR|os.O_APPEND, lspLogFilePermissions)
	if err != nil {
		if sandboxOwned {
			err = errors.Join(err, sandbox.Close())
		}
		l.Warn("LSP protocol logging disabled: could not open log file",
			logger_domain.String("path", logPath),
			logger_domain.Error(err))
		return nil
	}

	if !sandboxOwned {
		return &lspLogResources{logFile: logFile, sandbox: nil}
	}
	return &lspLogResources{logFile: logFile, sandbox: sandbox}
}

// openLogSandbox opens a read-write sandbox rooted at the protocol log directory.
//
// Takes logDirectory (string) which is the directory that holds the protocol log.
// Takes factory (safedisk.Factory) which creates the sandbox, or nil to create a real
// sandbox directly.
//
// Returns safedisk.Sandbox which is scoped to logDirectory.
// Returns error when the sandbox cannot be created.
func openLogSandbox(logDirectory string, factory safedisk.Factory) (safedisk.Sandbox, error) {
	if factory != nil {
		sandbox, err := factory.Create("lsp-log", logDirectory, safedisk.ModeReadWrite)
		if err != nil {
			return nil, fmt.Errorf("creating log sandbox from factory: %w", err)
		}
		return sandbox, nil
	}
	sandbox, err := safedisk.NewSandbox(logDirectory, safedisk.ModeReadWrite)
	if err != nil {
		return nil, fmt.Errorf("creating log sandbox: %w", err)
	}
	return sandbox, nil
}
