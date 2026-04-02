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
	"context"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func minimalStdlib() *inspector_dto.TypeData {
	return &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"fmt": {
				Name: "fmt",
				Path: "fmt",
				NamedTypes: map[string]*inspector_dto.Type{
					"Stringer": {
						Name:                 "Stringer",
						PackagePath:          "fmt",
						UnderlyingTypeString: "interface{String() string}",
						Methods: []*inspector_dto.Method{
							{
								Name:      "String",
								Signature: inspector_dto.FunctionSignature{Results: []string{"string"}},
							},
						},
					},
				},
				Funcs: map[string]*inspector_dto.Function{
					"Sprintf": {
						Name: "Sprintf",
						Signature: inspector_dto.FunctionSignature{
							Params:  []string{"string", "...any"},
							Results: []string{"string"},
						},
					},
				},
			},
			"time": {
				Name: "time",
				Path: "time",
				NamedTypes: map[string]*inspector_dto.Type{
					"Time": {
						Name:                 "Time",
						PackagePath:          "time",
						UnderlyingTypeString: "struct{...}",
					},
					"Duration": {
						Name:                 "Duration",
						PackagePath:          "time",
						UnderlyingTypeString: "int64",
					},
				},
			},
			"net/http": {
				Name: "http",
				Path: "net/http",
				NamedTypes: map[string]*inspector_dto.Type{
					"Request": {
						Name:                 "Request",
						PackagePath:          "net/http",
						UnderlyingTypeString: "struct{...}",
					},
				},
			},
		},
		FileToPackage: map[string]string{},
	}
}

func liteConfig() inspector_dto.Config {
	return inspector_dto.Config{
		ModuleName: "testproject",
		BaseDir:    "/project",
	}
}

func buildLite(t *testing.T, sources map[string][]byte) (*LiteBuilder, *inspector_dto.TypeData) {
	t.Helper()

	lb, err := NewLiteBuilder(minimalStdlib(), liteConfig())
	require.NoError(t, err)

	err = lb.Build(context.Background(), sources)
	require.NoError(t, err)

	td, err := lb.GetTypeData()
	require.NoError(t, err)

	return lb, td
}

func TestLiteBuilder_Constructor(t *testing.T) {
	t.Parallel()
	t.Run("should reject nil stdlib data", func(t *testing.T) {
		t.Parallel()
		_, err := NewLiteBuilder(nil, liteConfig())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("should create builder with valid stdlib", func(t *testing.T) {
		t.Parallel()
		lb, err := NewLiteBuilder(minimalStdlib(), liteConfig())
		require.NoError(t, err)
		assert.NotNil(t, lb)
		assert.False(t, lb.IsBuilt())
	})
}

func TestLiteBuilder_PreBuildState(t *testing.T) {
	t.Parallel()
	t.Run("GetTypeData should error before Build", func(t *testing.T) {
		t.Parallel()
		lb, err := NewLiteBuilder(minimalStdlib(), liteConfig())
		require.NoError(t, err)

		_, err = lb.GetTypeData()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not been built")
	})

	t.Run("GetQuerier should return nil before Build", func(t *testing.T) {
		t.Parallel()
		lb, err := NewLiteBuilder(minimalStdlib(), liteConfig())
		require.NoError(t, err)

		querier, ok := lb.GetQuerier()
		assert.False(t, ok)
		assert.Nil(t, querier)
	})
}

func TestLiteBuilder_SimpleStruct(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

type User struct {
	Name string
	Age  int
}
`),
	})

	userPackage := td.Packages["testproject"]
	require.NotNil(t, userPackage, "user package should be in merged TypeData")
	assert.Equal(t, "main", userPackage.Name)

	userType := userPackage.NamedTypes["User"]
	require.NotNil(t, userType)
	assert.Equal(t, "User", userType.Name)
	assert.Equal(t, "struct{...}", userType.UnderlyingTypeString)
	require.Len(t, userType.Fields, 2)

	fieldNames := make(map[string]string)
	for _, f := range userType.Fields {
		fieldNames[f.Name] = f.TypeString
	}
	assert.Equal(t, "string", fieldNames["Name"])
	assert.Equal(t, "int", fieldNames["Age"])
}

func TestLiteBuilder_PointerFields(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

type Config struct {
	Name    *string
	Timeout *int
}
`),
	})

	typ := td.Packages["testproject"].NamedTypes["Config"]
	require.NotNil(t, typ)

	fieldTypes := make(map[string]string)
	for _, f := range typ.Fields {
		fieldTypes[f.Name] = f.TypeString
	}
	assert.Equal(t, "*string", fieldTypes["Name"])
	assert.Equal(t, "*int", fieldTypes["Timeout"])
}

