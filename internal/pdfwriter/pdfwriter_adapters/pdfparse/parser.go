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

package pdfparse

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

// xrefSearchWindow is the maximum number of bytes from the end of the file
// to scan when searching for the startxref marker.
const xrefSearchWindow = 1024

// defaultMaxDecompressedStreamBytes is the default cap on the size of a
// single decompressed FlateDecode stream (64 MiB). The cap exists to
// guard against zip-bomb PDFs that decompress to terabytes.
const defaultMaxDecompressedStreamBytes int64 = 64 << 20

// maxXRefDepth caps the number of /Prev cross-reference sections the
// parser will follow before giving up. Traditional PDFs rarely chain
// more than a handful of incremental updates; this cap stops malformed
// chains from exhausting the stack even in the absence of cycles.
const maxXRefDepth = 64

// maxParseDepth caps the recursion depth of parseToken/parseDictionary/parseArray.
//
// Legitimate PDFs nest dictionaries and arrays for resources, fonts, and
// form fields, but rarely beyond a few dozen levels. The cap rejects
// malicious payloads engineered to blow the stack while leaving genuine
// documents untouched. The value is deliberately set above downstream
// transformer nesting caps (e.g. the 256-level encrypt cap) so those
// layers can still produce their own targeted errors before the
// parser-level backstop fires.
const maxParseDepth = 512

// ErrXRefStreamsUnsupported is returned when the parser encounters a
// cross-reference stream instead of a traditional xref table.
var ErrXRefStreamsUnsupported = errors.New("xref streams not yet supported; only traditional xref tables are handled")

// ErrFlateStreamTooLarge is returned when a FlateDecode stream
// decompresses to more bytes than the configured cap, indicating a
// likely zip-bomb attack rather than a legitimate document.
var ErrFlateStreamTooLarge = errors.New("FlateDecode stream exceeds maximum decompressed size")

// ErrXRefCycle is returned when the parser detects a cycle in the
// /Prev chain of cross-reference sections. Following a cycle would
// recurse forever and exhaust the stack.
var ErrXRefCycle = errors.New("xref /Prev chain contains a cycle")

// ErrXRefDepthExceeded is returned when the /Prev chain is longer
// than [maxXRefDepth] sections. The cap defends against pathological
// chains that, while not strictly cyclic, are deep enough to threaten
// the stack.
var ErrXRefDepthExceeded = errors.New("xref /Prev chain exceeds maximum depth")

// ErrParseDepthExceeded is returned when nested dictionaries or
// arrays exceed [maxParseDepth] levels of recursion.
var ErrParseDepthExceeded = errors.New("parser recursion depth exceeded")

// ErrInvalidObjectOffset is returned when a cross-reference entry
// points to a byte offset that lies outside the file content.
var ErrInvalidObjectOffset = errors.New("xref offset outside file bounds")

// ErrMalformedStreamLength is returned when a stream's /Length
// exceeds the bytes remaining in the file, indicating a truncated or
// hostile stream rather than a legitimate document.
var ErrMalformedStreamLength = errors.New("stream /Length exceeds remaining file bytes")

// maxDecompressedStreamBytes holds the active cap on FlateDecode
// decompression output. It defaults to defaultMaxDecompressedStreamBytes
// and may be overridden via [SetMaxDecompressedStreamBytes].
var maxDecompressedStreamBytes = defaultMaxDecompressedStreamBytes

// SetMaxDecompressedStreamBytes sets the cap on the size of a single
// decompressed FlateDecode stream. Callers may raise this for trusted
// inputs but should not lower it below the largest legitimate stream
// they expect to encounter.
//
// Takes limit (int64) which is the new cap in bytes. Values <= 0 are
// ignored to keep the existing limit safe.
func SetMaxDecompressedStreamBytes(limit int64) {
	if limit <= 0 {
		return
	}
	maxDecompressedStreamBytes = limit
}

// MaxDecompressedStreamBytes returns the active cap on FlateDecode
// decompression output.
//
// Returns int64 which is the active cap in bytes.
func MaxDecompressedStreamBytes() int64 { return maxDecompressedStreamBytes }

// xrefEntry represents one entry in the cross-reference table.
type xrefEntry struct {
	// offset is the byte offset of the object in the file.
	offset int64

	// generation is the generation number of the object.
	generation int

	// inUse is true for objects in use ('n'), false for free objects ('f').
	inUse bool
}

