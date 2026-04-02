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

package render_domain

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	qt "github.com/valyala/quicktemplate"
	"go.opentelemetry.io/otel/trace/noop"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/pml/pml_domain"
	"piko.sh/piko/internal/pml/pml_dto"
	"piko.sh/piko/internal/premailer"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestExtractPreservedBlocks_FindsMSOComments(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:    ast_domain.NodeRawHTML,
				TextContent: "<!--[if mso]><table><tr><td>Outlook content</td></tr></table><![endif]-->",
			},
			{
				NodeType:    ast_domain.NodeElement,
				TagName:     "p",
				TextContent: "Regular content",
			},
		},
	}

	blocks := ro.extractPreservedBlocks(ast)

	require.Len(t, blocks, 1)
	assert.Contains(t, blocks[0], "<!--[if mso]>")
	assert.Contains(t, blocks[0], "Outlook content")
}

func TestExtractPreservedBlocks_MultipleMSOBlocks(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:    ast_domain.NodeRawHTML,
				TextContent: "<!--[if mso]>Block 1<![endif]-->",
			},
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
			},
			{
				NodeType:    ast_domain.NodeRawHTML,
				TextContent: "<!--[if mso 15]>Block 2<![endif]-->",
			},
		},
	}

	blocks := ro.extractPreservedBlocks(ast)

	require.Len(t, blocks, 2)
	assert.Contains(t, blocks[0], "Block 1")
	assert.Contains(t, blocks[1], "Block 2")
}

func TestExtractPreservedBlocks_NoMSOComments(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:    ast_domain.NodeElement,
				TagName:     "p",
				TextContent: "Regular paragraph",
			},
			{
				NodeType:    ast_domain.NodeRawHTML,
				TextContent: "<!-- Regular HTML comment -->",
			},
		},
	}

	blocks := ro.extractPreservedBlocks(ast)

	assert.Empty(t, blocks)
}

func TestExtractPreservedBlocks_NilAST(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	blocks := ro.extractPreservedBlocks(nil)

	assert.Nil(t, blocks)
}

func TestExtractPreservedBlocks_EmptyAST(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{},
	}

	blocks := ro.extractPreservedBlocks(ast)

	assert.Empty(t, blocks)
}

func TestExtractPreservedBlocks_NestedMSOComment(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType:    ast_domain.NodeRawHTML,
						TextContent: "<!--[if mso]>Nested MSO<![endif]-->",
					},
				},
			},
		},
	}

	blocks := ro.extractPreservedBlocks(ast)

	require.Len(t, blocks, 1)
	assert.Contains(t, blocks[0], "Nested MSO")
}

func TestExtractPremailerLeftoverCSS_FindsStyleTag(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:   ast_domain.NodeElement,
				TagName:    "head",
				Attributes: []ast_domain.HTMLAttribute{},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType:   ast_domain.NodeElement,
						TagName:    "style",
						Attributes: []ast_domain.HTMLAttribute{},
						Children: []*ast_domain.TemplateNode{
							{
								NodeType:    ast_domain.NodeText,
								TextContent: "@media screen { .mobile { display: block; } }",
							},
						},
					},
				},
			},
		},
	}

	css := ro.extractPremailerLeftoverCSS(context.Background(), ast)

	assert.Contains(t, css, "@media screen")
	assert.Contains(t, css, ".mobile")
}

func TestExtractPremailerLeftoverCSS_IgnoresStyleWithAttributes(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "head",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "style",
						Attributes: []ast_domain.HTMLAttribute{
							{Name: "type", Value: "text/css"},
						},
						Children: []*ast_domain.TemplateNode{
							{
								NodeType:    ast_domain.NodeText,
								TextContent: "body { color: red; }",
							},
						},
					},
				},
			},
		},
	}

	css := ro.extractPremailerLeftoverCSS(context.Background(), ast)

	assert.Empty(t, css)
}

