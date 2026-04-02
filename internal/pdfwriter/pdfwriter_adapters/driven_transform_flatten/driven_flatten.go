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

package driven_transform_flatten

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	// defaultPriority is the execution order for the flatten transformer.
	defaultPriority = 120

	// xobjPrefix is the resource name prefix for flattened XObject
	// appearances.
	xobjPrefix = "FLT"

	// rectElements is the expected number of elements in a PDF Rect or
	// BBox array.
	rectElements = 4

	// contentsKey is the PDF dictionary key for page content streams.
	contentsKey = "Contents"

	// resourcesKey is the PDF dictionary key for page resources.
	resourcesKey = "Resources"

	// annotsKey is the PDF dictionary key for page annotations.
	annotsKey = "Annots"

	// xobjectKey is the PDF dictionary key for XObject resources.
	xobjectKey = "XObject"

	// subtypeWidget is the annotation subtype for form field widgets.
	subtypeWidget = "Widget"
)

// FlattenTransformer converts interactive PDF elements into static page content.
//
// For each annotation with a normal appearance stream, it adds the appearance
// as a Form XObject resource on the page and appends a content stream that
// draws it at the annotation's Rect position using a Do operator. The original
// annotation is then removed. Form field flattening also removes the AcroForm
// dictionary from the document catalog. Transparency flattening removes the
// Group entry from page dictionaries.
type FlattenTransformer struct {
	// name is the transformer identifier.
	name string

	// priority is the execution order.
	priority int
}

var _ pdfwriter_domain.PdfTransformerPort = (*FlattenTransformer)(nil)

// New creates a new flatten transformer with default name and priority.
//
// Returns *FlattenTransformer which is the initialised transformer.
func New() *FlattenTransformer {
	return &FlattenTransformer{
		name:     "flatten",
		priority: defaultPriority,
	}
}

// Name returns the transformer's name.
//
// Returns string which identifies this transformer.
func (t *FlattenTransformer) Name() string { return t.name }

// Type returns TransformerContent.
//
// Returns pdfwriter_dto.TransformerType which categorises this as a content
// transformer.
func (*FlattenTransformer) Type() pdfwriter_dto.TransformerType {
	return pdfwriter_dto.TransformerContent
}

// Priority returns the execution priority.
//
// Returns int which is the transformer's position in the processing order.
func (t *FlattenTransformer) Priority() int { return t.priority }

// Transform flattens interactive elements in the PDF according to the
// provided options.
//
// If no flattening flags are set, the PDF is returned unchanged.
//
// Takes pdf ([]byte) which is the input PDF document.
// Takes options (any) which must be FlattenOptions or *FlattenOptions.
//
// Returns []byte which is the flattened PDF.
// Returns error when the PDF cannot be parsed or flattening fails.
func (*FlattenTransformer) Transform(_ context.Context, pdf []byte, options any) ([]byte, error) {
	opts, err := castOptions(options)
	if err != nil {
		return nil, err
	}
	if !opts.FormFields && !opts.Annotations && !opts.Transparency {
		return pdf, nil
	}

	doc, err := pdfparse.Parse(pdf)
	if err != nil {
		return nil, fmt.Errorf("flatten: parsing PDF: %w", err)
	}

	writer, err := pdfparse.NewWriterFromDocument(doc)
	if err != nil {
		return nil, fmt.Errorf("flatten: creating writer: %w", err)
	}

	pageRefs, err := collectPageRefs(doc)
	if err != nil {
		return nil, fmt.Errorf("flatten: collecting pages: %w", err)
	}

	fctx := &flattenContext{
		doc:    doc,
		writer: writer,
	}

	if err := flattenPages(fctx, pageRefs, &opts); err != nil {
		return nil, err
	}

	output, err := writer.Write()
	if err != nil {
		return nil, fmt.Errorf("flatten: writing PDF: %w", err)
	}
	return output, nil
}

