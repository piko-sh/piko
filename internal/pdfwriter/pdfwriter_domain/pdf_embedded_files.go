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

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
)

// AFRelationship describes how an embedded associated file relates to the visible
// document, per ISO 32000-2. It is written to the /AFRelationship entry of a file
// specification.
type AFRelationship string

const (
	// AFSource marks the embedded file as the authoritative source the page was rendered
	// from (for example the JSON or XML a document was generated from).
	AFSource AFRelationship = "Source"

	// AFData marks the embedded file as data referenced by the document.
	AFData AFRelationship = "Data"

	// AFAlternative marks the embedded file as an alternative representation of the document
	// content (for example a machine-readable equivalent of the visible page).
	AFAlternative AFRelationship = "Alternative"

	// AFSupplement marks the embedded file as supplementary material.
	AFSupplement AFRelationship = "Supplement"

	// AFUnspecified leaves the relationship unspecified.
	AFUnspecified AFRelationship = "Unspecified"
)

const (
	// pdfNameLowByte is the highest byte value escaped as #xx in a PDF name (space and
	// below).
	pdfNameLowByte = 0x20

	// pdfNameHighByte is the lowest high byte value escaped as #xx in a PDF name (delete and
	// above), which also escapes every non-ASCII UTF-8 continuation byte.
	pdfNameHighByte = 0x7f

	// defaultEmbeddedMIMEType is substituted for an empty MIMEType so the embedded stream
	// /Subtype is a valid, non-empty PDF name rather than the bare "/".
	defaultEmbeddedMIMEType = "application/octet-stream"
)

// EmbeddedFile describes a machine-readable payload embedded in the PDF as an associated
// file. The payload rides alongside the visible page so that parsers (recruiter systems,
// invoice processors, LLM ingestion) can read exact structured data rather than
// reconstruct it from glyphs.
type EmbeddedFile struct {
	// Name holds the embedded file name, for example "resume.json" or "factur-x.xml".
	Name string

	// MIMEType holds the media type written to the embedded stream /Subtype, for example
	// "application/json" or "application/ld+json".
	MIMEType string

	// Description holds an optional human-readable description (/Desc).
	Description string

	// Relationship holds the relationship to the document. Empty defaults to AFSource.
	Relationship AFRelationship

	// Data holds the raw payload bytes.
	Data []byte
}

// writeEmbeddedFiles emits the embedded-file stream and file-specification objects, the
// EmbeddedFiles name tree, and the document-level /AF associated-files array, returning
// the catalog-level entries to append. It returns an empty string when no files are
// embedded.
//
// Takes writer (*PdfDocumentWriter) which receives the objects.
// Takes created (time.Time) which is the document timestamp used for embedded file dates.
//
// Returns string which holds the catalog entries (/AF and /Names /EmbeddedFiles).
// Returns error when the context is cancelled while writing the payloads.
func (painter *PdfPainter) writeEmbeddedFiles(ctx context.Context, writer *PdfDocumentWriter, created time.Time) (string, error) {
	if len(painter.embeddedFiles) == 0 {
		return "", nil
	}

	files := append([]EmbeddedFile(nil), painter.embeddedFiles...)

	type encodedFile struct {
		encodedKey string
		file       EmbeddedFile
	}
	encoded := make([]encodedFile, 0, len(files))
	for _, file := range files {
		encoded = append(encoded, encodedFile{file: file, encodedKey: encodePdfTextString(file.Name)})
	}
	slices.SortFunc(encoded, func(a, b encodedFile) int {
		return cmp.Compare(a.encodedKey, b.encodedKey)
	})

	afRefs := make([]string, 0, len(encoded))
	pairs := make([]nameTreePair, 0, len(encoded))
	dateString := formatPdfDate(created)

	for _, item := range encoded {
		if cancelErr := ctx.Err(); cancelErr != nil {
			return "", fmt.Errorf("pdfwriter: embedded file write cancelled: %w", cancelErr)
		}
		file := item.file
		mimeType := file.MIMEType
		if mimeType == "" {
			mimeType = defaultEmbeddedMIMEType
		}
		streamNumber := writer.AllocateObject()
		streamDict := fmt.Sprintf("/Type /EmbeddedFile /Subtype /%s /Params << /Size %d /ModDate %s >>",
			escapePdfName(mimeType), len(file.Data), dateString)
		writer.WriteStreamObject(streamNumber, streamDict, file.Data)

		relationship := normaliseAFRelationship(file.Relationship)
		specNumber := writer.AllocateObject()
		writer.WriteObject(specNumber, fmt.Sprintf(
			"<< /Type /Filespec /F %s /UF %s%s /EF << /F %s >> /AFRelationship /%s >>",
			encodePdfTextString(file.Name),
			encodeUTF16BEHexString(file.Name),
			embeddedFileDescription(file.Description),
			FormatReference(streamNumber),
			relationship))

		afRefs = append(afRefs, FormatReference(specNumber))
		pairs = append(pairs, nameTreePair{
			key:   item.encodedKey,
			value: FormatReference(specNumber),
		})
	}

	nameTreeNumber := buildNameTreeDict(writer, pairs)
	return fmt.Sprintf(" /AF [%s] /Names << /EmbeddedFiles %s >>",
		strings.Join(afRefs, " "), FormatReference(nameTreeNumber)), nil
}

// embeddedFileDescription returns the optional /Desc entry for a file specification.
//
// Takes description (string) which is the human-readable description, or empty to omit.
//
// Returns string which is the " /Desc <token>" entry, or empty.
func embeddedFileDescription(description string) string {
	if description == "" {
		return ""
	}
	return " /Desc " + encodePdfTextString(description)
}

// normaliseAFRelationship returns the relationship when it is a known value, otherwise
// AFSource. This prevents a caller-supplied value from injecting arbitrary tokens into
// the /AFRelationship name.
//
// Takes relationship (AFRelationship) which is the requested relationship.
//
// Returns AFRelationship which is a known-good value.
func normaliseAFRelationship(relationship AFRelationship) AFRelationship {
	switch relationship {
	case AFSource, AFData, AFAlternative, AFSupplement, AFUnspecified:
		return relationship
	default:
		return AFSource
	}
}

// escapePdfName encodes a string as the body of a PDF name token, escaping the number
// sign and every byte that is whitespace, a delimiter, or outside printable ASCII as #xx.
// This keeps a caller-supplied value such as a MIME type from breaking out of the name
// token.
//
// Takes value (string) which is the raw name body, for example "application/json".
//
// Returns string which is the escaped PDF name body.
func escapePdfName(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for index := range len(value) {
		character := value[index]
		if character == '#' || character <= pdfNameLowByte || character >= pdfNameHighByte || isPdfNameDelimiter(character) {
			fmt.Fprintf(&builder, "#%02X", character)
			continue
		}
		builder.WriteByte(character)
	}
	return builder.String()
}

// isPdfNameDelimiter reports whether a byte is a PDF delimiter character that terminates
// a name token.
//
// Takes character (byte) which is the byte to test.
//
// Returns bool which is true for a delimiter.
func isPdfNameDelimiter(character byte) bool {
	switch character {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%':
		return true
	default:
		return false
	}
}
