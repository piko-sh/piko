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

package quickpackages

import (
	"context"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
	"piko.sh/piko/wdk/safedisk"
)

func makeLoaderPkg(id string, importPaths []string, importMap map[string]string) *loaderPkg {
	return &loaderPkg{
		Package: &packages.Package{
			ID:      id,
			PkgPath: id,
			Name:    id,
			Imports: make(map[string]*packages.Package),
		},
		importPaths: importPaths,
		importMap:   importMap,
	}
}

func TestResolveImportPath(t *testing.T) {
	t.Parallel()

	t.Run("nil map returns original", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "fmt", resolveImportPath(nil, "fmt"))
	})

	t.Run("missing key returns original", func(t *testing.T) {
		t.Parallel()
		importMap := map[string]string{"x": "y"}
		assert.Equal(t, "fmt", resolveImportPath(importMap, "fmt"))
	})

	t.Run("found key returns mapped value", func(t *testing.T) {
		t.Parallel()
		importMap := map[string]string{"golang.org/x/net": "vendor/golang.org/x/net"}
		assert.Equal(t, "vendor/golang.org/x/net", resolveImportPath(importMap, "golang.org/x/net"))
	})
}

func TestGoarch(t *testing.T) {
	t.Parallel()

	t.Run("empty env defaults to runtime", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, runtime.GOARCH, goarch(nil))
	})

	t.Run("explicit GOARCH", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "arm64", goarch([]string{"GOARCH=arm64"}))
	})

	t.Run("GOARCH among other vars", func(t *testing.T) {
		t.Parallel()
		env := []string{"GOPATH=/tmp", "GOARCH=amd64", "HOME=/home"}
		assert.Equal(t, "amd64", goarch(env))
	})
}

func TestSelectParseInitial(t *testing.T) {
	t.Parallel()

	t.Run("nil callback returns default with comments", func(t *testing.T) {
		t.Parallel()
		parseFn := selectParseInitial(nil)
		fset := token.NewFileSet()
		file, err := parseFn(fset, "test.go", []byte("package foo\n// A comment.\nvar X int\n"))
		require.NoError(t, err)
		require.NotNil(t, file)
		assert.NotEmpty(t, file.Comments)
	})

	t.Run("non-nil callback is returned as-is", func(t *testing.T) {
		t.Parallel()
		called := false
		custom := func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
			called = true
			return parser.ParseFile(fset, filename, src, parser.AllErrors)
		}
		parseFn := selectParseInitial(custom)
		fset := token.NewFileSet()
		_, err := parseFn(fset, "test.go", []byte("package foo\n"))
		require.NoError(t, err)
		assert.True(t, called)
	})
}

func TestParseDep(t *testing.T) {
	t.Parallel()

	t.Run("regular function body stripped", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		file, err := parseDep(fset, "test.go", []byte("package foo\nfunc Foo() { x := 1; _ = x }\n"))
		require.NoError(t, err)
		require.NotNil(t, file)
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			assert.Nil(t, fn.Body)
		}
	})

	t.Run("init body preserved", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		file, err := parseDep(fset, "test.go", []byte("package foo\nfunc init() { x := 1; _ = x }\n"))
		require.NoError(t, err)
		require.NotNil(t, file)
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fn.Name.Name == "init" {
				assert.NotNil(t, fn.Body)
			}
		}
	})

	t.Run("generic function body preserved", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		src := "package foo\nfunc Map[T any](s []T) []T { return s }\n"
		file, err := parseDep(fset, "test.go", []byte(src))
		require.NoError(t, err)
		require.NotNil(t, file)
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fn.Name.Name == "Map" {
				assert.NotNil(t, fn.Body)
			}
		}
	})

	t.Run("parse error returns error and partial file", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		file, err := parseDep(fset, "test.go", []byte("package foo\nfunc {"))
		assert.Error(t, err)
		assert.NotNil(t, file)
	})
}

