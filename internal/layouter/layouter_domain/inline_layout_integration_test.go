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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const integrationEpsilon = 0.001

func TestLayoutTextRun(t *testing.T) {
	fm := &mockFontMetrics{}

	t.Run("measures Hello at fontSize 12", func(t *testing.T) {
		box := &LayoutBox{
			Type:  BoxTextRun,
			Style: DefaultComputedStyle(),
			Text:  "Hello",
		}
		box.Style.FontSize = 12

		box.Style.LineHeight = 0

		frag := layoutTextRun(box, fm)

		assert.InDelta(t, 30, frag.ContentWidth, integrationEpsilon, "width should be 30pt")

		assert.InDelta(t, 13.2, frag.ContentHeight, integrationEpsilon, "height should be 13.2pt")
		assert.NotNil(t, box.Glyphs, "glyphs should be populated")
		assert.Equal(t, 5, len(box.Glyphs), "should have one glyph per rune")
	})

	t.Run("empty text produces zero width", func(t *testing.T) {
		box := &LayoutBox{
			Type:  BoxTextRun,
			Style: DefaultComputedStyle(),
			Text:  "",
		}
		box.Style.FontSize = 12
		box.Style.LineHeight = 0

		frag := layoutTextRun(box, fm)

		assert.InDelta(t, 0, frag.ContentWidth, integrationEpsilon, "empty text should have zero width")

		assert.InDelta(t, 13.2, frag.ContentHeight, integrationEpsilon, "height should still reflect font metrics")
	})

	t.Run("explicit LineHeight larger than font metrics is used", func(t *testing.T) {
		box := &LayoutBox{
			Type:  BoxTextRun,
			Style: DefaultComputedStyle(),
			Text:  "AB",
		}
		box.Style.FontSize = 10
		box.Style.LineHeight = 30

		frag := layoutTextRun(box, fm)

		assert.InDelta(t, 10, frag.ContentWidth, integrationEpsilon)

		assert.InDelta(t, 30, frag.ContentHeight, integrationEpsilon, "explicit line-height should take precedence")
	})

	t.Run("default computed style uses default line height", func(t *testing.T) {
		box := &LayoutBox{
			Type:  BoxTextRun,
			Style: DefaultComputedStyle(),
			Text:  "X",
		}

		frag := layoutTextRun(box, fm)

		assert.InDelta(t, 6, frag.ContentWidth, integrationEpsilon)

		assert.InDelta(t, 16.8, frag.ContentHeight, integrationEpsilon)
	})
}

func TestInlineLayout_SingleTextRun(t *testing.T) {
	root := makeRoot(500)
	root.Style.Height = DimensionAuto()

	text_box := &LayoutBox{
		Type:   BoxTextRun,
		Style:  DefaultComputedStyle(),
		Parent: root,
		Text:   "Hello",
	}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12

	root.Children = []*LayoutBox{text_box}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 30, text_box.ContentWidth, integrationEpsilon, "text width should be 30pt")
	assert.True(t, text_box.ContentHeight > 0, "text height should be positive")

	assert.Equal(t, 1, len(root.Children), "no wrapping should occur")
}

func TestInlineLayout_TextWrapping(t *testing.T) {
	root := makeRoot(50)
	root.Style.Height = DimensionAuto()

	text_box := &LayoutBox{
		Type:   BoxTextRun,
		Style:  DefaultComputedStyle(),
		Parent: root,
		Text:   "Hello world foo bar",
	}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12

	root.Children = []*LayoutBox{text_box}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, len(root.Children) > 1,
		"text should be split across multiple lines, got %d children", len(root.Children))

	for i, child := range root.Children {
		assert.NotEmpty(t, child.Text, "child %d should have text", i)
		assert.True(t, child.ContentWidth > 0, "child %d should have positive width", i)
	}

	parts := make([]string, 0, len(root.Children))
	for _, child := range root.Children {
		parts = append(parts, child.Text)
	}
	reconstructed := strings.Join(parts, "")

	for _, word := range []string{"Hello", "world", "foo", "bar"} {
		assert.Contains(t, reconstructed, word,
			"reconstructed text should contain word %q", word)
	}
}

