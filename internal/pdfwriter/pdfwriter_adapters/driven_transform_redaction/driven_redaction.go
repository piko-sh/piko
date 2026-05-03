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
	"strings"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	// defaultPriority is the execution order for the redaction transformer.
	defaultPriority = 100

	// contentsKey is the PDF dictionary key for page content streams.
	contentsKey = "Contents"

	// annotsKey is the PDF dictionary key for the page annotation array.
	annotsKey = "Annots"

	// titleKey is the PDF dictionary key for an annotation's title (/T).
	titleKey = "T"

	// resourcesKey is the PDF dictionary key for a page or form XObject
	// resources dictionary.
	resourcesKey = "Resources"

	// xobjectKey is the PDF dictionary key for the XObject sub-dictionary
	// of a /Resources entry.
	xobjectKey = "XObject"

	// subtypeKey is the PDF dictionary key for an XObject subtype.
	subtypeKey = "Subtype"

	// formSubtype is the /Subtype value of a form XObject.
	formSubtype = "Form"

	// spaceReplacement is the byte used to overwrite redacted text.
	spaceReplacement = ' '

	// defaultMaxPatternLength caps the length of a single user-supplied
	// regex pattern (2 KiB). Pathological huge alternations are
	// rejected before [regexp.Compile] sees them.
	defaultMaxPatternLength = 2 << 10

	// defaultMaxPatternCount caps the number of patterns accepted in a
	// single redaction operation.
	defaultMaxPatternCount = 256

	// defaultCancellationCheckEvery controls how often the text matcher
	// re-checks the context for cancellation while iterating matches.
	defaultCancellationCheckEvery = 1024

	// maxPageTreeDepth caps the recursion depth of the page-tree walk.
	// Genuine documents rarely nest the /Pages tree more than a handful
	// of levels; this cap defends against malformed trees that would
	// otherwise exhaust the stack.
	maxPageTreeDepth = 256

	// maxXObjectDepth caps the recursion depth when walking nested form
	// XObjects. Form XObjects may reference each other through their own
	// /Resources/XObject dictionaries; this cap defends against malformed
	// or adversarial nesting.
	maxXObjectDepth = 64
)

// ErrTooManyPatterns is returned when the caller supplies more
// redaction patterns than the configured maximum.
var ErrTooManyPatterns = errors.New("redaction: too many text patterns")

// ErrPatternTooLong is returned when a single redaction pattern exceeds
// the configured maximum length.
var ErrPatternTooLong = errors.New("redaction: pattern length exceeds limit")

// stringFieldKeys lists the dictionary string keys whose values are
// redacted on annotation objects when string-field redaction is enabled.
//
// /Contents holds the comment body of a text annotation; /T holds the
// annotation's title (typically the author or label). Both are surfaced
// to PDF readers and are common leakage paths for PII.
var stringFieldKeys = []string{contentsKey, titleKey}

// RedactionTransformer applies text pattern redaction, region redaction,
// and metadata stripping to a PDF document.
//
// # Coverage
//
// The transformer redacts the following surfaces when text patterns are
// configured:
//
//   - /Contents page content streams (text drawn on the page).
//   - /Annots annotation /Contents and /T string values (when
//     RedactStringFields is enabled, the default).
//   - /ActualText and /Alt accessibility strings, where they appear
//     inside content streams as marked-content properties (covered by the
//     stream byte-level walk).
//   - Form XObject content streams referenced from a page's
//     /Resources/XObject dictionary, walked recursively with a cycle
//     guard.
//
// The transformer does not currently cover the following surfaces:
//
//   - Image XObjects (rasterised text inside JPEG/JBIG2/CCITT streams)
//     are out of scope; pattern matching is text-only.
//   - Embedded fonts. If a font subset still contains the original
//     glyph data for sensitive characters, those glyphs remain inside
//     the font object even after the on-page text is spaced over.
//   - /StructTreeRoot logical structure metadata. Tagged-PDF structure
//     elements may carry a copy of the visible text; this is rare for
//     PII leakage and is out of scope.
//   - /JavaScript actions and embedded files attached via /Names or
//     /EmbeddedFiles trees.
//
// # Byte-length preservation
//
// Matched text is overwritten with U+0020 spaces of the same byte length
// as the original match. This preserves the on-page layout and avoids
// having to re-encode content streams. Length itself is therefore still
// observable: a redacted account number occupies the same number of
// bytes as the original. Callers that need length-hiding should combine
// redaction with region-based black bars over the same areas.
type RedactionTransformer struct {
	// name is the transformer identifier.
	name string

	// priority is the execution order.
	priority int

	// maxPatternLength caps the length of a single user-supplied regex
	// pattern.
	maxPatternLength int

	// maxPatternCount caps the number of patterns accepted in a single
	// redaction operation.
	maxPatternCount int

	// cancellationCheckEvery controls how often the matcher re-checks
	// the context for cancellation while iterating matches.
	cancellationCheckEvery int

	// redactStringFields enables redaction of dictionary string values
	// in annotations (/Contents, /T) and recursive form XObject content
	// streams. Defaults to true.
	redactStringFields bool
}

