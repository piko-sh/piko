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
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/asmgen"

	interp_amd64 "piko.sh/piko/internal/interp/interp_domain/asm/asmgen_arch_amd64"
	interp_arm64 "piko.sh/piko/internal/interp/interp_domain/asm/asmgen_arch_arm64"
)

func TestFileGroupsReturnsAllExpectedGroups(t *testing.T) {
	groups := FileGroups()

	require.Len(t, groups, 23, "expected 23 file groups")

	expected_base_names := []string{
		"asm_vm_dispatch_arith",
		"asm_vm_dispatch_cmp",
		"asm_vm_dispatch_string",
		"asm_vm_dispatch_super",
		"asm_vm_dispatch_init",
		"asm_vm_dispatch_inline",
		"asm_vm_dispatch_direct_exits",
		"asm_vm_dispatch_tier1_slice_typed",
		"asm_vm_dispatch_tier1_super_range_check",
		"asm_vm_dispatch_tier1_complex",
		"asm_vm_dispatch_tier1_math",
		"asm_vm_dispatch_tier1_strconv",
		"asm_vm_dispatch_tier1_runtime",
		"asm_vm_dispatch_tier1_move",
		"asm_vm_dispatch_tier1_struct_field_incdec",
		"asm_vm_dispatch_tier1_unary",
		"asm_vm_dispatch_tier1_conversion",
		"asm_vm_dispatch_tier1_math_unary",
		"asm_vm_dispatch_tier1_string",
		"asm_vm_dispatch_tier2_inplace",
		"asm_vm_dispatch_tier2_lift",
		"asm_vm_dispatch_flat_install",
		"asm_vm_dispatch_truncate",
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
		"asm_vm_dispatch_arith":                     30,
		"asm_vm_dispatch_cmp":                       21,
		"asm_vm_dispatch_string":                    6,
		"asm_vm_dispatch_super":                     11,
		"asm_vm_dispatch_init":                      10,
		"asm_vm_dispatch_inline":                    8,
		"asm_vm_dispatch_direct_exits":              0,
		"asm_vm_dispatch_tier1_slice_typed":         35,
		"asm_vm_dispatch_tier1_super_range_check":   1,
		"asm_vm_dispatch_tier1_complex":             4,
		"asm_vm_dispatch_tier1_math":                5,
		"asm_vm_dispatch_tier1_strconv":             3,
		"asm_vm_dispatch_tier1_runtime":             10,
		"asm_vm_dispatch_tier1_move":                5,
		"asm_vm_dispatch_tier1_struct_field_incdec": 4,
		"asm_vm_dispatch_tier2_inplace":             4,
		"asm_vm_dispatch_tier1_unary":               3,
		"asm_vm_dispatch_tier1_conversion":          4,
		"asm_vm_dispatch_tier1_math_unary":          6,
		"asm_vm_dispatch_tier1_string":              1,
		"asm_vm_dispatch_tier2_lift":                0,
		"asm_vm_dispatch_flat_install":              1,
		"asm_vm_dispatch_truncate":                  1,
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

}

func TestInlineGoCallRealHandlersHaveNoLocalPointers(t *testing.T) {
	architectures := []BytecodeArchitecturePort{
		interp_amd64.New(),
		interp_arm64.New(),
	}
	groups := FileGroups()
	writer := &memWriter{files: make(map[string][]byte)}
	err := asmgen.GenerateFiles(writer, architectures, groups, nil, nil)
	require.NoError(t, err)

	affected_filenames := map[string]bool{
		"asm_vm_dispatch_tier1_runtime_amd64.s": true,
		"asm_vm_dispatch_tier1_runtime_arm64.s": true,
		"asm_vm_dispatch_tier1_math_amd64.s":    true,
		"asm_vm_dispatch_tier1_math_arm64.s":    true,
		"asm_vm_dispatch_tier1_strconv_amd64.s": true,
		"asm_vm_dispatch_tier1_strconv_arm64.s": true,
	}

	real_text_pattern := regexp.MustCompile(`(?m)^TEXT ·(handler\w+)\(SB\)`)

	for path, data := range writer.files {
		basename := path[strings.LastIndex(path, "/")+1:]
		if !affected_filenames[basename] {
			continue
		}

		content := string(data)
		real_handler_matches := real_text_pattern.FindAllStringSubmatch(content, -1)
		require.NotEmpty(t, real_handler_matches,
			"%s should contain at least one handler", path)

		for _, match := range real_handler_matches {
			handler_name := match[1]

			handler_start := strings.Index(content, match[0])
			require.NotEqual(t, -1, handler_start, "handler %s start not found", handler_name)

			search_from := handler_start + len(match[0])
			next_text := strings.Index(content[search_from:], "\nTEXT ")
			var body string
			if next_text == -1 {
				body = content[search_from:]
			} else {
				body = content[search_from : search_from+next_text]
			}

			if !strings.Contains(body, "CALL") && !strings.Contains(body, " BL ") && !strings.Contains(body, "\tBL\t") {
				continue
			}

			assert.Contains(t, body, "NO_LOCAL_POINTERS",
				"%s: handler %s lacks NO_LOCAL_POINTERS directive - "+
					"runtime will panic 'missing stackmap' on GC scan",
				path, handler_name)
		}
	}
}

func TestInlineGoCallRealHandlersHaveNoSpilledPointers(t *testing.T) {
	architectures := []BytecodeArchitecturePort{
		interp_amd64.New(),
		interp_arm64.New(),
	}
	groups := FileGroups()
	writer := &memWriter{files: make(map[string][]byte)}
	err := asmgen.GenerateFiles(writer, architectures, groups, nil, nil)
	require.NoError(t, err)

	affected_filenames := map[string]bool{
		"asm_vm_dispatch_tier1_runtime_amd64.s": true,
		"asm_vm_dispatch_tier1_runtime_arm64.s": true,
		"asm_vm_dispatch_tier1_math_amd64.s":    true,
		"asm_vm_dispatch_tier1_math_arm64.s":    true,
		"asm_vm_dispatch_tier1_strconv_amd64.s": true,
		"asm_vm_dispatch_tier1_strconv_arm64.s": true,
	}

	real_text_pattern := regexp.MustCompile(`(?m)^TEXT ·(handler\w+)\(SB\)`)

	forbidden_amd64 := regexp.MustCompile(`MOVQ\s+R15,\s+(\d+)\(SP\)`)
	forbidden_arm64 := regexp.MustCompile(`MOVD\s+R19,\s+(\d+)\(RSP\)`)

	for path, data := range writer.files {
		basename := path[strings.LastIndex(path, "/")+1:]
		if !affected_filenames[basename] {
			continue
		}

		is_arm64 := strings.HasSuffix(basename, "_arm64.s")
		pattern := forbidden_amd64
		register := "R15"
		stack_pointer := "SP"
		allowed_offset := "0"
		if is_arm64 {
			pattern = forbidden_arm64
			register = "R19"
			stack_pointer = "RSP"
			allowed_offset = "8"
		}

		content := string(data)
		real_handler_matches := real_text_pattern.FindAllStringSubmatch(content, -1)
		require.NotEmpty(t, real_handler_matches, "%s should contain handlers", path)

		for _, match := range real_handler_matches {
			handler_name := match[1]
			handler_start := strings.Index(content, match[0])
			search_from := handler_start + len(match[0])
			next_text := strings.Index(content[search_from:], "\nTEXT ")
			var body string
			if next_text == -1 {
				body = content[search_from:]
			} else {
				body = content[search_from : search_from+next_text]
			}

			for _, spill_match := range pattern.FindAllStringSubmatch(body, -1) {
				offset := spill_match[1]
				if offset == allowed_offset {

					continue
				}
				t.Errorf("%s: handler %s spills %s to %s+%s - re-introduces "+
					"the pointer-in-locals problem the no-spill restructure "+
					"eliminated. Recover ctx from the abi0 return slot instead.",
					path, handler_name, register, stack_pointer, offset)
			}
		}
	}
}

func TestAffectedFilesIncludeFuncdataHeader(t *testing.T) {
	architectures := []BytecodeArchitecturePort{
		interp_amd64.New(),
		interp_arm64.New(),
	}
	groups := FileGroups()
	writer := &memWriter{files: make(map[string][]byte)}
	err := asmgen.GenerateFiles(writer, architectures, groups, nil, nil)
	require.NoError(t, err)

	affected_filenames := map[string]bool{
		"asm_vm_dispatch_tier1_runtime_amd64.s": true,
		"asm_vm_dispatch_tier1_runtime_arm64.s": true,
		"asm_vm_dispatch_tier1_math_amd64.s":    true,
		"asm_vm_dispatch_tier1_math_arm64.s":    true,
		"asm_vm_dispatch_tier1_strconv_amd64.s": true,
		"asm_vm_dispatch_tier1_strconv_arm64.s": true,
	}

	for path, data := range writer.files {
		basename := path[strings.LastIndex(path, "/")+1:]
		if !affected_filenames[basename] {
			continue
		}
		assert.Contains(t, string(data), `#include "funcdata.h"`,
			"%s must include funcdata.h for NO_LOCAL_POINTERS macro", path)
	}
}

func TestHeaderFilesReturnsThreeHeaders(t *testing.T) {
	headers := HeaderFiles(testOffsetsForHeaderFiles())

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
