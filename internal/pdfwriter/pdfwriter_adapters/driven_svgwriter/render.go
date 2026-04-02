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

import (
	"context"
	"math"
	"strconv"
	"strings"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
)

// kappa is the cubic Bezier control-point coefficient for a quarter-circle
// arc approximation (4*(sqrt(2)-1)/3).
const kappa = 0.5522847498

// SVGWriter implements pdfwriter_domain.SVGWriterPort by parsing SVG
// markup and emitting native PDF drawing commands.
type SVGWriter struct{}

var _ pdfwriter_domain.SVGWriterPort = (*SVGWriter)(nil)

// New creates a new SVGWriter.
//
// Returns *SVGWriter which holds the initialised renderer instance.
func New() *SVGWriter {
	return &SVGWriter{}
}

// RenderSVG parses the SVG string and renders it as native
// PDF vector commands into the provided content stream at the
// given position and size.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes svgData (string) which holds the raw SVG markup
// to parse.
// Takes svgRenderContext
// (pdfwriter_domain.SVGRenderContext) which provides the
// PDF stream and resource managers.
// Takes x (float64) which specifies the horizontal
// placement offset in PDF points.
// Takes y (float64) which specifies the vertical placement
// offset in PDF points.
// Takes renderW (float64) which specifies the target
// rendering width in PDF points.
// Takes renderH (float64) which specifies the target
// rendering height in PDF points.
//
// Returns error which indicates a parse failure if the SVG
// markup is malformed.
func (*SVGWriter) RenderSVG(ctx context.Context, svgData string, svgRenderContext pdfwriter_domain.SVGRenderContext, x, y, renderW, renderH float64) error {
	svg, err := ParseSVGString(svgData)
	if err != nil {
		return err
	}

	rc := &renderContext{
		stream:           svgRenderContext.Stream,
		shadingManager:   svgRenderContext.ShadingManager,
		extGStateManager: svgRenderContext.ExtGStateManager,
		fontEmbedder:     svgRenderContext.FontEmbedder,
		imageEmbedder:    svgRenderContext.ImageEmbedder,
		defs:             svg.Defs,
		registerFont:     svgRenderContext.RegisterFont,
		measureText:      svgRenderContext.MeasureText,
		getImageData:     svgRenderContext.GetImageData,
	}

	rc.stream.SaveState()

	rc.stream.ConcatMatrix(1, 0, 0, 1, x, y)

	applyViewportTransform(rc, svg, renderW, renderH)

	renderNode(ctx, rc, svg.Root, new(DefaultStyle()))

	rc.stream.RestoreState()
	return nil
}

// renderContext holds the state needed while traversing the SVG tree.
type renderContext struct {
	// stream holds the PDF content stream to write drawing commands into.
	stream *pdfwriter_domain.ContentStream

	// shadingManager holds the manager for registering gradient shading patterns.
	shadingManager *pdfwriter_domain.ShadingManager

	// extGStateManager holds the manager for registering
	// extended graphics state resources such as opacity.
	extGStateManager *pdfwriter_domain.ExtGStateManager

	// fontEmbedder holds the manager for embedding font resources into the PDF.
	fontEmbedder *pdfwriter_domain.FontEmbedder

	// imageEmbedder holds the manager for embedding raster image resources into the PDF.
	imageEmbedder *pdfwriter_domain.ImageEmbedder

	// defs holds the map of reusable SVG element definitions keyed by ID.
	defs map[string]*Node

	// registerFont holds the callback that registers a font
	// and returns its PDF resource name.
	registerFont func(family string, weight int, style int, size float64) string

	// measureText holds the callback that measures the rendered width of a text string.
	measureText func(family string, weight int, style int, size float64, text string) float64

	// getImageData holds the callback that retrieves image
	// data and format from a source reference.
	getImageData func(ctx context.Context, source string) ([]byte, string, error)
}

