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
	"strings"

	protocol "github.com/politepixels/golang-language-server"
	"piko.sh/piko/cmd/pikopls/internal/lsp/gopls_bridge"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// goplsDiagnosticSource labels diagnostics that originate from the gopls bridge.
	goplsDiagnosticSource = "gopls"

	// lineEndColumnSentinel is a column far beyond any real line. The LSP spec requires
	// editors to clamp a character past the line length back to the line length, so it
	// reliably extends a clamped diagnostic to the end of its line.
	lineEndColumnSentinel = 1 << 30
)

// goplsOverlayInfo records how a virtual Go overlay maps back to its .pk file so gopls
// diagnostics published against the overlay can be reverse-mapped, and how many lines the
// Go block spans so diagnostics past its end (gopls EOF markers) are not painted onto the
// closing </script> tag.
type goplsOverlayInfo struct {
	// mapper reverse-maps overlay positions back into the .pk file.
	mapper *gopls_bridge.Mapper

	// realURI is the .pk document this overlay belongs to.
	realURI protocol.DocumentURI

	// lineCount is the number of lines the Go block spans.
	lineCount int
}

// registerGoplsOverlay records the mapping from a virtual overlay URI to its .pk file,
// enabling diagnostics published by gopls for that overlay to be reverse-mapped and
// merged.
//
// Takes virtualURI (protocol.DocumentURI) which is the synthetic overlay URI gopls
// reports against.
// Takes realURI (protocol.DocumentURI) which is the .pk document the overlay belongs to.
// Takes mapper (*gopls_bridge.Mapper) which reverse-maps overlay positions into the .pk
// file.
// Takes lineCount (int) which is the number of lines in the Go block.
//
// Concurrency: acquires goplsDiagnosticsMu while mutating the goplsOverlays map.
func (w *workspace) registerGoplsOverlay(virtualURI, realURI protocol.DocumentURI, mapper *gopls_bridge.Mapper, lineCount int) {
	w.goplsDiagnosticsMu.Lock()
	defer w.goplsDiagnosticsMu.Unlock()
	if w.goplsOverlays == nil {
		w.goplsOverlays = make(map[protocol.DocumentURI]*goplsOverlayInfo)
	}
	w.goplsOverlays[virtualURI] = &goplsOverlayInfo{mapper: mapper, realURI: realURI, lineCount: lineCount}
}

// handleGoplsDiagnostics receives gopls diagnostics for a virtual overlay, windows them
// to the Go block, drops the artefacts of the import rewrite, reverse-maps the survivors
// into the .pk file, and republishes the merged set. Diagnostics for unknown URIs
// (satellite overlays of other connections) are ignored.
//
// Takes params (*protocol.PublishDiagnosticsParams) which carries the overlay URI and
// gopls diagnostics.
//
// Concurrency: acquires goplsDiagnosticsMu while reading the overlay map.
func (w *workspace) handleGoplsDiagnostics(ctx context.Context, params *protocol.PublishDiagnosticsParams) {
	w.goplsDiagnosticsMu.Lock()
	info, known := w.goplsOverlays[params.URI]
	w.goplsDiagnosticsMu.Unlock()
	if !known {
		return
	}

	mapped := make([]protocol.Diagnostic, 0, len(params.Diagnostics))
	for i := range params.Diagnostics {
		diagnostic := params.Diagnostics[i]
		if int(diagnostic.Range.Start.Line) >= info.lineCount {
			continue
		}
		if isRewriteInducedDiagnostic(diagnostic.Message) {
			continue
		}
		rng := diagnostic.Range
		if int(rng.End.Line) >= info.lineCount {
			rng.End.Line = safeconv.IntToUint32(info.lineCount - 1)
			rng.End.Character = lineEndColumnSentinel
		}
		diagnostic.Range = info.mapper.RangeToReal(rng)
		if diagnostic.Source == "" {
			diagnostic.Source = goplsDiagnosticSource
		}
		mapped = append(mapped, diagnostic)
	}

	w.setGoplsDiagnostics(info.realURI, mapped)
	w.republishGoplsDiagnostics(ctx, info.realURI)
}

// setGoplsDiagnostics stores the reverse-mapped gopls diagnostics for a .pk URI.
//
// Takes uri (protocol.DocumentURI) which is the .pk document the diagnostics belong to.
// Takes diagnostics ([]protocol.Diagnostic) which are the reverse-mapped gopls
// diagnostics to cache.
//
// Concurrency: acquires goplsDiagnosticsMu while mutating the goplsDiagnostics map.
func (w *workspace) setGoplsDiagnostics(uri protocol.DocumentURI, diagnostics []protocol.Diagnostic) {
	w.goplsDiagnosticsMu.Lock()
	defer w.goplsDiagnosticsMu.Unlock()
	if w.goplsDiagnostics == nil {
		w.goplsDiagnostics = make(map[protocol.DocumentURI][]protocol.Diagnostic)
	}
	w.goplsDiagnostics[uri] = diagnostics
}

