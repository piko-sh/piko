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
	"slices"
	"strconv"
	"strings"
	"time"

	protocol "github.com/politepixels/golang-language-server"
	"piko.sh/piko/cmd/lsp/internal/lsp/gopls_bridge"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/sfcparser"
)

const (
	// goplsOverlayAnalysisTimeout bounds how long a request waits for gopls to type-check a
	// freshly opened overlay before forwarding anyway. It is generous because the first
	// request on a cold gopls must wait for the whole module to load; once warm, the
	// analysed signal fires in well under a second.
	goplsOverlayAnalysisTimeout = 20 * time.Second

	// goplsRequestTimeout bounds a single forwarded interactive request. A wedged or crashed
	// gopls must never hang the editor: on timeout (or the child dying) the forward is
	// abandoned and the caller falls back to piko-only intelligence.
	goplsRequestTimeout = 3 * time.Second

	// maxSatelliteOverlays caps the sibling-component overlays opened per gopls child.
	//
	// The cap bounds cross-component resolution so a project with very many .pk components
	// cannot push an unbounded overlay set into gopls. The default is high enough to be out
	// of the way for any realistic project; past it, cross-component resolution degrades
	// rather than exhausting memory.
	maxSatelliteOverlays = 512
)

var (
	// errGoplsRequestDeadline is the cause attached to a forwarded gopls request's bounded
	// context, so a request abandoned because gopls was too slow is distinguishable from one
	// abandoned because the editor cancelled it.
	errGoplsRequestDeadline = errors.New("gopls request deadline exceeded")
)

// goplsRequest holds the prepared state for forwarding one request to a gopls child for a
// position inside a .pk Go block.
type goplsRequest struct {
	// child is the gopls child the request is forwarded to.
	child *gopls_bridge.Child

	// mapper translates positions between the .pk file and the virtual Go document.
	mapper *gopls_bridge.Mapper

	// satellites holds the sibling-component overlay URIs whose results are dropped.
	satellites map[protocol.DocumentURI]struct{}

	// virtualURI is the synthetic overlay URI gopls sees for this Go block.
	virtualURI protocol.DocumentURI

	// mappedPosition is the request position translated into virtual coordinates.
	mappedPosition protocol.Position
}

// goplsSatelliteState caches one module root's satellite overlays against the child they
// opened on.
type goplsSatelliteState struct {
	// child is the gopls child the cached satellites were opened against.
	child *gopls_bridge.Child

	// module is the VirtualModule generation the satellites were derived from.
	module *annotator_dto.VirtualModule

	// satellites holds the synthetic satellite overlay URIs for this module root.
	satellites map[protocol.DocumentURI]struct{}
}

// callGoplsResult carries a forwarded gopls call's outcome across the goroutine boundary
// used to enforce the request deadline.
type callGoplsResult[T any] struct {
	// value is the result the forwarded call produced.
	value T

	// err is the error the forwarded call returned, if any.
	err error
}

// goplsChildSignal exposes the death signal of a gopls child. The forwarding path depends
// only on this, so callGopls can be exercised without a live child.
type goplsChildSignal interface {
	// Done returns a channel that is closed when the gopls child dies.
	Done() <-chan struct{}
}

// callGopls runs a forwarded gopls RPC under a bounded deadline that aborts when the
// child dies.
//
// Takes child (goplsChildSignal) which exposes the death signal aborting the call.
// Takes call (func(context.Context) (T, error)) which performs the forwarded RPC.
//
// Returns T which is the call's result, valid only when the bool is true.
// Returns bool which is false on timeout, child death, RPC error, or panic.
//
// Concurrency: runs call on a goroutine and selects over a buffered result channel, the
// child's death channel, and the bounded context's Done channel.
func callGopls[T any](ctx context.Context, child goplsChildSignal, call func(context.Context) (T, error)) (T, bool) {
	ctx, l := logger_domain.From(ctx, log)
	callCtx, cancel := context.WithTimeoutCause(ctx, goplsRequestTimeout,
		fmt.Errorf("gopls request exceeded %s: %w", goplsRequestTimeout, errGoplsRequestDeadline))
	defer cancel()

	resultChan := make(chan callGoplsResult[T], 1)
	go func() {
		value, err := goroutine.SafeCall1(callCtx, "gopls_bridge.facade.callGopls", func() (T, error) {
			return call(callCtx)
		})
		resultChan <- callGoplsResult[T]{value: value, err: err}
	}()

	var zero T
	select {
	case result := <-resultChan:
		if result.err != nil {
			l.Trace("gopls bridge: forwarded request failed", logger_domain.Error(result.err))
			return zero, false
		}
		return result.value, true
	case <-child.Done():

		l.Trace("gopls bridge: forwarded request abandoned, gopls child died")
		return zero, false
	case <-callCtx.Done():
		l.Trace("gopls bridge: forwarded request deadline reached or cancelled",
			logger_domain.Error(context.Cause(callCtx)))
		return zero, false
	}
}

// goplsBridgeActive reports whether the Go-block bridge should handle requests for this
// connection.
//
// Returns bool which is true when the bridge is enabled and gopls is available.
//
// Concurrency: reads goplsBridgeEnabled under s.mu, then queries the manager outside the
// lock.
func (s *Server) goplsBridgeActive() bool {
	s.mu.Lock()
	enabled := s.goplsBridgeEnabled
	s.mu.Unlock()
	return enabled && s.goplsManager != nil && s.goplsManager.Enabled()
}

// applyGoplsInitOptions lets a client opt in or out of the Go-block bridge per connection
// via initializationOptions.goBridge, overriding the process-level --gopls-bridge
// default. VS Code sets it true; IDEA leaves it unset so the bridge stays off and its
// native Go injection wins.
//
// Takes params (*protocol.InitializeParams) which may carry
// initializationOptions.goBridge.
func (s *Server) applyGoplsInitOptions(params *protocol.InitializeParams) {
	options, ok := params.InitializationOptions.(map[string]any)
	if !ok {
		return
	}
	if enabled, ok := options["goBridge"].(bool); ok {
		s.goplsBridgeEnabled = enabled
	}
}

