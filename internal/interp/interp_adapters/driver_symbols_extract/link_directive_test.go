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

package driver_symbols_extract

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

func loadFixturePackage(t *testing.T, files fixtureFiles) *packages.Package {
	t.Helper()
	root := writeFixtureProject(t, files)
	config := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedName | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedFiles,
		Dir:  root,
	}
	pkgs, err := packages.Load(config, "./...")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	require.Empty(t, pkgs[0].Errors)
	return pkgs[0]
}

func TestCollectLinkDirectivesFindsValidDirective(t *testing.T) {
	t.Parallel()

	pkg := loadFixturePackage(t, fixtureFiles{
		"go.mod": "module example.com/linkfixture\n\ngo 1.22\n",
		"pkg.go": `package linkfixture

import "reflect"

//piko:link GetItemLink
func GetItem[T any](slot int) T {
	var zero T
	return zero
}

func GetItemLink(tType reflect.Type, slot int) reflect.Value {
	return reflect.New(tType).Elem()
}
`,
	})

	directives, err := collectLinkDirectives(pkg)
	require.NoError(t, err)
	require.Len(t, directives, 1)
	require.Equal(t, "GetItem", directives[0].GenericName)
	require.Equal(t, "GetItemLink", directives[0].LinkTarget)
}

func TestCollectLinkDirectivesIgnoresUnrelatedFunctions(t *testing.T) {
	t.Parallel()

	pkg := loadFixturePackage(t, fixtureFiles{
		"go.mod": "module example.com/linkfixture\n\ngo 1.22\n",
		"pkg.go": `package linkfixture

// Plain is just a regular function.
func Plain() int { return 1 }

// Other does math and has a doc comment but no directive.
func Other(x int) int { return x * 2 }
`,
	})

	directives, err := collectLinkDirectives(pkg)
	require.NoError(t, err)
	require.Empty(t, directives)
}

func TestCollectLinkDirectivesRejectsMalformed(t *testing.T) {
	t.Parallel()

	pkg := loadFixturePackage(t, fixtureFiles{
		"go.mod": "module example.com/linkfixture\n\ngo 1.22\n",
		"pkg.go": `package linkfixture

//piko:link
func Bad[T any]() T { var zero T; return zero }
`,
	})

	_, err := collectLinkDirectives(pkg)
	require.ErrorIs(t, err, errLinkDirectiveMalformed)
}

func TestCollectLinkDirectivesRejectsExtraTokens(t *testing.T) {
	t.Parallel()

	pkg := loadFixturePackage(t, fixtureFiles{
		"go.mod": "module example.com/linkfixture\n\ngo 1.22\n",
		"pkg.go": `package linkfixture

//piko:link TargetA TargetB
func Bad[T any]() T { var zero T; return zero }
`,
	})

	_, err := collectLinkDirectives(pkg)
	require.ErrorIs(t, err, errLinkDirectiveMalformed)
}

func TestCollectLinkDirectivesRejectsDuplicate(t *testing.T) {
	t.Parallel()

	pkg := loadFixturePackage(t, fixtureFiles{
		"go.mod": "module example.com/linkfixture\n\ngo 1.22\n",
		"pkg.go": `package linkfixture

//piko:link Sibling1
//piko:link Sibling2
func Dup[T any]() T { var zero T; return zero }
`,
	})

	_, err := collectLinkDirectives(pkg)
	require.ErrorIs(t, err, errLinkDirectiveDuplicate)
}

func TestValidateLinkDirectivesRejectsMissingTarget(t *testing.T) {
	t.Parallel()

	pkg := loadFixturePackage(t, fixtureFiles{
		"go.mod": "module example.com/linkfixture\n\ngo 1.22\n",
		"pkg.go": `package linkfixture

//piko:link DoesNotExist
func GetItem[T any]() T { var zero T; return zero }
`,
	})

	directives, err := collectLinkDirectives(pkg)
	require.NoError(t, err)
	_, err = validateLinkDirectives(pkg, directives)
	require.ErrorIs(t, err, errLinkTargetMissing)
}

func TestValidateLinkDirectivesRejectsArityMismatch(t *testing.T) {
	t.Parallel()

	pkg := loadFixturePackage(t, fixtureFiles{
		"go.mod": "module example.com/linkfixture\n\ngo 1.22\n",
		"pkg.go": `package linkfixture

import "reflect"

//piko:link TooFew
func GetItem[T any](slot int, extra string) T { var zero T; return zero }

// TooFew expects 3 params (1 type arg + 2 regular), but gives only 2.
func TooFew(t reflect.Type, slot int) reflect.Value {
	return reflect.New(t).Elem()
}
`,
	})

	directives, err := collectLinkDirectives(pkg)
	require.NoError(t, err)
	_, err = validateLinkDirectives(pkg, directives)
	require.ErrorIs(t, err, errLinkTargetArity)
}

