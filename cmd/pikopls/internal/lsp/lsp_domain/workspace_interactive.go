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
	"time"

	protocol "github.com/politepixels/golang-language-server"
)

const (
	// interactiveAnalysisBudget caps how long a read-only intelligence request waits.
	//
	// Read-only intelligence requests (hover, definition, document colour, folding,
	// signature help, and similar) wait for analysis before they fall back to the last
	// analysed document. Interactive handlers run on the jsonrpc read path, so an unbounded
	// wait there freezes the whole pipe; this bounds that to a short interval and lets the
	// authoritative background analysis refresh diagnostics.
	interactiveAnalysisBudget = 2 * time.Second
)

var (
	// errInteractiveBudgetExceeded is the cause attached to a bounded interactive analysis
	// so a budget exhaustion is distinguishable from a client cancellation when the context
	// cause is inspected downstream.
	errInteractiveBudgetExceeded = errors.New("interactive analysis budget exceeded")
)

// RunAnalysisForInteractiveRequest returns the best available document for a read-only
// request.
//
// A read-only intelligence request never freezes the jsonrpc pipe. A clean cached result
// is returned immediately. If a background analysis is already running for the URI it
// awaits completion of that analysis only within interactiveAnalysisBudget rather than
// starting a competing build that would supersede (and strand) the authoritative one.
// Otherwise it runs a strictly time-bounded analysis. On any budget exhaustion or
// cancellation it falls back to the last analysed document so a feature degrades to
// slightly-stale data instead of a hang; the background analysis from the originating
// edit remains responsible for committing fresh results and publishing diagnostics.
//
// Takes uri (protocol.DocumentURI) which identifies the document to analyse.
//
// Returns the best available document and any non-fallback error.
//
// Safe for concurrent use.
func (w *workspace) RunAnalysisForInteractiveRequest(ctx context.Context, uri protocol.DocumentURI) (*document, error) {
	if document := w.getCachedCleanDocument(ctx, uri); document != nil {
		return document, nil
	}

	boundedCtx, cancel := context.WithTimeoutCause(ctx, interactiveAnalysisBudget, errInteractiveBudgetExceeded)
	defer cancel()

	w.mu.RLock()
	doneChan, inFlight := w.analysisDone[uri]
	w.mu.RUnlock()

	if inFlight {
		select {
		case <-doneChan:
		case <-boundedCtx.Done():
		}
		if document := w.getCachedCleanDocument(ctx, uri); document != nil {
			return document, nil
		}
		if last := w.lastAnalysedDocument(uri); last != nil {
			return last, nil
		}
		return nil, boundedCtx.Err()
	}

	document, err := w.runAnalysisForGeneration(boundedCtx, uri, w.currentGeneration(uri))
	if document != nil {
		return document, err
	}
	if last := w.lastAnalysedDocument(uri); last != nil {
		return last, nil
	}
	return document, err
}

// lastAnalysedDocument returns the most recently committed analysed document for uri.
//
// The document may be marked dirty by a later edit. It lets an interactive request
// degrade to last-known-good rather than block.
//
// Takes uri (protocol.DocumentURI) which identifies the document to look up.
//
// Returns the last analysed document, or nil when none has been analysed.
//
// Concurrency: acquires mu (read lock) while reading the documents map.
func (w *workspace) lastAnalysedDocument(uri protocol.DocumentURI) *document {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.documents[uri]
}
