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

package cart

import (
	"fmt"
	"time"

	"piko.sh/piko"
	"piko.sh/piko/wdk/maths"
)

// PurchaseAction processes a cart checkout.
//
//	action.cart.Purchase({ items, totalAmount, currency })
type PurchaseAction struct {
	piko.ActionMetadata
}

// PurchaseItem represents a single line item.
type PurchaseItem struct {
	ProductID string  `json:"productId"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
}

// PurchaseInput is the checkout payload.
type PurchaseInput struct {
	Items       []PurchaseItem `json:"items" validate:"required,min=1"`
	TotalAmount string         `json:"totalAmount" validate:"required"`
	Currency    string         `json:"currency" validate:"required"`
}

// PurchaseResponse is the order confirmation.
type PurchaseResponse struct {
	OrderID       string `json:"orderId"`
	ChargedAmount string `json:"chargedAmount"`
	ItemCount     int    `json:"itemCount"`
}

// Call processes the purchase and records analytics.
func (a PurchaseAction) Call(input PurchaseInput) (PurchaseResponse, error) {
	orderID := fmt.Sprintf("ORD-%s", time.Now().Format("20060102-150405"))

	revenue := maths.NewMoneyFromString(input.TotalAmount, input.Currency)
	piko.SetAnalyticsRevenue(a.Ctx(), revenue)
	piko.SetAnalyticsEventName(a.Ctx(), "purchase")
	piko.AddAnalyticsProperty(a.Ctx(), "order_id", orderID)
	piko.AddAnalyticsProperty(a.Ctx(), "item_count", fmt.Sprintf("%d", len(input.Items)))
	piko.AddAnalyticsProperty(a.Ctx(), "user_id", "user-alex-demo")

	for index, item := range input.Items {
		key := fmt.Sprintf("item_%d", index)
		piko.AddAnalyticsProperty(a.Ctx(), key, fmt.Sprintf("%s x%d", item.Name, item.Quantity))
	}

	chargedAmount := input.TotalAmount

	return PurchaseResponse{
		OrderID:       orderID,
		ChargedAmount: chargedAmount,
		ItemCount:     len(input.Items),
	}, nil
}
