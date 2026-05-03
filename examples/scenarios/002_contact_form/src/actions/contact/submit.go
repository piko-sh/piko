package contact

import "piko.sh/piko"

// SubmitAction handles contact form submissions.
// See https://piko.sh/docs/reference/server-actions
type SubmitAction struct {
	piko.ActionMetadata
}

// SubmitInput defines the expected form fields. The json tags map HTML
// field names to struct fields; validate tags are checked before Call runs.
type SubmitInput struct {
	Name    string `json:"name" validate:"required"`
	Email   string `json:"email" validate:"required,email"`
	Message string `json:"message" validate:"required"`
}

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
