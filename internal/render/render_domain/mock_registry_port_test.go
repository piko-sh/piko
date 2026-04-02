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

package render_domain

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_dto"
)

func TestMockRegistryPort_GetComponentMetadata(t *testing.T) {
	t.Parallel()

	t.Run("nil GetComponentMetadataFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryPort{}

		got, err := m.GetComponentMetadata(context.Background(), "card")

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetComponentMetadataCallCount))
	})

	t.Run("delegates to GetComponentMetadataFunc", func(t *testing.T) {
		t.Parallel()
		want := &render_dto.ComponentMetadata{TagName: "piko-card"}
		m := &MockRegistryPort{
			GetComponentMetadataFunc: func(_ context.Context, componentType string) (*render_dto.ComponentMetadata, error) {
				assert.Equal(t, "card", componentType)
				return want, nil
			},
		}

		got, err := m.GetComponentMetadata(context.Background(), "card")

		require.NoError(t, err)
		assert.Same(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetComponentMetadataCallCount))
	})

	t.Run("propagates error from GetComponentMetadataFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("registry unavailable")
		m := &MockRegistryPort{
			GetComponentMetadataFunc: func(context.Context, string) (*render_dto.ComponentMetadata, error) {
				return nil, wantErr
			},
		}

		got, err := m.GetComponentMetadata(context.Background(), "card")

		assert.Nil(t, got)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockRegistryPort_BulkGetComponentMetadata(t *testing.T) {
	t.Parallel()

	t.Run("nil BulkGetComponentMetadataFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryPort{}

		got, err := m.BulkGetComponentMetadata(context.Background(), []string{"a", "b"})

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.BulkGetComponentMetadataCallCount))
	})

	t.Run("delegates to BulkGetComponentMetadataFunc", func(t *testing.T) {
		t.Parallel()
		want := map[string]*render_dto.ComponentMetadata{
			"a": {TagName: "piko-a"},
		}
		m := &MockRegistryPort{
			BulkGetComponentMetadataFunc: func(_ context.Context, types []string) (map[string]*render_dto.ComponentMetadata, error) {
				assert.Equal(t, []string{"a", "b"}, types)
				return want, nil
			},
		}

		got, err := m.BulkGetComponentMetadata(context.Background(), []string{"a", "b"})

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.BulkGetComponentMetadataCallCount))
	})

	t.Run("propagates error from BulkGetComponentMetadataFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("bulk fetch failed")
		m := &MockRegistryPort{
			BulkGetComponentMetadataFunc: func(context.Context, []string) (map[string]*render_dto.ComponentMetadata, error) {
				return nil, wantErr
			},
		}

		got, err := m.BulkGetComponentMetadata(context.Background(), []string{"x"})

		assert.Nil(t, got)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockRegistryPort_GetAssetRawSVG(t *testing.T) {
	t.Parallel()

	t.Run("nil GetAssetRawSVGFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryPort{}

		got, err := m.GetAssetRawSVG(context.Background(), "icon-home")

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetAssetRawSVGCallCount))
	})

	t.Run("delegates to GetAssetRawSVGFunc", func(t *testing.T) {
		t.Parallel()
		want := &ParsedSvgData{InnerHTML: "<path/>"}
		m := &MockRegistryPort{
			GetAssetRawSVGFunc: func(_ context.Context, assetID string) (*ParsedSvgData, error) {
				assert.Equal(t, "icon-home", assetID)
				return want, nil
			},
		}

		got, err := m.GetAssetRawSVG(context.Background(), "icon-home")

		require.NoError(t, err)
		assert.Same(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetAssetRawSVGCallCount))
	})

	t.Run("propagates error from GetAssetRawSVGFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("SVG not found")
		m := &MockRegistryPort{
			GetAssetRawSVGFunc: func(context.Context, string) (*ParsedSvgData, error) {
				return nil, wantErr
			},
		}

		got, err := m.GetAssetRawSVG(context.Background(), "missing")

		assert.Nil(t, got)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockRegistryPort_BulkGetAssetRawSVG(t *testing.T) {
	t.Parallel()

	t.Run("nil BulkGetAssetRawSVGFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryPort{}

		got, err := m.BulkGetAssetRawSVG(context.Background(), []string{"a", "b"})

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.BulkGetAssetRawSVGCallCount))
	})

	t.Run("delegates to BulkGetAssetRawSVGFunc", func(t *testing.T) {
		t.Parallel()
		want := map[string]*ParsedSvgData{
			"icon-a": {InnerHTML: "<circle/>"},
		}
		m := &MockRegistryPort{
			BulkGetAssetRawSVGFunc: func(_ context.Context, ids []string) (map[string]*ParsedSvgData, error) {
				assert.Equal(t, []string{"icon-a"}, ids)
				return want, nil
			},
		}

		got, err := m.BulkGetAssetRawSVG(context.Background(), []string{"icon-a"})

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.BulkGetAssetRawSVGCallCount))
	})

	t.Run("propagates error from BulkGetAssetRawSVGFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("bulk SVG failed")
		m := &MockRegistryPort{
			BulkGetAssetRawSVGFunc: func(context.Context, []string) (map[string]*ParsedSvgData, error) {
				return nil, wantErr
			},
		}

		got, err := m.BulkGetAssetRawSVG(context.Background(), []string{"x"})

		assert.Nil(t, got)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockRegistryPort_GetStats(t *testing.T) {
	t.Parallel()

	t.Run("nil GetStatsFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryPort{}

		got := m.GetStats()

		assert.Equal(t, RegistryAdapterStats{}, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetStatsCallCount))
	})

	t.Run("delegates to GetStatsFunc", func(t *testing.T) {
		t.Parallel()
		want := RegistryAdapterStats{ComponentCacheSize: 42, SVGCacheSize: 7}
		m := &MockRegistryPort{
			GetStatsFunc: func() RegistryAdapterStats {
				return want
			},
		}

		got := m.GetStats()

		assert.Equal(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetStatsCallCount))
	})
}

