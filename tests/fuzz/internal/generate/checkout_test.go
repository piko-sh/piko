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

package generate

import (
	"go/parser"
	"go/token"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestCheckoutEmitsPricedTemplates(t *testing.T) {
	c := casefile.Case{Files: map[string]string{}, Expected: casefile.Expected{Client: &casefile.ClientTransition{}}}
	p := Project{Count: 2, Followup: 1}
	require.NoError(t, addCheckout(&c, p))
	unit := strconv.Itoa(105 + 37)
	for _, name := range []string{"pages/checkout.pk", "pages/checkout-review.pk", "pdfs/order.pk", "emails/order.pk"} {
		require.NotEmpty(t, c.Files[name], name)
		assert.NotContains(t, c.Files[name], "INVOICE_UNIT", name)
		assert.NotContains(t, c.Files[name], "INVOICE_SKU", name)
	}
	_, err := parser.ParseFile(token.NewFileSet(), "place.go", c.Files["actions/shop/place.go"], parser.AllErrors)
	require.NoError(t, err)
	assert.Contains(t, c.Files["actions/shop/place.go"], unit)
	assert.Contains(t, c.Files["actions/shop/place.go"], strconv.Quote("linen"))
	require.NotNil(t, c.Expected.Client.Checkout)
	assert.Len(t, c.Probes, 2)
}