// applyViewportTransform applies the viewport-to-viewBox
// coordinate transformation including aspect ratio handling.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes svg (*SVG) which holds the parsed SVG document.
// Takes renderW (float64) which specifies the target
// rendering width in PDF points.
// Takes renderH (float64) which specifies the target
// rendering height in PDF points.
func applyViewportTransform(rc *renderContext, svg *SVG, renderW, renderH float64) {
	vbox := svg.VBox
	if !vbox.Valid {
		vbox = computeDefaultViewBox(svg)
		if !vbox.Valid {
			return
		}
	}

	scaleX := renderW / vbox.Width
	scaleY := renderH / vbox.Height
	translateX := 0.0
	translateY := 0.0

	par := svg.PreserveAspectRatio
	if par.Align != "none" {
		scaleX, scaleY, translateX, translateY = computeAspectRatioTransform(
			par, scaleX, scaleY, renderW, renderH, vbox,
		)
	}

	rc.stream.ConcatMatrix(scaleX, 0, 0, -scaleY,
		translateX, translateY+scaleY*vbox.Height)
	rc.stream.ConcatMatrix(1, 0, 0, 1, -vbox.MinX, -vbox.MinY)
}

// computeDefaultViewBox constructs a fallback viewBox from
// the SVG intrinsic dimensions.
//
// Takes svg (*SVG) which holds the parsed SVG document.
//
// Returns ViewBox which holds the computed fallback viewBox.
func computeDefaultViewBox(svg *SVG) ViewBox {
	svgW := svg.IntrinsicWidth()
	svgH := svg.IntrinsicHeight()
	if svgW <= 0 || svgH <= 0 {
		return ViewBox{}
	}
	return ViewBox{MinX: 0, MinY: 0, Width: svgW, Height: svgH, Valid: true}
}

// computeAspectRatioTransform calculates the uniform scale
// and translation for preserveAspectRatio alignment.
//
// Takes par (AspectRatio) which holds the parsed
// preserveAspectRatio value.
// Takes scaleX (float64) which specifies the initial
// horizontal scale factor.
// Takes scaleY (float64) which specifies the initial
// vertical scale factor.
// Takes renderW (float64) which specifies the target
// rendering width in PDF points.
// Takes renderH (float64) which specifies the target
// rendering height in PDF points.
// Takes vbox (ViewBox) which holds the SVG viewBox
// dimensions.
//
// Returns finalScaleX (float64) which holds the computed
// horizontal scale.
// Returns finalScaleY (float64) which holds the computed
// vertical scale.
// Returns translateX (float64) which holds the horizontal
// alignment offset.
// Returns translateY (float64) which holds the vertical
// alignment offset.
func computeAspectRatioTransform(
	par AspectRatio, scaleX, scaleY, renderW, renderH float64, vbox ViewBox,
) (finalScaleX float64, finalScaleY float64, translateX float64, translateY float64) {
	scale := scaleX
	if par.MeetOrSlice == "meet" {
		scale = math.Min(scaleX, scaleY)
	} else {
		scale = math.Max(scaleX, scaleY)
	}

	scaledW := vbox.Width * scale
	scaledH := vbox.Height * scale

	switch {
	case strings.HasPrefix(par.Align, "xMin"):
		translateX = 0
	case strings.HasPrefix(par.Align, "xMid"):
		translateX = (renderW - scaledW) / 2
	case strings.HasPrefix(par.Align, "xMax"):
		translateX = renderW - scaledW
	}

	switch {
	case strings.HasSuffix(par.Align, "YMin"):
		translateY = 0
	case strings.HasSuffix(par.Align, "YMid"):
		translateY = (renderH - scaledH) / 2
	case strings.HasSuffix(par.Align, "YMax"):
		translateY = renderH - scaledH
	}

	return scale, scale, translateX, translateY
}

// renderNode dispatches rendering for a single SVG node
// based on its tag type.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes node (*Node) which holds the SVG node to render.
// Takes parentStyle (*Style) which holds the inherited
// style properties.
func renderNode(ctx context.Context, rc *renderContext, node *Node, parentStyle *Style) {
	if node == nil {
		return
	}

	style := ResolveStyle(node, parentStyle)

	if style.Display == "none" {
		return
	}

	switch node.Tag {
	case "defs", "title", "desc", "metadata":
		return
	case "svg", "g":
		renderGroup(ctx, rc, node, &style)
	case "use":
		renderUse(ctx, rc, node, &style)
	default:
		renderShapeOrContent(ctx, rc, node, &style)
	}
}