func TestCollectOverlayFiles(t *testing.T) {
	t.Parallel()

	t.Run("empty map", func(t *testing.T) {
		t.Parallel()
		result := collectOverlayFiles(map[string][]byte{})
		assert.Empty(t, result)
	})

	t.Run("populated map", func(t *testing.T) {
		t.Parallel()
		overlay := map[string][]byte{
			"/a.go": []byte("package a"),
			"/b.go": []byte("package b"),
		}
		result := collectOverlayFiles(overlay)
		assert.Len(t, result, 2)
		assert.Equal(t, "package a", result["/a.go"])
		assert.Equal(t, "package b", result["/b.go"])
	})
}

func TestCreatePackages(t *testing.T) {
	t.Parallel()

	noopParse := func(_ *token.FileSet, _ string, _ []byte) (*ast.File, error) {
		return nil, nil
	}

	t.Run("single initial package", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		listed := []goListPkg{
			{
				ImportPath: "test/foo",
				Name:       "foo",
				Dir:        "/src/foo",
				GoFiles:    []string{"foo.go"},
				DepOnly:    false,
			},
		}
		pkgMap, roots := createPackages(listed, fset, noopParse, noopParse)

		require.Len(t, pkgMap, 1)
		require.Len(t, roots, 1)

		lp := pkgMap["test/foo"]
		require.NotNil(t, lp)
		assert.Equal(t, "test/foo", lp.PkgPath)
		assert.Equal(t, "foo", lp.Name)
		assert.True(t, lp.initial)
		assert.Equal(t, []string{"/src/foo/foo.go"}, lp.goFiles)
	})

	t.Run("initial and dep-only", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		listed := []goListPkg{
			{ImportPath: "test/root", Name: "root", DepOnly: false},
			{ImportPath: "test/dep", Name: "dep", DepOnly: true},
		}
		pkgMap, roots := createPackages(listed, fset, noopParse, noopParse)

		assert.Len(t, pkgMap, 2)
		assert.Len(t, roots, 1)
		assert.Equal(t, "test/root", roots[0].PkgPath)
		assert.True(t, pkgMap["test/root"].initial)
		assert.False(t, pkgMap["test/dep"].initial)
	})

	t.Run("error propagated from go list", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		listed := []goListPkg{
			{
				ImportPath: "test/broken",
				Name:       "broken",
				Error:      &goListError{Pos: "broken.go:1:1", Err: "something went wrong"},
			},
		}
		pkgMap, _ := createPackages(listed, fset, noopParse, noopParse)

		lp := pkgMap["test/broken"]
		require.Len(t, lp.Errors, 1)
		assert.Equal(t, packages.ListError, lp.Errors[0].Kind)
		assert.Equal(t, "something went wrong", lp.Errors[0].Msg)
	})
}

func TestWireImports(t *testing.T) {
	t.Parallel()

	t.Run("simple chain A imports B", func(t *testing.T) {
		t.Parallel()
		pkgA := makeLoaderPkg("A", []string{"B"}, nil)
		pkgB := makeLoaderPkg("B", nil, nil)
		pkgMap := map[string]*loaderPkg{"A": pkgA, "B": pkgB}

		wireImports(pkgMap)

		assert.Contains(t, pkgA.Imports, "B")
		assert.Equal(t, pkgB.Package, pkgA.Imports["B"])
		assert.Contains(t, pkgB.preds, pkgA)
		assert.Equal(t, int32(1), pkgA.unfinishedDeps.Load())
		assert.Equal(t, int32(0), pkgB.unfinishedDeps.Load())
	})

	t.Run("diamond dependency", func(t *testing.T) {
		t.Parallel()
		pkgA := makeLoaderPkg("A", []string{"B", "C"}, nil)
		pkgB := makeLoaderPkg("B", []string{"D"}, nil)
		pkgC := makeLoaderPkg("C", []string{"D"}, nil)
		pkgD := makeLoaderPkg("D", nil, nil)
		pkgMap := map[string]*loaderPkg{"A": pkgA, "B": pkgB, "C": pkgC, "D": pkgD}

		wireImports(pkgMap)

		assert.Equal(t, int32(2), pkgA.unfinishedDeps.Load())
		assert.Equal(t, int32(1), pkgB.unfinishedDeps.Load())
		assert.Equal(t, int32(1), pkgC.unfinishedDeps.Load())
		assert.Equal(t, int32(0), pkgD.unfinishedDeps.Load())
		assert.Len(t, pkgD.preds, 2)
	})

	t.Run("missing dep skipped", func(t *testing.T) {
		t.Parallel()
		pkgA := makeLoaderPkg("A", []string{"missing"}, nil)
		pkgMap := map[string]*loaderPkg{"A": pkgA}

		wireImports(pkgMap)

		assert.NotContains(t, pkgA.Imports, "missing")
		assert.Equal(t, int32(0), pkgA.unfinishedDeps.Load())
	})

	t.Run("import map resolution", func(t *testing.T) {
		t.Parallel()
		pkgA := makeLoaderPkg("A", []string{"golang.org/x/net"}, map[string]string{
			"golang.org/x/net": "vendor/golang.org/x/net",
		})
		pkgVendored := makeLoaderPkg("vendor/golang.org/x/net", nil, nil)
		pkgMap := map[string]*loaderPkg{"A": pkgA, "vendor/golang.org/x/net": pkgVendored}

		wireImports(pkgMap)

		assert.Contains(t, pkgA.Imports, "golang.org/x/net")
		assert.Equal(t, pkgVendored.Package, pkgA.Imports["golang.org/x/net"])
		assert.Equal(t, int32(1), pkgA.unfinishedDeps.Load())
	})
}

