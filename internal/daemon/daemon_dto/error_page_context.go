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

package daemon_dto

import "context"

// ErrorPageContext carries error details into a custom error page's Render
// function. When a page returns an error (or a route is not found), the
// runtime injects this context before rendering the matching error page.
type ErrorPageContext struct {
	// Message is the human-readable error message safe for display to users.
	Message string

	// InternalMessage contains the full internal error details. This is only
	// populated in development mode (dev or dev-i) and is empty in production.
	InternalMessage string

	// OriginalPath is the request path that triggered the error.
	OriginalPath string

	// StatusCode is the HTTP status code for the error (e.g. 404, 500).
	StatusCode int
}

// ctxKeyErrorPage is the context key for ErrorPageContext.
// Retained as a fallback for non-request contexts (tests).
type ctxKeyErrorPage struct{}

// WithErrorPageContext stores the given ErrorPageContext on the
// PikoRequestCtx carrier if present, avoiding a context.WithValue
// allocation.
//
// Falls back to a standalone context value for non-request contexts
// (e.g. tests).
//
// Takes parent (context.Context) which is the original request
// context.
// Takes epc (ErrorPageContext) which contains the error details.
//
// Returns context.Context with the error page context attached.
func WithErrorPageContext(parent context.Context, epc ErrorPageContext) context.Context {
	if pctx := PikoRequestCtxFromContext(parent); pctx != nil {
		pctx.ErrorPage = &epc
		return parent
	}
	return context.WithValue(parent, ctxKeyErrorPage{}, epc)
}

// GetErrorPageContext retrieves the ErrorPageContext from the PikoRequestCtx
// carrier first, falling back to a standalone context value.
//
// Takes ctx (context.Context) which is the request context to inspect.
//
// Returns ErrorPageContext which contains the error details.
// Returns bool which indicates whether an error page context was found.
func GetErrorPageContext(ctx context.Context) (ErrorPageContext, bool) {
	if pctx := PikoRequestCtxFromContext(ctx); pctx != nil && pctx.ErrorPage != nil {
		return *pctx.ErrorPage, true
	}
	epc, ok := ctx.Value(ctxKeyErrorPage{}).(ErrorPageContext)
	return epc, ok
}
