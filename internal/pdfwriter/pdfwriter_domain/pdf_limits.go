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
	"errors"
	"fmt"
)

const (
	// defaultMaxEmbeddedFiles is the default ceiling on the number of associated files a
	// single render may embed.
	defaultMaxEmbeddedFiles = 256

	// defaultMaxEmbeddedFileBytes is the default ceiling on a single embedded payload,
	// chosen high enough to stay out of the way of legitimate documents.
	defaultMaxEmbeddedFileBytes = 64 << 20

	// defaultMaxEmbeddedTotalBytes is the default ceiling on the aggregate size of all
	// embedded payloads. The per-file and count limits bound each dimension independently;
	// without this their product (worst case roughly 16 GiB) could be held in memory.
	defaultMaxEmbeddedTotalBytes = 256 << 20

	// defaultMaxStructuredMetadataBytes is the default ceiling on the structured metadata
	// (schema.org JSON-LD) embedded in the XMP packet.
	defaultMaxStructuredMetadataBytes = 4 << 20

	// defaultMaxEmbeddedNameBytes is the default ceiling on an embedded file name length.
	defaultMaxEmbeddedNameBytes = 4096

	// defaultMaxEmbeddedDescriptionBytes is the default ceiling on an embedded file
	// description length.
	defaultMaxEmbeddedDescriptionBytes = 8192
)

var (
	// ErrTooManyEmbeddedFiles is returned when a render embeds more associated files than
	// the configured limit allows.
	ErrTooManyEmbeddedFiles = errors.New("pdfwriter: too many embedded files")

	// ErrEmbeddedFileTooLarge is returned when an embedded payload exceeds the configured
	// per-file size limit.
	ErrEmbeddedFileTooLarge = errors.New("pdfwriter: embedded file exceeds maximum size")

	// ErrEmbeddedTotalTooLarge is returned when the combined size of all embedded payloads
	// exceeds the configured aggregate limit.
	ErrEmbeddedTotalTooLarge = errors.New("pdfwriter: embedded files exceed maximum total size")

	// ErrStructuredMetadataTooLarge is returned when the structured metadata exceeds the
	// configured size limit.
	ErrStructuredMetadataTooLarge = errors.New("pdfwriter: structured metadata exceeds maximum size")

	// ErrEmbeddedFileMetadataTooLong is returned when an embedded file name or description
	// exceeds the configured length limit.
	ErrEmbeddedFileMetadataTooLong = errors.New("pdfwriter: embedded file name or description too long")

	// ErrEmbeddedMIMETypeTooLong is returned when an embedded file MIME type exceeds the
	// configured length limit. The MIME type is written into a PDF /Subtype name, so an
	// unbounded value would bloat the document.
	ErrEmbeddedMIMETypeTooLong = errors.New("pdfwriter: embedded file MIME type too long")
)

// EmbeddedDataLimits bounds the machine-readable data a render may embed, guarding
// against unbounded memory use. A zero field falls back to the high built-in default, so
// callers set only the limits they wish to tighten.
type EmbeddedDataLimits struct {
	// MaxFiles is the maximum number of embedded associated files. Zero uses the default.
	MaxFiles int

	// MaxFileBytes is the maximum size of a single embedded payload. Zero uses the default.
	MaxFileBytes int

	// MaxTotalBytes is the maximum combined size of all embedded payloads. Zero uses the
	// default.
	MaxTotalBytes int

	// MaxStructuredMetadataBytes is the maximum size of the structured metadata embedded in
	// XMP. Zero uses the default.
	MaxStructuredMetadataBytes int

	// MaxNameBytes is the maximum length of an embedded file name. Zero uses the default.
	MaxNameBytes int

	// MaxDescriptionBytes is the maximum length of an embedded file description. Zero uses
	// the default.
	MaxDescriptionBytes int
}