func TestMockRegistryPort_ClearComponentCache(t *testing.T) {
	t.Parallel()

	t.Run("nil ClearComponentCacheFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryPort{}

		m.ClearComponentCache(context.Background(), "card")

		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ClearComponentCacheCallCount))
	})

	t.Run("delegates to ClearComponentCacheFunc", func(t *testing.T) {
		t.Parallel()
		var calledWith string
		m := &MockRegistryPort{
			ClearComponentCacheFunc: func(_ context.Context, componentType string) {
				calledWith = componentType
			},
		}

		m.ClearComponentCache(context.Background(), "navbar")

		assert.Equal(t, "navbar", calledWith)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ClearComponentCacheCallCount))
	})
}

func TestMockRegistryPort_ClearSvgCache(t *testing.T) {
	t.Parallel()

	t.Run("nil ClearSvgCacheFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryPort{}

		m.ClearSvgCache(context.Background(), "icon-star")

		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ClearSvgCacheCallCount))
	})

	t.Run("delegates to ClearSvgCacheFunc", func(t *testing.T) {
		t.Parallel()
		var calledWith string
		m := &MockRegistryPort{
			ClearSvgCacheFunc: func(_ context.Context, svgID string) {
				calledWith = svgID
			},
		}

		m.ClearSvgCache(context.Background(), "icon-star")

		assert.Equal(t, "icon-star", calledWith)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ClearSvgCacheCallCount))
	})
}

