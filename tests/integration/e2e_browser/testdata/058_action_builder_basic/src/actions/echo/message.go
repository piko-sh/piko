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
package echo

import "piko.sh/piko"

type MessageAction struct {
	piko.ActionMetadata
}

type MessageInput struct {
	Text   string `json:"text" validate:"required"`
	Repeat int    `json:"repeat"`
}

type MessageResponse struct {
	Echo  string `json:"echo"`
	Count int    `json:"count"`
}

func (a MessageAction) Call(input MessageInput) (MessageResponse, error) {
	repeat := input.Repeat
	if repeat < 1 {
		repeat = 1
	}
	return MessageResponse{
		Echo:  input.Text,
		Count: repeat,
	}, nil
}
