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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

var extractAndEncode = inspector_domain.ExtractAndEncodeForTest

func setupEncoderTest(t *testing.T, sources map[string]string) *inspector_dto.TypeData {
	t.Helper()

	tempDir := t.TempDir()
	moduleName := "testmodule"
	err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module "+moduleName+"\n\ngo 1.22\n"), 0644)
	require.NoError(t, err)

	overlay := make(map[string][]byte)
	patterns := make(map[string]bool)

	for path, content := range sources {

		fullPath := filepath.Join(tempDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)
		overlay[fullPath] = []byte(content)

		pkgDir := filepath.Dir(path)
		if pkgDir == "." {
			patterns[moduleName] = true
		} else {
			patterns[moduleName+"/"+pkgDir] = true
		}
	}

	loadPatterns := make([]string, 0, len(patterns))
	for p := range patterns {
		loadPatterns = append(loadPatterns, p)
	}

	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedDeps | packages.NeedTypesInfo |
			packages.NeedSyntax | packages.NeedImports | packages.NeedModule,
		Dir:     tempDir,
		Overlay: overlay,
		Env:     append(os.Environ(), "GOWORK=off"),
		Tests:   false,
	}

	loadedPackages, err := packages.Load(config, loadPatterns...)
	require.NoError(t, err, "go/packages.Load should not fail for valid test source")
	require.False(t, packages.PrintErrors(loadedPackages) > 0, "There should be no type-checking errors in the test source")

	typeData, err := extractAndEncode(loadedPackages, moduleName)
	require.NoError(t, err, "extractAndEncode should not fail")
	require.NotNil(t, typeData)

	return typeData
}

