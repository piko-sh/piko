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

package generate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestDiagnosticPairPointsAtTheBrokenExpression(t *testing.T) {
	ids := map[string]bool{}
	for index := range uint64(8) {
		c, err := Build("source-diagnostics", 7, index, "")
		require.NoError(t, err)
		diagnostic := c.Expected.Diagnostic
		require.NotNil(t, diagnostic, "index %d", index)
		ids[diagnostic.ID] = true
		assert.Equal(t, "valid/"+diagnostic.ID+".pk", diagnostic.GoodPath)
		assert.Equal(t, "invalid/"+diagnostic.ID+".pk", diagnostic.BadPath)
		assert.Contains(t, []string{"T004", "T005", "T010"}, diagnostic.Code)
		require.NotEqual(t, c.Files[diagnostic.GoodPath], c.Files[diagnostic.BadPath])

		lines := strings.Split(c.Files[diagnostic.BadPath], "\n")
		require.Greater(t, len(lines), diagnostic.Line-1, "index %d", index)
		line := []rune(lines[diagnostic.Line-1])
		require.Greater(t, len(line), diagnostic.Column-1, "index %d", index)
		remainder := string(line[diagnostic.Column-1:])
		if diagnostic.ID == "member" {
			assert.True(t, strings.HasPrefix(remainder, `."`), "index %d: %q", index, remainder)
			assert.Empty(t, diagnostic.Expression)
		} else {
			assert.Zero(t, diagnostic.ExpressionOffset)
			assert.Contains(t, remainder, diagnostic.Expression+"'")
		}

		requireSectionOffsets(t, c.Files[diagnostic.SectionPath], diagnostic.Sections)
		matrices := Matrices(&c)
		require.Len(t, matrices, 1)
		assert.Equal(t, []string{diagnostic.ID}, matrices[0].Cells)
		assert.Equal(t, diagnosticCells, matrices[0].Feasible)
	}
	assert.Len(t, ids, 4)
}

func TestSectionSourceRotatesBlocksWithExactOffsets(t *testing.T) {
	first, source := sectionSource(0, 0)
	requireSectionOffsets(t, source, first)
	require.Len(t, first, 6)
	assert.Equal(t, "template", first[0].Tag)
	assert.Equal(t, map[string]string{"name": "location-probe"}, first[0].Attributes)

	rotated, rotatedSource := sectionSource(1, 2)
	requireSectionOffsets(t, rotatedSource, rotated)
	assert.Equal(t, "style", rotated[0].Tag)
	assert.Equal(t, "template", rotated[len(rotated)-1].Tag)
	assert.True(t, strings.HasPrefix(rotatedSource, "\n\n\n<!-- "))
}

func requireSectionOffsets(t *testing.T, source string, sections []casefile.SourceSection) {
	t.Helper()
	require.NotEmpty(t, sections)
	previous := 0
	for _, section := range sections {
		require.GreaterOrEqual(t, section.Start, previous)
		require.Less(t, section.Start, section.ContentStart)
		require.LessOrEqual(t, section.ContentStart, section.ContentEnd)
		assert.True(t, strings.HasPrefix(source[section.Start:section.ContentStart], "<"+section.Tag), section.Tag)
		assert.True(t, strings.HasPrefix(source[section.ContentEnd:], "</"+section.Tag+">"), section.Tag)
		previous = section.ContentEnd
	}
}