func TestValidateLinkDirectivesRejectsWrongTypePrefix(t *testing.T) {
	t.Parallel()

	pkg := loadFixturePackage(t, fixtureFiles{
		"go.mod": "module example.com/linkfixture\n\ngo 1.22\n",
		"pkg.go": `package linkfixture

import "reflect"

//piko:link BadPrefix
func GetItem[T any](slot int) T { var zero T; return zero }

// BadPrefix has the right arity (2 params = 1 type arg + 1 regular)
// but the leading slot is typed as int instead of reflect.Type. The
// runtime dispatcher would place a reflect.Type into the int slot
// and panic; extract must reject the shape.
func BadPrefix(slot int, extra int) reflect.Value {
	return reflect.ValueOf(slot + extra)
}
`,
	})

	directives, err := collectLinkDirectives(pkg)
	require.NoError(t, err)
	_, err = validateLinkDirectives(pkg, directives)
	require.ErrorIs(t, err, errLinkTargetTypePrefix)
}

func TestValidateLinkDirectivesRejectsNonGeneric(t *testing.T) {
	t.Parallel()

	pkg := loadFixturePackage(t, fixtureFiles{
		"go.mod": "module example.com/linkfixture\n\ngo 1.22\n",
		"pkg.go": `package linkfixture

//piko:link Sibling
func Plain(slot int) int { return slot }

func Sibling(slot int) int { return slot }
`,
	})

	directives, err := collectLinkDirectives(pkg)
	require.NoError(t, err)
	_, err = validateLinkDirectives(pkg, directives)
	require.ErrorIs(t, err, errLinkDirectiveOnNonGeneric)
}

func TestExtractSurfacesLinkedGenericFuncs(t *testing.T) {
	t.Parallel()

	pkg := loadFixturePackage(t, fixtureFiles{
		"go.mod": "module example.com/linkfixture\n\ngo 1.22\n",
		"pkg.go": `package linkfixture

import "reflect"

//piko:link GetItemLink
func GetItem[T any](slot int) T {
	var zero T
	return zero
}

func GetItemLink(tType reflect.Type, slot int) reflect.Value {
	return reflect.New(tType).Elem()
}
`,
	})

	ep, err := extractPackage(pkg, false)
	require.NoError(t, err)
	require.Len(t, ep.LinkedGenericFuncs, 1)
	linked := ep.LinkedGenericFuncs[0]
	require.Equal(t, "GetItem", linked.Name)
	require.Equal(t, "GetItemLink", linked.LinkTarget)
	require.Equal(t, 1, linked.TypeArgCount)

	for _, generic := range ep.GenericFuncs {
		require.NotEqual(t, "GetItem", generic.Name,
			"linked generics must not also appear in GenericFuncs")
	}
}

func TestGenerateFileEmitsInterpLinkWrap(t *testing.T) {
	t.Parallel()

	pkg := ExtractedPackage{
		ImportPath: "example.com/linkfixture",
		Name:       "linkfixture",
		LinkedGenericFuncs: []LinkedGenericFuncInfo{
			{Name: "GetItem", LinkTarget: "GetItemLink", TypeArgCount: 1},
		},
	}

	source, err := GenerateFile(pkg, "gen_pkg", PackageConfig{})
	require.NoError(t, err)
	require.NotNil(t, source)

	src := string(source)
	require.Contains(t, src, `"piko.sh/piko/wdk/interp/interp_link"`)
	require.Contains(t, src, "interp_link.WrapFunc(1, linkfixture.GetItemLink")
	require.Contains(t, src, `reflect.ValueOf(interp_link.WrapFunc`)
	require.True(t, strings.Contains(src, `"GetItem":`),
		"map key must be the generic's exported name, got:\n%s", src)
}

func TestIsValidIdentifier(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		"":            false,
		"1abc":        false,
		"abc":         true,
		"Abc":         true,
		"_foo":        true,
		"foo_bar":     true,
		"foo1":        true,
		"hello world": false,
		"foo-bar":     false,
	}
	for input, want := range cases {
		require.Equal(t, want, isValidIdentifier(input), "input=%q", input)
	}
}
