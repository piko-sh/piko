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

package annotator_dto

import (
	"sort"
	"testing"
)

func TestBuildReverseDependencyMapFromGraph(t *testing.T) {
	t.Parallel()

	t.Run("single import creates reverse edge", func(t *testing.T) {
		t.Parallel()

		graph := &ComponentGraph{
			Components: map[string]*ParsedComponent{
				"login_hash": {
					SourcePath: "/project/pages/login.pk",
					PikoImports: []PikoImport{
						{Path: "mymodule/components/card.pk"},
					},
				},
			},
		}

		result := BuildReverseDependencyMapFromGraph(graph, "/project")

		deps := result["components/card.pk"]
		if len(deps) != 1 || deps[0] != "pages/login.pk" {
			t.Fatalf("expected [pages/login.pk], got %v", deps)
		}
	})

	t.Run("multiple importers of same component", func(t *testing.T) {
		t.Parallel()

		graph := &ComponentGraph{
			Components: map[string]*ParsedComponent{
				"login_hash": {
					SourcePath: "/project/pages/login.pk",
					PikoImports: []PikoImport{
						{Path: "mymodule/components/card.pk"},
					},
				},
				"settings_hash": {
					SourcePath: "/project/pages/settings.pk",
					PikoImports: []PikoImport{
						{Path: "mymodule/components/card.pk"},
					},
				},
			},
		}

		result := BuildReverseDependencyMapFromGraph(graph, "/project")

		deps := result["components/card.pk"]
		sort.Strings(deps)
		if len(deps) != 2 {
			t.Fatalf("expected 2 dependents, got %d: %v", len(deps), deps)
		}
		if deps[0] != "pages/login.pk" || deps[1] != "pages/settings.pk" {
			t.Fatalf("expected [pages/login.pk, pages/settings.pk], got %v", deps)
		}
	})

	t.Run("component with multiple imports", func(t *testing.T) {
		t.Parallel()

		graph := &ComponentGraph{
			Components: map[string]*ParsedComponent{
				"login_hash": {
					SourcePath: "/project/pages/login.pk",
					PikoImports: []PikoImport{
						{Path: "mymodule/components/card.pk"},
						{Path: "mymodule/components/button.pk"},
					},
				},
			},
		}

		result := BuildReverseDependencyMapFromGraph(graph, "/project")

		if len(result["components/card.pk"]) != 1 {
			t.Fatalf("expected 1 dependent for card, got %v", result["components/card.pk"])
		}
		if len(result["components/button.pk"]) != 1 {
			t.Fatalf("expected 1 dependent for button, got %v", result["components/button.pk"])
		}
	})

	t.Run("empty graph returns empty map", func(t *testing.T) {
		t.Parallel()

		graph := &ComponentGraph{
			Components: map[string]*ParsedComponent{},
		}

		result := BuildReverseDependencyMapFromGraph(graph, "/project")

		if len(result) != 0 {
			t.Fatalf("expected empty map, got %v", result)
		}
	})

	t.Run("nil components map returns empty map", func(t *testing.T) {
		t.Parallel()

		graph := &ComponentGraph{}

		result := BuildReverseDependencyMapFromGraph(graph, "/project")

		if len(result) != 0 {
			t.Fatalf("expected empty map, got %v", result)
		}
	})

	t.Run("import path without slash kept as-is", func(t *testing.T) {
		t.Parallel()

		graph := &ComponentGraph{
			Components: map[string]*ParsedComponent{
				"login_hash": {
					SourcePath: "/project/pages/login.pk",
					PikoImports: []PikoImport{
						{Path: "bare_import"},
					},
				},
			},
		}

		result := BuildReverseDependencyMapFromGraph(graph, "/project")

		deps := result["bare_import"]
		if len(deps) != 1 || deps[0] != "pages/login.pk" {
			t.Fatalf("expected [pages/login.pk], got %v", deps)
		}
	})
}

