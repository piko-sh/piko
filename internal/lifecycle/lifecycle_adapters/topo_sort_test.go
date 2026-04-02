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

package lifecycle_adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/generator/generator_dto"
)

func TestExtractRelativePath(t *testing.T) {
	t.Parallel()

	sorter := &topologicalSorter{}

	tests := []struct {
		name       string
		importPath string
		want       string
	}{
		{name: "with slash", importPath: "myproject/components/header.pk", want: "components/header.pk"},
		{name: "no slash", importPath: "header.pk", want: "header.pk"},
		{name: "multiple slashes", importPath: "myproject/components/ui/button.pk", want: "components/ui/button.pk"},
		{name: "empty string", importPath: "", want: ""},
		{name: "only slash", importPath: "/", want: ""},
		{name: "leading slash", importPath: "/components/header.pk", want: "components/header.pk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sorter.extractRelativePath(tt.importPath)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestInitialiseQueue(t *testing.T) {
	t.Parallel()

	t.Run("all zero in-degree", func(t *testing.T) {
		t.Parallel()

		sorter := &topologicalSorter{
			allPaths: []string{"a", "b", "c"},
			inDegree: map[string]int{"a": 0, "b": 0, "c": 0},
		}

		queue := sorter.initialiseQueue()
		assert.Equal(t, []string{"a", "b", "c"}, queue)
	})

	t.Run("mixed in-degrees", func(t *testing.T) {
		t.Parallel()

		sorter := &topologicalSorter{
			allPaths: []string{"a", "b", "c", "d"},
			inDegree: map[string]int{"a": 0, "b": 1, "c": 0, "d": 2},
		}

		queue := sorter.initialiseQueue()
		assert.Equal(t, []string{"a", "c"}, queue)
	})

	t.Run("no zero in-degree", func(t *testing.T) {
		t.Parallel()

		sorter := &topologicalSorter{
			allPaths: []string{"a", "b"},
			inDegree: map[string]int{"a": 1, "b": 1},
		}

		queue := sorter.initialiseQueue()
		assert.Nil(t, queue)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		sorter := &topologicalSorter{
			allPaths: []string{},
			inDegree: map[string]int{},
		}

		queue := sorter.initialiseQueue()
		assert.Nil(t, queue)
	})
}

func TestProcessNodeDependents(t *testing.T) {
	t.Parallel()

	t.Run("decrements and enqueues ready nodes", func(t *testing.T) {
		t.Parallel()

		sorter := &topologicalSorter{
			adjacency: map[string][]string{
				"a": {"b", "c"},
			},
			inDegree: map[string]int{"b": 1, "c": 2},
		}

		queue := sorter.processNodeDependents("a", nil)
		assert.Equal(t, []string{"b"}, queue)
		assert.Equal(t, 0, sorter.inDegree["b"])
		assert.Equal(t, 1, sorter.inDegree["c"])
	})

	t.Run("no dependents", func(t *testing.T) {
		t.Parallel()

		sorter := &topologicalSorter{
			adjacency: map[string][]string{
				"a": {},
			},
			inDegree: map[string]int{},
		}

		queue := sorter.processNodeDependents("a", []string{"x"})
		assert.Equal(t, []string{"x"}, queue)
	})

	t.Run("multiple become ready", func(t *testing.T) {
		t.Parallel()

		sorter := &topologicalSorter{
			adjacency: map[string][]string{
				"root": {"x", "y", "z"},
			},
			inDegree: map[string]int{"x": 1, "y": 1, "z": 1},
		}

		queue := sorter.processNodeDependents("root", nil)
		assert.Equal(t, []string{"x", "y", "z"}, queue)
	})
}

func TestValidateAndReturn(t *testing.T) {
	t.Parallel()

	t.Run("valid sort", func(t *testing.T) {
		t.Parallel()

		artefact1 := &generator_dto.GeneratedArtefact{SuggestedPath: "a.go"}
		artefact2 := &generator_dto.GeneratedArtefact{SuggestedPath: "b.go"}

		sorter := &topologicalSorter{
			artefactByPackagePath: map[string]*generator_dto.GeneratedArtefact{
				"pkg/a": artefact1,
				"pkg/b": artefact2,
			},
		}

		sorted := []*generator_dto.GeneratedArtefact{artefact1, artefact2}
		result, err := sorter.validateAndReturn(sorted)
		require.NoError(t, err)
		assert.Equal(t, sorted, result)
	})

	t.Run("cycle detected", func(t *testing.T) {
		t.Parallel()

		sorter := &topologicalSorter{
			artefactByPackagePath: map[string]*generator_dto.GeneratedArtefact{
				"pkg/a": {},
				"pkg/b": {},
				"pkg/c": {},
			},
		}

		sorted := []*generator_dto.GeneratedArtefact{{}}
		_, err := sorter.validateAndReturn(sorted)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "circular dependency detected")
		assert.Contains(t, err.Error(), "2 of 3")
	})

	t.Run("empty is valid", func(t *testing.T) {
		t.Parallel()

		sorter := &topologicalSorter{
			artefactByPackagePath: map[string]*generator_dto.GeneratedArtefact{},
		}

		result, err := sorter.validateAndReturn(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestExtractImportRelativePath(t *testing.T) {
	t.Parallel()

	o := &InterpretedBuildOrchestrator{}

	tests := []struct {
		name       string
		importPath string
		want       string
	}{
		{name: "with slash", importPath: "myproject/components/card.pk", want: "components/card.pk"},
		{name: "no slash", importPath: "card.pk", want: "card.pk"},
		{name: "multiple slashes", importPath: "proj/a/b/c.pk", want: "a/b/c.pk"},
		{name: "empty string", importPath: "", want: ""},
		{name: "leading slash", importPath: "/path/to/file.pk", want: "path/to/file.pk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := o.extractImportRelativePath(tt.importPath)
			assert.Equal(t, tt.want, result)
		})
	}
}
