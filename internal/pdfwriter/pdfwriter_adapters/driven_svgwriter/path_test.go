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

package driven_svgwriter

import (
	"math"
	"testing"
)

func approxEqual(a, b, tol float64) bool {
	return math.Abs(a-b) < tol
}

const defaultTol = 1e-9

func TestParsePathData_MoveTo_Absolute(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].Type != 'M' {
		t.Errorf("expected type M, got %c", cmds[0].Type)
	}
	if !approxEqual(cmds[0].Args[0], 10, defaultTol) || !approxEqual(cmds[0].Args[1], 20, defaultTol) {
		t.Errorf("expected (10,20), got (%v,%v)", cmds[0].Args[0], cmds[0].Args[1])
	}
}

func TestParsePathData_MoveTo_Relative(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("m10 20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].Type != 'M' {
		t.Errorf("expected type M, got %c", cmds[0].Type)
	}
	if !approxEqual(cmds[0].Args[0], 10, defaultTol) || !approxEqual(cmds[0].Args[1], 20, defaultTol) {
		t.Errorf("expected (10,20), got (%v,%v)", cmds[0].Args[0], cmds[0].Args[1])
	}
}

func TestParsePathData_ImplicitLineTo_AfterMoveTo(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 10 20 20 30 30")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 3 {
		t.Fatalf("expected 3 commands (1 M + 2 L), got %d", len(cmds))
	}
	if cmds[0].Type != 'M' {
		t.Errorf("cmds[0]: expected M, got %c", cmds[0].Type)
	}
	if cmds[1].Type != 'L' {
		t.Errorf("cmds[1]: expected L (implicit lineTo), got %c", cmds[1].Type)
	}
	if cmds[2].Type != 'L' {
		t.Errorf("cmds[2]: expected L (implicit lineTo), got %c", cmds[2].Type)
	}
	if !approxEqual(cmds[2].Args[0], 30, defaultTol) || !approxEqual(cmds[2].Args[1], 30, defaultTol) {
		t.Errorf("cmds[2]: expected (30,30), got (%v,%v)", cmds[2].Args[0], cmds[2].Args[1])
	}
}

func TestParsePathData_LineTo_Absolute(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 L50 60")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if cmds[1].Type != 'L' {
		t.Errorf("expected type L, got %c", cmds[1].Type)
	}
	if !approxEqual(cmds[1].Args[0], 50, defaultTol) || !approxEqual(cmds[1].Args[1], 60, defaultTol) {
		t.Errorf("expected (50,60), got (%v,%v)", cmds[1].Args[0], cmds[1].Args[1])
	}
}

func TestParsePathData_LineTo_Relative(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 10 l5 5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if cmds[1].Type != 'L' {
		t.Errorf("expected type L, got %c", cmds[1].Type)
	}
	if !approxEqual(cmds[1].Args[0], 15, defaultTol) || !approxEqual(cmds[1].Args[1], 15, defaultTol) {
		t.Errorf("expected (15,15), got (%v,%v)", cmds[1].Args[0], cmds[1].Args[1])
	}
}

func TestParsePathData_HorizontalLineTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		path  string
		wantX float64
		wantY float64
	}{
		{
			name:  "absolute H",
			path:  "M10 20 H50",
			wantX: 50,
			wantY: 20,
		},
		{
			name:  "relative h",
			path:  "M10 20 h30",
			wantX: 40,
			wantY: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmds, err := ParsePathData(tt.path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cmds) != 2 {
				t.Fatalf("expected 2 commands, got %d", len(cmds))
			}

			if cmds[1].Type != 'L' {
				t.Errorf("expected type L, got %c", cmds[1].Type)
			}
			if !approxEqual(cmds[1].Args[0], tt.wantX, defaultTol) || !approxEqual(cmds[1].Args[1], tt.wantY, defaultTol) {
				t.Errorf("expected (%v,%v), got (%v,%v)", tt.wantX, tt.wantY, cmds[1].Args[0], cmds[1].Args[1])
			}
		})
	}
}

