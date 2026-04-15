package turnstilefail

import "piko.sh/piko"

// SubmitAction handles captcha-protected form submissions using Cloudflare Turnstile (always fail).
type SubmitAction struct {
	piko.ActionMetadata
}

// CaptchaConfig marks this action as requiring captcha verification via the turnstile_fail provider.
func (SubmitAction) CaptchaConfig() *piko.CaptchaConfig {
	return &piko.CaptchaConfig{Provider: "turnstile_fail"}
}

// SubmitInput defines the expected form fields.
type SubmitInput struct {
	Name    string `json:"name" validate:"required"`
	Email   string `json:"email" validate:"required,email"`
	Message string `json:"message" validate:"required"`
}

// SubmitResponse is returned to the client on success.
type SubmitResponse struct {
	Name         string  `json:"name"`
	Email        string  `json:"email"`
	Message      string  `json:"message"`
	CaptchaScore float64 `json:"captchaScore"`
}

// Call receives the validated input and returns the response.
func (a SubmitAction) Call(input SubmitInput) (SubmitResponse, error) {
	score := 0.0
	if s := a.Request().CaptchaScore; s != nil {
		score = *s
	}

	return SubmitResponse{
		Name:         input.Name,
		Email:        input.Email,
		Message:      input.Message,
		CaptchaScore: score,
	}, nil
}
