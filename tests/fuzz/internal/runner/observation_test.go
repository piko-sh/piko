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
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

const (
	fakeComponentScript = "/_piko/assets/generated/components/x-card_0123456789abcdef.js"
)

func TestFixtureObservations(t *testing.T) {
	repo, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	for _, fixture := range []struct {
		name, title string
		lines       []string
	}{
		{name: "01_static_page", title: "Hello, Piko!", lines: []string{"This is a completely static page rendered by the interpreted runner."}},
		{name: "02_simple_dynamic_page", title: "Dynamic Page Title", lines: []string{"The message is: This content comes from the interpreted Go Render function.", "Request Path: /main", "<text> & more", "Label: <text> & more"}},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			testCase := casefile.Case{Expected: casefile.Expected{Title: fixture.title, Rows: fixture.lines, Status: http.StatusOK}}
			var snapshots []Observation
			for _, suite := range []string{"tests/integration/interpreted_runner", "internal/templater/templater_test/compiled_runner"} {
				golden := filepath.Join(repo, suite, "testdata", fixture.name, "golden", "golden.html")
				data, err := os.ReadFile(golden)
				require.NoError(t, err)
				markup := strings.ReplaceAll(string(data), "http://NORMALISED/", "http://example.com/")
				o := Observation{RawHTML: markup, Status: http.StatusOK, ContentType: "text/html"}
				_, err = extract(&o, "")
				require.NoError(t, err)
				require.NoError(t, Check(&testCase, &o))
				snapshots = append(snapshots, o)
			}
			require.NoError(t, compareObservations(&snapshots[0], &snapshots[1]))
			changed := snapshots[0]
			changed.ApplicationDOM = strings.Replace(changed.ApplicationDOM, "p-key=\"r.0\"", "p-key=\"wrong\"", 1)
			assert.Error(t, compareObservations(&changed, &snapshots[1]), "changed user key passed comparison")
			changed = snapshots[0]
			changed.Heading = "broken in both"
			assert.Error(t, Check(&testCase, &changed), "incorrect shared heading passed independent check")
		})
	}
}

func TestScopedVolatility(t *testing.T) {
	build := func(data string) Observation {
		t.Helper()
		o := Observation{RawHTML: data}
		_, err := extract(&o, "fuzz-counter")
		require.NoError(t, err)
		return o
	}
	page := `<head><meta name="csrf-token" content="alpha"></head><body><div id="app"><main id="case-root"><h1 id="case-title">title</h1><ul id="case-list"><li>a</li><li>b</li></ul><fuzz-counter data-token="alpha"></fuzz-counter></main></div></body>`
	base := build(page)
	require.NoError(t, compareObservations(&base, new(build(strings.Replace(page, `content="alpha"`, `content="different"`, 1)))))
	for _, mutated := range []string{strings.Replace(page, "data-token=\"alpha\"", "data-token=\"beta\"", 1), strings.Replace(page, "a</li><li>b", "b</li><li>a", 1)} {
		assert.Error(t, compareObservations(&base, new(build(mutated))), "application difference went undetected")
	}
	digest := sha512.Sum384([]byte("asset"))
	integrity := "sha384-" + base64.StdEncoding.EncodeToString(digest[:])
	assert.True(t, validIntegrity(integrity, []byte("asset")), "integrity digest incorrectly rejected")
	assert.False(t, validIntegrity(integrity, []byte("other")), "integrity digest incorrectly accepted")
}

func TestComponentVariant(t *testing.T) {
	makeResult := func(name, tag string) Observation {
		return Observation{Scripts: []string{name}, ComponentAsset: name, ComponentTag: tag}
	}
	prefix := "/_piko/assets/generated/components/"
	source := makeResult(prefix+"fuzz-counter_b0aa91f7cb653f34.js", "fuzz-counter")
	production := makeResult(prefix+"fuzz-counter_ce941491278fbf1b.min.js", "fuzz-counter")
	require.NoError(t, compareObservations(&source, &production))
	assert.Error(t, compareObservations(&source, new(makeResult(prefix+"wrong_ce941491278fbf1b.min.js", "fuzz-counter"))), "wrong component accepted")
	leaf := "fuzz-leaf"
	addChild := func(o Observation, path string) Observation {
		o.Scripts = append(o.Scripts, path)
		o.ComponentModules = map[string]string{leaf: path}
		return o
	}
	nestedSource := addChild(source, prefix+"fuzz-leaf_b0aa91f7cb653f34.js")
	nestedProduction := addChild(production, prefix+"fuzz-leaf_ce941491278fbf1b.min.js")
	require.NoError(t, compareObservations(&nestedSource, &nestedProduction))
	nestedProduction.ComponentModules = nil
	assert.Error(t, compareObservations(&nestedSource, &nestedProduction), "unverified child script accepted")
}

