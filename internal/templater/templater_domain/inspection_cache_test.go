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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/templater/templater_domain"
)

func TestInMemoryInspectionCache_StoreAndGet(t *testing.T) {
	t.Parallel()

	cache := templater_domain.NewInMemoryInspectionCache()
	path := "pages/home.pk"
	scriptHash := "abc123"
	result := &templater_domain.InspectionResult{
		Component: &annotator_dto.VirtualComponent{
			HashedName: "test_component",
			IsPage:     true,
			IsPublic:   true,
		},
		Timestamp:  time.Now(),
		ScriptHash: scriptHash,
	}

	cache.Store(path, scriptHash, result)
	retrieved, found := cache.Get(path, scriptHash)

	assert.True(t, found, "should find stored result")
	require.NotNil(t, retrieved)
	assert.Equal(t, result.ScriptHash, retrieved.ScriptHash)
	assert.Equal(t, result.Component.HashedName, retrieved.Component.HashedName)
	assert.Equal(t, result.Component.IsPage, retrieved.Component.IsPage)
}

func TestInMemoryInspectionCache_GetNonExistent(t *testing.T) {
	t.Parallel()

	cache := templater_domain.NewInMemoryInspectionCache()

	result1, found1 := cache.Get("nonexistent/path.pk", "hash1")

	cache.Store("pages/home.pk", "hash1", &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "test"},
		Timestamp:  time.Now(),
		ScriptHash: "hash1",
	})
	result2, found2 := cache.Get("pages/home.pk", "different-hash")

	assert.False(t, found1, "should not find non-existent path")
	assert.Nil(t, result1)

	assert.False(t, found2, "should not find non-existent hash for existing path")
	assert.Nil(t, result2)
}

func TestInMemoryInspectionCache_StoreOverwrite(t *testing.T) {
	t.Parallel()

	cache := templater_domain.NewInMemoryInspectionCache()
	path := "pages/home.pk"
	scriptHash := "abc123"

	originalResult := &templater_domain.InspectionResult{
		Component: &annotator_dto.VirtualComponent{
			HashedName: "original",
			IsPage:     true,
		},
		Timestamp:  time.Now(),
		ScriptHash: scriptHash,
	}

	updatedResult := &templater_domain.InspectionResult{
		Component: &annotator_dto.VirtualComponent{
			HashedName: "updated",
			IsPage:     true,
			IsEmail:    true,
		},
		Timestamp:  time.Now(),
		ScriptHash: scriptHash,
	}

	cache.Store(path, scriptHash, originalResult)
	stats1 := cache.Stats()

	cache.Store(path, scriptHash, updatedResult)
	stats2 := cache.Stats()

	retrieved, found := cache.Get(path, scriptHash)

	assert.True(t, found)
	require.NotNil(t, retrieved)
	assert.Equal(t, "updated", retrieved.Component.HashedName, "should have updated component")
	assert.True(t, retrieved.Component.IsEmail, "should have updated email flag")

	assert.Equal(t, int64(0), stats1.Evictions, "first store should not be an eviction")
	assert.Equal(t, int64(1), stats2.Evictions, "overwrite should count as eviction")
}

func TestInMemoryInspectionCache_MultiplePaths(t *testing.T) {
	t.Parallel()

	cache := templater_domain.NewInMemoryInspectionCache()

	path1 := "pages/home.pk"
	path2 := "pages/about.pk"
	path3 := "components/header.pk"

	hash1 := "hash1"
	hash2 := "hash2"

	result1 := &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "comp1"},
		Timestamp:  time.Now(),
		ScriptHash: hash1,
	}
	result2 := &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "comp2"},
		Timestamp:  time.Now(),
		ScriptHash: hash2,
	}
	result3 := &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "comp3"},
		Timestamp:  time.Now(),
		ScriptHash: hash1,
	}

	cache.Store(path1, hash1, result1)
	cache.Store(path1, hash2, result2)
	cache.Store(path2, hash1, result3)

	r1, found1 := cache.Get(path1, hash1)
	r2, found2 := cache.Get(path1, hash2)
	r3, found3 := cache.Get(path2, hash1)
	_, found4 := cache.Get(path3, hash1)

	assert.True(t, found1)
	assert.Equal(t, "comp1", r1.Component.HashedName)

	assert.True(t, found2)
	assert.Equal(t, "comp2", r2.Component.HashedName)

	assert.True(t, found3)
	assert.Equal(t, "comp3", r3.Component.HashedName)

	assert.False(t, found4)
}