func TestLiteBuilder_SliceAndArrayFields(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

type Collection struct {
	Items  []string
	Matrix [3]int
}
`),
	})

	typ := td.Packages["testproject"].NamedTypes["Collection"]
	require.NotNil(t, typ)

	fieldTypes := make(map[string]string)
	for _, f := range typ.Fields {
		fieldTypes[f.Name] = f.TypeString
	}
	assert.Equal(t, "[]string", fieldTypes["Items"])
	assert.Equal(t, "[...]int", fieldTypes["Matrix"])
}

func TestLiteBuilder_MapFields(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

type Lookup struct {
	Data map[string]int
}
`),
	})

	typ := td.Packages["testproject"].NamedTypes["Lookup"]
	require.NotNil(t, typ)
	require.Len(t, typ.Fields, 1)
	assert.Equal(t, "map[string]int", typ.Fields[0].TypeString)
}

func TestLiteBuilder_ImportedTypes(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

import "time"

type Event struct {
	When time.Time
	Dur  time.Duration
}
`),
	})

	typ := td.Packages["testproject"].NamedTypes["Event"]
	require.NotNil(t, typ)

	fieldTypes := make(map[string]string)
	for _, f := range typ.Fields {
		fieldTypes[f.Name] = f.TypeString
	}
	assert.Equal(t, "time.Time", fieldTypes["When"])
	assert.Equal(t, "time.Duration", fieldTypes["Dur"])
}

func TestLiteBuilder_Methods(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

type User struct {
	Name string
}

func (u *User) GetName() string {
	return u.Name
}

func (u User) IsValid() bool {
	return u.Name != ""
}
`),
	})

	typ := td.Packages["testproject"].NamedTypes["User"]
	require.NotNil(t, typ)
	require.Len(t, typ.Methods, 2)

	methodMap := make(map[string]*inspector_dto.Method)
	for _, m := range typ.Methods {
		methodMap[m.Name] = m
	}

	require.Contains(t, methodMap, "GetName")
	assert.True(t, methodMap["GetName"].IsPointerReceiver)

	require.Contains(t, methodMap, "IsValid")
	assert.False(t, methodMap["IsValid"].IsPointerReceiver)
}

func TestLiteBuilder_TopLevelFunctions(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

func NewUser(name string) *User {
	return nil
}

func Add(a, b int) int {
	return a + b
}
`),
	})

	pkg := td.Packages["testproject"]
	require.NotNil(t, pkg)
	assert.Contains(t, pkg.Funcs, "NewUser")
	assert.Contains(t, pkg.Funcs, "Add")
}

func TestLiteBuilder_InterfaceType(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

type Doer interface {
	Do() error
	Undo() error
}
`),
	})

	typ := td.Packages["testproject"].NamedTypes["Doer"]
	require.NotNil(t, typ)
	assert.Contains(t, typ.UnderlyingTypeString, "interface")
	require.Len(t, typ.Methods, 2)

	methodNames := make(map[string]bool)
	for _, m := range typ.Methods {
		methodNames[m.Name] = true
	}
	assert.True(t, methodNames["Do"])
	assert.True(t, methodNames["Undo"])
}

func TestLiteBuilder_FuncType(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

type Handler func(string) error
`),
	})

	typ := td.Packages["testproject"].NamedTypes["Handler"]
	require.NotNil(t, typ)
	assert.Contains(t, typ.UnderlyingTypeString, "func")
}

func TestLiteBuilder_MultiplePackages(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

type App struct {
	Name string
}
`),
		"/project/models/user.go": []byte(`package models

type User struct {
	Name string
}
`),
	})

	mainPackage := td.Packages["testproject"]
	require.NotNil(t, mainPackage)
	assert.Contains(t, mainPackage.NamedTypes, "App")

	modelsPackage := td.Packages["testproject/models"]
	require.NotNil(t, modelsPackage)
	assert.Contains(t, modelsPackage.NamedTypes, "User")
}

