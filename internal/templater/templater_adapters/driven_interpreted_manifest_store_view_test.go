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

package templater_adapters

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindErrorPageTier(t *testing.T) {
	t.Parallel()

	rootCatchAll := &PageEntry{ErrorDispatch: &ErrorPageDispatch{ScopePath: "/", IsCatchAll: true}}
	appCatchAll := &PageEntry{ErrorDispatch: &ErrorPageDispatch{ScopePath: "/app/", IsCatchAll: true}}
	exact404 := &PageEntry{ErrorDispatch: &ErrorPageDispatch{ScopePath: "/", StatusCode: 404}}
	range4xx := &PageEntry{ErrorDispatch: &ErrorPageDispatch{ScopePath: "/", StatusCodeMin: 400, StatusCodeMax: 499}}

	cache := map[string]*PageEntry{
		"pages/!error.pk":     rootCatchAll,
		"pages/app/!error.pk": appCatchAll,
		"pages/!404.pk":       exact404,
		"pages/!400-499.pk":   range4xx,
		"pages/home.pk":       {},
	}

	t.Run("prefers longer scope on catch-all", func(t *testing.T) {
		t.Parallel()
		entry, ok := findErrorPageTier(cache, "/app/settings", func(d *ErrorPageDispatch) bool {
			return d.IsCatchAll
		})
		assert.True(t, ok)
		assert.Same(t, appCatchAll, entry)
	})

	t.Run("falls back to root scope when request is outside deeper scope", func(t *testing.T) {
		t.Parallel()
		entry, ok := findErrorPageTier(cache, "/docs", func(d *ErrorPageDispatch) bool {
			return d.IsCatchAll
		})
		assert.True(t, ok)
		assert.Same(t, rootCatchAll, entry)
	})

	t.Run("matches exact status code tier", func(t *testing.T) {
		t.Parallel()
		entry, ok := findErrorPageTier(cache, "/any", func(d *ErrorPageDispatch) bool {
			return !d.IsCatchAll && d.StatusCodeMin == 0 && d.StatusCodeMax == 0 && d.StatusCode == 404
		})
		assert.True(t, ok)
		assert.Same(t, exact404, entry)
	})

	t.Run("matches range tier", func(t *testing.T) {
		t.Parallel()
		entry, ok := findErrorPageTier(cache, "/any", func(d *ErrorPageDispatch) bool {
			return !d.IsCatchAll && d.StatusCodeMin > 0 && d.StatusCodeMax > 0 &&
				403 >= d.StatusCodeMin && 403 <= d.StatusCodeMax
		})
		assert.True(t, ok)
		assert.Same(t, range4xx, entry)
	})

	t.Run("returns not ok when no entry matches the predicate", func(t *testing.T) {
		t.Parallel()
		_, ok := findErrorPageTier(cache, "/any", func(d *ErrorPageDispatch) bool {
			return d.StatusCode == 999
		})
		assert.False(t, ok)
	})

	t.Run("ignores entries without ErrorDispatch", func(t *testing.T) {
		t.Parallel()
		onlyPages := map[string]*PageEntry{
			"pages/home.pk": {},
		}
		_, ok := findErrorPageTier(onlyPages, "/home", func(d *ErrorPageDispatch) bool {
			return true
		})
		assert.False(t, ok)
	})
}

func TestInterpretedManifestStoreViewFindErrorPage(t *testing.T) {
	t.Parallel()

	rootCatchAll := &PageEntry{ErrorDispatch: &ErrorPageDispatch{ScopePath: "/", IsCatchAll: true}}
	appExact404 := &PageEntry{ErrorDispatch: &ErrorPageDispatch{ScopePath: "/app/", StatusCode: 404}}
	range4xx := &PageEntry{ErrorDispatch: &ErrorPageDispatch{ScopePath: "/", StatusCodeMin: 400, StatusCodeMax: 499}}

	runner := &InterpretedManifestRunner{
		progCache: map[string]*PageEntry{
			"pages/!error.pk":   rootCatchAll,
			"pages/app/!404.pk": appExact404,
			"pages/!400-499.pk": range4xx,
		},
		cacheLock: sync.RWMutex{},
	}
	view := &interpretedManifestStoreView{r: runner}

	t.Run("prefers exact tier over range and catch-all", func(t *testing.T) {
		t.Parallel()
		entry, ok := view.FindErrorPage(404, "/app/settings/missing")
		assert.True(t, ok)
		assert.Same(t, appExact404, entry)
	})

	t.Run("falls back to range tier when no exact match", func(t *testing.T) {
		t.Parallel()
		entry, ok := view.FindErrorPage(403, "/anything")
		assert.True(t, ok)
		assert.Same(t, range4xx, entry)
	})

	t.Run("falls back to catch-all when no status-specific match", func(t *testing.T) {
		t.Parallel()
		entry, ok := view.FindErrorPage(503, "/anywhere")
		assert.True(t, ok)
		assert.Same(t, rootCatchAll, entry)
	})

	t.Run("returns false when no error page entries exist", func(t *testing.T) {
		t.Parallel()
		empty := &InterpretedManifestRunner{
			progCache: map[string]*PageEntry{"pages/home.pk": {}},
			cacheLock: sync.RWMutex{},
		}
		emptyView := &interpretedManifestStoreView{r: empty}
		_, ok := emptyView.FindErrorPage(404, "/missing")
		assert.False(t, ok)
	})
}