// goScriptAtPosition returns the Go script block that contains the position.
//
// Takes position (protocol.Position) which is the cursor location to locate.
//
// Returns the Go script block containing the position, and a bool that is false when no
// Go block covers it.
func (d *document) goScriptAtPosition(position protocol.Position) (*sfcparser.Script, bool) {
	sfc := d.getSFCResult()
	if sfc == nil {
		return nil, false
	}
	for i := range sfc.Scripts {
		script := &sfc.Scripts[i]
		if script.IsGo() && d.isPositionInScriptContent(script, position) {
			return script, true
		}
	}
	return nil, false
}

// prepareGoplsRequest builds and syncs the virtual Go overlay for the Go block under the
// cursor and returns the state needed to forward a request to gopls.
//
// Takes document (*document) which holds the .pk content under the cursor.
// Takes position (protocol.Position) which is the cursor location to serve.
//
// Returns the prepared request state, and a bool that is false when the bridge cannot
// serve this position and the caller should fall back to piko.
func (s *Server) prepareGoplsRequest(ctx context.Context, document *document, position protocol.Position) (*goplsRequest, bool) {
	if !s.goplsBridgeActive() {
		return nil, false
	}
	script, ok := document.goScriptAtPosition(position)
	if !ok {
		return nil, false
	}
	overlay, ok := s.prepareGoplsOverlay(ctx, document, script, false)
	if !ok {
		return nil, false
	}
	return &goplsRequest{
		child:          overlay.child,
		mapper:         overlay.mapper,
		satellites:     overlay.satellites,
		virtualURI:     overlay.virtualURI,
		mappedPosition: overlay.mapper.ToVirtual(position),
	}, true
}

// goplsOverlayPrep is the shared state of a synced virtual Go overlay, used by both
// per-request forwarding and the proactive document-sync mirror.
type goplsOverlayPrep struct {
	// child is the gopls child the overlay was opened against.
	child *gopls_bridge.Child

	// mapper translates positions between the .pk file and the virtual Go document.
	mapper *gopls_bridge.Mapper

	// satellites holds the sibling-component overlay URIs whose results are dropped.
	satellites map[protocol.DocumentURI]struct{}

	// virtualURI is the synthetic overlay URI gopls sees for this Go block.
	virtualURI protocol.DocumentURI
}

// prepareGoplsOverlay opens the virtual Go overlay for a .pk Go block against gopls.
//
// It generates the virtual Go document for the block, opens its satellite and primary
// overlays against the module's gopls child, and registers the overlay for diagnostics
// reverse-mapping. When wait is true (the background warm path) it blocks until gopls is
// ready and the overlay is analysed; when false (an interactive request) it never blocks
// and returns false if gopls is not yet warm, so the request falls back to piko rather
// than freezing the editor.
//
// Takes document (*document) which holds the .pk content for the block.
// Takes script (*sfcparser.Script) which is the Go block to mirror into gopls.
// Takes wait (bool) which is true to block until gopls has analysed the overlay.
//
// Returns the synced overlay state, and a bool that is false when the bridge cannot serve
// the document.
//
// Concurrency: reads moduleRoot and goplsConnectionID under s.mu, then performs gopls I/O
// outside the lock.
func (s *Server) prepareGoplsOverlay(ctx context.Context, document *document, script *sfcparser.Script, wait bool) (*goplsOverlayPrep, bool) {
	ctx, l := logger_domain.From(ctx, log)

	virtualModule := documentVirtualModule(document)
	if virtualModule == nil {
		l.Trace("gopls bridge: no virtual module available")
		return nil, false
	}

	s.mu.Lock()
	moduleRoot := s.moduleRoot
	owner := s.goplsConnectionID
	s.mu.Unlock()
	if moduleRoot == "" {
		return nil, false
	}

	primary, ok := lookupPrimaryComponent(virtualModule, document.URI.Filename())
	if !ok {
		l.Trace("gopls bridge: primary component not found", logger_domain.String("path", document.URI.Filename()))
		return nil, false
	}

	child, acquireErr := s.goplsManager.Acquire(ctx, moduleRoot)
	if acquireErr != nil {
		return nil, false
	}
	if wait {
		if !child.WaitReady(ctx) {
			return nil, false
		}
	} else if !child.IsReady() {
		return nil, false
	}

	satellites := s.syncSatellitesIfStale(ctx, child, owner, moduleRoot, virtualModule, primary.VirtualGoFilePath)

	virtualDoc := gopls_bridge.BuildVirtualDoc(document.URI, gopls_bridge.VirtualDocInput{
		AliasToCanonical: buildAliasToCanonical(virtualModule, primary),
		ModuleRoot:       moduleRoot,
		HashedName:       primary.HashedName,
		BlockContent:     script.Content,
		ContentLine:      script.ContentLocation.Line,
		ContentColumn:    firstContentLineUTF16Column(document.Content, script.ContentLocation),
	})
	virtualURI := virtualDoc.Mapper.VirtualURI()
	blockLineCount := strings.Count(script.Content, "\n") + 1
	s.workspace.registerGoplsOverlay(virtualURI, document.URI, virtualDoc.Mapper, blockLineCount)

	if syncErr := child.SyncOverlay(ctx, owner, virtualURI, virtualDoc.Content); syncErr != nil {
		l.Trace("gopls bridge: overlay sync failed", logger_domain.Error(syncErr))
		return nil, false
	}
	if wait {
		child.WaitOverlayAnalysed(ctx, virtualURI, goplsOverlayAnalysisTimeout)
	} else if !child.IsOverlayAnalysed(virtualURI) {
		return nil, false
	}

	return &goplsOverlayPrep{child: child, mapper: virtualDoc.Mapper, satellites: satellites, virtualURI: virtualURI}, true
}

