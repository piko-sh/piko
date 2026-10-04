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
	goast "go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/sfcparser"
)

func TestParsePK(t *testing.T) {
	t.Run("should parse a complete and valid pk file", func(t *testing.T) {
		source := `
<script type="application/x-go">
package main
import card "partials/card.pk"
import "fmt"

type Props struct {
   Name string
}

func Render(p Props) (*string, error) {
   return nil, nil
}
</script>

<template>
   <div>Hello</div>
</template>

<style scoped>
.container { color: red; }
</style>

<style global>
body { margin: 0; }
</style>

<i18n lang="json">
{
 "en": { "greeting": "Hello" },
 "fr": { "greeting": "Bonjour" }
}
</i18n>
`
		parsedComponent, srcs, err := ParsePK(context.Background(), []byte(source), "/project/src/main.pk")
		require.NoError(t, err)
		require.NotNil(t, parsedComponent)

		assert.Contains(t, srcs.TemplateSource, "<div>Hello</div>")
		assert.Contains(t, srcs.ScriptSource, "package main")
		require.Len(t, srcs.StyleBlocks, 2)
		assert.Contains(t, srcs.StyleBlocks[0].Content, ".container")
		assert.Contains(t, srcs.StyleBlocks[1].Content, "body")

		assert.Equal(t, "/project/src/main.pk", parsedComponent.SourcePath)
		assert.NotNil(t, parsedComponent.Template)
		require.Len(t, parsedComponent.StyleBlocks, 2)
		assert.Contains(t, parsedComponent.StyleBlocks[0].Attributes, "scoped")
		assert.Contains(t, parsedComponent.StyleBlocks[1].Attributes, "global")

		require.NotNil(t, parsedComponent.LocalTranslations)
		assert.Equal(t, "Hello", parsedComponent.LocalTranslations["en"]["greeting"])
		assert.Equal(t, "Bonjour", parsedComponent.LocalTranslations["fr"]["greeting"])

		require.Len(t, parsedComponent.PikoImports, 1, "Should find exactly one .pk import")
		pikoImport := parsedComponent.PikoImports[0]
		assert.Equal(t, "card", pikoImport.Alias)
		assert.Equal(t, "partials/card.pk", pikoImport.Path)
		assert.NotZero(t, pikoImport.Location.Line, "Import location should be populated")
		assert.NotZero(t, pikoImport.Location.Column, "Import location should be populated")

		require.NotNil(t, parsedComponent.Script)
		assert.Equal(t, "main", parsedComponent.Script.GoPackageName)
		require.NotNil(t, parsedComponent.Script.AST)
		require.NotNil(t, parsedComponent.Script.PropsTypeExpression)
		require.NotNil(t, parsedComponent.Script.RenderReturnTypeExpression)
		assert.NotZero(t, parsedComponent.Script.ScriptStartLocation.Line)

		require.Len(t, parsedComponent.Script.AST.Imports, 1, "Script AST should only contain standard Go imports")
		assert.Equal(t, `"fmt"`, parsedComponent.Script.AST.Imports[0].Path.Value)
	})

	t.Run("should handle file with no script block gracefully", func(t *testing.T) {
		source := `<template><div>Just a template</div></template>`
		parsedComponent, _, err := ParsePK(context.Background(), []byte(source), "test.pk")
		require.NoError(t, err)
		require.NotNil(t, parsedComponent)

		require.NotNil(t, parsedComponent.Script, "A default script block should be created")
		assert.Equal(t, "piko_default", parsedComponent.Script.GoPackageName)
		assert.Empty(t, parsedComponent.PikoImports)
	})

	t.Run("should handle template-only file", func(t *testing.T) {
		source := `<template><div>Content</div></template>`
		parsedComponent, _, err := ParsePK(context.Background(), []byte(source), "test.pk")
		require.NoError(t, err)
		require.NotNil(t, parsedComponent)
		assert.NotNil(t, parsedComponent.Template)
		require.NotNil(t, parsedComponent.Script)
		assert.Equal(t, "piko_default", parsedComponent.Script.GoPackageName)
	})

	t.Run("should handle script-only file", func(t *testing.T) {
		source := `<script type="application/x-go">package main; func Test() {}</script>`
		parsedComponent, _, err := ParsePK(context.Background(), []byte(source), "test.pk")
		require.NoError(t, err)
		require.NotNil(t, parsedComponent)
		assert.Nil(t, parsedComponent.Template)
		require.NotNil(t, parsedComponent.Script)
		assert.Equal(t, "main", parsedComponent.Script.GoPackageName)
	})

	t.Run("should handle empty file", func(t *testing.T) {
		source := ``
		parsedComponent, _, err := ParsePK(context.Background(), []byte(source), "test.pk")
		require.NoError(t, err)
		require.NotNil(t, parsedComponent)
		assert.Nil(t, parsedComponent.Template)
		require.NotNil(t, parsedComponent.Script)
		assert.Equal(t, "piko_default", parsedComponent.Script.GoPackageName)
		assert.Empty(t, parsedComponent.StyleBlocks)
		assert.Empty(t, parsedComponent.LocalTranslations)
	})

	t.Run("should return a diagnostic error for invalid template syntax", func(t *testing.T) {
		source := `<template><div>{{ unterminated</div></template>`
		_, _, err := ParsePK(context.Background(), []byte(source), "test.pk")
		require.Error(t, err, "An error is expected for invalid template syntax")

		var diagErr *ParseDiagnosticError
		require.ErrorAs(t, err, &diagErr, "Error should be of type ParseDiagnosticError")
		assert.NotEmpty(t, diagErr.Diagnostics, "Diagnostics list should not be empty")
		assert.Contains(t, diagErr.Diagnostics[0].Message, "Unterminated text interpolation")
	})

	t.Run("should return an error for invalid Go syntax in script block", func(t *testing.T) {
		source := `<script type="application/x-go">package main; func Render() {</script>`
		_, _, err := ParsePK(context.Background(), []byte(source), "test.pk")
		require.Error(t, err, "An error is expected for invalid Go syntax")

		var scriptErr *scriptBlockParseError
		require.ErrorAs(t, err, &scriptErr, "Error should be of type scriptBlockParseError")
		assert.Contains(t, scriptErr.reason, "found 'EOF'")
	})

	t.Run("should resolve a collection source alias from the script imports", func(t *testing.T) {
		source := `<template p-collection="docs" p-collection-source="docs"><div></div></template>
<script type="application/x-go">
package main
import _ "github.com/myorg/docs/v2"
import "github.com/myorg/docs/v2"
</script>`
		parsedComponent, _, err := ParsePK(context.Background(), []byte(source), "test.pk")
		require.NoError(t, err)
		require.NotNil(t, parsedComponent)
		assert.True(t, parsedComponent.HasCollection)
		assert.Equal(t, "github.com/myorg/docs/v2", parsedComponent.ContentModulePath)
	})

	t.Run("should return a diagnostic error for an unresolvable collection source alias", func(t *testing.T) {
		source := `<template p-collection="docs" p-collection-source="docz"><div></div></template>
<script type="application/x-go">
package main
import docs "github.com/myorg/docs"
</script>`
		parsedComponent, _, err := ParsePK(context.Background(), []byte(source), "test.pk")
		require.Error(t, err)
		require.NotNil(t, parsedComponent)
		assert.Empty(t, parsedComponent.ContentModulePath)

		var diagErr *ParseDiagnosticError
		require.ErrorAs(t, err, &diagErr)
		require.Len(t, diagErr.Diagnostics, 1)
		assert.Equal(t, annotator_dto.CodeCollectionSourceNotFound, diagErr.Diagnostics[0].Code)
		assert.Equal(t, ast_domain.Error, diagErr.Diagnostics[0].Severity)
		assert.Equal(t, 1, diagErr.Diagnostics[0].Location.Line)
		require.NotNil(t, parsedComponent.Template)
		assert.Contains(t, parsedComponent.Template.Diagnostics, diagErr.Diagnostics[0])
	})

	t.Run("should report the script error rather than the collection source when the script is invalid", func(t *testing.T) {
		source := `<template p-collection="docs" p-collection-source="docs"><div></div></template>
<script type="application/x-go">package main; func Render() {</script>`
		_, _, err := ParsePK(context.Background(), []byte(source), "test.pk")
		require.Error(t, err)

		var scriptErr *scriptBlockParseError
		require.ErrorAs(t, err, &scriptErr)
	})

	t.Run("should return an error for invalid i18n JSON", func(t *testing.T) {
		source := `<i18n lang="json">{"en": {"greeting": "Hello",}}</i18n>`
		_, _, err := ParsePK(context.Background(), []byte(source), "test.pk")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse i18n JSON block")
	})
}

