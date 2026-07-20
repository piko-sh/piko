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

package gopls_bridge

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	protocol "github.com/politepixels/golang-language-server"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/uri"
	"piko.sh/piko/wdk/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// goplsShutdownTimeout bounds how long Close waits for gopls to acknowledge shutdown
	// before the process is killed.
	goplsShutdownTimeout = 5 * time.Second

	// goplsReadyTimeout bounds how long callers wait for gopls to finish its initial
	// workspace load before overlays are sent regardless.
	goplsReadyTimeout = 30 * time.Second

	// goplsHandshakeTimeout bounds the initialize/initialized exchange so a wedged or
	// non-responsive gopls cannot hang the goroutine that triggered the spawn.
	goplsHandshakeTimeout = 10 * time.Second

	// goplsWriteTimeout bounds a single write to gopls's stdin.
	goplsWriteTimeout = 5 * time.Second
)

var (
	// errGoplsShutdownTimeout is the cause attached to the bounded shutdown context so a
	// gopls that fails to acknowledge shutdown within the deadline is distinguishable from a
	// deliberate cancellation when the returned error is inspected.
	errGoplsShutdownTimeout = errors.New("gopls shutdown deadline exceeded")
)

// DiagnosticsHandler receives gopls diagnostics for a virtual document so the higher
// layer can window, reverse-map and merge them into the .pk stream.
type DiagnosticsHandler func(ctx context.Context, params *protocol.PublishDiagnosticsParams)

// Child is one gopls process serving a single Go module root. pikopls drives it as an LSP
// client over the process's stdio.
type Child struct {
	// capabilities holds the server capabilities gopls reported at initialise.
	capabilities protocol.ServerCapabilities

	// server is the LSP proxy used to forward requests to gopls.
	server protocol.Server

	// conn is the underlying JSON-RPC connection to the gopls process.
	conn jsonrpc2.Conn

	// stream closes the process stdio once the child is torn down.
	stream io.Closer

	// handler dispatches the inbound notifications gopls sends back.
	handler *clientHandler

	// command is the spawned gopls process handle, nil for in-memory test streams.
	command *exec.Cmd

	// overlays maps each open virtual document URI to its overlay state.
	overlays map[protocol.DocumentURI]*overlayState

	// done is closed once the gopls process has exited.
	done chan struct{}

	// onDead, when set, is invoked after the process exits so the manager can evict it.
	onDead func(ctx context.Context, moduleRoot string)

	// moduleRoot is the Go module directory this child serves.
	moduleRoot string

	// maxOverlays caps how many virtual documents may stay open against this child.
	maxOverlays int

	// overlayMu guards the overlays map.
	overlayMu sync.Mutex

	// notifyMu serialises notifications sent to gopls.
	notifyMu sync.Mutex

	// lastUsed records the nanosecond timestamp of the most recent use, for the idle reaper.
	lastUsed atomic.Int64

	// dead reports whether the gopls process has exited.
	dead atomic.Bool
}

// overlayState tracks the latest content and version of one virtual document.
type overlayState struct {
	// analysed is closed once gopls first publishes diagnostics for this overlay.
	analysed chan struct{}

	// owners is the set of connection identifiers holding this overlay open.
	owners map[uint64]struct{}

	// content is the latest virtual document body sent to gopls.
	content []byte

	// analysedOnce guards a single close of the analysed channel.
	analysedOnce sync.Once

	// version is the LSP document version of the latest content.
	version int32

	// didOpenSucceeded records that gopls accepted the initial DidOpen.
	didOpenSucceeded bool
}

// processStream couples a child process's stdout (reads) and stdin (writes) into one
// stream for JSON-RPC.
type processStream struct {
	io.ReadCloser

	// writer is the child process's stdin, written under a per-write deadline.
	writer io.WriteCloser

	// onWriteTimeout, when set, is invoked once a write misses its deadline.
	onWriteTimeout func()
}

// IsAlive reports whether the gopls process is still running.
//
// Returns true while the process has not yet exited.
func (c *Child) IsAlive() bool {
	return !c.dead.Load()
}

// Done returns a channel closed when the gopls process has exited, letting request
// forwarders abort a call the instant the child dies instead of waiting for the request
// deadline.
//
// Returns a channel that closes once the process exits.
func (c *Child) Done() <-chan struct{} {
	return c.done
}

// IsReady reports, without blocking, whether gopls has finished its initial workspace
// load. Interactive request handlers use this to avoid stalling on a cold gopls and fall
// back to piko's own intelligence until the background warm completes.
//
// Returns true once the initial workspace load has signalled readiness.
func (c *Child) IsReady() bool {
	select {
	case <-c.handler.ready:
		return true
	default:
		return false
	}
}

