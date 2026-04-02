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

func TestObjectFitType_String(t *testing.T) {
	testCases := []struct {
		value    ObjectFitType
		expected string
	}{
		{ObjectFitFill, "Fill"},
		{ObjectFitContain, "Contain"},
		{ObjectFitCover, "Cover"},
		{ObjectFitNone, "None"},
		{ObjectFitScaleDown, "ScaleDown"},
		{ObjectFitType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == ObjectFitType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestBackgroundImageType_String(t *testing.T) {
	testCases := []struct {
		value    BackgroundImageType
		expected string
	}{
		{BackgroundImageNone, "none"},
		{BackgroundImageURL, "url"},
		{BackgroundImageLinearGradient, "linear-gradient"},
		{BackgroundImageRadialGradient, "radial-gradient"},
		{BackgroundImageType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == BackgroundImageType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestBorderImageRepeatType_String(t *testing.T) {
	testCases := []struct {
		value    BorderImageRepeatType
		expected string
	}{
		{BorderImageRepeatStretch, "stretch"},
		{BorderImageRepeatRepeat, "repeat"},
		{BorderImageRepeatRound, "round"},
		{BorderImageRepeatSpace, "space"},
		{BorderImageRepeatType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == BorderImageRepeatType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}
