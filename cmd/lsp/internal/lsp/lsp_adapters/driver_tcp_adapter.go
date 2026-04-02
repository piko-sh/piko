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
	"fmt"
	"io"
	"log/slog"
	"net"

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
)

// tcpAdapter listens on a TCP socket and creates a new LSP server for each
// client connection. It implements LSPServerPort.
type tcpAdapter struct {
	// coordinatorService handles LSP coordination for connections.
	coordinatorService coordinator_domain.CoordinatorService

	// resolver finds symbol definitions for LSP operations.
	resolver resolver_domain.ResolverPort

	// lspReader provides file system access for the LSP server.
	lspReader annotator_domain.FSReaderPort

	// typeInspectorManager builds type data for LSP requests.
	typeInspectorManager *inspector_domain.TypeBuilder

	// docCache stores parsed documentation for files to avoid repeated parsing.
	docCache *lsp_domain.DocumentCache

	// pathsConfig provides workspace path settings for document processing.
	pathsConfig *config.PathsConfig

	// addr is the TCP address to listen on (e.g. "localhost:8080").
	addr string

	// formattingEnabled controls whether formatting capabilities are advertised.
	formattingEnabled bool
}

var _ lsp_domain.LSPServerPort = (*tcpAdapter)(nil)

// TCPAdapterDeps holds the dependencies for creating a TCP adapter.
type TCPAdapterDeps struct {
	// CoordinatorService handles LSP coordination for connections.
	CoordinatorService coordinator_domain.CoordinatorService

	// Resolver finds symbol definitions for LSP operations.
	Resolver resolver_domain.ResolverPort

	// LSPReader provides file system access for the LSP server.
	LSPReader annotator_domain.FSReaderPort

	// TypeInspectorManager builds type data for LSP requests.
	TypeInspectorManager *inspector_domain.TypeBuilder

	// DocCache stores parsed documentation for files.
	DocCache *lsp_domain.DocumentCache

	// PathsConfig provides workspace path settings for document processing.
	PathsConfig *config.PathsConfig

	// Addr is the TCP address to listen on (e.g. "localhost:8080").
	Addr string

	// FormattingEnabled controls whether formatting capabilities are advertised.
	FormattingEnabled bool
}

// Run starts the TCP server and accepts client connections in a loop.
// The stream parameter is ignored as TCP creates its own connections.
//
// Returns error when the TCP listener cannot be created.
//
// Spawns a new goroutine for each accepted connection. These goroutines
// run until the connection is closed or the context is cancelled.
func (a *tcpAdapter) Run(ctx context.Context, _ io.ReadWriteCloser) error {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("Starting Piko LSP TCP server", logger_domain.String("address", a.addr))
	listener, err := net.Listen("tcp", a.addr)
	if err != nil {
		return fmt.Errorf("listening on TCP address %s: %w", a.addr, err)
	}
	defer func() { _ = listener.Close() }()

	for {
		conn, err := listener.Accept()
		if err != nil {
			l.Debug("Error accepting connection", logger_domain.Error(err))
			continue
		}

		go a.handleConnection(ctx, conn)
	}
}

// handleConnection manages a single client connection over TCP.
//
// Takes conn (net.Conn) which is the TCP connection to handle.
func (a *tcpAdapter) handleConnection(ctx context.Context, conn net.Conn) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("Client connected", logger_domain.String("remoteAddr", conn.RemoteAddr().String()))
	defer func() { _ = conn.Close() }()

	stream := jsonrpc2.NewStream(conn)
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

	_, jsonrpcConn, client := protocol.NewServer(ctx, pikoServer, stream, slog.Default())
	pikoServer.SetClient(client)
	pikoServer.SetConn(jsonrpcConn)

	l.Debug("Handling connection. Waiting for it to close.")
	<-jsonrpcConn.Done()
	l.Debug("Client disconnected", logger_domain.String("remoteAddr", conn.RemoteAddr().String()))
}

// NewTCPAdapter creates a new TCP adapter for the LSP server.
//
// Takes deps (TCPAdapterDeps) which provides all dependencies the LSP server
// needs.
//
// Returns lsp_domain.LSPServerPort which is the configured adapter ready for
// use.
//
// Panics if any required dependency field in deps is nil, including
// CoordinatorService, Resolver, TypeInspectorManager, DocCache, LSPReader, or
// PathsConfig.
func NewTCPAdapter(deps TCPAdapterDeps) lsp_domain.LSPServerPort {
	if deps.CoordinatorService == nil {
		panic("NewTCPAdapter: coordinatorService cannot be nil")
	}
	if deps.Resolver == nil {
		panic("NewTCPAdapter: resolver cannot be nil")
	}
	if deps.TypeInspectorManager == nil {
		panic("NewTCPAdapter: typeInspectorManager cannot be nil")
	}
	if deps.DocCache == nil {
		panic("NewTCPAdapter: docCache cannot be nil")
	}
	if deps.LSPReader == nil {
		panic("NewTCPAdapter: lspReader cannot be nil")
	}
	if deps.PathsConfig == nil {
		panic("NewTCPAdapter: pathsConfig cannot be nil")
	}
	return &tcpAdapter{
		addr:                 deps.Addr,
		coordinatorService:   deps.CoordinatorService,
		resolver:             deps.Resolver,
		typeInspectorManager: deps.TypeInspectorManager,
		docCache:             deps.DocCache,
		lspReader:            deps.LSPReader,
		pathsConfig:          deps.PathsConfig,
		formattingEnabled:    deps.FormattingEnabled,
	}
}
