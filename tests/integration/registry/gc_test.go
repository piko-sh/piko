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

//go:build integration

package registry_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_dto"
)

func runGCHints(t *testing.T, config Config) {
	if !config.SupportsGCHints {
		t.Skip("backend does not advertise SupportsGCHints")
	}

	t.Run("hints round-trip through pop", func(t *testing.T) {
		store := config.NewStore(t)
		addGCHints(t, store, []registry_dto.GCHint{
			{BackendID: "local_disk_cache", StorageKey: "gc/1"},
			{BackendID: "local_disk_cache", StorageKey: "gc/2"},
		})

		popped, err := store.PopGCHints(ctx(t), 10)
		require.NoError(t, err)
		keys := hintKeys(popped)
		assert.ElementsMatch(t, []string{"gc/1", "gc/2"}, keys, "every added hint must come back")
	})

	t.Run("pop removes what it returns", func(t *testing.T) {
		store := config.NewStore(t)
		addGCHints(t, store, []registry_dto.GCHint{
			{BackendID: "local_disk_cache", StorageKey: "gc/a"},
			{BackendID: "local_disk_cache", StorageKey: "gc/b"},
			{BackendID: "local_disk_cache", StorageKey: "gc/c"},
		})

		first, err := store.PopGCHints(ctx(t), 2)
		require.NoError(t, err)
		require.Len(t, first, 2, "pop must honour the limit")

		second, err := store.PopGCHints(ctx(t), 10)
		require.NoError(t, err)

		all := append(hintKeys(first), hintKeys(second)...)
		assert.ElementsMatch(t, []string{"gc/a", "gc/b", "gc/c"}, all,
			"a popped hint must not be returned again; the two pops together see each hint once")

		third, err := store.PopGCHints(ctx(t), 10)
		require.NoError(t, err)
		assert.Empty(t, third, "once drained, pop returns nothing")
	})
}

func hintKeys(hints []registry_dto.GCHint) []string {
	keys := make([]string, len(hints))
	for i := range hints {
		keys[i] = hints[i].StorageKey
	}
	return keys
}