// Option configures a [RedactionTransformer] at construction time.
type Option func(*RedactionTransformer)

// WithMaxPatternLength overrides the per-pattern length cap.
//
// Takes limit (int) which is the maximum pattern length in bytes. Values
// less than or equal to zero are ignored.
//
// Returns Option which applies the override.
func WithMaxPatternLength(limit int) Option {
	return func(t *RedactionTransformer) {
		if limit > 0 {
			t.maxPatternLength = limit
		}
	}
}

// WithMaxPatternCount overrides the cap on the number of patterns
// accepted per redaction call.
//
// Takes limit (int) which is the maximum pattern count. Values less
// than or equal to zero are ignored.
//
// Returns Option which applies the override.
func WithMaxPatternCount(limit int) Option {
	return func(t *RedactionTransformer) {
		if limit > 0 {
			t.maxPatternCount = limit
		}
	}
}

// WithCancellationCheckEvery overrides how often the matcher checks the
// context for cancellation.
//
// Takes every (int) which is the check interval expressed as a match
// count. Values less than or equal to zero are ignored.
//
// Returns Option which applies the override.
func WithCancellationCheckEvery(every int) Option {
	return func(t *RedactionTransformer) {
		if every > 0 {
			t.cancellationCheckEvery = every
		}
	}
}

// WithRedactStringFields toggles redaction of annotation string fields
// (/Contents, /T) and recursive form XObject content streams.
//
// Defaults to true. Callers whose content legitimately contains pattern
// matches inside annotation titles or form XObjects (for example, a
// stamp annotation whose text is meant to remain visible) can opt out by
// passing false.
//
// Takes enabled (bool) which selects whether string-field redaction
// runs.
//
// Returns Option which applies the override.
func WithRedactStringFields(enabled bool) Option {
	return func(t *RedactionTransformer) {
		t.redactStringFields = enabled
	}
}

var _ pdfwriter_domain.PdfTransformerPort = (*RedactionTransformer)(nil)

