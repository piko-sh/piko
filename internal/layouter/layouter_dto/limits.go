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
	"fmt"
)

const (
	// defaultMaxRawHTMLBytes is the default ceiling on the total raw HTML (p-html and raw
	// HTML nodes) expanded during one layout.
	defaultMaxRawHTMLBytes = 4 << 20

	// defaultMaxNestingDepth is the default ceiling on document nesting depth, matching the
	// depth at which browser HTML parsers stop nesting elements.
	defaultMaxNestingDepth = 512

	// defaultMaxBoxNodes is the default ceiling on the number of document nodes and layout
	// boxes one layout may hold.
	defaultMaxBoxNodes = 200_000

	// defaultMaxColspan is the default ceiling on a table cell's colspan, matching the HTML
	// standard's clamp.
	defaultMaxColspan = 1000

	// defaultMaxRowspan is the default ceiling on a table cell's rowspan, matching the HTML
	// standard's clamp.
	defaultMaxRowspan = 65534

	// defaultMaxTableColumns is the default ceiling on the number of columns in one table.
	defaultMaxTableColumns = 10_000

	// defaultMaxGridTracks is the default ceiling on the number of tracks on one axis of a
	// grid container.
	defaultMaxGridTracks = 10_000

	// defaultMaxGridCells is the default ceiling on the grid cells examined while placing
	// grid items across one layout.
	defaultMaxGridCells = 10_000_000

	// defaultMaxRepeatCount is the default ceiling on the integer count of a CSS repeat()
	// track function.
	defaultMaxRepeatCount = 10_000

	// defaultMaxPages is the default ceiling on the number of pages one layout may produce.
	defaultMaxPages = 10_000
)

var (
	// ErrLayoutLimitExceeded is the parent of every layout limit error, so callers can
	// detect any breached limit with a single errors.Is check.
	ErrLayoutLimitExceeded = errors.New("layout limit exceeded")

	// ErrRawHTMLTooLarge is returned when the raw HTML expanded during a layout exceeds
	// MaxRawHTMLBytes.
	ErrRawHTMLTooLarge = fmt.Errorf("%w: raw HTML too large", ErrLayoutLimitExceeded)

	// ErrNestingTooDeep is returned when the document nests deeper than MaxNestingDepth.
	ErrNestingTooDeep = fmt.Errorf("%w: document nesting too deep", ErrLayoutLimitExceeded)

	// ErrTooManyBoxes is returned when a layout holds more nodes or boxes than MaxBoxNodes.
	ErrTooManyBoxes = fmt.Errorf("%w: too many layout boxes", ErrLayoutLimitExceeded)

	// ErrTooManyTableColumns is returned when a table has more columns than MaxTableColumns.
	ErrTooManyTableColumns = fmt.Errorf("%w: too many table columns", ErrLayoutLimitExceeded)

	// ErrTooManyGridTracks is returned when a grid axis has more tracks than MaxGridTracks.
	ErrTooManyGridTracks = fmt.Errorf("%w: too many grid tracks", ErrLayoutLimitExceeded)

	// ErrGridTooLarge is returned when grid item placement examines more cells than
	// MaxGridCells.
	ErrGridTooLarge = fmt.Errorf("%w: grid placement too large", ErrLayoutLimitExceeded)

	// ErrRepeatCountTooLarge is returned when a CSS repeat() count exceeds MaxRepeatCount.
	ErrRepeatCountTooLarge = fmt.Errorf("%w: repeat() count too large", ErrLayoutLimitExceeded)

	// ErrTooManyPages is returned when pagination produces more pages than MaxPages.
	ErrTooManyPages = fmt.Errorf("%w: too many pages", ErrLayoutLimitExceeded)
)

