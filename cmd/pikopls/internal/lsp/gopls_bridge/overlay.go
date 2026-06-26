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
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	protocol "github.com/politepixels/golang-language-server"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// goLanguageID is the LSP language identifier for virtual Go documents.
	goLanguageID = protocol.LanguageIdentifier("go")
)

var (
	// ErrOverlayLimit is returned by SyncOverlay when a child already holds the maximum
	// number of open overlays and a new document is requested. Callers treat it like the
	// other bridge-unavailable signals and degrade to piko-only intelligence for that
	// document rather than surfacing an editor-visible error.
	ErrOverlayLimit = errors.New("gopls overlay limit reached for module child")
)

// SyncOverlay opens or updates a virtual document against gopls.
//
// On first use it opens the document; subsequent calls send a full-content change,
// skipping unchanged content. The owner identifies the connection that holds the overlay
// open: a single gopls child is shared across every editor connection on a module root,
// so the same .pk file open in two editors maps to one overlay with two owners. The
// overlay is only closed against gopls once every owner has released it (see
// CloseOverlay), so one connection closing a document cannot evict an overlay another is
// still using. When two owners hold divergent unsaved buffers for the same file, the last
// writer's content wins (gopls gets one view); this is acceptable for read-only
// intelligence and never corrupts a file.
//
// Takes owner (uint64) which identifies the connection holding the overlay open.
// Takes uri (protocol.DocumentURI) which addresses the virtual document.
// Takes content (byte slice) which is the full buffer contents to mirror to gopls.
//
// Returns error when the open or change notification to gopls fails, or ErrOverlayLimit
// when the child already holds the maximum number of overlays.
//
// Concurrency: safe for concurrent use; serialises through overlayMu and notifyMu.
func (c *Child) SyncOverlay(ctx context.Context, owner uint64, uri protocol.DocumentURI, content []byte) error {
	c.overlayMu.Lock()
	state, exists := c.overlays[uri]
	if !exists {
		return c.openNewOverlay(ctx, owner, uri, content)
	}

	state.owners[owner] = struct{}{}
	if bytes.Equal(state.content, content) {
		c.overlayMu.Unlock()
		return nil
	}

	state.version++
	state.content = bytes.Clone(content)
	version := state.version
	c.notifyMu.Lock()
	c.overlayMu.Unlock()
	err := c.server.DidChange(ctx, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
			Version:                version,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{{Text: string(content)}},
	})
	c.notifyMu.Unlock()
	return err
}

// openNewOverlay registers a brand-new overlay and opens it against gopls.
//
// It is called with overlayMu held and always releases it. A failed initial DidOpen
// evicts the speculative registration (see rollbackUnopenedOverlay) so a later sync
// re-opens rather than sending a DidChange for a document gopls never opened.
//
// Takes owner (uint64) which identifies the connection holding the overlay open.
// Takes uri (protocol.DocumentURI) which addresses the virtual document.
// Takes content (byte slice) which is the initial buffer contents.
//
// Returns error when the overlay limit is reached or the DidOpen to gopls fails.
//
// Concurrency: entered with overlayMu held; releases it before notifying gopls under
// notifyMu.
func (c *Child) openNewOverlay(ctx context.Context, owner uint64, uri protocol.DocumentURI, content []byte) error {
	if c.maxOverlays > 0 && len(c.overlays) >= c.maxOverlays {
		c.overlayMu.Unlock()
		return fmt.Errorf("opening overlay %s: %w", uri, ErrOverlayLimit)
	}
	state := &overlayState{
		content:  bytes.Clone(content),
		version:  1,
		analysed: make(chan struct{}),
		owners:   map[uint64]struct{}{owner: {}},
	}
	c.overlays[uri] = state

	c.notifyMu.Lock()
	c.overlayMu.Unlock()
	err := c.server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: goLanguageID,
			Version:    1,
			Text:       string(content),
		},
	})
	c.notifyMu.Unlock()
	if err != nil {
		c.rollbackUnopenedOverlay(uri, state)
		return err
	}
	c.overlayMu.Lock()
	state.didOpenSucceeded = true
	c.overlayMu.Unlock()
	return nil
}

// rollbackUnopenedOverlay removes the registration left by a failed initial DidOpen.
//
// A later sync then re-attempts a fresh open rather than sending a DidChange for a
// document gopls never opened (which would leave it never analysed for its lifetime). It
// evicts only the exact state object whose open failed and only while its open never
// succeeded, so a concurrent sync that replaced the registration, or one that raced an
// already-successful open, is left untouched.
//
// Takes uri (protocol.DocumentURI) which addresses the virtual document.
// Takes failed (overlayState pointer) which is the registration to evict.
//
// Concurrency: acquires overlayMu while evicting the failed registration.
func (c *Child) rollbackUnopenedOverlay(uri protocol.DocumentURI, failed *overlayState) {
	c.overlayMu.Lock()
	if current, ok := c.overlays[uri]; ok && current == failed && !current.didOpenSucceeded {
		delete(c.overlays, uri)
	}
	c.overlayMu.Unlock()
}