func TestExtractPremailerLeftoverCSS_NoHeadTag(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "body",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType:    ast_domain.NodeText,
						TextContent: "Body content",
					},
				},
			},
		},
	}

	css := ro.extractPremailerLeftoverCSS(context.Background(), ast)

	assert.Empty(t, css)
}

func TestExtractPremailerLeftoverCSS_NilAST(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	css := ro.extractPremailerLeftoverCSS(context.Background(), nil)

	assert.Empty(t, css)
}

func TestExtractPremailerLeftoverCSS_EmptyHeadTag(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "head",
				Children: []*ast_domain.TemplateNode{},
			},
		},
	}

	css := ro.extractPremailerLeftoverCSS(context.Background(), ast)

	assert.Empty(t, css)
}

func TestCombineEmailCSS_BothInputsPresent(t *testing.T) {
	premailerCSS := "@media screen { .desktop { display: block; } }"
	pmlCSS := ".pml-button { background: blue; }"

	bodyStyles, finalCSS := combineEmailCSS(context.Background(), premailerCSS, pmlCSS)

	assert.Empty(t, bodyStyles)
	assert.Contains(t, finalCSS, "@media screen")
	assert.Contains(t, finalCSS, ".pml-button")
}

func TestCombineEmailCSS_OnlyPremailerCSS(t *testing.T) {
	premailerCSS := ".container { max-width: 600px; }"
	pmlCSS := ""

	bodyStyles, finalCSS := combineEmailCSS(context.Background(), premailerCSS, pmlCSS)

	assert.Empty(t, bodyStyles)
	assert.Contains(t, finalCSS, ".container")
	assert.Contains(t, finalCSS, "max-width")
}

func TestCombineEmailCSS_OnlyPmlCSS(t *testing.T) {
	premailerCSS := ""
	pmlCSS := ".pml-container { width: 100%; }"

	bodyStyles, finalCSS := combineEmailCSS(context.Background(), premailerCSS, pmlCSS)

	assert.Empty(t, bodyStyles)
	assert.Contains(t, finalCSS, ".pml-container")
}

func TestCombineEmailCSS_BothEmpty(t *testing.T) {
	bodyStyles, finalCSS := combineEmailCSS(context.Background(), "", "")

	assert.Empty(t, bodyStyles)
	assert.Empty(t, finalCSS)
}

func TestCombineEmailCSS_PreservesWhitespace(t *testing.T) {
	premailerCSS := "  .trimmed { color: red; }  "
	pmlCSS := "  .also-trimmed { color: blue; }  "

	_, finalCSS := combineEmailCSS(context.Background(), premailerCSS, pmlCSS)

	assert.False(t, strings.HasPrefix(finalCSS, " "))
	assert.False(t, strings.HasSuffix(finalCSS, " "))
}

func TestLogPmlErrors_NoErrors(t *testing.T) {

	logPmlErrors(context.Background(), []*pml_domain.Error{})
}

func TestLogPmlErrors_WarningLevel(t *testing.T) {
	errors := []*pml_domain.Error{
		{
			Message:  "Missing alt attribute",
			Severity: pml_domain.SeverityWarning,
			Location: ast_domain.Location{Line: 10, Column: 5},
			TagName:  "pml-img",
		},
	}

	logPmlErrors(context.Background(), errors)
}

func TestLogPmlErrors_ErrorLevel(t *testing.T) {
	errors := []*pml_domain.Error{
		{
			Message:  "Invalid attribute value",
			Severity: pml_domain.SeverityError,
			Location: ast_domain.Location{Line: 25, Column: 10},
			TagName:  "pml-button",
		},
	}

	logPmlErrors(context.Background(), errors)
}