// syncSatellitesIfStale refreshes the sibling-component overlays when analysis has
// advanced.
//
// Takes child (*gopls_bridge.Child) which the satellites are opened against.
// Takes owner (uint64) which is the connection identifier holding the overlays.
// Takes moduleRoot (string) which keys the cached satellite state.
// Takes virtualModule (*annotator_dto.VirtualModule) which supplies the components.
// Takes primaryVirtualPath (string) which is the block's own path to exclude.
//
// Returns the current satellite overlay URI set for this module.
//
// Concurrency: reads and writes s.goplsSatellites under s.mu, performing gopls I/O
// outside the lock.
func (s *Server) syncSatellitesIfStale(
	ctx context.Context,
	child *gopls_bridge.Child,
	owner uint64,
	moduleRoot string,
	virtualModule *annotator_dto.VirtualModule,
	primaryVirtualPath string,
) map[protocol.DocumentURI]struct{} {
	s.mu.Lock()
	cached, ok := s.goplsSatellites[moduleRoot]
	if ok && cached.module == virtualModule && cached.child == child {
		satellites := cached.satellites
		s.mu.Unlock()
		return satellites
	}
	s.mu.Unlock()

	satellites := satelliteOverlayURIs(virtualModule, primaryVirtualPath)
	syncSatelliteOverlays(ctx, child, owner, virtualModule, primaryVirtualPath)

	s.mu.Lock()
	if s.goplsSatellites == nil {
		s.goplsSatellites = make(map[string]*goplsSatelliteState)
	}

	var dropped []protocol.DocumentURI
	if prior, exists := s.goplsSatellites[moduleRoot]; exists && prior.child == child {
		for satelliteURI := range prior.satellites {
			if _, kept := satellites[satelliteURI]; !kept {
				dropped = append(dropped, satelliteURI)
			}
		}
	}
	s.goplsSatellites[moduleRoot] = &goplsSatelliteState{child: child, module: virtualModule, satellites: satellites}
	s.mu.Unlock()

	for _, satelliteURI := range dropped {
		_ = child.CloseOverlay(ctx, owner, satelliteURI)
	}
	return satellites
}

// takeGoplsSatelliteURIs snapshots and clears every cached satellite overlay URI for this
// connection.
//
// Returns the snapshot of cached satellite overlay URIs.
//
// Concurrency: reads and clears s.goplsSatellites under s.mu.
func (s *Server) takeGoplsSatelliteURIs() []protocol.DocumentURI {
	s.mu.Lock()
	defer s.mu.Unlock()
	var satelliteURIs []protocol.DocumentURI
	for _, state := range s.goplsSatellites {
		for satelliteURI := range state.satellites {
			satelliteURIs = append(satelliteURIs, satelliteURI)
		}
	}
	s.goplsSatellites = nil
	return satelliteURIs
}

// closeGoplsOverlaysForDocument releases the gopls overlays for a closed .pk document.
//
// Takes uri (protocol.DocumentURI) which is the closed .pk document to release.
//
// Concurrency: briefly acquires s.mu to read the module root and connection id; safe to
// run on a background goroutine, as CloseOverlay performs the gopls I/O.
func (s *Server) closeGoplsOverlaysForDocument(ctx context.Context, uri protocol.DocumentURI) {
	if !s.goplsBridgeActive() {
		return
	}

	virtualURIs := s.workspace.unregisterGoplsOverlays(uri)
	if len(virtualURIs) == 0 {
		return
	}

	s.mu.Lock()
	moduleRoot := s.moduleRoot
	owner := s.goplsConnectionID
	s.mu.Unlock()
	if moduleRoot == "" {
		return
	}

	child, ok := s.goplsManager.Existing(moduleRoot)
	if !ok {
		return
	}
	for _, virtualURI := range virtualURIs {
		_ = child.CloseOverlay(ctx, owner, virtualURI)
	}
}

// closeGoplsOverlaysForURIs releases, on a background goroutine, the gopls overlays of a
// batch of .pk documents that were renamed away or deleted, so a non-conforming client
// that skips didClose does not leak overlays. It is a no-op when the bridge is off or
// none of the URIs had overlays.
//
// Takes uris ([]protocol.DocumentURI) which are the documents whose overlays close.
func (s *Server) closeGoplsOverlaysForURIs(uris []protocol.DocumentURI) {
	if !s.goplsBridgeActive() || len(uris) == 0 {
		return
	}
	s.goBackground(func(ctx context.Context) {
		for _, uri := range uris {
			s.closeGoplsOverlaysForDocument(ctx, uri)
		}
	})
}

// releaseGoplsOverlays closes every overlay this connection holds against the shared
// gopls child.
//
// Connection teardown must not pin a child with orphaned overlays the idle reaper would
// then refuse to reclaim. It looks up an existing child only (never spawns one) and is a
// no-op when the bridge is off.
//
// Concurrency: reads moduleRoot, goplsConnectionID, and workspace under s.mu, then
// performs gopls I/O outside the lock.
func (s *Server) releaseGoplsOverlays(ctx context.Context) {
	if s.goplsManager == nil {
		return
	}
	s.mu.Lock()
	moduleRoot := s.moduleRoot
	owner := s.goplsConnectionID
	workspace := s.workspace
	s.mu.Unlock()
	if moduleRoot == "" || workspace == nil {
		return
	}

	virtualURIs := workspace.unregisterAllGoplsOverlays()
	virtualURIs = append(virtualURIs, s.takeGoplsSatelliteURIs()...)
	if len(virtualURIs) == 0 {
		return
	}
	child, ok := s.goplsManager.Existing(moduleRoot)
	if !ok {
		return
	}
	for _, virtualURI := range virtualURIs {
		_ = child.CloseOverlay(ctx, owner, virtualURI)
	}
}

