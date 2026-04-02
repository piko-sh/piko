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

package compiler_domain

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/js_ast"
)

func TestNewRegistryContext(t *testing.T) {
	t.Run("creates fresh registry context", func(t *testing.T) {
		rc := NewRegistryContext()

		require.NotNil(t, rc)
		require.NotNil(t, rc.identifiers)
		require.NotNil(t, rc.bindings)
		require.NotNil(t, rc.locRefs)
	})

	t.Run("each call creates independent registry", func(t *testing.T) {
		rc1 := NewRegistryContext()
		rc2 := NewRegistryContext()

		identifier := rc1.MakeIdentifier("test")
		rc1.RegisterIdentifierName(identifier, "test")

		assert.Equal(t, "test", rc1.LookupIdentifierName(identifier))

		assert.NotSame(t, rc1.identifiers, rc2.identifiers)
	})
}

func TestRegistryContext_MakeIdentifier(t *testing.T) {
	rc := NewRegistryContext()

	t.Run("creates identifier with name", func(t *testing.T) {
		identifier := rc.MakeIdentifier("myVar")

		require.NotNil(t, identifier)
		assert.Equal(t, "myVar", rc.LookupIdentifierName(identifier))
	})

	t.Run("creates unique identifiers", func(t *testing.T) {
		ident1 := rc.MakeIdentifier("var1")
		ident2 := rc.MakeIdentifier("var2")

		assert.NotSame(t, ident1, ident2)
		assert.Equal(t, "var1", rc.LookupIdentifierName(ident1))
		assert.Equal(t, "var2", rc.LookupIdentifierName(ident2))
	})

	t.Run("same name creates different identifiers", func(t *testing.T) {
		ident1 := rc.MakeIdentifier("sameName")
		ident2 := rc.MakeIdentifier("sameName")

		assert.NotSame(t, ident1, ident2)
		assert.Equal(t, "sameName", rc.LookupIdentifierName(ident1))
		assert.Equal(t, "sameName", rc.LookupIdentifierName(ident2))
	})
}

func TestRegistryContext_MakeIdentifierExpr(t *testing.T) {
	rc := NewRegistryContext()

	t.Run("creates expression containing identifier", func(t *testing.T) {
		expression := rc.MakeIdentifierExpr("myVar")

		require.NotNil(t, expression.Data)
		identifier, ok := expression.Data.(*js_ast.EIdentifier)
		require.True(t, ok)
		assert.Equal(t, "myVar", rc.LookupIdentifierName(identifier))
	})
}

func TestRegistryContext_RegisterIdentifierName(t *testing.T) {
	rc := NewRegistryContext()

	t.Run("registers name for identifier", func(t *testing.T) {
		identifier := &js_ast.EIdentifier{Ref: ast.Ref{}}
		rc.RegisterIdentifierName(identifier, "testName")

		assert.Equal(t, "testName", rc.LookupIdentifierName(identifier))
	})

	t.Run("nil identifier is ignored", func(t *testing.T) {

		rc.RegisterIdentifierName(nil, "test")
	})

	t.Run("empty name is ignored", func(t *testing.T) {
		identifier := &js_ast.EIdentifier{Ref: ast.Ref{}}
		rc.RegisterIdentifierName(identifier, "")

		assert.Empty(t, rc.LookupIdentifierName(identifier))
	})

	t.Run("overwrites existing registration", func(t *testing.T) {
		identifier := &js_ast.EIdentifier{Ref: ast.Ref{}}
		rc.RegisterIdentifierName(identifier, "first")
		rc.RegisterIdentifierName(identifier, "second")

		assert.Equal(t, "second", rc.LookupIdentifierName(identifier))
	})
}

func TestRegistryContext_LookupIdentifierName(t *testing.T) {
	rc := NewRegistryContext()

	t.Run("returns registered name", func(t *testing.T) {
		identifier := rc.MakeIdentifier("testVar")
		assert.Equal(t, "testVar", rc.LookupIdentifierName(identifier))
	})

	t.Run("returns empty for unknown identifier", func(t *testing.T) {
		identifier := &js_ast.EIdentifier{Ref: ast.Ref{}}

		result := rc.LookupIdentifierName(identifier)

		assert.NotPanics(t, func() { rc.LookupIdentifierName(identifier) })
		_ = result
	})

	t.Run("returns empty for nil identifier", func(t *testing.T) {
		assert.Empty(t, rc.LookupIdentifierName(nil))
	})
}

