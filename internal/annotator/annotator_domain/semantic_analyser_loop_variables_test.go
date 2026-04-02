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

func TestLoopVariableManager_ValidateLoopVariable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		variableName            string
		expectedMessageContains string
		expectedDiagCount       int
		expectedSeverity        ast_domain.Severity
	}{
		{
			name:              "safe variable name",
			variableName:      "item",
			expectedDiagCount: 0,
		},
		{
			name:              "safe variable name - user",
			variableName:      "user",
			expectedDiagCount: 0,
		},
		{
			name:                    "shadows reserved symbol - request",
			variableName:            "request",
			expectedDiagCount:       1,
			expectedSeverity:        ast_domain.Warning,
			expectedMessageContains: "shadows a built-in Piko system symbol",
		},
		{
			name:                    "shadows reserved symbol - state",
			variableName:            "state",
			expectedDiagCount:       1,
			expectedSeverity:        ast_domain.Warning,
			expectedMessageContains: "shadows a built-in Piko system symbol",
		},
		{
			name:                    "shadows reserved symbol - props",
			variableName:            "props",
			expectedDiagCount:       1,
			expectedSeverity:        ast_domain.Warning,
			expectedMessageContains: "shadows a built-in Piko system symbol",
		},
		{
			name:                    "shadows reserved symbol - req",
			variableName:            "req",
			expectedDiagCount:       1,
			expectedSeverity:        ast_domain.Warning,
			expectedMessageContains: "shadows a built-in Piko system symbol",
		},
		{
			name:                    "shadows reserved symbol - r",
			variableName:            "r",
			expectedDiagCount:       1,
			expectedSeverity:        ast_domain.Warning,
			expectedMessageContains: "shadows a built-in Piko system symbol",
		},
		{
			name:                    "shadows reserved symbol - s",
			variableName:            "s",
			expectedDiagCount:       1,
			expectedSeverity:        ast_domain.Warning,
			expectedMessageContains: "shadows a built-in Piko system symbol",
		},
		{
			name:                    "shadows reserved symbol - p",
			variableName:            "p",
			expectedDiagCount:       1,
			expectedSeverity:        ast_domain.Warning,
			expectedMessageContains: "shadows a built-in Piko system symbol",
		},
		{
			name:                    "shadows reserved symbol - T",
			variableName:            "T",
			expectedDiagCount:       1,
			expectedSeverity:        ast_domain.Warning,
			expectedMessageContains: "shadows a built-in Piko system symbol",
		},
		{
			name:                    "shadows reserved symbol - LT",
			variableName:            "LT",
			expectedDiagCount:       1,
			expectedSeverity:        ast_domain.Warning,
			expectedMessageContains: "shadows a built-in Piko system symbol",
		},
		{
			name:                    "shadows built-in function - len",
			variableName:            "len",
			expectedDiagCount:       1,
			expectedSeverity:        ast_domain.Warning,
			expectedMessageContains: "shadows a global built-in function",
		},
		{
			name:                    "shadows built-in function - cap",
			variableName:            "cap",
			expectedDiagCount:       1,
			expectedSeverity:        ast_domain.Warning,
			expectedMessageContains: "shadows a global built-in function",
		},
		{
			name:                    "shadows built-in function - append",
			variableName:            "append",
			expectedDiagCount:       1,
			expectedSeverity:        ast_domain.Warning,
			expectedMessageContains: "shadows a global built-in function",
		},
		{
			name:                    "shadows built-in function - min",
			variableName:            "min",
			expectedDiagCount:       1,
			expectedSeverity:        ast_domain.Warning,
			expectedMessageContains: "shadows a global built-in function",
		},
		{
			name:                    "shadows built-in function - max",
			variableName:            "max",
			expectedDiagCount:       1,
			expectedSeverity:        ast_domain.Warning,
			expectedMessageContains: "shadows a global built-in function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			diagnostics := make([]*ast_domain.Diagnostic, 0)
			ctx := NewRootAnalysisContext(
				&diagnostics,
				"test/package",
				"testpkg",
				"test.go",
				"test.piko",
			)
			manager := newLoopVariableManager(ctx)
			directive := &ast_domain.Directive{
				Type:     ast_domain.DirectiveFor,
				Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			}
			identifier := &ast_domain.Identifier{
				Name:             tt.variableName,
				RelativeLocation: ast_domain.Location{Line: 0, Column: 5, Offset: 0},
			}

			manager.ValidateLoopVariable(identifier, directive)

			if len(diagnostics) != tt.expectedDiagCount {
				t.Errorf("Expected %d diagnostics, got %d", tt.expectedDiagCount, len(diagnostics))
				for i, d := range diagnostics {
					t.Logf("  Diagnostic %d: [%s] %s", i, d.Severity, d.Message)
				}
				return
			}

			if tt.expectedDiagCount > 0 {
				diagnostic := diagnostics[0]
				if diagnostic.Severity != tt.expectedSeverity {
					t.Errorf("Expected severity %s, got %s", tt.expectedSeverity, diagnostic.Severity)
				}
				if tt.expectedMessageContains != "" {
					if !contains(diagnostic.Message, tt.expectedMessageContains) {
						t.Errorf("Expected message to contain '%s', got '%s'", tt.expectedMessageContains, diagnostic.Message)
					}
				}
			}
		})
	}
}

