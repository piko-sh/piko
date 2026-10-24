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

	"piko.sh/piko/internal/ast/ast_domain"
)

func whitespaceTestBox(boxType BoxType, parent *LayoutBox) *LayoutBox {
	style := DefaultComputedStyle()
	switch boxType {
	case BoxFlex:
		style.Display = DisplayFlex
	case BoxInline, BoxReplaced:
		style.Display = DisplayInline
	default:
		style.Display = DisplayBlock
	}
	box := newLayoutBox(boxType, &style, parent)
	if parent != nil {
		parent.Children = append(parent.Children, box)
	}
	return box
}

func whitespaceTestText(parent *LayoutBox, text string) *LayoutBox {
	run := newLayoutBox(BoxTextRun, &parent.Style, parent)
	run.Style.Display = DisplayInline
	run.Text = text
	parent.Children = append(parent.Children, run)
	return run
}

func TestWrapFlexGridTextRuns_TextBecomesAnAnonymousItem(t *testing.T) {
	flex := whitespaceTestBox(BoxFlex, nil)
	span := whitespaceTestBox(BoxFlexItem, flex)
	run := whitespaceTestText(flex, " Documentation ")

	wrapFlexGridTextRuns(flex)
	collapseWhiteSpace(flex)

	require.Len(t, flex.Children, 2)
	assert.Same(t, span, flex.Children[0])
	item := flex.Children[1]
	assert.Equal(t, BoxAnonymousBlock, item.Type)
	require.Len(t, item.Children, 1)
	assert.Same(t, run, item.Children[0])
	assert.Equal(t, "Documentation", run.Text)
}

func TestWrapFlexGridTextRuns_WhiteSpaceBetweenItemsIsDropped(t *testing.T) {
	flex := whitespaceTestBox(BoxFlex, nil)
	first := whitespaceTestBox(BoxFlexItem, flex)
	whitespaceTestText(flex, "\n  ")
	second := whitespaceTestBox(BoxFlexItem, flex)

	wrapFlexGridTextRuns(flex)

	require.Len(t, flex.Children, 2)
	assert.Same(t, first, flex.Children[0])
	assert.Same(t, second, flex.Children[1])
}

func TestWrapFlexGridTextRuns_IconAndLabelMakeTwoItems(t *testing.T) {
	flex := whitespaceTestBox(BoxFlex, nil)
	icon := whitespaceTestBox(BoxReplaced, flex)
	whitespaceTestText(flex, "\n        ")
	label := whitespaceTestText(flex, "go.dev/doc/asm\n      ")

	wrapFlexGridTextRuns(flex)
	collapseWhiteSpace(flex)

	require.Len(t, flex.Children, 2)
	assert.Same(t, icon, flex.Children[0])
	assert.Equal(t, BoxAnonymousBlock, flex.Children[1].Type)
	require.Len(t, flex.Children[1].Children, 1)
	assert.Same(t, label, flex.Children[1].Children[0])
	assert.Equal(t, "go.dev/doc/asm", label.Text)
}

func TestIsFlexOrGridContainer_InlineFlexItem(t *testing.T) {
	flex := whitespaceTestBox(BoxFlex, nil)
	chip := whitespaceTestBox(BoxFlexItem, flex)
	chip.Style.Display = DisplayInlineFlex
	run := whitespaceTestText(chip, "\n          Go\n        ")

	require.True(t, isFlexOrGridContainer(chip))
	wrapFlexGridTextRuns(chip)
	collapseWhiteSpace(flex)

	require.Len(t, chip.Children, 1)
	assert.Equal(t, "Go", run.Text)
}

func TestCollapseWhiteSpace_InlineContextLosesTheSpaceAtEachEdge(t *testing.T) {
	block := whitespaceTestBox(BoxBlock, nil)
	lead := whitespaceTestText(block, "\n  ab ")
	inner := whitespaceTestBox(BoxInline, block)
	tail := whitespaceTestText(inner, " cd \n")

	collapseWhiteSpace(block)

	assert.Equal(t, "ab ", lead.Text)

	assert.Equal(t, "cd", tail.Text)
}

func TestCollapseWhiteSpace_SegmentBreaksAndTabsBecomeOneSpace(t *testing.T) {
	block := whitespaceTestBox(BoxBlock, nil)
	run := whitespaceTestText(block, "one\ntwo\t\tthree \n  four")

	collapseWhiteSpace(block)

	assert.Equal(t, "one two three four", run.Text)
}

func TestCollapseWhiteSpace_NoBreakSpaceIsKept(t *testing.T) {
	block := whitespaceTestBox(BoxBlock, nil)
	run := whitespaceTestText(block, "a\u00a0 \u00a0b")

	collapseWhiteSpace(block)

	assert.Equal(t, "a\u00a0 \u00a0b", run.Text)
}

