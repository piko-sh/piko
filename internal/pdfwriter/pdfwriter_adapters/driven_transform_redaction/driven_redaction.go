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

package driven_transform_redaction

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	// defaultPriority is the execution order for the redaction transformer.
	defaultPriority = 100

	// contentsKey is the PDF dictionary key for page content streams.
	contentsKey = "Contents"

	// spaceReplacement is the byte used to overwrite redacted text.
	spaceReplacement = ' '
)

// RedactionTransformer applies text pattern redaction, region redaction,
// and metadata stripping to a PDF document.
//
// For text patterns, it decodes each page's content streams, replaces
// regex-matched text bytes with spaces, and rewrites the stream. For regions,
// it appends a filled black rectangle content stream on top of existing page
// content. For metadata, it removes the /Info dictionary from the trailer and
// the /Metadata reference from the catalog.
type RedactionTransformer struct {
	// name is the transformer identifier.
	name string

	// priority is the execution order.
	priority int
}

var _ pdfwriter_domain.PdfTransformerPort = (*RedactionTransformer)(nil)

// New creates a new redaction transformer with default name and priority.
//
// Returns *RedactionTransformer which is the initialised transformer.
func New() *RedactionTransformer {
	return &RedactionTransformer{
		name:     "redaction",
		priority: defaultPriority,
	}
}

// Name returns the transformer's name.
//
// Returns string which identifies this transformer.
func (t *RedactionTransformer) Name() string { return t.name }

// Type returns TransformerContent.
//
// Returns pdfwriter_dto.TransformerType which categorises this as a content
// transformer.
func (*RedactionTransformer) Type() pdfwriter_dto.TransformerType {
	return pdfwriter_dto.TransformerContent
}

// Priority returns the execution priority.
//
// Returns int which is the transformer's position in the processing order.
func (t *RedactionTransformer) Priority() int { return t.priority }

// Transform applies redaction to the PDF according to the provided
// options.
//
// If no redaction actions are configured (empty TextPatterns, empty Regions,
// StripMetadata false), the PDF is returned unchanged.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes pdf ([]byte) which is the input PDF document.
// Takes options (any) which must be RedactionOptions or *RedactionOptions.
//
// Returns []byte which is the redacted PDF.
// Returns error when the PDF cannot be parsed or redaction fails.
func (*RedactionTransformer) Transform(ctx context.Context, pdf []byte, options any) ([]byte, error) {
	opts, err := castOptions(options)
	if err != nil {
		return nil, err
	}
	if !isActive(&opts) {
		return pdf, nil
	}

	compiledPatterns, err := compilePatterns(opts.TextPatterns)
	if err != nil {
		return nil, err
	}

	doc, err := pdfparse.Parse(pdf)
	if err != nil {
		return nil, fmt.Errorf("redaction: parsing PDF: %w", err)
	}

	writer, err := pdfparse.NewWriterFromDocument(doc)
	if err != nil {
		return nil, fmt.Errorf("redaction: creating writer: %w", err)
	}

	pageRefs, err := collectPageRefs(doc)
	if err != nil {
		return nil, fmt.Errorf("redaction: collecting pages: %w", err)
	}

	if err := redactPages(ctx, writer, doc, pageRefs, compiledPatterns, opts.Regions); err != nil {
		return nil, err
	}

	if opts.StripMetadata {
		stripMetadata(writer)
	}

	output, err := writer.Write()
	if err != nil {
		return nil, fmt.Errorf("redaction: writing PDF: %w", err)
	}
	return output, nil
}

// redactPages applies text pattern and region redaction across all pages.
//
// Takes writer (*pdfparse.Writer) which is the PDF writer.
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes pageRefs ([]int) which holds the page object numbers.
// Takes compiledPatterns ([]*regexp.Regexp) which holds the text patterns to redact.
// Takes regions ([]pdfwriter_dto.RedactionRegion) which holds the regions to redact.
//
// Returns error when redaction of any page fails or context is cancelled.
func redactPages(
	ctx context.Context,
	writer *pdfparse.Writer,
	doc *pdfparse.Document,
	pageRefs []int,
	compiledPatterns []*regexp.Regexp,
	regions []pdfwriter_dto.RedactionRegion,
) error {
	regionsByPage := groupRegionsByPage(regions)

	for i, pageObjNum := range pageRefs {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("redaction: cancelled at page %d: %w", i, err)
		}
		if len(compiledPatterns) > 0 {
			if err := redactTextOnPage(writer, doc, i, pageObjNum, compiledPatterns); err != nil {
				return fmt.Errorf("redaction: text redact page %d (obj %d): %w", i, pageObjNum, err)
			}
		}
		if pageRegions, ok := regionsByPage[i]; ok {
			if err := redactRegionsOnPage(writer, pageObjNum, pageRegions); err != nil {
				return fmt.Errorf("redaction: region redact page %d (obj %d): %w", i, pageObjNum, err)
			}
		}
	}

	return nil
}

