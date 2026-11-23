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

package generate

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/fixturego"
)

const (
	// publicationPage holds the publication fixture's page markup.
	publicationPage = `<main id="publication" data-page="PAGE"><output id="user" p-text="state.User"></output><output id="role" p-text="state.Role"></output>` +
		`<output id="meeting" p-text="state.Meeting"></output><output id="last-visit" p-text="state.Last"></output></main>`

	// publicationRender holds the publication fixture's render function.
	publicationRender = `type Response struct { User, Role, Meeting, Last string }
func Render(r *piko.RequestData,_ piko.NoProps)(Response,piko.Metadata,error){
	s := Response{User:"guest", Role:"visitor", Meeting:"single", Last:"none"}
	if a := r.Auth(); a != nil && a.IsAuthenticated() {
		s.User = a.UserID()
		s.Role, _ = a.Get("publicationRole").(string)
		s.Meeting, _ = a.Get("meeting").(string)
	}
	if prev, e := r.Cookie("publication_visit"); e == nil { s.Last=prev.Value }
	ON_OPEN
	return s,piko.Metadata{},nil
}
`

	// publicationDeskOpen issues the visit cookie when the desk page opens.
	publicationDeskOpen = `r.SetCookie(&http.Cookie{Name:"publication_visit",Value:s.User,Path:"/",HttpOnly:true})`

	// publicationDeskPolicy restricts the desk page to editors and publishers.
	publicationDeskPolicy = `func AuthPolicy() piko.AuthPolicy {return piko.AuthPolicy{Required:true, Roles:[]string{"editor","publisher"}}}`

	// authGuest names the unauthenticated actor and its displayed user.
	authGuest = "guest"

	// authReader names the reader actor and its granted role.
	authReader = "reader"

	// authEditor names the editor actor and its granted role.
	authEditor = "editor"

	// authFrontPage names the public publication page.
	authFrontPage = "front"

	// authProfilePage names the sign-in-required publication page.
	authProfilePage = "profile"

	// authDeskPage names the editor-only publication page.
	authDeskPage = "desk"

	// authNoVisit is the previous-visit value shown before any visit cookie exists.
	authNoVisit = "none"

	// authPrivateCache is the Cache-Control value the profile page sends.
	authPrivateCache = "private, no-store"

	// authForbiddenCode is the refusal code for an authenticated actor without the role.
	authForbiddenCode = "FORBIDDEN"

	// publicationProfile holds the publication fixture's profile page.
	publicationProfile = `func AuthPolicy() piko.AuthPolicy {return piko.AuthPolicy{Required:true}}
func CachePolicy() piko.CachePolicy {return piko.CachePolicy{NoStore:true}}
func Middlewares() []func(http.Handler)http.Handler {
	return []func(http.Handler)http.Handler{privateResponse}
}
func privateResponse(next http.Handler)http.Handler{
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){ w.Header().Set("Cache-Control","private, no-store"); next.ServeHTTP(w,r) })
}
`
)

var (
	// authPublishSource is the formatted Go source staged in the generated project.
	authPublishSource = fixturego.Fixed("actions/publication/publish.go")

	// authLedgerSource is the formatted Go source staged in the generated project.
	authLedgerSource = fixturego.Fixed("actions/publication/read_ledger.go")
)

