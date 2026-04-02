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

package browser_provider_chromedp

import (
	"context"
	"fmt"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

// SetExtraHTTPHeaders sets additional HTTP headers that will be sent with all
// requests. These headers are added to all subsequent requests until cleared
// or changed.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
// Takes headers (map[string]string) which contains header name to value pairs.
//
// Returns error when the network domain cannot be enabled or headers cannot
// be set.
func SetExtraHTTPHeaders(ctx *ActionContext, headers map[string]string) error {
	networkHeaders := make(network.Headers, len(headers))
	for k, v := range headers {
		networkHeaders[k] = v
	}

	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		if err := network.Enable().Do(ctx2); err != nil {
			return fmt.Errorf("enabling network: %w", err)
		}
		return network.SetExtraHTTPHeaders(networkHeaders).Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("setting extra HTTP headers: %w", err)
	}
	return nil
}

// SetAuthorizationHeader sets an Authorization header for all requests.
// This is a convenience function for Bearer token authentication.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes token (string) which is the Bearer token value.
//
// Returns error when the header cannot be set.
func SetAuthorizationHeader(ctx *ActionContext, token string) error {
	return SetExtraHTTPHeaders(ctx, map[string]string{
		"Authorization": "Bearer " + token,
	})
}

// SetBasicAuthHeader sets a Basic Authorization header for all requests.
// The credentials should be in "username:password" format.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes credentials (string) which specifies the username:password value.
//
// Returns error when the header cannot be set.
func SetBasicAuthHeader(ctx *ActionContext, credentials string) error {
	return SetExtraHTTPHeaders(ctx, map[string]string{
		"Authorization": "Basic " + credentials,
	})
}

// SetUserAgent sets the User-Agent header for all requests.
// For full device emulation, use the EmulateDevice functions instead.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes userAgent (string) which specifies the User-Agent header value.
//
// Returns error when the user agent cannot be set.
func SetUserAgent(ctx *ActionContext, userAgent string) error {
	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return emulation.SetUserAgentOverride(userAgent).Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("setting user agent: %w", err)
	}
	return nil
}

// ClearExtraHTTPHeaders clears all extra HTTP headers by setting an empty map.
//
// Takes ctx (*ActionContext) which provides the browser action context.
//
// Returns error when the headers cannot be cleared.
func ClearExtraHTTPHeaders(ctx *ActionContext) error {
	return SetExtraHTTPHeaders(ctx, map[string]string{})
}

// SetAcceptLanguageHeader sets the Accept-Language header for all requests.
// Use it to test localisation.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes language (string) which specifies the language code to use.
//
// Returns error when setting the HTTP header fails.
func SetAcceptLanguageHeader(ctx *ActionContext, language string) error {
	return SetExtraHTTPHeaders(ctx, map[string]string{
		"Accept-Language": language,
	})
}

// SetCustomHeader sets a single custom header.
// Use it when only one header is needed.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes name (string) which specifies the header name.
// Takes value (string) which specifies the header value.
//
// Returns error when the header cannot be set.
func SetCustomHeader(ctx *ActionContext, name, value string) error {
	return SetExtraHTTPHeaders(ctx, map[string]string{
		name: value,
	})
}

// GetResponseHeaders returns the response headers from the most recent
// navigation. This requires the network domain to be enabled first.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
//
// Returns map[string]string which contains the response headers as key-value
// pairs.
// Returns error when the JavaScript evaluation fails.
func GetResponseHeaders(ctx *ActionContext) (map[string]string, error) {
	js := scripts.MustGet("get_response_headers.js")

	var result map[string]any
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("getting response headers: %w", err)
	}

	headers := make(map[string]string, len(result))
	for k, v := range result {
		if str, ok := v.(string); ok {
			headers[k] = str
		}
	}
	return headers, nil
}