// CloseOverlay releases one owner's hold on a virtual document.
//
// It closes the document against gopls only once no owner remains. Closing an unknown
// document, or one still held by another connection, is a no-op for gopls.
//
// Takes owner (uint64) which identifies the connection releasing the overlay.
// Takes uri (protocol.DocumentURI) which addresses the virtual document.
//
// Returns error when the DidClose notification to gopls fails.
//
// Concurrency: safe for concurrent use; serialises through overlayMu and notifyMu.
func (c *Child) CloseOverlay(ctx context.Context, owner uint64, uri protocol.DocumentURI) error {
	c.overlayMu.Lock()
	state, exists := c.overlays[uri]
	if !exists {
		c.overlayMu.Unlock()
		return nil
	}
	delete(state.owners, owner)
	if len(state.owners) > 0 {
		c.overlayMu.Unlock()
		return nil
	}
	delete(c.overlays, uri)
	c.notifyMu.Lock()
	c.overlayMu.Unlock()
	err := c.server.DidClose(ctx, &protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	c.notifyMu.Unlock()
	return err
}

// hasOverlay reports whether a virtual document is currently open against gopls (held by
// any owner).
//
// Takes uri (protocol.DocumentURI) which addresses the virtual document.
//
// Returns bool which is true when an overlay for the URI is registered.
//
// Concurrency: safe for concurrent use; guarded by overlayMu.
func (c *Child) hasOverlay(uri protocol.DocumentURI) bool {
	c.overlayMu.Lock()
	defer c.overlayMu.Unlock()
	_, exists := c.overlays[uri]
	return exists
}

// IsOverlayAnalysed reports, without blocking, whether gopls has type-checked an open
// overlay at least once. Interactive request handlers use this to avoid querying an
// overlay that is still loading.
//
// Takes uri (protocol.DocumentURI) which addresses the virtual document.
//
// Returns bool which is true when gopls has analysed the overlay at least once.
//
// Concurrency: safe for concurrent use; reads overlay state under overlayMu and polls the
// analysed channel.
func (c *Child) IsOverlayAnalysed(uri protocol.DocumentURI) bool {
	c.overlayMu.Lock()
	state, exists := c.overlays[uri]
	c.overlayMu.Unlock()
	if !exists {
		return false
	}
	select {
	case <-state.analysed:
		return true
	default:
		return false
	}
}

// markOverlayAnalysed records that gopls has type-checked an overlay (it has published
// diagnostics for it), unblocking any waiter.
//
// Takes uri (protocol.DocumentURI) which addresses the virtual document.
//
// Concurrency: safe for concurrent use; reads overlay state under overlayMu and closes
// the analysed channel once via sync.Once.
func (c *Child) markOverlayAnalysed(uri protocol.DocumentURI) {
	c.overlayMu.Lock()
	state, exists := c.overlays[uri]
	c.overlayMu.Unlock()
	if exists {
		state.analysedOnce.Do(func() { close(state.analysed) })
	}
}

// WaitOverlayAnalysed blocks until gopls has analysed the overlay, the child dies, the
// context is cancelled, or the timeout elapses. It returns false when the child has died
// so callers skip forwarding to a dead process.
//
// Takes uri (protocol.DocumentURI) which addresses the virtual document.
// Takes timeout (time.Duration) which bounds how long to await analysis.
//
// Returns bool which is true once analysis completes or the timeout elapses, and false
// when the child has died or the context is cancelled.
//
// Concurrency: safe for concurrent use; reads overlay state under overlayMu then selects
// across the analysed, done, and context channels.
func (c *Child) WaitOverlayAnalysed(ctx context.Context, uri protocol.DocumentURI, timeout time.Duration) bool {
	c.overlayMu.Lock()
	state, exists := c.overlays[uri]
	c.overlayMu.Unlock()
	if !exists {
		return false
	}
	select {
	case <-state.analysed:
		return true
	case <-c.done:
		return false
	case <-ctx.Done():
		return false
	case <-time.After(timeout):
		_, l := logger_domain.From(ctx, log)
		l.Trace("gopls did not analyse overlay before the timeout; querying it anyway",
			logger_domain.String("uri", uri.Filename()))
		return true
	}
}
