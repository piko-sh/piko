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
	"net"
	"sync"
	"sync/atomic"
	"time"

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
	"piko.sh/piko/wdk/goroutine"
)

const (
	// defaultMaxLSPMessageBytes caps the CUMULATIVE bytes read over a connection's lifetime
	// (not a single message) at 512 MiB.
	//
	// Real LSP traffic is small structured JSON; an unbounded Content-Length lets a
	// malformed or hostile peer drive the server to OOM via io.ReadFull. The cap is enforced
	// by the cappedReadWriteCloser, which counts bytes for the life of the connection
	// because the byte stream below the jsonrpc2 framer has no message boundaries to reset
	// on. The ceiling is therefore generous enough that a long full-sync editing session
	// never reaches it, while still bounding a runaway peer.
	defaultMaxLSPMessageBytes = 512 * 1024 * 1024

	// defaultLSPConnectionWriteTimeout bounds a single write to an editor. A wedged editor
	// that stops reading must not block the server (and, for the shared gopls child, must
	// not stall diagnostics fan-out to co-tenant connections); the write fails after this
	// deadline and the connection is torn down.
	defaultLSPConnectionWriteTimeout = 15 * time.Second

	// defaultMaxConcurrentLSPConnections caps concurrent in-flight LSP connections handled
	// by the TCP adapter. Excess connections are accepted, rejected with an error
	// notification, and immediately closed so a hostile peer cannot exhaust file descriptors
	// or goroutines.
	defaultMaxConcurrentLSPConnections = 64

	// defaultLSPConnectionInactivityTimeout sets the initial read deadline on each accepted
	// connection. Editors keep the conn open for the lifetime of the workspace, so a
	// per-message reset is impractical at this layer; we instead expose a long inactivity
	// window that protects against leaked half-open connections without disrupting normal
	// usage.
	defaultLSPConnectionInactivityTimeout = 30 * time.Minute

	// shutdownDrainTimeout caps how long Run waits for in-flight handler goroutines to
	// finish after the listener returns. Keeps the shutdown path bounded if a misbehaving
	// peer wedges a handler.
	shutdownDrainTimeout = 30 * time.Second

	// minAcceptBackoff and maxAcceptBackoff bound the exponential backoff applied after a
	// transient listener Accept error (e.g. fd exhaustion), so the accept loop throttles
	// instead of spinning until descriptors free up.
	minAcceptBackoff = 5 * time.Millisecond

	// maxAcceptBackoff is the upper bound on the exponential accept backoff delay.
	maxAcceptBackoff = 1 * time.Second

	// attributeKeyRemoteAddr is the structured-log attribute key for the remote address of
	// an accepted TCP connection.
	attributeKeyRemoteAddr = "remoteAddr"
)

// tcpAdapter listens on a TCP socket and creates a new LSP server for each client
// connection. It implements LSPServerPort.
type tcpAdapter struct {
	// coordinatorService handles LSP coordination for connections.
	coordinatorService coordinator_domain.CoordinatorService

	// resolver finds symbol definitions for LSP operations.
	resolver resolver_domain.ResolverPort

	// lspReader provides file system access for the LSP server.
	lspReader annotator_domain.FSReaderPort

	// connectionSemaphore caps concurrent in-flight handler goroutines. nil means unlimited
	// (only used when MaxConcurrentConnections is set to a non-positive value via the
	// option).
	connectionSemaphore chan struct{}

	// docCache stores parsed documentation for files to avoid repeated parsing.
	docCache *lsp_domain.DocumentCache

	// pathsConfig provides workspace path settings for document processing.
	pathsConfig *config.PathsConfig

	// typeInspectorManager builds type data for LSP requests.
	typeInspectorManager *inspector_domain.TypeBuilder

	// goplsManager owns the shared gopls child processes for Go-block requests, shared
	// across every connection this adapter serves.
	goplsManager *gopls_bridge.Manager

	// addr is the TCP address to listen on (e.g. "localhost:8080").
	addr string

	// goroutineWG tracks spawned handler goroutines so Run can drain them on shutdown.
	goroutineWG sync.WaitGroup

	// maxMessageBytes caps the size of a single LSP message read from the wire.
	maxMessageBytes int64

	// connectionInactivityTimeout sets the initial conn read deadline.
	connectionInactivityTimeout time.Duration

	// formattingEnabled controls whether formatting capabilities are advertised.
	formattingEnabled bool

	// goplsBridgeEnabled is the process-level default for the Go-block bridge.
	goplsBridgeEnabled bool
}

