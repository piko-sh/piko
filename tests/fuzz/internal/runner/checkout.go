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
	"io"
	"io/fs"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"golang.org/x/net/html"
	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/process"
)

const (
	// checkoutDocumentMax bounds independently read document bytes.
	checkoutDocumentMax = 1 << 20

	// checkoutLedgerMax bounds the persisted order ledger in bytes.
	checkoutLedgerMax = 4096

	// checkoutLedgerName is the persisted order ledger, relative to the working directory.
	checkoutLedgerName = "checkout-state.json"

	// checkoutMIMEDepthMax bounds nested multipart levels in the generated invoice mail.
	checkoutMIMEDepthMax = 4

	// checkoutMIMEPartsMax bounds the parts read from one multipart level.
	checkoutMIMEPartsMax = 12

	// checkoutOutboxName is the generated mail outbox, relative to the working directory.
	checkoutOutboxName = "checkout-outbox"

	// checkoutOutboxListMax bounds the number of inspected outbox entries.
	checkoutOutboxListMax = 3

	// checkoutRegistry registers the checkout's native DTO for interpreted documents.
	checkoutRegistry = `package main
import (
	"reflect"
	_ "example.org/pikofuzzcase/dist"
	"example.org/pikofuzzcase/pkg/invoice"
)
func init() {
	extraSymbols = map[string]map[string]reflect.Value{
		"example.org/pikofuzzcase/pkg/invoice": {"Order": reflect.ValueOf((*invoice.Order)(nil))},
	}
}
`
)

var (
	// errCheckoutLedgerOversize reports a persisted order ledger above checkoutLedgerMax.
	errCheckoutLedgerOversize = errors.New("checkout ledger exceeds byte limit")
)

// checkoutError identifies a checkout field mismatch.
type checkoutError struct {
	// err holds the underlying failure, when one caused the mismatch.
	err error

	// step identifies the checkout step that failed.
	step string

	// field identifies the mismatched checkout property.
	field string

	// detail describes the mismatch when no underlying error is present.
	detail string
}

// checkoutLedger is the persisted order ledger written by the checkout fixture.
type checkoutLedger struct {
	// Order holds the persisted order.
	Order checkoutLedgerOrder

	// Attempts counts server-side submission attempts.
	Attempts int

	// Commits counts committed orders.
	Commits int
}

// checkoutLedgerOrder is the persisted order within the checkout ledger.
type checkoutLedgerOrder struct {
	// Customer is the persisted customer name.
	Customer string

	// SKU is the persisted stock-keeping unit.
	SKU string

	// Quantity is the persisted line quantity.
	Quantity int

	// Unit is the persisted unit price in cents.
	Unit int

	// Total is the persisted order total in cents.
	Total int
}

// checkoutProperty pairs one invoice property name with its observed and expected values.
type checkoutProperty struct {
	// got holds the observed value.
	got any

	// want holds the independently expected value.
	want any

	// name identifies the invoice property.
	name string
}

// checkoutMailWalker extracts invoice evidence from one generated MIME message.
type checkoutMailWalker struct {
	// out receives the extracted evidence.
	out *casefile.CheckoutDocuments

	// work is the invocation-private working directory holding the outbox.
	work string
}

// Error formats the recorded failure.
//
// Returns string which is the formatted failure message.
func (e checkoutError) Error() string {
	detail := e.detail
	if e.err != nil {
		detail = e.err.Error()
	}
	return fmt.Sprintf("checkout/%s/%s: %s", e.step, e.field, detail)
}

// Unwrap exposes the underlying failure.
//
// Returns error which is the underlying failure, or nil for a pure mismatch.
func (e checkoutError) Unwrap() error {
	return e.err
}

// FailureSubject identifies the failed property independently of generated values.
//
// Returns string which is the stable failed-property identity.
func (e checkoutError) FailureSubject() string {
	return "/checkout/" + e.step + "/" + e.field
}

