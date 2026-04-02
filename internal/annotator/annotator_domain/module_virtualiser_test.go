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
	"context"
	"errors"
	"fmt"
	goast "go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

func newMockResolver(moduleName, baseDir string) *resolver_domain.MockResolver {
	return &resolver_domain.MockResolver{
		GetBaseDirFunc:    func() string { return baseDir },
		GetModuleNameFunc: func() string { return moduleName },
		ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
			if !filepath.IsAbs(importPath) {
				return filepath.Join(baseDir, importPath), nil
			}
			return importPath, nil
		},
		ResolveCSSPathFunc: func(_ context.Context, importPath string, containingDir string) (string, error) {
			if filepath.IsAbs(importPath) {
				return importPath, nil
			}
			return filepath.Join(containingDir, importPath), nil
		},
		ResolveAssetPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
			if filepath.IsAbs(importPath) {
				return importPath, nil
			}
			return filepath.Join(baseDir, importPath), nil
		},
		ConvertEntryPointPathToManifestKeyFunc: func(entryPointPath string) string {
			prefix := moduleName + "/"
			if result, found := strings.CutPrefix(entryPointPath, prefix); found {
				return result
			}
			return entryPointPath
		},
	}
}

type virtualiserTestHarness struct {
	t               *testing.T
	graph           *annotator_dto.ComponentGraph
	originalGoFiles map[string][]byte
	resolver        *resolver_domain.MockResolver
	pathsConfig     AnnotatorPathsConfig
}

func newVirtualiserTestHarness(t *testing.T, moduleName, baseDir string) *virtualiserTestHarness {
	t.Helper()
	return &virtualiserTestHarness{
		t: t,
		graph: &annotator_dto.ComponentGraph{
			Components:        make(map[string]*annotator_dto.ParsedComponent),
			PathToHashedName:  make(map[string]string),
			HashedNameToPath:  make(map[string]string),
			AllSourceContents: make(map[string][]byte),
		},
		originalGoFiles: make(map[string][]byte),
		resolver:        newMockResolver(moduleName, baseDir),
		pathsConfig:     AnnotatorPathsConfig{},
	}
}

func (h *virtualiserTestHarness) addComponent(absPath, scriptContent string, pikoImports ...annotator_dto.PikoImport) {
	h.t.Helper()
	hashedName := buildAliasFromPath(absPath)

	var parsedScript *annotator_dto.ParsedScript
	if scriptContent != "" {
		parsed, _, err := analyseGoScript(scriptContent, ast_domain.Location{Line: 1, Column: 1, Offset: 0})
		require.NoError(h.t, err)
		parsedScript = parsed
	}

	parsedComponent := &annotator_dto.ParsedComponent{
		SourcePath:  absPath,
		Script:      parsedScript,
		PikoImports: pikoImports,
	}

	h.graph.Components[hashedName] = parsedComponent
	h.graph.PathToHashedName[absPath] = hashedName
	h.graph.HashedNameToPath[hashedName] = absPath
}

func (h *virtualiserTestHarness) addGoFile(path, content string) {
	h.t.Helper()
	h.originalGoFiles[filepath.Join(h.resolver.GetBaseDir(), path)] = []byte(content)
}

func (h *virtualiserTestHarness) makeEntryPoints() []annotator_dto.EntryPoint {
	entryPoints := make([]annotator_dto.EntryPoint, 0, len(h.graph.Components))
	for absPath := range h.graph.PathToHashedName {

		entryPoints = append(entryPoints, annotator_dto.EntryPoint{
			Path:   absPath,
			IsPage: true,
		})
	}
	return entryPoints
}

