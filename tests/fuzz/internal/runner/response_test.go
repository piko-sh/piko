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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

type responseTransport func(*http.Request) (*http.Response, error)

type locatedFailure interface {
	error
	FailureSubject() string
}

func (f responseTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestObservedResponseContracts(t *testing.T) {
	const home = `<html lang="en"><head><title>Piko Page</title><link rel="canonical" href="/main"></head><body><div id="app"><h1>Home</h1></div></body></html>`
	const detail = `<html lang="en-GB"><head><title>Shop</title><link rel="canonical" href="/shop"><meta name="description" content="items"><meta property="og:title" content="Shop"><meta name="twitter:card" content="summary"></head><body><div id="token">Buy this</div><a data-fuzz-element="buy" href="/item" aria-labelledby="token">Buy</a><form data-fuzz-element="form" action="/find" method="get" aria-label="Search"><label for="q">Find item</label><input data-fuzz-element="query" id="q" name="query"></form></body></html>`
	primary := &casefile.ResponseContract{Head: map[string][]string{"canonical": {"/main"}}}
	secondary := &casefile.ResponseContract{Head: map[string][]string{"lang": {"en-GB"}, "title": {"Shop"}, "canonical": {"/shop"}, "description": {"items"}, "og:title": {"Shop"}, "twitter:card": {"summary"}}, Elements: []casefile.PageElement{
		{Key: "buy", Tag: "a", Name: "Buy this", Attributes: map[string]string{"href": "/item"}},
		{Key: "form", Tag: "form", Name: "Search", Attributes: map[string]string{"method": "get", "action": "/find"}},
		{Key: "query", Tag: "input", Name: "Find item", Attributes: map[string]string{"name": "query"}},
	}}
	testCase := casefile.Case{Request: "/main", Expected: casefile.Expected{Title: "Home", Status: http.StatusOK, Semantic: primary}, Probes: []casefile.Probe{{Path: "/shop", Status: http.StatusOK, Semantic: secondary}}}
	observePages := func(home, detail string) Observation {
		t.Helper()
		client := &http.Client{Transport: responseTransport(func(request *http.Request) (*http.Response, error) {
			body := detail
			if request.URL.Path == "/main" {
				body = home
			}
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/html"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})}
		o, err := observe(t.Context(), client, "http://local", &testCase)
		require.NoError(t, err)
		return o
	}
	good := observePages(home, detail)
	require.NoError(t, Check(&testCase, &good))
	reordered := observePages(home, strings.Replace(detail, `href="/item" aria-labelledby="token"`, `aria-labelledby="token" href="/item"`, 1))
	require.NoError(t, compareObservations(&good, &reordered))
	for _, mutation := range []struct{ where, previous, replacement, label string }{
		{where: "primary", previous: `href="/main"`, replacement: `href="/other"`, label: "primary/head/canonical"},
		{where: "probe", previous: `href="/item"`, replacement: `href="/old"`, label: "probe/semantic/href"},
		{where: "probe", previous: `id="token"`, replacement: `id="wrong"`, label: "probe/semantic/name"},
		{where: "probe", previous: `for="q"`, replacement: `for="bad"`, label: "probe/semantic/name"},
		{where: "probe", previous: `action="/find"`, replacement: `action="/old"`, label: "probe/semantic/action"},
		{where: "probe", previous: `method="get"`, replacement: `method="post"`, label: "probe/semantic/method"},
		{where: "probe", previous: `lang="en-GB"`, replacement: `lang="fr"`, label: "probe/head/lang"},
		{where: "probe", previous: `content="items"`, replacement: `content="old"`, label: "probe/head/description"},
		{where: "probe", previous: `content="summary"`, replacement: `content="other"`, label: "probe/head/twitter:card"},
		{where: "probe", previous: `content="Shop"`, replacement: `content="Old"`, label: "probe/head/og:title"},
		{where: "probe", previous: `</head>`, replacement: `<link rel="canonical" href="/shop"></head>`, label: "probe/head/canonical"},
	} {
		mainPage, secondPage := home, detail
		if mutation.where == "primary" {
			mainPage = strings.Replace(mainPage, mutation.previous, mutation.replacement, 1)
		} else {
			secondPage = strings.Replace(secondPage, mutation.previous, mutation.replacement, 1)
		}
		bad := observePages(mainPage, secondPage)
		err := Check(&testCase, &bad)
		if assert.Error(t, err, "independent %s", mutation.label) {
			assert.Contains(t, err.Error(), mutation.label)
		}
		assert.Error(t, compareObservations(&good, &bad), "differential missed %s", mutation.label)
	}
}

func TestActorCookieHistory(t *testing.T) {
	actors := map[string]*http.Client{}
	base := &http.Client{Transport: responseTransport(func(request *http.Request) (*http.Response, error) {
		theme, err := request.Cookie("theme")
		if err != nil {
			return nil, err
		}
		marker := "first"
		if visited, err := request.Cookie("visited"); err == nil {
			marker = visited.Value
		}
		body := fmt.Sprintf("%s/%s", theme.Value, marker)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Set-Cookie": {"visited=" + theme.Value + "; Path=/"}}}, nil
	})}
	var got []string
	for _, probe := range []casefile.Probe{{Actor: "A", Cookies: map[string]string{"theme": "light"}}, {Actor: "B", Cookies: map[string]string{"theme": "dark"}}, {Actor: "A"}} {
		actor, err := actorClient(base, actors, "http://local", probe)
		require.NoError(t, err)
		_, _, body, err := get(t.Context(), actor, "http://local", "/shop")
		require.NoError(t, err)
		got = append(got, string(body))
	}
	assert.Equal(t, []string{"light/first", "dark/first", "light/light"}, got)
}

func TestModeSpecificDraftDoesNotHideIncorrectCommonOrModeResponses(t *testing.T) {
	testCase := casefile.Case{Expected: casefile.Expected{Status: http.StatusOK, Title: "Home"}, Probes: []casefile.Probe{
		{Path: "/publish", Status: http.StatusOK, Semantic: &casefile.ResponseContract{Head: map[string][]string{"title": {"Published"}}}},
		{Path: "/draft", ByMode: map[string]casefile.ModeResponse{}},
	}}
	snapshot := func(title string) *casefile.ResponseContract {
		return &casefile.ResponseContract{Head: map[string][]string{"title": {title}}}
	}
	makeMode := func(mode, title string, status int) Observation {
		testCase.Probes[1].ByMode[mode] = casefile.ModeResponse{Status: status, Semantic: *snapshot(title)}
		return Observation{Mode: mode, Heading: "Home", MetadataTitle: "Piko Page", Status: http.StatusOK, ContentType: "text/html", Probes: []ProbeObservation{
			{Path: "/publish", Status: http.StatusOK, Text: "Published", Semantic: snapshot("Published")},
			{Path: "/draft", Status: status, Text: title, Semantic: snapshot(title)},
		}}
	}
	interp := makeMode("interp", "Draft", http.StatusOK)
	compiled := makeMode("compiled", "Custom missing", http.StatusNotFound)
	require.NoError(t, CompareCase(&testCase, &interp, &compiled))
	for _, step := range []int{0, 1} {
		changed := compiled
		changed.Probes = append([]ProbeObservation(nil), changed.Probes...)
		changed.Probes[step].Semantic = snapshot("Wrong")
		assert.Error(t, CompareCase(&testCase, &interp, &changed), "ignored mismatch on %s", testCase.Probes[step].Path)
	}
	compiled.Probes[1].Semantic.Head["canonical"] = []string{"/wrong"}
	require.Error(t, CompareCase(&testCase, &interp, &compiled), "suppressed an undeclared draft head difference")
	delete(compiled.Probes[1].Semantic.Head, "canonical")
	for _, o := range []Observation{interp, compiled} {
		want := testCase.Probes[1].ByMode[o.Mode]
		element := casefile.PageElement{Key: "home", Tag: "a", Name: want.Semantic.Head["title"][0]}
		want.Semantic.Elements = []casefile.PageElement{element}
		testCase.Probes[1].ByMode[o.Mode] = want
		element.Attributes = map[string]string{"href": "/"}
		o.Probes[1].Semantic.Elements = []casefile.PageElement{element}
	}
	require.NoError(t, CompareCase(&testCase, &interp, &compiled), "declared accessible name change rejected")
	compiled.Probes[1].Semantic.Elements[0].Attributes["href"] = "/incorrect"
	require.Error(t, CompareCase(&testCase, &interp, &compiled), "suppressed undeclared href because a common landmark changed names")
	compiled.Probes[1].Semantic.Elements[0].Attributes["href"] = "/"
	compiled.Probes[1].Status = http.StatusOK
	require.Error(t, CompareCase(&testCase, &interp, &compiled), "status difference was accepted without independent confirmation")
}

func TestExplicitEmptyOrdinaryContractReplays(t *testing.T) {
	original := casefile.ResponseContract{Ordinary: []casefile.PageElement{}}
	data, err := json.Marshal(original)
	require.NoError(t, err)
	var saved casefile.ResponseContract
	require.NoError(t, json.Unmarshal(data, &saved))
	require.NoError(t, checkResponseContract(&saved, &casefile.ResponseContract{}))
	assert.Error(t, checkResponseContract(&saved, &casefile.ResponseContract{Ordinary: []casefile.PageElement{{Tag: "input"}}}), "explicit absence lost on replay")
}

func TestUnmarkedFollowupLinkAndMainCanonicalParity(t *testing.T) {
	parse := func(markup string) *casefile.ResponseContract {
		t.Helper()
		document, err := html.Parse(strings.NewReader(markup))
		require.NoError(t, err)
		return readResponseContract(document)
	}
	good := parse(`<head><link rel="canonical" href="/shop"></head><main><a href="/buy/a" aria-label="Buy A">Buy</a></main>`)
	for _, markup := range []string{
		`<head><link rel="canonical" href="/else"></head><main><a href="/buy/a" aria-label="Buy A">Buy</a></main>`,
		`<head><link rel="canonical" href="/shop"></head><main><a href="/buy/b" aria-label="Buy B">Buy</a></main>`,
	} {
		bad := parse(markup)
		assert.Error(t, compareResponseContracts(good, bad), "accepted a changed link")
		assert.Error(t, compareResponseContracts(bad, good), "accepted a changed link")
		assert.Error(t, checkResponseContract(good, bad), "accepted a changed modelled response")
	}
}

func TestHistoricFallbackOnlyAcceptsKnownErrorDelivery(t *testing.T) {
	probe := casefile.Probe{Path: "/missing", Status: http.StatusNotFound}
	testCase := casefile.Case{Files: map[string]string{"pages/missing.pk": `&piko.NotFoundError{Resource:"item", ID:"missing"}`}, Features: []string{"typed-error"}, Probes: []casefile.Probe{probe}, Expected: casefile.Expected{Title: "Home", Status: http.StatusOK}}
	newObservation := func(mode, raw string) Observation {
		return Observation{Mode: mode, Status: http.StatusOK, Heading: "Home", MetadataTitle: "Piko Page", ContentType: "text/html", Probes: []ProbeObservation{{Path: "/missing", Status: http.StatusNotFound, RawHTML: raw, Text: raw}}}
	}
	interp := newObservation("interp", `<title>404 Not Found</title><h1>404 Not Found</h1><pre>renderPage: runPage failed for pages/missing.pk: item not found: missing</pre>`)
	native := newObservation("compiled", "Not Found\n")
	require.NoError(t, CompareCase(&testCase, &interp, &native))
	for _, bad := range []Observation{newObservation("compiled", "Internal Server Error\n"), newObservation("interp", `<h1>404 Not Found</h1><pre>wrong page</pre>`)} {
		assert.Error(t, Check(&testCase, &bad), "accepted wrong fallback")
	}
	testCase.Files["pages/!404.pk"] = "<template>custom</template>"
	assert.Error(t, CompareCase(&testCase, &interp, &native), "suppressed different custom pages")
}

func TestWatchedCardSubjectsDistinguishStaleAndInvalidPages(t *testing.T) {
	expected := &casefile.ResponseContract{Head: map[string][]string{"canonical": {"/shop"}}, Elements: []casefile.PageElement{{Key: "buy", Tag: "a", Name: "Inspect"}}}
	for _, testCase := range []struct {
		actual  casefile.ResponseContract
		subject string
	}{
		{actual: casefile.ResponseContract{Head: map[string][]string{"canonical": {"/fallback"}}}, subject: "head/canonical"},
		{actual: casefile.ResponseContract{Head: expected.Head, Elements: []casefile.PageElement{{Key: "buy", Tag: "a", Name: "Buy"}}}, subject: "semantic/buy/name"},
	} {
		wrapped := fmt.Errorf("catalog-watch/edit-card: %w", checkResponseContract(expected, &testCase.actual))
		assertFailureSubject(t, wrapped, testCase.subject)
	}
}

func assertFailureSubject(t *testing.T, err error, want string) bool {
	t.Helper()
	located, ok := errors.AsType[locatedFailure](err)
	if !assert.True(t, ok, "error without a failure subject: %v", err) {
		return false
	}
	return assert.Equal(t, want, located.FailureSubject(), "%v", err)
}
