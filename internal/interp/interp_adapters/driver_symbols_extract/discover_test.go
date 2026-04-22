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
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

func TestDiscoverFiltersRegisteredStdlib(t *testing.T) {
	t.Parallel()

	root := writeFixtureProject(t, fixtureFiles{
		"go.mod": "module example.com/fixture\n\ngo 1.22\n",
		"pages/home.pk": `<template><div></div></template>
<script type="application/x-go">
package main

import (
	"fmt"
	"strings"
)

func Hello() string {
	return fmt.Sprintf("hello %s", strings.ToUpper("world"))
}
</script>`,
	})

	result, err := Discover(context.Background(), DiscoverOptions{Root: root})
	require.NoError(t, err)
	require.Empty(t, result.RequiredImports,
		"fmt and strings are registered stdlib and must be filtered, got %v", result.RequiredImports)
	require.Empty(t, result.SkippedCgo)
	require.Empty(t, result.GenericCandidates)
}

func TestDiscoverSurfacesUnregisteredStdlib(t *testing.T) {
	t.Parallel()

	root := writeFixtureProject(t, fixtureFiles{
		"go.mod": "module example.com/fixture\n\ngo 1.22\n",
		"pages/home.pk": `<template><div></div></template>
<script type="application/x-go">
package main

import (
	"net/http"
	"encoding/xml"
)

var _ = http.MethodGet
var _ = xml.Name{}
</script>`,
	})

	result, err := Discover(context.Background(), DiscoverOptions{Root: root})
	require.NoError(t, err)
	require.Contains(t, result.RequiredImports, "net/http",
		"unregistered stdlib must surface so user can register it")
	require.Contains(t, result.RequiredImports, "encoding/xml",
		"unregistered stdlib must surface so user can register it")
}

func TestDiscoverIncludesProjectGoPackages(t *testing.T) {
	t.Parallel()

	root := writeFixtureProject(t, fixtureFiles{
		"go.mod": "module example.com/fixture\n\ngo 1.22\n",
		"pkg/helpers/helpers.go": `package helpers

func Double(x int) int { return x * 2 }
`,
		"pages/home.pk": `<template><div></div></template>
<script type="application/x-go">
package main

import (
	"example.com/fixture/pkg/helpers"
)

func Hello() int { return helpers.Double(21) }
</script>`,
	})

	result, err := Discover(context.Background(), DiscoverOptions{Root: root})
	require.NoError(t, err)
	require.Contains(t, result.RequiredImports, "example.com/fixture/pkg/helpers",
		"project-local Go packages consumed from .pk scripts must surface")
}

func TestDiscoverDropsVirtualPathsWithoutGoFiles(t *testing.T) {
	t.Parallel()

	root := writeFixtureProject(t, fixtureFiles{
		"go.mod":                    "module example.com/fixture\n\ngo 1.22\n",
		"pkg/not_a_package/doc.txt": "this folder has no Go files",
		"pages/home.pk": `<template><div></div></template>
<script type="application/x-go">
package main

import _ "example.com/fixture/pkg/not_a_package"
</script>`,
	})

	result, err := Discover(context.Background(), DiscoverOptions{Root: root})
	require.NoError(t, err)
	require.NotContains(t, result.RequiredImports, "example.com/fixture/pkg/not_a_package",
		"paths without Go source must not surface as required")
}

func TestDiscoverRespectsExtraIgnored(t *testing.T) {
	t.Parallel()

	root := writeFixtureProject(t, fixtureFiles{
		"go.mod": "module example.com/fixture\n\ngo 1.22\n",
		"pages/home.pk": `<template><div></div></template>
<script type="application/x-go">
package main

import "example.com/external/thing"

var _ = thing.Name
</script>`,
	})

	result, err := Discover(context.Background(), DiscoverOptions{
		Root:         root,
		ExtraIgnored: []string{"example.com/external/thing"},
	})
	require.NoError(t, err)
	require.NotContains(t, result.RequiredImports, "example.com/external/thing")
}

func TestDiscoverSurfacesTheDirectPkSeam(t *testing.T) {
	t.Parallel()

	root := writeFixtureProject(t, fixtureFiles{
		"go.mod": "module example.com/fixture\n\ngo 1.22\n",
		"pkg/local/local.go": `package local

func Something() int { return 1 }
`,
		"pages/home.pk": `<template><div></div></template>
<script type="application/x-go">
package main

import (
	"fmt"
	"piko.sh/piko"
	"example.com/fixture/pkg/local"
	header "example.com/fixture/partials/shared/header.pk"
)

var (
	_ = fmt.Sprint
	_ = local.Something
	_ = header
)
</script>`,
	})

	result, err := Discover(context.Background(), DiscoverOptions{Root: root})
	require.NoError(t, err)
	require.NotContains(t, result.RequiredImports, "fmt")
	require.NotContains(t, result.RequiredImports, "piko.sh/piko")
	require.NotContains(t, result.RequiredImports, "example.com/fixture/partials/shared/header.pk")
	require.Contains(t, result.RequiredImports, "example.com/fixture/pkg/local")
}

