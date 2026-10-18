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

package render_adapters

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

const (
	invalidationTestSVGID     = "module/lib/icon.svg"
	invalidationTestScriptID  = "captcha/init-test.js"
	invalidationTestServeBase = "/_piko/assets"
)

func newSwitchableArtefactRegistry(blobs map[string]string) (*registry_domain.MockRegistryService, *atomic.Pointer[registry_dto.ArtefactMeta]) {
	current := &atomic.Pointer[registry_dto.ArtefactMeta]{}
	registry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
			return current.Load(), nil
		},
		GetMultipleArtefactsFunc: func(context.Context, []string) ([]*registry_dto.ArtefactMeta, error) {
			return []*registry_dto.ArtefactMeta{current.Load()}, nil
		},
		GetVariantDataFunc: func(_ context.Context, variant *registry_dto.Variant) (io.ReadCloser, error) {
			blob, ok := blobs[variant.StorageKey]
			if !ok {
				return nil, registry_domain.ErrBlobNotFound
			}
			return io.NopCloser(strings.NewReader(blob)), nil
		},
		UpsertArtefactFunc: func(context.Context, string, string, io.Reader, string, []registry_dto.NamedProfile) (*registry_dto.ArtefactMeta, error) {
			return current.Load(), nil
		},
	}
	return registry, current
}

func newInvalidationTestAdapter(t *testing.T, registry registry_domain.RegistryService) *DataLoaderRegistryAdapter {
	t.Helper()
	adapter, ok := NewDataLoaderRegistryAdapter(registry, nil, invalidationTestServeBase).(*DataLoaderRegistryAdapter)
	require.True(t, ok)
	t.Cleanup(adapter.Close)
	return adapter
}

func svgArtefact(sourceKey, minifiedKey string, minifiedStatus registry_dto.VariantStatus) *registry_dto.ArtefactMeta {
	return &registry_dto.ArtefactMeta{
		ID: invalidationTestSVGID,
		ActualVariants: []registry_dto.Variant{
			{
				VariantID:    "source",
				StorageKey:   sourceKey,
				MetadataTags: registry_dto.TagsFromMap(map[string]string{"type": "source"}),
				Status:       registry_dto.VariantStatusReady,
			},
			{
				VariantID:    "minified",
				StorageKey:   minifiedKey,
				MetadataTags: registry_dto.TagsFromMap(map[string]string{"type": "minified-svg"}),
				Status:       minifiedStatus,
			},
		},
	}
}

func scriptArtefact(sourceKey, minifiedKey string, minifiedStatus registry_dto.VariantStatus) *registry_dto.ArtefactMeta {
	return &registry_dto.ArtefactMeta{
		ID: invalidationTestScriptID,
		ActualVariants: []registry_dto.Variant{
			{
				VariantID:    "source",
				StorageKey:   sourceKey,
				MetadataTags: registry_dto.TagsFromMap(map[string]string{"type": "source"}),
				Status:       registry_dto.VariantStatusReady,
			},
			{
				VariantID:    minifiedVariantID,
				StorageKey:   minifiedKey,
				MetadataTags: registry_dto.TagsFromMap(map[string]string{"type": "minified-js"}),
				Status:       minifiedStatus,
			},
		},
	}
}

func renderedSVG(t *testing.T, adapter *DataLoaderRegistryAdapter) string {
	t.Helper()
	parsed, err := adapter.GetAssetRawSVG(context.Background(), invalidationTestSVGID)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	return parsed.InnerHTML
}

func TestGetAssetRawSVG_RenderDuringMinifyWindow(t *testing.T) {
	t.Parallel()

	blobs := map[string]string{
		"src-v1": `<svg viewBox="0 0 24 24"><path id="first-source"></path></svg>`,
		"min-v1": `<svg viewBox="0 0 24 24"><path id="first-minified"></path></svg>`,
		"src-v2": `<svg viewBox="0 0 24 24"><path id="second-source"></path></svg>`,
		"min-v2": `<svg viewBox="0 0 24 24"><path id="second-minified"></path></svg>`,
	}
	registry, current := newSwitchableArtefactRegistry(blobs)
	adapter := newInvalidationTestAdapter(t, registry)

	current.Store(svgArtefact("src-v1", "min-v1", registry_dto.VariantStatusReady))
	require.Contains(t, renderedSVG(t, adapter), "first-minified")

	current.Store(svgArtefact("src-v2", "min-v1", registry_dto.VariantStatusStale))
	adapter.ClearSvgCache(context.Background(), invalidationTestSVGID)
	assert.Contains(t, renderedSVG(t, adapter), "second-source",
		"while the minified variant is rebuilt the edited source must be rendered, not the previous minified output")

	current.Store(svgArtefact("src-v2", "min-v2", registry_dto.VariantStatusReady))
	assert.Contains(t, renderedSVG(t, adapter), "second-source",
		"without an invalidation the parsed SVG stays cached")

	adapter.InvalidateArtefact(context.Background(), invalidationTestSVGID)
	assert.Contains(t, renderedSVG(t, adapter), "second-minified",
		"the artefact event for the rebuilt variant must let the minified output replace the cached source")
}

func TestGetArtefactServePath_CacheInvalidation(t *testing.T) {
	t.Parallel()

	registry, current := newSwitchableArtefactRegistry(nil)
	adapter := newInvalidationTestAdapter(t, registry)
	ctx := context.Background()

	current.Store(scriptArtefact("src-v1", "min-v1", registry_dto.VariantStatusReady))
	require.Equal(t, invalidationTestServeBase+"/min-v1", adapter.GetArtefactServePath(ctx, invalidationTestScriptID))

	current.Store(scriptArtefact("src-v2", "min-v1", registry_dto.VariantStatusStale))
	require.Equal(t, invalidationTestServeBase+"/min-v1", adapter.GetArtefactServePath(ctx, invalidationTestScriptID),
		"the serve path is cached until something invalidates it")

	_, err := adapter.UpsertArtefact(ctx, invalidationTestScriptID, "init-test.js", bytes.NewReader(nil), "local_disk_cache", nil)
	require.NoError(t, err)
	assert.Equal(t, invalidationTestServeBase+"/src-v2", adapter.GetArtefactServePath(ctx, invalidationTestScriptID),
		"after an upsert the stale minified variant must not be served; the source is served until it is rebuilt")

	current.Store(scriptArtefact("src-v2", "min-v2", registry_dto.VariantStatusReady))
	require.Equal(t, invalidationTestServeBase+"/src-v2", adapter.GetArtefactServePath(ctx, invalidationTestScriptID))

	adapter.InvalidateArtefact(ctx, invalidationTestScriptID)
	assert.Equal(t, invalidationTestServeBase+"/min-v2", adapter.GetArtefactServePath(ctx, invalidationTestScriptID),
		"an artefact event must drop the cached serve path")
}

func TestFindServableVariant(t *testing.T) {
	t.Parallel()

	t.Run("prefers the ready minified variant", func(t *testing.T) {
		t.Parallel()

		variant := findServableVariant(scriptArtefact("src", "min", registry_dto.VariantStatusReady))

		require.NotNil(t, variant)
		assert.Equal(t, "min", variant.StorageKey)
	})

	t.Run("falls back to the source while the minified variant is stale", func(t *testing.T) {
		t.Parallel()

		variant := findServableVariant(scriptArtefact("src", "min", registry_dto.VariantStatusStale))

		require.NotNil(t, variant)
		assert.Equal(t, "src", variant.StorageKey)
	})
}
