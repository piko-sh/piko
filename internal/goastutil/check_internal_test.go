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

package goastutil

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/packages"
)

func TestIsStandardLibraryPath(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		expected bool
	}{
		{name: "fmt", path: "fmt", expected: true},
		{name: "net/http", path: "net/http", expected: true},
		{name: "encoding/json", path: "encoding/json", expected: true},
		{name: "context", path: "context", expected: true},
		{name: "strings", path: "strings", expected: true},
		{name: "io", path: "io", expected: true},
		{name: "os", path: "os", expected: true},
		{name: "sync", path: "sync", expected: true},
		{name: "time", path: "time", expected: true},
		{name: "unsafe", path: "unsafe", expected: true},
		{name: "github.com third party", path: "github.com/user/repo", expected: false},
		{name: "golang.org/x third party", path: "golang.org/x/tools", expected: false},
		{name: "example.com third party", path: "example.com/pkg", expected: false},
		{name: "gopkg.in third party", path: "gopkg.in/yaml.v2", expected: false},
		{name: "google.golang.org third party", path: "google.golang.org/grpc", expected: false},
		{name: "single segment with dot", path: "some.pkg", expected: false},
		{name: "empty path", path: "", expected: true},
		{name: "path starting with slash", path: "/fmt", expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isStandardLibraryPath(tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCheckCompositeTypeInternal(t *testing.T) {
	emptyPackageMap := make(map[string]*packages.Package)

	t.Run("Array of primitive is internal", func(t *testing.T) {
		arrayType := types.NewArray(types.Typ[types.Int], 5)
		result := checkCompositeTypeInternal(arrayType, emptyPackageMap)
		assert.True(t, result)
	})

	t.Run("Slice of primitive is internal", func(t *testing.T) {
		sliceType := types.NewSlice(types.Typ[types.String])
		result := checkCompositeTypeInternal(sliceType, emptyPackageMap)
		assert.True(t, result)
	})

	t.Run("Pointer to primitive is internal", func(t *testing.T) {
		ptrType := types.NewPointer(types.Typ[types.Int])
		result := checkCompositeTypeInternal(ptrType, emptyPackageMap)
		assert.True(t, result)
	})

	t.Run("Channel of primitive is internal", func(t *testing.T) {
		chanType := types.NewChan(types.SendRecv, types.Typ[types.Int])
		result := checkCompositeTypeInternal(chanType, emptyPackageMap)
		assert.True(t, result)
	})

	t.Run("Map of primitives is internal", func(t *testing.T) {
		mapType := types.NewMap(types.Typ[types.String], types.Typ[types.Int])
		result := checkCompositeTypeInternal(mapType, emptyPackageMap)
		assert.True(t, result)
	})

	t.Run("Struct type returns false (default case)", func(t *testing.T) {
		structType := types.NewStruct(nil, nil)
		result := checkCompositeTypeInternal(structType, emptyPackageMap)
		assert.False(t, result)
	})

	t.Run("Interface type returns false (default case)", func(t *testing.T) {
		ifaceType := types.NewInterfaceType(nil, nil)
		result := checkCompositeTypeInternal(ifaceType, emptyPackageMap)
		assert.False(t, result)
	})
}

func TestCheckNamedOrAliasType(t *testing.T) {
	seen := make(map[types.Type]bool)
	pkg := types.NewPackage("example.com/test", "test")

	t.Run("Named type with package returns true (non-primitive)", func(t *testing.T) {
		namedType := types.NewNamed(
			types.NewTypeName(0, pkg, "MyType", nil),
			types.Typ[types.Int],
			nil,
		)
		result := checkNamedOrAliasType(namedType, seen)
		assert.True(t, result)
	})

	t.Run("Named type without package returns false (primitive)", func(t *testing.T) {

		errorType := types.Universe.Lookup("error").Type()
		result := checkNamedOrAliasType(errorType, make(map[types.Type]bool))
		assert.False(t, result)
	})

	t.Run("Basic type returns false", func(t *testing.T) {
		result := checkNamedOrAliasType(types.Typ[types.Int], seen)
		assert.False(t, result)
	})
}

func TestCheckUnderlyingPrimitive(t *testing.T) {
	seen := make(map[types.Type]bool)

	t.Run("Signature is primitive", func(t *testing.T) {
		sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		result := checkUnderlyingPrimitive(sig, seen)
		assert.True(t, result)
	})

	t.Run("Basic type is primitive", func(t *testing.T) {
		result := checkUnderlyingPrimitive(types.Typ[types.Int], seen)
		assert.True(t, result)
	})

	t.Run("Empty interface is primitive", func(t *testing.T) {
		emptyIface := types.NewInterfaceType(nil, nil)
		result := checkUnderlyingPrimitive(emptyIface, seen)
		assert.True(t, result)
	})

	t.Run("Interface with methods is not primitive", func(t *testing.T) {
		sig := types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.String])), false)
		method := types.NewFunc(0, nil, "String", sig)
		iface := types.NewInterfaceType([]*types.Func{method}, nil)
		iface.Complete()

		result := checkUnderlyingPrimitive(iface, make(map[types.Type]bool))
		assert.False(t, result)
	})

	t.Run("Slice without function is not primitive", func(t *testing.T) {
		sliceType := types.NewSlice(types.Typ[types.Int])
		result := checkUnderlyingPrimitive(sliceType, seen)
		assert.False(t, result)
	})

	t.Run("Slice with function is primitive", func(t *testing.T) {
		funcType := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		sliceType := types.NewSlice(funcType)
		result := checkUnderlyingPrimitive(sliceType, make(map[types.Type]bool))
		assert.True(t, result)
	})
}

