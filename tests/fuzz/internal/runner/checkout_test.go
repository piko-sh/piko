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

package runner

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestCheckoutOutboxAndOrderLedgerAreIndependent(t *testing.T) {
	directory := t.TempDir()
	outbox := filepath.Join(directory, checkoutOutboxName)
	require.NoError(t, os.Mkdir(outbox, 0700))
	mail := "To: reader@example.test\r\nSubject: invoice 2\r\nContent-Type: text/html\r\n\r\n<p>Item / 25 cents</p>"
	require.NoError(t, os.WriteFile(filepath.Join(outbox, "invoice.eml"), []byte(mail), 0600))
	got, err := readCheckoutMail(t.Context(), directory)
	require.NoError(t, err, "outbox without ledger")
	assert.Equal(t, 1, got.Files)
	assert.Equal(t, 0, got.Attempts)
	assert.Equal(t, []string{"Item / 25 cents"}, got.HTMLLines)
	require.NoError(t, os.RemoveAll(outbox))
	require.NoError(t, os.WriteFile(filepath.Join(directory, checkoutLedgerName), []byte(`{"Attempts":3,"Commits":1,"Order":{"Total":123}}`), 0600))
	got, err = readCheckoutMail(t.Context(), directory)
	require.NoError(t, err, "ledger without outbox")
	assert.Equal(t, 0, got.Files)
	assert.Equal(t, 3, got.Attempts)
	assert.Equal(t, 1, got.Commits)
	assert.Equal(t, 123, got.OrderTotal)
}

func TestCheckoutObservations(t *testing.T) {
	input := casefile.CheckoutInput{Customer: "Buyer 23", SKU: "linen", Quantity: 3, Unit: 142}
	want := casefile.ExpectedCheckout(input)
	verified, err := checkCheckout(want, want)
	require.NoError(t, err)
	require.Equal(t, 8, verified)
	for _, testCase := range []struct {
		alter          func(*casefile.CheckoutTrace)
		step, property string
		index          int
	}{
		{step: "review", property: "bound-draft", index: 1, alter: func(trace *casefile.CheckoutTrace) { trace.States[1].Customer = "stale" }},
		{step: "first-pending", property: "loading", index: 2, alter: func(trace *casefile.CheckoutTrace) { trace.States[2].Disabled = false }},
		{step: "duplicate-while-pending", property: "submission-count", index: 3, alter: func(trace *casefile.CheckoutTrace) { trace.States[3].Attempts++ }},
		{step: "denied", property: "total", index: 4, alter: func(trace *casefile.CheckoutTrace) { trace.States[4].Total = input.Quantity * input.Unit }},
		{step: "denied", property: "action-error", index: 4, alter: func(trace *casefile.CheckoutTrace) { trace.States[4].Validation.Fields["order"] = "unknown" }},
		{step: "retry-pending", property: "action-error", index: 5, alter: func(trace *casefile.CheckoutTrace) { trace.States[5].Error = "stale error" }},
		{step: "confirmed", property: "receipt", index: 6, alter: func(trace *casefile.CheckoutTrace) { trace.States[6].Display = "wrong order" }},
		{step: "resubmit", property: "submission-count", index: 7, alter: func(trace *casefile.CheckoutTrace) { trace.States[7].Committed++ }},
		{step: "invoice", property: "count", index: 8, alter: func(trace *casefile.CheckoutTrace) { trace.Documents.Files++ }},
		{step: "invoice", property: "server-attempts", index: 8, alter: func(trace *casefile.CheckoutTrace) { trace.Documents.Attempts++ }},
		{step: "invoice", property: "server-commits", index: 8, alter: func(trace *casefile.CheckoutTrace) { trace.Documents.Commits++ }},
		{step: "invoice", property: "persisted-order", index: 8, alter: func(trace *casefile.CheckoutTrace) { trace.Documents.Order.Quantity++ }},
		{step: "invoice", property: "persisted-total", index: 8, alter: func(trace *casefile.CheckoutTrace) { trace.Documents.OrderTotal++ }},
		{step: "invoice", property: "to", index: 8, alter: func(trace *casefile.CheckoutTrace) { trace.Documents.Recipients = nil }},
		{step: "invoice", property: "subject", index: 8, alter: func(trace *casefile.CheckoutTrace) { trace.Documents.Subjects = []string{"another order"} }},
		{step: "invoice", property: "email-lines", index: 8, alter: func(trace *casefile.CheckoutTrace) { trace.Documents.HTMLLines[0] = "stale" }},
		{step: "invoice", property: "pdf-lines", index: 8, alter: func(trace *casefile.CheckoutTrace) { trace.Documents.PDFLines[0] = "stale" }},
		{step: "invoice", property: "attachment", index: 8, alter: func(trace *casefile.CheckoutTrace) { trace.Documents.Attachment = "receipt.txt" }},
	} {
		t.Run(testCase.step+"/"+testCase.property, func(t *testing.T) {
			got := casefile.ExpectedCheckout(input)
			testCase.alter(got)
			verified, err := checkCheckout(got, want)
			assert.Equal(t, testCase.index, verified)
			failure, ok := errors.AsType[checkoutError](err)
			require.True(t, ok, "unexpected failure %v", err)
			assert.Equal(t, "/checkout/"+testCase.step+"/"+testCase.property, failure.FailureSubject())
		})
	}
}

