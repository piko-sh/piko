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

// Package captcha_provider_hcaptcha provides an hCaptcha captcha
// provider.
//
// This provider delegates verification to hCaptcha, a privacy-focused
// captcha service. The free tier returns a pass/fail outcome. The
// Enterprise tier returns risk scores on an inverted scale (0.0 =
// safe, 1.0 = threat); this provider normalises those scores to the
// standard convention used across Piko (0.0 = bot, 1.0 = human) so
// that callers can apply a single threshold regardless of the
// upstream tier.
//
// Tokens are valid for 2 minutes and are single-use.
//
// Obtain a site key and secret key from https://www.hcaptcha.com/:
//
//	provider, err := captcha_provider_hcaptcha.NewProvider(captcha_provider_hcaptcha.Config{
//	    SiteKey:   "10000000-ffff-ffff-ffff-000000000001",
//	    SecretKey: "0x0000000000000000000000000000000000000000",
//	})
//
// # Thread safety
//
// All methods on the provider returned by [NewProvider] are safe for concurrent use.
package captcha_provider_hcaptcha
