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

// Provides helpers for emitting rounded rectangle paths to a PDF
// content stream using cubic Bezier curves for the corner arcs.

import "math"

// kappa is the control-point distance for approximating a
// quarter-circle arc with a cubic Bezier curve. Equal to
// 4/3 * (sqrt(2) - 1).
const kappa = 0.5522847498

// emitRoundedRectPath builds a closed rounded rectangle path on the
// given content stream.
//
// All coordinates are in PDF space (origin at bottom-left, Y increasing
// upward). The rectangle has its lower-left corner at (x, y) with the
// given width and height. Radii are clamped per the CSS spec so that
// adjacent radii never exceed half the shared edge length.
//
// Takes stream (*ContentStream) which receives the path operators.
// Takes x, y (float64) which define the lower-left corner position.
// Takes width, height (float64) which define the rectangle dimensions.
// Takes tlr (float64) which is the top-left corner radius.
// Takes trr (float64) which is the top-right corner radius.
// Takes brr (float64) which is the bottom-right corner radius.
// Takes blr (float64) which is the bottom-left corner radius.
//
//nolint:revive // geometric primitive requires all params
func emitRoundedRectPath(stream *ContentStream, x, y, width, height, tlr, trr, brr, blr float64) {
	tlr, trr, brr, blr = clampRadii(width, height, tlr, trr, brr, blr)

	left := x
	bottom := y
	right := x + width
	top := y + height

	stream.MoveTo(left+tlr, top)

	stream.LineTo(right-trr, top)
	if trr > 0 {
		stream.CurveTo(
			right-trr+trr*kappa, top,
			right, top-trr+trr*kappa,
			right, top-trr)
	}

	stream.LineTo(right, bottom+brr)
	if brr > 0 {
		stream.CurveTo(
			right, bottom+brr-brr*kappa,
			right-brr+brr*kappa, bottom,
			right-brr, bottom)
	}

	stream.LineTo(left+blr, bottom)
	if blr > 0 {
		stream.CurveTo(
			left+blr-blr*kappa, bottom,
			left, bottom+blr-blr*kappa,
			left, bottom+blr)
	}

	stream.LineTo(left, top-tlr)
	if tlr > 0 {
		stream.CurveTo(
			left, top-tlr+tlr*kappa,
			left+tlr-tlr*kappa, top,
			left+tlr, top)
	}

	stream.ClosePath()
}

// clampRadii scales border radii so that adjacent radii never exceed
// half the length of the shared edge, per the CSS border-radius spec.
//
// Takes width, height (float64) which are the rectangle dimensions.
// Takes tlr, trr, brr, blr (float64) which are the four corner radii.
//
// Returns the clamped (tlr, trr, brr, blr) values.
func clampRadii(width, height, tlr, trr, brr, blr float64) (clampedTLR, clampedTRR, clampedBRR, clampedBLR float64) {
	factor := 1.0
	if s := tlr + trr; s > 0 {
		factor = math.Min(factor, width/s)
	}
	if s := trr + brr; s > 0 {
		factor = math.Min(factor, height/s)
	}
	if s := brr + blr; s > 0 {
		factor = math.Min(factor, width/s)
	}
	if s := blr + tlr; s > 0 {
		factor = math.Min(factor, height/s)
	}

	if factor < 1.0 {
		tlr *= factor
		trr *= factor
		brr *= factor
		blr *= factor
	}

	return tlr, trr, brr, blr
}