// walk extracts invoice evidence from one MIME entity and its descendants.
//
// Takes content (string) which is the entity Content-Type header.
// Takes encoding (string) which is the entity Content-Transfer-Encoding header.
// Takes disposition (string) which is the entity Content-Disposition header.
// Takes body (io.Reader) which supplies the entity body.
// Takes depth (int) which is the current multipart nesting depth.
//
// Returns error which reports malformed, oversized or unreadable MIME evidence, or nil on
// success.
func (w *checkoutMailWalker) walk(ctx context.Context, content, encoding, disposition string, body io.Reader, depth int) error {
	if depth > checkoutMIMEDepthMax {
		return fmt.Errorf("checkout/invoice/mime: nesting exceeds %d", checkoutMIMEDepthMax)
	}
	mediaType, parameters, err := mime.ParseMediaType(content)
	if err != nil {
		return err
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		return w.walkMultipart(ctx, multipart.NewReader(body, parameters["boundary"]), depth)
	}
	if strings.EqualFold(encoding, "base64") {
		body = base64.NewDecoder(base64.StdEncoding, body)
	}
	switch mediaType {
	case "text/html":
		return w.readHTML(body)
	case "application/pdf":
		return w.readPDF(ctx, disposition, body)
	}
	return nil
}

// walkMultipart walks the bounded parts of one multipart entity.
//
// Takes reader (*multipart.Reader) which yields the entity parts.
// Takes depth (int) which is the current multipart nesting depth.
//
// Returns error which reports malformed or excess parts, or nil on success.
func (w *checkoutMailWalker) walkMultipart(ctx context.Context, reader *multipart.Reader, depth int) error {
	for range checkoutMIMEPartsMax {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		err = w.walk(ctx, part.Header.Get(headerContentType), part.Header.Get("Content-Transfer-Encoding"), part.Header.Get("Content-Disposition"), part, depth+1)
		if closeErr := part.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
	}
	return errors.New("checkout/invoice/mime: excess MIME parts")
}

// readHTML records the priced lines of the HTML invoice body.
//
// Takes body (io.Reader) which supplies the HTML body.
//
// Returns error which reports malformed HTML, or nil on success.
func (w *checkoutMailWalker) readHTML(body io.Reader) error {
	document, err := html.Parse(body)
	if err != nil {
		return err
	}
	for node := range document.Descendants() {
		if node.Type == html.TextNode && strings.HasSuffix(strings.TrimSpace(node.Data), " cents") {
			w.out.HTMLLines = append(w.out.HTMLLines, strings.TrimSpace(node.Data))
		}
	}
	return nil
}

// readPDF isolates the PDF attachment and records its priced lines.
//
// Takes disposition (string) which is the attachment Content-Disposition header.
// Takes body (io.Reader) which supplies the decoded attachment bytes.
//
// Returns error which reports an oversized, unreadable or unparsable attachment, or nil
// on success.
func (w *checkoutMailWalker) readPDF(ctx context.Context, disposition string, body io.Reader) (err error) {
	if disposition != "" {
		_, parameters, err := mime.ParseMediaType(disposition)
		if err != nil {
			return err
		}
		w.out.Attachment = parameters["filename"]
	}
	data, err := io.ReadAll(io.LimitReader(body, checkoutDocumentMax+1))
	if err != nil {
		return err
	}
	if len(data) > checkoutDocumentMax {
		return fmt.Errorf("checkout/invoice/pdf-size: attachment exceeds %d bytes", checkoutDocumentMax)
	}
	destination, err := os.CreateTemp(w.work, "checkout-*")
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := os.Remove(destination.Name()); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()
	_, err = destination.Write(data)
	if closeErr := destination.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	text, err := process.Run(ctx, w.work, nil, "pdftotext", "-f", "1", "-l", "1", "-layout", destination.Name(), "-")
	if err != nil {
		return fmt.Errorf("checkout/invoice/pdf-read: %w", err)
	}
	for line := range strings.Lines(text) {
		if value := strings.Join(strings.Fields(line), " "); strings.HasSuffix(value, " cents") {
			w.out.PDFLines = append(w.out.PDFLines, value)
		}
	}
	return nil
}

