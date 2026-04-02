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

package driven_transform_watermark

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	// defaultPriority is the execution order for the watermark transformer.
	defaultPriority = 150

	// fontName is the resource name for the watermark Helvetica font.
	fontName = "FWM"

	// gsName is the resource name for the watermark ExtGState.
	gsName = "GSWM"

	// defaultFontSize is the default font size in points.
	defaultFontSize = 60.0

	// defaultColour is the default grey channel for watermark text.
	defaultColour = 0.85

	// defaultAngle is the default rotation angle in degrees.
	defaultAngle = 45.0

	// defaultOpacity is the default opacity.
	defaultOpacity = 0.3

	// charWidthFactor is the approximate character width as a fraction of
	// font size for Helvetica.
	charWidthFactor = 0.52

	// mediaBoxElements is the number of elements in a PDF MediaBox array.
	mediaBoxElements = 4

	// defaultPageWidth is the US Letter width in points.
	defaultPageWidth = 612.0

	// defaultPageHeight is the US Letter height in points.
	defaultPageHeight = 792.0

	// degreesPerRadian converts degrees to radians.
	degreesPerRadian = 180.0

	// contentsKey is the PDF dictionary key for page content streams.
	contentsKey = "Contents"
)

// WatermarkTransformer overlays a text watermark on each page of a PDF
// document. It parses the PDF, locates each page object, prepends a
// watermark content stream (rendered behind existing content), and adds
// the required font and graphics state resources.
//
// The watermark uses Helvetica (a built-in Type1 font) so no font
// embedding is needed.
type WatermarkTransformer struct {
	// name is the transformer identifier.
	name string

	// priority is the execution order.
	priority int
}

var _ pdfwriter_domain.PdfTransformerPort = (*WatermarkTransformer)(nil)

// New creates a new watermark transformer with default name and priority.
//
// Returns *WatermarkTransformer which is ready for use.
func New() *WatermarkTransformer {
	return &WatermarkTransformer{
		name:     "watermark",
		priority: defaultPriority,
	}
}

// Name returns the transformer's name.
//
// Returns string which identifies this transformer.
func (t *WatermarkTransformer) Name() string { return t.name }

// Type returns TransformerContent.
//
// Returns pdfwriter_dto.TransformerType which categorises this as a content
// transformer.
func (*WatermarkTransformer) Type() pdfwriter_dto.TransformerType {
	return pdfwriter_dto.TransformerContent
}

// Priority returns the execution priority.
//
// Returns int which is the transformer's position in the processing order.
func (t *WatermarkTransformer) Priority() int { return t.priority }

// Transform applies a text watermark to each page of the PDF. Options
// must be WatermarkOptions or *WatermarkOptions.
//
// Takes pdf ([]byte) which is the input PDF document.
// Takes options (any) which must be WatermarkOptions or *WatermarkOptions.
//
// Returns []byte which is the watermarked PDF.
// Returns error when the PDF cannot be parsed or the watermark cannot be
// applied.
func (*WatermarkTransformer) Transform(_ context.Context, pdf []byte, options any) ([]byte, error) {
	opts, err := castOptions(options)
	if err != nil {
		return nil, err
	}
	if opts.Text == "" {
		return pdf, nil
	}

	applyDefaults(&opts)

	doc, err := pdfparse.Parse(pdf)
	if err != nil {
		return nil, fmt.Errorf("watermark: parsing PDF: %w", err)
	}

	writer, err := pdfparse.NewWriterFromDocument(doc)
	if err != nil {
		return nil, fmt.Errorf("watermark: creating writer: %w", err)
	}

	pageRefs, err := collectPageRefs(doc)
	if err != nil {
		return nil, fmt.Errorf("watermark: collecting pages: %w", err)
	}

	pagesSet := buildPageSet(opts.Pages)

	for i, pageNum := range pageRefs {
		if len(pagesSet) > 0 && !pagesSet[i] {
			continue
		}
		if err := applyToPage(writer, doc, pageNum, &opts); err != nil {
			return nil, fmt.Errorf("watermark: page %d (obj %d): %w", i, pageNum, err)
		}
	}

	output, err := writer.Write()
	if err != nil {
		return nil, fmt.Errorf("watermark: writing PDF: %w", err)
	}
	return output, nil
}

