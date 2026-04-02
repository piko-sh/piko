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

package browser

import (
	"fmt"
	"time"

	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
)

// WaitForNetworkIdle waits until no network requests have been made for the
// specified duration.
//
// Takes idleDuration (time.Duration) which specifies how long the network must
// be idle before returning.
// Takes opts (...WaitOption) which provides optional behaviour controls.
//
// Returns *Page which allows method chaining.
func (p *Page) WaitForNetworkIdle(idleDuration time.Duration, opts ...WaitOption) *Page {
	config := defaultWaitConfig()
	for _, opt := range opts {
		opt(&config)
	}

	detail := idleDuration.String()
	p.beforeAction("WaitForNetworkIdle", detail)
	start := time.Now()
	err := browser_provider_chromedp.WaitForNetworkIdle(p.actionCtx(), idleDuration, config.timeout)
	p.afterAction("WaitForNetworkIdle", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("WaitForNetworkIdle(%s) failed: %v", idleDuration, err)
	}
	return p
}

// CheckRequestMade checks if a request matching the criteria was made.
//
// Takes matcher (browser_provider_chromedp.RequestMatcher) which specifies the
// criteria to match
// against recorded requests.
//
// Returns bool which is true if a matching request was found.
func (p *Page) CheckRequestMade(matcher browser_provider_chromedp.RequestMatcher) bool {
	made, err := browser_provider_chromedp.CheckRequestMade(p.actionCtx(), matcher)
	if err != nil {
		p.t.Fatalf("CheckRequestMade() failed: %v", err)
	}
	return made
}

// WaitForRequest waits for a request matching the criteria to be made.
//
// Takes matcher (browser_provider_chromedp.RequestMatcher) which specifies the
// criteria for
// matching requests.
// Takes opts (...WaitOption) which provides optional configuration such as
// timeout.
//
// Returns *browser_provider_chromedp.NetworkRequest which is the matched
// network request.
func (p *Page) WaitForRequest(matcher browser_provider_chromedp.RequestMatcher, opts ...WaitOption) *browser_provider_chromedp.NetworkRequest {
	config := defaultWaitConfig()
	for _, opt := range opts {
		opt(&config)
	}

	p.beforeAction("WaitForRequest", matcher.URLContains)
	start := time.Now()
	request, err := browser_provider_chromedp.WaitForRequest(p.actionCtx(), matcher, config.timeout)
	p.afterAction("WaitForRequest", matcher.URLContains, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("WaitForRequest() failed: %v", err)
	}
	return request
}

// InterceptRequest sets up a request interceptor to mock responses.
//
// Takes urlPattern (string) which specifies the URL pattern to intercept.
// Takes response (browser_provider_chromedp.MockResponse) which provides the
// mocked response.
//
// Returns *Page which allows method chaining.
func (p *Page) InterceptRequest(urlPattern string, response browser_provider_chromedp.MockResponse) *Page {
	p.beforeAction("InterceptRequest", urlPattern)
	start := time.Now()
	err := browser_provider_chromedp.InterceptRequest(p.actionCtx(), urlPattern, response)
	p.afterAction("InterceptRequest", urlPattern, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("InterceptRequest(%q) failed: %v", urlPattern, err)
	}
	return p
}

// RemoveRequestIntercept stops intercepting requests that match a URL pattern.
//
// Takes urlPattern (string) which specifies the URL pattern to stop
// intercepting.
//
// Returns *Page which allows method chaining.
func (p *Page) RemoveRequestIntercept(urlPattern string) *Page {
	p.beforeAction("RemoveRequestIntercept", urlPattern)
	start := time.Now()
	err := browser_provider_chromedp.RemoveRequestIntercept(p.actionCtx(), urlPattern)
	p.afterAction("RemoveRequestIntercept", urlPattern, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("RemoveRequestIntercept(%q) failed: %v", urlPattern, err)
	}
	return p
}

// ClearRequestIntercepts removes all request intercepts.
//
// Returns *Page which allows method chaining.
func (p *Page) ClearRequestIntercepts() *Page {
	p.beforeAction("ClearRequestIntercepts", "")
	start := time.Now()
	err := browser_provider_chromedp.ClearRequestIntercepts(p.actionCtx())
	p.afterAction("ClearRequestIntercepts", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("ClearRequestIntercepts() failed: %v", err)
	}
	return p
}

