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

func TestFileGroupsReturnsAllExpectedGroups(t *testing.T) {
	groups := FileGroups()

	require.Len(t, groups, 6, "expected 6 file groups")

	expected_base_names := []string{
		"asm_vm_dispatch_arith",
		"asm_vm_dispatch_cmp",
		"asm_vm_dispatch_string",
		"asm_vm_dispatch_super",
		"asm_vm_dispatch_init",
		"asm_vm_dispatch_inline",
	}

	for i, group := range groups {
		assert.Equal(t, expected_base_names[i], group.BaseName, "group %d base name mismatch", i)
	}
}

func TestAllHandlersHaveRequiredFields(t *testing.T) {
	groups := FileGroups()

	for _, group := range groups {
		for i, handler := range group.Handlers {
			t.Run(group.BaseName+"/"+handler.Name, func(t *testing.T) {
				assert.NotEmpty(t, handler.Name, "handler %d in %s has empty Name", i, group.BaseName)

				has_comment := handler.Comment != "" || handler.CommentFunction != nil
				assert.True(t, has_comment, "handler %q in %s has neither Comment nor CommentFunction", handler.Name, group.BaseName)

				assert.NotEmpty(t, handler.FrameSize, "handler %q in %s has empty FrameSize", handler.Name, group.BaseName)
				assert.NotEmpty(t, handler.Flags, "handler %q in %s has empty Flags", handler.Name, group.BaseName)
				assert.NotNil(t, handler.Emit, "handler %q in %s has nil Emit", handler.Name, group.BaseName)
			})
		}
	}
}

func TestHandlerCountPerGroup(t *testing.T) {
	groups := FileGroups()

	expected_counts := map[string]int{
		"asm_vm_dispatch_arith":  27,
		"asm_vm_dispatch_cmp":    24,
		"asm_vm_dispatch_string": 7,
		"asm_vm_dispatch_super":  11,
		"asm_vm_dispatch_init":   8,
		"asm_vm_dispatch_inline": 3,
	}

	for _, group := range groups {
		expected, exists := expected_counts[group.BaseName]
		require.True(t, exists, "unexpected group %s", group.BaseName)
		assert.Equal(t, expected, len(group.Handlers), "handler count mismatch for %s", group.BaseName)
	}
}

func TestArchitectureRestrictedHandlers(t *testing.T) {
	groups := FileGroups()

	handler_map := make(map[string]asmgen.HandlerDefinition[BytecodeArchitecturePort])
	for _, group := range groups {
		for _, handler := range group.Handlers {
			handler_map[handler.Name] = handler
		}
	}

	t.Run("initJumpTableSSE41 is amd64-only", func(t *testing.T) {
		handler, exists := handler_map["initJumpTableSSE41"]
		require.True(t, exists, "initJumpTableSSE41 not found")
		require.Len(t, handler.Architectures, 1)
		assert.Equal(t, asmgen.ArchitectureAMD64, handler.Architectures[0])
	})

	t.Run("handlerMathRound is arm64-only", func(t *testing.T) {
		handler, exists := handler_map["handlerMathRound"]
		require.True(t, exists, "handlerMathRound not found")
		require.Len(t, handler.Architectures, 1)
		assert.Equal(t, asmgen.ArchitectureARM64, handler.Architectures[0])
	})
}

func TestHeaderFilesReturnsThreeHeaders(t *testing.T) {
	headers := HeaderFiles()

	require.Len(t, headers, 3, "expected 3 header files")

	expected_names := []string{
		"asm_dispatch_offsets.h",
		"asm_dispatch_amd64.h",
		"asm_dispatch_arm64.h",
	}

	for i, header := range headers {
		assert.Equal(t, expected_names[i], header.Name, "header %d name mismatch", i)
	}
}
