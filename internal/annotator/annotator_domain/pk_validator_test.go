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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestPKValidator_OrphanedPartials(t *testing.T) {
	testCases := []struct {
		name             string
		clientScript     string
		importedPartials []string
		wantOrphaned     []string
	}{
		{
			name:             "no imports no orphans",
			clientScript:     `export function handleClick() {}`,
			importedPartials: []string{},
			wantOrphaned:     nil,
		},
		{
			name:             "used partial not orphaned",
			clientScript:     `export function handleClick() { reloadPartial('card'); }`,
			importedPartials: []string{"card"},
			wantOrphaned:     nil,
		},
		{
			name:             "unused partial is orphaned",
			clientScript:     `export function handleClick() { console.log('hello'); }`,
			importedPartials: []string{"card"},
			wantOrphaned:     []string{"card"},
		},
		{
			name:             "mixed used and unused",
			clientScript:     `export function handleClick() { reloadPartial('card'); }`,
			importedPartials: []string{"card", "unused"},
			wantOrphaned:     []string{"unused"},
		},
		{
			name: "reloadGroup usage",
			clientScript: `export function handleClick() {
				reloadGroup(['card', 'header']);
			}`,
			importedPartials: []string{"card", "header", "footer"},
			wantOrphaned:     []string{"footer"},
		},
		{
			name:             "double quoted partial",
			clientScript:     `export function handleClick() { reloadPartial("card"); }`,
			importedPartials: []string{"card"},
			wantOrphaned:     nil,
		},
		{
			name: "multiple reload calls",
			clientScript: `
				export function a() { reloadPartial('card'); }
				export function b() { reloadPartial('header'); }
			`,
			importedPartials: []string{"card", "header"},
			wantOrphaned:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := NewPKValidator(tc.clientScript, "/test/component.pk")
			validator.RegisterImportedPartials(tc.importedPartials)

			ctx := &AnalysisContext{
				Diagnostics: new([]*ast_domain.Diagnostic),
			}

			validator.ReportOrphanedPartials(ctx, ast_domain.Location{Line: 1, Column: 1, Offset: 0})

			gotOrphaned := make([]string, 0, len(*ctx.Diagnostics))
			for _, diagnostic := range *ctx.Diagnostics {
				gotOrphaned = append(gotOrphaned, diagnostic.Expression)
			}

			if len(gotOrphaned) != len(tc.wantOrphaned) {
				t.Errorf("got %d orphaned partials, want %d; got=%v, want=%v",
					len(gotOrphaned), len(tc.wantOrphaned), gotOrphaned, tc.wantOrphaned)
				return
			}

			orphanedSet := make(map[string]bool)
			for _, o := range gotOrphaned {
				orphanedSet[o] = true
			}
			for _, want := range tc.wantOrphaned {
				if !orphanedSet[want] {
					t.Errorf("expected partial '%s' to be reported as orphaned", want)
				}
			}
		})
	}
}

func TestExtractHandlerName(t *testing.T) {
	testCases := []struct {
		name     string
		rawExpr  string
		wantName string
	}{
		{
			name:     "simple call",
			rawExpr:  "handleClick()",
			wantName: "handleClick",
		},
		{
			name:     "call with arguments",
			rawExpr:  "handleSubmit(event, data)",
			wantName: "handleSubmit",
		},
		{
			name:     "bare identifier",
			rawExpr:  "handleClick",
			wantName: "handleClick",
		},
		{
			name:     "with whitespace",
			rawExpr:  "  handleClick()  ",
			wantName: "handleClick",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			directive := &ast_domain.Directive{
				RawExpression: tc.rawExpr,
			}
			got := extractHandlerName(directive)
			if got != tc.wantName {
				t.Errorf("extractHandlerName(%q) = %q, want %q", tc.rawExpr, got, tc.wantName)
			}
		})
	}
}

func TestIsValidPartialAlias(t *testing.T) {
	testCases := []struct {
		alias string
		valid bool
	}{
		{alias: "card", valid: true},
		{alias: "Card", valid: true},
		{alias: "_private", valid: true},
		{alias: "card123", valid: true},
		{alias: "card_item", valid: true},
		{alias: "", valid: false},
		{alias: "123card", valid: false},
		{alias: "card-item", valid: false},
		{alias: "card.item", valid: false},
		{alias: "card item", valid: false},
	}

	for _, tc := range testCases {
		t.Run(tc.alias, func(t *testing.T) {
			got := isValidPartialAlias(tc.alias)
			if got != tc.valid {
				t.Errorf("isValidPartialAlias(%q) = %v, want %v", tc.alias, got, tc.valid)
			}
		})
	}
}

