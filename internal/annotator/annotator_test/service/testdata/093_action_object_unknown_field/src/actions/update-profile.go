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

package actions

import "piko.sh/piko"

type UpdateProfileInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UpdateProfileOutput struct {
	Success bool `json:"success"`
}

type UpdateProfileAction struct {
	piko.ActionMetadata
}

func (a *UpdateProfileAction) Call(input UpdateProfileInput) (UpdateProfileOutput, error) {
	return UpdateProfileOutput{Success: true}, nil
}
