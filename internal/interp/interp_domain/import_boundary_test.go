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

package interp_domain

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

func parseImportBoundaryFile(t *testing.T, source string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "boundary_test.go", source, 0)
	require.NoError(t, err)
	return file
}

func TestCheckFileImportsCanonicalisesImportPath(t *testing.T) {
	t.Parallel()
	service := NewService(WithDeniedImports("unsafe"))

	tests := []struct {
		name   string
		source string
	}{
		{"double quoted", "package main\nimport _ \"unsafe\"\n"},
		{"raw string", "package main\nimport _ " + "`unsafe`" + "\n"},
		{"escaped", "package main\nimport _ \"\\x75nsafe\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := service.checkFileImports(nil, parseImportBoundaryFile(t, tt.source))
			require.ErrorIs(t, err, errParse)
			require.ErrorContains(t, err, "not permitted")
		})
	}
}

func TestCheckFileImportsDenylistOverridesLocalPackage(t *testing.T) {
	t.Parallel()
	service := NewService(WithImportAllowlist("host/widgets"), WithDeniedImports("unsafe"))
	parsed := map[string]*parsedPackage{"unsafe": {importPath: "unsafe"}}

	err := service.checkFileImports(parsed, parseImportBoundaryFile(t, "package main\nimport _ \"unsafe\"\n"))
	require.ErrorIs(t, err, errParse)
	require.ErrorContains(t, err, "not permitted")
}

func TestCheckFileImportsAllowlist(t *testing.T) {
	t.Parallel()

	t.Run("permits allowlisted and local imports", func(t *testing.T) {
		t.Parallel()
		service := NewService(WithImportAllowlist("host/widgets"))
		parsed := map[string]*parsedPackage{"testmod/lib": {importPath: "testmod/lib"}}
		require.NoError(t, service.checkFileImports(parsed, parseImportBoundaryFile(t, "package main\nimport _ \"host/widgets\"\n")))
		require.NoError(t, service.checkFileImports(parsed, parseImportBoundaryFile(t, "package main\nimport _ \"testmod/lib\"\n")))
	})

	t.Run("denies non-allowlisted external import", func(t *testing.T) {
		t.Parallel()
		service := NewService(WithImportAllowlist("host/widgets"))
		err := service.checkFileImports(nil, parseImportBoundaryFile(t, "package main\nimport _ \"net/http\"\n"))
		require.ErrorContains(t, err, "not permitted")
	})

	t.Run("empty allowlist denies every external import", func(t *testing.T) {
		t.Parallel()
		service := NewService(WithImportAllowlist())
		err := service.checkFileImports(nil, parseImportBoundaryFile(t, "package main\nimport _ \"fmt\"\n"))
		require.ErrorContains(t, err, "not permitted")
	})
}

func TestImportBoundaryDeniedAcrossEntrypoints(t *testing.T) {
	t.Parallel()
	const deniedImport = "os"
	mixedCode := "import \"" + deniedImport + "\"\n_ = " + deniedImport + ".Getpid()"
	fileSource := "package main\nimport _ \"" + deniedImport + "\"\n"
	programSources := map[string]map[string]string{
		"": {"main.go": "package main\nimport _ \"" + deniedImport + "\"\n\nfunc main() {}\n"},
	}

	entrypoints := []struct {
		name string
		run  func(ctx context.Context, service *Service) error
	}{
		{"Eval", func(ctx context.Context, service *Service) error {
			_, err := service.Eval(ctx, mixedCode)
			return err
		}},
		{"Compile", func(ctx context.Context, service *Service) error {
			_, err := service.Compile(ctx, mixedCode)
			return err
		}},
		{"CompileFileSet", func(ctx context.Context, service *Service) error {
			_, err := service.CompileFileSet(ctx, map[string]string{"main.go": fileSource})
			return err
		}},
		{"CompileProgram", func(ctx context.Context, service *Service) error {
			_, err := service.CompileProgram(ctx, "testmod", programSources)
			return err
		}},
	}

	for _, entrypoint := range entrypoints {
		t.Run(entrypoint.name, func(t *testing.T) {
			t.Parallel()
			service := NewService(WithImportAllowlist("testmod/lib"))
			err := entrypoint.run(context.Background(), service)
			require.Error(t, err)
			require.ErrorContains(t, err, "not permitted")
		})
	}
}
