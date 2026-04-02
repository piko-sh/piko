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
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Writer serialises a set of PDF objects into a valid PDF file with a
// traditional cross-reference table and trailer.
type Writer struct {
	// objects maps object numbers to their Object values.
	objects map[int]Object

	// trailer holds the trailer dictionary for the output file.
	trailer Dict

	// nextObjNum tracks the next available object number.
	nextObjNum int
}

// NewWriter creates a new PDF writer.
//
// Returns *Writer which is an empty writer ready to receive objects.
func NewWriter() *Writer {
	return &Writer{
		objects:    make(map[int]Object),
		nextObjNum: 1,
	}
}

// NewWriterFromDocument creates a writer pre-populated with all objects from
// a parsed Document.
//
// Takes doc (*Document) which is the parsed PDF to copy objects from.
//
// Returns *Writer which contains all objects from the document.
// Returns error when an object cannot be read from the document.
func NewWriterFromDocument(doc *Document) (*Writer, error) {
	w := NewWriter()

	for _, num := range doc.ObjectNumbers() {
		obj, err := doc.GetObject(num)
		if err != nil {
			return nil, fmt.Errorf("reading object %d: %w", num, err)
		}
		w.objects[num] = obj
		if num >= w.nextObjNum {
			w.nextObjNum = num + 1
		}
	}

	w.trailer = doc.Trailer()
	return w, nil
}

// SetTrailer sets the trailer dictionary for the output file.
//
// Takes trailer (Dict) which specifies the new trailer dictionary.
func (w *Writer) SetTrailer(trailer Dict) {
	w.trailer = trailer
}

// Trailer returns the current trailer dictionary.
//
// Returns Dict which holds the trailer entries.
func (w *Writer) Trailer() Dict {
	return w.trailer
}

// AddObject adds an object with the next available object number and
// returns the assigned number.
//
// Takes obj (Object) which is the object to add.
//
// Returns int which is the assigned object number.
func (w *Writer) AddObject(obj Object) int {
	num := w.nextObjNum
	w.nextObjNum++
	w.objects[num] = obj
	return num
}

// SetObject sets an object at a specific object number, replacing any
// existing object.
//
// Takes num (int) which specifies the object number to assign.
// Takes obj (Object) which is the object to store.
func (w *Writer) SetObject(num int, obj Object) {
	w.objects[num] = obj
	if num >= w.nextObjNum {
		w.nextObjNum = num + 1
	}
}

// GetObject returns the object at the given number, or a null Object if
// not found.
//
// Takes num (int) which specifies the object number to retrieve.
//
// Returns Object which is the stored object, or null if absent.
func (w *Writer) GetObject(num int) Object {
	obj, ok := w.objects[num]
	if !ok {
		return Null()
	}
	return obj
}

// RemoveObject removes the object with the given number.
//
// Takes num (int) which specifies the object number to delete.
func (w *Writer) RemoveObject(num int) {
	delete(w.objects, num)
}

// NextObjectNumber returns the next available object number without
// allocating it.
//
// Returns int which is the next object number that will be assigned.
func (w *Writer) NextObjectNumber() int {
	return w.nextObjNum
}

// Write serialises all objects into a complete PDF file.
//
// Returns []byte which is the valid PDF document.
// Returns error when serialisation fails.
func (w *Writer) Write() ([]byte, error) {
	var buf strings.Builder

	buf.WriteString("%PDF-1.7\n")
	buf.WriteString("%\xE2\xE3\xCF\xD3\n")

	objNums := make([]int, 0, len(w.objects))
	for num := range w.objects {
		objNums = append(objNums, num)
	}
	slices.Sort(objNums)

	offsets := make(map[int]int64, len(objNums))
	for _, num := range objNums {
		offsets[num] = int64(buf.Len())
		writeIndirectObject(&buf, num, w.objects[num])
	}

	xrefOffset := buf.Len()
	writeXRefTable(&buf, objNums, offsets)

	maxObjNum := 0
	for _, num := range objNums {
		if num > maxObjNum {
			maxObjNum = num
		}
	}
	w.trailer.Set("Size", Int(int64(maxObjNum+1)))

	buf.WriteString("trailer\n")
	writeObject(&buf, DictObj(w.trailer))
	buf.WriteByte('\n')

	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOffset)

	return []byte(buf.String()), nil
}