func TestAnalyseGoScript(t *testing.T) {
	type testCase struct {
		checkResult           func(t *testing.T, parsed *annotator_dto.ParsedScript, pikoImports []annotator_dto.PikoImport)
		description           string
		script                string
		expectedErrorContains string
		expectedPackageName   string
		expectedGoImports     int
		expectedPikoImports   int
		expectError           bool
	}

	testCases := []testCase{
		{
			description:         "Empty script block returns default struct",
			script:              ``,
			expectedPackageName: "piko_default",
			checkResult: func(t *testing.T, parsed *annotator_dto.ParsedScript, pikoImports []annotator_dto.PikoImport) {
				require.NotNil(t, parsed)
				assert.Equal(t, "piko_default", parsed.GoPackageName)
				assert.Empty(t, pikoImports)
			},
		},
		{
			description:         "Script with only comments returns default struct",
			script:              `// package main`,
			expectedPackageName: "piko_default",
			checkResult: func(t *testing.T, parsed *annotator_dto.ParsedScript, pikoImports []annotator_dto.PikoImport) {
				require.NotNil(t, parsed)
				assert.Equal(t, "piko_default", parsed.GoPackageName)
			},
		},
		{
			description:         "Script with only standard Go imports",
			script:              `package main; import "fmt"; import "context"`,
			expectedPackageName: "main",
			expectedGoImports:   2,
			checkResult: func(t *testing.T, parsed *annotator_dto.ParsedScript, pikoImports []annotator_dto.PikoImport) {
				require.NotNil(t, parsed.AST)
				require.Len(t, parsed.AST.Imports, 2)
				assert.Equal(t, `"fmt"`, parsed.AST.Imports[0].Path.Value)
				assert.Equal(t, `"context"`, parsed.AST.Imports[1].Path.Value)
			},
		},
		{
			description:         "Script with only piko imports",
			script:              `package main; import card "card.pk"; import _ "button.pk"`,
			expectedPackageName: "main",
			expectedPikoImports: 2,
			checkResult: func(t *testing.T, parsed *annotator_dto.ParsedScript, pikoImports []annotator_dto.PikoImport) {
				require.NotNil(t, parsed.AST, "AST should still be parsed")
				assert.Empty(t, parsed.AST.Imports, "AST import list should be empty after filtering")
				require.Len(t, pikoImports, 2)
				assert.Equal(t, "card", pikoImports[0].Alias)
				assert.Equal(t, "card.pk", pikoImports[0].Path)
				assert.NotZero(t, pikoImports[0].Location.Line)
				assert.Equal(t, "_", pikoImports[1].Alias)
				assert.Equal(t, "button.pk", pikoImports[1].Path)
				assert.NotZero(t, pikoImports[1].Location.Line)
			},
		},
		{
			description: "Full script with Render, Props, and all lifecycle funcs",
			script: `
               package main
			   import "context"
			   import "piko.sh/piko/internal/templater/templater_dto"

               type Props struct { Name string }
               func Render(ctx context.Context, props Props) (*string, error) { return nil, nil }
               func Middlewares() []string { return nil }
			   func CachePolicy() templater_dto.CachePolicy { return templater_dto.CachePolicy{} }
			   func SupportedLocales() []string { return nil }
           `,
			expectedPackageName: "main",
			expectedGoImports:   2,
			checkResult: func(t *testing.T, parsed *annotator_dto.ParsedScript, pikoImports []annotator_dto.PikoImport) {
				assert.True(t, parsed.HasMiddleware)
				assert.Equal(t, "Middlewares", parsed.MiddlewaresFuncName)
				assert.True(t, parsed.HasCachePolicy)
				assert.Equal(t, "CachePolicy", parsed.CachePolicyFuncName)
				assert.True(t, parsed.HasSupportedLocales)
				assert.Equal(t, "SupportedLocales", parsed.SupportedLocalesFuncName)

				require.NotNil(t, parsed.RenderReturnTypeExpression)
				assert.IsType(t, &goast.StarExpr{}, parsed.RenderReturnTypeExpression)
				require.NotNil(t, parsed.PropsTypeExpression)
				assert.Equal(t, "Props", parsed.PropsTypeExpression.(*goast.Ident).Name)
			},
		},
		{
			description: "Script with Preview convention function",
			script: `
               package main
               type Scenario struct { Name string }
               func Preview() []Scenario { return nil }
           `,
			expectedPackageName: "main",
			checkResult: func(t *testing.T, parsed *annotator_dto.ParsedScript, _ []annotator_dto.PikoImport) {
				assert.True(t, parsed.HasPreview, "HasPreview should be true")
				assert.Equal(t, "Preview", parsed.PreviewFuncName, "PreviewFuncName mismatch")
			},
		},
		{
			description: "Script without Preview has HasPreview false",
			script: `
               package main
               func Render() (*string, error) { return nil, nil }
           `,
			expectedPackageName: "main",
			checkResult: func(t *testing.T, parsed *annotator_dto.ParsedScript, _ []annotator_dto.PikoImport) {
				assert.False(t, parsed.HasPreview, "HasPreview should be false when no Preview func")
				assert.Empty(t, parsed.PreviewFuncName, "PreviewFuncName should be empty")
			},
		},
		{
			description:           "Invalid Go syntax",
			script:                `package main func Render() {}`,
			expectError:           true,
			expectedErrorContains: "expected ';', found 'func'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result, pikoImports, err := analyseGoScript(tc.script, ast_domain.Location{Line: 1, Column: 1, Offset: 0})

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrorContains)
				return
			}

			require.NoError(t, err)

			if tc.script != "" {
				require.NotNil(t, result)
				assert.Equal(t, tc.expectedPackageName, result.GoPackageName)
				assert.Len(t, pikoImports, tc.expectedPikoImports)

				if result.AST != nil {
					assert.Len(t, result.AST.Imports, tc.expectedGoImports)
				}
			}

			if tc.checkResult != nil {
				tc.checkResult(t, result, pikoImports)
			}
		})
	}
}