func TestModuleVirtualiser(t *testing.T) {
	baseDir := "/project"
	moduleName := "my-module"

	t.Run("should correctly assemble overlay with real and virtual go files", func(t *testing.T) {
		h := newVirtualiserTestHarness(t, moduleName, baseDir)
		mainPath := filepath.Join(baseDir, "main.pk")
		modelsPath := filepath.Join(baseDir, "models", "user.go")

		h.addComponent(mainPath, `package main`)
		h.addGoFile(filepath.Join("models", "user.go"), `package models; type User struct{}`)

		virtualiser := NewModuleVirtualiser(h.resolver, h.pathsConfig)
		result, err := virtualiser.Virtualise(context.Background(), h.graph, h.originalGoFiles, h.makeEntryPoints())
		require.NoError(t, err)

		assert.Len(t, result.SourceOverlay, 2, "Overlay should contain one real and one virtual file")
		_, hasRealGoFile := result.SourceOverlay[modelsPath]
		assert.True(t, hasRealGoFile, "Original .go file should be in the overlay")

		mainHash := buildAliasFromPath(mainPath)
		virtualMainGoPath := filepath.Join(baseDir, config.CompiledPagesTargetDir, mainHash, "generated.go")
		_, hasVirtualGoFile := result.SourceOverlay[virtualMainGoPath]
		assert.True(t, hasVirtualGoFile, "Virtual .go file for component should be in the overlay")
	})

	t.Run("should handle components with no script block gracefully", func(t *testing.T) {
		h := newVirtualiserTestHarness(t, moduleName, baseDir)
		iconPath := filepath.Join(baseDir, "icon.pk")
		h.addComponent(iconPath, "")

		virtualiser := NewModuleVirtualiser(h.resolver, h.pathsConfig)
		result, err := virtualiser.Virtualise(context.Background(), h.graph, h.originalGoFiles, h.makeEntryPoints())
		require.NoError(t, err)

		assert.Len(t, result.SourceOverlay, 1, "Overlay should contain one virtual file with default boilerplate")
		assert.Len(t, result.ComponentsByHash, 1)
		iconComp := result.ComponentsByHash[buildAliasFromPath(iconPath)]
		assert.NotNil(t, iconComp.RewrittenScriptAST, "Component should have a RewrittenScriptAST with default boilerplate")
	})

	t.Run("should correctly calculate canonical paths", func(t *testing.T) {
		h := newVirtualiserTestHarness(t, moduleName, baseDir)
		rootPath := filepath.Join(baseDir, "root.pk")
		nestedPath := filepath.Join(baseDir, "components", "card.pk")

		h.addComponent(rootPath, `package root`)
		h.addComponent(nestedPath, `package card`)

		virtualiser := NewModuleVirtualiser(h.resolver, h.pathsConfig)
		result, err := virtualiser.Virtualise(context.Background(), h.graph, h.originalGoFiles, h.makeEntryPoints())
		require.NoError(t, err)

		rootHash := buildAliasFromPath(rootPath)
		rootComp := result.ComponentsByHash[rootHash]
		expectedRootGoPath := moduleName + "/" + config.CompiledPagesTargetDir + "/" + rootHash
		assert.Equal(t, expectedRootGoPath, rootComp.CanonicalGoPackagePath)
		assert.Equal(t, filepath.Join(baseDir, config.CompiledPagesTargetDir, rootHash, "generated.go"), rootComp.VirtualGoFilePath)

		nestedHash := buildAliasFromPath(nestedPath)
		nestedComp := result.ComponentsByHash[nestedHash]

		expectedNestedGoPath := moduleName + "/" + config.CompiledPagesTargetDir + "/" + nestedHash
		assert.Equal(t, expectedNestedGoPath, nestedComp.CanonicalGoPackagePath)
		assert.Equal(t, filepath.Join(baseDir, config.CompiledPagesTargetDir, nestedHash, "generated.go"), nestedComp.VirtualGoFilePath)
	})

	t.Run("should correctly rewrite AST with new package name and imports", func(t *testing.T) {

		h := newVirtualiserTestHarness(t, moduleName, baseDir)
		mainPath := filepath.Join(baseDir, "pages", "main.pk")
		cardPath := filepath.Join(baseDir, "components", "card.pk")

		h.addComponent(cardPath, `package card_original`)

		mainScriptContent := `
			package main_original
			import "fmt"
			import card "components/card.pk"
		`
		mainPikoImport := annotator_dto.PikoImport{
			Alias: "card",
			Path:  "components/card.pk",
		}
		h.addComponent(mainPath, mainScriptContent, mainPikoImport)

		virtualiser := NewModuleVirtualiser(h.resolver, h.pathsConfig)
		result, err := virtualiser.Virtualise(context.Background(), h.graph, h.originalGoFiles, h.makeEntryPoints())
		require.NoError(t, err)

		mainHash := buildAliasFromPath(mainPath)
		mainComp, ok := result.ComponentsByHash[mainHash]
		require.True(t, ok, "Main virtual component should exist in the result")
		require.NotNil(t, mainComp.RewrittenScriptAST, "Rewritten AST should not be nil")

		virtualMainGoPath := mainComp.VirtualGoFilePath
		mainSourceBytes, ok := result.SourceOverlay[virtualMainGoPath]
		require.True(t, ok, "Generated Go source should exist in the overlay")
		mainSource := string(mainSourceBytes)

		fset := token.NewFileSet()
		parsedFile, err := parser.ParseFile(fset, "", mainSource, parser.ImportsOnly)
		require.NoError(t, err, "The generated virtual Go source code should be syntactically valid")

		assert.Equal(t, mainHash, parsedFile.Name.Name, "Package name in the rewritten AST should match the component's hash")

		cardHash := buildAliasFromPath(cardPath)
		cardComp := result.ComponentsByHash[cardHash]

		expectedImports := map[string]string{
			cardComp.CanonicalGoPackagePath: cardHash,
			"piko.sh/piko":                  "piko",
		}
		assert.Len(t, parsedFile.Imports, len(expectedImports), "Should have the correct number of total imports")

		for _, imp := range parsedFile.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			expectedAlias, pathIsExpected := expectedImports[path]
			require.True(t, pathIsExpected, "Found an unexpected import path in the rewritten AST: %s", path)

			var actualAlias string
			if imp.Name != nil {
				actualAlias = imp.Name.Name
			}
			assert.Equal(t, expectedAlias, actualAlias, "Import alias for path '%s' is incorrect", path)

			delete(expectedImports, path)
		}

		assert.Empty(t, expectedImports, "Not all expected imports were found in the rewritten AST")
	})

	t.Run("should rewrite Piko import references to hashed names", func(t *testing.T) {
		h := newVirtualiserTestHarness(t, moduleName, baseDir)
		mainPath := filepath.Join(baseDir, "pages", "main.pk")
		cardPath := filepath.Join(baseDir, "partials", "card.pk")

		h.addComponent(cardPath, `
package card
func FormatPrice(price int) string { return "" }
`)

		mainScriptContent := `
package main
import card "partials/card.pk"
func Render() string { return card.FormatPrice(100) }
`
		mainPikoImport := annotator_dto.PikoImport{
			Alias: "card",
			Path:  "partials/card.pk",
		}
		h.addComponent(mainPath, mainScriptContent, mainPikoImport)

		virtualiser := NewModuleVirtualiser(h.resolver, h.pathsConfig)
		result, err := virtualiser.Virtualise(context.Background(), h.graph, h.originalGoFiles, h.makeEntryPoints())
		require.NoError(t, err)

		mainHash := buildAliasFromPath(mainPath)
		mainComp, ok := result.ComponentsByHash[mainHash]
		require.True(t, ok, "Main virtual component should exist in the result")
		require.NotNil(t, mainComp.RewrittenScriptAST, "Rewritten AST should not be nil")

		cardHash := buildAliasFromPath(cardPath)

		virtualMainGoPath := mainComp.VirtualGoFilePath
		mainSourceBytes, ok := result.SourceOverlay[virtualMainGoPath]
		require.True(t, ok, "Generated Go source should exist in the overlay")
		mainSource := string(mainSourceBytes)

		assert.Contains(t, mainSource, cardHash+".FormatPrice(100)",
			"Function call 'card.FormatPrice(100)' should be rewritten to use the hashed name '%s.FormatPrice(100)'", cardHash)
		assert.NotContains(t, mainSource, "card.FormatPrice(100)",
			"Original alias 'card.FormatPrice(100)' should not appear in the rewritten source")

		require.NotNil(t, mainComp.PikoAliasToHash, "PikoAliasToHash mapping should be populated")
		assert.Equal(t, cardHash, mainComp.PikoAliasToHash["card"],
			"PikoAliasToHash should map 'card' to the hashed name")
	})

	t.Run("should not rewrite shadowed local variables that match Piko import aliases", func(t *testing.T) {
		h := newVirtualiserTestHarness(t, moduleName, baseDir)
		mainPath := filepath.Join(baseDir, "pages", "main.pk")
		cardPath := filepath.Join(baseDir, "partials", "card.pk")

		h.addComponent(cardPath, `
package card
func FormatPrice(price int) string { return "" }
`)

		mainScriptContent := `
package main
import card "partials/card.pk"
func Render() string {
	card := "shadowed"
	_ = card
	return ""
}
func Other() string {
	return card.FormatPrice(100)
}
`
		mainPikoImport := annotator_dto.PikoImport{
			Alias: "card",
			Path:  "partials/card.pk",
		}
		h.addComponent(mainPath, mainScriptContent, mainPikoImport)

		virtualiser := NewModuleVirtualiser(h.resolver, h.pathsConfig)
		result, err := virtualiser.Virtualise(context.Background(), h.graph, h.originalGoFiles, h.makeEntryPoints())
		require.NoError(t, err)

		mainHash := buildAliasFromPath(mainPath)
		mainComp, ok := result.ComponentsByHash[mainHash]
		require.True(t, ok, "Main virtual component should exist in the result")

		virtualMainGoPath := mainComp.VirtualGoFilePath
		mainSourceBytes, ok := result.SourceOverlay[virtualMainGoPath]
		require.True(t, ok, "Generated Go source should exist in the overlay")
		mainSource := string(mainSourceBytes)

		cardHash := buildAliasFromPath(cardPath)

		assert.Contains(t, mainSource, `card := "shadowed"`,
			"Local variable assignment 'card := \"shadowed\"' should remain unchanged")
		assert.Contains(t, mainSource, "_ = card",
			"Local variable usage '_ = card' should remain unchanged")
		assert.Contains(t, mainSource, cardHash+".FormatPrice(100)",
			"Package-qualified call 'card.FormatPrice(100)' should still be rewritten to hashed name")
	})

	t.Run("should rewrite struct field access when local shadows import alias", func(t *testing.T) {
		h := newVirtualiserTestHarness(t, moduleName, baseDir)
		mainPath := filepath.Join(baseDir, "pages", "main.pk")
		cardPath := filepath.Join(baseDir, "partials", "card.pk")

		h.addComponent(cardPath, `
package card
func FormatPrice(price int) string { return "" }
`)

		mainScriptContent := `
package main
import card "partials/card.pk"
type LocalCard struct { Field int }
func Render() string {
	card := LocalCard{Field: 42}
	_ = card.Field
	return ""
}
func Other() string {
	return card.FormatPrice(100)
}
`
		mainPikoImport := annotator_dto.PikoImport{
			Alias: "card",
			Path:  "partials/card.pk",
		}
		h.addComponent(mainPath, mainScriptContent, mainPikoImport)

		virtualiser := NewModuleVirtualiser(h.resolver, h.pathsConfig)
		result, err := virtualiser.Virtualise(context.Background(), h.graph, h.originalGoFiles, h.makeEntryPoints())
		require.NoError(t, err)

		mainHash := buildAliasFromPath(mainPath)
		mainComp, ok := result.ComponentsByHash[mainHash]
		require.True(t, ok, "Main virtual component should exist in the result")

		virtualMainGoPath := mainComp.VirtualGoFilePath
		mainSourceBytes, ok := result.SourceOverlay[virtualMainGoPath]
		require.True(t, ok, "Generated Go source should exist in the overlay")
		mainSource := string(mainSourceBytes)

		cardHash := buildAliasFromPath(cardPath)

		assert.Contains(t, mainSource, cardHash+".FormatPrice(100)",
			"Package-qualified call in Other() should be rewritten")

		assert.Contains(t, mainSource, cardHash+".Field",
			"Shadowed field access is also rewritten (warning emitted)")

		require.Len(t, result.Diagnostics, 1, "Should emit exactly one diagnostic for shadowed alias")
		assert.Equal(t, ast_domain.Warning, result.Diagnostics[0].Severity)
		assert.Contains(t, result.Diagnostics[0].Message, "card")
		assert.Contains(t, result.Diagnostics[0].Message, "shadows")
	})

	t.Run("should handle same alias used for different partials in nested chain", func(t *testing.T) {
		h := newVirtualiserTestHarness(t, moduleName, baseDir)
		mainPath := filepath.Join(baseDir, "pages", "main.pk")
		cardPath := filepath.Join(baseDir, "partials", "card.pk")
		priceCardPath := filepath.Join(baseDir, "partials", "price-card.pk")

		h.addComponent(priceCardPath, `
package pricecard
func FormatCurrency(amount int) string { return "" }
`)

		cardScriptContent := `
package card
import card "partials/price-card.pk"
func FormatPrice(price int) string { return card.FormatCurrency(price) }
`
		cardPikoImport := annotator_dto.PikoImport{
			Alias: "card",
			Path:  "partials/price-card.pk",
		}
		h.addComponent(cardPath, cardScriptContent, cardPikoImport)

		mainScriptContent := `
package main
import card "partials/card.pk"
func Render() string { return card.FormatPrice(100) }
`
		mainPikoImport := annotator_dto.PikoImport{
			Alias: "card",
			Path:  "partials/card.pk",
		}
		h.addComponent(mainPath, mainScriptContent, mainPikoImport)

		virtualiser := NewModuleVirtualiser(h.resolver, h.pathsConfig)
		result, err := virtualiser.Virtualise(context.Background(), h.graph, h.originalGoFiles, h.makeEntryPoints())
		require.NoError(t, err)

		mainHash := buildAliasFromPath(mainPath)
		cardHash := buildAliasFromPath(cardPath)
		priceCardHash := buildAliasFromPath(priceCardPath)

		mainComp := result.ComponentsByHash[mainHash]
		require.NotNil(t, mainComp.RewrittenScriptAST)
		mainSource := string(result.SourceOverlay[mainComp.VirtualGoFilePath])

		cardComp := result.ComponentsByHash[cardHash]
		require.NotNil(t, cardComp.RewrittenScriptAST)
		cardSource := string(result.SourceOverlay[cardComp.VirtualGoFilePath])

		assert.Contains(t, mainSource, cardHash+".FormatPrice(100)",
			"main.pk's 'card.FormatPrice(100)' should be rewritten to '%s.FormatPrice(100)'", cardHash)
		assert.NotContains(t, mainSource, priceCardHash,
			"main.pk should not reference price-card's hash; it only imports card.pk")

		assert.Contains(t, cardSource, priceCardHash+".FormatCurrency(price)",
			"card.pk's 'card.FormatCurrency(price)' should be rewritten to '%s.FormatCurrency(price)'", priceCardHash)
		assert.NotContains(t, cardSource, "card.FormatCurrency",
			"card.pk should not have the original 'card.FormatCurrency' call in the output")

		assert.Equal(t, cardHash, mainComp.PikoAliasToHash["card"],
			"main.pk's 'card' alias should map to card.pk's hash")
		assert.Equal(t, priceCardHash, cardComp.PikoAliasToHash["card"],
			"card.pk's 'card' alias should map to price-card.pk's hash")
		assert.NotEqual(t, mainComp.PikoAliasToHash["card"], cardComp.PikoAliasToHash["card"],
			"The same alias 'card' in different components should map to different hashed names")
	})
}

func TestGetModuleRootAndMember(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		expression     ast_domain.Expression
		expectedRoot   string
		expectedMember string
		rootNil        bool
	}{
		{
			name:           "simple identifier returns root with no member",
			expression:     &ast_domain.Identifier{Name: "util"},
			expectedRoot:   "util",
			expectedMember: "",
		},
		{
			name: "member expression returns root and member",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "util"},
				Property: &ast_domain.Identifier{Name: "FormatUser"},
			},
			expectedRoot:   "util",
			expectedMember: "FormatUser",
		},
		{
			name: "call expression on member returns root, member",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "util"},
					Property: &ast_domain.Identifier{Name: "FormatUser"},
				},
				Args: []ast_domain.Expression{&ast_domain.Identifier{Name: "state"}},
			},
			expectedRoot:   "util",
			expectedMember: "FormatUser",
		},
		{
			name: "index expression unwraps to base",
			expression: &ast_domain.IndexExpression{
				Base:  &ast_domain.Identifier{Name: "items"},
				Index: &ast_domain.Identifier{Name: "0"},
			},
			expectedRoot:   "items",
			expectedMember: "",
		},
		{
			name:           "deeply nested member expression extracts root and member",
			expression:     &ast_domain.MemberExpression{Base: &ast_domain.MemberExpression{Base: &ast_domain.MemberExpression{Base: &ast_domain.MemberExpression{Base: &ast_domain.MemberExpression{Base: &ast_domain.Identifier{Name: "a"}, Property: &ast_domain.Identifier{Name: "b"}}, Property: &ast_domain.Identifier{Name: "c"}}, Property: &ast_domain.Identifier{Name: "d"}}, Property: &ast_domain.Identifier{Name: "e"}}, Property: &ast_domain.Identifier{Name: "f"}},
			expectedRoot:   "a",
			expectedMember: "b",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, member := getModuleRootAndMember(tc.expression)
			if tc.rootNil {
				assert.Nil(t, root, "Expected root to be nil")
			} else {
				require.NotNil(t, root, "Expected root to be non-nil")
				if tc.expectedRoot != "" {
					assert.Equal(t, tc.expectedRoot, root.Name)
				}
			}
			assert.Equal(t, tc.expectedMember, member)
		})
	}
}

