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

package templater_domain_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/templater/templater_domain"
)

func TestMockManifestStoreView_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m templater_domain.MockManifestStoreView

	keys := m.GetKeys()
	assert.Nil(t, keys)

	entry, ok := m.GetPageEntry("anything")
	assert.Nil(t, entry)
	assert.False(t, ok)
}

func TestMockManifestStoreView_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &templater_domain.MockManifestStoreView{
		GetKeysFunc: func() []string { return []string{"pages/home.pk"} },
		GetPageEntryFunc: func(_ string) (templater_domain.PageEntryView, bool) {
			return &templater_domain.MockPageEntryView{}, true
		},
	}

	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			m.GetKeys()
			m.GetPageEntry("pages/home.pk")
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetKeysCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetPageEntryCallCount))
}

func TestMockManifestStoreView_GetKeys(t *testing.T) {
	t.Parallel()

	t.Run("nil GetKeysFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockManifestStoreView{}
		got := m.GetKeys()
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetKeysCallCount))
	})

	t.Run("delegates to GetKeysFunc", func(t *testing.T) {
		t.Parallel()
		expected := []string{"pages/home.pk", "partials/header.pk"}
		m := &templater_domain.MockManifestStoreView{
			GetKeysFunc: func() []string { return expected },
		}
		got := m.GetKeys()
		assert.Equal(t, expected, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetKeysCallCount))
	})
}

func TestMockManifestStoreView_GetPageEntry(t *testing.T) {
	t.Parallel()

	t.Run("nil GetPageEntryFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockManifestStoreView{}
		entry, ok := m.GetPageEntry("pages/home.pk")
		assert.Nil(t, entry)
		assert.False(t, ok)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetPageEntryCallCount))
	})

	t.Run("delegates to GetPageEntryFunc", func(t *testing.T) {
		t.Parallel()
		expectedEntry := &templater_domain.MockPageEntryView{
			GetOriginalPathFunc: func() string { return "pages/home.pk" },
		}
		m := &templater_domain.MockManifestStoreView{
			GetPageEntryFunc: func(path string) (templater_domain.PageEntryView, bool) {
				require.Equal(t, "pages/home.pk", path)
				return expectedEntry, true
			},
		}
		entry, ok := m.GetPageEntry("pages/home.pk")
		assert.True(t, ok)
		assert.Equal(t, "pages/home.pk", entry.GetOriginalPath())
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetPageEntryCallCount))
	})
}
