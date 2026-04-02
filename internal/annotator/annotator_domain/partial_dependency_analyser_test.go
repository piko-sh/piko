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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestExtractReloadedPartials(t *testing.T) {
	testCases := []struct {
		name   string
		script string
		want   []string
	}{
		{
			name:   "single reloadPartial",
			script: `reloadPartial('card')`,
			want:   []string{"card"},
		},
		{
			name:   "multiple reloadPartial",
			script: `reloadPartial('card'); reloadPartial('header');`,
			want:   []string{"card", "header"},
		},
		{
			name:   "reloadGroup",
			script: `reloadGroup(['card', 'header', 'footer'])`,
			want:   []string{"card", "header", "footer"},
		},
		{
			name:   "double quotes",
			script: `reloadPartial("card")`,
			want:   []string{"card"},
		},
		{
			name:   "mixed quotes in group",
			script: `reloadGroup(['card', "header"])`,
			want:   []string{"card", "header"},
		},
		{
			name:   "no partials",
			script: `console.log('hello')`,
			want:   []string{},
		},
		{
			name:   "nested in function",
			script: `export function handleClick() { reloadPartial('card'); }`,
			want:   []string{"card"},
		},
		{
			name: "complex script",
			script: `
				export function handleFilter() {
					const result = await fetch('/api/filter');
					reloadPartial('products');
				}

				export function handleReset() {
					reloadGroup(['products', 'filters']);
				}
			`,
			want: []string{"products", "filters"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractReloadedPartials(tc.script)

			if len(got) != len(tc.want) {
				t.Errorf("got %d partials, want %d; got=%v, want=%v",
					len(got), len(tc.want), mapKeys(got), tc.want)
				return
			}

			for _, want := range tc.want {
				if !got[want] {
					t.Errorf("expected partial '%s' to be extracted", want)
				}
			}
		})
	}
}

func TestPartialDependencyAnalyser_DetectCycles(t *testing.T) {
	testCases := []struct {
		dependencies map[string]map[string]bool
		name         string
		wantCycles   bool
	}{
		{
			name:         "no dependencies",
			dependencies: map[string]map[string]bool{},
			wantCycles:   false,
		},
		{
			name: "linear chain",
			dependencies: map[string]map[string]bool{
				"a": {"b": true},
				"b": {"c": true},
			},
			wantCycles: false,
		},
		{
			name: "simple cycle",
			dependencies: map[string]map[string]bool{
				"a": {"b": true},
				"b": {"a": true},
			},
			wantCycles: true,
		},
		{
			name: "three node cycle",
			dependencies: map[string]map[string]bool{
				"a": {"b": true},
				"b": {"c": true},
				"c": {"a": true},
			},
			wantCycles: true,
		},
		{
			name: "self loop",
			dependencies: map[string]map[string]bool{
				"a": {"a": true},
			},
			wantCycles: true,
		},
		{
			name: "complex with cycle",
			dependencies: map[string]map[string]bool{
				"a": {"b": true, "c": true},
				"b": {"d": true},
				"c": {"d": true},
				"d": {"e": true},
				"e": {"b": true},
				"f": {"a": true},
			},
			wantCycles: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			analyser := &PartialDependencyAnalyser{
				dependencies:     tc.dependencies,
				aliasToComponent: make(map[string]*componentInfo),
			}

			cycles := analyser.detectCycles()
			gotCycles := len(cycles) > 0

			if gotCycles != tc.wantCycles {
				t.Errorf("detectCycles() found cycles = %v, want %v; cycles=%v",
					gotCycles, tc.wantCycles, cycles)
			}
		})
	}
}