func TestGetModuleRootAndMemberWithCallInfo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		expression     ast_domain.Expression
		expectedRoot   string
		expectedMember string
		expectedIsCall bool
		rootNil        bool
	}{
		{
			name:           "simple identifier is not a call",
			expression:     &ast_domain.Identifier{Name: "foo"},
			expectedRoot:   "foo",
			expectedMember: "",
			expectedIsCall: false,
		},
		{
			name: "member expression is not a call",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "pkg"},
				Property: &ast_domain.Identifier{Name: "SomeValue"},
			},
			expectedRoot:   "pkg",
			expectedMember: "SomeValue",
			expectedIsCall: false,
		},
		{
			name: "call expression on package member is a call",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "pkg"},
					Property: &ast_domain.Identifier{Name: "Func"},
				},
				Args: nil,
			},
			expectedRoot:   "pkg",
			expectedMember: "Func",
			expectedIsCall: true,
		},
		{
			name: "call expression on bare identifier is a call with no member",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "doStuff"},
				Args:   nil,
			},
			expectedRoot:   "doStuff",
			expectedMember: "",
			expectedIsCall: false,
		},
		{
			name: "index expression unwraps to identifier",
			expression: &ast_domain.IndexExpression{
				Base:  &ast_domain.Identifier{Name: "arr"},
				Index: &ast_domain.Identifier{Name: "0"},
			},
			expectedRoot:   "arr",
			expectedMember: "",
			expectedIsCall: false,
		},
		{
			name: "member expression with non-identifier base returns nil",
			expression: &ast_domain.MemberExpression{
				Base: &ast_domain.CallExpression{
					Callee: &ast_domain.Identifier{Name: "getObj"},
					Args:   nil,
				},
				Property: &ast_domain.Identifier{Name: "Field"},
			},
			expectedRoot:   "getObj",
			expectedMember: "",
			expectedIsCall: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, member, isCall := getModuleRootAndMemberWithCallInfo(tc.expression)
			if tc.rootNil {
				assert.Nil(t, root, "Expected root to be nil")
			} else {
				require.NotNil(t, root, "Expected root to be non-nil")
				assert.Equal(t, tc.expectedRoot, root.Name)
			}
			assert.Equal(t, tc.expectedMember, member)
			assert.Equal(t, tc.expectedIsCall, isCall)
		})
	}
}

func TestTryExtractPackageMemberPattern(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		memberExpr     *ast_domain.MemberExpression
		expectedRoot   string
		expectedMember string
		expectedFound  bool
	}{
		{
			name: "valid pkg.Member pattern",
			memberExpr: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "fmt"},
				Property: &ast_domain.Identifier{Name: "Println"},
			},
			expectedRoot:   "fmt",
			expectedMember: "Println",
			expectedFound:  true,
		},
		{
			name: "non-identifier base returns not found",
			memberExpr: &ast_domain.MemberExpression{
				Base: &ast_domain.CallExpression{
					Callee: &ast_domain.Identifier{Name: "getObj"},
				},
				Property: &ast_domain.Identifier{Name: "Field"},
			},
			expectedRoot:   "",
			expectedMember: "",
			expectedFound:  false,
		},
		{
			name: "non-identifier property returns not found",
			memberExpr: &ast_domain.MemberExpression{
				Base: &ast_domain.Identifier{Name: "obj"},
				Property: &ast_domain.CallExpression{
					Callee: &ast_domain.Identifier{Name: "method"},
				},
			},
			expectedRoot:   "",
			expectedMember: "",
			expectedFound:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, member, found := tryExtractPackageMemberPattern(tc.memberExpr)
			assert.Equal(t, tc.expectedFound, found)
			if tc.expectedFound {
				require.NotNil(t, root)
				assert.Equal(t, tc.expectedRoot, root.Name)
				assert.Equal(t, tc.expectedMember, member)
			} else {
				assert.Nil(t, root)
				assert.Empty(t, member)
			}
		})
	}
}

func TestTryExtractCallExprMember(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		callExpr       *ast_domain.CallExpression
		expectedRoot   string
		expectedMember string
		expectNil      bool
	}{
		{
			name: "direct pkg.Func() call extracts member",
			callExpr: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "strings"},
					Property: &ast_domain.Identifier{Name: "Join"},
				},
			},
			expectedRoot:   "strings",
			expectedMember: "Join",
		},
		{
			name: "callee is not a member expression returns nil",
			callExpr: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "doSomething"},
			},
			expectNil: true,
		},
		{
			name: "member expression with non-identifier base returns nil",
			callExpr: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base: &ast_domain.CallExpression{
						Callee: &ast_domain.Identifier{Name: "getObj"},
					},
					Property: &ast_domain.Identifier{Name: "Method"},
				},
			},
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := tryExtractCallExprMember(tc.callExpr)
			if tc.expectNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tc.expectedRoot, result.root.Name)
				assert.Equal(t, tc.expectedMember, result.member)
			}
		})
	}
}

func TestHasFuncDecl(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		src          string
		functionName string
		expected     bool
	}{
		{
			name:         "finds existing function",
			src:          `package main; func Render() {}`,
			functionName: "Render",
			expected:     true,
		},
		{
			name:         "returns false for missing function",
			src:          `package main; func Other() {}`,
			functionName: "Render",
			expected:     false,
		},
		{
			name:         "returns false in empty file",
			src:          `package main`,
			functionName: "Render",
			expected:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "", tc.src, 0)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, hasFuncDecl(f, tc.functionName))
		})
	}
}

func TestCreateImportSpec(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		path          string
		alias         string
		expectedAlias string
		expectAlias   bool
	}{
		{
			name:        "with alias",
			path:        "piko.sh/piko",
			alias:       "piko",
			expectAlias: true,
		},
		{
			name:        "without alias",
			path:        "fmt",
			alias:       "",
			expectAlias: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			spec := createImportSpec(tc.path, tc.alias)
			require.NotNil(t, spec)
			assert.Contains(t, spec.Path.Value, tc.path)
			if tc.expectAlias {
				require.NotNil(t, spec.Name)
				assert.Equal(t, tc.alias, spec.Name.Name)
			} else {
				assert.Nil(t, spec.Name)
			}
		})
	}
}

func TestEnsureImportAndGetAlias(t *testing.T) {
	t.Parallel()

	t.Run("returns existing alias when import already exists with alias", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", `package main; import myalias "piko.sh/piko"`, 0)
		require.NoError(t, err)

		alias := ensureImportAndGetAlias(f, "piko.sh/piko", "piko")
		assert.Equal(t, "myalias", alias, "Should return the existing alias")
	})

	t.Run("returns default alias when import exists without alias", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", `package main; import "fmt"`, 0)
		require.NoError(t, err)

		alias := ensureImportAndGetAlias(f, "fmt", "fmt")
		assert.Equal(t, "fmt", alias)
	})

	t.Run("adds import and returns default alias when not present", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", `package main`, 0)
		require.NoError(t, err)

		alias := ensureImportAndGetAlias(f, "piko.sh/piko", "piko")
		assert.Equal(t, "piko", alias)
		assert.Len(t, f.Imports, 1)
		assert.Contains(t, f.Imports[0].Path.Value, "piko.sh/piko")
	})
}

func TestReconstructModuleImportBlock(t *testing.T) {
	t.Parallel()

	t.Run("removes old import decls and builds a new grouped one", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		src := `package main

import "fmt"
import "strings"

func Foo() {}
`
		f, err := parser.ParseFile(fset, "", src, 0)
		require.NoError(t, err)

		reconstructModuleImportBlock(f)

		importCount := 0
		for _, declaration := range f.Decls {
			if gen, ok := declaration.(*goast.GenDecl); ok && gen.Tok == token.IMPORT {
				importCount++
			}
		}
		assert.Equal(t, 1, importCount, "Should have exactly one import declaration block")
	})

	t.Run("handles file with no imports", func(t *testing.T) {
		t.Parallel()
		f := &goast.File{
			Name:    goast.NewIdent("main"),
			Imports: nil,
			Decls: []goast.Decl{
				&goast.FuncDecl{
					Name: goast.NewIdent("Foo"),
					Type: &goast.FuncType{},
					Body: &goast.BlockStmt{},
				},
			},
		}

		reconstructModuleImportBlock(f)

		for _, declaration := range f.Decls {
			if gen, ok := declaration.(*goast.GenDecl); ok && gen.Tok == token.IMPORT {
				t.Fatal("Should not have an import declaration when there are no imports")
			}
		}
	})
}

func TestBuildDefaultRenderDecl(t *testing.T) {
	t.Parallel()

	t.Run("uses NoProps when propsTypeExpr is nil", func(t *testing.T) {
		t.Parallel()
		declaration := buildDefaultRenderDecl("piko", nil)
		require.NotNil(t, declaration)
		assert.Equal(t, "Render", declaration.Name.Name)
		require.Len(t, declaration.Type.Params.List, 2)

		selExpr, ok := declaration.Type.Params.List[1].Type.(*goast.SelectorExpr)
		require.True(t, ok)
		assert.Equal(t, "NoProps", selExpr.Sel.Name)
	})

	t.Run("uses provided propsTypeExpr when non-nil", func(t *testing.T) {
		t.Parallel()
		customType := goast.NewIdent("MyProps")
		declaration := buildDefaultRenderDecl("piko", customType)
		require.NotNil(t, declaration)
		assert.Equal(t, customType, declaration.Type.Params.List[1].Type)
	})
}