// Document represents a parsed PDF document. It holds the raw bytes, the
// cross-reference table, the trailer dictionary, and a cache of parsed
// objects.
type Document struct {
	// xref maps object numbers to their cross-reference entries.
	xref map[int]xrefEntry

	// objectCache caches parsed objects by object number.
	objectCache map[int]Object

	// raw holds the original PDF bytes.
	raw []byte

	// trailer holds the parsed trailer dictionary.
	trailer Dict
}

// Parse reads a PDF document from raw bytes and returns a Document that
// can be used to inspect and modify objects.
//
// Takes data ([]byte) which is the complete PDF file bytes.
//
// Returns *Document which provides access to the parsed PDF structure.
// Returns error when the PDF cannot be parsed.
func Parse(data []byte) (*Document, error) {
	doc := &Document{
		raw:         data,
		xref:        make(map[int]xrefEntry),
		objectCache: make(map[int]Object),
	}

	xrefOffset, err := findXRefOffset(data)
	if err != nil {
		return nil, fmt.Errorf("finding xref offset: %w", err)
	}

	visited := make(map[int64]struct{})
	if err := doc.parseXRef(xrefOffset, visited, 0); err != nil {
		return nil, fmt.Errorf("parsing xref: %w", err)
	}

	return doc, nil
}

// Trailer returns the trailer dictionary.
//
// Returns Dict which holds the parsed trailer entries.
func (d *Document) Trailer() Dict { return d.trailer }

// ObjectCount returns the number of objects in the cross-reference table.
//
// Returns int which is the total object count including free entries.
func (d *Document) ObjectCount() int { return len(d.xref) }

// ObjectNumbers returns all in-use object numbers in ascending order.
//
// Returns []int which holds the object numbers marked as in-use.
func (d *Document) ObjectNumbers() []int {
	numbers := make([]int, 0, len(d.xref))
	for num, entry := range d.xref {
		if entry.inUse {
			numbers = append(numbers, num)
		}
	}
	slices.Sort(numbers)
	return numbers
}

// GetObject returns the parsed object for the given object number. Objects
// are cached after the first parse.
//
// Takes number (int) which is the object number to retrieve.
//
// Returns Object which is the parsed PDF object.
// Returns error when the object cannot be found or parsed.
func (d *Document) GetObject(number int) (Object, error) {
	if cached, ok := d.objectCache[number]; ok {
		return cached, nil
	}

	entry, ok := d.xref[number]
	if !ok || !entry.inUse {
		return Null(), fmt.Errorf("object %d not found in xref", number)
	}

	obj, err := d.parseObjectAt(entry.offset)
	if err != nil {
		return Null(), fmt.Errorf("parsing object %d at offset %d: %w", number, entry.offset, err)
	}

	d.objectCache[number] = obj
	return obj, nil
}

// Resolve follows an indirect reference to its target object. If the input
// is not a reference, it is returned unchanged.
//
// Takes obj (Object) which may be a reference or a direct object.
//
// Returns Object which is the resolved object.
// Returns error when a referenced object cannot be found.
func (d *Document) Resolve(obj Object) (Object, error) {
	if obj.Type != ObjectReference {
		return obj, nil
	}
	ref, ok := obj.Value.(Ref)
	if !ok {
		return Null(), errors.New("reference object has invalid value type")
	}
	return d.GetObject(ref.Number)
}

// Raw returns the original PDF bytes.
//
// Returns []byte which holds the raw file content.
func (d *Document) Raw() []byte { return d.raw }

// DecodeStream decompresses a stream object's data if it uses FlateDecode.
// If the stream has no filter or an unsupported filter, the raw data is
// returned.
//
// Takes obj (Object) which must be an ObjectStream.
//
// Returns []byte which is the decoded stream content.
// Returns error when decompression fails.
func DecodeStream(obj Object) ([]byte, error) {
	if obj.Type != ObjectStream {
		return nil, errors.New("not a stream object")
	}

	dict, ok := obj.Value.(Dict)
	if !ok {
		return nil, errors.New("stream object has invalid dictionary")
	}
	filter := dict.GetName("Filter")

	if filter == "FlateDecode" {
		return deflateDecode(obj.StreamData)
	}
	return obj.StreamData, nil
}

