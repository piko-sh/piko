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
package arguments

import (
	"encoding/json"

	"piko.sh/piko"
)

type TestAction struct {
	piko.ActionMetadata
}

type TestInput struct {
	Name  string `json:"name" validate:"required"`
	Value int    `json:"value" validate:"required"`
}

type TestResponse struct {
	Name    string `json:"name"`
	Value   int    `json:"value"`
	RawArgs string `json:"rawArgs"`
}

func (a TestAction) Call(input TestInput) (TestResponse, error) {
	rawBytes, _ := json.Marshal(input)

	return TestResponse{
		Name:    input.Name,
		Value:   input.Value,
		RawArgs: string(rawBytes),
	}, nil
}
