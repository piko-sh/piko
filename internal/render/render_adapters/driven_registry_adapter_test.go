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
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestApplyConfigDefaults_NilConfig_ReturnsDefaults(t *testing.T) {
	t.Parallel()

	result := applyConfigDefaults(nil)

	require.NotNil(t, result)
	assert.Equal(t, defaultComponentCacheCapacity, result.ComponentCacheCapacity)
	assert.Equal(t, defaultComponentCacheTTLMinutes*time.Minute, result.ComponentCacheTTL)
	assert.Equal(t, defaultSVGCacheCapacity, result.SVGCacheCapacity)
	assert.Equal(t, defaultSVGCacheTTLMinutes*time.Minute, result.SVGCacheTTL)
}

func TestApplyConfigDefaults_ZeroValues_ReturnsDefaults(t *testing.T) {
	t.Parallel()

	result := applyConfigDefaults(&DataLoaderAdapterConfig{})

	assert.Equal(t, defaultComponentCacheCapacity, result.ComponentCacheCapacity)
	assert.Equal(t, defaultComponentCacheTTLMinutes*time.Minute, result.ComponentCacheTTL)
	assert.Equal(t, defaultSVGCacheCapacity, result.SVGCacheCapacity)
	assert.Equal(t, defaultSVGCacheTTLMinutes*time.Minute, result.SVGCacheTTL)
}

func TestApplyConfigDefaults_CustomValues_PreservesValues(t *testing.T) {
	t.Parallel()

	config := &DataLoaderAdapterConfig{
		ComponentCacheCapacity: 50,
		ComponentCacheTTL:      2 * time.Minute,
		SVGCacheCapacity:       100,
		SVGCacheTTL:            10 * time.Minute,
	}

	result := applyConfigDefaults(config)

	assert.Equal(t, 50, result.ComponentCacheCapacity)
	assert.Equal(t, 2*time.Minute, result.ComponentCacheTTL)
	assert.Equal(t, 100, result.SVGCacheCapacity)
	assert.Equal(t, 10*time.Minute, result.SVGCacheTTL)
}

func TestApplyConfigDefaults_PartialValues_FillsMissing(t *testing.T) {
	t.Parallel()

	config := &DataLoaderAdapterConfig{
		ComponentCacheCapacity: 42,
		SVGCacheTTL:            15 * time.Minute,
	}

	result := applyConfigDefaults(config)

	assert.Equal(t, 42, result.ComponentCacheCapacity)
	assert.Equal(t, defaultComponentCacheTTLMinutes*time.Minute, result.ComponentCacheTTL)
	assert.Equal(t, defaultSVGCacheCapacity, result.SVGCacheCapacity)
	assert.Equal(t, 15*time.Minute, result.SVGCacheTTL)
}

func TestGetComponentMetadata_EmptyType_ReturnsError(t *testing.T) {
	t.Parallel()

	config := &DataLoaderAdapterConfig{
		ComponentCacheCapacity: 10,
		ComponentCacheTTL:      1 * time.Minute,
		SVGCacheCapacity:       10,
		SVGCacheTTL:            1 * time.Minute,
	}
	componentCache, svgCache := createCaches(config)
	defer componentCache.StopAllGoroutines()
	defer svgCache.StopAllGoroutines()

	adapter := &DataLoaderRegistryAdapter{
		componentCache: componentCache,
		svgCache:       svgCache,
	}

	result, err := adapter.GetComponentMetadata(context.Background(), "")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "empty componentType")
}

func TestGetAssetRawSVG_EmptyAssetID_ReturnsError(t *testing.T) {
	t.Parallel()

	config := &DataLoaderAdapterConfig{
		ComponentCacheCapacity: 10,
		ComponentCacheTTL:      1 * time.Minute,
		SVGCacheCapacity:       10,
		SVGCacheTTL:            1 * time.Minute,
	}
	componentCache, svgCache := createCaches(config)
	defer componentCache.StopAllGoroutines()
	defer svgCache.StopAllGoroutines()

	adapter := &DataLoaderRegistryAdapter{
		componentCache: componentCache,
		svgCache:       svgCache,
	}

	result, err := adapter.GetAssetRawSVG(context.Background(), "")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "empty assetID")
}

