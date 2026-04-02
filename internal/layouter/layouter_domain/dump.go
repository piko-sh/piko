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
	"fmt"
	"strings"
)

// dumpTextTruncationLimit is the maximum number of runes to
// display for a text run before truncating with an ellipsis.
const dumpTextTruncationLimit = 50

// DumpLayoutTree returns a human-readable text representation of the
// layout box tree, suitable for golden file comparison.
//
// Takes root (*LayoutBox) which is the root of the layout box tree
// to serialise.
//
// Returns string which is the formatted tree representation.
func DumpLayoutTree(root *LayoutBox) string {
	var builder strings.Builder

	builder.WriteString("--- BEGIN LAYOUT TREE ---\n\n")
	dumpBox(&builder, root, 0)
	builder.WriteString("\n--- END LAYOUT TREE ---\n")

	return builder.String()
}

// dumpBox writes the formatted representation of a single
// layout box and its children into the builder at the given
// indentation level.
//
// Takes builder (*strings.Builder) which accumulates the
// output text.
// Takes box (*LayoutBox) which is the box to serialise.
// Takes indent (int) which is the current nesting depth.
func dumpBox(builder *strings.Builder, box *LayoutBox, indent int) {
	prefix := strings.Repeat("  ", indent)

	tag := box.TagName()

	if box.Type == BoxTextRun || box.Type == BoxListMarker {
		text := box.Text
		runes := []rune(text)
		if len(runes) > dumpTextTruncationLimit {
			text = string(runes[:dumpTextTruncationLimit]) + "..."
		}
		fmt.Fprintf(builder, "%s[%s] %q (%.2f, %.2f) %.2f x %.2f\n",
			prefix, box.Type, text, box.ContentX, box.ContentY, box.ContentWidth, box.ContentHeight)
		dumpStyleDetails(builder, box, prefix)
		return
	}

	fmt.Fprintf(builder, "%s[%s] <%s> (%.2f, %.2f) %.2f x %.2f\n",
		prefix, box.Type, tag, box.ContentX, box.ContentY, box.ContentWidth, box.ContentHeight)

	dumpStyleDetails(builder, box, prefix)

	for _, child := range box.Children {
		dumpBox(builder, child, indent+1)
	}
}

// dumpStyleDetails writes non-default style properties for a
// layout box into the builder with the given indentation
// prefix. It delegates to sub-functions for box-model
// spacing, typography, visual effects, and border styles.
//
// Takes builder (*strings.Builder) which accumulates the
// output text.
// Takes box (*LayoutBox) which is the box whose styles are
// written.
// Takes prefix (string) which is the indentation to prepend
// to each line.
func dumpStyleDetails(builder *strings.Builder, box *LayoutBox, prefix string) {
	dumpBoxModelSpacing(builder, box, prefix)
	dumpTypographyDetails(builder, box, prefix)
	dumpVisualEffects(builder, box, prefix)
	dumpBorderStyleDetails(builder, box, prefix)
}

// dumpBoxModelSpacing writes non-zero padding, margin, and border
// widths for a layout box.
//
// Takes builder (*strings.Builder) which accumulates the output text.
// Takes box (*LayoutBox) which is the box whose spacing is written.
// Takes prefix (string) which is the indentation to prepend to each line.
func dumpBoxModelSpacing(builder *strings.Builder, box *LayoutBox, prefix string) {
	if box.Padding.Top != 0 || box.Padding.Right != 0 || box.Padding.Bottom != 0 || box.Padding.Left != 0 {
		fmt.Fprintf(builder, "%s  padding: %.2f %.2f %.2f %.2f\n",
			prefix, box.Padding.Top, box.Padding.Right, box.Padding.Bottom, box.Padding.Left)
	}

	if box.Margin.Top != 0 || box.Margin.Right != 0 || box.Margin.Bottom != 0 || box.Margin.Left != 0 {
		fmt.Fprintf(builder, "%s  margin: %.2f %.2f %.2f %.2f\n",
			prefix, box.Margin.Top, box.Margin.Right, box.Margin.Bottom, box.Margin.Left)
	}

	if box.Border.Top != 0 || box.Border.Right != 0 || box.Border.Bottom != 0 || box.Border.Left != 0 {
		fmt.Fprintf(builder, "%s  border: %.2f %.2f %.2f %.2f\n",
			prefix, box.Border.Top, box.Border.Right, box.Border.Bottom, box.Border.Left)
	}
}