func TestBuildDefaultCachePolicyDecl(t *testing.T) {
	t.Parallel()

	declaration := buildDefaultCachePolicyDecl("piko")
	require.NotNil(t, declaration)
	assert.Equal(t, "CachePolicy", declaration.Name.Name)
	require.NotNil(t, declaration.Type.Results)
	require.Len(t, declaration.Type.Results.List, 1)

	selExpr, ok := declaration.Type.Results.List[0].Type.(*goast.SelectorExpr)
	require.True(t, ok)
	assert.Equal(t, "CachePolicy", selExpr.Sel.Name)
}

func TestCalculateVirtualManifestKey(t *testing.T) {
	t.Parallel()
	vc := &virtualisationContext{}

	testCases := []struct {
		name        string
		vps         *annotator_dto.VirtualPageSource
		baseDir     string
		expectedKey string
	}{
		{
			name: "uses URL from page metadata when present",
			vps: &annotator_dto.VirtualPageSource{
				InitialProps: map[string]any{
					"page": map[string]any{
						collection_dto.MetaKeyURL: "/blog/my-post",
					},
				},
				TemplatePath: "/project/pages/blog/{slug}.pk",
			},
			baseDir:     "/project",
			expectedKey: "pages/blog/my-post.pk",
		},
		{
			name: "falls back to relative path when no URL in metadata",
			vps: &annotator_dto.VirtualPageSource{
				InitialProps: map[string]any{},
				TemplatePath: "/project/pages/index.pk",
			},
			baseDir:     "/project",
			expectedKey: "pages/index.pk",
		},
		{
			name: "replaces dynamic param with slug when slug is present",
			vps: &annotator_dto.VirtualPageSource{
				InitialProps: map[string]any{
					"page": map[string]any{
						collection_dto.MetaKeySlug: "hello-world",
					},
				},
				TemplatePath: "/project/pages/blog/{slug}.pk",
			},
			baseDir:     "/project",
			expectedKey: "pages/blog/hello-world.pk",
		},
		{
			name: "returns relative path when no slug and no URL",
			vps: &annotator_dto.VirtualPageSource{
				InitialProps: map[string]any{
					"page": map[string]any{
						"Title": "Test",
					},
				},
				TemplatePath: "/project/pages/static.pk",
			},
			baseDir:     "/project",
			expectedKey: "pages/static.pk",
		},
		{
			name: "uses template path directly when Rel fails",
			vps: &annotator_dto.VirtualPageSource{
				InitialProps: map[string]any{},
				TemplatePath: "relative/path.pk",
			},
			baseDir:     "/project",
			expectedKey: "relative/path.pk",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			key := vc.calculateVirtualManifestKey(tc.vps, tc.baseDir)
			assert.Equal(t, tc.expectedKey, key)
		})
	}
}

func TestCalculateVirtualRoute(t *testing.T) {
	t.Parallel()
	vc := &virtualisationContext{}

	testCases := []struct {
		name          string
		vps           *annotator_dto.VirtualPageSource
		expectedRoute string
	}{
		{
			name: "uses route override when present",
			vps: &annotator_dto.VirtualPageSource{
				RouteOverride: "/custom/route",
				InitialProps: map[string]any{
					"page": map[string]any{
						collection_dto.MetaKeyURL: "/from-metadata",
					},
				},
			},
			expectedRoute: "/custom/route",
		},
		{
			name: "uses URL from page metadata when no override",
			vps: &annotator_dto.VirtualPageSource{
				RouteOverride: "",
				InitialProps: map[string]any{
					"page": map[string]any{
						collection_dto.MetaKeyURL: "/blog/test-post",
					},
				},
			},
			expectedRoute: "/blog/test-post",
		},
		{
			name: "returns empty when no override and no URL",
			vps: &annotator_dto.VirtualPageSource{
				RouteOverride: "",
				InitialProps:  map[string]any{},
			},
			expectedRoute: "",
		},
		{
			name: "returns empty when page map has no URL key",
			vps: &annotator_dto.VirtualPageSource{
				RouteOverride: "",
				InitialProps: map[string]any{
					"page": map[string]any{
						"Title": "Test",
					},
				},
			},
			expectedRoute: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			route := vc.calculateVirtualRoute(tc.vps)
			assert.Equal(t, tc.expectedRoute, route)
		})
	}
}

func TestRecordEntryPointFlags(t *testing.T) {
	t.Parallel()
	vc := &virtualisationContext{}

	testCases := []struct {
		name         string
		ep           annotator_dto.EntryPoint
		expectPage   bool
		expectPublic bool
		expectEmail  bool
		expectE2E    bool
	}{
		{
			name:         "records page flag",
			ep:           annotator_dto.EntryPoint{IsPage: true},
			expectPage:   true,
			expectPublic: false,
			expectEmail:  false,
			expectE2E:    false,
		},
		{
			name:         "records public flag",
			ep:           annotator_dto.EntryPoint{IsPublic: true},
			expectPage:   false,
			expectPublic: true,
			expectEmail:  false,
			expectE2E:    false,
		},
		{
			name:         "records email flag",
			ep:           annotator_dto.EntryPoint{IsEmail: true},
			expectPage:   false,
			expectPublic: false,
			expectEmail:  true,
			expectE2E:    false,
		},
		{
			name:         "records e2e only flag",
			ep:           annotator_dto.EntryPoint{IsE2EOnly: true},
			expectPage:   false,
			expectPublic: false,
			expectEmail:  false,
			expectE2E:    true,
		},
		{
			name:         "records all flags together",
			ep:           annotator_dto.EntryPoint{IsPage: true, IsPublic: true, IsEmail: true, IsE2EOnly: true},
			expectPage:   true,
			expectPublic: true,
			expectEmail:  true,
			expectE2E:    true,
		},
		{
			name:         "records no flags when all false",
			ep:           annotator_dto.EntryPoint{},
			expectPage:   false,
			expectPublic: false,
			expectEmail:  false,
			expectE2E:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			meta := &entryPointMetadata{
				isPage:    make(map[string]bool),
				isPublic:  make(map[string]bool),
				isEmail:   make(map[string]bool),
				isE2EOnly: make(map[string]bool),
			}
			resolvedPath := "/test/component.pk"

			vc.recordEntryPointFlags(meta, resolvedPath, tc.ep)

			assert.Equal(t, tc.expectPage, meta.isPage[resolvedPath])
			assert.Equal(t, tc.expectPublic, meta.isPublic[resolvedPath])
			assert.Equal(t, tc.expectEmail, meta.isEmail[resolvedPath])
			assert.Equal(t, tc.expectE2E, meta.isE2EOnly[resolvedPath])
		})
	}
}

func TestResolveComponentPaths(t *testing.T) {
	t.Parallel()

	pathsConfig := AnnotatorPathsConfig{
		PartialsSourceDir: "partials",
		PartialServePath:  "/_piko/partials",
	}

	testCases := []struct {
		name              string
		expectedTargetDir string
		isPage            bool
		isEmail           bool
		isPdf             bool
		isErrorPage       bool
	}{
		{
			name:              "email components use emails target directory",
			isPage:            false,
			isEmail:           true,
			expectedTargetDir: config.CompiledEmailsTargetDir,
		},
		{
			name:              "pdf components use pdfs target directory",
			isPage:            false,
			isPdf:             true,
			expectedTargetDir: config.CompiledPdfsTargetDir,
		},
		{
			name:              "page components use pages target directory",
			isPage:            true,
			isEmail:           false,
			expectedTargetDir: config.CompiledPagesTargetDir,
		},
		{
			name:              "error page components use pages target directory",
			isPage:            false,
			isErrorPage:       true,
			expectedTargetDir: config.CompiledPagesTargetDir,
		},
		{
			name:              "partial components use partials target directory",
			isPage:            false,
			isEmail:           false,
			expectedTargetDir: config.CompiledPartialsTargetDir,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			vc := &virtualisationContext{
				pathsConfig: pathsConfig,
			}
			parsedComp := &annotator_dto.ParsedComponent{
				SourcePath: filepath.Join("/project", "partials", "card.pk"),
			}
			baseDir := "/project"

			targetSubDir, _, _ := vc.resolveComponentPaths(context.Background(), parsedComp, baseDir, tc.isPage, tc.isEmail, tc.isPdf, tc.isErrorPage)
			assert.Equal(t, tc.expectedTargetDir, targetSubDir)
		})
	}
}

func TestResolvePartialPaths(t *testing.T) {
	t.Parallel()

	pathsConfig := AnnotatorPathsConfig{
		PartialsSourceDir: "partials",
		PartialServePath:  "/_piko/partials",
	}

	vc := &virtualisationContext{
		pathsConfig: pathsConfig,
	}
	baseDir := "/project"
	sourcePath := filepath.Join("/project", "partials", "card.pk")

	targetSubDir, partialName, partialSrc := vc.resolvePartialPaths(context.Background(), sourcePath, baseDir)

	assert.Equal(t, "dist/partials", targetSubDir)
	assert.Equal(t, "card", partialName, "Partial name should strip .pk extension")
	assert.Contains(t, partialSrc, "card", "Partial source path should include the partial name")
	assert.Contains(t, partialSrc, "/_piko/partials", "Partial source path should include the serve path")
}

func TestCalculateRelativePartialPath(t *testing.T) {
	t.Parallel()

	pathsConfig := AnnotatorPathsConfig{
		PartialsSourceDir: "partials",
	}

	testCases := []struct {
		name         string
		sourcePath   string
		baseDir      string
		expectedPath string
	}{
		{
			name:         "path inside partials source directory",
			sourcePath:   filepath.Join("/project", "partials", "card.pk"),
			baseDir:      "/project",
			expectedPath: "card.pk",
		},
		{
			name:         "nested path inside partials source directory",
			sourcePath:   filepath.Join("/project", "partials", "ui", "button.pk"),
			baseDir:      "/project",
			expectedPath: filepath.Join("ui", "button.pk"),
		},
		{
			name:         "path outside partials directory falls back to relative from base",
			sourcePath:   filepath.Join("/project", "other", "card.pk"),
			baseDir:      "/project",
			expectedPath: filepath.Join("other", "card.pk"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			vc := &virtualisationContext{
				pathsConfig: pathsConfig,
			}
			result := vc.calculateRelativePartialPath(context.Background(), tc.sourcePath, tc.baseDir)
			assert.Equal(t, tc.expectedPath, result)
		})
	}
}

