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

package annotator_domain

import (
	goast "go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestNewSymbolTable(t *testing.T) {
	t.Parallel()

	t.Run("creates root symbol table with nil parent", func(t *testing.T) {
		t.Parallel()
		st := NewSymbolTable(nil)

		require.NotNil(t, st)
		assert.Nil(t, st.parent)
		assert.NotNil(t, st.symbols)
		assert.Empty(t, st.symbols)
	})

	t.Run("creates child symbol table with parent", func(t *testing.T) {
		t.Parallel()
		parent := NewSymbolTable(nil)
		child := NewSymbolTable(parent)

		require.NotNil(t, child)
		assert.Same(t, parent, child.parent)
		assert.NotNil(t, child.symbols)
	})

	t.Run("child symbols map is independent from parent", func(t *testing.T) {
		t.Parallel()
		parent := NewSymbolTable(nil)
		parent.Define(Symbol{Name: "parentVar"})

		child := NewSymbolTable(parent)

		assert.Empty(t, child.symbols)

		assert.Len(t, parent.symbols, 1)
	})
}

func TestSymbolTable_Define(t *testing.T) {
	t.Parallel()

	t.Run("defines symbol in current scope", func(t *testing.T) {
		t.Parallel()
		st := NewSymbolTable(nil)
		symbol := Symbol{
			Name:           "myVar",
			CodeGenVarName: "myVar",
			TypeInfo:       &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
		}

		st.Define(symbol)

		result, found := st.symbols["myVar"]
		require.True(t, found)
		assert.Equal(t, "myVar", result.Name)
	})

	t.Run("overwrites existing symbol in same scope", func(t *testing.T) {
		t.Parallel()
		st := NewSymbolTable(nil)
		st.Define(Symbol{Name: "x", CodeGenVarName: "x1"})
		st.Define(Symbol{Name: "x", CodeGenVarName: "x2"})

		result, found := st.symbols["x"]
		require.True(t, found)
		assert.Equal(t, "x2", result.CodeGenVarName)
	})

	t.Run("does not affect parent scope", func(t *testing.T) {
		t.Parallel()
		parent := NewSymbolTable(nil)
		child := NewSymbolTable(parent)

		child.Define(Symbol{Name: "childOnly"})

		_, foundInParent := parent.symbols["childOnly"]
		_, foundInChild := child.symbols["childOnly"]
		assert.False(t, foundInParent)
		assert.True(t, foundInChild)
	})
}

func TestSymbolTable_Find(t *testing.T) {
	t.Parallel()

	t.Run("finds symbol in current scope", func(t *testing.T) {
		t.Parallel()
		st := NewSymbolTable(nil)
		st.Define(Symbol{Name: "localVar", CodeGenVarName: "local"})

		result, found := st.Find("localVar")

		require.True(t, found)
		assert.Equal(t, "localVar", result.Name)
		assert.Equal(t, "local", result.CodeGenVarName)
	})

	t.Run("finds symbol in parent scope", func(t *testing.T) {
		t.Parallel()
		parent := NewSymbolTable(nil)
		parent.Define(Symbol{Name: "parentVar", CodeGenVarName: "p"})

		child := NewSymbolTable(parent)

		result, found := child.Find("parentVar")

		require.True(t, found)
		assert.Equal(t, "parentVar", result.Name)
	})

	t.Run("finds symbol in grandparent scope", func(t *testing.T) {
		t.Parallel()
		grandparent := NewSymbolTable(nil)
		grandparent.Define(Symbol{Name: "gpVar"})

		parent := NewSymbolTable(grandparent)
		child := NewSymbolTable(parent)

		result, found := child.Find("gpVar")

		require.True(t, found)
		assert.Equal(t, "gpVar", result.Name)
	})

	t.Run("local scope shadows parent scope", func(t *testing.T) {
		t.Parallel()
		parent := NewSymbolTable(nil)
		parent.Define(Symbol{Name: "x", CodeGenVarName: "parent_x"})

		child := NewSymbolTable(parent)
		child.Define(Symbol{Name: "x", CodeGenVarName: "child_x"})

		result, found := child.Find("x")

		require.True(t, found)
		assert.Equal(t, "child_x", result.CodeGenVarName)
	})

	t.Run("returns false for undefined symbol", func(t *testing.T) {
		t.Parallel()
		st := NewSymbolTable(nil)

		_, found := st.Find("nonexistent")

		assert.False(t, found)
	})

	t.Run("returns false for undefined symbol with parent", func(t *testing.T) {
		t.Parallel()
		parent := NewSymbolTable(nil)
		child := NewSymbolTable(parent)

		_, found := child.Find("nonexistent")

		assert.False(t, found)
	})
}