func TestBuildPackageGraph(t *testing.T) {
	t.Parallel()

	noopParse := func(_ *token.FileSet, _ string, _ []byte) (*ast.File, error) {
		return nil, nil
	}

	t.Run("two packages A imports B", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		listed := []goListPkg{
			{ImportPath: "A", Name: "a", Imports: []string{"B"}, DepOnly: false},
			{ImportPath: "B", Name: "b", DepOnly: true},
		}

		pkgMap, roots, err := buildPackageGraph(listed, fset, noopParse, noopParse)
		require.NoError(t, err)

		assert.Len(t, pkgMap, 2)
		assert.Len(t, roots, 1)
		assert.Equal(t, "A", roots[0].PkgPath)
		assert.Contains(t, pkgMap["A"].Imports, "B")
		assert.Equal(t, int32(1), pkgMap["A"].unfinishedDeps.Load())
		assert.Equal(t, int32(0), pkgMap["B"].unfinishedDeps.Load())
	})
}

func TestParsePackageFiles(t *testing.T) {
	t.Parallel()

	t.Run("overlay takes priority over readFile", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		st := &typeCheckState{
			fset:    fset,
			overlay: map[string][]byte{"/src/a.go": []byte("package a\n")},
			readFile: func(name string) ([]byte, error) {
				t.Fatal("readFile should not be called for overlay file")
				return nil, nil
			},
		}
		lp := &loaderPkg{
			Package: &packages.Package{},
			goFiles: []string{"/src/a.go"},
			parseFile: func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
				return parser.ParseFile(fset, filename, src, parser.AllErrors)
			},
		}

		result := st.parsePackageFiles(context.Background(), lp)
		assert.Len(t, result, 1)
		assert.Empty(t, lp.Errors)
	})

	t.Run("readFile error records parse error", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		st := &typeCheckState{
			fset:    fset,
			overlay: nil,
			readFile: func(name string) ([]byte, error) {
				return nil, errors.New("disk failure")
			},
		}
		lp := &loaderPkg{
			Package: &packages.Package{},
			goFiles: []string{"/src/missing.go"},
		}

		result := st.parsePackageFiles(context.Background(), lp)
		assert.Empty(t, result)
		require.Len(t, lp.Errors, 1)
		assert.Equal(t, packages.ParseError, lp.Errors[0].Kind)
		assert.Contains(t, lp.Errors[0].Msg, "disk failure")
	})

	t.Run("parse error with nil file skips file", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		st := &typeCheckState{
			fset:     fset,
			overlay:  map[string][]byte{"/src/bad.go": []byte("not valid go")},
			readFile: os.ReadFile,
		}
		lp := &loaderPkg{
			Package: &packages.Package{},
			goFiles: []string{"/src/bad.go"},
			parseFile: func(_ *token.FileSet, _ string, _ []byte) (*ast.File, error) {
				return nil, errors.New("parse failed completely")
			},
		}

		result := st.parsePackageFiles(context.Background(), lp)
		assert.Empty(t, result)
		require.Len(t, lp.Errors, 1)
	})

	t.Run("parse error with non-nil file includes file", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		partialFile := &ast.File{Name: ast.NewIdent("foo")}
		st := &typeCheckState{
			fset:     fset,
			overlay:  map[string][]byte{"/src/partial.go": []byte("package foo\nfunc {")},
			readFile: os.ReadFile,
		}
		lp := &loaderPkg{
			Package: &packages.Package{},
			goFiles: []string{"/src/partial.go"},
			parseFile: func(_ *token.FileSet, _ string, _ []byte) (*ast.File, error) {
				return partialFile, errors.New("partial parse")
			},
		}

		result := st.parsePackageFiles(context.Background(), lp)
		assert.Len(t, result, 1)
		assert.Equal(t, partialFile, result[0])
		require.Len(t, lp.Errors, 1)
	})

	t.Run("cancelled context returns early", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		st := &typeCheckState{
			fset:    fset,
			overlay: nil,
			readFile: func(name string) ([]byte, error) {
				t.Fatal("readFile should not be called with cancelled context")
				return nil, nil
			},
		}
		lp := &loaderPkg{
			Package: &packages.Package{},
			goFiles: []string{"/src/a.go", "/src/b.go"},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		result := st.parsePackageFiles(ctx, lp)
		assert.Empty(t, result)
	})
}

