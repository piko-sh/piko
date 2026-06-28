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
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func approxEqual(a, b, tol float64) bool {
	return math.Abs(a-b) < tol
}

const (
	defaultTol = 1e-9
)

func TestParsePathData_MoveTo_Absolute(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 20")
	require.NoError(t, err)
	require.Len(t, cmds, 1)
	assert.Equal(t, byte('M'), cmds[0].Type)
	assert.True(t, approxEqual(cmds[0].Args[0], 10, defaultTol) && approxEqual(cmds[0].Args[1], 20, defaultTol), "expected (10,20)")
}

func TestParsePathData_MinifiedArcFlags(t *testing.T) {
	t.Parallel()

	minified, err := ParsePathData("M4.5 4A2.5 2.5.0 0122 6.5")
	require.NoError(t, err, "minified parse error")
	spaced, err := ParsePathData("M4.5 4A2.5 2.5 0 0 1 22 6.5")
	require.NoError(t, err, "spaced parse error")

	require.GreaterOrEqual(t, len(minified), 2, "expected the arc to produce path commands")
	require.Equal(t, len(spaced), len(minified), "minified/spaced command count differs")
	for i := range minified {
		assert.Equal(t, spaced[i].Type, minified[i].Type, "cmd %d type differs", i)
		require.Len(t, minified[i].Args, len(spaced[i].Args), "cmd %d arg count differs", i)
		for j := range minified[i].Args {
			assert.InDelta(t, spaced[i].Args[j], minified[i].Args[j], defaultTol, "cmd %d arg %d differs", i, j)
		}
	}
}

func TestParsePathData_MoveTo_Relative(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("m10 20")
	require.NoError(t, err)
	require.Len(t, cmds, 1)
	assert.Equal(t, byte('M'), cmds[0].Type)
	assert.True(t, approxEqual(cmds[0].Args[0], 10, defaultTol) && approxEqual(cmds[0].Args[1], 20, defaultTol), "expected (10,20)")
}

func TestParsePathData_ImplicitLineTo_AfterMoveTo(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 10 20 20 30 30")
	require.NoError(t, err)
	require.Len(t, cmds, 3)
	assert.Equal(t, byte('M'), cmds[0].Type)
	assert.Equal(t, byte('L'), cmds[1].Type, "cmds[1]: expected L (implicit lineTo)")
	assert.Equal(t, byte('L'), cmds[2].Type, "cmds[2]: expected L (implicit lineTo)")
	assert.True(t, approxEqual(cmds[2].Args[0], 30, defaultTol) && approxEqual(cmds[2].Args[1], 30, defaultTol), "cmds[2]: expected (30,30)")
}

func TestParsePathData_LineTo_Absolute(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 L50 60")
	require.NoError(t, err)
	require.Len(t, cmds, 2)
	assert.Equal(t, byte('L'), cmds[1].Type)
	assert.True(t, approxEqual(cmds[1].Args[0], 50, defaultTol) && approxEqual(cmds[1].Args[1], 60, defaultTol), "expected (50,60)")
}

func TestParsePathData_LineTo_Relative(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 10 l5 5")
	require.NoError(t, err)
	require.Len(t, cmds, 2)
	assert.Equal(t, byte('L'), cmds[1].Type)
	assert.True(t, approxEqual(cmds[1].Args[0], 15, defaultTol) && approxEqual(cmds[1].Args[1], 15, defaultTol), "expected (15,15)")
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
			require.NoError(t, err)
			require.Len(t, cmds, 2)

			assert.Equal(t, byte('L'), cmds[1].Type)
			assert.True(t, approxEqual(cmds[1].Args[0], tt.wantX, defaultTol) && approxEqual(cmds[1].Args[1], tt.wantY, defaultTol), "expected (%v,%v)", tt.wantX, tt.wantY)
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
			require.NoError(t, err)
			require.Len(t, cmds, 2)
			assert.Equal(t, byte('L'), cmds[1].Type)
			assert.True(t, approxEqual(cmds[1].Args[0], tt.wantX, defaultTol) && approxEqual(cmds[1].Args[1], tt.wantY, defaultTol), "expected (%v,%v)", tt.wantX, tt.wantY)
		})
	}
}

func TestParsePathData_CubicBezier_Absolute(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 C10 20 30 40 50 60")
	require.NoError(t, err)
	require.Len(t, cmds, 2)
	assert.Equal(t, byte('C'), cmds[1].Type)

	wantArgs := []float64{10, 20, 30, 40, 50, 60}
	for i, want := range wantArgs {
		assert.InDelta(t, want, cmds[1].Args[i], defaultTol, "args[%d]", i)
	}
}