// castOptions extracts FlattenOptions from the generic options.
//
// Takes options (any) which must be FlattenOptions or *FlattenOptions.
//
// Returns pdfwriter_dto.FlattenOptions which holds the extracted options.
// Returns error when the options type is invalid or nil.
func castOptions(options any) (pdfwriter_dto.FlattenOptions, error) {
	switch v := options.(type) {
	case pdfwriter_dto.FlattenOptions:
		return v, nil
	case *pdfwriter_dto.FlattenOptions:
		if v == nil {
			return pdfwriter_dto.FlattenOptions{}, errors.New("flatten: nil options pointer")
		}
		return *v, nil
	default:
		return pdfwriter_dto.FlattenOptions{}, fmt.Errorf("flatten: expected FlattenOptions, got %T", options)
	}
}

// flattenContext holds shared state during the flatten operation.
type flattenContext struct {
	// doc holds the parsed PDF document.
	doc *pdfparse.Document

	// writer holds the PDF writer for output.
	writer *pdfparse.Writer

	// xobjID holds the next XObject identifier counter.
	xobjID int
}

// nextXObjName returns a unique XObject resource name for each flattened
// appearance.
//
// Returns string which is the generated XObject resource name.
func (c *flattenContext) nextXObjName() string {
	c.xobjID++
	return fmt.Sprintf("%s%d", xobjPrefix, c.xobjID)
}

// flattenPages processes all pages, flattening annotations and/or
// transparency as specified by opts.
//
// If form fields are flattened, the AcroForm is removed from the document
// catalog afterwards.
//
// Takes fctx (*flattenContext) which holds the shared flatten state.
// Takes pageRefs ([]int) which holds the page object numbers in document order.
// Takes opts (*pdfwriter_dto.FlattenOptions) which controls what to flatten.
//
// Returns error when flattening any page fails.
func flattenPages(fctx *flattenContext, pageRefs []int, opts *pdfwriter_dto.FlattenOptions) error {
	for i, pageObjNum := range pageRefs {
		if opts.FormFields || opts.Annotations {
			if err := flattenPageAnnotations(fctx, pageObjNum, opts); err != nil {
				return fmt.Errorf("flatten: page %d (obj %d): %w", i, pageObjNum, err)
			}
		}
		if opts.Transparency {
			removeTransparencyGroup(fctx, pageObjNum)
		}
	}

	if opts.FormFields {
		removeAcroForm(fctx)
	}

	return nil
}

// flattenPageAnnotations processes annotations on a single page,
// flattening those that match the options into static content.
//
// Each annotation with a usable normal appearance stream is converted to a
// Form XObject reference and drawn via a content stream appended to the page.
// Annotations without appearance streams are kept unchanged.
//
// Takes fctx (*flattenContext) which holds the shared flatten state.
// Takes pageObjNum (int) which is the page's object number.
// Takes opts (*pdfwriter_dto.FlattenOptions) which controls what to flatten.
//
// Returns error when annotation resolution or content stream creation fails.
func flattenPageAnnotations(
	fctx *flattenContext,
	pageObjNum int,
	opts *pdfwriter_dto.FlattenOptions,
) error {
	pageObj := fctx.writer.GetObject(pageObjNum)
	pageDict, ok := pageObj.Value.(pdfparse.Dict)
	if !ok {
		return errors.New("page object is not a dictionary")
	}

	annots := resolveAnnotsArray(fctx.doc, pageDict.Get(annotsKey))
	if len(annots) == 0 {
		return nil
	}

	var remaining []pdfparse.Object
	var streams []string

	for _, annotObj := range annots {
		annotDict, err := resolveAnnotDict(fctx.doc, annotObj)
		if err != nil {
			remaining = append(remaining, annotObj)
			continue
		}

		subtype := annotDict.GetName("Subtype")
		if !shouldFlatten(subtype, opts) {
			remaining = append(remaining, annotObj)
			continue
		}

		apObjNum, err := resolveNormalAppearance(fctx.doc, annotDict)
		if err != nil {
			remaining = append(remaining, annotObj)
			continue
		}

		annotRect := extractRect(annotDict)
		bbox := extractAppearanceBBox(fctx, apObjNum)
		name := fctx.nextXObjName()

		addXObjectResource(&pageDict, name, apObjNum)
		streams = append(streams, buildFlattenStream(name, annotRect, bbox))
	}

	if len(streams) == 0 {
		return nil
	}

	combined := strings.Join(streams, "")
	streamObjNum := fctx.writer.AddObject(
		pdfparse.StreamObj(pdfparse.Dict{}, []byte(combined)),
	)
	appendContentStream(&pageDict, streamObjNum)

	if len(remaining) == 0 {
		pageDict.Remove(annotsKey)
	} else {
		pageDict.Set(annotsKey, pdfparse.Arr(remaining...))
	}

	fctx.writer.SetObject(pageObjNum, pdfparse.DictObj(pageDict))
	return nil
}