func TestIndependentClientAndFollowupChecks(t *testing.T) {
	testCase := casefile.Case{
		Expected: casefile.Expected{Title: "seed", Tag: "fuzz-counter", Status: http.StatusOK, Count: 4},
		Probes: []casefile.Probe{
			{Path: "/moved", Status: http.StatusSeeOther, Location: "/main"},
			{Path: "/items/id", Status: http.StatusOK, Text: "id"},
		},
	}
	actual := Observation{
		ContentType: "text/html", Status: http.StatusOK, MetadataTitle: "seed", Heading: "seed",
		Component: true, ComponentCompiled: true, ComponentAsset: "/component.js", Counts: []string{"4", "5"},
		Probes: []ProbeObservation{
			{Path: "/moved", Status: http.StatusSeeOther, Location: "/main"},
			{Path: "/items/id", Status: http.StatusOK, Text: "item id"},
		},
	}
	require.NoError(t, Check(&testCase, &actual))
	for _, test := range []struct {
		change func(*Observation)
		name   string
		label  string
	}{
		{name: "client returned same wrong result in both modes", change: func(o *Observation) { o.Counts = []string{"4", "6"} }, label: "browser/count:"},
		{name: "existing route returned 404", change: func(o *Observation) { o.Probes[1].Status = http.StatusNotFound }, label: "probe/status:"},
		{name: "redirect reached wrong route", change: func(o *Observation) { o.Probes[0].Location = "/other" }, label: "probe/location:"},
		{name: "route parameter disappeared", change: func(o *Observation) { o.Probes[1].Text = "other" }, label: "probe/text:"},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := actual
			changed.Probes = append([]ProbeObservation(nil), actual.Probes...)
			test.change(&changed)
			err := Check(&testCase, &changed)
			require.Error(t, err)
			assert.True(t, strings.HasPrefix(err.Error(), test.label), "want %s failure; got %v", test.label, err)
		})
	}
}

func TestSiblingStateIsolation(t *testing.T) {
	expected := &casefile.ClientTransition{
		Before: []casefile.ClientInstance{{Count: "2", Label: "first"}, {Count: "2", Label: "second"}},
		After:  []casefile.ClientInstance{{Count: "3", Label: "first"}, {Count: "2", Label: "second"}},
	}
	observed := &casefile.ClientTransition{Before: expected.Before, After: append([]casefile.ClientInstance(nil), expected.After...)}
	observed.After[1].Count = "3"
	err := compareClient(observed, expected)
	require.Error(t, err, "sibling changed undetected")
	assert.Contains(t, err.Error(), "instance 1")
}

func TestPartialRequiresProjectedFrame(t *testing.T) {
	const body = `<main id="case-root"><h1 id="case-title">Hello</h1><ul id="case-list"></ul></main>`
	for _, testCase := range []struct {
		markup string
		frame  bool
	}{
		{markup: `<frame is="frame">` + body + "</frame>", frame: false},
		{markup: `<section id="case-frame"></section>` + body, frame: false},
		{markup: `<section id="case-frame">` + body + "</section>", frame: true},
	} {
		o := Observation{RawHTML: `<div id="app">` + testCase.markup + "</div>"}
		_, err := extract(&o, "fuzz-counter")
		require.NoError(t, err)
		assert.Equal(t, testCase.frame, o.Frame, "frame for %s", testCase.markup)
	}
}

