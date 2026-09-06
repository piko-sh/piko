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
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/layouter/layouter_dto"
)

func TestNewLimitTracker_ResolvesDefaults(t *testing.T) {
	t.Parallel()

	tracker := NewLimitTracker(layouter_dto.LayoutLimits{MaxPages: 3})

	assert.Equal(t, 3, tracker.Limits().MaxPages)
	assert.Equal(t, defaultLayoutLimits.MaxBoxNodes, tracker.Limits().MaxBoxNodes)
	assert.NoError(t, tracker.Err())
	assert.False(t, tracker.failed())
}

func TestLimitTracker_NilTrackerUsesDefaultsWithoutRecording(t *testing.T) {
	t.Parallel()

	var tracker *LimitTracker

	assert.Equal(t, defaultLayoutLimits, tracker.Limits())
	assert.NoError(t, tracker.Err())
	tracker.fail(layouter_dto.ErrTooManyPages)
	assert.False(t, tracker.failed())
	assert.True(t, tracker.addBoxes(1_000_000_000))
	assert.True(t, tracker.addGridCells(1_000_000_000))
}

func TestLimitTracker_Budgets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		add     func(*LimitTracker, int) bool
		wantErr error
		name    string
		limits  layouter_dto.LayoutLimits
		within  int
		breach  int
	}{
		{
			name:    "box budget",
			limits:  layouter_dto.LayoutLimits{MaxBoxNodes: 10},
			add:     (*LimitTracker).addBoxes,
			within:  10,
			breach:  1,
			wantErr: layouter_dto.ErrTooManyBoxes,
		},
		{
			name:    "grid cell budget",
			limits:  layouter_dto.LayoutLimits{MaxGridCells: 100},
			add:     (*LimitTracker).addGridCells,
			within:  60,
			breach:  41,
			wantErr: layouter_dto.ErrGridTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tracker := NewLimitTracker(tt.limits)
			require.True(t, tt.add(tracker, tt.within), "a count within the budget is accepted")
			require.NoError(t, tracker.Err())

			assert.False(t, tt.add(tracker, tt.breach), "a count beyond the budget is refused")
			assert.ErrorIs(t, tracker.Err(), tt.wantErr)
			assert.ErrorIs(t, tracker.Err(), layouter_dto.ErrLayoutLimitExceeded)
			assert.True(t, tracker.failed())
		})
	}
}

func TestLimitTracker_KeepsFirstBreach(t *testing.T) {
	t.Parallel()

	tracker := NewLimitTracker(layouter_dto.LayoutLimits{})
	tracker.fail(layouter_dto.ErrTooManyPages)
	tracker.fail(layouter_dto.ErrTooManyBoxes)

	assert.ErrorIs(t, tracker.Err(), layouter_dto.ErrTooManyPages)
	assert.False(t, errors.Is(tracker.Err(), layouter_dto.ErrTooManyBoxes))
}

func TestTrackerOrDefault(t *testing.T) {
	t.Parallel()

	tracker := NewLimitTracker(layouter_dto.LayoutLimits{MaxPages: 2})
	assert.Same(t, tracker, trackerOrDefault(tracker))

	substituted := trackerOrDefault(nil)
	require.NotNil(t, substituted)
	assert.Equal(t, defaultLayoutLimits, substituted.Limits())
}

func TestLayoutStageError(t *testing.T) {
	t.Parallel()

	t.Run("completed stage", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, layoutStageError(context.Background(), NewLimitTracker(layouter_dto.LayoutLimits{})))
	})

	t.Run("cancelled context", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(errors.New("stopped by test"))
		assert.ErrorIs(t, layoutStageError(ctx, nil), context.Canceled)
	})

	t.Run("breached limit", func(t *testing.T) {
		t.Parallel()
		tracker := NewLimitTracker(layouter_dto.LayoutLimits{})
		tracker.fail(layouter_dto.ErrGridTooLarge)
		assert.ErrorIs(t, layoutStageError(context.Background(), tracker), layouter_dto.ErrGridTooLarge)
	})
}

func TestLayoutBoxTree_ReportsLimitBreach(t *testing.T) {
	t.Parallel()

	root := makeRoot(400)
	grid := newGridContainerForTest(root, map[string]string{
		"display":               "grid",
		"grid-template-columns": "repeat(auto-fill, 0.001px)",
	})
	root.Children = []*LayoutBox{grid}

	tracker := NewLimitTracker(layouter_dto.LayoutLimits{MaxGridTracks: 100})
	_, err := LayoutBoxTree(context.Background(), root, &mockFontMetrics{}, tracker)

	require.Error(t, err)
	assert.ErrorIs(t, err, layouter_dto.ErrTooManyGridTracks)
}

func newGridContainerForTest(parent *LayoutBox, properties map[string]string) *LayoutBox {
	style := ResolveStyle(properties, nil, DefaultResolutionContext())
	style.Display = DisplayGrid
	grid := &LayoutBox{Type: BoxBlock, Style: style, Parent: parent}
	item := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: grid}
	item.Style.Display = DisplayBlock
	item.Style.Height = DimensionPt(10)
	grid.Children = []*LayoutBox{item}
	return grid
}
