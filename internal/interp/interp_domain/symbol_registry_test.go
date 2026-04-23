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
	"go/types"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSymbolRegistryHasPackage(t *testing.T) {
	t.Parallel()
	reg := NewSymbolRegistry(SymbolExports{
		"fmt":     {"Println": reflect.ValueOf(func(...any) {})},
		"strings": {"Contains": reflect.ValueOf(func(string, string) bool { return false })},
	})
	require.True(t, reg.HasPackage("fmt"))
	require.True(t, reg.HasPackage("strings"))
	require.False(t, reg.HasPackage("os"))
}

func TestSymbolRegistryAllPackages(t *testing.T) {
	t.Parallel()
	reg := NewSymbolRegistry(SymbolExports{
		"fmt":     {"Println": reflect.ValueOf(0)},
		"strings": {"Contains": reflect.ValueOf(0)},
	})
	pkgs := reg.AllPackages()
	sort.Strings(pkgs)
	require.Equal(t, []string{"fmt", "strings"}, pkgs)
}

func TestSymbolRegistryZeroValueForType(t *testing.T) {
	t.Parallel()

	type MyStruct struct {
		X int
	}

	reg := NewSymbolRegistry(SymbolExports{
		"mypkg": {
			"MyStruct": reflect.ValueOf((*MyStruct)(nil)),
		},
	})

	value, ok := reg.ZeroValueForType("mypkg", "MyStruct")
	require.True(t, ok)
	require.Equal(t, reflect.Struct, value.Kind())
	require.Equal(t, 0, int(value.FieldByName("X").Int()))

	_, ok = reg.ZeroValueForType("mypkg", "NoSuch")
	require.False(t, ok)
}

func TestSymbolRegistryZeroValueForTypeNonPointer(t *testing.T) {
	t.Parallel()
	reg := NewSymbolRegistry(SymbolExports{
		"pkg": {"Func": reflect.ValueOf(func() {})},
	})
	_, ok := reg.ZeroValueForType("pkg", "Func")
	require.False(t, ok)
}

func TestSymbolRegistryTypeOwnersFacadeAlias(t *testing.T) {
	t.Parallel()

	reg := NewSymbolRegistry(SymbolExports{
		"myfacade": {
			"MyTime": reflect.ValueOf((*time.Time)(nil)),
		},
		"consumer": {
			"Format": reflect.ValueOf(func(t time.Time) string { return t.String() }),
		},
	})

	rt := reflect.TypeFor[time.Time]()
	require.Equal(t, "myfacade", reg.typeOwners[rt])

	reg.SynthesiseAll()

	facadePkg, err := reg.Import("myfacade")
	require.NoError(t, err)
	obj := facadePkg.Scope().Lookup("MyTime")
	require.NotNil(t, obj, "MyTime should be in facade scope")

	consumerPkg, err := reg.Import("consumer")
	require.NoError(t, err)
	fmtObj := consumerPkg.Scope().Lookup("Format")
	require.NotNil(t, fmtObj, "Format should be in consumer scope")

	sig, ok := fmtObj.Type().(*types.Signature)
	require.True(t, ok, "expected *types.Signature, got %T", fmtObj.Type())
	paramType := sig.Params().At(0).Type()
	named, ok := paramType.(*types.Named)
	require.True(t, ok, "parameter should be named type, got %T: %v", paramType, paramType)
	require.Equal(t, "MyTime", named.Obj().Name())
	require.Equal(t, "myfacade", named.Obj().Pkg().Path())
}

type sectionLikeNode struct {
	Title    string
	Slug     string
	Children []sectionLikeNode
	Level    int
}

type linkedListNode struct {
	Value int
	Next  *linkedListNode
}

type treeByKey struct {
	Name     string
	Children map[string]*treeByKey
}

type mutualA struct {
	Label string
	B     *mutualB
}

type mutualB struct {
	Marker int
	A      *mutualA
}

type recursiveDAG struct {
	ID       string
	Children []*recursiveDAG
}

