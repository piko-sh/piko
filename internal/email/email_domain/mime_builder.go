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

package email_domain

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"

	"piko.sh/piko/internal/email/email_dto"
)

const (
	// defaultFromAddress is the email address used when no sender is given.
	defaultFromAddress = "sender@example.com"

	// maxBase64LineLength is the maximum line length for base64-encoded content
	// per RFC 2045.
	maxBase64LineLength = 76

	// mimeContentType is the Content-Type MIME header name.
	mimeContentType = "Content-Type"

	// mimeTransferEncoding is the Content-Transfer-Encoding MIME header name.
	mimeTransferEncoding = "Content-Transfer-Encoding"

	// mimeTextHTML is the MIME type for HTML content.
	mimeTextHTML = "text/html"

	// mimeTextPlain is the MIME type for plain text content.
	mimeTextPlain = "text/plain"

	// mimeCRLF is the carriage-return line-feed sequence used in MIME messages.
	mimeCRLF = "\r\n"

	// mimeBoundaryOpen is the format string for opening a MIME boundary.
	mimeBoundaryOpen = "--%s\r\n"

	// mimeBoundaryClose is the format string for closing a MIME boundary.
	mimeBoundaryClose = "--%s--\r\n"

	// maxASCII is the highest code point in the ASCII character set.
	maxASCII = 127

	// controlCharUpperBound is the upper bound (exclusive) of the C0 control
	// character range.
	controlCharUpperBound = 32
)

// BuildMIMEMessage creates a complete MIME email message from the given
// parameters.
//
// It sets the From, To, Cc, Bcc, and Subject headers. When both HTML and plain
// text content are provided, it creates a multipart/alternative body. Regular
// file attachments are added normally. Attachments with a non-empty ContentID
// field are embedded as inline images using the Content-ID MIME header. This
// allows HTML email bodies to reference them via src="cid:xxx" attributes, so
// images display at once without needing external HTTP requests (which email
// clients often block for security).
//
// The MIME structure follows RFC 2046 nesting conventions:
//
//   - multipart/mixed wraps body + regular attachments
//   - multipart/related wraps HTML body + inline images (Content-ID)
//   - multipart/alternative wraps plain text + HTML variants
//
// The returned bytes follow RFC 5322 and are ready for sending via SMTP or
// saving as an .eml file.
//
// Takes params (*email_dto.SendParams) which specifies the email content and
// recipients.
//
// Returns []byte which contains the complete MIME message.
// Returns error when addresses are not valid or the message cannot be written.
func BuildMIMEMessage(params *email_dto.SendParams) ([]byte, error) {
	if err := validateAllAddresses(params); err != nil {
		return nil, fmt.Errorf("setting MIME message addresses: %w", err)
	}

	var buffer bytes.Buffer

	from := defaultFromAddress
	if params.From != nil {
		from = *params.From
	}

	writeHeader(&buffer, "MIME-Version", "1.0")
	writeHeader(&buffer, "Date", time.Now().Format(time.RFC1123Z))
	writeHeader(&buffer, "Message-ID", generateMessageID())
	writeHeader(&buffer, "From", from)
	writeHeader(&buffer, "To", strings.Join(params.To, ", "))

	if len(params.Cc) > 0 {
		writeHeader(&buffer, "Cc", strings.Join(params.Cc, ", "))
	}

	if len(params.Bcc) > 0 {
		writeHeader(&buffer, "Bcc", strings.Join(params.Bcc, ", "))
	}

	writeHeader(&buffer, "Subject", encodeSubject(params.Subject))

	regularAttachments, inlineAttachments := splitAttachments(params.Attachments)
	hasRegularAttachments := len(regularAttachments) > 0
	hasInlineAttachments := len(inlineAttachments) > 0
	hasBothBodies := params.BodyHTML != "" && params.BodyPlain != ""

	needsMixed := hasRegularAttachments || hasInlineAttachments

	if !needsMixed {
		writeBodyWithoutAttachments(&buffer, params, hasBothBodies)
	} else {
		writeBodyWithAttachments(&buffer, params, hasBothBodies, regularAttachments, inlineAttachments)
	}

	return buffer.Bytes(), nil
}

// splitAttachments separates attachments into regular (Content-Disposition:
// attachment) and inline (Content-ID) groups.
//
// Takes attachments ([]email_dto.Attachment) which contains all attachments.
//
// Returns regular ([]email_dto.Attachment) which are standard file attachments.
// Returns inline ([]email_dto.Attachment) which are CID-referenced inline
// images.
func splitAttachments(attachments []email_dto.Attachment) (regular []email_dto.Attachment, inline []email_dto.Attachment) {
	for _, attachment := range attachments {
		if attachment.ContentID != "" {
			inline = append(inline, attachment)
		} else {
			regular = append(regular, attachment)
		}
	}
	return regular, inline
}