func TestSymbolTable_AllSymbolNames(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice for empty table", func(t *testing.T) {
		t.Parallel()
		st := NewSymbolTable(nil)

		result := st.AllSymbolNames()

		assert.Empty(t, result)
	})

	t.Run("returns names from current scope", func(t *testing.T) {
		t.Parallel()
		st := NewSymbolTable(nil)
		st.Define(Symbol{Name: "b"})
		st.Define(Symbol{Name: "a"})
		st.Define(Symbol{Name: "c"})

		result := st.AllSymbolNames()

		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("includes names from parent scope", func(t *testing.T) {
		t.Parallel()
		parent := NewSymbolTable(nil)
		parent.Define(Symbol{Name: "parentVar"})

		child := NewSymbolTable(parent)
		child.Define(Symbol{Name: "childVar"})

		result := child.AllSymbolNames()

		assert.Equal(t, []string{"childVar", "parentVar"}, result)
	})

	t.Run("deduplicates shadowed names", func(t *testing.T) {
		t.Parallel()
		parent := NewSymbolTable(nil)
		parent.Define(Symbol{Name: "x"})

		child := NewSymbolTable(parent)
		child.Define(Symbol{Name: "x"})
		child.Define(Symbol{Name: "y"})

		result := child.AllSymbolNames()

		assert.Equal(t, []string{"x", "y"}, result)
	})

	t.Run("includes grandparent names", func(t *testing.T) {
		t.Parallel()
		grandparent := NewSymbolTable(nil)
		grandparent.Define(Symbol{Name: "gpVar"})

		parent := NewSymbolTable(grandparent)
		parent.Define(Symbol{Name: "pVar"})

		child := NewSymbolTable(parent)
		child.Define(Symbol{Name: "cVar"})

		result := child.AllSymbolNames()

		assert.Equal(t, []string{"cVar", "gpVar", "pVar"}, result)
	})
}

func TestTranslationKeySet_HasLocalKey(t *testing.T) {
	t.Parallel()

	t.Run("returns false for nil TranslationKeySet", func(t *testing.T) {
		t.Parallel()
		var keys *TranslationKeySet
		result := keys.HasLocalKey("anyKey")
		assert.False(t, result)
	})

	t.Run("returns false for nil LocalKeys map", func(t *testing.T) {
		t.Parallel()
		keys := &TranslationKeySet{LocalKeys: nil}
		result := keys.HasLocalKey("anyKey")
		assert.False(t, result)
	})

	t.Run("returns false for missing key", func(t *testing.T) {
		t.Parallel()
		keys := &TranslationKeySet{
			LocalKeys: map[string]struct{}{"existingKey": {}},
		}
		result := keys.HasLocalKey("missingKey")
		assert.False(t, result)
	})

	t.Run("returns true for existing key", func(t *testing.T) {
		t.Parallel()
		keys := &TranslationKeySet{
			LocalKeys: map[string]struct{}{"myKey": {}},
		}
		result := keys.HasLocalKey("myKey")
		assert.True(t, result)
	})

	t.Run("returns true for empty string key if defined", func(t *testing.T) {
		t.Parallel()
		keys := &TranslationKeySet{
			LocalKeys: map[string]struct{}{"": {}},
		}
		result := keys.HasLocalKey("")
		assert.True(t, result)
	})
}

func TestTranslationKeySet_HasGlobalKey(t *testing.T) {
	t.Parallel()

	t.Run("returns false for nil TranslationKeySet", func(t *testing.T) {
		t.Parallel()
		var keys *TranslationKeySet
		result := keys.HasGlobalKey("anyKey")
		assert.False(t, result)
	})

	t.Run("returns false for nil GlobalKeys map", func(t *testing.T) {
		t.Parallel()
		keys := &TranslationKeySet{GlobalKeys: nil}
		result := keys.HasGlobalKey("anyKey")
		assert.False(t, result)
	})

	t.Run("returns false for missing key", func(t *testing.T) {
		t.Parallel()
		keys := &TranslationKeySet{
			GlobalKeys: map[string]struct{}{"existingKey": {}},
		}
		result := keys.HasGlobalKey("missingKey")
		assert.False(t, result)
	})

	t.Run("returns true for existing key", func(t *testing.T) {
		t.Parallel()
		keys := &TranslationKeySet{
			GlobalKeys: map[string]struct{}{"globalKey": {}},
		}
		result := keys.HasGlobalKey("globalKey")
		assert.True(t, result)
	})
}

func TestNewRootAnalysisContext(t *testing.T) {
	t.Parallel()

	t.Run("creates context with all fields set", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(
			diagnostics,
			"github.com/example/app",
			"app",
			"/src/app/page.go",
			"/src/app/page.pk",
		)

		require.NotNil(t, ctx)
		assert.Same(t, diagnostics, ctx.Diagnostics)
		assert.Equal(t, "github.com/example/app", ctx.CurrentGoFullPackagePath)
		assert.Equal(t, "app", ctx.CurrentGoPackageName)
		assert.Equal(t, "/src/app/page.go", ctx.CurrentGoSourcePath)
		assert.Equal(t, "/src/app/page.pk", ctx.SFCSourcePath)
		require.NotNil(t, ctx.Symbols)
		assert.Nil(t, ctx.TranslationKeys)
	})

	t.Run("creates root symbol table", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		assert.Nil(t, ctx.Symbols.parent)
	})
}

func TestAnalysisContext_ForChildScope(t *testing.T) {
	t.Parallel()

	t.Run("creates child with new symbol table", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		parent := NewRootAnalysisContext(diagnostics, "pkg", "pkg", "/a.go", "/a.pk")
		parent.Symbols.Define(Symbol{Name: "parentVar"})

		child := parent.ForChildScope()

		require.NotNil(t, child)
		require.NotNil(t, child.Symbols)
		assert.NotSame(t, parent.Symbols, child.Symbols)
		assert.Same(t, parent.Symbols, child.Symbols.parent)
	})

	t.Run("inherits diagnostics pointer", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		parent := NewRootAnalysisContext(diagnostics, "", "", "", "")

		child := parent.ForChildScope()

		assert.Same(t, diagnostics, child.Diagnostics)
	})

	t.Run("inherits package paths", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		parent := NewRootAnalysisContext(diagnostics, "full/path", "name", "/go.go", "/sfc.pk")

		child := parent.ForChildScope()

		assert.Equal(t, "full/path", child.CurrentGoFullPackagePath)
		assert.Equal(t, "name", child.CurrentGoPackageName)
		assert.Equal(t, "/go.go", child.CurrentGoSourcePath)
		assert.Equal(t, "/sfc.pk", child.SFCSourcePath)
	})

	t.Run("inherits translation keys", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		parent := NewRootAnalysisContext(diagnostics, "", "", "", "")
		keys := &TranslationKeySet{LocalKeys: map[string]struct{}{"key": {}}}
		parent.TranslationKeys = keys

		child := parent.ForChildScope()

		assert.Same(t, keys, child.TranslationKeys)
	})

	t.Run("inherits nil guards", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		parent := NewRootAnalysisContext(diagnostics, "", "", "", "")
		parent.KnownNonNilExpressions = map[string]bool{"props.User": true}

		child := parent.ForChildScope()

		assert.Equal(t, parent.KnownNonNilExpressions, child.KnownNonNilExpressions)
		assert.True(t, child.KnownNonNilExpressions["props.User"])
	})

	t.Run("child can find parent symbols", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		parent := NewRootAnalysisContext(diagnostics, "", "", "", "")
		parent.Symbols.Define(Symbol{Name: "parentVar"})

		child := parent.ForChildScope()

		_, found := child.Symbols.Find("parentVar")
		assert.True(t, found)
	})
}

