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

package cache_invalidation_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
	"golang.org/x/sys/unix"
	"io/fs"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"piko.sh/piko"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestGeneratedPikoCase(t *testing.T) {
	manifest := os.Getenv("PIKO_FUZZ_CASE_DIR")
	if _, err := os.Stat(filepath.Join(manifest, "src", "fuzz-history", "marker.txt")); err == nil {
		checkRecordedSourceHistory(t, manifest)
	} else {
		require.ErrorIs(t, err, os.ErrNotExist)
	}
	if _, err := os.Stat(filepath.Join(manifest, "src", "pages", "partial-graph.pk")); err == nil {
		checkGeneratedGraph(t, manifest)
		return
	} else {
		require.ErrorIs(t, err, os.ErrNotExist)
	}
	token := os.Getenv("PIKO_FUZZ_CASE_TOKEN")
	require.NotEmpty(t, token)
	base := "start-" + token
	edit := "edit-" + token
	warm := observeVersions(t, []string{base, edit, base})
	for i, marker := range []string{base, edit, base} {
		fresh := observeVersions(t, []string{marker})
		if !maps.Equal(fresh[0], warm[i]) {
			require.FailNow(t, fmt.Sprintf("PIKO_CHECK cold-equivalence: warm and cold HTML differ after stage %d", i))
		}
	}
}

func observeVersions(t *testing.T, versions []string) []map[string]string {
	t.Helper()
	resetGlobalStateForTestIsolation()
	result := setupStagedServer(t, testCase{Name: "07_partial_dependency_change", Path: "testdata/07_partial_dependency_change"})
	defer result.cleanup()
	defer result.server.Close()
	pagePath := filepath.Join(result.srcDir, "pages", "index.pk")
	page, err := os.ReadFile(pagePath)
	require.NoError(t, err)
	page = bytes.ReplaceAll(page, []byte("<layout is=\"layout\"></layout>"), []byte("<piko:partial is=\"layout\"></piko:partial>"))
	require.Contains(t, string(page), "<piko:partial")
	require.NoError(t, os.WriteFile(pagePath, page, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(result.srcDir, "pages", "twin.pk"), page, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(result.srcDir, "pages", "independent.pk"), []byte("<template><p>independent-page</p></template>"), 0600))
	path := filepath.Join(result.srcDir, "partials", "layout.pk")
	partial, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(partial), "Initial layout content")
	var states []map[string]string
	for _, marker := range versions {
		require.NoError(t, replaceStageFile(path, bytes.ReplaceAll(partial, []byte("Initial layout content"), []byte(marker))))
		require.NoError(t, result.server.Generate(t.Context(), piko.RunModeDevInterpreted))
		state := map[string]string{}
		for _, route := range []string{"/", "/twin", "/independent"} {
			rec := httptest.NewRecorder()
			result.server.GetHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, route, nil))
			if rec.Code != http.StatusOK {
				require.FailNow(t, fmt.Sprintf("PIKO_CHECK status: page %s returned HTTP %d", route, rec.Code))
			}
			html := string(normaliseHTML(rec.Body.Bytes()))
			if route == "/independent" {
				if !strings.Contains(html, "independent-page") || strings.Contains(html, marker) {
					require.FailNow(t, fmt.Sprintf("PIKO_CHECK independent-page: unrelated edit changed %s while rendering %s", route, marker))
				}
			} else {
				if !strings.Contains(html, marker) {
					require.FailNow(t, fmt.Sprintf("PIKO_CHECK partial-invalidation: page %s did not render partial content %q", route, marker))
				}
			}
			state[route] = html
		}
		states = append(states, state)
	}
	return states
}

type graphOutput struct {
	ID, Text, Value string
	HasValue        bool
}
type graphProbe struct {
	Path   string
	Output []graphOutput
}
type graphStage struct{ leaf, helper bool }

func checkGeneratedGraph(t *testing.T, dir string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "case.json"))
	require.NoError(t, err)
	var manifest struct{ Probes []graphProbe }
	require.NoError(t, json.Unmarshal(raw, &manifest))
	var probes []graphProbe
	for _, p := range manifest.Probes {
		if strings.HasPrefix(p.Path, "/partial-graph?") {
			probes = append(probes, p)
		}
	}
	require.Len(t, probes, 2)
	stages := []graphStage{{}, {leaf: true}, {leaf: true, helper: true}, {helper: true}, {}}
	warm := runGraphStages(t, dir, probes, stages)
	for i, stage := range stages {
		cold := runGraphStages(t, dir, probes, []graphStage{stage})
		if !maps.Equal(cold[0], warm[i]) {
			require.FailNow(t, fmt.Sprintf("PIKO_CHECK graph-clean-build: warm and clean output differ at %d", i))
		}
	}
}