func TestIsLikelyGoFunction(t *testing.T) {
	testCases := []struct {
		name string
		isGo bool
	}{
		{name: "HandleClick", isGo: true},
		{name: "ProcessForm", isGo: true},
		{name: "handleClick", isGo: false},
		{name: "processForm", isGo: false},
		{name: "", isGo: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isLikelyGoFunction(tc.name)
			if got != tc.isGo {
				t.Errorf("isLikelyGoFunction(%q) = %v, want %v", tc.name, got, tc.isGo)
			}
		})
	}
}

func TestIsCommonUtilityName(t *testing.T) {
	testCases := []struct {
		name   string
		isUtil bool
	}{
		{name: "init", isUtil: true},
		{name: "initComponent", isUtil: true},
		{name: "setup", isUtil: true},
		{name: "setupForm", isUtil: true},
		{name: "handleClick", isUtil: false},
		{name: "onClick", isUtil: false},
		{name: "getData", isUtil: true},
		{name: "setName", isUtil: true},
		{name: "formatDate", isUtil: true},
		{name: "parseJSON", isUtil: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isCommonUtilityName(tc.name)
			if got != tc.isUtil {
				t.Errorf("isCommonUtilityName(%q) = %v, want %v", tc.name, got, tc.isUtil)
			}
		})
	}
}

func TestPKValidator_HasClientScript(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		setup    func() *PKValidator
		name     string
		expected bool
	}{
		{
			name:     "nil validator returns false",
			setup:    func() *PKValidator { return nil },
			expected: false,
		},
		{
			name: "nil clientExports returns false",
			setup: func() *PKValidator {
				return &PKValidator{clientExports: nil}
			},
			expected: false,
		},
		{
			name: "empty exports returns false",
			setup: func() *PKValidator {
				return &PKValidator{
					clientExports: &ClientScriptExports{
						ExportedFunctions: map[string]ExportedFunction{},
					},
				}
			},
			expected: false,
		},
		{
			name: "with exports returns true",
			setup: func() *PKValidator {
				return &PKValidator{
					clientExports: &ClientScriptExports{
						ExportedFunctions: map[string]ExportedFunction{
							"handleClick": {Name: "handleClick"},
						},
					},
				}
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := tc.setup()
			assert.Equal(t, tc.expected, v.HasClientScript())
		})
	}
}

func TestPKValidator_ValidateEventHandler(t *testing.T) {
	t.Parallel()

	t.Run("nil validator does not panic", func(t *testing.T) {
		t.Parallel()
		var v *PKValidator
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		v.ValidateEventHandler(&ast_domain.Directive{}, ctx)
		assert.Empty(t, diagnostics)
	})

	t.Run("nil directive does not panic", func(t *testing.T) {
		t.Parallel()
		v := &PKValidator{
			usedHandlers: make(map[string]bool),
		}
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		v.ValidateEventHandler(nil, ctx)
		assert.Empty(t, diagnostics)
	})

	t.Run("valid handler marks as used", func(t *testing.T) {
		t.Parallel()
		v := &PKValidator{
			usedHandlers: make(map[string]bool),
			clientExports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{
					"handleClick": {Name: "handleClick"},
				},
			},
		}
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		directive := &ast_domain.Directive{
			Expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "handleClick"},
			},
		}
		v.ValidateEventHandler(directive, ctx)
		assert.True(t, v.usedHandlers["handleClick"])
		assert.Empty(t, diagnostics, "valid handler should not produce diagnostics")
	})

	t.Run("unknown handler produces diagnostic", func(t *testing.T) {
		t.Parallel()
		v := &PKValidator{
			usedHandlers: make(map[string]bool),
			clientExports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{
					"handleClick": {Name: "handleClick"},
				},
			},
		}
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		directive := &ast_domain.Directive{
			Expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "unknownHandler"},
			},
		}
		v.ValidateEventHandler(directive, ctx)
		assert.True(t, v.usedHandlers["unknownHandler"])
		require.Len(t, diagnostics, 1, "should produce one diagnostic for unknown handler")
		assert.Contains(t, diagnostics[0].Message, "not found in client script exports")
	})

	t.Run("Go-style function name skips diagnostic", func(t *testing.T) {
		t.Parallel()
		v := &PKValidator{
			usedHandlers: make(map[string]bool),
			clientExports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{
					"handleClick": {Name: "handleClick"},
				},
			},
		}
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		directive := &ast_domain.Directive{
			Expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "HandleSubmit"},
			},
		}
		v.ValidateEventHandler(directive, ctx)
		assert.Empty(t, diagnostics, "Go-style function should not produce diagnostic")
	})

	t.Run("nil clientExports marks handler but produces no diagnostic", func(t *testing.T) {
		t.Parallel()
		v := &PKValidator{
			usedHandlers:  make(map[string]bool),
			clientExports: nil,
		}
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		directive := &ast_domain.Directive{
			Expression: &ast_domain.Identifier{Name: "myHandler"},
		}
		v.ValidateEventHandler(directive, ctx)
		assert.True(t, v.usedHandlers["myHandler"])
		assert.Empty(t, diagnostics)
	})
}