func TestExtendedClientOracles(t *testing.T) {
	want := generate.New("components", 9, 3).Expected.Client
	require.GreaterOrEqual(t, len(want.Steps), 11)
	require.LessOrEqual(t, len(want.Steps), 12)
	raw, err := json.Marshal(want)
	require.NoError(t, err)
	for _, testCase := range []struct {
		damage       func(*casefile.ClientStep)
		name, bucket string
		op           casefile.ClientOp
	}{
		{name: "checkbox native reset stale", bucket: "browser/checkbox", op: casefile.ResetControls, damage: func(step *casefile.ClientStep) {
			form := step.After[step.Action.Instance].Controls.Form
			form.Checked = !form.Checked
		}},
		{name: "checkbox state stale", bucket: "browser/checkbox", op: casefile.ToggleCheck, damage: func(step *casefile.ClientStep) {
			step.After[step.Action.Instance].Controls.Form.CheckedState = "wrong"
		}},
		{name: "checkbox sibling reflected", bucket: "browser/checkbox", op: casefile.ToggleCheck, damage: func(step *casefile.ClientStep) {
			form := step.After[1-step.Action.Instance].Controls.Form
			form.HostChecked = !form.HostChecked
		}},
		{name: "select native reset stale", bucket: "browser/select", op: casefile.ResetControls, damage: func(step *casefile.ClientStep) {
			step.After[step.Action.Instance].Controls.Form.Choice = "stale"
		}},
		{name: "select did not bind", bucket: "browser/select", op: casefile.SelectOption, damage: func(step *casefile.ClientStep) {
			step.After[step.Action.Instance].Controls.Form.ChoiceState = "wrong"
		}},
		{name: "select sibling reflected", bucket: "browser/select", op: casefile.SelectOption, damage: func(step *casefile.ClientStep) {
			step.After[1-step.Action.Instance].Controls.Form.HostChoice = "wrong"
		}},
		{name: "native input stale", bucket: "browser/input", op: casefile.SetInput, damage: func(step *casefile.ClientStep) {
			step.After[step.Action.Instance].Controls.Input += "!"
		}},
		{name: "host reflection stale", bucket: "browser/reflection", op: casefile.SetInput, damage: func(step *casefile.ClientStep) {
			step.After[step.Action.Instance].Controls.HostLabel += "!"
		}},
		{name: "attribute did not update text", bucket: "browser/attribute", op: casefile.SetLabel, damage: func(step *casefile.ClientStep) {
			step.After[step.Action.Instance].Label += "!"
		}},
		{name: "keyed nodes recreated", bucket: "browser/key-identity", op: casefile.ReverseRows, damage: func(step *casefile.ClientStep) {
			step.After[step.Action.Instance].Controls.Keys[0] += 100
		}},
		{name: "removed row still visible", bucket: "browser/rows", op: casefile.RemoveRow, damage: func(step *casefile.ClientStep) {
			step.After[step.Action.Instance].Rows = append(step.After[step.Action.Instance].Rows, "ghost")
		}},
		{name: "default click allowed", bucket: "browser/prevent", op: casefile.ModifiedClick, damage: func(step *casefile.ClientStep) {
			step.Prevented = false
		}},
		{name: "click reached parent", bucket: "browser/propagation", op: casefile.ModifiedClick, damage: func(step *casefile.ClientStep) {
			step.After[step.Action.Instance].Controls.Bubbled = "7"
		}},
		{name: "wrong sibling updated", bucket: "browser/attribute", op: casefile.SetLabel, damage: func(step *casefile.ClientStep) {
			step.After[1-step.Action.Instance].Label += "!"
		}},
		{name: "parent handle missing", bucket: "browser/propagation", op: casefile.ParentClick, damage: func(step *casefile.ClientStep) {
			step.After[step.Action.Instance].Controls.Bubbled = "0"
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var got casefile.ClientTransition
			require.NoError(t, json.Unmarshal(raw, &got))
			for index := range got.Steps {
				if got.Steps[index].Action.Op == testCase.op {
					testCase.damage(&got.Steps[index])
					break
				}
			}
			err := compareClient(&got, want)
			require.Error(t, err, "want %s mismatch", testCase.bucket)
			assert.True(t, strings.HasPrefix(err.Error(), testCase.bucket+":"), "want %s mismatch; got %v", testCase.bucket, err)
		})
	}
}

