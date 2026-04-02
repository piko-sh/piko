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

package mocks

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/render/render_dto"
)

type spyTB struct {
	testing.TB
	errors       []string
	fatals       []string
	helperCalled int64
}

func (s *spyTB) Helper() { atomic.AddInt64(&s.helperCalled, 1) }
func (s *spyTB) Errorf(format string, arguments ...any) {
	s.errors = append(s.errors, format)
	_ = arguments
}
func (s *spyTB) Fatalf(format string, arguments ...any) {
	s.fatals = append(s.fatals, format)
	_ = arguments
}

func TestNewMockRegistry(t *testing.T) {
	t.Parallel()

	t.Run("returns non-nil registry with initialised maps", func(t *testing.T) {
		t.Parallel()
		m := NewMockRegistry(t)

		require.NotNil(t, m)

		assert.NotNil(t, m.GetComponentMetadataFunc)
		assert.NotNil(t, m.GetAssetRawSVGFunc)
		assert.NotNil(t, m.BulkGetAssetRawSVGFunc)
		assert.NotNil(t, m.BulkGetComponentMetadataFunc)
	})
}

func TestMockRegistry_OnGetComponent(t *testing.T) {
	t.Parallel()

	t.Run("nil OnGetComponent result causes error from GetComponentMetadata", func(t *testing.T) {
		t.Parallel()
		m := NewMockRegistry(t)

		got, err := m.GetComponentMetadata(context.Background(), "unknown")

		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown")
	})

	t.Run("delegates to registered component result", func(t *testing.T) {
		t.Parallel()
		m := NewMockRegistry(t)
		want := &render_dto.ComponentMetadata{TagName: "piko-card"}
		m.OnGetComponent("card", want)

		got, err := m.GetComponentMetadata(context.Background(), "card")

		require.NoError(t, err)
		assert.Same(t, want, got)
	})
}

func TestMockRegistry_OnGetSVG(t *testing.T) {
	t.Parallel()

	t.Run("nil OnGetSVG result causes error from GetAssetRawSVG", func(t *testing.T) {
		t.Parallel()
		m := NewMockRegistry(t)

		got, err := m.GetAssetRawSVG(context.Background(), "missing-icon")

		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing-icon")
	})

	t.Run("delegates to registered SVG result", func(t *testing.T) {
		t.Parallel()
		m := NewMockRegistry(t)
		svgData := &render_domain.ParsedSvgData{InnerHTML: "<path d='M0 0'/>"}
		m.OnGetSVG("icon-home", svgData)

		got, err := m.GetAssetRawSVG(context.Background(), "icon-home")

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "<path d='M0 0'/>", got.InnerHTML)
	})

	t.Run("pre-computes CachedSymbol when empty", func(t *testing.T) {
		t.Parallel()
		m := NewMockRegistry(t)
		svgData := &render_domain.ParsedSvgData{InnerHTML: "<rect/>"}
		m.OnGetSVG("icon-rect", svgData)

		got, err := m.GetAssetRawSVG(context.Background(), "icon-rect")

		require.NoError(t, err)
		require.NotNil(t, got)

		assert.NotEmpty(t, got.CachedSymbol)
	})

	t.Run("preserves existing CachedSymbol", func(t *testing.T) {
		t.Parallel()
		m := NewMockRegistry(t)
		svgData := &render_domain.ParsedSvgData{
			InnerHTML:    "<circle/>",
			CachedSymbol: "pre-computed-value",
		}
		m.OnGetSVG("icon-circle", svgData)

		got, err := m.GetAssetRawSVG(context.Background(), "icon-circle")

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "pre-computed-value", got.CachedSymbol)
	})
}

func TestMockRegistry_AssertComponentCalled(t *testing.T) {
	t.Parallel()

	t.Run("no calls reports zero", func(t *testing.T) {
		t.Parallel()
		spy := &spyTB{}
		m := NewMockRegistry(spy)

		m.AssertComponentCalled("never", 0)

		assert.Empty(t, spy.errors, "expected no error when assertion matches")
	})

	t.Run("delegates count correctly after calls", func(t *testing.T) {
		t.Parallel()
		m := NewMockRegistry(t)
		want := &render_dto.ComponentMetadata{TagName: "piko-btn"}
		m.OnGetComponent("btn", want)

		_, _ = m.GetComponentMetadata(context.Background(), "btn")
		_, _ = m.GetComponentMetadata(context.Background(), "btn")

		m.AssertComponentCalled("btn", 2)
	})

	t.Run("reports mismatch via Errorf", func(t *testing.T) {
		t.Parallel()
		spy := &spyTB{}
		m := NewMockRegistry(spy)
		m.OnGetComponent("c", &render_dto.ComponentMetadata{TagName: "piko-c"})

		_, _ = m.GetComponentMetadata(context.Background(), "c")

		m.AssertComponentCalled("c", 5)
		assert.NotEmpty(t, spy.errors, "expected Errorf to have been called on mismatch")
	})
}