func TestFormatAvailableExports(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		exports  *ClientScriptExports
		expected string
	}{
		{
			name:     "nil exports returns none",
			exports:  nil,
			expected: "(none)",
		},
		{
			name: "empty exports returns none",
			exports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{},
			},
			expected: "(none)",
		},
		{
			name: "single export",
			exports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{
					"handleClick": {Name: "handleClick"},
				},
			},
			expected: "handleClick",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := formatAvailableExports(tc.exports)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestPKValidator_MarkPartialRendered(t *testing.T) {
	t.Parallel()

	t.Run("nil validator does not panic", func(t *testing.T) {
		t.Parallel()
		var v *PKValidator
		v.MarkPartialRendered("card")
	})

	t.Run("empty alias is ignored", func(t *testing.T) {
		t.Parallel()
		v := &PKValidator{
			renderedPartials: make(map[string]bool),
		}
		v.MarkPartialRendered("")
		assert.Empty(t, v.renderedPartials)
	})

	t.Run("valid alias is recorded", func(t *testing.T) {
		t.Parallel()
		v := &PKValidator{
			renderedPartials: make(map[string]bool),
		}
		v.MarkPartialRendered("card")
		assert.True(t, v.renderedPartials["card"])
	})
}

func TestPKValidator_ReportUnusedExports_IsNoop(t *testing.T) {
	t.Parallel()
	v := &PKValidator{
		usedHandlers: make(map[string]bool),
		clientExports: &ClientScriptExports{
			ExportedFunctions: map[string]ExportedFunction{
				"unusedFunc": {Name: "unusedFunc"},
			},
		},
	}
	diagnostics := make([]*ast_domain.Diagnostic, 0)
	ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
	v.ReportUnusedExports(ctx, ast_domain.Location{})
	assert.Empty(t, diagnostics, "ReportUnusedExports should be a no-op")
}

func TestExtractHandlerName_WithExpression(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		directive *ast_domain.Directive
		expected  string
	}{
		{
			name:      "nil directive returns empty",
			directive: nil,
			expected:  "",
		},
		{
			name: "CallExpr with Identifier callee",
			directive: &ast_domain.Directive{
				Expression: &ast_domain.CallExpression{
					Callee: &ast_domain.Identifier{Name: "handleClick"},
				},
			},
			expected: "handleClick",
		},
		{
			name: "bare Identifier expression",
			directive: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "doSomething"},
			},
			expected: "doSomething",
		},
		{
			name: "CallExpr with non-Identifier callee falls back to raw",
			directive: &ast_domain.Directive{
				Expression: &ast_domain.CallExpression{
					Callee: &ast_domain.MemberExpression{
						Base:     &ast_domain.Identifier{Name: "obj"},
						Property: &ast_domain.Identifier{Name: "method"},
					},
				},
				RawExpression: "obj.method()",
			},
			expected: "obj.method",
		},
		{
			name: "empty raw expression returns empty",
			directive: &ast_domain.Directive{
				RawExpression: "   ",
			},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := extractHandlerName(tc.directive)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestPKValidator_RegisterImportedPartials(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver does not panic", func(t *testing.T) {
		t.Parallel()

		var v *PKValidator

		assert.NotPanics(t, func() {
			v.RegisterImportedPartials([]string{"a", "b"})
		})
	})

	t.Run("registers all aliases", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     make(map[string]bool),
			renderedPartials: make(map[string]bool),
			importedPartials: make(map[string]bool),
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}

		v.RegisterImportedPartials([]string{"Card", "Header", "Footer"})

		assert.True(t, v.importedPartials["Card"])
		assert.True(t, v.importedPartials["Header"])
		assert.True(t, v.importedPartials["Footer"])
		assert.Len(t, v.importedPartials, 3)
	})

	t.Run("empty slice is no-op", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     make(map[string]bool),
			renderedPartials: make(map[string]bool),
			importedPartials: make(map[string]bool),
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}

		v.RegisterImportedPartials([]string{})

		assert.Empty(t, v.importedPartials)
	})
}

