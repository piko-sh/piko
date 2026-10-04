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

package layouter_dto

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLayoutLimits_Resolved(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		limits LayoutLimits
		want   LayoutLimits
	}{
		{
			name:   "zero limits use every default",
			limits: LayoutLimits{},
			want:   defaultLayoutLimits(),
		},
		{
			name:   "negative fields use the default",
			limits: LayoutLimits{MaxPages: -1, MaxColspan: -5},
			want:   defaultLayoutLimits(),
		},
		{
			name: "positive fields are kept",
			limits: LayoutLimits{
				MaxRawHTMLBytes: 1,
				MaxNestingDepth: 2,
				MaxBoxNodes:     3,
				MaxColspan:      4,
				MaxRowspan:      5,
				MaxTableColumns: 6,
				MaxGridTracks:   7,
				MaxGridCells:    8,
				MaxRepeatCount:  9,
				MaxPages:        10,
			},
			want: LayoutLimits{
				MaxRawHTMLBytes: 1,
				MaxNestingDepth: 2,
				MaxBoxNodes:     3,
				MaxColspan:      4,
				MaxRowspan:      5,
				MaxTableColumns: 6,
				MaxGridTracks:   7,
				MaxGridCells:    8,
				MaxRepeatCount:  9,
				MaxPages:        10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.limits.Resolved())
		})
	}
}

func TestLayoutLimits_DefaultsAreGenerousAndStandard(t *testing.T) {
	t.Parallel()

	defaults := LayoutLimits{}.Resolved()
	assert.Equal(t, 1000, defaults.MaxColspan, "colspan default should match the HTML clamp")
	assert.Equal(t, 65534, defaults.MaxRowspan, "rowspan default should match the HTML clamp")
	assert.Equal(t, 512, defaults.MaxNestingDepth, "nesting default should match browser parsers")
	assert.Positive(t, defaults.MaxRawHTMLBytes)
	assert.Positive(t, defaults.MaxBoxNodes)
	assert.Positive(t, defaults.MaxTableColumns)
	assert.Positive(t, defaults.MaxGridTracks)
	assert.Positive(t, defaults.MaxGridCells)
	assert.Positive(t, defaults.MaxRepeatCount)
	assert.Positive(t, defaults.MaxPages)
}

func TestLayoutLimits_WithFallback(t *testing.T) {
	t.Parallel()

	override := LayoutLimits{MaxPages: 5, MaxColspan: 0, MaxGridTracks: -1}
	fallback := LayoutLimits{MaxPages: 50, MaxColspan: 20, MaxGridTracks: 30, MaxBoxNodes: 40}

	merged := override.WithFallback(fallback)

	assert.Equal(t, 5, merged.MaxPages, "a set override wins")
	assert.Equal(t, 20, merged.MaxColspan, "an unset override takes the fallback")
	assert.Equal(t, 30, merged.MaxGridTracks, "a negative override takes the fallback")
	assert.Equal(t, 40, merged.MaxBoxNodes, "a field set only in the fallback is kept")
	assert.Zero(t, merged.MaxRepeatCount, "a field unset in both stays unset")
	assert.Equal(t, defaultLayoutLimits().MaxRepeatCount, merged.Resolved().MaxRepeatCount)
}

func TestLayoutLimitErrors_WrapTheParentSentinel(t *testing.T) {
	t.Parallel()

	sentinels := []error{
		ErrRawHTMLTooLarge,
		ErrNestingTooDeep,
		ErrTooManyBoxes,
		ErrTooManyTableColumns,
		ErrTooManyGridTracks,
		ErrGridTooLarge,
		ErrRepeatCountTooLarge,
		ErrTooManyPages,
	}
	for index, sentinel := range sentinels {
		t.Run(sentinel.Error(), func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, sentinel, ErrLayoutLimitExceeded)
			for otherIndex, other := range sentinels {
				if otherIndex != index {
					assert.False(t, errors.Is(sentinel, other), "%v must not match %v", sentinel, other)
				}
			}
		})
	}
}

func TestPageConfig_ContentArea(t *testing.T) {
	t.Parallel()

	page := PageConfig{Width: 600, Height: 800, MarginTop: 10, MarginRight: 20, MarginBottom: 30, MarginLeft: 40}
	assert.InDelta(t, 540.0, page.ContentAreaWidth(), 1e-9)
	assert.InDelta(t, 760.0, page.ContentAreaHeight(), 1e-9)
}
