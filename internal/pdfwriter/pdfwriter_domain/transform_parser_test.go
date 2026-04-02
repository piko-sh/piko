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

package pdfwriter_domain

import (
	"math"
	"testing"
)

const transformEpsilon = 1e-9

func assertTransform(t *testing.T, label string, input string, ea, eb, ec, ed, ee, ef float64) {
	t.Helper()
	m, ok := ParseCSSTransform(input)
	if !ok {
		t.Fatalf("%s: ParseCSSTransform(%q) returned ok=false", label, input)
	}
	if math.Abs(m.a-ea) > transformEpsilon || math.Abs(m.b-eb) > transformEpsilon ||
		math.Abs(m.c-ec) > transformEpsilon || math.Abs(m.d-ed) > transformEpsilon ||
		math.Abs(m.e-ee) > transformEpsilon || math.Abs(m.f-ef) > transformEpsilon {
		t.Errorf("%s: ParseCSSTransform(%q) = [%f %f %f %f %f %f], want [%f %f %f %f %f %f]",
			label, input, m.a, m.b, m.c, m.d, m.e, m.f, ea, eb, ec, ed, ee, ef)
	}
}

func assertTransformFails(t *testing.T, label string, input string) {
	t.Helper()
	_, ok := ParseCSSTransform(input)
	if ok {
		t.Errorf("%s: ParseCSSTransform(%q) should have returned ok=false", label, input)
	}
}

func TestParseCSSTransform_None(t *testing.T) {
	assertTransform(t, "empty", "", 1, 0, 0, 1, 0, 0)
	assertTransform(t, "none", "none", 1, 0, 0, 1, 0, 0)
	assertTransform(t, "whitespace", "   ", 1, 0, 0, 1, 0, 0)
}

func TestParseCSSTransform_Translate(t *testing.T) {
	assertTransform(t, "translate-x-only", "translate(10)", 1, 0, 0, 1, 10, 0)
	assertTransform(t, "translate-xy", "translate(10, 20)", 1, 0, 0, 1, 10, 20)
	assertTransform(t, "translate-px", "translate(10px, 20px)", 1, 0, 0, 1, 10, 20)
}

func TestParseCSSTransform_TranslateX(t *testing.T) {
	assertTransform(t, "translateX", "translateX(15)", 1, 0, 0, 1, 15, 0)
	assertTransform(t, "translateX-px", "translateX(15px)", 1, 0, 0, 1, 15, 0)
}

func TestParseCSSTransform_TranslateY(t *testing.T) {
	assertTransform(t, "translateY", "translateY(25)", 1, 0, 0, 1, 0, 25)
}

func TestParseCSSTransform_Scale(t *testing.T) {
	assertTransform(t, "scale-uniform", "scale(2)", 2, 0, 0, 2, 0, 0)
	assertTransform(t, "scale-xy", "scale(2, 3)", 2, 0, 0, 3, 0, 0)
	assertTransform(t, "scale-fraction", "scale(0.5)", 0.5, 0, 0, 0.5, 0, 0)
}

func TestParseCSSTransform_ScaleX(t *testing.T) {
	assertTransform(t, "scaleX", "scaleX(3)", 3, 0, 0, 1, 0, 0)
}

func TestParseCSSTransform_ScaleY(t *testing.T) {
	assertTransform(t, "scaleY", "scaleY(0.5)", 1, 0, 0, 0.5, 0, 0)
}

func TestParseCSSTransform_Rotate(t *testing.T) {

	assertTransform(t, "rotate-90deg", "rotate(90deg)",
		math.Cos(math.Pi/2), math.Sin(math.Pi/2),
		-math.Sin(math.Pi/2), math.Cos(math.Pi/2), 0, 0)

	assertTransform(t, "rotate-0deg", "rotate(0deg)", 1, 0, 0, 1, 0, 0)

	assertTransform(t, "rotate-180deg", "rotate(180deg)",
		math.Cos(math.Pi), math.Sin(math.Pi),
		-math.Sin(math.Pi), math.Cos(math.Pi), 0, 0)

	assertTransform(t, "rotate-no-unit", "rotate(45)",
		math.Cos(math.Pi/4), math.Sin(math.Pi/4),
		-math.Sin(math.Pi/4), math.Cos(math.Pi/4), 0, 0)
}

func TestParseCSSTransform_RotateRadians(t *testing.T) {
	angle := 1.5708
	assertTransform(t, "rotate-rad", "rotate(1.5708rad)",
		math.Cos(angle), math.Sin(angle),
		-math.Sin(angle), math.Cos(angle), 0, 0)
}