// SetExtraHTTPHeaders sets additional HTTP headers that will be sent with all
// requests. These headers are added to all subsequent requests until cleared
// or changed.
//
// Takes headers (map[string]string) which contains the header names and values
// to include in requests.
//
// Returns *Page which allows method chaining.
func (p *Page) SetExtraHTTPHeaders(headers map[string]string) *Page {
	p.beforeAction("SetExtraHTTPHeaders", fmt.Sprintf("%d headers", len(headers)))
	start := time.Now()
	err := browser_provider_chromedp.SetExtraHTTPHeaders(p.actionCtx(), headers)
	p.afterAction("SetExtraHTTPHeaders", fmt.Sprintf("%d headers", len(headers)), err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("SetExtraHTTPHeaders() failed: %v", err)
	}
	return p
}

// SetAuthorisationHeader sets a Bearer token in the Authorisation header for
// all requests.
//
// Takes token (string) which is the Bearer token value.
//
// Returns *Page which allows method chaining.
func (p *Page) SetAuthorisationHeader(token string) *Page {
	p.beforeAction("SetAuthorisationHeader", "Bearer ***")
	start := time.Now()
	err := browser_provider_chromedp.SetAuthorizationHeader(p.actionCtx(), token)
	p.afterAction("SetAuthorisationHeader", "Bearer ***", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("SetAuthorisationHeader() failed: %v", err)
	}
	return p
}

// SetBasicAuthHeader sets a Basic Authorization header for all requests.
//
// Takes credentials (string) which is the base64-encoded "username:password".
//
// Returns *Page which allows method chaining.
func (p *Page) SetBasicAuthHeader(credentials string) *Page {
	p.beforeAction("SetBasicAuthHeader", "Basic ***")
	start := time.Now()
	err := browser_provider_chromedp.SetBasicAuthHeader(p.actionCtx(), credentials)
	p.afterAction("SetBasicAuthHeader", "Basic ***", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("SetBasicAuthHeader() failed: %v", err)
	}
	return p
}

// SetUserAgent sets the User-Agent header for all requests.
//
// Takes userAgent (string) which specifies the User-Agent header value.
//
// Returns *Page which allows method chaining.
func (p *Page) SetUserAgent(userAgent string) *Page {
	p.beforeAction("SetUserAgent", userAgent)
	start := time.Now()
	err := browser_provider_chromedp.SetUserAgent(p.actionCtx(), userAgent)
	p.afterAction("SetUserAgent", userAgent, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("SetUserAgent(%q) failed: %v", userAgent, err)
	}
	return p
}

// ClearExtraHTTPHeaders clears all extra HTTP headers.
//
// Returns *Page which allows method chaining.
func (p *Page) ClearExtraHTTPHeaders() *Page {
	p.beforeAction("ClearExtraHTTPHeaders", "")
	start := time.Now()
	err := browser_provider_chromedp.ClearExtraHTTPHeaders(p.actionCtx())
	p.afterAction("ClearExtraHTTPHeaders", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("ClearExtraHTTPHeaders() failed: %v", err)
	}
	return p
}

// SetAcceptLanguageHeader sets the Accept-Language header for all requests.
// Useful for testing localisation.
//
// Takes language (string) which specifies the language preference to send.
//
// Returns *Page which allows method chaining.
func (p *Page) SetAcceptLanguageHeader(language string) *Page {
	p.beforeAction("SetAcceptLanguageHeader", language)
	start := time.Now()
	err := browser_provider_chromedp.SetAcceptLanguageHeader(p.actionCtx(), language)
	p.afterAction("SetAcceptLanguageHeader", language, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("SetAcceptLanguageHeader(%q) failed: %v", language, err)
	}
	return p
}

// SetCustomHeader sets a single custom header.
//
// Takes name (string) which specifies the header name.
// Takes value (string) which specifies the header value.
//
// Returns *Page which allows method chaining.
func (p *Page) SetCustomHeader(name, value string) *Page {
	detail := fmt.Sprintf("%s: %s", name, value)
	p.beforeAction("SetCustomHeader", detail)
	start := time.Now()
	err := browser_provider_chromedp.SetCustomHeader(p.actionCtx(), name, value)
	p.afterAction("SetCustomHeader", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("SetCustomHeader(%q, %q) failed: %v", name, value, err)
	}
	return p
}