func TestInlineLayout_WordBreakAll(t *testing.T) {
	root := makeRoot(30)
	root.Style.Height = DimensionAuto()

	text_box := &LayoutBox{
		Type:   BoxTextRun,
		Style:  DefaultComputedStyle(),
		Parent: root,
		Text:   "ABCDEFGHIJKLMNOP",
	}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12
	text_box.Style.WordBreak = WordBreakBreakAll

	root.Children = []*LayoutBox{text_box}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, len(root.Children) > 1,
		"break-all should split into multiple segments, got %d", len(root.Children))

	var combined strings.Builder
	for _, child := range root.Children {
		combined.WriteString(child.Text)
	}
	assert.Equal(t, "ABCDEFGHIJKLMNOP", combined.String(),
		"all characters should be preserved after breaking")

	for i, child := range root.Children {
		assert.True(t, child.ContentWidth <= 30+integrationEpsilon,
			"segment %d width (%f) should not exceed 30pt", i, child.ContentWidth)
	}
}

func TestInlineLayout_OverflowWrapBreakWord(t *testing.T) {
	root := makeRoot(30)
	root.Style.Height = DimensionAuto()

	text_box := &LayoutBox{
		Type:   BoxTextRun,
		Style:  DefaultComputedStyle(),
		Parent: root,
		Text:   "ABCDEFGHIJKLMNOP",
	}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12
	text_box.Style.OverflowWrap = OverflowWrapBreakWord

	root.Children = []*LayoutBox{text_box}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, len(root.Children) > 1,
		"overflow-wrap break-word should produce multiple segments, got %d", len(root.Children))

	var combined strings.Builder
	for _, child := range root.Children {
		combined.WriteString(child.Text)
	}
	assert.Equal(t, "ABCDEFGHIJKLMNOP", combined.String(),
		"all characters should be preserved")
}

func TestInlineLayout_Ellipsis(t *testing.T) {
	root := makeRoot(50)
	root.Style.Height = DimensionAuto()
	root.Style.OverflowX = OverflowHidden
	root.Style.WhiteSpace = WhiteSpaceNowrap
	root.Style.TextOverflow = TextOverflowEllipsis

	text_box := &LayoutBox{
		Type:   BoxTextRun,
		Style:  DefaultComputedStyle(),
		Parent: root,
		Text:   "Hello world this is long text",
	}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12

	root.Children = []*LayoutBox{text_box}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, strings.HasSuffix(text_box.Text, "\u2026"),
		"text should end with ellipsis, got %q", text_box.Text)

	assert.True(t, text_box.ContentWidth <= 50+integrationEpsilon,
		"truncated width (%f) should not exceed 50pt", text_box.ContentWidth)

	assert.True(t, len(text_box.Text) < len("Hello world this is long text"),
		"truncated text should be shorter than original")
}

func TestInlineLayout_InlineBlock(t *testing.T) {
	root := makeRoot(300)
	root.Style.Height = DimensionAuto()

	inline_block := &LayoutBox{
		Type:   BoxInlineBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	inline_block.Style.Display = DisplayInlineBlock
	inline_block.Style.Width = DimensionPt(100)
	inline_block.Style.Height = DimensionPt(50)

	root.Children = []*LayoutBox{inline_block}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 100, inline_block.ContentWidth, integrationEpsilon,
		"inline-block should have width 100")
	assert.InDelta(t, 50, inline_block.ContentHeight, integrationEpsilon,
		"inline-block should have height 50")
	assert.True(t, inline_block.ContentX >= 0, "inline-block X should be >= 0")
}

