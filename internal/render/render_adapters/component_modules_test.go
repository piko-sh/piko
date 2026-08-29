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

package render_adapters

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/jsimport"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestExtractRequiredModules(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		componentJS string
		want        []string
	}{
		{
			name: "collects application library imports",
			componentJS: `import { piko } from "/_piko/dist/ppframework.core.es.js";
import { helper } from "/_piko/assets/mod/lib/helper.js";
import { other } from "/_piko/assets/mod/lib/other.js";
customElements.define("x-y", class {});`,
			want: []string{"/_piko/assets/mod/lib/helper.js", "/_piko/assets/mod/lib/other.js"},
		},
		{
			name: "excludes the framework runtime",
			componentJS: `import { piko } from "/_piko/dist/ppframework.core.es.js";
import { PPElement } from "/_piko/dist/ppframework.components.es.js";`,
			want: nil,
		},
		{
			name: "excludes dynamic imports so laziness survives",
			componentJS: `import { helper } from "/_piko/assets/mod/lib/helper.js";
async function load() { return import("/_piko/assets/mod/lib/heavy.js"); }`,
			want: []string{"/_piko/assets/mod/lib/helper.js"},
		},
		{
			name: "deduplicates repeated specifiers",
			componentJS: `import { a } from "/_piko/assets/mod/lib/shared.js";
import { b } from "/_piko/assets/mod/lib/shared.js";`,
			want: []string{"/_piko/assets/mod/lib/shared.js"},
		},
		{
			name:        "unparseable source yields nothing rather than failing the render",
			componentJS: `import { from "`,
			want:        nil,
		},
		{
			name:        "empty source yields nothing",
			componentJS: "",
			want:        nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, extractRequiredModules(t.Context(), testCase.componentJS))
		})
	}
}

func TestExtractRequiredModules_MatchesThePrefixTheCompilerWrites(t *testing.T) {
	t.Parallel()

	componentJS := fmt.Sprintf(`import { helper } from %q;`,
		jsimport.ResolveModuleAlias("@/lib/helper", "example.com/app"))

	assert.Equal(t, []string{"/_piko/assets/example.com/app/lib/helper.js"},
		extractRequiredModules(t.Context(), componentJS),
		"the prefix must be the one the compiler emits, not a configured serve path")
}

func TestAttachRequiredModules(t *testing.T) {
	t.Parallel()

	componentJS := `import { piko } from "/_piko/dist/ppframework.core.es.js";
import { helper } from "/_piko/assets/mod/lib/helper.js";`

	artefacts := []*registry_dto.ArtefactMeta{
		{
			ID: "mod/components/widget-a.pkc",
			ActualVariants: []registry_dto.Variant{
				{
					StorageKey: "generated/widget-a_abcdef0123456789.js",
					MetadataTags: registry_dto.TagsFromMap(map[string]string{
						"type":    "component-js",
						"role":    "entrypoint",
						"tagName": "widget-a",
					}),
				},
			},
		},
	}

	results := buildComponentResults(artefacts, map[string]struct{}{"widget-a": {}}, "/_piko/assets")
	require.Contains(t, results, "widget-a")
	require.Nil(t, results["widget-a"].RequiredModules)

	registry := &registry_domain.MockRegistryService{
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(componentJS)), nil
		},
	}

	attachRequiredModules(t.Context(), registry, artefacts, results)

	assert.Equal(t, []string{"/_piko/assets/mod/lib/helper.js"}, results["widget-a"].RequiredModules)
}