func runGraphStages(t *testing.T, manifestDir string, probes []graphProbe, stages []graphStage) []map[string]string {
	t.Helper()
	resetGlobalStateForTestIsolation()
	result := setupStagedServer(t, testCase{Name: "07_partial_dependency_change", Path: "testdata/07_partial_dependency_change"})
	defer result.cleanup()
	defer result.server.Close()
	graphSource := os.Getenv("PIKO_FUZZ_GRAPH_SOURCE")
	require.NotEmpty(t, graphSource, "PIKO_CHECK graph-staging: materialised graph editions must be present")
	read := func(edition, name string) string {
		b, err := os.ReadFile(filepath.Join(graphSource, edition, name))
//line piko_fuzz_scenario_test.go:155
		require.NoError(t, err)
		return string(b)
	}
	write := func(name, src string) {
		path := filepath.Join(result.srcDir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
		require.NoError(t, os.WriteFile(path, []byte(src), 0600))
	}
	graphDir := "partials/graph"
	root := filepath.Join(manifestDir, "src")
	err := filepath.WalkDir(filepath.Join(root, graphDir), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		name, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		write(name, read("base", name))
		return nil
	})
	require.NoError(t, err)
	page := read("base", "pages/partial-graph.pk")
	write("pages/partial-graph.pk", page)
	write("pages/twin.pk", page)
	write("pages/independent.pk", `<template><main id="fuzz-program"><output data-fuzz-output="independent">constant</output></main></template>`)
	leaf := "partials/graph/leaf.pk"
	helper := "pkg/graphtext/text.go"
	if _, err := os.Stat(filepath.Join(manifestDir, "src", helper)); errors.Is(err, fs.ErrNotExist) {
		helper = "pkg/graphmodel/label.go"
		if _, err := os.Stat(filepath.Join(manifestDir, "src", helper)); errors.Is(err, fs.ErrNotExist) {
			helper = "lib/graphmodel/label.go"
		}
	} else {
		for _, name := range []string{"pkg/graphmodel/label.go", "pkg/graphmodel/format.go"} {
			write(name, read("base", name))
		}
	}
	baseLeaf, baseHelper := read("base", leaf), read("base", helper)
	editedLeaf, editedHelper := read("edited", leaf), read("edited", helper)
	var observed []map[string]string
	for _, stage := range stages {
		l, h := baseLeaf, baseHelper
		if stage.leaf {
			l = editedLeaf
		}
		if stage.helper {
			h = editedHelper
		}
		write(leaf, l)
		write(helper, h)
//line piko_fuzz_scenario_test.go:207
		require.NoError(t, result.server.Generate(t.Context(), piko.RunModeDevInterpreted))
		state := map[string]string{}
		check := func(route string, want []graphOutput) {
			rec := httptest.NewRecorder()
			result.server.GetHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, route, nil))
			if rec.Code != http.StatusOK {
				require.FailNow(t, fmt.Sprintf("PIKO_CHECK graph-status: %s HTTP %d", route, rec.Code))
			}
			raw := rec.Body.String()
			got := graphOutputs(t, raw)
			if !slices.Equal(got, want) {
				require.FailNow(t, fmt.Sprintf("PIKO_CHECK graph-outputs: %s stage %+v got %v want %v", route, stage, got, want))
			}
			state[route] = string(normaliseHTML([]byte(raw)))
		}
		for _, probe := range probes {
			req, err := http.NewRequest(http.MethodGet, probe.Path, nil)
			require.NoError(t, err)
			run := req.URL.Query().Get("run")
			want := slices.Clone(probe.Output)
			for i := range want {
				o := &want[i]
				if stage.helper {
					o.Value = strings.Replace(o.Value, run+"-", run+"-changed-", 1)
					o.Text = strings.Replace(o.Text, run+"-", run+"-changed-", 1)
				}
				if stage.leaf && o.ID != "caption" {
					o.Text = "edited:" + o.Text
				}
			}
			check(probe.Path, want)
			check("/twin?"+req.URL.RawQuery, want)
		}
		check("/independent", []graphOutput{{ID: "independent", Text: "constant"}})
		observed = append(observed, state)
	}
	return observed
}

