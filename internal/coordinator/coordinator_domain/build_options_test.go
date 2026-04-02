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

	"piko.sh/piko/internal/resolver/resolver_domain"
)

func TestApplyBuildOptions_EmptySlice(t *testing.T) {
	t.Parallel()
	opts := applyBuildOptions([]BuildOption{})

	require.NotNil(t, opts)
	assert.Empty(t, opts.CausationID)
	assert.False(t, opts.FaultTolerant)
	assert.False(t, opts.SkipInspection)
	assert.Nil(t, opts.Resolver)
	assert.Nil(t, opts.ChangedFiles)
	assert.Nil(t, opts.InspectionCacheHints)
}

func TestWithCausationID_EmptyString(t *testing.T) {
	t.Parallel()
	opts := applyBuildOptions([]BuildOption{WithCausationID("")})

	assert.Empty(t, opts.CausationID)
}

func TestWithResolver_BuildOption(t *testing.T) {
	t.Parallel()
	resolver := &resolver_domain.MockResolver{
		GetModuleNameFunc:                      func() string { return "stub" },
		GetBaseDirFunc:                         func() string { return "/tmp" },
		ConvertEntryPointPathToManifestKeyFunc: func(p string) string { return p },
	}
	opts := applyBuildOptions([]BuildOption{WithResolver(resolver)})

	assert.Same(t, resolver, opts.Resolver)
}

func TestWithResolver_Nil(t *testing.T) {
	t.Parallel()
	opts := applyBuildOptions([]BuildOption{WithResolver(nil)})

	assert.Nil(t, opts.Resolver)
}

func TestWithSkipInspection_NilChangedFiles(t *testing.T) {
	t.Parallel()
	opts := applyBuildOptions([]BuildOption{withSkipInspection(nil)})

	assert.True(t, opts.SkipInspection)
	assert.Nil(t, opts.ChangedFiles)
}

func TestWithFullInspection_Standalone(t *testing.T) {
	t.Parallel()
	opts := applyBuildOptions([]BuildOption{withFullInspection()})

	assert.False(t, opts.SkipInspection)
}

func TestApplyBuildOptions_AllOptions(t *testing.T) {
	t.Parallel()
	resolver := &resolver_domain.MockResolver{
		GetModuleNameFunc:                      func() string { return "stub" },
		GetBaseDirFunc:                         func() string { return "/tmp" },
		ConvertEntryPointPathToManifestKeyFunc: func(p string) string { return p },
	}
	changed := []string{"page.pk"}
	hints := map[string]string{"a.pk": "hash1"}

	opts := applyBuildOptions([]BuildOption{
		WithCausationID("build-all"),
		WithFaultTolerance(),
		WithResolver(resolver),
		withSkipInspection(changed),
		withInspectionCacheHints(hints),
	})

	assert.Equal(t, "build-all", opts.CausationID)
	assert.True(t, opts.FaultTolerant)
	assert.Same(t, resolver, opts.Resolver)
	assert.True(t, opts.SkipInspection)
	assert.Equal(t, changed, opts.ChangedFiles)
	assert.Equal(t, hints, opts.InspectionCacheHints)
}

func TestWithCausationID_LastOneWins(t *testing.T) {
	t.Parallel()
	opts := applyBuildOptions([]BuildOption{
		WithCausationID("first"),
		WithCausationID("second"),
	})

	assert.Equal(t, "second", opts.CausationID)
}