// WriteIncremental appends only the specified objects as an incremental
// update to existingPDF. The new xref section references only the
// appended objects, and the trailer includes a /Prev entry pointing to
// the original startxref offset so readers can follow the chain.
//
// Takes existingPDF ([]byte) which is the original signed PDF.
// Takes dirtyNums ([]int) which lists the object numbers to write.
// Takes prevStartXRef (int64) which is the startxref offset of the
// existing PDF's xref table.
//
// Returns []byte which is the complete PDF with incremental update.
// Returns error when serialisation fails.
func (w *Writer) WriteIncremental(existingPDF []byte, dirtyNums []int, prevStartXRef int64) ([]byte, error) {
	var buf strings.Builder
	buf.Write(existingPDF)

	slices.Sort(dirtyNums)

	offsets := make(map[int]int64, len(dirtyNums))
	for _, num := range dirtyNums {
		obj, ok := w.objects[num]
		if !ok {
			continue
		}
		offsets[num] = int64(buf.Len())
		writeIndirectObject(&buf, num, obj)
	}

	xrefOffset := buf.Len()
	writeXRefTable(&buf, dirtyNums, offsets)

	maxObjNum := 0
	for num := range w.objects {
		if num > maxObjNum {
			maxObjNum = num
		}
	}
	w.trailer.Set("Size", Int(int64(maxObjNum+1)))
	w.trailer.Set("Prev", Int(prevStartXRef))

	buf.WriteString("trailer\n")
	writeObject(&buf, DictObj(w.trailer))
	buf.WriteByte('\n')

	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOffset)

	return []byte(buf.String()), nil
}

// writeIndirectObject writes a complete "N 0 obj ... endobj" block to buf.
//
// Takes buf (*strings.Builder) which is the output buffer.
// Takes num (int) which is the object number.
// Takes obj (Object) which is the object to serialise.
func writeIndirectObject(buf *strings.Builder, num int, obj Object) {
	fmt.Fprintf(buf, "%d 0 obj\n", num)

	if obj.Type == ObjectStream {
		dict, ok := obj.Value.(Dict)
		if !ok {
			buf.WriteString("null")
			buf.WriteString("\nendobj\n")
			return
		}
		data := obj.StreamData

		filter := dict.GetName("Filter")
		if filter == "" && len(data) > 0 {
			compressed, err := flateEncode(data)
			if err == nil && len(compressed) < len(data) {
				data = compressed
				dict.Set("Filter", Name("FlateDecode"))
			}
		}

		dict.Set("Length", Int(int64(len(data))))
		writeObject(buf, DictObj(dict))
		buf.WriteString("\nstream\n")
		buf.Write(data)
		buf.WriteString("\nendstream")
	} else {
		writeObject(buf, obj)
	}

	buf.WriteString("\nendobj\n")
}

// writeObject writes a single PDF object value to buf in PDF syntax.
//
// Takes buf (*strings.Builder) which is the output buffer.
// Takes obj (Object) which is the object to serialise.
func writeObject(buf *strings.Builder, obj Object) {
	switch obj.Type {
	case ObjectNull:
		buf.WriteString("null")
	case ObjectBoolean:
		writeBoolean(buf, obj)
	case ObjectInteger:
		if v, ok := obj.Value.(int64); ok {
			fmt.Fprintf(buf, "%d", v)
		}
	case ObjectReal:
		if v, ok := obj.Value.(float64); ok {
			buf.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
		}
	case ObjectString:
		if s, ok := obj.Value.(string); ok {
			writeLiteralString(buf, s)
		}
	case ObjectHexString:
		if s, ok := obj.Value.(string); ok {
			writeHexString(buf, s)
		}
	case ObjectName:
		writeName(buf, obj)
	case ObjectArray:
		writeArray(buf, obj)
	case ObjectDictionary:
		writeDictionary(buf, obj)
	case ObjectReference:
		if ref, ok := obj.Value.(Ref); ok {
			fmt.Fprintf(buf, "%d %d R", ref.Number, ref.Generation)
		}
	case ObjectStream:

		buf.WriteString("stream(...)")
	}
}

// writeBoolean writes a PDF boolean value to buf.
//
// Takes buf (*strings.Builder) which is the output buffer.
// Takes obj (Object) which holds the boolean value.
func writeBoolean(buf *strings.Builder, obj Object) {
	if v, ok := obj.Value.(bool); ok && v {
		buf.WriteString("true")
	} else {
		buf.WriteString("false")
	}
}