// LayoutLimits bounds the work a single layout may perform, guarding against documents
// (for example raw HTML from p-html) crafted to exhaust memory or CPU. A zero field falls
// back to the high built-in default, so callers set only the limits they wish to tighten.
type LayoutLimits struct {
	// MaxRawHTMLBytes is the maximum total size of raw HTML expanded during one layout. Zero
	// uses the default.
	MaxRawHTMLBytes int

	// MaxNestingDepth is the maximum element nesting depth of the document. Zero uses the
	// default.
	MaxNestingDepth int

	// MaxBoxNodes is the maximum number of document nodes and layout boxes one layout may
	// hold, including boxes cloned for repeated headers, footers and fixed elements. Zero
	// uses the default.
	MaxBoxNodes int

	// MaxColspan is the value a table cell's colspan is clamped to. Zero uses the default.
	MaxColspan int

	// MaxRowspan is the value a table cell's rowspan is clamped to. Zero uses the default.
	MaxRowspan int

	// MaxTableColumns is the maximum number of columns in one table. Zero uses the default.
	MaxTableColumns int

	// MaxGridTracks is the maximum number of tracks on one axis of a grid container,
	// including tracks created by repeat(), auto-fill and implicit placement. Zero uses the
	// default.
	MaxGridTracks int

	// MaxGridCells is the maximum total number of grid cells examined while placing grid
	// items across one layout. Zero uses the default.
	MaxGridCells int

	// MaxRepeatCount is the maximum integer count of a CSS repeat() track function. Zero
	// uses the default.
	MaxRepeatCount int

	// MaxPages is the maximum number of pages one layout may produce. Zero uses the default.
	MaxPages int
}

// Resolved returns the effective limits, substituting the built-in default for every
// unset (non-positive) field.
//
// Returns LayoutLimits which holds a positive value in every field.
func (l LayoutLimits) Resolved() LayoutLimits {
	return l.WithFallback(defaultLayoutLimits())
}

// WithFallback returns the limits with every unset (non-positive) field taken from
// fallback, letting a per-render override sit on top of a service-wide configuration.
//
// Takes fallback (LayoutLimits) which supplies the values for unset fields.
//
// Returns LayoutLimits which holds the merged limits.
func (l LayoutLimits) WithFallback(fallback LayoutLimits) LayoutLimits {
	return LayoutLimits{
		MaxRawHTMLBytes: firstPositive(l.MaxRawHTMLBytes, fallback.MaxRawHTMLBytes),
		MaxNestingDepth: firstPositive(l.MaxNestingDepth, fallback.MaxNestingDepth),
		MaxBoxNodes:     firstPositive(l.MaxBoxNodes, fallback.MaxBoxNodes),
		MaxColspan:      firstPositive(l.MaxColspan, fallback.MaxColspan),
		MaxRowspan:      firstPositive(l.MaxRowspan, fallback.MaxRowspan),
		MaxTableColumns: firstPositive(l.MaxTableColumns, fallback.MaxTableColumns),
		MaxGridTracks:   firstPositive(l.MaxGridTracks, fallback.MaxGridTracks),
		MaxGridCells:    firstPositive(l.MaxGridCells, fallback.MaxGridCells),
		MaxRepeatCount:  firstPositive(l.MaxRepeatCount, fallback.MaxRepeatCount),
		MaxPages:        firstPositive(l.MaxPages, fallback.MaxPages),
	}
}

// defaultLayoutLimits returns the built-in limits applied to unset fields.
//
// Returns LayoutLimits which holds the default for every field.
func defaultLayoutLimits() LayoutLimits {
	return LayoutLimits{
		MaxRawHTMLBytes: defaultMaxRawHTMLBytes,
		MaxNestingDepth: defaultMaxNestingDepth,
		MaxBoxNodes:     defaultMaxBoxNodes,
		MaxColspan:      defaultMaxColspan,
		MaxRowspan:      defaultMaxRowspan,
		MaxTableColumns: defaultMaxTableColumns,
		MaxGridTracks:   defaultMaxGridTracks,
		MaxGridCells:    defaultMaxGridCells,
		MaxRepeatCount:  defaultMaxRepeatCount,
		MaxPages:        defaultMaxPages,
	}
}

// firstPositive returns value when it is positive, otherwise fallback.
//
// Takes value (int) which is the preferred value.
// Takes fallback (int) which is used when value is not positive.
//
// Returns int which is the selected value.
func firstPositive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
