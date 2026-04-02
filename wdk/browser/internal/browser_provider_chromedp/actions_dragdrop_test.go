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

package browser_provider_chromedp

import (
	"math"
	"testing"
)

func TestInterpolateDragPoint(t *testing.T) {
	testCases := []struct {
		name       string
		startX     float64
		startY     float64
		endX       float64
		endY       float64
		step       int
		totalSteps int
		expectedX  float64
		expectedY  float64
	}{
		{
			name:   "start to end in one step",
			startX: 0, startY: 0,
			endX: 100, endY: 200,
			step: 1, totalSteps: 1,
			expectedX: 100, expectedY: 200,
		},
		{
			name:   "midpoint of two steps",
			startX: 0, startY: 0,
			endX: 100, endY: 100,
			step: 1, totalSteps: 2,
			expectedX: 50, expectedY: 50,
		},
		{
			name:   "first of ten steps",
			startX: 0, startY: 0,
			endX: 100, endY: 100,
			step: 1, totalSteps: 10,
			expectedX: 10, expectedY: 10,
		},
		{
			name:   "last of ten steps",
			startX: 0, startY: 0,
			endX: 100, endY: 100,
			step: 10, totalSteps: 10,
			expectedX: 100, expectedY: 100,
		},
		{
			name:   "same start and end",
			startX: 50, startY: 75,
			endX: 50, endY: 75,
			step: 5, totalSteps: 10,
			expectedX: 50, expectedY: 75,
		},
		{
			name:   "negative direction",
			startX: 100, startY: 200,
			endX: 0, endY: 0,
			step: 5, totalSteps: 10,
			expectedX: 50, expectedY: 100,
		},
		{
			name:   "non-zero origin",
			startX: 10, startY: 20,
			endX: 110, endY: 120,
			step: 5, totalSteps: 10,
			expectedX: 60, expectedY: 70,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			x, y := interpolateDragPoint(tc.startX, tc.startY, tc.endX, tc.endY, tc.step, tc.totalSteps)
			if math.Abs(x-tc.expectedX) > 0.001 {
				t.Errorf("x = %f, expected %f", x, tc.expectedX)
			}
			if math.Abs(y-tc.expectedY) > 0.001 {
				t.Errorf("y = %f, expected %f", y, tc.expectedY)
			}
		})
	}
}

func TestDragStepConstants(t *testing.T) {
	if dragStepCount <= 0 {
		t.Errorf("dragStepCount should be positive, got %d", dragStepCount)
	}
	if dragStepInterval <= 0 {
		t.Errorf("dragStepInterval should be positive, got %v", dragStepInterval)
	}
}
