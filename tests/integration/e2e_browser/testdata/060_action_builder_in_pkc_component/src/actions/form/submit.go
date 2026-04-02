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
package form

import "piko.sh/piko"

type SubmitAction struct {
	piko.ActionMetadata
}

type SubmitInput struct {
	Name    string `json:"name" validate:"required"`
	Email   string `json:"email" validate:"required,email"`
	Message string `json:"message"`
}

type SubmitResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Summary string `json:"summary"`
}

func (a SubmitAction) Call(input SubmitInput) (SubmitResponse, error) {
	return SubmitResponse{
		Success: true,
		ID:      "sub_12345",
		Summary: "Received from " + input.Name,
	}, nil
}