// FindStartXRefOffset returns the byte offset recorded in the last
// startxref marker of a PDF.
//
// Takes data ([]byte) which holds the complete PDF file bytes.
//
// Returns int64 which is the byte offset of the cross-reference table.
// Returns error when the startxref marker cannot be found or parsed.
func FindStartXRefOffset(data []byte) (int64, error) {
	return findXRefOffset(data)
}

// findXRefOffset locates the byte offset of the xref table by reading the
// startxref marker near the end of the file.
//
// Takes data ([]byte) which holds the complete PDF file bytes.
//
// Returns int64 which is the byte offset of the cross-reference table.
// Returns error when the startxref marker cannot be found or parsed.
func findXRefOffset(data []byte) (int64, error) {
	searchLen := min(len(data), xrefSearchWindow)
	tail := data[len(data)-searchLen:]

	idx := bytes.LastIndex(tail, []byte("startxref"))
	if idx < 0 {
		return 0, errors.New("startxref not found")
	}

	pos := idx + len("startxref")
	numStr := strings.TrimSpace(string(tail[pos:]))

	if eofIdx := strings.Index(numStr, "%%EOF"); eofIdx >= 0 {
		numStr = strings.TrimSpace(numStr[:eofIdx])
	}
	if nlIdx := strings.IndexByte(numStr, '\n'); nlIdx >= 0 {
		numStr = strings.TrimSpace(numStr[:nlIdx])
	}

	offset, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid startxref offset %q: %w", numStr, err)
	}
	return offset, nil
}

// parseXRef parses the cross-reference table and trailer starting at the
// given byte offset.
//
// The visited map records every offset that has already been entered so
// that a cyclic /Prev chain (e.g. one that points back to the section
// that originally referenced it) is detected before it can blow the
// stack. depth is incremented per recursion and capped by
// [maxXRefDepth] as belt-and-braces against pathological non-cyclic
// chains.
//
// Takes offset (int64) which specifies the byte position of the xref section.
// Takes visited (map[int64]struct{}) which records xref offsets already entered.
// Takes depth (int) which is the current /Prev recursion depth.
//
// Returns error when parsing fails or a cycle/depth limit is hit.
func (d *Document) parseXRef(offset int64, visited map[int64]struct{}, depth int) error {
	if depth >= maxXRefDepth {
		return fmt.Errorf("%w: depth %d", ErrXRefDepthExceeded, depth)
	}
	if _, seen := visited[offset]; seen {
		return fmt.Errorf("%w: offset %d already visited", ErrXRefCycle, offset)
	}
	visited[offset] = struct{}{}

	if offset < 0 || offset >= int64(len(d.raw)) {
		return fmt.Errorf("xref offset %d beyond file size %d", offset, len(d.raw))
	}
	pos := int(offset)

	if bytes.HasPrefix(d.raw[pos:], []byte("xref")) {
		return d.parseTraditionalXRef(pos, visited, depth)
	}

	return parseXRefStream()
}

// parseTraditionalXRef parses a traditional "xref ... trailer" section.
//
// Takes pos (int) which specifies the byte offset of the "xref" keyword.
// Takes visited (map[int64]struct{}) which records xref offsets already entered.
// Takes depth (int) which is the current /Prev chain depth.
//
// Returns error when the xref section is malformed.
func (d *Document) parseTraditionalXRef(pos int, visited map[int64]struct{}, depth int) error {
	sc := newScanner(d.raw, pos)

	token := sc.next()
	if token != "xref" {
		return fmt.Errorf("expected 'xref', got %q", token)
	}

	for {
		token = sc.peek()
		if token == "trailer" {
			break
		}
		if err := d.parseXRefSubsection(sc); err != nil {
			return err
		}
	}

	return d.parseTrailerAndFollowPrev(sc, visited, depth)
}

// parseXRefSubsection parses one "startObj count" subsection of the xref
// table.
//
// Takes sc (*scanner) which provides token-level reading of the xref data.
//
// Returns error when the subsection entries are malformed.
func (d *Document) parseXRefSubsection(sc *scanner) error {
	startObj, err := strconv.Atoi(sc.next())
	if err != nil {
		return fmt.Errorf("invalid xref start object: %w", err)
	}
	count, err := strconv.Atoi(sc.next())
	if err != nil {
		return fmt.Errorf("invalid xref count: %w", err)
	}

	for range count {
		offsetStr := sc.next()
		genStr := sc.next()
		flag := sc.next()

		entryOffset, err := strconv.ParseInt(offsetStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid xref offset: %w", err)
		}
		gen, err := strconv.Atoi(genStr)
		if err != nil {
			return fmt.Errorf("invalid xref generation: %w", err)
		}

		objNum := startObj
		startObj++
		if _, exists := d.xref[objNum]; !exists {
			d.xref[objNum] = xrefEntry{
				offset:     entryOffset,
				generation: gen,
				inUse:      flag == "n",
			}
		}
	}
	return nil
}

