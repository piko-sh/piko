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
	"strconv"

	"piko.sh/piko"
)

// ProfileAction tests an object literal with all static values (no state refs).
type ProfileAction struct {
	piko.ActionMetadata
}

// ProfileInput has mixed types from static literal values.
type ProfileInput struct {
	Name   string `json:"name"`
	Age    int    `json:"age"`
	Active bool   `json:"active"`
}

// ProfileResponse echoes back the received values.
type ProfileResponse struct {
	ReceivedName   string `json:"received_name"`
	ReceivedAge    int    `json:"received_age"`
	ReceivedActive bool   `json:"received_active"`
}

// Call echoes the static object fields back.
func (a ProfileAction) Call(input ProfileInput) (ProfileResponse, error) {
	a.Response().AddHelper("showResult",
		input.Name,
		strconv.Itoa(input.Age),
		strconv.FormatBool(input.Active),
	)

	return ProfileResponse{
		ReceivedName:   input.Name,
		ReceivedAge:    input.Age,
		ReceivedActive: input.Active,
	}, nil
}
