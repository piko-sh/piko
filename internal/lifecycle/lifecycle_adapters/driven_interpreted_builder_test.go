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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safedisk"
)

func TestBuildRunnerCompilesAndLinksEveryComponent(t *testing.T) {
	t.Parallel()
	const moduleName = "build_runner_links"
	interpreter := &fakeInterpreter{compile: registeringCompile}
	orchestrator, projectRoot := newTestOrchestrator(t, moduleName, &fakePool{interpreter: interpreter}, nil)
	first := newTestArtefact(projectRoot, moduleName, "pages/a.pk", "package a\n")
	second := newTestArtefact(projectRoot, moduleName, "pages/b.pk", "package b\n")

	runner, err := orchestrator.BuildRunner(context.Background(), newProjectResult(first, second))
	require.NoError(t, err)
	require.NotNil(t, runner)

	assert.True(t, orchestrator.IsInitialised())
	assert.ElementsMatch(t, []string{"pages/a.pk", "pages/b.pk"}, orchestrator.GetAllCachedKeys())
	calls := interpreter.compileCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, moduleName, calls[0].modulePath)
	assert.Contains(t, calls[0].packages, "pages/a")
	assert.Contains(t, calls[0].packages, "pages/b")
}

func TestBuildRunnerReportsFailures(t *testing.T) {
	t.Parallel()
	poolFailure := errors.New("pool unavailable")
	testCases := []struct {
		pool    *fakePool
		name    string
		wantErr string
	}{
		{name: "pool failure", pool: &fakePool{getErr: poolFailure}, wantErr: "pool unavailable"},
		{name: "compile failure", pool: &fakePool{interpreter: &fakeInterpreter{compile: func(compileCall) error { return errors.New("does not compile") }}}, wantErr: "does not compile"},
		{name: "unregistered builder", pool: &fakePool{interpreter: &fakeInterpreter{}}, wantErr: "BuildAST not found"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			moduleName := "build_runner_failure_" + testCase.name
			orchestrator, projectRoot := newTestOrchestrator(t, moduleName, testCase.pool, nil)
			_, err := orchestrator.BuildRunner(context.Background(), newProjectResult(newTestArtefact(projectRoot, moduleName, "pages/a.pk", "package a\n")))
			require.ErrorContains(t, err, testCase.wantErr)
			assert.False(t, orchestrator.IsInitialised())
		})
	}
}

func TestBuildRunnerWithoutPool(t *testing.T) {
	t.Parallel()
	const moduleName = "build_runner_no_pool"
	orchestrator, projectRoot := newTestOrchestrator(t, moduleName, nil, nil)
	_, err := orchestrator.BuildRunner(context.Background(), newProjectResult(newTestArtefact(projectRoot, moduleName, "pages/a.pk", "package a\n")))
	require.ErrorIs(t, err, errNoInterpreterPool)
}

func TestBuildRunnerUsesFreshPoolAfterUserPackagesChange(t *testing.T) {
	t.Parallel()
	const moduleName = "build_runner_fresh_pool"
	initialInterpreter := &fakeInterpreter{compile: registeringCompile}
	freshInterpreter := &fakeInterpreter{compile: registeringCompile}
	initialPool := &fakePool{interpreter: initialInterpreter}
	provider := &fakeProvider{pool: &fakePool{interpreter: freshInterpreter}}
	orchestrator, projectRoot := newTestOrchestrator(t, moduleName, initialPool, provider)
	result := newProjectResult(newTestArtefact(projectRoot, moduleName, "pages/a.pk", "package a\n"))

	_, err := orchestrator.BuildRunner(context.Background(), result)
	require.NoError(t, err)
	require.True(t, orchestrator.IsInitialised())
	assert.Zero(t, provider.created.Load(), "the first build uses the pool it was given")

	orchestrator.InvalidateUserPackages()
	assert.False(t, orchestrator.IsInitialised(), "changed user packages force a full build")

	_, err = orchestrator.BuildRunner(context.Background(), result)
	require.NoError(t, err)
	assert.True(t, orchestrator.IsInitialised())
	assert.Equal(t, int32(1), provider.created.Load())
	assert.Equal(t, int32(1), provider.pool.loadCalls.Load(), "the fresh pool has its modules loaded")
	assert.Same(t, provider.pool, orchestrator.interpreterPool)
	assert.Len(t, freshInterpreter.compileCalls(), 1)
	assert.Len(t, initialInterpreter.compileCalls(), 1)
}

