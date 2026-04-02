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
	"context"
	"testmodule/internal/dto"
	"testmodule/internal/env"

	"piko.sh/piko/wdk/email"
)

// sendContactInternalEmail sends a notification email to The Norman Kitchen staff
// about a new contact form submission.
func sendContactInternalEmail(ctx context.Context, props dto.ContactInternalEmailProps) error {
	builder, err := email.NewTemplatedEmailBuilderFromDefault[dto.ContactInternalEmailProps]()
	if err != nil {
		return err
	}
	return builder.
		To(env.Get().NormanKitchenEmail).
		Subject("New Contact Form Submission - " + props.Name).
		BodyTemplate("emails/contact_internal.pk").
		Props(props).
		Do(ctx)
}

// sendContactConfirmationEmail sends a confirmation email to the user who submitted
// the contact form.
func sendContactConfirmationEmail(ctx context.Context, toEmail string, props dto.ContactConfirmationEmailProps) error {
	builder, err := email.NewTemplatedEmailBuilderFromDefault[dto.ContactConfirmationEmailProps]()
	if err != nil {
		return err
	}
	return builder.
		To(toEmail).
		Subject("Thank you for contacting The Norman Kitchen!").
		BodyTemplate("emails/contact_confirmation.pk").
		Props(props).
		Do(ctx)
}