func TestValidateScriptBlocks(t *testing.T) {
	t.Parallel()

	t.Run("should return no diagnostics for recognised Go script blocks", func(t *testing.T) {
		t.Parallel()

		sfcResult := &sfcparser.ParseResult{
			Scripts: []sfcparser.Script{
				{Attributes: map[string]string{"type": "application/x-go"}},
			},
		}

		diagnostics := validateScriptBlocks(sfcResult, "/test.pk")

		assert.Empty(t, diagnostics)
	})

	t.Run("should return no diagnostics for recognised TypeScript script blocks", func(t *testing.T) {
		t.Parallel()

		sfcResult := &sfcparser.ParseResult{
			Scripts: []sfcparser.Script{
				{Attributes: map[string]string{"lang": "ts"}},
			},
		}

		diagnostics := validateScriptBlocks(sfcResult, "/test.pk")

		assert.Empty(t, diagnostics)
	})

	t.Run("should return no diagnostics for recognised JavaScript script blocks", func(t *testing.T) {
		t.Parallel()

		sfcResult := &sfcparser.ParseResult{
			Scripts: []sfcparser.Script{
				{Attributes: map[string]string{"lang": "js"}},
			},
		}

		diagnostics := validateScriptBlocks(sfcResult, "/test.pk")

		assert.Empty(t, diagnostics)
	})

	t.Run("should warn about unrecognised lang attribute", func(t *testing.T) {
		t.Parallel()

		sfcResult := &sfcparser.ParseResult{
			Scripts: []sfcparser.Script{
				{
					Attributes: map[string]string{"lang": "python"},
					Location:   sfcparser.Location{Line: 5, Column: 1},
				},
			},
		}

		diagnostics := validateScriptBlocks(sfcResult, "/test.pk")

		require.Len(t, diagnostics, 1)
		assert.Equal(t, ast_domain.Warning, diagnostics[0].Severity)
		assert.Contains(t, diagnostics[0].Message, `Unrecognised script block attribute lang="python"`)
	})

	t.Run("should warn about unrecognised type attribute", func(t *testing.T) {
		t.Parallel()

		sfcResult := &sfcparser.ParseResult{
			Scripts: []sfcparser.Script{
				{
					Attributes: map[string]string{"type": "text/python"},
					Location:   sfcparser.Location{Line: 3, Column: 2},
				},
			},
		}

		diagnostics := validateScriptBlocks(sfcResult, "/test.pk")

		require.Len(t, diagnostics, 1)
		assert.Equal(t, ast_domain.Warning, diagnostics[0].Severity)
		assert.Contains(t, diagnostics[0].Message, `Unrecognised script block attribute type="text/python"`)
	})

	t.Run("should warn about missing lang and type attributes", func(t *testing.T) {
		t.Parallel()

		sfcResult := &sfcparser.ParseResult{
			Scripts: []sfcparser.Script{
				{
					Attributes: map[string]string{},
					Location:   sfcparser.Location{Line: 1, Column: 1},
				},
			},
		}

		diagnostics := validateScriptBlocks(sfcResult, "/test.pk")

		require.Len(t, diagnostics, 1)
		assert.Equal(t, ast_domain.Warning, diagnostics[0].Severity)
		assert.Contains(t, diagnostics[0].Message, "Script block is missing a lang or type attribute")
	})

	t.Run("should handle multiple script blocks with mixed validity", func(t *testing.T) {
		t.Parallel()

		sfcResult := &sfcparser.ParseResult{
			Scripts: []sfcparser.Script{
				{Attributes: map[string]string{"type": "application/x-go"}},
				{Attributes: map[string]string{"lang": "ruby"}, Location: sfcparser.Location{Line: 10, Column: 1}},
				{Attributes: map[string]string{"type": "module"}},
			},
		}

		diagnostics := validateScriptBlocks(sfcResult, "/test.pk")

		require.Len(t, diagnostics, 1)
		assert.Contains(t, diagnostics[0].Message, `lang="ruby"`)
	})

	t.Run("should return no diagnostics for empty scripts list", func(t *testing.T) {
		t.Parallel()

		sfcResult := &sfcparser.ParseResult{
			Scripts: []sfcparser.Script{},
		}

		diagnostics := validateScriptBlocks(sfcResult, "/test.pk")

		assert.Empty(t, diagnostics)
	})
}