func TestSymbolRegistrySelfReferentialStruct(t *testing.T) {
	t.Parallel()

	reg := NewSymbolRegistry(SymbolExports{
		"tree": {
			"Node": reflect.ValueOf((*sectionLikeNode)(nil)),
		},
	})

	pkg, err := reg.Import("tree")
	require.NoError(t, err)

	obj := pkg.Scope().Lookup("Node")
	require.NotNil(t, obj, "Node should be registered")
	named, ok := obj.Type().(*types.Named)
	require.True(t, ok, "Node should be a named type, got %T", obj.Type())

	underlying, ok := named.Underlying().(*types.Struct)
	require.True(t, ok, "underlying should be a struct, got %T", named.Underlying())

	childrenField := findStructField(t, underlying, "Children")
	slice, ok := childrenField.Type().(*types.Slice)
	require.True(t, ok, "Children should be a slice, got %T", childrenField.Type())

	elem := slice.Elem()
	elemNamed, ok := elem.(*types.Named)
	require.True(t, ok,
		"Children element type should be the named SectionNode-like type, got %T (%v)",
		elem, elem)
	require.Same(t, named, elemNamed,
		"Children element should be the same *types.Named as the parent")
}

func TestSymbolRegistrySelfReferentialPointer(t *testing.T) {
	t.Parallel()

	reg := NewSymbolRegistry(SymbolExports{
		"ll": {
			"Node": reflect.ValueOf((*linkedListNode)(nil)),
		},
	})

	pkg, err := reg.Import("ll")
	require.NoError(t, err)

	named, ok := pkg.Scope().Lookup("Node").Type().(*types.Named)
	require.True(t, ok)

	underlying, ok := named.Underlying().(*types.Struct)
	require.True(t, ok, "underlying should be a struct, got %T", named.Underlying())

	nextField := findStructField(t, underlying, "Next")
	pointer, ok := nextField.Type().(*types.Pointer)
	require.True(t, ok, "Next should be a pointer, got %T", nextField.Type())
	elem, ok := pointer.Elem().(*types.Named)
	require.True(t, ok, "Next should point to named type, got %T", pointer.Elem())
	require.Same(t, named, elem)
}

func TestSymbolRegistrySelfReferentialMap(t *testing.T) {
	t.Parallel()

	reg := NewSymbolRegistry(SymbolExports{
		"tm": {
			"Tree": reflect.ValueOf((*treeByKey)(nil)),
		},
	})

	pkg, err := reg.Import("tm")
	require.NoError(t, err)

	named, ok := pkg.Scope().Lookup("Tree").Type().(*types.Named)
	require.True(t, ok)

	underlying, ok := named.Underlying().(*types.Struct)
	require.True(t, ok, "underlying should be a struct, got %T", named.Underlying())

	childrenField := findStructField(t, underlying, "Children")
	mapType, ok := childrenField.Type().(*types.Map)
	require.True(t, ok, "Children should be a map, got %T", childrenField.Type())
	pointer, ok := mapType.Elem().(*types.Pointer)
	require.True(t, ok, "Children value should be pointer, got %T", mapType.Elem())
	elem, ok := pointer.Elem().(*types.Named)
	require.True(t, ok, "map value pointer should reference named type, got %T", pointer.Elem())
	require.Same(t, named, elem)
}

func TestSymbolRegistryMutuallyRecursiveStructs(t *testing.T) {
	t.Parallel()

	reg := NewSymbolRegistry(SymbolExports{
		"pair": {
			"A": reflect.ValueOf((*mutualA)(nil)),
			"B": reflect.ValueOf((*mutualB)(nil)),
		},
	})

	pkg, err := reg.Import("pair")
	require.NoError(t, err)

	aNamed, ok := pkg.Scope().Lookup("A").Type().(*types.Named)
	require.True(t, ok)
	bNamed, ok := pkg.Scope().Lookup("B").Type().(*types.Named)
	require.True(t, ok)

	aStruct, ok := aNamed.Underlying().(*types.Struct)
	require.True(t, ok, "A underlying should be a struct, got %T", aNamed.Underlying())
	bStruct, ok := bNamed.Underlying().(*types.Struct)
	require.True(t, ok, "B underlying should be a struct, got %T", bNamed.Underlying())

	assertFieldPointsToNamed(t, aStruct, "B", bNamed)
	assertFieldPointsToNamed(t, bStruct, "A", aNamed)
}