func TestLogPmlErrors_MultipleErrors(t *testing.T) {
	errors := []*pml_domain.Error{
		{
			Message:  "Warning 1",
			Severity: pml_domain.SeverityWarning,
			Location: ast_domain.Location{Line: 1, Column: 1},
			TagName:  "pml-container",
		},
		{
			Message:  "Error 1",
			Severity: pml_domain.SeverityError,
			Location: ast_domain.Location{Line: 2, Column: 2},
			TagName:  "pml-row",
		},
		{
			Message:  "Warning 2",
			Severity: pml_domain.SeverityWarning,
			Location: ast_domain.Location{Line: 3, Column: 3},
			TagName:  "pml-col",
		},
	}

	logPmlErrors(context.Background(), errors)
}

func TestLogEmailDiagnostics_NoWarningsOrErrors(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	logEmailDiagnostics(context.Background(), rctx)
}

func TestLogEmailDiagnostics_WithWarnings(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	rctx.diagnostics.AddWarning("test", "Test warning", nil)

	logEmailDiagnostics(context.Background(), rctx)
}

func TestLogEmailDiagnostics_WithErrors(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	rctx.diagnostics.AddError("test", nil, "Test error", nil)

	logEmailDiagnostics(context.Background(), rctx)
}

func TestLogEmailDiagnostics_WithBothWarningsAndErrors(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	rctx.diagnostics.AddWarning("test", "Test warning", nil)
	rctx.diagnostics.AddError("test", nil, "Test error", nil)

	logEmailDiagnostics(context.Background(), rctx)
}

func TestPerformPremailerPass(t *testing.T) {
	testCases := []struct {
		inputAST         *ast_domain.TemplateAST
		premailerOptions *premailer.Options
		name             string
		styling          string
		wantCSSContains  string
		wantSameAST      bool
		wantCSSEmpty     bool
	}{
		{
			name: "empty AST with no styling returns transformed AST",
			inputAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{},
			},
			styling:      "",
			wantSameAST:  false,
			wantCSSEmpty: true,
		},
		{
			name: "styling inlines CSS into AST elements",
			inputAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						Attributes: []ast_domain.HTMLAttribute{
							{Name: "class", Value: "container"},
						},
					},
				},
			},
			styling:      ".container { color: red; }",
			wantSameAST:  false,
			wantCSSEmpty: true,
		},
		{
			name: "media queries remain as leftover CSS",
			inputAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						Attributes: []ast_domain.HTMLAttribute{
							{Name: "class", Value: "mobile"},
						},
					},
				},
			},
			styling:         ".mobile { color: blue; } @media screen and (max-width: 480px) { .mobile { font-size: 14px; } }",
			wantSameAST:     false,
			wantCSSContains: "@media",
		},
		{
			name: "nil premailer options uses defaults",
			inputAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "p",
					},
				},
			},
			styling:          "p { margin: 0; }",
			premailerOptions: nil,
			wantSameAST:      false,
			wantCSSEmpty:     true,
		},
		{
			name: "premailer options with RemoveClasses are respected",
			inputAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "p",
						Attributes: []ast_domain.HTMLAttribute{
							{Name: "class", Value: "test"},
						},
					},
				},
			},
			styling: ".test { color: green; }",
			premailerOptions: &premailer.Options{
				RemoveClasses: true,
			},
			wantSameAST:  false,
			wantCSSEmpty: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ro := NewTestOrchestratorBuilder().Build()
			span := noop.Span{}

			resultAST, leftoverCSS := ro.performPremailerPass(context.Background(), tc.inputAST, tc.styling, tc.premailerOptions, span)

			if tc.wantSameAST {
				assert.Equal(t, tc.inputAST, resultAST, "expected the same AST to be returned")
			} else {
				assert.NotNil(t, resultAST, "expected a non-nil result AST")
			}

			if tc.wantCSSEmpty {
				assert.Empty(t, leftoverCSS, "expected no leftover CSS")
			}

			if tc.wantCSSContains != "" {
				assert.Contains(t, leftoverCSS, tc.wantCSSContains)
			}
		})
	}
}

