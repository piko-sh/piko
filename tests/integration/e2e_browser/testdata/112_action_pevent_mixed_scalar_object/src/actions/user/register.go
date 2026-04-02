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

// RegisterAction tests mixing a scalar argument with an object literal argument.
type RegisterAction struct {
	piko.ActionMetadata
}

// AddressInput is the struct for the object literal argument.
type AddressInput struct {
	City     string `json:"city"`
	Postcode string `json:"postcode"`
}

// RegisterResponse echoes back the received values.
type RegisterResponse struct {
	ReceivedName     string `json:"received_name"`
	ReceivedCity     string `json:"received_city"`
	ReceivedPostcode string `json:"received_postcode"`
}

// Call accepts a scalar string and a struct to test mixed argument types.
func (a RegisterAction) Call(Name string, Address AddressInput) (RegisterResponse, error) {
	a.Response().AddHelper("showResult", Name, Address.City, Address.Postcode)

	return RegisterResponse{
		ReceivedName:     Name,
		ReceivedCity:     Address.City,
		ReceivedPostcode: Address.Postcode,
	}, nil
}
