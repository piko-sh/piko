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
//
// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package email

import (
	"fmt"
	"testmodule/internal/dto"
	"time"

	"piko.sh/piko"
)

// ContactAction handles contact form submissions from the About page.
type ContactAction struct {
	piko.ActionMetadata
}

// ContactInput contains the form fields submitted by the contact form.
type ContactInput struct {
	Name    string `json:"name" validate:"required"`
	Email   string `json:"email" validate:"required,email"`
	Reason  string `json:"reason" validate:"required"`
	Message string `json:"message" validate:"required"`
}

// ContactResponse is returned after processing the contact form.
type ContactResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// Call processes the contact form submission.
func (a ContactAction) Call(input ContactInput) (ContactResponse, error) {
	ctx := a.Ctx()
	submittedAt := time.Now().Format("2 January 2006 at 15:04")

	// Send internal notification email to The Norman Kitchen
	internalProps := dto.ContactInternalEmailProps{
		Name:        input.Name,
		Email:       input.Email,
		Reason:      formatReason(input.Reason),
		Message:     input.Message,
		SubmittedAt: submittedAt,
	}

	if err := sendContactInternalEmail(ctx, internalProps); err != nil {
		a.Response().AddHelper("showToast", "Failed to send your message. Please try again or send us a carrier pigeon.", "error")
		return ContactResponse{}, err
	}

	// Send confirmation email to the user
	confirmationProps := dto.ContactConfirmationEmailProps{
		Name: input.Name,
	}

	if err := sendContactConfirmationEmail(ctx, input.Email, confirmationProps); err != nil {
		// Log the error but don't fail - the internal email was sent successfully
		fmt.Printf("Warning: failed to send confirmation email to %s: %v\n", input.Email, err)
	}

	a.Response().AddHelper("showToast", "Whey to go! Your message has been sent. The mice are on it!", "success")
	a.Response().AddHelper("resetForm")

	return ContactResponse{Success: true, Message: "Contact form submitted successfully"}, nil
}

// formatReason converts the reason select value to a display-friendly format.
func formatReason(reason string) string {
	switch reason {
	case "general":
		return "General Enquiry"
	case "recipe":
		return "Recipe Suggestion"
	case "emergency":
		return "Cheese Emergency"
	case "bug":
		return "Bug Report"
	case "hello":
		return "Just Saying Hi"
	default:
		return reason
	}
}