// shouldFlatten returns true if the given annotation subtype should be
// flattened based on the current options.
//
// Widget annotations are flattened when FormFields is true; all other
// annotations are flattened when Annotations is true.
//
// Takes subtype (string) which is the annotation's /Subtype name.
// Takes opts (*pdfwriter_dto.FlattenOptions) which controls what to flatten.
//
// Returns bool which indicates whether the annotation should be flattened.
func shouldFlatten(subtype string, opts *pdfwriter_dto.FlattenOptions) bool {
	if subtype == subtypeWidget {
		return opts.FormFields
	}
	return opts.Annotations
}

// resolveAnnotsArray resolves the /Annots value to a slice of Objects.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF for resolving references.
// Takes obj (pdfparse.Object) which is the /Annots dictionary value.
//
// Returns []pdfparse.Object which holds the resolved annotation objects.
func resolveAnnotsArray(doc *pdfparse.Document, obj pdfparse.Object) []pdfparse.Object {
	resolved, err := doc.Resolve(obj)
	if err != nil {
		return nil
	}
	if arr, ok := resolved.Value.([]pdfparse.Object); ok {
		return arr
	}
	return nil
}

// resolveAnnotDict resolves an annotation object to its dictionary.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF for resolving references.
// Takes obj (pdfparse.Object) which is the annotation object or reference.
//
// Returns pdfparse.Dict which is the resolved annotation dictionary.
// Returns error when the object cannot be resolved or is not a dictionary.
func resolveAnnotDict(doc *pdfparse.Document, obj pdfparse.Object) (pdfparse.Dict, error) {
	resolved, err := doc.Resolve(obj)
	if err != nil {
		return pdfparse.Dict{}, err
	}
	dict, ok := resolved.Value.(pdfparse.Dict)
	if !ok {
		return pdfparse.Dict{}, errors.New("annotation is not a dictionary")
	}
	return dict, nil
}

// resolveNormalAppearance finds the object number of the normal appearance
// stream for an annotation. The normal appearance is AP -> N, which can
// be either a direct reference to a Form XObject stream, or a dictionary
// mapping state names to streams (selected by the annotation's AS entry).
//
// Takes doc (*pdfparse.Document) which is the parsed PDF for resolving
// references.
// Takes annotDict (pdfparse.Dict) which is the annotation dictionary.
//
// Returns int which is the object number of the appearance stream.
// Returns error when no usable normal appearance is found.
func resolveNormalAppearance(doc *pdfparse.Document, annotDict pdfparse.Dict) (int, error) {
	apObj := annotDict.Get("AP")
	resolved, err := doc.Resolve(apObj)
	if err != nil || resolved.IsNull() {
		return 0, errors.New("no /AP entry")
	}

	apDict, ok := resolved.Value.(pdfparse.Dict)
	if !ok {
		return 0, errors.New("/AP is not a dictionary")
	}

	nObj := apDict.Get("N")
	if nObj.IsNull() {
		return 0, errors.New("no /AP/N entry")
	}

	if nObj.Type == pdfparse.ObjectReference {
		return extractRefNumber(nObj)
	}

	if nObj.Type == pdfparse.ObjectDictionary {
		return resolveStateAppearance(annotDict, nObj)
	}

	return 0, errors.New("/AP/N is neither a reference nor a dictionary")
}

// extractRefNumber returns the object number from a reference Object.
//
// Takes obj (pdfparse.Object) which is the reference object to extract from.
//
// Returns int which is the extracted object number.
// Returns error when the object value is not a valid reference.
func extractRefNumber(obj pdfparse.Object) (int, error) {
	ref, ok := obj.Value.(pdfparse.Ref)
	if !ok {
		return 0, errors.New("reference has invalid value type")
	}
	return ref.Number, nil
}

