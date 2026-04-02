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

package pdfwriter_domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

func securityTransformer(name string, priority int) *mockTransformer {
	return &mockTransformer{
		name:     name,
		ttype:    pdfwriter_dto.TransformerSecurity,
		priority: priority,
	}
}

func complianceTransformer(name string, priority int) *mockTransformer {
	return &mockTransformer{
		name:     name,
		ttype:    pdfwriter_dto.TransformerCompliance,
		priority: priority,
	}
}

func deliveryTransformer(name string, priority int) *mockTransformer {
	return &mockTransformer{
		name:     name,
		ttype:    pdfwriter_dto.TransformerDelivery,
		priority: priority,
	}
}

func TestValidateChain_Empty(t *testing.T) {
	err := pdfwriter_domain.ValidateChain(nil)
	assert.NoError(t, err)
}

func TestValidateChain_SingleTransformer(t *testing.T) {
	transformers := []pdfwriter_domain.PdfTransformerPort{
		securityTransformer("pdf-encrypt-aes-256", 400),
	}
	err := pdfwriter_domain.ValidateChain(transformers)
	assert.NoError(t, err)
}

func TestValidateChain_ValidFullPipeline(t *testing.T) {
	transformers := []pdfwriter_domain.PdfTransformerPort{
		&mockTransformer{name: "redaction", ttype: pdfwriter_dto.TransformerContent, priority: 100},
		&mockTransformer{name: "flatten", ttype: pdfwriter_dto.TransformerContent, priority: 120},
		&mockTransformer{name: "watermark", ttype: pdfwriter_dto.TransformerContent, priority: 150},
		complianceTransformer("pdfa-2b", 200),
		deliveryTransformer("linearise", 350),
		securityTransformer("pdf-encrypt-aes-256", 400),
		securityTransformer("pades-b-b", 450),
	}
	err := pdfwriter_domain.ValidateChain(transformers)
	assert.NoError(t, err)
}

func TestValidateChain_TooManyEncryptors(t *testing.T) {
	transformers := []pdfwriter_domain.PdfTransformerPort{
		securityTransformer("pdf-encrypt-aes-256", 400),
		securityTransformer("pdf-encrypt-aes-128", 401),
	}
	err := pdfwriter_domain.ValidateChain(transformers)
	assert.ErrorIs(t, err, pdfwriter_domain.ErrTooManyEncryptors)
}

func TestValidateChain_TooManySigners(t *testing.T) {
	transformers := []pdfwriter_domain.PdfTransformerPort{
		securityTransformer("pades-b-b", 450),
		securityTransformer("pades-b-t", 451),
	}
	err := pdfwriter_domain.ValidateChain(transformers)
	assert.ErrorIs(t, err, pdfwriter_domain.ErrTooManySigners)
}

func TestValidateChain_TooManyPdfALevels(t *testing.T) {
	transformers := []pdfwriter_domain.PdfTransformerPort{
		complianceTransformer("pdfa-1b", 200),
		complianceTransformer("pdfa-2b", 201),
	}
	err := pdfwriter_domain.ValidateChain(transformers)
	assert.ErrorIs(t, err, pdfwriter_domain.ErrTooManyPdfALevels)
}

func TestValidateChain_SigningBeforeEncryption(t *testing.T) {
	transformers := []pdfwriter_domain.PdfTransformerPort{
		securityTransformer("pades-b-b", 300),
		securityTransformer("pdf-encrypt-aes-256", 400),
	}
	err := pdfwriter_domain.ValidateChain(transformers)
	assert.ErrorIs(t, err, pdfwriter_domain.ErrSigningBeforeEncryption)
}

func TestValidateChain_SigningAfterEncryption_Valid(t *testing.T) {
	transformers := []pdfwriter_domain.PdfTransformerPort{
		securityTransformer("pdf-encrypt-aes-256", 400),
		securityTransformer("pades-b-b", 450),
	}
	err := pdfwriter_domain.ValidateChain(transformers)
	assert.NoError(t, err)
}

func TestValidateChain_LinearisationWithPdfA1(t *testing.T) {
	transformers := []pdfwriter_domain.PdfTransformerPort{
		complianceTransformer("pdfa-1b", 200),
		deliveryTransformer("linearise", 350),
	}
	err := pdfwriter_domain.ValidateChain(transformers)
	assert.ErrorIs(t, err, pdfwriter_domain.ErrLinearisationWithPdfA1)
}

func TestValidateChain_LinearisationWithPdfA2_Valid(t *testing.T) {
	transformers := []pdfwriter_domain.PdfTransformerPort{
		complianceTransformer("pdfa-2b", 200),
		deliveryTransformer("linearise", 350),
	}
	err := pdfwriter_domain.ValidateChain(transformers)
	assert.NoError(t, err)
}
