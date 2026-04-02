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

package coordinator_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
)

func TestMatchesScriptHashes_NilEntry(t *testing.T) {
	t.Parallel()
	var e *IntrospectionCacheEntry
	assert.False(t, e.MatchesScriptHashes(map[string]string{"a": "b"}))
}

func TestMatchesScriptHashes_NilHashes(t *testing.T) {
	t.Parallel()
	e := &IntrospectionCacheEntry{ScriptHashes: nil}
	assert.False(t, e.MatchesScriptHashes(map[string]string{"a": "b"}))
}

func TestMatchesScriptHashes_ExactMatch(t *testing.T) {
	t.Parallel()
	e := &IntrospectionCacheEntry{
		ScriptHashes: map[string]string{"file1.pk": "abc123", "file2.pk": "def456"},
	}
	current := map[string]string{"file1.pk": "abc123", "file2.pk": "def456"}
	assert.True(t, e.MatchesScriptHashes(current))
}

func TestMatchesScriptHashes_DifferentHash(t *testing.T) {
	t.Parallel()
	e := &IntrospectionCacheEntry{
		ScriptHashes: map[string]string{"file1.pk": "abc123"},
	}
	current := map[string]string{"file1.pk": "changed"}
	assert.False(t, e.MatchesScriptHashes(current))
}

func TestMatchesScriptHashes_MissingFileInCurrent(t *testing.T) {
	t.Parallel()
	e := &IntrospectionCacheEntry{
		ScriptHashes: map[string]string{"file1.pk": "abc123", "file2.pk": "def456"},
	}
	current := map[string]string{"file1.pk": "abc123"}
	assert.False(t, e.MatchesScriptHashes(current))
}

func TestMatchesScriptHashes_ExtraFileInCurrent(t *testing.T) {
	t.Parallel()
	e := &IntrospectionCacheEntry{
		ScriptHashes: map[string]string{"file1.pk": "abc123"},
	}
	current := map[string]string{"file1.pk": "abc123", "file2.pk": "def456"}
	assert.False(t, e.MatchesScriptHashes(current))
}

func TestMatchesScriptHashes_EmptyBoth(t *testing.T) {
	t.Parallel()
	e := &IntrospectionCacheEntry{
		ScriptHashes: map[string]string{},
	}
	current := map[string]string{}
	assert.True(t, e.MatchesScriptHashes(current))
}

func TestNewIntrospectionCacheEntry_PointerIdentity(t *testing.T) {
	t.Parallel()
	vm := &annotator_dto.VirtualModule{}
	tr := &annotator_domain.TypeResolver{}
	cg := &annotator_dto.ComponentGraph{}
	hashes := map[string]string{"a.pk": "hash1"}

	entry := newIntrospectionCacheEntry(vm, tr, cg, hashes)

	require.NotNil(t, entry)
	assert.Same(t, vm, entry.VirtualModule)
	assert.Same(t, tr, entry.TypeResolver)
	assert.Same(t, cg, entry.ComponentGraph)
	assert.Equal(t, hashes, entry.ScriptHashes)
	assert.Equal(t, CurrentIntrospectionCacheVersion, entry.Version)
	assert.False(t, entry.Timestamp.IsZero())
}

func TestCurrentIntrospectionCacheVersion(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 1, CurrentIntrospectionCacheVersion)
}