func TestParsePathData_CubicBezier_Relative(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 10 c5 5 10 10 15 15")
	require.NoError(t, err)
	require.Len(t, cmds, 2)
	assert.Equal(t, byte('C'), cmds[1].Type)

	wantArgs := []float64{15, 15, 20, 20, 25, 25}
	for i, want := range wantArgs {
		assert.InDelta(t, want, cmds[1].Args[i], defaultTol, "args[%d]", i)
	}
}

func TestParsePathData_SmoothCubic_Absolute(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 C10 20 30 40 50 50 S70 80 90 90")
	require.NoError(t, err)
	require.Len(t, cmds, 3)

	assert.Equal(t, byte('C'), cmds[2].Type)
	assert.True(t, approxEqual(cmds[2].Args[0], 70, defaultTol) && approxEqual(cmds[2].Args[1], 60, defaultTol), "reflected cp1: expected (70,60)")
	assert.True(t, approxEqual(cmds[2].Args[2], 70, defaultTol) && approxEqual(cmds[2].Args[3], 80, defaultTol), "cp2: expected (70,80)")
	assert.True(t, approxEqual(cmds[2].Args[4], 90, defaultTol) && approxEqual(cmds[2].Args[5], 90, defaultTol), "end: expected (90,90)")
}

func TestParsePathData_SmoothCubic_WithoutPriorCubic(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 10 S30 40 50 50")
	require.NoError(t, err)
	require.Len(t, cmds, 2)

	assert.True(t, approxEqual(cmds[1].Args[0], 10, defaultTol) && approxEqual(cmds[1].Args[1], 10, defaultTol), "reflected cp1: expected (10,10)")
}

func TestParsePathData_SmoothCubic_Relative(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 C10 20 30 40 50 50 s20 30 40 40")
	require.NoError(t, err)
	require.Len(t, cmds, 3)

	assert.True(t, approxEqual(cmds[2].Args[0], 70, defaultTol) && approxEqual(cmds[2].Args[1], 60, defaultTol), "reflected cp1: expected (70,60)")
	assert.True(t, approxEqual(cmds[2].Args[4], 90, defaultTol) && approxEqual(cmds[2].Args[5], 90, defaultTol), "end: expected (90,90)")
}

func TestParsePathData_QuadraticBezier_Absolute(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 Q50 100 100 0")
	require.NoError(t, err)
	require.Len(t, cmds, 2)
	assert.Equal(t, byte('C'), cmds[1].Type, "expected type C (quadratic promoted to cubic)")

	assert.True(t, approxEqual(cmds[1].Args[4], 100, defaultTol) && approxEqual(cmds[1].Args[5], 0, defaultTol), "end: expected (100,0)")

	wantCP1X := 2.0 / 3.0 * 50
	wantCP1Y := 2.0 / 3.0 * 100
	assert.True(t, approxEqual(cmds[1].Args[0], wantCP1X, 0.01) && approxEqual(cmds[1].Args[1], wantCP1Y, 0.01), "cp1: expected (~%.2f,~%.2f)", wantCP1X, wantCP1Y)
}

func TestParsePathData_QuadraticBezier_Relative(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 10 q25 50 50 0")
	require.NoError(t, err)
	require.Len(t, cmds, 2)
	assert.Equal(t, byte('C'), cmds[1].Type)

	assert.True(t, approxEqual(cmds[1].Args[4], 60, defaultTol) && approxEqual(cmds[1].Args[5], 10, defaultTol), "end: expected (60,10)")
}

func TestParsePathData_SmoothQuadratic_Absolute(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 Q50 100 100 0 T200 0")
	require.NoError(t, err)
	require.Len(t, cmds, 3)

	assert.Equal(t, byte('C'), cmds[2].Type)

	assert.True(t, approxEqual(cmds[2].Args[4], 200, defaultTol) && approxEqual(cmds[2].Args[5], 0, defaultTol), "end: expected (200,0)")
}

func TestParsePathData_SmoothQuadratic_WithoutPriorQuad(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 10 T50 50")
	require.NoError(t, err)
	require.Len(t, cmds, 2)
	assert.Equal(t, byte('C'), cmds[1].Type)

	assert.True(t, approxEqual(cmds[1].Args[4], 50, defaultTol) && approxEqual(cmds[1].Args[5], 50, defaultTol), "end: expected (50,50)")
}

