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