// resolveEmbeddedDataLimits returns the effective limits, substituting the built-in
// defaults for any unset (non-positive) field.
//
// Takes configured (*EmbeddedDataLimits) which holds the caller overrides, or nil.
//
// Returns EmbeddedDataLimits which holds the effective limits.
func resolveEmbeddedDataLimits(configured *EmbeddedDataLimits) EmbeddedDataLimits {
	limits := EmbeddedDataLimits{
		MaxFiles:                   defaultMaxEmbeddedFiles,
		MaxFileBytes:               defaultMaxEmbeddedFileBytes,
		MaxTotalBytes:              defaultMaxEmbeddedTotalBytes,
		MaxStructuredMetadataBytes: defaultMaxStructuredMetadataBytes,
		MaxNameBytes:               defaultMaxEmbeddedNameBytes,
		MaxDescriptionBytes:        defaultMaxEmbeddedDescriptionBytes,
	}
	if configured == nil {
		return limits
	}
	if configured.MaxFiles > 0 {
		limits.MaxFiles = configured.MaxFiles
	}
	if configured.MaxFileBytes > 0 {
		limits.MaxFileBytes = configured.MaxFileBytes
	}
	if configured.MaxTotalBytes > 0 {
		limits.MaxTotalBytes = configured.MaxTotalBytes
	}
	if configured.MaxStructuredMetadataBytes > 0 {
		limits.MaxStructuredMetadataBytes = configured.MaxStructuredMetadataBytes
	}
	if configured.MaxNameBytes > 0 {
		limits.MaxNameBytes = configured.MaxNameBytes
	}
	if configured.MaxDescriptionBytes > 0 {
		limits.MaxDescriptionBytes = configured.MaxDescriptionBytes
	}
	return limits
}

// validateEmbeddedData checks the embedded files and structured metadata against the
// limits, returning a wrapped sentinel when a payload is too large or too numerous. It
// detects the breach rather than silently truncating, which would corrupt a structured
// payload.
//
// Takes files ([]EmbeddedFile) which are the payloads to embed.
// Takes structured (*StructuredMetadata) which holds the structured metadata, or nil.
// Takes configured (*EmbeddedDataLimits) which holds the caller overrides, or nil.
//
// Returns error which wraps a sentinel when a limit is exceeded.
func validateEmbeddedData(files []EmbeddedFile, structured *StructuredMetadata, configured *EmbeddedDataLimits) error {
	limits := resolveEmbeddedDataLimits(configured)
	if len(files) > limits.MaxFiles {
		return fmt.Errorf("pdfwriter: %d embedded files exceeds limit %d: %w",
			len(files), limits.MaxFiles, ErrTooManyEmbeddedFiles)
	}
	var totalBytes int64
	for _, file := range files {
		if len(file.Data) > limits.MaxFileBytes {
			return fmt.Errorf("pdfwriter: embedded file %q is %d bytes, exceeds limit %d: %w",
				file.Name, len(file.Data), limits.MaxFileBytes, ErrEmbeddedFileTooLarge)
		}

		totalBytes += int64(len(file.Data))
		if totalBytes > int64(limits.MaxTotalBytes) {
			return fmt.Errorf("pdfwriter: embedded files total %d bytes, exceeds limit %d: %w",
				totalBytes, limits.MaxTotalBytes, ErrEmbeddedTotalTooLarge)
		}
		if len(file.Name) > limits.MaxNameBytes {
			return fmt.Errorf("pdfwriter: embedded file name is %d bytes, exceeds limit %d: %w",
				len(file.Name), limits.MaxNameBytes, ErrEmbeddedFileMetadataTooLong)
		}

		if len(file.MIMEType) > limits.MaxNameBytes {
			return fmt.Errorf("pdfwriter: embedded file %q MIME type is %d bytes, exceeds limit %d: %w",
				file.Name, len(file.MIMEType), limits.MaxNameBytes, ErrEmbeddedMIMETypeTooLong)
		}
		if len(file.Description) > limits.MaxDescriptionBytes {
			return fmt.Errorf("pdfwriter: embedded file %q description is %d bytes, exceeds limit %d: %w",
				file.Name, len(file.Description), limits.MaxDescriptionBytes, ErrEmbeddedFileMetadataTooLong)
		}
	}
	if structured != nil && len(structured.SchemaOrgJSONLD) > limits.MaxStructuredMetadataBytes {
		return fmt.Errorf("pdfwriter: structured metadata is %d bytes, exceeds limit %d: %w",
			len(structured.SchemaOrgJSONLD), limits.MaxStructuredMetadataBytes, ErrStructuredMetadataTooLarge)
	}
	return nil
}