func TestBuildParsedComponent(t *testing.T) {
	t.Parallel()

	t.Run("should build a minimal component with nil template and script", func(t *testing.T) {
		t.Parallel()

		sfcResult := &sfcparser.ParseResult{
			Styles:             []sfcparser.Style{},
			TemplateAttributes: map[string]string{},
		}

		comp := buildParsedComponent(nil, nil, nil, "/test.pk", sfcResult, nil)

		assert.Equal(t, "/test.pk", comp.SourcePath)
		assert.Nil(t, comp.Template)
		assert.Nil(t, comp.Script)
		assert.Empty(t, comp.PikoImports)
		assert.Empty(t, comp.ClientScript)
		assert.False(t, comp.HasCollection)
	})

	t.Run("should include client script from sfcResult", func(t *testing.T) {
		t.Parallel()

		sfcResult := &sfcparser.ParseResult{
			Scripts: []sfcparser.Script{
				{
					Attributes: map[string]string{"lang": "js"},
					Content:    "console.log('hello')",
				},
			},
			TemplateAttributes: map[string]string{},
			Styles:             []sfcparser.Style{},
		}

		comp := buildParsedComponent(nil, nil, nil, "/test.pk", sfcResult, nil)

		assert.Equal(t, "console.log('hello')", comp.ClientScript)
	})

	t.Run("should preserve style blocks and piko imports", func(t *testing.T) {
		t.Parallel()

		sfcResult := &sfcparser.ParseResult{
			Styles: []sfcparser.Style{
				{Content: ".card { color: red; }", Attributes: map[string]string{"scoped": ""}},
			},
			TemplateAttributes: map[string]string{},
		}

		pikoImports := []annotator_dto.PikoImport{
			{Alias: "card", Path: "partials/card.pk"},
		}

		comp := buildParsedComponent(nil, nil, nil, "/test.pk", sfcResult, pikoImports)

		require.Len(t, comp.StyleBlocks, 1)
		assert.Contains(t, comp.StyleBlocks[0].Content, ".card")
		require.Len(t, comp.PikoImports, 1)
		assert.Equal(t, "card", comp.PikoImports[0].Alias)
	})
}