func TestInMemoryInspectionCache_Remove(t *testing.T) {
	t.Parallel()

	cache := templater_domain.NewInMemoryInspectionCache()
	path := "pages/home.pk"

	result1 := &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "test1"},
		Timestamp:  time.Now(),
		ScriptHash: "hash1",
	}
	result2 := &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "test2"},
		Timestamp:  time.Now(),
		ScriptHash: "hash2",
	}

	cache.Store(path, "hash1", result1)
	cache.Store(path, "hash2", result2)

	stats1 := cache.Stats()

	cache.Remove(path)

	stats2 := cache.Stats()
	_, found := cache.Get(path, "hash1")

	assert.False(t, found, "should not find removed path")
	assert.Equal(t, 2, stats1.TotalEntries, "should have 2 entries before removal")
	assert.Equal(t, 0, stats2.TotalEntries, "should have 0 entries after removal")
	assert.Equal(t, int64(2), stats2.Evictions, "removing 2 hashes should count as 2 evictions")
}

func TestInMemoryInspectionCache_RemoveNonExistent(t *testing.T) {
	t.Parallel()

	cache := templater_domain.NewInMemoryInspectionCache()
	cache.Store("pages/home.pk", "hash1", &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "test"},
		Timestamp:  time.Now(),
		ScriptHash: "test",
	})

	stats1 := cache.Stats()

	cache.Remove("pages/nonexistent.pk")

	stats2 := cache.Stats()

	assert.Equal(t, stats1.TotalEntries, stats2.TotalEntries, "entries should not change")
	assert.Equal(t, stats1.Evictions, stats2.Evictions, "evictions should not change")
}

func TestInMemoryInspectionCache_Clear(t *testing.T) {
	t.Parallel()

	cache := templater_domain.NewInMemoryInspectionCache()

	cache.Store("pages/home.pk", "hash1", &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "test"},
		Timestamp:  time.Now(),
		ScriptHash: "test",
	})
	cache.Store("pages/home.pk", "hash2", &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "test"},
		Timestamp:  time.Now(),
		ScriptHash: "test",
	})
	cache.Store("pages/about.pk", "hash1", &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "test"},
		Timestamp:  time.Now(),
		ScriptHash: "test",
	})
	cache.Store("components/header.pk", "hash1", &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "test"},
		Timestamp:  time.Now(),
		ScriptHash: "test",
	})

	stats1 := cache.Stats()

	cache.Clear()

	stats2 := cache.Stats()
	_, found := cache.Get("pages/home.pk", "hash1")

	assert.Equal(t, 4, stats1.TotalEntries, "should have 4 entries before clear")
	assert.Equal(t, 0, stats2.TotalEntries, "should have 0 entries after clear")
	assert.Equal(t, int64(4), stats2.Evictions, "clearing 4 entries should count as 4 evictions")
	assert.False(t, found, "should not find any entries after clear")
}

func TestInMemoryInspectionCache_ClearEmpty(t *testing.T) {
	t.Parallel()

	cache := templater_domain.NewInMemoryInspectionCache()

	cache.Clear()

	stats := cache.Stats()

	assert.Equal(t, 0, stats.TotalEntries)
	assert.Equal(t, int64(0), stats.Evictions)
}