// writeBodyWithoutAttachments writes the message body when there are no
// attachments.
//
// Takes buffer (*bytes.Buffer) which is the output buffer.
// Takes params (*email_dto.SendParams) which provides the body content.
// Takes hasBothBodies (bool) which indicates whether both HTML and plain text
// are present.
func writeBodyWithoutAttachments(buffer *bytes.Buffer, params *email_dto.SendParams, hasBothBodies bool) {
	if hasBothBodies {
		boundary := generateBoundary()
		writeHeader(buffer, mimeContentType, mime.FormatMediaType("multipart/alternative", map[string]string{"boundary": boundary}))
		_, _ = buffer.WriteString(mimeCRLF)
		writeTextPart(buffer, boundary, mimeTextPlain, params.BodyPlain)
		writeTextPart(buffer, boundary, mimeTextHTML, params.BodyHTML)
		_, _ = fmt.Fprintf(buffer, mimeBoundaryClose, boundary)
	} else if params.BodyHTML != "" {
		writeSingleBody(buffer, mimeTextHTML, params.BodyHTML)
	} else {
		writeSingleBody(buffer, mimeTextPlain, params.BodyPlain)
	}
}

// writeBodyWithAttachments writes the message body wrapped in the correct MIME
// multipart structure alongside attachment parts.
//
// The structure follows RFC 2046 conventions:
//
//   - multipart/mixed contains the body and regular attachments
//   - multipart/related wraps HTML and inline images when inline attachments
//     are present
//   - multipart/alternative wraps plain and HTML text when both are provided
//
// Takes buffer (*bytes.Buffer) which is the output buffer.
// Takes params (*email_dto.SendParams) which provides the body content.
// Takes hasBothBodies (bool) which indicates whether both HTML and plain text
// are present.
// Takes regularAttachments ([]email_dto.Attachment) which are standard file
// attachments.
// Takes inlineAttachments ([]email_dto.Attachment) which are CID-referenced
// inline images.
func writeBodyWithAttachments(
	buffer *bytes.Buffer,
	params *email_dto.SendParams,
	hasBothBodies bool,
	regularAttachments []email_dto.Attachment,
	inlineAttachments []email_dto.Attachment,
) {
	mixedBoundary := generateBoundary()
	writeHeader(buffer, mimeContentType, mime.FormatMediaType("multipart/mixed", map[string]string{"boundary": mixedBoundary}))
	_, _ = buffer.WriteString(mimeCRLF)

	_, _ = fmt.Fprintf(buffer, mimeBoundaryOpen, mixedBoundary)

	if len(inlineAttachments) > 0 {
		writeRelatedPart(buffer, params, hasBothBodies, inlineAttachments)
	} else if hasBothBodies {
		writeAlternativeBodyPart(buffer, params.BodyPlain, params.BodyHTML)
	} else if params.BodyHTML != "" {
		writeSingleBodyPart(buffer, mimeTextHTML, params.BodyHTML)
	} else {
		writeSingleBodyPart(buffer, mimeTextPlain, params.BodyPlain)
	}

	for _, attachment := range regularAttachments {
		_, _ = fmt.Fprintf(buffer, mimeBoundaryOpen, mixedBoundary)
		writeAttachmentPart(buffer, attachment)
	}

	_, _ = fmt.Fprintf(buffer, mimeBoundaryClose, mixedBoundary)
}

// writeRelatedPart writes a multipart/related section containing the HTML body
// and inline images. When both plain and HTML bodies are present, the HTML is
// nested inside a multipart/alternative.
//
// Takes buffer (*bytes.Buffer) which is the output buffer.
// Takes params (*email_dto.SendParams) which provides the body content.
// Takes hasBothBodies (bool) which indicates whether both body types exist.
// Takes inlineAttachments ([]email_dto.Attachment) which are CID-referenced
// inline images.
func writeRelatedPart(
	buffer *bytes.Buffer,
	params *email_dto.SendParams,
	hasBothBodies bool,
	inlineAttachments []email_dto.Attachment,
) {
	relatedBoundary := generateBoundary()
	writeHeader(buffer, mimeContentType, mime.FormatMediaType("multipart/related", map[string]string{"boundary": relatedBoundary}))
	_, _ = buffer.WriteString(mimeCRLF)

	_, _ = fmt.Fprintf(buffer, mimeBoundaryOpen, relatedBoundary)
	if hasBothBodies {
		writeAlternativeBodyPart(buffer, params.BodyPlain, params.BodyHTML)
	} else if params.BodyHTML != "" {
		writeSingleBodyPart(buffer, mimeTextHTML, params.BodyHTML)
	} else {
		writeSingleBodyPart(buffer, mimeTextPlain, params.BodyPlain)
	}

	for _, attachment := range inlineAttachments {
		_, _ = fmt.Fprintf(buffer, mimeBoundaryOpen, relatedBoundary)
		writeInlinePart(buffer, attachment)
	}

	_, _ = fmt.Fprintf(buffer, mimeBoundaryClose, relatedBoundary)
}

