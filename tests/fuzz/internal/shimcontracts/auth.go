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

// Package shimcontracts checks Piko's security middleware and captcha adapters against
// independently modelled contracts.
package shimcontracts

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"piko.sh/json"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/security/security_adapters"
)

const (
	// guestActor is the session of an unauthenticated visitor.
	guestActor = "guest"

	// authHistoryLimit bounds the replayed choice bytes appended to the full grid.
	authHistoryLimit = 24

	// publicRoutes counts the leading authRoutes that the guard must leave public.
	publicRoutes = 3

	// sessionHeader carries the fixture session to the authentication provider.
	sessionHeader = "X-Fixture-Session"
)

var (
	// errAuthOffline is the controlled authentication-provider outage.
	errAuthOffline = errors.New("local fixture unavailable")

	// authActors lists the fixture sessions.
	authActors = []string{guestActor, "reader", "editor", "pending", "offline"}

	// authLabels lists the identity each actor in authActors must expose.
	authLabels = []string{guestActor, "reader/reader", "editor/editor", "/pending", guestActor}

	// authRoutes lists the public paths followed by the protected paths.
	authRoutes = []string{"/news", "/static/logo", "/sign-in", "/desk?view=latest", "/staticx", "/static/../desk", "/static%2F..%2Fdesk"}

	// authClients lists the requesting client kinds.
	authClients = []authClient{
		{accept: "text/html", destination: "document", machine: false},
		{accept: "application/json", destination: "empty", machine: true},
		{accept: "text/event-stream", destination: "", machine: true},
	}
)

// member identifies one fixture user.
type member string

// IsAuthenticated reports whether the fixture user is authenticated.
//
// Returns bool which reports whether the fixture identity is authenticated.
func (m member) IsAuthenticated() bool { return m == "reader" || m == "editor" }

// UserID returns the fixture user's public identity.
//
// Returns string which is the fixture user's public identity.
func (m member) UserID() string {
	if m.IsAuthenticated() {
		return string(m)
	}
	return ""
}

// Get returns the selected fixture user claim.
//
// Takes key (string) which identifies the requested field or row.
//
// Returns any which is the requested fixture claim, or nil when absent.
func (m member) Get(key string) any {
	if key == "role" {
		return string(m)
	}
	return nil
}

// authContextKey marks fixture authentication context values.
type authContextKey struct{}

// authProvider records authentication calls and context forwarding.
type authProvider struct {
	// calls counts observed calls.
	calls int

	// forwarded reports whether request context reached the provider.
	forwarded bool
}

// Authenticate selects the fixture identity and records context forwarding.
//
// Takes request (*http.Request) which is the incoming request.
//
// Returns daemon_dto.AuthContext which holds the controlled fixture identity, or is nil
// for a guest or offline provider.
// Returns error which reports the fixture provider's controlled authentication failure,
// or nil on success.
func (p *authProvider) Authenticate(ctx context.Context, request *http.Request) (daemon_dto.AuthContext, error) {
	p.calls++
	p.forwarded = ctx.Value(authContextKey{}) == p.calls && request.Context().Value(authContextKey{}) == p.calls
	actor := request.Header.Get(sessionHeader)
	switch actor {
	case guestActor:
		return nil, nil
	case "offline":
		return nil, errAuthOffline
	default:
		return member(actor), nil
	}
}

// guardReceipt records a guard's calls, callbacks and actor.
type guardReceipt struct {
	// actor identifies the request's session.
	actor string

	// calls counts handler invocations.
	calls int

	// callbacks counts unauthenticated-callback invocations.
	callbacks int
}

// authClient describes one kind of requesting client.
type authClient struct {
	// accept is the client's Accept header.
	accept string

	// destination is the client's Sec-Fetch-Dest header.
	destination string

	// machine reports whether the client expects a JSON refusal instead of a redirect.
	machine bool
}

// authHarness drives one provider and guard configuration through the request grid.
type authHarness struct {
	// provider is the recording authentication provider.
	provider *authProvider

	// handler is the authentication and guard middleware chain.
	handler http.Handler

	// got records the guarded handler's and callback's observations.
	got guardReceipt

	// callback reports whether the guard has a custom unauthenticated callback.
	callback bool
}

// authRefusal is the guard's JSON refusal body.
type authRefusal struct {
	// Error is the refusal code.
	Error string

	// Login is the login location the client should visit.
	Login string
}

// check sends one grid cell through the middleware chain and compares the outcome with
// the independently authored policy.
//
// Takes index (int) which is the request's position in the run.
// Takes cell (int) which selects the actor, route and client.
//
// Returns error which reports a model mismatch.
func (h *authHarness) check(ctx context.Context, index, cell int) error {
	actor := cell % len(authActors)
	route := cell / len(authActors) % len(authRoutes)
	client := &authClients[cell/(len(authActors)*len(authRoutes))]
	h.got = guardReceipt{}
	holder := daemon_dto.AcquirePikoRequestCtx()
	requestCtx := daemon_dto.WithPikoRequestCtx(context.WithValue(ctx, authContextKey{}, index+1), holder)
	request := httptest.NewRequestWithContext(requestCtx, http.MethodGet, authRoutes[route], nil)
	request.Header.Set(sessionHeader, authActors[actor])
	request.Header.Set("Accept", client.accept)
	request.Header.Set("Sec-Fetch-Dest", client.destination)
	recorder := httptest.NewRecorder()
	h.handler.ServeHTTP(recorder, request)
	daemon_dto.ReleasePikoRequestCtx(holder)
	if !h.provider.forwarded || h.provider.calls != index+1 {
		return errors.New("auth-provider-context: provider call count or request context differs")
	}
	label := authLabels[actor]
	allowed := route < publicRoutes || actor == 1 || actor == 2
	switch {
	case allowed:
		if recorder.Code != http.StatusNoContent || h.got != (guardReceipt{actor: label, calls: 1, callbacks: 0}) {
			return fmt.Errorf("auth-handler: %s %s status=%d receipt=%+v", authActors[actor], authRoutes[route], recorder.Code, h.got)
		}
	case h.callback:
		if recorder.Code != http.StatusUnauthorized || h.got != (guardReceipt{actor: label, calls: 0, callbacks: 1}) {
			return fmt.Errorf("auth-callback: %s status=%d receipt=%+v", authActors[actor], recorder.Code, h.got)
		}
	default:
		if h.got != (guardReceipt{}) {
			return errors.New("auth-guard-calls: a refusal invoked the handler or custom callback")
		}
		return checkRefusal(recorder, client, authRoutes[route])
	}
	return nil
}