// GetResponseHeaders returns limited response headers from the current
// document. Full headers require network monitoring; this returns basic
// document info.
//
// Returns map[string]string which contains the available response headers.
func (p *Page) GetResponseHeaders() map[string]string {
	headers, err := browser_provider_chromedp.GetResponseHeaders(p.actionCtx())
	if err != nil {
		p.t.Fatalf("GetResponseHeaders() failed: %v", err)
	}
	return headers
}

// SetupDialogAutoAccept sets up automatic acceptance of all dialogues.
//
// Takes promptText (string) which specifies the text to use when accepting
// prompt dialogues.
//
// Returns *Page which allows method chaining.
func (p *Page) SetupDialogAutoAccept(promptText string) *Page {
	p.beforeAction("SetupDialogAutoAccept", "")
	start := time.Now()

	if p.dialogHandler != nil {
		p.dialogHandler.Stop()
	}

	p.dialogHandler = browser_provider_chromedp.SetupDialogAutoAccept(p.actionCtx(), promptText)
	p.afterAction("SetupDialogAutoAccept", "", false, time.Since(start))
	return p
}

// SetupDialogAutoDismiss sets up automatic dismissal of all dialogues.
//
// Returns *Page which allows method chaining.
func (p *Page) SetupDialogAutoDismiss() *Page {
	p.beforeAction("SetupDialogAutoDismiss", "")
	start := time.Now()

	if p.dialogHandler != nil {
		p.dialogHandler.Stop()
	}

	p.dialogHandler = browser_provider_chromedp.SetupDialogAutoDismiss(p.actionCtx())
	p.afterAction("SetupDialogAutoDismiss", "", false, time.Since(start))
	return p
}

// HandleAlert accepts an alert dialog.
//
// Returns *Page which allows method chaining.
func (p *Page) HandleAlert() *Page {
	p.beforeAction("HandleAlert", "")
	start := time.Now()
	err := browser_provider_chromedp.HandleAlert(p.actionCtx())
	p.afterAction("HandleAlert", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("HandleAlert() failed: %v", err)
	}
	return p
}

// HandleConfirm responds to a confirm dialog.
//
// Takes accept (bool) which is true to accept or false to dismiss the dialog.
//
// Returns *Page which allows method chaining.
func (p *Page) HandleConfirm(accept bool) *Page {
	detail := fmt.Sprintf("accept=%v", accept)
	p.beforeAction("HandleConfirm", detail)
	start := time.Now()
	err := browser_provider_chromedp.HandleConfirm(p.actionCtx(), accept)
	p.afterAction("HandleConfirm", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("HandleConfirm(%v) failed: %v", accept, err)
	}
	return p
}

// HandlePrompt handles a prompt dialog.
//
// Takes accept (bool) which indicates whether to accept or dismiss the prompt.
// Takes text (string) which is the text to enter into the prompt input field.
//
// Returns *Page which allows method chaining for further actions.
func (p *Page) HandlePrompt(accept bool, text string) *Page {
	detail := fmt.Sprintf("accept=%v, text=%q", accept, text)
	p.beforeAction("HandlePrompt", detail)
	start := time.Now()
	err := browser_provider_chromedp.HandlePrompt(p.actionCtx(), accept, text)
	p.afterAction("HandlePrompt", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("HandlePrompt(%v, %q) failed: %v", accept, text, err)
	}
	return p
}

// DismissDialog closes any open dialog by clicking Cancel or No.
//
// Returns *Page which enables method chaining.
func (p *Page) DismissDialog() *Page {
	p.beforeAction("DismissDialog", "")
	start := time.Now()
	err := browser_provider_chromedp.DismissDialog(p.actionCtx())
	p.afterAction("DismissDialog", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("DismissDialog() failed: %v", err)
	}
	return p
}

// AcceptDialog clicks OK or Yes on any open dialog box.
//
// Returns *Page which allows method chaining.
func (p *Page) AcceptDialog() *Page {
	p.beforeAction("AcceptDialog", "")
	start := time.Now()
	err := browser_provider_chromedp.AcceptDialog(p.actionCtx())
	p.afterAction("AcceptDialog", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("AcceptDialog() failed: %v", err)
	}
	return p
}