func TestNormaliseCycle(t *testing.T) {
	testCases := []struct {
		name  string
		want  string
		cycle []string
	}{
		{
			name:  "already normalised",
			cycle: []string{"a", "b", "c", "a"},
			want:  "a → b → c → a",
		},
		{
			name:  "needs rotation",
			cycle: []string{"c", "a", "b", "c"},
			want:  "a → b → c → a",
		},
		{
			name:  "two elements",
			cycle: []string{"b", "a", "b"},
			want:  "a → b → a",
		},
		{
			name:  "single element",
			cycle: []string{"a", "a"},
			want:  "a → a",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			normalised := normaliseCycle(tc.cycle)
			got := strings.Join(normalised, " → ")
			if got != tc.want {
				t.Errorf("normaliseCycle(%v) = %q, want %q", tc.cycle, got, tc.want)
			}
		})
	}
}

func TestPartialDependencyAnalyser_AnalyseVirtualModule(t *testing.T) {
	t.Run("nil module returns nil", func(t *testing.T) {
		analyser := NewPartialDependencyAnalyser()
		diagnostics := analyser.AnalyseVirtualModule(nil, nil)
		if diagnostics != nil {
			t.Errorf("expected nil diagnostics for nil module, got %v", diagnostics)
		}
	})

	t.Run("empty module no cycles", func(t *testing.T) {
		analyser := NewPartialDependencyAnalyser()
		module := &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
		}
		diagnostics := analyser.AnalyseVirtualModule(module, nil)
		if len(diagnostics) != 0 {
			t.Errorf("expected no diagnostics for empty module, got %d", len(diagnostics))
		}
	})
}

func TestExtractCycle(t *testing.T) {
	testCases := []struct {
		name   string
		path   []string
		target string
		want   []string
	}{
		{
			name:   "full path is cycle",
			path:   []string{"a", "b", "c"},
			target: "a",
			want:   []string{"a", "b", "c", "a"},
		},
		{
			name:   "partial path",
			path:   []string{"x", "a", "b", "c"},
			target: "a",
			want:   []string{"a", "b", "c", "a"},
		},
		{
			name:   "target at end",
			path:   []string{"a", "b"},
			target: "b",
			want:   []string{"b", "b"},
		},
		{
			name:   "target not in path",
			path:   []string{"a", "b", "c"},
			target: "x",
			want:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractCycle(tc.path, tc.target)
			if len(got) != len(tc.want) {
				t.Errorf("extractCycle(%v, %q) = %v, want %v",
					tc.path, tc.target, got, tc.want)
				return
			}
			for i, want := range tc.want {
				if got[i] != want {
					t.Errorf("extractCycle(%v, %q)[%d] = %q, want %q",
						tc.path, tc.target, i, got[i], want)
				}
			}
		})
	}
}

func mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestCyclesToDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for empty cycles", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		result := analyser.cyclesToDiagnostics([][]string{}, nil)

		assert.Nil(t, result)
	})

	t.Run("creates diagnostic for single cycle", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		analyser.aliasToComponent["a"] = &componentInfo{
			sourcePath: "/test/a.piko",
			location:   ast_domain.Location{Line: 5, Column: 10},
		}

		cycles := [][]string{{"a", "b", "a"}}
		result := analyser.cyclesToDiagnostics(cycles, nil)

		require.Len(t, result, 1)
		assert.Equal(t, ast_domain.Error, result[0].Severity)
		assert.Contains(t, result[0].Message, "Circular partial reload dependency detected")
		assert.Contains(t, result[0].Message, "infinite reload loops")
		assert.Equal(t, "/test/a.piko", result[0].SourcePath)
	})

	t.Run("deduplicates normalised cycles", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		analyser.aliasToComponent["a"] = &componentInfo{sourcePath: "/test/a.piko"}
		analyser.aliasToComponent["b"] = &componentInfo{sourcePath: "/test/b.piko"}

		cycles := [][]string{
			{"a", "b", "a"},
			{"b", "a", "b"},
		}
		result := analyser.cyclesToDiagnostics(cycles, nil)

		require.Len(t, result, 1)
	})

	t.Run("uses main component source path when alias not found", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		mainComp := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/main/page.piko",
			},
		}

		cycles := [][]string{{"x", "y", "x"}}
		result := analyser.cyclesToDiagnostics(cycles, mainComp)

		require.Len(t, result, 1)
		assert.Equal(t, "/main/page.piko", result[0].SourcePath)
	})

	t.Run("handles nil main component gracefully", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		cycles := [][]string{{"x", "y", "x"}}
		result := analyser.cyclesToDiagnostics(cycles, nil)

		require.Len(t, result, 1)
		assert.Empty(t, result[0].SourcePath)
	})
}