func TestBuildRunnerKeepsCurrentPoolWhenFreshPoolFails(t *testing.T) {
	t.Parallel()
	const moduleName = "build_runner_fresh_pool_fails"
	initialPool := &fakePool{interpreter: &fakeInterpreter{compile: registeringCompile}}
	provider := &fakeProvider{pool: &fakePool{loadErr: errors.New("module failed to load")}}
	orchestrator, projectRoot := newTestOrchestrator(t, moduleName, initialPool, provider)
	result := newProjectResult(newTestArtefact(projectRoot, moduleName, "pages/a.pk", "package a\n"))

	_, err := orchestrator.BuildRunner(context.Background(), result)
	require.NoError(t, err)
	orchestrator.InvalidateUserPackages()

	_, err = orchestrator.BuildRunner(context.Background(), result)
	require.ErrorContains(t, err, "module failed to load")
	assert.Same(t, initialPool, orchestrator.interpreterPool)
	assert.False(t, orchestrator.IsInitialised(), "the next build retries with a fresh pool")
}

func TestBuildRunnerWithoutProviderKeepsPoolAfterUserPackagesChange(t *testing.T) {
	t.Parallel()
	const moduleName = "build_runner_no_provider"
	pool := &fakePool{interpreter: &fakeInterpreter{compile: registeringCompile}}
	orchestrator, projectRoot := newTestOrchestrator(t, moduleName, pool, nil)
	result := newProjectResult(newTestArtefact(projectRoot, moduleName, "pages/a.pk", "package a\n"))

	_, err := orchestrator.BuildRunner(context.Background(), result)
	require.NoError(t, err)
	orchestrator.InvalidateUserPackages()
	_, err = orchestrator.BuildRunner(context.Background(), result)
	require.NoError(t, err)
	assert.True(t, orchestrator.IsInitialised())
	assert.Same(t, pool, orchestrator.interpreterPool)
}

func TestBuildRunnerEmptyResults(t *testing.T) {
	t.Parallel()
	orchestrator, _ := newTestOrchestrator(t, "build_runner_empty", &fakePool{interpreter: &fakeInterpreter{}}, nil)
	runner, err := orchestrator.BuildRunner(context.Background(), newProjectResult())
	require.NoError(t, err)
	require.NotNil(t, runner)
	assert.False(t, orchestrator.IsInitialised())
}

func TestDiscoverUserPackages(t *testing.T) {
	t.Parallel()
	const moduleName = "discover_module"
	projectRoot := t.TempDir()
	writeFile := func(relativePath, content string) {
		t.Helper()
		path := filepath.Join(projectRoot, relativePath)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}
	writeFile("util/util.go", "package util\n\nimport \"discover_module/helpers\"\n\nfunc Answer() int { return helpers.Base() }\n")
	writeFile("util/util_test.go", "package util\n")
	writeFile("helpers/helpers.go", "package helpers\n\nfunc Base() int { return 1 }\n")
	writeFile("unused/unused.go", "package unused\n")

	testCases := []struct {
		registered map[string]bool
		name       string
		want       []string
	}{
		{name: "follows imports transitively", registered: nil, want: []string{"pages/a", "util", "helpers"}},
		{name: "skips registered packages", registered: map[string]bool{"discover_module/util": true}, want: []string{"pages/a"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			orchestrator := NewInterpretedBuildOrchestrator(InterpretedBuildOrchestratorDeps{
				ModuleName:  moduleName,
				ProjectRoot: projectRoot,
				Clock:       clock.RealClock(),
			})
			packages := map[string]map[string]string{
				"pages/a": {"generated.go": "package a\n\nimport \"discover_module/util\"\n\nvar _ = util.Answer\n"},
			}
			orchestrator.discoverUserPackages(context.Background(), packages, &fakeInterpreter{registered: testCase.registered})
			got := make([]string, 0, len(packages))
			for relativePath := range packages {
				got = append(got, relativePath)
			}
			assert.ElementsMatch(t, testCase.want, got)
			if files, ok := packages["util"]; ok {
				assert.Contains(t, files, "util.go")
				assert.NotContains(t, files, "util_test.go")
			}
		})
	}
}