func TestLiteBuilder_EmptySources(t *testing.T) {
	t.Parallel()
	lb, err := NewLiteBuilder(minimalStdlib(), liteConfig())
	require.NoError(t, err)

	err = lb.Build(context.Background(), map[string][]byte{})
	require.NoError(t, err)

	td, err := lb.GetTypeData()
	require.NoError(t, err)

	assert.Contains(t, td.Packages, "fmt", "stdlib should still be in merged data")
	assert.Contains(t, td.Packages, "time")
}

func TestLiteBuilder_IsBuiltStateTransition(t *testing.T) {
	t.Parallel()
	lb, err := NewLiteBuilder(minimalStdlib(), liteConfig())
	require.NoError(t, err)

	assert.False(t, lb.IsBuilt())

	err = lb.Build(context.Background(), map[string][]byte{
		"/project/main.go": []byte(`package main`),
	})
	require.NoError(t, err)
	assert.True(t, lb.IsBuilt())
}

func TestLiteBuilder_QuerierAfterBuild(t *testing.T) {
	t.Parallel()
	lb, _ := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

type User struct {
	Name string
}
`),
	})

	querier, ok := lb.GetQuerier()
	require.True(t, ok)
	require.NotNil(t, querier)

	pkgs := querier.GetAllPackages()
	assert.Contains(t, pkgs, "testproject")
	assert.Contains(t, pkgs, "fmt")
}

func TestLiteBuilder_GenericTypeRejected(t *testing.T) {
	t.Parallel()
	lb, err := NewLiteBuilder(minimalStdlib(), liteConfig())
	require.NoError(t, err)

	err = lb.Build(context.Background(), map[string][]byte{
		"/project/main.go": []byte(`package main

type Box[T any] struct {
	Value T
}

type User struct {
	Name string
}
`),
	})
	require.NoError(t, err, "build should not fail; generic types are skipped with a warning")

	td, err := lb.GetTypeData()
	require.NoError(t, err)

	pkg := td.Packages["testproject"]
	require.NotNil(t, pkg)
	assert.NotContains(t, pkg.NamedTypes, "Box", "generic type should be skipped")
	assert.Contains(t, pkg.NamedTypes, "User", "non-generic type should still be extracted")
}

func TestLiteBuilder_UnexportedTypesSkipped(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

type Public struct {
	Name string
}

type private struct {
	secret string
}
`),
	})

	pkg := td.Packages["testproject"]
	require.NotNil(t, pkg)
	assert.Contains(t, pkg.NamedTypes, "Public")
	assert.NotContains(t, pkg.NamedTypes, "private")
}

func TestLiteBuilder_UnexportedFieldsSkipped(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

type User struct {
	Name   string
	secret string
}
`),
	})

	typ := td.Packages["testproject"].NamedTypes["User"]
	require.NotNil(t, typ)

	for _, f := range typ.Fields {
		assert.NotEqual(t, "secret", f.Name, "unexported field should not be extracted")
	}
}

func TestLiteBuilder_EmbeddedFields(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

type Base struct {
	ID int
}

type Derived struct {
	Base
	Name string
}
`),
	})

	typ := td.Packages["testproject"].NamedTypes["Derived"]
	require.NotNil(t, typ)

	fieldNames := make(map[string]bool)
	for _, f := range typ.Fields {
		fieldNames[f.Name] = true
	}
	assert.True(t, fieldNames["Name"])
	assert.True(t, fieldNames["Base"], "embedded field should be extracted")
}

func TestLiteBuilder_SimpleNamedType(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

type UserID string
type Count int
`),
	})

	pkg := td.Packages["testproject"]
	require.NotNil(t, pkg)

	userID := pkg.NamedTypes["UserID"]
	require.NotNil(t, userID)
	assert.Equal(t, "string", userID.UnderlyingTypeString)

	count := pkg.NamedTypes["Count"]
	require.NotNil(t, count)
	assert.Equal(t, "int", count.UnderlyingTypeString)
}

func TestLiteBuilder_StdlibMergedCorrectly(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main`),
	})

	assert.Contains(t, td.Packages, "fmt")
	assert.Contains(t, td.Packages, "time")
	assert.Contains(t, td.Packages, "net/http")

	stringer := td.Packages["fmt"].NamedTypes["Stringer"]
	require.NotNil(t, stringer)
	assert.Contains(t, stringer.UnderlyingTypeString, "interface")
}