func TestAdvancedClientOracles(t *testing.T) {
	families := map[string]*casefile.ClientTransition{}
	for index := range uint64(128) {
		client := generate.New("components", 31, index).Expected.Client
		for field := range client.Before[0].Advanced {
			families[field] = client
		}
	}
	for _, testCase := range []struct {
		field   string
		op      casefile.ClientOp
		sibling bool
	}{
		{field: "radio-model", op: casefile.ChooseRadio},
		{field: "radio-values", op: casefile.ChooseRadio},
		{field: "multi", op: casefile.SelectMany},
		{field: "multi-model", op: casefile.SelectMany, sibling: true},
		{field: "number-model", op: casefile.TypeNumber},
		{field: "branch", op: casefile.RefEdit},
		{field: "branch", op: casefile.HideMemo},
		{field: "branch", op: casefile.ShowRefocus},
		{field: "markup", op: casefile.HideMemo},
		{field: "note-attr", op: casefile.RefEdit, sibling: true},
		{field: "phase-attr", op: casefile.HideMemo},
		{field: "context-order", op: casefile.HideMemo},
		{field: "context-identity", op: casefile.ShowRefocus},
		{field: "shown", op: casefile.HideMemo},
		{field: "editor-identity", op: casefile.ShowRefocus},
		{field: "focus", op: casefile.RefEdit},
		{field: "class", op: casefile.HideMemo},
		{field: "opacity", op: casefile.HideMemo},
		{field: "memo-stable", op: casefile.HideMemo},
		{field: "memo", op: casefile.HideMemo},
		{field: "row-identity", op: casefile.HideMemo},
		{field: "parent-picked", op: casefile.ChildPick},
		{field: "hooks", op: casefile.RemountChild},
		{field: "leaf-identity", op: casefile.RemountChild, sibling: true},
		{field: "timeline-hidden", op: casefile.SeekTimeline},
		{field: "timeline-class", op: casefile.SeekTimeline, sibling: true},
		{field: "timeline-ref", op: casefile.SeekTimeline},
	} {
		t.Run(testCase.field, func(t *testing.T) {
			want := families[testCase.field]
			require.NotNil(t, want, "fixture family missing")
			data, err := json.Marshal(want)
			require.NoError(t, err)
			var actual casefile.ClientTransition
			require.NoError(t, json.Unmarshal(data, &actual))
			for _, step := range actual.Steps {
				if step.Action.Op == testCase.op {
					instance := step.Action.Instance
					if testCase.sibling {
						instance = 1 - instance
					}
					step.After[instance].Advanced[testCase.field] = "wrong"
					break
				}
			}
			err = compareClient(&actual, want)
			require.Error(t, err, "expected %s rejection", testCase.field)
			assert.True(t, strings.HasPrefix(err.Error(), "browser/"+testCase.field+":"), "expected %s rejection, got %v", testCase.field, err)
		})
	}
}

func TestLegacyClientRecipeChoosesItsEvidence(t *testing.T) {
	before := []casefile.ClientInstance{{Count: "1", Advanced: map[string]string{"timeline-hidden": "true"}}}
	observed := []casefile.ClientInstance{{Count: "1", Advanced: map[string]string{"timeline-hidden": "true", "timeline-ref": "true"}}}
	require.NoError(t, compareInstances("legacy", observed, before))
	before[0].Advanced = nil
	require.NoError(t, compareInstances("legacy", observed, before))
}