func TestPKValidator_FindClosingBracket(t *testing.T) {
	t.Parallel()

	v := &PKValidator{
		clientExports:    nil,
		usedHandlers:     make(map[string]bool),
		usedPartials:     make(map[string]bool),
		renderedPartials: make(map[string]bool),
		importedPartials: make(map[string]bool),
		sfcSourcePath:    "/test.pk",
		clientScript:     "",
	}

	testCases := []struct {
		name       string
		script     string
		startIndex int
		expected   int
	}{
		{
			name:       "simple bracket pair",
			script:     "['a','b']",
			startIndex: 1,
			expected:   9,
		},
		{
			name:       "nested brackets",
			script:     "[[1,2],3]",
			startIndex: 1,
			expected:   9,
		},
		{
			name:       "no closing bracket returns startIndex",
			script:     "['a','b'",
			startIndex: 1,
			expected:   1,
		},
		{
			name:       "empty content",
			script:     "[]",
			startIndex: 1,
			expected:   2,
		},
		{
			name:       "start at end of string",
			script:     "[",
			startIndex: 1,
			expected:   1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := v.findClosingBracket(tc.script, tc.startIndex)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPKValidator_ExtractAliasesFromArray(t *testing.T) {
	t.Parallel()

	t.Run("single-quoted aliases", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     make(map[string]bool),
			renderedPartials: make(map[string]bool),
			importedPartials: make(map[string]bool),
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}

		v.extractAliasesFromArray("'Card', 'Header'")

		assert.True(t, v.usedPartials["Card"])
		assert.True(t, v.usedPartials["Header"])
		assert.Len(t, v.usedPartials, 2)
	})

	t.Run("double-quoted aliases", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     make(map[string]bool),
			renderedPartials: make(map[string]bool),
			importedPartials: make(map[string]bool),
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}

		v.extractAliasesFromArray(`"Card", "Header"`)

		assert.True(t, v.usedPartials["Card"])
		assert.True(t, v.usedPartials["Header"])
		assert.Len(t, v.usedPartials, 2)
	})

	t.Run("mixed quotes", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     make(map[string]bool),
			renderedPartials: make(map[string]bool),
			importedPartials: make(map[string]bool),
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}

		v.extractAliasesFromArray(`'Card', "Header"`)

		assert.True(t, v.usedPartials["Card"])
		assert.True(t, v.usedPartials["Header"])
	})

	t.Run("empty string is no-op", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     make(map[string]bool),
			renderedPartials: make(map[string]bool),
			importedPartials: make(map[string]bool),
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}

		v.extractAliasesFromArray("")

		assert.Empty(t, v.usedPartials)
	})

	t.Run("invalid alias not added", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     make(map[string]bool),
			renderedPartials: make(map[string]bool),
			importedPartials: make(map[string]bool),
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}

		v.extractAliasesFromArray("'123invalid', 'Valid'")

		assert.False(t, v.usedPartials["123invalid"])
		assert.True(t, v.usedPartials["Valid"])
	})
}

func TestPKValidator_ReportOrphanedPartials(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver does not panic", func(t *testing.T) {
		t.Parallel()

		var v *PKValidator
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "pkg", "/test.go", "/test.pk")

		assert.NotPanics(t, func() {
			v.ReportOrphanedPartials(ctx, ast_domain.Location{Line: 1, Column: 1, Offset: 0})
		})
		assert.Empty(t, diagnostics)
	})

	t.Run("no imported partials produces no warnings", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     make(map[string]bool),
			renderedPartials: make(map[string]bool),
			importedPartials: make(map[string]bool),
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "pkg", "/test.go", "/test.pk")

		v.ReportOrphanedPartials(ctx, ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		assert.Empty(t, diagnostics)
	})

	t.Run("rendered partial is not orphaned", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     make(map[string]bool),
			renderedPartials: map[string]bool{"Card": true},
			importedPartials: map[string]bool{"Card": true},
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "pkg", "/test.go", "/test.pk")

		v.ReportOrphanedPartials(ctx, ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		assert.Empty(t, diagnostics)
	})

	t.Run("reload-used partial is not orphaned", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     map[string]bool{"Card": true},
			renderedPartials: make(map[string]bool),
			importedPartials: map[string]bool{"Card": true},
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "pkg", "/test.go", "/test.pk")

		v.ReportOrphanedPartials(ctx, ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		assert.Empty(t, diagnostics)
	})

	t.Run("unused imported partial produces warning", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     make(map[string]bool),
			renderedPartials: make(map[string]bool),
			importedPartials: map[string]bool{"Orphan": true},
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "pkg", "/test.go", "/test.pk")

		v.ReportOrphanedPartials(ctx, ast_domain.Location{Line: 1, Column: 1, Offset: 0})

		require.Len(t, diagnostics, 1)
		assert.Equal(t, ast_domain.Warning, diagnostics[0].Severity)
		assert.Contains(t, diagnostics[0].Message, "Orphan")
		assert.Contains(t, diagnostics[0].Message, "never used")
	})
}

