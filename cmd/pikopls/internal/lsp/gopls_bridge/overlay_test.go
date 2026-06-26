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
	"sync"
	"testing"
	"time"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingServer struct {
	protocol.Server
	opens   []protocol.DidOpenTextDocumentParams
	changes []protocol.DidChangeTextDocumentParams
	closes  []protocol.DidCloseTextDocumentParams
	mu      sync.Mutex
}

func (r *recordingServer) DidOpen(_ context.Context, params *protocol.DidOpenTextDocumentParams) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.opens = append(r.opens, *params)
	return nil
}

func (r *recordingServer) DidChange(_ context.Context, params *protocol.DidChangeTextDocumentParams) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.changes = append(r.changes, *params)
	return nil
}

func (r *recordingServer) DidClose(_ context.Context, params *protocol.DidCloseTextDocumentParams) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closes = append(r.closes, *params)
	return nil
}

func (r *recordingServer) counts() (int, int, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.opens), len(r.changes), len(r.closes)
}

const (
	testOwner uint64 = 1
)

func newTestChild(server protocol.Server) *Child {
	return &Child{
		server:   server,
		overlays: make(map[protocol.DocumentURI]*overlayState),
		done:     make(chan struct{}),
	}
}

func TestRollbackUnopenedOverlay_EvictsUnopenedAfterVersionBump(t *testing.T) {
	t.Parallel()

	child := newTestChild(&recordingServer{})
	uri := protocol.DocumentURI("file:///x.pk.go")
	state := &overlayState{
		version:          2,
		didOpenSucceeded: false,
		analysed:         make(chan struct{}),
		owners:           map[uint64]struct{}{},
	}
	child.overlays[uri] = state

	child.rollbackUnopenedOverlay(uri, state)

	assert.False(t, child.hasOverlay(uri),
		"an unopened overlay must be evicted even after a concurrent version bump")
}

func TestRollbackUnopenedOverlay_KeepsOpenedAndReplacedOverlays(t *testing.T) {
	t.Parallel()

	child := newTestChild(&recordingServer{})
	uri := protocol.DocumentURI("file:///y.pk.go")

	opened := &overlayState{version: 1, didOpenSucceeded: true, analysed: make(chan struct{}), owners: map[uint64]struct{}{}}
	child.overlays[uri] = opened
	child.rollbackUnopenedOverlay(uri, opened)
	assert.True(t, child.hasOverlay(uri), "a successfully-opened overlay must not be evicted")

	stale := &overlayState{version: 1, analysed: make(chan struct{}), owners: map[uint64]struct{}{}}
	current := &overlayState{version: 1, analysed: make(chan struct{}), owners: map[uint64]struct{}{}}
	child.overlays[uri] = current
	child.rollbackUnopenedOverlay(uri, stale)
	assert.True(t, child.hasOverlay(uri), "rollback must only evict its own failed state object")
}

func TestSyncOverlay_EnforcesPerChildLimit(t *testing.T) {
	t.Parallel()

	server := &recordingServer{}
	child := newTestChild(server)
	child.maxOverlays = 2
	ctx := context.Background()

	require.NoError(t, child.SyncOverlay(ctx, testOwner, "file:///m/pikopls/a/source.pk.go", []byte("a")))
	require.NoError(t, child.SyncOverlay(ctx, testOwner, "file:///m/pikopls/b/source.pk.go", []byte("b")))

	err := child.SyncOverlay(ctx, testOwner, "file:///m/pikopls/c/source.pk.go", []byte("c"))
	require.ErrorIs(t, err, ErrOverlayLimit)
	assert.False(t, child.hasOverlay("file:///m/pikopls/c/source.pk.go"))

	require.NoError(t, child.SyncOverlay(ctx, testOwner, "file:///m/pikopls/a/source.pk.go", []byte("a edited")))
}

func TestSyncOverlayLifecycle(t *testing.T) {
	t.Parallel()

	server := &recordingServer{}
	child := newTestChild(server)
	ctx := context.Background()
	uri := protocol.DocumentURI("file:///m/pikopls/abc/source.pk.go")

	require.NoError(t, child.SyncOverlay(ctx, testOwner, uri, []byte("package main")))
	opens, changes, _ := server.counts()
	assert.Equal(t, 1, opens, "first sync opens the overlay")
	assert.Equal(t, 0, changes)
	assert.True(t, child.hasOverlay(uri))

	require.NoError(t, child.SyncOverlay(ctx, testOwner, uri, []byte("package main")))
	opens, changes, _ = server.counts()
	assert.Equal(t, 1, opens)
	assert.Equal(t, 0, changes, "unchanged content is not re-sent")

	require.NoError(t, child.SyncOverlay(ctx, testOwner, uri, []byte("package main // edited")))
	opens, changes, _ = server.counts()
	assert.Equal(t, 1, opens)
	assert.Equal(t, 1, changes, "changed content sends a didChange")

	server.mu.Lock()
	version := server.changes[0].TextDocument.Version
	server.mu.Unlock()
	assert.Equal(t, int32(2), version, "the change carries the incremented version")
}

