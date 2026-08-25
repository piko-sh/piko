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

package report

import "piko.sh/piko"

type FetchInput struct {
	Period string `json:"period" validate:"required"`
}

// FetchAction returns a bare Go map, which has no name a TypeScript interface could be
// declared under and has to be spelled out at each use site instead.
type FetchAction struct {
	piko.ActionMetadata
}

func (a *FetchAction) Call(input FetchInput) (map[string]any, error) {
	return map[string]any{"period": input.Period, "total": 42}, nil
}
