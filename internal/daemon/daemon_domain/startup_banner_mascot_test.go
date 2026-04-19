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

package daemon_domain

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMascotPixelData(t *testing.T) {
	t.Run("Dimensions", func(t *testing.T) {
		t.Parallel()

		assert.Len(t, mascotPixels, 37, "expected 37 rows")
		assert.Len(t, mascotPixels[0], 50, "expected 50 columns")
	})

	t.Run("AlphaValuesAreOnlyZeroOrFull", func(t *testing.T) {
		t.Parallel()

		for y := range mascotPixels {
			for x := range mascotPixels[y] {
				a := mascotPixels[y][x][3]
				assert.Truef(t, a == 0 || a == 255,
					"pixel (%d,%d) has unexpected alpha %d; expected 0 or 255", x, y, a)
			}
		}
	})
}

func TestMascotSmallLines(t *testing.T) {
	t.Run("LineCount", func(t *testing.T) {
		t.Parallel()

		lines := mascotSmallLines()
		assert.Len(t, lines, 6, "expected 6 lines")
	})

	t.Run("ContainsMascotArt", func(t *testing.T) {
		t.Parallel()

		lines := mascotSmallLines()
		joined := strings.Join(lines, "\n")
		stripped := stripANSI(joined)

		for _, want := range []string{"▄██▄▄██▄", "(●)(●)", "╰────────╯"} {
			assert.Containsf(t, stripped, want, "expected %q in small mascot output", want)
		}
	})
}

func TestMascotLargeLines(t *testing.T) {
	t.Run("LineCount", func(t *testing.T) {
		t.Parallel()

		lines := mascotLargeLines()

		assert.Len(t, lines, 19, "expected 19 lines (ceil(37/2))")
	})

	t.Run("ContainsHalfBlockCharacters", func(t *testing.T) {
		t.Parallel()

		lines := mascotLargeLines()
		joined := strings.Join(lines, "")

		assert.True(t, strings.Contains(joined, "▀") || strings.Contains(joined, "▄"),
			"expected half-block characters in large mascot output")
	})

	t.Run("ContainsANSIEscapes", func(t *testing.T) {
		t.Parallel()

		lines := mascotLargeLines()
		joined := strings.Join(lines, "")

		assert.Contains(t, joined, "\033[38;2;",
			"expected ANSI 24-bit colour escapes in large mascot output")
	})

	t.Run("ConsistentDisplayWidth", func(t *testing.T) {
		t.Parallel()

		lines := mascotLargeLines()
		for i, line := range lines {
			width := utf8.RuneCountInString(stripANSI(line))
			assert.Equalf(t, mascotPixelWidth, width,
				"line %d has display width %d, expected %d", i, width, mascotPixelWidth)
		}
	})
}

func TestCombineSideBySide(t *testing.T) {
	t.Run("EqualLength", func(t *testing.T) {
		t.Parallel()

		left := []string{"AA", "BB"}
		right := []string{"XX", "YY"}

		result := combineSideBySide(left, right, 2)

		require.Len(t, result, 2, "expected 2 lines")
		assert.Equal(t, "AA  XX", result[0], "line 0")
		assert.Equal(t, "BB  YY", result[1], "line 1")
	})

	t.Run("LeftTallerCentresRight", func(t *testing.T) {
		t.Parallel()

		left := []string{"A", "B", "C", "D", "E"}
		right := []string{"X"}

		result := combineSideBySide(left, right, 1)

		require.Len(t, result, 5, "expected 5 lines")

		assert.Contains(t, result[2], "X", "expected right content on line 2")

		for i, line := range result {
			if i != 2 {
				assert.NotContainsf(t, line, "X", "unexpected right content on line %d", i)
			}
		}
	})

	t.Run("RightTallerExtends", func(t *testing.T) {
		t.Parallel()

		left := []string{"A"}
		right := []string{"X", "Y", "Z"}

		result := combineSideBySide(left, right, 1)

		require.Len(t, result, 3, "expected 3 lines")
		assert.Contains(t, result[0], "X", "line 0 should contain X")
		assert.Contains(t, result[2], "Z", "line 2 should contain Z")
	})

	t.Run("PadsUnevenLeftWidths", func(t *testing.T) {
		t.Parallel()

		left := []string{"A", "BBBB"}
		right := []string{"X", "Y"}

		result := combineSideBySide(left, right, 1)

		assert.Equal(t, "A    X", result[0], "line 0")
		assert.Equal(t, "BBBB Y", result[1], "line 1")
	})

	t.Run("EmptyInputs", func(t *testing.T) {
		t.Parallel()

		result := combineSideBySide(nil, nil, 1)
		assert.Empty(t, result, "expected 0 lines")
	})

	t.Run("EmptyLeftNonEmptyRight", func(t *testing.T) {
		t.Parallel()

		result := combineSideBySide(nil, []string{"X", "Y"}, 1)
		require.Len(t, result, 2, "expected 2 lines")
		assert.Contains(t, result[0], "X", "line 0 should contain X")
	})

	t.Run("ANSILeftDoesNotAffectAlignment", func(t *testing.T) {
		t.Parallel()

		left := []string{"\x1b[31mAB\x1b[0m", "CCCC"}
		right := []string{"X", "Y"}

		result := combineSideBySide(left, right, 1)

		stripped0 := stripANSI(result[0])
		stripped1 := stripANSI(result[1])

		idx0 := strings.Index(stripped0, "X")
		idx1 := strings.Index(stripped1, "Y")
		assert.Equalf(t, idx0, idx1, "right column mismatch: X at %d, Y at %d", idx0, idx1)
	})

	t.Run("GapSize", func(t *testing.T) {
		t.Parallel()

		left := []string{"A"}
		right := []string{"X"}

		for _, gap := range []int{0, 1, 5} {
			result := combineSideBySide(left, right, gap)
			want := "A" + strings.Repeat(" ", gap) + "X"
			assert.Equalf(t, want, result[0], "gap %d", gap)
		}
	})
}