// syncGoplsOverlayForDocument opens or refreshes the Go-block overlay for a .pk document.
//
// It keeps gopls producing diagnostics proactively on open and change. It is a no-op when
// the bridge is off or the document has no Go block, and is safe to run on a background
// goroutine (it may block on gopls analysis).
//
// Takes document (*document) which holds the .pk content to mirror into gopls.
func (s *Server) syncGoplsOverlayForDocument(ctx context.Context, document *document) {
	if !s.goplsBridgeActive() {
		return
	}
	sfc := document.getSFCResult()
	if sfc == nil {
		return
	}
	script, ok := sfc.GoScript()
	if !ok {
		return
	}
	_, _ = s.prepareGoplsOverlay(ctx, document, script, true)
}

// goplsHover forwards a hover request to gopls for a Go-block position and maps the
// result range back into the .pk file.
//
// Takes document (*document) which holds the .pk content under the cursor.
// Takes position (protocol.Position) which is the cursor location to hover.
//
// Returns the mapped hover, and a bool that is false when the bridge produced no result
// and the caller should fall back to piko.
func (s *Server) goplsHover(ctx context.Context, document *document, position protocol.Position) (*protocol.Hover, bool) {
	ctx, l := logger_domain.From(ctx, log)

	request, ok := s.prepareGoplsRequest(ctx, document, position)
	if !ok {
		return nil, false
	}

	hover, ok := callGopls(ctx, request.child, func(callCtx context.Context) (*protocol.Hover, error) {
		return request.child.Server().Hover(callCtx, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: request.virtualURI},
				Position:     request.mappedPosition,
			},
		})
	})
	if !ok {
		l.Trace("gopls bridge: hover unavailable")
		return nil, false
	}
	if hover == nil || hover.Contents.Value == "" {
		l.Trace("gopls bridge: hover empty")
		return nil, false
	}
	l.Trace("gopls bridge: hover hit", logger_domain.Int("contentLength", len(hover.Contents.Value)))

	if hover.Range != nil {
		mapped := request.mapper.RangeToReal(*hover.Range)
		hover.Range = &mapped
	}
	return hover, true
}

// virtualPositionParams builds the gopls text-document position params for the virtual
// document at the mapped position.
//
// Returns the position params targeting the virtual URI at the mapped position.
func (r *goplsRequest) virtualPositionParams() protocol.TextDocumentPositionParams {
	return protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: r.virtualURI},
		Position:     r.mappedPosition,
	}
}

// mapLocations maps gopls locations back into .pk coordinates: locations in the primary
// overlay are rewritten to the .pk file, real dependency files pass through unchanged,
// and locations in other components' synthetic overlays are dropped (cross-.pk navigation
// is handled by a later pass).
//
// Takes locations ([]protocol.Location) which are the gopls results to map.
//
// Returns the locations rewritten into .pk coordinates, with synthetic ones dropped.
func (r *goplsRequest) mapLocations(locations []protocol.Location) []protocol.Location {
	mapped := make([]protocol.Location, 0, len(locations))
	for _, location := range locations {
		switch {
		case location.URI == r.virtualURI:
			location.URI = r.mapper.RealURI()
			location.Range = r.mapper.RangeToReal(location.Range)
			mapped = append(mapped, location)
		case r.isDroppableSynthetic(location.URI):
			continue
		default:
			mapped = append(mapped, location)
		}
	}
	return mapped
}

// isDroppableSynthetic reports whether a gopls result URI points at a synthetic overlay
// to hide.
//
// Takes uri (protocol.DocumentURI) which is the gopls result URI to classify.
//
// Returns true when the URI is a synthetic overlay the editor must never see.
func (r *goplsRequest) isDroppableSynthetic(uri protocol.DocumentURI) bool {
	if uri == r.virtualURI {
		return false
	}
	if _, ok := r.satellites[uri]; ok {
		return true
	}
	return strings.Contains(string(uri), gopls_bridge.OverlayPathMarker)
}

// goplsLocationCall issues a location-returning LSP request against a gopls server using
// already-mapped virtual coordinates, under the bridge's bounded request context.
type goplsLocationCall func(ctx context.Context, server protocol.Server, params protocol.TextDocumentPositionParams) ([]protocol.Location, error)

// goplsLocations forwards a location-returning request to gopls (under a bounded
// deadline) and maps the results back into .pk coordinates.
//
// Takes document (*document) which holds the .pk content under the cursor.
// Takes position (protocol.Position) which is the cursor location to serve.
// Takes call (goplsLocationCall) which issues the specific gopls location request.
//
// Returns the mapped locations, and a bool that is false when the bridge produced no
// usable result.
func (s *Server) goplsLocations(ctx context.Context, document *document, position protocol.Position, call goplsLocationCall) ([]protocol.Location, bool) {
	ctx, l := logger_domain.From(ctx, log)

	request, ok := s.prepareGoplsRequest(ctx, document, position)
	if !ok {
		return nil, false
	}
	locations, ok := callGopls(ctx, request.child, func(callCtx context.Context) ([]protocol.Location, error) {
		return call(callCtx, request.child.Server(), request.virtualPositionParams())
	})
	if !ok {
		l.Trace("gopls bridge: location request unavailable")
		return nil, false
	}
	if len(locations) == 0 {
		l.Trace("gopls bridge: no locations")
		return nil, false
	}
	mapped := request.mapLocations(locations)
	if len(mapped) == 0 {
		return nil, false
	}
	return mapped, true
}

// goplsDefinition forwards go-to-definition to gopls for a Go-block position.
//
// Takes document (*document) which holds the .pk content under the cursor.
// Takes position (protocol.Position) which is the cursor location to resolve.
//
// Returns the mapped definition locations, and a bool that is false on no result.
func (s *Server) goplsDefinition(ctx context.Context, document *document, position protocol.Position) ([]protocol.Location, bool) {
	return s.goplsLocations(ctx, document, position, func(callCtx context.Context, server protocol.Server, params protocol.TextDocumentPositionParams) ([]protocol.Location, error) {
		return server.Definition(callCtx, &protocol.DefinitionParams{TextDocumentPositionParams: params})
	})
}

