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

package record

import "piko.sh/piko"

type SaveInput struct {
	FullName string `json:"full name"`
	Kind     string `json:"my-kind"`
}

type SaveOutput struct {
	Saved   bool   `json:"saved"`
	Version string `json:"2fa-version"`
}

// SaveAction carries JSON tags that are not identifiers, so the generated TypeScript has
// to quote them rather than emit them bare.
type SaveAction struct {
	piko.ActionMetadata
}

func (a *SaveAction) Call(input SaveInput) (SaveOutput, error) {
	return SaveOutput{Saved: input.FullName != "", Version: input.Kind}, nil
}
