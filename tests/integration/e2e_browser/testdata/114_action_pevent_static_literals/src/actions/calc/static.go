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
package calc

import (
	"strconv"

	"piko.sh/piko"
)

// StaticAction tests passing fully static literal values as separate arguments.
type StaticAction struct {
	piko.ActionMetadata
}

// StaticResponse echoes back the received values.
type StaticResponse struct {
	ReceivedName  string `json:"received_name"`
	ReceivedCount int    `json:"received_count"`
	ReceivedFlag  bool   `json:"received_flag"`
}

// Call accepts three separate typed parameters with static literal values.
func (a StaticAction) Call(Name string, Count int, Flag bool) (StaticResponse, error) {
	a.Response().AddHelper("showResult",
		Name,
		strconv.Itoa(Count),
		strconv.FormatBool(Flag),
	)

	return StaticResponse{
		ReceivedName:  Name,
		ReceivedCount: Count,
		ReceivedFlag:  Flag,
	}, nil
}