// parseTrailerAndFollowPrev parses the trailer dictionary and follows any
// /Prev link to earlier xref sections.
//
// Takes sc (*scanner) which provides token-level reading positioned at
// the trailer keyword.
// Takes visited (map[int64]struct{}) which records xref offsets already entered.
// Takes depth (int) which is the current /Prev chain depth.
//
// Returns error when the trailer dictionary is malformed or a /Prev link
// cannot be followed.
func (d *Document) parseTrailerAndFollowPrev(sc *scanner, visited map[int64]struct{}, depth int) error {
	sc.next()
	trailerObj, err := d.parseObjectFromScanner(sc)
	if err != nil {
		return fmt.Errorf("parsing trailer: %w", err)
	}
	if trailerObj.Type != ObjectDictionary {
		return errors.New("trailer is not a dictionary")
	}

	trailerDict, ok := trailerObj.Value.(Dict)
	if !ok {
		return errors.New("trailer dictionary has invalid type")
	}

	if len(d.trailer.Pairs) == 0 {
		d.trailer = trailerDict
	}

	prevObj := trailerDict.Get("Prev")
	if prevObj.Type == ObjectInteger {
		if prevVal, ok := prevObj.Value.(int64); ok {
			return d.parseXRef(prevVal, visited, depth+1)
		}
	}

	return nil
}

// parseXRefStream is a stub for xref stream parsing (PDF 1.5+).
//
// Returns error which is always ErrXRefStreamsUnsupported.
func parseXRefStream() error {
	return ErrXRefStreamsUnsupported
}

// parseObjectAt parses an indirect object definition at the given byte
// offset.
//
// The offset is bounds-checked against the document length so that a
// malformed xref entry pointing past EOF cannot trigger out-of-range
// reads or silent data corruption.
//
// Takes offset (int64) which specifies the byte offset of the object header.
//
// Returns Object which is the parsed object value.
// Returns error when the object definition is malformed.
func (d *Document) parseObjectAt(offset int64) (Object, error) {
	if offset < 0 || offset >= int64(len(d.raw)) {
		return Null(), fmt.Errorf("%w: offset %d, file size %d", ErrInvalidObjectOffset, offset, len(d.raw))
	}
	pos := int(offset)
	sc := newScanner(d.raw, pos)

	sc.next()
	sc.next()
	objToken := sc.next()
	if objToken != "obj" {
		return Null(), fmt.Errorf("expected 'obj', got %q", objToken)
	}

	obj, err := d.parseObjectFromScanner(sc)
	if err != nil {
		return Null(), err
	}

	if obj.Type == ObjectDictionary {
		nextToken := sc.peek()
		if nextToken == "stream" {
			sc.next()
			dict, ok := obj.Value.(Dict)
			if !ok {
				return Null(), errors.New("stream dictionary has invalid type")
			}
			streamData, err := d.readStreamData(sc, dict)
			if err != nil {
				return Null(), fmt.Errorf("reading stream data: %w", err)
			}
			return StreamObj(dict, streamData), nil
		}
	}

	return obj, nil
}

// parseObjectFromScanner parses a single PDF object value from the scanner
// starting at depth zero.
//
// Takes sc (*scanner) which provides token-level reading of PDF content.
//
// Returns Object which is the parsed PDF object.
// Returns error when the token sequence is invalid or the recursion
// depth limit is exceeded.
func (d *Document) parseObjectFromScanner(sc *scanner) (Object, error) {
	token := sc.next()
	return d.parseToken(token, sc, 0)
}

