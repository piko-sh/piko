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

package annotator_domain

import (
	"slices"
	"testing"

	goast "go/ast"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

const (
	annotatorDeterminismRuns = 32
)

func newAmbiguousModelPackages() map[string]*inspector_dto.Package {
	newPackage := func(path, name string) *inspector_dto.Package {
		return &inspector_dto.Package{
			Path: path,
			Name: name,
			NamedTypes: map[string]*inspector_dto.Type{
				"User": {Name: "User", PackagePath: path},
			},
		}
	}
	return map[string]*inspector_dto.Package{
		"example.com/d/models":  newPackage("example.com/d/models", "models"),
		"example.com/b/models":  newPackage("example.com/b/models", "models"),
		"example.com/0/models":  newPackage("example.com/0/models", "legacymodels"),
		"example.com/c/models":  newPackage("example.com/c/models", "models"),
		"example.com/a/models":  newPackage("example.com/a/models", "models"),
		"example.com/e/models":  newPackage("example.com/e/models", "models"),
		"example.com/f/models":  newPackage("example.com/f/models", "models"),
		"example.com/g/unknown": newPackage("example.com/g/unknown", "models"),
	}
}

func TestMaybeAddTemplateOnlyUseDecl_PicksSmallestNonCallMember(t *testing.T) {
	t.Parallel()

	ar := &astRewriter{}
	templateUses := map[string]map[string]templateMemberInfo{
		"models": {
			"StatusDraft":    {isCall: false},
			"StatusActive":   {isCall: false},
			"StatusArchived": {isCall: false},
			"Lookup":         {isCall: true},
			"Zeta":           {isCall: false},
			"Omega":          {isCall: false},
			"Kappa":          {isCall: false},
			"Delta":          {isCall: false},
			"Alpha":          {isCall: true},
		},
	}

	for range annotatorDeterminismRuns {
		var useDecls []goast.Decl

		ar.maybeAddTemplateOnlyUseDecl("models", false, false, true, templateUses, &useDecls)

		require.Len(t, useDecls, 1)
		genDecl, ok := useDecls[0].(*goast.GenDecl)
		require.True(t, ok)
		valueSpec, ok := genDecl.Specs[0].(*goast.ValueSpec)
		require.True(t, ok)
		selector, ok := valueSpec.Values[0].(*goast.SelectorExpr)
		require.True(t, ok)
		require.Equal(t, "Delta", selector.Sel.Name, "the use declaration must name the same member on every run")
	}
}

func TestFindPackagePathDefiningType_PicksSmallestPath(t *testing.T) {
	t.Parallel()

	packages := newAmbiguousModelPackages()

	for range annotatorDeterminismRuns {
		require.Equal(t, "example.com/a/models", findPackagePathDefiningType(packages, "models", "User"))
	}
	require.Empty(t, findPackagePathDefiningType(packages, "models", "Missing"))
	require.Empty(t, findPackagePathDefiningType(packages, "absent", "User"))
}

func TestExtractTypeInfoFromString_AmbiguousAliasIsDeterministic(t *testing.T) {
	t.Parallel()

	packages := newAmbiguousModelPackages()

	for range annotatorDeterminismRuns {
		typeInfo := extractTypeInfoFromString("*models.User", packages)

		require.Equal(t, "User", typeInfo.Name)
		require.Equal(t, "example.com/a/models", typeInfo.PackagePath, "an exact package name match on the smallest path must win on every run")
		require.Equal(t, "models", typeInfo.PackageName)
	}
}

func TestExtractTypeInfoFromString_SuffixMatchUsedWhenNoNameMatches(t *testing.T) {
	t.Parallel()

	packages := newAmbiguousModelPackages()
	delete(packages, "example.com/g/unknown")
	for path, pkg := range packages {
		if pkg.Name == "models" {
			delete(packages, path)
		}
	}

	typeInfo := extractTypeInfoFromString("models.User", packages)

	require.Equal(t, "example.com/0/models", typeInfo.PackagePath)
	require.Equal(t, "legacymodels", typeInfo.PackageName)
}

func TestBuildSrcsetMetadata_TiesOrderedIndependentlyOfInput(t *testing.T) {
	t.Parallel()

	newVariant := func(width, density, format string) *annotator_dto.StaticAssetDependency {
		return &annotator_dto.StaticAssetDependency{
			SourcePath: "hero.jpg",
			AssetType:  "img",
			TransformationParams: map[string]string{
				"width":    width,
				"_density": density,
				"format":   format,
			},
		}
	}
	variants := []*annotator_dto.StaticAssetDependency{
		newVariant("400px", "1x", "webp"),
		newVariant("400px", "1x", "avif"),
		newVariant("400px", "1x", "jpeg"),
		newVariant("800px", "2x", "webp"),
		newVariant("800px", "2x", "avif"),
	}

	want := buildSrcsetMetadata(variants, AnnotatorPathsConfig{})
	wantKeys := make([]string, 0, len(want))
	for _, metadata := range want {
		wantKeys = append(wantKeys, metadata.VariantKey)
	}
	require.Equal(t, []string{"image_w400_avif", "image_w400_jpeg", "image_w400_webp", "image_w800_avif", "image_w800_webp"}, wantKeys)

	reversed := slices.Clone(variants)
	slices.Reverse(reversed)
	require.Equal(t, want, buildSrcsetMetadata(reversed, AnnotatorPathsConfig{}))

	rotated := append(slices.Clone(variants[2:]), variants[:2]...)
	require.Equal(t, want, buildSrcsetMetadata(rotated, AnnotatorPathsConfig{}))
}

func TestBuildSrcsetMetadata_ResultIsSortedTotally(t *testing.T) {
	t.Parallel()

	metadata := []ast_domain.ResponsiveVariantMetadata{
		{Width: 400, Density: "1x", VariantKey: "b", URL: "/b"},
		{Width: 400, Density: "1x", VariantKey: "a", URL: "/a"},
	}

	require.Equal(t, -1, compareResponsiveVariantMetadata(metadata[1], metadata[0]))
	require.Equal(t, 1, compareResponsiveVariantMetadata(metadata[0], metadata[1]))
	require.Equal(t, 0, compareResponsiveVariantMetadata(metadata[0], metadata[0]))
}
