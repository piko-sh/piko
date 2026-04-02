package dto

// ConfirmationEmailProps holds the data passed to the plain HTML confirmation
// email template.
type ConfirmationEmailProps struct {
	Name    string `prop:"name"`
	Email   string `prop:"email"`
	Message string `prop:"message"`
}

// NotificationEmailProps holds the data passed to the PML (Outlook-compatible)
// internal notification email template.
type NotificationEmailProps struct {
	Name    string `prop:"name"`
	Email   string `prop:"email"`
	Message string `prop:"message"`
}