func TestLiteBuilder_FileToPackageIndex(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

type App struct{}
`),
	})

	assert.Equal(t, "testproject", td.FileToPackage["/project/main.go"])
}

func TestLiteBuilder_FileImportsExtracted(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

import (
	"fmt"
	"time"
)

type T struct{}
`),
	})

	pkg := td.Packages["testproject"]
	require.NotNil(t, pkg)

	imports, ok := pkg.FileImports["/project/main.go"]
	require.True(t, ok)
	assert.Equal(t, "fmt", imports["fmt"])
	assert.Equal(t, "time", imports["time"])
}

func TestLiteBuilder_AliasedImport(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

import h "net/http"

type Server struct {
	Req *h.Request
}
`),
	})

	pkg := td.Packages["testproject"]
	require.NotNil(t, pkg)

	imports := pkg.FileImports["/project/main.go"]
	assert.Equal(t, "net/http", imports["h"])
}

func TestLiteBuilder_BlankImportIgnored(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

import _ "net/http"

type T struct{}
`),
	})

	pkg := td.Packages["testproject"]
	require.NotNil(t, pkg)

	imports := pkg.FileImports["/project/main.go"]
	assert.NotContains(t, imports, "_")
}

func TestLiteBuilder_MethodsWithParamsAndReturns(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

type Calculator struct{}

func (c *Calculator) Add(a, b int) int {
	return a + b
}

func (c Calculator) Describe() (string, error) {
	return "", nil
}
`),
	})

	typ := td.Packages["testproject"].NamedTypes["Calculator"]
	require.NotNil(t, typ)

	methodMap := make(map[string]*inspector_dto.Method)
	for _, m := range typ.Methods {
		methodMap[m.Name] = m
	}

	require.Contains(t, methodMap, "Add")
	require.Contains(t, methodMap, "Describe")
}

func TestLiteBuilder_InvalidGoSourceSkipped(t *testing.T) {
	t.Parallel()
	lb, err := NewLiteBuilder(minimalStdlib(), liteConfig())
	require.NoError(t, err)

	err = lb.Build(context.Background(), map[string][]byte{
		"/project/main.go": []byte(`package main

type User struct {
	Name string
}
`),
		"/project/broken.go": []byte(`this is not valid go`),
	})

	require.NoError(t, err, "invalid files should be skipped, not cause build failure")

	td, err := lb.GetTypeData()
	require.NoError(t, err)

	pkg := td.Packages["testproject"]
	require.NotNil(t, pkg)
	assert.Contains(t, pkg.NamedTypes, "User")
}

func TestLiteBuilder_DefinitionLocationPopulated(t *testing.T) {
	t.Parallel()
	_, td := buildLite(t, map[string][]byte{
		"/project/main.go": []byte(`package main

type User struct {
	Name string
}
`),
	})

	typ := td.Packages["testproject"].NamedTypes["User"]
	require.NotNil(t, typ)
	assert.Equal(t, "/project/main.go", typ.DefinedInFilePath)
	assert.Greater(t, typ.DefinitionLine, 0)
	assert.Greater(t, typ.DefinitionColumn, 0)
}

func TestLiteBuildError(t *testing.T) {
	t.Parallel()
	t.Run("should format error with all fields", func(t *testing.T) {
		t.Parallel()
		err := newLiteBuildError(
			"generic type", "Box", "/src/box.go",
			fakePos(10),
			"generics are not supported",
		)
		message := err.Error()
		assert.Contains(t, message, "generic type")
		assert.Contains(t, message, "not supported")
		assert.Contains(t, message, "Box")
		assert.Contains(t, message, "/src/box.go")
		assert.Contains(t, message, "10")
		assert.Contains(t, message, "generics are not supported")
	})

	t.Run("should format error without optional fields", func(t *testing.T) {
		t.Parallel()
		err := &liteBuildError{Construct: "channel type"}
		message := err.Error()
		assert.Contains(t, message, "channel type")
		assert.Contains(t, message, "not supported")
		assert.NotContains(t, message, "in type")
	})
}

func TestTypeRegistry(t *testing.T) {
	t.Parallel()
	t.Run("LookupType should find registered types", func(t *testing.T) {
		t.Parallel()
		reg := newTypeRegistry(minimalStdlib())

		typ, ok := reg.LookupType("fmt", "Stringer")
		assert.True(t, ok)
		assert.Equal(t, "Stringer", typ.Name)
	})

	t.Run("LookupType should return false for missing package", func(t *testing.T) {
		t.Parallel()
		reg := newTypeRegistry(minimalStdlib())

		_, ok := reg.LookupType("missing", "Stringer")
		assert.False(t, ok)
	})

	t.Run("LookupType should return false for missing type", func(t *testing.T) {
		t.Parallel()
		reg := newTypeRegistry(minimalStdlib())

		_, ok := reg.LookupType("fmt", "Missing")
		assert.False(t, ok)
	})

	t.Run("LookupPackage should find registered packages", func(t *testing.T) {
		t.Parallel()
		reg := newTypeRegistry(minimalStdlib())

		pkg, ok := reg.LookupPackage("fmt")
		assert.True(t, ok)
		assert.Equal(t, "fmt", pkg.Name)
	})

	t.Run("LookupPackage should return false for missing package", func(t *testing.T) {
		t.Parallel()
		reg := newTypeRegistry(minimalStdlib())

		_, ok := reg.LookupPackage("missing")
		assert.False(t, ok)
	})

	t.Run("RegisterPackage should add new packages", func(t *testing.T) {
		t.Parallel()
		reg := newTypeRegistry(minimalStdlib())

		newPackage := &inspector_dto.Package{
			Name: "custom",
			Path: "my/custom",
			NamedTypes: map[string]*inspector_dto.Type{
				"MyType": {Name: "MyType"},
			},
		}
		reg.RegisterPackage(newPackage)

		pkg, ok := reg.LookupPackage("my/custom")
		assert.True(t, ok)
		assert.Equal(t, "custom", pkg.Name)

		typ, ok := reg.LookupType("my/custom", "MyType")
		assert.True(t, ok)
		assert.Equal(t, "MyType", typ.Name)
	})
}

func TestLiteBuilder_MapField(t *testing.T) {
	t.Parallel()
	t.Run("should resolve map field with key and value composite parts", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/src/app.go": []byte(`package app

type Config struct {
	Settings map[string]int
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject/src"]
		require.NotNil(t, pkg)

		configType := pkg.NamedTypes["Config"]
		require.NotNil(t, configType)
		require.Len(t, configType.Fields, 1)

		field := configType.Fields[0]
		assert.Equal(t, "Settings", field.Name)
		assert.Equal(t, "map[string]int", field.TypeString)
		assert.Equal(t, inspector_dto.CompositeTypeMap, field.CompositeType)
		require.Len(t, field.CompositeParts, 2)
		assert.Equal(t, "key", field.CompositeParts[0].Role)
		assert.Equal(t, "value", field.CompositeParts[1].Role)
	})
}

