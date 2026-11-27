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

package daemon_adapters

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/render/render_dto"
)

const (
	// buildHeadersMaxParts is the maximum number of parts in a Link header (URL + rel + as +
	// type + crossorigin).
	buildHeadersMaxParts = 5

	// linkCSPTokenParameterStart opens the Link header parameter that carries the CSP token.
	// The parameter name is fixed by the HTML standard.
	linkCSPTokenParameterStart = `; nonce="`
	// asciiDelete is the DEL control character, which an HTTP quoted-string cannot hold.
	asciiDelete = 0x7f
)

var (
	// errUnsafeLinkCSPToken is returned when the CSP token holds a character that cannot sit
	// inside a quoted Link header parameter.
	errUnsafeLinkCSPToken = errors.New("CSP token cannot be quoted in a Link header")
)

// buildHeaders builds an HTTP Link header value from link header data.
//
// Takes lh (render_dto.LinkHeader) which contains the link URL, relation type, and
// optional attributes such as resource type and cross-origin settings.
//
// Returns string which is the formatted header value with parts joined by semicolons.
func buildHeaders(lh render_dto.LinkHeader) string {
	parts := make([]string, 0, buildHeadersMaxParts)
	parts = append(parts,
		fmt.Sprintf("<%s>", lh.URL),
		fmt.Sprintf("rel=%s", lh.Rel),
	)

	if lh.As != "" {
		parts = append(parts, fmt.Sprintf("as=%s", lh.As))
	}
	if lh.Type != "" {
		parts = append(parts, fmt.Sprintf("type=%q", lh.Type))
	}
	if lh.CrossOrigin == "anonymous" || lh.CrossOrigin == "use-credentials" {
		parts = append(parts, fmt.Sprintf("crossorigin=%s", lh.CrossOrigin))
	} else if lh.CrossOrigin != "" {
		parts = append(parts, "crossorigin")
	}
	return strings.Join(parts, "; ")
}

// sendSpecificEarlyHints sends HTTP 103 Early Hints with the specified link headers to
// enable browser preloading.
//
// Takes w (http.ResponseWriter) which receives the early hints response.
// Takes r (*http.Request) which provides the incoming request used to prepare early
// hints.
// Takes headersToSend ([]render_dto.LinkHeader) which specifies the link headers to send
// as early hints.
//
// Each hint carries the request's CSP token when the policy uses one, so a token-based
// policy lets the browser act on it.
//
// Returns bool which indicates whether the early hints were sent and flushed
// successfully.
func sendSpecificEarlyHints(w http.ResponseWriter, r *http.Request, headersToSend []render_dto.LinkHeader) bool {
	if len(headersToSend) == 0 {
		return false
	}

	ctx := r.Context()
	_, l := logger_domain.From(ctx, log)

	cspToken := daemon_dto.CSPTokenFromContext(ctx)
	var tokenErr error
	for _, lh := range headersToSend {
		value, err := withLinkCSPToken(buildHeaders(lh), cspToken)
		if err != nil {
			tokenErr = err
		}
		w.Header().Add(headerLink, value)
	}
	if tokenErr != nil {
		l.Warn("Sending Early Hints without the CSP token", logger_domain.Error(tokenErr))
	}

	l.Trace("Attempting to send 103 Early Hints", logger_domain.Int("count", len(headersToSend)))
	w.WriteHeader(http.StatusEarlyHints)

	flusher, ok := w.(http.Flusher)
	if !ok {
		l.Warn("ResponseWriter does not support flushing, 103 Early Hints might not be sent effectively.")
		return false
	}
	flusher.Flush()
	l.Trace("Flushed 103 Early Hints")
	return true
}

// withLinkCSPToken adds the CSP token to a Link header value as a quoted parameter.
//
// Takes value (string) which is the Link header value.
// Takes cspToken (string) which is the request's CSP token, or empty.
//
// Returns string which is the value with the token added, or value unchanged when the
// token is empty or cannot be quoted.
// Returns error when the token holds a character that would end the quoted parameter or
// the header line.
func withLinkCSPToken(value, cspToken string) (string, error) {
	if cspToken == "" {
		return value, nil
	}
	if !isQuotedStringText(cspToken) {
		return value, errUnsafeLinkCSPToken
	}
	return value + linkCSPTokenParameterStart + cspToken + `"`, nil
}

// isQuotedStringText reports whether text can sit inside an HTTP quoted-string without
// escaping. Quotes, backslashes and control characters other than tab cannot.
//
// Takes text (string) which is the text to check.
//
// Returns bool which is true when every byte is allowed.
func isQuotedStringText(text string) bool {
	for index := range len(text) {
		character := text[index]
		if character == '"' || character == '\\' || character == asciiDelete {
			return false
		}
		if character < ' ' && character != '\t' {
			return false
		}
	}
	return true
}
