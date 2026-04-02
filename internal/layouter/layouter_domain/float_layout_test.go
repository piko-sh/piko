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
	"testing"

	"github.com/stretchr/testify/assert"
)

const floatEpsilon = 0.001

func TestPlaceLeftFloat(t *testing.T) {
	tests := []struct {
		name           string
		existing       []floatEntry
		cursorY        float64
		width          float64
		height         float64
		containerX     float64
		containerWidth float64
		expectedX      float64
		expectedY      float64
	}{
		{
			name:           "empty context places float at container origin",
			existing:       nil,
			cursorY:        0,
			width:          100,
			height:         50,
			containerX:     0,
			containerWidth: 500,
			expectedX:      0,
			expectedY:      0,
		},
		{
			name: "second left float placed beside first at same Y",
			existing: []floatEntry{
				{x: 0, y: 0, width: 100, height: 50},
			},
			cursorY:        0,
			width:          100,
			height:         50,
			containerX:     0,
			containerWidth: 500,
			expectedX:      100,
			expectedY:      0,
		},
		{
			name: "float wraps below existing when it would exceed container width",
			existing: []floatEntry{
				{x: 0, y: 0, width: 400, height: 50},
			},
			cursorY:        0,
			width:          200,
			height:         50,
			containerX:     0,
			containerWidth: 500,
			expectedX:      0,
			expectedY:      50,
		},
		{
			name: "float at cursor Y below existing floats is placed at container origin",
			existing: []floatEntry{
				{x: 0, y: 0, width: 100, height: 50},
			},
			cursorY:        100,
			width:          150,
			height:         40,
			containerX:     0,
			containerWidth: 500,
			expectedX:      0,
			expectedY:      100,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &floatContext{
				leftFloats: append([]floatEntry{}, tc.existing...),
			}

			got_x, got_y := ctx.placeLeftFloat(
				tc.cursorY, tc.width, tc.height,
				tc.containerX, tc.containerWidth,
			)

			assert.InDelta(t, tc.expectedX, got_x, floatEpsilon, "floatX")
			assert.InDelta(t, tc.expectedY, got_y, floatEpsilon, "floatY")
		})
	}
}

func TestPlaceRightFloat(t *testing.T) {
	tests := []struct {
		name           string
		existing       []floatEntry
		cursorY        float64
		width          float64
		height         float64
		containerX     float64
		containerWidth float64
		expectedX      float64
		expectedY      float64
	}{
		{
			name:           "empty context places float right-aligned",
			existing:       nil,
			cursorY:        0,
			width:          100,
			height:         50,
			containerX:     0,
			containerWidth: 500,
			expectedX:      400,
			expectedY:      0,
		},
		{
			name: "second right float placed beside first",
			existing: []floatEntry{
				{x: 400, y: 0, width: 100, height: 50},
			},
			cursorY:        0,
			width:          100,
			height:         50,
			containerX:     0,
			containerWidth: 500,
			expectedX:      300,
			expectedY:      0,
		},
		{
			name: "float wraps below existing when it would extend past left edge",
			existing: []floatEntry{
				{x: 100, y: 0, width: 400, height: 50},
			},
			cursorY:        0,
			width:          200,
			height:         50,
			containerX:     0,
			containerWidth: 500,
			expectedX:      300,
			expectedY:      50,
		},
		{
			name: "float at cursor Y below existing floats is right-aligned",
			existing: []floatEntry{
				{x: 400, y: 0, width: 100, height: 50},
			},
			cursorY:        100,
			width:          150,
			height:         40,
			containerX:     0,
			containerWidth: 500,
			expectedX:      350,
			expectedY:      100,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &floatContext{
				rightFloats: append([]floatEntry{}, tc.existing...),
			}

			got_x, got_y := ctx.placeRightFloat(
				tc.cursorY, tc.width, tc.height,
				tc.containerX, tc.containerWidth,
			)

			assert.InDelta(t, tc.expectedX, got_x, floatEpsilon, "floatX")
			assert.InDelta(t, tc.expectedY, got_y, floatEpsilon, "floatY")
		})
	}
}

func TestClearLeftY(t *testing.T) {
	tests := []struct {
		name        string
		leftFloats  []floatEntry
		rightFloats []floatEntry
		expected    float64
	}{
		{
			name:     "empty context returns zero",
			expected: 0,
		},
		{
			name: "single left float returns its bottom edge",
			leftFloats: []floatEntry{
				{y: 10, height: 50},
			},
			expected: 60,
		},
		{
			name: "multiple left floats returns max bottom edge",
			leftFloats: []floatEntry{
				{y: 0, height: 30},
				{y: 10, height: 50},
				{y: 5, height: 20},
			},
			expected: 60,
		},
		{
			name: "no left floats but has right floats returns zero",
			rightFloats: []floatEntry{
				{y: 0, height: 100},
			},
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &floatContext{
				leftFloats:  tc.leftFloats,
				rightFloats: tc.rightFloats,
			}

			assert.InDelta(t, tc.expected, ctx.clearLeftY(), floatEpsilon)
		})
	}
}