func TestPerformPremailerPass_InlinesStyleAttribute(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	span := noop.Span{}

	inputAST := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "class", Value: "highlight"},
				},
			},
		},
	}

	resultAST, _ := ro.performPremailerPass(context.Background(), inputAST, ".highlight { background: yellow; }", nil, span)

	require.NotNil(t, resultAST)
	require.NotEmpty(t, resultAST.RootNodes)

	var hasStyle bool
	for _, attr := range resultAST.RootNodes[0].Attributes {
		if attr.Name == "style" {
			hasStyle = true
			assert.Contains(t, attr.Value, "background")
		}
	}
	assert.True(t, hasStyle, "expected a style attribute to be inlined onto the element")
}

func TestPerformPmlTransformation(t *testing.T) {
	testCases := []struct {
		inputAST                  *ast_domain.TemplateAST
		transformForEmailFunction func(*ast_domain.TemplateAST, *pml_dto.Config) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*pml_domain.Error)
		name                      string
		wantCSS                   string
		wantAssetRequestsLen      int
		wantNilAST                bool
	}{
		{
			name: "returns transformed AST from engine",
			inputAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "pml-button",
					},
				},
			},
			transformForEmailFunction: func(_ *ast_domain.TemplateAST, _ *pml_dto.Config) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*pml_domain.Error) {
				return &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "a",
						},
					},
				}, ".pml-button { display: inline-block; }", nil, nil
			},
			wantNilAST:           false,
			wantCSS:              ".pml-button { display: inline-block; }",
			wantAssetRequestsLen: 0,
		},
		{
			name: "stores asset requests on orchestrator",
			inputAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "pml-img",
					},
				},
			},
			transformForEmailFunction: func(_ *ast_domain.TemplateAST, _ *pml_dto.Config) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*pml_domain.Error) {
				return &ast_domain.TemplateAST{
						RootNodes: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "img",
							},
						},
					}, "", []*email_dto.EmailAssetRequest{
						{SourcePath: "assets/logo.png", Profile: "email-default"},
						{SourcePath: "assets/banner.jpg", Profile: "email-outlook"},
					}, nil
			},
			wantNilAST:           false,
			wantCSS:              "",
			wantAssetRequestsLen: 2,
		},
		{
			name: "handles PML errors without failing",
			inputAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "pml-section",
					},
				},
			},
			transformForEmailFunction: func(inputAST *ast_domain.TemplateAST, _ *pml_dto.Config) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*pml_domain.Error) {
				return inputAST, "", nil, []*pml_domain.Error{
					{
						Message:  "Unknown tag",
						Severity: pml_domain.SeverityWarning,
						Location: ast_domain.Location{Line: 1, Column: 1},
						TagName:  "pml-section",
					},
				}
			},
			wantNilAST:           false,
			wantCSS:              "",
			wantAssetRequestsLen: 0,
		},
		{
			name: "nil AST from engine returns nil",
			inputAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{},
			},
			transformForEmailFunction: func(_ *ast_domain.TemplateAST, _ *pml_dto.Config) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*pml_domain.Error) {
				return nil, "", nil, nil
			},
			wantNilAST:           true,
			wantCSS:              "",
			wantAssetRequestsLen: 0,
		},
		{
			name: "default mock engine returns input AST unchanged",
			inputAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType:    ast_domain.NodeText,
						TextContent: "passthrough",
					},
				},
			},
			transformForEmailFunction: nil,
			wantNilAST:                false,
			wantCSS:                   "",
			wantAssetRequestsLen:      0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockEngine := &pml_domain.MockTransformer{}
			if tc.transformForEmailFunction != nil {
				mockEngine.TransformForEmailFunc = tc.transformForEmailFunction
			}
			ro := NewTestOrchestratorBuilder().
				WithPmlEngine(mockEngine).
				Build()

			resultAST, css := ro.performPmlTransformation(context.Background(), tc.inputAST, false)

			if tc.wantNilAST {
				assert.Nil(t, resultAST)
			} else {
				assert.NotNil(t, resultAST)
			}

			assert.Equal(t, tc.wantCSS, css)
			assert.Len(t, ro.GetLastEmailAssetRequests(), tc.wantAssetRequestsLen)
		})
	}
}

