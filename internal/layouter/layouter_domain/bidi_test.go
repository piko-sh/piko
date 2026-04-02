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

package layouter_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitIntoBidiRuns_EmptyString(t *testing.T) {
	runs := splitIntoBidiRuns("", DirectionLTR, UnicodeBidiNormal)
	assert.Nil(t, runs)
}

func TestSplitIntoBidiRuns_PureLTR(t *testing.T) {
	runs := splitIntoBidiRuns("Hello World", DirectionLTR, UnicodeBidiNormal)
	assert.Len(t, runs, 1)
	assert.Equal(t, "Hello World", runs[0].text)
	assert.Equal(t, DirectionLTR, runs[0].direction)
}

func TestSplitIntoBidiRuns_PureRTL(t *testing.T) {

	runs := splitIntoBidiRuns("\u05E9\u05DC\u05D5\u05DD", DirectionRTL, UnicodeBidiNormal)
	assert.Len(t, runs, 1)
	assert.Equal(t, DirectionRTL, runs[0].direction)
}

func TestSplitIntoBidiRuns_MixedLTRAndRTL(t *testing.T) {

	text := "Hello \u05E9\u05DC\u05D5\u05DD World"
	runs := splitIntoBidiRuns(text, DirectionLTR, UnicodeBidiNormal)

	assert.GreaterOrEqual(t, len(runs), 2, "expected at least 2 runs for mixed bidi text")

	hasLTR := false
	hasRTL := false
	for _, run := range runs {
		if run.direction == DirectionLTR {
			hasLTR = true
		}
		if run.direction == DirectionRTL {
			hasRTL = true
		}
	}
	assert.True(t, hasLTR, "expected at least one LTR run")
	assert.True(t, hasRTL, "expected at least one RTL run")
}

func TestSplitIntoBidiRuns_BidiOverride(t *testing.T) {
	text := "Hello World"
	runs := splitIntoBidiRuns(text, DirectionRTL, UnicodeBidiBidiOverride)

	assert.Len(t, runs, 1)
	assert.Equal(t, text, runs[0].text)
	assert.Equal(t, DirectionRTL, runs[0].direction)
}

func TestSplitIntoBidiRuns_RTLBaseWithLTR(t *testing.T) {

	text := "\u05E9\u05DC\u05D5\u05DD Hello \u05E9\u05DC\u05D5\u05DD"
	runs := splitIntoBidiRuns(text, DirectionRTL, UnicodeBidiNormal)

	assert.GreaterOrEqual(t, len(runs), 2, "expected at least 2 runs")

	hasLTR := false
	hasRTL := false
	for _, run := range runs {
		if run.direction == DirectionLTR {
			hasLTR = true
		}
		if run.direction == DirectionRTL {
			hasRTL = true
		}
	}
	assert.True(t, hasLTR, "expected LTR run for English text")
	assert.True(t, hasRTL, "expected RTL run for Hebrew text")
}

func TestSplitIntoBidiRuns_RunTextCoversOriginal(t *testing.T) {
	text := "Hello \u05E9\u05DC\u05D5\u05DD World"
	runs := splitIntoBidiRuns(text, DirectionLTR, UnicodeBidiNormal)

	reconstructed := ""
	for _, run := range runs {
		reconstructed += run.text
	}

	for _, run := range runs {
		assert.Equal(t, text[run.start:run.end], run.text,
			"run text should match the slice of original text at its byte offsets")
	}
}

func TestSplitIntoBidiRuns_Isolate(t *testing.T) {

	text := "Hello"
	runs := splitIntoBidiRuns(text, DirectionLTR, UnicodeBidiIsolate)
	assert.Len(t, runs, 1)
	assert.Equal(t, text, runs[0].text)
	assert.Equal(t, DirectionLTR, runs[0].direction)
}

func TestSplitIntoBidiRuns_IsolateOverride(t *testing.T) {

	text := "Hello"
	runs := splitIntoBidiRuns(text, DirectionRTL, UnicodeBidiIsolateOverride)
	assert.Len(t, runs, 1)
	assert.Equal(t, text, runs[0].text)
	assert.Equal(t, DirectionRTL, runs[0].direction)
}
