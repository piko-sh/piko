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

type RegisterInput struct {
	Email    string  `json:"email" validate:"required,email"`
	Password string  `json:"password" validate:"required,min=8,max=128"`
	Username string  `json:"username" validate:"required,min=3,max=50,alphanum"`
	Age      int     `json:"age" validate:"gte=13,lte=150"`
	Website  *string `json:"website" validate:"omitempty,url"`
}

type RegisterOutput struct {
	UserID   string `json:"userId"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type RegisterAction struct {
	piko.ActionMetadata
}

func (a *RegisterAction) Call(input RegisterInput) (RegisterOutput, error) {
	return RegisterOutput{
		UserID:   "usr_new",
		Email:    input.Email,
		Username: input.Username,
	}, nil
}
