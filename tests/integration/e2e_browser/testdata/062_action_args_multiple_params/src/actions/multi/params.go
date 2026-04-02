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
package multi

import (
	"fmt"

	"piko.sh/piko"
)

type ParamsAction struct {
	piko.ActionMetadata
}

type ParamsInput struct {
	StringVal string  `json:"stringVal" validate:"required"`
	IntVal    int     `json:"intVal"`
	FloatVal  float64 `json:"floatVal"`
	BoolVal   bool    `json:"boolVal"`
}

type ParamsResponse struct {
	StringVal string  `json:"stringVal"`
	IntVal    int     `json:"intVal"`
	FloatVal  float64 `json:"floatVal"`
	BoolVal   bool    `json:"boolVal"`
	Summary   string  `json:"summary"`
}

func (a ParamsAction) Call(input ParamsInput) (ParamsResponse, error) {
	return ParamsResponse{
		StringVal: input.StringVal,
		IntVal:    input.IntVal,
		FloatVal:  input.FloatVal,
		BoolVal:   input.BoolVal,
		Summary:   fmt.Sprintf("s=%s i=%d f=%.2f b=%t", input.StringVal, input.IntVal, input.FloatVal, input.BoolVal),
	}, nil
}