func TestInlineLayout_InlineBlockWrapping(t *testing.T) {
	root := makeRoot(150)
	root.Style.Height = DimensionAuto()

	block_a := &LayoutBox{
		Type:   BoxInlineBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	block_a.Style.Display = DisplayInlineBlock
	block_a.Style.Width = DimensionPt(100)
	block_a.Style.Height = DimensionPt(30)

	block_b := &LayoutBox{
		Type:   BoxInlineBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	block_b.Style.Display = DisplayInlineBlock
	block_b.Style.Width = DimensionPt(100)
	block_b.Style.Height = DimensionPt(30)

	root.Children = []*LayoutBox{block_a, block_b}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, block_b.ContentY > block_a.ContentY,
		"second inline-block should wrap to next line (Y: %f > %f)",
		block_b.ContentY, block_a.ContentY)
}

func TestInlineLayout_TextAlignCentre(t *testing.T) {
	root := makeRoot(300)
	root.Style.Height = DimensionAuto()
	root.Style.TextAlign = TextAlignCentre

	text_box := &LayoutBox{
		Type:   BoxTextRun,
		Style:  DefaultComputedStyle(),
		Parent: root,
		Text:   "Hi",
	}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12

	root.Children = []*LayoutBox{text_box}

	require.NotPanics(t, func() { runLayout(root) })

	expected_x := (300.0 - 12.0) / 2.0
	assert.InDelta(t, expected_x, text_box.ContentX, integrationEpsilon,
		"text should be centred")
}

func TestInlineLayout_TextAlignRight(t *testing.T) {
	root := makeRoot(300)
	root.Style.Height = DimensionAuto()
	root.Style.TextAlign = TextAlignRight

	text_box := &LayoutBox{
		Type:   BoxTextRun,
		Style:  DefaultComputedStyle(),
		Parent: root,
		Text:   "Hi",
	}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12

	root.Children = []*LayoutBox{text_box}

	require.NotPanics(t, func() { runLayout(root) })

	expected_x := 300.0 - 12.0
	assert.InDelta(t, expected_x, text_box.ContentX, integrationEpsilon,
		"text should be right-aligned")
}

func TestInlineLayout_TextAlignJustify(t *testing.T) {
	root := makeRoot(100)
	root.Style.Height = DimensionAuto()
	root.Style.TextAlign = TextAlignJustify

	text_box := &LayoutBox{
		Type:   BoxTextRun,
		Style:  DefaultComputedStyle(),
		Parent: root,
		Text:   "AA BB CC DD EE FF GG",
	}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12

	root.Children = []*LayoutBox{text_box}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, len(root.Children) > 1,
		"text should wrap into multiple segments for justify to apply")

	assert.True(t, root.ContentHeight > 0 || root.ContentWidth > 0,
		"root should have layout dimensions")
}

