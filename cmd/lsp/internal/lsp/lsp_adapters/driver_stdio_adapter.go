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
	"io"
	"log/slog"
	"os"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"piko.sh/piko/cmd/lsp/internal/lsp/lsp_domain"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/formatter/formatter_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/wdk/safedisk"
)

// lspLogFilePermissions is the file permission for LSP log files.
const lspLogFilePermissions = 0660

// stdioAdapter is a driving adapter for the LSP hexagon.
//
// It implements lsp_domain.LSPServerPort to connect the core LSP domain logic
// to an external communication channel provided as an io.ReadWriteCloser
// (typically stdin/stdout).
//
// It is also the composition root. It receives pre-built dependencies and uses
// them to instantiate the lsp_domain.Server. It then drives the domain by
// connecting it to the JSON-RPC stream.
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

	// sandboxFactory creates sandboxes for filesystem access. When nil,
	// a no-op sandbox is used as a fallback.
	sandboxFactory safedisk.Factory

	// formattingEnabled controls whether formatting features are shown to clients.
	formattingEnabled bool
}

var _ lsp_domain.LSPServerPort = (*stdioAdapter)(nil)

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

// Run implements the LSPServerPort interface. It sets up the JSON-RPC
// communication over the provided stream and starts the language server,
// blocking until the session is complete.
//
// Takes stream (io.ReadWriteCloser) which provides the bidirectional
// communication channel for JSON-RPC messages.
//
// Returns error when the connection closes with an error condition.
func (a *stdioAdapter) Run(ctx context.Context, stream io.ReadWriteCloser) error {
	ctx, l := logger_domain.From(ctx, log)

	logRes := setupLogFile(nil, a.sandboxFactory)
	defer logRes.close()

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
		FormattingEnabled:    a.formattingEnabled,
	})

	_, conn, client := protocol.NewServer(ctx, pikoServer, loggingStream, slog.Default())
	pikoServer.SetClient(client)
	pikoServer.SetConn(conn)

	l.Debug("Piko LSP server is running over stdio. Waiting for connection to close.")
	<-conn.Done()

	return conn.Err()
}

// NewStdioAdapter is the factory function for creating the stdio driving
// adapter. It requires all the dependencies that the LSP server needs.
//
// Takes coordinatorService (coordinator_domain.CoordinatorService) which
// provides coordination between build and annotation operations.
// Takes resolver (resolver_domain.ResolverPort) which resolves module paths.
// Takes typeInspectorManager (*inspector_domain.TypeBuilder) which inspects
// Go types for documentation analysis.
// Takes docCache (*lsp_domain.DocumentCache) which caches parsed documents.
// Takes lspReader (annotator_domain.FSReaderPort) which reads files from the
// filesystem.
// Takes pathsConfig (*config.PathsConfig) which supplies workspace path settings.
// Takes formattingEnabled (bool) which controls whether formatting is applied.
//
// Returns lsp_domain.LSPServerPort which is the configured LSP server adapter
// ready to handle stdio communication.
//
// Panics if any required dependency is nil.
func NewStdioAdapter(
	coordinatorService coordinator_domain.CoordinatorService,
	resolver resolver_domain.ResolverPort,
	typeInspectorManager *inspector_domain.TypeBuilder,
	docCache *lsp_domain.DocumentCache,
	lspReader annotator_domain.FSReaderPort,
	pathsConfig *config.PathsConfig,
	formattingEnabled bool,
) lsp_domain.LSPServerPort {
	if coordinatorService == nil {
		panic("NewStdioAdapter: coordinatorService cannot be nil")
	}
	if resolver == nil {
		panic("NewStdioAdapter: resolver cannot be nil")
	}
	if typeInspectorManager == nil {
		panic("NewStdioAdapter: typeInspectorManager cannot be nil")
	}
	if docCache == nil {
		panic("NewStdioAdapter: docCache cannot be nil")
	}
	if lspReader == nil {
		panic("NewStdioAdapter: lspReader cannot be nil")
	}
	if pathsConfig == nil {
		panic("NewStdioAdapter: pathsConfig cannot be nil")
	}
	return &stdioAdapter{
		coordinatorService:   coordinatorService,
		resolver:             resolver,
		typeInspectorManager: typeInspectorManager,
		docCache:             docCache,
		lspReader:            lspReader,
		pathsConfig:          pathsConfig,
		formattingEnabled:    formattingEnabled,
	}
}

// setupLogFile creates the log file and sandbox for LSP protocol logging.
//
// Takes injectedSandbox (safedisk.Sandbox) which is an optional sandbox for
// testing. When nil, a sandbox is created for the temp directory.
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access. When nil, a no-op sandbox is used as a fallback.
//
// Returns *lspLogResources which holds the log file and sandbox handles,
// or nil if the log file could not be created. The LSP server will continue
// without protocol logging in that case.
func setupLogFile(injectedSandbox safedisk.Sandbox, factory safedisk.Factory) *lspLogResources {
	_, l := logger_domain.From(context.Background(), log)

	const logFileName = "piko-lsp.log"

	sandbox := injectedSandbox
	sandboxOwned := false
	if sandbox == nil {
		var err error
		if factory != nil {
			sandbox, err = factory.Create("lsp-log", os.TempDir(), safedisk.ModeReadWrite)
		} else {
			sandbox, err = safedisk.NewNoOpSandbox(os.TempDir(), safedisk.ModeReadWrite)
		}
		if err != nil {
			l.Warn("LSP protocol logging disabled: could not create sandbox for temp directory",
				logger_domain.Error(err))
			return nil
		}
		sandboxOwned = true
	}

	f, err := sandbox.OpenFile(logFileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, lspLogFilePermissions)
	if err != nil {
		if sandboxOwned {
			_ = sandbox.Close()
		}
		l.Warn("LSP protocol logging disabled: could not open log file",
			logger_domain.Error(err))
		return nil
	}

	var resSandbox safedisk.Sandbox
	if sandboxOwned {
		resSandbox = sandbox
	}
	return &lspLogResources{logFile: f, sandbox: resSandbox}
}
