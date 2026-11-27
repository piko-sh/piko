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

package render_domain

import (
	"strings"

	"piko.sh/piko/internal/daemon/daemon_dto"
)

const (
	// scriptStartTag is how every framework-built script element starts.
	scriptStartTag = "<script "

	// modulePreloadStartTag is how every framework-built module preload link starts.
	modulePreloadStartTag = `<link rel="modulepreload" `
)

// cspToken returns the Content Security Policy token of the request being rendered.
//
// Returns string which is the token, or empty when the policy uses none or the render has
// no request.
func (rctx *renderContext) cspToken() string {
	if rctx.httpRequest == nil {
		return ""
	}
	return daemon_dto.CSPTokenFromContext(rctx.httpRequest.Context())
}

// withCSPToken adds the CSP token to every script element and module preload link in
// markup the framework built, so a token-based policy lets them load.
//
// The markup must come from the framework's own tag writers, which start every such
// element in the same way. Page content must never pass through here.
//
// Takes markup (string) which holds the framework-built elements.
// Takes token (string) which is the request's CSP token, or empty.
//
// Returns string which is markup with the token added, or markup unchanged when the token
// or the markup is empty.
func withCSPToken(markup, token string) string {
	if token == "" || markup == "" {
		return markup
	}
	attribute := daemon_dto.CSPTokenAttribute(token)
	replacer := strings.NewReplacer(
		scriptStartTag, insertAfterTagStart(scriptStartTag, attribute),
		modulePreloadStartTag, insertAfterTagStart(modulePreloadStartTag, attribute),
	)
	return replacer.Replace(markup)
}

// insertAfterTagStart places an attribute after the opening of a start tag, keeping the
// space that separates it from the attributes that follow.
//
// Takes tagStart (string) which is the opening of the start tag, ending in a space.
// Takes attribute (string) which starts with a space.
//
// Returns string which is the tag opening with the attribute added.
func insertAfterTagStart(tagStart, attribute string) string {
	return strings.TrimSuffix(tagStart, " ") + attribute + " "
}
