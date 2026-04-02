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

package resource

import "piko.sh/piko"

type FetchInput struct {
	ID string `json:"id" validate:"required"`
}

type FetchOutput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Data string `json:"data"`
}

type FetchAction struct {
	piko.ActionMetadata
}

func (a *FetchAction) Method() piko.HTTPMethod {
	return piko.MethodGet
}

func (a *FetchAction) Call(input FetchInput) (FetchOutput, error) {
	return FetchOutput{
		ID:   input.ID,
		Name: "Sample Resource",
		Data: "Resource data content",
	}, nil
}
