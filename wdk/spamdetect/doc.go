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

// Package spamdetect provides the public API for spam detection in Piko.
//
// This is a facade that re-exports types from internal packages,
// providing a stable import path for application developers.
//
// # Usage
//
//	response, err := spamdetect.Check(ctx, &spamdetect.CheckRequest{
//	    Content:    messageBody,
//	    AuthorName: authorName,
//	})
//	if err != nil {
//	    // service error
//	}
//	if response.IsSpam {
//	    // reject submission
//	}
//
// # Providers
//
// A built-in rules engine is included for zero-dependency spam detection.
// It bundles honeypot detection, gibberish scoring, link density analysis,
// keyword blocklist matching, and submission timing checks.
//
// Register the built-in provider:
//
//	import (
//	    "piko.sh/piko"
//	    "piko.sh/piko/wdk/spamdetect/spamdetect_provider_builtin_rules"
//	)
//
//	provider, _ := spamdetect_provider_builtin_rules.NewProvider(
//	    spamdetect_provider_builtin_rules.Config{},
//	)
//
//	server := piko.New(
//	    piko.WithSpamDetectProvider("builtin_rules", provider),
//	)
//
// # Thread safety
//
// The spam detection service and its methods are safe for concurrent use.
package spamdetect
