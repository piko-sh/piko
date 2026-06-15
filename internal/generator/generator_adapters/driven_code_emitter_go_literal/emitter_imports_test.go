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

package driven_code_emitter_go_literal

import (
	goast "go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

func TestAddUserScriptImports(t *testing.T) {
	t.Parallel()

	t.Run("adds imports from user script", func(t *testing.T) {
		t.Parallel()

		importSet := make(map[string]goast.Spec)

		mainComponent := &annotator_dto.VirtualComponent{
			RewrittenScriptAST: &goast.File{
				Decls: []goast.Decl{
					&goast.GenDecl{
						Tok: token.IMPORT,
						Specs: []goast.Spec{
							&goast.ImportSpec{
								Path: &goast.BasicLit{Value: `"fmt"`},
							},
							&goast.ImportSpec{
								Name: goast.NewIdent("sql"),
								Path: &goast.BasicLit{Value: `"database/sql"`},
							},
						},
					},
				},
			},
		}

		addUserScriptImports(importSet, mainComponent)

		assert.Len(t, importSet, 2, "Should add both imports")

		fmtSpec, hasFmt := importSet["fmt"]
		assert.True(t, hasFmt, "Should have fmt import")
		assert.NotNil(t, fmtSpec)

		sqlSpec, hasSQL := importSet["database/sql"]
		assert.True(t, hasSQL, "Should have database/sql import")
		sqlImport := requireAstImportSpec(t, sqlSpec, "database/sql import")
		assert.Equal(t, "sql", sqlImport.Name.Name, "Should preserve import alias")
	})

	t.Run("handles nil script AST", func(t *testing.T) {
		t.Parallel()

		importSet := make(map[string]goast.Spec)

		mainComponent := &annotator_dto.VirtualComponent{
			RewrittenScriptAST: nil,
		}

		addUserScriptImports(importSet, mainComponent)

		assert.Empty(t, importSet, "Should not add imports when script is nil")
	})

	t.Run("handles nil component", func(t *testing.T) {
		t.Parallel()

		importSet := make(map[string]goast.Spec)

		addUserScriptImports(importSet, nil)

		assert.Empty(t, importSet, "Should handle nil component gracefully")
	})

	t.Run("skips non-import declarations", func(t *testing.T) {
		t.Parallel()

		importSet := make(map[string]goast.Spec)

		mainComponent := &annotator_dto.VirtualComponent{
			RewrittenScriptAST: &goast.File{
				Decls: []goast.Decl{

					&goast.GenDecl{
						Tok: token.VAR,
						Specs: []goast.Spec{
							&goast.ValueSpec{
								Names: []*goast.Ident{goast.NewIdent("x")},
							},
						},
					},

					&goast.FuncDecl{
						Name: goast.NewIdent("MyFunc"),
					},
				},
			},
		}

		addUserScriptImports(importSet, mainComponent)

		assert.Empty(t, importSet, "Should not add non-import declarations")
	})
}

func TestAddPartialImports(t *testing.T) {
	t.Parallel()

	t.Run("adds imports for partial invocations", func(t *testing.T) {
		t.Parallel()

		importSet := make(map[string]goast.Spec)

		result := &annotator_dto.AnnotationResult{
			UniqueInvocations: []*annotator_dto.PartialInvocation{
				{
					PartialHashedName: "c_header",
				},
				{
					PartialHashedName: "c_footer",
				},
			},
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"c_header": {
						HashedName:             "c_header",
						CanonicalGoPackagePath: "github.com/user/app/partials/header",
					},
					"c_footer": {
						HashedName:             "c_footer",
						CanonicalGoPackagePath: "github.com/user/app/partials/footer",
					},
				},
			},
		}

		currentComponentHash := "c_home"

		addPartialImports(importSet, result, currentComponentHash)

		assert.Len(t, importSet, 2, "Should add imports for both partials")

		headerSpec, hasHeader := importSet["github.com/user/app/partials/header"]
		assert.True(t, hasHeader, "Should have header import")
		headerImport := requireAstImportSpec(t, headerSpec, "header import")
		assert.Equal(t, "c_header", headerImport.Name.Name, "Should use hashed name as alias")

		footerSpec, hasFooter := importSet["github.com/user/app/partials/footer"]
		assert.True(t, hasFooter, "Should have footer import")
		footerImport := requireAstImportSpec(t, footerSpec, "footer import")
		assert.Equal(t, "c_footer", footerImport.Name.Name, "Should use hashed name as alias")
	})

	t.Run("skips self-referencing component", func(t *testing.T) {
		t.Parallel()

		importSet := make(map[string]goast.Spec)

		result := &annotator_dto.AnnotationResult{
			UniqueInvocations: []*annotator_dto.PartialInvocation{
				{
					PartialHashedName: "c_home",
				},
				{
					PartialHashedName: "c_header",
				},
			},
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"c_home": {
						HashedName:             "c_home",
						CanonicalGoPackagePath: "github.com/user/app/home",
					},
					"c_header": {
						HashedName:             "c_header",
						CanonicalGoPackagePath: "github.com/user/app/partials/header",
					},
				},
			},
		}

		currentComponentHash := "c_home"

		addPartialImports(importSet, result, currentComponentHash)

		assert.Len(t, importSet, 1, "Should only add non-self imports")

		_, hasSelf := importSet["github.com/user/app/home"]
		assert.False(t, hasSelf, "Should not import itself")

		_, hasHeader := importSet["github.com/user/app/partials/header"]
		assert.True(t, hasHeader, "Should import other partials")
	})

	t.Run("handles empty invocations list", func(t *testing.T) {
		t.Parallel()

		importSet := make(map[string]goast.Spec)

		result := &annotator_dto.AnnotationResult{
			UniqueInvocations: []*annotator_dto.PartialInvocation{},
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: make(map[string]*annotator_dto.VirtualComponent),
			},
		}

		addPartialImports(importSet, result, "c_home")

		assert.Empty(t, importSet, "Should not add imports when no invocations")
	})
}