func TestClose_NilCaches_DoesNotPanic(t *testing.T) {
	t.Parallel()

	adapter := &DataLoaderRegistryAdapter{}

	assert.NotPanics(t, func() {
		adapter.Close()
	})
}

func TestClose_ValidCaches_StopsGoroutines(t *testing.T) {
	t.Parallel()

	config := &DataLoaderAdapterConfig{
		ComponentCacheCapacity: 10,
		ComponentCacheTTL:      1 * time.Minute,
		SVGCacheCapacity:       10,
		SVGCacheTTL:            1 * time.Minute,
	}
	componentCache, svgCache := createCaches(config)

	adapter := &DataLoaderRegistryAdapter{
		componentCache: componentCache,
		svgCache:       svgCache,
	}

	assert.NotPanics(t, func() {
		adapter.Close()
	})
}

func TestClearComponentCache_NilCache_DoesNotPanic(t *testing.T) {
	t.Parallel()

	adapter := &DataLoaderRegistryAdapter{}

	assert.NotPanics(t, func() {
		adapter.ClearComponentCache(context.Background(), "test-component")
	})
}

func TestClearSvgCache_NilCache_DoesNotPanic(t *testing.T) {
	t.Parallel()

	adapter := &DataLoaderRegistryAdapter{}

	assert.NotPanics(t, func() {
		adapter.ClearSvgCache(context.Background(), "test-svg")
	})
}

func TestGetStats_EmptyCaches_ReturnsZero(t *testing.T) {
	t.Parallel()

	config := &DataLoaderAdapterConfig{
		ComponentCacheCapacity: 10,
		ComponentCacheTTL:      1 * time.Minute,
		SVGCacheCapacity:       10,
		SVGCacheTTL:            1 * time.Minute,
	}
	componentCache, svgCache := createCaches(config)
	defer componentCache.StopAllGoroutines()
	defer svgCache.StopAllGoroutines()

	adapter := &DataLoaderRegistryAdapter{
		componentCache: componentCache,
		svgCache:       svgCache,
	}

	stats := adapter.GetStats()

	assert.Equal(t, 0, stats.ComponentCacheSize)
	assert.Equal(t, 0, stats.SVGCacheSize)
}

func TestGetComponentCacheSize_EmptyCache_ReturnsZero(t *testing.T) {
	t.Parallel()

	config := &DataLoaderAdapterConfig{
		ComponentCacheCapacity: 10,
		ComponentCacheTTL:      1 * time.Minute,
		SVGCacheCapacity:       10,
		SVGCacheTTL:            1 * time.Minute,
	}
	componentCache, svgCache := createCaches(config)
	defer componentCache.StopAllGoroutines()
	defer svgCache.StopAllGoroutines()

	adapter := &DataLoaderRegistryAdapter{
		componentCache: componentCache,
		svgCache:       svgCache,
	}

	assert.Equal(t, 0, adapter.GetComponentCacheSize())
}

func TestGetSVGCacheSize_EmptyCache_ReturnsZero(t *testing.T) {
	t.Parallel()

	config := &DataLoaderAdapterConfig{
		ComponentCacheCapacity: 10,
		ComponentCacheTTL:      1 * time.Minute,
		SVGCacheCapacity:       10,
		SVGCacheTTL:            1 * time.Minute,
	}
	componentCache, svgCache := createCaches(config)
	defer componentCache.StopAllGoroutines()
	defer svgCache.StopAllGoroutines()

	adapter := &DataLoaderRegistryAdapter{
		componentCache: componentCache,
		svgCache:       svgCache,
	}

	assert.Equal(t, 0, adapter.GetSVGCacheSize())
}