func TestAnalysisContext_ForChildScopeWithNilGuards(t *testing.T) {
	t.Parallel()

	t.Run("adds new guards to child scope", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		parent := NewRootAnalysisContext(diagnostics, "", "", "", "")

		child := parent.ForChildScopeWithNilGuards([]string{"props.User"})

		assert.True(t, child.KnownNonNilExpressions["props.User"])
	})

	t.Run("preserves parent guards", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		parent := NewRootAnalysisContext(diagnostics, "", "", "", "")
		parent.KnownNonNilExpressions = map[string]bool{"props.Config": true}

		child := parent.ForChildScopeWithNilGuards([]string{"props.User"})

		assert.True(t, child.KnownNonNilExpressions["props.Config"])
		assert.True(t, child.KnownNonNilExpressions["props.User"])
	})

	t.Run("does not modify parent guards", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		parent := NewRootAnalysisContext(diagnostics, "", "", "", "")
		parent.KnownNonNilExpressions = map[string]bool{"existing": true}

		child := parent.ForChildScopeWithNilGuards([]string{"newGuard"})

		assert.False(t, parent.KnownNonNilExpressions["newGuard"])
		assert.True(t, child.KnownNonNilExpressions["newGuard"])
	})

	t.Run("empty guards inherits parent guards", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		parent := NewRootAnalysisContext(diagnostics, "", "", "", "")
		parent.KnownNonNilExpressions = map[string]bool{"guard": true}

		child := parent.ForChildScopeWithNilGuards([]string{})

		assert.Equal(t, parent.KnownNonNilExpressions, child.KnownNonNilExpressions)
		assert.True(t, child.KnownNonNilExpressions["guard"])
	})

	t.Run("nil guards with nil parent guards returns nil", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		parent := NewRootAnalysisContext(diagnostics, "", "", "", "")

		child := parent.ForChildScopeWithNilGuards(nil)

		assert.Nil(t, child.KnownNonNilExpressions)
	})
}

func TestAnalysisContext_IsKnownNonNil(t *testing.T) {
	t.Parallel()

	t.Run("returns false when KnownNonNilExpressions is nil", func(t *testing.T) {
		t.Parallel()
		ctx := &AnalysisContext{KnownNonNilExpressions: nil}

		result := ctx.IsKnownNonNil("props.User")

		assert.False(t, result)
	})

	t.Run("returns false for unknown expression", func(t *testing.T) {
		t.Parallel()
		ctx := &AnalysisContext{
			KnownNonNilExpressions: map[string]bool{"props.Config": true},
		}

		result := ctx.IsKnownNonNil("props.User")

		assert.False(t, result)
	})

	t.Run("returns true for known non-nil expression", func(t *testing.T) {
		t.Parallel()
		ctx := &AnalysisContext{
			KnownNonNilExpressions: map[string]bool{"props.User": true},
		}

		result := ctx.IsKnownNonNil("props.User")

		assert.True(t, result)
	})

	t.Run("exact match required", func(t *testing.T) {
		t.Parallel()
		ctx := &AnalysisContext{
			KnownNonNilExpressions: map[string]bool{"props.User": true},
		}

		assert.False(t, ctx.IsKnownNonNil("props"))
		assert.False(t, ctx.IsKnownNonNil("props.User.Profile"))
	})
}

func TestAnalysisContext_ForNewPackageContext(t *testing.T) {
	t.Parallel()

	t.Run("creates context with new package paths", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		parent := NewRootAnalysisContext(diagnostics, "old/pkg", "old", "/old.go", "/old.pk")

		child := parent.ForNewPackageContext("new/pkg", "new", "/new.go", "/new.pk")

		assert.Equal(t, "new/pkg", child.CurrentGoFullPackagePath)
		assert.Equal(t, "new", child.CurrentGoPackageName)
		assert.Equal(t, "/new.go", child.CurrentGoSourcePath)
		assert.Equal(t, "/new.pk", child.SFCSourcePath)
	})

	t.Run("inherits diagnostics pointer", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		parent := NewRootAnalysisContext(diagnostics, "", "", "", "")

		child := parent.ForNewPackageContext("", "", "", "")

		assert.Same(t, diagnostics, child.Diagnostics)
	})

	t.Run("inherits translation keys", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		parent := NewRootAnalysisContext(diagnostics, "", "", "", "")
		keys := &TranslationKeySet{GlobalKeys: map[string]struct{}{"key": {}}}
		parent.TranslationKeys = keys

		child := parent.ForNewPackageContext("", "", "", "")

		assert.Same(t, keys, child.TranslationKeys)
	})

	t.Run("creates child symbol table linked to parent", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		parent := NewRootAnalysisContext(diagnostics, "", "", "", "")
		parent.Symbols.Define(Symbol{Name: "parentSym"})

		child := parent.ForNewPackageContext("", "", "", "")

		_, found := child.Symbols.Find("parentSym")
		assert.True(t, found)
	})
}

func TestAnalysisContext_SetTranslationKeys(t *testing.T) {
	t.Parallel()

	t.Run("sets translation keys", func(t *testing.T) {
		t.Parallel()
		ctx := &AnalysisContext{}
		keys := &TranslationKeySet{
			LocalKeys:  map[string]struct{}{"local": {}},
			GlobalKeys: map[string]struct{}{"global": {}},
		}

		ctx.SetTranslationKeys(keys)

		assert.Same(t, keys, ctx.TranslationKeys)
	})

	t.Run("can set nil keys", func(t *testing.T) {
		t.Parallel()
		ctx := &AnalysisContext{
			TranslationKeys: &TranslationKeySet{},
		}

		ctx.SetTranslationKeys(nil)

		assert.Nil(t, ctx.TranslationKeys)
	})
}

func TestAnalysisContext_SetLocalTranslationKeys(t *testing.T) {
	t.Parallel()

	t.Run("creates TranslationKeySet if nil", func(t *testing.T) {
		t.Parallel()
		ctx := &AnalysisContext{TranslationKeys: nil}
		localKeys := map[string]struct{}{"key1": {}}

		ctx.SetLocalTranslationKeys(localKeys)

		require.NotNil(t, ctx.TranslationKeys)
		assert.Equal(t, localKeys, ctx.TranslationKeys.LocalKeys)
	})

	t.Run("updates existing LocalKeys", func(t *testing.T) {
		t.Parallel()
		ctx := &AnalysisContext{
			TranslationKeys: &TranslationKeySet{
				LocalKeys:  map[string]struct{}{"old": {}},
				GlobalKeys: map[string]struct{}{"global": {}},
			},
		}
		newKeys := map[string]struct{}{"new": {}}

		ctx.SetLocalTranslationKeys(newKeys)

		assert.Equal(t, newKeys, ctx.TranslationKeys.LocalKeys)

		assert.Equal(t, map[string]struct{}{"global": {}}, ctx.TranslationKeys.GlobalKeys)
	})
}

