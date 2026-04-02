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
	"strings"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

// chainStats holds counts and flags gathered during a single pass over the
// transformer list, used by ValidateChain to check constraints.
type chainStats struct {
	// encryptorCount holds the number of encryption transformers found.
	encryptorCount int

	// signerCount holds the number of signing transformers found.
	signerCount int

	// pdfaCount holds the number of PDF/A compliance transformers found.
	pdfaCount int

	// encryptorPriority holds the priority of the encryption transformer.
	encryptorPriority int

	// signerPriority holds the priority of the signing transformer.
	signerPriority int

	// hasLinearisation indicates whether a linearisation transformer is present.
	hasLinearisation bool

	// hasPdfA1 indicates whether a PDF/A-1b transformer is present.
	hasPdfA1 bool
}

// ValidateChain checks that the transformer combination is valid. It enforces
// ordering and mutual exclusion constraints that must hold for the chain to
// produce correct output.
//
// Rules enforced:
//   - At most one encryption transformer per chain.
//   - At most one signing transformer per chain.
//   - At most one PDF/A level per chain.
//   - Signing priority must be >= encryption priority if both are present.
//   - Linearisation must not coexist with PDF/A-1b.
//
// Takes transformers ([]PdfTransformerPort) which is the sorted transformer
// list to validate.
//
// Returns error describing the first constraint violation found, or nil if
// the chain is valid.
func ValidateChain(transformers []PdfTransformerPort) error {
	stats := gatherChainStats(transformers)
	return checkChainConstraints(stats)
}

// gatherChainStats iterates over all transformers and collects counts
// and flags into a chainStats value for constraint checking.
//
// Takes transformers ([]PdfTransformerPort) which is the list to analyse.
//
// Returns chainStats which holds the aggregated counts and flags.
func gatherChainStats(transformers []PdfTransformerPort) chainStats {
	var stats chainStats
	for _, t := range transformers {
		name := t.Name()
		classifySecurityTransformer(t, name, &stats)
		classifyComplianceTransformer(name, &stats)
		if isLineariser(name) {
			stats.hasLinearisation = true
		}
	}
	return stats
}

// classifySecurityTransformer updates stats with encryptor or signer
// counts if the transformer has a security type.
//
// Takes t (PdfTransformerPort) which is the transformer to classify.
// Takes name (string) which is the transformer name.
// Takes stats (*chainStats) which accumulates the classification results.
func classifySecurityTransformer(t PdfTransformerPort, name string, stats *chainStats) {
	if t.Type() != pdfwriter_dto.TransformerSecurity {
		return
	}
	if isEncryptor(name) {
		stats.encryptorCount++
		stats.encryptorPriority = t.Priority()
	}
	if isSigner(name) {
		stats.signerCount++
		stats.signerPriority = t.Priority()
	}
}

// classifyComplianceTransformer updates stats with PDF/A counts if the
// transformer name indicates a compliance transformer.
//
// Takes name (string) which is the transformer name.
// Takes stats (*chainStats) which accumulates the classification results.
func classifyComplianceTransformer(name string, stats *chainStats) {
	if !isPdfA(name) {
		return
	}
	stats.pdfaCount++
	if isPdfA1(name) {
		stats.hasPdfA1 = true
	}
}

// checkChainConstraints validates the gathered stats against all
// transformer chain rules and returns the first violation found.
//
// Takes stats (chainStats) which holds the aggregated counts and flags.
//
// Returns error describing the constraint violation, or nil if valid.
func checkChainConstraints(stats chainStats) error {
	if stats.encryptorCount > 1 {
		return ErrTooManyEncryptors
	}
	if stats.signerCount > 1 {
		return ErrTooManySigners
	}
	if stats.pdfaCount > 1 {
		return ErrTooManyPdfALevels
	}
	if stats.encryptorCount > 0 && stats.signerCount > 0 && stats.signerPriority < stats.encryptorPriority {
		return ErrSigningBeforeEncryption
	}
	if stats.hasLinearisation && stats.hasPdfA1 {
		return ErrLinearisationWithPdfA1
	}
	return nil
}

// isEncryptor reports whether the transformer name identifies an
// encryption transformer.
//
// Takes name (string) which is the transformer name.
//
// Returns bool which is true if the name starts with "pdf-encrypt".
func isEncryptor(name string) bool {
	return strings.HasPrefix(name, "pdf-encrypt")
}

// isSigner reports whether the transformer name identifies a signing
// transformer.
//
// Takes name (string) which is the transformer name.
//
// Returns bool which is true if the name starts with "pades-".
func isSigner(name string) bool {
	return strings.HasPrefix(name, "pades-")
}

// isPdfA reports whether the transformer name identifies a PDF/A
// compliance transformer.
//
// Takes name (string) which is the transformer name.
//
// Returns bool which is true if the name starts with "pdfa-".
func isPdfA(name string) bool {
	return strings.HasPrefix(name, "pdfa-")
}

// isPdfA1 reports whether the transformer name identifies the PDF/A-1b
// compliance level specifically.
//
// Takes name (string) which is the transformer name.
//
// Returns bool which is true if the name is exactly "pdfa-1b".
func isPdfA1(name string) bool {
	return name == "pdfa-1b"
}

// isLineariser reports whether the transformer name identifies a
// linearisation transformer.
//
// Takes name (string) which is the transformer name.
//
// Returns bool which is true if the name is exactly "linearise".
func isLineariser(name string) bool {
	return name == "linearise"
}