var (
	_ lsp_domain.LSPServerPort = (*tcpAdapter)(nil)
)

// TCPAdapterDeps holds the dependencies for creating a TCP adapter.
type TCPAdapterDeps struct {
	// Resolver finds symbol definitions for LSP operations.
	Resolver resolver_domain.ResolverPort

	// LSPReader provides file system access for the LSP server.
	LSPReader annotator_domain.FSReaderPort

	// CoordinatorService handles LSP coordination for connections.
	CoordinatorService coordinator_domain.CoordinatorService

	// GoplsManager owns the shared gopls child processes for Go-block requests.
	GoplsManager *gopls_bridge.Manager

	// TypeInspectorManager builds type data for LSP requests.
	TypeInspectorManager *inspector_domain.TypeBuilder

	// DocCache stores parsed documentation for files.
	DocCache *lsp_domain.DocumentCache

	// PathsConfig provides workspace path settings for document processing.
	PathsConfig *config.PathsConfig

	// Addr is the TCP address to listen on (e.g. "localhost:8080").
	Addr string

	// MaxConcurrentConnections caps concurrent handler goroutines. Zero or negative falls
	// back to defaultMaxConcurrentLSPConnections; pass a very large value to effectively
	// disable the cap.
	MaxConcurrentConnections int

	// ConnectionInactivityTimeout sets the initial conn read deadline. Zero or negative
	// falls back to defaultLSPConnectionInactivityTimeout.
	ConnectionInactivityTimeout time.Duration

	// MaxMessageBytes caps the size of a single LSP message read from the wire. Zero or
	// negative falls back to defaultMaxLSPMessageBytes.
	MaxMessageBytes int64

	// FormattingEnabled controls whether formatting capabilities are advertised.
	FormattingEnabled bool

	// GoplsBridgeEnabled is the process-level default for the Go-block bridge.
	GoplsBridgeEnabled bool
}

// Run starts the TCP server and accepts client connections in a loop. The stream
// parameter is ignored as TCP creates its own connections.
//
// Returns error when the TCP listener cannot be created.
//
// Spawns a new goroutine for each accepted connection. These goroutines run until the
// connection is closed or the context is cancelled. On context cancellation, the listener
// stops accepting and Run blocks (up to shutdownDrainTimeout) waiting for in-flight
// handler goroutines via goroutineWG so callers can rely on a clean shutdown.
func (a *tcpAdapter) Run(ctx context.Context, _ io.ReadWriteCloser) error {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("Starting Piko LSP TCP server", logger_domain.String("address", a.addr))
	listener, err := net.Listen("tcp", a.addr)
	if err != nil {
		return fmt.Errorf("listening on TCP address %s: %w", a.addr, err)
	}

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	var acceptBackoff time.Duration
	for {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			if ctx.Err() != nil {
				l.Debug("LSP TCP listener stopped",
					logger_domain.String("cause", context.Cause(ctx).Error()))
				break
			}
			if errors.Is(acceptErr, net.ErrClosed) {
				l.Debug("LSP TCP listener closed", logger_domain.Error(acceptErr))
				break
			}
			acceptBackoff = backoffOnAcceptError(ctx, acceptErr, acceptBackoff)
			continue
		}
		acceptBackoff = 0

		if !a.acquireConnectionSlot() {
			l.Warn("Rejecting LSP TCP connection: concurrent-connection cap reached",
				logger_domain.String(attributeKeyRemoteAddr, conn.RemoteAddr().String()))
			_ = conn.Close()
			continue
		}

		a.goroutineWG.Go(func() {
			defer a.releaseConnectionSlot()
			defer goroutine.RecoverPanic(ctx, "lsp.tcpAdapter.handleConnection")
			a.handleConnection(ctx, conn)
		})
	}

	a.drainHandlers(ctx)
	if a.goplsManager != nil {
		_ = a.goplsManager.Close(context.WithoutCancel(ctx))
	}
	return nil
}

