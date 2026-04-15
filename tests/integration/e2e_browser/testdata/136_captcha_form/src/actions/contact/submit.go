package contact

import "piko.sh/piko"

// SubmitAction handles captcha-protected contact form submissions.
type SubmitAction struct {
	piko.ActionMetadata
}

// CaptchaConfig marks this action as requiring captcha verification.
func (SubmitAction) CaptchaConfig() *piko.CaptchaConfig {
	return new(piko.CaptchaConfig)
}

// SubmitInput defines the expected form fields.
type SubmitInput struct {
	Name    string `json:"name" validate:"required"`
	Email   string `json:"email" validate:"required,email"`
	Message string `json:"message" validate:"required"`
}

// SubmitResponse is returned to the client on success.
type SubmitResponse struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

// Call receives the validated input and returns the response.
func (a SubmitAction) Call(input SubmitInput) (SubmitResponse, error) {
	return SubmitResponse{
		Name:    input.Name,
		Email:   input.Email,
		Message: input.Message,
	}, nil
}