func TestContainsFunction(t *testing.T) {
	t.Run("Signature type contains function", func(t *testing.T) {
		sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		result := containsFunction(sig, make(map[types.Type]bool))
		assert.True(t, result)
	})

	t.Run("Basic type does not contain function", func(t *testing.T) {
		result := containsFunction(types.Typ[types.Int], make(map[types.Type]bool))
		assert.False(t, result)
	})

	t.Run("Nil type does not contain function", func(t *testing.T) {
		result := containsFunction(nil, make(map[types.Type]bool))
		assert.False(t, result)
	})

	t.Run("Map with function value contains function", func(t *testing.T) {
		funcType := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		mapType := types.NewMap(types.Typ[types.String], funcType)
		result := containsFunction(mapType, make(map[types.Type]bool))
		assert.True(t, result)
	})

	t.Run("Map with function key contains function", func(t *testing.T) {
		funcType := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		mapType := types.NewMap(funcType, types.Typ[types.String])
		result := containsFunction(mapType, make(map[types.Type]bool))
		assert.True(t, result)
	})

	t.Run("Slice of functions contains function", func(t *testing.T) {
		funcType := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		sliceType := types.NewSlice(funcType)
		result := containsFunction(sliceType, make(map[types.Type]bool))
		assert.True(t, result)
	})

	t.Run("Array of functions contains function", func(t *testing.T) {
		funcType := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		arrayType := types.NewArray(funcType, 5)
		result := containsFunction(arrayType, make(map[types.Type]bool))
		assert.True(t, result)
	})

	t.Run("Pointer to function contains function", func(t *testing.T) {
		funcType := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		ptrType := types.NewPointer(funcType)
		result := containsFunction(ptrType, make(map[types.Type]bool))
		assert.True(t, result)
	})

	t.Run("Channel of function contains function", func(t *testing.T) {
		funcType := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		chanType := types.NewChan(types.SendRecv, funcType)
		result := containsFunction(chanType, make(map[types.Type]bool))
		assert.True(t, result)
	})

	t.Run("Cycle detection prevents infinite recursion", func(t *testing.T) {

		funcType := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		seen := map[types.Type]bool{funcType: true}
		result := containsFunction(funcType, seen)
		assert.False(t, result)
	})
}

var _ = packages.NeedName