func TestAnalysisContext_WithSymbol(t *testing.T) {
	t.Parallel()

	t.Run("adds symbol with simple type", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		result := ctx.WithSymbol("myVar", goast.NewIdent("string"))

		assert.Same(t, ctx, result, "should return same context for chaining")
		sym, found := ctx.Symbols.Find("myVar")
		require.True(t, found)
		assert.Equal(t, "myVar", sym.Name)
		assert.Equal(t, "myVar", sym.CodeGenVarName)
		require.NotNil(t, sym.TypeInfo)
	})

	t.Run("enables method chaining", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		ctx.WithSymbol("a", goast.NewIdent("int")).
			WithSymbol("b", goast.NewIdent("string")).
			WithSymbol("c", goast.NewIdent("bool"))

		_, foundA := ctx.Symbols.Find("a")
		_, foundB := ctx.Symbols.Find("b")
		_, foundC := ctx.Symbols.Find("c")

		assert.True(t, foundA)
		assert.True(t, foundB)
		assert.True(t, foundC)
	})

	t.Run("symbol has correct CodeGenVarName matching name", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		ctx.WithSymbol("userName", goast.NewIdent("string"))

		sym, _ := ctx.Symbols.Find("userName")
		assert.Equal(t, "userName", sym.CodeGenVarName)
	})
}

func TestAnalysisContext_WithTypedSymbol(t *testing.T) {
	t.Parallel()

	t.Run("adds fully specified symbol", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		sym := Symbol{
			Name:                "item",
			CodeGenVarName:      "loopItem",
			TypeInfo:            &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			SourceInvocationKey: "inv_abc",
		}

		result := ctx.WithTypedSymbol(sym)

		assert.Same(t, ctx, result, "should return same context for chaining")
		found, ok := ctx.Symbols.Find("item")
		require.True(t, ok)
		assert.Equal(t, "loopItem", found.CodeGenVarName)
		assert.Equal(t, "inv_abc", found.SourceInvocationKey)
	})

	t.Run("enables method chaining with mixed methods", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		ctx.WithSymbol("simple", goast.NewIdent("string")).
			WithTypedSymbol(Symbol{Name: "complex", CodeGenVarName: "cplx"})

		_, foundSimple := ctx.Symbols.Find("simple")
		found, foundComplex := ctx.Symbols.Find("complex")

		assert.True(t, foundSimple)
		assert.True(t, foundComplex)
		assert.Equal(t, "cplx", found.CodeGenVarName)
	})
}

func TestAnalysisContext_addDiagnosticWithPath(t *testing.T) {
	t.Parallel()

	t.Run("adds diagnostic with custom path", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "/default.pk")

		ctx.addDiagnosticWithPath(
			ast_domain.Error,
			"test error",
			"testExpr",
			ast_domain.Location{Line: 10, Column: 5},
			"/custom/path.pk",
			"",
		)

		require.Len(t, *diagnostics, 1)
		assert.Equal(t, "test error", (*diagnostics)[0].Message)
		assert.Equal(t, "/custom/path.pk", (*diagnostics)[0].SourcePath)
	})

	t.Run("uses default path when path is empty", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "/default.pk")

		ctx.addDiagnosticWithPath(
			ast_domain.Warning,
			"test warning",
			"expr",
			ast_domain.Location{},
			"",
			"",
		)

		require.Len(t, *diagnostics, 1)
		assert.Equal(t, "/default.pk", (*diagnostics)[0].SourcePath)
	})
}

func TestAnalysisContext_addDiagnostic(t *testing.T) {
	t.Parallel()

	t.Run("uses SFCSourcePath when annotations is nil", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "/default.pk")

		ctx.addDiagnostic(
			ast_domain.Error,
			"test error",
			"expr",
			ast_domain.Location{Line: 1, Column: 1},
			nil,
			"",
		)

		require.Len(t, *diagnostics, 1)
		assert.Equal(t, "/default.pk", (*diagnostics)[0].SourcePath)
	})

	t.Run("uses SFCSourcePath when OriginalSourcePath is nil", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "/default.pk")
		ann := &ast_domain.GoGeneratorAnnotation{OriginalSourcePath: nil}

		ctx.addDiagnostic(
			ast_domain.Warning,
			"test warning",
			"expr",
			ast_domain.Location{Line: 1, Column: 1},
			ann,
			"",
		)

		require.Len(t, *diagnostics, 1)
		assert.Equal(t, "/default.pk", (*diagnostics)[0].SourcePath)
	})

	t.Run("uses OriginalSourcePath when provided", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "/default.pk")
		ann := &ast_domain.GoGeneratorAnnotation{OriginalSourcePath: new("/custom/source.pk")}

		ctx.addDiagnostic(
			ast_domain.Error,
			"test error",
			"expr",
			ast_domain.Location{Line: 5, Column: 10},
			ann,
			"",
		)

		require.Len(t, *diagnostics, 1)
		assert.Equal(t, "/custom/source.pk", (*diagnostics)[0].SourcePath)
	})
}

func TestAnalysisContext_addDiagnosticForExpression(t *testing.T) {
	t.Parallel()

	t.Run("uses SFCSourcePath when annotations is nil", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "/default.pk")
		expression := &ast_domain.Identifier{Name: "testExpr"}

		ctx.addDiagnosticForExpression(
			ast_domain.Error,
			"test error",
			expression,
			ast_domain.Location{Line: 1, Column: 1},
			nil,
			"",
		)

		require.Len(t, *diagnostics, 1)
		assert.Equal(t, "/default.pk", (*diagnostics)[0].SourcePath)
	})

	t.Run("uses OriginalSourcePath when provided", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "/default.pk")
		ann := &ast_domain.GoGeneratorAnnotation{OriginalSourcePath: new("/custom/source.pk")}
		expression := &ast_domain.Identifier{Name: "testExpr"}

		ctx.addDiagnosticForExpression(
			ast_domain.Warning,
			"test warning",
			expression,
			ast_domain.Location{Line: 3, Column: 5},
			ann,
			"",
		)

		require.Len(t, *diagnostics, 1)
		assert.Equal(t, "/custom/source.pk", (*diagnostics)[0].SourcePath)
	})
}