// dumpTypographyDetails writes non-default display, position, font,
// colour, and text decoration properties for a layout box.
//
// Takes builder (*strings.Builder) which accumulates the output text.
// Takes box (*LayoutBox) which is the box whose typography is written.
// Takes prefix (string) which is the indentation to prepend to each line.
func dumpTypographyDetails(builder *strings.Builder, box *LayoutBox, prefix string) {
	if box.Style.Display != DisplayBlock || box.Style.Position != PositionStatic {
		fmt.Fprintf(builder, "%s  style: display=%s position=%s\n",
			prefix, box.Style.Display, box.Style.Position)
	}

	if box.Style.FontSize != defaultFontSizePt {
		fmt.Fprintf(builder, "%s  font-size: %.2fpt\n", prefix, box.Style.FontSize)
	}

	if box.Style.FontWeight != defaultFontWeight {
		fmt.Fprintf(builder, "%s  font-weight: %d\n", prefix, box.Style.FontWeight)
	}

	if box.Style.Colour != ColourBlack {
		fmt.Fprintf(builder, "%s  colour: %s\n", prefix, box.Style.Colour)
	}

	if box.Style.BackgroundColour != ColourTransparent {
		fmt.Fprintf(builder, "%s  background: %s\n", prefix, box.Style.BackgroundColour)
	}

	if box.Style.TextDecoration != 0 {
		var parts []string
		if box.Style.TextDecoration&TextDecorationUnderline != 0 {
			parts = append(parts, "underline")
		}
		if box.Style.TextDecoration&TextDecorationOverline != 0 {
			parts = append(parts, "overline")
		}
		if box.Style.TextDecoration&TextDecorationLineThrough != 0 {
			parts = append(parts, "line-through")
		}
		fmt.Fprintf(builder, "%s  text-decoration: %s\n", prefix, strings.Join(parts, " "))
	}

	if box.Style.TextDecorationStyle != TextDecorationStyleSolid {
		fmt.Fprintf(builder, "%s  text-decoration-style: %s\n", prefix, box.Style.TextDecorationStyle)
	}

	if box.Style.TextDecorationColourSet {
		fmt.Fprintf(builder, "%s  text-decoration-colour: %s\n", prefix, box.Style.TextDecorationColour)
	}
}

// dumpVisualEffects writes non-default opacity, visibility,
// transform, overflow, outline, and shadow properties for a layout
// box.
//
// Takes builder (*strings.Builder) which accumulates the output text.
// Takes box (*LayoutBox) which is the box whose visual effects are written.
// Takes prefix (string) which is the indentation to prepend to each line.
func dumpVisualEffects(builder *strings.Builder, box *LayoutBox, prefix string) {
	if box.Style.Opacity < 1.0 {
		fmt.Fprintf(builder, "%s  opacity: %.2f\n", prefix, box.Style.Opacity)
	}

	if box.Style.Visibility != VisibilityVisible {
		fmt.Fprintf(builder, "%s  visibility: %s\n", prefix, box.Style.Visibility)
	}

	if box.Style.HasTransform {
		fmt.Fprintf(builder, "%s  transform: %s\n", prefix, box.Style.TransformValue)
	}

	if box.Style.OverflowX != OverflowVisible {
		fmt.Fprintf(builder, "%s  overflow-x: %s\n", prefix, box.Style.OverflowX)
	}

	if box.Style.OverflowY != OverflowVisible {
		fmt.Fprintf(builder, "%s  overflow-y: %s\n", prefix, box.Style.OverflowY)
	}

	if box.Style.OutlineWidth > 0 && box.Style.OutlineStyle != BorderStyleNone {
		fmt.Fprintf(builder, "%s  outline: %.2f %s %s\n", prefix, box.Style.OutlineWidth, box.Style.OutlineStyle, box.Style.OutlineColour)
	}

	if len(box.Style.BoxShadow) > 0 {
		fmt.Fprintf(builder, "%s  box-shadow: %d layers\n", prefix, len(box.Style.BoxShadow))
	}

	if len(box.Style.TextShadow) > 0 {
		fmt.Fprintf(builder, "%s  text-shadow: %d layers\n", prefix, len(box.Style.TextShadow))
	}
}

// dumpBorderStyleDetails writes non-default, non-solid border style
// values for each side of a layout box that has a non-zero border
// width.
//
// Takes builder (*strings.Builder) which accumulates the output text.
// Takes box (*LayoutBox) which is the box whose border styles are written.
// Takes prefix (string) which is the indentation to prepend to each line.
func dumpBorderStyleDetails(builder *strings.Builder, box *LayoutBox, prefix string) {
	if box.Style.BorderTopStyle != BorderStyleNone && box.Style.BorderTopStyle != BorderStyleSolid && box.Border.Top > 0 {
		fmt.Fprintf(builder, "%s  border-top-style: %s\n", prefix, box.Style.BorderTopStyle)
	}
	if box.Style.BorderBottomStyle != BorderStyleNone && box.Style.BorderBottomStyle != BorderStyleSolid && box.Border.Bottom > 0 {
		fmt.Fprintf(builder, "%s  border-bottom-style: %s\n", prefix, box.Style.BorderBottomStyle)
	}
	if box.Style.BorderLeftStyle != BorderStyleNone && box.Style.BorderLeftStyle != BorderStyleSolid && box.Border.Left > 0 {
		fmt.Fprintf(builder, "%s  border-left-style: %s\n", prefix, box.Style.BorderLeftStyle)
	}
	if box.Style.BorderRightStyle != BorderStyleNone && box.Style.BorderRightStyle != BorderStyleSolid && box.Border.Right > 0 {
		fmt.Fprintf(builder, "%s  border-right-style: %s\n", prefix, box.Style.BorderRightStyle)
	}
}
