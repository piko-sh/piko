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
package order

import "piko.sh/piko"

type SubmitAction struct {
	piko.ActionMetadata
}

type SubmitInput struct {
	ProductID string `json:"productId" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required,min=1"`
}

type SubmitResponse struct {
	OrderID   string `json:"orderId"`
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
	Status    string `json:"status"`
}

func (a SubmitAction) Call(input SubmitInput) (SubmitResponse, error) {
	return SubmitResponse{
		OrderID:   "ord_2024_001",
		ProductID: input.ProductID,
		Quantity:  input.Quantity,
		Status:    "confirmed",
	}, nil
}
