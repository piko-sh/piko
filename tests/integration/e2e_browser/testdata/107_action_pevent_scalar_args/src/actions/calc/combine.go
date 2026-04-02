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
	"piko.sh/piko"
)

// CombineAction echoes back three separately-passed scalar arguments so the
// test can verify that multiple arguments from a p-event are correctly delivered.
type CombineAction struct {
	piko.ActionMetadata
}

// CombineResponse returns the concatenated result.
type CombineResponse struct {
	Result string `json:"result"`
}

// Call accepts three separate string parameters (not a struct) and
// concatenates them. This tests the multi-param Call signature.
func (a CombineAction) Call(A string, B string, C string) (CombineResponse, error) {
	result := A + "-" + B + "-" + C
	a.Response().AddHelper("showResult", result)

	return CombineResponse{
		Result: result,
	}, nil
}
