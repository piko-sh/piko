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
package feedback

import (
	"time"

	"piko.sh/piko"
)

type SubmitAction struct {
	piko.ActionMetadata
}

type SubmitInput struct {
	Name    string `json:"name" validate:"required"`
	Comment string `json:"comment" validate:"required"`
}

type SubmitResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
}

func (a SubmitAction) Call(input SubmitInput) (SubmitResponse, error) {
	time.Sleep(300 * time.Millisecond)
	a.Response().AddHelper("showToast", "Thank you for your feedback!", "success")
	return SubmitResponse{
		Success: true,
		ID:      "fb_123",
	}, nil
}