func TestGetTransitiveDependents(t *testing.T) {
	t.Parallel()

	t.Run("direct dependents only", func(t *testing.T) {
		t.Parallel()

		reverseDeps := map[string][]string{
			"components/card.pk": {"pages/login.pk", "pages/settings.pk"},
		}

		affected := GetTransitiveDependents(reverseDeps, "components/card.pk")
		sort.Strings(affected)

		if len(affected) != 2 {
			t.Fatalf("expected 2 affected, got %d: %v", len(affected), affected)
		}
		if affected[0] != "pages/login.pk" || affected[1] != "pages/settings.pk" {
			t.Fatalf("unexpected affected: %v", affected)
		}
	})

	t.Run("transitive dependents via chain", func(t *testing.T) {
		t.Parallel()

		reverseDeps := map[string][]string{
			"partials/c.pk":   {"components/b.pk"},
			"components/b.pk": {"pages/a.pk"},
		}

		affected := GetTransitiveDependents(reverseDeps, "partials/c.pk")
		sort.Strings(affected)

		if len(affected) != 2 {
			t.Fatalf("expected 2 affected, got %d: %v", len(affected), affected)
		}
		if affected[0] != "components/b.pk" || affected[1] != "pages/a.pk" {
			t.Fatalf("unexpected affected: %v", affected)
		}
	})

	t.Run("cycle terminates without infinite loop", func(t *testing.T) {
		t.Parallel()

		reverseDeps := map[string][]string{
			"a.pk": {"b.pk"},
			"b.pk": {"a.pk"},
		}

		affected := GetTransitiveDependents(reverseDeps, "a.pk")
		sort.Strings(affected)

		if len(affected) != 2 {
			t.Fatalf("expected 2 affected, got %d: %v", len(affected), affected)
		}
		if affected[0] != "a.pk" || affected[1] != "b.pk" {
			t.Fatalf("unexpected affected: %v", affected)
		}
	})

	t.Run("no dependents returns empty slice", func(t *testing.T) {
		t.Parallel()

		reverseDeps := map[string][]string{
			"components/card.pk": {"pages/login.pk"},
		}

		affected := GetTransitiveDependents(reverseDeps, "pages/login.pk")

		if len(affected) != 0 {
			t.Fatalf("expected 0 affected, got %d: %v", len(affected), affected)
		}
	})

	t.Run("empty map returns empty slice", func(t *testing.T) {
		t.Parallel()

		affected := GetTransitiveDependents(map[string][]string{}, "anything.pk")

		if len(affected) != 0 {
			t.Fatalf("expected 0 affected, got %v", affected)
		}
	})

	t.Run("diamond dependency deduplicates", func(t *testing.T) {
		t.Parallel()

		reverseDeps := map[string][]string{
			"partials/d.pk":   {"components/b.pk", "components/c.pk"},
			"components/b.pk": {"pages/a.pk"},
			"components/c.pk": {"pages/a.pk"},
		}

		affected := GetTransitiveDependents(reverseDeps, "partials/d.pk")
		sort.Strings(affected)

		if len(affected) != 3 {
			t.Fatalf("expected 3 affected (b, c, a), got %d: %v", len(affected), affected)
		}
		if affected[0] != "components/b.pk" || affected[1] != "components/c.pk" || affected[2] != "pages/a.pk" {
			t.Fatalf("unexpected affected: %v", affected)
		}
	})
}

func TestFilterEntryPointsByRelativePaths(t *testing.T) {
	t.Parallel()

	allEntryPoints := []EntryPoint{
		{Path: "mymodule/pages/login.pk", IsPage: true},
		{Path: "mymodule/pages/settings.pk", IsPage: true},
		{Path: "mymodule/partials/header.pk", IsPage: false},
		{Path: "mymodule/pages/about.pk", IsPage: true},
	}

	t.Run("filters to matching paths only", func(t *testing.T) {
		t.Parallel()

		filtered := FilterEntryPointsByRelativePaths(
			allEntryPoints,
			[]string{"pages/login.pk", "pages/settings.pk"},
			"mymodule",
		)

		if len(filtered) != 2 {
			t.Fatalf("expected 2 filtered, got %d: %v", len(filtered), filtered)
		}
		if filtered[0].Path != "mymodule/pages/login.pk" {
			t.Fatalf("expected login.pk first, got %s", filtered[0].Path)
		}
		if filtered[1].Path != "mymodule/pages/settings.pk" {
			t.Fatalf("expected settings.pk second, got %s", filtered[1].Path)
		}
	})

	t.Run("no matches returns empty slice", func(t *testing.T) {
		t.Parallel()

		filtered := FilterEntryPointsByRelativePaths(
			allEntryPoints,
			[]string{"pages/nonexistent.pk"},
			"mymodule",
		)

		if len(filtered) != 0 {
			t.Fatalf("expected empty, got %v", filtered)
		}
	})

	t.Run("empty relPaths returns empty slice", func(t *testing.T) {
		t.Parallel()

		filtered := FilterEntryPointsByRelativePaths(
			allEntryPoints,
			[]string{},
			"mymodule",
		)

		if len(filtered) != 0 {
			t.Fatalf("expected empty, got %v", filtered)
		}
	})

	t.Run("preserves entry point metadata", func(t *testing.T) {
		t.Parallel()

		filtered := FilterEntryPointsByRelativePaths(
			allEntryPoints,
			[]string{"partials/header.pk"},
			"mymodule",
		)

		if len(filtered) != 1 {
			t.Fatalf("expected 1, got %d", len(filtered))
		}
		if filtered[0].IsPage {
			t.Fatal("expected IsPage=false for partial")
		}
	})
}

func TestExtractImportRelativePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"module prefixed path", "mymodule/components/card.pk", "components/card.pk"},
		{"deeply nested path", "mymodule/components/ui/button.pk", "components/ui/button.pk"},
		{"bare import without slash", "card", "card"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := extractImportRelativePath(tt.input)
			if result != tt.expected {
				t.Fatalf("extractImportRelativePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