func TestLiteBuilder_MapWithImportedTypes(t *testing.T) {
	t.Parallel()
	t.Run("should resolve map with imported value type", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/src/app.go": []byte(`package app

import "time"

type Schedule struct {
	Times map[string]time.Time
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject/src"]
		require.NotNil(t, pkg)

		schedType := pkg.NamedTypes["Schedule"]
		require.NotNil(t, schedType)
		require.Len(t, schedType.Fields, 1)

		field := schedType.Fields[0]
		assert.Equal(t, "map[string]time.Time", field.TypeString)
		assert.Equal(t, inspector_dto.CompositeTypeMap, field.CompositeType)
	})
}

func TestLiteBuilder_FuncTypeField(t *testing.T) {
	t.Parallel()
	t.Run("should resolve func type field", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/src/app.go": []byte(`package app

type Handler struct {
	Callback func(string, int) error
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject/src"]
		require.NotNil(t, pkg)

		handlerType := pkg.NamedTypes["Handler"]
		require.NotNil(t, handlerType)
		require.Len(t, handlerType.Fields, 1)

		field := handlerType.Fields[0]
		assert.Equal(t, "Callback", field.Name)
		assert.Contains(t, field.TypeString, "func(")
	})
}

func TestLiteBuilder_FuncTypeFieldMultiReturn(t *testing.T) {
	t.Parallel()
	t.Run("should resolve func field with multiple returns", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/src/app.go": []byte(`package app

type Provider struct {
	Fetch func(string) (int, error)
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject/src"]
		require.NotNil(t, pkg)

		provType := pkg.NamedTypes["Provider"]
		require.NotNil(t, provType)
		require.Len(t, provType.Fields, 1)

		field := provType.Fields[0]
		assert.Equal(t, "Fetch", field.Name)
		assert.Contains(t, field.TypeString, "func(string) (int, error)")
	})
}

