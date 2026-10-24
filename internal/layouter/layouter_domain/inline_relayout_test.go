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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func relayoutTestBlock(text string) (*LayoutBox, *LayoutBox) {
	style := DefaultComputedStyle()
	style.Display = DisplayBlock
	block := newLayoutBox(BoxBlock, &style, nil)
	run := newLayoutBox(BoxTextRun, &block.Style, block)
	run.Style.Display = DisplayInline
	run.Text = text
	block.Children = []*LayoutBox{run}
	return block, run
}

func segmentTexts(box *LayoutBox) []string {
	var texts []string
	for _, child := range box.Children {
		if child.Type == BoxTextRun {
			texts = append(texts, child.Text)
		}
	}
	return texts
}

func TestLayoutInlineContent_RelayoutStartsFromTheOriginalRuns(t *testing.T) {
	ctx := context.Background()
	metrics := &mockFontMetrics{}
	block, run := relayoutTestBlock("alpha beta gamma")

	narrow := newRootLayoutInput(metrics, newLayoutCache(), trackerOrDefault(nil), 50, 800)
	layoutBox(ctx, block, narrow)
	require.Equal(t, []*LayoutBox{run}, block.Children, "a layout pass must not rewrite the tree")

	wide := newRootLayoutInput(metrics, newLayoutCache(), trackerOrDefault(nil), 400, 800)
	fragment := layoutBox(ctx, block, wide)
	writeFragmentsToBoxTree(fragment, 0, 0)

	assert.Equal(t, []string{"alpha beta gamma"}, segmentTexts(block))
}

func TestLayoutInlineContent_ReplacementsApplyWhenWrittenBack(t *testing.T) {
	ctx := context.Background()
	metrics := &mockFontMetrics{}
	block, _ := relayoutTestBlock("alpha beta gamma")

	narrow := newRootLayoutInput(metrics, newLayoutCache(), trackerOrDefault(nil), 50, 800)
	fragment := layoutBox(ctx, block, narrow)
	writeFragmentsToBoxTree(fragment, 0, 0)

	assert.Greater(t, len(segmentTexts(block)), 1, "the narrow layout wraps onto several lines")
	for _, text := range segmentTexts(block) {
		assert.NotContains(t, text, "  ")
	}
}

func TestPreprocessTextChild_StartsFromTheSourceText(t *testing.T) {
	block, run := relayoutTestBlock("mixed Case")
	run.Style.TextTransform = TextTransformUppercase
	inlineContext := &inlineLayoutContext{}

	require.True(t, inlineContext.preprocessTextChild(run))
	assert.Equal(t, "MIXED CASE", run.Text)

	run.Style.TextTransform = TextTransformNone
	require.True(t, inlineContext.preprocessTextChild(run))
	assert.Equal(t, "mixed Case", run.Text)
	assert.Same(t, block, run.Parent)
}

func TestColumnFlexItem_IsProbedAtTheContainerWidth(t *testing.T) {
	textBlock := func(parent *LayoutBox) *LayoutBox {
		block, run := relayoutTestBlock("The closing line")
		block.Parent = parent
		run.Parent = block
		parent.Children = append(parent.Children, block)
		return block
	}

	reference := makeRoot(200)
	plain := textBlock(reference)
	runLayout(reference)

	root := makeRoot(400)
	flexStyle := DefaultComputedStyle()
	flexStyle.Display = DisplayFlex
	flexStyle.FlexDirection = FlexDirectionColumn
	flexStyle.Width = DimensionPt(200)
	column := newLayoutBox(BoxFlex, &flexStyle, root)
	root.Children = []*LayoutBox{column}
	item := textBlock(column)
	item.Type = BoxFlexItem
	runLayout(root)

	require.Greater(t, plain.ContentHeight, 0.0)
	assert.InDelta(t, plain.ContentHeight, item.ContentHeight, 0.001,
		"the item is one line tall, as the same text is in a 200pt block")
	assert.Equal(t, []string{"The closing line"}, segmentTexts(item))
}
