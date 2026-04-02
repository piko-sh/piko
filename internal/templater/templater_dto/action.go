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

package templater_dto

import (
	"reflect"

	"piko.sh/piko/internal/json"
)

// ActionArgument represents a single argument passed to a client-side action. It
// encapsulates the value and its type for proper serialisation.
//
// Type values:
//   - "s": Static literal value (string, number, boolean)
//   - "v": Variable expression evaluated at render time
//   - "e": Event placeholder ($event) - Value is nil, browser event injected at
//     runtime
//   - "f": Form placeholder ($form) - Value is nil, form data injected at
//     runtime
type ActionArgument struct {
	// Value holds the argument's data in its native type.
	// For type "e" (event) or "f" (form) placeholders, this is nil.
	Value any `json:"v,omitempty"`

	// Type specifies the argument type identifier: "s" (static), "v" (variable),
	// "e" (event), or "f" (form).
	Type string `json:"t"`
}

// ActionPayload represents a request to run a function on the client side.
// It holds the function name and the arguments to pass to it.
type ActionPayload struct {
	// Function is the full name of the function where the action happened.
	Function string `json:"f"`

	// Args holds the arguments to pass to the action.
	Args []ActionArgument `json:"a"`
}

func init() {
	_ = json.Pretouch(reflect.TypeFor[ActionPayload]())
	_ = json.Pretouch(reflect.TypeFor[ActionArgument]())
}