func TestApplyCollectionDirective(t *testing.T) {
	t.Parallel()

	contentImports := []*goast.ImportSpec{
		{
			Name: goast.NewIdent("content"),
			Path: &goast.BasicLit{Kind: token.STRING, Value: `"github.com/myorg/mysite/content"`},
		},
	}

	testCases := []struct {
		name                  string
		attributes            map[string]string
		imports               []*goast.ImportSpec
		wantCollection        bool
		wantName              string
		wantProvider          string
		wantParam             string
		wantContentModulePath string
		wantDiagnostic        bool
	}{
		{
			name:       "leaves the component untouched without p-collection",
			attributes: map[string]string{},
		},
		{
			name: "populates collection fields when p-collection is present",
			attributes: map[string]string{
				"p-collection": "posts",
				"p-provider":   "filesystem",
				"p-param":      "id",
			},
			wantCollection: true,
			wantName:       "posts",
			wantProvider:   "filesystem",
			wantParam:      "id",
		},
		{
			name: "resolves the collection source alias from Go imports",
			attributes: map[string]string{
				"p-collection":        "posts",
				"p-collection-source": "content",
			},
			imports:               contentImports,
			wantCollection:        true,
			wantName:              "posts",
			wantProvider:          "markdown",
			wantParam:             "slug",
			wantContentModulePath: "github.com/myorg/mysite/content",
		},
		{
			name: "reports an error when the collection source alias matches no import",
			attributes: map[string]string{
				"p-collection":        "posts",
				"p-collection-source": "nonexistent",
			},
			imports: []*goast.ImportSpec{
				{Path: &goast.BasicLit{Kind: token.STRING, Value: `"fmt"`}},
			},
			wantCollection: true,
			wantName:       "posts",
			wantProvider:   "markdown",
			wantParam:      "slug",
			wantDiagnostic: true,
		},
		{
			name: "reports an error when the component has no script",
			attributes: map[string]string{
				"p-collection":        "posts",
				"p-collection-source": "content",
			},
			wantCollection: true,
			wantName:       "posts",
			wantProvider:   "markdown",
			wantParam:      "slug",
			wantDiagnostic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sfcResult := &sfcparser.ParseResult{
				TemplateAttributes: tc.attributes,
				TemplateLocation:   sfcparser.Location{Line: 3, Column: 1},
				Styles:             []sfcparser.Style{},
			}
			var parsedScript *annotator_dto.ParsedScript
			if tc.imports != nil {
				parsedScript = &annotator_dto.ParsedScript{
					AST: &goast.File{Name: goast.NewIdent("main"), Imports: tc.imports},
				}
			}

			component := buildParsedComponent(nil, parsedScript, nil, "/test.pk", sfcResult, nil)
			diagnostic := applyCollectionDirective(component, sfcResult, parsedScript, "/test.pk")

			assert.Equal(t, tc.wantCollection, component.HasCollection)
			assert.Equal(t, tc.wantName, component.CollectionName)
			assert.Equal(t, tc.wantProvider, component.CollectionProvider)
			assert.Equal(t, tc.wantParam, component.CollectionParamName)
			assert.Equal(t, tc.wantContentModulePath, component.ContentModulePath)

			if !tc.wantDiagnostic {
				assert.Nil(t, diagnostic)
				return
			}
			require.NotNil(t, diagnostic)
			assert.Equal(t, ast_domain.Error, diagnostic.Severity)
			assert.Equal(t, annotator_dto.CodeCollectionSourceNotFound, diagnostic.Code)
			assert.Equal(t, "/test.pk", diagnostic.SourcePath)
			assert.Equal(t, 3, diagnostic.Location.Line)
			assert.Contains(t, diagnostic.Message, tc.attributes["p-collection-source"])
		})
	}
}

