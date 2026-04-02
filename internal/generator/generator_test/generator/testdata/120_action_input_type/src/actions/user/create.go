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

package user

import "piko.sh/piko"

type CreateInput struct {
	Email     string `json:"email" validate:"required,email"`
	FirstName string `json:"firstName" validate:"required"`
	LastName  string `json:"lastName" validate:"required"`
	Age       int    `json:"age" validate:"gte=0,lte=150"`
}

type CreateOutput struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Age       int    `json:"age"`
}

type CreateAction struct {
	piko.ActionMetadata
}

func (a *CreateAction) Call(input CreateInput) (CreateOutput, error) {
	return CreateOutput{
		ID:        "usr_123",
		Email:     input.Email,
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Age:       input.Age,
	}, nil
}