func TestBuildLookupSet(t *testing.T) {
	t.Parallel()

	t.Run("empty input returns empty set", func(t *testing.T) {
		t.Parallel()

		result := buildLookupSet(nil)

		assert.Empty(t, result)
	})

	t.Run("single item returns set with one entry", func(t *testing.T) {
		t.Parallel()

		result := buildLookupSet([]string{"alpha"})

		assert.Len(t, result, 1)
		_, exists := result["alpha"]
		assert.True(t, exists)
	})

	t.Run("multiple items returns set with all entries", func(t *testing.T) {
		t.Parallel()

		result := buildLookupSet([]string{"alpha", "beta", "gamma"})

		assert.Len(t, result, 3)
		for _, key := range []string{"alpha", "beta", "gamma"} {
			_, exists := result[key]
			assert.True(t, exists)
		}
	})

	t.Run("duplicates are deduplicated", func(t *testing.T) {
		t.Parallel()

		result := buildLookupSet([]string{"alpha", "alpha", "beta"})

		assert.Len(t, result, 2)
	})
}

func TestFindJSVariant(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for empty variants", func(t *testing.T) {
		t.Parallel()

		result := findJSVariant(nil)

		assert.Nil(t, result)
	})

	t.Run("returns nil when no component-js variant exists", func(t *testing.T) {
		t.Parallel()

		variants := []registry_dto.Variant{
			{
				VariantID:    "v1",
				MetadataTags: registry_dto.TagsFromMap(map[string]string{"type": "component-css"}),
			},
		}

		result := findJSVariant(variants)

		assert.Nil(t, result)
	})

	t.Run("returns nil when component-js exists but role is not entrypoint", func(t *testing.T) {
		t.Parallel()

		variants := []registry_dto.Variant{
			{
				VariantID:    "v1",
				MetadataTags: registry_dto.TagsFromMap(map[string]string{"type": "component-js", "role": "minified"}),
			},
		}

		result := findJSVariant(variants)

		assert.Nil(t, result)
	})

	t.Run("returns matching variant", func(t *testing.T) {
		t.Parallel()

		variants := []registry_dto.Variant{
			{
				VariantID:    "v1",
				MetadataTags: registry_dto.TagsFromMap(map[string]string{"type": "component-css"}),
			},
			{
				VariantID:    "v2",
				StorageKey:   "dist/component.js",
				MetadataTags: registry_dto.TagsFromMap(map[string]string{"type": "component-js", "role": "entrypoint"}),
			},
		}

		result := findJSVariant(variants)

		require.NotNil(t, result)
		assert.Equal(t, "v2", result.VariantID)
	})

	t.Run("returns first matching variant when multiple exist", func(t *testing.T) {
		t.Parallel()

		variants := []registry_dto.Variant{
			{
				VariantID:    "first",
				MetadataTags: registry_dto.TagsFromMap(map[string]string{"type": "component-js", "role": "entrypoint"}),
			},
			{
				VariantID:    "second",
				MetadataTags: registry_dto.TagsFromMap(map[string]string{"type": "component-js", "role": "entrypoint"}),
			},
		}

		result := findJSVariant(variants)

		require.NotNil(t, result)
		assert.Equal(t, "first", result.VariantID)
	})
}

func TestFindSVGVariant(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for empty variants", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{}

		result := findSVGVariant(artefact)

		assert.Nil(t, result)
	})

	t.Run("prefers minified-svg over source", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{
				{
					VariantID:    "source-v",
					MetadataTags: registry_dto.TagsFromMap(map[string]string{"type": "source"}),
				},
				{
					VariantID:    "minified-v",
					MetadataTags: registry_dto.TagsFromMap(map[string]string{"type": "minified-svg"}),
				},
			},
		}

		result := findSVGVariant(artefact)

		require.NotNil(t, result)
		assert.Equal(t, "minified-v", result.VariantID)
	})

	t.Run("falls back to source when no minified variant", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{
				{
					VariantID:    "source-v",
					MetadataTags: registry_dto.TagsFromMap(map[string]string{"type": "source"}),
				},
			},
		}

		result := findSVGVariant(artefact)

		require.NotNil(t, result)
		assert.Equal(t, "source-v", result.VariantID)
	})

	t.Run("returns nil when no matching type", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{
				{
					VariantID:    "v1",
					MetadataTags: registry_dto.TagsFromMap(map[string]string{"type": "component-js"}),
				},
			},
		}

		result := findSVGVariant(artefact)

		assert.Nil(t, result)
	})
}