func TestParsePathData_VerticalLineTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		path  string
		wantX float64
		wantY float64
	}{
		{
			name:  "absolute V",
			path:  "M10 20 V80",
			wantX: 10,
			wantY: 80,
		},
		{
			name:  "relative v",
			path:  "M10 20 v30",
			wantX: 10,
			wantY: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmds, err := ParsePathData(tt.path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cmds) != 2 {
				t.Fatalf("expected 2 commands, got %d", len(cmds))
			}
			if cmds[1].Type != 'L' {
				t.Errorf("expected type L, got %c", cmds[1].Type)
			}
			if !approxEqual(cmds[1].Args[0], tt.wantX, defaultTol) || !approxEqual(cmds[1].Args[1], tt.wantY, defaultTol) {
				t.Errorf("expected (%v,%v), got (%v,%v)", tt.wantX, tt.wantY, cmds[1].Args[0], cmds[1].Args[1])
			}
		})
	}
}

func TestParsePathData_CubicBezier_Absolute(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 C10 20 30 40 50 60")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if cmds[1].Type != 'C' {
		t.Errorf("expected type C, got %c", cmds[1].Type)
	}

	wantArgs := []float64{10, 20, 30, 40, 50, 60}
	for i, want := range wantArgs {
		if !approxEqual(cmds[1].Args[i], want, defaultTol) {
			t.Errorf("args[%d]: expected %v, got %v", i, want, cmds[1].Args[i])
		}
	}
}

func TestParsePathData_CubicBezier_Relative(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 10 c5 5 10 10 15 15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if cmds[1].Type != 'C' {
		t.Errorf("expected type C, got %c", cmds[1].Type)
	}

	wantArgs := []float64{15, 15, 20, 20, 25, 25}
	for i, want := range wantArgs {
		if !approxEqual(cmds[1].Args[i], want, defaultTol) {
			t.Errorf("args[%d]: expected %v, got %v", i, want, cmds[1].Args[i])
		}
	}
}

func TestParsePathData_SmoothCubic_Absolute(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 C10 20 30 40 50 50 S70 80 90 90")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(cmds))
	}

	if cmds[2].Type != 'C' {
		t.Errorf("expected type C, got %c", cmds[2].Type)
	}
	if !approxEqual(cmds[2].Args[0], 70, defaultTol) || !approxEqual(cmds[2].Args[1], 60, defaultTol) {
		t.Errorf("reflected cp1: expected (70,60), got (%v,%v)", cmds[2].Args[0], cmds[2].Args[1])
	}

	if !approxEqual(cmds[2].Args[2], 70, defaultTol) || !approxEqual(cmds[2].Args[3], 80, defaultTol) {
		t.Errorf("cp2: expected (70,80), got (%v,%v)", cmds[2].Args[2], cmds[2].Args[3])
	}
	if !approxEqual(cmds[2].Args[4], 90, defaultTol) || !approxEqual(cmds[2].Args[5], 90, defaultTol) {
		t.Errorf("end: expected (90,90), got (%v,%v)", cmds[2].Args[4], cmds[2].Args[5])
	}
}

func TestParsePathData_SmoothCubic_WithoutPriorCubic(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 10 S30 40 50 50")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}

	if !approxEqual(cmds[1].Args[0], 10, defaultTol) || !approxEqual(cmds[1].Args[1], 10, defaultTol) {
		t.Errorf("reflected cp1: expected (10,10), got (%v,%v)", cmds[1].Args[0], cmds[1].Args[1])
	}
}

func TestParsePathData_SmoothCubic_Relative(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 C10 20 30 40 50 50 s20 30 40 40")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(cmds))
	}

	if !approxEqual(cmds[2].Args[0], 70, defaultTol) || !approxEqual(cmds[2].Args[1], 60, defaultTol) {
		t.Errorf("reflected cp1: expected (70,60), got (%v,%v)", cmds[2].Args[0], cmds[2].Args[1])
	}
	if !approxEqual(cmds[2].Args[4], 90, defaultTol) || !approxEqual(cmds[2].Args[5], 90, defaultTol) {
		t.Errorf("end: expected (90,90), got (%v,%v)", cmds[2].Args[4], cmds[2].Args[5])
	}
}