func TestCreateVirtualInstance(t *testing.T) {
	t.Parallel()

	pathsConfig := AnnotatorPathsConfig{
		PartialsSourceDir: "partials",
	}

	vc := &virtualisationContext{
		pathsConfig: pathsConfig,
	}

	ep := annotator_dto.EntryPoint{
		Path: "/project/pages/blog/{slug}.pk",
		VirtualPageSource: &annotator_dto.VirtualPageSource{
			RouteOverride: "/blog/test-post",
			TemplatePath:  "/project/pages/blog/{slug}.pk",
			InitialProps: map[string]any{
				"page": map[string]any{
					collection_dto.MetaKeyURL:  "/blog/test-post",
					collection_dto.MetaKeySlug: "test-post",
				},
			},
		},
	}

	instance := vc.createVirtualInstance(ep, "/project")

	assert.Equal(t, "/blog/test-post", instance.Route)
	assert.Equal(t, "pages/blog/test-post.pk", instance.ManifestKey)
	assert.NotNil(t, instance.InitialProps)
}

func TestDeepCopyASTFile(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns nil", func(t *testing.T) {
		t.Parallel()
		result := deepCopyASTFile(nil)
		assert.Nil(t, result)
	})

	t.Run("creates independent copy", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		original, err := parser.ParseFile(fset, "", `package main; func Foo() {}`, 0)
		require.NoError(t, err)

		copied := deepCopyASTFile(original)
		require.NotNil(t, copied)
		assert.Equal(t, original.Name.Name, copied.Name.Name)

		copied.Name.Name = "changed"
		assert.Equal(t, "main", original.Name.Name)
		assert.Equal(t, "changed", copied.Name.Name)
	})
}

func TestCreateUseDecl(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		alias     string
		member    string
		expectNil bool
	}{
		{
			name:      "valid alias and member creates declaration",
			alias:     "fmt",
			member:    "Println",
			expectNil: false,
		},
		{
			name:      "empty alias returns nil",
			alias:     "",
			member:    "Println",
			expectNil: true,
		},
		{
			name:      "blank alias returns nil",
			alias:     "_",
			member:    "Println",
			expectNil: true,
		},
		{
			name:      "dot alias returns nil",
			alias:     ".",
			member:    "Println",
			expectNil: true,
		},
		{
			name:      "empty member returns nil",
			alias:     "fmt",
			member:    "",
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := createUseDecl(tc.alias, tc.member)
			if tc.expectNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				genDecl, ok := result.(*goast.GenDecl)
				require.True(t, ok)
				assert.Equal(t, token.VAR, genDecl.Tok)
			}
		})
	}
}

func TestGetAliasFromSpec(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		spec          *goast.ImportSpec
		expectedAlias string
	}{
		{
			name: "returns explicit alias when present",
			spec: &goast.ImportSpec{
				Name: goast.NewIdent("myalias"),
				Path: &goast.BasicLit{Kind: token.STRING, Value: `"fmt"`},
			},
			expectedAlias: "myalias",
		},
		{
			name: "returns last path segment when no alias",
			spec: &goast.ImportSpec{
				Path: &goast.BasicLit{Kind: token.STRING, Value: `"encoding/json"`},
			},
			expectedAlias: "json",
		},
		{
			name: "returns full path when no slash and no alias",
			spec: &goast.ImportSpec{
				Path: &goast.BasicLit{Kind: token.STRING, Value: `"fmt"`},
			},
			expectedAlias: "fmt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			alias := getAliasFromSpec(tc.spec)
			assert.Equal(t, tc.expectedAlias, alias)
		})
	}
}

func TestBuildImportDecl(t *testing.T) {
	t.Parallel()

	t.Run("single import has no parentheses", func(t *testing.T) {
		t.Parallel()
		imports := map[string]*goast.ImportSpec{
			"fmt": createImportSpec("fmt", ""),
		}
		declaration := buildImportDecl(imports)
		require.NotNil(t, declaration)
		assert.Equal(t, token.IMPORT, declaration.Tok)
		assert.Len(t, declaration.Specs, 1)
		assert.Equal(t, token.Pos(0), declaration.Lparen, "Single import should not have parentheses")
	})

	t.Run("multiple imports have parentheses and are sorted", func(t *testing.T) {
		t.Parallel()
		imports := map[string]*goast.ImportSpec{
			"strings": createImportSpec("strings", ""),
			"fmt":     createImportSpec("fmt", ""),
			"os":      createImportSpec("os", ""),
		}
		declaration := buildImportDecl(imports)
		require.NotNil(t, declaration)
		assert.Equal(t, token.Pos(1), declaration.Lparen, "Multiple imports should have parentheses")
		assert.Len(t, declaration.Specs, 3)

		paths := make([]string, 0, len(declaration.Specs))
		for _, spec := range declaration.Specs {
			imp, ok := spec.(*goast.ImportSpec)
			require.True(t, ok)
			paths = append(paths, strings.Trim(imp.Path.Value, `"`))
		}
		assert.Equal(t, []string{"fmt", "os", "strings"}, paths)
	})
}

func TestWalkModuleNodeExpressions(t *testing.T) {
	t.Parallel()

	t.Run("nil node does not panic", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() {
			walkModuleNodeExpressions(nil, func(_ ast_domain.Expression) {
				t.Fatal("visit should not be called for nil node")
			})
		})
	})

	t.Run("visits directive expressions", func(t *testing.T) {
		t.Parallel()
		var visited []string
		visit := func(expression ast_domain.Expression) {
			visited = append(visited, expression.String())
		}

		node := &ast_domain.TemplateNode{
			DirIf: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "showItem"},
			},
			DirFor: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "items"},
			},
			DirShow: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "visible"},
			},
		}

		walkModuleNodeExpressions(node, visit)
		assert.Contains(t, visited, "showItem")
		assert.Contains(t, visited, "items")
		assert.Contains(t, visited, "visible")
	})

	t.Run("visits dynamic attribute expressions", func(t *testing.T) {
		t.Parallel()
		var visited []string
		visit := func(expression ast_domain.Expression) {
			visited = append(visited, expression.String())
		}

		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{
					Name:       "title",
					Expression: &ast_domain.Identifier{Name: "pageTitle"},
				},
				{
					Name:       "href",
					Expression: &ast_domain.Identifier{Name: "linkURL"},
				},
			},
		}

		walkModuleNodeExpressions(node, visit)
		assert.Contains(t, visited, "pageTitle")
		assert.Contains(t, visited, "linkURL")
	})

	t.Run("visits rich text expressions", func(t *testing.T) {
		t.Parallel()
		var visited []string
		visit := func(expression ast_domain.Expression) {
			visited = append(visited, expression.String())
		}

		node := &ast_domain.TemplateNode{
			RichText: []ast_domain.TextPart{
				{IsLiteral: true, Literal: "Hello "},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "userName"}},
			},
		}

		walkModuleNodeExpressions(node, visit)
		assert.Contains(t, visited, "userName")
		assert.Len(t, visited, 1, "Literal text parts should not be visited")
	})

	t.Run("visits event expressions", func(t *testing.T) {
		t.Parallel()
		var visited []string
		visit := func(expression ast_domain.Expression) {
			visited = append(visited, expression.String())
		}

		node := &ast_domain.TemplateNode{
			OnEvents: map[string][]ast_domain.Directive{
				"click": {
					{Expression: &ast_domain.Identifier{Name: "handleClick"}},
				},
			},
			CustomEvents: map[string][]ast_domain.Directive{
				"update": {
					{Expression: &ast_domain.Identifier{Name: "handleUpdate"}},
				},
			},
		}

		walkModuleNodeExpressions(node, visit)
		assert.Contains(t, visited, "handleClick")
		assert.Contains(t, visited, "handleUpdate")
	})

	t.Run("visits bind expressions", func(t *testing.T) {
		t.Parallel()
		var visited []string
		visit := func(expression ast_domain.Expression) {
			visited = append(visited, expression.String())
		}

		node := &ast_domain.TemplateNode{
			Binds: map[string]*ast_domain.Directive{
				"value": {Expression: &ast_domain.Identifier{Name: "inputVal"}},
				"empty": nil,
			},
		}

		walkModuleNodeExpressions(node, visit)
		assert.Contains(t, visited, "inputVal")
		assert.Len(t, visited, 1, "Nil bind directives should not be visited")
	})

	t.Run("visits all directive types", func(t *testing.T) {
		t.Parallel()
		var visited []string
		visit := func(expression ast_domain.Expression) {
			visited = append(visited, expression.String())
		}

		node := &ast_domain.TemplateNode{
			DirElseIf: &ast_domain.Directive{Expression: &ast_domain.Identifier{Name: "elseIfExpr"}},
			DirModel:  &ast_domain.Directive{Expression: &ast_domain.Identifier{Name: "modelExpr"}},
			DirClass:  &ast_domain.Directive{Expression: &ast_domain.Identifier{Name: "classExpr"}},
			DirStyle:  &ast_domain.Directive{Expression: &ast_domain.Identifier{Name: "styleExpr"}},
			DirText:   &ast_domain.Directive{Expression: &ast_domain.Identifier{Name: "textExpr"}},
			DirHTML:   &ast_domain.Directive{Expression: &ast_domain.Identifier{Name: "htmlExpr"}},
			DirKey:    &ast_domain.Directive{Expression: &ast_domain.Identifier{Name: "keyExpr"}},
		}

		walkModuleNodeExpressions(node, visit)
		assert.Contains(t, visited, "elseIfExpr")
		assert.Contains(t, visited, "modelExpr")
		assert.Contains(t, visited, "classExpr")
		assert.Contains(t, visited, "styleExpr")
		assert.Contains(t, visited, "textExpr")
		assert.Contains(t, visited, "htmlExpr")
		assert.Contains(t, visited, "keyExpr")
	})
}

