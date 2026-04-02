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

const matrixTol = 1e-9

func matrixApproxEqual(a, b Matrix, tol float64) bool {
	return math.Abs(a.A-b.A) < tol &&
		math.Abs(a.B-b.B) < tol &&
		math.Abs(a.C-b.C) < tol &&
		math.Abs(a.D-b.D) < tol &&
		math.Abs(a.E-b.E) < tol &&
		math.Abs(a.F-b.F) < tol
}

func TestScale(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sx   float64
		sy   float64
		want Matrix
	}{
		{
			name: "identity (1,1)",
			sx:   1, sy: 1,
			want: Matrix{A: 1, B: 0, C: 0, D: 1, E: 0, F: 0},
		},
		{
			name: "uniform (2,2)",
			sx:   2, sy: 2,
			want: Matrix{A: 2, B: 0, C: 0, D: 2, E: 0, F: 0},
		},
		{
			name: "non-uniform (2,3)",
			sx:   2, sy: 3,
			want: Matrix{A: 2, B: 0, C: 0, D: 3, E: 0, F: 0},
		},
		{
			name: "zero scale",
			sx:   0, sy: 0,
			want: Matrix{A: 0, B: 0, C: 0, D: 0, E: 0, F: 0},
		},
		{
			name: "negative scale",
			sx:   -1, sy: -1,
			want: Matrix{A: -1, B: 0, C: 0, D: -1, E: 0, F: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Scale(tt.sx, tt.sy)
			if !matrixApproxEqual(got, tt.want, matrixTol) {
				t.Errorf("Scale(%v,%v) = %+v, want %+v", tt.sx, tt.sy, got, tt.want)
			}
		})
	}
}

func TestRotate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		deg  float64
		want Matrix
	}{
		{
			name: "0 degrees (identity)",
			deg:  0,
			want: Matrix{A: 1, B: 0, C: 0, D: 1, E: 0, F: 0},
		},
		{
			name: "90 degrees",
			deg:  90,
			want: Matrix{A: math.Cos(math.Pi / 2), B: math.Sin(math.Pi / 2), C: -math.Sin(math.Pi / 2), D: math.Cos(math.Pi / 2)},
		},
		{
			name: "180 degrees",
			deg:  180,
			want: Matrix{A: math.Cos(math.Pi), B: math.Sin(math.Pi), C: -math.Sin(math.Pi), D: math.Cos(math.Pi)},
		},
		{
			name: "360 degrees (full rotation)",
			deg:  360,
			want: Matrix{A: 1, B: 0, C: 0, D: 1, E: 0, F: 0},
		},
		{
			name: "45 degrees",
			deg:  45,
			want: Matrix{A: math.Cos(math.Pi / 4), B: math.Sin(math.Pi / 4), C: -math.Sin(math.Pi / 4), D: math.Cos(math.Pi / 4)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Rotate(tt.deg)
			if !matrixApproxEqual(got, tt.want, matrixTol) {
				t.Errorf("Rotate(%v) = %+v, want %+v", tt.deg, got, tt.want)
			}
		})
	}
}

func TestSkewX(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		deg  float64
		want Matrix
	}{
		{
			name: "0 degrees (identity)",
			deg:  0,
			want: Matrix{A: 1, B: 0, C: 0, D: 1},
		},
		{
			name: "45 degrees",
			deg:  45,
			want: Matrix{A: 1, B: 0, C: math.Tan(math.Pi / 4), D: 1},
		},
		{
			name: "30 degrees",
			deg:  30,
			want: Matrix{A: 1, B: 0, C: math.Tan(math.Pi / 6), D: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := SkewX(tt.deg)
			if !matrixApproxEqual(got, tt.want, matrixTol) {
				t.Errorf("SkewX(%v) = %+v, want %+v", tt.deg, got, tt.want)
			}
		})
	}
}

