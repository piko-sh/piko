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

package driven_svgwriter

const (
	// minPolygonPoints holds the minimum number of coordinate values needed
	// to form a polygon (two x,y pairs).
	minPolygonPoints = 4
)

// applyClipPath looks up a clip-path url(#id) reference in defs and
// renders the clip path's children as clipping shapes.
//
// Takes rc (*renderContext) which provides the PDF stream and defs map.
// Takes node (*Node) which is the element that references the clip path.
// Takes style (*Style) which may contain a clip-path reference in its display property.
//
// Returns bool which is true if a clip path was applied.
func applyClipPath(rc *renderContext, node *Node, style *Style) bool {
	ref := resolveClipRef(node, style)
	if ref == "" {
		return false
	}

	clipNode, ok := rc.defs[ref]
	if !ok || clipNode.Tag != "clipPath" {
		return false
	}

	for _, child := range clipNode.Children {
		emitShapeAsPath(rc, child)
	}

	clipRule := "nonzero"
	if rule, ok := clipNode.Attrs["clip-rule"]; ok {
		clipRule = rule
	}

	if clipRule == "evenodd" {
		rc.stream.ClipEvenOdd()
	} else {
		rc.stream.ClipNonZero()
	}

	return true
}

// resolveClipRef extracts the clip-path reference id from the node's
// style property, clip-path attribute, or inline style attribute.
//
// Takes node (*Node) which is the element to inspect for clip-path references.
// Takes style (*Style) which may contain a clip-path url in its display property.
//
// Returns string which is the resolved clip-path id, or "" if none is found.
func resolveClipRef(node *Node, style *Style) string {
	ref := ParseURLRef(style.Display)
	if ref == "" {
		if cpAttr, ok := node.Attrs["clip-path"]; ok {
			ref = ParseURLRef(cpAttr)
		}
	}

	if ref == "" {
		if styleAttr, ok := node.Attrs["style"]; ok {
			props := parseInlineStyle(styleAttr)
			if cp, ok := props["clip-path"]; ok {
				ref = ParseURLRef(cp)
			}
		}
	}
	return ref
}

// emitShapeAsPath emits a shape element as path operators without
// painting, for use in clipping.
//
// Takes rc (*renderContext) which provides the PDF stream to write path operators to.
// Takes node (*Node) which is the shape element to emit.
func emitShapeAsPath(rc *renderContext, node *Node) {
	switch node.Tag {
	case "rect":
		emitRectClip(rc, node)
	case "circle":
		emitCircleClip(rc, node)
	case "ellipse":
		emitEllipseClip(rc, node)
	case "polygon":
		emitPolygonClip(rc, node)
	case "path":
		emitPathClip(rc, node)
	case "use":
		emitUseClip(rc, node)
	}
}

// emitRectClip emits a rectangle clip path from a <rect> element's
// x, y, width, and height attributes.
//
// Takes rc (*renderContext) which provides the PDF stream.
// Takes node (*Node) which is the rect element.
func emitRectClip(rc *renderContext, node *Node) {
	x := attrFloat(node, "x", 0)
	y := attrFloat(node, "y", 0)
	w := attrFloat(node, "width", 0)
	h := attrFloat(node, "height", 0)
	if w > 0 && h > 0 {
		rc.stream.Rectangle(x, y, w, h)
	}
}

// emitCircleClip emits an ellipse clip path from a <circle> element's
// cx, cy, and r attributes.
//
// Takes rc (*renderContext) which provides the PDF stream.
// Takes node (*Node) which is the circle element.
func emitCircleClip(rc *renderContext, node *Node) {
	cx := attrFloat(node, "cx", 0)
	cy := attrFloat(node, "cy", 0)
	r := attrFloat(node, "r", 0)
	if r > 0 {
		emitEllipse(rc.stream, cx, cy, r, r)
	}
}

// emitEllipseClip emits an ellipse clip path from an <ellipse> element's
// cx, cy, rx, and ry attributes.
//
// Takes rc (*renderContext) which provides the PDF stream.
// Takes node (*Node) which is the ellipse element.
func emitEllipseClip(rc *renderContext, node *Node) {
	cx := attrFloat(node, "cx", 0)
	cy := attrFloat(node, "cy", 0)
	rx := attrFloat(node, "rx", 0)
	ry := attrFloat(node, "ry", 0)
	if rx > 0 && ry > 0 {
		emitEllipse(rc.stream, cx, cy, rx, ry)
	}
}

// emitPolygonClip emits a polygon clip path from a <polygon> element's
// points attribute.
//
// Takes rc (*renderContext) which provides the PDF stream.
// Takes node (*Node) which is the polygon element.
func emitPolygonClip(rc *renderContext, node *Node) {
	points := parsePointsList(node.Attrs["points"])
	if len(points) >= minPolygonPoints {
		rc.stream.MoveTo(points[0], points[1])
		for i := 2; i+1 < len(points); i += 2 {
			rc.stream.LineTo(points[i], points[i+1])
		}
		rc.stream.ClosePath()
	}
}

// emitPathClip emits a clip path from a <path> element's d attribute
// by parsing the path data into move, line, curve, and close operators.
//
// Takes rc (*renderContext) which provides the PDF stream.
// Takes node (*Node) which is the path element.
func emitPathClip(rc *renderContext, node *Node) {
	d, ok := node.Attrs["d"]
	if !ok {
		return
	}
	commands, err := ParsePathData(d)
	if err != nil {
		return
	}
	for _, cmd := range commands {
		switch cmd.Type {
		case 'M':
			rc.stream.MoveTo(cmd.Args[0], cmd.Args[1])
		case 'L':
			rc.stream.LineTo(cmd.Args[0], cmd.Args[1])
		case 'C':
			rc.stream.CurveTo(cmd.Args[0], cmd.Args[1],
				cmd.Args[2], cmd.Args[3],
				cmd.Args[4], cmd.Args[5])
		case 'Z':
			rc.stream.ClosePath()
		}
	}
}

// emitUseClip resolves a <use> element's href and emits the referenced
// shape as a clip path.
//
// Takes rc (*renderContext) which provides the PDF stream and defs map.
// Takes node (*Node) which is the use element.
func emitUseClip(rc *renderContext, node *Node) {
	href := node.Attrs["href"]
	if href == "" {
		href = node.Attrs["xlink:href"]
	}
	href = trimHash(href)
	if target, ok := rc.defs[href]; ok {
		emitShapeAsPath(rc, target)
	}
}

// trimHash removes a leading '#' character from a string, returning
// the string unchanged if no '#' prefix is present.
//
// Takes s (string) which is the string to trim.
//
// Returns string which is the input without its leading '#'.
func trimHash(s string) string {
	if len(s) > 0 && s[0] == '#' {
		return s[1:]
	}
	return s
}