// New creates a new redaction transformer with default name and priority.
// Optional functional options override per-pattern length, total pattern
// count, the cancellation check interval used during matching, and
// whether annotation/form XObject redaction runs.
//
// Takes opts (...Option) which override the defaults.
//
// Returns *RedactionTransformer which is the initialised transformer.
func New(opts ...Option) *RedactionTransformer {
	t := &RedactionTransformer{
		name:                   "redaction",
		priority:               defaultPriority,
		maxPatternLength:       defaultMaxPatternLength,
		maxPatternCount:        defaultMaxPatternCount,
		cancellationCheckEvery: defaultCancellationCheckEvery,
		redactStringFields:     true,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(t)
		}
	}
	return t
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
func (t *RedactionTransformer) Transform(ctx context.Context, pdf []byte, options any) ([]byte, error) {
	opts, err := castOptions(options)
	if err != nil {
		return nil, err
	}
	if !isActive(&opts) {
		return pdf, nil
	}

	compiledPatterns, err := t.compilePatterns(opts.TextPatterns)
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

	if err := t.redactPages(ctx, writer, doc, pageRefs, compiledPatterns, opts.Regions); err != nil {
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
// Takes ctx (context.Context) which carries cancellation checked between
// pages and during long match loops.
// Takes writer (*pdfparse.Writer) which is the PDF writer.
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes pageRefs ([]int) which holds the page object numbers.
// Takes compiledPatterns ([]*regexp.Regexp) which holds the text patterns to redact.
// Takes regions ([]pdfwriter_dto.RedactionRegion) which holds the regions to redact.
//
// Returns error when redaction of any page fails or context is cancelled.
func (t *RedactionTransformer) redactPages(
	ctx context.Context,
	writer *pdfparse.Writer,
	doc *pdfparse.Document,
	pageRefs []int,
	compiledPatterns []*regexp.Regexp,
	regions []pdfwriter_dto.RedactionRegion,
) error {
	regionsByPage := groupRegionsByPage(regions)
	xobjectVisited := make(map[int]struct{})

	for i, pageObjNum := range pageRefs {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("redaction: cancelled at page %d: %w", i, err)
		}
		if len(compiledPatterns) > 0 {
			if err := t.redactTextOnPage(ctx, writer, doc, i, pageObjNum, compiledPatterns, xobjectVisited); err != nil {
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
// expressions, rejecting input that exceeds the configured count or
// per-pattern length caps. The caps protect against malicious huge
// alternations that, while RE2-safe from catastrophic backtracking,
// can still spend pathological CPU during compilation and matching.
//
// Takes patterns ([]string) which holds the regex pattern strings.
//
// Returns []*regexp.Regexp which holds the compiled patterns.
// Returns error when any pattern is invalid or a cap is exceeded.
func (t *RedactionTransformer) compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	if len(patterns) > t.maxPatternCount {
		return nil, fmt.Errorf("%w: got %d, limit %d", ErrTooManyPatterns, len(patterns), t.maxPatternCount)
	}
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if len(pattern) > t.maxPatternLength {
			return nil, fmt.Errorf("%w: got %d bytes, limit %d", ErrPatternTooLong, len(pattern), t.maxPatternLength)
		}
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
// matching any compiled pattern with spaces. When string-field redaction
// is enabled, it also walks /Annots and /Resources/XObject form
// references on the page.
//
// Takes ctx (context.Context) which is checked periodically inside the
// match loop so callers can cancel a runaway redaction.
// Takes writer (*pdfparse.Writer) which is the PDF writer.
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes pageIndex (int) which is the zero-based page index for error messages.
// Takes pageObjNum (int) which is the page's object number.
// Takes patterns ([]*regexp.Regexp) which holds the compiled text patterns.
// Takes xobjectVisited (map[int]struct{}) which records form XObject
// objects already redacted, shared across all pages so that a form
// XObject used by multiple pages is redacted exactly once.
//
// Returns error when content streams cannot be read or decoded, or when
// the supplied context is cancelled mid-stream.
func (t *RedactionTransformer) redactTextOnPage(
	ctx context.Context,
	writer *pdfparse.Writer,
	doc *pdfparse.Document,
	pageIndex int,
	pageObjNum int,
	patterns []*regexp.Regexp,
	xobjectVisited map[int]struct{},
) error {
	pageObj := writer.GetObject(pageObjNum)
	pageDict, ok := pageObj.Value.(pdfparse.Dict)
	if !ok {
		return errors.New("page object is not a dictionary")
	}

	if err := t.redactStreamRefs(ctx, writer, doc, pageIndex, resolveContentRefs(pageDict), patterns); err != nil {
		return err
	}

	if !t.redactStringFields {
		return nil
	}

	if err := t.redactAnnotations(ctx, writer, pageDict, patterns); err != nil {
		return fmt.Errorf("redacting annotations: %w", err)
	}

	if err := t.redactPageFormXObjects(ctx, writer, doc, pageDict, patterns, xobjectVisited); err != nil {
		return fmt.Errorf("redacting form XObjects: %w", err)
	}

	return nil
}

// redactStreamRefs decodes each referenced content stream, applies text
// redaction to its bytes, and writes the modified stream back.
//
// Takes ctx (context.Context) which is checked between streams.
// Takes writer (*pdfparse.Writer) which is the PDF writer.
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes pageIndex (int) which is the zero-based page index for error messages.
// Takes refs ([]int) which holds stream object numbers.
// Takes patterns ([]*regexp.Regexp) which holds the compiled text patterns.
//
// Returns error when a stream cannot be read, decoded, or redacted.
func (t *RedactionTransformer) redactStreamRefs(
	ctx context.Context,
	writer *pdfparse.Writer,
	doc *pdfparse.Document,
	pageIndex int,
	refs []int,
	patterns []*regexp.Regexp,
) error {
	for _, ref := range refs {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("redaction: cancelled while redacting page %d: %w", pageIndex, err)
		}
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

		modified, err := t.applyTextRedaction(ctx, decoded, patterns)
		if err != nil {
			return fmt.Errorf("redaction: page %d: %w", pageIndex, err)
		}
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

// applyTextRedaction replaces all regex matches in the stream bytes
// with space characters. The match loop periodically re-checks the
// context so a malicious pattern that produces many matches cannot
// monopolise CPU after the caller has cancelled.
//
// Takes ctx (context.Context) which is sampled every
// cancellationCheckEvery matches.
// Takes data ([]byte) which is the stream content to redact.
// Takes patterns ([]*regexp.Regexp) which holds the compiled text patterns.
//
// Returns []byte which is the redacted stream content.
// Returns error when the context is cancelled mid-match.
func (t *RedactionTransformer) applyTextRedaction(ctx context.Context, data []byte, patterns []*regexp.Regexp) ([]byte, error) {
	result := make([]byte, len(data))
	copy(result, data)

	checkEvery := t.cancellationCheckEvery
	if checkEvery <= 0 {
		checkEvery = defaultCancellationCheckEvery
	}

	for _, re := range patterns {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("redaction: cancelled before pattern match: %w", err)
		}
		if err := redactPatternMatches(ctx, result, re, checkEvery); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// redactPatternMatches replaces every match of re inside result with spaceReplacement.
//
// Cancellation is sampled every checkEvery matches so a malicious pattern that
// produces many matches cannot monopolise CPU after the caller has cancelled.
//
// Takes ctx (context.Context) which is sampled every checkEvery matches.
// Takes result ([]byte) which is the buffer mutated in place at every match
// span.
// Takes re (*regexp.Regexp) which is the pattern matched against result.
// Takes checkEvery (int) which is the cancellation sampling interval.
//
// Returns error when the context is cancelled mid-match.
func redactPatternMatches(ctx context.Context, result []byte, re *regexp.Regexp, checkEvery int) error {
	matches := re.FindAllIndex(result, -1)
	for idx, match := range matches {
		if idx%checkEvery == 0 {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("redaction: cancelled mid-match (idx %d): %w", idx, err)
			}
		}
		for i := match[0]; i < match[1]; i++ {
			result[i] = spaceReplacement
		}
	}
	return nil
}

// redactStringValue replaces every regex match inside value with the same
// number of U+0020 spaces, preserving byte length.
//
// Takes value (string) which is the original string to redact.
// Takes patterns ([]*regexp.Regexp) which holds the compiled text patterns.
//
// Returns string which is the redacted value.
func redactStringValue(value string, patterns []*regexp.Regexp) string {
	if value == "" {
		return value
	}
	out := value
	for _, re := range patterns {
		out = re.ReplaceAllStringFunc(out, func(match string) string {
			return strings.Repeat(" ", len(match))
		})
	}
	return out
}

// redactAnnotations walks /Annots on a page and redacts any /Contents and
// /T string values on each annotation dictionary. Annotations may be
// inline dictionaries or indirect references; both are handled.
//
// Takes ctx (context.Context) which is checked between annotations.
// Takes writer (*pdfparse.Writer) which is the PDF writer.
// Takes pageDict (pdfparse.Dict) which is the page dictionary.
// Takes patterns ([]*regexp.Regexp) which holds the compiled text patterns.
//
// Returns error when context is cancelled.
func (*RedactionTransformer) redactAnnotations(
	ctx context.Context,
	writer *pdfparse.Writer,
	pageDict pdfparse.Dict,
	patterns []*regexp.Regexp,
) error {
	annots := pageDict.GetArray(annotsKey)
	if len(annots) == 0 {
		return nil
	}

	for i := range annots {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("redaction: cancelled in annotations: %w", err)
		}
		entry := annots[i]
		switch entry.Type {
		case pdfparse.ObjectReference:
			ref, ok := entry.Value.(pdfparse.Ref)
			if !ok {
				continue
			}
			obj := writer.GetObject(ref.Number)
			dict, ok := obj.Value.(pdfparse.Dict)
			if !ok {
				continue
			}
			if redactDictStringFields(&dict, patterns, stringFieldKeys) {
				writer.SetObject(ref.Number, pdfparse.DictObj(dict))
			}
		case pdfparse.ObjectDictionary:
			continue
		}
	}
	return nil
}

// redactDictStringFields applies redactStringValue to every dictionary
// entry whose key is in keys and whose value is a literal or hex string.
//
// Takes dict (*pdfparse.Dict) which is mutated in place.
// Takes patterns ([]*regexp.Regexp) which holds the compiled text patterns.
// Takes keys ([]string) which lists the string-valued keys to redact.
//
// Returns bool which is true when at least one value changed.
func redactDictStringFields(dict *pdfparse.Dict, patterns []*regexp.Regexp, keys []string) bool {
	changed := false
	for _, key := range keys {
		obj := dict.Get(key)
		if obj.Type != pdfparse.ObjectString && obj.Type != pdfparse.ObjectHexString {
			continue
		}
		original, ok := obj.Value.(string)
		if !ok {
			continue
		}
		redacted := redactStringValue(original, patterns)
		if redacted == original {
			continue
		}
		dict.Set(key, pdfparse.Object{Type: obj.Type, Value: redacted})
		changed = true
	}
	return changed
}

// redactPageFormXObjects walks the /Resources/XObject map of a page and,
// for each entry whose /Subtype is /Form, redacts the form's own content
// stream and recurses into nested form XObjects.
//
// The visited set is shared across all pages so a form XObject used by
// multiple pages is redacted exactly once. The set also defends against
// cyclic /Resources/XObject references that would otherwise loop forever.
//
// Takes ctx (context.Context) which is checked between XObjects.
// Takes writer (*pdfparse.Writer) which is the PDF writer.
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes pageDict (pdfparse.Dict) which is the page dictionary.
// Takes patterns ([]*regexp.Regexp) which holds the compiled text patterns.
// Takes visited (map[int]struct{}) which records XObjects already
// processed.
//
// Returns error when a form XObject cannot be read or context is cancelled.
func (t *RedactionTransformer) redactPageFormXObjects(
	ctx context.Context,
	writer *pdfparse.Writer,
	doc *pdfparse.Document,
	pageDict pdfparse.Dict,
	patterns []*regexp.Regexp,
	visited map[int]struct{},
) error {
	xobjectRefs := collectXObjectRefs(pageDict)
	for _, ref := range xobjectRefs {
		if err := t.redactFormXObject(ctx, writer, doc, ref, patterns, visited, 0); err != nil {
			return err
		}
	}
	return nil
}

// collectXObjectRefs extracts object numbers from a /Resources/XObject
// sub-dictionary, returning empty when the resources are absent or
// malformed.
//
// Takes parent (pdfparse.Dict) which holds /Resources, typically a page
// dictionary or another form XObject dictionary.
//
// Returns []int which holds the XObject object numbers in document order.
func collectXObjectRefs(parent pdfparse.Dict) []int {
	resources := parent.GetDict(resourcesKey)
	xobject := resources.GetDict(xobjectKey)
	if len(xobject.Pairs) == 0 {
		return nil
	}
	refs := make([]int, 0, len(xobject.Pairs))
	for _, pair := range xobject.Pairs {
		if pair.Value.Type != pdfparse.ObjectReference {
			continue
		}
		ref, ok := pair.Value.Value.(pdfparse.Ref)
		if !ok {
			continue
		}
		refs = append(refs, ref.Number)
	}
	return refs
}

// redactFormXObject redacts a single form XObject's content stream and
// recurses into its own /Resources/XObject children. Image XObjects and
// non-form streams are skipped silently.
//
// Takes ctx (context.Context) which is checked at every recursion step.
// Takes writer (*pdfparse.Writer) which is the PDF writer.
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes objNum (int) which is the form XObject's object number.
// Takes patterns ([]*regexp.Regexp) which holds the compiled text patterns.
// Takes visited (map[int]struct{}) which records XObjects already processed.
// Takes depth (int) which is the current recursion depth.
//
// Returns error when context is cancelled, the depth cap is hit, or the
// stream cannot be redacted.
func (t *RedactionTransformer) redactFormXObject(
	ctx context.Context,
	writer *pdfparse.Writer,
	doc *pdfparse.Document,
	objNum int,
	patterns []*regexp.Regexp,
	visited map[int]struct{},
	depth int,
) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("redaction: cancelled at form XObject %d: %w", objNum, err)
	}
	if depth >= maxXObjectDepth {
		return fmt.Errorf("redaction: form XObject depth exceeded at object %d (depth %d)", objNum, depth)
	}
	if _, seen := visited[objNum]; seen {
		return nil
	}
	visited[objNum] = struct{}{}

	streamObj, err := doc.GetObject(objNum)
	if err != nil {
		return fmt.Errorf("redaction: cannot retrieve form XObject %d: %w", objNum, err)
	}
	if streamObj.Type != pdfparse.ObjectStream {
		return nil
	}
	dict, ok := streamObj.Value.(pdfparse.Dict)
	if !ok {
		return nil
	}
	if dict.GetName(subtypeKey) != formSubtype {
		return nil
	}

	decoded, err := pdfparse.DecodeStream(streamObj)
	if err != nil {
		return fmt.Errorf("redaction: cannot decode form XObject %d: %w", objNum, err)
	}
	modified, err := t.applyTextRedaction(ctx, decoded, patterns)
	if err != nil {
		return fmt.Errorf("redaction: form XObject %d: %w", objNum, err)
	}
	if !bytesEqual(decoded, modified) {
		writer.SetObject(objNum, pdfparse.StreamObj(dict, modified))
	}

	for _, childRef := range collectXObjectRefs(dict) {
		if err := t.redactFormXObject(ctx, writer, doc, childRef, patterns, visited, depth+1); err != nil {
			return err
		}
	}
	return nil
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

	visited := make(map[int]struct{})
	return walkPageTree(doc, pagesRef.Number, visited, 0)
}

// walkPageTree recursively collects leaf Page object numbers from a Pages
// tree node.
//
// The visited set records every node already entered so that cyclic
// /Kids references skip the already-seen branch instead of recursing
// forever. Skipped branches return no pages and no error so the rest
// of the tree continues to be walked.
//
// Takes doc (*pdfparse.Document) which is the parsed PDF document.
// Takes objNum (int) which is the current tree node's object number.
// Takes visited (map[int]struct{}) which records nodes already entered.
// Takes depth (int) which is the current recursion depth.
//
// Returns []int which holds the collected page object numbers.
// Returns error when any node cannot be read or the depth cap is hit.
func walkPageTree(doc *pdfparse.Document, objNum int, visited map[int]struct{}, depth int) ([]int, error) {
	if depth >= maxPageTreeDepth {
		return nil, fmt.Errorf("page tree depth exceeded at object %d (depth %d)", objNum, depth)
	}
	if _, seen := visited[objNum]; seen {
		return nil, nil
	}
	visited[objNum] = struct{}{}

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
		childPages, err := walkPageTree(doc, ref.Number, visited, depth+1)
		if err != nil {
			return nil, err
		}
		pages = append(pages, childPages...)
	}
	return pages, nil
}
