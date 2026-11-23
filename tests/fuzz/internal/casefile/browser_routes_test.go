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

package casefile

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalBrowserPath(t *testing.T) {
	for _, path := range []string{"/", "/a/b?c=d", "/東京"} {
		assert.True(t, LocalBrowserPath(path), path)
	}
	for _, path := range []string{"", "relative", "//host/path", "https://example.test/", "/a#b", "/a\\b", "/" + strings.Repeat("a", maxBrowserPathBytes)} {
		assert.False(t, LocalBrowserPath(path), path)
	}
}

func TestBrowserProgramPathsListsSelectedRoutes(t *testing.T) {
	c := Case{Request: "/", Probes: []Probe{
		{Path: "/one", Program: true, BrowserProgram: true},
		{Path: "/server-only", Program: true},
		{Path: "/two", Program: true, BrowserProgram: true},
	}}

	paths, err := c.BrowserProgramPaths()

	require.NoError(t, err)
	assert.Equal(t, []string{"/one", "/two"}, paths)
}

func TestBrowserProgramPathsRejectsInvalidRoutes(t *testing.T) {
	testCases := []struct {
		name   string
		probes []Probe
	}{
		{name: "not a program", probes: []Probe{{Path: "/one", BrowserProgram: true}}},
		{name: "duplicate", probes: []Probe{{Path: "/one", Program: true, BrowserProgram: true}, {Path: "/one", Program: true, BrowserProgram: true}}},
		{name: "initial request", probes: []Probe{{Path: "/", Program: true, BrowserProgram: true}}},
		{name: "remote", probes: []Probe{{Path: "//remote", Program: true, BrowserProgram: true}}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := Case{Request: "/", Probes: testCase.probes}

			_, err := c.BrowserProgramPaths()

			assert.Error(t, err)
		})
	}
}

func TestBrowserProgramPathsBoundsTheRouteCount(t *testing.T) {
	c := Case{Request: "/"}
	for index := range MaxBrowserProgramRoutes + 1 {
		c.Probes = append(c.Probes, Probe{Path: "/route-" + strings.Repeat("x", index+1), Program: true, BrowserProgram: true})
	}

	_, err := c.BrowserProgramPaths()

	assert.Error(t, err)
}