func TestProcessComponent(t *testing.T) {
	t.Parallel()

	t.Run("skips component with nil source", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		comp := &annotator_dto.VirtualComponent{Source: nil}
		analyser.processComponent(comp)

		assert.Empty(t, analyser.dependencies)
	})

	t.Run("skips component with empty client script", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		comp := &annotator_dto.VirtualComponent{
			HashedName: "comp_abc",
			Source: &annotator_dto.ParsedComponent{
				SourcePath:   "/test/comp.piko",
				ClientScript: "",
				PikoImports:  []annotator_dto.PikoImport{},
			},
		}
		analyser.processComponent(comp)

		assert.Empty(t, analyser.dependencies)
	})

	t.Run("records dependencies from client script", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		comp := &annotator_dto.VirtualComponent{
			HashedName: "main_comp",
			Source: &annotator_dto.ParsedComponent{
				SourcePath:   "/test/page.piko",
				ClientScript: `reloadPartial('card')`,
				PikoImports: []annotator_dto.PikoImport{
					{Alias: "card", Path: "card.pk"},
				},
			},
		}
		analyser.processComponent(comp)

		assert.True(t, analyser.dependencies["main_comp"]["card"])
	})

	t.Run("uses partial name when available", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		comp := &annotator_dto.VirtualComponent{
			HashedName:  "comp_abc",
			PartialName: "my-partial",
			Source: &annotator_dto.ParsedComponent{
				SourcePath:   "/test/partial.piko",
				ClientScript: `reloadPartial('other')`,
				PikoImports: []annotator_dto.PikoImport{
					{Alias: "other", Path: "other.pk"},
				},
			},
		}
		analyser.processComponent(comp)

		assert.Contains(t, analyser.dependencies, "my-partial")
	})
}

func TestRecordDependencies(t *testing.T) {
	t.Parallel()

	t.Run("only records aliases that are in import list", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		comp := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				ClientScript: `reloadPartial('card'); reloadPartial('unknown')`,
			},
		}

		importAliases := map[string]bool{"card": true}
		analyser.recordDependencies(comp, "source", importAliases)

		require.NotNil(t, analyser.dependencies["source"])
		assert.True(t, analyser.dependencies["source"]["card"])
		assert.False(t, analyser.dependencies["source"]["unknown"])
	})

	t.Run("handles empty import aliases", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		comp := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				ClientScript: `reloadPartial('card')`,
			},
		}

		analyser.recordDependencies(comp, "source", map[string]bool{})

		assert.Empty(t, analyser.dependencies)
	})
}

func TestCollectImportAliases(t *testing.T) {
	t.Parallel()

	t.Run("collects non-blank aliases", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		comp := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				PikoImports: []annotator_dto.PikoImport{
					{Alias: "card", Path: "card.pk"},
					{Alias: "header", Path: "header.pk"},
				},
			},
		}

		result := analyser.collectImportAliases(comp)

		assert.True(t, result["card"])
		assert.True(t, result["header"])
		assert.Len(t, result, 2)
	})

	t.Run("skips blank and underscore aliases", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		comp := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				PikoImports: []annotator_dto.PikoImport{
					{Alias: "", Path: "anon.pk"},
					{Alias: "_", Path: "side-effect.pk"},
					{Alias: "valid", Path: "valid.pk"},
				},
			},
		}

		result := analyser.collectImportAliases(comp)

		assert.Len(t, result, 1)
		assert.True(t, result["valid"])
	})
}

func TestGetSourceName(t *testing.T) {
	t.Parallel()

	t.Run("returns partial name when set", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		comp := &annotator_dto.VirtualComponent{
			HashedName:  "hashed_abc",
			PartialName: "my-partial",
		}

		assert.Equal(t, "my-partial", analyser.getSourceName(comp))
	})

	t.Run("returns hashed name when partial name is empty", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		comp := &annotator_dto.VirtualComponent{
			HashedName:  "hashed_abc",
			PartialName: "",
		}

		assert.Equal(t, "hashed_abc", analyser.getSourceName(comp))
	})
}

