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

// Parses CSS transform function lists into a 2D affine matrix suitable
// for the PDF cm operator. Supports translate, scale, rotate, skew, and
// matrix functions. Multiple functions are composed by left-to-right
// matrix multiplication.

import (
	"math"
	"strconv"
	"strings"
)

const (
	// matrixArgCount is the number of arguments in a CSS matrix() function.
	matrixArgCount = 6

	// gradToPiFactor converts gradians to radians (pi / 200).
	gradToPiFactor = 200

	// degreesToPiFactor converts degrees to radians (pi / 180).
	degreesToPiFactor = 180
)

// AffineMatrix holds the six components of a 2D affine transformation.
type AffineMatrix struct {
	// a holds the horizontal scaling component.
	a float64

	// b holds the vertical shearing component.
	b float64

	// c holds the horizontal shearing component.
	c float64

	// d holds the vertical scaling component.
	d float64

	// e holds the horizontal translation component.
	e float64

	// f holds the vertical translation component.
	f float64
}

// compose multiplies the current matrix by another (left-to-right).
//
// Takes fn (AffineMatrix) which is the matrix to compose with.
//
// Returns AffineMatrix which is the product of the two matrices.
func (m AffineMatrix) compose(fn AffineMatrix) AffineMatrix {
	return AffineMatrix{
		a: m.a*fn.a + m.b*fn.c,
		b: m.a*fn.b + m.b*fn.d,
		c: m.c*fn.a + m.d*fn.c,
		d: m.c*fn.b + m.d*fn.d,
		e: m.e*fn.a + m.f*fn.c + fn.e,
		f: m.e*fn.b + m.f*fn.d + fn.f,
	}
}

// ParseCSSTransform parses a CSS transform value into a
// 2D affine matrix [a, b, c, d, e, f].
//
// Takes value (string) which is the CSS transform property
// value (e.g. "rotate(45deg) scale(2)").
//
// Returns ok == false if the value cannot be parsed.
func ParseCSSTransform(value string) (AffineMatrix, bool) {
	result := AffineMatrix{a: 1, d: 1}
	value = strings.TrimSpace(value)
	if value == "" || value == "none" {
		return result, true
	}

	remaining := value
	parsedAny := false
	for remaining != "" {
		remaining = strings.TrimSpace(remaining)
		if remaining == "" {
			break
		}

		paren := strings.Index(remaining, "(")
		if paren < 0 {
			return AffineMatrix{}, false
		}
		fnName := strings.TrimSpace(remaining[:paren])
		closeIdx := findMatchingParen(remaining, paren)
		if closeIdx < 0 {
			return AffineMatrix{}, false
		}
		argsStr := remaining[paren+1 : closeIdx]
		remaining = remaining[closeIdx+1:]

		fn, fnOk := parseTransformFunction(fnName, argsStr)
		if !fnOk {
			return AffineMatrix{}, false
		}

		result = result.compose(fn)
		parsedAny = true
	}

	return result, parsedAny
}

