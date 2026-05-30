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
)

func TestVectormathsFileGroupsReturnsAllExpectedGroups(t *testing.T) {
	groups := FileGroups()

	require.Len(t, groups, 7, "expected 7 file groups (dot f32, euclid, normalise, sum, add, dot f64, scale f64)")

	expectedBaseNames := []string{
		"asm_dot_f32",
		"asm_euclid_sq_f32",
		"asm_normalise_f32",
		"asm_sum_f64",
		"asm_add_f64",
		"asm_dot_f64",
		"asm_scale_f64",
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
		"asm_sum_f64":       3,
		"asm_add_f64":       3,
		"asm_dot_f64":       3,
		"asm_scale_f64":     3,
	}

	for _, group := range groups {
		expected, exists := expectedCounts[group.BaseName]
		require.True(t, exists, "unexpected group %s", group.BaseName)
		assert.Equal(t, expected, len(group.Handlers), "handler count mismatch for %s", group.BaseName)
	}
}

func TestVectormathsFrameSizes(t *testing.T) {
	groups := FileGroups()

	expectedFrameSize := map[string]string{
		"asm_dot_f32":       "$0-52",
		"asm_euclid_sq_f32": "$0-52",
		"asm_normalise_f32": "$0-24",
		"asm_sum_f64":       "$0-32",
		"asm_add_f64":       "$0-72",
		"asm_dot_f64":       "$0-56",
		"asm_scale_f64":     "$0-32",
	}

	for _, group := range groups {
		for _, handler := range group.Handlers {
			t.Run(handler.Name, func(t *testing.T) {
				want := expectedFrameSize[group.BaseName]
				assert.Equal(t, want, handler.FrameSize, "expected %s for %s", want, handler.Name)
			})
		}
	}
}