// addAuthContracts adds source and expected observations for publication authentication.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
//
// Returns error which reports a stale provider fixture edit, or nil on success.
func addAuthContracts(c *casefile.Case, p Project) error {
	edition := fmt.Sprintf("%d-%d", p.Count, c.Index)

	public := []string{"/__piko_fuzz_ready", "/main", "/publication/front", "/publication/desk"}
	for index := range c.Probes {
		parsed, err := url.Parse(c.Probes[index].Path)
		if err != nil {
			return fmt.Errorf("parsing probe path %q: %w", c.Probes[index].Path, err)
		}
		public = append(public, parsed.Path)
	}
	slices.Sort(public)
	claims := [...]string{"string", "string-list", "stringer", "mixed-list"}
	claim := claims[(c.Index+c.Seed)%uint64(len(claims))]
	provider, err := authProvider(edition, public, claim)
	if err != nil {
		return err
	}
	c.Files["internal/authfixture/provider.go"] = provider
	pages := []struct{ name, imports, open, policy string }{
		{name: authFrontPage, imports: "", open: "", policy: ""},
		{name: authProfilePage, imports: `"net/http"`, open: "", policy: publicationProfile},

		{name: authDeskPage, imports: `"net/http"`, open: publicationDeskOpen, policy: publicationDeskPolicy},
	}
	for _, page := range pages {
		script := strings.ReplaceAll(publicationRender, "ON_OPEN", page.open) + page.policy
		c.Files["pages/publication/"+page.name+".pk"] = graphSFC(strings.ReplaceAll(publicationPage, "PAGE", page.name), page.imports, script)
	}
	c.Files["pages/publication/sign-in.pk"] = `<template><main>Publication sign-in</main></template>`
	c.Files["actions/publication/publish.go"] = authPublishSource
	c.Files["actions/publication/read_ledger.go"] = authLedgerSource
	reader, editor := "subscriber-"+edition, "publisher-"+edition
	c.Expected.Auth = &casefile.AuthContract{
		RoleClaim:     claim,
		ReaderSession: "reader-ticket-" + edition,
		EditorSession: "editor-ticket-" + edition,
		Steps:         authSteps(reader, editor),
		Actions:       authActions(editor, "Edition "+edition),
	}
	c.Features = append(c.Features, "auth-provider", "auth-guard", "page-role-policy", "actor-cookie-isolation",
		"overlapping-principals", "private-response", "auth-provider-recovery", "guarded-publication-action")
	return nil
}

// authSteps models the page requests each actor makes and their expected responses.
//
// Takes reader (string) which is the reader actor's user identifier.
// Takes editor (string) which is the editor actor's user identifier.
//
// Returns []casefile.AuthStep which lists the ordered page requests.
func authSteps(reader, editor string) []casefile.AuthStep {
	var steps []casefile.AuthStep
	row := func(name, actor, page, user, role, previous string) {
		steps = append(steps, authStep(name, actor, "/publication/"+page, authPage(page, user, role, previous)))
	}
	deny := func(name, actor, page string, status int, location string) {
		steps = append(steps, authStep(name, actor, "/publication/"+page, authRefusal(status, location)))
	}
	row("guest-front", authGuest, authFrontPage, authGuest, "visitor", authNoVisit)
	deny("guest-profile", authGuest, authProfilePage, http.StatusSeeOther, "/publication/sign-in?next=%2Fpublication%2Fprofile")
	deny("guest-desk", authGuest, authDeskPage, http.StatusSeeOther, "/publication/sign-in?next=%2Fpublication%2Fdesk")
	row("guest-recovery", authGuest, authFrontPage, authGuest, "visitor", authNoVisit)
	row("first-reader", authReader, authProfilePage, reader, authReader, authNoVisit)
	row("first-editor", authEditor, authProfilePage, editor, authEditor, authNoVisit)
	deny("reader-desk", authReader, authDeskPage, http.StatusForbidden, "")
	row("reader-recovery", authReader, authProfilePage, reader, authReader, authNoVisit)
	row("editor-desk", authEditor, authDeskPage, editor, authEditor, authNoVisit)
	row("editor-fresh", authEditor, authProfilePage, editor, authEditor, editor)
	row("reader-fresh", authReader, authProfilePage, reader, authReader, authNoVisit)
	for _, actor := range []string{authReader, authEditor} {
		user, previous := reader, authNoVisit
		if actor == authEditor {
			user, previous = editor, editor
		}
		row("paired-"+actor, actor, authProfilePage, user, actor, previous)
		step := &steps[len(steps)-1]
		step.Parallel = true
		step.Path += "?meeting=paired"
		step.Want.Rendezvous = "paired"
	}
	row("final-guest", authGuest, authFrontPage, authGuest, "visitor", authNoVisit)
	renewal := "/publication/profile?edition=renewal"
	return append(steps,
		authStep("provider-unavailable", authReader, renewal, authRefusal(http.StatusSeeOther, "/publication/sign-in?next=%2Fpublication%2Fprofile%3Fedition%3Drenewal")),
		authStep("provider-recovered", authReader, renewal, authPage(authProfilePage, reader, authReader, authNoVisit)),
	)
}

