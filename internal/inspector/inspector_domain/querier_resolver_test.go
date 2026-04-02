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

package inspector_domain_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func setupResolverInspectorWithCode(t *testing.T, sources map[string]string) *inspector_domain.TypeQuerier {
	t.Helper()

	baseDir := t.TempDir()
	moduleName := "testproject"
	goModPath := filepath.Join(baseDir, "go.mod")
	goModContent := []byte("module " + moduleName + "\n\ngo 1.23\n")
	err := os.WriteFile(goModPath, goModContent, 0644)
	require.NoError(t, err, "Failed to write dummy go.mod")

	sourceContents := make(map[string][]byte, len(sources))
	for path, content := range sources {
		fullPath := filepath.Join(baseDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)

		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)

		sourceContents[fullPath] = []byte(content)
	}

	config := inspector_dto.Config{
		BaseDir:    baseDir,
		ModuleName: moduleName,
	}

	provider := inspector_adapters.NewInMemoryProvider(nil)
	manager := inspector_domain.NewTypeBuilder(config, inspector_domain.WithProvider(provider))

	err = manager.Build(context.Background(), sourceContents, map[string]string{})
	require.NoError(t, err, "Inspector manager failed to build")

	inspector, ok := manager.GetQuerier()
	require.True(t, ok, "Failed to get querier from manager")
	require.NotNil(t, inspector, "Inspector should not be nil")

	return inspector
}