func TestAttachRequiredModules_SurvivesAnUnreadableBlob(t *testing.T) {
	t.Parallel()

	artefacts := []*registry_dto.ArtefactMeta{
		{
			ID: "mod/components/widget-a.pkc",
			ActualVariants: []registry_dto.Variant{
				{
					StorageKey: "generated/widget-a_abcdef0123456789.js",
					MetadataTags: registry_dto.TagsFromMap(map[string]string{
						"type":    "component-js",
						"role":    "entrypoint",
						"tagName": "widget-a",
					}),
				},
			},
		},
	}

	results := buildComponentResults(artefacts, map[string]struct{}{"widget-a": {}}, "/_piko/assets")

	registry := &registry_domain.MockRegistryService{
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return nil, errors.New("blob not written yet")
		},
	}

	attachRequiredModules(t.Context(), registry, artefacts, results)

	assert.Nil(t, results["widget-a"].RequiredModules,
		"a cold-start blob miss must cost a preload, never the page")
	assert.Equal(t, "widget-a", results["widget-a"].TagName, "the component itself must still render")
}

func TestExtractRequiredModules_RefusesAnOversizedSource(t *testing.T) {
	t.Parallel()

	oversized := "import { a } from \"/_piko/assets/mod/lib/a.js\";\n" +
		strings.Repeat("// padding to exceed the scan limit\n", (maxComponentJSBytes/36)+1)
	require.Greater(t, len(oversized), maxComponentJSBytes)

	assert.Nil(t, extractRequiredModules(t.Context(), oversized),
		"an unbounded parse can exhaust the goroutine stack, which no recover catches")
}

func TestExtractRequiredModules_PathologicalSourceDoesNotEscape(t *testing.T) {
	t.Parallel()

	deeplyNested := strings.Repeat("(", 60000) + "1" + strings.Repeat(")", 60000)

	assert.NotPanics(t, func() {
		extractRequiredModules(t.Context(), deeplyNested)
	}, "a component blob must never take down the render path")
}

func TestReadVariantString_RefusesAnOversizedVariant(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("x", 64)
	registry := &registry_domain.MockRegistryService{
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(body)), nil
		},
	}

	_, err := readVariantString(t.Context(), registry, &registry_dto.Variant{}, 16)

	require.ErrorIs(t, err, errVariantTooLarge,
		"a truncated component parses to nonsense, so the cap must be reported not absorbed")
}

func TestReadVariantString_AcceptsAVariantAtTheLimit(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("x", 16)
	registry := &registry_domain.MockRegistryService{
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(body)), nil
		},
	}

	content, err := readVariantString(t.Context(), registry, &registry_dto.Variant{}, 16)

	require.NoError(t, err)
	assert.Equal(t, body, content)
}

func componentArtefactWithVariants(tagName string, withMinified bool) *registry_dto.ArtefactMeta {
	t := &registry_dto.ArtefactMeta{
		ID: "mod/components/" + tagName + ".pkc",
		ActualVariants: []registry_dto.Variant{
			{
				VariantID:  "source",
				StorageKey: "generated/" + tagName + "_abcdef0123456789.js",
				MetadataTags: registry_dto.TagsFromMap(map[string]string{
					"type":    "component-js",
					"role":    "entrypoint",
					"tagName": tagName,
				}),
			},
		},
	}

	if withMinified {
		t.ActualVariants = append(t.ActualVariants, registry_dto.Variant{
			VariantID:  minifiedVariantID,
			StorageKey: "generated/" + tagName + "_fedcba9876543210.min.js",
			MetadataTags: registry_dto.TagsFromMap(map[string]string{
				"type":    "component-js",
				"role":    "entrypoint",
				"tagName": tagName,
			}),
		})
	}

	return t
}

func TestAttachRequiredModules_ReadsTheVariantTheBrowserWillReceive(t *testing.T) {
	daemon_frontend.SetSRIEnabled(true)
	t.Cleanup(func() { daemon_frontend.SetSRIEnabled(false) })

	artefacts := []*registry_dto.ArtefactMeta{componentArtefactWithVariants("widget-a", true)}
	results := buildComponentResults(artefacts, map[string]struct{}{"widget-a": {}}, "/_piko/assets")

	var readKeys []string
	registry := &registry_domain.MockRegistryService{
		GetVariantDataFunc: func(_ context.Context, variant *registry_dto.Variant) (io.ReadCloser, error) {
			readKeys = append(readKeys, variant.StorageKey)

			return io.NopCloser(strings.NewReader(
				`import { helper } from "/_piko/assets/mod/lib/helper.js";`)), nil
		},
	}

	attachRequiredModules(t.Context(), registry, artefacts, results)

	require.Len(t, readKeys, 1)
	assert.Equal(t, "generated/widget-a_fedcba9876543210.min.js", readKeys[0],
		"deriving imports from bytes the browser never downloads would preload the wrong modules")
	assert.Contains(t, results["widget-a"].BaseJSPath, "fedcba9876543210.min.js",
		"the scanned variant must be the one BaseJSPath points at")
}