func TestRegistryContext_MakeBinding(t *testing.T) {
	rc := NewRegistryContext()

	t.Run("creates binding with name", func(t *testing.T) {
		binding := rc.MakeBinding("param")

		require.NotNil(t, binding.Data)
		bIdent, ok := binding.Data.(*js_ast.BIdentifier)
		require.True(t, ok)
		assert.Equal(t, "param", rc.LookupBindingName(bIdent))
	})

	t.Run("creates unique bindings", func(t *testing.T) {
		binding1 := rc.MakeBinding("param1")
		binding2 := rc.MakeBinding("param2")

		bIdent1, ok := binding1.Data.(*js_ast.BIdentifier)
		require.True(t, ok, "binding1.Data should be *js_ast.BIdentifier")
		bIdent2, ok := binding2.Data.(*js_ast.BIdentifier)
		require.True(t, ok, "binding2.Data should be *js_ast.BIdentifier")

		assert.NotSame(t, bIdent1, bIdent2)
		assert.Equal(t, "param1", rc.LookupBindingName(bIdent1))
		assert.Equal(t, "param2", rc.LookupBindingName(bIdent2))
	})
}

func TestRegistryContext_RegisterBindingName(t *testing.T) {
	rc := NewRegistryContext()

	t.Run("registers name for binding", func(t *testing.T) {
		bind := &js_ast.BIdentifier{Ref: ast.Ref{}}
		rc.RegisterBindingName(bind, "testBinding")

		assert.Equal(t, "testBinding", rc.LookupBindingName(bind))
	})

	t.Run("nil binding is ignored", func(t *testing.T) {
		rc.RegisterBindingName(nil, "test")
	})

	t.Run("empty name is ignored", func(t *testing.T) {
		bind := &js_ast.BIdentifier{Ref: ast.Ref{}}
		rc.RegisterBindingName(bind, "")

		assert.Empty(t, rc.LookupBindingName(bind))
	})
}

func TestRegistryContext_LookupBindingName(t *testing.T) {
	rc := NewRegistryContext()

	t.Run("returns registered name", func(t *testing.T) {
		binding := rc.MakeBinding("testParam")
		bIdent, ok := binding.Data.(*js_ast.BIdentifier)
		require.True(t, ok, "binding.Data should be *js_ast.BIdentifier")

		assert.Equal(t, "testParam", rc.LookupBindingName(bIdent))
	})

	t.Run("returns empty for nil binding", func(t *testing.T) {
		assert.Empty(t, rc.LookupBindingName(nil))
	})
}

func TestRegistryContext_MakeLocRef(t *testing.T) {
	rc := NewRegistryContext()

	t.Run("creates LocRef with name", func(t *testing.T) {
		locRef := rc.MakeLocRef("ClassName")

		require.NotNil(t, locRef)
		assert.Equal(t, "ClassName", rc.LookupLocRefName(locRef))
	})

	t.Run("creates unique LocRefs", func(t *testing.T) {
		locRef1 := rc.MakeLocRef("Class1")
		locRef2 := rc.MakeLocRef("Class2")

		assert.NotSame(t, locRef1, locRef2)
		assert.Equal(t, "Class1", rc.LookupLocRefName(locRef1))
		assert.Equal(t, "Class2", rc.LookupLocRefName(locRef2))
	})
}

func TestRegistryContext_RegisterLocRefName(t *testing.T) {
	rc := NewRegistryContext()

	t.Run("registers name for LocRef", func(t *testing.T) {
		locRef := &ast.LocRef{Ref: ast.Ref{}}
		rc.RegisterLocRefName(locRef, "TestClass")

		assert.Equal(t, "TestClass", rc.LookupLocRefName(locRef))
	})

	t.Run("nil LocRef is ignored", func(t *testing.T) {
		rc.RegisterLocRefName(nil, "test")
	})

	t.Run("empty name is ignored", func(t *testing.T) {
		locRef := &ast.LocRef{Ref: ast.Ref{}}
		rc.RegisterLocRefName(locRef, "")

		assert.Empty(t, rc.LookupLocRefName(locRef))
	})
}

func TestRegistryContext_LookupLocRefName(t *testing.T) {
	rc := NewRegistryContext()

	t.Run("returns registered name", func(t *testing.T) {
		locRef := rc.MakeLocRef("TestFunc")
		assert.Equal(t, "TestFunc", rc.LookupLocRefName(locRef))
	})

	t.Run("returns empty for nil LocRef", func(t *testing.T) {
		assert.Empty(t, rc.LookupLocRefName(nil))
	})
}