func TestSymbolRegistryRecursiveDAG(t *testing.T) {
	t.Parallel()

	reg := NewSymbolRegistry(SymbolExports{
		"ast": {
			"Node": reflect.ValueOf((*recursiveDAG)(nil)),
		},
	})

	pkg, err := reg.Import("ast")
	require.NoError(t, err)

	named, ok := pkg.Scope().Lookup("Node").Type().(*types.Named)
	require.True(t, ok, "Node should be a named type")
	underlying, ok := named.Underlying().(*types.Struct)
	require.True(t, ok, "underlying should be a struct, got %T", named.Underlying())

	childrenField := findStructField(t, underlying, "Children")
	slice, ok := childrenField.Type().(*types.Slice)
	require.True(t, ok, "Children should be a slice, got %T", childrenField.Type())
	pointer, ok := slice.Elem().(*types.Pointer)
	require.True(t, ok, "slice element should be pointer, got %T", slice.Elem())
	elem, ok := pointer.Elem().(*types.Named)
	require.True(t, ok, "pointer element should be named, got %T", pointer.Elem())
	require.Same(t, named, elem)
}

func TestSymbolRegistrySelfReferentialStructTypeChecks(t *testing.T) {
	t.Parallel()

	service := newTestServiceWithSymbols(t, SymbolExports{
		"tree": {
			"Node": reflect.ValueOf((*sectionLikeNode)(nil)),
			"Make": reflect.ValueOf(func() sectionLikeNode {
				return sectionLikeNode{
					Title: "root",
					Slug:  "root",
					Level: 1,
					Children: []sectionLikeNode{
						{Title: "a", Slug: "a", Level: 2},
						{Title: "b", Slug: "b", Level: 2, Children: []sectionLikeNode{
							{Title: "b1", Slug: "b1", Level: 3},
						}},
					},
				}
			}),
		},
	})

	result, err := service.Eval(context.Background(), `
import "tree"
func walk(node tree.Node) int {
	total := 1
	for _, child := range node.Children {
		total += walk(child)
	}
	return total
}
walk(tree.Make())
`)
	require.NoError(t, err, "eval should not fail on self-referential type")
	require.Equal(t, int64(4), result)
}

func TestSymbolRegistryMutuallyRecursiveTypeChecks(t *testing.T) {
	t.Parallel()

	service := newTestServiceWithSymbols(t, SymbolExports{
		"pair": {
			"A": reflect.ValueOf((*mutualA)(nil)),
			"B": reflect.ValueOf((*mutualB)(nil)),
			"Make": reflect.ValueOf(func() *mutualA {
				b := &mutualB{Marker: 7}
				a := &mutualA{Label: "root", B: b}
				b.A = a
				return a
			}),
		},
	})

	result, err := service.Eval(context.Background(), `
import "pair"
a := pair.Make()
a.Label + "/" + a.B.A.Label
`)
	require.NoError(t, err)
	require.Equal(t, "root/root", result)
}

func findStructField(t *testing.T, st *types.Struct, name string) *types.Var {
	t.Helper()
	for field := range st.Fields() {
		if field.Name() == name {
			return field
		}
	}
	t.Fatalf("field %s not found in struct", name)
	return nil
}

func assertFieldPointsToNamed(t *testing.T, st *types.Struct, name string, expected *types.Named) {
	t.Helper()
	field := findStructField(t, st, name)
	pointer, ok := field.Type().(*types.Pointer)
	require.True(t, ok, "field %s should be a pointer, got %T", name, field.Type())
	named, ok := pointer.Elem().(*types.Named)
	require.True(t, ok, "field %s should point to named type, got %T", name, pointer.Elem())
	require.Same(t, expected, named, "field %s should reference expected named type", name)
}

func TestCompositeSymbolProvider(t *testing.T) {
	t.Parallel()
	p1 := &mockSymbolProvider{
		exports: SymbolExports{
			"pkg1": {"A": reflect.ValueOf(1)},
		},
	}
	p2 := &mockSymbolProvider{
		exports: SymbolExports{
			"pkg2": {"B": reflect.ValueOf(2)},
		},
	}
	composite := newCompositeSymbolProvider(p1, p2)
	exports := composite.Exports()
	require.Contains(t, exports, "pkg1")
	require.Contains(t, exports, "pkg2")
}

func TestCompositeSymbolProviderOverride(t *testing.T) {
	t.Parallel()
	p1 := &mockSymbolProvider{
		exports: SymbolExports{
			"pkg": {"X": reflect.ValueOf(1)},
		},
	}
	p2 := &mockSymbolProvider{
		exports: SymbolExports{
			"pkg": {"X": reflect.ValueOf(2)},
		},
	}
	composite := newCompositeSymbolProvider(p1, p2)
	exports := composite.Exports()
	require.Equal(t, 2, int(exports["pkg"]["X"].Int()))
}