func TestInMemoryInspectionCache_Stats(t *testing.T) {
	t.Parallel()

	cache := templater_domain.NewInMemoryInspectionCache()

	initialStats := cache.Stats()
	assert.Equal(t, 0, initialStats.TotalEntries)
	assert.Equal(t, int64(0), initialStats.Hits)
	assert.Equal(t, int64(0), initialStats.Misses)
	assert.Equal(t, int64(0), initialStats.Evictions)

	cache.Store("pages/home.pk", "hash1", &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "test"},
		Timestamp:  time.Now(),
		ScriptHash: "test",
	})
	cache.Store("pages/home.pk", "hash2", &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "test"},
		Timestamp:  time.Now(),
		ScriptHash: "test",
	})
	cache.Store("pages/about.pk", "hash1", &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "test"},
		Timestamp:  time.Now(),
		ScriptHash: "test",
	})

	cache.Get("pages/home.pk", "hash1")
	cache.Get("pages/home.pk", "hash1")
	cache.Get("pages/about.pk", "hash1")
	cache.Get("pages/nonexistent.pk", "hash1")
	cache.Get("pages/home.pk", "hash99")

	stats := cache.Stats()

	assert.Equal(t, 3, stats.TotalEntries, "should have 3 entries")
	assert.Equal(t, int64(3), stats.Hits, "should have 3 cache hits")
	assert.Equal(t, int64(2), stats.Misses, "should have 2 cache misses")
	assert.Equal(t, int64(0), stats.Evictions, "no evictions yet")

	cache.Store("pages/home.pk", "hash1", &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "test"},
		Timestamp:  time.Now(),
		ScriptHash: "test",
	})

	statsAfterOverwrite := cache.Stats()
	assert.Equal(t, int64(1), statsAfterOverwrite.Evictions, "should have 1 eviction")
}

func TestInMemoryInspectionCache_StatsAfterRemove(t *testing.T) {
	t.Parallel()

	cache := templater_domain.NewInMemoryInspectionCache()

	cache.Store("pages/home.pk", "hash1", &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "test"},
		Timestamp:  time.Now(),
		ScriptHash: "test",
	})
	cache.Store("pages/home.pk", "hash2", &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "test"},
		Timestamp:  time.Now(),
		ScriptHash: "test",
	})
	cache.Store("pages/about.pk", "hash1", &templater_domain.InspectionResult{
		Component:  &annotator_dto.VirtualComponent{HashedName: "test"},
		Timestamp:  time.Now(),
		ScriptHash: "test",
	})

	cache.Remove("pages/home.pk")

	stats := cache.Stats()

	assert.Equal(t, 1, stats.TotalEntries, "should have 1 entry remaining")
	assert.Equal(t, int64(2), stats.Evictions, "removing 2 hashes should count as 2 evictions")
}

func TestInMemoryInspectionCache_ConcurrentWrites(t *testing.T) {
	t.Parallel()

	cache := templater_domain.NewInMemoryInspectionCache()
	var wg sync.WaitGroup
	numGoroutines := 100

	for i := range numGoroutines {
		id := i
		wg.Go(func() {
			path := "pages/page" + string(rune('0'+id%10)) + ".pk"
			hash := "hash" + string(rune('0'+id%10))
			result := &templater_domain.InspectionResult{
				Component:  &annotator_dto.VirtualComponent{HashedName: "comp" + string(rune('0'+id%10))},
				Timestamp:  time.Now(),
				ScriptHash: hash,
			}
			cache.Store(path, hash, result)
		})
	}

	wg.Wait()

	stats := cache.Stats()
	assert.Greater(t, stats.TotalEntries, 0, "should have entries from concurrent writes")
}

func TestInMemoryInspectionCache_ConcurrentReads(t *testing.T) {
	t.Parallel()

	cache := templater_domain.NewInMemoryInspectionCache()

	numEntries := 50
	for i := range numEntries {
		path := "pages/page" + string(rune('0'+i%10)) + ".pk"
		hash := "hash" + string(rune('0'+i%10))
		cache.Store(path, hash, &templater_domain.InspectionResult{
			Component:  &annotator_dto.VirtualComponent{HashedName: "comp" + string(rune('0'+i%10))},
			Timestamp:  time.Now(),
			ScriptHash: hash,
		})
	}

	var wg sync.WaitGroup
	numReaders := 100

	for i := range numReaders {
		id := i
		wg.Go(func() {
			path := "pages/page" + string(rune('0'+id%10)) + ".pk"
			hash := "hash" + string(rune('0'+id%10))
			result, found := cache.Get(path, hash)
			if found {
				_ = result.Component
			}
		})
	}

	wg.Wait()

	stats := cache.Stats()
	assert.Greater(t, stats.Hits, int64(0), "should have cache hits from concurrent reads")
}