// goplsTypeDefinition forwards go-to-type-definition to gopls.
//
// Takes document (*document) which holds the .pk content under the cursor.
// Takes position (protocol.Position) which is the cursor location to resolve.
//
// Returns the mapped type-definition locations, and a bool that is false on no result.
func (s *Server) goplsTypeDefinition(ctx context.Context, document *document, position protocol.Position) ([]protocol.Location, bool) {
	return s.goplsLocations(ctx, document, position, func(callCtx context.Context, server protocol.Server, params protocol.TextDocumentPositionParams) ([]protocol.Location, error) {
		return server.TypeDefinition(callCtx, &protocol.TypeDefinitionParams{TextDocumentPositionParams: params})
	})
}

// goplsImplementation forwards go-to-implementation to gopls.
//
// Takes document (*document) which holds the .pk content under the cursor.
// Takes position (protocol.Position) which is the cursor location to resolve.
//
// Returns the mapped implementation locations, and a bool that is false on no result.
func (s *Server) goplsImplementation(ctx context.Context, document *document, position protocol.Position) ([]protocol.Location, bool) {
	return s.goplsLocations(ctx, document, position, func(callCtx context.Context, server protocol.Server, params protocol.TextDocumentPositionParams) ([]protocol.Location, error) {
		return server.Implementation(callCtx, &protocol.ImplementationParams{TextDocumentPositionParams: params})
	})
}

// goplsReferences forwards find-references to gopls.
//
// Takes document (*document) which holds the .pk content under the cursor.
// Takes position (protocol.Position) which is the cursor location to resolve.
//
// Returns the mapped reference locations, and a bool that is false on no result.
func (s *Server) goplsReferences(ctx context.Context, document *document, position protocol.Position) ([]protocol.Location, bool) {
	return s.goplsLocations(ctx, document, position, func(callCtx context.Context, server protocol.Server, params protocol.TextDocumentPositionParams) ([]protocol.Location, error) {
		return server.References(callCtx, &protocol.ReferenceParams{
			TextDocumentPositionParams: params,
			Context:                    protocol.ReferenceContext{IncludeDeclaration: true},
		})
	})
}

// goplsCompletion forwards completion to gopls and maps every edit range back into .pk
// coordinates so auto-imports and insertions land correctly.
//
// Takes document (*document) which holds the .pk content under the cursor.
// Takes position (protocol.Position) which is the cursor location to complete.
//
// Returns the completion list with ranges mapped, and a bool that is false on no result.
func (s *Server) goplsCompletion(ctx context.Context, document *document, position protocol.Position) (*protocol.CompletionList, bool) {
	ctx, l := logger_domain.From(ctx, log)

	request, ok := s.prepareGoplsRequest(ctx, document, position)
	if !ok {
		return nil, false
	}
	list, ok := callGopls(ctx, request.child, func(callCtx context.Context) (*protocol.CompletionList, error) {
		return request.child.Server().Completion(callCtx, &protocol.CompletionParams{
			TextDocumentPositionParams: request.virtualPositionParams(),
			Context:                    &protocol.CompletionContext{TriggerKind: protocol.CompletionTriggerKindInvoked},
		})
	})
	if !ok {
		l.Trace("gopls bridge: completion unavailable")
		return nil, false
	}
	if list == nil || len(list.Items) == 0 {
		l.Trace("gopls bridge: completion empty")
		return nil, false
	}
	l.Trace("gopls bridge: completion hit", logger_domain.Int("items", len(list.Items)))
	for i := range list.Items {
		request.mapCompletionItem(&list.Items[i])
	}
	return list, true
}

// mapCompletionItem maps a completion item's edit ranges from the virtual document back
// into the .pk file.
//
// Takes item (*protocol.CompletionItem) whose edit ranges are rewritten in place.
func (r *goplsRequest) mapCompletionItem(item *protocol.CompletionItem) {
	if item.TextEdit != nil {
		item.TextEdit.Range = r.mapper.RangeToReal(item.TextEdit.Range)
	}
	for j := range item.AdditionalTextEdits {
		item.AdditionalTextEdits[j].Range = r.mapper.RangeToReal(item.AdditionalTextEdits[j].Range)
	}
}

// goplsSignatureHelp forwards signature help to gopls (no document ranges to map).
//
// Takes document (*document) which holds the .pk content under the cursor.
// Takes position (protocol.Position) which is the cursor location to serve.
//
// Returns the signature help, and a bool that is false on no result.
func (s *Server) goplsSignatureHelp(ctx context.Context, document *document, position protocol.Position) (*protocol.SignatureHelp, bool) {
	request, ok := s.prepareGoplsRequest(ctx, document, position)
	if !ok {
		return nil, false
	}
	help, ok := callGopls(ctx, request.child, func(callCtx context.Context) (*protocol.SignatureHelp, error) {
		return request.child.Server().SignatureHelp(callCtx, &protocol.SignatureHelpParams{
			TextDocumentPositionParams: request.virtualPositionParams(),
		})
	})
	if !ok || help == nil || len(help.Signatures) == 0 {
		return nil, false
	}
	return help, true
}

// goplsDocumentHighlight forwards document highlight to gopls and maps each highlight
// range back into the .pk file.
//
// Takes document (*document) which holds the .pk content under the cursor.
// Takes position (protocol.Position) which is the cursor location to highlight.
//
// Returns the highlights with ranges mapped, and a bool that is false on no result.
func (s *Server) goplsDocumentHighlight(ctx context.Context, document *document, position protocol.Position) ([]protocol.DocumentHighlight, bool) {
	request, ok := s.prepareGoplsRequest(ctx, document, position)
	if !ok {
		return nil, false
	}
	highlights, ok := callGopls(ctx, request.child, func(callCtx context.Context) ([]protocol.DocumentHighlight, error) {
		return request.child.Server().DocumentHighlight(callCtx, &protocol.DocumentHighlightParams{
			TextDocumentPositionParams: request.virtualPositionParams(),
		})
	})
	if !ok || len(highlights) == 0 {
		return nil, false
	}
	for i := range highlights {
		highlights[i].Range = request.mapper.RangeToReal(highlights[i].Range)
	}
	return highlights, true
}