func TestVisitDirectiveExpr(t *testing.T) {
	t.Parallel()

	t.Run("nil directive does not call visit", func(t *testing.T) {
		t.Parallel()
		visitDirectiveExpr(nil, func(_ ast_domain.Expression) {
			t.Fatal("visit should not be called for nil directive")
		})
	})

	t.Run("directive with nil expression does not call visit", func(t *testing.T) {
		t.Parallel()
		visitDirectiveExpr(&ast_domain.Directive{Expression: nil}, func(_ ast_domain.Expression) {
			t.Fatal("visit should not be called for nil expression")
		})
	})

	t.Run("directive with expression calls visit", func(t *testing.T) {
		t.Parallel()
		called := false
		visitDirectiveExpr(&ast_domain.Directive{
			Expression: &ast_domain.Identifier{Name: "test"},
		}, func(expression ast_domain.Expression) {
			called = true
			assert.Equal(t, "test", expression.String())
		})
		assert.True(t, called)
	})
}

func TestVisitDynamicAttrExprs(t *testing.T) {
	t.Parallel()

	t.Run("empty slice does not call visit", func(t *testing.T) {
		t.Parallel()
		visitDynamicAttrExprs(nil, func(_ ast_domain.Expression) {
			t.Fatal("visit should not be called for empty slice")
		})
	})

	t.Run("skips nil expressions", func(t *testing.T) {
		t.Parallel()
		var visited []string
		attrs := []ast_domain.DynamicAttribute{
			{Name: "noexpr", Expression: nil},
			{Name: "withexpr", Expression: &ast_domain.Identifier{Name: "val"}},
		}
		visitDynamicAttrExprs(attrs, func(expression ast_domain.Expression) {
			visited = append(visited, expression.String())
		})
		assert.Equal(t, []string{"val"}, visited)
	})
}

func TestVisitRichTextExprs(t *testing.T) {
	t.Parallel()

	t.Run("empty slice does not call visit", func(t *testing.T) {
		t.Parallel()
		visitRichTextExprs(nil, func(_ ast_domain.Expression) {
			t.Fatal("visit should not be called for empty slice")
		})
	})

	t.Run("skips literals and nil expressions", func(t *testing.T) {
		t.Parallel()
		var visited []string
		parts := []ast_domain.TextPart{
			{IsLiteral: true, Literal: "static"},
			{IsLiteral: false, Expression: nil},
			{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "dynamic"}},
		}
		visitRichTextExprs(parts, func(expression ast_domain.Expression) {
			visited = append(visited, expression.String())
		})
		assert.Equal(t, []string{"dynamic"}, visited)
	})
}

func TestVisitEventExprs(t *testing.T) {
	t.Parallel()

	t.Run("empty map does not call visit", func(t *testing.T) {
		t.Parallel()
		visitEventExprs(nil, func(_ ast_domain.Expression) {
			t.Fatal("visit should not be called for empty map")
		})
	})

	t.Run("skips nil expressions", func(t *testing.T) {
		t.Parallel()
		var visited []string
		events := map[string][]ast_domain.Directive{
			"click": {
				{Expression: nil},
				{Expression: &ast_domain.Identifier{Name: "handler"}},
			},
		}
		visitEventExprs(events, func(expression ast_domain.Expression) {
			visited = append(visited, expression.String())
		})
		assert.Equal(t, []string{"handler"}, visited)
	})
}

func TestVisitBindExprs(t *testing.T) {
	t.Parallel()

	t.Run("empty map does not call visit", func(t *testing.T) {
		t.Parallel()
		visitBindExprs(nil, func(_ ast_domain.Expression) {
			t.Fatal("visit should not be called for empty map")
		})
	})

	t.Run("skips nil directives and nil expressions", func(t *testing.T) {
		t.Parallel()
		var visited []string
		binds := map[string]*ast_domain.Directive{
			"nilDir":  nil,
			"nilExpr": {Expression: nil},
			"valid":   {Expression: &ast_domain.Identifier{Name: "bound"}},
		}
		visitBindExprs(binds, func(expression ast_domain.Expression) {
			visited = append(visited, expression.String())
		})
		assert.Equal(t, []string{"bound"}, visited)
	})
}

func TestHandleSideEffectImport(t *testing.T) {
	t.Parallel()
	ar := &astRewriter{}

	testCases := []struct {
		templateUses     map[string]map[string]templateMemberInfo
		name             string
		alias            string
		path             string
		expectedAlias    string
		isPikoImport     bool
		isUsedInTemplate bool
		expectedUsed     bool
	}{
		{
			name:             "non-side-effect import returns unchanged",
			alias:            "fmt",
			path:             "fmt",
			isPikoImport:     false,
			templateUses:     map[string]map[string]templateMemberInfo{},
			isUsedInTemplate: false,
			expectedAlias:    "fmt",
			expectedUsed:     false,
		},
		{
			name:             "piko import returns unchanged even with side-effect alias",
			alias:            "_",
			path:             "partials/card.pk",
			isPikoImport:     true,
			templateUses:     map[string]map[string]templateMemberInfo{},
			isUsedInTemplate: false,
			expectedAlias:    "_",
			expectedUsed:     false,
		},
		{
			name:         "side-effect import resolves to package name when used in template",
			alias:        "_",
			path:         "github.com/example/utils",
			isPikoImport: false,
			templateUses: map[string]map[string]templateMemberInfo{
				"utils": {"Format": {isCall: false}},
			},
			isUsedInTemplate: false,
			expectedAlias:    "utils",
			expectedUsed:     true,
		},
		{
			name:         "side-effect import not found in template stays as underscore",
			alias:        "_",
			path:         "github.com/example/utils",
			isPikoImport: false,
			templateUses: map[string]map[string]templateMemberInfo{
				"other": {"Format": {isCall: false}},
			},
			isUsedInTemplate: false,
			expectedAlias:    "_",
			expectedUsed:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			alias, used := ar.handleSideEffectImport(
				tc.alias, tc.path, tc.isPikoImport, tc.templateUses, tc.isUsedInTemplate,
			)
			assert.Equal(t, tc.expectedAlias, alias)
			assert.Equal(t, tc.expectedUsed, used)
		})
	}
}

func TestMaybeAddTemplateOnlyUseDecl(t *testing.T) {
	t.Parallel()
	ar := &astRewriter{}

	t.Run("does not add use decl when used in script", func(t *testing.T) {
		t.Parallel()
		var useDecls []goast.Decl
		templateUses := map[string]map[string]templateMemberInfo{
			"fmt": {"Println": {isCall: false}},
		}
		ar.maybeAddTemplateOnlyUseDecl("fmt", false, true, true, templateUses, &useDecls)
		assert.Empty(t, useDecls)
	})

	t.Run("does not add use decl when not used in template", func(t *testing.T) {
		t.Parallel()
		var useDecls []goast.Decl
		templateUses := map[string]map[string]templateMemberInfo{}
		ar.maybeAddTemplateOnlyUseDecl("fmt", false, false, false, templateUses, &useDecls)
		assert.Empty(t, useDecls)
	})

	t.Run("does not add use decl for piko imports", func(t *testing.T) {
		t.Parallel()
		var useDecls []goast.Decl
		templateUses := map[string]map[string]templateMemberInfo{
			"card": {"Render": {isCall: false}},
		}
		ar.maybeAddTemplateOnlyUseDecl("card", true, false, true, templateUses, &useDecls)
		assert.Empty(t, useDecls)
	})

	t.Run("adds use decl for template-only non-call member", func(t *testing.T) {
		t.Parallel()
		var useDecls []goast.Decl
		templateUses := map[string]map[string]templateMemberInfo{
			"fmt": {"Sprintf": {isCall: false}},
		}
		ar.maybeAddTemplateOnlyUseDecl("fmt", false, false, true, templateUses, &useDecls)
		assert.Len(t, useDecls, 1)
	})

	t.Run("does not add use decl when all members are calls", func(t *testing.T) {
		t.Parallel()
		var useDecls []goast.Decl
		templateUses := map[string]map[string]templateMemberInfo{
			"fmt": {"Sprintf": {isCall: true}},
		}
		ar.maybeAddTemplateOnlyUseDecl("fmt", false, false, true, templateUses, &useDecls)
		assert.Empty(t, useDecls)
	})

	t.Run("does not add use decl when no members exist for alias", func(t *testing.T) {
		t.Parallel()
		var useDecls []goast.Decl
		templateUses := map[string]map[string]templateMemberInfo{
			"fmt": {},
		}
		ar.maybeAddTemplateOnlyUseDecl("fmt", false, false, true, templateUses, &useDecls)
		assert.Empty(t, useDecls)
	})
}

func TestRecordExpressionUsage(t *testing.T) {
	t.Parallel()
	ar := &astRewriter{}

	t.Run("records member access usage", func(t *testing.T) {
		t.Parallel()
		uses := make(map[string]map[string]templateMemberInfo)
		expression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "fmt"},
			Property: &ast_domain.Identifier{Name: "Sprintf"},
		}
		ar.recordExpressionUsage(expression, uses)

		require.Contains(t, uses, "fmt")
		require.Contains(t, uses["fmt"], "Sprintf")
		assert.False(t, uses["fmt"]["Sprintf"].isCall)
	})

	t.Run("records call expression usage as call", func(t *testing.T) {
		t.Parallel()
		uses := make(map[string]map[string]templateMemberInfo)
		expression := &ast_domain.CallExpression{
			Callee: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "util"},
				Property: &ast_domain.Identifier{Name: "Format"},
			},
		}
		ar.recordExpressionUsage(expression, uses)

		require.Contains(t, uses, "util")
		require.Contains(t, uses["util"], "Format")
		assert.True(t, uses["util"]["Format"].isCall)
	})

	t.Run("records bare identifier with no member", func(t *testing.T) {
		t.Parallel()
		uses := make(map[string]map[string]templateMemberInfo)
		expression := &ast_domain.Identifier{Name: "myVar"}
		ar.recordExpressionUsage(expression, uses)

		require.Contains(t, uses, "myVar")
		assert.Empty(t, uses["myVar"], "No members should be recorded for bare identifier")
	})

	t.Run("nil root does not record anything", func(t *testing.T) {
		t.Parallel()
		uses := make(map[string]map[string]templateMemberInfo)

		expression := &ast_domain.CallExpression{
			Callee: &ast_domain.CallExpression{
				Callee: &ast_domain.CallExpression{
					Callee: &ast_domain.MemberExpression{
						Base: &ast_domain.CallExpression{
							Callee: &ast_domain.Identifier{Name: "factory"},
						},
						Property: &ast_domain.Identifier{Name: "create"},
					},
				},
			},
		}
		ar.recordExpressionUsage(expression, uses)

		require.Contains(t, uses, "factory")
	})
}

