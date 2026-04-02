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
	"math"
	"strings"
	"testing"
)

const epsilon = 1e-6

func floatsEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestClampRadii_NoClamping(t *testing.T) {

	tlr, trr, brr, blr := clampRadii(100, 80, 10, 15, 5, 8)

	if !floatsEqual(tlr, 10) {
		t.Errorf("expected tlr=10, got %f", tlr)
	}
	if !floatsEqual(trr, 15) {
		t.Errorf("expected trr=15, got %f", trr)
	}
	if !floatsEqual(brr, 5) {
		t.Errorf("expected brr=5, got %f", brr)
	}
	if !floatsEqual(blr, 8) {
		t.Errorf("expected blr=8, got %f", blr)
	}
}

func TestClampRadii_ProportionalScaling(t *testing.T) {

	tlr, trr, brr, blr := clampRadii(100, 80, 60, 60, 60, 60)

	expected := 60.0 * (80.0 / 120.0)
	if !floatsEqual(tlr, expected) {
		t.Errorf("expected tlr=%f, got %f", expected, tlr)
	}
	if !floatsEqual(trr, expected) {
		t.Errorf("expected trr=%f, got %f", expected, trr)
	}
	if !floatsEqual(brr, expected) {
		t.Errorf("expected brr=%f, got %f", expected, brr)
	}
	if !floatsEqual(blr, expected) {
		t.Errorf("expected blr=%f, got %f", expected, blr)
	}
}

func TestClampRadii_ZeroRadiiPassThrough(t *testing.T) {
	tlr, trr, brr, blr := clampRadii(100, 80, 0, 0, 0, 0)

	if !floatsEqual(tlr, 0) {
		t.Errorf("expected tlr=0, got %f", tlr)
	}
	if !floatsEqual(trr, 0) {
		t.Errorf("expected trr=0, got %f", trr)
	}
	if !floatsEqual(brr, 0) {
		t.Errorf("expected brr=0, got %f", brr)
	}
	if !floatsEqual(blr, 0) {
		t.Errorf("expected blr=0, got %f", blr)
	}
}

func TestClampRadii_SingleLargeRadius(t *testing.T) {

	tlr, trr, brr, blr := clampRadii(100, 80, 90, 0, 0, 0)

	expected_tlr := 90.0 * (80.0 / 90.0)
	if !floatsEqual(tlr, expected_tlr) {
		t.Errorf("expected tlr=%f, got %f", expected_tlr, tlr)
	}

	if !floatsEqual(trr, 0) {
		t.Errorf("expected trr=0, got %f", trr)
	}
	if !floatsEqual(brr, 0) {
		t.Errorf("expected brr=0, got %f", brr)
	}
	if !floatsEqual(blr, 0) {
		t.Errorf("expected blr=0, got %f", blr)
	}
}

func TestClampRadii_AsymmetricRadiiOnSameEdge(t *testing.T) {

	tlr, trr, brr, blr := clampRadii(100, 80, 70, 50, 0, 0)

	factor := 100.0 / 120.0
	expected_tlr := 70.0 * factor
	expected_trr := 50.0 * factor

	if !floatsEqual(tlr, expected_tlr) {
		t.Errorf("expected tlr=%f, got %f", expected_tlr, tlr)
	}
	if !floatsEqual(trr, expected_trr) {
		t.Errorf("expected trr=%f, got %f", expected_trr, trr)
	}
	if !floatsEqual(brr, 0) {
		t.Errorf("expected brr=0, got %f", brr)
	}
	if !floatsEqual(blr, 0) {
		t.Errorf("expected blr=0, got %f", blr)
	}
}

func TestEmitRoundedRectPath_AllRadiiZero(t *testing.T) {

	var stream ContentStream
	emitRoundedRectPath(&stream, 10, 20, 100, 80, 0, 0, 0, 0)
	output := stream.String()

	if strings.Contains(output, " c\n") {
		t.Error("expected no CurveTo operators when all radii are zero")
	}

	if !strings.Contains(output, " m\n") {
		t.Error("expected at least one MoveTo operator")
	}
	if !strings.Contains(output, " l\n") {
		t.Error("expected at least one LineTo operator")
	}
	if !strings.HasSuffix(output, "h\n") {
		t.Error("expected path to end with ClosePath (h)")
	}
}

func TestEmitRoundedRectPath_UniformRadii(t *testing.T) {

	var stream ContentStream
	emitRoundedRectPath(&stream, 0, 0, 100, 80, 10, 10, 10, 10)
	output := stream.String()

	curve_count := strings.Count(output, " c\n")
	if curve_count != 4 {
		t.Errorf("expected 4 CurveTo operators for 4 rounded corners, got %d", curve_count)
	}
}

func TestEmitRoundedRectPath_MixedRadii(t *testing.T) {

	var stream ContentStream
	emitRoundedRectPath(&stream, 0, 0, 100, 80, 0, 15, 0, 15)
	output := stream.String()

	curve_count := strings.Count(output, " c\n")
	if curve_count != 2 {
		t.Errorf("expected 2 CurveTo operators for 2 rounded corners, got %d", curve_count)
	}
}

func TestEmitRoundedRectPath_ClosedPath(t *testing.T) {

	var stream ContentStream
	emitRoundedRectPath(&stream, 5, 10, 200, 150, 20, 15, 10, 5)
	output := stream.String()

	if !strings.HasSuffix(output, "h\n") {
		t.Error("expected path to be closed with 'h' operator")
	}
}
