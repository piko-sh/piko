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
package step

import "piko.sh/piko"

type OneAction struct {
	piko.ActionMetadata
}

type OneInput struct{}

type OneResponse struct {
	Step int  `json:"step"`
	Next bool `json:"next"`
}

func (a OneAction) Call(_ OneInput) (OneResponse, error) {
	return OneResponse{
		Step: 1,
		Next: true,
	}, nil
}

type TwoAction struct {
	piko.ActionMetadata
}

type TwoInput struct{}

type TwoResponse struct {
	Step     int  `json:"step"`
	Complete bool `json:"complete"`
}

func (a TwoAction) Call(_ TwoInput) (TwoResponse, error) {
	return TwoResponse{
		Step:     2,
		Complete: true,
	}, nil
}