// Capabilities returns the server capabilities gopls reported at initialise.
//
// Returns the captured gopls server capabilities.
func (c *Child) Capabilities() protocol.ServerCapabilities {
	return c.capabilities
}

// Server exposes the gopls LSP proxy for forwarding requests.
//
// Returns the gopls server proxy.
func (c *Child) Server() protocol.Server {
	return c.server
}

// WaitReady blocks until gopls has finished its initial workspace load, the child dies,
// the context is cancelled, or the ready timeout elapses.
//
// Returns true once gopls is ready or the timeout forces overlays anyway, and false when
// the child has died so callers skip forwarding to a dead process.
func (c *Child) WaitReady(ctx context.Context) bool {
	select {
	case <-c.handler.ready:
		return true
	case <-c.done:
		return false
	case <-ctx.Done():
		return false
	case <-time.After(goplsReadyTimeout):
		_, l := logger_domain.From(ctx, log)
		l.Trace("gopls did not signal workspace-load completion before the ready timeout; forwarding overlays anyway",
			logger_domain.String("moduleRoot", c.moduleRoot))
		c.handler.markReady()
		return true
	}
}

// Close shuts down gopls gracefully and then ensures the process is gone.
//
// Calling it repeatedly is harmless. When the process has already exited (c.dead), the
// graceful shutdown RPCs are skipped because they would otherwise hang until their
// deadline elapses against a dead peer.
//
// Returns error when a non-benign teardown failure surfaces while closing the child.
func (c *Child) Close(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeoutCause(
		context.WithoutCancel(ctx),
		goplsShutdownTimeout,
		fmt.Errorf("gopls shutdown exceeded %s: %w", goplsShutdownTimeout, errGoplsShutdownTimeout),
	)
	defer cancel()

	var closeErrors []error
	if !c.dead.Load() {
		closeErrors = append(closeErrors, c.server.Shutdown(shutdownCtx), c.server.Exit(shutdownCtx))
	}
	if c.conn != nil {
		closeErrors = append(closeErrors, c.conn.Close())
	}
	if c.stream != nil {
		closeErrors = append(closeErrors, c.stream.Close())
	}
	if c.command != nil && c.command.Process != nil {
		closeErrors = append(closeErrors, c.command.Process.Kill())
		select {
		case <-c.done:
		case <-time.After(goplsShutdownTimeout):
		}
	}

	significant := make([]error, 0, len(closeErrors))
	for _, closeErr := range closeErrors {
		if closeErr == nil || isBenignCloseError(closeErr) {
			continue
		}
		significant = append(significant, closeErr)
	}
	if len(significant) == 0 {
		return nil
	}
	return fmt.Errorf("closing gopls child for %s: %w", c.moduleRoot, errors.Join(significant...))
}

// overlayCount reports how many virtual documents are currently open against this child,
// so the reaper never evicts a child with live overlays.
//
// Returns the count of open overlays.
//
// Concurrency: acquires overlayMu while reading the overlays map.
func (c *Child) overlayCount() int {
	c.overlayMu.Lock()
	defer c.overlayMu.Unlock()
	return len(c.overlays)
}

// monitor reaps the gopls process and marks the child dead when it exits.
func (c *Child) monitor(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	defer goroutine.RecoverPanic(ctx, "gopls_bridge.child.monitor")
	exitErr := c.command.Wait()
	c.dead.Store(true)

	if exitErr != nil {
		l.Internal("gopls child process exited",
			logger_domain.String("moduleRoot", c.moduleRoot),
			logger_domain.Error(exitErr))
	}
	if c.onDead != nil {
		c.onDead(ctx, c.moduleRoot)
	}
	close(c.done)
}

// touch records that the child was just used, so the idle reaper can tell a busy child
// from an abandoned one.
//
// Takes nowNanos (int64) which is the current time in Unix nanoseconds to store as the
// last-used timestamp.
func (c *Child) touch(nowNanos int64) {
	c.lastUsed.Store(nowNanos)
}

// Write sends bytes to the child process's standard input under a write deadline.
//
// Takes payload ([]byte) which holds the bytes to write to gopls's stdin.
//
// Returns the number of bytes written, then error when the write fails or misses its
// deadline.
func (s *processStream) Write(payload []byte) (int, error) {
	if deadliner, ok := s.writer.(interface{ SetWriteDeadline(time.Time) error }); ok {
		_ = deadliner.SetWriteDeadline(time.Now().Add(goplsWriteTimeout))
	}
	written, err := s.writer.Write(payload)
	if err != nil && errors.Is(err, os.ErrDeadlineExceeded) && s.onWriteTimeout != nil {
		s.onWriteTimeout()
	}
	return written, err
}

