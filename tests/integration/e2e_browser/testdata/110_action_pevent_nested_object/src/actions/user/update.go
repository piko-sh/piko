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
package user

import (
	"piko.sh/piko"
)

// UpdateAction tests nested struct binding from an object literal.
type UpdateAction struct {
	piko.ActionMetadata
}

// Address is a nested struct used inside UpdateInput.
type Address struct {
	City     string `json:"city"`
	Postcode string `json:"postcode"`
}

// UpdateInput contains a flat field and a nested struct.
type UpdateInput struct {
	Name    string  `json:"name"`
	Address Address `json:"address"`
}

// UpdateResponse echoes back the received values.
type UpdateResponse struct {
	ReceivedName     string `json:"received_name"`
	ReceivedCity     string `json:"received_city"`
	ReceivedPostcode string `json:"received_postcode"`
}

// Call echoes the nested input fields back.
func (a UpdateAction) Call(input UpdateInput) (UpdateResponse, error) {
	a.Response().AddHelper("showResult", input.Name, input.Address.City, input.Address.Postcode)

	return UpdateResponse{
		ReceivedName:     input.Name,
		ReceivedCity:     input.Address.City,
		ReceivedPostcode: input.Address.Postcode,
	}, nil
}