func TestProcessEmailPipeline(t *testing.T) {
	testCases := []struct {
		inputAST                  *ast_domain.TemplateAST
		premailerOptions          *premailer.Options
		transformForEmailFunction func(*ast_domain.TemplateAST, *pml_dto.Config) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*pml_domain.Error)
		name                      string
		styling                   string
		wantFinalCSSContains      string
		wantNilHTMLAST            bool
	}{
		{
			name: "basic pipeline with passthrough engine",
			inputAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType:    ast_domain.NodeText,
						TextContent: "Hello",
					},
				},
			},
			styling:        "",
			wantNilHTMLAST: false,
		},
		{
			name: "pipeline combines premailer leftover and PML CSS",
			inputAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						Attributes: []ast_domain.HTMLAttribute{
							{Name: "class", Value: "wrapper"},
						},
					},
				},
			},
			styling: ".wrapper { padding: 10px; } @media (max-width: 600px) { .wrapper { padding: 5px; } }",
			transformForEmailFunction: func(inputAST *ast_domain.TemplateAST, _ *pml_dto.Config) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*pml_domain.Error) {
				return inputAST, ".pml-generated { font-size: 16px; }", nil, nil
			},
			wantNilHTMLAST:       false,
			wantFinalCSSContains: ".pml-generated",
		},
		{
			name: "nil AST from PML engine falls back to pre-inlined AST",
			inputAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType:    ast_domain.NodeText,
						TextContent: "Fallback content",
					},
				},
			},
			transformForEmailFunction: func(_ *ast_domain.TemplateAST, _ *pml_dto.Config) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*pml_domain.Error) {
				return nil, "", nil, nil
			},
			wantNilHTMLAST: false,
		},
		{
			name: "empty styling and no PML CSS produces empty final CSS",
			inputAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType:    ast_domain.NodeText,
						TextContent: "Plain",
					},
				},
			},
			styling:        "",
			wantNilHTMLAST: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engineBuilder := NewTestOrchestratorBuilder()
			if tc.transformForEmailFunction != nil {
				engineBuilder = engineBuilder.WithPmlEngine(&pml_domain.MockTransformer{
					TransformForEmailFunc: tc.transformForEmailFunction,
				})
			}
			ro := engineBuilder.Build()
			span := noop.Span{}

			result := ro.processEmailPipeline(context.Background(), tc.inputAST, tc.styling, tc.premailerOptions, false, span)

			if tc.wantNilHTMLAST {
				assert.Nil(t, result.HTMLAST)
			} else {
				assert.NotNil(t, result.HTMLAST)
			}

			if tc.wantFinalCSSContains != "" {
				assert.Contains(t, result.FinalCSS, tc.wantFinalCSSContains)
			}
		})
	}
}

