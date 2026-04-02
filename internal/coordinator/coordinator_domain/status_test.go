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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
)

func TestStateString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		want  string
		state state
	}{
		{name: "idle", want: "Idle", state: stateIdle},
		{name: "building", want: "Building", state: stateBuilding},
		{name: "ready", want: "Ready", state: stateReady},
		{name: "failed", want: "Failed", state: stateFailed},
		{name: "unknown", want: "Unknown", state: state(99)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.state.String())
		})
	}
}

func TestIntrospectionCacheEntryIsValid(t *testing.T) {
	t.Parallel()

	t.Run("nil entry", func(t *testing.T) {
		t.Parallel()

		var e *IntrospectionCacheEntry
		assert.False(t, e.IsValid())
	})

	t.Run("wrong version", func(t *testing.T) {
		t.Parallel()

		e := &IntrospectionCacheEntry{
			VirtualModule:  &annotator_dto.VirtualModule{},
			TypeResolver:   &annotator_domain.TypeResolver{},
			ComponentGraph: &annotator_dto.ComponentGraph{},
			Version:        CurrentIntrospectionCacheVersion + 1,
		}
		assert.False(t, e.IsValid())
	})

	t.Run("nil virtual module", func(t *testing.T) {
		t.Parallel()

		e := &IntrospectionCacheEntry{
			VirtualModule:  nil,
			TypeResolver:   &annotator_domain.TypeResolver{},
			ComponentGraph: &annotator_dto.ComponentGraph{},
			Version:        CurrentIntrospectionCacheVersion,
		}
		assert.False(t, e.IsValid())
	})

	t.Run("nil type resolver", func(t *testing.T) {
		t.Parallel()

		e := &IntrospectionCacheEntry{
			VirtualModule:  &annotator_dto.VirtualModule{},
			TypeResolver:   nil,
			ComponentGraph: &annotator_dto.ComponentGraph{},
			Version:        CurrentIntrospectionCacheVersion,
		}
		assert.False(t, e.IsValid())
	})

	t.Run("nil component graph", func(t *testing.T) {
		t.Parallel()

		e := &IntrospectionCacheEntry{
			VirtualModule:  &annotator_dto.VirtualModule{},
			TypeResolver:   &annotator_domain.TypeResolver{},
			ComponentGraph: nil,
			Version:        CurrentIntrospectionCacheVersion,
		}
		assert.False(t, e.IsValid())
	})

	t.Run("valid entry", func(t *testing.T) {
		t.Parallel()

		e := &IntrospectionCacheEntry{
			VirtualModule:  &annotator_dto.VirtualModule{},
			TypeResolver:   &annotator_domain.TypeResolver{},
			ComponentGraph: &annotator_dto.ComponentGraph{},
			Version:        CurrentIntrospectionCacheVersion,
		}
		assert.True(t, e.IsValid())
	})
}

func TestNewIntrospectionCacheEntry(t *testing.T) {
	t.Parallel()

	vm := &annotator_dto.VirtualModule{}
	tr := &annotator_domain.TypeResolver{}
	cg := &annotator_dto.ComponentGraph{}
	hashes := map[string]string{"a.pk": "abc123"}

	entry := newIntrospectionCacheEntry(vm, tr, cg, hashes)

	require.NotNil(t, entry)
	assert.Equal(t, vm, entry.VirtualModule)
	assert.Equal(t, tr, entry.TypeResolver)
	assert.Equal(t, cg, entry.ComponentGraph)
	assert.Equal(t, hashes, entry.ScriptHashes)
	assert.Equal(t, CurrentIntrospectionCacheVersion, entry.Version)
	assert.WithinDuration(t, time.Now(), entry.Timestamp, 5*time.Second)
}

func TestApplyBuildOptions(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		opts := applyBuildOptions(nil)
		require.NotNil(t, opts)
		assert.Empty(t, opts.CausationID)
		assert.Nil(t, opts.ChangedFiles)
		assert.Nil(t, opts.Resolver)
		assert.Nil(t, opts.InspectionCacheHints)
		assert.False(t, opts.SkipInspection)
		assert.False(t, opts.FaultTolerant)
	})

	t.Run("with causation ID", func(t *testing.T) {
		t.Parallel()

		opts := applyBuildOptions([]BuildOption{WithCausationID("test-123")})
		assert.Equal(t, "test-123", opts.CausationID)
	})

	t.Run("with fault tolerance", func(t *testing.T) {
		t.Parallel()

		opts := applyBuildOptions([]BuildOption{WithFaultTolerance()})
		assert.True(t, opts.FaultTolerant)
	})

	t.Run("with skip inspection", func(t *testing.T) {
		t.Parallel()

		files := []string{"a.pk", "b.pk"}
		opts := applyBuildOptions([]BuildOption{withSkipInspection(files)})
		assert.True(t, opts.SkipInspection)
		assert.Equal(t, files, opts.ChangedFiles)
	})

	t.Run("with inspection cache hints", func(t *testing.T) {
		t.Parallel()

		hints := map[string]string{"a.pk": "hash1"}
		opts := applyBuildOptions([]BuildOption{withInspectionCacheHints(hints)})
		assert.Equal(t, hints, opts.InspectionCacheHints)
	})

	t.Run("with full inspection overrides skip", func(t *testing.T) {
		t.Parallel()

		opts := applyBuildOptions([]BuildOption{
			withSkipInspection([]string{"a.pk"}),
			withFullInspection(),
		})
		assert.False(t, opts.SkipInspection)
	})

	t.Run("multiple options", func(t *testing.T) {
		t.Parallel()

		opts := applyBuildOptions([]BuildOption{
			WithCausationID("evt-1"),
			WithFaultTolerance(),
			withInspectionCacheHints(map[string]string{"x": "y"}),
		})
		assert.Equal(t, "evt-1", opts.CausationID)
		assert.True(t, opts.FaultTolerant)
		assert.Equal(t, map[string]string{"x": "y"}, opts.InspectionCacheHints)
	})
}

func TestApplyCoordinatorOptions(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		opts := applyCoordinatorOptions()
		assert.NotNil(t, opts.clock)
		assert.Equal(t, 750*time.Millisecond, opts.debounceDuration)
		assert.Nil(t, opts.fileHashCache)
		assert.Nil(t, opts.codeEmitter)
		assert.Nil(t, opts.diagnosticOutput)
		assert.Nil(t, opts.baseDirSandbox)
	})

	t.Run("with debounce duration", func(t *testing.T) {
		t.Parallel()

		opts := applyCoordinatorOptions(WithDebounceDuration(2 * time.Second))
		assert.Equal(t, 2*time.Second, opts.debounceDuration)
	})

	t.Run("zero debounce duration ignored", func(t *testing.T) {
		t.Parallel()

		opts := applyCoordinatorOptions(WithDebounceDuration(0))
		assert.Equal(t, 750*time.Millisecond, opts.debounceDuration)
	})

	t.Run("negative debounce duration ignored", func(t *testing.T) {
		t.Parallel()

		opts := applyCoordinatorOptions(WithDebounceDuration(-1 * time.Second))
		assert.Equal(t, 750*time.Millisecond, opts.debounceDuration)
	})
}
