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

package inspector_domain

import (
	goast "go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestSwitchToDefiningPackageContext(t *testing.T) {
	t.Parallel()

	runtimePackage := &inspector_dto.Package{
		Path: "example.com/runtime",
		Name: "runtime",
		NamedTypes: map[string]*inspector_dto.Type{
			"SearchResult": {
				TypeString:        "runtime.SearchResult[T]",
				PackagePath:       "example.com/runtime",
				DefinedInFilePath: "/src/runtime/types.go",
				Fields: []*inspector_dto.Field{
					{Name: "Item", TypeString: "T"},
					{Name: "Score", TypeString: "float64"},
				},
			},
		},
		FileImports: map[string]map[string]string{},
	}

	importerPackage := &inspector_dto.Package{
		Path: "example.com/app",
		Name: "main",
		NamedTypes: map[string]*inspector_dto.Type{
			"Response": {
				TypeString:        "main.Response",
				DefinedInFilePath: "/src/app/main.go",
			},
		},
		FileImports: map[string]map[string]string{
			"/src/app/main.go": {
				"runtime": "example.com/runtime",
			},
		},
	}

	typeData := &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"example.com/runtime": runtimePackage,
			"example.com/app":     importerPackage,
		},
		FileToPackage: map[string]string{
			"/src/app/main.go":      "example.com/app",
			"/src/runtime/types.go": "example.com/runtime",
		},
	}

	querier := &TypeQuerier{typeData: typeData}

	t.Run("SelectorExpr resolves to defining package", func(t *testing.T) {
		t.Parallel()

		baseType := goastutil.TypeStringToAST("runtime.SearchResult")

		_, resolvedPackagePath, resolvedFilePath := querier.switchToDefiningPackageContext(
			baseType, "example.com/app", "/src/app/main.go",
		)

		assert.Equal(t, "example.com/runtime", resolvedPackagePath)
		assert.Equal(t, "/src/runtime/types.go", resolvedFilePath)
	})

	t.Run("IndexExpr generic type resolves to defining package", func(t *testing.T) {
		t.Parallel()

		baseType := &goast.IndexExpr{
			X:     goastutil.TypeStringToAST("runtime.SearchResult"),
			Index: goastutil.TypeStringToAST("models.Doc"),
		}

		_, resolvedPackagePath, resolvedFilePath := querier.switchToDefiningPackageContext(
			baseType, "example.com/app", "/src/app/main.go",
		)

		assert.Equal(t, "example.com/runtime", resolvedPackagePath)
		assert.Equal(t, "/src/runtime/types.go", resolvedFilePath)
	})

	t.Run("unknown package alias keeps original context", func(t *testing.T) {
		t.Parallel()

		baseType := goastutil.TypeStringToAST("unknown.SomeType")

		_, resolvedPackagePath, resolvedFilePath := querier.switchToDefiningPackageContext(
			baseType, "example.com/app", "/src/app/main.go",
		)

		assert.Equal(t, "example.com/app", resolvedPackagePath)
		assert.Equal(t, "/src/app/main.go", resolvedFilePath)
	})

	t.Run("unqualified ident keeps original context", func(t *testing.T) {
		t.Parallel()

		baseType := goast.NewIdent("LocalType")

		_, resolvedPackagePath, resolvedFilePath := querier.switchToDefiningPackageContext(
			baseType, "example.com/app", "/src/app/main.go",
		)

		assert.Equal(t, "example.com/app", resolvedPackagePath)
		assert.Equal(t, "/src/app/main.go", resolvedFilePath)
	})
}