func TestRegisterImportAliases(t *testing.T) {
	t.Parallel()

	t.Run("registers valid import aliases", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		comp := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/comp.piko",
				PikoImports: []annotator_dto.PikoImport{
					{
						Alias:    "card",
						Path:     "card.pk",
						Location: ast_domain.Location{Line: 3, Column: 1},
					},
				},
			},
		}

		analyser.registerImportAliases(comp)

		require.NotNil(t, analyser.aliasToComponent["card"])
		assert.Equal(t, "/test/comp.piko", analyser.aliasToComponent["card"].sourcePath)
		assert.Equal(t, 3, analyser.aliasToComponent["card"].location.Line)
	})

	t.Run("skips empty and underscore aliases", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		comp := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				PikoImports: []annotator_dto.PikoImport{
					{Alias: "", Path: "anon.pk"},
					{Alias: "_", Path: "side.pk"},
				},
			},
		}

		analyser.registerImportAliases(comp)

		assert.Empty(t, analyser.aliasToComponent)
	})
}

func TestExtractPartialCallsToMap(t *testing.T) {
	t.Parallel()

	t.Run("extracts single-quoted calls", func(t *testing.T) {
		t.Parallel()

		result := make(map[string]bool)
		extractPartialCallsToMap("reloadPartial('card')", "reloadPartial", result)

		assert.True(t, result["card"])
	})

	t.Run("extracts double-quoted calls", func(t *testing.T) {
		t.Parallel()

		result := make(map[string]bool)
		extractPartialCallsToMap(`reloadPartial("card")`, "reloadPartial", result)

		assert.True(t, result["card"])
	})

	t.Run("extracts multiple calls", func(t *testing.T) {
		t.Parallel()

		result := make(map[string]bool)
		extractPartialCallsToMap("reloadPartial('a'); reloadPartial('b')", "reloadPartial", result)

		assert.True(t, result["a"])
		assert.True(t, result["b"])
	})

	t.Run("handles no matches", func(t *testing.T) {
		t.Parallel()

		result := make(map[string]bool)
		extractPartialCallsToMap("console.log('hello')", "reloadPartial", result)

		assert.Empty(t, result)
	})

	t.Run("skips empty alias", func(t *testing.T) {
		t.Parallel()

		result := make(map[string]bool)
		extractPartialCallsToMap("reloadPartial('')", "reloadPartial", result)

		assert.Empty(t, result)
	})
}

