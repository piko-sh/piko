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
	"strconv"
	"strings"
	"unicode"
)

const (
	// degreesToRadiansTransform holds the conversion factor from degrees to radians.
	degreesToRadiansTransform = math.Pi / 180

	// rotateWithCentreArgCount holds the expected argument
	// count for a rotate transform with centre coordinates.
	rotateWithCentreArgCount = 3

	// matrixArgCount holds the expected argument count for a full affine matrix transform.
	matrixArgCount = 6

	// matrixArgCIndex holds the index of the C element in a matrix argument list.
	matrixArgCIndex = 2

	// matrixArgDIndex holds the index of the D element in a matrix argument list.
	matrixArgDIndex = 3

	// matrixArgEIndex holds the index of the E element in a matrix argument list.
	matrixArgEIndex = 4

	// matrixArgFIndex holds the index of the F element in a matrix argument list.
	matrixArgFIndex = 5
)

// Matrix is a 2D affine transformation matrix [a b c d e f].
//
// Maps directly to the PDF cm operator:
//
//	x' = a*x + c*y + e
//	y' = b*x + d*y + f
type Matrix struct {
	// A holds the horizontal scaling component.
	A float64

	// B holds the vertical shearing component.
	B float64

	// C holds the horizontal shearing component.
	C float64

	// D holds the vertical scaling component.
	D float64

	// E holds the horizontal translation component.
	E float64

	// F holds the vertical translation component.
	F float64
}

// Identity returns the identity matrix.
//
// Returns Matrix which represents no transformation.
func Identity() Matrix { return Matrix{A: 1, D: 1} }

// Translate returns a translation matrix.
//
// Takes tx (float64) which specifies the horizontal translation offset.
// Takes ty (float64) which specifies the vertical translation offset.
//
// Returns Matrix which applies the specified translation.
func Translate(tx, ty float64) Matrix {
	return Matrix{A: 1, D: 1, E: tx, F: ty}
}

// Scale returns a scaling matrix.
//
// Takes sx (float64) which specifies the horizontal scale factor.
// Takes sy (float64) which specifies the vertical scale factor.
//
// Returns Matrix which applies the specified scaling.
func Scale(sx, sy float64) Matrix {
	return Matrix{A: sx, D: sy}
}

// Rotate returns a rotation matrix for the given angle in degrees.
//
// Takes angleDeg (float64) which specifies the rotation angle in degrees.
//
// Returns Matrix which applies the specified rotation.
func Rotate(angleDeg float64) Matrix {
	rad := angleDeg * degreesToRadiansTransform
	cos := math.Cos(rad)
	sin := math.Sin(rad)
	return Matrix{A: cos, B: sin, C: -sin, D: cos}
}

// SkewX returns a horizontal skew matrix for the given angle in degrees.
//
// Takes angleDeg (float64) which specifies the skew angle in degrees.
//
// Returns Matrix which applies the horizontal skew.
func SkewX(angleDeg float64) Matrix {
	rad := angleDeg * degreesToRadiansTransform
	return Matrix{A: 1, C: math.Tan(rad), D: 1}
}

// SkewY returns a vertical skew matrix for the given angle in degrees.
//
// Takes angleDeg (float64) which specifies the skew angle in degrees.
//
// Returns Matrix which applies the vertical skew.
func SkewY(angleDeg float64) Matrix {
	rad := angleDeg * degreesToRadiansTransform
	return Matrix{A: 1, B: math.Tan(rad), D: 1}
}

// Multiply returns the product m * n (apply n first, then m).
//
// Takes n (Matrix) which specifies the matrix to multiply with.
//
// Returns Matrix which holds the combined transformation.
func (m Matrix) Multiply(n Matrix) Matrix {
	return Matrix{
		A: m.A*n.A + m.C*n.B,
		B: m.B*n.A + m.D*n.B,
		C: m.A*n.C + m.C*n.D,
		D: m.B*n.C + m.D*n.D,
		E: m.A*n.E + m.C*n.F + m.E,
		F: m.B*n.E + m.D*n.F + m.F,
	}
}

// IsIdentity reports whether m is the identity matrix.
//
// Returns bool which indicates whether no transformation is applied.
func (m Matrix) IsIdentity() bool {
	return m.A == 1 && m.B == 0 && m.C == 0 && m.D == 1 && m.E == 0 && m.F == 0
}