// writeAlternativeBodyPart writes a multipart/alternative section containing
// both plain text and HTML body parts.
//
// Takes buffer (*bytes.Buffer) which is the output buffer.
// Takes plain (string) which is the plain text body.
// Takes html (string) which is the HTML body.
func writeAlternativeBodyPart(buffer *bytes.Buffer, plain string, html string) {
	alternativeBoundary := generateBoundary()
	writeHeader(buffer, mimeContentType, mime.FormatMediaType("multipart/alternative", map[string]string{"boundary": alternativeBoundary}))
	_, _ = buffer.WriteString(mimeCRLF)
	writeTextPart(buffer, alternativeBoundary, mimeTextPlain, plain)
	writeTextPart(buffer, alternativeBoundary, mimeTextHTML, html)
	_, _ = fmt.Fprintf(buffer, mimeBoundaryClose, alternativeBoundary)
}

// validateAddress checks whether an email address is syntactically valid
// according to RFC 5322.
//
// Takes address (string) which is the email address to validate.
//
// Returns error when the address is not valid.
func validateAddress(address string) error {
	_, err := mail.ParseAddress(address)
	return err
}

// validateAllAddresses validates every address in the given parameters.
//
// Takes params (*email_dto.SendParams) which holds all address fields.
//
// Returns error when any address is not valid.
func validateAllAddresses(params *email_dto.SendParams) error {
	from := defaultFromAddress
	if params.From != nil {
		from = *params.From
	}

	if err := validateAddress(from); err != nil {
		return fmt.Errorf("invalid From address %q: %w", from, err)
	}

	for _, address := range params.To {
		if err := validateAddress(address); err != nil {
			return fmt.Errorf("invalid To address %q: %w", address, err)
		}
	}

	for _, address := range params.Cc {
		if err := validateAddress(address); err != nil {
			return fmt.Errorf("invalid Cc address %q: %w", address, err)
		}
	}

	for _, address := range params.Bcc {
		if err := validateAddress(address); err != nil {
			return fmt.Errorf("invalid Bcc address %q: %w", address, err)
		}
	}

	return nil
}

// generateBoundary produces a random MIME boundary string.
//
// Returns string which is a 32-character hexadecimal boundary.
func generateBoundary() string {
	var randomBytes [16]byte
	_, _ = rand.Read(randomBytes[:])

	return fmt.Sprintf("%x", randomBytes)
}

// generateMessageID produces a unique Message-ID header value.
//
// Returns string in the format "<hex@piko.localhost>".
func generateMessageID() string {
	var randomBytes [16]byte
	_, _ = rand.Read(randomBytes[:])

	return fmt.Sprintf("<%x@piko.localhost>", randomBytes)
}

// encodeSubject encodes a subject string using RFC 2047 Q-encoding when it
// contains non-ASCII characters. Pure ASCII subjects are returned unchanged.
//
// Takes subject (string) which is the raw subject text.
//
// Returns string which is the encoded subject ready for a header.
func encodeSubject(subject string) string {
	for _, r := range subject {
		if r > maxASCII {
			return mime.QEncoding.Encode("utf-8", subject)
		}
	}
	return subject
}

// sanitiseFilename removes control characters and characters that could cause
// MIME header injection or filesystem issues from a filename.
//
// Takes filename (string) which is the original filename.
//
// Returns string which is the sanitised filename safe for MIME headers.
func sanitiseFilename(filename string) string {
	var builder strings.Builder
	builder.Grow(len(filename))
	for _, r := range filename {
		switch {
		case r < controlCharUpperBound, r == maxASCII:
		case r == '"', r == '/', r == ':', r == '<', r == '>', r == '?', r == '\\', r == '|':
		default:
			_, _ = builder.WriteRune(r)
		}
	}
	return builder.String()
}

// writeHeader writes a single RFC 5322 header line to the buffer.
//
// Takes buffer (*bytes.Buffer) which is the output buffer.
// Takes key (string) which is the header name.
// Takes value (string) which is the header value.
func writeHeader(buffer *bytes.Buffer, key string, value string) {
	_, _ = fmt.Fprintf(buffer, "%s: %s\r\n", key, value)
}

// writeSingleBody writes a single-part body directly after the message
// headers, including the Content-Type and Content-Transfer-Encoding headers.
//
// Takes buffer (*bytes.Buffer) which is the output buffer.
// Takes contentType (string) which is the MIME content type.
// Takes body (string) which is the text content.
func writeSingleBody(buffer *bytes.Buffer, contentType string, body string) {
	writeHeader(buffer, mimeContentType, mime.FormatMediaType(contentType, map[string]string{"charset": "utf-8"}))
	writeHeader(buffer, mimeTransferEncoding, "quoted-printable")
	_, _ = buffer.WriteString(mimeCRLF)
	writeQuotedPrintable(buffer, body)
}