func TestLiteBuilder_StructFieldType(t *testing.T) {
	t.Parallel()
	t.Run("should resolve anonymous struct field", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/src/app.go": []byte(`package app

type Wrapper struct {
	Inner struct{}
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject/src"]
		require.NotNil(t, pkg)

		wType := pkg.NamedTypes["Wrapper"]
		require.NotNil(t, wType)
		require.Len(t, wType.Fields, 1)
		assert.Equal(t, "struct{}", wType.Fields[0].TypeString)
	})
}

func TestLiteBuilder_InterfaceFieldType(t *testing.T) {
	t.Parallel()
	t.Run("should resolve anonymous interface field", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/src/app.go": []byte(`package app

type Container struct {
	Any interface{}
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject/src"]
		require.NotNil(t, pkg)

		cType := pkg.NamedTypes["Container"]
		require.NotNil(t, cType)
		require.Len(t, cType.Fields, 1)
		assert.Equal(t, "interface{}", cType.Fields[0].TypeString)
	})
}

func TestLiteBuilder_VariadicFunction(t *testing.T) {
	t.Parallel()
	t.Run("should extract variadic parameter in function", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/src/app.go": []byte(`package app

func Sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject/src"]
		require.NotNil(t, pkg)

		inspectedFunction := pkg.Funcs["Sum"]
		require.NotNil(t, inspectedFunction)
		require.NotEmpty(t, inspectedFunction.Signature.Params)
		require.NotEmpty(t, inspectedFunction.Signature.Results)
	})
}

func TestLiteBuilder_FunctionNoReturn(t *testing.T) {
	t.Parallel()
	t.Run("should extract function with no return value", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/src/app.go": []byte(`package app

func DoNothing() {
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject/src"]
		require.NotNil(t, pkg)

		inspectedFunction := pkg.Funcs["DoNothing"]
		require.NotNil(t, inspectedFunction)
		assert.Empty(t, inspectedFunction.Signature.Params)
		assert.Empty(t, inspectedFunction.Signature.Results)
	})
}

func TestLiteBuilder_TypeAlias(t *testing.T) {
	t.Parallel()
	t.Run("should extract type alias with IsAlias set", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/src/app.go": []byte(`package app

type MyString = string

type Real struct {
	Name string
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject/src"]
		require.NotNil(t, pkg)

		aliasType, aliasFound := pkg.NamedTypes["MyString"]
		require.True(t, aliasFound)
		assert.True(t, aliasType.IsAlias)
		assert.Equal(t, "string", aliasType.UnderlyingTypeString)

		realType, realFound := pkg.NamedTypes["Real"]
		require.True(t, realFound)
		assert.False(t, realType.IsAlias)
	})
}