// castOptions extracts WatermarkOptions from the generic options.
//
// Takes options (any) which is the untyped options value to assert.
//
// Returns pdfwriter_dto.WatermarkOptions which holds the typed options.
// Returns error when the options type does not match.
func castOptions(options any) (pdfwriter_dto.WatermarkOptions, error) {
	switch v := options.(type) {
	case pdfwriter_dto.WatermarkOptions:
		return v, nil
	case *pdfwriter_dto.WatermarkOptions:
		if v == nil {
			return pdfwriter_dto.WatermarkOptions{}, errors.New("watermark: nil options pointer")
		}
		return *v, nil
	default:
		return pdfwriter_dto.WatermarkOptions{}, fmt.Errorf("watermark: expected WatermarkOptions, got %T", options)
	}
}

// applyDefaults fills zero-value fields with sensible defaults.
//
// Takes opts (*pdfwriter_dto.WatermarkOptions) which is the options struct
// to populate.
func applyDefaults(opts *pdfwriter_dto.WatermarkOptions) {
	if opts.FontSize == 0 {
		opts.FontSize = defaultFontSize
	}
	if opts.ColourR == 0 && opts.ColourG == 0 && opts.ColourB == 0 {
		opts.ColourR = defaultColour
		opts.ColourG = defaultColour
		opts.ColourB = defaultColour
	}
	if opts.Angle == 0 {
		opts.Angle = defaultAngle
	}
	if opts.Opacity == 0 {
		opts.Opacity = defaultOpacity
	}
}

// buildPageSet converts a page index slice to a set for O(1) lookup.
//
// An empty slice means "all pages".
//
// Takes pages ([]int) which specifies the zero-based page indices to include.
//
// Returns map[int]bool which holds the page index set, or nil when all pages
// are selected.
func buildPageSet(pages []int) map[int]bool {
	if len(pages) == 0 {
		return nil
	}
	set := make(map[int]bool, len(pages))
	for _, p := range pages {
		set[p] = true
	}
	return set
}

// collectPageRefs walks the page tree and returns object numbers for all
// leaf Page objects in document order.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
//
// Returns []int which holds the object numbers of leaf Page objects.
// Returns error when the page tree cannot be traversed.
func collectPageRefs(doc *pdfparse.Document) ([]int, error) {
	trailer := doc.Trailer()
	rootRef := trailer.GetRef("Root")
	if rootRef.Number == 0 {
		return nil, errors.New("no /Root in trailer")
	}

	catalog, err := doc.GetObject(rootRef.Number)
	if err != nil {
		return nil, fmt.Errorf("reading catalog: %w", err)
	}

	catalogDict, ok := catalog.Value.(pdfparse.Dict)
	if !ok {
		return nil, errors.New("catalog is not a dictionary")
	}

	pagesRef := catalogDict.GetRef("Pages")
	if pagesRef.Number == 0 {
		return nil, errors.New("no /Pages in catalog")
	}

	return walkPageTree(doc, pagesRef.Number)
}

// walkPageTree recursively collects leaf Page object numbers from a Pages
// tree node.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes objNum (int) which is the object number of the current tree node.
//
// Returns []int which holds the collected leaf Page object numbers.
// Returns error when a node cannot be read or is not a dictionary.
func walkPageTree(doc *pdfparse.Document, objNum int) ([]int, error) {
	obj, err := doc.GetObject(objNum)
	if err != nil {
		return nil, err
	}

	dict, ok := obj.Value.(pdfparse.Dict)
	if !ok {
		return nil, fmt.Errorf("object %d is not a dictionary", objNum)
	}

	nodeType := dict.GetName("Type")
	if nodeType == "Page" {
		return []int{objNum}, nil
	}

	kids := dict.GetArray("Kids")
	var pages []int
	for _, kid := range kids {
		ref, ok := kid.Value.(pdfparse.Ref)
		if !ok {
			continue
		}
		childPages, err := walkPageTree(doc, ref.Number)
		if err != nil {
			return nil, err
		}
		pages = append(pages, childPages...)
	}
	return pages, nil
}

