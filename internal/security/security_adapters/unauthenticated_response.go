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

package security_adapters

import (
	"net/http"
	"slices"
	"strings"

	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/security/security_dto"
)

const (
	// mediaTypeEventStream is the media type a client asks for to receive a stream.
	mediaTypeEventStream = "text/event-stream"

	// mediaTypeJSON is the media type a machine caller asks for.
	mediaTypeJSON = "application/json"

	// mediaTypeHTML is the media type a browser navigation asks for.
	mediaTypeHTML = "text/html"

	// headerSecFetchDest states what the browser will do with the response. A page cannot
	// forge it, which is why it is consulted first.
	headerSecFetchDest = "Sec-Fetch-Dest"

	// headerRequestedWith is the long-standing convention for marking a scripted request.
	headerRequestedWith = "X-Requested-With"

	// requestedWithXMLHTTP is the value that convention carries.
	requestedWithXMLHTTP = "xmlhttprequest"

	// unauthenticatedBodyFallback is sent when the response body cannot be encoded, so a
	// caller still receives valid JSON.
	unauthenticatedBodyFallback = `{"error":"unauthenticated","message":"Authentication required"}`
)

var (
	// documentFetchDestinations lists the Sec-Fetch-Dest values that mean the response will
	// be shown to a person as a page, and so should redirect rather than return a status.
	documentFetchDestinations = []string{"document", "iframe", "frame", "object", "embed"}
)

// unauthenticatedResponse is the body a machine caller receives, naming where to
// authenticate so a client can act on it without scraping a redirect.
type unauthenticatedResponse struct {
	// Error is the stable machine-readable code.
	Error string `json:"error"`

	// Message is the human-readable explanation.
	Message string `json:"message"`

	// Login is the URL the caller should send a person to.
	Login string `json:"login"`
}

// RespondUnauthenticated answers a request that failed authentication.
//
// Takes writer (http.ResponseWriter) which receives the response.
// Takes request (*http.Request) which supplies the headers the decision is made from.
// Takes auth (daemon_dto.AuthContext) which is handed to a custom handler, and may be
// nil.
// Takes config (daemon_dto.AuthGuardConfig) which supplies the login path and parameter.
func RespondUnauthenticated(
	writer http.ResponseWriter,
	request *http.Request,
	auth daemon_dto.AuthContext,
	config daemon_dto.AuthGuardConfig,
) {
	if config.OnUnauthenticated != nil {
		config.OnUnauthenticated(writer, request, auth)

		return
	}

	loginPath := config.LoginPath
	if loginPath == "" {
		loginPath = defaultLoginPath
	}

	redirectParam := config.RedirectParam
	if redirectParam == "" {
		redirectParam = defaultRedirectParam
	}

	loginURL := buildLoginRedirect(loginPath, redirectParam, request)

	if !wantsMachineReadableResponse(request) {
		http.Redirect(writer, request, loginURL, http.StatusSeeOther)

		return
	}

	writeUnauthenticatedJSON(writer, loginURL)
}

// wantsMachineReadableResponse reports whether a caller wants a status code rather than a
// redirect to a login page.
//
// Takes request (*http.Request) which supplies the headers.
//
// Returns bool which is true when the caller should receive a status rather than a
// redirect.
func wantsMachineReadableResponse(request *http.Request) bool {
	switch destination := strings.ToLower(request.Header.Get(headerSecFetchDest)); {
	case slices.Contains(documentFetchDestinations, destination):
		return false
	case destination == "empty":
		return true
	}

	if strings.EqualFold(request.Header.Get(headerRequestedWith), requestedWithXMLHTTP) {
		return true
	}

	accept := request.Header.Get("Accept")
	if security_dto.AcceptsMediaType(accept, mediaTypeEventStream) {
		return true
	}
	if security_dto.AcceptsMediaType(accept, mediaTypeJSON) &&
		!security_dto.AcceptsMediaType(accept, mediaTypeHTML) {
		return true
	}

	return strings.HasPrefix(request.Header.Get("Content-Type"), mediaTypeJSON)
}

// writeUnauthenticatedJSON sends the 401 body a machine caller receives.
//
// Takes writer (http.ResponseWriter) which receives the response.
// Takes loginURL (string) which names where a person should be sent to authenticate.
func writeUnauthenticatedJSON(writer http.ResponseWriter, loginURL string) {
	writer.Header().Set("Content-Type", mediaTypeJSON)
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(http.StatusUnauthorized)

	body, err := json.ConfigDefault.Marshal(unauthenticatedResponse{
		Error:   "unauthenticated",
		Message: "Authentication required",
		Login:   loginURL,
	})
	if err != nil {
		_, _ = writer.Write([]byte(unauthenticatedBodyFallback))

		return
	}

	_, _ = writer.Write(body)
}