// writeName writes a PDF name value to buf with a leading slash.
//
// Takes buf (*strings.Builder) which is the output buffer.
// Takes obj (Object) which holds the name string.
func writeName(buf *strings.Builder, obj Object) {
	if s, ok := obj.Value.(string); ok {
		buf.WriteByte('/')
		buf.WriteString(encodeName(s))
	}
}

// writeArray writes a PDF array value to buf in bracket notation.
//
// Takes buf (*strings.Builder) which is the output buffer.
// Takes obj (Object) which holds the array items.
func writeArray(buf *strings.Builder, obj Object) {
	items, ok := obj.Value.([]Object)
	if !ok {
		buf.WriteString("[]")
		return
	}
	buf.WriteByte('[')
	for i, item := range items {
		if i > 0 {
			buf.WriteByte(' ')
		}
		writeObject(buf, item)
	}
	buf.WriteByte(']')
}

// writeDictionary writes a PDF dictionary value to buf in double angle
// bracket notation.
//
// Takes buf (*strings.Builder) which is the output buffer.
// Takes obj (Object) which holds the Dict value.
func writeDictionary(buf *strings.Builder, obj Object) {
	dict, ok := obj.Value.(Dict)
	if !ok {
		buf.WriteString("<<>>")
		return
	}
	buf.WriteString("<<")
	for _, pair := range dict.Pairs {
		buf.WriteByte('/')
		buf.WriteString(encodeName(pair.Key))
		buf.WriteByte(' ')
		writeObject(buf, pair.Value)
		buf.WriteByte(' ')
	}
	buf.WriteString(">>")
}

// writeXRefTable writes a traditional cross-reference table to buf.
//
// Takes buf (*strings.Builder) which is the output buffer.
// Takes objNums ([]int) which lists the object numbers to include.
// Takes offsets (map[int]int64) which maps object numbers to byte offsets.
func writeXRefTable(buf *strings.Builder, objNums []int, offsets map[int]int64) {
	buf.WriteString("xref\n")

	buf.WriteString("0 1\n")
	buf.WriteString("0000000000 65535 f \n")

	if len(objNums) == 0 {
		return
	}

	i := 0
	for i < len(objNums) {
		start := objNums[i]
		end := start

		for i+1 < len(objNums) && objNums[i+1] == end+1 {
			i++
			end = objNums[i]
		}
		count := end - start + 1
		fmt.Fprintf(buf, "%d %d\n", start, count)
		for num := start; num <= end; num++ {
			fmt.Fprintf(buf, "%010d 00000 n \n", offsets[num])
		}
		i++
	}
}

// writeLiteralString writes a PDF literal string to buf with proper
// escaping of special characters.
//
// Takes buf (*strings.Builder) which is the output buffer.
// Takes s (string) which is the string content to write.
func writeLiteralString(buf *strings.Builder, s string) {
	buf.WriteByte('(')
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch ch {
		case '(', ')', '\\':
			buf.WriteByte('\\')
			buf.WriteByte(ch)
		case '\n':
			buf.WriteString("\\n")
		case '\r':
			buf.WriteString("\\r")
		case '\t':
			buf.WriteString("\\t")
		default:
			buf.WriteByte(ch)
		}
	}
	buf.WriteByte(')')
}

// writeHexString writes a PDF hex string to buf with each byte encoded as
// two uppercase hex digits.
//
// Takes buf (*strings.Builder) which is the output buffer.
// Takes s (string) which is the raw byte content to hex-encode.
func writeHexString(buf *strings.Builder, s string) {
	buf.WriteByte('<')
	for i := 0; i < len(s); i++ {
		fmt.Fprintf(buf, "%02X", s[i])
	}
	buf.WriteByte('>')
}

// encodeName encodes a PDF name string by escaping characters that require
// hex encoding in the PDF name syntax.
//
// Takes name (string) which is the raw name value to encode.
//
// Returns string which is the encoded name suitable for PDF output.
func encodeName(name string) string {
	var b strings.Builder
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch < '!' || ch > '~' || ch == '#' || ch == '/' || ch == '%' ||
			ch == '(' || ch == ')' || ch == '<' || ch == '>' ||
			ch == '[' || ch == ']' || ch == '{' || ch == '}' {
			fmt.Fprintf(&b, "#%02X", ch)
		} else {
			b.WriteByte(ch)
		}
	}
	return b.String()
}

// flateEncode compresses data using zlib (FlateDecode).
//
// Takes data ([]byte) which holds the uncompressed content.
//
// Returns []byte which is the compressed output.
// Returns error when compression fails.
func flateEncode(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := zlib.NewWriter(&buf)
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
