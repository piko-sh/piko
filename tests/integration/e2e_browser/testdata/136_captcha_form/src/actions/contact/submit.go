package contact

import "piko.sh/piko"

type SubmitAction struct {
	piko.ActionMetadata
}

func (SubmitAction) CaptchaConfig() *piko.CaptchaConfig {
	return new(piko.CaptchaConfig)
}

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

func (a SubmitAction) Call(input SubmitInput) (SubmitResponse, error) {
	return SubmitResponse{
		Name:    input.Name,
		Email:   input.Email,
		Message: input.Message,
	}, nil
}