func TestSymbol_Fields(t *testing.T) {
	t.Parallel()

	t.Run("all fields are accessible", func(t *testing.T) {
		t.Parallel()
		typeInfo := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")}

		symbol := Symbol{
			Name:                "myVar",
			CodeGenVarName:      "generated_myVar",
			TypeInfo:            typeInfo,
			SourceInvocationKey: "inv_123",
		}

		assert.Equal(t, "myVar", symbol.Name)
		assert.Equal(t, "generated_myVar", symbol.CodeGenVarName)
		assert.Same(t, typeInfo, symbol.TypeInfo)
		assert.Equal(t, "inv_123", symbol.SourceInvocationKey)
	})

	t.Run("zero value is valid", func(t *testing.T) {
		t.Parallel()
		var symbol Symbol

		assert.Empty(t, symbol.Name)
		assert.Empty(t, symbol.CodeGenVarName)
		assert.Nil(t, symbol.TypeInfo)
		assert.Empty(t, symbol.SourceInvocationKey)
	})
}

func TestDefineExportedSymbol(t *testing.T) {
	t.Parallel()

	t.Run("defines exported symbol with type expression", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "github.com/example/pkg", "pkg", "/src/file.go", "/src/file.pk")

		defineExportedSymbol(ctx, "MyConst", goast.NewIdent("string"))

		sym, found := ctx.Symbols.Find("MyConst")
		require.True(t, found)
		assert.Equal(t, "MyConst", sym.Name)
		assert.Equal(t, "MyConst", sym.CodeGenVarName)
		require.NotNil(t, sym.TypeInfo)
		assert.Equal(t, "pkg", sym.TypeInfo.PackageAlias)
		assert.Equal(t, "github.com/example/pkg", sym.TypeInfo.CanonicalPackagePath)
		assert.True(t, sym.TypeInfo.IsExportedPackageSymbol)
	})

	t.Run("ignores unexported symbol", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		defineExportedSymbol(ctx, "privateVar", goast.NewIdent("int"))

		_, found := ctx.Symbols.Find("privateVar")
		assert.False(t, found)
	})

	t.Run("uses any type when typeExpr is nil", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		defineExportedSymbol(ctx, "ExportedVar", nil)

		sym, found := ctx.Symbols.Find("ExportedVar")
		require.True(t, found)
		require.NotNil(t, sym.TypeInfo)
		identifier, ok := sym.TypeInfo.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "any", identifier.Name)
	})

	t.Run("symbol has empty SourceInvocationKey", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		defineExportedSymbol(ctx, "PublicConst", goast.NewIdent("bool"))

		sym, found := ctx.Symbols.Find("PublicConst")
		require.True(t, found)
		assert.Empty(t, sym.SourceInvocationKey)
	})

	t.Run("handles selector type expression", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")
		selectorExpr := &goast.SelectorExpr{
			X:   goast.NewIdent("time"),
			Sel: goast.NewIdent("Duration"),
		}

		defineExportedSymbol(ctx, "Timeout", selectorExpr)

		sym, found := ctx.Symbols.Find("Timeout")
		require.True(t, found)
		require.NotNil(t, sym.TypeInfo)
		assert.Equal(t, selectorExpr, sym.TypeInfo.TypeExpression)
	})
}

func TestProcessValueSpec(t *testing.T) {
	t.Parallel()

	t.Run("processes value spec with single exported name", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		spec := &goast.ValueSpec{
			Names: []*goast.Ident{goast.NewIdent("PublicVar")},
			Type:  goast.NewIdent("int"),
		}

		processValueSpec(ctx, spec)

		_, found := ctx.Symbols.Find("PublicVar")
		assert.True(t, found)
	})

	t.Run("processes value spec with multiple names", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		spec := &goast.ValueSpec{
			Names: []*goast.Ident{
				goast.NewIdent("First"),
				goast.NewIdent("Second"),
				goast.NewIdent("Third"),
			},
			Type: goast.NewIdent("string"),
		}

		processValueSpec(ctx, spec)

		_, foundFirst := ctx.Symbols.Find("First")
		_, foundSecond := ctx.Symbols.Find("Second")
		_, foundThird := ctx.Symbols.Find("Third")
		assert.True(t, foundFirst)
		assert.True(t, foundSecond)
		assert.True(t, foundThird)
	})

	t.Run("ignores unexported names", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		spec := &goast.ValueSpec{
			Names: []*goast.Ident{
				goast.NewIdent("privateVar"),
				goast.NewIdent("PublicVar"),
			},
			Type: goast.NewIdent("int"),
		}

		processValueSpec(ctx, spec)

		_, foundPrivate := ctx.Symbols.Find("privateVar")
		_, foundPublic := ctx.Symbols.Find("PublicVar")
		assert.False(t, foundPrivate)
		assert.True(t, foundPublic)
	})

	t.Run("ignores non-ValueSpec", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		spec := &goast.ImportSpec{
			Path: &goast.BasicLit{Value: `"fmt"`},
		}

		processValueSpec(ctx, spec)

		assert.Empty(t, ctx.Symbols.symbols)
	})
}