func TestPKValidator_ExtractPartialCalls(t *testing.T) {
	t.Parallel()

	t.Run("extracts single-quoted alias from reloadPartial call", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     make(map[string]bool),
			renderedPartials: make(map[string]bool),
			importedPartials: make(map[string]bool),
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}

		v.extractPartialCalls("reloadPartial('Card')", "reloadPartial")

		assert.True(t, v.usedPartials["Card"])
	})

	t.Run("extracts double-quoted alias from reloadPartial call", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     make(map[string]bool),
			renderedPartials: make(map[string]bool),
			importedPartials: make(map[string]bool),
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}

		v.extractPartialCalls(`reloadPartial("Card")`, "reloadPartial")

		assert.True(t, v.usedPartials["Card"])
	})

	t.Run("extracts multiple calls", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     make(map[string]bool),
			renderedPartials: make(map[string]bool),
			importedPartials: make(map[string]bool),
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}

		v.extractPartialCalls("reloadPartial('A'); reloadPartial('B');", "reloadPartial")

		assert.True(t, v.usedPartials["A"])
		assert.True(t, v.usedPartials["B"])
	})

	t.Run("no match produces no entries", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     make(map[string]bool),
			renderedPartials: make(map[string]bool),
			importedPartials: make(map[string]bool),
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}

		v.extractPartialCalls("someOtherFunction('Card')", "reloadPartial")

		assert.Empty(t, v.usedPartials)
	})
}

func TestPKValidator_ExtractReloadGroupCalls(t *testing.T) {
	t.Parallel()

	t.Run("extracts aliases from reloadGroup array", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     make(map[string]bool),
			renderedPartials: make(map[string]bool),
			importedPartials: make(map[string]bool),
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}

		v.extractReloadGroupCalls("reloadGroup(['Card', 'Header'])")

		assert.True(t, v.usedPartials["Card"])
		assert.True(t, v.usedPartials["Header"])
	})

	t.Run("no reloadGroup call produces no entries", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     make(map[string]bool),
			renderedPartials: make(map[string]bool),
			importedPartials: make(map[string]bool),
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}

		v.extractReloadGroupCalls("someOtherCall(['A'])")

		assert.Empty(t, v.usedPartials)
	})

	t.Run("handles space after reloadGroup(", func(t *testing.T) {
		t.Parallel()

		v := &PKValidator{
			clientExports:    nil,
			usedHandlers:     make(map[string]bool),
			usedPartials:     make(map[string]bool),
			renderedPartials: make(map[string]bool),
			importedPartials: make(map[string]bool),
			sfcSourcePath:    "/test.pk",
			clientScript:     "",
		}

		v.extractReloadGroupCalls("reloadGroup( ['Card'])")

		assert.True(t, v.usedPartials["Card"])
	})
}

func TestIsValidPartialAlias_AdditionalCases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		alias    string
		expected bool
	}{
		{name: "underscore prefix", alias: "_Card", expected: true},
		{name: "all underscores", alias: "___", expected: true},
		{name: "digits after first char", alias: "a123", expected: true},
		{name: "starts with digit", alias: "1card", expected: false},
		{name: "contains hyphen", alias: "my-card", expected: false},
		{name: "contains dot", alias: "my.card", expected: false},
		{name: "contains space", alias: "my card", expected: false},
		{name: "single letter", alias: "a", expected: true},
		{name: "mixed case", alias: "MyCard123", expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isValidPartialAlias(tc.alias)

			assert.Equal(t, tc.expected, result)
		})
	}
}