func TestCollectShadowedFromNode(t *testing.T) {
	t.Parallel()

	t.Run("detects shadowed var declaration", func(t *testing.T) {
		t.Parallel()
		ar := &astRewriter{
			pikoAliasToHash: map[string]string{
				"card": "card_abc123",
			},
		}
		shadowedSet := make(map[string]bool)

		node := &goast.ValueSpec{
			Names: []*goast.Ident{goast.NewIdent("card")},
		}
		ar.collectShadowedFromNode(node, shadowedSet)
		assert.True(t, shadowedSet["card"])
	})

	t.Run("detects shadowed range statement key", func(t *testing.T) {
		t.Parallel()
		ar := &astRewriter{
			pikoAliasToHash: map[string]string{
				"card": "card_abc123",
			},
		}
		shadowedSet := make(map[string]bool)

		node := &goast.RangeStmt{
			Key:   goast.NewIdent("card"),
			Value: goast.NewIdent("value"),
		}
		ar.collectShadowedFromNode(node, shadowedSet)
		assert.True(t, shadowedSet["card"])
	})

	t.Run("detects shadowed range statement value", func(t *testing.T) {
		t.Parallel()
		ar := &astRewriter{
			pikoAliasToHash: map[string]string{
				"card": "card_abc123",
			},
		}
		shadowedSet := make(map[string]bool)

		node := &goast.RangeStmt{
			Key:   goast.NewIdent("index"),
			Value: goast.NewIdent("card"),
		}
		ar.collectShadowedFromNode(node, shadowedSet)
		assert.True(t, shadowedSet["card"])
	})

	t.Run("detects shadowed for statement init", func(t *testing.T) {
		t.Parallel()
		ar := &astRewriter{
			pikoAliasToHash: map[string]string{
				"card": "card_abc123",
			},
		}
		shadowedSet := make(map[string]bool)

		node := &goast.ForStmt{
			Init: &goast.AssignStmt{
				Lhs: []goast.Expr{goast.NewIdent("card")},
				Tok: token.DEFINE,
				Rhs: []goast.Expr{goast.NewIdent("0")},
			},
		}
		ar.collectShadowedFromNode(node, shadowedSet)
		assert.True(t, shadowedSet["card"])
	})

	t.Run("for statement with non-define assign does not shadow", func(t *testing.T) {
		t.Parallel()
		ar := &astRewriter{
			pikoAliasToHash: map[string]string{
				"card": "card_abc123",
			},
		}
		shadowedSet := make(map[string]bool)

		node := &goast.ForStmt{
			Init: &goast.AssignStmt{
				Lhs: []goast.Expr{goast.NewIdent("card")},
				Tok: token.ASSIGN,
				Rhs: []goast.Expr{goast.NewIdent("0")},
			},
		}
		ar.collectShadowedFromNode(node, shadowedSet)
		assert.False(t, shadowedSet["card"])
	})

	t.Run("for statement with non-assign init does not shadow", func(t *testing.T) {
		t.Parallel()
		ar := &astRewriter{
			pikoAliasToHash: map[string]string{
				"card": "card_abc123",
			},
		}
		shadowedSet := make(map[string]bool)

		node := &goast.ForStmt{
			Init: &goast.ExprStmt{
				X: goast.NewIdent("card"),
			},
		}
		ar.collectShadowedFromNode(node, shadowedSet)
		assert.Empty(t, shadowedSet)
	})

	t.Run("func decl with nil type does not panic", func(t *testing.T) {
		t.Parallel()
		ar := &astRewriter{
			pikoAliasToHash: map[string]string{
				"card": "card_abc123",
			},
		}
		shadowedSet := make(map[string]bool)

		node := &goast.FuncDecl{
			Name: goast.NewIdent("test"),
			Type: nil,
		}
		assert.NotPanics(t, func() {
			ar.collectShadowedFromNode(node, shadowedSet)
		})
	})

	t.Run("func decl params shadow aliases", func(t *testing.T) {
		t.Parallel()
		ar := &astRewriter{
			pikoAliasToHash: map[string]string{
				"card": "card_abc123",
			},
		}
		shadowedSet := make(map[string]bool)

		node := &goast.FuncDecl{
			Name: goast.NewIdent("test"),
			Type: &goast.FuncType{
				Params: &goast.FieldList{
					List: []*goast.Field{
						{Names: []*goast.Ident{goast.NewIdent("card")}},
					},
				},
			},
		}
		ar.collectShadowedFromNode(node, shadowedSet)
		assert.True(t, shadowedSet["card"])
	})
}

func TestRewritePackageName(t *testing.T) {
	t.Parallel()

	t.Run("sets package name to hashed name", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", `package original`, 0)
		require.NoError(t, err)

		ar := &astRewriter{
			ast: f,
			vc: &annotator_dto.VirtualComponent{
				HashedName: "hashed_abc123",
			},
		}
		ar.rewritePackageName()
		assert.Equal(t, "hashed_abc123", f.Name.Name)
	})

	t.Run("creates package name ident when nil", func(t *testing.T) {
		t.Parallel()
		f := &goast.File{Name: nil}

		ar := &astRewriter{
			ast: f,
			vc: &annotator_dto.VirtualComponent{
				HashedName: "hashed_abc123",
			},
		}
		ar.rewritePackageName()
		require.NotNil(t, f.Name)
		assert.Equal(t, "hashed_abc123", f.Name.Name)
	})
}

func TestRebuildDecls(t *testing.T) {
	t.Parallel()

	t.Run("rebuilds decls with imports and use decls", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", `package main; import "fmt"; func Foo() {}`, 0)
		require.NoError(t, err)

		ar := &astRewriter{ast: f}

		finalImports := map[string]*goast.ImportSpec{
			"strings": createImportSpec("strings", ""),
		}
		useDecl := createUseDecl("strings", "Join")
		ar.rebuildDecls(finalImports, []goast.Decl{useDecl})

		require.True(t, len(f.Decls) >= 3)
		gen, ok := f.Decls[0].(*goast.GenDecl)
		require.True(t, ok)
		assert.Equal(t, token.IMPORT, gen.Tok)
	})

	t.Run("rebuilds decls without imports when empty", func(t *testing.T) {
		t.Parallel()
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", `package main; import "fmt"; func Foo() {}`, 0)
		require.NoError(t, err)

		ar := &astRewriter{ast: f}
		ar.rebuildDecls(map[string]*goast.ImportSpec{}, nil)

		for _, declaration := range f.Decls {
			if gen, ok := declaration.(*goast.GenDecl); ok {
				assert.NotEqual(t, token.IMPORT, gen.Tok, "Should not have import declaration when empty")
			}
		}
	})
}

func TestDynamicParamRegex(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		input       string
		replacement string
		expected    string
	}{
		{
			name:        "replaces single param",
			input:       "pages/blog/{slug}.pk",
			replacement: "hello-world",
			expected:    "pages/blog/hello-world.pk",
		},
		{
			name:        "replaces multiple params",
			input:       "pages/{category}/{slug}.pk",
			replacement: "test",
			expected:    "pages/test/test.pk",
		},
		{
			name:        "no params leaves string unchanged",
			input:       "pages/index.pk",
			replacement: "test",
			expected:    "pages/index.pk",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := dynamicParamRegex.ReplaceAllString(tc.input, tc.replacement)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVirtualiserWithEmailAndPartialEntryPoints(t *testing.T) {
	t.Parallel()

	baseDir := "/project"
	moduleName := "my-module"

	t.Run("should mark email components correctly", func(t *testing.T) {
		t.Parallel()
		h := newVirtualiserTestHarness(t, moduleName, baseDir)
		h.pathsConfig.PartialsSourceDir = "partials"
		h.pathsConfig.PartialServePath = "/_piko/partials"

		emailPath := filepath.Join(baseDir, "emails", "welcome.pk")
		h.addComponent(emailPath, `package welcome`)

		entryPoints := []annotator_dto.EntryPoint{
			{
				Path:    emailPath,
				IsEmail: true,
			},
		}

		virtualiser := NewModuleVirtualiser(h.resolver, h.pathsConfig)
		result, err := virtualiser.Virtualise(context.Background(), h.graph, h.originalGoFiles, entryPoints)
		require.NoError(t, err)

		emailHash := buildAliasFromPath(emailPath)
		emailComp := result.ComponentsByHash[emailHash]
		require.NotNil(t, emailComp)
		assert.True(t, emailComp.IsEmail, "Component should be marked as email")
		assert.Contains(t, emailComp.CanonicalGoPackagePath, "dist/emails")
	})

	t.Run("should mark partial components correctly and resolve partial paths", func(t *testing.T) {
		t.Parallel()
		h := newVirtualiserTestHarness(t, moduleName, baseDir)
		h.pathsConfig.PartialsSourceDir = "partials"
		h.pathsConfig.PartialServePath = "/_piko/partials"

		partialPath := filepath.Join(baseDir, "partials", "card.pk")
		h.addComponent(partialPath, `package card`)

		entryPoints := []annotator_dto.EntryPoint{
			{
				Path:   partialPath,
				IsPage: false,
			},
		}

		virtualiser := NewModuleVirtualiser(h.resolver, h.pathsConfig)
		result, err := virtualiser.Virtualise(context.Background(), h.graph, h.originalGoFiles, entryPoints)
		require.NoError(t, err)

		partialHash := buildAliasFromPath(partialPath)
		partialComp := result.ComponentsByHash[partialHash]
		require.NotNil(t, partialComp)
		assert.False(t, partialComp.IsPage, "Component should not be marked as page")
		assert.False(t, partialComp.IsEmail, "Component should not be marked as email")
		assert.Contains(t, partialComp.CanonicalGoPackagePath, "dist/partials")
		assert.NotEmpty(t, partialComp.PartialName, "Partial name should be set")
		assert.NotEmpty(t, partialComp.PartialSrc, "Partial source path should be set")
	})
}

func TestVirtualiserWithVirtualPageSource(t *testing.T) {
	t.Parallel()

	baseDir := "/project"
	moduleName := "my-module"

	h := newVirtualiserTestHarness(t, moduleName, baseDir)
	h.pathsConfig.PartialsSourceDir = "partials"
	h.pathsConfig.PartialServePath = "/_piko/partials"

	blogPath := filepath.Join(baseDir, "pages", "blog", "{slug}.pk")
	h.addComponent(blogPath, `package blog`)

	entryPoints := []annotator_dto.EntryPoint{
		{
			Path:   blogPath,
			IsPage: true,
			VirtualPageSource: &annotator_dto.VirtualPageSource{
				RouteOverride: "/blog/test-post",
				TemplatePath:  blogPath,
				InitialProps: map[string]any{
					"page": map[string]any{
						collection_dto.MetaKeyURL:  "/blog/test-post",
						collection_dto.MetaKeySlug: "test-post",
					},
				},
			},
		},
	}

	virtualiser := NewModuleVirtualiser(h.resolver, h.pathsConfig)
	result, err := virtualiser.Virtualise(context.Background(), h.graph, h.originalGoFiles, entryPoints)
	require.NoError(t, err)

	blogHash := buildAliasFromPath(blogPath)
	blogComp := result.ComponentsByHash[blogHash]
	require.NotNil(t, blogComp)
	assert.True(t, blogComp.IsPage)
	require.Len(t, blogComp.VirtualInstances, 1)
	assert.Equal(t, "/blog/test-post", blogComp.VirtualInstances[0].Route)
	assert.Equal(t, "pages/blog/test-post.pk", blogComp.VirtualInstances[0].ManifestKey)
}

func TestDiscoverTemplateUses(t *testing.T) {
	t.Parallel()

	t.Run("should return empty map when template is nil", func(t *testing.T) {
		t.Parallel()

		ar := &astRewriter{
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					Template: nil,
				},
			},
		}

		uses := ar.discoverTemplateUses()

		assert.Empty(t, uses)
	})

	t.Run("should discover package usage from directive expressions", func(t *testing.T) {
		t.Parallel()

		templateAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					DirIf: &ast_domain.Directive{
						Expression: &ast_domain.MemberExpression{
							Base:     &ast_domain.Identifier{Name: "utils"},
							Property: &ast_domain.Identifier{Name: "IsActive"},
						},
					},
				},
			},
		}

		ar := &astRewriter{
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					Template: templateAST,
				},
			},
		}

		uses := ar.discoverTemplateUses()

		require.Contains(t, uses, "utils")
		require.Contains(t, uses["utils"], "IsActive")
		assert.False(t, uses["utils"]["IsActive"].isCall)
	})

	t.Run("should discover package usage from call expressions in template", func(t *testing.T) {
		t.Parallel()

		templateAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					DirText: &ast_domain.Directive{
						Expression: &ast_domain.CallExpression{
							Callee: &ast_domain.MemberExpression{
								Base:     &ast_domain.Identifier{Name: "fmt"},
								Property: &ast_domain.Identifier{Name: "Sprintf"},
							},
						},
					},
				},
			},
		}

		ar := &astRewriter{
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					Template: templateAST,
				},
			},
		}

		uses := ar.discoverTemplateUses()

		require.Contains(t, uses, "fmt")
		require.Contains(t, uses["fmt"], "Sprintf")
		assert.True(t, uses["fmt"]["Sprintf"].isCall)
	})
}

