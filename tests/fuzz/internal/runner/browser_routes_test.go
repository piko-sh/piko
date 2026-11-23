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
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

func TestBrowserProgramRouteSensitivity(t *testing.T) {
	testCase := generate.New("pages", 7, 1)
	var probes []casefile.Probe
	for _, probe := range testCase.Probes {
		if probe.BrowserProgram {
			probes = append(probes, probe)
		}
	}
	require.Len(t, probes, 2, "want two scoped variants")
	valid := func() Observation {
		client := *testCase.Expected.Client
		client.BrowserRoutes = []casefile.BrowserRouteVisit{{Path: testCase.Request, Landed: testCase.Request, Status: http.StatusOK}}
		for _, probe := range probes {
			client.BrowserRoutes = append(client.BrowserRoutes, casefile.BrowserRouteVisit{Path: probe.Path, Landed: probe.Path, Status: probe.Status, Program: true, ProgramRoot: probe.ProgramRoot, Output: slices.Clone(probe.Output)})
		}
		return Observation{Client: &client}
	}
	_, err := BrowserRouteResults(&testCase, new(valid()), true)
	require.NoError(t, err)
	for _, mutation := range []struct {
		mutate     func(*casefile.BrowserRouteVisit)
		name, want string
	}{
		{name: "url alone", want: "/root", mutate: func(route *casefile.BrowserRouteVisit) { route.Program = false; route.Output = nil }},
		{name: "wrong document", want: "/navigation", mutate: func(route *casefile.BrowserRouteVisit) { route.Landed = "/main" }},
		{name: "wrong status", want: "/status", mutate: func(route *casefile.BrowserRouteVisit) { route.Status = http.StatusNotFound }},
		{name: "wrong key", want: "/left@" + probes[0].Output[0].FullKey + "/key", mutate: func(route *casefile.BrowserRouteVisit) { route.Output[0].ScopedKey = "second/detail" }},
		{name: "wrong text", want: "/markup/text", mutate: func(route *casefile.BrowserRouteVisit) { route.Output[2].Text += "wrong" }},
		{name: "wrong order", want: "/left@" + probes[0].Output[0].FullKey + "/order", mutate: func(route *casefile.BrowserRouteVisit) {
			route.Output[0], route.Output[1] = route.Output[1], route.Output[0]
		}},
		{name: "wrong value", want: "/markup/value", mutate: func(route *casefile.BrowserRouteVisit) { route.Output[2].Value = "wrong" }},
		{name: "driver exception", want: "/driver", mutate: func(route *casefile.BrowserRouteVisit) { route.DriverError = "did not read DOM" }},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			o := valid()
			mutation.mutate(&o.Client.BrowserRoutes[1])
			data, err := BrowserRouteResults(&testCase, &o, true)
			require.Error(t, err)
			assert.Contains(t, err.Error(), probes[0].Path+mutation.want)
			assert.False(t, data[probes[0].Path].Verified, "route metrics=%v", data)
			assert.True(t, data[probes[1].Path].Verified, "route metrics=%v", data)
			err = Check(&testCase, &o)
			require.Error(t, err, "missing independent Check error")
			assert.True(t, strings.HasPrefix(err.Error(), "browser/route:"), "%v", err)
		})
	}
	o := valid()
	o.Client.BrowserRoutes = o.Client.BrowserRoutes[:1]
	_, err = BrowserRouteResults(&testCase, &o, true)
	require.Error(t, err, "missing secondary routes passed")
	testCase.BrowserSampleVersion = 0
	got, err := BrowserRouteResults(&testCase, &o, true)
	require.NoError(t, err)
	assert.Empty(t, got, "archived contract changed")
}

func TestBrowserRouteResultsVerifyModelledJourneys(t *testing.T) {
	for _, target := range []string{"browser-lifecycle", "checkout-workflow", "partial-components", "browser-events", "markup"} {
		t.Run(target, func(t *testing.T) {
			c := generate.New(target, 7, 0)
			o := modelledObservation(&c)

			evidence, _ := BrowserRouteResults(&c, &o, true)

			assert.NotEmpty(t, evidence)
		})
	}
}