func TestProcessConstVarDecl(t *testing.T) {
	t.Parallel()

	t.Run("processes const declaration", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		declaration := &goast.GenDecl{
			Tok: token.CONST,
			Specs: []goast.Spec{
				&goast.ValueSpec{
					Names: []*goast.Ident{goast.NewIdent("MyConst")},
					Type:  goast.NewIdent("int"),
				},
			},
		}

		processConstVarDecl(ctx, declaration)

		_, found := ctx.Symbols.Find("MyConst")
		assert.True(t, found)
	})

	t.Run("processes var declaration", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		declaration := &goast.GenDecl{
			Tok: token.VAR,
			Specs: []goast.Spec{
				&goast.ValueSpec{
					Names: []*goast.Ident{goast.NewIdent("MyVar")},
					Type:  goast.NewIdent("string"),
				},
			},
		}

		processConstVarDecl(ctx, declaration)

		_, found := ctx.Symbols.Find("MyVar")
		assert.True(t, found)
	})

	t.Run("ignores import declaration", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		declaration := &goast.GenDecl{
			Tok: token.IMPORT,
			Specs: []goast.Spec{
				&goast.ImportSpec{
					Path: &goast.BasicLit{Value: `"fmt"`},
				},
			},
		}

		processConstVarDecl(ctx, declaration)

		assert.Empty(t, ctx.Symbols.symbols)
	})

	t.Run("ignores type declaration", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		declaration := &goast.GenDecl{
			Tok: token.TYPE,
			Specs: []goast.Spec{
				&goast.TypeSpec{
					Name: goast.NewIdent("MyType"),
					Type: goast.NewIdent("int"),
				},
			},
		}

		processConstVarDecl(ctx, declaration)

		assert.Empty(t, ctx.Symbols.symbols)
	})

	t.Run("ignores FuncDecl", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		declaration := &goast.FuncDecl{
			Name: goast.NewIdent("MyFunc"),
			Type: &goast.FuncType{},
		}

		processConstVarDecl(ctx, declaration)

		assert.Empty(t, ctx.Symbols.symbols)
	})

	t.Run("processes multiple specs in single declaration", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		declaration := &goast.GenDecl{
			Tok: token.CONST,
			Specs: []goast.Spec{
				&goast.ValueSpec{
					Names: []*goast.Ident{goast.NewIdent("ConstA")},
					Type:  goast.NewIdent("int"),
				},
				&goast.ValueSpec{
					Names: []*goast.Ident{goast.NewIdent("ConstB")},
					Type:  goast.NewIdent("int"),
				},
			},
		}

		processConstVarDecl(ctx, declaration)

		_, foundA := ctx.Symbols.Find("ConstA")
		_, foundB := ctx.Symbols.Find("ConstB")
		assert.True(t, foundA)
		assert.True(t, foundB)
	})
}

func TestInferDataType(t *testing.T) {
	t.Parallel()

	t.Run("returns map[string]interface{} type", func(t *testing.T) {
		t.Parallel()

		result := inferDataType(nil, nil)

		require.NotNil(t, result)
		mapType, ok := result.TypeExpression.(*goast.MapType)
		require.True(t, ok, "expected MapType")

		keyIdent, ok := mapType.Key.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", keyIdent.Name)

		valueIdent, ok := mapType.Value.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "interface{}", valueIdent.Name)
	})

	t.Run("returns non-synthetic type", func(t *testing.T) {
		t.Parallel()

		result := inferDataType(nil, nil)

		assert.False(t, result.IsSynthetic)
		assert.False(t, result.IsExportedPackageSymbol)
	})

	t.Run("returns type with empty package paths", func(t *testing.T) {
		t.Parallel()

		result := inferDataType(nil, nil)

		assert.Empty(t, result.PackageAlias)
		assert.Empty(t, result.CanonicalPackagePath)
		assert.Empty(t, result.InitialPackagePath)
		assert.Empty(t, result.InitialFilePath)
	})
}

func TestDefineAndValidateLocalFunctions(t *testing.T) {
	t.Parallel()

	t.Run("does nothing when script is nil", func(t *testing.T) {
		t.Parallel()

		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "pkg/test", "test", "/test.go", "/test.pk")

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				Script: nil,
			},
		}

		defineAndValidateLocalFunctions(ctx, vc)

		assert.Empty(t, ctx.Symbols.symbols)
		assert.Empty(t, *diagnostics)
	})

	t.Run("does nothing when AST is nil", func(t *testing.T) {
		t.Parallel()

		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "pkg/test", "test", "/test.go", "/test.pk")

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				Script: &annotator_dto.ParsedScript{
					AST: nil,
				},
			},
		}

		defineAndValidateLocalFunctions(ctx, vc)

		assert.Empty(t, ctx.Symbols.symbols)
		assert.Empty(t, *diagnostics)
	})

	t.Run("defines exported function", func(t *testing.T) {
		t.Parallel()

		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "pkg/test", "test", "/test.go", "/test.pk")

		fset := token.NewFileSet()
		fset.AddFile("test.go", 1, 100)

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test.pk",
				Script: &annotator_dto.ParsedScript{
					AST: &goast.File{
						Name: goast.NewIdent("test"),
						Decls: []goast.Decl{
							&goast.FuncDecl{
								Name: goast.NewIdent("MyHelper"),
								Type: &goast.FuncType{},
							},
						},
					},
					Fset:                fset,
					ScriptStartLocation: ast_domain.Location{Line: 1, Column: 1},
				},
			},
		}

		defineAndValidateLocalFunctions(ctx, vc)

		sym, found := ctx.Symbols.Find("MyHelper")
		require.True(t, found)
		assert.Equal(t, "MyHelper", sym.Name)
		assert.Equal(t, "MyHelper", sym.CodeGenVarName)
		assert.True(t, sym.TypeInfo.IsExportedPackageSymbol)
	})

	t.Run("skips unexported function", func(t *testing.T) {
		t.Parallel()

		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "pkg/test", "test", "/test.go", "/test.pk")

		fset := token.NewFileSet()
		fset.AddFile("test.go", 1, 100)

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test.pk",
				Script: &annotator_dto.ParsedScript{
					AST: &goast.File{
						Name: goast.NewIdent("test"),
						Decls: []goast.Decl{
							&goast.FuncDecl{
								Name: goast.NewIdent("privateFunc"),
								Type: &goast.FuncType{},
							},
						},
					},
					Fset:                fset,
					ScriptStartLocation: ast_domain.Location{Line: 1, Column: 1},
				},
			},
		}

		defineAndValidateLocalFunctions(ctx, vc)

		_, found := ctx.Symbols.Find("privateFunc")
		assert.False(t, found)
	})

	t.Run("skips method receivers", func(t *testing.T) {
		t.Parallel()

		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "pkg/test", "test", "/test.go", "/test.pk")

		fset := token.NewFileSet()
		fset.AddFile("test.go", 1, 100)

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test.pk",
				Script: &annotator_dto.ParsedScript{
					AST: &goast.File{
						Name: goast.NewIdent("test"),
						Decls: []goast.Decl{
							&goast.FuncDecl{
								Name: goast.NewIdent("Method"),
								Recv: &goast.FieldList{
									List: []*goast.Field{{Type: goast.NewIdent("MyType")}},
								},
								Type: &goast.FuncType{},
							},
						},
					},
					Fset:                fset,
					ScriptStartLocation: ast_domain.Location{Line: 1, Column: 1},
				},
			},
		}

		defineAndValidateLocalFunctions(ctx, vc)

		_, found := ctx.Symbols.Find("Method")
		assert.False(t, found)
	})
}

