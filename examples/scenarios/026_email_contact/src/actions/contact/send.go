package contact

import (
	"context"

	"piko.sh/piko"
	"piko.sh/piko/wdk/email"

	"testmodule/internal/dto"
)

// SendAction handles contact form submissions and sends two emails:
// a plain HTML confirmation to the user, and a PML (Outlook-compatible)
// notification to the site owner.
//
//	action.contact.Send($form)
type SendAction struct {
	piko.ActionMetadata
}

// SendInput defines the expected form fields.
type SendInput struct {
	Name    string `json:"name" validate:"required,min=1,max=200"`
	Email   string `json:"email" validate:"required,email"`
	Message string `json:"message" validate:"required,min=1,max=5000"`
}

// SendResponse is returned to the client after both emails are sent.
type SendResponse struct {
	Success bool   `json:"success"`
	Detail  string `json:"detail"`
}

// Call sends a confirmation email to the user and a notification email to the
// site owner. Both are rendered from .pk templates with typed props.
func (a SendAction) Call(input SendInput) (SendResponse, error) {
	ctx := a.Ctx()

	// Send the plain HTML confirmation to the user.
	if err := sendConfirmation(ctx, input); err != nil {
		return SendResponse{}, err
	}

	// Send the PML notification to the site owner.
	if err := sendNotification(ctx, input); err != nil {
		return SendResponse{}, err
	}

	return SendResponse{
		Success: true,
		Detail:  "Emails sent (check server terminal for both)",
	}, nil
}

func sendConfirmation(ctx context.Context, input SendInput) error {
	builder, err := email.NewTemplatedEmailBuilderFromDefault[dto.ConfirmationEmailProps]()
	if err != nil {
		return err
	}

	return builder.
		To(input.Email).
		Subject("Thanks for your message, " + input.Name).
		BodyTemplate("emails/confirmation.pk").
		Props(dto.ConfirmationEmailProps{
			Name:    input.Name,
			Email:   input.Email,
			Message: input.Message,
		}).
		Do(ctx)
}

func sendNotification(ctx context.Context, input SendInput) error {
	builder, err := email.NewTemplatedEmailBuilderFromDefault[dto.NotificationEmailProps]()
	if err != nil {
		return err
	}

	return builder.
		To("admin@example.com").
		Subject("New contact from " + input.Name).
		BodyTemplate("emails/notification.pk").
		Props(dto.NotificationEmailProps{
			Name:    input.Name,
			Email:   input.Email,
			Message: input.Message,
		}).
		Do(ctx)
}