// renderGroup renders an SVG group or nested SVG element
// by applying its transform and rendering children.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes node (*Node) which holds the group node to render.
// Takes style (*Style) which holds the resolved style
// properties.
func renderGroup(ctx context.Context, rc *renderContext, node *Node, style *Style) {
	rc.stream.SaveState()
	if !node.Transform.IsIdentity() {
		m := node.Transform
		rc.stream.ConcatMatrix(m.A, m.B, m.C, m.D, m.E, m.F)
	}
	applyGroupOpacity(rc, style)
	applyClipPath(rc, node, style)
	for _, child := range node.Children {
		renderNode(ctx, rc, child, style)
	}
	rc.stream.RestoreState()
}

// renderShapeOrContent renders individual SVG shape or
// content elements such as rect, circle, path, and text.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes node (*Node) which holds the shape node to render.
// Takes style (*Style) which holds the resolved style
// properties.
func renderShapeOrContent(ctx context.Context, rc *renderContext, node *Node, style *Style) {
	switch node.Tag {
	case "rect":
		rc.stream.SaveState()
		applyTransform(rc, node)
		renderRect(rc, node, style)
		rc.stream.RestoreState()
	case "circle":
		rc.stream.SaveState()
		applyTransform(rc, node)
		renderCircle(rc, node, style)
		rc.stream.RestoreState()
	case "ellipse":
		rc.stream.SaveState()
		applyTransform(rc, node)
		renderEllipse(rc, node, style)
		rc.stream.RestoreState()
	case "line":
		rc.stream.SaveState()
		applyTransform(rc, node)
		renderLine(rc, node, style)
		rc.stream.RestoreState()
	case "polyline":
		rc.stream.SaveState()
		applyTransform(rc, node)
		renderPolyline(rc, node, style, false)
		rc.stream.RestoreState()
	case "polygon":
		rc.stream.SaveState()
		applyTransform(rc, node)
		renderPolyline(rc, node, style, true)
		rc.stream.RestoreState()
	case "path":
		rc.stream.SaveState()
		applyTransform(rc, node)
		renderPath(rc, node, style)
		rc.stream.RestoreState()
	case "text":
		rc.stream.SaveState()
		applyTransform(rc, node)
		renderText(rc, node, style)
		rc.stream.RestoreState()
	case "image":
		rc.stream.SaveState()
		applyTransform(rc, node)
		renderImage(ctx, rc, node)
		rc.stream.RestoreState()
	}
}

// applyTransform concatenates the node transform matrix to
// the current graphics state if it is not the identity.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes node (*Node) which holds the node whose transform
// to apply.
func applyTransform(rc *renderContext, node *Node) {
	if !node.Transform.IsIdentity() {
		m := node.Transform
		rc.stream.ConcatMatrix(m.A, m.B, m.C, m.D, m.E, m.F)
	}
}

// applyGroupOpacity registers an extended graphics state
// for the group opacity if it is less than fully opaque.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes style (*Style) which holds the resolved style
// properties including opacity.
func applyGroupOpacity(rc *renderContext, style *Style) {
	if style.Opacity < 1 {
		gsName := rc.extGStateManager.RegisterOpacity(style.Opacity)
		rc.stream.SetExtGState(gsName)
	}
}

// renderUse renders an SVG use element by resolving its
// href to a definition and rendering the referenced node.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes node (*Node) which holds the use element to
// resolve.
// Takes parentStyle (*Style) which holds the inherited
// style properties.
func renderUse(ctx context.Context, rc *renderContext, node *Node, parentStyle *Style) {
	href := node.Attrs["href"]
	if href == "" {
		href = node.Attrs["xlink:href"]
	}
	href = strings.TrimPrefix(href, "#")
	if href == "" {
		return
	}
	target, ok := rc.defs[href]
	if !ok {
		return
	}

	rc.stream.SaveState()
	applyTransform(rc, node)

	x := attrFloat(node, "x", 0)
	y := attrFloat(node, "y", 0)
	if x != 0 || y != 0 {
		rc.stream.ConcatMatrix(1, 0, 0, 1, x, y)
	}

	renderNode(ctx, rc, target, parentStyle)
	rc.stream.RestoreState()
}

