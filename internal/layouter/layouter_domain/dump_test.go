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
)

func TestDumpLayoutTree_EmptyRoot(t *testing.T) {
	root := &LayoutBox{
		Type: BoxBlock,
	}

	output := DumpLayoutTree(root)

	assert.Contains(t, output, "--- BEGIN LAYOUT TREE ---")
	assert.Contains(t, output, "--- END LAYOUT TREE ---")
	assert.Contains(t, output, "[Block]")
	assert.Contains(t, output, "<#root>")
}

func TestDumpLayoutTree_TextRun(t *testing.T) {
	textBox := &LayoutBox{
		Type:         BoxTextRun,
		Text:         "Hello, world!",
		ContentX:     10,
		ContentY:     20,
		ContentWidth: 80,
	}

	root := &LayoutBox{
		Type:     BoxBlock,
		Children: []*LayoutBox{textBox},
	}

	output := DumpLayoutTree(root)

	assert.Contains(t, output, "[TextRun]")
	assert.Contains(t, output, `"Hello, world!"`)
}

func TestDumpLayoutTree_LongTextTruncation(t *testing.T) {

	longText := strings.Repeat("a", 60)

	textBox := &LayoutBox{
		Type: BoxTextRun,
		Text: longText,
	}

	root := &LayoutBox{
		Type:     BoxBlock,
		Children: []*LayoutBox{textBox},
	}

	output := DumpLayoutTree(root)

	truncated := strings.Repeat("a", 50) + "..."
	assert.Contains(t, output, truncated)

	assert.NotContains(t, output, strings.Repeat("a", 60))
}

func TestDumpLayoutTree_NestedTree(t *testing.T) {
	textBox := &LayoutBox{
		Type: BoxTextRun,
		Text: "nested text",
	}

	childBlock := &LayoutBox{
		Type:     BoxBlock,
		Children: []*LayoutBox{textBox},
	}

	root := &LayoutBox{
		Type:     BoxBlock,
		Children: []*LayoutBox{childBlock},
	}

	output := DumpLayoutTree(root)

	lines := strings.Split(output, "\n")

	foundRoot := false
	foundChild := false
	foundText := false

	for _, line := range lines {
		if strings.Contains(line, "[Block]") && !strings.HasPrefix(line, "  ") {
			foundRoot = true
		}
		if strings.HasPrefix(line, "  [Block]") {
			foundChild = true
		}
		if strings.HasPrefix(line, "    [TextRun]") {
			foundText = true
		}
	}

	assert.True(t, foundRoot, "root block should appear at indent level 0")
	assert.True(t, foundChild, "child block should appear at indent level 1")
	assert.True(t, foundText, "text run should appear at indent level 2")
}

func TestDumpLayoutTree_StyleDetails(t *testing.T) {
	root := &LayoutBox{
		Type:         BoxBlock,
		ContentWidth: 200,
		Padding:      BoxEdges{Top: 5, Right: 5, Bottom: 5, Left: 5},
		Margin:       BoxEdges{Top: 10, Right: 0, Bottom: 10, Left: 0},
		Border:       BoxEdges{Top: 1, Right: 1, Bottom: 1, Left: 1},
		Style: ComputedStyle{
			Display:          DisplayFlex,
			Position:         PositionRelative,
			FontSize:         16,
			FontWeight:       700,
			Colour:           NewRGBA(1, 0, 0, 1),
			BackgroundColour: NewRGBA(1, 1, 1, 1),
		},
	}

	output := DumpLayoutTree(root)

	assert.Contains(t, output, "padding:")

	assert.Contains(t, output, "margin:")

	assert.Contains(t, output, "border:")

	assert.Contains(t, output, "style: display=flex position=relative")

	assert.Contains(t, output, "font-size: 16.00pt")

	assert.Contains(t, output, "font-weight: 700")

	assert.Contains(t, output, "colour:")

	assert.Contains(t, output, "background:")
}