func TestParseCSSTransform_RotateTurn(t *testing.T) {

	angle := 0.25 * 2 * math.Pi
	assertTransform(t, "rotate-turn", "rotate(0.25turn)",
		math.Cos(angle), math.Sin(angle),
		-math.Sin(angle), math.Cos(angle), 0, 0)
}

func TestParseCSSTransform_RotateGrad(t *testing.T) {

	angle := 100.0 * math.Pi / 200
	assertTransform(t, "rotate-grad", "rotate(100grad)",
		math.Cos(angle), math.Sin(angle),
		-math.Sin(angle), math.Cos(angle), 0, 0)
}

func TestParseCSSTransform_SkewX(t *testing.T) {
	angle := 30.0 * math.Pi / 180
	assertTransform(t, "skewX", "skewX(30deg)", 1, 0, math.Tan(angle), 1, 0, 0)
}

func TestParseCSSTransform_SkewY(t *testing.T) {
	angle := 20.0 * math.Pi / 180
	assertTransform(t, "skewY", "skewY(20deg)", 1, math.Tan(angle), 0, 1, 0, 0)
}

func TestParseCSSTransform_Matrix(t *testing.T) {
	assertTransform(t, "matrix", "matrix(1, 2, 3, 4, 5, 6)", 1, 2, 3, 4, 5, 6)
	assertTransform(t, "matrix-identity", "matrix(1, 0, 0, 1, 0, 0)", 1, 0, 0, 1, 0, 0)
}

func TestParseCSSTransform_Combined(t *testing.T) {

	assertTransform(t, "translate-then-scale", "translate(10, 20) scale(2)",
		2, 0, 0, 2, 20, 40)
}

func TestParseCSSTransform_CombinedScaleTranslate(t *testing.T) {

	assertTransform(t, "scale-then-translate", "scale(2) translate(10, 20)",
		2, 0, 0, 2, 10, 20)
}

func TestParseCSSTransform_TripleComposition(t *testing.T) {

	assertTransform(t, "triple-compose", "translate(5, 0) rotate(90deg) scale(2)",
		0, 2, -2, 0, 0, 10)
}

func TestParseCSSTransform_Invalid(t *testing.T) {
	assertTransformFails(t, "no-paren", "rotate 45deg")
	assertTransformFails(t, "unclosed-paren", "rotate(45deg")
	assertTransformFails(t, "unknown-fn", "foo(1)")
	assertTransformFails(t, "matrix-too-few", "matrix(1, 2, 3)")
	assertTransformFails(t, "empty-args-translate", "translate()")
	assertTransformFails(t, "empty-args-scale", "scale()")
	assertTransformFails(t, "empty-args-rotate", "rotate()")
}

func TestParseCSSTransform_CaseInsensitive(t *testing.T) {
	assertTransform(t, "uppercase-rotate", "ROTATE(90deg)",
		math.Cos(math.Pi/2), math.Sin(math.Pi/2),
		-math.Sin(math.Pi/2), math.Cos(math.Pi/2), 0, 0)
	assertTransform(t, "mixed-case-scale", "Scale(2)", 2, 0, 0, 2, 0, 0)
}

func TestParseTranslateX_EmptyParts(t *testing.T) {
	t.Parallel()
	_, ok := parseTranslateX(nil)
	if ok {
		t.Error("expected false for nil parts")
	}
}

func TestParseTranslateY_EmptyParts(t *testing.T) {
	t.Parallel()
	_, ok := parseTranslateY(nil)
	if ok {
		t.Error("expected false for nil parts")
	}
}

func TestParseScaleX_EmptyParts(t *testing.T) {
	t.Parallel()
	_, ok := parseScaleX(nil)
	if ok {
		t.Error("expected false for nil parts")
	}
}

func TestParseScaleY_EmptyParts(t *testing.T) {
	t.Parallel()
	_, ok := parseScaleY(nil)
	if ok {
		t.Error("expected false for nil parts")
	}
}

func TestParseSkewX_EmptyParts(t *testing.T) {
	t.Parallel()
	_, ok := parseSkewX(nil)
	if ok {
		t.Error("expected false for nil parts")
	}
}

func TestParseSkewY_EmptyParts(t *testing.T) {
	t.Parallel()
	_, ok := parseSkewY(nil)
	if ok {
		t.Error("expected false for nil parts")
	}
}