func TestClearRightY(t *testing.T) {
	tests := []struct {
		name        string
		rightFloats []floatEntry
		expected    float64
	}{
		{
			name:     "empty context returns zero",
			expected: 0,
		},
		{
			name: "single right float returns its bottom edge",
			rightFloats: []floatEntry{
				{y: 20, height: 80},
			},
			expected: 100,
		},
		{
			name: "multiple right floats returns max bottom edge",
			rightFloats: []floatEntry{
				{y: 0, height: 40},
				{y: 10, height: 70},
				{y: 50, height: 10},
			},
			expected: 80,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &floatContext{
				rightFloats: tc.rightFloats,
			}

			assert.InDelta(t, tc.expected, ctx.clearRightY(), floatEpsilon)
		})
	}
}

func TestClearBothY(t *testing.T) {
	tests := []struct {
		name        string
		leftFloats  []floatEntry
		rightFloats []floatEntry
		expected    float64
	}{
		{
			name:     "empty context returns zero",
			expected: 0,
		},
		{
			name: "left float only returns left max",
			leftFloats: []floatEntry{
				{y: 0, height: 50},
			},
			expected: 50,
		},
		{
			name: "right float only returns right max",
			rightFloats: []floatEntry{
				{y: 0, height: 70},
			},
			expected: 70,
		},
		{
			name: "both sides returns overall max",
			leftFloats: []floatEntry{
				{y: 0, height: 50},
			},
			rightFloats: []floatEntry{
				{y: 0, height: 70},
			},
			expected: 70,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &floatContext{
				leftFloats:  tc.leftFloats,
				rightFloats: tc.rightFloats,
			}

			assert.InDelta(t, tc.expected, ctx.clearBothY(), floatEpsilon)
		})
	}
}

func TestClearY(t *testing.T) {

	shared_ctx := &floatContext{
		leftFloats: []floatEntry{
			{y: 0, height: 50},
		},
		rightFloats: []floatEntry{
			{y: 0, height: 70},
		},
	}

	tests := []struct {
		name      string
		clearType ClearType
		expected  float64
	}{
		{
			name:      "ClearNone returns zero",
			clearType: ClearNone,
			expected:  0,
		},
		{
			name:      "ClearLeft returns clearLeftY",
			clearType: ClearLeft,
			expected:  50,
		},
		{
			name:      "ClearRight returns clearRightY",
			clearType: ClearRight,
			expected:  70,
		},
		{
			name:      "ClearBoth returns clearBothY",
			clearType: ClearBoth,
			expected:  70,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.InDelta(t, tc.expected, shared_ctx.clearY(tc.clearType), floatEpsilon)
		})
	}
}

func TestAvailableWidthAtY(t *testing.T) {
	tests := []struct {
		name           string
		leftFloats     []floatEntry
		rightFloats    []floatEntry
		y              float64
		height         float64
		containerX     float64
		containerWidth float64
		expected       float64
	}{
		{
			name:           "empty context returns full container width",
			y:              0,
			height:         1,
			containerX:     0,
			containerWidth: 500,
			expected:       500,
		},
		{
			name: "left float overlapping Y narrows from the left",
			leftFloats: []floatEntry{
				{x: 0, y: 0, width: 100, height: 50},
			},
			y:              25,
			height:         1,
			containerX:     0,
			containerWidth: 500,
			expected:       400,
		},
		{
			name: "right float overlapping Y narrows from the right",
			rightFloats: []floatEntry{
				{x: 400, y: 0, width: 100, height: 50},
			},
			y:              25,
			height:         1,
			containerX:     0,
			containerWidth: 500,
			expected:       400,
		},
		{
			name: "both left and right floats overlapping Y narrow from both sides",
			leftFloats: []floatEntry{
				{x: 0, y: 0, width: 100, height: 50},
			},
			rightFloats: []floatEntry{
				{x: 400, y: 0, width: 100, height: 50},
			},
			y:              25,
			height:         1,
			containerX:     0,
			containerWidth: 500,
			expected:       300,
		},
		{
			name: "Y below all floats returns full container width",
			leftFloats: []floatEntry{
				{x: 0, y: 0, width: 100, height: 50},
			},
			rightFloats: []floatEntry{
				{x: 400, y: 0, width: 100, height: 50},
			},
			y:              100,
			height:         1,
			containerX:     0,
			containerWidth: 500,
			expected:       500,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &floatContext{
				leftFloats:  tc.leftFloats,
				rightFloats: tc.rightFloats,
			}

			got := ctx.availableWidthAtY(
				tc.y, tc.height,
				tc.containerX, tc.containerWidth,
			)

			assert.InDelta(t, tc.expected, got, floatEpsilon)
		})
	}
}

func TestLeftOffsetAtY(t *testing.T) {
	tests := []struct {
		name       string
		leftFloats []floatEntry
		y          float64
		height     float64
		containerX float64
		expected   float64
	}{
		{
			name:       "empty context returns containerX",
			y:          0,
			height:     1,
			containerX: 0,
			expected:   0,
		},
		{
			name: "left float overlapping Y returns right edge of float",
			leftFloats: []floatEntry{
				{x: 0, y: 0, width: 100, height: 50},
			},
			y:          25,
			height:     1,
			containerX: 0,
			expected:   100,
		},
		{
			name: "Y below float returns containerX",
			leftFloats: []floatEntry{
				{x: 0, y: 0, width: 100, height: 50},
			},
			y:          100,
			height:     1,
			containerX: 0,
			expected:   0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &floatContext{
				leftFloats: tc.leftFloats,
			}

			got := ctx.leftOffsetAtY(tc.y, tc.height, tc.containerX)

			assert.InDelta(t, tc.expected, got, floatEpsilon)
		})
	}
}