func TestLoopVariableManager_ValidateLoopVariable_NilCases(t *testing.T) {
	t.Parallel()

	diagnostics := make([]*ast_domain.Diagnostic, 0)
	ctx := NewRootAnalysisContext(
		&diagnostics,
		"test/package",
		"testpkg",
		"test.go",
		"test.piko",
	)
	manager := newLoopVariableManager(ctx)

	directive := &ast_domain.Directive{
		Type:     ast_domain.DirectiveFor,
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	manager.ValidateLoopVariable(nil, directive)

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics for nil variable, got %d", len(diagnostics))
	}
}

func TestLoopVariableManager_GenerateUniqueLoopVarName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		expectedName    string
		existingSymbols []string
		depth           int
	}{
		{
			name:            "base name available",
			existingSymbols: []string{},
			depth:           0,
			expectedName:    "__pikoLoopIdx",
		},
		{
			name:            "base name taken - use suffix 2",
			existingSymbols: []string{"__pikoLoopIdx"},
			depth:           1,
			expectedName:    "__pikoLoopIdx2",
		},
		{
			name:            "base and suffix 2 taken - use suffix 3",
			existingSymbols: []string{"__pikoLoopIdx", "__pikoLoopIdx2"},
			depth:           2,
			expectedName:    "__pikoLoopIdx3",
		},
		{
			name:            "multiple nested levels",
			existingSymbols: []string{"__pikoLoopIdx", "__pikoLoopIdx2", "__pikoLoopIdx3", "__pikoLoopIdx4"},
			depth:           4,
			expectedName:    "__pikoLoopIdx5",
		},
		{
			name:            "non-sequential gap - should find first available",
			existingSymbols: []string{"__pikoLoopIdx", "__pikoLoopIdx3"},
			depth:           1,
			expectedName:    "__pikoLoopIdx2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := NewRootAnalysisContext(
				new([]*ast_domain.Diagnostic),
				"test/package",
				"testpkg",
				"test.go",
				"test.piko",
			)
			for _, symbolName := range tt.existingSymbols {
				ctx.Symbols.Define(Symbol{
					Name: symbolName,
				})
			}
			manager := newLoopVariableManager(ctx)

			result := manager.GenerateUniqueLoopVarName(tt.depth)

			if result != tt.expectedName {
				t.Errorf("Expected name '%s', got '%s'", tt.expectedName, result)
			}

			if _, found := ctx.Symbols.Find(result); found {
				t.Errorf("Generated name '%s' should not already exist in symbol table", result)
			}
		})
	}
}

func TestLoopVariableManager_GenerateUniqueLoopVarName_FallbackCase(t *testing.T) {
	t.Parallel()

	ctx := NewRootAnalysisContext(
		new([]*ast_domain.Diagnostic),
		"test/package",
		"testpkg",
		"test.go",
		"test.piko",
	)

	ctx.Symbols.Define(Symbol{Name: "__pikoLoopIdx"})
	for i := 2; i < 100; i++ {
		ctx.Symbols.Define(Symbol{Name: "__pikoLoopIdx" + intToString(i)})
	}

	manager := newLoopVariableManager(ctx)
	depth := 50

	result := manager.GenerateUniqueLoopVarName(depth)

	expected := "__pikoLoopIdx50"
	if result != expected {
		t.Errorf("Expected fallback name '%s', got '%s'", expected, result)
	}
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}

	digits := []rune{}
	for n > 0 {
		digits = append([]rune{rune('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) != -1
}

func findSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestLoopVariableManager_pool(t *testing.T) {
	t.Parallel()

	t.Run("get and put cycle works", func(t *testing.T) {
		t.Parallel()

		ctx := createTestAnalysisContext()
		lvm := getLoopVariableManager(ctx)

		require.NotNil(t, lvm)
		assert.Equal(t, ctx, lvm.ctx)

		putLoopVariableManager(lvm)
	})

	t.Run("multiple get calls return valid managers", func(t *testing.T) {
		t.Parallel()

		ctx1 := createTestAnalysisContext()
		ctx2 := createTestAnalysisContext()

		lvm1 := getLoopVariableManager(ctx1)
		lvm2 := getLoopVariableManager(ctx2)

		require.NotNil(t, lvm1)
		require.NotNil(t, lvm2)
		assert.Equal(t, ctx1, lvm1.ctx)
		assert.Equal(t, ctx2, lvm2.ctx)

		putLoopVariableManager(lvm1)
		putLoopVariableManager(lvm2)
	})
}
