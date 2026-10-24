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

const (
	breakTestFontSize = 10
)

func breakTestBlock() *LayoutBox {
	style := DefaultComputedStyle()
	style.Display = DisplayBlock
	style.FontSize = breakTestFontSize
	return newLayoutBox(BoxBlock, &style, nil)
}

func breakTestRun(parent *LayoutBox, text string) *LayoutBox {
	run := newLayoutBox(BoxTextRun, &parent.Style, parent)
	run.Style.Display = DisplayInline
	run.Text = text
	parent.Children = append(parent.Children, run)
	return run
}

func breakTestInline(parent *LayoutBox) *LayoutBox {
	style := parent.Style
	style.Display = DisplayInline
	inline := newLayoutBox(BoxInline, &style, parent)
	parent.Children = append(parent.Children, inline)
	return inline
}

func lineTexts(t *testing.T, block *LayoutBox, width float64) [][]string {
	t.Helper()
	input := newRootLayoutInput(&mockFontMetrics{}, newLayoutCache(), trackerOrDefault(nil), width, 800)
	result := layoutInlineContent(context.Background(), block, input)
	var lines [][]string
	for _, line := range result.Children {
		var texts []string
		var collect func(fragment *Fragment)
		collect = func(fragment *Fragment) {
			if fragment.Box != nil && fragment.Box.Type == BoxTextRun {
				texts = append(texts, fragment.Box.Text)
			}
			for _, child := range fragment.Children {
				collect(child)
			}
		}
		collect(line)
		lines = append(lines, texts)
	}
	return lines
}

func TestInlineLayout_PunctuationGluedToAnInlineBoxNeverStartsALine(t *testing.T) {
	block := breakTestBlock()
	bold := breakTestInline(block)
	breakTestRun(bold, "GitOps")
	breakTestRun(block, ", service meshes")

	lines := lineTexts(t, block, 30)

	require.GreaterOrEqual(t, len(lines), 2)
	assert.Equal(t, []string{"GitOps", ","}, lines[0])
	assert.Equal(t, "service", lines[1][0])
	for _, line := range lines {
		assert.NotRegexp(t, `^[,.;:]`, line[0], "no line starts with punctuation")
	}
}

func TestInlineLayout_LineEndSpaceIsDropped(t *testing.T) {
	block := breakTestBlock()
	first := breakTestRun(block, "aaa ")
	bold := breakTestInline(block)
	breakTestRun(bold, "bbbbbbb")

	input := newRootLayoutInput(&mockFontMetrics{}, newLayoutCache(), trackerOrDefault(nil), 40, 800)
	result := layoutInlineContent(context.Background(), block, input)

	require.Len(t, result.Children, 2, "the bold word wraps to a line of its own")
	assert.Equal(t, "aaa", first.Text)
	assert.InDelta(t, 15, result.Children[0].Children[0].ContentWidth, 0.001)
}

func TestInlineLayout_FitToleranceKeepsAnAlmostFittingWordOnTheLine(t *testing.T) {
	block := breakTestBlock()
	breakTestRun(block, "aaa bbb")

	lines := lineTexts(t, block, 35-0.005)

	assert.Len(t, lines, 1)
}