// parseToken interprets a token and produces a PDF Object. depth tracks
// the current nesting level so that cyclic or pathologically nested
// dictionaries and arrays cannot exhaust the stack.
//
// Takes token (string) which is the initial token to interpret.
// Takes sc (*scanner) which provides additional tokens if needed.
// Takes depth (int) which is the current recursion depth.
//
// Returns Object which is the parsed PDF object.
// Returns error when the token cannot be interpreted or the recursion
// depth limit is exceeded.
func (d *Document) parseToken(token string, sc *scanner, depth int) (Object, error) {
	if depth >= maxParseDepth {
		return Null(), fmt.Errorf("%w: depth %d", ErrParseDepthExceeded, depth)
	}
	switch {
	case token == "null":
		return Null(), nil
	case token == "true":
		return Bool(true), nil
	case token == "false":
		return Bool(false), nil
	case token == "<<":
		return d.parseDictionary(sc, depth+1)
	case token == "[":
		return d.parseArray(sc, depth+1)
	case strings.HasPrefix(token, "/"):
		return Name(token[1:]), nil
	case strings.HasPrefix(token, "("):
		return Str(decodeLiteralString(token)), nil
	case strings.HasPrefix(token, "<"):
		return HexStr(decodeHexString(token)), nil
	default:
		return parseNumberOrRef(token, sc)
	}
}

// parseDictionary parses key-value pairs until the closing >> delimiter.
//
// Takes sc (*scanner) which provides token-level reading positioned after
// the opening << delimiter.
// Takes depth (int) which is the current recursion depth.
//
// Returns Object which is the parsed dictionary object.
// Returns error when the dictionary content is malformed or the
// recursion depth limit is exceeded.
func (d *Document) parseDictionary(sc *scanner, depth int) (Object, error) {
	if depth >= maxParseDepth {
		return Null(), fmt.Errorf("%w: dictionary depth %d", ErrParseDepthExceeded, depth)
	}
	dict := Dict{}
	for {
		token := sc.next()
		if token == ">>" || token == "" {
			break
		}
		if !strings.HasPrefix(token, "/") {
			return Null(), fmt.Errorf("expected name key in dict, got %q", token)
		}
		key := token[1:]

		valueToken := sc.next()
		value, err := d.parseToken(valueToken, sc, depth)
		if err != nil {
			return Null(), fmt.Errorf("parsing dict value for key %q: %w", key, err)
		}
		dict.Set(key, value)
	}
	return DictObj(dict), nil
}

// parseArray parses array elements until the closing ] delimiter.
//
// Takes sc (*scanner) which provides token-level reading positioned after
// the opening [ delimiter.
// Takes depth (int) which is the current recursion depth.
//
// Returns Object which is the parsed array object.
// Returns error when an array element cannot be parsed or the recursion
// depth limit is exceeded.
func (d *Document) parseArray(sc *scanner, depth int) (Object, error) {
	if depth >= maxParseDepth {
		return Null(), fmt.Errorf("%w: array depth %d", ErrParseDepthExceeded, depth)
	}
	var items []Object
	for {
		token := sc.next()
		if token == "]" || token == "" {
			break
		}
		item, err := d.parseToken(token, sc, depth)
		if err != nil {
			return Null(), fmt.Errorf("parsing array item: %w", err)
		}
		items = append(items, item)
	}
	return Arr(items...), nil
}

// parseNumberOrRef parses a numeric token as an integer, real, or indirect
// reference.
//
// Takes token (string) which is the first numeric token.
// Takes sc (*scanner) which provides look-ahead for "N G R" reference
// syntax.
//
// Returns Object which is the parsed number or reference.
// Returns error when the token is not a valid number or reference.
func parseNumberOrRef(token string, sc *scanner) (Object, error) {
	intVal, err := strconv.ParseInt(token, 10, 64)
	if err == nil {
		peeked := sc.peek()
		if isInteger(peeked) {
			savedPos := sc.pos
			genToken := sc.next()
			rToken := sc.peek()
			if rToken == "R" {
				sc.next()
				gen, _ := strconv.Atoi(genToken)
				return RefObj(int(intVal), gen), nil
			}

			sc.pos = savedPos
		}
		return Int(intVal), nil
	}

	floatVal, err := strconv.ParseFloat(token, 64)
	if err == nil {
		return Real(floatVal), nil
	}

	return Null(), fmt.Errorf("unexpected token %q", token)
}