// renderRect renders an SVG rect element, handling
// optional rounded corners.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes node (*Node) which holds the rect element to
// render.
// Takes style (*Style) which holds the resolved style
// properties.
func renderRect(rc *renderContext, node *Node, style *Style) {
	x := attrFloat(node, "x", 0)
	y := attrFloat(node, "y", 0)
	w := attrFloat(node, "width", 0)
	h := attrFloat(node, "height", 0)
	if w <= 0 || h <= 0 {
		return
	}

	rx := attrFloat(node, "rx", 0)
	ry := attrFloat(node, "ry", 0)
	if rx < 0 {
		rx = 0
	}
	if ry < 0 {
		ry = 0
	}

	if rx > 0 && ry == 0 {
		ry = rx
	}
	if ry > 0 && rx == 0 {
		rx = ry
	}
	rx = math.Min(rx, w/2)
	ry = math.Min(ry, h/2)

	if rx == 0 && ry == 0 {
		rc.stream.Rectangle(x, y, w, h)
	} else {
		emitRoundedRect(rc.stream, x, y, w, h, rx, ry)
	}
	paintPath(rc, style)
}

// renderCircle renders an SVG circle element as a
// Bezier-approximated ellipse.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes node (*Node) which holds the circle element to
// render.
// Takes style (*Style) which holds the resolved style
// properties.
func renderCircle(rc *renderContext, node *Node, style *Style) {
	cx := attrFloat(node, "cx", 0)
	cy := attrFloat(node, "cy", 0)
	r := attrFloat(node, "r", 0)
	if r <= 0 {
		return
	}
	emitEllipse(rc.stream, cx, cy, r, r)
	paintPath(rc, style)
}

// renderEllipse renders an SVG ellipse element as a
// Bezier-approximated path.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes node (*Node) which holds the ellipse element to
// render.
// Takes style (*Style) which holds the resolved style
// properties.
func renderEllipse(rc *renderContext, node *Node, style *Style) {
	cx := attrFloat(node, "cx", 0)
	cy := attrFloat(node, "cy", 0)
	rx := attrFloat(node, "rx", 0)
	ry := attrFloat(node, "ry", 0)
	if rx <= 0 || ry <= 0 {
		return
	}
	emitEllipse(rc.stream, cx, cy, rx, ry)
	paintPath(rc, style)
}

// renderLine renders an SVG line element as a stroked path
// between two points.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes node (*Node) which holds the line element to
// render.
// Takes style (*Style) which holds the resolved style
// properties.
func renderLine(rc *renderContext, node *Node, style *Style) {
	x1 := attrFloat(node, "x1", 0)
	y1 := attrFloat(node, "y1", 0)
	x2 := attrFloat(node, "x2", 0)
	y2 := attrFloat(node, "y2", 0)
	rc.stream.MoveTo(x1, y1)
	rc.stream.LineTo(x2, y2)

	applyStrokeStyle(rc, style)
	rc.stream.Stroke()
}

// renderPolyline renders an SVG polyline or polygon element
// from its points attribute.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes node (*Node) which holds the polyline element to
// render.
// Takes style (*Style) which holds the resolved style
// properties.
// Takes closed (bool) which controls whether the path is
// closed as a polygon.
func renderPolyline(rc *renderContext, node *Node, style *Style, closed bool) {
	points := parsePointsList(node.Attrs["points"])
	if len(points) < 2 {
		return
	}
	rc.stream.MoveTo(points[0], points[1])
	for i := 2; i+1 < len(points); i += 2 {
		rc.stream.LineTo(points[i], points[i+1])
	}
	if closed {
		rc.stream.ClosePath()
	}
	paintPath(rc, style)
}

