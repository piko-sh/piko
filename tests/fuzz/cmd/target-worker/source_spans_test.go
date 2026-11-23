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

package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

func TestSourceSectionSpanCanaries(t *testing.T) {
	for index := range uint64(6) {
		savedCase := generate.New("source-diagnostics", 7, index)
		expected := savedCase.Expected.Diagnostic
		require.Len(t, expected.Sections, 6, "missing recorded source model")
		require.NotZero(t, expected.Column, "missing recorded source model")
		require.NoError(t, checkDiagnosticPair(t.Context(), nil, &savedCase))

		original := savedCase.Files[expected.SectionPath]
		compiler := compiler_domain.NewCompilerOrchestrator(nil, nil, compiler_domain.WithOrchestratorModuleName(casefile.Module))
		compiled, err := compiler.CompileSFCBytes(t.Context(), expected.SectionPath, []byte(original))
		require.NoError(t, err)
		require.NotNil(t, compiled)
		require.Equal(t, "location-probe", compiled.TagName)

		savedCase.Files[expected.SectionPath] = strings.Replace(original, "🪶", "🪶é", 1)
		assert.Error(t, checkDiagnosticPair(t.Context(), nil, &savedCase), "undetected position shift")
		savedCase.Files[expected.SectionPath] = original

		blocks, err := sfcparser.Parse([]byte(original))
		require.NoError(t, err)
		blocks.Styles[0], blocks.Styles[1] = blocks.Styles[1], blocks.Styles[0]
		assert.Error(t, checkSectionSpans(original, blocks, expected.Sections), "undetected repeated section reorder")
		blocks.Styles[0], blocks.Styles[1] = blocks.Styles[1], blocks.Styles[0]
		blocks.Timelines[0].Content += " "
		assert.Error(t, checkSectionSpans(original, blocks, expected.Sections), "undetected span extension")

		expected.Column++
		assert.Error(t, checkDiagnosticPair(t.Context(), nil, &savedCase), "credited wrong column on correct line")
	}
}

func TestSourcePosition(t *testing.T) {
	source := "ab\nçd\n🙂x"

	assert.Equal(t, sfcparser.Location{Line: 1, Column: 1}, sourcePosition(source, 0))
	assert.Equal(t, sfcparser.Location{Line: 2, Column: 2}, sourcePosition(source, strings.Index(source, "d")))
	assert.Equal(t, sfcparser.Location{Line: 3, Column: 2}, sourcePosition(source, strings.Index(source, "x")))
}

func TestCompareLocation(t *testing.T) {
	assert.Negative(t, compareLocation(sfcparser.Location{Line: 1, Column: 9}, sfcparser.Location{Line: 2, Column: 1}))
	assert.Positive(t, compareLocation(sfcparser.Location{Line: 2, Column: 3}, sfcparser.Location{Line: 2, Column: 1}))
	assert.Zero(t, compareLocation(sfcparser.Location{Line: 2, Column: 3}, sfcparser.Location{Line: 2, Column: 3}))
}