func TestRegistryContext_Isolation(t *testing.T) {
	t.Run("contexts are isolated from each other", func(t *testing.T) {
		rc1 := NewRegistryContext()
		rc2 := NewRegistryContext()

		ident1 := rc1.MakeIdentifier("fromRC1")
		binding1 := rc1.MakeBinding("bindingRC1")
		locRef1 := rc1.MakeLocRef("classRC1")

		ident2 := rc2.MakeIdentifier("fromRC2")
		binding2 := rc2.MakeBinding("bindingRC2")
		locRef2 := rc2.MakeLocRef("classRC2")

		assert.Equal(t, "fromRC1", rc1.LookupIdentifierName(ident1))
		assert.Equal(t, "bindingRC1", rc1.LookupBindingName(binding1.Data.(*js_ast.BIdentifier)))
		assert.Equal(t, "classRC1", rc1.LookupLocRefName(locRef1))

		assert.Equal(t, "fromRC2", rc2.LookupIdentifierName(ident2))
		assert.Equal(t, "bindingRC2", rc2.LookupBindingName(binding2.Data.(*js_ast.BIdentifier)))
		assert.Equal(t, "classRC2", rc2.LookupLocRefName(locRef2))

		assert.NotEqual(t, "fromRC1", rc2.LookupIdentifierName(ident1))
	})
}

func TestRegistryContext_ConcurrentAccess(t *testing.T) {
	t.Run("handles concurrent registration", func(t *testing.T) {
		rc := NewRegistryContext()
		var wg sync.WaitGroup
		numGoroutines := 100

		wg.Add(numGoroutines)
		for i := range numGoroutines {
			go func(index int) {
				defer wg.Done()

				identifier := rc.MakeIdentifier("var")
				_ = rc.LookupIdentifierName(identifier)
			}(i)
		}

		wg.Wait()

	})

	t.Run("handles concurrent lookup", func(t *testing.T) {
		rc := NewRegistryContext()
		identifier := rc.MakeIdentifier("sharedVar")

		var wg sync.WaitGroup
		numGoroutines := 100

		wg.Add(numGoroutines)
		for range numGoroutines {
			go func() {
				defer wg.Done()
				name := rc.LookupIdentifierName(identifier)
				assert.Equal(t, "sharedVar", name)
			}()
		}

		wg.Wait()
	})
}

func TestGlobalIdentifierRegistry(t *testing.T) {

	ClearIdentifierRegistry()

	t.Run("makeIdentifier registers in global registry", func(t *testing.T) {
		identifier := makeIdentifier("globalVar")

		assert.Equal(t, "globalVar", lookupIdentifierName(identifier))
	})

	t.Run("ClearIdentifierRegistry removes all entries", func(t *testing.T) {
		identifier := makeIdentifier("toBeCleared")
		assert.Equal(t, "toBeCleared", lookupIdentifierName(identifier))

		ClearIdentifierRegistry()

		assert.Empty(t, lookupIdentifierName(identifier))
	})
}

func TestGlobalBindingRegistry(t *testing.T) {
	ClearBindingRegistry()

	t.Run("makeBinding registers in global registry", func(t *testing.T) {
		binding := makeBinding("globalBinding")
		bIdent, ok := binding.Data.(*js_ast.BIdentifier)
		require.True(t, ok, "binding.Data should be *js_ast.BIdentifier")

		assert.Equal(t, "globalBinding", lookupBindingName(bIdent))
	})

	t.Run("ClearBindingRegistry removes all entries", func(t *testing.T) {
		binding := makeBinding("toBeCleared")
		bIdent, ok := binding.Data.(*js_ast.BIdentifier)
		require.True(t, ok, "binding.Data should be *js_ast.BIdentifier")

		ClearBindingRegistry()

		assert.Empty(t, lookupBindingName(bIdent))
	})
}

func TestGlobalLocRefRegistry(t *testing.T) {
	ClearLocRefRegistry()

	t.Run("makeLocRef registers in global registry", func(t *testing.T) {
		locRef := makeLocRef("GlobalClass")

		assert.Equal(t, "GlobalClass", lookupLocRefName(locRef))
	})

	t.Run("ClearLocRefRegistry removes all entries", func(t *testing.T) {
		locRef := makeLocRef("ToBeCleared")

		ClearLocRefRegistry()

		assert.Empty(t, lookupLocRefName(locRef))
	})
}