func TestEmbedEmailSVGSprite(t *testing.T) {
	testCases := []struct {
		registry        *MockRegistryPort
		name            string
		wantContains    string
		wantNotContain  string
		svgSymbols      []svgSymbolEntry
		wantOutputEmpty bool
	}{
		{
			name: "with SVG symbols embeds sprite sheet",
			registry: newTestRegistryBuilder().
				withSVG("icon-star", `<path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>`, ast_domain.HTMLAttribute{Name: "viewBox", Value: "0 0 24 24"}).
				build(),
			svgSymbols: func() []svgSymbolEntry {
				svg := &ParsedSvgData{
					InnerHTML:  `<path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>`,
					Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 24 24"}},
				}
				svg.CachedSymbol = ComputeSymbolString("icon-star", svg)
				return []svgSymbolEntry{{id: "icon-star", data: svg}}
			}(),
			wantOutputEmpty: false,
			wantContains:    "icon-star",
		},
		{
			name: "SVG error still writes sprite container but no symbols",
			registry: newTestRegistryBuilder().
				withSVGError(assert.AnError).
				build(),
			svgSymbols: []svgSymbolEntry{
				{id: "missing-icon", data: nil},
			},
			wantOutputEmpty: false,
			wantNotContain:  "missing-icon",
		},
		{
			name: "multiple SVG symbols embeds all in sprite",
			registry: newTestRegistryBuilder().
				withSVG("icon-home", `<path d="M10 20v-6h4v6"/>`, ast_domain.HTMLAttribute{Name: "viewBox", Value: "0 0 24 24"}).
				withSVG("icon-mail", `<rect x="2" y="4" width="20" height="16"/>`, ast_domain.HTMLAttribute{Name: "viewBox", Value: "0 0 24 24"}).
				build(),
			svgSymbols: func() []svgSymbolEntry {
				svgHome := &ParsedSvgData{
					InnerHTML:  `<path d="M10 20v-6h4v6"/>`,
					Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 24 24"}},
				}
				svgHome.CachedSymbol = ComputeSymbolString("icon-home", svgHome)
				svgMail := &ParsedSvgData{
					InnerHTML:  `<rect x="2" y="4" width="20" height="16"/>`,
					Attributes: []ast_domain.HTMLAttribute{{Name: "viewBox", Value: "0 0 24 24"}},
				}
				svgMail.CachedSymbol = ComputeSymbolString("icon-mail", svgMail)
				return []svgSymbolEntry{
					{id: "icon-home", data: svgHome},
					{id: "icon-mail", data: svgMail},
				}
			}(),
			wantOutputEmpty: false,
			wantContains:    "icon-home",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ro := NewTestOrchestratorBuilder().
				WithRegistry(tc.registry).
				Build()

			rctx := NewTestRenderContextBuilder().
				WithRegistry(tc.registry).
				Build()

			if tc.svgSymbols != nil {
				rctx.requiredSvgSymbols = tc.svgSymbols
			}

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)
			span := noop.Span{}

			ro.embedEmailSVGSprite(context.Background(), qw, rctx, span)
			output := buffer.String()

			if tc.wantOutputEmpty {
				assert.Empty(t, output)
			} else {
				assert.NotEmpty(t, output)
			}

			if tc.wantContains != "" {
				assert.Contains(t, output, tc.wantContains)
			}

			if tc.wantNotContain != "" {
				assert.NotContains(t, output, tc.wantNotContain)
			}
		})
	}
}