func TestLoadFromExportData(t *testing.T) {
	t.Parallel()

	t.Run("openFile error returns false", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		st := &typeCheckState{
			fset:          fset,
			exportImports: make(map[string]*types.Package),
			openFile: func(name string) (io.ReadCloser, error) {
				return nil, errors.New("file not found")
			},
		}
		lp := &loaderPkg{
			Package:    &packages.Package{PkgPath: "test/pkg"},
			exportFile: "/cache/test/pkg.a",
		}

		result := st.loadFromExportData(context.Background(), lp)
		assert.False(t, result)
		assert.Nil(t, lp.Types)
	})

	t.Run("cancelled context returns false", func(t *testing.T) {
		t.Parallel()
		st := &typeCheckState{
			openFile: func(name string) (io.ReadCloser, error) {
				t.Fatal("openFile should not be called with cancelled context")
				return nil, nil
			},
		}
		lp := &loaderPkg{
			Package:    &packages.Package{PkgPath: "test/pkg"},
			exportFile: "/cache/test/pkg.a",
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		result := st.loadFromExportData(ctx, lp)
		assert.False(t, result)
	})

	t.Run("invalid export data returns false", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		st := &typeCheckState{
			fset:          fset,
			exportImports: make(map[string]*types.Package),
			openFile: func(name string) (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader("not valid export data")), nil
			},
		}
		lp := &loaderPkg{
			Package:    &packages.Package{PkgPath: "test/pkg"},
			exportFile: "/cache/test/pkg.a",
		}

		result := st.loadFromExportData(context.Background(), lp)
		assert.False(t, result)
	})
}