// goplsPrepareRename forwards prepare-rename to gopls and maps the returned range back
// into the .pk file.
//
// Takes document (*document) which holds the .pk content under the cursor.
// Takes position (protocol.Position) which is the symbol location to rename.
//
// Returns the mapped rename range, and a bool that is false on no result.
func (s *Server) goplsPrepareRename(ctx context.Context, document *document, position protocol.Position) (*protocol.Range, bool) {
	request, ok := s.prepareGoplsRequest(ctx, document, position)
	if !ok {
		return nil, false
	}
	rng, ok := callGopls(ctx, request.child, func(callCtx context.Context) (*protocol.Range, error) {
		return request.child.Server().PrepareRename(callCtx, &protocol.PrepareRenameParams{
			TextDocumentPositionParams: request.virtualPositionParams(),
		})
	})
	if !ok || rng == nil {
		return nil, false
	}
	mapped := request.mapper.RangeToReal(*rng)
	return &mapped, true
}

// goplsRename forwards rename to gopls and remaps the resulting workspace edit back into
// .pk coordinates.
//
// Takes document (*document) which holds the .pk content under the cursor.
// Takes position (protocol.Position) which is the symbol location to rename.
// Takes newName (string) which is the replacement identifier.
//
// Returns the remapped workspace edit, and a bool that is false on no result.
func (s *Server) goplsRename(ctx context.Context, document *document, position protocol.Position, newName string) (*protocol.WorkspaceEdit, bool) {
	request, ok := s.prepareGoplsRequest(ctx, document, position)
	if !ok {
		return nil, false
	}
	edit, ok := callGopls(ctx, request.child, func(callCtx context.Context) (*protocol.WorkspaceEdit, error) {
		return request.child.Server().Rename(callCtx, &protocol.RenameParams{
			TextDocumentPositionParams: request.virtualPositionParams(),
			NewName:                    newName,
		})
	})
	if !ok || edit == nil {
		return nil, false
	}
	return request.remapWorkspaceEdit(edit), true
}

// goplsCodeAction forwards code actions to gopls for a Go-block range and remaps every
// resulting workspace edit back into .pk coordinates.
//
// Takes document (*document) which holds the .pk content under the cursor.
// Takes codeRange (protocol.Range) which is the .pk range to request actions for.
//
// Returns the code actions with edits remapped, and a bool that is false on no result.
func (s *Server) goplsCodeAction(ctx context.Context, document *document, codeRange protocol.Range) ([]protocol.CodeAction, bool) {
	request, ok := s.prepareGoplsRequest(ctx, document, codeRange.Start)
	if !ok {
		return nil, false
	}
	actions, ok := callGopls(ctx, request.child, func(callCtx context.Context) ([]protocol.CodeAction, error) {
		return request.child.Server().CodeAction(callCtx, &protocol.CodeActionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: request.virtualURI},
			Range:        request.mapper.RangeToVirtual(codeRange),
		})
	})
	if !ok || len(actions) == 0 {
		return nil, false
	}
	for i := range actions {
		if actions[i].Edit != nil {
			actions[i].Edit = request.remapWorkspaceEdit(actions[i].Edit)
		}
	}
	return actions, true
}

// mergeTemplateRenameEdits augments a gopls Go-block rename with template-side
// references.
//
// Takes document (*document) which holds the .pk content being renamed.
// Takes position (protocol.Position) which is the symbol location to rename.
// Takes newName (string) which is the replacement identifier.
// Takes goplsEdit (*protocol.WorkspaceEdit) which is the gopls rename to augment.
//
// Returns the augmented workspace edit, or goplsEdit unchanged on any uncertainty.
func (s *Server) mergeTemplateRenameEdits(ctx context.Context, document *document, position protocol.Position, newName string, goplsEdit *protocol.WorkspaceEdit) *protocol.WorkspaceEdit {
	if goplsEdit == nil {
		return nil
	}
	script, ok := document.goScriptAtPosition(position)
	if !ok {
		return goplsEdit
	}
	blockStartLine := script.ContentLocation.Line - 1
	blockEndLine := blockStartLine + strings.Count(script.Content, "\n")

	locations, err := s.workspace.FindAllReferences(context.WithoutCancel(ctx), document.URI, position)
	if err != nil || len(locations) == 0 {
		return goplsEdit
	}
	return mergeTemplateReferences(goplsEdit, document.URI, blockStartLine, blockEndLine, locations, newName)
}

// mergeTemplateReferences is the data-loss-sensitive core of the template rename merge.
//
// Takes goplsEdit (*protocol.WorkspaceEdit) which the template edits are appended to.
// Takes docURI (protocol.DocumentURI) which is the .pk file the references must be in.
// Takes blockStartLine (int) which is the first line of the Go block to skip.
// Takes blockEndLine (int) which is the last line of the Go block to skip.
// Takes locations ([]protocol.Location) which are piko's template references.
// Takes newName (string) which is the replacement identifier.
//
// Returns goplsEdit with surviving template references appended.
func mergeTemplateReferences(goplsEdit *protocol.WorkspaceEdit, docURI protocol.DocumentURI, blockStartLine, blockEndLine int, locations []protocol.Location, newName string) *protocol.WorkspaceEdit {
	seen := existingEditRanges(goplsEdit)
	var templateEdits []protocol.TextEdit
	for _, location := range locations {
		if location.URI != docURI {
			continue
		}

		if int(location.Range.Start.Line) >= blockStartLine && int(location.Range.Start.Line) <= blockEndLine {
			continue
		}
		key := rangeKey(location.URI, location.Range)
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		templateEdits = append(templateEdits, protocol.TextEdit{Range: location.Range, NewText: newName})
	}
	if len(templateEdits) == 0 {
		return goplsEdit
	}
	addEditsForURI(goplsEdit, docURI, templateEdits)
	return goplsEdit
}