func TestExtractTagContent(t *testing.T) {
	t.Parallel()

	t.Run("simple SVG", func(t *testing.T) {
		t.Parallel()

		tagContent, innerHTML := extractTagContent(`<svg viewBox="0 0 24 24"><path d="M12 2"></path></svg>`, "svg")

		assert.Equal(t, ` viewBox="0 0 24 24"`, tagContent)
		assert.Equal(t, `<path d="M12 2"></path>`, innerHTML)
	})

	t.Run("no matching tag returns empty tagContent and full HTML", func(t *testing.T) {
		t.Parallel()

		tagContent, innerHTML := extractTagContent(`<div>hello</div>`, "svg")

		assert.Equal(t, "", tagContent)
		assert.Equal(t, `<div>hello</div>`, innerHTML)
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		t.Parallel()

		tagContent, innerHTML := extractTagContent(`<SVG viewBox="0 0 24 24"><path d="M12 2"></path></SVG>`, "svg")

		assert.Equal(t, ` viewBox="0 0 24 24"`, tagContent)
		assert.Equal(t, `<path d="M12 2"></path>`, innerHTML)
	})

	t.Run("self-closing tag returns empty innerHTML", func(t *testing.T) {
		t.Parallel()

		tagContent, innerHTML := extractTagContent(`<svg viewBox="0 0 24 24"/>`, "svg")

		assert.Equal(t, ` viewBox="0 0 24 24"/`, tagContent)
		assert.Equal(t, "", innerHTML)
	})

	t.Run("empty input returns empty values", func(t *testing.T) {
		t.Parallel()

		tagContent, innerHTML := extractTagContent("", "svg")

		assert.Equal(t, "", tagContent)
		assert.Equal(t, "", innerHTML)
	})

	t.Run("whitespace around SVG", func(t *testing.T) {
		t.Parallel()

		tagContent, innerHTML := extractTagContent("  \n  <svg viewBox=\"0 0 24 24\"><path/></svg>  \n  ", "svg")

		assert.Equal(t, ` viewBox="0 0 24 24"`, tagContent)
		assert.Equal(t, "<path/>", innerHTML)
	})
}

func TestIndexFoldCase(t *testing.T) {
	t.Parallel()

	t.Run("empty substring returns 0", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 0, indexFoldCase("hello", ""))
	})

	t.Run("substring longer than string returns -1", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, -1, indexFoldCase("hi", "hello"))
	})

	t.Run("exact match returns 0", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 0, indexFoldCase("hello", "hello"))
	})

	t.Run("case insensitive match", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 0, indexFoldCase("HELLO", "hello"))
	})

	t.Run("match in middle", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 6, indexFoldCase("hello world", "world"))
	})

	t.Run("no match returns -1", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, -1, indexFoldCase("hello", "xyz"))
	})
}

func TestLastIndexFoldCase(t *testing.T) {
	t.Parallel()

	t.Run("empty substring returns string length", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 5, lastIndexFoldCase("hello", ""))
	})

	t.Run("substring longer than string returns -1", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, -1, lastIndexFoldCase("hi", "hello"))
	})

	t.Run("finds last occurrence", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 6, lastIndexFoldCase("ab ab ab", "ab"))
	})

	t.Run("case insensitive last occurrence", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 6, lastIndexFoldCase("AB ab AB", "ab"))
	})

	t.Run("no match returns -1", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, -1, lastIndexFoldCase("hello", "xyz"))
	})
}

