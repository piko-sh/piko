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

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

// Package captcha provides the public API for captcha verification in
// Piko.
//
// This is a facade that re-exports types from internal packages,
// providing a stable import path for application developers.
//
// # Usage
//
//	err := captcha.Verify(ctx, token, remoteIP, action)
//	if err != nil {
//	    // verification failed
//	}
//
// # Providers
//
// A built-in HMAC challenge provider is included for development and
// testing. External providers are available in the captcha_provider_*
// sub-packages:
//
//   - [captcha_provider_turnstile]: Cloudflare Turnstile
//   - [captcha_provider_recaptcha_v3]: Google reCAPTCHA v3
//   - [captcha_provider_hcaptcha]: hCaptcha
//   - [captcha_provider_hmac_challenge]: Built-in HMAC challenge (testing only)
//
// # Quick start
//
// Register a provider and start the server:
//
//	import (
//	    "piko.sh/piko"
//	    "piko.sh/piko/wdk/captcha/captcha_provider_turnstile"
//	)
//
//	provider, _ := captcha_provider_turnstile.NewProvider(captcha_provider_turnstile.Config{
//	    SiteKey:   "your-site-key",
//	    SecretKey: "your-secret-key",
//	})
//
//	server := piko.New(
//	    piko.WithCaptchaProvider("turnstile", provider),
//	    piko.WithDefaultCaptchaProvider("turnstile"),
//	)
//
// # Reading scores in an action
//
// After verification, inspect the normalised score to decide how to
// proceed:
//
//	service := captcha.GetDefaultService()
//	response, err := service.Verify(ctx, captcha.VerifyRequest{
//	    Token:  token,
//	    Action: "login",
//	})
//	if err != nil { return err }
//
//	switch {
//	case response.Score >= 0.7:
//	    // allow the action
//	case response.Score >= 0.3:
//	    // require additional verification
//	default:
//	    // block the request
//	}
//
// # Thread safety
//
// The captcha service and its methods are safe for concurrent use.
package captcha