// renderPath renders an SVG path element by parsing its d
// attribute into PDF path commands.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes node (*Node) which holds the path element to
// render.
// Takes style (*Style) which holds the resolved style
// properties.
func renderPath(rc *renderContext, node *Node, style *Style) {
	d, ok := node.Attrs["d"]
	if !ok || d == "" {
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
	paintPath(rc, style)
}

// renderImage renders an SVG <image> element by embedding the
// referenced raster image as a PDF XObject.
//
// Takes ctx (context.Context) which controls cancellation of image data retrieval.
// Takes rc (*renderContext) which provides the PDF stream and resource managers.
func renderImage(ctx context.Context, rc *renderContext, node *Node) {
	if rc.getImageData == nil || rc.imageEmbedder == nil {
		return
	}

	source := node.Attrs["href"]
	if source == "" {
		source = node.Attrs["xlink:href"]
	}
	if source == "" {
		return
	}

	x := attrFloat(node, "x", 0)
	y := attrFloat(node, "y", 0)
	w := attrFloat(node, "width", 0)
	h := attrFloat(node, "height", 0)
	if w <= 0 || h <= 0 {
		return
	}

	data, format, err := rc.getImageData(ctx, source)
	if err != nil || len(data) == 0 {
		return
	}

	pixelW, pixelH := pdfwriter_domain.ExtractImageDimensions(data, format)
	if pixelW == 0 || pixelH == 0 {
		return
	}

	resourceName := rc.imageEmbedder.RegisterImage(source, data, format, pixelW, pixelH)

	rc.stream.SaveState()

	rc.stream.ConcatMatrix(w, 0, 0, -h, x, y+h)
	rc.stream.PaintXObject(resourceName)
	rc.stream.RestoreState()
}

// paintPath applies the appropriate fill and stroke
// painting operations for the current path.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes style (*Style) which holds the resolved style
// properties.
func paintPath(rc *renderContext, style *Style) {
	hasFill := style.Fill != nil
	hasGradientFill := style.FillRef != ""
	hasStroke := style.Stroke != nil && style.StrokeWidth > 0

	if hasGradientFill {
		paintGradientPath(rc, style, hasStroke)
		return
	}

	paintSolidPath(rc, style, hasFill, hasStroke)
}

// paintGradientPath attempts to paint the path with a
// gradient fill, falling back to solid painting on failure.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes style (*Style) which holds the resolved style
// properties.
// Takes hasStroke (bool) which indicates whether a stroke
// should also be applied.
func paintGradientPath(rc *renderContext, style *Style, hasStroke bool) {
	rc.stream.SaveState()
	if hasStroke {
		applyStrokeStyle(rc, style)
	}
	if resolveGradientFill(rc, style.FillRef, style) {
		rc.stream.RestoreState()
		return
	}
	rc.stream.RestoreState()

	hasFill := style.Fill != nil
	paintSolidPath(rc, style, hasFill, hasStroke)
}

// paintSolidPath applies solid fill and stroke colours and
// issues the appropriate PDF paint operator.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes style (*Style) which holds the resolved style
// properties.
// Takes hasFill (bool) which indicates whether a fill
// should be applied.
// Takes hasStroke (bool) which indicates whether a stroke
// should be applied.
func paintSolidPath(rc *renderContext, style *Style, hasFill, hasStroke bool) {
	if hasFill {
		applyFillStyle(rc, style)
	}
	if hasStroke {
		applyStrokeStyle(rc, style)
	}

	switch {
	case hasFill && hasStroke:
		if style.FillRule == "evenodd" {
			rc.stream.FillEvenOddAndStroke()
		} else {
			rc.stream.FillAndStroke()
		}
	case hasFill:
		if style.FillRule == "evenodd" {
			rc.stream.FillEvenOdd()
		} else {
			rc.stream.Fill()
		}
	case hasStroke:
		rc.stream.Stroke()
	default:
		rc.stream.EndPath()
	}
}

// applyFillStyle sets the PDF fill colour and opacity from
// the resolved SVG style.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes style (*Style) which holds the resolved style
// properties including fill colour and opacity.
func applyFillStyle(rc *renderContext, style *Style) {
	if style.Fill == nil {
		return
	}
	c := *style.Fill
	alpha := c.A * style.FillOpacity * style.Opacity
	rc.stream.SetFillColourRGB(c.R, c.G, c.B)
	if alpha < 1 {
		gsName := rc.extGStateManager.RegisterOpacity(alpha)
		rc.stream.SetExtGState(gsName)
	}
}

// applyStrokeStyle sets the PDF stroke colour, line width,
// cap, join, dash pattern, and opacity from the resolved
// SVG style.
//
// Takes rc (*renderContext) which provides the PDF stream
// and resource managers.
// Takes style (*Style) which holds the resolved style
// properties including stroke attributes.
func applyStrokeStyle(rc *renderContext, style *Style) {
	if style.Stroke == nil {
		return
	}
	c := *style.Stroke
	alpha := c.A * style.StrokeOpacity * style.Opacity
	rc.stream.SetStrokeColourRGB(c.R, c.G, c.B)
	rc.stream.SetLineWidth(style.StrokeWidth)

	switch style.StrokeLineCap {
	case "round":
		rc.stream.SetLineCap(1)
	case "square":
		rc.stream.SetLineCap(2)
	default:
		rc.stream.SetLineCap(0)
	}

	switch style.StrokeLineJoin {
	case "round":
		rc.stream.SetLineJoin(1)
	case "bevel":
		rc.stream.SetLineJoin(2)
	default:
		rc.stream.SetLineJoin(0)
		if style.StrokeMitreLimit > 0 {
			rc.stream.SetMiterLimit(style.StrokeMitreLimit)
		}
	}

	if len(style.StrokeDashArray) > 0 {
		rc.stream.SetDashPattern(style.StrokeDashArray, style.StrokeDashOffset)
	}

	if alpha < 1 {
		gsName := rc.extGStateManager.RegisterOpacity(alpha)
		rc.stream.SetExtGState(gsName)
	}
}

// emitEllipse emits four cubic Bezier curves approximating
// an ellipse centred at (cx, cy) with radii rx and ry.
//
// Takes stream (*pdfwriter_domain.ContentStream) which
// holds the PDF content stream to write into.
// Takes cx (float64) which specifies the centre x
// coordinate.
// Takes cy (float64) which specifies the centre y
// coordinate.
// Takes rx (float64) which specifies the horizontal radius.
// Takes ry (float64) which specifies the vertical radius.
func emitEllipse(stream *pdfwriter_domain.ContentStream, cx, cy, rx, ry float64) {
	kx := kappa * rx
	ky := kappa * ry
	stream.MoveTo(cx+rx, cy)
	stream.CurveTo(cx+rx, cy+ky, cx+kx, cy+ry, cx, cy+ry)
	stream.CurveTo(cx-kx, cy+ry, cx-rx, cy+ky, cx-rx, cy)
	stream.CurveTo(cx-rx, cy-ky, cx-kx, cy-ry, cx, cy-ry)
	stream.CurveTo(cx+kx, cy-ry, cx+rx, cy-ky, cx+rx, cy)
	stream.ClosePath()
}

// emitRoundedRect emits a rectangle path with rounded
// corners using cubic Bezier curves.
//
// Takes stream (*pdfwriter_domain.ContentStream) which
// holds the PDF content stream to write into.
// Takes x (float64) which specifies the left edge.
// Takes y (float64) which specifies the top edge.
// Takes w (float64) which specifies the rectangle width.
// Takes h (float64) which specifies the rectangle height.
// Takes rx (float64) which specifies the horizontal corner
// radius.
// Takes ry (float64) which specifies the vertical corner
// radius.
func emitRoundedRect(stream *pdfwriter_domain.ContentStream, x, y, w, h, rx, ry float64) {
	kx := kappa * rx
	ky := kappa * ry

	stream.MoveTo(x+rx, y)

	stream.LineTo(x+w-rx, y)
	stream.CurveTo(x+w-rx+kx, y, x+w, y+ry-ky, x+w, y+ry)

	stream.LineTo(x+w, y+h-ry)
	stream.CurveTo(x+w, y+h-ry+ky, x+w-rx+kx, y+h, x+w-rx, y+h)

	stream.LineTo(x+rx, y+h)
	stream.CurveTo(x+rx-kx, y+h, x, y+h-ry+ky, x, y+h-ry)

	stream.LineTo(x, y+ry)
	stream.CurveTo(x, y+ry-ky, x+rx-kx, y, x+rx, y)
	stream.ClosePath()
}

// attrFloat reads a named attribute from the node and
// parses it as a float, returning the fallback on failure.
//
// Takes node (*Node) which holds the element to read from.
// Takes name (string) which specifies the attribute name.
// Takes fallback (float64) which specifies the default
// value if the attribute is missing or invalid.
//
// Returns float64 which holds the parsed attribute value
// or the fallback.
func attrFloat(node *Node, name string, fallback float64) float64 {
	s, ok := node.Attrs[name]
	if !ok {
		return fallback
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return fallback
	}
	return v
}

// parsePointsList parses a space-or-comma-separated list
// of float coordinates from an SVG points attribute.
//
// Takes s (string) which holds the raw points attribute
// value.
//
// Returns []float64 which holds the parsed coordinate
// values.
func parsePointsList(s string) []float64 {
	s = strings.ReplaceAll(s, ",", " ")
	parts := strings.Fields(s)
	result := make([]float64, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.ParseFloat(p, 64)
		if err != nil {
			continue
		}
		result = append(result, v)
	}
	return result
}