// VerifiedCheckoutSteps counts the verified prefix of the checkout journey.
//
// Takes got (*casefile.CheckoutTrace) which holds the observed result.
// Takes want (*casefile.CheckoutTrace) which holds the independently expected result.
//
// Returns int which counts the matching prefix of checkout states.
func VerifiedCheckoutSteps(got, want *casefile.CheckoutTrace) int {
	verified, _ := checkCheckout(got, want)
	return verified
}

// checkCheckout checks checkout screens and independent generated documents.
//
// Takes got (*casefile.CheckoutTrace) which holds the observed result.
// Takes want (*casefile.CheckoutTrace) which holds the independently expected result.
//
// Returns int which counts the matching prefix of checkout states.
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkCheckout(got, want *casefile.CheckoutTrace) (int, error) {
	if got == nil && want == nil {
		return 0, nil
	}
	if got == nil || want == nil || got.Input != want.Input {
		return 0, checkoutError{err: nil, step: "draft", field: "recipe", detail: "missing or mismatched checkout"}
	}
	for index := range want.States {
		expected := &want.States[index]
		if index >= len(got.States) {
			return index, checkoutError{err: nil, step: expected.Step, field: "driver", detail: got.DriverError}
		}
		if field := checkoutStateMismatch(got.States[index], *expected, want.TypedErrors); field != "" {
			return index, checkoutError{err: nil, step: expected.Step, field: field, detail: fmt.Sprintf("got %+v want %+v", got.States[index], *expected)}
		}
	}
	count := len(want.States)
	if len(got.States) != count || got.DriverError != "" {
		return count, checkoutError{err: nil, step: "final", field: "driver", detail: got.DriverError}
	}
	return count, checkCheckoutDocuments(got.Documents, want.Documents)
}

// checkCheckoutDocuments compares the persisted order and generated documents.
//
// Takes observed (*casefile.CheckoutDocuments) which holds the outbox measurement.
// Takes expected (*casefile.CheckoutDocuments) which holds the modelled documents, nil
// when none are expected.
//
// Returns error which reports a missing measurement or the first mismatched property.
func checkCheckoutDocuments(observed, expected *casefile.CheckoutDocuments) error {
	if expected == nil {
		return nil
	}
	if observed == nil {
		return checkoutError{err: nil, step: "invoice", field: "missing", detail: "no local outbox measurement"}
	}
	properties := []checkoutProperty{
		{name: "count", got: observed.Files, want: expected.Files},
		{name: "server-attempts", got: observed.Attempts, want: expected.Attempts},
		{name: "server-commits", got: observed.Commits, want: expected.Commits},
		{name: "persisted-order", got: observed.Order, want: expected.Order},
		{name: "persisted-total", got: observed.OrderTotal, want: expected.OrderTotal},
		{name: "to", got: observed.Recipients, want: expected.Recipients},
		{name: "subject", got: observed.Subjects, want: expected.Subjects},
		{name: "email-lines", got: observed.HTMLLines, want: expected.HTMLLines},
		{name: "attachment", got: observed.Attachment, want: expected.Attachment},
		{name: "pdf-lines", got: observed.PDFLines, want: expected.PDFLines},
	}
	for _, property := range properties {
		if !reflect.DeepEqual(property.got, property.want) {
			return checkoutError{err: nil, step: "invoice", field: property.name, detail: fmt.Sprintf("got %v want %v", property.got, property.want)}
		}
	}
	return nil
}

