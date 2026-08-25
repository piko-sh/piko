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

package reflect

import "piko.sh/piko"

type ScanInput struct {
	Target string `json:"target" validate:"required"`
}

type ScanOutput struct {
	Scanned string `json:"scanned"`
}

// ScanAction lives in a package called "reflect", which the generated registry also
// imports from the standard library to pretouch JSON types.
type ScanAction struct {
	piko.ActionMetadata
}

func (a *ScanAction) Call(input ScanInput) (ScanOutput, error) {
	return ScanOutput{Scanned: input.Target}, nil
}
