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
	"fmt"
	"net/url"
	"strings"
)

// BrowserRouteVisit records a real document navigation and its final URL.
type BrowserRouteVisit struct {
	// Path is the requested browser route.
	Path string

	// Landed is the final browser route.
	Landed string

	// ProgramRoot is the observed program root tag.
	ProgramRoot string `json:",omitempty"`

	// DriverError records the browser driver's failure.
	DriverError string `json:",omitempty"`

	// Output holds the ordered program leaves.
	Output []ProgramOutput `json:",omitempty"`

	// Status is the HTTP response status.
	Status int

	// RootHidden reports whether the program root is hidden.
	RootHidden bool `json:",omitempty"`

	// Program reports whether the browser observed the program root and outputs.
	Program bool `json:",omitempty"`
}

const (
	// MaxBrowserProgramRoutes bounds browser program visits in one case.
	MaxBrowserProgramRoutes = 8

	// MaxBrowserReceiptBytes bounds one worker JSON result before decoding.
	MaxBrowserReceiptBytes = 2 << 20

	// maxBrowserPathBytes bounds one local browser destination.
	maxBrowserPathBytes = 1024
)

// BrowserProgramPaths is the only part of the program contract sent to Chromium.
//
// Returns []string which lists selected browser-program routes in probe order.
// Returns error which reports an invalid, excessive or nonlocal route list, or nil on
// success.
func (c *Case) BrowserProgramPaths() ([]string, error) {
	var paths []string
	seen := map[string]bool{}
	for index := range c.Probes {
		probe := &c.Probes[index]
		if !probe.BrowserProgram {
			continue
		}
		if !probe.Program || seen[probe.Path] || probe.Path == c.Request || !LocalBrowserPath(probe.Path) ||
			len(paths) >= MaxBrowserProgramRoutes {
			return nil, fmt.Errorf("invalid or duplicate browser program route %q", probe.Path)
		}
		seen[probe.Path] = true
		paths = append(paths, probe.Path)
	}
	return paths, nil
}

// LocalBrowserPath reports whether a path is a bounded local browser destination.
//
// Takes destination (string) which is the proposed browser destination.
//
// Returns bool which reports whether the path is a bounded same-origin destination.
func LocalBrowserPath(destination string) bool {
	parsed, err := url.ParseRequestURI(destination)
	return len(destination) <= maxBrowserPathBytes && err == nil && !parsed.IsAbs() && parsed.Host == "" &&
		strings.HasPrefix(destination, "/") && !strings.HasPrefix(destination, "//") &&
		!strings.ContainsAny(destination, "\\#")
}