func TestParsePathData_QuadraticBezier_Absolute(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 Q50 100 100 0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if cmds[1].Type != 'C' {
		t.Errorf("expected type C (quadratic promoted to cubic), got %c", cmds[1].Type)
	}

	if !approxEqual(cmds[1].Args[4], 100, defaultTol) || !approxEqual(cmds[1].Args[5], 0, defaultTol) {
		t.Errorf("end: expected (100,0), got (%v,%v)", cmds[1].Args[4], cmds[1].Args[5])
	}

	wantCP1X := 2.0 / 3.0 * 50
	wantCP1Y := 2.0 / 3.0 * 100
	if !approxEqual(cmds[1].Args[0], wantCP1X, 0.01) || !approxEqual(cmds[1].Args[1], wantCP1Y, 0.01) {
		t.Errorf("cp1: expected (~%.2f,~%.2f), got (%v,%v)", wantCP1X, wantCP1Y, cmds[1].Args[0], cmds[1].Args[1])
	}
}

func TestParsePathData_QuadraticBezier_Relative(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 10 q25 50 50 0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if cmds[1].Type != 'C' {
		t.Errorf("expected type C, got %c", cmds[1].Type)
	}

	if !approxEqual(cmds[1].Args[4], 60, defaultTol) || !approxEqual(cmds[1].Args[5], 10, defaultTol) {
		t.Errorf("end: expected (60,10), got (%v,%v)", cmds[1].Args[4], cmds[1].Args[5])
	}
}

func TestParsePathData_SmoothQuadratic_Absolute(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 Q50 100 100 0 T200 0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(cmds))
	}

	if cmds[2].Type != 'C' {
		t.Errorf("expected type C, got %c", cmds[2].Type)
	}

	if !approxEqual(cmds[2].Args[4], 200, defaultTol) || !approxEqual(cmds[2].Args[5], 0, defaultTol) {
		t.Errorf("end: expected (200,0), got (%v,%v)", cmds[2].Args[4], cmds[2].Args[5])
	}
}

func TestParsePathData_SmoothQuadratic_WithoutPriorQuad(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 10 T50 50")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if cmds[1].Type != 'C' {
		t.Errorf("expected type C, got %c", cmds[1].Type)
	}

	if !approxEqual(cmds[1].Args[4], 50, defaultTol) || !approxEqual(cmds[1].Args[5], 50, defaultTol) {
		t.Errorf("end: expected (50,50), got (%v,%v)", cmds[1].Args[4], cmds[1].Args[5])
	}
}

func TestParsePathData_SmoothQuadratic_Relative(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 Q50 100 100 0 t100 0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(cmds))
	}

	if !approxEqual(cmds[2].Args[4], 200, defaultTol) || !approxEqual(cmds[2].Args[5], 0, defaultTol) {
		t.Errorf("end: expected (200,0), got (%v,%v)", cmds[2].Args[4], cmds[2].Args[5])
	}
}

func TestParsePathData_Arc_Absolute(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M100 0 A100 100 0 0 1 0 100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cmds) < 2 {
		t.Fatalf("expected at least 2 commands, got %d", len(cmds))
	}
	if cmds[0].Type != 'M' {
		t.Errorf("cmds[0]: expected M, got %c", cmds[0].Type)
	}

	for i := 1; i < len(cmds); i++ {
		if cmds[i].Type != 'C' {
			t.Errorf("cmds[%d]: expected C, got %c", i, cmds[i].Type)
		}
	}

	last := cmds[len(cmds)-1]
	if !approxEqual(last.Args[4], 0, 0.01) || !approxEqual(last.Args[5], 100, 0.01) {
		t.Errorf("final endpoint: expected (~0,~100), got (%v,%v)", last.Args[4], last.Args[5])
	}
}

func TestParsePathData_Arc_Relative(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M100 0 a100 100 0 0 1 -100 100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) < 2 {
		t.Fatalf("expected at least 2 commands, got %d", len(cmds))
	}
	last := cmds[len(cmds)-1]
	if !approxEqual(last.Args[4], 0, 0.01) || !approxEqual(last.Args[5], 100, 0.01) {
		t.Errorf("final endpoint: expected (~0,~100), got (%v,%v)", last.Args[4], last.Args[5])
	}
}

