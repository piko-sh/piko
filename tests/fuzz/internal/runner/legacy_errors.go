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
	"fmt"
	"net/http"
	"path"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// missingCollectionItemPath is the collection route whose document exists only in the
	// application, so the external collection has no item for it.
	missingCollectionItemPath = "/catalog/local-only"

	// missingCollectionItemSource is the application document behind that route.
	missingCollectionItemSource = "content/catalog/local-only.md"
)

// legacyNotFound identifies missing-route probes answered by the framework's own 404
// fallback, which writes a plain body in production and a diagnostic page in development.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes probe (*casefile.Probe) which holds the saved request and its expected response.
//
// Returns bool which reports whether the request uses a legacy missing-route contract.
func legacyNotFound(c *casefile.Case, probe *casefile.Probe) bool {
	if probe.Status != http.StatusNotFound || probe.Semantic != nil || probe.Program || probe.Text != "" || len(probe.ByMode) > 0 {
		return false
	}
	for file := range c.Files {
		if strings.HasPrefix(file, "pages/") && strings.HasPrefix(path.Base(file), "!") {
			return false
		}
	}
	switch probe.Path {
	case "/missing":
		return slices.Contains(c.Features, "typed-error") && strings.Contains(c.Files["pages/missing.pk"], `&piko.NotFoundError{Resource:"item", ID:"missing"}`)
	case "/missing-route":
		return slices.Contains(c.Features, "unmatched-route") && c.Files["pages/missing-route.pk"] == ""
	case missingCollectionItemPath:
		return c.UsesCollections() && c.Files[missingCollectionItemSource] != ""
	}
	return false
}

// checkLegacyNotFound checks the mode-specific legacy missing-route response.
//
// Takes probe (*casefile.Probe) which holds the saved request and its expected response.
// Takes got (*ProbeObservation) which holds the observed result.
// Takes mode (string) which selects the server mode.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkLegacyNotFound(probe *casefile.Probe, got *ProbeObservation, mode string) error {
	if mode == modeCompiled {
		want := "Not Found\n"
		if probe.Path == "/missing-route" {
			want = "404 page not found\n"
		}
		if got.RawHTML != want {
			return fmt.Errorf("probe/fallback: %s production body %q, expected %q", probe.Path, got.RawHTML, want)
		}
		return nil
	}
	if mode != modeInterpreted {
		return fmt.Errorf("probe/fallback-mode: %s unknown mode %q", probe.Path, mode)
	}
	return checkDevelopmentNotFound(probe, got.RawHTML)
}

// checkDevelopmentNotFound checks the interpreted host's development 404 page.
//
// Takes probe (*casefile.Probe) which holds the saved request.
// Takes page (string) which holds the served markup.
//
// Returns error which reports malformed markup or unexpected detail, or nil when the page
// matches.
func checkDevelopmentNotFound(probe *casefile.Probe, page string) error {
	document, err := html.Parse(strings.NewReader(page))
	if err != nil {
		return err
	}
	fields := map[string][]string{}
	for node := range document.Descendants() {
		if node.Type == html.ElementNode && (node.Data == "title" || node.Data == "h1" || node.Data == "pre") {
			fields[node.Data] = append(fields[node.Data], contents(node))
		}
	}
	if !slices.Equal(fields["title"], []string{"404 Not Found"}) || !slices.Equal(fields["h1"], []string{"404 Not Found"}) || len(fields["pre"]) != 1 {
		return fmt.Errorf("probe/fallback: %s malformed development 404", probe.Path)
	}
	detail := fields["pre"][0]
	unexpectedRoute := probe.Path == "/missing-route" && detail != "no route matches /missing-route"
	unexpectedItem := probe.Path == "/missing" && (!strings.HasSuffix(detail, "item not found: missing") || !strings.Contains(detail, "pages/missing.pk"))
	unexpectedCollectionItem := probe.Path == missingCollectionItemPath &&
		!strings.HasSuffix(detail, `collection item not found: collection "catalog", route "local-only"`)
	if unexpectedRoute || unexpectedItem || unexpectedCollectionItem {
		return fmt.Errorf("probe/fallback: %s unexpected development detail %q", probe.Path, detail)
	}
	return nil
}
