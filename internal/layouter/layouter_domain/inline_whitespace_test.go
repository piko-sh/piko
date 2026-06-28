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
	"github.com/stretchr/testify/require"
)

func makeInlineRun(root *LayoutBox, text string) *LayoutBox {
	box := &LayoutBox{Type: BoxTextRun, Style: DefaultComputedStyle(), Parent: root, Text: text}
	box.Style.Display = DisplayInline
	box.Style.FontSize = 12
	return box
}

func makeInlineWrapper(root *LayoutBox, text string) (*LayoutBox, *LayoutBox) {
	wrapper := &LayoutBox{Type: BoxInline, Style: DefaultComputedStyle(), Parent: root}
	wrapper.Style.Display = DisplayInline
	wrapper.Style.FontSize = 12
	inner := &LayoutBox{Type: BoxTextRun, Style: DefaultComputedStyle(), Parent: wrapper, Text: text}
	inner.Style.Display = DisplayInline
	inner.Style.FontSize = 12
	wrapper.Children = []*LayoutBox{inner}
	return wrapper, inner
}

func firstTextRun(box *LayoutBox) *LayoutBox {
	if box.Type == BoxTextRun {
		return box
	}
	for _, c := range box.Children {
		if r := firstTextRun(c); r != nil {
			return r
		}
	}
	return nil
}

func TestInlineWhitespace_PreservedAroundInlineWrapper(t *testing.T) {
	root := makeRoot(500)
	root.Style.Height = DimensionAuto()

	before := makeInlineRun(root, "a strong ")
	bold, _ := makeInlineWrapper(root, "DevOps")
	after := makeInlineRun(root, " culture")
	root.Children = []*LayoutBox{before, bold, after}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 54, before.ContentWidth, integrationEpsilon,
		`"a strong " must keep its trailing space (54pt, not 48pt)`)

	boldText := firstTextRun(bold)
	require.NotNil(t, boldText, "bold wrapper should still have a text run")
	assert.InDelta(t, 54, boldText.ContentX, integrationEpsilon,
		"the bold word must start after the preceding space, at x=54")

	assert.InDelta(t, 48, after.ContentWidth, integrationEpsilon,
		`" culture" must keep its leading space (48pt, not 42pt)`)
	assert.InDelta(t, 90, after.ContentX, integrationEpsilon,
		"the trailing run must start immediately after the bold word, at x=90")
}

func TestInlineWrap_NestedRunNotOrphanedAtOrigin(t *testing.T) {
	root := makeRoot(120)
	root.Style.Height = DimensionAuto()

	before := makeInlineRun(root, "the fundamentals of ")
	bold, _ := makeInlineWrapper(root, "Speed of Development")
	after := makeInlineRun(root, " matter most")
	root.Children = []*LayoutBox{before, bold, after}

	require.NotPanics(t, func() { runLayout(root) })

	boldText := firstTextRun(bold)
	require.NotNil(t, boldText, "bold wrapper should still have a text run after wrapping")
	assert.Equal(t, "Speed of Development", boldText.Text,
		"the wrapped run should retain its full text")
	assert.True(t, boldText.ContentY > 0,
		"wrapped bold run must move to a later line, not stay at the origin (got y=%.1f)", boldText.ContentY)
	assert.True(t, boldText.ContentWidth > 0,
		"wrapped bold run must have a real width, not be left un-laid-out (got w=%.1f)", boldText.ContentWidth)
}

func TestInlineWrap_TrailingSpaceBeforeInlineWrapper(t *testing.T) {
	root := makeRoot(120)
	root.Style.Height = DimensionAuto()

	before := makeInlineRun(root, "pipelines, which enabled ")
	bold, _ := makeInlineWrapper(root, "developer")
	root.Children = []*LayoutBox{before, bold}

	require.NotPanics(t, func() { runLayout(root) })

	boldText := firstTextRun(bold)
	require.NotNil(t, boldText, "bold wrapper should have a text run")

	var enabledSeg *LayoutBox
	var walk func(b *LayoutBox)
	walk = func(b *LayoutBox) {
		if b.Type == BoxTextRun && b.Text != "" && b.Text[len(b.Text)-1] == ' ' &&
			len(b.Text) >= 7 && b.Text[len(b.Text)-8:] == "enabled " {
			enabledSeg = b
		}
		for _, c := range b.Children {
			walk(c)
		}
	}
	walk(root)
	require.NotNil(t, enabledSeg, "should find the wrapped segment ending in 'enabled '")

	assert.InDelta(t, enabledSeg.ContentY, boldText.ContentY, integrationEpsilon,
		"the bold word must stay on the same line as 'enabled'")
	gap := boldText.ContentX - (enabledSeg.ContentX + enabledSeg.ContentWidth)
	assert.InDelta(t, 0, gap, integrationEpsilon,
		"the trailing space is part of the 'enabled ' segment, so the bold abuts its end (got gap=%.1f)", gap)

	assert.True(t, enabledSeg.ContentWidth >= 48-integrationEpsilon,
		"the wrapped segment must include its trailing space (got w=%.1f, want >=48)", enabledSeg.ContentWidth)
}