func TestParsePathData_Arc_HalfCircle(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 A50 50 0 0 1 100 0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) < 2 {
		t.Fatalf("expected at least 2 commands, got %d", len(cmds))
	}
	last := cmds[len(cmds)-1]
	if !approxEqual(last.Args[4], 100, 0.01) || !approxEqual(last.Args[5], 0, 0.01) {
		t.Errorf("final endpoint: expected (~100,~0), got (%v,%v)", last.Args[4], last.Args[5])
	}
}

func TestParsePathData_Arc_LargeArcFlag(t *testing.T) {
	t.Parallel()

	cmdsSmall, err := ParsePathData("M100 0 A100 100 0 0 1 0 100")
	if err != nil {
		t.Fatalf("unexpected error (small): %v", err)
	}
	cmdsLarge, err := ParsePathData("M100 0 A100 100 0 1 0 0 100")
	if err != nil {
		t.Fatalf("unexpected error (large): %v", err)
	}

	smallArcs := len(cmdsSmall) - 1
	largeArcs := len(cmdsLarge) - 1
	if largeArcs <= smallArcs {
		t.Errorf("large-arc (%d arc segments) should produce more segments than small-arc (%d arc segments)",
			largeArcs, smallArcs)
	}
}

func TestParsePathData_ClosePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{name: "uppercase Z", path: "M10 10 L50 50 Z"},
		{name: "lowercase z", path: "M10 10 L50 50 z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmds, err := ParsePathData(tt.path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cmds) != 3 {
				t.Fatalf("expected 3 commands, got %d", len(cmds))
			}
			if cmds[2].Type != 'Z' {
				t.Errorf("expected type Z, got %c", cmds[2].Type)
			}
		})
	}
}

func TestParsePathData_MultipleSubpaths(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 L50 50 Z M100 100 L150 150 Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cmds) != 6 {
		t.Fatalf("expected 6 commands, got %d", len(cmds))
	}
	if cmds[0].Type != 'M' || cmds[3].Type != 'M' {
		t.Errorf("expected M at positions 0 and 3, got %c and %c", cmds[0].Type, cmds[3].Type)
	}
	if cmds[2].Type != 'Z' || cmds[5].Type != 'Z' {
		t.Errorf("expected Z at positions 2 and 5, got %c and %c", cmds[2].Type, cmds[5].Type)
	}
}

func TestParsePathData_ExponentNotation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		path  string
		wantX float64
		wantY float64
	}{
		{
			name:  "1e2 = 100",
			path:  "M1e2 0",
			wantX: 100,
			wantY: 0,
		},
		{
			name:  "2.5E-1 = 0.25",
			path:  "M0 2.5E-1",
			wantX: 0,
			wantY: 0.25,
		},
		{
			name:  "1.5e+2 = 150",
			path:  "M1.5e+2 0",
			wantX: 150,
			wantY: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmds, err := ParsePathData(tt.path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cmds) != 1 {
				t.Fatalf("expected 1 command, got %d", len(cmds))
			}
			if !approxEqual(cmds[0].Args[0], tt.wantX, defaultTol) || !approxEqual(cmds[0].Args[1], tt.wantY, defaultTol) {
				t.Errorf("expected (%v,%v), got (%v,%v)", tt.wantX, tt.wantY, cmds[0].Args[0], cmds[0].Args[1])
			}
		})
	}
}

func TestParsePathData_EmptyPath(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmds != nil {
		t.Errorf("expected nil, got %v", cmds)
	}
}

func TestParsePathData_WhitespaceOnly(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("   ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmds != nil {
		t.Errorf("expected nil, got %v", cmds)
	}
}

func TestParsePathData_InvalidCommand(t *testing.T) {
	t.Parallel()

	_, err := ParsePathData("X10 20")
	if err == nil {
		t.Fatal("expected error for invalid command, got nil")
	}
}

