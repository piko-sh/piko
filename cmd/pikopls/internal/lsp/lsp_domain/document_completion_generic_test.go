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

package lsp_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestMethodCallSnippetSuffix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		signature inspector_dto.FunctionSignature
		expected  string
	}{
		{
			name: "non-generic method takes the plain call snippet",
			signature: inspector_dto.FunctionSignature{
				Params:  []string{"string"},
				Results: []string{"error"},
			},
			expected: "($1)$0",
		},
		{
			name: "type parameter inferable from a parameter needs no explicit arguments",
			signature: inspector_dto.FunctionSignature{
				Params:         []string{"K", "V"},
				Results:        []string{"map[K]V"},
				TypeParamNames: []string{"K", "V"},
			},
			expected: "($1)$0",
		},
		{
			name: "type parameter appearing only in the results must be supplied explicitly",
			signature: inspector_dto.FunctionSignature{
				Params:         []string{"string"},
				Results:        []string{"T"},
				TypeParamNames: []string{"T"},
			},
			expected: "[$1]($2)$0",
		},
		{
			name: "type parameter nested inside a parameter type is still inferable",
			signature: inspector_dto.FunctionSignature{
				Params:         []string{"[]T", "func(T) U"},
				Results:        []string{"[]U"},
				TypeParamNames: []string{"T", "U"},
			},
			expected: "($1)$0",
		},
		{
			name: "a name that only occurs inside a longer identifier is not a match",
			signature: inspector_dto.FunctionSignature{
				Params:         []string{"time.Time", "Things"},
				Results:        []string{"T"},
				TypeParamNames: []string{"T"},
			},
			expected: "[$1]($2)$0",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, methodCallSnippetSuffix(testCase.signature))
		})
	}
}

func TestTypeParamAppearsIn(t *testing.T) {
	t.Parallel()

	assert.True(t, typeParamAppearsIn("T", []string{"[]T"}))
	assert.True(t, typeParamAppearsIn("T", []string{"map[string]T"}))
	assert.True(t, typeParamAppearsIn("T", []string{"func(T) error"}))
	assert.True(t, typeParamAppearsIn("T", []string{"*T"}))
	assert.False(t, typeParamAppearsIn("T", []string{"time.Time"}))
	assert.False(t, typeParamAppearsIn("T", []string{"Things"}))
	assert.False(t, typeParamAppearsIn("T", []string{"myT"}))
	assert.False(t, typeParamAppearsIn("", []string{"T"}))
}
