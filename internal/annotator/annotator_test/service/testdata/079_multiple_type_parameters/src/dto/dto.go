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

package dto

import (
	"testcase_79_multiple_type_parameters/fields"
)

type User struct {
	Name  fields.Text `json:"name"`
	Email fields.Text `json:"email"`
}

func (u User) FullInfo() string {
	return u.Name.String() + " <" + u.Email.String() + ">"
}

type Product struct {
	Title fields.Text `json:"title"`
	Price fields.Text `json:"price"`
}

func (p Product) Description() string {
	return p.Title.String() + " - " + p.Price.String()
}

type ErrorInfo struct {
	Code    int         `json:"code"`
	Message fields.Text `json:"message"`
}

func (e ErrorInfo) Display() string {
	return e.Message.String()
}

type PageData struct {
	UserProduct fields.Pair[User, Product]             `json:"user_product"`
	Mixed       fields.Triple[User, Product, ErrorInfo] `json:"mixed"`
	UserResult  fields.Result[User, ErrorInfo]          `json:"user_result"`
}
