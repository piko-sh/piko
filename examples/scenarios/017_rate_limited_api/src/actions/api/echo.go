package api

import (
	"piko.sh/piko"
)

type EchoInput struct {
	Message string `json:"message"`
}

type EchoOutput struct {
	Message string `json:"message"`
}

// EchoAction is a simple echo endpoint with rate limiting.
// See https://piko.sh/docs/guide/advanced-actions#rate-limiting
type EchoAction struct {
	piko.ActionMetadata
}

func (a *EchoAction) Call(input EchoInput) (EchoOutput, error) {
	return EchoOutput{Message: input.Message}, nil
}

// RateLimit restricts this action to 3 requests/min per IP.
// Exceeding the limit returns HTTP 429 with Retry-After headers.
func (a *EchoAction) RateLimit() *piko.RateLimit {
	return &piko.RateLimit{
		KeyFunc:           piko.RateLimitByIP,
		RequestsPerMinute: 3,
		BurstSize:         3,
	}
}