func TestMarkDirtyPropagatesToDependentsAndJITCompilesThem(t *testing.T) {
	t.Parallel()
	const moduleName = "mark_dirty_module"
	interpreter := &fakeInterpreter{compile: registeringCompile}
	orchestrator, projectRoot := newTestOrchestrator(t, moduleName, &fakePool{interpreter: interpreter}, nil)
	card := newTestArtefact(projectRoot, moduleName, "partials/card.pk", "package card\n")
	page := withPikoImports(newTestArtefact(projectRoot, moduleName, "pages/a.pk", "package a\n"), moduleName+"/partials/card.pk")

	_, err := orchestrator.BuildRunner(context.Background(), newProjectResult(card, page))
	require.NoError(t, err)
	assert.Equal(t, []string{"pages/a.pk"}, orchestrator.GetAffectedComponents("partials/card.pk"))

	changedCard := newTestArtefact(projectRoot, moduleName, "partials/card.pk", "package card // edited\n")
	require.NoError(t, orchestrator.MarkDirty(context.Background(), newProjectResult(changedCard)))
	assert.Equal(t, "package card // edited\n", string(orchestrator.dirtyCodeCache["partials/card.pk"]))
	assert.Contains(t, orchestrator.dirtyCodeCache, "pages/a.pk", "the page importing the card is dirty too")

	require.NoError(t, orchestrator.JITCompile(context.Background(), "pages/a.pk"))
	assert.Empty(t, orchestrator.dirtyCodeCache)
	calls := interpreter.compileCalls()
	require.Len(t, calls, 2)
	assert.Equal(t, "package card // edited\n", calls[1].packages["partials/card"]["generated.go"])
	assert.Contains(t, calls[1].packages, "pages/a")
}

func TestMarkComponentsDirtyMergesTheManifest(t *testing.T) {
	t.Parallel()
	const moduleName = "mark_components_dirty_module"
	orchestrator, projectRoot := newTestOrchestrator(t, moduleName, &fakePool{interpreter: &fakeInterpreter{compile: registeringCompile}}, nil)
	first := newTestArtefact(projectRoot, moduleName, "pages/a.pk", "package a\n")
	second := newTestArtefact(projectRoot, moduleName, "pages/b.pk", "package b\n")
	_, err := orchestrator.BuildRunner(context.Background(), newProjectResult(first, second))
	require.NoError(t, err)

	changed := newTestArtefact(projectRoot, moduleName, "pages/b.pk", "package b // edited\n")
	require.NoError(t, orchestrator.MarkComponentsDirty(context.Background(), newProjectResult(changed)))
	assert.Equal(t, []string{"pages/b.pk"}, keysOf(orchestrator.dirtyCodeCache))
	assert.Contains(t, orchestrator.artefactByPackagePath, first.Component.CanonicalGoPackagePath)

	orchestrator.RemoveComponent(context.Background(), "pages/b.pk")
	assert.NotContains(t, orchestrator.dirtyCodeCache, "pages/b.pk")
	assert.NotContains(t, orchestrator.progCache, "pages/b.pk")
	assert.NotContains(t, orchestrator.artefactByPackagePath, changed.Component.CanonicalGoPackagePath)
}

func TestMarkDirtyIgnoresEmptyResults(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		artefacts any
		name      string
		wantErr   bool
	}{
		{name: "nil artefacts", artefacts: nil},
		{name: "no artefacts", artefacts: []*generator_dto.GeneratedArtefact{}},
		{name: "wrong artefact type", artefacts: "unexpected", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			orchestrator, _ := newTestOrchestrator(t, "mark_dirty_empty", &fakePool{}, nil)
			result := &annotator_dto.ProjectAnnotationResult{FinalGeneratedArtefacts: testCase.artefacts}
			for _, mark := range []func(context.Context, *annotator_dto.ProjectAnnotationResult) error{orchestrator.MarkDirty, orchestrator.MarkComponentsDirty} {
				err := mark(context.Background(), result)
				if testCase.wantErr {
					require.Error(t, err)
					continue
				}
				require.NoError(t, err)
			}
			assert.Empty(t, orchestrator.dirtyCodeCache)
		})
	}
}

func TestBuildRunnerRejectsMissingArtefacts(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		artefacts any
		name      string
	}{
		{name: "nil artefacts", artefacts: nil},
		{name: "wrong artefact type", artefacts: "unexpected"},
		{name: "no artefacts", artefacts: []*generator_dto.GeneratedArtefact{}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			const moduleName = "build_runner_missing_artefacts"
			orchestrator, projectRoot := newTestOrchestrator(t, moduleName, &fakePool{}, nil)
			result := newProjectResult(newTestArtefact(projectRoot, moduleName, "pages/a.pk", "package a\n"))
			result.FinalGeneratedArtefacts = testCase.artefacts
			_, err := orchestrator.BuildRunner(context.Background(), result)
			require.Error(t, err)
		})
	}
}