func TestResolveImportPath(t *testing.T) {
	t.Parallel()

	t.Run("should return path and alias for non-piko imports", func(t *testing.T) {
		t.Parallel()

		ar := &astRewriter{}

		canonicalPath, targetName, err := ar.resolveImportPath(context.Background(), "fmt", "fmt", false)

		require.NoError(t, err)
		assert.Equal(t, "fmt", canonicalPath)
		assert.Equal(t, "fmt", targetName)
	})

	t.Run("should return path and alias for non-piko import with custom alias", func(t *testing.T) {
		t.Parallel()

		ar := &astRewriter{}

		canonicalPath, targetName, err := ar.resolveImportPath(context.Background(), "encoding/json", "json", false)

		require.NoError(t, err)
		assert.Equal(t, "encoding/json", canonicalPath)
		assert.Equal(t, "json", targetName)
	})

	t.Run("should delegate to resolvePikoImport for piko imports", func(t *testing.T) {
		t.Parallel()

		baseDir := "/project"
		cardPath := filepath.Join(baseDir, "partials", "card.pk")
		cardHash := buildAliasFromPath(cardPath)

		graph := &annotator_dto.ComponentGraph{
			PathToHashedName: map[string]string{cardPath: cardHash},
		}

		resolver := newMockResolver("my-module", baseDir)

		virtualModule := &annotator_dto.VirtualModule{
			Graph:            graph,
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
		}
		virtualModule.ComponentsByHash[cardHash] = &annotator_dto.VirtualComponent{
			HashedName:             cardHash,
			CanonicalGoPackagePath: "my-module/" + cardHash,
		}

		vCtx := &virtualisationContext{
			resolver:      resolver,
			graph:         graph,
			virtualModule: virtualModule,
		}

		ar := &astRewriter{
			vCtx: vCtx,
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					SourcePath: filepath.Join(baseDir, "pages", "main.pk"),
				},
			},
		}

		canonicalPath, targetName, err := ar.resolveImportPath(context.Background(), "partials/card.pk", "card", true)

		require.NoError(t, err)
		assert.Equal(t, "my-module/"+cardHash, canonicalPath)
		assert.Equal(t, cardHash, targetName)
	})
}

func TestResolvePikoImport(t *testing.T) {
	t.Parallel()

	baseDir := "/project"

	t.Run("should resolve a valid piko import", func(t *testing.T) {
		t.Parallel()

		cardPath := filepath.Join(baseDir, "partials", "card.pk")
		cardHash := buildAliasFromPath(cardPath)

		graph := &annotator_dto.ComponentGraph{
			PathToHashedName: map[string]string{cardPath: cardHash},
		}

		resolver := newMockResolver("my-module", baseDir)

		virtualModule := &annotator_dto.VirtualModule{
			Graph:            graph,
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
		}
		virtualModule.ComponentsByHash[cardHash] = &annotator_dto.VirtualComponent{
			HashedName:             cardHash,
			CanonicalGoPackagePath: "my-module/" + cardHash,
		}

		vCtx := &virtualisationContext{
			resolver:      resolver,
			graph:         graph,
			virtualModule: virtualModule,
		}

		ar := &astRewriter{
			vCtx: vCtx,
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					SourcePath: filepath.Join(baseDir, "pages", "main.pk"),
				},
			},
		}

		canonicalPath, hashedName, err := ar.resolvePikoImport(context.Background(), "partials/card.pk")

		require.NoError(t, err)
		assert.Equal(t, "my-module/"+cardHash, canonicalPath)
		assert.Equal(t, cardHash, hashedName)
	})

	t.Run("should return error when path cannot be resolved", func(t *testing.T) {
		t.Parallel()

		failResolver := &mockResolverWithFailures{
			baseDir:    baseDir,
			moduleName: "my-module",
			failPaths:  map[string]bool{"nonexistent.pk": true},
		}

		vCtx := &virtualisationContext{
			resolver: failResolver,
		}

		ar := &astRewriter{
			vCtx: vCtx,
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					SourcePath: filepath.Join(baseDir, "main.pk"),
				},
			},
		}

		_, _, err := ar.resolvePikoImport(context.Background(), "nonexistent.pk")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not resolve import")
	})

	t.Run("should return error when hash is not found in graph", func(t *testing.T) {
		t.Parallel()

		cardPath := filepath.Join(baseDir, "partials", "card.pk")

		graph := &annotator_dto.ComponentGraph{
			PathToHashedName: map[string]string{},
		}

		resolver := newMockResolver("my-module", baseDir)

		vCtx := &virtualisationContext{
			resolver: resolver,
			graph:    graph,
		}

		ar := &astRewriter{
			vCtx: vCtx,
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					SourcePath: filepath.Join(baseDir, "main.pk"),
				},
			},
		}

		_, _, err := ar.resolvePikoImport(context.Background(), "partials/card.pk")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not find hash for resolved path")
		_ = cardPath
	})

	t.Run("should return error when virtual component is not found", func(t *testing.T) {
		t.Parallel()

		cardPath := filepath.Join(baseDir, "partials", "card.pk")
		cardHash := buildAliasFromPath(cardPath)

		graph := &annotator_dto.ComponentGraph{
			PathToHashedName: map[string]string{cardPath: cardHash},
		}

		resolver := newMockResolver("my-module", baseDir)

		virtualModule := &annotator_dto.VirtualModule{
			Graph:            graph,
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
		}

		vCtx := &virtualisationContext{
			resolver:      resolver,
			graph:         graph,
			virtualModule: virtualModule,
		}

		ar := &astRewriter{
			vCtx: vCtx,
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					SourcePath: filepath.Join(baseDir, "main.pk"),
				},
			},
		}

		_, _, err := ar.resolvePikoImport(context.Background(), "partials/card.pk")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not find virtual component for hash")
	})
}

type mockResolverWithFailures struct {
	failPaths  map[string]bool
	baseDir    string
	moduleName string
}

func (m *mockResolverWithFailures) DetectLocalModule(_ context.Context) error { return nil }
func (m *mockResolverWithFailures) GetBaseDir() string                        { return m.baseDir }
func (m *mockResolverWithFailures) GetModuleName() string                     { return m.moduleName }
func (m *mockResolverWithFailures) ResolvePKPath(_ context.Context, importPath string, _ string) (string, error) {
	if m.failPaths[importPath] {
		return "", fmt.Errorf("resolution failed for: %s", importPath)
	}
	if filepath.IsAbs(importPath) {
		return importPath, nil
	}
	return filepath.Join(m.baseDir, importPath), nil
}
func (m *mockResolverWithFailures) ResolveCSSPath(_ context.Context, importPath, containingDir string) (string, error) {
	return filepath.Join(containingDir, importPath), nil
}
func (m *mockResolverWithFailures) ConvertEntryPointPathToManifestKey(ep string) string { return ep }
func (m *mockResolverWithFailures) ResolveAssetPath(_ context.Context, importPath string, _ string) (string, error) {
	return filepath.Join(m.baseDir, importPath), nil
}
func (*mockResolverWithFailures) GetModuleDir(_ context.Context, _ string) (string, error) {
	return "", errors.New("not implemented")
}
func (*mockResolverWithFailures) FindModuleBoundary(_ context.Context, _ string) (string, string, error) {
	return "", "", errors.New("not implemented")
}

var _ resolver_domain.ResolverPort = (*mockResolverWithFailures)(nil)