// checkoutStateMismatch names the first mismatched property of one checkout state.
//
// Takes got (casefile.CheckoutState) which is the observed state.
// Takes want (casefile.CheckoutState) which is the expected state.
// Takes typedErrors (bool) which selects typed validation comparison.
//
// Returns string which names the mismatched property, or "" when the states agree.
func checkoutStateMismatch(got, want casefile.CheckoutState, typedErrors bool) string {
	switch {
	case want.Step != got.Step || want.Path != got.Path:
		return "destination"
	case want.Customer != got.Customer || want.Quantity != got.Quantity || want.Unit != got.Unit:
		return "bound-draft"
	case want.Status != got.Status || want.Disabled != got.Disabled:
		return "loading"
	case typedErrors && (!reflect.DeepEqual(want.Validation, got.Validation) || (want.Validation == nil && got.Error != "")):
		return "action-error"
	case want.Total != got.Total:
		return "total"
	case want.Attempts != got.Attempts || want.Committed != got.Committed:
		return "submission-count"
	case want.Display != got.Display:
		return "receipt"
	}
	return ""
}

// readCheckoutLedger reads the checkout's persisted order ledger.
//
// Takes work (string) which is the invocation-private working directory holding the
// ledger.
// Takes out (*casefile.CheckoutDocuments) which receives the independently read persisted
// order.
//
// Returns error which reports a malformed or oversized persisted order ledger, or nil on
// success or when no ledger was written.
func readCheckoutLedger(work string, out *casefile.CheckoutDocuments) error {
	data, err := readBoundedFile(work, checkoutLedgerName, checkoutLedgerMax)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if errors.Is(err, errFileOversize) {
		return fmt.Errorf("%w: more than %d bytes", errCheckoutLedgerOversize, checkoutLedgerMax)
	}
	if err != nil {
		return fmt.Errorf("checkout ledger: %w", err)
	}
	var ledger checkoutLedger
	if err = json.Unmarshal(data, &ledger); err != nil {
		return fmt.Errorf("checkout ledger: %w", err)
	}
	out.Order = casefile.CheckoutInput{Customer: ledger.Order.Customer, SKU: ledger.Order.SKU, Quantity: ledger.Order.Quantity, Unit: ledger.Order.Unit}
	out.OrderTotal, out.Attempts, out.Commits = ledger.Order.Total, ledger.Attempts, ledger.Commits
	return nil
}

// readCheckoutMail reads generated mail and PDF evidence independently.
//
// Takes work (string) which is the invocation-private working directory holding the
// outbox and the order ledger.
//
// Returns *casefile.CheckoutDocuments which records independent email and PDF document
// evidence.
// Returns error which reports unreadable or invalid generated mail or PDF evidence, or
// nil on success.
func readCheckoutMail(ctx context.Context, work string) (*casefile.CheckoutDocuments, error) {
	out := &casefile.CheckoutDocuments{}
	if err := readCheckoutLedger(work, out); err != nil {
		return out, err
	}
	names, err := readCheckoutOutbox(work)
	if errors.Is(err, fs.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	out.Files = len(names)
	if len(names) != 1 {
		return out, nil
	}
	data, err := readBoundedFile(work, filepath.Join(checkoutOutboxName, names[0]), checkoutDocumentMax)
	if err != nil {
		return out, fmt.Errorf("checkout/invoice/size: %w", err)
	}
	message, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		return out, err
	}
	recipients, err := message.Header.AddressList("To")
	if err != nil {
		return out, err
	}
	for _, recipient := range recipients {
		out.Recipients = append(out.Recipients, recipient.Address)
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
	if err != nil {
		return out, err
	}
	out.Subjects = []string{subject}
	walker := &checkoutMailWalker{out: out, work: work}
	err = walker.walk(ctx, message.Header.Get(headerContentType), message.Header.Get("Content-Transfer-Encoding"), message.Header.Get("Content-Disposition"), message.Body, 0)
	return out, err
}

// readCheckoutOutbox lists at most three entries of the checkout outbox.
//
// Takes work (string) which is the invocation-private working directory holding the
// outbox.
//
// Returns []string which holds the listed entry names.
// Returns error which reports an unreadable outbox, wrapping fs.ErrNotExist when absent.
func readCheckoutOutbox(work string) (names []string, err error) {
	directory, err := os.OpenInRoot(work, checkoutOutboxName)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, directory.Close()) }()
	names, err = directory.Readdirnames(checkoutOutboxListMax)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return names, nil
}