// backoffOnAcceptError sleeps for an exponentially increasing, capped delay after a
// transient listener Accept error.
//
// The delay throttles the accept loop instead of spinning (e.g. under fd exhaustion)
// until descriptors free up, and returns immediately when ctx is cancelled so shutdown is
// not delayed. Mirrors net/http's Server.Serve.
//
// Takes acceptErr (error) which is the transient Accept error that triggered the backoff.
// Takes current (time.Duration) which is the previous backoff delay, doubled and capped
// to derive the next one.
//
// Returns time.Duration which is the next backoff delay.
func backoffOnAcceptError(ctx context.Context, acceptErr error, current time.Duration) time.Duration {
	ctx, l := logger_domain.From(ctx, log)
	next := minAcceptBackoff
	if current != 0 {
		next = min(current*2, maxAcceptBackoff)
	}
	l.Debug("Error accepting LSP TCP connection; backing off",
		logger_domain.Error(acceptErr),
		logger_domain.String("retryIn", next.String()))
	timer := time.NewTimer(next)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
	}
	return next
}

// acquireConnectionSlot reserves a semaphore slot for a new handler goroutine.
//
// Returns bool which is true when a slot was reserved, false when the cap is reached and
// the connection should be rejected.
func (a *tcpAdapter) acquireConnectionSlot() bool {
	if a.connectionSemaphore == nil {
		return true
	}
	select {
	case a.connectionSemaphore <- struct{}{}:
		return true
	default:
		return false
	}
}

// releaseConnectionSlot returns a previously reserved semaphore slot.
func (a *tcpAdapter) releaseConnectionSlot() {
	if a.connectionSemaphore == nil {
		return
	}
	<-a.connectionSemaphore
}

// drainHandlers waits for in-flight handler goroutines to finish, with an upper bound so
// a wedged handler cannot block shutdown forever.
//
// Takes ctx (context.Context) used for diagnostic logging via logger_domain.
//
// Concurrency: spawns a goroutine that waits on a.goroutineWG and signals completion via
// the done channel. Caller blocks until either drain completes or shutdownDrainTimeout
// elapses.
func (a *tcpAdapter) drainHandlers(ctx context.Context) {
	_, l := logger_domain.From(ctx, log)

	done := make(chan struct{})
	go func() {
		a.goroutineWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		l.Debug("LSP TCP handlers drained")
	case <-time.After(shutdownDrainTimeout):
		l.Warn("LSP TCP handlers did not drain within timeout",
			logger_domain.String("timeout", shutdownDrainTimeout.String()))
	}
}

// handleConnection manages a single client connection over TCP.
//
// The conn is wrapped with a cappedReadWriteCloser so a hostile peer cannot announce a
// massive Content-Length and force io.ReadFull to allocate it. An initial read deadline
// is set so leaked half-open connections are closed once the deadline elapses.
//
// Takes conn (net.Conn) which is the TCP connection to handle.
func (a *tcpAdapter) handleConnection(ctx context.Context, conn net.Conn) {
	ctx, l := logger_domain.From(ctx, log)

	remoteAddr := conn.RemoteAddr().String()
	l.Debug("LSP client connected", logger_domain.String(attributeKeyRemoteAddr, remoteAddr))
	defer func() { _ = conn.Close() }()

	if a.connectionInactivityTimeout > 0 {
		if err := conn.SetReadDeadline(time.Now().Add(a.connectionInactivityTimeout)); err != nil {
			l.Debug("Failed to set LSP TCP read deadline",
				logger_domain.Error(err),
				logger_domain.String(attributeKeyRemoteAddr, remoteAddr))
		}
	}

	deadlined := &writeDeadlineConn{Conn: conn, timeout: defaultLSPConnectionWriteTimeout}
	capped := newCappedReadWriteCloser(deadlined, a.maxMessageBytes)
	stream := jsonrpc2.NewStream(capped)
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
	})

	defer func() { _ = pikoServer.Shutdown(context.WithoutCancel(ctx)) }()

	_, jsonrpcConn, client := protocol.NewServer(ctx, pikoServer, stream, slog.Default())
	pikoServer.SetClient(client)
	pikoServer.SetConn(jsonrpcConn)

	l.Debug("Handling LSP connection; waiting for close",
		logger_domain.String(attributeKeyRemoteAddr, remoteAddr))
	select {
	case <-jsonrpcConn.Done():
	case <-ctx.Done():
		_ = jsonrpcConn.Close()
		<-jsonrpcConn.Done()
	}

	logConnectionClose(ctx, remoteAddr, jsonrpcConn.Err())
}

