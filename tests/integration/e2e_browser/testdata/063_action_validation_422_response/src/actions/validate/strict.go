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
package validate

import (
	"regexp"

	"piko.sh/piko"
)

type StrictAction struct {
	piko.ActionMetadata
}

type StrictInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

type StrictResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func (a StrictAction) Call(input StrictInput) (StrictResponse, error) {
	errors := make(map[string]string)

	if input.Name == "" {
		errors["name"] = "name is required"
	} else if len(input.Name) < 3 {
		errors["name"] = "name must be at least 3 characters"
	}

	if input.Email == "" {
		errors["email"] = "email is required"
	} else if !emailRegex.MatchString(input.Email) {
		errors["email"] = "invalid email format"
	}

	if input.Age < 18 {
		errors["age"] = "age must be at least 18"
	} else if input.Age > 120 {
		errors["age"] = "age must be at most 120"
	}

	if len(errors) > 0 {
		return StrictResponse{}, piko.NewValidationError(errors)
	}

	return StrictResponse{
		Valid:   true,
		Message: "Validation passed",
	}, nil
}
