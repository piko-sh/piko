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

package casefile

import (
	"fmt"
	"net/http"
)

const (
	// checkoutFinalAttempts is the submission count after the confirmed order is
	// resubmitted.
	checkoutFinalAttempts = 3
)

// CheckoutInput supplies the checkout's customer and order values.
type CheckoutInput struct {
	// Customer identifies the checkout customer.
	Customer string

	// SKU identifies the ordered item.
	SKU string

	// Quantity is the ordered item quantity.
	Quantity int

	// Unit is the item's unit price in cents.
	Unit int
}

// CheckoutState records one checkout screen and its submission counts.
type CheckoutState struct {
	// Error records the displayed error.
	Error string `json:",omitempty"`

	// Validation holds typed validation evidence.
	Validation *CheckoutValidation `json:",omitempty"`

	// Step names the checkout screen's interaction step.
	Step string

	// Path is the checkout screen's route.
	Path string

	// Customer is the displayed customer.
	Customer string

	// Status is the displayed submission status.
	Status string

	// Display is the displayed invoice line.
	Display string

	// Quantity is the displayed item quantity.
	Quantity int

	// Unit is the displayed unit price in cents.
	Unit int

	// Total is the displayed order total in cents.
	Total int

	// Attempts counts submission attempts.
	Attempts int

	// Committed counts committed orders.
	Committed int

	// Disabled reports whether the submit control is disabled.
	Disabled bool
}

// CheckoutValidation records the action client's typed validation error.
type CheckoutValidation struct {
	// Fields maps field names to validation messages.
	Fields map[string]string

	// Status is the HTTP response status.
	Status int

	// IsValidation reports whether the client classified the error as a validation error.
	IsValidation bool

	// IsNetwork reports whether the client classified the error as a network error.
	IsNetwork bool

	// IsAuth reports whether the client classified the error as an authentication error.
	IsAuth bool
}

// CheckoutDocuments records the persisted order and generated documents.
type CheckoutDocuments struct {
	// Attachment records the attached document name.
	Attachment string

	// Recipients lists the generated mail recipients.
	Recipients []string

	// Subjects lists the generated mail subjects.
	Subjects []string

	// HTMLLines lists the generated mail's HTML text lines.
	HTMLLines []string

	// PDFLines lists the generated PDF's text lines.
	PDFLines []string

	// Order holds the independently read persisted order.
	Order CheckoutInput

	// OrderTotal is the persisted order price in cents.
	OrderTotal int

	// Attempts counts persisted submission attempts.
	Attempts int

	// Commits counts persisted order commits.
	Commits int

	// Files counts generated document files.
	Files int
}

// CheckoutTrace records the checkout journey and its final documents.
type CheckoutTrace struct {
	// Documents holds independent order and document reads.
	Documents *CheckoutDocuments `json:",omitempty"`

	// DriverError records the browser driver's failure.
	DriverError string `json:",omitempty"`

	// States holds the ordered interaction snapshots.
	States []CheckoutState

	// Input holds the supplied interaction values.
	Input CheckoutInput

	// TypedErrors reports whether typed client errors are checked.
	TypedErrors bool `json:",omitempty"`
}

// InvoiceLine formats the checkout's independently calculated invoice line.
//
// Returns string which is the order's independently calculated invoice text.
func (in CheckoutInput) InvoiceLine() string {
	return fmt.Sprintf("%s / %s / %d x %d = %d cents", in.Customer, in.SKU, in.Quantity, in.Unit, in.Quantity*in.Unit)
}

// ExpectedCheckout models the checkout journey and generated documents.
//
// Takes in (CheckoutInput) which supplies the checkout customer and order values.
//
// Returns *CheckoutTrace which holds the expected checkout states and generated
// documents.
func ExpectedCheckout(in CheckoutInput) *CheckoutTrace {
	trace := &CheckoutTrace{Input: in, TypedErrors: true, States: nil, Documents: nil, DriverError: ""}
	state := CheckoutState{
		Error:      "",
		Validation: nil,
		Step:       "draft",
		Path:       "/checkout",
		Customer:   in.Customer,
		Status:     "",
		Display:    "",
		Quantity:   in.Quantity,
		Unit:       0,
		Total:      0,
		Attempts:   0,
		Committed:  0,
		Disabled:   false,
	}
	trace.States = append(trace.States, state)
	state.Path, state.Step, state.Status, state.Unit = "/checkout-review", "review", "ready", in.Unit
	trace.States = append(trace.States, state)
	add := func(step, status string, total, attempts, committed int, disabled bool) {
		state.Step, state.Status, state.Total, state.Attempts, state.Committed, state.Disabled = step, status, total, attempts, committed, disabled
		trace.States = append(trace.States, state)
	}
	total := in.Quantity * in.Unit
	add("first-pending", "pending", total, 1, 0, true)
	add("duplicate-while-pending", "pending", total, 1, 0, true)
	add("denied", "retry", 0, 1, 0, false)
	trace.States[len(trace.States)-1].Validation = &CheckoutValidation{
		Status:       http.StatusUnprocessableEntity,
		Fields:       map[string]string{"order": "order declined"},
		IsValidation: true,
		IsNetwork:    false,
		IsAuth:       false,
	}
	add("retry-pending", "pending", total, 2, 0, true)
	state.Display = in.InvoiceLine()
	add("confirmed", "saved", total, 2, 1, false)
	add("resubmit", "saved", total, checkoutFinalAttempts, 1, false)
	trace.Documents = &CheckoutDocuments{
		Order:      in,
		OrderTotal: in.Unit * in.Quantity,
		Attempts:   checkoutFinalAttempts,
		Commits:    1,
		Recipients: []string{"buyer@example.test"},
		Subjects:   []string{"Order " + in.SKU},
		HTMLLines:  []string{in.InvoiceLine()},
		PDFLines:   []string{in.InvoiceLine()},
		Attachment: "invoice.pdf",
		Files:      1,
	}
	return trace
}