func TestCopyUserCode(t *testing.T) {
	t.Parallel()

	t.Run("copies non-import declarations", func(t *testing.T) {
		t.Parallel()

		fileAST := &goast.File{
			Decls: []goast.Decl{},
		}

		mainComponent := &annotator_dto.VirtualComponent{
			RewrittenScriptAST: &goast.File{
				Decls: []goast.Decl{

					&goast.GenDecl{
						Tok: token.IMPORT,
						Specs: []goast.Spec{
							&goast.ImportSpec{Path: &goast.BasicLit{Value: `"fmt"`}},
						},
					},

					&goast.FuncDecl{
						Name: goast.NewIdent("calculateTotal"),
					},

					&goast.GenDecl{
						Tok: token.TYPE,
						Specs: []goast.Spec{
							&goast.TypeSpec{Name: goast.NewIdent("MyType")},
						},
					},
				},
			},
		}

		copyUserCode(fileAST, mainComponent, nil)

		assert.Len(t, fileAST.Decls, 2, "Should copy 2 non-import declarations")

		funcDecl, ok := fileAST.Decls[0].(*goast.FuncDecl)
		assert.True(t, ok, "First declaration should be function")
		assert.Equal(t, "calculateTotal", funcDecl.Name.Name)

		typeDecl, ok := fileAST.Decls[1].(*goast.GenDecl)
		assert.True(t, ok, "Second declaration should be type")
		assert.Equal(t, token.TYPE, typeDecl.Tok)
	})

	t.Run("handles nil script AST", func(t *testing.T) {
		t.Parallel()

		fileAST := &goast.File{
			Decls: []goast.Decl{},
		}

		mainComponent := &annotator_dto.VirtualComponent{
			RewrittenScriptAST: nil,
		}

		copyUserCode(fileAST, mainComponent, nil)

		assert.Empty(t, fileAST.Decls, "Should not add declarations when script is nil")
	})

	t.Run("handles nil component", func(t *testing.T) {
		t.Parallel()

		fileAST := &goast.File{
			Decls: []goast.Decl{},
		}

		copyUserCode(fileAST, nil, nil)

		assert.Empty(t, fileAST.Decls, "Should handle nil component gracefully")
	})
}

func fileASTUsing(qualifiers ...string) *goast.File {
	decls := buildBoilerplateVarAcks()
	if len(qualifiers) > 0 {
		stmts := make([]goast.Stmt, 0, len(qualifiers))
		for _, q := range qualifiers {
			stmts = append(stmts, &goast.ExprStmt{
				X: &goast.SelectorExpr{X: goast.NewIdent(q), Sel: goast.NewIdent("Ref")},
			})
		}
		decls = append(decls, &goast.FuncDecl{
			Name: goast.NewIdent("_uses"),
			Type: &goast.FuncType{Params: &goast.FieldList{}},
			Body: &goast.BlockStmt{List: stmts},
		})
	}
	return &goast.File{Name: goast.NewIdent("p"), Decls: decls}
}

func importSpecsForPath(decl *goast.GenDecl, path string) []*goast.ImportSpec {
	var out []*goast.ImportSpec
	for _, s := range decl.Specs {
		if imp, ok := s.(*goast.ImportSpec); ok && importSpecPath(imp) == path {
			out = append(out, imp)
		}
	}
	return out
}

func TestCollectUsedQualifiers(t *testing.T) {
	t.Parallel()

	file := &goast.File{
		Name: goast.NewIdent("p"),
		Decls: []goast.Decl{
			&goast.FuncDecl{
				Name: goast.NewIdent("f"),
				Type: &goast.FuncType{Params: &goast.FieldList{}},
				Body: &goast.BlockStmt{List: []goast.Stmt{
					&goast.ExprStmt{X: &goast.SelectorExpr{X: goast.NewIdent("a"), Sel: goast.NewIdent("B")}},
					&goast.ExprStmt{X: &goast.SelectorExpr{
						X:   &goast.SelectorExpr{X: goast.NewIdent("c"), Sel: goast.NewIdent("D")},
						Sel: goast.NewIdent("E"),
					}},
				}},
			},
		},
	}

	used := collectUsedQualifiers(file)
	assert.Contains(t, used, "a")
	assert.Contains(t, used, "c")
	assert.NotContains(t, used, "z")
}