func TestCheckoutErrorExposesUnderlyingFailure(t *testing.T) {
	err := error(checkoutError{step: "invoice", field: "read", err: fmt.Errorf("outbox: %w", context.Canceled)})

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, "checkout/invoice/read: outbox: context canceled", err.Error())
	mismatch, ok := errors.AsType[checkoutError](err)
	require.True(t, ok)
	assert.Equal(t, "/checkout/invoice/read", mismatch.FailureSubject())
	assert.ErrorIs(t, checkoutError{step: "invoice", field: "read", err: fs.ErrNotExist}, fs.ErrNotExist)
}

func TestCheckoutLedgerRejectsOversizedLedger(t *testing.T) {
	directory := t.TempDir()
	oversized := `{"Attempts":1,"Padding":"` + strings.Repeat("x", checkoutLedgerMax) + `"}`
	require.NoError(t, os.WriteFile(filepath.Join(directory, checkoutLedgerName), []byte(oversized), 0600))

	_, err := readCheckoutMail(t.Context(), directory)

	assert.ErrorIs(t, err, errCheckoutLedgerOversize)
}

func TestVerifiedCheckoutStepsCreditsTheModelledJourney(t *testing.T) {
	want := casefile.ExpectedCheckout(casefile.CheckoutInput{Customer: "Ada", SKU: "tea", Quantity: 2, Unit: 5})
	got := casefile.ExpectedCheckout(want.Input)

	assert.Equal(t, len(want.States), VerifiedCheckoutSteps(got, want))
	got.States = got.States[:2]
	assert.Equal(t, 2, VerifiedCheckoutSteps(got, want))
}

func minimalPDF(t *testing.T, line string) []byte {
	t.Helper()
	stream := "BT /F1 12 Tf 72 720 Td (" + line + ") Tj ET"
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var document bytes.Buffer
	document.WriteString("%PDF-1.4\n")
	offsets := make([]int, 0, len(objects))
	for index, object := range objects {
		offsets = append(offsets, document.Len())
		fmt.Fprintf(&document, "%d 0 obj\n%s\nendobj\n", index+1, object)
	}
	table := document.Len()
	fmt.Fprintf(&document, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&document, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&document, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, table)
	return document.Bytes()
}

func writeInvoiceMail(t *testing.T, directory, body string) {
	t.Helper()
	outbox := filepath.Join(directory, checkoutOutboxName)
	require.NoError(t, os.MkdirAll(outbox, 0700))
	mail := "To: Reader <reader@example.test>\r\nSubject: =?utf-8?q?Invoice_=C3=A9?=\r\n" + body
	require.NoError(t, os.WriteFile(filepath.Join(outbox, "invoice.eml"), []byte(mail), 0600))
}