// logConnectionClose logs an LSP disconnect, surfacing the read-cap cause distinctly so a
// peer that exceeded the size cap is not mistaken for a normal client close.
//
// Takes remoteAddr (string) which is the remote address of the disconnected client.
// Takes connErr (error) which is the connection's terminal error, if any.
func logConnectionClose(ctx context.Context, remoteAddr string, connErr error) {
	_, l := logger_domain.From(ctx, log)
	switch {
	case errors.Is(connErr, errMessageTooLarge):
		l.Warn("LSP client disconnected: message exceeded the read cap",
			logger_domain.String(attributeKeyRemoteAddr, remoteAddr), logger_domain.Error(connErr))
	case connErr != nil:
		l.Debug("LSP client disconnected", logger_domain.String(attributeKeyRemoteAddr, remoteAddr), logger_domain.Error(connErr))
	default:
		l.Debug("LSP client disconnected", logger_domain.String(attributeKeyRemoteAddr, remoteAddr))
	}
}

// NewTCPAdapter creates a new TCP adapter for the LSP server.
//
// Takes deps (TCPAdapterDeps) which provides all dependencies the LSP server needs.
// Optional knobs (MaxMessageBytes, MaxConcurrentConnections, ConnectionInactivityTimeout)
// fall back to high defaults when zero or negative.
//
// Returns lsp_domain.LSPServerPort which is the configured adapter ready for use.
// Returns error when any required dependency field in deps is nil, including
// CoordinatorService, Resolver, TypeInspectorManager, DocCache, LSPReader, or
// PathsConfig.
func NewTCPAdapter(deps TCPAdapterDeps) (lsp_domain.LSPServerPort, error) {
	switch {
	case deps.CoordinatorService == nil:
		return nil, errors.New("NewTCPAdapter: coordinatorService cannot be nil")
	case deps.Resolver == nil:
		return nil, errors.New("NewTCPAdapter: resolver cannot be nil")
	case deps.TypeInspectorManager == nil:
		return nil, errors.New("NewTCPAdapter: typeInspectorManager cannot be nil")
	case deps.DocCache == nil:
		return nil, errors.New("NewTCPAdapter: docCache cannot be nil")
	case deps.LSPReader == nil:
		return nil, errors.New("NewTCPAdapter: lspReader cannot be nil")
	case deps.PathsConfig == nil:
		return nil, errors.New("NewTCPAdapter: pathsConfig cannot be nil")
	}

	maxMessageBytes := deps.MaxMessageBytes
	if maxMessageBytes <= 0 {
		maxMessageBytes = defaultMaxLSPMessageBytes
	}

	maxConnections := deps.MaxConcurrentConnections
	if maxConnections <= 0 {
		maxConnections = defaultMaxConcurrentLSPConnections
	}

	inactivity := deps.ConnectionInactivityTimeout
	if inactivity <= 0 {
		inactivity = defaultLSPConnectionInactivityTimeout
	}

	return &tcpAdapter{
		addr:                        deps.Addr,
		coordinatorService:          deps.CoordinatorService,
		resolver:                    deps.Resolver,
		typeInspectorManager:        deps.TypeInspectorManager,
		docCache:                    deps.DocCache,
		lspReader:                   deps.LSPReader,
		pathsConfig:                 deps.PathsConfig,
		goplsManager:                deps.GoplsManager,
		formattingEnabled:           deps.FormattingEnabled,
		goplsBridgeEnabled:          deps.GoplsBridgeEnabled,
		maxMessageBytes:             maxMessageBytes,
		connectionInactivityTimeout: inactivity,
		connectionSemaphore:         make(chan struct{}, maxConnections),
	}, nil
}

