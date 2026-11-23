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
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/generator/generator_dto"
)

const (
	resolverTestModule = "example.org/shop/catalog/v2"
)

func newResolverTestArtefacts(projectRoot string, relativePaths ...string) []*generator_dto.GeneratedArtefact {
	artefacts := make([]*generator_dto.GeneratedArtefact, 0, len(relativePaths))
	for _, relativePath := range relativePaths {
		artefacts = append(artefacts, newTestArtefact(projectRoot, resolverTestModule, relativePath, "package x\n"))
	}
	return artefacts
}

func TestImportResolverImportRelativePath(t *testing.T) {
	t.Parallel()

	resolver := newImportResolver(t.TempDir(), resolverTestModule, slices.Values([]*generator_dto.GeneratedArtefact(nil)))

	testCases := []struct {
		name       string
		importPath string
		want       string
		wantLocal  bool
	}{
		{name: "another module", importPath: "myproject/components/card.pk", want: "components/card.pk", wantLocal: false},
		{name: "this module", importPath: resolverTestModule + "/partials/list/card.pk", want: "partials/list/card.pk", wantLocal: true},
		{name: "no slash", importPath: "card.pk", want: "card.pk", wantLocal: false},
		{name: "multiple slashes", importPath: "proj/a/b/c.pk", want: "a/b/c.pk", wantLocal: false},
		{name: "empty string", importPath: "", want: "", wantLocal: false},
		{name: "leading slash", importPath: "/path/to/file.pk", want: "path/to/file.pk", wantLocal: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, local := resolver.importRelativePath(testCase.importPath)
			assert.Equal(t, testCase.want, got)
			assert.Equal(t, testCase.wantLocal, local)
		})
	}
}

func TestImportResolverResolve(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	resolver := newImportResolver(projectRoot, resolverTestModule, slices.Values(newResolverTestArtefacts(projectRoot,
		"partials/card.pk", "partials/mycard.pk", "shop/x/card.pk", "admin/x/card.pk", "partials/list.pk",
	)))

	testCases := []struct {
		wantErr     error
		name        string
		importPath  string
		want        string
		wantPackage string
	}{
		{name: "exact path", importPath: "other/partials/card.pk", want: "partials/card.pk", wantPackage: resolverTestModule + "/partials/card"},
		{name: "suffix on a path boundary", importPath: "other/list.pk", want: "partials/list.pk", wantPackage: resolverTestModule + "/partials/list"},
		{name: "no match without a boundary", importPath: "other/ycard.pk"},
		{name: "unrelated", importPath: "other/other.pk"},
		{name: "empty", importPath: ""},
		{name: "ambiguous suffix", importPath: "other/x/card.pk", wantErr: errAmbiguousImport},
		{name: "local exact match", importPath: resolverTestModule + "/partials/mycard.pk", want: "partials/mycard.pk", wantPackage: resolverTestModule + "/partials/mycard"},
		{name: "local import never suffix matches", importPath: resolverTestModule + "/list.pk"},
		{name: "local card does not match mycard", importPath: resolverTestModule + "/card.pk"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, packagePath, err := resolver.resolve(testCase.importPath)
			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
				assert.ErrorContains(t, err, "admin/x/card.pk, shop/x/card.pk")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.want, got)
			assert.Equal(t, testCase.wantPackage, packagePath)
		})
	}
}

func TestImportResolverMatchesListsEveryAmbiguousCandidate(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	resolver := newImportResolver(projectRoot, resolverTestModule, slices.Values(newResolverTestArtefacts(projectRoot,
		"shop/x/card.pk", "admin/x/card.pk",
	)))

	assert.Equal(t, []string{"admin/x/card.pk", "shop/x/card.pk"}, resolver.matches("other/x/card.pk"))
	assert.Nil(t, resolver.matches(resolverTestModule+"/x/card.pk"))
}

func TestCollectDependenciesUsesTheSharedResolver(t *testing.T) {
	t.Parallel()

	o, projectRoot := newTestOrchestrator(t, resolverTestModule, nil, nil)
	card := newTestArtefact(projectRoot, resolverTestModule, "partials/card.pk", "package card\n")
	myCard := newTestArtefact(projectRoot, resolverTestModule, "partials/mycard.pk", "package mycard\n")
	page := withPikoImports(newTestArtefact(projectRoot, resolverTestModule, "pages/home.pk", "package home\n"), "other/card.pk")
	o.artefactByPackagePath = o.buildArtefactLookupMap([]*generator_dto.GeneratedArtefact{myCard, card, page})

	collected, err := o.collectDependencies("pages/home.pk")
	require.NoError(t, err)
	require.Len(t, collected, 2)
	assert.Same(t, card, collected[0], "card.pk resolves to partials/card.pk, never partials/mycard.pk")
	assert.Same(t, page, collected[1])

	ambiguous := newTestArtefact(projectRoot, resolverTestModule, "shop/card.pk", "package shopcard\n")
	o.artefactByPackagePath = o.buildArtefactLookupMap([]*generator_dto.GeneratedArtefact{card, ambiguous, page})
	_, err = o.collectDependencies("pages/home.pk")
	require.ErrorIs(t, err, errAmbiguousImport)

	_, err = o.collectDependencies("pages/missing.pk")
	require.Error(t, err)
}

func TestBuildReverseDependencyMapKeysByResolvedComponent(t *testing.T) {
	t.Parallel()

	o, projectRoot := newTestOrchestrator(t, resolverTestModule, nil, nil)
	card := newTestArtefact(projectRoot, resolverTestModule, "partials/card.pk", "package card\n")
	myCard := newTestArtefact(projectRoot, resolverTestModule, "partials/mycard.pk", "package mycard\n")
	shopCard := newTestArtefact(projectRoot, resolverTestModule, "shop/card.pk", "package shopcard\n")
	page := withPikoImports(newTestArtefact(projectRoot, resolverTestModule, "pages/home.pk", "package home\n"),
		resolverTestModule+"/partials/mycard.pk", "other/card.pk", "other/unknown.pk")
	self := withPikoImports(newTestArtefact(projectRoot, resolverTestModule, "pages/self.pk", "package self\n"), resolverTestModule+"/pages/self.pk")

	reverse := o.buildReverseDependencyMap([]*generator_dto.GeneratedArtefact{card, myCard, shopCard, page, self})

	assert.Equal(t, map[string][]string{
		"partials/mycard.pk": {"pages/home.pk"},
		"partials/card.pk":   {"pages/home.pk"},
		"shop/card.pk":       {"pages/home.pk"},
	}, reverse, "an ambiguous import marks every candidate, unknown imports and self imports are skipped")
}