// resolveStateAppearance picks the correct state-specific appearance from
// an AP/N dictionary.
//
// The current state is read from the annotation's AS entry; if absent, the
// first entry in the dictionary is used.
//
// Takes annotDict (pdfparse.Dict) which is the annotation
// dictionary containing the AS entry.
// Takes nObj (pdfparse.Object) which is the /AP/N dictionary
// object.
//
// Returns int which is the object number of the selected appearance stream.
// Returns error when no valid state appearance reference is found.
func resolveStateAppearance(annotDict pdfparse.Dict, nObj pdfparse.Object) (int, error) {
	nDict, ok := nObj.Value.(pdfparse.Dict)
	if !ok {
		return 0, errors.New("/AP/N dict is invalid")
	}

	state := annotDict.GetName("AS")
	if state == "" {
		if len(nDict.Pairs) == 0 {
			return 0, errors.New("/AP/N dict is empty")
		}
		state = nDict.Pairs[0].Key
	}

	stateObj := nDict.Get(state)
	if stateObj.Type == pdfparse.ObjectReference {
		return extractRefNumber(stateObj)
	}
	return 0, errors.New("/AP/N state entry is not a reference")
}

// pdfRect holds the four coordinates of a PDF rectangle.
type pdfRect struct {
	// llx holds the lower-left x coordinate.
	llx float64

	// lly holds the lower-left y coordinate.
	lly float64

	// urx holds the upper-right x coordinate.
	urx float64

	// ury holds the upper-right y coordinate.
	ury float64
}

// width returns the horizontal extent of the rectangle.
//
// Returns float64 which is the difference between urx and llx.
func (r pdfRect) width() float64 { return r.urx - r.llx }

// height returns the vertical extent of the rectangle.
//
// Returns float64 which is the difference between ury and lly.
func (r pdfRect) height() float64 { return r.ury - r.lly }

// extractRect reads the /Rect array from a dictionary.
//
// Takes dict (pdfparse.Dict) which is the dictionary containing the /Rect entry.
//
// Returns pdfRect which holds the extracted rectangle coordinates.
func extractRect(dict pdfparse.Dict) pdfRect {
	arr := dict.GetArray("Rect")
	if len(arr) < rectElements {
		return pdfRect{}
	}
	return pdfRect{
		llx: objectToFloat(arr[0]),
		lly: objectToFloat(arr[1]),
		urx: objectToFloat(arr[2]),
		ury: objectToFloat(arr[3]),
	}
}

// extractAppearanceBBox reads the /BBox from an appearance stream's
// dictionary.
//
// Takes fctx (*flattenContext) which holds the shared flatten state.
// Takes objNum (int) which is the appearance stream's object number.
//
// Returns pdfRect which holds the bounding box, or a unit square if missing.
func extractAppearanceBBox(fctx *flattenContext, objNum int) pdfRect {
	obj := fctx.writer.GetObject(objNum)
	dict, ok := obj.Value.(pdfparse.Dict)
	if !ok {
		return pdfRect{urx: 1, ury: 1}
	}
	arr := dict.GetArray("BBox")
	if len(arr) < rectElements {
		return pdfRect{urx: 1, ury: 1}
	}
	return pdfRect{
		llx: objectToFloat(arr[0]),
		lly: objectToFloat(arr[1]),
		urx: objectToFloat(arr[2]),
		ury: objectToFloat(arr[3]),
	}
}