func graphOutputs(t *testing.T, raw string) []graphOutput {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(raw))
	require.NoError(t, err)
	for parent := range doc.Descendants() {
		for _, a := range parent.Attr {
			if a.Key != "id" || a.Val != "fuzz-program" {
				continue
			}
			var outputs []graphOutput
			for n := range parent.Descendants() {
				if n.Type != html.ElementNode || n.Data != "output" {
					continue
				}
				var out graphOutput
				for _, at := range n.Attr {
					switch at.Key {
					case "data-fuzz-output":
						out.ID = at.Val
					case "data-value":
						out.Value = at.Val
						out.HasValue = true
					}
				}
				for child := range n.Descendants() {
					if child.Type == html.TextNode {
						out.Text += child.Data
					}
				}
				outputs = append(outputs, out)
			}
			return outputs
		}
	}
	require.FailNow(t, "PIKO_CHECK graph-root: missing #fuzz-program")
	return nil
}

func checkRecordedSourceHistory(t *testing.T, manifest string) {
	t.Helper()
	read := func(file string) string {
		b, err := os.ReadFile(filepath.Join(manifest, "src", "fuzz-history", file))
		require.NoError(t, err)
		return string(b)
	}
	marker, base, repaired, invalid := read("marker.txt"), read("page.pk"), read("repaired.pk"), read("invalid.pk")
	const untouched = `<template><main id="fuzz-program"><output data-fuzz-output="edition">unaffected</output></main></template>`
	type version struct {
		name   string
		source string
		at     string
		value  string
	}
	stages := []version{
		{name: "baseline"},
		{name: "add", source: base, at: "edition", value: marker},
		{name: "rename", source: base, at: "renamed", value: marker},
		{name: "delete"},
		{name: "restore", source: base, at: "edition", value: marker},
		{name: "invalid", source: invalid, at: "edition"},
		{name: "repair", source: repaired, at: "edition", value: marker + "-repaired"},
		{name: "undo", source: base, at: "edition", value: marker},
	}
	observe := func(server *piko.SSRServer, v version) (map[string]string, error) {
		result := map[string]string{}
		for _, route := range []string{"edition", "renamed", "untouched"} {
			status, data, err := fetch(server, "/"+route)
			if err != nil {
				return nil, err
			}
			wantStatus, expected := http.StatusNotFound, ""
			if route == "untouched" {
				wantStatus, expected = http.StatusOK, "unaffected"
			} else if route == v.at {
				wantStatus, expected = http.StatusOK, v.value
			}
			if status != wantStatus {
				return nil, fmt.Errorf("/%s HTTP %d, want %d", route, status, wantStatus)
			}
			if status == http.StatusOK {
				doc, err := html.Parse(bytes.NewReader(data))
				if err != nil {
					return nil, err
				}
				var labels []string
				for n := range doc.Descendants() {
					for _, a := range n.Attr {
						if n.Type == html.ElementNode && n.Data == "output" && a.Key == "data-fuzz-output" && a.Val == "edition" {
							var text strings.Builder
							for c := range n.Descendants() {
								if c.Type == html.TextNode {
									text.WriteString(c.Data)
								}
							}
							labels = append(labels, text.String())
						}
					}
				}
				if !slices.Equal(labels, []string{expected}) {
					return nil, fmt.Errorf("/%s labels %q, want %q", route, labels, expected)
				}
			}
			result[route] = fmt.Sprintf("%d/%s", status, expected)
		}
		return result, nil
	}
	var live []map[string]string
	func() {
		resetGlobalStateForTestIsolation()
		w := setupWatchServerWithFiles(t, map[string]string{"pages/untouched.pk": untouched})
		defer w.cleanup()
		diagnostics, restoreDiagnostics := captureEditionDiagnostic(t)
		defer restoreDiagnostics()
		edition, renamed := filepath.Join(w.srcDir, "pages", "edition.pk"), filepath.Join(w.srcDir, "pages", "renamed.pk")
		for _, v := range stages {
			switch v.name {
			case "add", "restore", "invalid", "repair", "undo":
				require.NoError(t, os.WriteFile(edition, []byte(v.source), 0600))
			case "rename":
				require.NoError(t, os.Rename(edition, renamed))
			case "delete":
				require.NoError(t, os.Remove(renamed))
			}
			if v.name == "invalid" {
				deadline := time.NewTimer(watchPollTimeout)
				select {
				case <-diagnostics:
					deadline.Stop()
				case <-deadline.C:
					require.FailNow(t, "PIKO_CHECK edit-invalid-diagnostic: missing pages/edition.pk JIT diagnostic for FuzzMissingEdition")
				}
				status, body := doGet(t, w.server, "/untouched")
				if status != http.StatusOK || !bytes.Contains(body, []byte(">unaffected<")) {
					require.FailNow(t, fmt.Sprintf("PIKO_CHECK edit-invalid-sibling: status %d body %q", status, body))
				}
				live = append(live, map[string]string{"diagnostic": "FuzzMissingEdition"})
				continue
			}
			var got map[string]string
			require.EventuallyWithT(t, func(collect *assert.CollectT) {
				observed, err := observe(w.server, v)
				if assert.NoError(collect, err, "PIKO_CHECK edit-%s-route: route observations did not settle", v.name) {
					got = observed
				}
			}, watchPollTimeout, watchPollInterval)
			live = append(live, got)
		}
	}()
	for i, v := range stages {
		func() {
			resetGlobalStateForTestIsolation()
			files := map[string]string{"pages/untouched.pk": untouched}
			if v.at != "" {
				files["pages/"+v.at+".pk"] = v.source
			}
			if v.name == "invalid" {
				s := setupStagedServer(t, testCase{Name: "01_simple_page_modification", Path: "testdata/01_simple_page_modification"})
				defer s.cleanup()
				defer s.server.Close()
				for name, data := range files {
					require.NoError(t, os.WriteFile(filepath.Join(s.srcDir, name), []byte(data), 0600))
				}
				err := s.server.Generate(t.Context(), piko.GenerateModeAll)
				var diagnostic string
				if err != nil {
					diagnostic = err.Error()
				} else {
					cmd := exec.CommandContext(t.Context(), "go", "build", "-buildvcs=false", "./dist/...")
					cmd.Dir = s.srcDir
					cmd.Env = append(os.Environ(), "GOWORK="+filepath.Join(s.srcDir, "go.work"))
					data, buildErr := cmd.CombinedOutput()
					if buildErr == nil {
						require.FailNow(t, "PIKO_CHECK edit-clean-invalid: undefined identifier compiled successfully")
					}
					diagnostic = string(data)
				}
				if !strings.Contains(diagnostic, live[i]["diagnostic"]) {
					require.FailNow(t, fmt.Sprintf("PIKO_CHECK edit-clean-invalid: %s", diagnostic))
				}
				return
			}
			w := setupWatchServerWithFiles(t, files)
			defer w.cleanup()
			got, err := observe(w.server, v)
			if err != nil || !maps.Equal(got, live[i]) {
				require.FailNow(t, fmt.Sprintf("PIKO_CHECK edit-%s-clean: %v got %v want %v", v.name, err, got, live[i]))
			}
		}()
	}
}