func TestTypeCheckPackage(t *testing.T) {
	t.Parallel()

	t.Run("initial package gets TypesInfo", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		src := "package foo\nvar X int = 1\n"
		file, err := parser.ParseFile(fset, "foo.go", src, parser.AllErrors)
		require.NoError(t, err)

		lp := &loaderPkg{
			Package: &packages.Package{
				PkgPath: "test/foo",
				Name:    "foo",
				Syntax:  []*ast.File{file},
				Imports: make(map[string]*packages.Package),
			},
			initial: true,
		}
		lp.Types = types.NewPackage("test/foo", "foo")

		st := &typeCheckState{
			fset:     fset,
			sizes:    types.SizesFor("gc", runtime.GOARCH),
			cpuLimit: make(chan struct{}, 1),
		}

		st.typeCheckPackage(context.Background(), lp)

		assert.NotNil(t, lp.TypesInfo)
		assert.NotEmpty(t, lp.TypesInfo.Defs)
		assert.Equal(t, st.sizes, lp.TypesSizes)
	})

	t.Run("dep package gets nil TypesInfo", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		src := "package bar\nvar Y string\n"
		file, err := parser.ParseFile(fset, "bar.go", src, parser.AllErrors)
		require.NoError(t, err)

		lp := &loaderPkg{
			Package: &packages.Package{
				PkgPath: "test/bar",
				Name:    "bar",
				Syntax:  []*ast.File{file},
				Imports: make(map[string]*packages.Package),
			},
			initial: false,
		}
		lp.Types = types.NewPackage("test/bar", "bar")

		st := &typeCheckState{
			fset:     fset,
			sizes:    types.SizesFor("gc", runtime.GOARCH),
			cpuLimit: make(chan struct{}, 1),
		}

		st.typeCheckPackage(context.Background(), lp)

		assert.Nil(t, lp.TypesInfo)
		assert.True(t, lp.Types.Complete())
	})

	t.Run("import resolution across packages", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()

		depFile, err := parser.ParseFile(fset, "dep.go", "package dep\nvar Value int = 42\n", parser.AllErrors)
		require.NoError(t, err)
		depTypes := types.NewPackage("test/dep", "dep")
		depSt := &typeCheckState{
			fset:     fset,
			sizes:    types.SizesFor("gc", runtime.GOARCH),
			cpuLimit: make(chan struct{}, 1),
		}
		depLP := &loaderPkg{
			Package: &packages.Package{
				PkgPath: "test/dep",
				Name:    "dep",
				Syntax:  []*ast.File{depFile},
				Imports: make(map[string]*packages.Package),
			},
			initial: true,
		}
		depLP.Types = depTypes
		depSt.typeCheckPackage(context.Background(), depLP)
		require.True(t, depTypes.Complete())

		mainFile, err := parser.ParseFile(fset, "main.go", "package main\nimport \"test/dep\"\nvar _ = dep.Value\n", parser.AllErrors)
		require.NoError(t, err)
		mainLP := &loaderPkg{
			Package: &packages.Package{
				PkgPath: "test/main",
				Name:    "main",
				Syntax:  []*ast.File{mainFile},
				Imports: map[string]*packages.Package{
					"test/dep": depLP.Package,
				},
			},
			initial: true,
		}
		mainLP.Types = types.NewPackage("test/main", "main")

		mainSt := &typeCheckState{
			fset:     fset,
			sizes:    types.SizesFor("gc", runtime.GOARCH),
			cpuLimit: make(chan struct{}, 1),
		}
		mainSt.typeCheckPackage(context.Background(), mainLP)

		assert.NotNil(t, mainLP.TypesInfo)
		assert.Empty(t, mainLP.Errors)
	})

	t.Run("cancelled context skips type-checking", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		lp := &loaderPkg{
			Package: &packages.Package{
				PkgPath: "test/foo",
				Name:    "foo",
				Imports: make(map[string]*packages.Package),
			},
			initial: true,
		}
		lp.Types = types.NewPackage("test/foo", "foo")

		st := &typeCheckState{
			fset:     fset,
			sizes:    types.SizesFor("gc", runtime.GOARCH),
			cpuLimit: make(chan struct{}, 1),
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		st.typeCheckPackage(ctx, lp)
		assert.Nil(t, lp.TypesInfo)
	})
}