// readStreamData reads the raw stream bytes following a "stream" keyword.
//
// Takes sc (*scanner) which is positioned immediately after the "stream"
// token.
// Takes dict (Dict) which holds the stream dictionary containing /Length.
//
// Returns []byte which is the raw stream content.
// Returns error when the /Length cannot be resolved or is invalid.
func (d *Document) readStreamData(sc *scanner, dict Dict) ([]byte, error) {
	lengthObj := dict.Get("Length")

	var length int64
	switch lengthObj.Type {
	case ObjectInteger:
		if v, ok := lengthObj.Value.(int64); ok {
			length = v
		}
	case ObjectReference:
		resolved, err := d.Resolve(lengthObj)
		if err != nil {
			return nil, fmt.Errorf("resolving stream /Length reference: %w", err)
		}
		if resolved.Type != ObjectInteger {
			return nil, errors.New("stream /Length reference did not resolve to integer")
		}
		if v, ok := resolved.Value.(int64); ok {
			length = v
		}
	default:
		return nil, fmt.Errorf("stream /Length is %v, expected integer", lengthObj.Type)
	}

	streamStart := sc.pos

	if streamStart < len(d.raw) && d.raw[streamStart] == '\r' {
		streamStart++
	}
	if streamStart < len(d.raw) && d.raw[streamStart] == '\n' {
		streamStart++
	}

	if length < 0 {
		return nil, fmt.Errorf("%w: negative /Length %d", ErrMalformedStreamLength, length)
	}
	remaining := int64(len(d.raw) - streamStart)
	if length > remaining {
		return nil, fmt.Errorf("%w: /Length %d, remaining %d", ErrMalformedStreamLength, length, remaining)
	}

	end := streamStart + int(length)
	sc.pos = end
	return d.raw[streamStart:end], nil
}

// deflateDecode decompresses zlib-compressed data, enforcing the
// active FlateDecode size cap to guard against zip-bomb PDFs.
//
// Takes data ([]byte) which holds the compressed stream bytes.
//
// Returns []byte which is the decompressed content.
// Returns error when decompression fails or the output exceeds the
// configured cap.
func deflateDecode(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating zlib reader: %w", err)
	}
	defer reader.Close()

	limit := maxDecompressedStreamBytes
	decoded, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, fmt.Errorf("decompressing FlateDecode stream: %w", err)
	}
	if int64(len(decoded)) > limit {
		return nil, fmt.Errorf("decompressing FlateDecode stream (limit %d bytes): %w", limit, ErrFlateStreamTooLarge)
	}
	return decoded, nil
}

// scanner provides simple token-level reading of PDF content.
type scanner struct {
	// data holds the raw PDF bytes being scanned.
	data []byte

	// pos holds the current byte position in data.
	pos int
}

// newScanner creates a scanner positioned at the given offset in data.
//
// Takes data ([]byte) which holds the raw PDF bytes.
// Takes pos (int) which specifies the starting byte position.
//
// Returns *scanner which is ready to read tokens.
func newScanner(data []byte, pos int) *scanner {
	return &scanner{data: data, pos: pos}
}

// skipWhitespaceAndComments advances past whitespace and PDF comments.
func (s *scanner) skipWhitespaceAndComments() {
	for s.pos < len(s.data) {
		ch := s.data[s.pos]
		if ch == '%' {
			for s.pos < len(s.data) && s.data[s.pos] != '\n' && s.data[s.pos] != '\r' {
				s.pos++
			}
			continue
		}
		if isWhitespace(ch) {
			s.pos++
			continue
		}
		break
	}
}

// next returns the next token and advances the position.
//
// Returns string which is the next PDF token, or empty string at end of
// data.
func (s *scanner) next() string {
	s.skipWhitespaceAndComments()
	if s.pos >= len(s.data) {
		return ""
	}

	ch := s.data[s.pos]

	switch ch {
	case '[', ']':
		s.pos++
		return string(ch)
	case '<':
		return s.readAngleBracketToken()
	case '>':
		return s.readCloseAngleBracketToken()
	case '(':
		return s.readLiteralString()
	case '/':
		return s.readName()
	}

	start := s.pos
	for s.pos < len(s.data) {
		c := s.data[s.pos]
		if isWhitespace(c) || isDelimiter(c) {
			break
		}
		s.pos++
	}
	return string(s.data[start:s.pos])
}

// peek returns the next token without advancing the position.
//
// Returns string which is the next PDF token, or empty string at end of
// data.
func (s *scanner) peek() string {
	saved := s.pos
	token := s.next()
	s.pos = saved
	return token
}

