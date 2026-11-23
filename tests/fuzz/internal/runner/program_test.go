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
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestProgramProbeRejectsIdenticalWrongRenders(t *testing.T) {
	const markup = `<output>outside</output><main id="fuzz-program"><output data-fuzz-output="a" data-value="">1</output><section><output data-fuzz-output="b" data-value="x&amp;y">-2</output></section></main>`
	want := casefile.Probe{Path: "/program?input=right", Status: http.StatusOK, Program: true, Output: []casefile.ProgramOutput{{ID: "a", HasValue: true, Text: "1"}, {ID: "b", HasValue: true, Value: "x&y", Text: "-2"}}}
	testCase := casefile.Case{Expected: casefile.Expected{Status: http.StatusOK}, Probes: []casefile.Probe{want}}
	for _, mutation := range []struct{ previous, replacement, failed string }{
		{previous: "", replacement: "", failed: ""},
		{previous: "fuzz-program", replacement: "absent", failed: "/root"},
		{previous: `<output data-fuzz-output="a" data-value="">1</output>`, replacement: "", failed: "/count"},
		{previous: `data-fuzz-output="a"`, replacement: `data-fuzz-output="b"`, failed: "/a/order"},
		{previous: `data-value=""`, replacement: "", failed: "/a/attribute"},
		{previous: "x&amp;y", replacement: "Z", failed: "/b/value"},
		{previous: "-2", replacement: "2", failed: "/b/text"},
		{previous: `<output data-fuzz-output="b"`, replacement: `<span data-fuzz-output="b"`, failed: "/count"},
	} {
		document, err := html.Parse(strings.NewReader(strings.Replace(markup, mutation.previous, mutation.replacement, 1)))
		require.NoError(t, err)
		var probe ProbeObservation
		readProgram(document, &probe, false)
		root, output := probe.Program, probe.Output
		o := Observation{Status: http.StatusOK, ContentType: "text/html", MetadataTitle: "Piko Page", Probes: []ProbeObservation{{Path: want.Path, Status: http.StatusOK, Program: root, Output: output}}}
		require.NoError(t, compareObservations(&o, &o), "identical executions disagree")
		err = Check(&testCase, &o)
		if mutation.failed == "" {
			require.NoError(t, err, "correct output rejected")
			require.True(t, slices.Equal(want.Output, output), "correct output rejected: %+v", output)
			continue
		}
		failure, ok := errors.AsType[programMismatch](err)
		if assert.True(t, ok, "%q to %q: expected %s, got %v", mutation.previous, mutation.replacement, mutation.failed, err) {
			assert.Equal(t, want.Path+mutation.failed, failure.FailureSubject())
		}
	}
}

func TestNamedKeysDistinguishEqualLocalNames(t *testing.T) {
	want := casefile.Probe{Path: "/scope", Program: true, Output: []casefile.ProgramOutput{{ID: "a", ScopedKey: "first/detail"}, {ID: "b", ScopedKey: "second/detail"}}}
	source := `<main id="fuzz-program"><output data-fuzz-output="a" data-fuzz-check-key="true" p-key="first.0:0.detail"></output><output data-fuzz-output="b" data-fuzz-check-key="true" p-key="second:0.1.detail"></output></main>`
	for _, bad := range []string{"", "first.0:0.detail", "second.0:0.other", "second"} {
		markup := source
		if bad != "" {
			markup = strings.Replace(source, "second:0.1.detail", bad, 1)
		}
		document, err := html.Parse(strings.NewReader(markup))
		require.NoError(t, err)
		var probe ProbeObservation
		readProgram(document, &probe, false)
		err = checkProgram(want, probe)
		require.Equal(t, bad != "", err != nil, "replacement %q: %v", bad, err)
	}
}

func TestProgramLandmarksVisibilityAndCompleteKeys(t *testing.T) {
	const markup = `<main id="fuzz-program"><section><output data-fuzz-output="chapter" data-fuzz-check-key="true" p-key="doc.7:chapter">one</output></section></main>`
	want := casefile.Probe{Path: "/list", Actor: "editor", Stage: "after-edit", Status: http.StatusOK, Program: true, ProgramRoot: "main", Output: []casefile.ProgramOutput{{ID: "chapter", Text: "one", ScopedKey: "doc/chapter", FullKey: "doc.7:chapter", Owner: "main#fuzz-program/section"}}}
	testCase := casefile.Case{Expected: casefile.Expected{Status: http.StatusOK}, Probes: []casefile.Probe{want}}
	observeMarkup := func(markup string) Observation {
		t.Helper()
		document, err := html.Parse(strings.NewReader(markup))
		require.NoError(t, err)
		probe := ProbeObservation{Path: want.Path, Actor: want.Actor, Status: http.StatusOK}
		readProgram(document, &probe, true)
		return Observation{Status: http.StatusOK, ContentType: "text/html", MetadataTitle: "Piko Page", Probes: []ProbeObservation{probe}}
	}
	good := observeMarkup(markup)
	require.NoError(t, CompareCase(&testCase, &good, &good))
	for _, mutation := range []struct{ from, to, property string }{
		{from: "main", to: "aside", property: "root"},
		{from: "<main", to: "<main hidden", property: "root"},
		{from: "section", to: "aside", property: "owner"},
		{from: "<section", to: "<section style=\"display: none !important\"", property: "visibility"},
		{from: "<output", to: "<output aria-hidden=\"true\"", property: "visibility"},
		{from: "doc.7:chapter", to: "doc.9:chapter", property: "key"},
	} {
		bad := observeMarkup(strings.ReplaceAll(markup, mutation.from, mutation.to))
		subject := probeIdentity(want) + "/chapter@doc.7:chapter/" + mutation.property
		if mutation.property == "root" {
			subject = probeIdentity(want) + "/root"
		}
		for _, err := range []error{Check(&testCase, &bad), CompareCase(&testCase, &good, &bad), CompareCase(&testCase, &bad, &bad)} {
			assertFailureSubject(t, err, subject)
		}
		assert.Error(t, compareObservations(&good, &bad), "mode projection missed %s", mutation.to)
	}
	legacy := testCase
	legacy.Probes = slices.Clone(testCase.Probes)
	legacy.Probes[0].ProgramRoot = ""
	legacy.Probes[0].Output = []casefile.ProgramOutput{{ID: "chapter", Text: "one", ScopedKey: "doc/chapter"}}
	require.NoError(t, Check(&legacy, &good))
}
