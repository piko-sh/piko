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
	"embed"
	"fmt"
	"strconv"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/fixturego"
)

const (
	// checkoutQuantityChoices bounds the drawn order quantity.
	checkoutQuantityChoices = 5

	// checkoutBaseUnit is the unit price in minor units before follow-up adjustment.
	checkoutBaseUnit = 105

	// checkoutUnitStep is the unit price increase per follow-up choice.
	checkoutUnitStep = 37
)

var (
	// checkoutTemplates embeds the checkout page, review, PDF and email templates.
	//
	//go:embed checkout_*.pk.tmpl
	checkoutTemplates embed.FS

	// checkoutRelease is the formatted release action staged in the generated project.
	checkoutRelease = fixturego.Fixed("actions/shop/release.go")

	// checkoutDTO is the formatted invoice DTO staged in the generated project.
	checkoutDTO = fixturego.Fixed("pkg/invoice/dto.go")
)

// addCheckout adds source and expected observations for checkout interactions.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
//
// Returns error which reports an unavailable checkout template or a stale fixture edit,
// or nil on success.
func addCheckout(c *casefile.Case, p Project) error {
	customers := []string{"Buyer", "A & B", "Buyer <B>", "Café"}
	skus := []string{"linen", "amber"}
	input := casefile.CheckoutInput{
		Customer: fmt.Sprintf("%s %d", customers[p.Count%len(customers)], 20+p.Count),
		SKU:      skus[(c.Seed+c.Index)%uint64(len(skus))],
		Quantity: 1 + p.Count%checkoutQuantityChoices,
		Unit:     checkoutBaseUnit + p.Followup*checkoutUnitStep,
	}
	c.Expected.Client.Checkout = casefile.ExpectedCheckout(input)
	templates := []struct{ output, template string }{
		{output: "pages/checkout.pk", template: "checkout_page.pk.tmpl"},
		{output: "pages/checkout-review.pk", template: "checkout_review.pk.tmpl"},
		{output: "pdfs/order.pk", template: "checkout_pdf.pk.tmpl"},
		{output: "emails/order.pk", template: "checkout_email.pk.tmpl"},
	}
	replacer := strings.NewReplacer("INVOICE_UNIT", strconv.Itoa(input.Unit), "INVOICE_SKU", input.SKU)
	for _, entry := range templates {
		data, err := checkoutTemplates.ReadFile(entry.template)
		if err != nil {
			return fmt.Errorf("reading checkout template %s: %w", entry.template, err)
		}
		c.Files[entry.output] = replacer.Replace(string(data))
	}
	place, err := checkoutPlace(input)
	if err != nil {
		return err
	}
	c.Files["actions/shop/place.go"] = place
	c.Files["actions/shop/release.go"] = checkoutRelease
	c.Files["pkg/invoice/dto.go"] = checkoutDTO
	c.Probes = append(c.Probes,
		textProbe("/checkout", "Customer and quantity"),
		textProbe("/checkout-review", "Review order"),
	)
	c.Features = append(c.Features, "cross-route-draft", "optimistic-retry", "deduplicated-order", "priced-pdf", "local-email-attachment")
	return nil
}
