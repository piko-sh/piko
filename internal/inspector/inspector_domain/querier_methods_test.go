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
	"fmt"
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

func setupMethodsInspector(t *testing.T, sourceFiles map[string]string) (*inspector_domain.TypeQuerier, string) {
	t.Helper()
	ctx := context.Background()

	tempDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module testmodule\n\ngo 1.18\n"), 0644)
	require.NoError(t, err)

	sourceContents := make(map[string][]byte, len(sourceFiles))
	for path, content := range sourceFiles {
		fullPath := filepath.Join(tempDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
		sourceContents[fullPath] = []byte(content)
	}

	config := inspector_dto.Config{
		BaseDir:    tempDir,
		ModuleName: "testmodule",
	}
	provider := inspector_adapters.NewInMemoryProvider(nil)
	manager := inspector_domain.NewTypeBuilder(config, inspector_domain.WithProvider(provider))

	err = manager.Build(ctx, sourceContents, map[string]string{})
	require.NoError(t, err, "Inspector build should not fail")

	inspector, ok := manager.GetQuerier()
	require.True(t, ok, "Failed to get a valid querier from the manager")

	return inspector, tempDir
}

func sig(params []string, results []string) *inspector_dto.FunctionSignature {
	return &inspector_dto.FunctionSignature{Params: params, Results: results}
}

func TestFindMethodSignature(t *testing.T) {
	t.Parallel()

	basicTypesSource := `
package main
type T struct{}
func (t T) ValueMethod() int { return 0 }
func (t *T) PointerMethod() string { return "" }`

	embeddedTypesSource := `
package main
type E2 struct{}
func (e E2) MethodE2() {}
type E1 struct{ E2 }
func (e E1) MethodE1() {}
type S struct{ E1 }`

	ambiguityTypesSource := `
package main
type E1 struct{}
func (e E1) AmbiguousMethod() {}
type E2 struct{}
func (e E2) AmbiguousMethod() {}
type S struct { E1; E2 }`

	shadowingTypesSource := `
package main
type E struct{}
func (e E) ShadowedMethod() int { return 1 }
type S struct { E }
func (s S) ShadowedMethod() string { return "s" }`

	diamondTypesSource := `
package main
type A struct{}
func (a A) DiamondMethod() {}
type B struct{ A }
type C struct{ A }
type S struct { B; C }`

	genericTypesSource := `
package main
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
	~float32 | ~float64 |
	~string
}
type Box[T any] struct{}
func (b Box[T]) Get() T { var zero T; return zero }
func (b Box[T]) Set(val T) {}
type StringBox struct{ Box[string] }
type IntBox struct{ Box[int] }
type Sorter[T Ordered] struct{}
func (s Sorter[T]) Sort() {}
type IntSorter struct{ Sorter[int] }
`

	aliasTypesSource := `
package main
type S struct{}
func (s S) OriginalMethod() {}
func (s *S) PointerMethod() {}
type MyS = S
type PtrS = *S
type C = S
type B = C
type A = B`

	interfaceTypesSource := `
package main
import "io"
type Drawer interface { Draw() }
type Stringer[T any] interface { String() T }
type ReadWriter interface {
	io.Reader
	io.Writer
}
var _ io.Reader`

	unexportedSource := `
package main
type T struct{}
func (t T) unexportedMethod() {}`

	testCases := []struct {
		sourceFiles         map[string]string
		expectSignature     *inspector_dto.FunctionSignature
		name                string
		importerPackagePath string
		importerFileName    string
		baseTypeString      string
		methodName          string
	}{

		{
			name:                "Basic/Value Receiver on Value Type",
			sourceFiles:         map[string]string{"main.go": basicTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "T",
			methodName:          "ValueMethod",
			expectSignature:     sig(nil, []string{"int"}),
		},
		{
			name:                "Basic/Pointer Receiver on Pointer Type",
			sourceFiles:         map[string]string{"main.go": basicTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "*T",
			methodName:          "PointerMethod",
			expectSignature:     sig(nil, []string{"string"}),
		},
		{
			name:                "Basic/Value Receiver on Pointer Type",
			sourceFiles:         map[string]string{"main.go": basicTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "*T",
			methodName:          "ValueMethod",
			expectSignature:     sig(nil, []string{"int"}),
		},
		{
			name:                "Basic/Pointer Receiver on Value Type (Local)",
			sourceFiles:         map[string]string{"main.go": basicTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "T",
			methodName:          "PointerMethod",
			expectSignature:     nil,
		},
		{
			name: "Basic/Pointer Receiver on Addressable Value (External)",
			sourceFiles: map[string]string{
				"models/user.go": "package models\n type User struct{}\n func (u *User) PointerMethod() {}",
				"main.go":        "package main\n import \"testmodule/models\"\n var _ models.User",
			},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "models.User",
			methodName:          "PointerMethod",
			expectSignature:     sig(nil, nil),
		},

		{
			name:                "Promotion/Simple Promotion from Embedded Value",
			sourceFiles:         map[string]string{"main.go": embeddedTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "S",
			methodName:          "MethodE1",
			expectSignature:     sig(nil, nil),
		},
		{
			name:                "Promotion/Deeply Nested Promotion",
			sourceFiles:         map[string]string{"main.go": embeddedTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "S",
			methodName:          "MethodE2",
			expectSignature:     sig(nil, nil),
		},
		{
			name: "Promotion/Pointer-Receiver Method from Embedded Value",
			sourceFiles: map[string]string{"main.go": `
package main
type E struct{}
func (e *E) PtrMethod() {}
type S struct{ E }`},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "S",
			methodName:          "PtrMethod",
			expectSignature:     sig(nil, nil),
		},

		{
			name:                "Ambiguity/Ambiguous Selector Fails",
			sourceFiles:         map[string]string{"main.go": ambiguityTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "S",
			methodName:          "AmbiguousMethod",
			expectSignature:     nil,
		},
		{
			name:                "Ambiguity/Method Shadowing (Top Level)",
			sourceFiles:         map[string]string{"main.go": shadowingTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "S",
			methodName:          "ShadowedMethod",
			expectSignature:     sig(nil, []string{"string"}),
		},
		{
			name:                "Ambiguity/Valid Diamond Embedding",
			sourceFiles:         map[string]string{"main.go": diamondTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "S",
			methodName:          "DiamondMethod",
			expectSignature:     sig(nil, nil),
		},

		{
			name:                "Generics/Promotion from Instantiated Generic Type",
			sourceFiles:         map[string]string{"main.go": genericTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "StringBox",
			methodName:          "Get",
			expectSignature:     sig(nil, []string{"string"}),
		},
		{
			name:                "Generics/Correct Signature from Promoted Generic Method (Parameter)",
			sourceFiles:         map[string]string{"main.go": genericTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "IntBox",
			methodName:          "Set",
			expectSignature:     sig([]string{"int"}, nil),
		},
		{
			name:                "Generics/Method on Generic Type Itself",
			sourceFiles:         map[string]string{"main.go": genericTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "Box[float64]",
			methodName:          "Get",
			expectSignature:     sig(nil, []string{"float64"}),
		},
		{
			name:                "Generics/Promotion from Generic with Constraints",
			sourceFiles:         map[string]string{"main.go": genericTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "IntSorter",
			methodName:          "Sort",
			expectSignature:     sig(nil, nil),
		},

		{
			name:                "Aliases/Method on a Simple Type Alias",
			sourceFiles:         map[string]string{"main.go": aliasTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "MyS",
			methodName:          "OriginalMethod",
			expectSignature:     sig(nil, nil),
		},
		{
			name:                "Aliases/Method on a Chained Type Alias",
			sourceFiles:         map[string]string{"main.go": aliasTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "A",
			methodName:          "OriginalMethod",
			expectSignature:     sig(nil, nil),
		},
		{
			name:                "Aliases/Method on Pointer to Type Alias",
			sourceFiles:         map[string]string{"main.go": aliasTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "PtrS",
			methodName:          "PointerMethod",
			expectSignature:     sig(nil, nil),
		},
		{
			name: "Aliases/Method on an Alias to an External Type",
			sourceFiles: map[string]string{
				"models/user.go": "package models\n type User struct{}\n func (u User) GetID() int { return 1 }",
				"main.go":        "package main\n import \"testmodule/models\"\n type MyUser = models.User\n var _ models.User",
			},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "MyUser",
			methodName:          "GetID",
			expectSignature:     sig(nil, []string{"int"}),
		},

		{
			name:                "Interfaces/Method on a Simple Interface",
			sourceFiles:         map[string]string{"main.go": interfaceTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "Drawer",
			methodName:          "Draw",
			expectSignature:     sig(nil, nil),
		},
		{
			name:                "Interfaces/Method on an Embedded Interface",
			sourceFiles:         map[string]string{"main.go": interfaceTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "ReadWriter",
			methodName:          "Read",
			expectSignature:     sig([]string{"[]byte"}, []string{"int", "error"}),
		},
		{
			name:                "Interfaces/Method on a Generic Interface",
			sourceFiles:         map[string]string{"main.go": interfaceTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "Stringer[int]",
			methodName:          "String",
			expectSignature:     sig(nil, []string{"int"}),
		},
		{
			name:                "Interfaces/Method Not Present on Interface",
			sourceFiles:         map[string]string{"main.go": interfaceTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "Drawer",
			methodName:          "Erase",
			expectSignature:     nil,
		},

		{
			name: "Rare/Promotion from Embedded Pointer",
			sourceFiles: map[string]string{"main.go": `
package main
type E struct{}
func (e E) ValueMethod() {}
type S struct{ *E }`},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "S",
			methodName:          "ValueMethod",
			expectSignature:     sig(nil, nil),
		},
		{
			name: "Rare/Shadowing via Deeper Embedding",
			sourceFiles: map[string]string{"main.go": `
package main
type E2 struct{}
func (e E2) Method() int { return 2 }
type E1 struct{ E2 }
func (e E1) Method() string { return "1" }
type S struct{ E1 }`},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "S",
			methodName:          "Method",
			expectSignature:     sig(nil, []string{"string"}),
		},
		{
			name: "Rare/Method on an Alias to an Interface",
			sourceFiles: map[string]string{"main.go": `
package main
import "io"
type MyReadCloser = io.ReadCloser
var _ io.ReadCloser`},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "MyReadCloser",
			methodName:          "Close",
			expectSignature:     sig(nil, []string{"error"}),
		},
		{
			name: "Rare/Method on Unnamed Struct with Embedded Type",
			sourceFiles: map[string]string{"main.go": `
package main
type T struct{}
func (t T) Method() {}`},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "struct{ T }",
			methodName:          "Method",
			expectSignature:     sig(nil, nil),
		},
		{
			name: "Rare/Method on Dot-Imported Type",
			sourceFiles: map[string]string{
				"models/user.go": "package models\n type User struct{}\n func (u User) GetName() string { return \"\" }",
				"main.go":        "package main\n import . \"testmodule/models\"\n var _ User",
			},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "User",
			methodName:          "GetName",
			expectSignature:     sig(nil, []string{"string"}),
		},
		{
			name: "Rare/Promotion through Alias of Embedded Struct",
			sourceFiles: map[string]string{"main.go": `
package main
type E struct{}
func (e E) Method() {}
type AliasE = E
type S struct { AliasE }`},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "S",
			methodName:          "Method",
			expectSignature:     sig(nil, nil),
		},
		{
			name: "Rare/Method on a Recursive Type",
			sourceFiles: map[string]string{"main.go": `
package main
type Node struct{ Next *Node }
func (n *Node) IsTail() bool { return n.Next == nil }`},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "*Node",
			methodName:          "IsTail",
			expectSignature:     sig(nil, []string{"bool"}),
		},
		{
			name: "Rare/Negative - No Promotion from Embedded Type Parameter",
			sourceFiles: map[string]string{"main.go": `
package main
type MyInt int
func (i MyInt) Method() {}
type Wrapper[T any] struct { 
	Value T // Use a named field instead of embedding
}`},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "Wrapper[MyInt]",
			methodName:          "Method",
			expectSignature:     nil,
		},
		{
			name: "Rare/Negative - No Method on Alias to Slice",
			sourceFiles: map[string]string{"main.go": `
package main
type IntSlice = []int
`},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "IntSlice",
			methodName:          "Method",
			expectSignature:     nil,
		},

		{
			name:                "Negative/Method Does Not Exist",
			sourceFiles:         map[string]string{"main.go": basicTypesSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "T",
			methodName:          "DoesNotExist",
			expectSignature:     nil,
		},
		{
			name:                "Negative/Unexported Method",
			sourceFiles:         map[string]string{"main.go": unexportedSource},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "T",
			methodName:          "unexportedMethod",
			expectSignature:     nil,
		},
		{
			name: "Context/File-Scoped Import Alias",
			sourceFiles: map[string]string{
				"models/user.go": "package models\n type User struct{}\n func (u User) GetName() string { return \"\" }",
				"main/file_a.go": "package main",
				"main/file_b.go": "package main\n import m \"testmodule/models\"\n var _ m.User",
			},
			importerPackagePath: "testmodule/main",
			importerFileName:    "main/file_b.go",
			baseTypeString:      "m.User",
			methodName:          "GetName",
			expectSignature:     sig(nil, []string{"string"}),
		},
		{
			name:                "Negative/Base Type is a Built-in",
			sourceFiles:         map[string]string{"main.go": "package main"},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "string",
			methodName:          "SomeMethod",
			expectSignature:     nil,
		},
		{
			name:                "Negative/Base Type is any",
			sourceFiles:         map[string]string{"main.go": "package main"},
			importerPackagePath: "testmodule",
			importerFileName:    "main.go",
			baseTypeString:      "any",
			methodName:          "SomeMethod",
			expectSignature:     nil,
		},
	}

	t.Run("FindMethodReturnType", func(t *testing.T) {
		t.Parallel()
		t.Run("Basic Return Type", func(t *testing.T) {
			t.Parallel()
			inspector, tempDir := setupMethodsInspector(t, map[string]string{"main.go": basicTypesSource})
			baseTypeAST := goastutil.TypeStringToAST("T")
			importerFilePath := filepath.Join(tempDir, "main.go")
			retType := inspector.FindMethodReturnType(baseTypeAST, "ValueMethod", "testmodule", importerFilePath)
			require.NotNil(t, retType)
			assert.Equal(t, "int", goastutil.ASTToTypeString(retType))
		})

		t.Run("Generic Return Type", func(t *testing.T) {
			t.Parallel()
			inspector, tempDir := setupMethodsInspector(t, map[string]string{"main.go": genericTypesSource})
			baseTypeAST := goastutil.TypeStringToAST("StringBox")
			importerFilePath := filepath.Join(tempDir, "main.go")
			retType := inspector.FindMethodReturnType(baseTypeAST, "Get", "testmodule", importerFilePath)
			require.NotNil(t, retType)
			assert.Equal(t, "string", goastutil.ASTToTypeString(retType))
		})

		t.Run("Non-existent method", func(t *testing.T) {
			t.Parallel()
			inspector, tempDir := setupMethodsInspector(t, map[string]string{"main.go": basicTypesSource})
			baseTypeAST := goastutil.TypeStringToAST("T")
			importerFilePath := filepath.Join(tempDir, "main.go")
			retTypeNil := inspector.FindMethodReturnType(baseTypeAST, "DoesNotExist", "testmodule", importerFilePath)
			assert.Nil(t, retTypeNil)
		})
	})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			inspector, tempDir := setupMethodsInspector(t, tc.sourceFiles)

			baseTypeAST := goastutil.TypeStringToAST(tc.baseTypeString)
			importerFilePath := filepath.Join(tempDir, tc.importerFileName)

			actualSignature := inspector.FindMethodSignature(
				baseTypeAST,
				tc.methodName,
				tc.importerPackagePath,
				importerFilePath,
			)

			if tc.expectSignature == nil {
				assert.Nil(t, actualSignature, "Expected a nil signature but got one")
			} else {
				require.NotNil(t, actualSignature, fmt.Sprintf("Expected signature '%v' but got nil", tc.expectSignature.ToSignatureString()))

				assert.Equal(t, tc.expectSignature.ToSignatureString(), actualSignature.ToSignatureString())
			}
		})
	}
}
