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
package notify

import "piko.sh/piko"

type SendAction struct {
	piko.ActionMetadata
}

type SendInput struct {
	Title string `json:"title" validate:"required"`
}

type SendResponse struct {
	Success bool `json:"success"`
}

func (a SendAction) Call(input SendInput) (SendResponse, error) {
	a.Response().AddHelper("showToast", "Notification sent!", "success")
	a.Response().AddHelper("updateBadge", "3")
	return SendResponse{
		Success: true,
	}, nil
}