func TestFindMatchingBracket(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		script     string
		startIndex int
		expected   int
	}{
		{
			name:       "simple bracket",
			script:     "['a', 'b'])",
			startIndex: 1,
			expected:   10,
		},
		{
			name:       "nested brackets",
			script:     "['a', ['b']])",
			startIndex: 1,
			expected:   12,
		},
		{
			name:       "unmatched bracket returns startIndex",
			script:     "['a', 'b'",
			startIndex: 1,
			expected:   1,
		},
		{
			name:       "empty content",
			script:     "]",
			startIndex: 0,
			expected:   1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := findMatchingBracket(tc.script, tc.startIndex)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractAliasesFromArrayToMap(t *testing.T) {
	t.Parallel()

	t.Run("extracts single-quoted aliases", func(t *testing.T) {
		t.Parallel()

		result := make(map[string]bool)
		extractAliasesFromArrayToMap("'card', 'header'", result)

		assert.True(t, result["card"])
		assert.True(t, result["header"])
	})

	t.Run("extracts double-quoted aliases", func(t *testing.T) {
		t.Parallel()

		result := make(map[string]bool)
		extractAliasesFromArrayToMap(`"card", "header"`, result)

		assert.True(t, result["card"])
		assert.True(t, result["header"])
	})

	t.Run("extracts mixed quotes", func(t *testing.T) {
		t.Parallel()

		result := make(map[string]bool)
		extractAliasesFromArrayToMap(`'card', "header"`, result)

		assert.True(t, result["card"])
		assert.True(t, result["header"])
	})

	t.Run("skips empty aliases", func(t *testing.T) {
		t.Parallel()

		result := make(map[string]bool)
		extractAliasesFromArrayToMap("'', \"\"", result)

		assert.Empty(t, result)
	})

	t.Run("handles whitespace", func(t *testing.T) {
		t.Parallel()

		result := make(map[string]bool)
		extractAliasesFromArrayToMap("  'card'  ,  'header'  ", result)

		assert.True(t, result["card"])
		assert.True(t, result["header"])
	})

	t.Run("skips invalid aliases", func(t *testing.T) {
		t.Parallel()

		result := make(map[string]bool)
		extractAliasesFromArrayToMap("'valid', '123invalid'", result)

		assert.True(t, result["valid"])
		assert.False(t, result["123invalid"])
	})
}

func TestExtractReloadGroupCallsToMap(t *testing.T) {
	t.Parallel()

	t.Run("extracts from reloadGroup call", func(t *testing.T) {
		t.Parallel()

		result := make(map[string]bool)
		extractReloadGroupCallsToMap("reloadGroup(['card', 'header'])", result)

		assert.True(t, result["card"])
		assert.True(t, result["header"])
	})

	t.Run("extracts from reloadGroup with space", func(t *testing.T) {
		t.Parallel()

		result := make(map[string]bool)
		extractReloadGroupCallsToMap("reloadGroup( ['card'])", result)

		assert.True(t, result["card"])
	})

	t.Run("handles multiple reloadGroup calls", func(t *testing.T) {
		t.Parallel()

		result := make(map[string]bool)
		extractReloadGroupCallsToMap("reloadGroup(['a']); reloadGroup(['b'])", result)

		assert.True(t, result["a"])
		assert.True(t, result["b"])
	})
}

func TestNewPartialDependencyAnalyser(t *testing.T) {
	t.Parallel()

	t.Run("creates analyser with initialised maps", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()

		require.NotNil(t, analyser)
		assert.NotNil(t, analyser.dependencies)
		assert.NotNil(t, analyser.aliasToComponent)
		assert.Empty(t, analyser.dependencies)
		assert.Empty(t, analyser.aliasToComponent)
	})
}

func TestAnalyseVirtualModule_WithCycle(t *testing.T) {
	t.Parallel()

	t.Run("detects cycle across components with matching aliases", func(t *testing.T) {
		t.Parallel()

		analyser := NewPartialDependencyAnalyser()
		module := &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"alpha": {
					HashedName: "alpha",
					Source: &annotator_dto.ParsedComponent{
						SourcePath:   "/test/alpha.piko",
						ClientScript: `reloadPartial('beta')`,
						PikoImports: []annotator_dto.PikoImport{
							{Alias: "beta", Path: "beta.pk"},
						},
					},
				},
				"beta": {
					HashedName: "beta",
					Source: &annotator_dto.ParsedComponent{
						SourcePath:   "/test/beta.piko",
						ClientScript: `reloadPartial('alpha')`,
						PikoImports: []annotator_dto.PikoImport{
							{Alias: "alpha", Path: "alpha.pk"},
						},
					},
				},
			},
		}

		mainComp := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/main.piko",
			},
		}

		diagnostics := analyser.AnalyseVirtualModule(module, mainComp)

		require.NotEmpty(t, diagnostics)
		assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
		assert.Contains(t, diagnostics[0].Message, "Circular partial reload dependency")
	})
}

func TestNormaliseCycle_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty cycle returns empty", func(t *testing.T) {
		t.Parallel()

		result := normaliseCycle([]string{})
		assert.Empty(t, result)
	})

	t.Run("single element cycle returns as-is", func(t *testing.T) {
		t.Parallel()

		result := normaliseCycle([]string{"a"})
		assert.Equal(t, []string{"a"}, result)
	})
}