func TestGeneratedSiblingRoutesRequireOneExactBadge(t *testing.T) {
	project := generate.New("pages", 7, 8)
	var probes []casefile.Probe
	for _, probe := range project.Probes {
		if probe.Path == "/left" || probe.Path == "/right" {
			probes = append(probes, probe)
		}
	}
	require.Len(t, probes, 2, "missing generated siblings: %+v", probes)
	require.Contains(t, project.Files["partials/badge.pk"], "data-fuzz-element")
	testCase := casefile.Case{Request: "/main", Expected: casefile.Expected{Title: "OK", Status: http.StatusOK}, Probes: probes}
	serve := func(markup [2]string) Observation {
		t.Helper()
		client := &http.Client{Transport: responseTransport(func(request *http.Request) (*http.Response, error) {
			body := "<h1>OK</h1>"
			for index, probe := range probes {
				if probe.Path == request.URL.Path {
					body = markup[index]
				}
			}
			page := "<html><head><title>Piko Page</title></head><body><div id=\"app\">" + body + "</div></body></html>"
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/html"}}, Body: io.NopCloser(strings.NewReader(page))}, nil
		})}
		o, err := observe(t.Context(), client, "http://local", &testCase)
		require.NoError(t, err)
		return o
	}
	badge := func(value string) string { return fmt.Sprintf(`<span data-fuzz-element="badge">%s</span>`, value) }
	goodMarkup := [2]string{badge(probes[0].Text), badge(probes[1].Text)}
	good := serve(goodMarkup)
	require.NoError(t, CompareCase(&testCase, &good, &good))
	for index, probe := range probes {
		for _, mutation := range []struct{ name, text, field string }{
			{name: "replaced by sibling", text: goodMarkup[1-index], field: "semantic/app-text"},
			{name: "joined labels", text: badge(probes[0].Text + " " + probes[1].Text), field: "semantic/app-text"},
			{name: "stray sibling", text: goodMarkup[index] + badge(probes[1-index].Text), field: "semantic/app-text"},
			{name: "wrong element", text: strings.ReplaceAll(goodMarkup[index], "span", "em"), field: "semantic/badge/element"},
			{name: "hidden badge", text: strings.Replace(goodMarkup[index], "<span", "<span hidden", 1), field: "semantic/app-text"},
		} {
			t.Run(probe.Path+"/"+mutation.name, func(t *testing.T) {
				markup := goodMarkup
				markup[index] = mutation.text
				bad := serve(markup)
				for _, err := range []error{Check(&testCase, &bad), CompareCase(&testCase, &good, &bad), CompareCase(&testCase, &bad, &bad)} {
					assertFailureSubject(t, err, probe.Path+"/"+mutation.field)
				}
			})
		}
	}
	swapped := serve([2]string{goodMarkup[1], goodMarkup[0]})
	failures := IndependentFailures(&testCase, &swapped)
	require.Len(t, failures, 2, "both swapped siblings must survive: %v", failures)
	for index, err := range failures {
		assertFailureSubject(t, err, probes[index].Path+"/semantic/app-text")
	}
}

func TestRouteHeadersHeadAndFallbackSensitivity(t *testing.T) {
	const source = `<html lang="en"><head><title>Bulletin</title><link rel="alternate" hreflang="fr" href="/fr/bulletin"><meta name="robots" content="noindex"></head><body><main id="app"><h2 data-fuzz-element="caption">Lead</h2><h2 data-fuzz-element="caption">Fallback</h2></main></body></html>`
	want := casefile.Probe{
		Path:            "/bulletin",
		Status:          http.StatusForbidden,
		ResponseHeaders: map[string][]string{"X-Edition-Trace": {"outer", "inner"}},
		Semantic: &casefile.ResponseContract{
			Head:     map[string][]string{"title": {"Bulletin"}, "alternate/fr": {"/fr/bulletin"}, "robots": {"noindex"}},
			Elements: []casefile.PageElement{{Key: "caption", Tag: "h2", Name: "Lead"}, {Key: "caption", Tag: "h2", Name: "Fallback"}},
		},
	}
	healthyChain := []string{"outer", "inner"}
	for _, testCase := range []struct {
		name, source string
		headers      []string
		status       int
		good         bool
	}{
		{name: "healthy", source: source, headers: healthyChain, status: http.StatusForbidden, good: true},
		{name: "wrong-chain", source: source, headers: []string{"inner", "outer"}, status: http.StatusForbidden},
		{name: "unselected-page-hook", source: source, status: http.StatusForbidden},
		{name: "wrong-status", source: source, headers: healthyChain, status: http.StatusOK},
		{name: "wrong-lang", source: strings.Replace(source, `hreflang="fr"`, `hreflang="en"`, 1), headers: healthyChain, status: http.StatusForbidden},
		{name: "missing-error-policy", source: strings.Replace(source, `name="robots"`, `name="unused"`, 1), headers: healthyChain, status: http.StatusForbidden},
		{name: "wrong-owner", source: strings.Replace(source, ">Fallback<", ">Lead<", 1), headers: healthyChain, status: http.StatusForbidden},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := html.Parse(strings.NewReader(testCase.source))
			require.NoError(t, err)
			observation := ProbeObservation{Path: want.Path, Status: testCase.status, Semantic: readResponseContract(document), ResponseHeaders: map[string][]string{"X-Edition-Trace": testCase.headers}}
			for _, mode := range []string{"compiled", "interp"} {
				err := checkProbe(&casefile.Case{}, mode, want, &observation)
				assert.Equal(t, testCase.good, err == nil, "%s: %v", mode, err)
			}
		})
	}
}