// authLabel formats the public authentication identity.
//
// Takes authentication (daemon_dto.AuthContext) which is the public authentication
// context.
//
// Returns string which is the public fixture identity marker.
func authLabel(authentication daemon_dto.AuthContext) string {
	if authentication == nil {
		return guestActor
	}
	return fmt.Sprintf("%s/%v", authentication.UserID(), authentication.Get("role"))
}

// RunAuth exercises fresh pooled request contexts through the configured provider and
// guard.
//
// Takes data ([]byte) which holds the replayable choice bytes.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func RunAuth(ctx context.Context, data []byte) error {
	var start byte
	if len(data) > 0 {
		start = data[0]
	}
	for _, callback := range []bool{false, true} {
		harness := newAuthHarness(callback)
		cells := len(authActors) * len(authRoutes) * len(authClients)
		steps := cells + min(authHistoryLimit, len(data))
		for index := range steps {
			if err := ctx.Err(); err != nil {
				return err
			}
			cell := (index + int(start)) % cells
			if index >= cells {
				cell = int(data[index-cells]) % cells
			}
			if err := harness.check(ctx, index, cell); err != nil {
				return err
			}
		}
	}
	return nil
}

// newAuthHarness builds the authentication and guard middleware chain.
//
// Takes callback (bool) which installs a custom unauthenticated callback.
//
// Returns *authHarness which drives the chain.
func newAuthHarness(callback bool) *authHarness {
	harness := &authHarness{provider: &authProvider{calls: 0, forwarded: false}, handler: nil, got: guardReceipt{}, callback: callback}
	config := daemon_dto.AuthGuardConfig{
		PublicPaths:       []string{"/news"},
		PublicPrefixes:    []string{"/static"},
		LoginPath:         "/sign-in?lang=en",
		RedirectParam:     "next",
		OnUnauthenticated: nil,
	}
	if callback {
		config.OnUnauthenticated = func(writer http.ResponseWriter, _ *http.Request, authentication daemon_dto.AuthContext) {
			harness.got.callbacks++
			harness.got.actor = authLabel(authentication)
			writer.WriteHeader(http.StatusUnauthorized)
		}
	}
	guarded := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		harness.got.calls++
		authentication, ok := daemon_dto.PikoRequestCtxFromContext(request.Context()).CachedAuth.(daemon_dto.AuthContext)
		if !ok {
			authentication = nil
		}
		harness.got.actor = authLabel(authentication)
		writer.WriteHeader(http.StatusNoContent)
	})
	harness.handler = security_adapters.NewAuthMiddleware(harness.provider).Handler(security_adapters.NewAuthGuardMiddleware(config).Handler(guarded))
	return harness
}

// checkRefusal checks a guard refusal's redirect or JSON body and its login location.
//
// Takes recorder (*httptest.ResponseRecorder) which holds the refusal.
// Takes client (*authClient) which is the requesting client kind.
// Takes route (string) which is the refused request path.
//
// Returns error which reports a refusal of the wrong kind or a lost login location.
func checkRefusal(recorder *httptest.ResponseRecorder, client *authClient, route string) error {
	location := recorder.Header().Get("Location")
	if client.machine {
		login, err := machineLogin(recorder, location)
		if err != nil {
			return err
		}
		location = login
	} else if recorder.Code != http.StatusSeeOther {
		return fmt.Errorf("auth-redirect: status %d", recorder.Code)
	}
	destination, err := url.Parse(location)
	if err != nil || destination.Host != "" || destination.Path != "/sign-in" || destination.Query().Get("lang") != "en" || destination.Query().Get("next") != route {
		return errors.New("auth-login-location: redirect lost the login route or return destination")
	}
	return nil
}

// machineLogin checks a machine client's JSON refusal and returns its login location.
//
// Takes recorder (*httptest.ResponseRecorder) which holds the refusal.
// Takes location (string) which is the refusal's Location header.
//
// Returns string which is the refusal body's login location.
// Returns error which reports a refusal that is not the expected JSON body.
func machineLogin(recorder *httptest.ResponseRecorder, location string) (string, error) {
	var reply authRefusal
	header := recorder.Header()
	if recorder.Code != http.StatusUnauthorized || location != "" || header.Get("Cache-Control") != "no-store" ||
		!strings.HasPrefix(header.Get("Content-Type"), "application/json") ||
		json.Unmarshal(recorder.Body.Bytes(), &reply) != nil || reply.Error != "unauthenticated" {
		return "", errors.New("auth-json: machine client did not receive the expected refusal")
	}
	return reply.Login, nil
}