func TestRenderEmailContent(t *testing.T) {
	testCases := []struct {
		metadata    *templater_dto.InternalMetadata
		name        string
		wantContain string
		params      emailContentParams
		wantErr     bool
	}{
		{
			name: "renders email with simple text node",
			params: emailContentParams{
				HTMLAST: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "Hello Email",
						},
					},
				},
				FinalCSS:         ".test { color: red; }",
				BodyInlineStyles: "background-color: white;",
			},
			metadata: &templater_dto.InternalMetadata{
				Metadata: templater_dto.Metadata{Title: "Test Email", Language: "en"},
			},
			wantErr:     false,
			wantContain: "Hello Email",
		},
		{
			name: "renders email with nil AST and no error",
			params: emailContentParams{
				HTMLAST:  nil,
				FinalCSS: "",
			},
			metadata: &templater_dto.InternalMetadata{
				Metadata: templater_dto.Metadata{Title: "Empty Email", Language: "en"},
			},
			wantErr: false,
		},
		{
			name: "renders email with element node containing children",
			params: emailContentParams{
				HTMLAST: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "table",
							Children: []*ast_domain.TemplateNode{
								{
									NodeType: ast_domain.NodeElement,
									TagName:  "tr",
									Children: []*ast_domain.TemplateNode{
										{
											NodeType: ast_domain.NodeElement,
											TagName:  "td",
											Children: []*ast_domain.TemplateNode{
												{
													NodeType:    ast_domain.NodeText,
													TextContent: "Cell Content",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			metadata: &templater_dto.InternalMetadata{
				Metadata: templater_dto.Metadata{Title: "Table Email", Language: "en"},
			},
			wantErr:     false,
			wantContain: "Cell Content",
		},
		{
			name: "includes preserved head blocks in output",
			params: emailContentParams{
				HTMLAST: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "Body",
						},
					},
				},
				PreservedBlocks: []string{
					"<!--[if mso]><style>body{margin:0}</style><![endif]-->",
				},
			},
			metadata: &templater_dto.InternalMetadata{
				Metadata: templater_dto.Metadata{Title: "MSO Email", Language: "en"},
			},
			wantErr:     false,
			wantContain: "<!--[if mso]>",
		},
		{
			name: "includes styling in output",
			params: emailContentParams{
				HTMLAST: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "Styled",
						},
					},
				},
				FinalCSS: ".custom-class { font-weight: bold; }",
			},
			metadata: &templater_dto.InternalMetadata{
				Metadata: templater_dto.Metadata{Title: "Styled Email", Language: "en"},
			},
			wantErr:     false,
			wantContain: ".custom-class",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ro := NewTestOrchestratorBuilder().Build()

			rctx := NewTestRenderContextBuilder().Build()
			rctx.isEmailMode = true
			rctx.skipPrerenderedHTML = true

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)
			span := noop.Span{}

			err := ro.renderEmailContent(context.Background(), qw, rctx, tc.metadata, tc.params, span)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			output := buffer.String()
			if tc.wantContain != "" {
				assert.Contains(t, output, tc.wantContain)
			}
		})
	}
}

func TestRenderEmail(t *testing.T) {
	testCases := []struct {
		transformForEmailFunction func(*ast_domain.TemplateAST, *pml_dto.Config) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*pml_domain.Error)
		opts                      RenderEmailOptions
		name                      string
		wantContains              string
		wantErr                   bool
	}{
		{
			name: "empty template renders without error",
			opts: RenderEmailOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{},
				},
				Metadata: &templater_dto.InternalMetadata{
					Metadata: templater_dto.Metadata{Title: "Empty Template", Language: "en"},
				},
				PageID: "test-empty",
			},
			wantErr: false,
		},
		{
			name: "basic email with text node",
			opts: RenderEmailOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "Welcome to Piko",
						},
					},
				},
				Metadata: &templater_dto.InternalMetadata{
					Metadata: templater_dto.Metadata{Title: "Welcome", Language: "en"},
				},
				PageID:  "test-basic",
				Styling: "",
			},
			wantErr:      false,
			wantContains: "Welcome to Piko",
		},
		{
			name: "email with styling applies premailer pass",
			opts: RenderEmailOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "p",
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "class", Value: "intro"},
							},
							Children: []*ast_domain.TemplateNode{
								{
									NodeType:    ast_domain.NodeText,
									TextContent: "Styled paragraph",
								},
							},
						},
					},
				},
				Metadata: &templater_dto.InternalMetadata{
					Metadata: templater_dto.Metadata{Title: "Styled", Language: "en"},
				},
				PageID:  "test-styled",
				Styling: ".intro { color: navy; }",
			},
			wantErr:      false,
			wantContains: "Styled paragraph",
		},
		{
			name: "email with PML engine generating CSS",
			opts: RenderEmailOptions{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "PML content",
						},
					},
				},
				Metadata: &templater_dto.InternalMetadata{
					Metadata: templater_dto.Metadata{Title: "PML Email", Language: "en"},
				},
				PageID: "test-pml",
			},
			transformForEmailFunction: func(inputAST *ast_domain.TemplateAST, _ *pml_dto.Config) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*pml_domain.Error) {
				return inputAST, ".pml-generated { margin: 0; }", nil, nil
			},
			wantErr:      false,
			wantContains: ".pml-generated",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := NewTestOrchestratorBuilder()
			if tc.transformForEmailFunction != nil {
				builder = builder.WithPmlEngine(&pml_domain.MockTransformer{
					TransformForEmailFunc: tc.transformForEmailFunction,
				})
			}
			ro := builder.Build()

			var buffer bytes.Buffer
			request := testHTTPRequest()
			ctx := context.Background()

			err := ro.RenderEmail(ctx, &buffer, request, tc.opts)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			output := buffer.String()
			if tc.wantContains != "" {
				assert.Contains(t, output, tc.wantContains)
			}
		})
	}
}