// writeSingleBodyPart writes a single-part body as a MIME part (used inside a
// multipart boundary, without the boundary delimiter itself).
//
// Takes buffer (*bytes.Buffer) which is the output buffer.
// Takes contentType (string) which is the MIME content type.
// Takes body (string) which is the text content.
func writeSingleBodyPart(buffer *bytes.Buffer, contentType string, body string) {
	writeHeader(buffer, mimeContentType, mime.FormatMediaType(contentType, map[string]string{"charset": "utf-8"}))
	writeHeader(buffer, mimeTransferEncoding, "quoted-printable")
	_, _ = buffer.WriteString(mimeCRLF)
	writeQuotedPrintable(buffer, body)
}

// writeTextPart writes a text MIME part inside a multipart boundary.
//
// Takes buffer (*bytes.Buffer) which is the output buffer.
// Takes boundary (string) which is the multipart boundary delimiter.
// Takes contentType (string) which is the MIME content type.
// Takes body (string) which is the text content.
func writeTextPart(buffer *bytes.Buffer, boundary string, contentType string, body string) {
	_, _ = fmt.Fprintf(buffer, mimeBoundaryOpen, boundary)
	writeHeader(buffer, mimeContentType, mime.FormatMediaType(contentType, map[string]string{"charset": "utf-8"}))
	writeHeader(buffer, mimeTransferEncoding, "quoted-printable")
	_, _ = buffer.WriteString(mimeCRLF)
	writeQuotedPrintable(buffer, body)
}

// writeQuotedPrintable encodes the given text using quoted-printable encoding
// and writes it to the buffer.
//
// Takes buffer (*bytes.Buffer) which is the output buffer.
// Takes text (string) which is the content to encode.
func writeQuotedPrintable(buffer *bytes.Buffer, text string) {
	writer := quotedprintable.NewWriter(buffer)
	_, _ = writer.Write([]byte(text))
	_ = writer.Close()

	_, _ = buffer.WriteString(mimeCRLF)
}

// writeAttachmentPart writes a regular file attachment as a MIME part with
// Content-Disposition: attachment.
//
// Takes buffer (*bytes.Buffer) which is the output buffer.
// Takes attachment (email_dto.Attachment) which is the file to encode.
func writeAttachmentPart(buffer *bytes.Buffer, attachment email_dto.Attachment) {
	mimeType := attachment.MIMEType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	safeFilename := sanitiseFilename(attachment.Filename)

	writeHeader(buffer, mimeContentType, mime.FormatMediaType(mimeType, map[string]string{"name": safeFilename}))
	writeHeader(buffer, mimeTransferEncoding, "base64")
	writeHeader(buffer, "Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": safeFilename}))

	_, _ = buffer.WriteString(mimeCRLF)
	writeBase64Content(buffer, attachment.Content)
}

// writeInlinePart writes an inline image attachment as a MIME part with
// Content-Disposition: inline and a Content-ID header for CID references.
//
// Takes buffer (*bytes.Buffer) which is the output buffer.
// Takes attachment (email_dto.Attachment) which is the inline image to encode.
func writeInlinePart(buffer *bytes.Buffer, attachment email_dto.Attachment) {
	mimeType := attachment.MIMEType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	safeFilename := sanitiseFilename(attachment.Filename)

	writeHeader(buffer, mimeContentType, mime.FormatMediaType(mimeType, map[string]string{"name": safeFilename}))
	writeHeader(buffer, mimeTransferEncoding, "base64")
	writeHeader(buffer, "Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": safeFilename}))
	writeHeader(buffer, "Content-ID", fmt.Sprintf("<%s>", attachment.ContentID))

	_, _ = buffer.WriteString(mimeCRLF)
	writeBase64Content(buffer, attachment.Content)
}

// writeBase64Content encodes binary data as base64 with 76-character line
// wrapping per RFC 2045 and writes it to the buffer.
//
// Takes buffer (*bytes.Buffer) which is the output buffer.
// Takes data ([]byte) which is the binary content to encode.
func writeBase64Content(buffer *bytes.Buffer, data []byte) {
	encoded := base64.StdEncoding.EncodeToString(data)

	for len(encoded) > maxBase64LineLength {
		_, _ = buffer.WriteString(encoded[:maxBase64LineLength])
		_, _ = buffer.WriteString(mimeCRLF)
		encoded = encoded[maxBase64LineLength:]
	}

	if len(encoded) > 0 {
		_, _ = buffer.WriteString(encoded)
		_, _ = buffer.WriteString(mimeCRLF)
	}
}