// ParseTransform parses an SVG transform attribute such as
// "translate(10,20) rotate(45) scale(2)" into a combined matrix.
//
// Takes s (string) which specifies the SVG transform attribute value.
//
// Returns Matrix which holds the combined transformation result.
func ParseTransform(s string) Matrix {
	result := Identity()
	s = strings.TrimSpace(s)
	for len(s) > 0 {
		s = strings.TrimLeftFunc(s, func(r rune) bool {
			return unicode.IsSpace(r) || r == ','
		})
		if len(s) == 0 {
			break
		}

		parenIdx := strings.Index(s, "(")
		if parenIdx < 0 {
			break
		}
		fname := strings.TrimSpace(s[:parenIdx])

		closeIdx := strings.Index(s[parenIdx:], ")")
		if closeIdx < 0 {
			break
		}
		closeIdx += parenIdx

		args := parseTransformArgs(s[parenIdx+1 : closeIdx])
		s = s[closeIdx+1:]

		m := applyTransformFunc(fname, args)
		result = result.Multiply(m)
	}
	return result
}

// parseTransformArgs parses a comma-or-space-separated
// list of numeric arguments from a transform function.
//
// Takes s (string) which holds the raw argument string
// from inside the parentheses.
//
// Returns []float64 which holds the parsed numeric
// arguments.
func parseTransformArgs(s string) []float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, ",", " ")
	parts := strings.Fields(s)
	var result []float64
	for _, p := range parts {
		v, err := strconv.ParseFloat(p, 64)
		if err != nil {
			continue
		}
		result = append(result, v)
	}
	return result
}

// applyTransformFunc dispatches a named SVG transform
// function to the corresponding matrix constructor.
//
// Takes name (string) which specifies the transform
// function name.
// Takes args ([]float64) which holds the numeric arguments.
//
// Returns Matrix which holds the resulting transformation.
func applyTransformFunc(name string, args []float64) Matrix {
	switch name {
	case "translate":
		return applyTranslate(args)
	case "scale":
		return applyScale(args)
	case "rotate":
		return applyRotate(args)
	case "skewX":
		if len(args) >= 1 {
			return SkewX(args[0])
		}
		return Identity()
	case "skewY":
		if len(args) >= 1 {
			return SkewY(args[0])
		}
		return Identity()
	case "matrix":
		if len(args) >= matrixArgCount {
			return Matrix{
				A: args[0], B: args[1],
				C: args[matrixArgCIndex], D: args[matrixArgDIndex],
				E: args[matrixArgEIndex], F: args[matrixArgFIndex],
			}
		}
		return Identity()
	default:
		return Identity()
	}
}

// applyTranslate builds a translation matrix from the
// given argument list, defaulting missing values to zero.
//
// Takes args ([]float64) which holds the translate
// arguments.
//
// Returns Matrix which holds the resulting translation.
func applyTranslate(args []float64) Matrix {
	tx, ty := 0.0, 0.0
	if len(args) >= 1 {
		tx = args[0]
	}
	if len(args) >= 2 {
		ty = args[1]
	}
	return Translate(tx, ty)
}

// applyScale builds a scaling matrix from the given
// argument list, using uniform scaling when only one
// argument is provided.
//
// Takes args ([]float64) which holds the scale arguments.
//
// Returns Matrix which holds the resulting scale.
func applyScale(args []float64) Matrix {
	sx, sy := 1.0, 1.0
	if len(args) >= 1 {
		sx = args[0]
		sy = args[0]
	}
	if len(args) >= 2 {
		sy = args[1]
	}
	return Scale(sx, sy)
}

// applyRotate builds a rotation matrix from the given
// argument list, supporting optional centre-point rotation.
//
// Takes args ([]float64) which holds the rotation
// arguments.
//
// Returns Matrix which holds the resulting rotation.
func applyRotate(args []float64) Matrix {
	if len(args) == 0 {
		return Identity()
	}
	angle := args[0]
	if len(args) >= rotateWithCentreArgCount {
		cx, cy := args[1], args[2]
		return Translate(cx, cy).Multiply(Rotate(angle)).Multiply(Translate(-cx, -cy))
	}
	return Rotate(angle)
}