func TestRenderEmail_IncludesBaseStyles(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	var buffer bytes.Buffer
	request := testHTTPRequest()
	ctx := context.Background()

	err := ro.RenderEmail(ctx, &buffer, request, RenderEmailOptions{
		Template: &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:    ast_domain.NodeText,
					TextContent: "Test",
				},
			},
		},
		Metadata: &templater_dto.InternalMetadata{
			Metadata: templater_dto.Metadata{Title: "Base Styles Test", Language: "en"},
		},
		PageID: "test-base-styles",
	})

	require.NoError(t, err)

	output := buffer.String()

	assert.Contains(t, output, "#outlook a{padding:0}")
	assert.Contains(t, output, "border-collapse:collapse")
}

func TestRenderEmail_TransformedASTAppearsInOutput(t *testing.T) {

	transformedAST := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:    ast_domain.NodeText,
				TextContent: "TRANSFORMED",
			},
		},
	}

	ro := NewTestOrchestratorBuilder().
		WithPmlEngine(&pml_domain.MockTransformer{
			TransformForEmailFunc: func(_ *ast_domain.TemplateAST, _ *pml_dto.Config) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*pml_domain.Error) {
				return transformedAST, "", nil, nil
			},
		}).
		Build()

	var buffer bytes.Buffer
	request := testHTTPRequest()
	ctx := context.Background()

	err := ro.RenderEmail(ctx, &buffer, request, RenderEmailOptions{
		Template: &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:    ast_domain.NodeText,
					TextContent: "ORIGINAL",
				},
			},
		},
		Metadata: &templater_dto.InternalMetadata{
			Metadata: templater_dto.Metadata{Title: "Transform Test", Language: "en"},
		},
		PageID: "test-transform",
	})

	require.NoError(t, err)
	output := buffer.String()
	assert.Contains(t, output, "TRANSFORMED")
}

func TestRenderEmail_StoresAssetRequestsFromPipeline(t *testing.T) {
	expectedAssets := []*email_dto.EmailAssetRequest{
		{SourcePath: "assets/logo.png", Profile: "email-default"},
	}

	ro := NewTestOrchestratorBuilder().
		WithPmlEngine(&pml_domain.MockTransformer{
			TransformForEmailFunc: func(inputAST *ast_domain.TemplateAST, _ *pml_dto.Config) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*pml_domain.Error) {
				return inputAST, "", expectedAssets, nil
			},
		}).
		Build()

	var buffer bytes.Buffer
	request := testHTTPRequest()
	ctx := context.Background()

	err := ro.RenderEmail(ctx, &buffer, request, RenderEmailOptions{
		Template: &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:    ast_domain.NodeText,
					TextContent: "Image email",
				},
			},
		},
		Metadata: &templater_dto.InternalMetadata{
			Metadata: templater_dto.Metadata{Title: "Asset Test", Language: "en"},
		},
		PageID: "test-assets",
	})

	require.NoError(t, err)
	requests := ro.GetLastEmailAssetRequests()
	require.Len(t, requests, 1)
	assert.Equal(t, "assets/logo.png", requests[0].SourcePath)
	assert.Equal(t, "email-default", requests[0].Profile)
}