func TestCloseOverlay(t *testing.T) {
	t.Parallel()

	server := &recordingServer{}
	child := newTestChild(server)
	ctx := context.Background()
	uri := protocol.DocumentURI("file:///m/pikopls/abc/source.pk.go")

	require.NoError(t, child.CloseOverlay(ctx, testOwner, uri))
	_, _, closes := server.counts()
	assert.Equal(t, 0, closes)

	require.NoError(t, child.SyncOverlay(ctx, testOwner, uri, []byte("package main")))
	require.NoError(t, child.CloseOverlay(ctx, testOwner, uri))
	_, _, closes = server.counts()
	assert.Equal(t, 1, closes, "an open overlay is closed against gopls")
	assert.False(t, child.hasOverlay(uri))
}

func TestCloseOverlayRefcountedAcrossOwners(t *testing.T) {
	t.Parallel()

	const (
		ownerA uint64 = 1
		ownerB uint64 = 2
	)
	server := &recordingServer{}
	child := newTestChild(server)
	ctx := context.Background()
	uri := protocol.DocumentURI("file:///m/pikopls/abc/source.pk.go")

	require.NoError(t, child.SyncOverlay(ctx, ownerA, uri, []byte("package main")))
	require.NoError(t, child.SyncOverlay(ctx, ownerB, uri, []byte("package main")))
	opens, _, _ := server.counts()
	assert.Equal(t, 1, opens, "the shared overlay is opened once")

	require.NoError(t, child.CloseOverlay(ctx, ownerA, uri))
	_, _, closes := server.counts()
	assert.Equal(t, 0, closes, "a still-referenced overlay is not closed against gopls")
	assert.True(t, child.hasOverlay(uri), "overlay survives while another owner holds it")

	require.NoError(t, child.CloseOverlay(ctx, ownerB, uri))
	_, _, closes = server.counts()
	assert.Equal(t, 1, closes, "the overlay is closed when the last owner releases it")
	assert.False(t, child.hasOverlay(uri))
}

func TestOverlayAnalysedSignal(t *testing.T) {
	t.Parallel()

	server := &recordingServer{}
	child := newTestChild(server)
	ctx := context.Background()
	uri := protocol.DocumentURI("file:///m/pikopls/abc/source.pk.go")

	assert.False(t, child.IsOverlayAnalysed(uri), "unknown overlay is not analysed")

	require.NoError(t, child.SyncOverlay(ctx, testOwner, uri, []byte("package main")))
	assert.False(t, child.IsOverlayAnalysed(uri), "freshly opened overlay is not analysed yet")

	child.markOverlayAnalysed(uri)
	assert.True(t, child.IsOverlayAnalysed(uri), "marking records analysis")

	child.markOverlayAnalysed(uri)
	assert.True(t, child.IsOverlayAnalysed(uri))
}

func TestWaitOverlayAnalysed(t *testing.T) {
	t.Parallel()

	t.Run("returns false for an unknown overlay", func(t *testing.T) {
		t.Parallel()

		child := newTestChild(&recordingServer{})
		assert.False(t, child.WaitOverlayAnalysed(context.Background(), "file:///unknown", time.Second))
	})

	t.Run("returns true once analysed", func(t *testing.T) {
		t.Parallel()

		server := &recordingServer{}
		child := newTestChild(server)
		ctx := context.Background()
		uri := protocol.DocumentURI("file:///m/pikopls/abc/source.pk.go")
		require.NoError(t, child.SyncOverlay(ctx, testOwner, uri, []byte("package main")))

		go child.markOverlayAnalysed(uri)
		assert.True(t, child.WaitOverlayAnalysed(ctx, uri, 2*time.Second))
	})

	t.Run("returns false when the child dies", func(t *testing.T) {
		t.Parallel()

		server := &recordingServer{}
		child := newTestChild(server)
		ctx := context.Background()
		uri := protocol.DocumentURI("file:///m/pikopls/abc/source.pk.go")
		require.NoError(t, child.SyncOverlay(ctx, testOwner, uri, []byte("package main")))

		close(child.done)
		assert.False(t, child.WaitOverlayAnalysed(ctx, uri, 2*time.Second))
	})

	t.Run("returns false when the context is cancelled", func(t *testing.T) {
		t.Parallel()

		server := &recordingServer{}
		child := newTestChild(server)
		uri := protocol.DocumentURI("file:///m/pikopls/abc/source.pk.go")
		require.NoError(t, child.SyncOverlay(context.Background(), testOwner, uri, []byte("package main")))

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.False(t, child.WaitOverlayAnalysed(ctx, uri, 2*time.Second))
	})

	t.Run("returns true on timeout so a slow load never blocks forever", func(t *testing.T) {
		t.Parallel()

		server := &recordingServer{}
		child := newTestChild(server)
		uri := protocol.DocumentURI("file:///m/pikopls/abc/source.pk.go")
		require.NoError(t, child.SyncOverlay(context.Background(), testOwner, uri, []byte("package main")))

		assert.True(t, child.WaitOverlayAnalysed(context.Background(), uri, time.Millisecond))
	})
}
