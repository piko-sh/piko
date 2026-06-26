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
	"testing"
	"time"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newInteractiveTestWorkspace(uri protocol.DocumentURI, doc *document, inFlight bool) *workspace {
	ws := &workspace{
		documents:           map[protocol.DocumentURI]*document{},
		docCache:            NewDocumentCache(),
		analysisDone:        map[protocol.DocumentURI]chan struct{}{},
		requestedGeneration: map[protocol.DocumentURI]uint64{},
	}
	if doc != nil {
		ws.documents[uri] = doc
	}
	if inFlight {

		ws.analysisDone[uri] = make(chan struct{})
	}
	return ws
}

func TestRunAnalysisForInteractiveRequest_ReturnsCleanCacheImmediately(t *testing.T) {
	t.Parallel()

	uri := protocol.DocumentURI("file:///clean.pk")
	clean := &document{URI: uri, dirty: false}
	ws := newInteractiveTestWorkspace(uri, clean, false)

	start := time.Now()
	doc, err := ws.RunAnalysisForInteractiveRequest(context.Background(), uri)

	require.NoError(t, err)
	assert.Same(t, clean, doc, "a clean cached document must be returned directly")
	assert.Less(t, time.Since(start), interactiveAnalysisBudget, "the clean fast path must not wait")
}

func TestRunAnalysisForInteractiveRequest_BoundedFallbackToLastKnownGood(t *testing.T) {
	t.Parallel()

	uri := protocol.DocumentURI("file:///dirty.pk")
	lastKnownGood := &document{URI: uri, dirty: true}
	ws := newInteractiveTestWorkspace(uri, lastKnownGood, true)

	start := time.Now()
	doc, err := ws.RunAnalysisForInteractiveRequest(context.Background(), uri)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Same(t, lastKnownGood, doc, "must degrade to the last analysed document, not block")
	assert.Less(t, elapsed, interactiveAnalysisBudget+2*time.Second, "must return within the bounded budget, never hang")
}

func TestSetupAnalysisContext_OlderSetupSelfAborts(t *testing.T) {
	t.Parallel()
	workspace := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")
	ctx := context.Background()

	latestCtx, _, latestToken := workspace.setupAnalysisContext(ctx, uri, 2)
	require.NotZero(t, latestToken, "the newer generation must take ownership")
	require.NoError(t, latestCtx.Err())

	olderCtx, olderDone, olderToken := workspace.setupAnalysisContext(ctx, uri, 1)
	assert.Zero(t, olderToken, "an older setup must not take ownership")
	assert.Error(t, olderCtx.Err(), "an older setup must return an already-cancelled context")
	select {
	case <-olderDone:
	default:
		t.Error("an older setup must return a closed done channel")
	}
	assert.NoError(t, latestCtx.Err(), "the newer survivor must remain live after an older setup aborts")
}

func TestSetupAnalysisContext_NewerSupersedesOlder(t *testing.T) {
	t.Parallel()
	workspace := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")
	ctx := context.Background()

	olderCtx, olderDone, olderToken := workspace.setupAnalysisContext(ctx, uri, 1)
	require.NotZero(t, olderToken)

	newerCtx, _, newerToken := workspace.setupAnalysisContext(ctx, uri, 2)
	require.NotZero(t, newerToken)

	assert.Error(t, olderCtx.Err(), "the older analysis must be cancelled by the newer setup")
	select {
	case <-olderDone:
	default:
		t.Error("the older done channel must be closed by the newer setup")
	}
	assert.NoError(t, newerCtx.Err(), "the newer analysis must remain live")
}

func TestSupersede_CancelsOnlyStrictlyOlder(t *testing.T) {
	t.Parallel()
	workspace := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")
	ctx := context.Background()

	olderCtx, _, _ := workspace.setupAnalysisContext(ctx, uri, 1)
	workspace.supersede(uri, 2)
	assert.Error(t, olderCtx.Err(), "supersede must cancel a strictly-older in-flight analysis")

	liveCtx, _, _ := workspace.setupAnalysisContext(ctx, uri, 5)
	workspace.supersede(uri, 5)
	assert.NoError(t, liveCtx.Err(), "supersede with an equal generation must be a no-op")
	workspace.supersede(uri, 3)
	assert.NoError(t, liveCtx.Err(), "supersede with an older generation must be a no-op")
}

