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

type CreateAction struct {
	piko.ActionMetadata
}

type Address struct {
	Street   string `json:"street"`
	City     string `json:"city"`
	Postcode string `json:"postcode"`
}

type CreateInput struct {
	Product         string  `json:"product"`
	Quantity        int     `json:"quantity"`
	ShippingAddress Address `json:"shippingAddress"`
}

type CreateResponse struct {
	Product  string `json:"product"`
	Quantity int    `json:"quantity"`
	Street   string `json:"street"`
	City     string `json:"city"`
	Postcode string `json:"postcode"`
}

func (a CreateAction) Call(input CreateInput) (CreateResponse, error) {
	return CreateResponse{
		Product:  input.Product,
		Quantity: input.Quantity,
		Street:   input.ShippingAddress.Street,
		City:     input.ShippingAddress.City,
		Postcode: input.ShippingAddress.Postcode,
	}, nil
}