func TestInMemoryInspectionCache_ConcurrentReadWrite(t *testing.T) {
	t.Parallel()

	cache := templater_domain.NewInMemoryInspectionCache()
	var wg sync.WaitGroup
	numGoroutines := 50

	for i := range numGoroutines {
		id := i

		wg.Go(func() {
			path := "pages/page" + string(rune('0'+id%10)) + ".pk"
			hash := "hash" + string(rune('0'+id%10))
			cache.Store(path, hash, &templater_domain.InspectionResult{
				Component:  &annotator_dto.VirtualComponent{HashedName: "test"},
				Timestamp:  time.Now(),
				ScriptHash: "test",
			})
		})

		wg.Go(func() {
			path := "pages/page" + string(rune('0'+id%10)) + ".pk"
			hash := "hash" + string(rune('0'+id%10))
			_, _ = cache.Get(path, hash)
		})
	}

	wg.Wait()

	stats := cache.Stats()
	assert.Greater(t, stats.TotalEntries, 0, "should have entries")
}

func TestInMemoryInspectionCache_ConcurrentClear(t *testing.T) {
	t.Parallel()

	cache := templater_domain.NewInMemoryInspectionCache()

	for i := range 20 {
		cache.Store("pages/page.pk", "hash"+string(rune('0'+i)), &templater_domain.InspectionResult{
			Component:  &annotator_dto.VirtualComponent{HashedName: "test"},
			Timestamp:  time.Now(),
			ScriptHash: "test",
		})
	}

	var wg sync.WaitGroup

	for i := range 10 {
		wg.Go(func() {
			cache.Clear()
		})

		id := i
		wg.Go(func() {
			_, _ = cache.Get("pages/page.pk", "hash"+string(rune('0'+id)))
		})
	}

	wg.Wait()

	stats := cache.Stats()
	assert.Equal(t, 0, stats.TotalEntries, "cache should be empty after clears")
}

func TestInMemoryInspectionCache_RealUsageScenario(t *testing.T) {
	t.Parallel()

	cache := templater_domain.NewInMemoryInspectionCache()

	path := "pages/home.pk"
	hash1 := "abc123def456"

	inspection1 := &templater_domain.InspectionResult{
		Component: &annotator_dto.VirtualComponent{
			HashedName: "home_v1",
			IsPage:     true,
		},
		Timestamp:  time.Now(),
		ScriptHash: hash1,
	}

	result, found := cache.Get(path, hash1)
	assert.False(t, found, "first access should be a cache miss")
	assert.Nil(t, result)

	cache.Store(path, hash1, inspection1)

	result, found = cache.Get(path, hash1)
	assert.True(t, found, "second access should be a cache hit")
	require.NotNil(t, result)
	assert.Equal(t, "home_v1", result.Component.HashedName)

	hash2 := "xyz789abc000"
	inspection2 := &templater_domain.InspectionResult{
		Component: &annotator_dto.VirtualComponent{
			HashedName: "home_v2",
			IsPage:     true,
			IsEmail:    true,
		},
		Timestamp:  time.Now(),
		ScriptHash: hash2,
	}

	_, found = cache.Get(path, hash2)
	assert.False(t, found, "new hash should be a cache miss")

	cache.Store(path, hash2, inspection2)

	oldInsp, foundOld := cache.Get(path, hash1)
	newInsp, foundNew := cache.Get(path, hash2)

	assert.True(t, foundOld, "old hash should still be cached")
	assert.True(t, foundNew, "new hash should be cached")
	assert.False(t, oldInsp.Component.IsEmail)
	assert.True(t, newInsp.Component.IsEmail)

	cache.Remove(path)

	_, found = cache.Get(path, hash1)
	assert.False(t, found, "cache should be cleared for this file")

	stats := cache.Stats()
	assert.Equal(t, int64(3), stats.Hits, "should have 3 cache hits total")
	assert.Equal(t, int64(3), stats.Misses, "should have 3 cache misses total")
	assert.Equal(t, int64(2), stats.Evictions, "removing 2 hashes should be 2 evictions")
}
