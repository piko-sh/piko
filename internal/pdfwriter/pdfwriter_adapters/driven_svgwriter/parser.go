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
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	// millimetresPerPoint holds the number of millimetres per PDF point.
	millimetresPerPoint = 2.83465

	// centimetresPerPoint holds the number of centimetres per PDF point.
	centimetresPerPoint = 28.3465

	// pointsPerInch holds the number of PDF points per inch.
	pointsPerInch = 72.0

	// defaultEmSize holds the default em size in points for SVG em/rem units.
	defaultEmSize = 16.0

	// viewBoxPartCount holds the expected number of parts in an SVG viewBox attribute.
	viewBoxPartCount = 4
)

// ParseSVGString parses raw SVG XML into an SVG document.
//
// Takes svgXML (string) which is the raw SVG XML markup to parse.
//
// Returns *SVG which holds the parsed SVG document with dimensions and defs.
// Returns error when XML parsing fails or no <svg> element is found.
func ParseSVGString(svgXML string) (*SVG, error) {
	return parseSVGReader(strings.NewReader(svgXML))
}

// parseSVGReader parses SVG XML from a reader into an SVG document.
//
// Takes r (io.Reader) which provides the raw SVG XML bytes.
//
// Returns *SVG which holds the parsed document with extracted dimensions and defs.
// Returns error when XML parsing fails or no <svg> element is found.
func parseSVGReader(r io.Reader) (*SVG, error) {
	dec := xml.NewDecoder(r)
	dec.Strict = false
	dec.AutoClose = xml.HTMLAutoClose
	dec.Entity = xml.HTMLEntity

	root, err := parseElement(dec)
	if err != nil {
		return nil, fmt.Errorf("svg: parse error: %w", err)
	}
	if root == nil {
		return nil, errors.New("svg: empty document")
	}

	svgRoot := findSVGRoot(root)
	if svgRoot == nil {
		return nil, errors.New("svg: no <svg> element found")
	}

	s := &SVG{Root: svgRoot}
	s.extractDimensions()
	s.indexDefs()
	return s, nil
}

// extractDimensions reads width, height, viewBox, and preserveAspectRatio
// from the root SVG element and populates the corresponding SVG fields.
func (s *SVG) extractDimensions() {
	if s.Root == nil {
		return
	}
	if w, ok := s.Root.Attrs["width"]; ok {
		s.Width = parseDimension(w)
	}
	if h, ok := s.Root.Attrs["height"]; ok {
		s.Height = parseDimension(h)
	}
	if vb, ok := s.Root.Attrs["viewBox"]; ok {
		s.VBox = parseViewBox(vb)
	}
	if par, ok := s.Root.Attrs["preserveAspectRatio"]; ok {
		s.PreserveAspectRatio = parsePreserveAspectRatio(par)
	} else {
		s.PreserveAspectRatio = DefaultAspectRatio()
	}
}

// indexDefs recursively indexes all elements with an id attribute into
// the Defs map, including nested elements inside <defs> blocks and
// top-level ids.
func (s *SVG) indexDefs() {
	s.Defs = make(map[string]*Node)
	if s.Root == nil {
		return
	}
	indexDefsRecursive(s.Root, s.Defs)
}

// indexDefsRecursive walks the node tree and registers every element
// that has an id attribute into the defs map.
//
// Takes node (*Node) which is the current node to index.
// Takes defs (map[string]*Node) which accumulates id-to-node mappings.
func indexDefsRecursive(node *Node, defs map[string]*Node) {
	if id, ok := node.Attrs["id"]; ok && id != "" {
		defs[id] = node
	}
	for _, child := range node.Children {
		indexDefsRecursive(child, defs)
	}
}

// parseElement reads tokens from the decoder until a complete element
// is parsed or EOF is reached.
//
// Takes dec (*xml.Decoder) which provides the XML token stream.
//
// Returns *Node which holds the parsed element, or nil at EOF.
// Returns error when the decoder encounters invalid XML.
func parseElement(dec *xml.Decoder) (*Node, error) {
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}

		node, done, parseErr := handleElementToken(dec, tok)
		if parseErr != nil {
			return nil, parseErr
		}
		if done {
			return node, nil
		}
	}
}

// handleElementToken processes a single XML token during element parsing.
//
// Takes dec (*xml.Decoder) which provides the token stream for child parsing.
// Takes tok (xml.Token) which is the token to process.
//
// Returns *Node which holds the parsed node if a start element was found.
// Returns bool which indicates whether element parsing is complete.
// Returns error when child parsing fails.
func handleElementToken(dec *xml.Decoder, tok xml.Token) (*Node, bool, error) {
	switch t := tok.(type) {
	case xml.StartElement:
		node := &Node{
			Tag:       t.Name.Local,
			Attrs:     make(map[string]string),
			Transform: Identity(),
		}
		for _, attr := range t.Attr {
			node.Attrs[attr.Name.Local] = attr.Value
		}
		if tf, ok := node.Attrs["transform"]; ok {
			node.Transform = ParseTransform(tf)
		}
		if err := parseChildren(dec, node); err != nil {
			return nil, false, err
		}
		return node, true, nil

	case xml.EndElement:
		return nil, true, nil

	default:
		_ = t
		return nil, false, nil
	}
}

