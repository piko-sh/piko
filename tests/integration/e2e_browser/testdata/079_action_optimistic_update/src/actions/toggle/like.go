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
package toggle

import (
	"errors"
	"time"

	"piko.sh/piko"
)

type LikeAction struct {
	piko.ActionMetadata
}

type LikeInput struct {
	CurrentCount int `json:"current_count"`
}

type LikeResponse struct {
	Liked bool `json:"liked"`
	Count int  `json:"count"`
}

func (a LikeAction) Call(input LikeInput) (LikeResponse, error) {
	time.Sleep(500 * time.Millisecond)
	return LikeResponse{
		Liked: true,
		Count: input.CurrentCount + 1,
	}, nil
}

type FailLikeAction struct {
	piko.ActionMetadata
}

type FailLikeInput struct {
	CurrentCount int `json:"current_count"`
}

type FailLikeResponse struct {
	Liked bool `json:"liked"`
	Count int  `json:"count"`
}

func (a FailLikeAction) Call(_ FailLikeInput) (FailLikeResponse, error) {
	time.Sleep(500 * time.Millisecond)
	return FailLikeResponse{}, errors.New("like service unavailable")
}