// cappedReadWriteCloser caps cumulative reads from an io.ReadWriteCloser.
//
// The jsonrpc2 stream parses Content-Length from headers and then issues an io.ReadFull
// for the announced size; without this wrapper a hostile peer can announce 100 GiB and
// exhaust memory before the read fails. Once the cap is reached, every Read returns
// errMessageTooLarge so jsonrpc2 surfaces the failure and we close the conn.
type cappedReadWriteCloser struct {
	// inner is the underlying conn whose reads are capped.
	inner io.ReadWriteCloser

	// limit is the hard cap on aggregate bytes returned from Read.
	limit int64

	// readSoFar tracks how many bytes have been returned from Read.
	readSoFar atomic.Int64
}

var (
	// errMessageTooLarge is returned from Read when an LSP peer attempts to push more bytes
	// than maxMessageBytes through the conn. Surfacing it as a sentinel lets tests assert
	// against errors.Is.
	errMessageTooLarge = errors.New("lsp: message exceeds maximum size")
)

// newCappedReadWriteCloser wraps inner so cumulative Read sizes cannot exceed limit. A
// non-positive limit disables the cap.
//
// Takes inner (io.ReadWriteCloser) which is the underlying conn.
// Takes limit (int64) which is the cumulative byte cap; <= 0 disables.
//
// Returns *cappedReadWriteCloser which delegates Write/Close to inner and enforces the
// cap on Read.
func newCappedReadWriteCloser(inner io.ReadWriteCloser, limit int64) *cappedReadWriteCloser {
	return &cappedReadWriteCloser{inner: inner, limit: limit}
}

// Read delegates to the wrapped conn but trims the requested length so the cumulative
// bytes returned never exceed limit, returning errMessageTooLarge once the cap is
// reached.
//
// Takes p ([]byte) which receives the read bytes.
//
// Returns int which is the number of bytes read.
// Returns error which wraps the inner Read error or errMessageTooLarge.
func (c *cappedReadWriteCloser) Read(p []byte) (int, error) {
	if c.limit <= 0 {
		return c.inner.Read(p)
	}

	already := c.readSoFar.Load()
	remaining := c.limit - already
	if remaining <= 0 {
		return 0, fmt.Errorf("%w: read cap of %d bytes exhausted", errMessageTooLarge, c.limit)
	}

	if int64(len(p)) > remaining {
		p = p[:remaining]
	}

	n, err := c.inner.Read(p)
	if n > 0 {
		c.readSoFar.Add(int64(n))
	}
	return n, err
}

// Write delegates to the wrapped conn unchanged.
//
// Takes p ([]byte) which is the bytes to write.
//
// Returns int which is the number of bytes written.
// Returns error which is the inner Write error.
func (c *cappedReadWriteCloser) Write(p []byte) (int, error) {
	return c.inner.Write(p)
}

// Close delegates to the wrapped conn.
//
// Returns error which is the inner Close error.
func (c *cappedReadWriteCloser) Close() error {
	return c.inner.Close()
}

// writeDeadlineConn applies a per-write deadline to an editor connection so a client that
// stops reading cannot block the server. On timeout the write returns an error, jsonrpc2
// surfaces it, and the connection is torn down.
type writeDeadlineConn struct {
	net.Conn

	// timeout bounds how long a single write to the editor may block.
	timeout time.Duration
}

// Write sets the write deadline before delegating, bounding how long a single write to a
// wedged editor can block.
//
// Takes p ([]byte) which is the bytes to write.
//
// Returns int which is the number of bytes written.
// Returns error which is the inner Write error.
func (c *writeDeadlineConn) Write(p []byte) (int, error) {
	if c.timeout > 0 {
		_ = c.SetWriteDeadline(time.Now().Add(c.timeout))
	}
	return c.Conn.Write(p)
}