func TestParsePathData_SmoothQuadratic_Relative(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 Q50 100 100 0 t100 0")
	require.NoError(t, err)
	require.Len(t, cmds, 3)

	assert.True(t, approxEqual(cmds[2].Args[4], 200, defaultTol) && approxEqual(cmds[2].Args[5], 0, defaultTol), "end: expected (200,0)")
}

func TestParsePathData_Arc_Absolute(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M100 0 A100 100 0 0 1 0 100")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(cmds), 2)
	assert.Equal(t, byte('M'), cmds[0].Type)

	for i := 1; i < len(cmds); i++ {
		assert.Equal(t, byte('C'), cmds[i].Type, "cmds[%d]", i)
	}

	last := cmds[len(cmds)-1]
	assert.True(t, approxEqual(last.Args[4], 0, 0.01) && approxEqual(last.Args[5], 100, 0.01), "final endpoint: expected (~0,~100)")
}

func TestParsePathData_Arc_Relative(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M100 0 a100 100 0 0 1 -100 100")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(cmds), 2)
	last := cmds[len(cmds)-1]
	assert.True(t, approxEqual(last.Args[4], 0, 0.01) && approxEqual(last.Args[5], 100, 0.01), "final endpoint: expected (~0,~100)")
}

func TestParsePathData_Arc_HalfCircle(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 A50 50 0 0 1 100 0")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(cmds), 2)
	last := cmds[len(cmds)-1]
	assert.True(t, approxEqual(last.Args[4], 100, 0.01) && approxEqual(last.Args[5], 0, 0.01), "final endpoint: expected (~100,~0)")
}

func TestParsePathData_Arc_LargeArcFlag(t *testing.T) {
	t.Parallel()

	cmdsSmall, err := ParsePathData("M100 0 A100 100 0 0 1 0 100")
	require.NoError(t, err, "unexpected error (small)")
	cmdsLarge, err := ParsePathData("M100 0 A100 100 0 1 0 0 100")
	require.NoError(t, err, "unexpected error (large)")

	smallArcs := len(cmdsSmall) - 1
	largeArcs := len(cmdsLarge) - 1
	assert.Greater(t, largeArcs, smallArcs, "large-arc should produce more segments than small-arc")
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
			require.NoError(t, err)
			require.Len(t, cmds, 3)
			assert.Equal(t, byte('Z'), cmds[2].Type)
		})
	}
}

func TestParsePathData_MultipleSubpaths(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 L50 50 Z M100 100 L150 150 Z")
	require.NoError(t, err)

	require.Len(t, cmds, 6)
	assert.Equal(t, byte('M'), cmds[0].Type)
	assert.Equal(t, byte('M'), cmds[3].Type)
	assert.Equal(t, byte('Z'), cmds[2].Type)
	assert.Equal(t, byte('Z'), cmds[5].Type)
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
			require.NoError(t, err)
			require.Len(t, cmds, 1)
			assert.True(t, approxEqual(cmds[0].Args[0], tt.wantX, defaultTol) && approxEqual(cmds[0].Args[1], tt.wantY, defaultTol), "expected (%v,%v)", tt.wantX, tt.wantY)
		})
	}
}

func TestParsePathData_EmptyPath(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("")
	require.NoError(t, err)
	assert.Nil(t, cmds)
}

func TestParsePathData_WhitespaceOnly(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("   ")
	require.NoError(t, err)
	assert.Nil(t, cmds)
}

func TestParsePathData_InvalidCommand(t *testing.T) {
	t.Parallel()

	_, err := ParsePathData("X10 20")
	require.Error(t, err, "expected error for invalid command")
}

func TestParsePathData_NumberWithoutCommand(t *testing.T) {
	t.Parallel()

	_, err := ParsePathData("10 20")
	require.Error(t, err, "expected error when path starts with a number")
}

func TestParsePathData_CommaSeparated(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10,20 L30,40")
	require.NoError(t, err)
	require.Len(t, cmds, 2)
	assert.True(t, approxEqual(cmds[1].Args[0], 30, defaultTol) && approxEqual(cmds[1].Args[1], 40, defaultTol), "expected (30,40)")
}

func TestArcToCubics_CoincidentEndpoints(t *testing.T) {
	t.Parallel()

	result := ArcToCubics(10, 20, 50, 50, 0, false, true, 10, 20)
	assert.Empty(t, result, "expected 0 segments for coincident endpoints")
}