func captureEditionDiagnostic(t *testing.T) (<-chan struct{}, func()) {
	t.Helper()
	fd := int(os.Stderr.Fd())
	savedFD, err := unix.FcntlInt(os.Stderr.Fd(), unix.F_DUPFD_CLOEXEC, 0)
	require.NoError(t, err)
	saved := os.NewFile(uintptr(savedFD), "saved-test-stderr")
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, unix.Dup2(int(writer.Fd()), fd))
	require.NoError(t, writer.Close())
	found, done := make(chan struct{}, 1), make(chan struct{})
	go func() {
		defer close(done)
		lines := bufio.NewScanner(reader)
		lines.Buffer(make([]byte, 4096), 1<<20)
		for lines.Scan() {
			line := lines.Bytes()
			_, _ = saved.Write(line)
			_, _ = saved.Write([]byte("\n"))
			if bytes.Contains(line, []byte("Failed to compile component")) && bytes.Contains(line, []byte("pages/edition.pk")) && bytes.Contains(line, []byte("FuzzMissingEdition")) {
				select {
				case found <- struct{}{}:
				default:
				}
			}
		}
	}()
	return found, func() {
		require.NoError(t, unix.Dup2(savedFD, fd))
		<-done
		require.NoError(t, reader.Close())
		require.NoError(t, saved.Close())
	}
}