// authActions models the publish action attempts and their ledger effects.
//
// Takes editor (string) which is the editor actor's user identifier.
// Takes headline (string) which is the headline the editor publishes.
//
// Returns []casefile.AuthActionStep which lists the ordered action attempts.
func authActions(editor, headline string) []casefile.AuthActionStep {
	approved := casefile.AuthLedger{Calls: 2, Writes: 1, Owner: editor, Headline: headline}
	unchanged := approved
	unchanged.Calls++
	forbidden := casefile.AuthActionResult{Status: http.StatusForbidden, Code: authForbiddenCode, Login: "", By: "", Headline: ""}
	return []casefile.AuthActionStep{
		{
			Name: "guest-publish", Actor: authGuest, Headline: "Guest draft",
			Want: casefile.AuthActionResult{
				Status: http.StatusUnauthorized, Code: "unauthenticated",
				Login: "/publication/sign-in?next=%2F_piko%2Factions%2Fpublication.Publish", By: "", Headline: "",
			},
			Ledger: casefile.AuthLedger{Calls: 0, Writes: 0, Owner: "", Headline: ""},
		},
		{
			Name: "reader-publish", Actor: authReader, Headline: "Reader draft", Want: forbidden,
			Ledger: casefile.AuthLedger{Calls: 1, Writes: 0, Owner: "", Headline: ""},
		},
		{
			Name: "editor-publish", Actor: authEditor, Headline: headline,
			Want:   casefile.AuthActionResult{Status: http.StatusOK, Code: "", Login: "", By: editor, Headline: headline},
			Ledger: approved,
		},
		{Name: "reader-after-publication", Actor: authReader, Headline: "Second reader draft", Want: forbidden, Ledger: unchanged},
	}
}

// authStep builds one sequential publication page request.
//
// Takes name (string) which identifies the step.
// Takes actor (string) which selects the requesting actor.
// Takes path (string) which is the requested path.
// Takes want (casefile.AuthResponse) which holds the expected response.
//
// Returns casefile.AuthStep which holds the request and its expectation.
func authStep(name, actor, path string, want casefile.AuthResponse) casefile.AuthStep {
	return casefile.AuthStep{Name: name, Actor: actor, Path: path, Parallel: false, Want: want}
}

// authPage models a successfully rendered publication page.
//
// Takes page (string) which names the rendered page.
// Takes user (string) which is the displayed user.
// Takes role (string) which is the displayed role.
// Takes previous (string) which is the displayed previous visit.
//
// Returns casefile.AuthResponse which holds the expected rendered response.
func authPage(page, user, role, previous string) casefile.AuthResponse {
	response := authRefusal(http.StatusOK, "")
	response.Page = page
	response.User = user
	response.Role = role
	response.Rendezvous = "single"
	response.PreviousVisit = previous
	if page == authProfilePage {
		response.CacheControl = authPrivateCache
	}
	if page == authDeskPage {
		response.IssuedVisit = []string{user}
	}
	return response
}

// authRefusal models a publication page request that is redirected or refused.
//
// Takes status (int) which is the expected HTTP status.
// Takes location (string) which is the expected redirect target, or empty.
//
// Returns casefile.AuthResponse which holds the expected refusal.
func authRefusal(status int, location string) casefile.AuthResponse {
	var response casefile.AuthResponse
	response.Status = status
	response.Location = location
	return response
}
