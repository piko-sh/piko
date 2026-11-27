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

import (
	"context"
	"html"
)

const (
	// cspTokenAttributeStart opens the HTML attribute that carries the CSP token. The
	// attribute name is fixed by the HTML standard.
	cspTokenAttributeStart = ` nonce="`

	// cspTokenAttributeEnd closes the HTML attribute that carries the CSP token.
	cspTokenAttributeEnd = `"`
)

// CSPTokenFromContext reads the request's Content Security Policy token from the
// PikoRequestCtx on the context.
//
// Returns string which is the token, or empty when the policy uses none or the context
// carries no PikoRequestCtx.
func CSPTokenFromContext(ctx context.Context) string {
	if pctx := PikoRequestCtxFromContext(ctx); pctx != nil {
		return pctx.CSPToken
	}
	return ""
}

// CSPTokenAttribute returns the CSP token as an HTML attribute with a leading space, for
// use inside a start tag.
//
// Takes token (string) which is the request's CSP token, or empty when the policy uses
// none.
//
// Returns string which is the attribute with the token HTML-escaped, or empty when the
// token is empty.
func CSPTokenAttribute(token string) string {
	if token == "" {
		return ""
	}
	return cspTokenAttributeStart + html.EscapeString(token) + cspTokenAttributeEnd
}