// Close closes both ends of the process stream.
//
// Returns error when closing either the write or read end fails.
func (s *processStream) Close() error {
	writeErr := s.writer.Close()
	readErr := s.ReadCloser.Close()
	if writeErr != nil {
		return writeErr
	}
	return readErr
}

// spawnGopls starts a `gopls serve` process for a module root and returns a stream over
// its stdio plus the command handle for lifecycle control.
//
// Takes goplsPath (string) which is the resolved path to the gopls binary.
// Takes moduleRoot (string) which is the Go module directory the process serves.
//
// Returns the stdio stream, the command handle, then error when a pipe cannot be opened
// or the process fails to start.
func spawnGopls(goplsPath, moduleRoot string) (io.ReadWriteCloser, *exec.Cmd, error) {
	command := exec.Command(goplsPath, "serve")
	command.Dir = moduleRoot
	command.Stderr = os.Stderr

	command.Env = goSubprocessEnv()

	stdin, stdinErr := command.StdinPipe()
	if stdinErr != nil {
		return nil, nil, fmt.Errorf("gopls stdin pipe: %w", stdinErr)
	}
	stdout, stdoutErr := command.StdoutPipe()
	if stdoutErr != nil {
		_ = stdin.Close()
		return nil, nil, fmt.Errorf("gopls stdout pipe: %w", stdoutErr)
	}
	if startErr := command.Start(); startErr != nil {
		return nil, nil, fmt.Errorf("starting gopls: %w", startErr)
	}

	stream := &processStream{
		ReadCloser: stdout,
		writer:     stdin,

		onWriteTimeout: func() { _ = command.Process.Kill() },
	}
	return stream, command, nil
}