// readAngleBracketToken reads a "<<" dictionary delimiter or a hex string
// token starting with "<".
//
// Returns string which is the parsed angle bracket token.
func (s *scanner) readAngleBracketToken() string {
	if s.pos+1 < len(s.data) && s.data[s.pos+1] == '<' {
		s.pos += 2
		return "<<"
	}

	start := s.pos
	s.pos++
	for s.pos < len(s.data) && s.data[s.pos] != '>' {
		s.pos++
	}
	if s.pos < len(s.data) {
		s.pos++
	}
	return string(s.data[start:s.pos])
}

// readCloseAngleBracketToken reads a ">>" dictionary delimiter or a
// single ">" character.
//
// Returns string which is the parsed closing angle bracket token.
func (s *scanner) readCloseAngleBracketToken() string {
	if s.pos+1 < len(s.data) && s.data[s.pos+1] == '>' {
		s.pos += 2
		return ">>"
	}
	s.pos++
	return ">"
}

// readLiteralString reads a parenthesised PDF literal string, handling
// nested parentheses and backslash escapes.
//
// Returns string which is the raw literal string including parentheses.
func (s *scanner) readLiteralString() string {
	start := s.pos
	s.pos++
	depth := 1
	for s.pos < len(s.data) && depth > 0 {
		ch := s.data[s.pos]
		switch ch {
		case '\\':
			s.pos += 2
			continue
		case '(':
			depth++
		case ')':
			depth--
		}
		s.pos++
	}
	return string(s.data[start:s.pos])
}

// readName reads a PDF name token starting with "/" and extending until
// the next whitespace or delimiter.
//
// Returns string which is the raw name token including the leading slash.
func (s *scanner) readName() string {
	start := s.pos
	s.pos++
	for s.pos < len(s.data) {
		c := s.data[s.pos]
		if isWhitespace(c) || isDelimiter(c) {
			break
		}
		s.pos++
	}
	return string(s.data[start:s.pos])
}

// isWhitespace reports whether ch is a PDF whitespace character.
//
// Takes ch (byte) which is the byte to test.
//
// Returns bool which indicates whether the byte is whitespace.
func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f' || ch == 0
}

// isDelimiter reports whether ch is a PDF delimiter character.
//
// Takes ch (byte) which is the byte to test.
//
// Returns bool which indicates whether the byte is a delimiter.
func isDelimiter(ch byte) bool {
	return ch == '(' || ch == ')' || ch == '<' || ch == '>' ||
		ch == '[' || ch == ']' || ch == '{' || ch == '}' ||
		ch == '/' || ch == '%'
}

// isInteger reports whether s represents a valid decimal integer.
//
// Takes s (string) which is the string to test.
//
// Returns bool which indicates whether the string is an integer.
func isInteger(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// decodeLiteralString decodes a PDF literal string token by stripping
// outer parentheses and interpreting backslash escape sequences.
//
// Takes token (string) which is the raw literal string token.
//
// Returns string which is the decoded string content.
func decodeLiteralString(token string) string {
	if len(token) >= 2 && token[0] == '(' && token[len(token)-1] == ')' {
		token = token[1 : len(token)-1]
	}

	var b strings.Builder
	for i := 0; i < len(token); i++ {
		if token[i] == '\\' && i+1 < len(token) {
			i++
			switch token[i] {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			default:

				b.WriteByte(token[i])
			}
		} else {
			b.WriteByte(token[i])
		}
	}
	return b.String()
}

// decodeHexString decodes a PDF hex string token by stripping angle
// brackets and converting hex digit pairs to bytes.
//
// Takes token (string) which is the raw hex string token.
//
// Returns string which is the decoded byte content as a string.
func decodeHexString(token string) string {
	if len(token) >= 2 && token[0] == '<' && token[len(token)-1] == '>' {
		token = token[1 : len(token)-1]
	}

	var b strings.Builder
	hex := strings.ReplaceAll(token, " ", "")
	for i := 0; i+1 < len(hex); i += 2 {
		val, err := strconv.ParseUint(hex[i:i+2], 16, 8)
		if err == nil {
			b.WriteByte(byte(val))
		}
	}

	if len(hex)%2 == 1 {
		val, err := strconv.ParseUint(string(hex[len(hex)-1])+"0", 16, 8)
		if err == nil {
			b.WriteByte(byte(val))
		}
	}
	return b.String()
}