func TestExtractComponentMetadata(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when no JS variant", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{
				{MetadataTags: registry_dto.TagsFromMap(map[string]string{"type": "component-css"})},
			},
		}

		result := extractComponentMetadata(artefact, map[string]struct{}{}, "/artefacts")

		assert.Nil(t, result)
	})

	t.Run("returns nil when JS variant has no tag name", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{
				{MetadataTags: registry_dto.TagsFromMap(map[string]string{"type": "component-js", "role": "entrypoint"})},
			},
		}

		result := extractComponentMetadata(artefact, map[string]struct{}{}, "/artefacts")

		assert.Nil(t, result)
	})

	t.Run("returns nil when tag name not in requested set", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{
				{MetadataTags: registry_dto.TagsFromMap(map[string]string{
					"type":    "component-js",
					"role":    "entrypoint",
					"tagName": "my-widget",
				})},
			},
		}

		result := extractComponentMetadata(artefact, map[string]struct{}{"other-widget": {}}, "/artefacts")

		assert.Nil(t, result)
	})

	t.Run("returns metadata when tag name is requested", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{
				{
					StorageKey: "dist/widget.js",
					MetadataTags: registry_dto.TagsFromMap(map[string]string{
						"type":    "component-js",
						"role":    "entrypoint",
						"tagName": "my-widget",
					}),
				},
			},
		}

		result := extractComponentMetadata(artefact, map[string]struct{}{"my-widget": {}}, "/artefacts")

		require.NotNil(t, result)
		assert.Equal(t, "my-widget", result.TagName)
		assert.Equal(t, "/artefacts/dist/widget.js", result.BaseJSPath)
	})
}

func TestBuildComponentResults(t *testing.T) {
	t.Parallel()

	t.Run("empty artefacts returns empty map", func(t *testing.T) {
		t.Parallel()

		result := buildComponentResults(nil, map[string]struct{}{"test": {}}, "/artefacts")

		assert.Empty(t, result)
	})

	t.Run("filters artefacts to requested types", func(t *testing.T) {
		t.Parallel()

		artefacts := []*registry_dto.ArtefactMeta{
			{
				ActualVariants: []registry_dto.Variant{
					{
						StorageKey: "dist/a.js",
						MetadataTags: registry_dto.TagsFromMap(map[string]string{
							"type":    "component-js",
							"role":    "entrypoint",
							"tagName": "widget-a",
						}),
					},
				},
			},
			{
				ActualVariants: []registry_dto.Variant{
					{
						StorageKey: "dist/b.js",
						MetadataTags: registry_dto.TagsFromMap(map[string]string{
							"type":    "component-js",
							"role":    "entrypoint",
							"tagName": "widget-b",
						}),
					},
				},
			},
		}

		requested := map[string]struct{}{"widget-a": {}}
		result := buildComponentResults(artefacts, requested, "/artefacts")

		assert.Len(t, result, 1)
		assert.Contains(t, result, "widget-a")
	})
}

func TestSVGBufferPool_GetAndPut(t *testing.T) {
	t.Parallel()

	buffer := svgBufferPool.Get()
	require.NotNil(t, buffer)

	bufferPtr, ok := buffer.(*[]byte)
	require.True(t, ok)
	assert.Equal(t, 0, len(*bufferPtr))
	assert.Equal(t, defaultSVGBufferCapacity, cap(*bufferPtr))

	svgBufferPool.Put(bufferPtr)
}

func TestGetRawSVGFromArtefactZeroCopy_NilArtefact_ReturnsError(t *testing.T) {
	t.Parallel()

	var frozenBuffers []*[]byte

	result, err := getRawSVGFromArtefactZeroCopy(context.Background(), nil, nil, &frozenBuffers)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil artefact")
	assert.Empty(t, result)
}

func TestGetRawSVGFromArtefactZeroCopy_NoSuitableVariant_ReturnsError(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		ID: "test-svg",
		ActualVariants: []registry_dto.Variant{
			{MetadataTags: registry_dto.TagsFromMap(map[string]string{"type": "component-js"})},
		},
	}
	var frozenBuffers []*[]byte

	result, err := getRawSVGFromArtefactZeroCopy(context.Background(), nil, artefact, &frozenBuffers)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no suitable variant")
	assert.Empty(t, result)
}