func TestLiteBuilder_CrossPackageFieldResolution(t *testing.T) {
	t.Parallel()
	t.Run("should resolve fields referencing other user packages", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/src/models/user.go": []byte(`package models

type User struct {
	Name string
}
`),
			"/project/src/app/handler.go": []byte(`package app

import "testproject/src/models"

type Handler struct {
	User models.User
}
`),
		}

		_, td := buildLite(t, sources)

		appPackage := td.Packages["testproject/src/app"]
		require.NotNil(t, appPackage)

		handlerType := appPackage.NamedTypes["Handler"]
		require.NotNil(t, handlerType)
		require.Len(t, handlerType.Fields, 1)

		field := handlerType.Fields[0]
		assert.Equal(t, "User", field.Name)
		assert.Equal(t, "models.User", field.TypeString)
		assert.Equal(t, "testproject/src/models", field.PackagePath)
	})
}

func TestLiteBuilder_PointerToImportedType(t *testing.T) {
	t.Parallel()
	t.Run("should resolve pointer to imported type", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/src/app.go": []byte(`package app

import "time"

type Event struct {
	When *time.Time
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject/src"]
		require.NotNil(t, pkg)

		eventType := pkg.NamedTypes["Event"]
		require.NotNil(t, eventType)
		require.Len(t, eventType.Fields, 1)

		field := eventType.Fields[0]
		assert.Equal(t, "*time.Time", field.TypeString)
		assert.Equal(t, inspector_dto.CompositeTypePointer, field.CompositeType)
	})
}

func TestLiteBuilder_SliceOfImportedType(t *testing.T) {
	t.Parallel()
	t.Run("should resolve slice of imported type", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/src/app.go": []byte(`package app

import "time"

type Timeline struct {
	Events []time.Time
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject/src"]
		require.NotNil(t, pkg)

		tlType := pkg.NamedTypes["Timeline"]
		require.NotNil(t, tlType)
		require.Len(t, tlType.Fields, 1)

		field := tlType.Fields[0]
		assert.Equal(t, "[]time.Time", field.TypeString)
		assert.Equal(t, inspector_dto.CompositeTypeSlice, field.CompositeType)
	})
}

func TestLiteBuilder_FuncWithImportedParam(t *testing.T) {
	t.Parallel()
	t.Run("should resolve function with imported parameter types", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/src/app.go": []byte(`package app

import "net/http"

func HandleRequest(r *http.Request) string {
	return ""
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject/src"]
		require.NotNil(t, pkg)

		inspectedFunction := pkg.Funcs["HandleRequest"]
		require.NotNil(t, inspectedFunction)
		require.Len(t, inspectedFunction.Signature.Params, 1)
		require.Len(t, inspectedFunction.Signature.Results, 1)
	})
}

func TestLiteBuilder_MethodWithMultipleParams(t *testing.T) {
	t.Parallel()
	t.Run("should extract method with multiple params and returns", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/src/app.go": []byte(`package app

type Service struct {
	Name string
}

func (s *Service) Process(input string, count int) (string, error) {
	return "", nil
}

func (s Service) Info() string {
	return s.Name
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject/src"]
		require.NotNil(t, pkg)

		svcType := pkg.NamedTypes["Service"]
		require.NotNil(t, svcType)
		require.Len(t, svcType.Methods, 2)

		var processMethod, infoMethod *inspector_dto.Method
		for _, m := range svcType.Methods {
			switch m.Name {
			case "Process":
				processMethod = m
			case "Info":
				infoMethod = m
			}
		}

		require.NotNil(t, processMethod)
		assert.True(t, processMethod.IsPointerReceiver)
		assert.Len(t, processMethod.Signature.Params, 2)
		assert.Len(t, processMethod.Signature.Results, 2)

		require.NotNil(t, infoMethod)
		assert.False(t, infoMethod.IsPointerReceiver)
		assert.Len(t, infoMethod.Signature.Results, 1)
	})
}

func TestLiteBuilder_EmbeddedFieldFromImport(t *testing.T) {
	t.Parallel()
	t.Run("should handle embedded imported type", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/src/models/base.go": []byte(`package models

type Base struct {
	ID int
}
`),
			"/project/src/app/user.go": []byte(`package app

import "testproject/src/models"

type User struct {
	models.Base
	Name string
}
`),
		}

		_, td := buildLite(t, sources)

		appPackage := td.Packages["testproject/src/app"]
		require.NotNil(t, appPackage)

		userType := appPackage.NamedTypes["User"]
		require.NotNil(t, userType)

		require.GreaterOrEqual(t, len(userType.Fields), 2)

		var embeddedField *inspector_dto.Field
		for _, f := range userType.Fields {
			if f.IsEmbedded {
				embeddedField = f
				break
			}
		}
		require.NotNil(t, embeddedField, "should have an embedded field")
	})
}

func TestLiteBuilder_SamePackageEmbedding(t *testing.T) {
	t.Parallel()
	t.Run("should handle embedded type from same package", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/main.go": []byte(`package main

type Base struct {
	ID int
}

type Derived struct {
	Base
	Extra string
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject"]
		require.NotNil(t, pkg)

		derived := pkg.NamedTypes["Derived"]
		require.NotNil(t, derived)

		var embeddedField *inspector_dto.Field
		for _, f := range derived.Fields {
			if f.IsEmbedded {
				embeddedField = f
				break
			}
		}
		require.NotNil(t, embeddedField)
		assert.Equal(t, "Base", embeddedField.Name)
		assert.True(t, embeddedField.IsEmbedded)
	})
}

