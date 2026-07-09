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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/layouter/layouter_dto"
)

func approxEqual(a, b float64) bool { return math.Abs(a-b) < 0.01 }

func TestApplyPageCSS_MarginShorthandAndSize(t *testing.T) {
	in := layouter_dto.PageConfig{Width: 100, Height: 100}
	out := applyPageCSS(`@page { size: A4; margin: 13mm 14mm; }`, in)

	assert.InDelta(t, 595.28, out.Width, 0.01, "size A4 width not applied")
	assert.InDelta(t, 841.89, out.Height, 0.01, "size A4 height not applied")
	assert.InDelta(t, 13*ptPerMM, out.MarginTop, 0.01, "vertical margin top")
	assert.InDelta(t, 13*ptPerMM, out.MarginBottom, 0.01, "vertical margin bottom")
	assert.InDelta(t, 14*ptPerMM, out.MarginLeft, 0.01, "horizontal margin left")
	assert.InDelta(t, 14*ptPerMM, out.MarginRight, 0.01, "horizontal margin right")
}

func TestApplyPageCSS_Longhand(t *testing.T) {
	in := layouter_dto.PageConfig{Width: 595.28, Height: 841.89}
	out := applyPageCSS(`@page { margin-top: 1in; margin-left: 36pt; }`, in)
	assert.InDelta(t, 72, out.MarginTop, 0.01, "margin-top 1in")
	assert.InDelta(t, 36, out.MarginLeft, 0.01, "margin-left 36pt")

	assert.Zero(t, out.MarginRight, "unspecified right margin should stay 0")
	assert.Zero(t, out.MarginBottom, "unspecified bottom margin should stay 0")
}

func TestApplyPageCSS_NoRuleUnchanged(t *testing.T) {
	in := layouter_dto.PageConfig{Width: 595.28, Height: 841.89, MarginTop: 10}
	out := applyPageCSS(`.cv { color: red; }`, in)
	assert.Equal(t, in, out, "no @page rule should leave config unchanged")
}

func TestApplyPageCSS_IgnoresNamedPage(t *testing.T) {
	in := layouter_dto.PageConfig{Width: 595.28, Height: 841.89}
	out := applyPageCSS(`@page :first { margin: 50mm; }`, in)
	assert.Zero(t, out.MarginTop, "@page :first should be ignored")
}

func TestParseLengthPt_RejectsNonFiniteAndOversized(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantOK    bool
		wantValue float64
	}{
		{name: "positive infinity", value: "inf", wantOK: false},
		{name: "negative infinity", value: "-inf", wantOK: false},
		{name: "not a number", value: "nan", wantOK: false},
		{name: "oversized value in points", value: "100000pt", wantOK: false},
		{name: "oversized value in inches", value: "300in", wantOK: false},
		{name: "negative value", value: "-5pt", wantOK: false},
		{name: "normal value in points", value: "72pt", wantOK: true, wantValue: 72},
		{name: "normal value in inches", value: "1in", wantOK: true, wantValue: 72},
		{name: "at the upper bound", value: "200in", wantOK: true, wantValue: maxPageLengthPt},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, ok := parseLengthPt(test.value)
			require.Equal(t, test.wantOK, ok, "parseLengthPt(%q) ok", test.value)
			if test.wantOK {
				assert.InDelta(t, test.wantValue, value, 0.01, "parseLengthPt(%q)", test.value)
			}
		})
	}
}

func TestApplyPageCSS_NonFiniteSizeIgnored(t *testing.T) {
	in := layouter_dto.PageConfig{Width: 595.28, Height: 841.89}
	out := applyPageCSS(`@page { size: inf inf; margin: nan; }`, in)
	assert.Equal(t, in.Width, out.Width, "non-finite width should be ignored")
	assert.Equal(t, in.Height, out.Height, "non-finite height should be ignored")
	assert.Zero(t, out.MarginTop, "non-finite margin should be ignored")
}