// castOptions extracts RedactionOptions from the generic options.
//
// Takes options (any) which must be RedactionOptions or *RedactionOptions.
//
// Returns pdfwriter_dto.RedactionOptions which holds the extracted options.
// Returns error when the options type is invalid or nil.
func castOptions(options any) (pdfwriter_dto.RedactionOptions, error) {
	switch v := options.(type) {
	case pdfwriter_dto.RedactionOptions:
		return v, nil
	case *pdfwriter_dto.RedactionOptions:
		if v == nil {
			return pdfwriter_dto.RedactionOptions{}, errors.New("redaction: nil options pointer")
		}
		return *v, nil
	default:
		return pdfwriter_dto.RedactionOptions{}, fmt.Errorf("redaction: expected RedactionOptions, got %T", options)
	}
}

// isActive returns true if any redaction action is configured.
//
// Takes opts (*pdfwriter_dto.RedactionOptions) which holds the options to check.
//
// Returns bool which indicates whether any redaction is configured.
func isActive(opts *pdfwriter_dto.RedactionOptions) bool {
	return len(opts.TextPatterns) > 0 || len(opts.Regions) > 0 || opts.StripMetadata
}

// compilePatterns compiles the text pattern strings into regular
// expressions.
//
// Takes patterns ([]string) which holds the regex pattern strings.
//
// Returns []*regexp.Regexp which holds the compiled patterns.
// Returns error when any pattern is invalid.
func compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("redaction: invalid pattern %q: %w", pattern, err)
		}
		compiled = append(compiled, re)
	}
	return compiled, nil
}

// groupRegionsByPage organises redaction regions into a map keyed by
// zero-based page index.
//
// Takes regions ([]pdfwriter_dto.RedactionRegion) which holds the regions to group.
//
// Returns map[int][]pdfwriter_dto.RedactionRegion which maps page indices to their regions.
func groupRegionsByPage(regions []pdfwriter_dto.RedactionRegion) map[int][]pdfwriter_dto.RedactionRegion {
	if len(regions) == 0 {
		return nil
	}
	grouped := make(map[int][]pdfwriter_dto.RedactionRegion)
	for _, r := range regions {
		grouped[r.Page] = append(grouped[r.Page], r)
	}
	return grouped
}

// redactTextOnPage decodes the page's content streams and replaces text
// matching any compiled pattern with spaces.
//
// Takes writer (*pdfparse.Writer) which is the PDF writer.
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes pageIndex (int) which is the zero-based page index for error messages.
// Takes pageObjNum (int) which is the page's object number.
// Takes patterns ([]*regexp.Regexp) which holds the compiled text patterns.
//
// Returns error when content streams cannot be read or decoded.
func redactTextOnPage(
	writer *pdfparse.Writer,
	doc *pdfparse.Document,
	pageIndex int,
	pageObjNum int,
	patterns []*regexp.Regexp,
) error {
	pageObj := writer.GetObject(pageObjNum)
	pageDict, ok := pageObj.Value.(pdfparse.Dict)
	if !ok {
		return errors.New("page object is not a dictionary")
	}

	contentRefs := resolveContentRefs(pageDict)
	if len(contentRefs) == 0 {
		return nil
	}

	for _, ref := range contentRefs {
		streamObj, err := doc.GetObject(ref)
		if err != nil {
			return fmt.Errorf("redaction: cannot retrieve content stream object %d for page %d: %w", ref, pageIndex, err)
		}
		if streamObj.Type != pdfparse.ObjectStream {
			continue
		}

		decoded, err := pdfparse.DecodeStream(streamObj)
		if err != nil {
			return fmt.Errorf("redaction: cannot decode content stream for page %d: %w", pageIndex, err)
		}

		modified := applyTextRedaction(decoded, patterns)
		if !bytesEqual(decoded, modified) {
			writer.SetObject(ref, pdfparse.StreamObj(pdfparse.Dict{}, modified))
		}
	}

	return nil
}