func TestParsePathData_NumberWithoutCommand(t *testing.T) {
	t.Parallel()

	_, err := ParsePathData("10 20")
	if err == nil {
		t.Fatal("expected error when path starts with a number, got nil")
	}
}

func TestParsePathData_CommaSeparated(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10,20 L30,40")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if !approxEqual(cmds[1].Args[0], 30, defaultTol) || !approxEqual(cmds[1].Args[1], 40, defaultTol) {
		t.Errorf("expected (30,40), got (%v,%v)", cmds[1].Args[0], cmds[1].Args[1])
	}
}

func TestArcToCubics_CoincidentEndpoints(t *testing.T) {
	t.Parallel()

	result := ArcToCubics(10, 20, 50, 50, 0, false, true, 10, 20)
	if len(result) != 0 {
		t.Errorf("expected 0 segments for coincident endpoints, got %d", len(result))
	}
}

func TestArcToCubics_ZeroRadii(t *testing.T) {
	t.Parallel()

	result := ArcToCubics(0, 0, 0, 50, 0, false, true, 100, 0)
	if len(result) != 1 {
		t.Fatalf("expected 1 segment for zero radius, got %d", len(result))
	}
	if result[0].Type != 'C' {
		t.Errorf("expected type C, got %c", result[0].Type)
	}

	if !approxEqual(result[0].Args[4], 100, defaultTol) || !approxEqual(result[0].Args[5], 0, defaultTol) {
		t.Errorf("end: expected (100,0), got (%v,%v)", result[0].Args[4], result[0].Args[5])
	}
}

func TestArcToCubics_ZeroRy(t *testing.T) {
	t.Parallel()

	result := ArcToCubics(0, 0, 50, 0, 0, false, true, 100, 0)
	if len(result) != 1 {
		t.Fatalf("expected 1 segment for zero ry, got %d", len(result))
	}
	if result[0].Type != 'C' {
		t.Errorf("expected type C, got %c", result[0].Type)
	}
}

func TestArcToCubics_QuarterCircle(t *testing.T) {
	t.Parallel()

	result := ArcToCubics(100, 0, 100, 100, 0, false, true, 0, 100)
	if len(result) != 1 {
		t.Fatalf("expected 1 segment for quarter circle, got %d", len(result))
	}

	if !approxEqual(result[0].Args[4], 0, 0.5) || !approxEqual(result[0].Args[5], 100, 0.5) {
		t.Errorf("end: expected (~0,~100), got (%v,%v)", result[0].Args[4], result[0].Args[5])
	}
}

func TestArcToCubics_HalfCircle(t *testing.T) {
	t.Parallel()

	result := ArcToCubics(50, 0, 50, 50, 0, false, true, -50, 0)
	if len(result) < 2 {
		t.Fatalf("expected at least 2 segments for half circle, got %d", len(result))
	}

	last := result[len(result)-1]
	if !approxEqual(last.Args[4], -50, 0.5) || !approxEqual(last.Args[5], 0, 0.5) {
		t.Errorf("end: expected (~-50,~0), got (%v,%v)", last.Args[4], last.Args[5])
	}
}

func TestArcToCubics_FullCircle_LargeArc(t *testing.T) {
	t.Parallel()

	result := ArcToCubics(100, 0, 100, 100, 0, true, true, 100, 0.001)
	if len(result) < 3 {
		t.Fatalf("expected at least 3 segments for large arc, got %d", len(result))
	}
}

func TestArcToCubics_NegativeRadii(t *testing.T) {
	t.Parallel()

	resultPos := ArcToCubics(100, 0, 100, 100, 0, false, true, 0, 100)
	resultNeg := ArcToCubics(100, 0, -100, -100, 0, false, true, 0, 100)
	if len(resultPos) != len(resultNeg) {
		t.Fatalf("expected same segment count for negative radii, got %d vs %d", len(resultPos), len(resultNeg))
	}
	for i := range resultPos {
		for j := range resultPos[i].Args {
			if !approxEqual(resultPos[i].Args[j], resultNeg[i].Args[j], 0.01) {
				t.Errorf("segment %d arg %d: positive=%v, negative=%v",
					i, j, resultPos[i].Args[j], resultNeg[i].Args[j])
			}
		}
	}
}

