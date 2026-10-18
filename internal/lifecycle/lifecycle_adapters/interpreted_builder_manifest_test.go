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
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/generator/generator_dto"
)

const (
	manifestTestCardJS = "pk-js/partials/card.js"
)

func newManifestTestArtefact(projectRoot, moduleName, relativePath, componentType, content string) *generator_dto.GeneratedArtefact {
	artefact := newTestArtefact(projectRoot, moduleName, relativePath, content)
	artefact.Component.IsPublic = true
	artefact.Component.IsPage = componentType == "page"
	artefact.Component.Source.ComponentType = componentType
	artefact.Component.Source.ModuleImportPath = moduleName + "/" + relativePath
	artefact.Component.Source.Script = &annotator_dto.ParsedScript{}
	if componentType == "partial" {
		artefact.JSArtefactID = manifestTestCardJS
	}
	return artefact
}

func newPartialProjectResult(rebuilt []*generator_dto.GeneratedArtefact, all ...*generator_dto.GeneratedArtefact) *annotator_dto.ProjectAnnotationResult {
	result := newProjectResult(all...)
	result.FinalGeneratedArtefacts = rebuilt
	result.ComponentResults = make(map[string]*annotator_dto.AnnotationResult, len(rebuilt))
	for _, artefact := range rebuilt {
		result.ComponentResults[artefact.Component.HashedName] = artefact.Result
	}
	return result
}

func newFullProjectResult(artefacts ...*generator_dto.GeneratedArtefact) *annotator_dto.ProjectAnnotationResult {
	return newPartialProjectResult(artefacts, artefacts...)
}

func cachedPageKeys(orchestrator *InterpretedBuildOrchestrator) []string {
	orchestrator.stateLock.RLock()
	defer orchestrator.stateLock.RUnlock()
	return slices.Sorted(maps.Keys(orchestrator.cachedManifest.Pages))
}

func TestMarkDirtyMergesAPartialResultIntoTheManifest(t *testing.T) {
	t.Parallel()
	const moduleName = "mark_dirty_partial_result"
	orchestrator, projectRoot := newTestOrchestrator(t, moduleName, &fakePool{interpreter: &fakeInterpreter{compile: registeringCompile}}, nil)
	home := newManifestTestArtefact(projectRoot, moduleName, "pages/home.pk", "page", "package home\n")
	about := newManifestTestArtefact(projectRoot, moduleName, "pages/about.pk", "page", "package about\n")
	_, err := orchestrator.BuildRunner(context.Background(), newFullProjectResult(home, about))
	require.NoError(t, err)
	require.Equal(t, []string{"pages/about.pk", "pages/home.pk"}, cachedPageKeys(orchestrator))

	editedAbout := newManifestTestArtefact(projectRoot, moduleName, "pages/about.pk", "page", "package about // edited\n")
	require.NoError(t, orchestrator.MarkDirty(context.Background(), newPartialProjectResult(
		[]*generator_dto.GeneratedArtefact{editedAbout}, home, editedAbout)))

	assert.Equal(t, []string{"pages/about.pk", "pages/home.pk"}, cachedPageKeys(orchestrator),
		"a result that rebuilt only some components must not drop the others' routes")
	assert.Equal(t, []string{"pages/about.pk"}, keysOf(orchestrator.dirtyCodeCache))
}

func TestMarkDirtyReplacesTheManifestForAFullResult(t *testing.T) {
	t.Parallel()
	const moduleName = "mark_dirty_full_result"
	orchestrator, projectRoot := newTestOrchestrator(t, moduleName, &fakePool{interpreter: &fakeInterpreter{compile: registeringCompile}}, nil)
	home := newManifestTestArtefact(projectRoot, moduleName, "pages/home.pk", "page", "package home\n")
	about := newManifestTestArtefact(projectRoot, moduleName, "pages/about.pk", "page", "package about\n")
	_, err := orchestrator.BuildRunner(context.Background(), newFullProjectResult(home, about))
	require.NoError(t, err)

	require.NoError(t, orchestrator.MarkDirty(context.Background(), newFullProjectResult(home)))

	assert.Equal(t, []string{"pages/home.pk"}, cachedPageKeys(orchestrator),
		"a result covering its whole virtual module is authoritative")
}