func TestResolveToUnderlyingAST(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name         string
		sources      map[string]string
		startType    string
		filePath     string
		expectedType string
	}{

		{
			name:         "Basic: Primitive type should resolve to itself",
			sources:      map[string]string{"main.go": `package main`},
			startType:    "string",
			filePath:     "main.go",
			expectedType: "string",
		},
		{
			name:         "Basic: Unqualified struct should resolve to itself",
			sources:      map[string]string{"main.go": `package main; type User struct{}`},
			startType:    "User",
			filePath:     "main.go",
			expectedType: "main.User",
		},

		{
			name:         "Local Alias: Simple alias to primitive",
			sources:      map[string]string{"main.go": `package main; type UserID = string`},
			startType:    "UserID",
			filePath:     "main.go",
			expectedType: "string",
		},
		{
			name:         "Local Alias: Chained local aliases",
			sources:      map[string]string{"main.go": `package main; type ID = UserID; type UserID = int`},
			startType:    "ID",
			filePath:     "main.go",
			expectedType: "int",
		},
		{
			name: "Local Alias: Alias to an imported type",
			sources: map[string]string{
				"main.go":        `package main; import "testproject/models"; type LocalUser = models.User`,
				"models/user.go": `package models; type User struct{}`,
			}, startType: "LocalUser",
			filePath:     "main.go",
			expectedType: "models.User",
		},

		{
			name: "Deep Resolution: Multi-package chained aliases",
			sources: map[string]string{
				"main.go":         `package main; import "testproject/layer1"; type FinalAlias = layer1.L1Alias`,
				"layer1/types.go": `package layer1; import "testproject/layer2"; type L1Alias = layer2.L2Alias`,
				"layer2/types.go": `package layer2; import "testproject/models"; type L2Alias = models.User`,
				"models/user.go":  `package models; type User struct{}`,
			},
			startType:    "FinalAlias",
			filePath:     "main.go",
			expectedType: "models.User",
		},

		{
			name:         "Composites: Alias within a slice",
			sources:      map[string]string{"main.go": `package main; type UserID = string`},
			startType:    "[]UserID",
			filePath:     "main.go",
			expectedType: "[]string",
		},
		{
			name:         "Composites: Alias within a pointer",
			sources:      map[string]string{"main.go": `package main; type UserID = string`},
			startType:    "*UserID",
			filePath:     "main.go",
			expectedType: "*string",
		},
		{
			name: "Composites: Alias in map key and value",
			sources: map[string]string{
				"main.go":        `package main; import "testproject/models"; type UserID = string; type LocalUser = models.User`,
				"models/user.go": `package models; type User struct{}`,
			},
			startType:    "map[UserID]LocalUser",
			filePath:     "main.go",
			expectedType: "map[string]models.User",
		},
		{
			name: "Composites: Alias as generic type argument",
			sources: map[string]string{
				"main.go": `package main; type UserID = int; type Box[T any] struct{}`,
			},
			startType:    "Box[UserID]",
			filePath:     "main.go",
			expectedType: "main.Box[int]",
		},
		{
			name: "Composites: Alias as multiple generic type arguments",
			sources: map[string]string{
				"main.go":        `package main; import "testproject/models"; type Key = string; type Value = models.User`,
				"models/user.go": `package models; type User struct{}`,
			},
			startType:    "map[Key]Value",
			filePath:     "main.go",
			expectedType: "map[string]models.User",
		},

		{
			name:         "Edge Case: Should not resolve a type definition",
			sources:      map[string]string{"main.go": `package main; type UserID string`},
			startType:    "UserID",
			filePath:     "main.go",
			expectedType: "main.UserID",
		},

		{
			name: "Generic Alias: Generic type alias with concrete type argument",
			sources: map[string]string{
				"main.go": `package main
import "testproject/facade"
import "testproject/models"
type Response struct {
	Results []facade.SearchResult[models.Doc]
}`,
				"facade/types.go": `package facade
import "testproject/runtime"
type SearchResult[T any] = runtime.SearchResult[T]`,
				"runtime/types.go": `package runtime
type SearchResult[T any] struct {
	Item  T
	Score float64
}`,
				"models/doc.go": `package models
type Doc struct {
	Title string
	URL   string
}`,
			},
			startType:    "facade.SearchResult[models.Doc]",
			filePath:     "main.go",
			expectedType: "runtime.SearchResult[models.Doc]",
		},
		{
			name: "Generic Alias: Generic type alias with multiple type arguments",
			sources: map[string]string{
				"main.go": `package main
import "testproject/facade"
type Response struct {
	Data facade.Pair[string, int]
}`,
				"facade/types.go": `package facade
import "testproject/runtime"
type Pair[K any, V any] = runtime.Pair[K, V]`,
				"runtime/types.go": `package runtime
type Pair[K any, V any] struct {
	Key   K
	Value V
}`,
			},
			startType:    "facade.Pair[string, int]",
			filePath:     "main.go",
			expectedType: "runtime.Pair[string, int]",
		},
		{
			name: "Generic Alias: Nested generic alias resolution",
			sources: map[string]string{
				"main.go": `package main
import "testproject/facade"
import "testproject/models"
type Response struct {
	Items []facade.Container[models.Item]
}`,
				"facade/types.go": `package facade
import "testproject/layer1"
type Container[T any] = layer1.Wrapper[T]`,
				"layer1/types.go": `package layer1
import "testproject/runtime"
type Wrapper[T any] = runtime.Box[T]`,
				"runtime/types.go": `package runtime
type Box[T any] struct {
	Value T
}`,
				"models/item.go": `package models
type Item struct {
	ID string
}`,
			},
			startType:    "facade.Container[models.Item]",
			filePath:     "main.go",
			expectedType: "runtime.Box[models.Item]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			inspector := setupResolverInspectorWithCode(t, tc.sources)
			baseDir := inspector.Config.BaseDir
			fullFilePath := filepath.Join(baseDir, tc.filePath)

			startAST := goastutil.TypeStringToAST(tc.startType)
			resolvedAST := inspector.ResolveToUnderlyingAST(startAST, fullFilePath)

			assert.Equal(t, tc.expectedType, goastutil.ASTToTypeString(resolvedAST))
		})
	}
}