// parseChildren reads child elements and text content from the decoder
// until the parent element's closing tag is reached.
//
// Takes dec (*xml.Decoder) which provides the XML token stream.
// Takes parent (*Node) which accumulates parsed children and text.
//
// Returns error when child parsing encounters invalid XML.
func parseChildren(dec *xml.Decoder, parent *Node) error {
	var textBuf strings.Builder

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		done, childErr := handleChildToken(dec, parent, tok, &textBuf)
		if childErr != nil {
			return childErr
		}
		if done {
			return nil
		}
	}
}

// handleChildToken processes a single XML token during child parsing,
// building child nodes or accumulating character data.
//
// Takes dec (*xml.Decoder) which provides the token stream for recursive parsing.
// Takes parent (*Node) which accumulates child nodes.
// Takes tok (xml.Token) which is the token to process.
// Takes textBuf (*strings.Builder) which accumulates character data.
//
// Returns bool which indicates whether the parent's closing tag was reached.
// Returns error when recursive child parsing fails.
func handleChildToken(dec *xml.Decoder, parent *Node, tok xml.Token, textBuf *strings.Builder) (bool, error) {
	switch t := tok.(type) {
	case xml.StartElement:
		child := buildNodeFromStart(t)
		if err := parseChildren(dec, child); err != nil {
			return false, err
		}
		parent.Children = append(parent.Children, child)

	case xml.CharData:
		textBuf.Write(t)

	case xml.EndElement:
		text := strings.TrimSpace(textBuf.String())
		if text != "" {
			parent.Text = text
		}
		return true, nil

	default:
		_ = t
	}

	return false, nil
}

// buildNodeFromStart constructs a Node from an XML start element, copying
// attributes and parsing any transform attribute.
//
// Takes start (xml.StartElement) which is the XML start element to convert.
//
// Returns *Node which holds the constructed node with attributes and transform.
func buildNodeFromStart(start xml.StartElement) *Node {
	node := &Node{
		Tag:       start.Name.Local,
		Attrs:     make(map[string]string),
		Transform: Identity(),
	}
	for _, attr := range start.Attr {
		node.Attrs[attr.Name.Local] = attr.Value
	}
	if tf, ok := node.Attrs["transform"]; ok {
		node.Transform = ParseTransform(tf)
	}
	return node
}

// findSVGRoot searches the node tree for the first element with the
// tag "svg", returning nil if none is found.
//
// Takes node (*Node) which is the root of the tree to search.
//
// Returns *Node which is the first <svg> element, or nil if absent.
func findSVGRoot(node *Node) *Node {
	if node == nil {
		return nil
	}
	if node.Tag == "svg" {
		return node
	}
	for _, child := range node.Children {
		if found := findSVGRoot(child); found != nil {
			return found
		}
	}
	return nil
}

// parseDimension parses an SVG dimension string with an optional unit suffix.
// Supports px, pt, mm, cm, in, em, and rem; percentage suffixes are stripped.
//
// Takes s (string) which is the dimension string to parse.
//
// Returns float64 which is the dimension value converted to points, or 0 on failure.
func parseDimension(s string) float64 {
	s = strings.TrimSpace(s)

	type unitScale struct {
		suffix string
		factor float64
	}
	units := []unitScale{
		{"pt", 1.0},
		{"mm", millimetresPerPoint},
		{"cm", centimetresPerPoint},
		{"in", pointsPerInch},
		{"px", 1.0},
		{"em", defaultEmSize},
		{"rem", defaultEmSize},
	}

	for _, u := range units {
		if trimmed, found := strings.CutSuffix(s, u.suffix); found {
			trimmed = strings.TrimSpace(trimmed)
			v, err := strconv.ParseFloat(trimmed, 64)
			if err != nil {
				return 0
			}
			return v * u.factor
		}
	}

	s = strings.TrimSuffix(s, "%")
	s = strings.TrimSpace(s)

	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// parseViewBox parses an SVG viewBox attribute string into a ViewBox.
//
// Takes s (string) which is the viewBox value (e.g. "0 0 100 200").
//
// Returns ViewBox which holds the parsed minX, minY, width, and height.
func parseViewBox(s string) ViewBox {
	s = strings.TrimSpace(s)
	if s == "" {
		return ViewBox{}
	}
	s = strings.ReplaceAll(s, ",", " ")
	parts := strings.Fields(s)
	if len(parts) < viewBoxPartCount {
		return ViewBox{}
	}
	minX, e1 := strconv.ParseFloat(parts[0], 64)
	minY, e2 := strconv.ParseFloat(parts[1], 64)
	width, e3 := strconv.ParseFloat(parts[2], 64)
	height, e4 := strconv.ParseFloat(parts[3], 64)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
		return ViewBox{}
	}
	return ViewBox{
		MinX:   minX,
		MinY:   minY,
		Width:  width,
		Height: height,
		Valid:  width > 0 && height > 0,
	}
}

// parsePreserveAspectRatio parses an SVG preserveAspectRatio attribute
// into an AspectRatio, defaulting to "xMidYMid meet".
//
// Takes s (string) which is the preserveAspectRatio value.
//
// Returns AspectRatio which holds the parsed align and meetOrSlice values.
func parsePreserveAspectRatio(s string) AspectRatio {
	s = strings.TrimSpace(s)
	if s == "" {
		return DefaultAspectRatio()
	}
	parts := strings.Fields(s)
	result := AspectRatio{Align: parts[0], MeetOrSlice: "meet"}
	if len(parts) >= 2 {
		if parts[1] == "meet" || parts[1] == "slice" {
			result.MeetOrSlice = parts[1]
		}
	}
	return result
}
