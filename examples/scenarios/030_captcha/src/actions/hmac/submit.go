package hmac

import "piko.sh/piko"

// SubmitAction handles form submissions verified by the built-in HMAC provider.
type SubmitAction struct {
	piko.ActionMetadata
}

// CaptchaConfig requires CAPTCHA verification.
//
// Returns *piko.CaptchaConfig which uses the example's default HMAC provider.
func (SubmitAction) CaptchaConfig() *piko.CaptchaConfig {
	return new(piko.CaptchaConfig)
}

// SubmitInput defines the expected form fields.
type SubmitInput struct {
	// Name is the sender's name.
	Name string `json:"name" validate:"required"`
	// Email is the sender's email address.
	Email string `json:"email" validate:"required,email"`
	// Message is the text submitted by the sender.
	Message string `json:"message" validate:"required"`
}

// SubmitResponse is returned to the client on success.
type SubmitResponse struct {
	// Name echoes the submitted name.
	Name string `json:"name"`
	// Email echoes the submitted email address.
	Email string `json:"email"`
	// Message echoes the submitted message.
	Message string `json:"message"`
	// CaptchaScore is the verification score, or zero when unavailable.
	CaptchaScore float64 `json:"captchaScore"`
}

// Call echoes the submission and includes its CAPTCHA score.
//
// Calling without request metadata causes a panic.
//
// Takes input (SubmitInput) which holds the validated form fields.
//
// Returns SubmitResponse which contains the submitted fields and verification score.
// Returns error which is always nil.
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