func TestMockRegistryPort_UpsertArtefact(t *testing.T) {
	t.Parallel()

	t.Run("nil UpsertArtefactFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRegistryPort{}

		got, err := m.UpsertArtefact(context.Background(), "art-1", "/src/img.png", strings.NewReader("data"), "s3", nil)

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.UpsertArtefactCallCount))
	})

	t.Run("delegates to UpsertArtefactFunc", func(t *testing.T) {
		t.Parallel()
		want := &registry_dto.ArtefactMeta{ID: "art-1"}
		profiles := []registry_dto.NamedProfile{{Name: "thumb"}}
		m := &MockRegistryPort{
			UpsertArtefactFunc: func(_ context.Context, artefactID, sourcePath string, _ io.Reader, storageBackendID string, desiredProfiles []registry_dto.NamedProfile) (*registry_dto.ArtefactMeta, error) {
				assert.Equal(t, "art-1", artefactID)
				assert.Equal(t, "/src/img.png", sourcePath)
				assert.Equal(t, "s3", storageBackendID)
				assert.Equal(t, profiles, desiredProfiles)
				return want, nil
			},
		}

		got, err := m.UpsertArtefact(context.Background(), "art-1", "/src/img.png", strings.NewReader("data"), "s3", profiles)

		require.NoError(t, err)
		assert.Same(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.UpsertArtefactCallCount))
	})

	t.Run("propagates error from UpsertArtefactFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("upload failed")
		m := &MockRegistryPort{
			UpsertArtefactFunc: func(context.Context, string, string, io.Reader, string, []registry_dto.NamedProfile) (*registry_dto.ArtefactMeta, error) {
				return nil, wantErr
			},
		}

		got, err := m.UpsertArtefact(context.Background(), "art-1", "/x", nil, "s3", nil)

		assert.Nil(t, got)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockRegistryPort_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m MockRegistryPort
	ctx := context.Background()

	got1, err1 := m.GetComponentMetadata(ctx, "x")
	assert.Nil(t, got1)
	assert.NoError(t, err1)

	got2, err2 := m.BulkGetComponentMetadata(ctx, []string{"x"})
	assert.Nil(t, got2)
	assert.NoError(t, err2)

	got3, err3 := m.GetAssetRawSVG(ctx, "x")
	assert.Nil(t, got3)
	assert.NoError(t, err3)

	got4, err4 := m.BulkGetAssetRawSVG(ctx, []string{"x"})
	assert.Nil(t, got4)
	assert.NoError(t, err4)

	stats := m.GetStats()
	assert.Equal(t, RegistryAdapterStats{}, stats)

	m.ClearComponentCache(ctx, "x")
	m.ClearSvgCache(ctx, "x")

	got5, err5 := m.UpsertArtefact(ctx, "x", "/p", nil, "s3", nil)
	assert.Nil(t, got5)
	assert.NoError(t, err5)

	assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetComponentMetadataCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.BulkGetComponentMetadataCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetAssetRawSVGCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.BulkGetAssetRawSVGCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetStatsCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.ClearComponentCacheCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.ClearSvgCacheCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.UpsertArtefactCallCount))
}

func TestMockRegistryPort_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &MockRegistryPort{
		GetComponentMetadataFunc: func(context.Context, string) (*render_dto.ComponentMetadata, error) {
			return &render_dto.ComponentMetadata{TagName: "piko-c"}, nil
		},
		BulkGetComponentMetadataFunc: func(context.Context, []string) (map[string]*render_dto.ComponentMetadata, error) {
			return map[string]*render_dto.ComponentMetadata{}, nil
		},
		GetAssetRawSVGFunc: func(context.Context, string) (*ParsedSvgData, error) {
			return &ParsedSvgData{InnerHTML: "<path/>"}, nil
		},
		BulkGetAssetRawSVGFunc: func(context.Context, []string) (map[string]*ParsedSvgData, error) {
			return map[string]*ParsedSvgData{}, nil
		},
		GetStatsFunc: func() RegistryAdapterStats {
			return RegistryAdapterStats{ComponentCacheSize: 1}
		},
		ClearComponentCacheFunc: func(context.Context, string) {},
		ClearSvgCacheFunc:       func(context.Context, string) {},
		UpsertArtefactFunc: func(context.Context, string, string, io.Reader, string, []registry_dto.NamedProfile) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{ID: "a"}, nil
		},
	}

	const goroutines = 50
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			_, _ = m.GetComponentMetadata(ctx, "c")
			_, _ = m.BulkGetComponentMetadata(ctx, []string{"c"})
			_, _ = m.GetAssetRawSVG(ctx, "s")
			_, _ = m.BulkGetAssetRawSVG(ctx, []string{"s"})
			_ = m.GetStats()
			m.ClearComponentCache(ctx, "c")
			m.ClearSvgCache(ctx, "s")
			_, _ = m.UpsertArtefact(ctx, "a", "/p", nil, "s3", nil)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetComponentMetadataCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.BulkGetComponentMetadataCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetAssetRawSVGCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.BulkGetAssetRawSVGCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetStatsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ClearComponentCacheCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ClearSvgCacheCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.UpsertArtefactCallCount))
}
