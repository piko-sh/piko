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

package compiler_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/compiler/compiler_dto"
	"piko.sh/piko/internal/sfcparser"
)

func TestFindUnresolvedComponentRefs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		template string
		script   string
		want     []string
	}{
		{
			name:     "a ref the template declares is clean",
			template: `<div p-ref="track"><span p-if="state.open" p-ref="thumb"></span></div>`,
			script:   "pkc.refs.track.focus(); pkc.refs?.thumb;",
		},
		{
			name:     "an undeclared ref is reported with a suggestion",
			template: `<div p-ref="track"></div>`,
			script:   "pkc.refs.trak.focus();",
			want:     []string{`pkc.refs.trak matches no element with p-ref="trak" in the component's template, so it is always undefined. Did you mean 'track'?`},
		},
		{
			name:     "slotted content does not count",
			template: `<div><slot></slot></div>`,
			script:   "pkc.refs.light.focus();",
			want:     []string{"pkc.refs.light matches no element"},
		},
		{
			name:     "assigned names count as declared",
			template: `<div></div>`,
			script:   "pkc.refs.made = document.createElement('div'); pkc.refs.made.focus();",
		},
		{
			name:     "computed names are skipped",
			template: `<div></div>`,
			script:   "pkc.refs['row' + index]; pkc.refs[name];",
		},
		{
			name:     "a bound p-ref makes every name possible",
			template: `<div :p-ref="state.name"></div>`,
			script:   "pkc.refs.anything.focus();",
		},
		{
			name:     "page context refs are not component refs",
			template: `<div></div>`,
			script:   "pk.refs.anything.focus();",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			tree, err := ast_domain.ParseAndTransform(t.Context(), testCase.template, "card.pkc")
			require.NoError(t, err)

			diagnostics := findUnresolvedComponentRefs(testCase.script, sfcparser.Location{Line: 4, Column: 1}, tree, "card.pkc")

			require.Len(t, diagnostics, len(testCase.want))
			for index, want := range testCase.want {
				assert.Contains(t, diagnostics[index].Message, want)
				assert.Equal(t, annotator_dto.CodeUnresolvedRef, diagnostics[index].Code)
				assert.Equal(t, compiler_dto.DiagnosticSeverityWarning, diagnostics[index].Severity)
				assert.Equal(t, "card.pkc", diagnostics[index].SourceIdentifier)
			}
		})
	}
}

func TestFindUnresolvedComponentRefs_PlacesTheRefInTheSourceFile(t *testing.T) {
	t.Parallel()

	tree, err := ast_domain.ParseAndTransform(t.Context(), `<div></div>`, "card.pkc")
	require.NoError(t, err)

	diagnostics := findUnresolvedComponentRefs("x;\n    pkc.refs.gone;", sfcparser.Location{Line: 7, Column: 9}, tree, "card.pkc")

	require.Len(t, diagnostics, 1)
	assert.Equal(t, 8, diagnostics[0].Line)
	assert.Equal(t, 14, diagnostics[0].Column)
}

func TestCompileSFC_CarriesUnresolvedRefWarnings(t *testing.T) {
	t.Parallel()

	source := `<template name="ref-card">
  <div p-ref="surface"></div>
</template>

<script lang="ts">
pkc.onConnected(() => {
  pkc.refs.surfce.focus();
});
</script>
`
	artefact, err := NewSFCCompiler("testmodule", nil).CompileSFC(t.Context(), "ref-card.pkc", []byte(source))
	require.NoError(t, err)

	require.Len(t, artefact.Diagnostics, 1)
	assert.Equal(t, annotator_dto.CodeUnresolvedRef, artefact.Diagnostics[0].Code)
	assert.Contains(t, artefact.Diagnostics[0].Message, "Did you mean 'surface'?")
	assert.Equal(t, 7, artefact.Diagnostics[0].Line)
	assert.Equal(t, 12, artefact.Diagnostics[0].Column)
}
