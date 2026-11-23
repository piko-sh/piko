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

package runner

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

var (
	errAuthFixtureTimeout = errors.New("auth fixture: sessions did not overlap within the deadline")
)

func TestAuthContractsTraceAndControls(t *testing.T) {
	recipe := generate.New("auth-contracts", 7, 0).Expected.Auth
	require.NotNil(t, recipe, "auth-contracts dispatch did not attach a recipe")
	arrivals := make(chan string, 2)
	both := make(chan struct{})
	ctx, cancel := context.WithTimeoutCause(t.Context(), 3*time.Second, errAuthFixtureTimeout)
	defer cancel()
	go func() {
		var first, second string
		select {
		case first = <-arrivals:
		case <-ctx.Done():
			return
		}
		select {
		case second = <-arrivals:
		case <-ctx.Done():
			return
		}
		if first != second {
			close(both)
		}
	}()
	renewalUnavailable := true
	var ledger casefile.AuthLedger
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		identity, role, meeting, last := "guest", "visitor", "single", "none"
		if cookie, err := request.Cookie("publication_session"); err == nil {
			switch cookie.Value {
			case recipe.ReaderSession:
				identity, role = "subscriber-"+strings.TrimPrefix(cookie.Value, "reader-ticket-"), "reader"
			case recipe.EditorSession:
				identity, role = "publisher-"+strings.TrimPrefix(cookie.Value, "editor-ticket-"), "editor"
			default:
				assert.Failf(t, "unexpected test session", "%q", cookie.Value)
			}
		}
		if request.Method == http.MethodPost {
			writer.Header().Set("Content-Type", "application/json")
			var input map[string]string
			assert.NoError(t, json.NewDecoder(request.Body).Decode(&input))
			assert.Equal(t, "form-"+identity, input["_csrf_ephemeral_token"], "wrong form token for %s", identity)
			assert.Equal(t, "action-"+identity, request.Header.Get("X-CSRF-Action-Token"), "wrong action token for %s", identity)
			if request.URL.Path == "/_piko/actions/publication.ReadLedger" {
				assert.Equal(t, "editor", role, "audit reader must use the editor session")
				assert.NoError(t, json.NewEncoder(writer).Encode(ledger))
				return
			}
			assert.Equal(t, "/_piko/actions/publication.Publish", request.URL.Path, "unexpected route")
			if identity == "guest" {
				writer.WriteHeader(http.StatusUnauthorized)
				_, err := fmt.Fprint(writer, `{"error":"unauthenticated","login":"/publication/sign-in?next=%2F_piko%2Factions%2Fpublication.Publish"}`)
				assert.NoError(t, err)
				return
			}
			ledger.Calls++
			if role != "editor" {
				writer.WriteHeader(http.StatusForbidden)
				_, err := fmt.Fprint(writer, `{"code":"FORBIDDEN"}`)
				assert.NoError(t, err)
				return
			}
			ledger.Writes++
			ledger.Headline, ledger.Owner = input["Headline"], identity
			assert.NoError(t, json.NewEncoder(writer).Encode(map[string]string{"By": identity, "Headline": input["Headline"]}))
			return
		}
		page := strings.TrimPrefix(request.URL.Path, "/publication/")
		if request.URL.Query().Get("edition") == "renewal" && renewalUnavailable {
			renewalUnavailable = false
			http.Redirect(writer, request, "/publication/sign-in?next=%2Fpublication%2Fprofile%3Fedition%3Drenewal", http.StatusSeeOther)
			return
		}
		if page != "front" && identity == "guest" {
			http.Redirect(writer, request, "/publication/sign-in?next=%2Fpublication%2F"+page, http.StatusSeeOther)
			return
		}
		if page == "desk" && role != "editor" {
			http.Error(writer, "This desk is for editors", http.StatusForbidden)
			return
		}
		if page == "profile" {
			writer.Header().Set("Cache-Control", "private, no-store")
		}
		if page == "desk" {
			http.SetCookie(writer, &http.Cookie{Name: "publication_visit", Value: identity, Path: "/"})
		}
		if cookie, err := request.Cookie("publication_visit"); err == nil {
			last = cookie.Value
		}
		if request.URL.Query().Get("meeting") == "paired" {
			arrivals <- identity
			select {
			case <-both:
				meeting = "paired"
			case <-ctx.Done():
				assert.Fail(t, "sessions did not overlap", "%v", context.Cause(ctx))
				return
			}
		}
		_, err := fmt.Fprintf(writer, `<meta name="csrf-token" content="action-%s"><meta name="csrf-ephemeral" content="form-%s"><main id="publication" data-page="%s"><i id="user">%s</i><i id="role">%s</i><i id="meeting">%s</i><i id="last-visit">%s</i></main>`, identity, identity, page, identity, role, meeting, last)
		assert.NoError(t, err)
	})
	client := &http.Client{
		Transport: responseTransport(func(request *http.Request) (*http.Response, error) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			return recorder.Result(), nil
		}),
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	observed, err := observeAuth(ctx, client, "http://fixture.invalid", recipe)
	require.NoError(t, err)
	require.NoError(t, checkAuth(recipe, observed))
	require.NoError(t, compareAuth(observed, observed))
	for _, testCase := range []struct {
		mutate      func(*casefile.AuthResponse)
		step, field string
	}{
		{step: "provider-unavailable", field: "status", mutate: func(response *casefile.AuthResponse) { response.Status = http.StatusOK }},
		{step: "provider-recovered", field: "identity", mutate: func(response *casefile.AuthResponse) { response.User = "guest" }},
		{step: "guest-profile", field: "location", mutate: func(response *casefile.AuthResponse) { response.Location = "/publication/sign-in" }},
		{step: "reader-desk", field: "status", mutate: func(response *casefile.AuthResponse) { response.Status = http.StatusOK }},
		{step: "reader-desk", field: "issued-visit", mutate: func(response *casefile.AuthResponse) { response.IssuedVisit = []string{"unexpected"} }},
		{step: "reader-recovery", field: "identity", mutate: func(response *casefile.AuthResponse) { response.User = "guest" }},
		{step: "editor-desk", field: "issued-visit", mutate: func(response *casefile.AuthResponse) { response.IssuedVisit = nil }},
		{step: "editor-fresh", field: "previous-visit", mutate: func(response *casefile.AuthResponse) { response.PreviousVisit = "none" }},
		{step: "reader-fresh", field: "cache-control", mutate: func(response *casefile.AuthResponse) { response.CacheControl = "public" }},
		{step: "paired-editor", field: "overlap", mutate: func(response *casefile.AuthResponse) { response.Rendezvous = "single" }},
		{step: "paired-reader", field: "role", mutate: func(response *casefile.AuthResponse) { response.Role = "editor" }},
	} {
		t.Run(testCase.step+"/"+testCase.field, func(t *testing.T) {
			corrupt := *observed
			corrupt.Steps = append([]AuthObservation(nil), observed.Steps...)
			for index := range corrupt.Steps {
				row := &corrupt.Steps[index]
				if row.Name != testCase.step {
					continue
				}
				testCase.mutate(&row.Response)
				for _, err := range []error{checkAuth(recipe, &corrupt), compareAuth(observed, &corrupt)} {
					got, ok := errors.AsType[authMismatch](err)
					require.True(t, ok, "wrong failure: %v", err)
					require.Equal(t, "/auth/"+row.Name+"/"+row.Actor+"/"+testCase.field, got.FailureSubject())
				}
				return
			}
			require.Fail(t, "missing case")
		})
	}
	for _, testCase := range []struct {
		mutate      func(*AuthActionObservation)
		step, field string
	}{
		{step: "guest-publish", field: "handler-calls", mutate: func(action *AuthActionObservation) { action.Ledger.Calls++ }},
		{step: "reader-publish", field: "published-writes", mutate: func(action *AuthActionObservation) { action.Ledger.Writes++ }},
		{step: "editor-publish", field: "action-owner", mutate: func(action *AuthActionObservation) { action.Result.By = "other" }},
		{step: "editor-publish", field: "stored-headline", mutate: func(action *AuthActionObservation) { action.Ledger.Headline = "" }},
		{step: "reader-after-publication", field: "stored-owner", mutate: func(action *AuthActionObservation) { action.Ledger.Owner = "reader" }},
	} {
		t.Run(testCase.step+"/"+testCase.field, func(t *testing.T) {
			corrupt := *observed
			corrupt.Actions = append([]AuthActionObservation(nil), observed.Actions...)
			for index := range corrupt.Actions {
				action := &corrupt.Actions[index]
				if action.Name != testCase.step {
					continue
				}
				testCase.mutate(action)
				for _, err := range []error{checkAuth(recipe, &corrupt), compareAuth(observed, &corrupt)} {
					got, ok := errors.AsType[authMismatch](err)
					require.True(t, ok, "wrong failure: %v", err)
					require.Equal(t, "/auth/"+action.Name+"/"+action.Actor+"/"+testCase.field, got.FailureSubject())
				}
				return
			}
			require.Fail(t, "missing action")
		})
	}
}