// applyToPage adds a watermark content stream to a single page object and
// updates its resources.
//
// Takes writer (*pdfparse.Writer) which is the PDF writer for mutations.
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes pageObjNum (int) which is the object number of the page to modify.
// Takes opts (*pdfwriter_dto.WatermarkOptions) which specifies the watermark
// parameters.
//
// Returns error when the page object is not a dictionary.
func applyToPage(
	writer *pdfparse.Writer,
	doc *pdfparse.Document,
	pageObjNum int,
	opts *pdfwriter_dto.WatermarkOptions,
) error {
	pageObj := writer.GetObject(pageObjNum)
	pageDict, ok := pageObj.Value.(pdfparse.Dict)
	if !ok {
		return errors.New("page object is not a dictionary")
	}

	pageWidth, pageHeight := extractMediaBox(pageDict, doc)

	stream := buildContentStream(opts, pageWidth, pageHeight)
	streamObjNum := writer.AddObject(pdfparse.StreamObj(pdfparse.Dict{}, []byte(stream)))

	prependContentStream(&pageDict, streamObjNum)
	addResources(&pageDict, opts.Opacity)

	writer.SetObject(pageObjNum, pdfparse.DictObj(pageDict))
	return nil
}

// extractMediaBox reads the /MediaBox from a page dictionary.
//
// If not found directly, it tries to resolve inherited /MediaBox from the
// parent. Falls back to US Letter (612 x 792) if nothing is found.
//
// Takes pageDict (pdfparse.Dict) which is the page dictionary to inspect.
// Takes doc (*pdfparse.Document) which is the parsed PDF document for
// resolving inherited values.
//
// Returns width (float64) which is the page width in points.
// Returns height (float64) which is the page height in points.
func extractMediaBox(pageDict pdfparse.Dict, doc *pdfparse.Document) (width float64, height float64) {
	mediaBox := pageDict.GetArray("MediaBox")
	if mediaBox == nil {
		mediaBox = resolveInheritedMediaBox(pageDict, doc)
	}
	if len(mediaBox) >= mediaBoxElements {
		return objectToFloat(mediaBox[2]) - objectToFloat(mediaBox[0]),
			objectToFloat(mediaBox[3]) - objectToFloat(mediaBox[1])
	}
	return defaultPageWidth, defaultPageHeight
}

// resolveInheritedMediaBox walks up the /Parent chain to find an inherited
// /MediaBox.
//
// Takes pageDict (pdfparse.Dict) which is the current page or pages node
// dictionary.
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
//
// Returns []pdfparse.Object which holds the MediaBox array entries, or nil
// when no inherited MediaBox is found.
func resolveInheritedMediaBox(pageDict pdfparse.Dict, doc *pdfparse.Document) []pdfparse.Object {
	parentRef := pageDict.GetRef("Parent")
	if parentRef.Number == 0 {
		return nil
	}
	parent, err := doc.GetObject(parentRef.Number)
	if err != nil {
		return nil
	}
	parentDict, ok := parent.Value.(pdfparse.Dict)
	if !ok {
		return nil
	}
	mediaBox := parentDict.GetArray("MediaBox")
	if mediaBox != nil {
		return mediaBox
	}
	return resolveInheritedMediaBox(parentDict, doc)
}

// objectToFloat extracts a numeric value from a PDF Object.
//
// Takes obj (pdfparse.Object) which is the PDF object to convert.
//
// Returns float64 which is the numeric value, or 0 when the object is not
// numeric.
func objectToFloat(obj pdfparse.Object) float64 {
	switch obj.Type {
	case pdfparse.ObjectInteger:
		if v, ok := obj.Value.(int64); ok {
			return float64(v)
		}
	case pdfparse.ObjectReal:
		if v, ok := obj.Value.(float64); ok {
			return v
		}
	}
	return 0
}