func TestSkewY(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		deg  float64
		want Matrix
	}{
		{
			name: "0 degrees (identity)",
			deg:  0,
			want: Matrix{A: 1, B: 0, C: 0, D: 1},
		},
		{
			name: "45 degrees",
			deg:  45,
			want: Matrix{A: 1, B: math.Tan(math.Pi / 4), C: 0, D: 1},
		},
		{
			name: "30 degrees",
			deg:  30,
			want: Matrix{A: 1, B: math.Tan(math.Pi / 6), C: 0, D: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := SkewY(tt.deg)
			if !matrixApproxEqual(got, tt.want, matrixTol) {
				t.Errorf("SkewY(%v) = %+v, want %+v", tt.deg, got, tt.want)
			}
		})
	}
}

func TestApplyScale(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []float64
		want Matrix
	}{
		{
			name: "0 args - identity",
			args: nil,
			want: Matrix{A: 1, B: 0, C: 0, D: 1},
		},
		{
			name: "1 arg - uniform scale",
			args: []float64{3},
			want: Matrix{A: 3, B: 0, C: 0, D: 3},
		},
		{
			name: "2 args - non-uniform scale",
			args: []float64{2, 5},
			want: Matrix{A: 2, B: 0, C: 0, D: 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := applyScale(tt.args)
			if !matrixApproxEqual(got, tt.want, matrixTol) {
				t.Errorf("applyScale(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestApplyRotate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []float64
		want Matrix
	}{
		{
			name: "0 args - identity",
			args: nil,
			want: Identity(),
		},
		{
			name: "1 arg - simple rotation 90 degrees",
			args: []float64{90},
			want: Rotate(90),
		},
		{
			name: "3 args - rotate about centre (50,50)",
			args: []float64{90, 50, 50},

			want: Translate(50, 50).Multiply(Rotate(90)).Multiply(Translate(-50, -50)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := applyRotate(tt.args)
			if !matrixApproxEqual(got, tt.want, matrixTol) {
				t.Errorf("applyRotate(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestApplyTransformFunc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		funcName string
		args     []float64
		want     Matrix
	}{
		{
			name:     "translate with 2 args",
			funcName: "translate",
			args:     []float64{10, 20},
			want:     Translate(10, 20),
		},
		{
			name:     "translate with 1 arg (ty defaults to 0)",
			funcName: "translate",
			args:     []float64{10},
			want:     Translate(10, 0),
		},
		{
			name:     "translate with 0 args",
			funcName: "translate",
			args:     nil,
			want:     Translate(0, 0),
		},
		{
			name:     "scale with 1 arg (uniform)",
			funcName: "scale",
			args:     []float64{2},
			want:     Scale(2, 2),
		},
		{
			name:     "scale with 2 args",
			funcName: "scale",
			args:     []float64{2, 3},
			want:     Scale(2, 3),
		},
		{
			name:     "rotate with 1 arg",
			funcName: "rotate",
			args:     []float64{45},
			want:     Rotate(45),
		},
		{
			name:     "rotate with 3 args (about centre)",
			funcName: "rotate",
			args:     []float64{45, 100, 200},
			want:     Translate(100, 200).Multiply(Rotate(45)).Multiply(Translate(-100, -200)),
		},
		{
			name:     "skewX",
			funcName: "skewX",
			args:     []float64{30},
			want:     SkewX(30),
		},
		{
			name:     "skewX with 0 args - identity",
			funcName: "skewX",
			args:     nil,
			want:     Identity(),
		},
		{
			name:     "skewY",
			funcName: "skewY",
			args:     []float64{30},
			want:     SkewY(30),
		},
		{
			name:     "skewY with 0 args - identity",
			funcName: "skewY",
			args:     nil,
			want:     Identity(),
		},
		{
			name:     "matrix with 6 args",
			funcName: "matrix",
			args:     []float64{1, 2, 3, 4, 5, 6},
			want:     Matrix{A: 1, B: 2, C: 3, D: 4, E: 5, F: 6},
		},
		{
			name:     "matrix with fewer than 6 args - identity",
			funcName: "matrix",
			args:     []float64{1, 2, 3},
			want:     Identity(),
		},
		{
			name:     "unknown function - identity",
			funcName: "wobble",
			args:     []float64{42},
			want:     Identity(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := applyTransformFunc(tt.funcName, tt.args)
			if !matrixApproxEqual(got, tt.want, matrixTol) {
				t.Errorf("applyTransformFunc(%q, %v) = %+v, want %+v", tt.funcName, tt.args, got, tt.want)
			}
		})
	}
}

func TestParseTransform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    string
		want Matrix
	}{
		{
			name: "empty string",
			s:    "",
			want: Identity(),
		},
		{
			name: "whitespace only",
			s:    "   ",
			want: Identity(),
		},
		{
			name: "single translate",
			s:    "translate(10,20)",
			want: Translate(10, 20),
		},
		{
			name: "single scale",
			s:    "scale(2)",
			want: Scale(2, 2),
		},
		{
			name: "single rotate",
			s:    "rotate(90)",
			want: Rotate(90),
		},
		{
			name: "compound translate then scale",
			s:    "translate(10,20) scale(2)",
			want: Translate(10, 20).Multiply(Scale(2, 2)),
		},
		{
			name: "compound with spaces",
			s:    "  translate( 10 , 20 )  rotate( 45 )  ",
			want: Translate(10, 20).Multiply(Rotate(45)),
		},
		{
			name: "skewX",
			s:    "skewX(30)",
			want: SkewX(30),
		},
		{
			name: "skewY",
			s:    "skewY(30)",
			want: SkewY(30),
		},
		{
			name: "matrix",
			s:    "matrix(1 0 0 1 50 100)",
			want: Matrix{A: 1, B: 0, C: 0, D: 1, E: 50, F: 100},
		},
		{
			name: "rotate about centre",
			s:    "rotate(90, 50, 50)",
			want: Translate(50, 50).Multiply(Rotate(90)).Multiply(Translate(-50, -50)),
		},
		{
			name: "three chained transforms",
			s:    "translate(10,20) scale(2) rotate(45)",
			want: Translate(10, 20).Multiply(Scale(2, 2)).Multiply(Rotate(45)),
		},
		{
			name: "no closing paren - stops parsing gracefully",
			s:    "translate(10,20",
			want: Identity(),
		},
		{
			name: "no opening paren - stops parsing gracefully",
			s:    "translate",
			want: Identity(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ParseTransform(tt.s)
			if !matrixApproxEqual(got, tt.want, matrixTol) {
				t.Errorf("ParseTransform(%q) = %+v, want %+v", tt.s, got, tt.want)
			}
		})
	}
}

func TestIdentity(t *testing.T) {
	t.Parallel()

	m := Identity()
	if m.A != 1 || m.B != 0 || m.C != 0 || m.D != 1 || m.E != 0 || m.F != 0 {
		t.Errorf("Identity() = %+v, want {1 0 0 1 0 0}", m)
	}
	if !m.IsIdentity() {
		t.Error("Identity().IsIdentity() returned false")
	}
}

func TestIsIdentity_NonIdentity(t *testing.T) {
	t.Parallel()

	m := Translate(1, 0)
	if m.IsIdentity() {
		t.Error("Translate(1,0).IsIdentity() returned true")
	}
}

func TestMultiply_IdentityIsNeutral(t *testing.T) {
	t.Parallel()

	m := Matrix{A: 2, B: 3, C: 4, D: 5, E: 6, F: 7}
	got := m.Multiply(Identity())
	if !matrixApproxEqual(got, m, matrixTol) {
		t.Errorf("m * Identity = %+v, want %+v", got, m)
	}
	got = Identity().Multiply(m)
	if !matrixApproxEqual(got, m, matrixTol) {
		t.Errorf("Identity * m = %+v, want %+v", got, m)
	}
}

func TestMultiply_ScaleTranslate(t *testing.T) {
	t.Parallel()

	got := Scale(2, 2).Multiply(Translate(10, 20))
	want := Matrix{A: 2, B: 0, C: 0, D: 2, E: 20, F: 40}
	if !matrixApproxEqual(got, want, matrixTol) {
		t.Errorf("Scale(2,2).Multiply(Translate(10,20)) = %+v, want %+v", got, want)
	}
}