// buildFlattenStream creates a content stream that draws a Form XObject
// at the position and size specified by the annotation Rect.
//
// The transformation matrix scales the BBox to fit the Rect and translates
// to the Rect origin.
//
// Takes name (string) which is the XObject resource name.
// Takes annotRect (pdfRect) which is the annotation's position rectangle.
// Takes bbox (pdfRect) which is the appearance's bounding box.
//
// Returns string which is the PDF content stream operators.
func buildFlattenStream(name string, annotRect, bbox pdfRect) string {
	var buf strings.Builder

	bw := bbox.width()
	bh := bbox.height()
	if bw == 0 {
		bw = 1
	}
	if bh == 0 {
		bh = 1
	}

	sx := annotRect.width() / bw
	sy := annotRect.height() / bh
	tx := annotRect.llx - bbox.llx*sx
	ty := annotRect.lly - bbox.lly*sy

	buf.WriteString("q\n")
	fmt.Fprintf(&buf, "%g 0 0 %g %g %g cm\n", sx, sy, tx, ty)
	fmt.Fprintf(&buf, "/%s Do\n", name)
	buf.WriteString("Q\n")

	return buf.String()
}

// addXObjectResource adds a Form XObject reference to the page's
// Resources/XObject dictionary.
//
// Takes pageDict (*pdfparse.Dict) which is the page dictionary to modify.
// Takes name (string) which is the XObject resource name.
// Takes objNum (int) which is the object number of the Form XObject.
func addXObjectResource(pageDict *pdfparse.Dict, name string, objNum int) {
	resources := pageDict.GetDict(resourcesKey)
	xobjects := resources.GetDict(xobjectKey)
	xobjects.Set(name, pdfparse.RefObj(objNum, 0))
	resources.Set(xobjectKey, pdfparse.DictObj(xobjects))
	pageDict.Set(resourcesKey, pdfparse.DictObj(resources))
}

// appendContentStream adds a new stream object reference after any
// existing Contents on the page.
//
// Takes pageDict (*pdfparse.Dict) which is the page dictionary to modify.
// Takes streamObjNum (int) which is the object number of the new content stream.
func appendContentStream(pageDict *pdfparse.Dict, streamObjNum int) {
	newRef := pdfparse.RefObj(streamObjNum, 0)
	existing := pageDict.Get(contentsKey)

	switch existing.Type {
	case pdfparse.ObjectReference:
		pageDict.Set(contentsKey, pdfparse.Arr(existing, newRef))
	case pdfparse.ObjectArray:
		items, ok := existing.Value.([]pdfparse.Object)
		if ok {
			items = append(items, newRef)
			pageDict.Set(contentsKey, pdfparse.Arr(items...))
		} else {
			pageDict.Set(contentsKey, pdfparse.Arr(newRef))
		}
	default:
		pageDict.Set(contentsKey, pdfparse.Arr(newRef))
	}
}

// removeAcroForm removes the /AcroForm entry from the document catalog.
//
// Takes fctx (*flattenContext) which holds the shared flatten state.
func removeAcroForm(fctx *flattenContext) {
	trailer := fctx.writer.Trailer()
	rootRef := trailer.GetRef("Root")
	if rootRef.Number == 0 {
		return
	}

	catalogObj := fctx.writer.GetObject(rootRef.Number)
	catalogDict, ok := catalogObj.Value.(pdfparse.Dict)
	if !ok {
		return
	}

	if !catalogDict.Remove("AcroForm") {
		return
	}
	fctx.writer.SetObject(rootRef.Number, pdfparse.DictObj(catalogDict))
}

// removeTransparencyGroup removes the /Group entry from a page
// dictionary.
//
// Takes fctx (*flattenContext) which holds the shared flatten state.
// Takes pageObjNum (int) which is the page's object number.
func removeTransparencyGroup(fctx *flattenContext, pageObjNum int) {
	pageObj := fctx.writer.GetObject(pageObjNum)
	pageDict, ok := pageObj.Value.(pdfparse.Dict)
	if !ok {
		return
	}

	if !pageDict.Remove("Group") {
		return
	}
	fctx.writer.SetObject(pageObjNum, pdfparse.DictObj(pageDict))
}

// collectPageRefs walks the page tree and returns object numbers for all
// leaf Page objects in document order.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
//
// Returns []int which holds the page object numbers.
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
// Takes objNum (int) which is the current tree node's object number.
//
// Returns []int which holds the collected page object numbers.
// Returns error when any node cannot be read.
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

// objectToFloat extracts a numeric value from a PDF Object.
//
// Takes obj (pdfparse.Object) which is the object to extract from.
//
// Returns float64 which is the numeric value, or zero if not numeric.
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