// findMatchingParen returns the index of the closing parenthesis
// matching the opening parenthesis at position open.
//
// Takes s (string) which is the string to search.
// Takes open (int) which is the index of the opening parenthesis.
//
// Returns int which is the closing parenthesis index, or -1 if not
// found.
func findMatchingParen(s string, open int) int {
	depth := 1
	for i := open + 1; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// parseTransformFunction dispatches a single CSS transform function
// to its specific parser.
//
// Takes name (string) which is the function name.
// Takes args (string) which is the comma/space-separated argument
// string.
//
// Returns AffineMatrix which is the resulting transformation matrix.
// Returns bool which indicates whether parsing succeeded.
func parseTransformFunction(name, args string) (AffineMatrix, bool) {
	parts := splitArgs(args)

	switch strings.ToLower(name) {
	case "translate":
		return parseTranslate(parts)
	case "translatex":
		return parseTranslateX(parts)
	case "translatey":
		return parseTranslateY(parts)
	case "scale":
		return parseScale(parts)
	case "scalex":
		return parseScaleX(parts)
	case "scaley":
		return parseScaleY(parts)
	case "rotate":
		return parseRotate(parts)
	case "skewx":
		return parseSkewX(parts)
	case "skewy":
		return parseSkewY(parts)
	case "matrix":
		return parseMatrix(parts)
	default:
		return AffineMatrix{}, false
	}
}

// parseTranslate parses a CSS translate(tx[, ty]) function.
//
// Takes parts ([]string) which are the parsed arguments.
//
// Returns AffineMatrix which is the translation matrix.
// Returns bool which indicates whether parsing succeeded.
func parseTranslate(parts []string) (AffineMatrix, bool) {
	if len(parts) < 1 {
		return AffineMatrix{}, false
	}
	tx := parseLength(parts[0])
	ty := 0.0
	if len(parts) >= 2 {
		ty = parseLength(parts[1])
	}
	return AffineMatrix{a: 1, d: 1, e: tx, f: ty}, true
}

// parseTranslateX parses a CSS translateX(tx) function.
//
// Takes parts ([]string) which are the parsed arguments.
//
// Returns AffineMatrix which is the translation matrix.
// Returns bool which indicates whether parsing succeeded.
func parseTranslateX(parts []string) (AffineMatrix, bool) {
	if len(parts) < 1 {
		return AffineMatrix{}, false
	}
	return AffineMatrix{a: 1, d: 1, e: parseLength(parts[0])}, true
}

// parseTranslateY parses a CSS translateY(ty) function.
//
// Takes parts ([]string) which are the parsed arguments.
//
// Returns AffineMatrix which is the translation matrix.
// Returns bool which indicates whether parsing succeeded.
func parseTranslateY(parts []string) (AffineMatrix, bool) {
	if len(parts) < 1 {
		return AffineMatrix{}, false
	}
	return AffineMatrix{a: 1, d: 1, f: parseLength(parts[0])}, true
}

// parseScale parses a CSS scale(sx[, sy]) function.
//
// Takes parts ([]string) which are the parsed arguments.
//
// Returns AffineMatrix which is the scaling matrix.
// Returns bool which indicates whether parsing succeeded.
func parseScale(parts []string) (AffineMatrix, bool) {
	if len(parts) < 1 {
		return AffineMatrix{}, false
	}
	sx := parseNumber(parts[0])
	sy := sx
	if len(parts) >= 2 {
		sy = parseNumber(parts[1])
	}
	return AffineMatrix{a: sx, d: sy}, true
}

// parseScaleX parses a CSS scaleX(sx) function.
//
// Takes parts ([]string) which are the parsed arguments.
//
// Returns AffineMatrix which is the scaling matrix.
// Returns bool which indicates whether parsing succeeded.
func parseScaleX(parts []string) (AffineMatrix, bool) {
	if len(parts) < 1 {
		return AffineMatrix{}, false
	}
	return AffineMatrix{a: parseNumber(parts[0]), d: 1}, true
}

// parseScaleY parses a CSS scaleY(sy) function.
//
// Takes parts ([]string) which are the parsed arguments.
//
// Returns AffineMatrix which is the scaling matrix.
// Returns bool which indicates whether parsing succeeded.
func parseScaleY(parts []string) (AffineMatrix, bool) {
	if len(parts) < 1 {
		return AffineMatrix{}, false
	}
	return AffineMatrix{a: 1, d: parseNumber(parts[0])}, true
}

// parseRotate parses a CSS rotate(angle) function.
//
// Takes parts ([]string) which are the parsed arguments.
//
// Returns AffineMatrix which is the rotation matrix.
// Returns bool which indicates whether parsing succeeded.
func parseRotate(parts []string) (AffineMatrix, bool) {
	if len(parts) < 1 {
		return AffineMatrix{}, false
	}
	angle := parseAngle(parts[0])
	cosA := math.Cos(angle)
	sinA := math.Sin(angle)
	return AffineMatrix{a: cosA, b: sinA, c: -sinA, d: cosA}, true
}

// parseSkewX parses a CSS skewX(angle) function.
//
// Takes parts ([]string) which are the parsed arguments.
//
// Returns AffineMatrix which is the horizontal skew matrix.
// Returns bool which indicates whether parsing succeeded.
func parseSkewX(parts []string) (AffineMatrix, bool) {
	if len(parts) < 1 {
		return AffineMatrix{}, false
	}
	return AffineMatrix{a: 1, c: math.Tan(parseAngle(parts[0])), d: 1}, true
}

// parseSkewY parses a CSS skewY(angle) function.
//
// Takes parts ([]string) which are the parsed arguments.
//
// Returns AffineMatrix which is the vertical skew matrix.
// Returns bool which indicates whether parsing succeeded.
func parseSkewY(parts []string) (AffineMatrix, bool) {
	if len(parts) < 1 {
		return AffineMatrix{}, false
	}
	return AffineMatrix{a: 1, b: math.Tan(parseAngle(parts[0])), d: 1}, true
}

// parseMatrix parses a CSS matrix(a, b, c, d, e, f) function.
//
// Takes parts ([]string) which are the six matrix component strings.
//
// Returns AffineMatrix which holds the parsed values.
// Returns bool which indicates whether parsing succeeded.
func parseMatrix(parts []string) (AffineMatrix, bool) {
	if len(parts) < matrixArgCount {
		return AffineMatrix{}, false
	}
	return AffineMatrix{
		a: parseNumber(parts[0]), b: parseNumber(parts[1]),
		c: parseNumber(parts[2]), d: parseNumber(parts[3]),
		e: parseNumber(parts[4]), f: parseNumber(parts[5]),
	}, true
}

// splitArgs splits a CSS function argument string on commas, spaces,
// and tabs, discarding empty segments.
//
// Takes s (string) which is the raw argument string.
//
// Returns []string which holds the individual argument tokens.
func splitArgs(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// parseAngle parses a CSS angle value (deg, rad, grad, turn) and
// returns radians.
//
// Takes s (string) which is the angle string with optional unit suffix.
//
// Returns float64 which is the angle in radians.
func parseAngle(s string) float64 {
	s = strings.TrimSpace(s)

	if after, ok := strings.CutSuffix(s, "grad"); ok {
		return parseNumber(after) * math.Pi / gradToPiFactor
	}
	if after, ok := strings.CutSuffix(s, "rad"); ok {
		return parseNumber(after)
	}
	if after, ok := strings.CutSuffix(s, "turn"); ok {
		return parseNumber(after) * 2 * math.Pi
	}

	s = strings.TrimSuffix(s, "deg")
	return parseNumber(s) * math.Pi / degreesToPiFactor
}

// parseLength parses a CSS length value by stripping known unit
// suffixes and returning the numeric portion.
//
// Takes s (string) which is the length string with optional unit.
//
// Returns float64 which is the numeric value.
func parseLength(s string) float64 {
	s = strings.TrimSpace(s)

	for _, suffix := range []string{"px", "pt", "em", "rem", "%"} {
		s = strings.TrimSuffix(s, suffix)
	}
	return parseNumber(s)
}

// parseNumber parses a numeric string and returns the float64 value,
// returning 0 on failure.
//
// Takes s (string) which is the numeric string to parse.
//
// Returns float64 which is the parsed value or 0 on error.
func parseNumber(s string) float64 {
	s = strings.TrimSpace(s)
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
