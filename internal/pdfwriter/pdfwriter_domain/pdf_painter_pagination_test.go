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
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/layouter/layouter_domain"
)

func TestPaintPageBoxes_ChildOnLaterPageNotDuplicated(t *testing.T) {
	t.Parallel()

	const childFill = "0.20 0.40 0.80 rg"

	child := newLayoutBox().
		WithContentRect(0, 900, 595, 100).
		WithBoxType(layouter_domain.BoxBlock).
		WithBackground(testColour(0.2, 0.4, 0.8, 1.0)).
		WithPageIndex(1).
		Build()
	parent := newLayoutBox().
		WithContentRect(0, 0, 595, 1000).
		WithBoxType(layouter_domain.BoxBlock).
		WithPageIndex(0).
		WithChildren(child).
		Build()
	root := newLayoutBox().
		WithContentRect(0, 0, 595, 1684).
		WithBoxType(layouter_domain.BoxBlock).
		WithPageIndex(0).
		WithChildren(parent).
		Build()

	painter := newPainterWithDefaults()
	painter.setPageMargins(0, 0, 842)
	ctx := context.Background()

	page0 := &ContentStream{}
	painter.basePageYOffset = 0
	painter.pageYOffset = 0
	painter.paintPageBoxes(ctx, page0, root, 0)

	page1 := &ContentStream{}
	painter.basePageYOffset = painter.pageStride()
	painter.pageYOffset = painter.basePageYOffset
	painter.paintPageBoxes(ctx, page1, root, 1)

	assert.NotContains(t, page0.String(), childFill, "page 0 stream must NOT contain the page-1 child's fill (duplication bug)")
	assert.Contains(t, page1.String(), childFill, "page 1 stream must contain the page-1 child's fill")
}

func TestPaintPageBoxes_SamePageChildStillPainted(t *testing.T) {
	t.Parallel()

	const childFill = "0.90 0.10 0.10 rg"

	child := newLayoutBox().
		WithContentRect(0, 100, 595, 100).
		WithBoxType(layouter_domain.BoxBlock).
		WithBackground(testColour(0.9, 0.1, 0.1, 1.0)).
		WithPageIndex(0).
		Build()
	parent := newLayoutBox().
		WithContentRect(0, 0, 595, 842).
		WithBoxType(layouter_domain.BoxBlock).
		WithPageIndex(0).
		WithChildren(child).
		Build()

	painter := newPainterWithDefaults()
	painter.setPageMargins(0, 0, 842)

	page0 := &ContentStream{}
	painter.basePageYOffset = 0
	painter.pageYOffset = 0
	painter.paintPageBoxes(context.Background(), page0, parent, 0)

	assert.Contains(t, page0.String(), childFill, "page 0 stream must contain the same-page child's fill")
}

func TestSetPageMargins_MarginsLargerThanPageHeightDoNotProduceNegativeStride(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	painter.setPageMargins(10, 1000, -200)

	stride := painter.pageStride()
	require.Greater(t, stride, 0.0, "page stride must be positive")

	for page := range 3 {
		offset := float64(page) * stride
		assert.GreaterOrEqual(t, offset, 0.0, "base page Y offset for page %d is negative", page)
	}

	painter.setPageMargins(10, 10, 0.25)
	assert.GreaterOrEqual(t, painter.pageStride(), minContentPageHeight, "tiny positive usable height must clamp to >= %.2f", minContentPageHeight)
}

func TestCollectPageEntryBoxes(t *testing.T) {
	t.Parallel()

	grandchild := newLayoutBox().WithBoxType(layouter_domain.BoxBlock).WithPageIndex(0).Build()
	child := newLayoutBox().WithBoxType(layouter_domain.BoxBlock).WithPageIndex(1).WithChildren(grandchild).Build()
	hiddenChild := newLayoutBox().WithBoxType(layouter_domain.BoxBlock).WithPageIndex(1).Build()
	hidden := newLayoutBox().WithBoxType(layouter_domain.BoxNone).WithPageIndex(2).WithChildren(hiddenChild).Build()
	outOfRange := newLayoutBox().WithBoxType(layouter_domain.BoxBlock).WithPageIndex(7).Build()
	sibling := newLayoutBox().WithBoxType(layouter_domain.BoxBlock).WithPageIndex(1).Build()
	root := newLayoutBox().WithBoxType(layouter_domain.BoxBlock).WithPageIndex(0).
		WithChildren(child, hidden, outOfRange, sibling).Build()

	entries := collectPageEntryBoxes(root, 3)

	require.Len(t, entries, 3)
	assert.Equal(t, []*layouter_domain.LayoutBox{root}, entries[0], "boxes under an entry on the same page are painted within it")
	assert.Equal(t, []*layouter_domain.LayoutBox{child, sibling}, entries[1], "page 1 starts at each box whose ancestors are on other pages")
	assert.Empty(t, entries[2], "display:none subtrees and out-of-range pages are never painted")
}

func TestRenderPageStreams_MatchesWholeTreeWalk(t *testing.T) {
	t.Parallel()

	grandchild := newLayoutBox().WithContentRect(10, 40, 100, 20).WithBoxType(layouter_domain.BoxBlock).
		WithBackground(testColour(0.1, 0.2, 0.3, 1)).WithPageIndex(0).Build()
	child := newLayoutBox().WithContentRect(0, 900, 595, 100).WithBoxType(layouter_domain.BoxBlock).
		WithBackground(testColour(0.4, 0.5, 0.6, 1)).WithPageIndex(1).WithChildren(grandchild).Build()
	other := newLayoutBox().WithContentRect(0, 1700, 595, 100).WithBoxType(layouter_domain.BoxBlock).
		WithBackground(testColour(0.7, 0.8, 0.9, 1)).WithPageIndex(2).Build()
	root := newLayoutBox().WithContentRect(0, 0, 595, 2526).WithBoxType(layouter_domain.BoxBlock).
		WithBackground(testColour(1, 1, 1, 1)).WithPageIndex(0).WithChildren(child, other).Build()

	painter := newPainterWithDefaults()
	painter.setPageMargins(10, 20, 842)
	streams, err := painter.renderPageStreams(context.Background(), root, 3, "")
	require.NoError(t, err)

	reference := newPainterWithDefaults()
	reference.setPageMargins(10, 20, 842)
	for page := range 3 {
		expected := &ContentStream{}
		reference.basePageYOffset = float64(page) * reference.pageStride()
		reference.pageYOffset = reference.basePageYOffset
		expected.builder.WriteString("q\n1 0 0 1 10 -20 cm\n")
		reference.paintPageBoxes(context.Background(), expected, root, page)
		expected.builder.WriteString("Q\n")

		assert.Equal(t, expected.String(), streams[page].String(), "page %d", page)
	}
}

func TestRenderPageStreams_CancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("stopped by test"))

	root := newLayoutBox().WithBoxType(layouter_domain.BoxBlock).Build()
	streams, err := newPainterWithDefaults().renderPageStreams(ctx, root, 2, "")

	assert.Nil(t, streams)
	assert.ErrorIs(t, err, context.Canceled)
}
