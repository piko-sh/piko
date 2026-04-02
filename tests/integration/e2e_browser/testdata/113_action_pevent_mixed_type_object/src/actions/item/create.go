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
package item

import (
	"strconv"

	"piko.sh/piko"
)

// CreateAction tests an object literal with mixed Go types (string, int, bool).
type CreateAction struct {
	piko.ActionMetadata
}

// CreateInput has string, int, and bool fields.
type CreateInput struct {
	Label  string `json:"label"`
	Count  int    `json:"count"`
	Active bool   `json:"active"`
}

// CreateResponse echoes back the received values.
type CreateResponse struct {
	ReceivedLabel  string `json:"received_label"`
	ReceivedCount  int    `json:"received_count"`
	ReceivedActive bool   `json:"received_active"`
}

// Call echoes the mixed-type fields back.
func (a CreateAction) Call(input CreateInput) (CreateResponse, error) {
	a.Response().AddHelper("showResult",
		input.Label,
		strconv.Itoa(input.Count),
		strconv.FormatBool(input.Active),
	)

	return CreateResponse{
		ReceivedLabel:  input.Label,
		ReceivedCount:  input.Count,
		ReceivedActive: input.Active,
	}, nil
}