func TestLiteBuilder_PointerEmbeddedField(t *testing.T) {
	t.Parallel()
	t.Run("should handle pointer-embedded type", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/main.go": []byte(`package main

type Base struct {
	ID int
}

type Extended struct {
	*Base
	Name string
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject"]
		require.NotNil(t, pkg)

		extType := pkg.NamedTypes["Extended"]
		require.NotNil(t, extType)

		var embeddedField *inspector_dto.Field
		for _, f := range extType.Fields {
			if f.IsEmbedded {
				embeddedField = f
				break
			}
		}
		require.NotNil(t, embeddedField)
		assert.Equal(t, "Base", embeddedField.Name)
		assert.True(t, embeddedField.IsEmbedded)
		assert.Contains(t, embeddedField.TypeString, "Base")
	})
}

func TestLiteBuilder_UnexportedEmbeddedField(t *testing.T) {
	t.Parallel()
	t.Run("should skip unexported embedded field", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/main.go": []byte(`package main

type base struct {
	id int
}

type Public struct {
	base
	Name string
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject"]
		require.NotNil(t, pkg)

		pubType := pkg.NamedTypes["Public"]
		require.NotNil(t, pubType)

		for _, f := range pubType.Fields {
			if f.IsEmbedded {
				assert.Fail(t, "unexported embedded field should be excluded")
			}
		}
	})
}

func TestLiteBuilder_MultipleFilesInPackage(t *testing.T) {
	t.Parallel()
	t.Run("should merge types from multiple files in same package", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/user.go": []byte(`package main

type User struct {
	Name string
}
`),
			"/project/order.go": []byte(`package main

type Order struct {
	Total int
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject"]
		require.NotNil(t, pkg)

		_, hasUser := pkg.NamedTypes["User"]
		_, hasOrder := pkg.NamedTypes["Order"]
		assert.True(t, hasUser)
		assert.True(t, hasOrder)
	})
}

func TestLiteBuilder_ReceiverOnStarExpr(t *testing.T) {
	t.Parallel()
	t.Run("should resolve star expr receiver type name", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/main.go": []byte(`package main

type Cache struct {
	Data map[string]string
}

func (c *Cache) Get(key string) string {
	return ""
}

func (c *Cache) Set(key, value string) {
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject"]
		require.NotNil(t, pkg)

		cacheType := pkg.NamedTypes["Cache"]
		require.NotNil(t, cacheType)
		assert.Len(t, cacheType.Methods, 2)

		for _, m := range cacheType.Methods {
			assert.True(t, m.IsPointerReceiver)
		}
	})
}

func TestLiteBuilder_InterfaceWithMethods(t *testing.T) {
	t.Parallel()
	t.Run("should extract interface methods", func(t *testing.T) {
		t.Parallel()
		sources := map[string][]byte{
			"/project/src/app.go": []byte(`package app

type Reader interface {
	Read(p []byte) (int, error)
	Close() error
}
`),
		}

		_, td := buildLite(t, sources)

		pkg := td.Packages["testproject/src"]
		require.NotNil(t, pkg)

		readerType := pkg.NamedTypes["Reader"]
		require.NotNil(t, readerType)
		require.Len(t, readerType.Methods, 2)

		methodNames := make(map[string]bool)
		for _, m := range readerType.Methods {
			methodNames[m.Name] = true
		}
		assert.True(t, methodNames["Read"])
		assert.True(t, methodNames["Close"])
	})
}

func fakePos(line int) token.Position {
	return token.Position{Filename: "test.go", Line: line, Column: 1}
}