func TestValidIntegrityHonoursOnlyStrongestAlgorithm(t *testing.T) {
	body := []byte("asset")
	sum256 := sha256.Sum256(body)
	sum512 := sha512.Sum512(body)
	weak := "sha256-" + base64.StdEncoding.EncodeToString(sum256[:])
	strong := "sha512-" + base64.StdEncoding.EncodeToString(sum512[:])
	wrongStrong := "sha512-" + base64.StdEncoding.EncodeToString(make([]byte, sha512.Size))

	assert.True(t, validIntegrity(weak+" "+strong, body))
	assert.True(t, validIntegrity(wrongStrong+" "+strong, body), "any digest of the strongest algorithm may match")
	assert.False(t, validIntegrity(weak+" "+wrongStrong, body), "a matching weaker digest must not rescue a strong mismatch")
	assert.True(t, validIntegrity(strong+"?ct=application/javascript", body), "integrity options are ignored")
	assert.False(t, validIntegrity("md5-abc", body), "a declaration without a supported algorithm is rejected")
}

func newFakePageServer(t *testing.T, script []byte) *httptest.Server {
	t.Helper()
	sum := sha512.Sum384(script)
	integrity := "sha384-" + base64.StdEncoding.EncodeToString(sum[:])
	style := []byte("body{}")
	styleSum := sha256.Sum256(style)
	page := `<html><head><title>Shop</title>` +
		`<script src="` + fakeComponentScript + `" integrity="` + integrity + `"></script>` +
		`<script src="` + fakeComponentScript + `" integrity="` + integrity + `"></script>` +
		`<link rel="stylesheet" href="/style.css" integrity="sha256-` + base64.StdEncoding.EncodeToString(styleSum[:]) + `">` +
		`</head><body><div id="app"><section id="case-frame"><div id="case-root"><h2 id="case-title">Shop</h2>` +
		`<ul id="case-list"><li>one</li><li>two</li></ul><span id="case-visible"></span><x-card></x-card></div></section></div></body></html>`
	mux := http.NewServeMux()
	mux.HandleFunc("GET /shop", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set(headerContentType, "text/html; charset=utf-8")
		_, _ = io.WriteString(writer, page)
	})
	mux.HandleFunc("GET "+fakeComponentScript, func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set(headerContentType, "text/javascript")
		_, _ = writer.Write(script)
	})
	mux.HandleFunc("GET /style.css", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("ETag", `"style-v1"`)
		if request.Header.Get("If-None-Match") == `"style-v1"` {
			writer.WriteHeader(http.StatusNotModified)
			return
		}
		writer.Header().Set(headerContentType, "text/css")
		_, _ = writer.Write(style)
	})
	mux.HandleFunc("GET /account", func(writer http.ResponseWriter, request *http.Request) {
		theme, err := request.Cookie("theme")
		if err != nil {
			http.Error(writer, "no theme", http.StatusBadRequest)
			return
		}
		writer.Header().Set("Cache-Control", "private")
		_, _ = io.WriteString(writer, "<html><body><div id=\"app\">"+theme.Value+"</div></body></html>")
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func fakePageCase() casefile.Case {
	return casefile.Case{
		Request:  "/shop",
		Expected: casefile.Expected{Status: http.StatusOK, Tag: "x-card", Title: "Shop"},
		Probes: []casefile.Probe{
			{Path: "/account", Actor: "reader", Cookies: map[string]string{"theme": "dark"}, ResponseHeaders: map[string][]string{"Cache-Control": {"private"}}},
			{Path: "/style.css", Asset: &casefile.ResponseAsset{MediaType: "text/css", Revalidate: true}},
		},
	}
}

func TestObserveRecordsThePageAssetsAndProbes(t *testing.T) {
	server := newFakePageServer(t, []byte("customElements.define('x-card', class {})"))
	testCase := fakePageCase()

	o, err := observe(t.Context(), server.Client(), server.URL, &testCase)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, o.Status)
	assert.Equal(t, "text/html", o.ContentType)
	assert.Equal(t, "Shop", o.Heading)
	assert.Equal(t, []string{"one", "two"}, o.Rows)
	assert.True(t, o.Frame)
	assert.True(t, o.Visible)
	assert.True(t, o.Component)
	assert.Equal(t, "x-card", o.ComponentTag)
	assert.Equal(t, fakeComponentScript, o.ComponentAsset)
	assert.Equal(t, map[string]string{"x-card": fakeComponentScript}, o.ComponentModules)
	require.Len(t, o.Probes, 2)
	account := o.Probes[0]
	assert.Equal(t, "reader", account.Actor)
	assert.Equal(t, "dark", account.Text)
	assert.Equal(t, map[string][]string{"Cache-Control": {"private"}}, account.ResponseHeaders)
	require.NotNil(t, o.Probes[1].Asset)
	assert.Equal(t, sha256Hex([]byte("body{}")), o.Probes[1].Asset.SHA256)
	require.NotNil(t, o.Probes[1].Validation)
	assert.Equal(t, http.StatusNotModified, o.Probes[1].Validation.MatchedStatus)
	assert.Equal(t, http.StatusOK, o.Probes[1].Validation.UnmatchedStatus)
	assert.Equal(t, sha256Hex([]byte("body{}")), o.Probes[1].Validation.UnmatchedSHA256)
}

func TestObserveRejectsBrokenResponses(t *testing.T) {
	server := newFakePageServer(t, []byte("customElements.define('x-card', class {})"))
	for name, mutation := range map[string]struct {
		mutate func(*casefile.Case)
		want   string
	}{
		"status":        {mutate: func(c *casefile.Case) { c.Expected.Status = http.StatusCreated }, want: "status: got 200, want 201"},
		"remote path":   {mutate: func(c *casefile.Case) { c.Request = "//elsewhere/shop" }, want: "expected local path"},
		"probe limit":   {mutate: func(c *casefile.Case) { c.Probes = make([]casefile.Probe, probeLimit+1) }, want: "probe limit"},
		"probe request": {mutate: func(c *casefile.Case) { c.Probes[0].Path = "relative" }, want: "probe/request"},
		"probe cookie":  {mutate: func(c *casefile.Case) { c.Probes[0].Actor = "" }, want: "cookies need a named actor"},
		"missing app":   {mutate: func(c *casefile.Case) { c.Request = "/style.css" }, want: "document has no #app"},
	} {
		t.Run(name, func(t *testing.T) {
			testCase := fakePageCase()
			mutation.mutate(&testCase)
			_, err := observe(t.Context(), server.Client(), server.URL, &testCase)
			assert.ErrorContains(t, err, mutation.want)
		})
	}
}

func TestObserveRejectsTamperedAssets(t *testing.T) {
	script := []byte("customElements.define('x-card', class {})")
	server := newFakePageServer(t, script)
	testCase := fakePageCase()
	assets := []asset{{path: fakeComponentScript, integrity: "sha384-AAAA", script: true}}
	var o Observation

	err := verifyAssets(t.Context(), server.Client(), server.URL, &testCase, assets, &o)
	assert.ErrorContains(t, err, "integrity: digest mismatch")

	err = verifyAssets(t.Context(), server.Client(), server.URL, &testCase, []asset{{path: "/missing.js", script: true}}, &o)
	assert.ErrorContains(t, err, "asset: empty or failed response for /missing.js (HTTP 404)")
}

func TestSubsystemFailuresCollectsEachSelectedSubsystem(t *testing.T) {
	c := &casefile.Case{
		Target: "embedded-deploy",
		Expected: casefile.Expected{
			Auth:     generate.New("auth-contracts", 7, 0).Expected.Auth,
			FormShim: generate.New("form-shim-contracts", 7, 0).Expected.FormShim,
			Topology: &casefile.GoTopology{Initial: 1, Step: 1},
		},
	}

	failures := subsystemFailures(c, &Observation{})

	assert.Len(t, failures, 4)
	assert.Empty(t, subsystemFailures(&casefile.Case{}, &Observation{}))
}
