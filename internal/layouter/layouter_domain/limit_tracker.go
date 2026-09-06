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
	"fmt"

	"piko.sh/piko/internal/layouter/layouter_dto"
)

var (
	// defaultLayoutLimits holds the built-in limits, used where no tracker is available such
	// as intrinsic size measurement.
	defaultLayoutLimits = layouter_dto.LayoutLimits{}.Resolved()
)

// LimitTracker enforces the layout limits of a single layout and records the first limit
// the document breaches.
//
// One tracker is shared by every stage of a layout (style resolution, box tree
// construction, layout and pagination) so the box and grid budgets cover the whole
// document. The stages stop doing work once a limit is breached, and each stage returns
// the recorded error so a truncated layout is never mistaken for a finished one.
//
// A tracker is used by a single layout goroutine and is not safe for concurrent use. A
// nil tracker applies the default limits but cannot record a breach; the stage entry
// points substitute a fresh tracker when given nil so their errors are still reported.
type LimitTracker struct {
	// err holds the first limit breach, or nil when every limit has held.
	err error

	// limits holds the resolved limits with every field positive.
	limits layouter_dto.LayoutLimits

	// boxCount holds the number of layout boxes counted so far.
	boxCount int

	// gridCells holds the number of grid cells examined so far during grid placement.
	gridCells int
}

// NewLimitTracker creates a tracker enforcing the given limits, substituting the built-in
// default for every unset field.
//
// Takes limits (layouter_dto.LayoutLimits) which holds the configured limits.
//
// Returns *LimitTracker which is ready to track a single layout.
func NewLimitTracker(limits layouter_dto.LayoutLimits) *LimitTracker {
	return &LimitTracker{
		err:       nil,
		limits:    limits.Resolved(),
		boxCount:  0,
		gridCells: 0,
	}
}

// Limits returns the resolved limits the tracker enforces, or the defaults for a nil
// tracker.
//
// Returns layouter_dto.LayoutLimits which holds a positive value in every field.
func (t *LimitTracker) Limits() layouter_dto.LayoutLimits {
	if t == nil {
		return defaultLayoutLimits
	}
	return t.limits
}

// Err returns the first limit breach recorded by the tracker.
//
// Returns error which wraps a layouter_dto sentinel, or nil when no limit was breached.
func (t *LimitTracker) Err() error {
	if t == nil {
		return nil
	}
	return t.err
}

// fail records err as the breach unless one has already been recorded.
//
// Takes err (error) which describes the breach.
func (t *LimitTracker) fail(err error) {
	if t == nil || t.err != nil {
		return
	}
	t.err = err
}

// failed reports whether a breach has been recorded, so long-running stages can stop
// early.
//
// Returns bool which is true once any limit has been breached.
func (t *LimitTracker) failed() bool {
	return t != nil && t.err != nil
}

// addBoxes counts newly created layout boxes against MaxBoxNodes.
//
// Takes count (int) which is the number of boxes created.
//
// Returns bool which is false when the budget is exhausted and the breach was recorded.
func (t *LimitTracker) addBoxes(count int) bool {
	if t == nil {
		return true
	}
	t.boxCount += count
	if t.boxCount > t.limits.MaxBoxNodes {
		t.fail(fmt.Errorf("layout holds more than %d boxes: %w", t.limits.MaxBoxNodes, layouter_dto.ErrTooManyBoxes))
		return false
	}
	return true
}

// addGridCells counts grid cells examined during grid item placement against
// MaxGridCells.
//
// Takes count (int) which is the number of cells examined.
//
// Returns bool which is false when the budget is exhausted and the breach was recorded.
func (t *LimitTracker) addGridCells(count int) bool {
	if t == nil {
		return true
	}
	t.gridCells += count
	if t.gridCells > t.limits.MaxGridCells {
		t.fail(fmt.Errorf("grid placement examined more than %d cells: %w", t.limits.MaxGridCells, layouter_dto.ErrGridTooLarge))
		return false
	}
	return true
}

// trackerOrDefault returns tracker, or a fresh tracker with the default limits when
// tracker is nil, so a stage entry point can always report a breach.
//
// Takes tracker (*LimitTracker) which is the caller's tracker, or nil.
//
// Returns *LimitTracker which is never nil.
func trackerOrDefault(tracker *LimitTracker) *LimitTracker {
	if tracker != nil {
		return tracker
	}
	return NewLimitTracker(layouter_dto.LayoutLimits{})
}

// layoutStageError reports a cancelled context or breached layout limit that stopped a
// layout stage early.
//
// Takes limits (*LimitTracker) which holds any recorded limit breach.
//
// Returns error which wraps the cancellation or breach, or nil when layout completed.
func layoutStageError(ctx context.Context, limits *LimitTracker) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("box-tree layout cancelled: %w", err)
	}
	if err := limits.Err(); err != nil {
		return fmt.Errorf("box-tree layout stopped: %w", err)
	}
	return nil
}