func TestResolveCollectionSourceAlias(t *testing.T) {
	t.Parallel()

	namedImport := func(name, path string) *goast.ImportSpec {
		return &goast.ImportSpec{
			Name: goast.NewIdent(name),
			Path: &goast.BasicLit{Kind: token.STRING, Value: `"` + path + `"`},
		}
	}
	unnamedImport := func(path string) *goast.ImportSpec {
		return &goast.ImportSpec{Path: &goast.BasicLit{Kind: token.STRING, Value: `"` + path + `"`}}
	}

	testCases := []struct {
		name      string
		alias     string
		imports   []*goast.ImportSpec
		wantPath  string
		wantFound bool
	}{
		{
			name:      "resolves a named import",
			alias:     "content",
			imports:   []*goast.ImportSpec{namedImport("content", "github.com/myorg/mysite/content")},
			wantPath:  "github.com/myorg/mysite/content",
			wantFound: true,
		},
		{
			name:      "resolves an unnamed import by its last path element",
			alias:     "articles",
			imports:   []*goast.ImportSpec{unnamedImport("github.com/myorg/myblog/articles")},
			wantPath:  "github.com/myorg/myblog/articles",
			wantFound: true,
		},
		{
			name:      "skips a major-version suffix when matching an unnamed import",
			alias:     "docs",
			imports:   []*goast.ImportSpec{unnamedImport("github.com/myorg/docs/v2")},
			wantPath:  "github.com/myorg/docs/v2",
			wantFound: true,
		},
		{
			name:    "does not match the major-version suffix itself",
			alias:   "v2",
			imports: []*goast.ImportSpec{unnamedImport("github.com/myorg/docs/v2")},
		},
		{
			name:      "resolves a single-element import path",
			alias:     "content",
			imports:   []*goast.ImportSpec{unnamedImport("content")},
			wantPath:  "content",
			wantFound: true,
		},
		{
			name:    "does not match the path of an import renamed to something else",
			alias:   "content",
			imports: []*goast.ImportSpec{namedImport("other", "github.com/myorg/content")},
		},
		{
			name:    "reports not found when no import matches",
			alias:   "content",
			imports: []*goast.ImportSpec{unnamedImport("fmt")},
		},
		{
			name:  "reports not found for nil imports",
			alias: "content",
		},
		{
			name:    "reports not found for an empty alias",
			alias:   "",
			imports: []*goast.ImportSpec{unnamedImport("github.com/myorg/content")},
		},
		{
			name:    "does not match a blank import",
			alias:   "_",
			imports: []*goast.ImportSpec{namedImport("_", "github.com/myorg/content")},
		},
		{
			name:      "skips nil import specs",
			alias:     "content",
			imports:   []*goast.ImportSpec{nil, namedImport("content", "github.com/content")},
			wantPath:  "github.com/content",
			wantFound: true,
		},
		{
			name:  "skips import specs with a nil path",
			alias: "content",
			imports: []*goast.ImportSpec{
				{Name: goast.NewIdent("content"), Path: nil},
				namedImport("content", "github.com/real/content"),
			},
			wantPath:  "github.com/real/content",
			wantFound: true,
		},
		{
			name:  "skips import specs with an invalid path literal",
			alias: "content",
			imports: []*goast.ImportSpec{
				{Name: goast.NewIdent("content"), Path: &goast.BasicLit{Kind: token.STRING, Value: `"unterminated`}},
				namedImport("content", "github.com/real/content"),
			},
			wantPath:  "github.com/real/content",
			wantFound: true,
		},
		{
			name:  "uses the first matching import",
			alias: "myalias",
			imports: []*goast.ImportSpec{
				namedImport("myalias", "github.com/org/myalias-pkg"),
				unnamedImport("github.com/org/myalias"),
			},
			wantPath:  "github.com/org/myalias-pkg",
			wantFound: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path, found := resolveCollectionSourceAlias(tc.alias, tc.imports)

			assert.Equal(t, tc.wantFound, found)
			assert.Equal(t, tc.wantPath, path)
		})
	}
}