// dialChild performs the LSP handshake with a gopls instance and returns a ready Child.
//
// Takes stream (io.ReadWriteCloser) which carries the JSON-RPC traffic to and from gopls.
// Takes command (*exec.Cmd) which is the spawned process handle, nil for in-memory
// streams.
// Takes moduleRoot (string) which is the Go module directory the child serves.
// Takes diagnosticsHandler (DiagnosticsHandler) which receives gopls diagnostics for
// windowing and merging.
// Takes onDead (func(ctx context.Context, moduleRoot string)) which the monitor calls
// after the process exits.
//
// Returns the ready Child, then error when the handshake fails.
//
// Concurrency: starts the monitor goroutine on success and closes c.done immediately when
// there is no process to reap.
func dialChild(
	ctx context.Context,
	stream io.ReadWriteCloser,
	command *exec.Cmd,
	moduleRoot string,
	diagnosticsHandler DiagnosticsHandler,
	onDead func(ctx context.Context, moduleRoot string),
) (*Child, error) {
	handler := newClientHandler(moduleRoot)
	rpcStream := jsonrpc2.NewStream(stream)

	clientCtx := context.WithoutCancel(ctx)
	_, conn, server := protocol.NewClient(clientCtx, handler, rpcStream, slog.Default())

	reap := func() {
		_ = conn.Close()
		_ = stream.Close()
		if command != nil && command.Process != nil {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	}

	capabilities, handshakeErr := handshakeGopls(ctx, server, moduleRoot)
	if handshakeErr != nil {
		reap()
		return nil, handshakeErr
	}

	child := &Child{
		capabilities: capabilities,
		server:       server,
		conn:         conn,
		stream:       stream,
		handler:      handler,
		command:      command,
		overlays:     make(map[protocol.DocumentURI]*overlayState),
		done:         make(chan struct{}),
		onDead:       onDead,
		moduleRoot:   moduleRoot,
	}
	handler.setSink(func(sinkCtx context.Context, params *protocol.PublishDiagnosticsParams) {
		defer goroutine.RecoverPanic(sinkCtx, "gopls_bridge.diagnosticsSink")
		child.markOverlayAnalysed(params.URI)
		if diagnosticsHandler != nil {
			diagnosticsHandler(sinkCtx, params)
		}
	})
	if command != nil {
		go child.monitor(context.WithoutCancel(ctx))
	} else {
		close(child.done)
	}
	return child, nil
}

// handshakeGopls performs the bounded initialize/initialized exchange and returns the
// gopls server capabilities. It rejects an empty initialize result (a nil deref hazard)
// and a gopls older than the supported floor, routing both into the caller's
// graceful-degradation path.
//
// Takes server (protocol.Server) which is the gopls proxy to run the handshake against.
// Takes moduleRoot (string) which roots the initialise request at the module directory.
//
// Returns the gopls server capabilities, then error when the handshake fails or gopls is
// unsupported.
func handshakeGopls(ctx context.Context, server protocol.Server, moduleRoot string) (protocol.ServerCapabilities, error) {
	var capabilities protocol.ServerCapabilities

	handshakeCtx, cancel := context.WithTimeoutCause(
		context.WithoutCancel(ctx),
		goplsHandshakeTimeout,
		fmt.Errorf("gopls handshake exceeded %s: %w", goplsHandshakeTimeout, ErrGoplsUnavailable),
	)
	defer cancel()

	result, initErr := server.Initialize(handshakeCtx, buildGoplsInitializeParams(moduleRoot))
	if initErr != nil {
		return capabilities, fmt.Errorf("gopls initialize: %w", initErr)
	}
	if result == nil {
		return capabilities, fmt.Errorf("gopls returned an empty initialize result: %w", ErrGoplsUnavailable)
	}
	if result.ServerInfo != nil && !goplsVersionSupported(result.ServerInfo.Version) {
		return capabilities, fmt.Errorf("gopls %q is older than the minimum supported version: %w", result.ServerInfo.Version, ErrGoplsUnavailable)
	}
	if initialisedErr := server.Initialized(handshakeCtx, &protocol.InitializedParams{}); initialisedErr != nil {
		return capabilities, fmt.Errorf("gopls initialized: %w", initialisedErr)
	}
	return result.Capabilities, nil
}

// buildGoplsInitializeParams constructs the initialise request pikopls sends to its gopls
// child, rooting it at the module directory and enabling the features the bridge relies
// on.
//
// Takes moduleRoot (string) which is the Go module directory used as the workspace root.
//
// Returns the populated initialise parameters.
func buildGoplsInitializeParams(moduleRoot string) *protocol.InitializeParams {
	rootURI := fileURI(moduleRoot)
	return &protocol.InitializeParams{
		ProcessID:    safeconv.IntToInt32(os.Getpid()),
		RootURI:      rootURI,
		Capabilities: buildGoplsClientCapabilities(),
		WorkspaceFolders: []protocol.WorkspaceFolder{{
			URI:  string(rootURI),
			Name: filepath.Base(moduleRoot),
		}},
		InitializationOptions: map[string]any{
			"usePlaceholders":    true,
			"completeUnimported": true,
			"semanticTokens":     true,
		},
	}
}

// buildGoplsClientCapabilities advertises pikopls's capabilities to gopls so it returns
// rich responses and emits work-done progress (used for readiness).
//
// Returns the advertised client capabilities.
func buildGoplsClientCapabilities() protocol.ClientCapabilities {
	return protocol.ClientCapabilities{
		Window: &protocol.WindowClientCapabilities{
			WorkDoneProgress: true,
		},
		TextDocument: &protocol.TextDocumentClientCapabilities{
			Synchronization: &protocol.TextDocumentSyncClientCapabilities{DidSave: true},
			Hover: &protocol.HoverTextDocumentClientCapabilities{
				ContentFormat: []protocol.MarkupKind{protocol.Markdown, protocol.PlainText},
			},
			Completion:     &protocol.CompletionTextDocumentClientCapabilities{},
			SignatureHelp:  &protocol.SignatureHelpTextDocumentClientCapabilities{},
			Definition:     &protocol.DefinitionTextDocumentClientCapabilities{},
			TypeDefinition: &protocol.TypeDefinitionTextDocumentClientCapabilities{},
			SemanticTokens: &protocol.SemanticTokensClientCapabilities{},
		},
	}
}

// PathToFileURI converts an absolute filesystem path to a file:// document URI.
//
// Takes path (string) which is the absolute filesystem path to encode.
//
// Returns the file:// document URI for the path.
func PathToFileURI(path string) protocol.DocumentURI {
	return fileURI(path)
}

// fileURI converts an absolute filesystem path to a file:// document URI.
//
// Takes path (string) which is the absolute filesystem path to encode.
//
// Returns the canonical file:// document URI.
func fileURI(path string) protocol.DocumentURI {
	return protocol.DocumentURI(uri.File(path))
}

// isBenignCloseError reports whether a teardown error is the expected consequence of a
// gone peer.
//
// Takes err (error) which is the teardown error to classify.
//
// Returns true when the error is an expected close of an already-gone peer.
func isBenignCloseError(err error) bool {
	return errors.Is(err, net.ErrClosed) ||
		errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, os.ErrClosed) ||
		errors.Is(err, os.ErrProcessDone)
}