func TestDiscoverSortsAndDeduplicates(t *testing.T) {
	t.Parallel()

	root := writeFixtureProject(t, fixtureFiles{
		"go.mod": "module example.com/fixture\n\ngo 1.22\n",
		"pages/a.pk": withScript(`package main
import "fmt"
var _ = fmt.Sprint
`),
		"pages/b.pk": withScript(`package main
import "fmt"
var _ = fmt.Sprint
`),
		"partials/c.pk": withScript(`package main
import "strings"
var _ = strings.ToUpper
`),
	})

	result, err := Discover(context.Background(), DiscoverOptions{Root: root})
	require.NoError(t, err)
	require.Empty(t, result.RequiredImports)
	require.True(t, slices.IsSorted(result.RequiredImports))
}

func TestDiscoverMissingDirectoriesIsOK(t *testing.T) {
	t.Parallel()

	root := writeFixtureProject(t, fixtureFiles{
		"go.mod": "module example.com/fixture\n\ngo 1.22\n",
	})

	result, err := Discover(context.Background(), DiscoverOptions{
		Root:       root,
		SourceDirs: []string{"no_such_dir"},
	})
	require.NoError(t, err)
	require.Empty(t, result.RequiredImports)
}

func TestDiscoverRejectsCancelledContext(t *testing.T) {
	t.Parallel()

	root := writeFixtureProject(t, fixtureFiles{
		"go.mod":        "module example.com/fixture\n\ngo 1.22\n",
		"pages/home.pk": withScript("package main\n"),
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Discover(ctx, DiscoverOptions{Root: root})
	require.ErrorIs(t, err, context.Canceled)
}

func TestReadBoundedFileRejectsOversize(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "huge.pk")
	require.NoError(t, os.WriteFile(path, make([]byte, 2048), 0o644))

	_, err := readBoundedFile(path, 1024)
	require.ErrorIs(t, err, errPKFileTooLarge)
}

func TestReadBoundedFileAcceptsUnderLimit(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "ok.pk")
	payload := []byte("hello world")
	require.NoError(t, os.WriteFile(path, payload, 0o644))

	got, err := readBoundedFile(path, 1024)
	require.NoError(t, err)
	require.Equal(t, payload, got)
}

func TestPackageExportsGenericType(t *testing.T) {
	t.Parallel()

	genericPkg := loadPackageFromSource(t, "example.com/gen", `package gen

type Box[T any] struct { Value T }

func NewBox[T any](v T) Box[T] { return Box[T]{Value: v} }
`)
	require.True(t, packageExportsGenericType(genericPkg),
		"package with exported generic type must be flagged")

	nonGenericPkg := loadPackageFromSource(t, "example.com/nogen", `package nogen

type Simple struct { Name string }
`)
	require.False(t, packageExportsGenericType(nonGenericPkg),
		"package without generic types must not be flagged")

	nilPkg := &packages.Package{PkgPath: "example.com/empty"}
	require.False(t, packageExportsGenericType(nilPkg),
		"package without Types must not be flagged")
}

func TestPackageUsesCgo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pkg  *packages.Package
		want bool
	}{
		{
			name: "no cgo",
			pkg:  &packages.Package{PkgPath: "example.com/pure"},
			want: false,
		},
		{
			name: "imports C pseudo-package",
			pkg: &packages.Package{
				PkgPath: "example.com/cgoimport",
				Imports: map[string]*packages.Package{
					"C": {PkgPath: "C"},
				},
			},
			want: true,
		},
		{
			name: "has cgo1 generated file",
			pkg: &packages.Package{
				PkgPath: "example.com/cgogen",
				GoFiles: []string{"/tmp/x.cgo1.go"},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, packageUsesCgo(tt.pkg))
		})
	}
}

func loadPackageFromSource(t *testing.T, importPath, source string) *packages.Package {
	t.Helper()
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "source.go", source, parser.SkipObjectResolution)
	require.NoError(t, err)

	conf := types.Config{}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}}
	pkg, err := conf.Check(importPath, fileSet, []*ast.File{file}, info)
	require.NoError(t, err)

	return &packages.Package{
		PkgPath: importPath,
		Name:    pkg.Name(),
		Types:   pkg,
	}
}

type fixtureFiles map[string]string

func writeFixtureProject(t *testing.T, files fixtureFiles) string {
	t.Helper()
	root := t.TempDir()
	for relativePath, content := range files {
		full := filepath.Join(root, relativePath)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
	return root
}

func withScript(script string) string {
	return "<template><div></div></template>\n<script type=\"application/x-go\">\n" + script + "\n</script>"
}