func TestAssumedPackageName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		importPath string
		want       string
	}{
		{importPath: "fmt", want: "fmt"},
		{importPath: "net/http", want: "http"},
		{importPath: "github.com/org/repo/v2", want: "repo"},
		{importPath: "github.com/org/repo/v10", want: "repo"},
		{importPath: "example.com/v3", want: "example.com"},
		{importPath: "github.com/org/version", want: "version"},
		{importPath: "github.com/org/v", want: "v"},
		{importPath: "github.com/org/v2beta", want: "v2beta"},
	}

	for _, tc := range testCases {
		t.Run(tc.importPath, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, assumedPackageName(tc.importPath))
		})
	}
}

func TestAttachComponentDiagnostic(t *testing.T) {
	t.Parallel()

	newDiagnostic := func(message string) *ast_domain.Diagnostic {
		return ast_domain.NewDiagnosticWithCode(ast_domain.Error, message, "", annotator_dto.CodeCollectionSourceNotFound, ast_domain.Location{}, "/test.pk")
	}

	t.Run("creates a parse diagnostic error when there was no template error", func(t *testing.T) {
		t.Parallel()

		warning := ast_domain.NewDiagnostic(ast_domain.Warning, "existing warning", "", ast_domain.Location{}, "/test.pk")
		component := &annotator_dto.ParsedComponent{}
		component.SourcePath = "/test.pk"
		component.Template = &ast_domain.TemplateAST{}
		component.Template.Diagnostics = []*ast_domain.Diagnostic{warning}
		diagnostic := newDiagnostic("collection")

		err := attachComponentDiagnostic(component, nil, diagnostic, "<div></div>")

		diagErr, ok := errors.AsType[*ParseDiagnosticError](err)
		require.True(t, ok)
		assert.Equal(t, []*ast_domain.Diagnostic{warning, diagnostic}, diagErr.Diagnostics)
		assert.Equal(t, "/test.pk", diagErr.SourcePath)
		assert.Equal(t, "<div></div>", diagErr.TemplateSource)
		assert.Equal(t, []*ast_domain.Diagnostic{warning, diagnostic}, component.Template.Diagnostics)
	})

	t.Run("creates a parse diagnostic error for a component without a template", func(t *testing.T) {
		t.Parallel()

		component := &annotator_dto.ParsedComponent{}
		component.SourcePath = "/test.pk"
		diagnostic := newDiagnostic("collection")

		err := attachComponentDiagnostic(component, nil, diagnostic, "")

		diagErr, ok := errors.AsType[*ParseDiagnosticError](err)
		require.True(t, ok)
		assert.Equal(t, []*ast_domain.Diagnostic{diagnostic}, diagErr.Diagnostics)
	})

	t.Run("extends an existing parse diagnostic error without disturbing the template", func(t *testing.T) {
		t.Parallel()

		templateError := newDiagnostic("template")
		scriptWarning := ast_domain.NewDiagnostic(ast_domain.Warning, "script warning", "", ast_domain.Location{}, "/test.pk")
		shared := make([]*ast_domain.Diagnostic, 0, 4)
		shared = append(shared, templateError)
		existingErr := NewParseDiagnosticError(shared, "/test.pk", "")

		component := &annotator_dto.ParsedComponent{}
		component.Template = &ast_domain.TemplateAST{}
		component.Template.Diagnostics = append(shared, scriptWarning)
		diagnostic := newDiagnostic("collection")

		err := attachComponentDiagnostic(component, existingErr, diagnostic, "")

		require.Same(t, existingErr, err)
		assert.Equal(t, []*ast_domain.Diagnostic{templateError, diagnostic}, existingErr.Diagnostics)
		assert.Equal(t, []*ast_domain.Diagnostic{templateError, scriptWarning, diagnostic}, component.Template.Diagnostics)
	})

	t.Run("returns any other template error unchanged", func(t *testing.T) {
		t.Parallel()

		component := &annotator_dto.ParsedComponent{}
		fatal := errors.New("template parser failed")

		err := attachComponentDiagnostic(component, fatal, newDiagnostic("collection"), "")

		assert.Same(t, fatal, err)
	})
}

