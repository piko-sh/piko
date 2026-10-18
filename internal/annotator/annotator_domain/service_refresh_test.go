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

package annotator_domain

import (
	"context"
	"maps"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

const (
	refreshTestPagePath    = "/project/pages/index.pk"
	refreshTestPartialPath = "/project/partials/card.pk"
	refreshTestPageHash    = "pages_index_abc"
	refreshTestPartialHash = "partials_card_def"
	refreshTestPageGoPath  = "my-project/pages/pages_index_abc"
	refreshTestCardGoPath  = "my-project/partials/partials_card_def"
	refreshTestPageSource  = `<template><p>page</p></template>`
	refreshTestCardSource  = `<template><div>card</div></template>`
)

type refreshTestFixture struct {
	service  *AnnotatorService
	fsReader *mockFSReader
	cached   *IntrospectionSnapshot
}

func newRefreshTestFixture() *refreshTestFixture {
	page := &annotator_dto.ParsedComponent{SourcePath: refreshTestPagePath, Template: &ast_domain.TemplateAST{}}
	card := &annotator_dto.ParsedComponent{SourcePath: refreshTestPartialPath, Template: &ast_domain.TemplateAST{}}

	graph := &annotator_dto.ComponentGraph{
		Components: map[string]*annotator_dto.ParsedComponent{
			refreshTestPageHash:    page,
			refreshTestPartialHash: card,
		},
		AllSourceContents: map[string][]byte{
			refreshTestPagePath:    []byte(refreshTestPageSource),
			refreshTestPartialPath: []byte(refreshTestCardSource),
		},
	}
	pageVirtual := &annotator_dto.VirtualComponent{Source: page, HashedName: refreshTestPageHash, CanonicalGoPackagePath: refreshTestPageGoPath}
	cardVirtual := &annotator_dto.VirtualComponent{Source: card, HashedName: refreshTestPartialHash, CanonicalGoPackagePath: refreshTestCardGoPath}
	virtualModule := &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			refreshTestPageHash:    pageVirtual,
			refreshTestPartialHash: cardVirtual,
		},
		ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
			refreshTestPageGoPath: pageVirtual,
			refreshTestCardGoPath: cardVirtual,
		},
		Graph: graph,
	}

	fsReader := &mockFSReader{files: map[string][]byte{
		refreshTestPagePath:    []byte(refreshTestPageSource),
		refreshTestPartialPath: []byte(refreshTestCardSource),
	}}
	return &refreshTestFixture{
		service:  &AnnotatorService{fsReader: fsReader},
		fsReader: fsReader,
		cached: &IntrospectionSnapshot{
			ComponentGraph: graph,
			VirtualModule:  virtualModule,
			TypeResolver:   NewTypeResolver(nil, virtualModule, nil),
		},
	}
}

func (f *refreshTestFixture) editPage() {
	f.fsReader.files[refreshTestPagePath] = []byte(`<template><p>edited page</p></template><style>p { color: red; }</style>`)
}

func TestRefreshMutableComponentData_UnchangedContentReusesSnapshot(t *testing.T) {
	t.Parallel()

	fixture := newRefreshTestFixture()

	snapshot, err := fixture.service.refreshMutableComponentData(context.Background(), fixture.cached, nil)

	require.NoError(t, err)
	assert.Same(t, fixture.cached, snapshot, "with no file changed there is nothing to refresh, so no fork is made")
}