func TestReadCheckoutMailReadsTheHTMLAndPDFInvoice(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext is not installed")
	}
	directory := t.TempDir()
	attachment := base64.StdEncoding.EncodeToString(minimalPDF(t, "Linen x 3 / 426 cents"))
	writeInvoiceMail(t, directory, "Content-Type: multipart/mixed; boundary=outer\r\n\r\n"+
		"--outer\r\nContent-Type: multipart/alternative; boundary=inner\r\n\r\n"+
		"--inner\r\nContent-Type: text/plain\r\n\r\nplain body\r\n"+
		"--inner\r\nContent-Type: text/html\r\n\r\n<p>Linen x 3 / 426 cents</p>\r\n--inner--\r\n"+
		"--outer\r\nContent-Type: application/pdf\r\nContent-Transfer-Encoding: base64\r\n"+
		"Content-Disposition: attachment; filename=\"invoice.pdf\"\r\n\r\n"+attachment+"\r\n--outer--\r\n")

	got, err := readCheckoutMail(t.Context(), directory)

	require.NoError(t, err)
	assert.Equal(t, []string{"reader@example.test"}, got.Recipients)
	assert.Equal(t, []string{"Invoice é"}, got.Subjects)
	assert.Equal(t, []string{"Linen x 3 / 426 cents"}, got.HTMLLines)
	assert.Equal(t, []string{"Linen x 3 / 426 cents"}, got.PDFLines)
	assert.Equal(t, "invoice.pdf", got.Attachment)
	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "the PDF staging file was left behind")
}

func TestReadCheckoutMailRejectsMalformedMIME(t *testing.T) {
	nested := "Content-Type: multipart/mixed; boundary=b0\r\n\r\n"
	closing := ""
	for depth := range checkoutMIMEDepthMax + 1 {
		nested += fmt.Sprintf("--b%d\r\nContent-Type: multipart/mixed; boundary=b%d\r\n\r\n", depth, depth+1)
		closing = fmt.Sprintf("--b%d--\r\n", depth) + closing
	}
	excess := "Content-Type: multipart/mixed; boundary=p\r\n\r\n" +
		strings.Repeat("--p\r\nContent-Type: text/plain\r\n\r\nx\r\n", checkoutMIMEPartsMax+1) + "--p--\r\n"
	for name, testCase := range map[string]struct {
		body string
		want string
	}{
		"nesting":     {body: nested + closing, want: "nesting exceeds"},
		"excess":      {body: excess, want: "excess MIME parts"},
		"media":       {body: "Content-Type: ;;\r\n\r\nbody", want: "mime: no media type"},
		"disposition": {body: "Content-Type: application/pdf\r\nContent-Disposition: ;;\r\n\r\nbody", want: "mime: no media type"},
	} {
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			writeInvoiceMail(t, directory, testCase.body)
			_, err := readCheckoutMail(t.Context(), directory)
			assert.ErrorContains(t, err, testCase.want)
		})
	}
}

func TestReadCheckoutMailReportsUnreadablePDF(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext is not installed")
	}
	directory := t.TempDir()
	writeInvoiceMail(t, directory, "Content-Type: multipart/mixed; boundary=p\r\n\r\n--p\r\nContent-Type: application/pdf\r\n"+
		"Content-Disposition: attachment; filename=bad.pdf\r\n\r\nnot a pdf\r\n--p--\r\n")

	_, err := readCheckoutMail(t.Context(), directory)

	assert.ErrorContains(t, err, "checkout/invoice/pdf-read")
}

func TestReadCheckoutMailReadsATopLevelPDF(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext is not installed")
	}
	for _, testCase := range []struct {
		name       string
		header     string
		attachment string
	}{
		{name: "attachment", header: "Content-Disposition: attachment; filename=invoice.pdf\r\n", attachment: "invoice.pdf"},
		{name: "inline", header: "", attachment: ""},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			writeInvoiceMail(t, directory, "Content-Type: application/pdf\r\n"+testCase.header+"\r\nnot a pdf\r\n")

			documents, err := readCheckoutMail(t.Context(), directory)

			require.ErrorContains(t, err, "checkout/invoice/pdf-read", "the PDF body must reach the reader")
			assert.NotContains(t, err.Error(), "no media type")
			assert.Equal(t, testCase.attachment, documents.Attachment)
		})
	}
}