func TestResolveExprToNamedType(t *testing.T) {
	t.Parallel()
	sources := map[string]string{

		"main.go": `
            package main
            import u "testproject/uuid_a"
            import . "testproject/models"
            var _ u.UUID
            var _ User
        `,

		"other.go": `
            package main
            import u "testproject/uuid_b"
            var _ u.UUID
        `,
		"models/user.go": `package models; type User struct{}`,
		"uuid_a/uuid.go": `package uuid_a; type UUID struct{}`,
		"uuid_b/uuid.go": `package uuid_b; type UUID struct{}`,
	}

	inspector := setupResolverInspectorWithCode(t, sources)
	baseDir := inspector.Config.BaseDir
	mainPackagePath := "testproject"

	t.Run("should resolve unqualified type in same package", func(t *testing.T) {
		t.Parallel()

		sourcesWithMainType := map[string]string{"main.go": `package main; type MainType struct{}`}
		inspectorWithMainType := setupResolverInspectorWithCode(t, sourcesWithMainType)
		mainFilePath := filepath.Join(inspectorWithMainType.Config.BaseDir, "main.go")

		expression := goastutil.TypeStringToAST("MainType")
		namedType, _ := inspectorWithMainType.ResolveExprToNamedType(expression, mainPackagePath, mainFilePath)

		require.NotNil(t, namedType)
		assert.Equal(t, "MainType", namedType.Name)
	})

	t.Run("should resolve unqualified type from dot import", func(t *testing.T) {
		t.Parallel()
		mainFilePath := filepath.Join(baseDir, "main.go")
		expression := goastutil.TypeStringToAST("User")
		namedType, packageName := inspector.ResolveExprToNamedType(expression, mainPackagePath, mainFilePath)

		require.NotNil(t, namedType)
		assert.Equal(t, "User", namedType.Name)
		assert.Equal(t, "models", packageName, "Package name should be from the dot-imported package")
	})

	t.Run("should fail to resolve dot-imported type from wrong file context", func(t *testing.T) {
		t.Parallel()
		otherFilePath := filepath.Join(baseDir, "other.go")
		expression := goastutil.TypeStringToAST("User")
		namedType, _ := inspector.ResolveExprToNamedType(expression, mainPackagePath, otherFilePath)

		assert.Nil(t, namedType, "Should not find 'User' without the dot import context")
	})

	t.Run("should use file-scoped context to resolve conflicting aliases", func(t *testing.T) {
		t.Parallel()

		mainFilePath := filepath.Join(baseDir, "main.go")
		expression := goastutil.TypeStringToAST("u.UUID")
		namedTypeA, _ := inspector.ResolveExprToNamedType(expression, mainPackagePath, mainFilePath)

		require.NotNil(t, namedTypeA)
		assert.Equal(t, "UUID", namedTypeA.Name)

		assert.Contains(t, namedTypeA.DefinedInFilePath, "uuid_a/uuid.go")

		otherFilePath := filepath.Join(baseDir, "other.go")
		namedTypeB, _ := inspector.ResolveExprToNamedType(expression, mainPackagePath, otherFilePath)
		require.NotNil(t, namedTypeB)
		assert.Equal(t, "UUID", namedTypeB.Name)
		assert.Contains(t, namedTypeB.DefinedInFilePath, "uuid_b/uuid.go")
	})

	t.Run("should resolve through a local alias before finding the type DTO", func(t *testing.T) {
		t.Parallel()
		sourcesWithAlias := map[string]string{
			"main.go":        `package main; import "testproject/models"; type LocalUser = models.User`,
			"models/user.go": `package models; type User struct{}`,
		}
		inspector := setupInspectorWithCode(t, sourcesWithAlias)
		mainFilePath := filepath.Join(inspector.Config.BaseDir, "main.go")

		startExpr := goastutil.TypeStringToAST("LocalUser")

		resolvedExpr := inspector.ResolveToUnderlyingAST(startExpr, mainFilePath)

		namedType, _ := inspector.ResolveExprToNamedType(resolvedExpr, mainPackagePath, mainFilePath)

		require.NotNil(t, namedType)
		assert.Equal(t, "User", namedType.Name)
		assert.Contains(t, namedType.DefinedInFilePath, "models/user.go")
	})
}