func TestAttachRequiredModules_ReadsTheEntrypointWhenIntegrityIsOff(t *testing.T) {
	artefacts := []*registry_dto.ArtefactMeta{componentArtefactWithVariants("widget-a", true)}
	results := buildComponentResults(artefacts, map[string]struct{}{"widget-a": {}}, "/_piko/assets")

	var readKeys []string
	registry := &registry_domain.MockRegistryService{
		GetVariantDataFunc: func(_ context.Context, variant *registry_dto.Variant) (io.ReadCloser, error) {
			readKeys = append(readKeys, variant.StorageKey)

			return io.NopCloser(strings.NewReader("")), nil
		},
	}

	attachRequiredModules(t.Context(), registry, artefacts, results)

	require.Len(t, readKeys, 1)
	assert.Equal(t, "generated/widget-a_abcdef0123456789.js", readKeys[0])
}

func TestAttachRequiredModules_ScansManyComponentsConcurrently(t *testing.T) {
	t.Parallel()

	const componentCount = sequentialComponentScanThreshold + 5

	artefacts := make([]*registry_dto.ArtefactMeta, 0, componentCount)
	requested := make(map[string]struct{}, componentCount)
	for index := range componentCount {
		tagName := fmt.Sprintf("widget-%d", index)
		artefacts = append(artefacts, componentArtefactWithVariants(tagName, false))
		requested[tagName] = struct{}{}
	}

	results := buildComponentResults(artefacts, requested, "/_piko/assets")
	require.Len(t, results, componentCount)

	var reads atomic.Int64
	registry := &registry_domain.MockRegistryService{
		GetVariantDataFunc: func(_ context.Context, variant *registry_dto.Variant) (io.ReadCloser, error) {
			reads.Add(1)
			module := strings.TrimSuffix(strings.TrimPrefix(variant.StorageKey, "generated/"), ".js")

			return io.NopCloser(strings.NewReader(
				fmt.Sprintf("import { helper } from %q;", "/_piko/assets/mod/lib/"+module+".js"))), nil
		},
	}

	attachRequiredModules(t.Context(), registry, artefacts, results)

	assert.Equal(t, int64(componentCount), reads.Load(), "every requested component must be scanned")
	for tagName, meta := range results {
		assert.Len(t, meta.RequiredModules, 1, "component %s lost its module list", tagName)
	}
}

func TestAttachRequiredModules_StopsWhenTheRenderIsAbandoned(t *testing.T) {
	t.Parallel()

	const componentCount = sequentialComponentScanThreshold + 5

	artefacts := make([]*registry_dto.ArtefactMeta, 0, componentCount)
	requested := make(map[string]struct{}, componentCount)
	for index := range componentCount {
		tagName := fmt.Sprintf("widget-%d", index)
		artefacts = append(artefacts, componentArtefactWithVariants(tagName, false))
		requested[tagName] = struct{}{}
	}

	results := buildComponentResults(artefacts, requested, "/_piko/assets")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var reads atomic.Int64
	registry := &registry_domain.MockRegistryService{
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			reads.Add(1)

			return io.NopCloser(strings.NewReader("")), nil
		},
	}

	attachRequiredModules(ctx, registry, artefacts, results)

	assert.Zero(t, reads.Load(), "an abandoned render must not keep reading blobs")
}