// buildContentStream generates PDF content stream operators for a diagonal
// text watermark.
//
// Takes opts (*pdfwriter_dto.WatermarkOptions) which specifies the watermark
// text, font size, colour, angle, and opacity.
// Takes pageWidth (float64) which is the page width in points.
// Takes pageHeight (float64) which is the page height in points.
//
// Returns string which holds the PDF content stream operators.
func buildContentStream(opts *pdfwriter_dto.WatermarkOptions, pageWidth, pageHeight float64) string {
	var buf strings.Builder

	rad := opts.Angle * math.Pi / degreesPerRadian
	cosA := math.Cos(rad)
	sinA := math.Sin(rad)

	textWidth := float64(len(opts.Text)) * opts.FontSize * charWidthFactor
	cx := pageWidth/2 - (textWidth/2)*cosA
	cy := pageHeight/2 - (textWidth/2)*sinA

	buf.WriteString("q\n")
	fmt.Fprintf(&buf, "/%s gs\n", gsName)
	buf.WriteString("BT\n")
	fmt.Fprintf(&buf, "/%s %g Tf\n", fontName, opts.FontSize)
	fmt.Fprintf(&buf, "%g %g %g rg\n", opts.ColourR, opts.ColourG, opts.ColourB)
	fmt.Fprintf(&buf, "%g %g %g %g %g %g cm\n", cosA, sinA, -sinA, cosA, cx, cy)
	fmt.Fprintf(&buf, "(%s) Tj\n", escapeText(opts.Text))
	buf.WriteString("ET\n")
	buf.WriteString("Q\n")

	return buf.String()
}

// escapeText escapes special characters in a PDF literal string.
//
// Takes text (string) which is the raw text to escape.
//
// Returns string which holds the escaped text safe for PDF literal strings.
func escapeText(text string) string {
	var buf strings.Builder
	for i := range len(text) {
		ch := text[i]
		switch ch {
		case '(', ')', '\\':
			buf.WriteByte('\\')
		}
		buf.WriteByte(ch)
	}
	return buf.String()
}

// prependContentStream adds a new stream object reference before any
// existing /Contents on the page, so the watermark renders behind content.
//
// Takes pageDict (*pdfparse.Dict) which is the page dictionary to modify.
// Takes streamObjNum (int) which is the object number of the watermark
// content stream.
func prependContentStream(pageDict *pdfparse.Dict, streamObjNum int) {
	newRef := pdfparse.RefObj(streamObjNum, 0)
	existingContents := pageDict.Get(contentsKey)

	switch existingContents.Type {
	case pdfparse.ObjectReference:
		pageDict.Set(contentsKey, pdfparse.Arr(newRef, existingContents))
	case pdfparse.ObjectArray:
		items, ok := existingContents.Value.([]pdfparse.Object)
		if ok {
			newItems := make([]pdfparse.Object, 0, len(items)+1)
			newItems = append(newItems, newRef)
			newItems = append(newItems, items...)
			pageDict.Set(contentsKey, pdfparse.Arr(newItems...))
		} else {
			pageDict.Set(contentsKey, pdfparse.Arr(newRef))
		}
	default:
		pageDict.Set(contentsKey, pdfparse.Arr(newRef))
	}
}

// addResources ensures the page dictionary has the Helvetica font resource
// and the opacity ExtGState resource needed by the watermark stream.
//
// Takes pageDict (*pdfparse.Dict) which is the page dictionary to modify.
// Takes opacity (float64) which specifies the watermark opacity value.
func addResources(pageDict *pdfparse.Dict, opacity float64) {
	resources := pageDict.GetDict("Resources")

	fontDict := resources.GetDict("Font")
	fontDict.Set(fontName, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Font")},
		{Key: "Subtype", Value: pdfparse.Name("Type1")},
		{Key: "BaseFont", Value: pdfparse.Name("Helvetica")},
	}}))
	resources.Set("Font", pdfparse.DictObj(fontDict))

	gsDict := resources.GetDict("ExtGState")
	gsDict.Set(gsName, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("ExtGState")},
		{Key: "ca", Value: pdfparse.Real(opacity)},
		{Key: "CA", Value: pdfparse.Real(opacity)},
	}}))
	resources.Set("ExtGState", pdfparse.DictObj(gsDict))

	pageDict.Set("Resources", pdfparse.DictObj(resources))
}