// existingEditRanges collects the (uri, range) keys already present in a workspace edit,
// in whichever representation it uses, so merged edits can be de-duplicated.
//
// Takes edit (*protocol.WorkspaceEdit) whose existing edit ranges are collected.
//
// Returns the set of (uri, range) keys already present in the edit.
func existingEditRanges(edit *protocol.WorkspaceEdit) map[string]struct{} {
	seen := make(map[string]struct{})
	for uri, edits := range edit.Changes {
		for i := range edits {
			seen[rangeKey(uri, edits[i].Range)] = struct{}{}
		}
	}
	for i := range edit.DocumentChanges {
		change := edit.DocumentChanges[i]
		for j := range change.Edits {
			seen[rangeKey(change.TextDocument.URI, change.Edits[j].Range)] = struct{}{}
		}
	}
	return seen
}

// addEditsForURI appends edits for a URI using whichever representation the workspace
// edit already uses, preserving its form rather than mixing the two.
//
// Takes edit (*protocol.WorkspaceEdit) which the edits are appended to.
// Takes uri (protocol.DocumentURI) which is the document the edits target.
// Takes edits ([]protocol.TextEdit) which are the text edits to append.
func addEditsForURI(edit *protocol.WorkspaceEdit, uri protocol.DocumentURI, edits []protocol.TextEdit) {
	if len(edit.DocumentChanges) > 0 {
		edit.DocumentChanges = append(edit.DocumentChanges, protocol.TextDocumentEdit{
			TextDocument: protocol.OptionalVersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
			},
			Edits: edits,
		})
		return
	}
	if edit.Changes == nil {
		edit.Changes = make(map[protocol.DocumentURI][]protocol.TextEdit)
	}
	edit.Changes[uri] = append(edit.Changes[uri], edits...)
}

// rangeKey builds a stable de-duplication key for a URI and range.
//
// Takes uri (protocol.DocumentURI) which is the document the range belongs to.
// Takes rng (protocol.Range) which is the range to encode into the key.
//
// Returns the stable de-duplication key string.
func rangeKey(uri protocol.DocumentURI, rng protocol.Range) string {
	return string(uri) + "#" +
		strconv.Itoa(int(rng.Start.Line)) + ":" + strconv.Itoa(int(rng.Start.Character)) + "-" +
		strconv.Itoa(int(rng.End.Line)) + ":" + strconv.Itoa(int(rng.End.Character))
}

// remapWorkspaceEdit rewrites a gopls workspace edit back into .pk coordinates.
//
// Every edit targeting the primary overlay is rewritten back to the .pk file, edits
// targeting other components' synthetic overlays are dropped (cross-.pk edits are out of
// scope here), and edits to real dependency files pass through unchanged. This is a
// data-loss-class operation: an unmapped edit would corrupt the user's file.
//
// Takes edit (*protocol.WorkspaceEdit) which is the gopls edit to remap.
//
// Returns the remapped workspace edit, or nil when edit is nil.
func (r *goplsRequest) remapWorkspaceEdit(edit *protocol.WorkspaceEdit) *protocol.WorkspaceEdit {
	if edit == nil {
		return nil
	}
	if edit.Changes != nil {
		edit.Changes = r.remapEditChanges(edit.Changes)
	}
	if edit.DocumentChanges != nil {
		edit.DocumentChanges = r.remapDocumentChanges(edit.DocumentChanges)
	}
	return edit
}

// remapEditChanges rewrites the legacy Changes map of a workspace edit, mapping the
// primary overlay's edits back to the .pk file and dropping satellite overlays.
//
// Takes changes (map[protocol.DocumentURI][]protocol.TextEdit) which is the legacy edit
// map.
//
// Returns the remapped Changes map.
func (r *goplsRequest) remapEditChanges(changes map[protocol.DocumentURI][]protocol.TextEdit) map[protocol.DocumentURI][]protocol.TextEdit {
	remapped := make(map[protocol.DocumentURI][]protocol.TextEdit, len(changes))
	for uri, edits := range changes {
		if uri == r.virtualURI {
			for i := range edits {
				edits[i].Range = r.mapper.RangeToReal(edits[i].Range)
			}
			realURI := r.mapper.RealURI()
			remapped[realURI] = append(remapped[realURI], edits...)
			continue
		}
		if r.isDroppableSynthetic(uri) {
			continue
		}
		remapped[uri] = append(remapped[uri], edits...)
	}
	return remapped
}

// remapDocumentChanges rewrites the versioned DocumentChanges of a workspace edit,
// mapping the primary overlay's edits back to the .pk file and dropping satellite
// overlays.
//
// Takes changes ([]protocol.TextDocumentEdit) which are the versioned document changes.
//
// Returns the remapped document changes with satellite overlays dropped.
func (r *goplsRequest) remapDocumentChanges(changes []protocol.TextDocumentEdit) []protocol.TextDocumentEdit {
	kept := make([]protocol.TextDocumentEdit, 0, len(changes))
	for i := range changes {
		change := changes[i]
		if change.TextDocument.URI == r.virtualURI {
			change.TextDocument.URI = r.mapper.RealURI()
			for j := range change.Edits {
				change.Edits[j].Range = r.mapper.RangeToReal(change.Edits[j].Range)
			}
			kept = append(kept, change)
			continue
		}
		if r.isDroppableSynthetic(change.TextDocument.URI) {
			continue
		}
		kept = append(kept, change)
	}
	return kept
}

// firstContentLineUTF16Column converts the rune-based start column that sfcparser reports
// for a Go block to the 1-based UTF-16 column the LSP and gopls speak, so the position
// mapper's first-line delta stays correct even when the block's opening line carries
// non-ASCII content.
//
// Takes content ([]byte) which is the full .pk document the block lives in.
// Takes location (sfcparser.Location) which is the block's rune-based start.
//
// Returns the 1-based UTF-16 column of the block's first content line.
func firstContentLineUTF16Column(content []byte, location sfcparser.Location) int {
	if location.Column <= 1 {
		return location.Column
	}
	line := nthLine(content, location.Line)
	column := 1
	runesSeen := 0
	for _, runeValue := range line {
		if runesSeen >= location.Column-1 {
			break
		}
		column += utf16UnitsForRune(runeValue)
		runesSeen++
	}
	return column
}