func TestCollapseWhiteSpace_PreservedRunKeepsTheSpaceAtAnEdge(t *testing.T) {
	block := whitespaceTestBox(BoxBlock, nil)
	run := whitespaceTestText(block, "  ab  ")
	run.Style.WhiteSpace = WhiteSpacePre

	collapseWhiteSpace(block)

	assert.Equal(t, "  ab  ", run.Text)
}

func TestCollapseWhiteSpace_PreLineKeepsLineFeeds(t *testing.T) {
	block := whitespaceTestBox(BoxBlock, nil)
	run := whitespaceTestText(block, "a  b \n  c")
	run.Style.WhiteSpace = WhiteSpacePreLine

	collapseWhiteSpace(block)

	assert.Equal(t, "a b\nc", run.Text)
}

func TestCollapseWhiteSpace_SpaceBesideAnAtomicInlineAtAnEdgeStays(t *testing.T) {
	block := whitespaceTestBox(BoxBlock, nil)
	whitespaceTestBox(BoxReplaced, block)
	run := whitespaceTestText(block, " ab ")

	collapseWhiteSpace(block)

	assert.Equal(t, " ab", run.Text)
}

func TestCollapseWhiteSpace_TrailingTrimWalksBackOverEmptyRuns(t *testing.T) {
	block := whitespaceTestBox(BoxBlock, nil)
	first := whitespaceTestText(block, "Hello ")
	bold := whitespaceTestBox(BoxInline, block)
	whitespaceTestText(bold, " ")
	whitespaceTestText(block, "\n ")

	collapseWhiteSpace(block)

	assert.Equal(t, "Hello", first.Text)
	assert.Empty(t, bold.Children, "the empty run inside the inline box is pruned")
	require.Len(t, block.Children, 2, "the trailing blank run is pruned")
}

func TestCollapseWhiteSpace_EmptyAnonymousBlockIsPruned(t *testing.T) {
	block := whitespaceTestBox(BoxBlock, nil)
	whitespaceTestBox(BoxBlock, block)
	whitespaceTestText(block, "\n  ")
	whitespaceTestBox(BoxBlock, block)

	fixAnonymousBoxes(block)
	require.Len(t, block.Children, 3)
	collapseWhiteSpace(block)

	require.Len(t, block.Children, 2)
	for _, child := range block.Children {
		assert.Equal(t, BoxBlock, child.Type)
	}
}

func TestIsInlineLevelChild_ReplacedInlineIcon(t *testing.T) {
	block := whitespaceTestBox(BoxBlock, nil)
	icon := whitespaceTestBox(BoxReplaced, block)
	whitespaceTestText(block, "label")

	assert.True(t, isInlineLevelChild(icon))
	assert.True(t, holdsInlineContext(block))
	assert.True(t, hasOnlyInlineChildren(block))
}

func TestBuildBoxTree_InterpolatedTextInFlexContainers(t *testing.T) {
	interpolated := func(text string) *ast_domain.TemplateNode {
		writer := ast_domain.GetDirectWriter()
		writer.AppendEscapeString(text)
		return &ast_domain.TemplateNode{NodeType: ast_domain.NodeText, TextContentWriter: writer}
	}
	element := func(tag string, children ...*ast_domain.TemplateNode) *ast_domain.TemplateNode {
		return &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: tag, Children: children}
	}
	styled := func(display DisplayType) *ComputedStyle {
		style := DefaultComputedStyle()
		style.Display = display
		return &style
	}

	icon := element("svg")
	link := element("a", icon, interpolated("michael@example.com\n        "))
	chip := element("span", interpolated("\n          Go\n        "))
	row := element("div", link, interpolated("\n      "), chip)
	styles := StyleMap{
		row:  styled(DisplayFlex),
		link: styled(DisplayInlineFlex),
		icon: styled(DisplayInlineBlock),
		chip: styled(DisplayInlineFlex),
	}

	root, err := BuildBoxTree(context.Background(), newTestBoxTreeInput(
		&ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{row}}, styles, nil, nil, nil))
	require.NoError(t, err)

	rowBox := root.Children[0]
	require.Len(t, rowBox.Children, 2, "the blank run between the link and the chip makes no item")
	linkBox, chipBox := rowBox.Children[0], rowBox.Children[1]

	require.Len(t, linkBox.Children, 2, "the icon and one item for its label")
	assert.Equal(t, BoxReplaced, linkBox.Children[0].Type)
	label := linkBox.Children[1]
	require.Equal(t, BoxAnonymousBlock, label.Type)
	assert.Equal(t, "michael@example.com", label.Children[0].Text)

	require.Len(t, chipBox.Children, 1)
	assert.Equal(t, "Go", chipBox.Children[0].Children[0].Text)
}