func TestCleanup_OnlyOwnerReleasesURI(t *testing.T) {
	t.Parallel()
	workspace := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")
	ctx := context.Background()

	_, firstDone, firstToken := workspace.setupAnalysisContext(ctx, uri, 1)
	_, _, secondToken := workspace.setupAnalysisContext(ctx, uri, 2)
	require.NotEqual(t, firstToken, secondToken)

	workspace.cleanupAnalysisContext(ctx, uri, firstDone, firstToken)

	workspace.mu.RLock()
	owner, owned := workspace.inFlight[uri]
	workspace.mu.RUnlock()
	require.True(t, owned, "the newer owner must retain ownership after the older's cleanup")
	assert.Equal(t, secondToken, owner.token)
}

func TestCommitAnalysedDocument_RejectsStaleGeneration(t *testing.T) {
	t.Parallel()
	workspace := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")

	first := workspace.UpdateDocument(uri, []byte("v1"), 1)
	second := workspace.UpdateDocument(uri, []byte("v2"), 2)
	require.Equal(t, uint64(1), first)
	require.Equal(t, uint64(2), second)

	committed := workspace.commitAnalysedDocument(uri, &document{URI: uri, dirty: false}, first)
	assert.False(t, committed, "a stale-generation result must be rejected")
	stored, _ := workspace.GetDocument(uri)
	assert.True(t, stored.dirty, "dirty must remain set after a rejected stale commit")

	committed = workspace.commitAnalysedDocument(uri, &document{URI: uri, dirty: false}, second)
	assert.True(t, committed, "the latest-generation result must commit")
	stored, _ = workspace.GetDocument(uri)
	assert.False(t, stored.dirty, "dirty must clear once the latest content commits")
}

func TestCommitAnalysedDocument_TracksCommittedVersion(t *testing.T) {
	t.Parallel()
	workspace := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")

	first := workspace.UpdateDocument(uri, []byte("v50"), 50)
	require.True(t, workspace.commitAnalysedDocument(uri, &document{URI: uri}, first))
	assert.Equal(t, int32(50), workspace.committedVersion[uri])

	second := workspace.UpdateDocument(uri, []byte("v51"), 51)
	assert.Equal(t, int32(50), workspace.committedVersion[uri], "committed version must lag the requested version until commit")

	assert.False(t, workspace.commitAnalysedDocument(uri, &document{URI: uri}, first))
	assert.Equal(t, int32(50), workspace.committedVersion[uri])

	assert.True(t, workspace.commitAnalysedDocument(uri, &document{URI: uri}, second))
	assert.Equal(t, int32(51), workspace.committedVersion[uri])
}

func TestSettleGeneration(t *testing.T) {
	t.Parallel()
	workspace := createTestWorkspace()
	uri := protocol.DocumentURI("file:///test.pk")
	ctx := context.Background()

	assert.Zero(t, workspace.settleGeneration(uri, 0), "an unknown document needs no re-arm")

	generation := workspace.UpdateDocument(uri, []byte("v1"), 1)

	assert.Equal(t, generation, workspace.settleGeneration(uri, 0), "a newer pending edit must re-arm the latest generation")

	assert.Zero(t, workspace.settleGeneration(uri, generation), "the just-attempted latest generation must not re-arm itself")

	_, _, _ = workspace.setupAnalysisContext(ctx, uri, generation)
	assert.Zero(t, workspace.settleGeneration(uri, 0), "an owned URI must not re-arm")

	require.True(t, workspace.commitAnalysedDocument(uri, &document{URI: uri, dirty: false}, generation))
	assert.Zero(t, workspace.settleGeneration(uri, 0), "a clean document needs no re-arm")
}