// nthLine returns the 1-based line of content without its trailing newline, or an empty
// string when the line is out of range.
//
// Takes content ([]byte) which is the buffer to index into.
// Takes line (int) which is the 1-based line number to return.
//
// Returns the requested line, or an empty string when out of range.
func nthLine(content []byte, line int) string {
	if line < 1 {
		return ""
	}
	current := 1
	start := 0
	for index := range content {
		if content[index] != '\n' {
			continue
		}
		if current == line {
			return string(content[start:index])
		}
		current++
		start = index + 1
	}
	if current == line {
		return string(content[start:])
	}
	return ""
}

// documentVirtualModule returns the virtual module for a document from whichever analysis
// result carries it, since the per-document and full-project results populate it at
// different times.
//
// Takes document (*document) whose analysis results are inspected.
//
// Returns the virtual module, or nil when no result carries one.
func documentVirtualModule(document *document) *annotator_dto.VirtualModule {
	if document.AnnotationResult != nil && document.AnnotationResult.VirtualModule != nil {
		return document.AnnotationResult.VirtualModule
	}
	if document.ProjectResult != nil && document.ProjectResult.VirtualModule != nil {
		return document.ProjectResult.VirtualModule
	}
	return nil
}

// lookupPrimaryComponent finds the virtual component for an absolute .pk path.
//
// Takes virtualModule (*annotator_dto.VirtualModule) which holds the component graph.
// Takes absolutePath (string) which is the .pk file to look up.
//
// Returns the matching component, and a bool that is false when none is found.
func lookupPrimaryComponent(virtualModule *annotator_dto.VirtualModule, absolutePath string) (*annotator_dto.VirtualComponent, bool) {
	if virtualModule.Graph == nil {
		return nil, false
	}
	hashedName, ok := virtualModule.Graph.PathToHashedName[absolutePath]
	if !ok {
		return nil, false
	}
	component, ok := virtualModule.ComponentsByHash[hashedName]
	return component, ok
}

// buildAliasToCanonical maps a component's .pk import aliases to the canonical Go package
// paths of the components they reference, for the import rewrite.
//
// Takes virtualModule (*annotator_dto.VirtualModule) which resolves referenced
// components.
// Takes primary (*annotator_dto.VirtualComponent) which owns the aliases to map.
//
// Returns the alias-to-canonical-package-path map for the import rewrite.
func buildAliasToCanonical(virtualModule *annotator_dto.VirtualModule, primary *annotator_dto.VirtualComponent) map[string]string {
	aliasToCanonical := make(map[string]string, len(primary.PikoAliasToHash))
	for alias, hashedName := range primary.PikoAliasToHash {
		if component, ok := virtualModule.ComponentsByHash[hashedName]; ok {
			aliasToCanonical[alias] = component.CanonicalGoPackagePath
		}
	}
	return aliasToCanonical
}

// syncSatelliteOverlays (re)opens every sibling component's generated Go as a gopls
// overlay.
//
// Takes child (*gopls_bridge.Child) which the satellites are opened against.
// Takes owner (uint64) which is the connection identifier holding the overlays.
// Takes virtualModule (*annotator_dto.VirtualModule) which supplies the components.
// Takes primaryVirtualPath (string) which is the block's own path to exclude.
func syncSatelliteOverlays(ctx context.Context, child *gopls_bridge.Child, owner uint64, virtualModule *annotator_dto.VirtualModule, primaryVirtualPath string) {
	paths, truncated := satelliteVirtualPaths(virtualModule, primaryVirtualPath)
	if truncated {
		_, l := logger_domain.From(ctx, log)
		l.Trace("gopls bridge: satellite overlay cap reached; cross-component resolution degraded for the excess",
			logger_domain.Int("cap", maxSatelliteOverlays))
	}
	for _, virtualPath := range paths {
		_ = child.SyncOverlay(ctx, owner, gopls_bridge.PathToFileURI(virtualPath), virtualModule.SourceOverlay[virtualPath])
	}
}

// satelliteOverlayURIs returns the set of synthetic overlay URIs for every sibling
// component, used to recognise (and drop) gopls results that point into a generated
// satellite file rather than a real source file.
//
// Takes virtualModule (*annotator_dto.VirtualModule) which supplies the components.
// Takes primaryVirtualPath (string) which is the block's own path to exclude.
//
// Returns the set of synthetic satellite overlay URIs.
func satelliteOverlayURIs(virtualModule *annotator_dto.VirtualModule, primaryVirtualPath string) map[protocol.DocumentURI]struct{} {
	paths, _ := satelliteVirtualPaths(virtualModule, primaryVirtualPath)
	satellites := make(map[protocol.DocumentURI]struct{}, len(paths))
	for _, virtualPath := range paths {
		satellites[gopls_bridge.PathToFileURI(virtualPath)] = struct{}{}
	}
	return satellites
}

// satelliteVirtualPaths returns the sibling components to open as gopls overlays.
//
// Takes virtualModule (*annotator_dto.VirtualModule) which supplies the components.
// Takes primaryVirtualPath (string) which is the block's own path to exclude.
//
// Returns paths which are the sibling virtual Go files to open.
// Returns truncated which is true when the cap dropped components.
func satelliteVirtualPaths(virtualModule *annotator_dto.VirtualModule, primaryVirtualPath string) (paths []string, truncated bool) {
	for _, component := range virtualModule.ComponentsByHash {
		virtualPath := component.VirtualGoFilePath
		if virtualPath == "" || virtualPath == primaryVirtualPath {
			continue
		}
		if _, ok := virtualModule.SourceOverlay[virtualPath]; !ok {
			continue
		}
		paths = append(paths, virtualPath)
	}
	slices.Sort(paths)
	if len(paths) > maxSatelliteOverlays {
		return paths[:maxSatelliteOverlays], true
	}
	return paths, false
}