func TestArcToCubics_ZeroRadii(t *testing.T) {
	t.Parallel()

	result := ArcToCubics(0, 0, 0, 50, 0, false, true, 100, 0)
	require.Len(t, result, 1)
	assert.Equal(t, byte('C'), result[0].Type)

	assert.True(t, approxEqual(result[0].Args[4], 100, defaultTol) && approxEqual(result[0].Args[5], 0, defaultTol), "end: expected (100,0)")
}

func TestArcToCubics_ZeroRy(t *testing.T) {
	t.Parallel()

	result := ArcToCubics(0, 0, 50, 0, 0, false, true, 100, 0)
	require.Len(t, result, 1)
	assert.Equal(t, byte('C'), result[0].Type)
}

func TestArcToCubics_QuarterCircle(t *testing.T) {
	t.Parallel()

	result := ArcToCubics(100, 0, 100, 100, 0, false, true, 0, 100)
	require.Len(t, result, 1)

	assert.True(t, approxEqual(result[0].Args[4], 0, 0.5) && approxEqual(result[0].Args[5], 100, 0.5), "end: expected (~0,~100)")
}

func TestArcToCubics_HalfCircle(t *testing.T) {
	t.Parallel()

	result := ArcToCubics(50, 0, 50, 50, 0, false, true, -50, 0)
	require.GreaterOrEqual(t, len(result), 2, "expected at least 2 segments for half circle")

	last := result[len(result)-1]
	assert.True(t, approxEqual(last.Args[4], -50, 0.5) && approxEqual(last.Args[5], 0, 0.5), "end: expected (~-50,~0)")
}

func TestArcToCubics_FullCircle_LargeArc(t *testing.T) {
	t.Parallel()

	result := ArcToCubics(100, 0, 100, 100, 0, true, true, 100, 0.001)
	assert.GreaterOrEqual(t, len(result), 3, "expected at least 3 segments for large arc")
}

func TestArcToCubics_NegativeRadii(t *testing.T) {
	t.Parallel()

	resultPos := ArcToCubics(100, 0, 100, 100, 0, false, true, 0, 100)
	resultNeg := ArcToCubics(100, 0, -100, -100, 0, false, true, 0, 100)
	require.Equal(t, len(resultPos), len(resultNeg), "expected same segment count for negative radii")
	for i := range resultPos {
		for j := range resultPos[i].Args {
			assert.InDelta(t, resultPos[i].Args[j], resultNeg[i].Args[j], 0.01, "segment %d arg %d", i, j)
		}
	}
}

func TestArcToCubics_WithRotation(t *testing.T) {
	t.Parallel()

	result := ArcToCubics(0, 0, 100, 50, 45, false, true, 100, 100)
	require.GreaterOrEqual(t, len(result), 1, "expected at least 1 segment")

	last := result[len(result)-1]
	assert.True(t, approxEqual(last.Args[4], 100, 0.5) && approxEqual(last.Args[5], 100, 0.5), "end: expected (~100,~100)")
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
			assert.InDelta(t, tt.wantAngle, got, tt.tol)
		})
	}
}

func TestMakeAbsolute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []float64
		wantArgs []float64
		cx       float64
		cy       float64
		absCmd   byte
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
				assert.InDelta(t, want, argsCopy[i], defaultTol, "args[%d]", i)
			}
		})
	}
}

func TestParsePathData_Triangle(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 L100 0 L50 86.6 Z")
	require.NoError(t, err)
	require.Len(t, cmds, 4)
	assert.Equal(t, byte('M'), cmds[0].Type)
	assert.Equal(t, byte('L'), cmds[1].Type)
	assert.Equal(t, byte('L'), cmds[2].Type)
	assert.Equal(t, byte('Z'), cmds[3].Type)
}

func TestParsePathData_MixedRelativeAbsolute(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 10 L50 50 l10 10 Z")
	require.NoError(t, err)
	require.Len(t, cmds, 4)
	assert.True(t, approxEqual(cmds[2].Args[0], 60, defaultTol) && approxEqual(cmds[2].Args[1], 60, defaultTol), "expected (60,60)")
}

func TestParsePathData_NegativeCoordinates(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M-10 -20 L-30 -40")
	require.NoError(t, err)
	require.Len(t, cmds, 2)
	assert.True(t, approxEqual(cmds[0].Args[0], -10, defaultTol) && approxEqual(cmds[0].Args[1], -20, defaultTol), "M: expected (-10,-20)")
	assert.True(t, approxEqual(cmds[1].Args[0], -30, defaultTol) && approxEqual(cmds[1].Args[1], -40, defaultTol), "L: expected (-30,-40)")
}

