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

// UpdateUserInput holds the input parameters for the update user action.
type UpdateUserInput struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// UpdateUserOutput holds the result of the update user action.
type UpdateUserOutput struct {
	Success bool `json:"success"`
}

// UpdateUserAction handles updating a user.
type UpdateUserAction struct {
	piko.ActionMetadata
}

// Call processes the update user request.
func (a *UpdateUserAction) Call(input UpdateUserInput) (UpdateUserOutput, error) {
	return UpdateUserOutput{Success: true}, nil
}