func TestMarkDirtyRebuildsTheReverseDependencyMap(t *testing.T) {
	t.Parallel()
	const moduleName = "mark_dirty_reverse_deps"
	orchestrator, projectRoot := newTestOrchestrator(t, moduleName, &fakePool{interpreter: &fakeInterpreter{compile: registeringCompile}}, nil)
	card := newManifestTestArtefact(projectRoot, moduleName, "partials/card.pk", "partial", "package card\n")
	home := newManifestTestArtefact(projectRoot, moduleName, "pages/home.pk", "page", "package home\n")
	_, err := orchestrator.BuildRunner(context.Background(), newFullProjectResult(card, home))
	require.NoError(t, err)
	require.Empty(t, orchestrator.GetAffectedComponents("partials/card.pk"))

	homeImportingCard := withPikoImports(newManifestTestArtefact(projectRoot, moduleName, "pages/home.pk", "page", "package home // imports card\n"),
		moduleName+"/partials/card.pk")
	require.NoError(t, orchestrator.MarkDirty(context.Background(), newPartialProjectResult(
		[]*generator_dto.GeneratedArtefact{homeImportingCard}, card, homeImportingCard)))

	assert.Equal(t, []string{"pages/home.pk"}, orchestrator.GetAffectedComponents("partials/card.pk"),
		"an import added by the edit must be followed by later targeted rebuilds")
}

func TestMarkComponentsDirtyKeepsPartialScriptsOfARebuiltPage(t *testing.T) {
	t.Parallel()
	const moduleName = "mark_components_dirty_partial_js"
	orchestrator, projectRoot := newTestOrchestrator(t, moduleName, &fakePool{interpreter: &fakeInterpreter{compile: registeringCompile}}, nil)
	card := newManifestTestArtefact(projectRoot, moduleName, "partials/card.pk", "partial", "package card\n")
	newHome := func(content string) *generator_dto.GeneratedArtefact {
		return withPikoImports(newManifestTestArtefact(projectRoot, moduleName, "pages/home.pk", "page", content), moduleName+"/partials/card.pk")
	}
	_, err := orchestrator.BuildRunner(context.Background(), newFullProjectResult(card, newHome("package home\n")))
	require.NoError(t, err)
	require.Equal(t, []string{manifestTestCardJS}, orchestrator.cachedManifest.Pages["pages/home.pk"].JSArtefactIDs)

	editedHome := newHome("package home // style edit\n")
	require.NoError(t, orchestrator.MarkComponentsDirty(context.Background(), newPartialProjectResult(
		[]*generator_dto.GeneratedArtefact{editedHome}, card, editedHome)))

	orchestrator.stateLock.RLock()
	defer orchestrator.stateLock.RUnlock()
	assert.Equal(t, []string{manifestTestCardJS}, orchestrator.cachedManifest.Pages["pages/home.pk"].JSArtefactIDs,
		"a page rebuilt without its partials must keep their client scripts")
}

func TestRemoveComponentDropsItsManifestEntries(t *testing.T) {
	t.Parallel()
	const moduleName = "remove_component_manifest"
	orchestrator, projectRoot := newTestOrchestrator(t, moduleName, &fakePool{interpreter: &fakeInterpreter{compile: registeringCompile}}, nil)
	card := newManifestTestArtefact(projectRoot, moduleName, "partials/card.pk", "partial", "package card\n")
	home := newManifestTestArtefact(projectRoot, moduleName, "pages/home.pk", "page", "package home\n")
	old := newManifestTestArtefact(projectRoot, moduleName, "pages/old.pk", "page", "package old\n")
	_, err := orchestrator.BuildRunner(context.Background(), newFullProjectResult(card, home, old))
	require.NoError(t, err)

	orchestrator.RemoveComponent(context.Background(), "pages/old.pk")
	orchestrator.RemoveComponent(context.Background(), "partials/card.pk")

	assert.Equal(t, []string{"pages/home.pk"}, cachedPageKeys(orchestrator),
		"merging partial builds never drops entries, so removal must")
	assert.NotContains(t, orchestrator.cachedManifest.Partials, "partials/card.pk")
}
