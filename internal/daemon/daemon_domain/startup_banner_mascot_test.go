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
)

func TestMascotPixelData(t *testing.T) {
	t.Run("Dimensions", func(t *testing.T) {
		t.Parallel()

		if got := len(mascotPixels); got != 37 {
			t.Errorf("expected 37 rows, got %d", got)
		}
		if got := len(mascotPixels[0]); got != 50 {
			t.Errorf("expected 50 columns, got %d", got)
		}
	})

	t.Run("AlphaValuesAreOnlyZeroOrFull", func(t *testing.T) {
		t.Parallel()

		for y := range mascotPixels {
			for x := range mascotPixels[y] {
				a := mascotPixels[y][x][3]
				if a != 0 && a != 255 {
					t.Errorf("pixel (%d,%d) has unexpected alpha %d; expected 0 or 255", x, y, a)
				}
			}
		}
	})
}

func TestMascotSmallLines(t *testing.T) {
	t.Run("LineCount", func(t *testing.T) {
		t.Parallel()

		lines := mascotSmallLines()
		if len(lines) != 6 {
			t.Errorf("expected 6 lines, got %d", len(lines))
		}
	})

	t.Run("ContainsMascotArt", func(t *testing.T) {
		t.Parallel()

		lines := mascotSmallLines()
		joined := strings.Join(lines, "\n")
		stripped := stripANSI(joined)

		for _, want := range []string{"▄██▄▄██▄", "(●)(●)", "╰────────╯"} {
			if !strings.Contains(stripped, want) {
				t.Errorf("expected %q in small mascot output", want)
			}
		}
	})
}

func TestMascotLargeLines(t *testing.T) {
	t.Run("LineCount", func(t *testing.T) {
		t.Parallel()

		lines := mascotLargeLines()

		if len(lines) != 19 {
			t.Errorf("expected 19 lines (ceil(37/2)), got %d", len(lines))
		}
	})

	t.Run("ContainsHalfBlockCharacters", func(t *testing.T) {
		t.Parallel()

		lines := mascotLargeLines()
		joined := strings.Join(lines, "")

		if !strings.Contains(joined, "▀") && !strings.Contains(joined, "▄") {
			t.Error("expected half-block characters in large mascot output")
		}
	})

	t.Run("ContainsANSIEscapes", func(t *testing.T) {
		t.Parallel()

		lines := mascotLargeLines()
		joined := strings.Join(lines, "")

		if !strings.Contains(joined, "\033[38;2;") {
			t.Error("expected ANSI 24-bit colour escapes in large mascot output")
		}
	})

	t.Run("ConsistentDisplayWidth", func(t *testing.T) {
		t.Parallel()

		lines := mascotLargeLines()
		for i, line := range lines {
			width := utf8.RuneCountInString(stripANSI(line))
			if width != mascotPixelWidth {
				t.Errorf("line %d has display width %d, expected %d", i, width, mascotPixelWidth)
			}
		}
	})
}

func TestCombineSideBySide(t *testing.T) {
	t.Run("EqualLength", func(t *testing.T) {
		t.Parallel()

		left := []string{"AA", "BB"}
		right := []string{"XX", "YY"}

		result := combineSideBySide(left, right, 2)

		if len(result) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(result))
		}
		if result[0] != "AA  XX" {
			t.Errorf("line 0: got %q, want %q", result[0], "AA  XX")
		}
		if result[1] != "BB  YY" {
			t.Errorf("line 1: got %q, want %q", result[1], "BB  YY")
		}
	})

	t.Run("LeftTallerCentresRight", func(t *testing.T) {
		t.Parallel()

		left := []string{"A", "B", "C", "D", "E"}
		right := []string{"X"}

		result := combineSideBySide(left, right, 1)

		if len(result) != 5 {
			t.Fatalf("expected 5 lines, got %d", len(result))
		}

		if !strings.Contains(result[2], "X") {
			t.Errorf("expected right content on line 2, got %q", result[2])
		}

		for i, line := range result {
			if i != 2 && strings.Contains(line, "X") {
				t.Errorf("unexpected right content on line %d: %q", i, line)
			}
		}
	})

	t.Run("RightTallerExtends", func(t *testing.T) {
		t.Parallel()

		left := []string{"A"}
		right := []string{"X", "Y", "Z"}

		result := combineSideBySide(left, right, 1)

		if len(result) != 3 {
			t.Fatalf("expected 3 lines, got %d", len(result))
		}
		if !strings.Contains(result[0], "X") {
			t.Errorf("line 0 should contain X, got %q", result[0])
		}
		if !strings.Contains(result[2], "Z") {
			t.Errorf("line 2 should contain Z, got %q", result[2])
		}
	})

	t.Run("PadsUnevenLeftWidths", func(t *testing.T) {
		t.Parallel()

		left := []string{"A", "BBBB"}
		right := []string{"X", "Y"}

		result := combineSideBySide(left, right, 1)

		if result[0] != "A    X" {
			t.Errorf("line 0: got %q, want %q", result[0], "A    X")
		}
		if result[1] != "BBBB Y" {
			t.Errorf("line 1: got %q, want %q", result[1], "BBBB Y")
		}
	})

	t.Run("EmptyInputs", func(t *testing.T) {
		t.Parallel()

		result := combineSideBySide(nil, nil, 1)
		if len(result) != 0 {
			t.Errorf("expected 0 lines, got %d", len(result))
		}
	})

	t.Run("EmptyLeftNonEmptyRight", func(t *testing.T) {
		t.Parallel()

		result := combineSideBySide(nil, []string{"X", "Y"}, 1)
		if len(result) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(result))
		}
		if !strings.Contains(result[0], "X") {
			t.Errorf("line 0 should contain X, got %q", result[0])
		}
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
		if idx0 != idx1 {
			t.Errorf("right column mismatch: X at %d, Y at %d", idx0, idx1)
		}
	})

	t.Run("GapSize", func(t *testing.T) {
		t.Parallel()

		left := []string{"A"}
		right := []string{"X"}

		for _, gap := range []int{0, 1, 5} {
			result := combineSideBySide(left, right, gap)
			want := "A" + strings.Repeat(" ", gap) + "X"
			if result[0] != want {
				t.Errorf("gap %d: got %q, want %q", gap, result[0], want)
			}
		}
	})
}