func TestDefineExportedConstantsAndVariables(t *testing.T) {
	t.Parallel()

	t.Run("does nothing when script is nil", func(t *testing.T) {
		t.Parallel()

		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{Script: nil},
		}

		defineExportedConstantsAndVariables(ctx, vc)

		assert.Empty(t, ctx.Symbols.symbols)
	})

	t.Run("does nothing when AST is nil", func(t *testing.T) {
		t.Parallel()

		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				Script: &annotator_dto.ParsedScript{AST: nil},
			},
		}

		defineExportedConstantsAndVariables(ctx, vc)

		assert.Empty(t, ctx.Symbols.symbols)
	})

	t.Run("defines exported constants from AST", func(t *testing.T) {
		t.Parallel()

		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "pkg/test", "test", "/test.go", "/test.pk")

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				Script: &annotator_dto.ParsedScript{
					AST: &goast.File{
						Name: goast.NewIdent("test"),
						Decls: []goast.Decl{
							&goast.GenDecl{
								Tok: token.CONST,
								Specs: []goast.Spec{
									&goast.ValueSpec{
										Names: []*goast.Ident{goast.NewIdent("MaxRetries")},
										Type:  goast.NewIdent("int"),
									},
								},
							},
						},
					},
				},
			},
		}

		defineExportedConstantsAndVariables(ctx, vc)

		sym, found := ctx.Symbols.Find("MaxRetries")
		require.True(t, found)
		assert.Equal(t, "MaxRetries", sym.Name)
		assert.True(t, sym.TypeInfo.IsExportedPackageSymbol)
	})

	t.Run("defines exported variables from AST", func(t *testing.T) {
		t.Parallel()

		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "pkg/test", "test", "/test.go", "/test.pk")

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				Script: &annotator_dto.ParsedScript{
					AST: &goast.File{
						Name: goast.NewIdent("test"),
						Decls: []goast.Decl{
							&goast.GenDecl{
								Tok: token.VAR,
								Specs: []goast.Spec{
									&goast.ValueSpec{
										Names: []*goast.Ident{goast.NewIdent("DefaultTimeout")},
										Type:  goast.NewIdent("string"),
									},
								},
							},
						},
					},
				},
			},
		}

		defineExportedConstantsAndVariables(ctx, vc)

		_, found := ctx.Symbols.Find("DefaultTimeout")
		assert.True(t, found)
	})
}

func TestDefineComponentSymbols(t *testing.T) {
	t.Parallel()

	t.Run("does nothing when script is nil", func(t *testing.T) {
		t.Parallel()

		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{Script: nil},
		}

		defineComponentSymbols(ctx, nil, vc, "pageData", "props", "")

		_, foundState := ctx.Symbols.Find("state")
		_, foundProps := ctx.Symbols.Find("props")
		assert.False(t, foundState)
		assert.False(t, foundProps)
	})

	t.Run("defines state symbol when render return type is primitive", func(t *testing.T) {
		t.Parallel()

		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				Script: &annotator_dto.ParsedScript{
					RenderReturnTypeExpression: goast.NewIdent("string"),
					PropsTypeExpression:        nil,
				},
			},
		}

		tr := &TypeResolver{inspector: nil}
		defineComponentSymbols(ctx, tr, vc, "pageData", "props", "")

		sym, found := ctx.Symbols.Find("state")
		require.True(t, found)
		assert.Equal(t, "pageData", sym.CodeGenVarName)

		symAlias, foundAlias := ctx.Symbols.Find("s")
		require.True(t, foundAlias)
		assert.Equal(t, "pageData", symAlias.CodeGenVarName)
	})

	t.Run("defines props symbol when props type is primitive", func(t *testing.T) {
		t.Parallel()

		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				Script: &annotator_dto.ParsedScript{
					RenderReturnTypeExpression: nil,
					PropsTypeExpression:        goast.NewIdent("int"),
				},
			},
		}

		tr := &TypeResolver{inspector: nil}
		defineComponentSymbols(ctx, tr, vc, "pageData", "props", "")

		sym, found := ctx.Symbols.Find("props")
		require.True(t, found)
		assert.Equal(t, "props", sym.CodeGenVarName)

		symAlias, foundAlias := ctx.Symbols.Find("p")
		require.True(t, foundAlias)
		assert.Equal(t, "props", symAlias.CodeGenVarName)
	})

	t.Run("sets source invocation key for partial context", func(t *testing.T) {
		t.Parallel()

		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "", "", "", "")

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				Script: &annotator_dto.ParsedScript{
					RenderReturnTypeExpression: goast.NewIdent("bool"),
				},
			},
		}

		tr := &TypeResolver{inspector: nil}
		defineComponentSymbols(ctx, tr, vc, "cardData_inv1", "props_inv1", "inv1")

		sym, found := ctx.Symbols.Find("state")
		require.True(t, found)
		assert.Equal(t, "cardData_inv1", sym.CodeGenVarName)
		assert.Equal(t, "inv1", sym.SourceInvocationKey)
	})
}

