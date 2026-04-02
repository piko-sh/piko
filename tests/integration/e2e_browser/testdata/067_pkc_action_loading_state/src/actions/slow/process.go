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
package slow

import (
	"time"

	"piko.sh/piko"
)

type ProcessAction struct {
	piko.ActionMetadata
}

type ProcessInput struct {
	DelayMs int `json:"delayMs"`
}

type ProcessResponse struct {
	Completed bool   `json:"completed"`
	Message   string `json:"message"`
}

func (a ProcessAction) Call(input ProcessInput) (ProcessResponse, error) {
	delay := time.Duration(input.DelayMs) * time.Millisecond
	if delay > 0 {
		time.Sleep(delay)
	}
	return ProcessResponse{
		Completed: true,
		Message:   "Process completed",
	}, nil
}