func TestMockRegistry_BulkGetComponentMetadata(t *testing.T) {
	t.Parallel()

	t.Run("nil BulkGetComponentMetadataFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := NewMockRegistry(t)

		got, err := m.BulkGetComponentMetadata(context.Background(), []string{"a", "b"})

		require.NoError(t, err)

		assert.Empty(t, got)
	})

	t.Run("delegates to registered component results", func(t *testing.T) {
		t.Parallel()
		m := NewMockRegistry(t)
		m.OnGetComponent("a", &render_dto.ComponentMetadata{TagName: "piko-a"})
		m.OnGetComponent("b", &render_dto.ComponentMetadata{TagName: "piko-b"})

		got, err := m.BulkGetComponentMetadata(context.Background(), []string{"a", "b", "c"})

		require.NoError(t, err)
		assert.Len(t, got, 2)
		assert.Equal(t, "piko-a", got["a"].TagName)
		assert.Equal(t, "piko-b", got["b"].TagName)
	})
}

func TestMockRegistry_BulkGetAssetRawSVG(t *testing.T) {
	t.Parallel()

	t.Run("nil BulkGetAssetRawSVGFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := NewMockRegistry(t)

		got, err := m.BulkGetAssetRawSVG(context.Background(), []string{"x", "y"})

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("delegates to registered SVG results", func(t *testing.T) {
		t.Parallel()
		m := NewMockRegistry(t)
		m.OnGetSVG("icon-a", &render_domain.ParsedSvgData{InnerHTML: "<a/>"})
		m.OnGetSVG("icon-b", &render_domain.ParsedSvgData{InnerHTML: "<b/>"})

		got, err := m.BulkGetAssetRawSVG(context.Background(), []string{"icon-a", "icon-b", "icon-c"})

		require.NoError(t, err)
		assert.Len(t, got, 2)
		assert.Equal(t, "<a/>", got["icon-a"].InnerHTML)
		assert.Equal(t, "<b/>", got["icon-b"].InnerHTML)
	})
}

func TestNewMockCSRF(t *testing.T) {
	t.Parallel()

	t.Run("GenerateCSRFPair returns fixed tokens", func(t *testing.T) {
		t.Parallel()
		csrf := NewMockCSRF()

		var buffer bytes.Buffer
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		pair, err := csrf.GenerateCSRFPair(w, r, &buffer)

		require.NoError(t, err)
		assert.Equal(t, "mock-ephemeral-token", pair.RawEphemeralToken)
		assert.Equal(t, "mock-action-token-payload^mock-signature", string(pair.ActionToken))
	})

	t.Run("ValidateCSRFPair returns true", func(t *testing.T) {
		t.Parallel()
		csrf := NewMockCSRF()

		r := httptest.NewRequest(http.MethodPost, "/", nil)
		valid, err := csrf.ValidateCSRFPair(r, "any-token", []byte("any-action"))

		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("Name returns mock-csrf", func(t *testing.T) {
		t.Parallel()
		csrf := NewMockCSRF()

		assert.Equal(t, "mock-csrf", csrf.Name())
	})
}

func TestMockRegistry_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m MockRegistry
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
	assert.Equal(t, render_domain.RegistryAdapterStats{}, stats)

	m.ClearComponentCache(ctx, "x")
	m.ClearSvgCache(ctx, "x")

	got5, err5 := m.UpsertArtefact(ctx, "x", "/p", nil, "s3", nil)
	assert.Nil(t, got5)
	assert.NoError(t, err5)
}

func TestMockRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := NewMockRegistry(t)
	m.OnGetComponent("card", &render_dto.ComponentMetadata{TagName: "piko-card"})
	m.OnGetSVG("icon-star", &render_domain.ParsedSvgData{InnerHTML: "<star/>"})

	const goroutines = 50
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			_, _ = m.GetComponentMetadata(ctx, "card")
			_, _ = m.BulkGetComponentMetadata(ctx, []string{"card"})
			_, _ = m.GetAssetRawSVG(ctx, "icon-star")
			_, _ = m.BulkGetAssetRawSVG(ctx, []string{"icon-star"})
			_ = m.GetStats()
			m.ClearComponentCache(ctx, "card")
			m.ClearSvgCache(ctx, "icon-star")

			m.OnGetComponent("card", &render_dto.ComponentMetadata{TagName: "piko-card"})
			m.OnGetSVG("icon-star", &render_domain.ParsedSvgData{InnerHTML: "<star/>"})
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
}
