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

//go:build !integration

package pdfwriter_domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	epsilon = 1e-6
)

func TestClampRadii_NoClamping(t *testing.T) {

	tlr, trr, brr, blr := clampRadii(100, 80, 10, 15, 5, 8)

	assert.InDelta(t, 10.0, tlr, epsilon, "tlr")
	assert.InDelta(t, 15.0, trr, epsilon, "trr")
	assert.InDelta(t, 5.0, brr, epsilon, "brr")
	assert.InDelta(t, 8.0, blr, epsilon, "blr")
}

func TestClampRadii_ProportionalScaling(t *testing.T) {

	tlr, trr, brr, blr := clampRadii(100, 80, 60, 60, 60, 60)

	expected := 60.0 * (80.0 / 120.0)
	assert.InDelta(t, expected, tlr, epsilon, "tlr")
	assert.InDelta(t, expected, trr, epsilon, "trr")
	assert.InDelta(t, expected, brr, epsilon, "brr")
	assert.InDelta(t, expected, blr, epsilon, "blr")
}

func TestClampRadii_ZeroRadiiPassThrough(t *testing.T) {
	tlr, trr, brr, blr := clampRadii(100, 80, 0, 0, 0, 0)

	assert.InDelta(t, 0.0, tlr, epsilon, "tlr")
	assert.InDelta(t, 0.0, trr, epsilon, "trr")
	assert.InDelta(t, 0.0, brr, epsilon, "brr")
	assert.InDelta(t, 0.0, blr, epsilon, "blr")
}

func TestClampRadii_SingleLargeRadius(t *testing.T) {

	tlr, trr, brr, blr := clampRadii(100, 80, 90, 0, 0, 0)

	expected_tlr := 90.0 * (80.0 / 90.0)
	assert.InDelta(t, expected_tlr, tlr, epsilon, "tlr")

	assert.InDelta(t, 0.0, trr, epsilon, "trr")
	assert.InDelta(t, 0.0, brr, epsilon, "brr")
	assert.InDelta(t, 0.0, blr, epsilon, "blr")
}

func TestClampRadii_AsymmetricRadiiOnSameEdge(t *testing.T) {

	tlr, trr, brr, blr := clampRadii(100, 80, 70, 50, 0, 0)

	factor := 100.0 / 120.0
	expected_tlr := 70.0 * factor
	expected_trr := 50.0 * factor

	assert.InDelta(t, expected_tlr, tlr, epsilon, "tlr")
	assert.InDelta(t, expected_trr, trr, epsilon, "trr")
	assert.InDelta(t, 0.0, brr, epsilon, "brr")
	assert.InDelta(t, 0.0, blr, epsilon, "blr")
}

func TestEmitRoundedRectPath_AllRadiiZero(t *testing.T) {

	var stream ContentStream
	emitRoundedRectPath(&stream, 10, 20, 100, 80, 0, 0, 0, 0)
	output := stream.String()

	assert.NotContains(t, output, " c\n", "expected no CurveTo operators when all radii are zero")

	assert.Contains(t, output, " m\n", "expected at least one MoveTo operator")
	assert.Contains(t, output, " l\n", "expected at least one LineTo operator")
	assert.True(t, strings.HasSuffix(output, "h\n"), "expected path to end with ClosePath (h)")
}

func TestEmitRoundedRectPath_UniformRadii(t *testing.T) {

	var stream ContentStream
	emitRoundedRectPath(&stream, 0, 0, 100, 80, 10, 10, 10, 10)
	output := stream.String()

	curve_count := strings.Count(output, " c\n")
	assert.Equal(t, 4, curve_count, "expected 4 CurveTo operators for 4 rounded corners")
}

func TestEmitRoundedRectPath_MixedRadii(t *testing.T) {

	var stream ContentStream
	emitRoundedRectPath(&stream, 0, 0, 100, 80, 0, 15, 0, 15)
	output := stream.String()

	curve_count := strings.Count(output, " c\n")
	assert.Equal(t, 2, curve_count, "expected 2 CurveTo operators for 2 rounded corners")
}

func TestEmitRoundedRectPath_ClosedPath(t *testing.T) {

	var stream ContentStream
	emitRoundedRectPath(&stream, 5, 10, 200, 150, 20, 15, 10, 5)
	output := stream.String()

	assert.True(t, strings.HasSuffix(output, "h\n"), "expected path to be closed with 'h' operator")
}