func TestInlineLayout_NonTextChild(t *testing.T) {
	root := makeRoot(300)
	root.Style.Height = DimensionAuto()

	inline_child := &LayoutBox{
		Type:   BoxInline,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	inline_child.Style.Display = DisplayInline

	root.Children = []*LayoutBox{inline_child}

	require.NotPanics(t, func() { runLayout(root) })

	assert.Equal(t, 1, len(root.Children), "inline child should remain")
}

func TestInlineLayout_MultipleTextChildren(t *testing.T) {
	t.Run("two short texts fit on one line", func(t *testing.T) {
		root := makeRoot(300)
		root.Style.Height = DimensionAuto()

		text_a := &LayoutBox{
			Type:   BoxTextRun,
			Style:  DefaultComputedStyle(),
			Parent: root,
			Text:   "Hello",
		}
		text_a.Style.Display = DisplayInline
		text_a.Style.FontSize = 12

		text_b := &LayoutBox{
			Type:   BoxTextRun,
			Style:  DefaultComputedStyle(),
			Parent: root,
			Text:   "World",
		}
		text_b.Style.Display = DisplayInline
		text_b.Style.FontSize = 12

		root.Children = []*LayoutBox{text_a, text_b}

		require.NotPanics(t, func() { runLayout(root) })

		assert.True(t, text_b.ContentX >= text_a.ContentX+text_a.ContentWidth-integrationEpsilon,
			"second text should follow first on same line")

		assert.InDelta(t, text_a.ContentY, text_b.ContentY, integrationEpsilon,
			"both texts should be on the same line")
	})

	t.Run("overflow causes wrapping across lines", func(t *testing.T) {
		root := makeRoot(40)
		root.Style.Height = DimensionAuto()

		text_a := &LayoutBox{
			Type:   BoxTextRun,
			Style:  DefaultComputedStyle(),
			Parent: root,
			Text:   "AAAA",
		}
		text_a.Style.Display = DisplayInline
		text_a.Style.FontSize = 12

		text_b := &LayoutBox{
			Type:   BoxTextRun,
			Style:  DefaultComputedStyle(),
			Parent: root,
			Text:   "BBBB",
		}
		text_b.Style.Display = DisplayInline
		text_b.Style.FontSize = 12

		root.Children = []*LayoutBox{text_a, text_b}

		require.NotPanics(t, func() { runLayout(root) })

		has_content := false
		for _, child := range root.Children {
			if child.ContentWidth > 0 {
				has_content = true
				break
			}
		}
		assert.True(t, has_content, "children should have positive content widths")
	})
}

func TestInlineLayout_SoftHyphens(t *testing.T) {
	t.Run("soft hyphens trigger hyphenated breaks", func(t *testing.T) {

		root := makeRoot(50)
		root.Style.Height = DimensionAuto()

		text_box := &LayoutBox{
			Type:   BoxTextRun,
			Style:  DefaultComputedStyle(),
			Parent: root,
			Text:   "butter\u00ADfly",
		}
		text_box.Style.Display = DisplayInline
		text_box.Style.FontSize = 12

		root.Children = []*LayoutBox{text_box}

		require.NotPanics(t, func() { runLayout(root) })

		assert.True(t, len(root.Children) >= 2,
			"soft-hyphen word should be split, got %d children", len(root.Children))

		if len(root.Children) >= 2 {
			assert.True(t, strings.HasSuffix(root.Children[0].Text, "-"),
				"first segment should end with hyphen, got %q", root.Children[0].Text)
		}
	})

	t.Run("hyphens none strips soft hyphens and does not insert visible hyphens", func(t *testing.T) {

		root := makeRoot(40)
		root.Style.Height = DimensionAuto()

		text_box := &LayoutBox{
			Type:   BoxTextRun,
			Style:  DefaultComputedStyle(),
			Parent: root,
			Text:   "butter\u00ADfly",
		}
		text_box.Style.Display = DisplayInline
		text_box.Style.FontSize = 12
		text_box.Style.Hyphens = HyphensNone

		root.Children = []*LayoutBox{text_box}

		require.NotPanics(t, func() { runLayout(root) })

		for i, child := range root.Children {
			assert.False(t, strings.HasSuffix(child.Text, "-"),
				"child %d (%q) should not end with visible hyphen when hyphens: none", i, child.Text)
		}

		for i, child := range root.Children {
			assert.False(t, strings.Contains(child.Text, "\u00AD"),
				"child %d (%q) should not contain soft hyphen when hyphens: none", i, child.Text)
		}
	})
}

func TestInlineLayout_Nowrap(t *testing.T) {
	root := makeRoot(30)
	root.Style.Height = DimensionAuto()
	root.Style.WhiteSpace = WhiteSpaceNowrap

	text_box := &LayoutBox{
		Type:   BoxTextRun,
		Style:  DefaultComputedStyle(),
		Parent: root,
		Text:   "This text is quite long",
	}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12

	root.Children = []*LayoutBox{text_box}

	require.NotPanics(t, func() { runLayout(root) })

	assert.Equal(t, 1, len(root.Children),
		"nowrap should prevent text splitting")
	assert.InDelta(t, 138, text_box.ContentWidth, integrationEpsilon,
		"full text width should be measured even if it overflows")
}

func TestInlineLayout_InlineBlockWithContent(t *testing.T) {
	root := makeRoot(300)
	root.Style.Height = DimensionAuto()

	inline_block := &LayoutBox{
		Type:   BoxInlineBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	inline_block.Style.Display = DisplayInlineBlock
	inline_block.Style.Width = DimensionPt(120)
	inline_block.Style.Height = DimensionAuto()

	inner := &LayoutBox{
		Type:   BoxBlock,
		Style:  DefaultComputedStyle(),
		Parent: inline_block,
	}
	inner.Style.Display = DisplayBlock
	inner.Style.Width = DimensionPt(100)
	inner.Style.Height = DimensionPt(40)

	inline_block.Children = []*LayoutBox{inner}
	root.Children = []*LayoutBox{inline_block}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 120, inline_block.ContentWidth, integrationEpsilon,
		"inline-block width should be 120")
	assert.True(t, inline_block.ContentHeight >= 40,
		"inline-block height should accommodate its child (got %f)", inline_block.ContentHeight)
	assert.InDelta(t, 100, inner.ContentWidth, integrationEpsilon,
		"inner block width should be 100")
}

func TestInlineLayout_MixedInlineAndInlineBlock(t *testing.T) {
	root := makeRoot(500)
	root.Style.Height = DimensionAuto()

	text_box := &LayoutBox{
		Type:   BoxTextRun,
		Style:  DefaultComputedStyle(),
		Parent: root,
		Text:   "Hello",
	}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12

	inline_block := &LayoutBox{
		Type:   BoxInlineBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	inline_block.Style.Display = DisplayInlineBlock
	inline_block.Style.Width = DimensionPt(80)
	inline_block.Style.Height = DimensionPt(30)

	root.Children = []*LayoutBox{text_box, inline_block}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 30, text_box.ContentWidth, integrationEpsilon)
	assert.InDelta(t, 80, inline_block.ContentWidth, integrationEpsilon)

	assert.True(t, inline_block.ContentX >= text_box.ContentX+text_box.ContentWidth-integrationEpsilon,
		"inline-block should follow text on the same line")
}

func TestInlineLayout_TextAlignCentreWithWrapping(t *testing.T) {
	root := makeRoot(60)
	root.Style.Height = DimensionAuto()
	root.Style.TextAlign = TextAlignCentre

	text_box := &LayoutBox{
		Type:   BoxTextRun,
		Style:  DefaultComputedStyle(),
		Parent: root,
		Text:   "AA BB CC",
	}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12

	root.ContentWidth = 35
	root.Style.Width = DimensionPt(35)

	root.Children = []*LayoutBox{text_box}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, len(root.Children) >= 1, "should have at least one child")
	for _, child := range root.Children {
		assert.True(t, child.ContentWidth >= 0,
			"each child should have non-negative width")
	}
}