func TestPopulateRootContext(t *testing.T) {
	t.Parallel()

	t.Run("populates context with global symbols for page without collection", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath:    "/pages/home.pk",
				HasCollection: false,
				Script:        nil,
			},
		}

		PopulateRootContext(h.Context, h.Resolver, vc)

		_, foundReq := h.Context.Symbols.Find("request")
		assert.True(t, foundReq, "should have 'request' symbol")

		_, foundR := h.Context.Symbols.Find("r")
		assert.True(t, foundR, "should have 'r' alias")

		_, foundLen := h.Context.Symbols.Find("len")
		assert.True(t, foundLen, "should have 'len' built-in")

		_, foundT := h.Context.Symbols.Find("T")
		assert.True(t, foundT, "should have 'T' translation function")

		_, foundData := h.Context.Symbols.Find("data")
		assert.False(t, foundData, "should not have 'data' symbol without collection")
	})

	t.Run("populates context with data symbol for collection page", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath:    "/pages/blog.pk",
				HasCollection: true,
				Script:        nil,
			},
		}

		PopulateRootContext(h.Context, h.Resolver, vc)

		dataSym, foundData := h.Context.Symbols.Find("data")
		require.True(t, foundData, "should have 'data' symbol for collection page")
		assert.Equal(t, "data", dataSym.Name)
		assert.Equal(t, "data", dataSym.CodeGenVarName)
		require.NotNil(t, dataSym.TypeInfo)
		_, isMapType := dataSym.TypeInfo.TypeExpression.(*goast.MapType)
		assert.True(t, isMapType, "data type should be a map type")
	})

	t.Run("populates state and props symbols when script has types", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath:    "/pages/page.pk",
				HasCollection: false,
				Script: &annotator_dto.ParsedScript{
					RenderReturnTypeExpression: goast.NewIdent("PageState"),
					PropsTypeExpression:        goast.NewIdent("PageProps"),
				},
			},
		}

		PopulateRootContext(h.Context, h.Resolver, vc)

		stateSym, foundState := h.Context.Symbols.Find("state")
		require.True(t, foundState, "should have 'state' symbol")
		assert.Equal(t, "pageData", stateSym.CodeGenVarName)

		propsSym, foundProps := h.Context.Symbols.Find("props")
		require.True(t, foundProps, "should have 'props' symbol")
		assert.Equal(t, "props", propsSym.CodeGenVarName)

		_, foundS := h.Context.Symbols.Find("s")
		assert.True(t, foundS, "should have 's' alias for state")

		_, foundP := h.Context.Symbols.Find("p")
		assert.True(t, foundP, "should have 'p' alias for props")
	})
}

func TestDefineAndValidateLocalFunctions_ShadowWarnings(t *testing.T) {
	t.Parallel()

	t.Run("warns when exported function shadows reserved system symbol", func(t *testing.T) {
		t.Parallel()

		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "pkg/test", "test", "/test.go", "/test.pk")

		fset := token.NewFileSet()
		fset.AddFile("test.go", 1, 200)

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test.pk",
				Script: &annotator_dto.ParsedScript{
					AST: &goast.File{
						Name: goast.NewIdent("test"),
						Decls: []goast.Decl{
							&goast.FuncDecl{
								Name: goast.NewIdent("T"),
								Type: &goast.FuncType{},
							},
						},
					},
					Fset:                fset,
					ScriptStartLocation: ast_domain.Location{Line: 1, Column: 1},
				},
			},
		}

		defineAndValidateLocalFunctions(ctx, vc)

		require.NotEmpty(t, *diagnostics, "should have diagnostics for shadowing")
		assert.Contains(t, (*diagnostics)[0].Message, "shadows a built-in Piko system symbol")
		assert.Equal(t, ast_domain.Warning, (*diagnostics)[0].Severity)
	})

	t.Run("defines function symbol even when it shadows reserved name", func(t *testing.T) {
		t.Parallel()

		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "pkg/test", "test", "/test.go", "/test.pk")

		fset := token.NewFileSet()
		fset.AddFile("test.go", 1, 200)

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test.pk",
				Script: &annotator_dto.ParsedScript{
					AST: &goast.File{
						Name: goast.NewIdent("test"),
						Decls: []goast.Decl{
							&goast.FuncDecl{
								Name: goast.NewIdent("T"),
								Type: &goast.FuncType{},
							},
						},
					},
					Fset:                fset,
					ScriptStartLocation: ast_domain.Location{Line: 1, Column: 1},
				},
			},
		}

		defineAndValidateLocalFunctions(ctx, vc)

		sym, found := ctx.Symbols.Find("T")
		require.True(t, found, "should still define the symbol")
		assert.Equal(t, "T", sym.Name)
	})

	t.Run("defines multiple exported functions and skips unexported and methods", func(t *testing.T) {
		t.Parallel()

		diagnostics := &[]*ast_domain.Diagnostic{}

		ctx := NewRootAnalysisContext(diagnostics, "pkg/test", "test", "/test.go", "/test.pk")

		fset := token.NewFileSet()
		fset.AddFile("test.go", 1, 500)

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test.pk",
				Script: &annotator_dto.ParsedScript{
					AST: &goast.File{
						Name: goast.NewIdent("test"),
						Decls: []goast.Decl{
							&goast.FuncDecl{
								Name: goast.NewIdent("HelperOne"),
								Type: &goast.FuncType{},
							},
							&goast.FuncDecl{
								Name: goast.NewIdent("HelperTwo"),
								Type: &goast.FuncType{},
							},
							&goast.FuncDecl{
								Name: goast.NewIdent("privateHelper"),
								Type: &goast.FuncType{},
							},
							&goast.FuncDecl{
								Name: goast.NewIdent("MyMethod"),
								Recv: &goast.FieldList{
									List: []*goast.Field{{Type: goast.NewIdent("SomeType")}},
								},
								Type: &goast.FuncType{},
							},
							&goast.GenDecl{
								Tok: token.CONST,
								Specs: []goast.Spec{
									&goast.ValueSpec{
										Names: []*goast.Ident{goast.NewIdent("ExportedConst")},
										Type:  goast.NewIdent("int"),
									},
								},
							},
						},
					},
					Fset:                fset,
					ScriptStartLocation: ast_domain.Location{Line: 1, Column: 1},
				},
			},
		}

		defineAndValidateLocalFunctions(ctx, vc)

		_, foundH1 := ctx.Symbols.Find("HelperOne")
		assert.True(t, foundH1, "should define exported HelperOne")

		_, foundH2 := ctx.Symbols.Find("HelperTwo")
		assert.True(t, foundH2, "should define exported HelperTwo")

		_, foundPrivate := ctx.Symbols.Find("privateHelper")
		assert.False(t, foundPrivate, "should not define unexported function")

		_, foundMethod := ctx.Symbols.Find("MyMethod")
		assert.False(t, foundMethod, "should not define method receiver")

		_, foundConst := ctx.Symbols.Find("ExportedConst")
		assert.True(t, foundConst, "should define exported constant")
	})
}