func keysOf(values map[string][]byte) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func TestJITCompileUsesBuiltCodeForCleanDependencies(t *testing.T) {
	t.Parallel()
	const moduleName = "jit_clean_dependency_module"
	interpreter := &fakeInterpreter{compile: registeringCompile}
	orchestrator, projectRoot := newTestOrchestrator(t, moduleName, &fakePool{interpreter: interpreter}, nil)
	card := newTestArtefact(projectRoot, moduleName, "partials/card.pk", "package card\n")
	page := withPikoImports(newTestArtefact(projectRoot, moduleName, "pages/a.pk", "package a\n"), moduleName+"/partials/card.pk")
	_, err := orchestrator.BuildRunner(context.Background(), newProjectResult(card, page))
	require.NoError(t, err)

	changedPage := withPikoImports(newTestArtefact(projectRoot, moduleName, "pages/a.pk", "package a // edited\n"), moduleName+"/partials/card.pk")
	require.NoError(t, orchestrator.MarkComponentsDirty(context.Background(), newProjectResult(changedPage)))
	require.NoError(t, orchestrator.JITCompile(context.Background(), "pages/a.pk"))

	calls := interpreter.compileCalls()
	require.Len(t, calls, 2)
	assert.Equal(t, "package a // edited\n", calls[1].packages["pages/a"]["generated.go"])
	assert.Equal(t, "package card\n", calls[1].packages["partials/card"]["generated.go"])
}

func TestInterpretedBuildOrchestrator_ReadUserGoFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	packageDirectory := filepath.Join(root, "helpers")
	require.NoError(t, os.MkdirAll(filepath.Join(packageDirectory, "nested"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(packageDirectory, "format.go"), []byte("package helpers\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(packageDirectory, "format_test.go"), []byte("package helpers\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(packageDirectory, "notes.md"), []byte("notes"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(packageDirectory, "secret.go"), []byte("package helpers\n"), 0o000))
	sandbox, err := safedisk.NewSandbox(root, safedisk.ModeReadOnly)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	testCases := []struct {
		want      map[string]string
		name      string
		directory string
	}{
		{
			name:      "reads readable Go sources and skips tests, directories and other files",
			directory: "helpers",
			want:      map[string]string{"format.go": "package helpers\n"},
		},
		{
			name:      "a missing directory yields no sources",
			directory: "absent",
			want:      nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			files := (&InterpretedBuildOrchestrator{}).readUserGoFiles(context.Background(), sandbox, testCase.directory)

			assert.Equal(t, testCase.want, files)
		})
	}
}

func TestInterpretedBuildOrchestrator_MergeManifest(t *testing.T) {
	t.Parallel()

	newManifest := func(pagePaths ...string) *generator_dto.Manifest {
		manifest := &generator_dto.Manifest{}
		manifest.Pages = map[string]generator_dto.ManifestPageEntry{}
		manifest.Partials = map[string]generator_dto.ManifestPartialEntry{}
		manifest.Emails = map[string]generator_dto.ManifestEmailEntry{}
		manifest.ErrorPages = map[string]generator_dto.ManifestErrorPageEntry{}
		for _, pagePath := range pagePaths {
			manifest.Pages[pagePath] = generator_dto.ManifestPageEntry{}
		}
		return manifest
	}
	testCases := []struct {
		cached    *generator_dto.Manifest
		incoming  *generator_dto.Manifest
		name      string
		wantPages []string
	}{
		{
			name:      "the first build adopts the incoming manifest",
			cached:    nil,
			incoming:  newManifest("pages/home.pk"),
			wantPages: []string{"pages/home.pk"},
		},
		{
			name:      "a targeted build keeps pages it did not rebuild",
			cached:    newManifest("pages/home.pk", "pages/about.pk"),
			incoming:  newManifest("pages/about.pk", "pages/contact.pk"),
			wantPages: []string{"pages/home.pk", "pages/about.pk", "pages/contact.pk"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			orchestrator := &InterpretedBuildOrchestrator{}
			orchestrator.cachedManifest = testCase.cached

			orchestrator.mergeManifest(testCase.incoming)

			require.NotNil(t, orchestrator.cachedManifest)
			pagePaths := make([]string, 0, len(orchestrator.cachedManifest.Pages))
			for pagePath := range orchestrator.cachedManifest.Pages {
				pagePaths = append(pagePaths, pagePath)
			}
			assert.ElementsMatch(t, testCase.wantPages, pagePaths)
		})
	}
}