func TestParsePathData_RepeatedLineTo(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 L10 10 20 20 30 30")
	require.NoError(t, err)
	require.Len(t, cmds, 4)
	for i := 1; i <= 3; i++ {
		assert.Equal(t, byte('L'), cmds[i].Type, "cmds[%d]", i)
	}
}

func TestParsePathData_DecimalWithoutLeadingDigit(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M.5 .5")
	require.NoError(t, err)
	require.Len(t, cmds, 1)
	assert.True(t, approxEqual(cmds[0].Args[0], 0.5, defaultTol) && approxEqual(cmds[0].Args[1], 0.5, defaultTol), "expected (0.5,0.5)")
}

func TestParsePathData_ArcWithScaledRadii(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M0 0 A1 1 0 0 1 100 100")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(cmds), 2, "expected at least 2 commands (M + cubics)")

	for i := 1; i < len(cmds); i++ {
		assert.Equal(t, byte('C'), cmds[i].Type, "cmds[%d]", i)
	}
}

func TestParsePathData_ClosePathResetsPosition(t *testing.T) {
	t.Parallel()

	cmds, err := ParsePathData("M10 10 L50 50 Z l5 5")
	require.NoError(t, err)

	require.Len(t, cmds, 4)

	assert.True(t, approxEqual(cmds[3].Args[0], 15, defaultTol) && approxEqual(cmds[3].Args[1], 15, defaultTol), "expected (15,15) after Z + l5,5")
}

func TestParsePathFloatRejectsNonFiniteValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		token string
	}{
		{name: "overflow to positive infinity", token: "1e400"},
		{name: "overflow to negative infinity", token: "-1e400"},
		{name: "not a number", token: "NaN"},
		{name: "infinity literal", token: "Inf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parsePathFloat(tt.token)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrNonFiniteCoordinate)
		})
	}
}

func TestParsePathFloatAcceptsFiniteValues(t *testing.T) {
	t.Parallel()

	value, err := parsePathFloat("12.5")
	require.NoError(t, err)
	assert.InDelta(t, 12.5, value, defaultTol)
}

func TestParsePathDataRejectsNonFiniteCoordinate(t *testing.T) {
	t.Parallel()

	_, err := ParsePathData("M1e400 0 L10 10")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNonFiniteCoordinate)
}

func TestVecAngleReturnsZeroForZeroLengthVector(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0.0, vecAngle(0, 0, 1, 0))
	assert.Equal(t, 0.0, vecAngle(1, 0, 0, 0))
	assert.Equal(t, 0.0, vecAngle(0, 0, 0, 0))
}

func TestParsePathDataBoundsCommandCount(t *testing.T) {
	t.Parallel()

	var builder strings.Builder
	builder.WriteString("M0 0")
	for range maxPathCommands + 10 {
		builder.WriteString(" L1 1")
	}

	commands, err := ParsePathData(builder.String())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTooManyPathCommands)
	assert.LessOrEqual(t, len(commands), 0, "no commands should be returned on overflow")
}

func TestEmitArcSegmentsBoundsFanOut(t *testing.T) {
	t.Parallel()

	segments := emitArcSegments(0, 0, 10, 10, 0, 0, 1e9)
	assert.LessOrEqual(t, len(segments), maxArcSegments)
	assert.NotEmpty(t, segments)
}

func TestMakeAbsoluteCubicEndYRelativeConversion(t *testing.T) {
	t.Parallel()

	args := []float64{1, 2, 3, 4, 5, 6}
	makeAbsolute('C', args, 10, 1000)

	assert.InDelta(t, 11.0, args[0], defaultTol)
	assert.InDelta(t, 1002.0, args[1], defaultTol)
	assert.InDelta(t, 13.0, args[cubicCP1XIndex], defaultTol)
	assert.InDelta(t, 1004.0, args[cubicCP1YIndex], defaultTol)
	assert.InDelta(t, 15.0, args[cubicEndXIndex], defaultTol)
	assert.InDelta(t, 1006.0, args[cubicEndYIndex], defaultTol)
}

func TestErrNonFiniteCoordinateIsSentinel(t *testing.T) {
	t.Parallel()

	wrapped := errors.Join(ErrNonFiniteCoordinate, errors.New("context"))
	assert.ErrorIs(t, wrapped, ErrNonFiniteCoordinate)
}
