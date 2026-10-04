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

package layouter_domain

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveLength_NonFiniteValuesResolveToZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  float64
	}{
		{name: "NaN", value: "NaN", want: 0},
		{name: "NaN pixels", value: "NaNpx", want: 0},
		{name: "infinity", value: "inf", want: 0},
		{name: "infinite points", value: "+Infpt", want: 0},
		{name: "overflowing multiplication", value: "1e308in", want: 0},
		{name: "ordinary length", value: "12pt", want: 12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tt.want, resolveLength(tt.value, DefaultResolutionContext()), 1e-9)
		})
	}
}

func TestResolveGapLength_ClampsNegativeGaps(t *testing.T) {
	t.Parallel()

	assert.Zero(t, resolveGapLength("-100px", DefaultResolutionContext()))
	assert.InDelta(t, 10.0, resolveGapLength("10pt", DefaultResolutionContext()), 1e-9)

	style := ResolveStyle(map[string]string{"gap": "-5px -6px", "row-gap": "-1px"}, nil, DefaultResolutionContext())
	assert.GreaterOrEqual(t, style.RowGap, 0.0)
	assert.GreaterOrEqual(t, style.ColumnGap, 0.0)
}

func TestFiniteOrZero(t *testing.T) {
	t.Parallel()

	assert.Zero(t, finiteOrZero(math.NaN()))
	assert.Zero(t, finiteOrZero(math.Inf(1)))
	assert.Zero(t, finiteOrZero(math.Inf(-1)))
	assert.InDelta(t, -3.5, finiteOrZero(-3.5), 1e-12)
}

func TestParseFloatValue_NonFiniteValuesAreZero(t *testing.T) {
	t.Parallel()

	assert.Zero(t, parseFloatValue("NaN"))
	assert.Zero(t, parseFloatValue("inf"))
	assert.InDelta(t, 2.5, parseFloatValue(" 2.5 "), 1e-12)
}