func TestArcToCubics_WithRotation(t *testing.T) {
	t.Parallel()

	result := ArcToCubics(0, 0, 100, 50, 45, false, true, 100, 100)
	if len(result) < 1 {
		t.Fatal("expected at least 1 segment")
	}

	last := result[len(result)-1]
	if !approxEqual(last.Args[4], 100, 0.5) || !approxEqual(last.Args[5], 100, 0.5) {
		t.Errorf("end: expected (~100,~100), got (%v,%v)", last.Args[4], last.Args[5])
	}
}

func TestVecAngle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ux        float64
		uy        float64
		vx        float64
		vy        float64
		wantAngle float64
		tol       float64
	}{
		{
			name: "parallel same direction",
			ux:   1, uy: 0,
			vx: 2, vy: 0,
			wantAngle: 0,
			tol:       defaultTol,
		},
		{
			name: "perpendicular (90 degrees)",
			ux:   1, uy: 0,
			vx: 0, vy: 1,
			wantAngle: math.Pi / 2,
			tol:       defaultTol,
		},
		{
			name: "perpendicular (-90 degrees)",
			ux:   1, uy: 0,
			vx: 0, vy: -1,
			wantAngle: -math.Pi / 2,
			tol:       defaultTol,
		},
		{
			name: "opposite (180 degrees)",
			ux:   1, uy: 0,
			vx: -1, vy: 0,
			wantAngle: math.Pi,
			tol:       defaultTol,
		},
		{
			name: "45 degrees",
			ux:   1, uy: 0,
			vx: 1, vy: 1,
			wantAngle: math.Pi / 4,
			tol:       defaultTol,
		},
		{
			name: "-45 degrees",
			ux:   1, uy: 0,
			vx: 1, vy: -1,
			wantAngle: -math.Pi / 4,
			tol:       defaultTol,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := vecAngle(tt.ux, tt.uy, tt.vx, tt.vy)
			if !approxEqual(got, tt.wantAngle, tt.tol) {
				t.Errorf("vecAngle(%v,%v,%v,%v) = %v, want %v", tt.ux, tt.uy, tt.vx, tt.vy, got, tt.wantAngle)
			}
		})
	}
}

func TestMakeAbsolute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		absCmd   byte
		args     []float64
		cx       float64
		cy       float64
		wantArgs []float64
	}{
		{
			name:   "MoveTo (M)",
			absCmd: 'M',
			args:   []float64{5, 10},
			cx:     100, cy: 200,
			wantArgs: []float64{105, 210},
		},
		{
			name:   "LineTo (L)",
			absCmd: 'L',
			args:   []float64{5, 10},
			cx:     100, cy: 200,
			wantArgs: []float64{105, 210},
		},
		{
			name:   "SmoothQuad (T)",
			absCmd: 'T',
			args:   []float64{5, 10},
			cx:     100, cy: 200,
			wantArgs: []float64{105, 210},
		},
		{
			name:   "HorizontalLine (H)",
			absCmd: 'H',
			args:   []float64{5},
			cx:     100, cy: 200,
			wantArgs: []float64{105},
		},
		{
			name:   "VerticalLine (V)",
			absCmd: 'V',
			args:   []float64{5},
			cx:     100, cy: 200,
			wantArgs: []float64{205},
		},
		{
			name:   "Cubic (C)",
			absCmd: 'C',
			args:   []float64{1, 2, 3, 4, 5, 6},
			cx:     10, cy: 20,
			wantArgs: []float64{11, 22, 13, 24, 15, 26},
		},
		{
			name:   "SmoothCubic (S)",
			absCmd: 'S',
			args:   []float64{1, 2, 3, 4},
			cx:     10, cy: 20,
			wantArgs: []float64{11, 22, 13, 24},
		},
		{
			name:   "Quad (Q)",
			absCmd: 'Q',
			args:   []float64{1, 2, 3, 4},
			cx:     10, cy: 20,
			wantArgs: []float64{11, 22, 13, 24},
		},
		{
			name:   "Arc (A)",
			absCmd: 'A',
			args:   []float64{50, 50, 0, 1, 0, 10, 20},
			cx:     100, cy: 200,

			wantArgs: []float64{50, 50, 0, 1, 0, 110, 220},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			argsCopy := make([]float64, len(tt.args))
			copy(argsCopy, tt.args)
			makeAbsolute(tt.absCmd, argsCopy, tt.cx, tt.cy)

			for i, want := range tt.wantArgs {
				if !approxEqual(argsCopy[i], want, defaultTol) {
					t.Errorf("args[%d]: expected %v, got %v", i, want, argsCopy[i])
				}
			}
		})
	}
}

