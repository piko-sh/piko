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

package lifecycle_adapters

import (
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/templater/templater_adapters"
)

func TestPopulateProgCacheForComponentExpandsVirtualInstances(t *testing.T) {
	t.Parallel()

	manifest := &generator_dto.Manifest{
		Pages: map[string]generator_dto.ManifestPageEntry{
			"blog/post-a": {
				PackagePath:        "example.com/blog",
				OriginalSourcePath: "pages/blog.pk",
				RoutePatterns:      map[string]string{"/blog/post-a": "blog.post_a"},
			},
			"blog/post-b": {
				PackagePath:        "example.com/blog",
				OriginalSourcePath: "pages/blog.pk",
				RoutePatterns:      map[string]string{"/blog/post-b": "blog.post_b"},
			},
		},
		Partials: map[string]generator_dto.ManifestPartialEntry{},
		Emails:   map[string]generator_dto.ManifestEmailEntry{},
	}
	component := &annotator_dto.VirtualComponent{
		CanonicalGoPackagePath: "example.com/blog",
		HashedName:             "blog",
		VirtualInstances: []annotator_dto.VirtualPageInstance{
			{ManifestKey: "blog/post-a"},
			{ManifestKey: "blog/post-b"},
		},
	}
	orchestrator := &InterpretedBuildOrchestrator{projectRoot: t.TempDir()}
	target := map[string]*templater_adapters.PageEntry{}
	linkCalls := 0
	linkFn := func(_ *templater_adapters.PageEntry, _ *annotator_dto.VirtualComponent) error {
		linkCalls++
		return nil
	}

	err := orchestrator.populateProgCacheForComponent(
		t.Context(),
		manifest,
		component,
		"pages/blog.pk",
		linkFn,
		target,
	)

	require.NoError(t, err)
	require.Len(t, target, 2, "one entry per virtual instance")
	require.Contains(t, target, "blog/post-a")
	require.Contains(t, target, "blog/post-b")
	require.Equal(t, 2, linkCalls)
	require.NotContains(t, target, "pages/blog.pk",
		"collection-backed components must not emit a single .pk-path entry")
}

func TestPopulateProgCacheForComponentFallsBackForNonCollection(t *testing.T) {
	t.Parallel()

	manifest := &generator_dto.Manifest{
		Pages: map[string]generator_dto.ManifestPageEntry{
			"pages/about.pk": {
				PackagePath:        "example.com/site",
				OriginalSourcePath: "pages/about.pk",
				RoutePatterns:      map[string]string{"/about": "site.about"},
			},
		},
	}
	projectRoot := t.TempDir()
	component := &annotator_dto.VirtualComponent{
		CanonicalGoPackagePath: "example.com/site",
		HashedName:             "site",
		Source: &annotator_dto.ParsedComponent{
			SourcePath: projectRoot + "/pages/about.pk",
		},
	}
	orchestrator := &InterpretedBuildOrchestrator{projectRoot: projectRoot}
	target := map[string]*templater_adapters.PageEntry{}

	err := orchestrator.populateProgCacheForComponent(
		t.Context(),
		manifest,
		component,
		"pages/about.pk",
		func(*templater_adapters.PageEntry, *annotator_dto.VirtualComponent) error { return nil },
		target,
	)

	require.NoError(t, err)
	require.Contains(t, target, "pages/about.pk")
}

func TestPopulateProgCacheForComponentSkipsEmptyManifestKey(t *testing.T) {
	t.Parallel()

	manifest := &generator_dto.Manifest{
		Pages: map[string]generator_dto.ManifestPageEntry{},
	}
	component := &annotator_dto.VirtualComponent{
		CanonicalGoPackagePath: "example.com/blog",
		VirtualInstances: []annotator_dto.VirtualPageInstance{
			{ManifestKey: ""},
			{ManifestKey: "blog/post"},
		},
	}
	orchestrator := &InterpretedBuildOrchestrator{projectRoot: t.TempDir()}
	target := map[string]*templater_adapters.PageEntry{}

	err := orchestrator.populateProgCacheForComponent(
		t.Context(),
		manifest,
		component,
		"pages/blog.pk",
		func(*templater_adapters.PageEntry, *annotator_dto.VirtualComponent) error { return nil },
		target,
	)

	require.NoError(t, err)
	require.Len(t, target, 1, "empty ManifestKey entries must be skipped")
	require.Contains(t, target, "blog/post")
}
