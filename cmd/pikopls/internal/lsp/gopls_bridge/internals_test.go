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
	"runtime"
	"strings"
	"sync"
	"testing"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileURI(t *testing.T) {
	t.Parallel()

	t.Run("prefixes file scheme for an absolute path", func(t *testing.T) {
		t.Parallel()

		if runtime.GOOS == "windows" {
			t.Skip("path separator handling differs on Windows")
		}
		assert.Equal(t, protocol.DocumentURI("file:///a/b/c.go"), fileURI("/a/b/c.go"))
	})

	t.Run("percent-encodes spaces and non-ASCII so it matches gopls canonicalisation", func(t *testing.T) {
		t.Parallel()

		if runtime.GOOS == "windows" {
			t.Skip("path separator handling differs on Windows")
		}

		got := string(fileURI("/Users/First Last/projét/source.pk.go"))
		assert.Equal(t, "file:///Users/First%20Last/proj%C3%A9t/source.pk.go", got)
		assert.NotContains(t, got, " ", "a raw space would never match gopls")
	})

	t.Run("PathToFileURI is exported fileURI", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, fileURI("/x"), PathToFileURI("/x"))
	})
}

func TestMapperAccessorsAndRangeToVirtual(t *testing.T) {
	t.Parallel()

	mapper := NewMapper("file:///x.pk", "file:///x.pk.go", 10, 1)

	assert.Equal(t, protocol.DocumentURI("file:///x.pk"), mapper.RealURI())
	assert.Equal(t, protocol.DocumentURI("file:///x.pk.go"), mapper.VirtualURI())

	virtual := mapper.RangeToVirtual(protocol.Range{
		Start: protocol.Position{Line: 11, Character: 4},
		End:   protocol.Position{Line: 11, Character: 9},
	})
	assert.Equal(t, uint32(2), virtual.Start.Line)
	assert.Equal(t, uint32(4), virtual.Start.Character)
	assert.Equal(t, uint32(2), virtual.End.Line)
	assert.Equal(t, uint32(9), virtual.End.Character)
}

func TestSanitisePackageDir(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"pages_cards_abc123":     "pages_cards_abc123",
		"with/slash":             "with_slash",
		"weird name!@#":          "weird_name___",
		"keep-dash_and_under123": "keep-dash_and_under123",
		"":                       "block",

		"../escape": "___escape",
	}
	for input, want := range cases {
		assert.Equal(t, want, sanitisePackageDir(input), "input %q", input)
	}
}

func TestRewriteBlockUnparseableReturnedVerbatim(t *testing.T) {
	t.Parallel()

	block := "this is not valid go {{{"
	assert.Equal(t, block, RewriteBlock(block, map[string]string{"x": "y"}))
}

func TestRewriteBlockExplicitAliasRewritten(t *testing.T) {
	t.Parallel()

	block := "package main\n\nimport alias \"x/card.pk\"\n\nfunc F() { _ = alias.X }\n"
	rewritten := RewriteBlock(block, map[string]string{"alias": "mod/.piko/card_hash"})
	assert.Contains(t, rewritten, `alias "mod/.piko/card_hash"`)
	assert.NotContains(t, rewritten, "card.pk")
}

func TestRewriteBlockDefaultAliasFromPackageBase(t *testing.T) {
	t.Parallel()

	block := "package main\n\nimport \"x/card.pk\"\n\nfunc F() { _ = card.X }\n"
	rewritten := RewriteBlock(block, map[string]string{"card": "mod/.piko/card_hash"})
	assert.Contains(t, rewritten, `"mod/.piko/card_hash"`)
	assert.NotContains(t, rewritten, "card.pk")
}

func TestManagerSubscribeFanOut(t *testing.T) {
	t.Parallel()

	manager := NewManager(ManagerConfig{})

	var (
		mu    sync.Mutex
		seenA int
		seenB int
	)
	unsubA := manager.Subscribe(Subscriber{Diagnostics: func(_ context.Context, _ *protocol.PublishDiagnosticsParams) {
		mu.Lock()
		seenA++
		mu.Unlock()
	}})
	manager.Subscribe(Subscriber{Diagnostics: func(_ context.Context, _ *protocol.PublishDiagnosticsParams) {
		mu.Lock()
		seenB++
		mu.Unlock()
	}})

	manager.fanOut(context.Background(), &protocol.PublishDiagnosticsParams{})
	mu.Lock()
	assert.Equal(t, 1, seenA)
	assert.Equal(t, 1, seenB)
	mu.Unlock()

	unsubA()
	manager.fanOut(context.Background(), &protocol.PublishDiagnosticsParams{})
	mu.Lock()
	assert.Equal(t, 1, seenA, "unsubscribed handler receives nothing further")
	assert.Equal(t, 2, seenB)
	mu.Unlock()
}

func TestManagerFanOutIsolatesPanickingSubscriber(t *testing.T) {
	t.Parallel()

	manager := NewManager(ManagerConfig{})

	delivered := false
	manager.Subscribe(Subscriber{Diagnostics: func(_ context.Context, _ *protocol.PublishDiagnosticsParams) {
		panic("subscriber blew up")
	}})
	manager.Subscribe(Subscriber{Diagnostics: func(_ context.Context, _ *protocol.PublishDiagnosticsParams) {
		delivered = true
	}})

	assert.NotPanics(t, func() {
		manager.fanOut(context.Background(), &protocol.PublishDiagnosticsParams{})
	})
	assert.True(t, delivered, "a healthy subscriber still receives after a panicking one")
}

func TestGoplsSearchDirectoriesIncludesHomeFallbacks(t *testing.T) {
	t.Parallel()

	directories := goplsSearchDirectories()

	require.NotEmpty(t, directories)
	assert.Contains(t, strings.Join(directories, "\n"), "go")
}

func TestBuildGoplsInitializeParams(t *testing.T) {
	t.Parallel()

	params := buildGoplsInitializeParams("/module/root")
	require.NotNil(t, params)
	require.Len(t, params.WorkspaceFolders, 1)
	assert.Equal(t, "file:///module/root", params.WorkspaceFolders[0].URI)
	assert.Equal(t, "root", params.WorkspaceFolders[0].Name)
}

func newReadyTestChild() *Child {
	return &Child{
		handler:  newClientHandler("/m"),
		overlays: make(map[protocol.DocumentURI]*overlayState),
		done:     make(chan struct{}),
	}
}

func TestChildIsReady(t *testing.T) {
	t.Parallel()

	child := newReadyTestChild()
	assert.False(t, child.IsReady(), "a cold child is not ready")

	child.handler.markReady()
	assert.True(t, child.IsReady(), "once gopls finishes its load the child is ready")
}

func TestChildWaitReady(t *testing.T) {
	t.Parallel()

	t.Run("returns true once ready", func(t *testing.T) {
		t.Parallel()

		child := newReadyTestChild()
		go child.handler.markReady()
		assert.True(t, child.WaitReady(context.Background()))
	})

	t.Run("returns false when the child dies", func(t *testing.T) {
		t.Parallel()

		child := newReadyTestChild()
		close(child.done)
		assert.False(t, child.WaitReady(context.Background()))
	})

	t.Run("returns false when the context is cancelled", func(t *testing.T) {
		t.Parallel()

		child := newReadyTestChild()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.False(t, child.WaitReady(ctx))
	})
}