func TestParsePathData_Triangle(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 L100 0 L50 86.6 Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 4 {
		t.Fatalf("expected 4 commands (M L L Z), got %d", len(cmds))
	}
	if cmds[0].Type != 'M' || cmds[1].Type != 'L' || cmds[2].Type != 'L' || cmds[3].Type != 'Z' {
		t.Errorf("expected M L L Z, got %c %c %c %c", cmds[0].Type, cmds[1].Type, cmds[2].Type, cmds[3].Type)
	}
}

func TestParsePathData_MixedRelativeAbsolute(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 10 L50 50 l10 10 Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 4 {
		t.Fatalf("expected 4 commands, got %d", len(cmds))
	}
	if !approxEqual(cmds[2].Args[0], 60, defaultTol) || !approxEqual(cmds[2].Args[1], 60, defaultTol) {
		t.Errorf("expected (60,60), got (%v,%v)", cmds[2].Args[0], cmds[2].Args[1])
	}
}

func TestParsePathData_NegativeCoordinates(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M-10 -20 L-30 -40")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if !approxEqual(cmds[0].Args[0], -10, defaultTol) || !approxEqual(cmds[0].Args[1], -20, defaultTol) {
		t.Errorf("M: expected (-10,-20), got (%v,%v)", cmds[0].Args[0], cmds[0].Args[1])
	}
	if !approxEqual(cmds[1].Args[0], -30, defaultTol) || !approxEqual(cmds[1].Args[1], -40, defaultTol) {
		t.Errorf("L: expected (-30,-40), got (%v,%v)", cmds[1].Args[0], cmds[1].Args[1])
	}
}

func TestParsePathData_RepeatedLineTo(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 L10 10 20 20 30 30")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 4 {
		t.Fatalf("expected 4 commands (M + 3 L), got %d", len(cmds))
	}
	for i := 1; i <= 3; i++ {
		if cmds[i].Type != 'L' {
			t.Errorf("cmds[%d]: expected L, got %c", i, cmds[i].Type)
		}
	}
}

func TestParsePathData_DecimalWithoutLeadingDigit(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M.5 .5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if !approxEqual(cmds[0].Args[0], 0.5, defaultTol) || !approxEqual(cmds[0].Args[1], 0.5, defaultTol) {
		t.Errorf("expected (0.5,0.5), got (%v,%v)", cmds[0].Args[0], cmds[0].Args[1])
	}
}

func TestParsePathData_ArcWithScaledRadii(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 A1 1 0 0 1 100 100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) < 2 {
		t.Fatalf("expected at least 2 commands (M + cubics), got %d", len(cmds))
	}

	for i := 1; i < len(cmds); i++ {
		if cmds[i].Type != 'C' {
			t.Errorf("cmds[%d]: expected C, got %c", i, cmds[i].Type)
		}
	}
}

func TestParsePathData_ClosePathResetsPosition(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 10 L50 50 Z l5 5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cmds) != 4 {
		t.Fatalf("expected 4 commands, got %d", len(cmds))
	}

	if !approxEqual(cmds[3].Args[0], 15, defaultTol) || !approxEqual(cmds[3].Args[1], 15, defaultTol) {
		t.Errorf("expected (15,15) after Z + l5,5, got (%v,%v)", cmds[3].Args[0], cmds[3].Args[1])
	}
}