func TestBuildImportBlock_PrunesUnused(t *testing.T) {
	t.Parallel()

	em := &emitter{
		ctx:    NewEmitterContext(),
		config: EmitterConfig{CanonicalGoPackagePath: "test.com/pkg"},
	}
	em.ctx.requiredImports["example.com/used"] = "used"
	em.ctx.requiredImports["example.com/unused"] = "unused"

	result := createMinimalAnnotationResult("test")
	mainComponent := result.VirtualModule.ComponentsByHash["test"]

	decl := em.buildImportBlock(result, mainComponent, fileASTUsing("used"))

	assert.Len(t, importSpecsForPath(decl, "example.com/used"), 1, "referenced import kept")
	assert.Empty(t, importSpecsForPath(decl, "example.com/unused"), "unreferenced import pruned")

	for _, s := range decl.Specs {
		imp := requireAstImportSpec(t, s, "import spec")
		assert.NotEqual(t, ".", importSpecName(imp), "no dot import should be emitted")
	}
}

func TestBuildImportBlock_DualPath(t *testing.T) {
	t.Parallel()

	const path = "example.com/x/dto"

	newEmitter := func() *emitter {
		em := &emitter{
			ctx:    NewEmitterContext(),
			config: EmitterConfig{CanonicalGoPackagePath: "test.com/pkg"},
		}

		em.ctx.requiredImports[path] = "dto"
		return em
	}
	mainComponent := func() *annotator_dto.VirtualComponent {
		return &annotator_dto.VirtualComponent{
			HashedName: "test",
			RewrittenScriptAST: &goast.File{Decls: []goast.Decl{
				&goast.GenDecl{Tok: token.IMPORT, Specs: []goast.Spec{
					&goast.ImportSpec{Name: goast.NewIdent("tdto"), Path: &goast.BasicLit{Value: `"` + path + `"`}},
				}},
			}},
		}
	}
	result := createMinimalAnnotationResult("test")

	t.Run("both qualifiers referenced emits two specs", func(t *testing.T) {
		t.Parallel()
		decl := newEmitter().buildImportBlock(result, mainComponent(), fileASTUsing("tdto", "dto"))

		specs := importSpecsForPath(decl, path)
		assert.Len(t, specs, 2, "both the user alias and the canonical-name spec are emitted")

		var hasAliased, hasUnaliased bool
		for _, s := range specs {
			switch importSpecName(s) {
			case "tdto":
				hasAliased = true
			case "":
				hasUnaliased = true
			}
		}
		assert.True(t, hasAliased, "user alias spec present")
		assert.True(t, hasUnaliased, "canonical-name spec present and unaliased")
	})

	t.Run("only user alias referenced emits one spec", func(t *testing.T) {
		t.Parallel()
		decl := newEmitter().buildImportBlock(result, mainComponent(), fileASTUsing("tdto"))

		specs := importSpecsForPath(decl, path)
		assert.Len(t, specs, 1, "the unreferenced canonical-name spec is pruned")
		assert.Equal(t, "tdto", importSpecName(specs[0]))
	})
}

func TestInsertImportGroupBlankLine(t *testing.T) {
	t.Parallel()

	t.Run("separates stdlib and third-party groups", func(t *testing.T) {
		t.Parallel()
		src := "package p\n\nimport (\n\t\"fmt\"\n\t\"example.com/x\"\n)\n"
		want := "package p\n\nimport (\n\t\"fmt\"\n\n\t\"example.com/x\"\n)\n"
		assert.Equal(t, want, string(insertImportGroupBlankLine([]byte(src))))
	})

	t.Run("no blank line within a single group", func(t *testing.T) {
		t.Parallel()
		src := "package p\n\nimport (\n\t\"fmt\"\n\t\"sort\"\n)\n"
		assert.Equal(t, src, string(insertImportGroupBlankLine([]byte(src))))
	})

	t.Run("does not corrupt a user raw string that mimics an import block", func(t *testing.T) {
		t.Parallel()

		rawString := "`import (\n\"fmt\"\n\"example.com/x\"\n)`"
		src := "package p\n\nimport (\n\t\"fmt\"\n\t\"example.com/x\"\n)\n\nconst tmpl = " + rawString + "\n"
		want := "package p\n\nimport (\n\t\"fmt\"\n\n\t\"example.com/x\"\n)\n\nconst tmpl = " + rawString + "\n"
		got := string(insertImportGroupBlankLine([]byte(src)))
		assert.Equal(t, want, got)
		assert.Contains(t, got, rawString, "the user raw string must be left untouched")
	})
}
