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

package pdfwriter_dto

import "crypto"

// CompressAlgorithm identifies a compression algorithm for the compress
// transformer.
type CompressAlgorithm string

const (
	// CompressZstd selects Zstandard compression.
	CompressZstd CompressAlgorithm = "zstd"
)

// TransformConfig specifies which PDF transformers to apply and their
// per-transformer options. Transformers are applied in ascending priority
// order regardless of the order in EnabledTransformers.
type TransformConfig struct {
	// TransformerOptions maps transformer names to their specific settings.
	// Each transformer casts its options to the expected concrete type.
	TransformerOptions map[string]any

	// EnabledTransformers lists the names of transformers to apply. The
	// chain sorts them by priority; this list is a declaration of intent,
	// not an ordering directive.
	EnabledTransformers []string
}

// DefaultTransformConfig returns an empty transform configuration with no
// transformers enabled.
//
// Returns TransformConfig which is ready for use with empty transformer lists.
func DefaultTransformConfig() TransformConfig {
	return TransformConfig{
		TransformerOptions:  make(map[string]any),
		EnabledTransformers: nil,
	}
}

// WatermarkOptions configures the watermark transformer.
type WatermarkOptions struct {
	// Text is the watermark text to render. Empty when using image-only.
	Text string

	// ImageData holds raw image bytes for an image watermark. Nil for
	// text-only watermarks.
	ImageData []byte

	// Pages restricts the watermark to specific page indices. Empty means
	// all pages.
	Pages []int

	// FontSize is the font size in points for text watermarks.
	FontSize float64

	// ColourR is the red channel of the watermark colour in [0, 1].
	ColourR float64

	// ColourG is the green channel of the watermark colour in [0, 1].
	ColourG float64

	// ColourB is the blue channel of the watermark colour in [0, 1].
	ColourB float64

	// Angle is the rotation angle in degrees for the watermark.
	Angle float64

	// Opacity is the watermark opacity in [0, 1].
	Opacity float64
}

// EncryptionOptions configures the PDF encryption transformer.
type EncryptionOptions struct {
	// Algorithm specifies the encryption algorithm: "aes-256", "aes-128",
	// or "rc4-128".
	Algorithm string

	// OwnerPassword is the owner (permissions) password.
	OwnerPassword string

	// UserPassword is the user (open) password.
	UserPassword string

	// Permissions is the PDF permission flags bitmask (PDF spec table 22).
	Permissions uint32
}

// PadesSignOptions configures the PAdES digital signature transformer.
type PadesSignOptions struct {
	// PrivateKey is the signing key. Must implement crypto.Signer.
	PrivateKey crypto.Signer

	// Level specifies the PAdES conformance level: "b-b", "b-t", "b-lt",
	// or "b-lta".
	Level string

	// TimestampURL is the Time Stamping Authority URL, required for B-T
	// and above.
	TimestampURL string

	// Reason is the stated reason for signing.
	Reason string

	// Location is the stated location of signing.
	Location string

	// ContactInfo is contact information for the signer.
	ContactInfo string

	// CertificateChain is the signing certificate chain in DER encoding,
	// ordered from end-entity to root.
	CertificateChain [][]byte

	// OCSPResponses holds pre-fetched OCSP responses for long-term
	// validation (B-LT and above).
	OCSPResponses [][]byte

	// CRLs holds pre-fetched CRLs for long-term validation (B-LT and
	// above).
	CRLs [][]byte
}

// PdfUAOptions configures the PDF/UA enhancement transformer. PDF/UA
// enhancement does not currently require any configuration.
type PdfUAOptions struct{}

// PdfAOptions configures the PDF/A conversion transformer.
type PdfAOptions struct {
	// Level specifies the PDF/A conformance level: "1b", "2b", or "3b".
	Level string
}

// LineariseOptions configures the linearisation transformer. Linearisation
// does not require any configuration.
type LineariseOptions struct{}

// ObjStmOptions configures the object stream compression transformer.
// Object stream compression does not require any configuration.
type ObjStmOptions struct{}

// FlattenOptions configures the flattening transformer.
type FlattenOptions struct {
	// FormFields flattens interactive form fields into static content.
	FormFields bool

	// Annotations flattens annotations into page content.
	Annotations bool

	// Transparency flattens transparency groups.
	Transparency bool
}

// RedactionOptions configures the redaction transformer.
type RedactionOptions struct {
	// TextPatterns holds regular expression patterns to match and redact.
	TextPatterns []string

	// Regions holds page-specific rectangular regions to redact.
	Regions []RedactionRegion

	// StripMetadata removes document metadata (author, title, etc.).
	StripMetadata bool
}

// CompressOptions configures the compression transformer.
type CompressOptions struct {
	// Algorithm selects the compression algorithm. Defaults to
	// CompressZstd when empty.
	Algorithm CompressAlgorithm
}

// RedactionRegion specifies a rectangular area on a page to redact.
type RedactionRegion struct {
	// Page is the zero-based page index.
	Page int

	// X is the left edge of the redaction rectangle in points.
	X float64

	// Y is the bottom edge of the redaction rectangle in points.
	Y float64

	// Width is the width of the redaction rectangle in points.
	Width float64

	// Height is the height of the redaction rectangle in points.
	Height float64
}
