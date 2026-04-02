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

package pdfwriter_domain

// Minimal PDF file writer. Produces a valid PDF 1.7 document by writing
// the header, indirect objects with tracked byte offsets, a cross-reference
// table, and the trailer. No external dependencies.

import (
	"bytes"
	"compress/zlib"
	"fmt"
)

// pdfObject holds the byte offset of a single PDF indirect object.
type pdfObject struct {
	// byteOffset holds the position in the output buffer where this object begins.
	byteOffset int
}

// PdfDocumentWriter builds a complete PDF file in memory.
type PdfDocumentWriter struct {
	// objects holds the byte offset metadata for each allocated object.
	objects []pdfObject

	// buffer holds the accumulated PDF file bytes.
	buffer bytes.Buffer
}

// AllocateObject reserves an object number and returns it. Object
// numbers start at 1 (object 0 is the free-list head in the xref).
//
// Returns int which is the newly allocated object number.
func (writer *PdfDocumentWriter) AllocateObject() int {
	writer.objects = append(writer.objects, pdfObject{})
	return len(writer.objects)
}

// WriteHeader writes the PDF file header including the binary
// comment that marks this as a binary PDF.
func (writer *PdfDocumentWriter) WriteHeader() {
	_, _ = writer.buffer.WriteString("%PDF-1.7\n")
	_, _ = writer.buffer.Write([]byte{'%', 0xe2, 0xe3, 0xcf, 0xd3, '\n'})
}

// WriteObject writes an indirect object with the given number and
// body content. The byte offset is recorded for the xref table.
//
// Takes number (int) which is the object number from AllocateObject.
// Takes body (string) which is the PDF dictionary or value content.
func (writer *PdfDocumentWriter) WriteObject(number int, body string) {
	if number < 1 || number > len(writer.objects) {
		return
	}
	writer.objects[number-1].byteOffset = writer.buffer.Len()
	fmt.Fprintf(&writer.buffer, "%d 0 obj\n%s\nendobj\n", number, body)
}

// WriteStreamObject writes an indirect object whose body is a
// dictionary followed by a zlib-compressed stream.
//
// The /Length key is set automatically. If compression fails, the
// content is written uncompressed as a fallback.
//
// Takes number (int) which is the object number from AllocateObject.
// Takes dictionary (string) which holds additional dictionary entries
// beyond /Filter and /Length.
// Takes streamContent ([]byte) which is the raw bytes to compress and
// write.
func (writer *PdfDocumentWriter) WriteStreamObject(number int, dictionary string, streamContent []byte) {
	if number < 1 || number > len(writer.objects) {
		return
	}
	writer.objects[number-1].byteOffset = writer.buffer.Len()

	var compressed bytes.Buffer
	zlibWriter := zlib.NewWriter(&compressed)
	zlibError := writeAndCloseZlib(zlibWriter, streamContent)

	fmt.Fprintf(&writer.buffer, "%d 0 obj\n", number)
	if zlibError != nil {
		fmt.Fprintf(&writer.buffer, "<< %s /Length %d >>\n", dictionary, len(streamContent))
		_, _ = writer.buffer.WriteString("stream\n")
		_, _ = writer.buffer.Write(streamContent)
	} else {
		compressedBytes := compressed.Bytes()
		fmt.Fprintf(&writer.buffer, "<< %s /Filter /FlateDecode /Length %d >>\n", dictionary, len(compressedBytes))
		_, _ = writer.buffer.WriteString("stream\n")
		_, _ = writer.buffer.Write(compressedBytes)
	}
	_, _ = writer.buffer.WriteString("\nendstream\nendobj\n")
}

// WriteRawStreamObject writes an indirect object whose body is a
// pre-formatted dictionary followed by a raw, uncompressed stream.
//
// Unlike WriteStreamObject, the content is not compressed; the caller
// is responsible for any encoding (e.g. DCTDecode for JPEG, or
// pre-compressed FlateDecode data).
//
// Takes number (int) which is the object number from AllocateObject.
// Takes dictionary (string) which is the complete dictionary string
// including angle brackets.
// Takes streamContent ([]byte) which is the raw bytes to write.
func (writer *PdfDocumentWriter) WriteRawStreamObject(number int, dictionary string, streamContent []byte) {
	if number < 1 || number > len(writer.objects) {
		return
	}
	writer.objects[number-1].byteOffset = writer.buffer.Len()

	fmt.Fprintf(&writer.buffer, "%d 0 obj\n", number)
	fmt.Fprintf(&writer.buffer, "%s\n", dictionary)
	_, _ = writer.buffer.WriteString("stream\n")
	_, _ = writer.buffer.Write(streamContent)
	_, _ = writer.buffer.WriteString("\nendstream\nendobj\n")
}

// WriteTrailer writes the cross-reference table, trailer dictionary,
// and the startxref footer.
//
// Takes catalogueNumber (int) which is the object number of the
// document catalogue.
// Takes infoNumber (...int) which is the optional object number of
// the document info dictionary; pass 0 to omit it.
func (writer *PdfDocumentWriter) WriteTrailer(catalogueNumber int, infoNumber ...int) {
	crossReferenceOffset := writer.buffer.Len()

	objectCount := len(writer.objects) + 1
	fmt.Fprintf(&writer.buffer, "xref\n0 %d\n", objectCount)

	_, _ = writer.buffer.WriteString("0000000000 65535 f \n")
	for _, object := range writer.objects {
		fmt.Fprintf(&writer.buffer, "%010d 00000 n \n", object.byteOffset)
	}

	_, _ = writer.buffer.WriteString("trailer\n")
	trailer := fmt.Sprintf("<< /Size %d /Root %d 0 R", objectCount, catalogueNumber)
	if len(infoNumber) > 0 && infoNumber[0] > 0 {
		trailer += fmt.Sprintf(" /Info %d 0 R", infoNumber[0])
	}
	trailer += " >>\n"
	_, _ = writer.buffer.WriteString(trailer)
	fmt.Fprintf(&writer.buffer, "startxref\n%d\n%%%%EOF\n", crossReferenceOffset)
}

// Bytes returns the complete PDF file content.
//
// Returns []byte which holds the full PDF file bytes.
func (writer *PdfDocumentWriter) Bytes() []byte {
	return writer.buffer.Bytes()
}

// FormatReference formats an indirect object reference (e.g. "3 0 R").
//
// Takes objectNumber (int) which is the PDF object number.
//
// Returns string which is the formatted reference like "3 0 R".
func FormatReference(objectNumber int) string {
	return fmt.Sprintf("%d 0 R", objectNumber)
}

// FormatArray formats a PDF array (e.g. "[1 2 3]").
//
// Takes items (...string) which are the elements to include in the array.
//
// Returns string which is the formatted array like "[1 2 3]".
func FormatArray(items ...string) string {
	var builder bytes.Buffer
	_ = builder.WriteByte('[')
	for index, item := range items {
		if index > 0 {
			_ = builder.WriteByte(' ')
		}
		_, _ = builder.WriteString(item)
	}
	_ = builder.WriteByte(']')
	return builder.String()
}

// FormatNumber formats a float64 as a PDF number, rendering as an
// integer if whole or with two decimal places otherwise.
//
// Takes value (float64) which is the number to format.
//
// Returns string which is the formatted PDF number.
func FormatNumber(value float64) string {
	return formatFloat(value)
}
