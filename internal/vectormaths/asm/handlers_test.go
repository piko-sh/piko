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

package asm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/asmgen"
)

func TestVectormathsFileGroupsReturnsAllExpectedGroups(t *testing.T) {
	groups := FileGroups()

	require.Len(t, groups, 3, "expected 3 file groups")

	expectedBaseNames := []string{
		"asm_dot_f32",
		"asm_euclid_sq_f32",
		"asm_normalise_f32",
	}

	for i, group := range groups {
		assert.Equal(t, expectedBaseNames[i], group.BaseName, "group %d base name mismatch", i)
	}
}

func TestAllVectormathsHandlersHaveRequiredFields(t *testing.T) {
	groups := FileGroups()

	for _, group := range groups {
		for i, handler := range group.Handlers {
			t.Run(group.BaseName+"/"+handler.Name, func(t *testing.T) {
				assert.NotEmpty(t, handler.Name, "handler %d in %s has empty Name", i, group.BaseName)

				hasComment := handler.Comment != "" || handler.CommentFunction != nil
				assert.True(t, hasComment, "handler %q in %s has neither Comment nor CommentFunction", handler.Name, group.BaseName)

				assert.NotEmpty(t, handler.FrameSize, "handler %q in %s has empty FrameSize", handler.Name, group.BaseName)
				assert.NotEmpty(t, handler.Flags, "handler %q in %s has empty Flags", handler.Name, group.BaseName)
				assert.NotNil(t, handler.Emit, "handler %q in %s has nil Emit", handler.Name, group.BaseName)
			})
		}
	}
}

func TestVectormathsHandlerCounts(t *testing.T) {
	groups := FileGroups()

	expectedCounts := map[string]int{
		"asm_dot_f32":       3,
		"asm_euclid_sq_f32": 3,
		"asm_normalise_f32": 3,
	}

	for _, group := range groups {
		expected, exists := expectedCounts[group.BaseName]
		require.True(t, exists, "unexpected group %s", group.BaseName)
		assert.Equal(t, expected, len(group.Handlers), "handler count mismatch for %s", group.BaseName)
	}
}

func TestVectormathsDotProductArchitectureRestrictions(t *testing.T) {
	groups := FileGroups()

	var dotGroup asmgen.FileGroup[VectormathsArchitecturePort]
	for _, group := range groups {
		if group.BaseName == "asm_dot_f32" {
			dotGroup = group
			break
		}
	}
	require.NotEmpty(t, dotGroup.BaseName, "dot product group not found")

	handlerMap := make(map[string]asmgen.HandlerDefinition[VectormathsArchitecturePort])
	for _, handler := range dotGroup.Handlers {
		handlerMap[handler.Name] = handler
	}

	t.Run("SSE is amd64-only", func(t *testing.T) {
		handler, exists := handlerMap["dotF32SSE"]
		require.True(t, exists, "dotF32SSE not found")
		require.Len(t, handler.Architectures, 1)
		assert.Equal(t, asmgen.ArchitectureAMD64, handler.Architectures[0])
	})

	t.Run("AVX2 is amd64-only", func(t *testing.T) {
		handler, exists := handlerMap["dotF32AVX2"]
		require.True(t, exists, "dotF32AVX2 not found")
		require.Len(t, handler.Architectures, 1)
		assert.Equal(t, asmgen.ArchitectureAMD64, handler.Architectures[0])
	})

	t.Run("NEON is arm64-only", func(t *testing.T) {
		handler, exists := handlerMap["dotF32"]
		require.True(t, exists, "dotF32 (NEON) not found")
		require.Len(t, handler.Architectures, 1)
		assert.Equal(t, asmgen.ArchitectureARM64, handler.Architectures[0])
	})
}

func TestVectormathsFrameSizes(t *testing.T) {
	groups := FileGroups()

	for _, group := range groups {
		for _, handler := range group.Handlers {
			t.Run(handler.Name, func(t *testing.T) {
				switch group.BaseName {
				case "asm_dot_f32", "asm_euclid_sq_f32":
					assert.Equal(t, "$0-52", handler.FrameSize, "expected $0-52 for %s", handler.Name)
				case "asm_normalise_f32":
					assert.Equal(t, "$0-24", handler.FrameSize, "expected $0-24 for %s", handler.Name)
				}
			})
		}
	}
}