func TestEncoder(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		sources    map[string]string
		assertions func(t *testing.T, typeData *inspector_dto.TypeData)
		name       string
	}{

		{
			name: "A/01 - Basic Exported Fields",
			sources: map[string]string{
				"main.go": `package main; type User struct { Name string; Age int }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				pkg := td.Packages["testmodule"]
				require.NotNil(t, pkg)
				userType := pkg.NamedTypes["User"]
				require.NotNil(t, userType)
				require.Len(t, userType.Fields, 2)
				assert.Equal(t, "Name", userType.Fields[0].Name)
				assert.Equal(t, "string", userType.Fields[0].TypeString)
				assert.Equal(t, "Age", userType.Fields[1].Name)
				assert.Equal(t, "int", userType.Fields[1].TypeString)
			},
		},
		{
			name: "A/02 - Mix of Exported and Unexported Fields",
			sources: map[string]string{
				"main.go": `package main; type User struct { Name string; age int }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				userType := td.Packages["testmodule"].NamedTypes["User"]
				require.Len(t, userType.Fields, 1, "Should only encode exported fields")
				assert.Equal(t, "Name", userType.Fields[0].Name)
			},
		},
		{
			name: "A/03 - Struct Tag Capture",
			sources: map[string]string{
				"main.go": "package main; type User struct { Name string `json:\"name\" validate:\"required\"` }",
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				field := td.Packages["testmodule"].NamedTypes["User"].Fields[0]
				assert.Equal(t, `json:"name" validate:"required"`, field.RawTag)
			},
		},
		{
			name: "A/04 - Simple Embedding",
			sources: map[string]string{
				"main.go": `package main; type Base struct { ID int }; type User struct { Base }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				field := td.Packages["testmodule"].NamedTypes["User"].Fields[0]
				assert.Equal(t, "Base", field.Name)
				assert.True(t, field.IsEmbedded)
				assert.Equal(t, "main.Base", field.TypeString)
			},
		},
		{
			name: "A/05 - Pointer Embedding",
			sources: map[string]string{
				"main.go": `package main; type Base struct { ID int }; type User struct { *Base }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				field := td.Packages["testmodule"].NamedTypes["User"].Fields[0]
				assert.True(t, field.IsEmbedded)
				assert.Equal(t, "*main.Base", field.TypeString)
			},
		},

		{
			name: "B/06 - Value vs. Pointer Receivers",
			sources: map[string]string{
				"main.go": `package main; type T struct{}; func (t T) ValueMethod() {}; func (t *T) PointerMethod() {}`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				methods := td.Packages["testmodule"].NamedTypes["T"].Methods
				require.Len(t, methods, 2)
				methodMap := make(map[string]*inspector_dto.Method)
				for _, m := range methods {
					methodMap[m.Name] = m
				}
				assert.False(t, methodMap["ValueMethod"].IsPointerReceiver)
				assert.True(t, methodMap["PointerMethod"].IsPointerReceiver)
			},
		},
		{
			name: "B/07 - Method Promotion from Embedded Type",
			sources: map[string]string{
				"models/base.go": `package models; type Base struct{}; func (b Base) GetID() int { return 0 }`,
				"main.go":        `package main; import "testmodule/models"; type User struct { models.Base }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				method := td.Packages["testmodule"].NamedTypes["User"].Methods[0]
				require.NotNil(t, method)
				assert.Equal(t, "GetID", method.Name)
				assert.Equal(t, "testmodule/models", method.DeclaringPackagePath)
				assert.Equal(t, "Base", method.DeclaringTypeName)
			},
		},
		{
			name: "B/08 - Method Shadowing",
			sources: map[string]string{
				"main.go": `package main; type Base struct{}; func (b Base) Info() int {return 0}; type User struct { Base }; func (u User) Info() string {return ""}`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				methods := td.Packages["testmodule"].NamedTypes["User"].Methods
				require.Len(t, methods, 1)
				assert.Equal(t, "Info", methods[0].Name)
				assert.Equal(t, "string", methods[0].TypeString)
			},
		},

		{
			name: "C/09 - Basic Interface",
			sources: map[string]string{
				"main.go": `package main; type Logger interface { Log(message string) error }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				method := td.Packages["testmodule"].NamedTypes["Logger"].Methods[0]
				assert.Equal(t, "Log", method.Name)
				assert.Equal(t, []string{"string"}, method.Signature.Params)
				assert.Equal(t, []string{"error"}, method.Signature.Results)
			},
		},
		{
			name: "C/10 - Interface Embedding",
			sources: map[string]string{
				"main.go": `package main; import "io"; type ReadCloser interface { io.Reader; io.Closer }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				methods := td.Packages["testmodule"].NamedTypes["ReadCloser"].Methods
				require.Len(t, methods, 2)
				methodNames := []string{methods[0].Name, methods[1].Name}
				assert.Contains(t, methodNames, "Read")
				assert.Contains(t, methodNames, "Close")
			},
		},

		{
			name: "D/11 - Generic Type Definition",
			sources: map[string]string{
				"main.go": `package main; type Box[T any] struct { Value T }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				boxType := td.Packages["testmodule"].NamedTypes["Box"]
				require.NotNil(t, boxType)
				assert.Equal(t, []string{"T"}, boxType.TypeParams)
				field := boxType.Fields[0]
				assert.Equal(t, "Value", field.Name)
				assert.Equal(t, "T", field.TypeString)
			},
		},
		{
			name: "D/12 - Field with Instantiated Generic Type",
			sources: map[string]string{
				"main.go": `package main; type Box[T any] struct{}; type Container struct { IntBox Box[int] }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				field := td.Packages["testmodule"].NamedTypes["Container"].Fields[0]
				assert.Equal(t, "main.Box[int]", field.TypeString)
			},
		},
		{
			name: "D/13 - Method on a Generic Type",
			sources: map[string]string{
				"main.go": `package main; type Box[T any] struct{}; func (b Box[T]) Get() T { var zero T; return zero }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				method := td.Packages["testmodule"].NamedTypes["Box"].Methods[0]
				assert.Equal(t, "Get", method.Name)
				require.Len(t, method.Signature.Results, 1)
				assert.Equal(t, "T", method.Signature.Results[0])
			},
		},
		{
			name: "D/14 - Promoted Method from an Instantiated Embedded Generic",
			sources: map[string]string{
				"main.go": `package main; type Box[T any] struct{}; func (b Box[T]) Get() T { var zero T; return zero }; type StringBox struct { Box[string] }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				method := td.Packages["testmodule"].NamedTypes["StringBox"].Methods[0]
				assert.Equal(t, "Get", method.Name)
				require.Len(t, method.Signature.Results, 1)
				assert.Equal(t, "string", method.Signature.Results[0], "Return type must be substituted")
			},
		},
		{
			name: "D/15 - Package-Level Generic Function",
			sources: map[string]string{
				"main.go": `package main; func Map[T, U any](s []T, f func(T) U) []U { return nil }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				inspectedFunction := td.Packages["testmodule"].Funcs["Map"]
				require.NotNil(t, inspectedFunction)
				assert.Equal(t, []string{"[]T", "func(T) U"}, inspectedFunction.Signature.Params)
				assert.Equal(t, []string{"[]U"}, inspectedFunction.Signature.Results)
			},
		},

		{
			name: "E/16 - Type Alias vs. Type Definition",
			sources: map[string]string{
				"main.go": `package main; type UserID = string; type ProductID string`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				pkg := td.Packages["testmodule"]
				require.NotNil(t, pkg)

				aliasType := pkg.NamedTypes["UserID"]
				require.NotNil(t, aliasType)

				defType := pkg.NamedTypes["ProductID"]
				require.NotNil(t, defType)

				assert.True(t, aliasType.IsAlias, "UserID should be identified as an alias")

				assert.Equal(t, "string", aliasType.TypeString)

				assert.Equal(t, "string", aliasType.UnderlyingTypeString)

				assert.False(t, defType.IsAlias, "ProductID should be identified as a type definition")
				assert.Equal(t, "main.ProductID", defType.TypeString, "A type definition's TypeString is its own name")
				assert.Equal(t, "string", defType.UnderlyingTypeString, "The underlying type of ProductID is string")
			},
		},
		{
			name: "E/17 - Field Using a Same-Package Alias",
			sources: map[string]string{
				"main.go": `package main; type UserID = string; type User struct { ID UserID }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				field := td.Packages["testmodule"].NamedTypes["User"].Fields[0]
				assert.Equal(t, "main.UserID", field.TypeString, "Should preserve same-package alias")
			},
		},
		{
			name: "E/18 - Field Using a Cross-Package Alias",
			sources: map[string]string{
				"ids/ids.go": `package ids; type UserID = string`,
				"main.go":    `package main; import "testmodule/ids"; type User struct { ID ids.UserID }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				field := td.Packages["testmodule"].NamedTypes["User"].Fields[0]
				assert.Equal(t, "ids.UserID", field.TypeString, "Should preserve cross-package alias")
			},
		},

		{
			name: "F/19 - Aliased Import Qualification",
			sources: map[string]string{
				"uuid/uuid.go": `package uuid; type UUID struct{}`,
				"main.go":      `package main; import u "testmodule/uuid"; type Request struct { ID u.UUID }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				field := td.Packages["testmodule"].NamedTypes["Request"].Fields[0]
				assert.Equal(t, "u.UUID", field.TypeString)
			},
		},
		{
			name: "F/20 - Dot Import Qualification",
			sources: map[string]string{
				"uuid/uuid.go": `package uuid; type UUID struct{}`,
				"main.go":      `package main; import . "testmodule/uuid"; type Request struct { ID UUID }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				field := td.Packages["testmodule"].NamedTypes["Request"].Fields[0]
				assert.Equal(t, "uuid.UUID", field.TypeString, "Should re-qualify dot-imported type")
			},
		},
		{
			name: "F/21 - Two Files with Conflicting Aliases",
			sources: map[string]string{
				"a/a.go":    `package a; type Value struct{}`,
				"b/b.go":    `package b; type Value struct{}`,
				"main/a.go": `package main; import u "testmodule/a"; type ReqA struct { V u.Value }`,
				"main/b.go": `package main; import u "testmodule/b"; type ReqB struct { V u.Value }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				pkg := td.Packages["testmodule/main"]
				require.NotNil(t, pkg)
				require.Len(t, pkg.FileImports, 2)

				var foundPathA, foundPathB string
				for path := range pkg.FileImports {
					if strings.HasSuffix(path, "main/a.go") {
						foundPathA = path
					}
					if strings.HasSuffix(path, "main/b.go") {
						foundPathB = path
					}
				}
				require.NotEmpty(t, foundPathA, "Should find main/a.go path")
				require.NotEmpty(t, foundPathB, "Should find main/b.go path")
				assert.Equal(t, "testmodule/a", pkg.FileImports[foundPathA]["u"])
				assert.Equal(t, "testmodule/b", pkg.FileImports[foundPathB]["u"])

				fieldA := pkg.NamedTypes["ReqA"].Fields[0]
				fieldB := pkg.NamedTypes["ReqB"].Fields[0]
				assert.Equal(t, "u.Value", fieldA.TypeString)
				assert.Equal(t, "testmodule/a", fieldA.PackagePath)
				assert.Equal(t, "u.Value", fieldB.TypeString)
				assert.Equal(t, "testmodule/b", fieldB.PackagePath)
			},
		},

		{
			name: "G/22 - Variadic Function Signature",
			sources: map[string]string{
				"main.go": `package main; func Log(message string, arguments ...any) {}`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				inspectedFunction := td.Packages["testmodule"].Funcs["Log"]
				assert.Equal(t, []string{"string", "...any"}, inspectedFunction.Signature.Params)
			},
		},
		{
			name: "G/23 - Function with Multiple Named Returns",
			sources: map[string]string{
				"main.go": `package main; type User struct{}; func Fetch() (user *User, err error) { return nil, nil }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				inspectedFunction := td.Packages["testmodule"].Funcs["Fetch"]
				assert.Equal(t, []string{"*main.User", "error"}, inspectedFunction.Signature.Results)
			},
		},
		{
			name: "H/24 - Complex Nested Composite Type",
			sources: map[string]string{
				"models/models.go": `package models; type Event struct{}`,
				"main.go":          `package main; import "testmodule/models"; type Data struct { Events map[string][]*chan *models.Event }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				field := td.Packages["testmodule"].NamedTypes["Data"].Fields[0]
				assert.Equal(t, "map[string][]*chan *models.Event", field.TypeString)
			},
		},
		{
			name: "H/25 - Fixed-Size Array",
			sources: map[string]string{
				"main.go": `package main; type Data struct { MAC [6]byte }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				field := td.Packages["testmodule"].NamedTypes["Data"].Fields[0]
				assert.Equal(t, "[6]byte", field.TypeString)
			},
		},
		{
			name: "H/26 - Stringability via fmt.Stringer",
			sources: map[string]string{
				"main.go": `package main; type T struct{}; func (t T) String() string { return "" }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				typ := td.Packages["testmodule"].NamedTypes["T"]
				assert.Equal(t, inspector_dto.StringableViaStringer, typ.Stringability)
			},
		},
		{
			name: "H/27 - Stringability via encoding.TextMarshaler",
			sources: map[string]string{
				"main.go": `package main; type T struct{}; func (t T) MarshalText() ([]byte, error) { return nil, nil }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				typ := td.Packages["testmodule"].NamedTypes["T"]
				assert.Equal(t, inspector_dto.StringableViaTextMarshaler, typ.Stringability)
			},
		},
		{
			name: "I/28 - Empty Package",
			sources: map[string]string{
				"main.go": `package main`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				pkg := td.Packages["testmodule"]
				assert.Empty(t, pkg.NamedTypes)
				assert.Empty(t, pkg.Funcs)
			},
		},
		{
			name: "I/29 - Mutually Recursive Types",
			sources: map[string]string{
				"main.go": `package main; type A struct { B *B }; type B struct { A *A }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				pkg := td.Packages["testmodule"]
				typeA := pkg.NamedTypes["A"]
				typeB := pkg.NamedTypes["B"]
				assert.Equal(t, "*main.B", typeA.Fields[0].TypeString)
				assert.Equal(t, "*main.A", typeB.Fields[0].TypeString)
			},
		},
		{
			name: "I/30 - Unexported Embedded Type with Exported Methods",
			sources: map[string]string{
				"main.go": `package main; type private struct{}; func (p private) PublicMethod() {}; type Public struct{ private }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				typ := td.Packages["testmodule"].NamedTypes["Public"]
				assert.Empty(t, typ.Fields, "Unexported embedded field should not be encoded")
				require.Len(t, typ.Methods, 1)
				assert.Equal(t, "PublicMethod", typ.Methods[0].Name)
			},
		},

		{
			name: "J/31 - Embedding an Instantiated Generic Interface",
			sources: map[string]string{
				"main.go": `package main; type Producer[T any] interface { Produce() T }; type StringFactory struct { Producer[string] }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				method := td.Packages["testmodule"].NamedTypes["StringFactory"].Methods[0]
				assert.Equal(t, "Produce", method.Name)
				assert.Equal(t, "string", method.TypeString)
			},
		},
		{
			name: "J/32 - Diamond Embedding with Shadowing at Mid-level",
			sources: map[string]string{
				"main.go": `package main; type D struct{}; func (D) Method() int { return 0 }; type B struct{ D }; type C struct{ D }; func (C) Method() string { return "" }; type A struct{ B; C }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				method := td.Packages["testmodule"].NamedTypes["A"].Methods[0]
				assert.Equal(t, "Method", method.Name)
				assert.Equal(t, "string", method.TypeString, "Should pick the closer, shadowing method from C")
			},
		},
		{
			name: "J/33 - Embedding the built-in error type",
			sources: map[string]string{
				"main.go": `package main; type MyError struct { error; Code int }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				typ := td.Packages["testmodule"].NamedTypes["MyError"]
				assert.Len(t, typ.Fields, 1)
				require.Len(t, typ.Methods, 1)
				assert.Equal(t, "Error", typ.Methods[0].Name)
			},
		},

		{
			name: "K/34 - Function Signature with Unnamed Parameters",
			sources: map[string]string{
				"main.go": `package main; func F(string, int) (bool, error) { return false, nil }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				inspectedFunction := td.Packages["testmodule"].Funcs["F"]
				assert.Equal(t, []string{"string", "int"}, inspectedFunction.Signature.Params)
				assert.Equal(t, []string{"bool", "error"}, inspectedFunction.Signature.Results)
			},
		},
		{
			name: "K/35 - Type Definition on a Function Type",
			sources: map[string]string{
				"main.go": `package main; import "net/http"; type HandlerFunc func(http.ResponseWriter, *http.Request)`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				typ := td.Packages["testmodule"].NamedTypes["HandlerFunc"]
				assert.False(t, typ.IsAlias)
				assert.Equal(t, "func(http.ResponseWriter, *http.Request)", typ.UnderlyingTypeString)
			},
		},
		{
			name: "K/36 - Field of type unsafe.Pointer",
			sources: map[string]string{
				"main.go": `package main; import "unsafe"; type Buffer struct { Ptr unsafe.Pointer }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				field := td.Packages["testmodule"].NamedTypes["Buffer"].Fields[0]
				assert.Equal(t, "unsafe.Pointer", field.TypeString)
			},
		},

		{
			name: "L/37 - Package Name Differs from Directory Name",
			sources: map[string]string{
				"utils/helpers.go": `package helpers; type Util struct{}`,
				"main.go":          `package main; import "testmodule/utils"; var _ helpers.Util`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {

				pkg, ok := td.Packages["testmodule/utils"]
				require.True(t, ok)

				assert.Equal(t, "helpers", pkg.Name)
			},
		},
		{
			name: "L/38 - An Alias to an Alias to an Instantiated Generic",
			sources: map[string]string{
				"main.go": `package main; type Box[T any] struct{}; type StringBox = Box[string]; type MyBox = StringBox; type Container struct { TheBox MyBox }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				field := td.Packages["testmodule"].NamedTypes["Container"].Fields[0]
				assert.Equal(t, "main.MyBox", field.TypeString)
			},
		},

		{
			name: "M/39 - Generic Type where Parameter is Used as a Constraint",
			sources: map[string]string{
				"main.go": `package main; type Graph[Node comparable, Edge any] struct{}`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				typ := td.Packages["testmodule"].NamedTypes["Graph"]
				assert.Equal(t, []string{"Node", "Edge"}, typ.TypeParams)
				assert.Equal(t, "main.Graph[Node, Edge]", typ.TypeString)
			},
		},
		{
			name: "M/40 - Method Returning an Instantiated Generic with Receiver's Type Parameter",
			sources: map[string]string{
				"main.go": `package main; type Box[T any] struct{}; type Factory[T any] struct{}; func (f Factory[T]) NewBox() Box[T] { return Box[T]{} }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				method := td.Packages["testmodule"].NamedTypes["Factory"].Methods[0]
				assert.Equal(t, "NewBox", method.Name)
				assert.Equal(t, "main.Box[T]", method.TypeString)
			},
		},

		{
			name: "N/41 - Generic with Union Type Constraint",
			sources: map[string]string{
				"main.go": `package main; type Number interface { ~int | ~float64 }; type Value[T Number] struct{ V T }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				typ := td.Packages["testmodule"].NamedTypes["Value"]
				require.NotNil(t, typ)
				assert.Equal(t, "main.Value[T]", typ.TypeString)

				assert.Equal(t, "struct{V T}", typ.UnderlyingTypeString)
			},
		},
		{
			name: "N/42 - Field is a Generic Instantiated with a Type from Another Package",
			sources: map[string]string{
				"models/models.go": `package models; type Box[T any] struct { Value T }`,
				"main.go":          `package main; import "testmodule/models"; import "time"; type Container struct { TimeBox models.Box[time.Time] }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				field := td.Packages["testmodule"].NamedTypes["Container"].Fields[0]

				assert.Equal(t, "models.Box[time.Time]", field.TypeString)
			},
		},
		{
			name: "N/43 - Generic Type with Type Parameter Field",
			sources: map[string]string{
				"main.go": `package main; type MyInt int; func (i MyInt) IsZero() bool { return i == 0 }; type Wrapper[T any] struct { Value T }; var w Wrapper[MyInt]`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {

				typ := td.Packages["testmodule"].NamedTypes["Wrapper"]
				require.NotNil(t, typ)
				require.Len(t, typ.Fields, 1)
				assert.Equal(t, "Value", typ.Fields[0].Name)

				assert.Equal(t, "T", typ.Fields[0].TypeString)
			},
		},
		{

			name: "N/43b - Method on Generic Type using a Type Parameter in its Signature",
			sources: map[string]string{
				"main.go": `package main; type Processor[T any] struct{}; func (p Processor[T]) Process(val T) error { return nil }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				method := td.Packages["testmodule"].NamedTypes["Processor"].Methods[0]
				require.NotNil(t, method)
				assert.Equal(t, "Process", method.Name)
				require.Len(t, method.Signature.Params, 1)
				assert.Equal(t, "T", method.Signature.Params[0])
			},
		},
		{
			name: "N/44 - Generic Interface Embedding Another Generic Interface",
			sources: map[string]string{
				"main.go": `package main; type Getter[T any] interface { Get() T }; type AdvancedGetter[T any] interface { Getter[T]; Set(T) }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				typ := td.Packages["testmodule"].NamedTypes["AdvancedGetter"]
				require.NotNil(t, typ)
				require.Len(t, typ.Methods, 2)
				methodMap := make(map[string]*inspector_dto.Method)
				for _, m := range typ.Methods {
					methodMap[m.Name] = m
				}
				assert.Contains(t, methodMap, "Get")
				assert.Contains(t, methodMap, "Set")
				assert.Equal(t, "T", methodMap["Get"].TypeString)
			},
		},

		{
			name: "O/45 - Field is a Function Type with Complex Signature",
			sources: map[string]string{
				"main.go": `package main; import "net/http"; import "context"; type Server struct { Handler func(ctx context.Context, request *http.Request) (*http.Response, error) }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				field := td.Packages["testmodule"].NamedTypes["Server"].Fields[0]

				assert.Equal(t, "func(ctx context.Context, request *http.Request) (*http.Response, error)", field.TypeString)
			},
		},
		{
			name: "O/46 - Function Returning a Function",
			sources: map[string]string{
				"main.go": `package main; func NewMiddleware() func(int) (string, error) { return nil }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				inspectedFunction := td.Packages["testmodule"].Funcs["NewMiddleware"]
				require.NotNil(t, inspectedFunction)
				require.Len(t, inspectedFunction.Signature.Results, 1)
				assert.Equal(t, "func(int) (string, error)", inspectedFunction.Signature.Results[0])
			},
		},
		{
			name: "O/47 - Type Definition on a Function Type with Methods",
			sources: map[string]string{
				"main.go": `package main; type Handler func(); func (h Handler) Serve() {}`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				typ := td.Packages["testmodule"].NamedTypes["Handler"]
				require.NotNil(t, typ)
				assert.False(t, typ.IsAlias)
				assert.Equal(t, "func()", typ.UnderlyingTypeString)
				require.Len(t, typ.Methods, 1)
				assert.Equal(t, "Serve", typ.Methods[0].Name)
			},
		},

		{
			name: "P/48 - Field is an Anonymous Struct",
			sources: map[string]string{
				"main.go": `package main; type Response struct { Meta struct { Status int; Message string } }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				field := td.Packages["testmodule"].NamedTypes["Response"].Fields[0]

				assert.Equal(t, "struct{Status int; Message string}", field.TypeString)
			},
		},
		{
			name: "P/49 - Field is an Anonymous Interface",
			sources: map[string]string{
				"main.go": `package main; type Runner struct { Task interface { Run() error } }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				field := td.Packages["testmodule"].NamedTypes["Runner"].Fields[0]
				assert.Equal(t, "interface{Run() error}", field.TypeString)
			},
		},
		{
			name: "P/50 - Embedding a Named Struct with Promoted Fields",
			sources: map[string]string{
				"main.go": `package main; type Base struct { Name string }; type User struct { Base; Age int }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {

				typ := td.Packages["testmodule"].NamedTypes["User"]
				require.NotNil(t, typ)

				require.Len(t, typ.Fields, 2)

				assert.Equal(t, "Base", typ.Fields[0].Name)

				assert.Equal(t, "Age", typ.Fields[1].Name)
			},
		},

		{
			name: "Q/51 - Blank Identifier Import",
			sources: map[string]string{
				"driver/driver.go": `package driver`,
				"main.go":          `package main; import _ "testmodule/driver"`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				pkg := td.Packages["testmodule"]
				require.NotNil(t, pkg)

				var mainGoPath string
				for path := range pkg.FileImports {
					if strings.HasSuffix(path, "main.go") {
						mainGoPath = path
						break
					}
				}
				require.NotEmpty(t, mainGoPath, "Should find main.go in FileImports")
				assert.Equal(t, "testmodule/driver", pkg.FileImports[mainGoPath]["_"])
			},
		},
		{
			name: "Q/52 - Package Referring to its Own Types (Self-Qualification)",
			sources: map[string]string{
				"models/user.go": `package models; type User struct { Manager *User }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				field := td.Packages["testmodule/models"].NamedTypes["User"].Fields[0]

				assert.Equal(t, "*models.User", field.TypeString)
			},
		},
		{
			name: "Q/53 - Package Spanning Multiple Files",
			sources: map[string]string{
				"models/user.go":    `package models; type User struct { Profile Profile }`,
				"models/profile.go": `package models; type Profile struct { Email string }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				pkg := td.Packages["testmodule/models"]
				require.NotNil(t, pkg)
				userType := pkg.NamedTypes["User"]
				profileType := pkg.NamedTypes["Profile"]
				require.NotNil(t, userType)
				require.NotNil(t, profileType)

				field := userType.Fields[0]
				assert.Equal(t, "models.Profile", field.TypeString)
			},
		},
		{
			name: "Q/54 - Type from a Vendored Package",
			sources: map[string]string{
				"vendor/example.com/lib/lib.go": `package lib; type Thing struct{}`,
				"main.go":                       `package main; import "example.com/lib"; type Container struct { Item lib.Thing }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {

				libPackage, ok := td.Packages["example.com/lib"]
				require.True(t, ok, "Expected canonical path for vendored package")
				require.NotNil(t, libPackage)
				mainPackage := td.Packages["testmodule"]
				require.NotNil(t, mainPackage)
				field := mainPackage.NamedTypes["Container"].Fields[0]
				assert.Equal(t, "lib.Thing", field.TypeString)
				assert.Equal(t, "example.com/lib", field.PackagePath)
			},
		},
		{
			name: "Q/55 - Package named 'main' in a subdirectory",
			sources: map[string]string{
				"command/server/main.go": `package main; type Config struct{}`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				packagePath := "testmodule/command/server"
				pkg := td.Packages[packagePath]
				require.NotNil(t, pkg, "Package should exist at its file path location")

				assert.Equal(t, "main", pkg.Name)
				typ := pkg.NamedTypes["Config"]
				require.NotNil(t, typ)
				assert.Equal(t, "main.Config", typ.TypeString)
			},
		},

		{
			name: "R/56 - Type Definition on Slice with Methods",
			sources: map[string]string{
				"main.go": `package main; type IntSlice []int; func (s IntSlice) Sum() int { return 0 }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				typ := td.Packages["testmodule"].NamedTypes["IntSlice"]
				require.NotNil(t, typ)
				assert.False(t, typ.IsAlias)
				assert.Equal(t, "[]int", typ.UnderlyingTypeString)
				require.Len(t, typ.Methods, 1)
				assert.Equal(t, "Sum", typ.Methods[0].Name)
			},
		},
		{
			name: "R/57 - Type Definition on Map with Methods",
			sources: map[string]string{
				"main.go": `package main; type Headers map[string]string; func (h Headers) Get(key string) string { return "" }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				typ := td.Packages["testmodule"].NamedTypes["Headers"]
				require.NotNil(t, typ)
				assert.False(t, typ.IsAlias)
				assert.Equal(t, "map[string]string", typ.UnderlyingTypeString)
				require.Len(t, typ.Methods, 1)
				assert.Equal(t, "Get", typ.Methods[0].Name)
			},
		},
		{
			name: "R/58 - Alias to 'any'",
			sources: map[string]string{
				"main.go": `package main; type Data = any; type Container struct{ Value Data }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				alias := td.Packages["testmodule"].NamedTypes["Data"]
				require.NotNil(t, alias)
				assert.True(t, alias.IsAlias)
				assert.Equal(t, "any", alias.TypeString)

				field := td.Packages["testmodule"].NamedTypes["Container"].Fields[0]
				assert.Equal(t, "main.Data", field.TypeString)
			},
		},
		{
			name: "R/59 - Alias to a Type from a Dot-Imported Package",
			sources: map[string]string{
				"models/user.go": `package models; type User struct{}`,
				"main.go":        `package main; import . "testmodule/models"; type LocalUser = User`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				alias := td.Packages["testmodule"].NamedTypes["LocalUser"]
				require.NotNil(t, alias)
				assert.True(t, alias.IsAlias)

				assert.Equal(t, "models.User", alias.TypeString)
			},
		},
		{
			name: "R/60 - Method whose receiver is an instantiated generic type",
			sources: map[string]string{
				"main.go": `package main;
				type Box[T any] struct { Value T }
				func (b Box[string]) GetOrDefault(definition string) string { return b.Value }
				`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {

				typ := td.Packages["testmodule"].NamedTypes["Box"]
				require.NotNil(t, typ)
				require.Len(t, typ.Methods, 1)
				method := typ.Methods[0]
				assert.Equal(t, "GetOrDefault", method.Name)
				assert.Equal(t, "string", method.TypeString)
				assert.Equal(t, []string{"string"}, method.Signature.Params)
			},
		},
		{
			name: "R/61 - Deeply Nested Structs and Cross-Package Aliases",
			sources: map[string]string{

				"layer1/types.go": `
            package layer1
            import "testmodule/layer2"
            type Layer1Response struct {
                L2Data layer2.Layer2Response
            }`,
				"layer2/types.go": `
            package layer2
            import "testmodule/layer3"
            type Layer2Response struct {
                L3Data layer3.Layer3Response
            }
            type L3Alias = layer3.Layer3Response`,
				"layer3/types.go": `
            package layer3
            import "testmodule/models"
            type Layer3Response struct {
                FinalItem models.Data
            }`,
				"models/data.go": `
            package models
            type Data struct {
                Name string
            }`,
				"main.go": `
            package main
            import (
                "testmodule/layer1"
                "testmodule/layer2"
                "testmodule/layer3"
            )
            // This struct uses all the layers and the alias.
            type Response struct {
                L1Data  layer1.Layer1Response
                L2Alias layer2.L3Alias
                L3Data  layer3.Layer3Response
            }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {

				mainPackage := td.Packages["testmodule"]
				layer1Package := td.Packages["testmodule/layer1"]
				layer2Package := td.Packages["testmodule/layer2"]
				layer3Package := td.Packages["testmodule/layer3"]
				modelsPackage := td.Packages["testmodule/models"]
				require.NotNil(t, mainPackage, "main package should be encoded")
				require.NotNil(t, layer1Package, "layer1 package should be encoded")
				require.NotNil(t, layer2Package, "layer2 package should be encoded")
				require.NotNil(t, layer3Package, "layer3 package should be encoded")
				require.NotNil(t, modelsPackage, "models package should be encoded")

				mainResponseType := mainPackage.NamedTypes["Response"]
				require.NotNil(t, mainResponseType)
				require.Len(t, mainResponseType.Fields, 3)

				l1DataField := mainResponseType.Fields[0]
				assert.Equal(t, "L1Data", l1DataField.Name)
				assert.Equal(t, "layer1.Layer1Response", l1DataField.TypeString)
				assert.Equal(t, "testmodule/layer1", l1DataField.PackagePath, "PackagePath is vital for the querier to find the type")

				l2AliasField := mainResponseType.Fields[1]
				assert.Equal(t, "L2Alias", l2AliasField.Name)
				assert.Equal(t, "layer2.L3Alias", l2AliasField.TypeString, "TypeString must preserve the alias name")
				assert.Equal(t, "testmodule/layer3", l2AliasField.PackagePath)

				l3DataField := mainResponseType.Fields[2]
				assert.Equal(t, "L3Data", l3DataField.Name)
				assert.Equal(t, "layer3.Layer3Response", l3DataField.TypeString)
				assert.Equal(t, "testmodule/layer3", l3DataField.PackagePath)

				layer1Type := layer1Package.NamedTypes["Layer1Response"]
				require.NotNil(t, layer1Type)
				assert.Contains(t, layer1Type.DefinedInFilePath, "layer1/types.go", "Defining file path must be correct for context switching")
				require.Len(t, layer1Type.Fields, 1)
				assert.Equal(t, "layer2.Layer2Response", layer1Type.Fields[0].TypeString)
				assert.Equal(t, "testmodule/layer2", layer1Type.Fields[0].PackagePath)

				layer2Type := layer2Package.NamedTypes["Layer2Response"]
				require.NotNil(t, layer2Type)
				assert.Contains(t, layer2Type.DefinedInFilePath, "layer2/types.go")
				require.Len(t, layer2Type.Fields, 1)
				assert.Equal(t, "layer3.Layer3Response", layer2Type.Fields[0].TypeString)
				assert.Equal(t, "testmodule/layer3", layer2Type.Fields[0].PackagePath)

				l3AliasType := layer2Package.NamedTypes["L3Alias"]
				require.NotNil(t, l3AliasType, "L3Alias type must be encoded")
				assert.True(t, l3AliasType.IsAlias, "Crucially, IsAlias must be true")
				assert.Equal(t, "layer3.Layer3Response", l3AliasType.TypeString, "Alias TypeString must point to the fully resolved, qualified type")
				assert.Contains(t, l3AliasType.DefinedInFilePath, "layer2/types.go")

				modelsDataType := modelsPackage.NamedTypes["Data"]
				require.NotNil(t, modelsDataType)
				assert.Contains(t, modelsDataType.DefinedInFilePath, "models/data.go")
				require.Len(t, modelsDataType.Fields, 1)
				assert.Equal(t, "Name", modelsDataType.Fields[0].Name)
				assert.Equal(t, "string", modelsDataType.Fields[0].TypeString)
			},
		},
		{
			name: "Encoding/PackagePath on Same-Package Field Type",
			sources: map[string]string{
				"models/types.go": `package models

        type Address struct {
            Street string
        }

        type User struct {
            PrimaryAddress Address
        }`,
			},
			assertions: func(t *testing.T, td *inspector_dto.TypeData) {
				pkg := td.Packages["testmodule/models"]
				require.NotNil(t, pkg)
				userType := pkg.NamedTypes["User"]
				require.NotNil(t, userType)
				require.Len(t, userType.Fields, 1)

				addressField := userType.Fields[0]
				assert.Equal(t, "PrimaryAddress", addressField.Name)

				assert.Equal(t, "testmodule/models", addressField.PackagePath, "PackagePath MUST be populated even for types in the same package")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			typeData := setupEncoderTest(t, tc.sources)

			tc.assertions(t, typeData)
		})
	}
}