// goplsDiagnosticsFor returns the stored gopls diagnostics for a .pk URI.
//
// Takes uri (protocol.DocumentURI) which is the .pk document to look up.
//
// Returns []protocol.Diagnostic which are the cached gopls diagnostics, nil when none are
// stored.
//
// Concurrency: acquires goplsDiagnosticsMu while reading the goplsDiagnostics map.
func (w *workspace) goplsDiagnosticsFor(uri protocol.DocumentURI) []protocol.Diagnostic {
	w.goplsDiagnosticsMu.Lock()
	defer w.goplsDiagnosticsMu.Unlock()
	return w.goplsDiagnostics[uri]
}

// invalidateGoplsDiagnostics discards the cached gopls diagnostics for a .pk URI. It is
// called when the document changes so stale Go squiggles are not republished at pre-edit
// line positions during the edit-to-reanalysis window; the next gopls publish repopulates
// them.
//
// Takes uri (protocol.DocumentURI) which is the .pk document whose cache to clear.
//
// Concurrency: acquires goplsDiagnosticsMu while mutating the goplsDiagnostics map.
func (w *workspace) invalidateGoplsDiagnostics(uri protocol.DocumentURI) {
	w.goplsDiagnosticsMu.Lock()
	defer w.goplsDiagnosticsMu.Unlock()
	delete(w.goplsDiagnostics, uri)
}

// unregisterGoplsOverlays purges the overlay records and cached gopls diagnostics for a
// closed .pk document and returns the virtual overlay URIs to close against gopls, so a
// long editor session does not leak overlays.
//
// Takes realURI (protocol.DocumentURI) which is the closed .pk document whose overlays to
// purge.
//
// Returns []protocol.DocumentURI which are the virtual overlay URIs to close against
// gopls.
//
// Concurrency: acquires goplsDiagnosticsMu while mutating the overlay and diagnostic
// maps.
func (w *workspace) unregisterGoplsOverlays(realURI protocol.DocumentURI) []protocol.DocumentURI {
	w.goplsDiagnosticsMu.Lock()
	defer w.goplsDiagnosticsMu.Unlock()

	delete(w.goplsDiagnostics, realURI)

	var virtualURIs []protocol.DocumentURI
	for virtualURI, info := range w.goplsOverlays {
		if info.realURI == realURI {
			virtualURIs = append(virtualURIs, virtualURI)
			delete(w.goplsOverlays, virtualURI)
		}
	}
	return virtualURIs
}

// unregisterAllGoplsOverlays purges every overlay record and cached gopls diagnostic for
// this connection and returns the virtual overlay URIs to close against the shared gopls
// child, used on connection teardown so orphaned overlays do not pin the child against
// reclamation.
//
// Returns []protocol.DocumentURI which are the virtual overlay URIs to close against
// gopls.
//
// Concurrency: acquires goplsDiagnosticsMu while mutating the overlay and diagnostic
// maps.
func (w *workspace) unregisterAllGoplsOverlays() []protocol.DocumentURI {
	w.goplsDiagnosticsMu.Lock()
	defer w.goplsDiagnosticsMu.Unlock()

	virtualURIs := make([]protocol.DocumentURI, 0, len(w.goplsOverlays))
	for virtualURI := range w.goplsOverlays {
		virtualURIs = append(virtualURIs, virtualURI)
	}
	w.goplsOverlays = make(map[protocol.DocumentURI]*goplsOverlayInfo)
	w.goplsDiagnostics = make(map[protocol.DocumentURI][]protocol.Diagnostic)
	return virtualURIs
}

// onGoplsChildDeath clears every cached gopls diagnostic and republishes the affected .pk
// documents, so a gopls crash does not leave stale Go errors frozen on screen until the
// user next edits. The fresh child re-analyses and republishes on the next request.
//
// Concurrency: acquires goplsDiagnosticsMu while draining the goplsDiagnostics map.
func (w *workspace) onGoplsChildDeath(_ string) {
	w.goplsDiagnosticsMu.Lock()
	affected := make([]protocol.DocumentURI, 0, len(w.goplsDiagnostics))
	for uri := range w.goplsDiagnostics {
		affected = append(affected, uri)
	}
	w.goplsDiagnostics = make(map[protocol.DocumentURI][]protocol.Diagnostic)
	w.goplsDiagnosticsMu.Unlock()

	for _, uri := range affected {
		w.republishGoplsDiagnostics(context.Background(), uri)
	}
}

// republishGoplsDiagnostics re-publishes the merged diagnostic set for a .pk URI after
// its gopls diagnostics change, reusing the open document when available.
//
// Takes uri (protocol.DocumentURI) which is the .pk document to republish diagnostics
// for.
func (w *workspace) republishGoplsDiagnostics(ctx context.Context, uri protocol.DocumentURI) {
	document, exists := w.GetDocument(uri)
	if !exists || document.ProjectResult == nil {
		return
	}
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("republishing diagnostics with gopls results", logger_domain.String(keyURI, uri.Filename()))
	w.publishDiagnostics(ctx, uri, document)
}

// isRewriteInducedDiagnostic reports whether a gopls message is a synthetic-overlay
// artefact.
//
// Takes message (string) which is the gopls diagnostic message to classify.
//
// Returns true when the message references the synthetic overlay marker.
func isRewriteInducedDiagnostic(message string) bool {
	return strings.Contains(message, gopls_bridge.OverlayPathMarker)
}
