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
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvoiceLineCalculatesTheTotal(t *testing.T) {
	assert.Equal(t, "Ada / tea / 3 x 25 = 75 cents", CheckoutInput{Customer: "Ada", SKU: "tea", Quantity: 3, Unit: 25}.InvoiceLine())
}

func TestExpectedCheckoutModelsRetryAndDocuments(t *testing.T) {
	input := CheckoutInput{Customer: "Ada", SKU: "tea", Quantity: 3, Unit: 25}

	trace := ExpectedCheckout(input)

	steps := make([]string, 0, len(trace.States))
	for _, state := range trace.States {
		steps = append(steps, state.Step)
	}
	assert.Equal(t, []string{"draft", "review", "first-pending", "duplicate-while-pending", "denied", "retry-pending", "confirmed", "resubmit"}, steps)
	denied := trace.States[4]
	require.NotNil(t, denied.Validation)
	assert.Equal(t, http.StatusUnprocessableEntity, denied.Validation.Status)
	assert.Equal(t, input.InvoiceLine(), trace.States[len(trace.States)-1].Display)
	require.NotNil(t, trace.Documents)
	assert.Equal(t, 75, trace.Documents.OrderTotal)
	assert.Equal(t, []string{"Order tea"}, trace.Documents.Subjects)
}