// resolveContentRefs extracts object numbers from the page's /Contents
// entry, handling both single references and arrays.
//
// Takes pageDict (pdfparse.Dict) which is the page dictionary.
//
// Returns []int which holds the content stream object numbers.
func resolveContentRefs(pageDict pdfparse.Dict) []int {
	contentsObj := pageDict.Get(contentsKey)

	switch contentsObj.Type {
	case pdfparse.ObjectReference:
		ref, ok := contentsObj.Value.(pdfparse.Ref)
		if !ok {
			return nil
		}
		return []int{ref.Number}
	case pdfparse.ObjectArray:
		items, ok := contentsObj.Value.([]pdfparse.Object)
		if !ok {
			return nil
		}
		refs := make([]int, 0, len(items))
		for _, item := range items {
			ref, ok := item.Value.(pdfparse.Ref)
			if !ok {
				continue
			}
			refs = append(refs, ref.Number)
		}
		return refs
	default:
		return nil
	}
}

// applyTextRedaction replaces all regex matches in the stream bytes with
// space characters.
//
// Takes data ([]byte) which is the stream content to redact.
// Takes patterns ([]*regexp.Regexp) which holds the compiled text patterns.
//
// Returns []byte which is the redacted stream content.
func applyTextRedaction(data []byte, patterns []*regexp.Regexp) []byte {
	result := make([]byte, len(data))
	copy(result, data)

	for _, re := range patterns {
		matches := re.FindAllIndex(result, -1)
		for _, match := range matches {
			for i := match[0]; i < match[1]; i++ {
				result[i] = spaceReplacement
			}
		}
	}

	return result
}

// bytesEqual returns true if two byte slices have identical contents.
//
// Takes a ([]byte) which is the first byte slice.
// Takes b ([]byte) which is the second byte slice.
//
// Returns bool which indicates whether the slices are equal.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// redactRegionsOnPage appends a content stream with filled black
// rectangles for each redaction region on the page.
//
// Takes writer (*pdfparse.Writer) which is the PDF writer.
// Takes pageObjNum (int) which is the page's object number.
// Takes regions ([]pdfwriter_dto.RedactionRegion) which holds the regions to redact.
//
// Returns error when the page object is not a dictionary.
func redactRegionsOnPage(
	writer *pdfparse.Writer,
	pageObjNum int,
	regions []pdfwriter_dto.RedactionRegion,
) error {
	pageObj := writer.GetObject(pageObjNum)
	pageDict, ok := pageObj.Value.(pdfparse.Dict)
	if !ok {
		return errors.New("page object is not a dictionary")
	}

	stream := buildRegionStream(regions)
	streamObjNum := writer.AddObject(pdfparse.StreamObj(pdfparse.Dict{}, []byte(stream)))

	appendContentStream(&pageDict, streamObjNum)
	writer.SetObject(pageObjNum, pdfparse.DictObj(pageDict))

	return nil
}

// buildRegionStream generates PDF content stream operators that draw
// filled black rectangles for each redaction region.
//
// Takes regions ([]pdfwriter_dto.RedactionRegion) which holds the regions to draw.
//
// Returns string which is the PDF content stream operators.
func buildRegionStream(regions []pdfwriter_dto.RedactionRegion) string {
	buf := make([]byte, 0, len(regions)*64)
	for _, r := range regions {
		buf = append(buf, fmt.Sprintf("q 0 0 0 rg %g %g %g %g re f Q\n", r.X, r.Y, r.Width, r.Height)...)
	}
	return string(buf)
}

// appendContentStream adds a new stream object reference after any
// existing /Contents on the page.
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

// stripMetadata removes the /Info dictionary from the trailer and the
// /Metadata entry from the document catalog.
//
// Takes writer (*pdfparse.Writer) which is the PDF writer to modify.
func stripMetadata(writer *pdfparse.Writer) {
	trailer := writer.Trailer()

	if trailer.Remove("Info") {
		writer.SetTrailer(trailer)
	}

	rootRef := trailer.GetRef("Root")
	if rootRef.Number == 0 {
		return
	}

	catalogObj := writer.GetObject(rootRef.Number)
	catalogDict, ok := catalogObj.Value.(pdfparse.Dict)
	if !ok {
		return
	}

	if catalogDict.Remove("Metadata") {
		writer.SetObject(rootRef.Number, pdfparse.DictObj(catalogDict))
	}
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