func TestRefreshMutableComponentData_ForksInsteadOfWriting(t *testing.T) {
	t.Parallel()

	fixture := newRefreshTestFixture()
	cached := fixture.cached
	cachedComponents := maps.Clone(cached.ComponentGraph.Components)
	cachedContents := maps.Clone(cached.ComponentGraph.AllSourceContents)
	cachedByHash := maps.Clone(cached.VirtualModule.ComponentsByHash)
	cachedByGoPath := maps.Clone(cached.VirtualModule.ComponentsByGoPath)
	cachedPage := *cached.ComponentGraph.Components[refreshTestPageHash]
	cachedPageVirtual := *cached.VirtualModule.ComponentsByHash[refreshTestPageHash]

	fixture.editPage()
	snapshot, err := fixture.service.refreshMutableComponentData(context.Background(), cached, nil)
	require.NoError(t, err)

	assert.Equal(t, cachedComponents, cached.ComponentGraph.Components, "the cached graph's map must not be written")
	assert.Equal(t, cachedContents, cached.ComponentGraph.AllSourceContents)
	assert.Equal(t, cachedByHash, cached.VirtualModule.ComponentsByHash, "the cached virtual module's maps must not be written")
	assert.Equal(t, cachedByGoPath, cached.VirtualModule.ComponentsByGoPath)
	assert.Equal(t, cachedPage, *cached.ComponentGraph.Components[refreshTestPageHash], "the cached component must not be written")
	assert.Equal(t, cachedPageVirtual, *cached.VirtualModule.ComponentsByHash[refreshTestPageHash], "the cached virtual component must not be written")
	assert.Same(t, cached.VirtualModule, cached.TypeResolver.virtualModule)

	require.NotSame(t, cached, snapshot)
	require.NotSame(t, cached.ComponentGraph, snapshot.ComponentGraph)
	require.NotSame(t, cached.VirtualModule, snapshot.VirtualModule)

	refreshed := snapshot.ComponentGraph.Components[refreshTestPageHash]
	assert.NotSame(t, cached.ComponentGraph.Components[refreshTestPageHash], refreshed)
	assert.Len(t, refreshed.StyleBlocks, 1)
	assert.Equal(t, refreshTestPagePath, refreshed.SourcePath, "fields that are not refreshed are carried over")
	assert.Equal(t, fixture.fsReader.files[refreshTestPagePath], snapshot.ComponentGraph.AllSourceContents[refreshTestPagePath])

	forkedVirtual := snapshot.VirtualModule.ComponentsByHash[refreshTestPageHash]
	assert.Same(t, refreshed, forkedVirtual.Source)
	assert.Same(t, forkedVirtual, snapshot.VirtualModule.ComponentsByGoPath[refreshTestPageGoPath])
	assert.Equal(t, cachedPageVirtual.CanonicalGoPackagePath, forkedVirtual.CanonicalGoPackagePath)
	assert.Same(t, snapshot.ComponentGraph, snapshot.VirtualModule.Graph)

	assert.NotSame(t, cached.TypeResolver, snapshot.TypeResolver)
	assert.Same(t, snapshot.VirtualModule, snapshot.TypeResolver.virtualModule, "the resolver must be a view over the fork")

	assert.Same(t, cached.ComponentGraph.Components[refreshTestPartialHash], snapshot.ComponentGraph.Components[refreshTestPartialHash],
		"unchanged components are shared with the cached snapshot")
	assert.Same(t, cached.VirtualModule.ComponentsByHash[refreshTestPartialHash], snapshot.VirtualModule.ComponentsByHash[refreshTestPartialHash])

	again, err := fixture.service.refreshMutableComponentData(context.Background(), snapshot, nil)
	require.NoError(t, err)
	assert.Same(t, snapshot, again, "the fork records the content it was parsed from, so refreshing it again finds nothing new")
}

func TestRefreshMutableComponentData_OnlyConsidersChangedComponents(t *testing.T) {
	t.Parallel()

	fixture := newRefreshTestFixture()
	fixture.editPage()

	snapshot, err := fixture.service.refreshMutableComponentData(context.Background(), fixture.cached,
		map[string]bool{refreshTestPartialHash: true})

	require.NoError(t, err)
	assert.Same(t, fixture.cached, snapshot,
		"the edited page is outside the targeted set, so it is left for a later build to refresh")
}

func TestRefreshMutableComponentData_ConcurrentReadersOfCachedSnapshot(t *testing.T) {
	t.Parallel()

	fixture := newRefreshTestFixture()
	cached := fixture.cached
	fixture.editPage()

	stop := make(chan struct{})
	var readers sync.WaitGroup
	for range 4 {
		readers.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				for _, virtual := range cached.VirtualModule.ComponentsByHash {
					_ = virtual.Source.SourcePath
					_ = virtual.Source.LocalTranslations
				}
				for _, component := range cached.ComponentGraph.Components {
					_ = component.Template
				}
			}
		})
	}

	for range 20 {
		_, err := fixture.service.refreshMutableComponentData(context.Background(), cached, nil)
		require.NoError(t, err)
	}
	close(stop)
	readers.Wait()
}