func TestInlineLayout_OverflowWrapAnywhere(t *testing.T) {
	root := makeRoot(30)
	root.Style.Height = DimensionAuto()

	text_box := &LayoutBox{
		Type:   BoxTextRun,
		Style:  DefaultComputedStyle(),
		Parent: root,
		Text:   "ABCDEFGHIJ",
	}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12
	text_box.Style.OverflowWrap = OverflowWrapAnywhere

	root.Children = []*LayoutBox{text_box}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, len(root.Children) > 1,
		"overflow-wrap: anywhere should split oversized word, got %d children", len(root.Children))

	var combined strings.Builder
	for _, child := range root.Children {
		combined.WriteString(child.Text)
	}
	assert.Equal(t, "ABCDEFGHIJ", combined.String(),
		"all characters should be preserved")
}

func TestInlineLayout_EllipsisNoRoom(t *testing.T) {
	root := makeRoot(8)
	root.Style.Height = DimensionAuto()
	root.Style.OverflowX = OverflowHidden
	root.Style.WhiteSpace = WhiteSpaceNowrap
	root.Style.TextOverflow = TextOverflowEllipsis

	text_box := &LayoutBox{
		Type:   BoxTextRun,
		Style:  DefaultComputedStyle(),
		Parent: root,
		Text:   "Hello world",
	}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12

	root.Children = []*LayoutBox{text_box}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, strings.Contains(text_box.Text, "\u2026"),
		"text should contain ellipsis, got %q", text_box.Text)
}

func TestInlineLayout_GlyphsPopulated(t *testing.T) {
	root := makeRoot(40)
	root.Style.Height = DimensionAuto()

	text_box := &LayoutBox{
		Type:   BoxTextRun,
		Style:  DefaultComputedStyle(),
		Parent: root,
		Text:   "AB CD EF",
	}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12

	root.Children = []*LayoutBox{text_box}

	require.NotPanics(t, func() { runLayout(root) })

	for i, child := range root.Children {
		if child.Text != "" {
			assert.NotNil(t, child.Glyphs,
				"child %d (%q) should have glyphs", i, child.Text)
			assert.True(t, len(child.Glyphs) > 0,
				"child %d (%q) should have at least one glyph", i, child.Text)
		}
	}
}