func TestWriteOverlayJSON(t *testing.T) {
	t.Parallel()

	t.Run("nil overlay returns empty", func(t *testing.T) {
		t.Parallel()
		path, cleanup, err := writeOverlayJSON(nil)
		require.NoError(t, err)
		assert.Empty(t, path)
		cleanup()
	})

	t.Run("empty overlay returns empty", func(t *testing.T) {
		t.Parallel()
		path, cleanup, err := writeOverlayJSON(map[string][]byte{})
		require.NoError(t, err)
		assert.Empty(t, path)
		cleanup()
	})

	t.Run("populated overlay creates valid JSON", func(t *testing.T) {
		t.Parallel()
		overlay := map[string][]byte{
			"/src/a.go": []byte("package a"),
		}

		path, cleanup, err := writeOverlayJSON(overlay)
		require.NoError(t, err)
		defer cleanup()

		assert.NotEmpty(t, path)
		assert.True(t, strings.HasSuffix(path, "overlay.json"))

		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr)

		var result overlayJSON
		require.NoError(t, json.Unmarshal(data, &result))
		require.Contains(t, result.Replace, "/src/a.go")

		contentPath := result.Replace["/src/a.go"]
		content, readErr := os.ReadFile(contentPath)
		require.NoError(t, readErr)
		assert.Equal(t, "package a", string(content))
	})

	t.Run("cleanup removes temp directory", func(t *testing.T) {
		t.Parallel()
		overlay := map[string][]byte{"/src/b.go": []byte("package b")}

		path, cleanup, err := writeOverlayJSON(overlay)
		require.NoError(t, err)
		require.NotEmpty(t, path)

		dir := filepath.Dir(path)
		_, statErr := os.Stat(dir)
		require.NoError(t, statErr)

		cleanup()

		_, statErr = os.Stat(dir)
		assert.True(t, os.IsNotExist(statErr))
	})
}

func TestWriteOverlayToDisk(t *testing.T) {
	t.Parallel()

	t.Run("files written with correct content", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		sandbox, err := safedisk.NewSandbox(tmpDir, safedisk.ModeReadWrite)
		require.NoError(t, err)
		defer func() { _ = sandbox.Close() }()

		overlay := map[string][]byte{"/logical/a.go": []byte("package a")}
		newFiles := collectOverlayFiles(overlay)

		overlayPath, writeErr := writeOverlayToDisk(sandbox, newFiles, overlay)
		require.NoError(t, writeErr)
		require.NotEmpty(t, overlayPath)

		data, readErr := os.ReadFile(overlayPath)
		require.NoError(t, readErr)

		var result overlayJSON
		require.NoError(t, json.Unmarshal(data, &result))
		require.Contains(t, result.Replace, "/logical/a.go")

		replacePath := result.Replace["/logical/a.go"]
		assert.True(t, strings.HasPrefix(replacePath, sandbox.Root()))

		content, readErr := os.ReadFile(replacePath)
		require.NoError(t, readErr)
		assert.Equal(t, "package a", string(content))
	})

	t.Run("sandbox write error propagates", func(t *testing.T) {
		t.Parallel()
		mockSandbox := safedisk.NewMockSandbox("/mock", safedisk.ModeReadWrite)
		mockSandbox.WriteFileErr = errors.New("disk full")
		defer func() { _ = mockSandbox.Close() }()

		overlay := map[string][]byte{"/logical/c.go": []byte("package c")}
		newFiles := collectOverlayFiles(overlay)

		_, writeErr := writeOverlayToDisk(mockSandbox, newFiles, overlay)
		require.Error(t, writeErr)
		assert.Contains(t, writeErr.Error(), "disk full")
	})
}

func TestParseAndTypeCheck_Cancellation(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	noopParse := func(_ *token.FileSet, _ string, _ []byte) (*ast.File, error) {
		return nil, nil
	}

	listed := []goListPkg{
		{ImportPath: "test/a", Name: "a", DepOnly: false, GoFiles: []string{"a.go"}, Dir: "/src/a"},
	}
	pkgMap, _ := createPackages(listed, fset, noopParse, noopParse)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := parseAndTypeCheck(ctx, pkgMap, fset, types.SizesFor("gc", runtime.GOARCH), nil)
	require.NoError(t, err)

	assert.Nil(t, pkgMap["test/a"].Types)
}

func TestLoad_SimplePackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	t.Parallel()

	cfg := &packages.Config{
		Context: context.Background(),
	}

	pkgs, err := Load(cfg, "errors")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]
	assert.Equal(t, "errors", pkg.Name)
	assert.NotNil(t, pkg.Types)
	assert.True(t, pkg.Types.Complete())
	assert.NotNil(t, pkg.TypesInfo)
	assert.NotEmpty(t, pkg.Syntax)
	assert.Empty(t, pkg.Errors)
}
