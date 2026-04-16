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

package hmac_challenge

import (
	"net/http"

	"piko.sh/piko/internal/json"
)

// challengeResponse is the JSON body returned by the challenge endpoint.
type challengeResponse struct {
	// Token is the base64-encoded HMAC challenge token.
	Token string `json:"token"`
}

// ChallengeHandler returns an HTTP handler that generates HMAC challenge tokens.
// The handler responds to GET requests with a JSON body containing the token.
//
// Query parameters:
//   - action: the action name for the challenge
//
// Returns http.Handler which generates challenge tokens.
func (p *provider) ChallengeHandler() http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			http.Error(responseWriter, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		action := request.URL.Query().Get("action")

		if len(action) > maxActionLength {
			http.Error(responseWriter, "action name too long", http.StatusBadRequest)
			return
		}

		token, err := p.GenerateChallenge(action)
		if err != nil {
			http.Error(responseWriter, "failed to generate challenge", http.StatusInternalServerError)
			return
		}

		encoded, encodeErr := json.Marshal(challengeResponse{Token: token})
		if encodeErr != nil {
			http.Error(responseWriter, "failed to encode response", http.StatusInternalServerError)
			return
		}

		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.Header().Set("Cache-Control", "no-store")
		_, _ = responseWriter.Write(encoded)
	})
}