func TestParsePK_ReportsUnparseableTranslationTemplates(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		i18nBlock    string
		wantMessages []string
	}{
		{
			name:         "templates that parse produce no warnings",
			i18nBlock:    `{"en": {"greeting": "Hello ${name}", "items": "one|${count} items"}}`,
			wantMessages: nil,
		},
		{
			name:      "each unparseable template or plural form produces one warning",
			i18nBlock: `{"en": {"greeting": "Hello ${name"}, "fr": {"items": "un|${count articles"}}`,
			wantMessages: []string{
				"Translation template could not be parsed and renders as literal text: en:greeting: Unterminated expression: expected '}'",
				"Translation template could not be parsed and renders as literal text: fr:items[1]: Unterminated expression: expected '}'",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			source := "<template><div>Hi</div></template>\n<i18n lang=\"json\">" + tc.i18nBlock + "</i18n>"

			parsedComponent, _, err := ParsePK(context.Background(), []byte(source), "/project/src/main.pk")

			require.NoError(t, err)
			require.NotNil(t, parsedComponent.Template)
			var messages []string
			for _, diagnostic := range parsedComponent.Template.Diagnostics {
				assert.Equal(t, ast_domain.Warning, diagnostic.Severity)
				assert.Equal(t, "/project/src/main.pk", diagnostic.SourcePath)
				assert.Equal(t, 2, diagnostic.Location.Line)
				messages = append(messages, diagnostic.Message)
			}
			assert.Equal(t, tc.wantMessages, messages)
		})
	}
}

func TestNewI18nTemplateDiagnostics_WithoutI18nBlocks(t *testing.T) {
	t.Parallel()

	diagnostics := newI18nTemplateDiagnostics(nil, &sfcparser.ParseResult{}, "main.pk")

	assert.Nil(t, diagnostics)
}
